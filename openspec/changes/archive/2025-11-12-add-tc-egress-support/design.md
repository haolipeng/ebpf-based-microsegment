# 设计文档: TC Egress Hook 支持

## 文档元数据

- **变更 ID**: add-tc-egress-support
- **版本**: 1.0
- **最后更新**: 2025-11-11
- **状态**: design

## 1. 概述

### 1.1 目标

为微隔离系统添加 **TC Egress Hook** 支持，实现真正的双向流量控制：
- **Ingress 控制**: 谁可以访问我（已实现）
- **Egress 控制**: 我可以访问谁（本变更新增）

### 1.2 背景

当前系统仅实现 TC Ingress Hook，存在安全盲点：
- ❌ 容器可以主动发起任意出站连接
- ❌ 无法阻止数据外泄、横向移动、C&C 通信
- ❌ 不符合零信任原则

### 1.3 范围

**包含在内**:
- TC Egress Hook 实现（TCX + Legacy TC）
- 策略方向感知（ingress/egress/any）
- 流事件方向标记
- 双向策略管理 API

**不包含**:
- XDP Egress（XDP 本身不支持 egress）
- veth pair 架构改造（保持当前直接 attach 模式）
- 高级 egress 功能（DNS 拦截、URL 过滤等，留待后续）

## 2. 架构设计

### 2.1 系统架构

#### 当前架构 (仅 Ingress)

```
┌────────────────────────────────────────────┐
│         Container / Pod Namespace          │
│  ┌──────────────────────────────────────┐  │
│  │        Application Process           │  │
│  └──────────────────────────────────────┘  │
│                    ▲                        │
│                    │ Ingress Traffic        │
│                    │                        │
│  ┌─────────────────┴──────────────────┐    │
│  │       Network Interface (eth0)     │    │
│  └────────────────────────────────────┘    │
│                    ▲                        │
│                    │                        │
│         ┌──────────┴──────────┐            │
│         │  TC Ingress Hook    │            │
│         │  (已实现)            │            │
│         └─────────────────────┘            │
│                    ▲                        │
└────────────────────┼────────────────────────┘
                     │
         ┌───────────┴──────────┐
         │   eBPF Policy Engine │
         │   - policy_map       │
         │   - session_map      │
         └──────────────────────┘

问题: 无法控制 Egress 流量 ❌
```

#### 目标架构 (双向控制)

```
┌────────────────────────────────────────────┐
│         Container / Pod Namespace          │
│  ┌──────────────────────────────────────┐  │
│  │        Application Process           │  │
│  └──────────────────────────────────────┘  │
│            │                    ▲           │
│            │ Egress             │ Ingress   │
│            │                    │           │
│  ┌─────────▼────────────────────┴─────┐    │
│  │   Network Interface (eth0/veth)    │    │
│  └────────────────────────────────────┘    │
│            │                    ▲           │
│            │                    │           │
│   ┌────────▼────────┐  ┌────────┴────────┐ │
│   │ TC Egress Hook  │  │ TC Ingress Hook │ │
│   │ (✅ 新增)        │  │ (✅ 已存在)      │ │
│   └─────────────────┘  └─────────────────┘ │
│            │                    ▲           │
└────────────┼────────────────────┼───────────┘
             │                    │
             ▼                    │
    ┌────────────────────────────────────┐
    │      eBPF Policy Engine (共享)     │
    │  ┌──────────────────────────────┐  │
    │  │   policy_map (方向感知)      │  │
    │  │   wildcard_policy_map        │  │
    │  │   session_map (双向共享)     │  │
    │  │   stats_map (分方向统计)     │  │
    │  └──────────────────────────────┘  │
    └────────────────────────────────────┘
```

### 2.2 数据流

#### Ingress 数据流 (已存在)

```
外部数据包 → NIC → TC Ingress Hook → 策略匹配 → 会话查找/创建 → PASS/DROP → 容器
```

#### Egress 数据流 (新增)

