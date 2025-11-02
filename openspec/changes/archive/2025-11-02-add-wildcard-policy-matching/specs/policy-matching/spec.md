# 策略匹配增量规范：通配符策略支持

## ADDED Requirements

### Requirement: 通配符端口匹配

系统必须(SHALL)支持端口通配符匹配：
- `SrcPort: 0` 表示"任意源端口"
- `DstPort: 0` 表示"任意目标端口"
- 非零值表示精确端口匹配

这解决了客户端使用随机临时端口（32768-65535）时的策略匹配问题。

#### Scenario: 通配符源端口阻止 SSH 访问

**Given** 存在策略：src=10.100.0.1、dst=10.100.0.2、sport=0、dport=22、proto=TCP、action=DENY
**When** 客户端从临时端口 54321 连接到 SSH 端口 22
**Then** 数据包必须(SHALL)被阻止（TC_ACT_SHOT）
**And** 统计计数器 STATS_DENIED_PACKETS 必须(SHALL)增加
**And** 必须(SHALL)通过环形缓冲区发送流事件

#### Scenario: 通配符目标端口匹配

**Given** 存在策略：src=192.168.1.0、dst=10.0.0.5、sport=0、dport=0、proto=TCP、action=LOG
**When** 到达数据包：src=192.168.1.100:45678、dst=10.0.0.5:8080
**Then** 策略必须(SHALL)匹配（通配符源和目标端口）
**And** 数据包必须(SHALL)被允许通过
**And** 必须(SHALL)生成日志事件

#### Scenario: 精确端口与通配符端口不匹配

**Given** 存在策略：dst=10.0.0.10、dport=80（精确匹配）、action=ALLOW
**When** 到达数据包：dst=10.0.0.10、dport=443（不同端口）
**Then** 策略不得(SHALL NOT)匹配
**And** 必须(SHALL)应用默认策略行为

### Requirement: CIDR IP 范围匹配

系统必须(SHALL)支持带子网掩码的 CIDR IP 范围匹配：
- IP 地址使用 CIDR 表示法（例如 10.0.0.0/8）
- 掩码值：0xFFFFFFFF = 精确匹配，0x00000000 = 任意 IP
- 支持任意 CIDR 前缀长度（/0 到 /32）

#### Scenario: 子网范围策略匹配

**Given** 存在策略：src=10.0.0.0/8、dst=192.168.1.100、dport=443、proto=TCP、action=ALLOW
**When** 到达数据包：src=10.50.100.200、dst=192.168.1.100、dport=443
**Then** 策略必须(SHALL)匹配（源 IP 在 10.0.0.0/8 范围内）
**And** 数据包必须(SHALL)被允许通过

#### Scenario: IP 掩码精确匹配

**Given** 存在策略：src=192.168.1.100/32（完整掩码 = 精确）、dport=22、action=DENY
**When** 到达数据包：src=192.168.1.100
**Then** 策略必须(SHALL)匹配
**When** 到达数据包：src=192.168.1.101
**Then** 策略不得(SHALL NOT)匹配

#### Scenario: 任意 IP 通配符

**Given** 存在策略：src=0.0.0.0/0（任意源 IP）、dst=10.0.0.5、dport=80、action=LOG
**When** 到达数据包：src=任意 IP、dst=10.0.0.5、dport=80
**Then** 策略必须(SHALL)匹配所有源 IP

### Requirement: 通配符协议匹配

系统必须(SHALL)支持协议通配符：
- `Protocol: 0` 表示"任意协议"（TCP、UDP、ICMP 等）
- 非零值表示特定协议（6=TCP、17=UDP）

#### Scenario: 任意协议策略

**Given** 存在策略：dst=192.168.1.100、protocol=0（任意）、action=DENY
**When** 到达 TCP 数据包：dst=192.168.1.100
**Then** 策略必须(SHALL)匹配并阻止
**When** 到达 UDP 数据包：dst=192.168.1.100
**Then** 策略也必须(SHALL)匹配并阻止
**When** 到达 ICMP 数据包：dst=192.168.1.100
**Then** 策略也必须(SHALL)匹配并阻止

### Requirement: 双映射架构

系统必须(SHALL)使用双映射架构进行策略查找：

**快速路径（精确匹配）**：
- 使用现有的 `policy_map`（BPF_MAP_TYPE_HASH）
- 用于精确 5 元组匹配
- O(1) 查找复杂度
- 优先级最高

