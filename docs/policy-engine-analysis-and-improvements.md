# 策略匹配引擎分析与改进建议

## 文档信息
- **日期**: 2025-11-12
- **版本**: 1.0
- **作者**: 基于 Phase 4 集成测试结果的深度分析

---

## 目录
1. [当前实现分析](#当前实现分析)
2. [Cilium 策略引擎对比](#cilium-策略引擎对比)
3. [测试中发现的问题](#测试中发现的问题)
4. [改进建议](#改进建议)
5. [实施优先级](#实施优先级)

---

## 1. 当前实现分析

### 1.1 架构概览

我们的策略匹配引擎采用**两层查找架构**:

```
数据包到达
    ↓
┌─────────────────────────────────────┐
│  第1层: 精确匹配 (Fast Path)        │
│  - Hash Map (O(1) 查找)             │
│  - 完整 6-tuple: src_ip, dst_ip,   │
│    src_port, dst_port, protocol,    │
│    direction                         │
│  - 先匹配方向特定策略,再匹配双向   │
└─────────────────────────────────────┘
    ↓ (miss)
┌─────────────────────────────────────┐
│  第2层: 通配符匹配 (Slow Path)      │
│  - Array Map (线性扫描)             │
│  - 支持通配符: port=0, CIDR        │
│  - 选择优先级最高的匹配策略         │
│  - 最多扫描 100 条策略              │
└─────────────────────────────────────┘
    ↓ (miss)
┌─────────────────────────────────────┐
│  默认策略: ALLOW                    │
└─────────────────────────────────────┘
```

### 1.2 策略路由逻辑

**Go 侧判断** (`src/agent/pkg/policy/policy.go:122-139`):

```go
func hasWildcard(p *Policy) bool {
    if p.SrcPort == 0 {           // 源端口为 0 → 通配符
        return true
    }
    if p.SrcIP == "0.0.0.0/0" {   // 全局 CIDR → 通配符
        return true
    }
    if p.Protocol == "any" {      // 任意协议 → 通配符
        return true
    }
    return false                  // 否则 → 精确匹配
}
```

**关键特点**:
- ✅ **简单直观**: 只要有任何通配符字段就进入 wildcard_policy_map
- ⚠️ **粗粒度**: 不区分部分通配符程度 (例如 `src_port=0, dst_port=8080`)

### 1.3 通配符匹配算法

**eBPF 侧实现** (`src/bpf/headers/policy_match.h:131-171`):

```c
struct wildcard_policy *best_match = NULL;
__u8 best_priority = 0;

#pragma unroll
for (__u32 i = 0; i < 100; i++) {
    wildcard = bpf_map_lookup_elem(&wildcard_policy_map, &i);

    if (wildcard->rule_id == 0) continue;  // 跳过空槽

    if (!matches_wildcard(key, wildcard, direction))
        continue;

    // 选择优先级最高的策略
    if (!best_match || wildcard->priority > best_priority) {
        best_match = wildcard;
        best_priority = wildcard->priority;
    }
}

return best_match ? best_match->action : POLICY_ACTION_ALLOW;
```

**时间复杂度**:
- **最好情况**: O(1) - 第一条策略匹配且为最高优先级
- **最坏情况**: O(n) - 扫描所有 100 条策略
- **平均情况**: O(n/2) - 扫描一半策略

### 1.4 优先级处理

**规则**:
- 数值越大优先级越高 (例如 priority=200 > priority=100)
- 当多个策略匹配时,选择 **优先级最高** 的策略
- 如果优先级相同,选择 **最后扫描到的** 策略 (危险!)

**测试验证**:
```
策略 1: Client → Server:8080, DENY, priority=200
策略 2: Client → Server:*,    ALLOW, priority=100

结果: ✅ 策略 1 生效 (DENY)
验证: 高优先级正确覆盖低优先级
```

### 1.5 方向感知

**方向检测** (TC 程序中):
```c
__u8 direction = (skb->ingress_ifindex != 0)
    ? POLICY_DIR_INGRESS   // 从外部进入
    : POLICY_DIR_EGRESS;   // 从内部发出
```

**策略方向匹配**:
```c
if (wildcard->direction != POLICY_DIR_ANY &&
    wildcard->direction != direction)
    return false;  // 方向不匹配,跳过此策略
```

**支持的方向**:
- `POLICY_DIR_ANY (0)`: 双向策略,匹配所有流量
- `POLICY_DIR_INGRESS (1)`: 仅匹配入站流量
- `POLICY_DIR_EGRESS (2)`: 仅匹配出站流量

---

## 2. Cilium 策略引擎对比

### 2.1 Cilium 架构特点

基于对 Cilium 官方文档和代码的研究,其策略引擎具有以下特点:

#### 2.1.1 基于身份的匹配 (Identity-Based Security)

**核心概念**:
```
Pod Labels → Security Identity (整数) → Policy Map
```

**优势**:
- ✅ **解耦网络地址**: 身份不依赖 IP 地址,支持动态环境
- ✅ **高效查找**: 身份是整数,Hash 查找极快
- ✅ **可扩展性**: 大规模集群中身份数量远小于 Pod 数量

**Cilium Policy Map 结构**:
```
Key:   (identity, port, direction)
Value: (policy_action, deny_reason)

时间复杂度: O(1) Hash 查找
```

**对比我们的实现**:
```
Key:   (src_ip, dst_ip, src_port, dst_port, protocol, direction)
Value: (action, priority, rule_id)

时间复杂度:
- 精确匹配: O(1)
- 通配符匹配: O(n) 线性扫描
```

#### 2.1.2 LPM Trie for CIDR 匹配

**数据结构**: `BPF_MAP_TYPE_LPM_TRIE`

**特点**:
- ✅ 最长前缀匹配,天然支持 CIDR 策略
- ✅ O(log n) 查找复杂度
- ✅ 内核原生支持 (4.11+)

**局限性**:
- ⚠️ L4 CIDR 策略会导致 map 条目激增
  - 公式: `entries = ports × CIDRs`
  - 例子: 10 个端口 × 100 个 CIDR = 1000 条目

**我们的实现**:
- 使用 **IP 掩码匹配** (手动实现)
- 在通配符匹配中支持 CIDR
- 没有使用 LPM Trie (依赖线性扫描)

#### 2.1.3 分层策略执行 (L3/L4/L7)

**架构**:
```
┌──────────────────────────────────────┐
│  eBPF (内核空间)                     │
│  - L3/L4 策略 (TC hook)              │
│  - 基于身份的快速过滤                │
│  - 高性能数据路径                    │
└──────────────────────────────────────┘
         ↓ (需要 L7 检查)
┌──────────────────────────────────────┐
│  Envoy Proxy (用户空间)              │
│  - L7 策略 (HTTP, gRPC, Kafka...)   │
│  - 协议感知                          │
│  - 深度包检测                        │
└──────────────────────────────────────┘
```

**我们的实现**:
- 仅支持 L3/L4 (IP, Port, Protocol)
- 没有 L7 感知能力
- 所有策略在 eBPF 中执行

#### 2.1.4 性能优化策略

**Cilium 2024 优化**:

1. **Per-Endpoint Policy Maps**
   - 每个 endpoint 独立的策略 map
   - 减少单个 map 的条目数
   - 提高缓存命中率

2. **Policy Map 容量调优**
   - 默认限制可通过 `--bpf-policy-map-max` 调整
   - 支持动态扩展

3. **Tail Call 优化**
   - 使用 `cilium_call_policy` map 进行尾调用
   - 突破 eBPF 指令限制
   - 复杂策略模块化

4. **Per-CPU Maps**
   - 减少锁竞争
   - 提高并发性能

**我们的实现**:
- 全局策略 map (所有流量共享)
- 固定容量 (100 条通配符策略)
- 没有使用 Per-CPU Maps
- 没有 Tail Call 机制

### 2.2 对比总结表

| 特性 | 我们的实现 | Cilium | 差距分析 |
|------|-----------|--------|----------|
| **匹配基础** | IP 5-tuple + Direction | Security Identity + Port + Direction | 🔴 缺少身份抽象 |
| **CIDR 支持** | 手动掩码匹配 (线性扫描) | LPM Trie (O(log n)) | 🟡 性能较差 |
| **查找复杂度** | O(1) 精确 + O(n) 通配符 | O(1) Hash + O(log n) LPM | 🟡 通配符慢 |
| **可扩展性** | 最多 100 条通配符策略 | 数万策略 (分片 maps) | 🔴 容量受限 |
| **L7 支持** | ❌ 无 | ✅ Envoy 集成 | 🔴 功能缺失 |
| **优先级处理** | 简单数值比较 | 基于身份分层 | 🟢 基本满足 |
| **并发性能** | 单一 map | Per-CPU Maps | 🟡 可改进 |
| **动态更新** | ✅ 支持 | ✅ 支持 | 🟢 相当 |

---

## 3. 测试中发现的问题

### 3.1 严重Bug: 结构体字段顺序不一致 🔴

**问题描述**: [已修复]

Go 和 eBPF 的 `wildcard_policy` 结构体字段顺序不一致,导致所有 Egress DENY 策略失效。

**根本原因**:
- Go 侧字段顺序: `Protocol, Direction, Action, LogEnabled`
- eBPF 侧字段顺序: `Protocol, Action, LogEnabled, Direction`
- 内存布局不匹配,导致字段值错位

**影响**:
- 严重性: 🔴 **Critical**
- 影响范围: 所有 Egress DENY 策略
- 检测难度: 高 (需要 eBPF tracing 才能发现)

**经验教训**:
1. ✅ 跨语言数据结构必须严格一致
2. ✅ 需要自动化验证工具检测结构体对齐
3. ✅ 单元测试应该覆盖结构体序列化/反序列化

**详细文档**: [bugfix-wildcard-policy-struct-alignment.md](bugfix-wildcard-policy-struct-alignment.md)

### 3.2 策略路由判断过于简单 🟡

**问题描述**:

当前 `hasWildcard()` 判断逻辑只检查 `SrcPort == 0`,导致:

```go
// 以下策略都会进入通配符 map (线性扫描)
Policy 1: src_ip=10.1.1.1, src_port=0, dst_port=8080  // 部分通配符
Policy 2: src_ip=0.0.0.0/0, src_port=0, dst_port=0    // 完全通配符
```

**影响**:
- 性能降低: 部分通配符策略无法利用 Hash Map 的 O(1) 查找
- 可扩展性差: 所有部分通配符策略都进入同一个 Array Map

**改进方向**:
- 引入 **多级通配符 Map**:
  - Level 1: 完全精确匹配 (Hash Map)
  - Level 2: 部分通配符匹配 (多个 Hash Map,按通配符字段分组)
  - Level 3: 完全通配符/CIDR 匹配 (LPM Trie 或线性扫描)

### 3.3 优先级相同时的不确定性 🟡

**问题描述**:

当多个策略优先级相同时,选择 **最后扫描到的** 策略:

```c
if (!best_match || wildcard->priority > best_priority) {
    best_match = wildcard;
    best_priority = wildcard->priority;
}
```

**危险场景**:
```
策略 1 (slot 0): priority=100, action=DENY
策略 2 (slot 1): priority=100, action=ALLOW

结果: 策略 2 生效 (ALLOW) - 因为后扫描
```

**改进建议**:
1. **严格要求优先级唯一**: 创建策略时检查冲突
2. **引入 Rule ID 作为 tie-breaker**:
   ```c
   if (wildcard->priority > best_priority ||
       (wildcard->priority == best_priority &&
        wildcard->rule_id < best_rule_id)) {  // 更小的 ID 优先
       best_match = wildcard;
   }
   ```
3. **文档化行为**: 明确说明相同优先级的处理方式

### 3.4 通配符策略容量限制 🟡

**问题描述**:

硬编码最多 100 条通配符策略:

```c
#pragma unroll
for (__u32 i = 0; i < 100; i++) {
    // ...
}
```

**局限性**:
- 无法动态调整
- 对大规模部署不友好
- 循环展开增加 eBPF 程序大小

**改进建议**:
1. **使用配置参数**: 通过环境变量或配置文件调整上限
2. **分片策略**: 按方向、协议等维度分片,减少单次扫描数量
3. **引入 Hash Map 索引**:
   ```
   hash(direction, protocol) → 子策略列表
   减少实际扫描的策略数量
   ```

### 3.5 CIDR 匹配性能 🟡

**问题描述**:

CIDR 匹配使用手动掩码计算:

```c
if ((key->src_ip & wildcard->src_ip_mask) !=
    (wildcard->src_ip & wildcard->src_ip_mask))
    return false;
```

在线性扫描中,每次都要计算掩码匹配。

**性能影响**:
- 最坏情况: 100 次掩码计算 (如果所有策略都是 CIDR)
- 缓存不友好: 每个策略独立计算

**改进建议**:
1. **使用 LPM Trie Map** (`BPF_MAP_TYPE_LPM_TRIE`):
   - O(log n) 查找复杂度
   - 内核原生优化
   - 适合 CIDR 策略

2. **CIDR 策略分离**:
   ```
   精确 IP 策略 → Hash Map
   CIDR 策略 → LPM Trie Map
   ```

### 3.6 复杂双向策略测试失败 ⚠️

**问题描述**:

某些涉及多个策略冲突的测试场景失败:

```
TestE2E_BidirectionalPolicy_IngressDenyEgressAllow
TestE2E_BidirectionalPolicy_BothAllow
TestE2E_BidirectionalPolicy_BothDeny
```

**可能原因** (待深入调查):
1. 会话缓存机制与策略方向的交互
2. 多个策略之间的时序竞争
3. 测试框架的统计采样时机问题

**当前状态**:
- 简单的单向测试: ✅ 全部通过
- 优先级冲突测试: ✅ 通过
- 复杂双向测试: ⚠️ 部分失败

**建议**:
1. 简化测试场景,每个测试只验证一个明确的行为
2. 添加更详细的调试日志 (eBPF `bpf_printk`)
3. 分析会话缓存对双向流量的影响

---

## 4. 改进建议

### 4.1 高优先级改进 (P0)

#### 4.1.1 引入 LPM Trie Map 支持 CIDR

**目标**: 提升 CIDR 策略的查找性能从 O(n) 到 O(log n)

**实施方案**:

1. **创建 CIDR 策略 Map**:
   ```c
   struct {
       __uint(type, BPF_MAP_TYPE_LPM_TRIE);
       __type(key, struct cidr_key);
       __type(value, struct policy_value);
       __uint(max_entries, 10000);
       __uint(map_flags, BPF_F_NO_PREALLOC);
   } cidr_policy_map SEC(".maps");

   struct cidr_key {
       __u32 prefixlen;  // CIDR 前缀长度
       __u32 ip;         // IP 地址
       __u16 port;
       __u8  direction;
       __u8  protocol;
   };
   ```

2. **修改策略路由逻辑**:
   ```go
   func (pm *PolicyManager) addPolicyToMap(p *Policy) error {
       if isCIDR(p.SrcIP) || isCIDR(p.DstIP) {
           return pm.addCIDRPolicy(p)  // → LPM Trie
       } else if hasWildcard(p) {
           return pm.addWildcardPolicy(p)  // → Array Map
       }
       return pm.addExactPolicy(p)  // → Hash Map
   }
   ```

3. **查找顺序**:
   ```
   精确匹配 (Hash) → CIDR 匹配 (LPM Trie) → 通配符匹配 (Array) → 默认
   ```

**预期收益**:
- CIDR 策略查找: O(n) → O(log n)
- 支持更复杂的网络拓扑
- 与 Cilium 对齐

#### 4.1.2 优先级冲突检测

**目标**: 防止优先级相同导致的不确定行为

**实施方案**:

1. **创建策略时检查**:
   ```go
   func (pm *PolicyManager) AddPolicy(p *Policy) error {
       // 检查是否有相同优先级的冲突策略
       conflicts := pm.findConflicts(p)
       if len(conflicts) > 0 {
           return fmt.Errorf("priority conflict: %v", conflicts)
       }
       // ...
   }
   ```

2. **Tie-breaker 机制**:
   ```c
   // eBPF 侧: 使用 Rule ID 作为 tie-breaker
   if (wildcard->priority > best_priority ||
       (wildcard->priority == best_priority &&
        wildcard->rule_id < best_rule_id)) {
       best_match = wildcard;
       best_priority = wildcard->priority;
       best_rule_id = wildcard->rule_id;
   }
   ```

3. **自动分配优先级**:
   ```go
   // 如果用户不指定优先级,根据策略具体程度自动分配
   func (p *Policy) AutoPriority() uint16 {
       priority := uint16(100)
       if p.SrcPort != 0 { priority += 20 }    // 具体端口 +20
       if p.DstPort != 0 { priority += 20 }
       if !isCIDR(p.SrcIP) { priority += 10 }  // 精确 IP +10
       if !isCIDR(p.DstIP) { priority += 10 }
       return priority
   }
   ```

#### 4.1.3 结构体一致性验证

**目标**: 防止跨语言结构体对齐问题

**实施方案**:

1. **编译时检查**:
   ```go
   // pkg/policy/layout_test.go
   func TestWildcardPolicyLayout(t *testing.T) {
       const expectedSize = 32  // 与 eBPF 结构体大小一致

       wp := wildcardPolicyStruct{}
       actualSize := unsafe.Sizeof(wp)

       assert.Equal(t, expectedSize, actualSize,
           "Structure size mismatch with eBPF")

       // 验证字段偏移
       assert.Equal(t, 20, offsetOf(wp.Protocol))
       assert.Equal(t, 21, offsetOf(wp.Action))
       assert.Equal(t, 22, offsetOf(wp.LogEnabled))
       assert.Equal(t, 23, offsetOf(wp.Direction))
   }
   ```

2. **代码生成工具**:
   ```bash
   # 从 eBPF 头文件自动生成 Go 结构体
   $ bpf2go -cc clang -type wildcard_policy \
       wildcard_policy src/bpf/headers/common_types.h

   # 生成 wildcard_policy_bpfel.go (小端)
   # 确保完全一致
   ```

3. **运行时验证**:
   ```go
   func (pm *PolicyManager) verifyMapLayout() error {
       // 写入测试策略,读回验证
       testPolicy := createTestPolicy()
       pm.writeToMap(testPolicy)

       readBack := pm.readFromMap(testPolicy.ID)
       if !reflect.DeepEqual(testPolicy, readBack) {
           return fmt.Errorf("map layout mismatch")
       }
       return nil
   }
   ```

### 4.2 中优先级改进 (P1)

#### 4.2.1 多级策略 Map

**目标**: 减少通配符策略线性扫描的开销

**实施方案**:

1. **按通配符类型分组**:
   ```
   exact_policy_map        // 完全精确 (6-tuple)
   src_port_wildcard_map   // 仅 src_port=0
   dst_port_wildcard_map   // 仅 dst_port=0
   both_port_wildcard_map  // 两个端口都是 0
   full_wildcard_map       // 完全通配符
   ```

2. **查找顺序优化**:
   ```c
   // 1. 精确匹配
   policy = lookup_exact_policy(key, direction);
   if (policy) return policy->action;

   // 2. 部分通配符 (Hash 查找)
   policy = lookup_partial_wildcard(key, direction);
   if (policy) return policy->action;

   // 3. 完全通配符 (线性扫描,但数量少)
   policy = lookup_full_wildcard(key, direction);
   if (policy) return policy->action;

   // 4. 默认
   return POLICY_ACTION_ALLOW;
   ```

3. **性能预期**:
   - 大部分策略进入 Hash Map: O(1)
   - 少量完全通配符策略: O(k),k << n

#### 4.2.2 Per-Direction 策略分离

**目标**: 减少策略查找时的无效扫描

**实施方案**:

1. **分离 Ingress 和 Egress 策略**:
   ```c
   struct {
       __uint(type, BPF_MAP_TYPE_ARRAY);
       __type(key, __u32);
       __type(value, struct wildcard_policy);
       __uint(max_entries, 100);
   } ingress_wildcard_map SEC(".maps");

   struct {
       __uint(type, BPF_MAP_TYPE_ARRAY);
       __type(key, __u32);
       __type(value, struct wildcard_policy);
       __uint(max_entries, 100);
   } egress_wildcard_map SEC(".maps");
   ```

2. **方向感知查找**:
   ```c
   struct bpf_map *policy_map = (direction == POLICY_DIR_INGRESS)
       ? &ingress_wildcard_map
       : &egress_wildcard_map;

   // 只扫描相关方向的策略,减少一半扫描量
   ```

3. **收益**:
   - 扫描量减少 50% (只扫描单向策略)
   - 缓存局部性更好

#### 4.2.3 统计增强

**目标**: 提供更详细的策略匹配统计,便于调优

**新增统计**:
```go
type Statistics struct {
    // 现有统计 ...

    // 策略匹配统计
    ExactMatchHits      uint64  // 精确匹配命中次数
    WildcardMatchHits   uint64  // 通配符匹配命中次数
    CIDRMatchHits       uint64  // CIDR 匹配命中次数
    DefaultPolicyHits   uint64  // 默认策略命中次数

    // 性能统计
    AvgLookupTime       uint64  // 平均查找时间 (纳秒)
    MaxLookupTime       uint64  // 最大查找时间
    WildcardScanCount   uint64  // 通配符扫描次数
}
```

**用途**:
- 识别性能瓶颈
- 优化策略配置
- 监控和告警

### 4.3 低优先级改进 (P2)

#### 4.3.1 Identity-Based Security (长期)

**目标**: 向 Cilium 架构演进,支持基于身份的策略

**概念验证**:
```
Pod Labels → Identity → Policy
```

**优势**:
- 解耦 IP 地址
- 支持 Kubernetes 动态环境
- 更高的可扩展性

**实施挑战**:
- 需要与 Kubernetes API 集成
- 需要 Identity 分配和同步机制
- 大规模重构

#### 4.3.2 L7 策略支持 (长期)

**目标**: 支持应用层协议感知策略

**可能方案**:
1. 集成 Envoy Proxy (类似 Cilium)
2. eBPF 中简单的协议解析 (HTTP method, path)
3. 使用 eBPF sockops 重定向到用户空间

**应用场景**:
- HTTP 路径级别的访问控制
- gRPC 方法级别的策略
- API 网关功能

#### 4.3.3 策略模拟和验证工具

**目标**: 在应用策略前验证影响范围

**功能**:
```bash
# 模拟策略效果
$ policy-simulator --policy new_policy.yaml --traffic traffic_dump.pcap
Matched: 150 packets (75 ALLOW, 75 DENY)
Unmatched: 50 packets (default ALLOW)

# 检测策略冲突
$ policy-validator --policies policies/*.yaml
Warning: Priority conflict between policy_A and policy_B
Error: Unreachable policy detected: policy_C (shadowed by policy_D)
```

---

## 5. 实施优先级

### 5.1 立即实施 (本周)

1. ✅ **结构体一致性验证** (已通过测试验证)
2. 🔄 **优先级冲突检测** (正在实施)
3. 📝 **文档化当前行为** (本文档)

### 5.2 短期实施 (1-2 周)

1. **LPM Trie 支持** (提升 CIDR 性能)
2. **多级策略 Map** (减少线性扫描)
3. **统计增强** (性能监控)

### 5.3 中期实施 (1 个月)

1. **Per-Direction 分离** (优化查找)
2. **策略验证工具** (提升可靠性)
3. **完善测试覆盖** (双向策略场景)

### 5.4 长期规划 (3-6 个月)

1. **Identity-Based Security** (架构演进)
2. **L7 策略支持** (功能扩展)
3. **性能基准测试** (与 Cilium 对比)

---

## 6. 总结

### 6.1 当前实现的优势

✅ **简单直观**: 两层查找架构易于理解和维护
✅ **方向感知**: 正确支持 Ingress/Egress 策略
✅ **优先级机制**: 基本的优先级处理正常工作
✅ **动态更新**: 支持运行时策略添加/删除

### 6.2 主要差距

🔴 **可扩展性**: 通配符策略限制 100 条,无法满足大规模需求
🔴 **CIDR 性能**: 线性扫描 O(n) vs Cilium 的 LPM Trie O(log n)
🟡 **策略路由**: 粗粒度的通配符判断,未充分利用 Hash Map
🟡 **功能缺失**: 缺少 L7 支持和身份抽象

### 6.3 改进路线图

```
Phase 1 (当前): L3/L4 基础策略引擎 ✅
    └─ 支持 Ingress/Egress 方向
    └─ 精确匹配 + 通配符匹配
    └─ 优先级处理

Phase 2 (短期): 性能优化 🔄
    └─ LPM Trie 支持 CIDR
    └─ 多级策略 Map
    └─ 优先级冲突检测

Phase 3 (中期): 可扩展性增强 📋
    └─ Per-Direction 分离
    └─ 策略验证工具
    └─ 完善监控统计

Phase 4 (长期): 架构演进 🎯
    └─ Identity-Based Security
    └─ L7 策略支持
    └─ 与 Cilium 性能对标
```

### 6.4 建议的下一步行动

**立即行动**:
1. ✅ 完成 Phase 4 集成测试文档
2. 🔄 实施优先级冲突检测机制
3. 📝 编写策略引擎开发者指南

**近期规划**:
1. 设计并实施 LPM Trie CIDR 支持
2. 创建性能基准测试套件
3. 简化和修复复杂双向策略测试

---

## 附录

### A. 参考资料

1. **Cilium 文档**:
   - BPF Architecture: https://docs.cilium.io/en/latest/bpf/architecture/
   - Policy Maps: https://pkg.go.dev/github.com/cilium/cilium/pkg/maps/policymap

2. **eBPF 文档**:
   - LPM Trie: https://docs.ebpf.io/linux/map-type/BPF_MAP_TYPE_LPM_TRIE/
   - Hash Maps: https://docs.ebpf.io/linux/map-type/BPF_MAP_TYPE_HASH/

3. **内部文档**:
   - [bugfix-wildcard-policy-struct-alignment.md](bugfix-wildcard-policy-struct-alignment.md)
   - [phase4-integration-testing-summary.md](phase4-integration-testing-summary.md)

### B. 测试结果摘要

| 测试类型 | 通过率 | 备注 |
|---------|--------|------|
| 简单方向测试 | 100% (2/2) | ✅ Ingress/Egress DENY |
| Egress 出站测试 | 100% (3/3) | ✅ DENY/ALLOW/Default |
| 优先级冲突测试 | 100% (1/1) | ✅ 高优先级覆盖低优先级 |
| 双向策略测试 | 25% (1/4) | ⚠️ 复杂场景需简化 |
| **总计** | **70% (7/10)** | 核心功能正常 |

### C. 性能估算

**当前实现** (理论分析):
```
精确匹配:     O(1)     - Hash 查找
通配符匹配:   O(n)     - 最多扫描 100 条策略
CIDR 匹配:    O(n)     - 每条策略都要掩码计算
默认策略:     O(1)     - 常量时间

平均延迟:     ~500ns   - 1000ns (估算)
最坏延迟:     ~5us     - 10us (全扫描)
```

**改进后** (预期):
```
精确匹配:     O(1)     - Hash 查找
CIDR 匹配:    O(log n) - LPM Trie
通配符匹配:   O(k)     - k < n/10 (分层后)

平均延迟:     ~300ns   - 500ns (优化 40%)
最坏延迟:     ~2us     - 3us (优化 70%)
```

---

**文档版本**: 1.0
**最后更新**: 2025-11-12
**作者**: eBPF Microsegment Team
**状态**: ✅ 完成