```
容器 → 发送数据包 → TC Egress Hook → 策略匹配 → 会话查找/创建 → PASS/DROP → NIC
```

#### 会话状态共享

Ingress 和 Egress 共享同一个 `session_map`：

```
连接建立流程:
1. Egress: Pod A → Pod B (SYN)
   - TC Egress Hook 检查: Pod A 是否允许访问 Pod B
   - 创建会话: (A_IP:A_PORT, B_IP:B_PORT) → session_state

2. Ingress: Pod B 收到 (SYN)
   - TC Ingress Hook 检查: 是否允许来自 Pod A 的连接
   - 查找会话: 发现已存在 (反向查找)

3. Egress: Pod A 收到 (SYN-ACK)
   - 会话已建立，fast path

4. 双向数据传输
   - Ingress/Egress 都使用同一个 session entry
   - 更新统计信息 (bytes, packets)
```

### 2.3 组件设计

#### 2.3.1 eBPF 程序

**选项 1: 单一 eBPF 程序，双向 attach** (推荐)

```c
// src/bpf/tc_microsegment.bpf.c
SEC("tc")
int tc_microsegment_filter(struct __sk_buff *skb)
{
    // 1. 解析数据包
    struct packet_info pkt;
    if (parse_packet(skb, &pkt) < 0) {
        return TC_ACT_OK;  // 解析失败，放行
    }

    // 2. 确定方向
    // 从 skb->ingress_ifindex 判断（如果为 0，则是 egress）
    __u8 direction = (skb->ingress_ifindex != 0) ? DIR_INGRESS : DIR_EGRESS;

    // 3. 策略匹配
    struct policy_key key = {
        .src_ip = pkt.src_ip,
        .dst_ip = pkt.dst_ip,
        .dst_port = pkt.dst_port,
        .protocol = pkt.protocol,
        .direction = direction,  // ✅ 方向感知
    };

    struct policy_value *policy = bpf_map_lookup_elem(&policy_map, &key);

    // 4. 如果没有方向特定策略，尝试 direction=ANY
    if (!policy) {
        key.direction = DIR_ANY;
        policy = bpf_map_lookup_elem(&policy_map, &key);
    }

    // 5. 会话管理
    struct flow_key flow_key = {
        .src_ip = pkt.src_ip,
        .dst_ip = pkt.dst_ip,
        .src_port = pkt.src_port,
        .dst_port = pkt.dst_port,
        .protocol = pkt.protocol,
    };

    struct session_state *session = bpf_map_lookup_elem(&session_map, &flow_key);

    // 6. 执行动作
    if (policy && policy->action == ACTION_ALLOW) {
        // 记录流事件（标记方向）
        emit_flow_event(&flow_key, direction, ACTION_ALLOW, policy->rule_id);
        return TC_ACT_OK;  // 放行
    } else {
        emit_flow_event(&flow_key, direction, ACTION_DENY, 0);
        return TC_ACT_SHOT;  // 丢弃
    }
}
```

**优点**:
- 代码复用，维护简单
- 统一的策略匹配逻辑
- 自然共享 Maps

**选项 2: 两个独立的 eBPF 程序**

```c
// src/bpf/tc_ingress.bpf.c
SEC("tc/ingress")
int tc_ingress_filter(struct __sk_buff *skb) {
    // 仅处理 ingress 逻辑
}

// src/bpf/tc_egress.bpf.c
SEC("tc/egress")
int tc_egress_filter(struct __sk_buff *skb) {
    // 仅处理 egress 逻辑
}
```

**缺点**:
- 代码重复
- 需要手动 Map Pinning 共享状态
- 维护成本高

**决策**: 使用选项 1（单一程序，双向 attach）

#### 2.3.2 数据结构增强

**策略 Key 增强**:

```c
// src/bpf/policy_match.h

// 方向常量
#define DIR_ANY     0  // 双向都匹配
#define DIR_INGRESS 1  // 仅 ingress
#define DIR_EGRESS  2  // 仅 egress

struct policy_key {
    __u32 src_ip;       // 源 IP
    __u32 dst_ip;       // 目标 IP
    __u16 dst_port;     // 目标端口
    __u8  protocol;     // 协议 (TCP/UDP/ICMP)
    __u8  direction;    // ✅ 新增: 流量方向
} __attribute__((packed));

// 通配符策略 Key (保持不变，但添加 direction)
struct wildcard_key {
    __u32 dst_ip;
    __u16 dst_port;
    __u8  protocol;
    __u8  direction;    // ✅ 新增
} __attribute__((packed));
```

**流事件增强**:

```c
// src/bpf/flow_processing.h

struct flow_event {
    struct flow_key key;
    __u64 timestamp;
    __u32 policy_id;
    __u8  action;       // 0=deny, 1=allow
    __u8  direction;    // ✅ 新增: 1=ingress, 2=egress
    __u8  reserved[2];
} __attribute__((packed));
```

**会话状态** (保持不变，双向共享):

```c
// 会话状态不需要修改，双向流量使用同一个 session entry
struct session_state {
    __u64 start_time;
    __u64 last_seen;
    __u64 bytes_sent;
    __u64 bytes_recv;
    __u32 packets_sent;
    __u32 packets_recv;
    __u8  state;       // TCP state machine
    __u8  padding[3];
} __attribute__((packed));
```

**统计指标增强**:

```c
// 统计 key 扩展
#define STAT_TOTAL_PACKETS       0
#define STAT_ALLOWED_PACKETS     1
#define STAT_DENIED_PACKETS      2
#define STAT_NEW_SESSIONS        3
#define STAT_CLOSED_SESSIONS     4
#define STAT_ACTIVE_SESSIONS     5
#define STAT_POLICY_HITS         6
#define STAT_POLICY_MISSES       7

// ✅ 新增: 分方向统计
#define STAT_INGRESS_PACKETS     8
#define STAT_EGRESS_PACKETS      9
#define STAT_INGRESS_DENIED      10
#define STAT_EGRESS_DENIED       11
```

#### 2.3.3 TC Loader 扩展

**TCLoader 结构更新**:

```go
// src/agent/pkg/dataplane/tc_loader.go

type TCLoader struct {
    mode        DataPlaneMode
    iface       string
    ifaceIdx    int
    objs        *bpfObjects      // eBPF objects
    ingressLink link.Link        // ✅ Ingress link
    egressLink  link.Link        // ✅ 新增: Egress link
    maps        *DataPlaneMaps   // eBPF maps
}
```

**Load 方法更新**:

```go
func (l *TCLoader) Load() error {
    // 1. 加载 eBPF 程序
    spec, err := loadBpf()
    if err != nil {
        return fmt.Errorf("loading bpf spec: %w", err)
    }

    // 2. 加载 objects
    if err := spec.LoadAndAssign(&l.objs, nil); err != nil {
        return fmt.Errorf("loading bpf objects: %w", err)
    }

    // 3. Attach Ingress Hook
    if err := l.attachIngress(); err != nil {
        return fmt.Errorf("attaching ingress: %w", err)
    }

    // 4. ✅ Attach Egress Hook (新增)
    if err := l.attachEgress(); err != nil {
        l.detachIngress()  // 清理
        return fmt.Errorf("attaching egress: %w", err)
    }

    // 5. 初始化 Maps
    l.maps = &DataPlaneMaps{
        PolicyMap:         l.objs.PolicyMap,
        WildcardPolicyMap: l.objs.WildcardPolicyMap,
        SessionMap:        l.objs.SessionMap,
        StatsMap:          l.objs.StatsMap,
        FlowEventsRB:      l.objs.FlowEvents,
    }

    log.Infof("✓ TC dataplane loaded (mode=%v, ingress+egress)", l.mode)
    return nil
}
```

**Attach Egress 实现**:

```go
// ✅ 新增方法
func (l *TCLoader) attachEgress() error {
    switch l.mode {
    case ModeTCX:
        return l.attachTCXEgress()
    case ModeLegacyTC:
        return l.attachLegacyTCEgress()
    default:
        return fmt.Errorf("unsupported TC mode: %v", l.mode)
    }
}

// ✅ TCX Egress Hook
func (l *TCLoader) attachTCXEgress() error {
    link, err := link.AttachTCX(link.TCXOptions{
        Program:   l.objs.TcMicrosegmentFilter,  // 复用同一个程序
        Attach:    ebpf.AttachTCXEgress,         // ✅ Egress
        Interface: l.ifaceIdx,
    })
    if err != nil {
        return fmt.Errorf("attaching TCX egress: %w", err)
    }

    l.egressLink = link
    log.Debugf("Attached TCX egress hook to %s (ifindex=%d)", l.iface, l.ifaceIdx)
    return nil
}

// ✅ Legacy TC Egress Hook
func (l *TCLoader) attachLegacyTCEgress() error {
    // 1. 创建 egress qdisc (如果不存在)
    if err := netlink.QdiscAdd(&netlink.Clsact{
        QdiscAttrs: netlink.QdiscAttrs{
            LinkIndex: l.ifaceIdx,
            Handle:    netlink.MakeHandle(0xffff, 0),
            Parent:    netlink.HANDLE_CLSACT,
        },
    }); err != nil {
        if !os.IsExist(err) {
            return fmt.Errorf("adding clsact qdisc: %w", err)
        }
    }

    // 2. 创建 egress filter
    filter := &netlink.BpfFilter{
        FilterAttrs: netlink.FilterAttrs{
            LinkIndex: l.ifaceIdx,
            Parent:    netlink.HANDLE_MIN_EGRESS,  // ✅ Egress parent
            Handle:    1,
            Protocol:  unix.ETH_P_ALL,
            Priority:  1,
        },
        Fd:           l.objs.TcMicrosegmentFilter.FD(),
        Name:         "tc/egress",
        DirectAction: true,
    }

    if err := netlink.FilterAdd(filter); err != nil {
        return fmt.Errorf("adding egress filter: %w", err)
    }

    log.Debugf("Attached Legacy TC egress hook to %s", l.iface)
    return nil
}
```

**Unload 方法更新**:

```go
func (l *TCLoader) Unload() error {
    var errs []error

    // 1. Detach Egress Hook
    if err := l.detachEgress(); err != nil {
        errs = append(errs, err)
    }

    // 2. Detach Ingress Hook
    if err := l.detachIngress(); err != nil {
        errs = append(errs, err)
    }

    // 3. Close eBPF objects
    if l.objs != nil {
        if err := l.objs.Close(); err != nil {
            errs = append(errs, err)
        }
    }

    if len(errs) > 0 {
        return errors.Join(errs...)
    }

    log.Info("TC dataplane unloaded")
    return nil
}
```

#### 2.3.4 策略管理扩展

**Policy 结构增强**:

```go
// src/agent/pkg/policy/policy.go

type Policy struct {
    RuleID    uint32
    SrcIP     string  // CIDR format
    DstIP     string  // CIDR format
    DstPort   uint16
    Protocol  string  // "tcp", "udp", "icmp", "any"
    Action    string  // "allow", "deny"
    Direction string  // ✅ 新增: "any", "ingress", "egress"
}

// ✅ Direction 常量
const (
    DirectionAny     = "any"
    DirectionIngress = "ingress"
    DirectionEgress  = "egress"
)

// ✅ 转换为 eBPF direction 值
func (p *Policy) GetDirectionValue() uint8 {
    switch strings.ToLower(p.Direction) {
    case DirectionIngress:
        return 1  // DIR_INGRESS
    case DirectionEgress:
        return 2  // DIR_EGRESS
    default:
        return 0  // DIR_ANY
    }
}
```

**PolicyManager 更新**:

```go
// AddPolicy 更新
func (pm *PolicyManager) AddPolicy(policy *Policy) error {
    // 验证 direction
    if policy.Direction == "" {
        policy.Direction = DirectionAny  // 默认双向
    }

    direction := policy.GetDirectionValue()

    // 构造 policy key
    key := bpfPolicyKey{
        SrcIP:     ipToBinary(policy.SrcIP),
        DstIP:     ipToBinary(policy.DstIP),
        DstPort:   policy.DstPort,
        Protocol:  protocolToBinary(policy.Protocol),
        Direction: direction,  // ✅ 包含方向
    }

    value := bpfPolicyValue{
        RuleID: policy.RuleID,
        Action: actionToBinary(policy.Action),
    }

    // 写入 policy_map
    if err := pm.policyMap.Put(&key, &value); err != nil {
        return fmt.Errorf("adding policy to map: %w", err)
    }

    log.Infof("Policy added: rule_id=%d, direction=%s", policy.RuleID, policy.Direction)
    return nil
}
```

#### 2.3.5 API 扩展

**Policy API 增强**:

```go
// src/agent/pkg/api/handlers.go

// POST /api/v1/policies
// Request body:
type CreatePolicyRequest struct {
    SrcIP     string `json:"src_ip" binding:"required"`
    DstIP     string `json:"dst_ip" binding:"required"`
    DstPort   uint16 `json:"dst_port"`
    Protocol  string `json:"protocol" binding:"required"`
    Action    string `json:"action" binding:"required"`
    Direction string `json:"direction"`  // ✅ 新增: "any"/"ingress"/"egress"
}

func (s *Server) handleCreatePolicy(c *gin.Context) {
    var req CreatePolicyRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // 验证 direction
    direction := req.Direction
    if direction == "" {
        direction = "any"  // 默认
    }
    if direction != "any" && direction != "ingress" && direction != "egress" {
        c.JSON(400, gin.H{"error": "invalid direction, must be any/ingress/egress"})
        return
    }

    policy := &policy.Policy{
        RuleID:    generateRuleID(),
        SrcIP:     req.SrcIP,
        DstIP:     req.DstIP,
        DstPort:   req.DstPort,
        Protocol:  req.Protocol,
        Action:    req.Action,
        Direction: direction,  // ✅
    }

    if err := s.policyManager.AddPolicy(policy); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{"rule_id": policy.RuleID})
}
```

**Flow Events API 增强**:

```go
// GET /api/v1/flows 返回的 Flow 结构
type FlowInfo struct {
    SrcIP     string    `json:"src_ip"`
    DstIP     string    `json:"dst_ip"`
    SrcPort   uint16    `json:"src_port"`
    DstPort   uint16    `json:"dst_port"`
    Protocol  string    `json:"protocol"`
    Action    string    `json:"action"`
    Direction string    `json:"direction"`  // ✅ 新增
    Timestamp time.Time `json:"timestamp"`
    PolicyID  uint32    `json:"policy_id"`
}
```

## 3. 实现策略

### 3.1 开发阶段

#### Phase 1: eBPF 程序扩展 (2-3天)
1. 修改 `policy_match.h` 添加 direction 字段和常量
2. 修改 `flow_processing.h` 添加 direction 到 flow_event
3. 更新 `tc_microsegment.bpf.c`:
   - 从 skb 判断 direction
   - 策略匹配时考虑 direction
   - 流事件标记 direction
4. 更新统计指标
5. 编译测试

#### Phase 2: TC Loader 扩展 (2-3天)
1. 修改 `TCLoader` 结构添加 `egressLink`
2. 实现 `attachTCXEgress()` 方法
3. 实现 `attachLegacyTCEgress()` 方法
4. 更新 `Load()` 和 `Unload()` 生命周期
5. 单元测试

#### Phase 3: 策略管理扩展 (2天)
1. 修改 `Policy` 结构添加 `Direction` 字段
2. 更新 `PolicyManager.AddPolicy()` 支持 direction
3. 更新 `PolicyManager.ListPolicies()` 返回 direction
4. 验证逻辑更新

#### Phase 4: API 扩展 (1-2天)
1. 更新 Policy API 请求/响应结构
2. 更新 Flow API 响应结构
3. 参数验证
4. 文档更新