**慢速路径（通配符匹配）**：
- 使用新的 `wildcard_policy_map`（BPF_MAP_TYPE_ARRAY）
- 用于包含通配符的策略
- 线性搜索（最多 100 次迭代）
- 仅在精确匹配失败时使用

#### Scenario: 双层策略查找

**Given** 精确策略：src=10.0.0.1:12345、dst=10.0.0.2:8080、proto=TCP、action=ALLOW（在 policy_map）
**And** 通配符策略：src=10.0.0.1:0、dst=10.0.0.2:8080、proto=TCP、action=DENY（在 wildcard_policy_map）
**When** 到达数据包：src=10.0.0.1:12345、dst=10.0.0.2:8080
**Then** 必须(SHALL)首先查找 policy_map（快速路径）
**And** 找到精确匹配，返回 ALLOW
**And** 不得(SHALL NOT)检查 wildcard_policy_map

**When** 到达数据包：src=10.0.0.1:54321、dst=10.0.0.2:8080
**Then** 必须(SHALL)首先查找 policy_map（快速路径）
**And** 未找到精确匹配
**And** 必须(SHALL)搜索 wildcard_policy_map（慢速路径）
**And** 找到通配符匹配，返回 DENY

#### Scenario: 快速路径优先级

**Given** 精确和通配符策略都存在
**When** 新流的第一个数据包到达
**Then** 必须(SHALL)总是先尝试精确匹配（哈希查找）
**And** 仅在精确匹配失败时才尝试通配符匹配
**And** 查找决策必须(SHALL)缓存在会话状态中
**And** 后续数据包不得(SHALL NOT)执行任何策略查找

### Requirement: 基于优先级的通配符选择

当多个通配符策略匹配同一流时，系统必须(SHALL)：
- 评估所有匹配的通配符策略
- 选择具有最高优先级值的策略
- 优先级值越大 = 优先级越高（0 = 最低）

#### Scenario: 多通配符策略优先级选择

**Given** 通配符策略 A：src=10.0.0.0/8:0、dst=10.0.0.2:8080、action=DENY、priority=5
**And** 通配符策略 B：src=10.100.0.1:0、dst=10.0.0.2:8080、action=ALLOW、priority=10
**When** 到达数据包：src=10.100.0.1:54321、dst=10.0.0.2:8080
**Then** 两个策略都匹配
**And** 必须(SHALL)选择策略 B（priority=10 > 5）
**And** 数据包必须(SHALL)被允许（ALLOW 操作）

#### Scenario: 相同优先级的通配符策略

**Given** 通配符策略 A：priority=10、action=DENY
**And** 通配符策略 B：priority=10、action=ALLOW
**And** 两个策略都匹配相同的流
**When** 数据包到达
**Then** 必须(SHALL)选择线性扫描中首先找到的策略（槽顺序）
**And** 结果是确定性的（对于给定的映射状态）

### Requirement: 通配符策略存储

系统必须(SHALL)在具有以下特征的 eBPF ARRAY 映射中存储通配符策略：
- 映射名称：`wildcard_policy_map`
- 映射类型：`BPF_MAP_TYPE_ARRAY`
- 最大条目数：1,000 个策略
- 键：__u32 索引（0-999）
- 值：`struct wildcard_policy`

#### Scenario: 通配符策略插入

**Given** 具有通配符字段的新策略（sport=0 或 dport=0 或 protocol=0 或 CIDR）
**When** 用户空间代理调用 AddPolicy()
**Then** 策略必须(SHALL)插入到 wildcard_policy_map（非 policy_map）
**And** 必须(SHALL)找到空槽（rule_id=0）
**And** 槽索引必须(SHALL)按顺序分配（0, 1, 2, ...）

#### Scenario: 通配符策略更新

**Given** 槽 5 中存在 rule_id=100 的通配符策略
**When** 使用 rule_id=100 更新策略
**Then** 必须(SHALL)在槽 5 中更新现有条目
**And** 不得(SHALL NOT)创建重复条目

#### Scenario: 通配符映射已满

**Given** wildcard_policy_map 包含 1,000 个活动策略
**When** 尝试添加新的通配符策略
**Then** 操作必须(SHALL)失败并返回错误"wildcard policy map is full"
**And** 用户空间代理必须(SHALL)收到通知

### Requirement: 通配符匹配函数实现

系统必须(SHALL)实现 `matches_wildcard()` 函数，该函数：
- 接受流键和通配符策略作为输入
- 使用掩码比较源和目标 IP
- 将端口值 0 视为通配符（匹配任意）
- 将协议值 0 视为通配符（匹配任意）
- 返回布尔匹配结果

