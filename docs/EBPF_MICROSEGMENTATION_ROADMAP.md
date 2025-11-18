# eBPF 微隔离核心功能开发路线图

**文档版本**: v1.0
**创建日期**: 2025-11-18
**预计总工期**: 15-20 工作日
**当前完成度**: 约 60-65%

---

## 📋 目录

1. [当前实现状态](#当前实现状态)
2. [缺失功能分析](#缺失功能分析)
3. [开发优先级](#开发优先级)
4. [详细开发计划](#详细开发计划)
5. [技术挑战和解决方案](#技术挑战和解决方案)
6. [性能目标](#性能目标)
7. [测试策略](#测试策略)

---

## 当前实现状态

### ✅ 已完成的核心功能

| 功能模块 | 完成度 | 说明 |
|---------|--------|------|
| **基础数据平面** | 100% | TC + XDP 双模式支持 |
| **会话跟踪** | 100% | LRU_HASH 自动淘汰，100K+ 会话 |
| **5元组匹配** | 100% | 精确匹配 + 通配符匹配 |
| **TCP 状态机** | 100% | 完整 TCP 状态跟踪 |
| **IPv4/IPv6 支持** | 100% | 统一 128 位地址格式 |
| **VLAN 支持** | 100% | 802.1Q/802.1ad 标签识别 |
| **协议索引优化** | 100% | 支持 200+ 通配符策略 |
| **会话超时** | 100% | 自动清理超时会话 |
| **Flow 事件上报** | 100% | Ring Buffer 实时推送 |
| **统计收集** | 100% | Per-CPU 无锁统计 |
| **增强 TCP 跟踪** | 100% | 序列号、ACK、窗口大小 |
| **策略方向控制** | 100% | Ingress/Egress/Both |

### 🔨 部分实现的功能

| 功能模块 | 完成度 | 缺失部分 |
|---------|--------|----------|
| **协议支持** | 80% | ICMP 仅基础支持，缺少类型过滤 |
| **连接跟踪** | 70% | 缺少重传检测逻辑、窗口管理 |
| **策略匹配** | 85% | 缺少优先级冲突解决、默认拒绝 |
| **数据包处理** | 60% | 缺少分片处理、大包处理 |

### ❌ 未实现的关键功能

| 功能类别 | 优先级 | 功能清单 |
|---------|--------|----------|
| **核心隔离** | P0 | NAT 支持、分片处理、默认拒绝 |
| **高级功能** | P1 | 应用层协议识别、进程关联、限流 |
| **增强功能** | P2 | 多租户、加密识别、DDoS 防护 |

---

## 缺失功能分析

### P0 优先级 - 核心微隔离功能（必须）

#### 1. NAT 支持和检测 🔴
**当前问题**:
- 无法正确处理 NAT 环境下的流量
- SNAT/DNAT 会导致策略匹配失败
- Docker/K8s 环境大量使用 NAT

**需要实现**:
- Conntrack 集成（读取内核连接跟踪表）
- NAT 状态检测和标记
- 原始地址恢复逻辑
- NAT 前/后策略匹配选项

**技术难点**:
- eBPF 无法直接访问 conntrack
- 需要通过 BPF helper 或用户态辅助
- 性能影响需控制

**预计工期**: 3-4 天

---

#### 2. 分片数据包处理 🔴
**当前问题**:
- 分片数据包无法正确解析 5 元组
- 后续分片缺少端口信息
- 可能被用于绕过策略

**需要实现**:
- IP 分片重组跟踪
- 分片状态 Map（记录首个分片信息）
- 后续分片关联到首个分片策略
- 分片超时清理

**数据结构**:
```c
struct frag_key {
    __u32 src_ip[4];
    __u32 dst_ip[4];
    __u32 frag_id;      // IPv4: identification, IPv6: fragment ID
    __u8  protocol;
    __u8  ip_version;
};

struct frag_value {
    struct flow_key complete_key;  // 完整的 5 元组
    __u64 timestamp;               // 分片到达时间
    __u8  policy_action;           // 首个分片匹配的策略动作
};
```

**预计工期**: 3-4 天

---

#### 3. 默认拒绝策略（Default Deny）🔴
**当前问题**:
- 无明确策略时行为不确定
- 不符合零信任原则
- 缺少全局策略配置

**需要实现**:
- 全局默认动作配置 Map
- 策略未匹配时的兜底逻辑
- 分级默认策略（全局/Per-Interface/Per-Workload）
- 默认策略审计日志

**配置示例**:
```c
enum default_policy {
    DEFAULT_ALLOW = 0,   // 宽松模式（学习阶段）
    DEFAULT_DENY = 1,    // 严格模式（生产环境）
    DEFAULT_LOG = 2,     // 仅记录（监控模式）
};

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, __u8);  // enum default_policy
} default_policy_map SEC(".maps");
```

**预计工期**: 1-2 天

---

#### 4. ICMP 协议增强支持 🔴
**当前问题**:
- ICMP 仅基础支持
- 无法按类型/代码过滤（如仅允许 ping）
- 缺少 ICMP 错误消息关联

**需要实现**:
- ICMP 类型和代码解析
- ICMP 错误消息的内嵌 IP 头解析
- ICMP 与关联会话的映射
- ICMPv6 特殊消息支持（ND/MLD）

**策略扩展**:
```c
struct icmp_policy {
    __u8 icmp_type;        // ICMP 类型（0 = Echo Reply, 8 = Echo Request）
    __u8 icmp_code;        // ICMP 代码（0 = any）
    __u8 action;           // ALLOW/DENY
    __u8 pad;
};

// 添加到 wildcard_policy
struct wildcard_policy {
    // ... existing fields
    __u8  icmp_type;       // 0 = any type
    __u8  icmp_code;       // 0 = any code
};
```

**预计工期**: 2 天

---

#### 5. 连接跟踪增强 🟠
**当前问题**:
- TCP 重传检测未实现（数据结构已预留）
- TCP 窗口缩放未处理
- 乱序包处理不完善

**需要实现**:
- TCP 序列号跟踪和验证
- 重传检测逻辑
- 窗口缩放因子存储
- 乱序数据包容忍度配置

**实现方案**:
```c
// 在 session_value 中利用已有字段
__u32 tcp_seq_client;     // ✅ 已有
__u32 tcp_seq_server;     // ✅ 已有
__u32 tcp_ack_client;     // ✅ 已有
__u32 tcp_ack_server;     // ✅ 已有
__u8  tcp_retrans_count;  // ✅ 已有

// 新增逻辑
static __always_inline bool is_tcp_retransmission(
    struct session_value *session,
    __u32 seq, __u32 ack, bool to_server)
{
    if (to_server) {
        // Check if seq <= last_seq and not a keep-alive
        return (seq < session->tcp_seq_client) && (ack == session->tcp_ack_client);
    } else {
        return (seq < session->tcp_seq_server) && (ack == session->tcp_ack_server);
    }
}
```

**预计工期**: 2-3 天

---

### P1 优先级 - 高级功能（重要）

#### 6. 应用层协议识别 🟡
**业务价值**:
- 基于协议的细粒度策略（如仅允许 HTTPS，拒绝 HTTP）
- 检测协议滥用（SSH 隧道、DNS 隧道）
- 增强可视化（拓扑图显示协议）

**需要实现**:
- 常见协议特征识别（HTTP/HTTPS/DNS/SSH/MySQL）
- DPI（Deep Packet Inspection）基础框架
- 协议状态机（如 HTTP 请求/响应）
- 协议匹配结果存储到 session

**技术方案**:
```c
// 协议识别结果
enum app_protocol {
    APP_PROTO_UNKNOWN = 0,
    APP_PROTO_HTTP = 1,
    APP_PROTO_HTTPS = 2,
    APP_PROTO_DNS = 3,
    APP_PROTO_SSH = 4,
    APP_PROTO_MYSQL = 5,
    // ...
};

// 添加到 session_value
struct session_value {
    // ... existing fields
    __u8  app_protocol;    // 应用层协议
    __u8  proto_confidence; // 识别置信度 (0-100)
};

// HTTP 特征检测示例
static __always_inline bool is_http_traffic(
    void *data, void *data_end, __u16 dport)
{
    // Check common HTTP ports
    if (dport != 80 && dport != 8080) return false;

    // Check for HTTP methods (GET, POST, etc.)
    if (data + 4 > data_end) return false;

    char *payload = (char *)data;
    return (payload[0] == 'G' && payload[1] == 'E' &&
            payload[2] == 'T' && payload[3] == ' ') ||
           (payload[0] == 'P' && payload[1] == 'O' &&
            payload[2] == 'S' && payload[3] == 'T');
}
```

**预计工期**: 4-5 天

---

#### 7. 进程级别策略关联 🟡
**业务价值**:
- 容器级别隔离（Kubernetes Pod 隔离）
- 进程级别策略（仅允许特定进程访问）
- 与 cgroup 集成

**需要实现**:
- Socket-level eBPF hook（cgroup/sock_ops）
- 进程 PID/cgroup ID 提取
- 进程元数据存储（PID -> 容器/命名空间）
- 策略匹配时的 cgroup 过滤

**技术方案**:
```c
// 使用 cgroup/sock_ops 或 cgroup/connect4 hook
SEC("cgroup/connect4")
int cgroup_connect4(struct bpf_sock_addr *ctx)
{
    __u64 cgroup_id = bpf_get_current_cgroup_id();
    __u32 pid = bpf_get_current_pid_tgid() >> 32;

    // Store cgroup ID in session metadata
    // Associate with existing flow_key
}

// 扩展 session_value
struct session_value {
    // ... existing fields
    __u64 cgroup_id;       // Cgroup ID（容器标识）
    __u32 pid;             // 发起进程 PID
};

// 扩展 wildcard_policy 支持 cgroup 过滤
struct wildcard_policy {
    // ... existing fields
    __u64 cgroup_id;       // 0 = any cgroup
};
```

**预计工期**: 3-4 天

---

#### 8. 带宽限流（Rate Limiting）🟡
**业务价值**:
- 防止流量洪水攻击
- 保护后端服务
- 公平资源分配

**需要实现**:
- Token Bucket 算法（Per-Flow/Per-IP）
- 动态速率调整
- 突发流量容忍（Burst）
- 超限处理（Drop/Mark/Log）

**技术方案**:
```c
struct rate_limit_config {
    __u64 rate_bps;        // 速率限制（字节/秒）
    __u64 burst_bytes;     // 突发容量
    __u64 last_refill_ts;  // 上次填充时间
    __u64 tokens;          // 当前令牌数
};

// Per-Flow rate limiting map
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 10000);
    __type(key, struct flow_key);
    __type(value, struct rate_limit_config);
} rate_limit_map SEC(".maps");

// Token bucket algorithm
static __always_inline bool check_rate_limit(
    struct rate_limit_config *rl,
    __u64 now_ns,
    __u32 packet_bytes)
{
    // Refill tokens based on elapsed time
    __u64 elapsed_ns = now_ns - rl->last_refill_ts;
    __u64 new_tokens = (elapsed_ns * rl->rate_bps) / 1000000000ULL;

    rl->tokens += new_tokens;
    if (rl->tokens > rl->burst_bytes) {
        rl->tokens = rl->burst_bytes;
    }
    rl->last_refill_ts = now_ns;

    // Check if enough tokens
    if (rl->tokens >= packet_bytes) {
        rl->tokens -= packet_bytes;
        return true;  // ALLOW
    }
    return false;  // DROP (rate limited)
}
```

**预计工期**: 3 天

---

#### 9. 连接数限制（Connection Limiting）🟡
**业务价值**:
- 防止连接洪水攻击
- 保护服务端资源
- 检测扫描行为

**需要实现**:
- Per-IP 连接计数
- Per-Service 连接计数
- 时间窗口内的新建连接速率限制
- 超限响应（REJECT/DROP/LOG）

**技术方案**:
```c
struct conn_limit_key {
    __u32 ip[4];           // 源 IP
    __u16 dst_port;        // 目标端口（服务）
    __u8  protocol;
    __u8  pad;
};

struct conn_limit_value {
    __u32 active_conns;    // 当前活跃连接数
    __u32 new_conns_count; // 时间窗口内新建连接数
    __u64 window_start_ts; // 时间窗口起始时间
};

// Connection limit map
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 50000);
    __type(key, struct conn_limit_key);
    __type(value, struct conn_limit_value);
} conn_limit_map SEC(".maps");

// 配置 Map
struct conn_limit_config {
    __u32 max_conns_per_ip;      // 每IP最大连接数
    __u32 max_new_conns_per_sec; // 每秒最大新建连接数
};
```

**预计工期**: 2 天

---

#### 10. 策略模拟模式（Dry-run）🟡
**业务价值**:
- 测试策略影响而不实际执行
- 策略变更前的验证
- 生成 "What-if" 报告

**需要实现**:
- Per-Policy 模拟标志
- 模拟模式下仅记录，不执行动作
- 模拟结果统计（如果生效会 ALLOW/DENY 多少流）
- 用户态 API 查询模拟结果

**技术方案**:
```c
// 扩展 policy_value 和 wildcard_policy
struct policy_value {
    // ... existing fields
    __u8  simulate_mode;   // 1 = 仅模拟，不执行
};

// 模拟统计 Map
struct simulate_stats {
    __u64 would_allow;     // 如果生效会允许的流数
    __u64 would_deny;      // 如果生效会拒绝的流数
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, __u32);    // rule_id
    __type(value, struct simulate_stats);
} simulate_stats_map SEC(".maps");

// 策略匹配逻辑修改
if (policy->simulate_mode) {
    // 仅记录统计，返回 ALLOW（放行）
    struct simulate_stats *stats = bpf_map_lookup_elem(&simulate_stats_map, &policy->rule_id);
    if (stats) {
        if (policy->action == POLICY_ACTION_ALLOW) {
            __sync_fetch_and_add(&stats->would_allow, 1);
        } else {
            __sync_fetch_and_add(&stats->would_deny, 1);
        }
    }
    return POLICY_ACTION_ALLOW;
}
```

**预计工期**: 2 天

---

### P2 优先级 - 增强功能（可选）

#### 11. 多租户隔离 🟢
**业务价值**:
- 支持多客户共享同一套系统
- 数据隔离和安全性
- 策略命名空间

**需要实现**:
- Tenant ID 字段
- Per-Tenant 策略 Map
- 租户级别资源配额
- 租户间流量隔离

**预计工期**: 4-5 天

---

#### 12. DDoS 防护基础功能 🟢
**业务价值**:
- 检测和缓解 SYN Flood
- 检测和缓解 UDP Flood
- 异常流量模式识别

**需要实现**:
- SYN Cookie 支持（内核集成）
- 流量速率异常检测
- 自动黑名单机制
- 告警和自动响应

**预计工期**: 5-6 天

---

#### 13. 加密流量检测 🟢
**业务价值**:
- 识别 TLS/SSL 流量（不解密）
- 检测加密隧道（VPN/Tor）
- 策略基于加密状态

**需要实现**:
- TLS Client Hello 解析（SNI 提取）
- IPSec 标识
- 加密协议特征库

**预计工期**: 3-4 天

---

## 开发优先级

### 优先级 P0 (1-2 周)
**核心微隔离功能，必须完成**

```
Week 1-2: P0 核心功能
├─ Day 1-4:   NAT 支持和检测        ████████████████
├─ Day 5-8:   分片数据包处理        ████████████████
├─ Day 9-10:  默认拒绝策略          ████████
├─ Day 11-12: ICMP 协议增强         ████████
└─ Day 13-15: 连接跟踪增强          ████████████
```

**关键产出**:
- NAT 环境下正确匹配策略
- 分片攻击防护
- 零信任默认拒绝
- 完整 ICMP 支持
- TCP 重传检测

---

### 优先级 P1 (2-3 周)
**高级功能，显著提升产品竞争力**

```
Week 3-5: P1 高级功能
├─ Day 1-5:   应用层协议识别        ████████████████████
├─ Day 6-9:   进程级别策略关联      ████████████████
├─ Day 10-12: 带宽限流              ████████████
├─ Day 13-14: 连接数限制            ████████
└─ Day 15-16: 策略模拟模式          ████████
```

**关键产出**:
- HTTP/DNS/SSH 协议识别
- 容器级别隔离
- DoS/DDoS 基础防护
- 策略验证工具

---

### 优先级 P2 (3-4 周)
**增强功能，长期规划**

```
Week 6-8: P2 增强功能（可选）
├─ Week 6:    多租户隔离            ████████████████████
├─ Week 7:    DDoS 防护增强         ████████████████████████
└─ Week 8:    加密流量检测          ████████████████
```

---

## 详细开发计划

### Phase 1: P0 核心功能 (10-12 天)

#### Day 1-4: NAT 支持和检测

**任务 1.1: Conntrack 集成方案设计**
- [ ] 研究 eBPF 访问 conntrack 的方法
  - BPF helper: `bpf_ct_lookup_tcp/udp()`（Kernel >= 5.18）
  - 用户态辅助：通过 netlink 同步 conntrack 状态
- [ ] 设计 conntrack 状态缓存 Map
- [ ] 定义 NAT 检测逻辑

**任务 1.2: 实现 NAT 检测逻辑**
- [ ] 创建 `src/bpf/headers/nat_support.h`
- [ ] 实现 conntrack 查询函数
  ```c
  static __always_inline bool lookup_conntrack(
      struct flow_key *key,
      struct flow_key *original_key)
  {
      // 使用 bpf_ct_lookup_tcp/udp
      // 或查询用户态同步的 conntrack map
  }
  ```
- [ ] 在策略匹配前调用 NAT 检测
- [ ] 更新 session_value 存储原始地址

**任务 1.3: NAT 配置和测试**
- [ ] 添加 NAT 模式配置选项
  - `MATCH_ORIGINAL`: 匹配 NAT 前地址
  - `MATCH_TRANSLATED`: 匹配 NAT 后地址
- [ ] Docker NAT 环境测试
- [ ] Kubernetes NodePort/ClusterIP 测试
- [ ] 性能测试（NAT 查询开销）

**验收标准**:
- [x] Docker 容器间通信策略正常
- [x] Kubernetes Service 访问策略正常
- [x] NAT 检测延迟 < 5μs
- [x] 单元测试覆盖 NAT 场景

---

#### Day 5-8: 分片数据包处理

**任务 1.4: 分片跟踪 Map 设计**
- [ ] 定义 frag_key 和 frag_value 结构
- [ ] 创建 `fragment_track_map`（LRU_HASH）
- [ ] 实现分片超时清理（用户态定期扫描）

**任务 1.5: 分片识别和处理**
- [ ] 创建 `src/bpf/headers/fragment_handling.h`
- [ ] IPv4 分片检测（MF 标志、Fragment Offset）
- [ ] IPv6 分片检测（Fragment Extension Header）
- [ ] 首个分片存储完整 5 元组
- [ ] 后续分片关联到首个分片

**任务 1.6: 分片策略匹配**
```c
static __always_inline __u8 handle_fragment(
    struct __sk_buff *skb,
    struct flow_key *key,
    bool is_first_fragment,
    __u32 frag_id)
{
    if (is_first_fragment) {
        // 正常策略匹配
        __u8 action = lookup_policy_action(key, ...);

        // 存储到 fragment_track_map
        struct frag_key fkey = {
            .src_ip = key->src_ip,
            .dst_ip = key->dst_ip,
            .frag_id = frag_id,
            .protocol = key->protocol,
        };
        struct frag_value fval = {
            .complete_key = *key,
            .timestamp = get_timestamp_ns(),
            .policy_action = action,
        };
        bpf_map_update_elem(&fragment_track_map, &fkey, &fval, BPF_ANY);

        return action;
    } else {
        // 后续分片：查找首个分片的策略
        struct frag_key fkey = {...};
        struct frag_value *fval = bpf_map_lookup_elem(&fragment_track_map, &fkey);
        if (fval) {
            return fval->policy_action;
        }
        // 未找到：使用默认策略
        return default_action;
    }
}
```

**任务 1.7: 测试和性能调优**
- [ ] 生成分片测试包（IPv4/IPv6）
- [ ] 测试分片重组攻击防护
- [ ] 性能测试（分片处理开销）

**验收标准**:
- [x] 正确处理 IPv4/IPv6 分片
- [x] 后续分片关联到首个分片策略
- [x] 分片处理延迟 < 5μs
- [x] 防护分片重组攻击

---

#### Day 9-10: 默认拒绝策略

**任务 1.8: 默认策略 Map 和配置**
- [ ] 创建 `default_policy_map`
  ```c
  struct {
      __uint(type, BPF_MAP_TYPE_ARRAY);
      __uint(max_entries, 1);
      __type(key, __u32);
      __type(value, __u8);  // enum default_policy
  } default_policy_map SEC(".maps");
  ```
- [ ] 用户态 API 配置默认策略
  ```go
  func (dp *DataPlane) SetDefaultPolicy(action PolicyAction) error
  ```

**任务 1.9: 默认策略逻辑集成**
- [ ] 修改策略匹配失败时的处理
  ```c
  // If no policy matched
  __u8 default_action = POLICY_ACTION_DENY;
  __u32 key = 0;
  __u8 *config = bpf_map_lookup_elem(&default_policy_map, &key);
  if (config) {
      default_action = *config;
  }

  if (default_action == POLICY_ACTION_DENY) {
      update_stats(STATS_DENIED_PACKETS);
      push_flow_event(..., FLOW_EVENT_DENIED);
      return TC_ACT_SHOT;
  }
  ```
- [ ] 添加默认策略审计日志

**任务 1.10: 测试和验证**
- [ ] 测试默认 ALLOW 模式
- [ ] 测试默认 DENY 模式
- [ ] 测试默认 LOG 模式
- [ ] E2E 测试（无策略时的行为）

**验收标准**:
- [x] 可配置全局默认策略
- [x] 默认拒绝时正确记录日志
- [x] 零信任模式（Default Deny）工作正常

---

#### Day 11-12: ICMP 协议增强

**任务 1.11: ICMP 类型/代码解析**
- [ ] 创建 `src/bpf/headers/icmp_support.h`
- [ ] ICMP/ICMPv6 头解析函数
  ```c
  static __always_inline bool parse_icmp(
      void *data, void *data_end,
      __u8 *icmp_type, __u8 *icmp_code)
  {
      struct icmphdr *icmp = data;
      if ((void *)(icmp + 1) > data_end) return false;
      *icmp_type = icmp->type;
      *icmp_code = icmp->code;
      return true;
  }
  ```

**任务 1.12: ICMP 策略扩展**
- [ ] 扩展 wildcard_policy 支持 ICMP 类型/代码
- [ ] 实现 ICMP 策略匹配逻辑
- [ ] 常见 ICMP 类型预定义
  ```c
  #define ICMP_TYPE_ECHO_REPLY      0
  #define ICMP_TYPE_ECHO_REQUEST    8
  #define ICMP_TYPE_DEST_UNREACH    3
  #define ICMP_TYPE_TIME_EXCEEDED   11
  ```

**任务 1.13: ICMP 错误消息关联**
- [ ] 解析 ICMP 错误消息的内嵌 IP 头
- [ ] 关联到原始会话
- [ ] 允许关联会话的 ICMP 错误

**任务 1.14: 测试**
- [ ] Ping 测试（Echo Request/Reply）
- [ ] Path MTU Discovery 测试
- [ ] Traceroute 测试
- [ ] ICMPv6 ND 测试

**验收标准**:
- [x] 支持按 ICMP 类型/代码过滤
- [x] ICMP 错误消息正确关联
- [x] ICMPv6 特殊消息正常工作

---

#### Day 13-15: 连接跟踪增强

**任务 1.15: TCP 重传检测**
- [ ] 实现 `is_tcp_retransmission()` 函数
- [ ] 更新 session->tcp_retrans_count
- [ ] 设置 CONN_FLAG_RETRANS 标志
- [ ] 记录重传统计

**任务 1.16: TCP 窗口管理**
- [ ] 解析 TCP Window Scale 选项（SYN 包）
- [ ] 存储窗口缩放因子
- [ ] 计算实际窗口大小
- [ ] 检测窗口满（Zero Window）

**任务 1.17: 乱序包处理**
- [ ] 定义乱序容忍度配置
- [ ] 验证序列号在合理范围内
- [ ] 记录乱序包统计

**任务 1.18: 测试和验证**
- [ ] 模拟丢包导致的重传
- [ ] 窗口缩放测试
- [ ] 乱序包测试
- [ ] 性能测试

**验收标准**:
- [x] 正确检测 TCP 重传
- [x] 窗口管理准确
- [x] 乱序包不影响会话跟踪

---

### Phase 2: P1 高级功能 (16 天)

#### Day 1-5: 应用层协议识别

**详细任务列表省略，参考上文"应用层协议识别"章节**

#### Day 6-9: 进程级别策略关联

**详细任务列表省略，参考上文"进程级别策略关联"章节**

#### Day 10-12: 带宽限流

**详细任务列表省略，参考上文"带宽限流"章节**

#### Day 13-14: 连接数限制

**详细任务列表省略，参考上文"连接数限制"章节**

#### Day 15-16: 策略模拟模式

**详细任务列表省略，参考上文"策略模拟模式"章节**

---

## 技术挑战和解决方案

### 挑战 1: NAT 环境下的连接跟踪

**问题**:
- eBPF 程序无法直接访问内核 conntrack 表
- 需要额外性能开销

**解决方案**:
1. **BPF Helper 方式**（Kernel >= 5.18）:
   - 使用 `bpf_ct_lookup_tcp/udp()` helper
   - 优点：内核原生支持，性能好
   - 缺点：内核版本要求高

2. **用户态同步方式**（兼容性好）:
   - 用户态程序监听 conntrack events
   - 同步到 eBPF Map
   - 优点：兼容低版本内核
   - 缺点：额外内存开销

**选择**: 优先使用 BPF Helper，兼容模式提供用户态同步

---

### 挑战 2: 分片处理的性能影响

**问题**:
- 分片跟踪需要额外 Map 查询
- 分片重组攻击风险

**解决方案**:
1. **LRU Map 自动淘汰**: 防止分片 Map 溢出
2. **超时清理**: 用户态定期清理过期分片
3. **最大分片数限制**: 防止资源耗尽
4. **快速路径优化**: 非分片包不经过分片逻辑

---

### 挑战 3: 应用层协议识别的准确性

**问题**:
- 仅基于端口不准确（非标准端口）
- 加密流量无法识别
- DPI 性能开销

**解决方案**:
1. **启发式识别**: 端口 + 特征码
2. **渐进式识别**: 前几个包识别，后续包重用结果
3. **可选功能**: 通过配置启用/禁用
4. **统计反馈**: 识别准确率监控

---

## 性能目标

| 指标 | 当前值 | 目标值 | 备注 |
|------|--------|--------|------|
| **数据包处理延迟** | < 10μs | < 15μs | P0 功能后可接受的增量 |
| **NAT 查询开销** | N/A | < 5μs | Conntrack 查询 |
| **分片处理开销** | N/A | < 5μs | 非首个分片查询 |
| **协议识别开销** | N/A | < 10μs | DPI 扫描 |
| **并发会话** | 100K+ | 100K+ | 保持现有能力 |
| **策略匹配性能** | < 5μs | < 8μs | 索引查找 + NAT 检测 |
| **CPU 开销** | < 5% | < 8% | 增加功能后可接受 |
| **内存占用** | < 500MB | < 800MB | 新增 Map 开销 |

---

## 测试策略

### 单元测试

**eBPF 层**:
- [ ] NAT 检测单元测试
- [ ] 分片处理单元测试
- [ ] 协议识别单元测试
- [ ] 限流算法单元测试

**用户态**:
- [ ] Conntrack 同步逻辑测试
- [ ] 配置 API 测试
- [ ] 统计聚合测试

### 集成测试

- [ ] Docker 环境端到端测试
- [ ] Kubernetes 环境测试
- [ ] NAT 环境策略测试
- [ ] 分片攻击防护测试
- [ ] 协议识别准确性测试

### 性能测试

- [ ] 基准测试（wrk/iperf3）
- [ ] 延迟测试（ping RTT）
- [ ] 吞吐量测试（10Gbps 网卡）
- [ ] CPU/内存 Profiling

### 压力测试

- [ ] 100K+ 并发连接
- [ ] SYN Flood 测试
- [ ] 分片洪水测试
- [ ] 策略大量变更测试

---

## 验收标准

### Phase 1 (P0) 验收

**功能完整性**:
- [x] NAT 环境下策略正常工作
- [x] 分片包正确处理
- [x] 默认拒绝策略生效
- [x] ICMP 类型过滤工作
- [x] TCP 重传检测准确

**性能指标**:
- [x] 数据包处理延迟 < 15μs
- [x] NAT 查询开销 < 5μs
- [x] CPU 开销 < 8%

**测试覆盖**:
- [x] 单元测试覆盖率 > 80%
- [x] E2E 测试通过
- [x] 性能测试达标

---

### Phase 2 (P1) 验收

**功能完整性**:
- [x] HTTP/DNS 协议识别率 > 95%
- [x] 容器级别策略生效
- [x] 带宽限流准确
- [x] 连接数限制生效
- [x] 策略模拟模式工作

**性能指标**:
- [x] 协议识别延迟 < 10μs
- [x] 限流精度误差 < 5%

---

## 里程碑

| 里程碑 | 完成时间 | 关键产出 |
|--------|----------|----------|
| M1: P0 核心功能 | Day 15 | NAT/分片/默认拒绝/ICMP/连接跟踪 |
| M2: 应用层识别 | Day 20 | HTTP/DNS/SSH 协议识别 |
| M3: 进程关联 | Day 24 | 容器级别隔离 |
| M4: 流量控制 | Day 28 | 带宽限流/连接限制 |
| M5: 策略模拟 | Day 30 | Dry-run 模式 |

---

## 风险和依赖

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| **内核版本兼容性** | 高 | 提供降级方案（用户态辅助）|
| **性能下降** | 中 | 渐进式添加功能，持续性能测试 |
| **测试覆盖不足** | 中 | 自动化测试，CI/CD 集成 |
| **复杂度增加** | 低 | 模块化设计，清晰文档 |

---

**文档维护者**: 开发团队
**最后更新**: 2025-11-18
**版本**: v1.0