#### Phase 5: 集成测试 (2-3天)
1. Egress 策略执行测试
2. 双向策略测试
3. 会话状态一致性测试
4. 性能测试

#### Phase 6: 文档与示例 (1天)
1. 架构文档更新
2. API 文档更新
3. 示例配置

**总计**: 约 10-14 天

### 3.2 测试策略

#### 单元测试

**TCLoader 测试**:
```go
// src/agent/pkg/dataplane/tc_loader_test.go

func TestTCLoaderAttachEgress(t *testing.T) {
    // 测试 egress hook attach/detach
}

func TestTCLoaderDualHooks(t *testing.T) {
    // 测试同时加载 ingress + egress
}
```

**Policy 测试**:
```go
// src/agent/pkg/policy/policy_test.go

func TestPolicyDirection(t *testing.T) {
    // 测试 direction 字段验证
}

func TestPolicyManagerDirectionAware(t *testing.T) {
    // 测试方向感知策略添加
}
```

#### 集成测试

**测试场景 1: Egress 策略阻止出站连接**

```go
func TestEgressPolicyDeny(t *testing.T) {
    // 1. 添加 egress deny 策略
    policy := &Policy{
        SrcIP:     "192.168.1.10/32",
        DstIP:     "8.8.8.8/32",
        DstPort:   53,
        Protocol:  "udp",
        Action:    "deny",
        Direction: "egress",  // ✅
    }
    pm.AddPolicy(policy)

    // 2. 尝试从容器发起 DNS 查询
    err := sendUDPPacket("192.168.1.10", "8.8.8.8", 53)

    // 3. 验证被阻止
    assert.Error(t, err)

    // 4. 验证统计
    stats := dp.GetStatistics()
    assert.Equal(t, uint64(1), stats.EgressDenied)
}
```

**测试场景 2: 双向策略**

```go
func TestBidirectionalPolicy(t *testing.T) {
    // 1. 添加 ingress allow 策略
    pm.AddPolicy(&Policy{
        SrcIP:     "0.0.0.0/0",
        DstIP:     "192.168.1.10/32",
        DstPort:   80,
        Protocol:  "tcp",
        Action:    "allow",
        Direction: "ingress",
    })

    // 2. 添加 egress deny 策略（禁止主动访问外部）
    pm.AddPolicy(&Policy{
        SrcIP:     "192.168.1.10/32",
        DstIP:     "0.0.0.0/0",
        DstPort:   0,
        Protocol:  "any",
        Action:    "deny",
        Direction: "egress",
    })

    // 3. 测试 ingress: 外部可以访问容器 80 端口 ✅
    err := sendTCPPacket("external", "192.168.1.10", 80)
    assert.NoError(t, err)

    // 4. 测试 egress: 容器不能主动访问外部 ❌
    err = sendTCPPacket("192.168.1.10", "external", 443)
    assert.Error(t, err)
}
```

**测试场景 3: 会话状态共享**

```go
func TestSessionSharing(t *testing.T) {
    // 1. Egress: 容器发起连接
    sendTCPSYN("192.168.1.10", "192.168.1.20", 8080)

    // 2. 检查 session_map 创建
    session, err := getSession("192.168.1.10:12345", "192.168.1.20:8080")
    assert.NoError(t, err)
    assert.NotNil(t, session)

    // 3. Ingress: 返回 SYN-ACK
    sendTCPSYNACK("192.168.1.20", "192.168.1.10", 12345)

    // 4. 验证同一个 session
    session2, _ := getSession("192.168.1.10:12345", "192.168.1.20:8080")
    assert.Equal(t, session.StartTime, session2.StartTime)
}
```

#### 性能测试

**Egress Hook 延迟测试**:
```bash
# 测试 egress hook 对延迟的影响
./scripts/benchmark_egress.sh

# 期望: < 5μs 额外延迟
```

**吞吐量回归测试**:
```bash
# 测试添加 egress hook 后吞吐量变化
iperf3 -c <target> -t 60

# 期望: 吞吐量下降 < 5%
```