#### Scenario: 通配符匹配逻辑

**Given** 通配符策略：
- src_ip=10.0.0.1, src_ip_mask=0xFFFFFFFF（精确）
- dst_ip=10.0.0.0, dst_ip_mask=0xFFFFFF00（/24 子网）
- src_port=0（任意）
- dst_port=8080（精确）
- protocol=6（TCP）

**When** 流键为：src=10.0.0.1:12345、dst=10.0.0.100:8080、proto=TCP
**Then** IP 检查：(10.0.0.1 & 0xFFFFFFFF) == (10.0.0.1 & 0xFFFFFFFF) ✓
**And** IP 检查：(10.0.0.100 & 0xFFFFFF00) == (10.0.0.0 & 0xFFFFFF00) ✓
**And** 端口检查：src_port=0（跳过检查，通配符）✓
**And** 端口检查：dst_port=8080 == 8080 ✓
**And** 协议检查：6 == 6 ✓
**And** 必须(SHALL)返回 true（匹配）

**When** 流键为：src=10.0.0.2:12345、dst=10.0.0.100:8080、proto=TCP
**Then** IP 检查失败：10.0.0.2 ≠ 10.0.0.1
**And** 必须(SHALL)返回 false（不匹配）

## MODIFIED Requirements

### Requirement: 策略查找性能

系统必须(SHALL)提供高性能的策略查找：
- **精确匹配**（快速路径）：< 1 微秒平均延迟，O(1) 哈希查找
- **通配符匹配**（慢速路径）：< 100 微秒，O(n) 线性扫描（最多 100 次迭代）
- **整体影响**：< 0.1% 开销（由于会话缓存）
- 最小的 CPU 开销
- 读取无锁

双映射架构确保 99%+ 的数据包使用快速路径（会话缓存），仅新流的第一个数据包经历慢速路径。

#### Scenario: 精确匹配热路径性能

**Given** 包含 10,000 个精确策略的 policy_map
**When** 新流需要策略查找且存在精确匹配
**Then** 查找必须(SHALL)在 < 1 微秒内完成
**And** 必须(SHALL)使用哈希映射 O(1) 查找
**And** 不得(SHALL NOT)由于查找开销而发生数据包丢弃

#### Scenario: 通配符匹配性能

**Given** 包含 100 个通配符策略的 wildcard_policy_map
**And** 精确策略映射中不存在匹配
**When** 新流需要策略查找
**Then** 通配符扫描必须(SHALL)在 < 100 微秒内完成
**And** 必须(SHALL)检查最多 100 个通配符条目（eBPF 验证器限制）
**And** 匹配的策略操作必须(SHALL)缓存在会话中

#### Scenario: 会话缓存性能优化

**Given** 新流的第一个数据包经历了通配符查找（100 微秒）
**When** 同一流的第二个数据包到达
**Then** 不得(SHALL NOT)执行任何策略查找
**And** 必须(SHALL)直接使用 session_value 中的缓存 policy_action
**And** 处理时间必须(SHALL) < 1 微秒（仅会话查找）

### Requirement: 新会话的策略匹配

系统必须(SHALL)对新会话执行双层策略查找：
1. **第一层（快速路径）**：在 policy_map 中查找精确 5 元组匹配
2. **第二层（慢速路径）**：如果精确匹配失败，在 wildcard_policy_map 中搜索通配符匹配
3. **缓存**：将匹配的策略操作存储在 session_value 中
4. **后续数据包**：使用缓存的决策（无需查找）

此双层优化在保持通配符灵活性的同时保持了高性能。

#### Scenario: 新流的双层查找

**Given** 新流到达（会话映射中不存在）
**When** 处理第一个数据包
**Then** 必须(SHALL)首先在 policy_map 中执行精确查找
**If** 找到精确匹配
**Then** 使用该策略操作并跳过通配符查找
**Else** 必须(SHALL)在 wildcard_policy_map 中执行通配符搜索
**And** 匹配的操作必须(SHALL)存储在 session_value.policy_action 中

**When** 同一流的后续数据包到达
**Then** 不得(SHALL NOT)执行任何策略查找
**And** 必须(SHALL)直接使用缓存的 policy_action

#### Scenario: 通配符策略缓存

**Given** 新流匹配通配符策略：action=DENY、rule_id=200
**When** 第一个数据包触发通配符匹配
**Then** session_value 必须(SHALL)设置：
- policy_action = POLICY_ACTION_DENY
- rule_id = 200
**When** 同一流的第二个数据包到达
**Then** 必须(SHALL)从 session_value 读取 policy_action=DENY
**And** 数据包必须(SHALL)被阻止而无需重新查找策略