## 4. 向后兼容性

### 4.1 配置兼容

**旧配置** (无 direction 字段):
```yaml
policies:
  - rule_id: 1
    src_ip: "0.0.0.0/0"
    dst_ip: "192.168.1.10/32"
    dst_port: 80
    protocol: tcp
    action: allow
    # 缺少 direction
```

**处理**: 默认 `direction: "any"` (双向都匹配)

**新配置** (显式指定 direction):
```yaml
policies:
  - rule_id: 1
    src_ip: "0.0.0.0/0"
    dst_ip: "192.168.1.10/32"
    dst_port: 80
    protocol: tcp
    action: allow
    direction: ingress  # ✅ 显式指定
```

### 4.2 API 兼容

**旧 API 请求** (无 direction):
```json
{
  "src_ip": "0.0.0.0/0",
  "dst_ip": "192.168.1.10/32",
  "dst_port": 80,
  "protocol": "tcp",
  "action": "allow"
}
```

**处理**: 自动填充 `direction: "any"`

**新 API 响应** (包含 direction):
```json
{
  "rule_id": 1,
  "src_ip": "0.0.0.0/0",
  "dst_ip": "192.168.1.10/32",
  "dst_port": 80,
  "protocol": "tcp",
  "action": "allow",
  "direction": "any"
}
```

### 4.3 升级路径

**从 v1.x (无 egress) → v2.x (有 egress)**:

1. **安装新版本**: `sudo systemctl restart microsegment-agent`
2. **自动迁移**: 现有策略自动获得 `direction: "any"`
3. **功能增强**: 可以逐步添加显式的 ingress/egress 策略
4. **无破坏性**: 现有 ingress 功能继续工作

## 5. 安全考虑

### 5.1 威胁模型

**威胁 1: 策略绕过**
- **场景**: 攻击者尝试利用 ingress/egress 策略的差异绕过检查
- **缓解**: 会话状态在两个方向共享，确保双向一致性

**威胁 2: 拒绝服务**
- **场景**: 大量策略导致性能下降
- **缓解**: eBPF map 高效查找，性能测试验证

**威胁 3: 策略冲突**
- **场景**: Ingress 允许但 egress 拒绝，或反之
- **缓解**:
  - 默认策略: 最严格（ingress 和 egress 都要 allow）
  - 清晰的文档和最佳实践

### 5.2 最佳实践

**推荐策略配置**:

```yaml
# 示例 1: Web 服务 (仅允许 ingress 80/443，禁止主动出站)
policies:
  # 允许外部访问 80/443
  - rule_id: 1
    src_ip: "0.0.0.0/0"
    dst_ip: "web-pod/32"
    dst_port: 80
    protocol: tcp
    action: allow
    direction: ingress

  - rule_id: 2
    src_ip: "0.0.0.0/0"
    dst_ip: "web-pod/32"
    dst_port: 443
    protocol: tcp
    action: allow
    direction: ingress

  # 允许访问数据库 (egress)
  - rule_id: 3
    src_ip: "web-pod/32"
    dst_ip: "db-pod/32"
    dst_port: 5432
    protocol: tcp
    action: allow
    direction: egress

  # 默认拒绝其他 egress
  - rule_id: 999
    src_ip: "web-pod/32"
    dst_ip: "0.0.0.0/0"
    dst_port: 0
    protocol: any
    action: deny
    direction: egress
```

## 6. 性能优化

### 6.1 eBPF 程序优化

**优化 1: Early Return**
```c
// 快速路径: 已建立的会话直接放行
struct session_state *session = bpf_map_lookup_elem(&session_map, &flow_key);
if (session && session->state == SESSION_ESTABLISHED) {
    update_session_stats(session, skb->len);
    return TC_ACT_OK;  // Fast path
}
```

**优化 2: 共享策略匹配逻辑**
```c
// 将策略匹配逻辑提取为 inline 函数，避免代码重复
static __always_inline struct policy_value *
match_policy(struct packet_info *pkt, __u8 direction) {
    // 统一的策略匹配逻辑
}
```

### 6.2 Map 优化

**优化 1: 合理的 Map 大小**
```c
// policy_map: 根据实际策略数量调整
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);  // 支持 10k 策略
    __type(key, struct policy_key);
    __type(value, struct policy_value);
} policy_map SEC(".maps");
```

**优化 2: Per-CPU Maps**
```c
// stats_map 使用 Per-CPU 避免锁竞争
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 20);  // 20 个统计指标
    __type(key, __u32);
    __type(value, __u64);
} stats_map SEC(".maps");
```

## 7. 监控与可观测性

### 7.1 新增指标

**Egress 统计**:
- `egress_packets_total`: Egress 总数据包数
- `egress_packets_allowed`: Egress 允许的数据包数
- `egress_packets_denied`: Egress 拒绝的数据包数

**方向分布**:
- `ingress_vs_egress_ratio`: Ingress/Egress 流量比例

### 7.2 日志增强

**Egress 拒绝日志**:
```
[EGRESS DENY] 192.168.1.10:12345 -> 8.8.8.8:53 proto=UDP rule_id=0 (no matching policy)
```

### 7.3 调试工具

**查看 Egress Hook 状态**:
```bash
# 检查 TCX egress hook
tc filter show dev eth0 egress

# 检查 bpftool
bpftool prog show
```

## 8. 文档需求

### 8.1 用户文档

1. **架构文档**: 更新数据平面架构图
2. **配置指南**: Egress 策略配置示例
3. **API 文档**: 更新 Policy API 添加 direction 参数
4. **故障排查**: Egress 策略不生效的调试方法

### 8.2 开发者文档

1. **eBPF 程序设计**: Direction 字段的使用
2. **Loader 实现**: TCX/Legacy TC egress attach 细节
3. **测试指南**: 如何测试 egress 功能

## 9. 风险与缓解

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|---------|
| 性能下降 | 高 | 中 | 性能测试验证，优化 eBPF 程序 |
| 策略复杂度 | 中 | 高 | 清晰的文档和示例 |
| 向后兼容性问题 | 中 | 低 | 默认 direction=any，平滑迁移 |
| 双向策略冲突 | 中 | 中 | 验证工具，最佳实践文档 |
| 内核兼容性 | 低 | 低 | TCX fallback 到 Legacy TC |

## 10. 未来扩展

### 10.1 XDP + TC Egress 混合模式

```
Ingress: XDP (高性能) → TC Ingress (回退)
Egress:  TC Egress (唯一选择)
```

### 10.2 高级 Egress 功能

- **DNS 拦截**: 拦截 DNS 查询，实现域名级别的 egress 控制
- **URL 过滤**: 结合 HTTP 解析实现 URL 级别控制
- **证书验证**: 拦截 TLS 连接，验证证书

### 10.3 容器运行时集成

- **CNI 插件**: 自动配置 egress 策略
- **Kubernetes 集成**: 从 NetworkPolicy 自动生成 ingress/egress 策略

## 11. 参考实现

### NeuVector 方案对比

| 维度 | NeuVector | 我们的方案 |
|------|-----------|-----------|
| 架构 | veth pair (两端 TC ingress) | 单网卡双向 attach |
| Egress 实现 | TC ingress on inPort | TC egress on 同一网卡 |
| 复杂度 | 高 (需要 veth pair) | 低 (直接 attach) |
| 性能 | 中 (额外 veth 开销) | 高 (无额外 hop) |
| 侵入性 | 高 (改变网络拓扑) | 低 (透明) |

**结论**: 我们的方案更简洁、高效。

## 12. 批准清单

- [ ] 架构设计评审通过
- [ ] 安全评审通过
- [ ] 性能评估通过
- [ ] API 设计确认
- [ ] 测试计划确认
- [ ] 文档计划确认

---

**文档版本**: 1.0
**作者**: Claude Code AI Assistant
**日期**: 2025-11-11
**状态**: 待评审