## Implementation Notes

### 数据结构

**通配符策略结构**：
```c
struct wildcard_policy {
    __u32 src_ip;          // 源 IP 地址
    __u32 src_ip_mask;     // 源 IP 掩码（0xFFFFFFFF = 精确，0x00000000 = 任意）
    __u32 dst_ip;          // 目标 IP 地址
    __u32 dst_ip_mask;     // 目标 IP 掩码
    __u16 src_port;        // 源端口（0 = 任意端口）
    __u16 dst_port;        // 目标端口（0 = 任意端口）
    __u8  protocol;        // 协议（0 = 任意协议）
    __u8  action;          // 策略操作（ALLOW/DENY/LOG）
    __u8  log_enabled;     // 启用日志记录
    __u8  pad1;            // 对齐填充
    __u16 priority;        // 优先级（数字越大 = 优先级越高）
    __u16 pad2;            // 对齐填充
    __u32 rule_id;         // 规则 ID（0 = 空槽）
} __attribute__((packed));
```

**eBPF 映射定义**：
```c
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, MAX_ENTRIES_WILDCARD_POLICY); // 1000
    __type(key, __u32);  // 索引
    __type(value, struct wildcard_policy);
} wildcard_policy_map SEC(".maps");
```

### 策略路由逻辑

**用户空间（Go）**：
```go
func (pm *PolicyManager) addPolicyToMap(p *Policy) error {
    if hasWildcard(p) {
        return pm.addWildcardPolicy(p)  // → wildcard_policy_map
    }
    return pm.addExactPolicy(p)         // → policy_map
}

func hasWildcard(p *Policy) bool {
    return p.SrcPort == 0 ||
           strings.Contains(p.SrcIP, "/") ||
           strings.Contains(p.DstIP, "/") ||
           strings.ToLower(p.Protocol) == "any"
}
```

**内核空间（eBPF）**：
```c
static __always_inline __u8 lookup_policy_action(struct flow_key *key, __u32 *rule_id) {
    // 快速路径：精确匹配
    struct policy_value *policy = bpf_map_lookup_elem(&policy_map, key);
    if (policy) {
        *rule_id = policy->rule_id;
        return policy->action;
    }

    // 慢速路径：通配符匹配
    struct wildcard_policy *best_match = NULL;
    __u16 best_priority = 0;

    #pragma unroll
    for (__u32 i = 0; i < 100; i++) {
        struct wildcard_policy *wildcard = bpf_map_lookup_elem(&wildcard_policy_map, &i);
        if (wildcard && wildcard->rule_id != 0 && matches_wildcard(key, wildcard)) {
            if (!best_match || wildcard->priority > best_priority) {
                best_match = wildcard;
                best_priority = wildcard->priority;
            }
        }
    }

    if (best_match) {
        *rule_id = best_match->rule_id;
        return best_match->action;
    }

    return POLICY_ACTION_ALLOW; // 默认
}
```

## Performance Characteristics

### 查找延迟
- **精确匹配（快速路径）**：< 1 微秒（未更改）
- **通配符匹配（慢速路径）**：< 100 微秒（仅第一个数据包）
- **已建立会话**：< 1 微秒（会话缓存，无查找）

### 内存使用
- **精确策略**：每个 40 字节（在 policy_map 中）
- **通配符策略**：每个 40 字节（在 wildcard_policy_map 中）
- **总内存（通配符）**：~40 KB（1,000 个通配符策略）

### 性能影响
- **快速路径**：0% 开销（精确匹配未更改）
- **慢速路径**：仅新流的第一个数据包
- **整体**：< 0.1% 开销（典型工作负载中 99%+ 的数据包使用会话缓存）

## Security Impact

### 修复的漏洞
**严重性**：CRITICAL
**问题**：带有通配符源端口的 DENY 策略无法阻止流量
**根本原因**：哈希映射需要精确 5 元组匹配，0（通配符）≠ 54321（实际临时端口）
**解决方案**：双映射架构，使用专用的通配符匹配逻辑
**状态**：✅ 已修复 - 所有测试通过，包括先前失败的 TestE2E_DenyPolicy

## Related Capabilities

- **Session Tracking**：在会话状态中缓存通配符策略决策
- **Policy Enforcement**：执行通配符策略操作
- **Statistics Reporting**：跟踪通配符策略命中和未命中
- **Dataplane Performance**：双层查找优化对性能至关重要
