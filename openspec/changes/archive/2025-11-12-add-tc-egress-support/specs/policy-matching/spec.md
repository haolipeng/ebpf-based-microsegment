# Spec Delta: 策略匹配 - 方向感知策略匹配

## MODIFIED Requirements

### Requirement: 5 元组精确匹配

系统必须(SHALL)支持基于 **6 元组**的精确匹配（扩展为包含方向）：
- 源 IP 地址（IPv4）
- 目标 IP 地址（IPv4）
- 源端口（0-65535）
- 目标端口（0-65535）
- 协议（TCP、UDP、ICMP、ANY）
- **方向（INGRESS、EGRESS、ANY）** ✅ 新增

**变更说明**: 从 5 元组扩展为 6 元组，添加方向维度。

#### Scenario: SSH 流量的方向感知精确匹配

**Given** 存在策略：src=192.168.1.0、dst=10.0.0.5、sport=ANY、dport=22、proto=TCP、direction=INGRESS
**When** ingress 数据包到达：src=192.168.1.100、dst=10.0.0.5、sport=45678、dport=22、proto=TCP
**Then** 策略查找必须(SHALL)匹配该策略（因为方向为 INGRESS）
**When** egress 数据包到达：src=10.0.0.5、dst=192.168.1.100、sport=22、dport=45678、proto=TCP
**Then** 策略查找必须(SHALL)不匹配该策略（因为方向为 EGRESS 而策略要求 INGRESS）

#### Scenario: 方向通配符匹配（direction=ANY）

**Given** 存在策略：dst=10.0.0.10、dport=80、proto=TCP、direction=ANY、action=ALLOW
**When** ingress 数据包到达 10.0.0.10:80
**Then** 系统必须(SHALL)匹配该策略（ANY 包含 INGRESS）
**When** egress 数据包到达 10.0.0.10:80
**Then** 系统必须(SHALL)匹配该策略（ANY 包含 EGRESS）

### Requirement: 策略存储

系统必须(SHALL)在 eBPF HASH 映射中存储策略，键结构更新为包含方向：
- 最大条目数：10,000 个策略
- 键：策略 6 元组（struct policy_key，包含 direction 字段）
- 值：策略操作和元数据（struct policy_value）
- O(1) 查找复杂度

**变更说明**: `policy_key` 结构添加 `direction` 字段（uint8）。

#### Scenario: 方向感知策略插入

**Given** 具有 6 元组（包含 direction）和操作的新策略规则
**When** 用户空间代理调用 AddPolicy()
**Then** 策略必须(SHALL)插入到 eBPF 策略映射中
**And** 策略的 direction 字段必须(SHALL)正确设置（1=INGRESS, 2=EGRESS, 0=ANY）
**And** 策略必须(SHALL)立即对新流生效

## ADDED Requirements

### Requirement: 方向感知策略匹配

系统必须(SHALL)在策略匹配时考虑数据包方向：
- **优先匹配**: 先尝试匹配方向特定策略（direction=INGRESS 或 EGRESS）
- **回退匹配**: 如果没有方向特定策略，回退到 direction=ANY

#### Scenario: 优先匹配方向特定策略

**Given** 存在两个策略：
  - 策略 1: dst=10.0.0.10、dport=80、direction=INGRESS、action=ALLOW
  - 策略 2: dst=10.0.0.10、dport=80、direction=ANY、action=DENY
**When** ingress 数据包到达 10.0.0.10:80
**Then** 系统必须(SHALL)优先匹配策略 1（direction=INGRESS）
**And** 数据包必须(SHALL)被允许（不使用策略 2）

#### Scenario: 回退到通配符策略

**Given** 存在策略：dst=10.0.0.10、dport=80、direction=ANY、action=ALLOW
**And** 不存在 direction=INGRESS 的特定策略
**When** ingress 数据包到达 10.0.0.10:80
**Then** 系统必须(SHALL)查找 direction=INGRESS 的策略（未找到）
**And** 系统必须(SHALL)回退查找 direction=ANY 的策略（找到）
**And** 数据包必须(SHALL)被允许

#### Scenario: Egress 流量独立策略

**Given** 存在 egress allow 策略：src=192.168.1.10、dst=10.0.0.20、dport=5432、direction=EGRESS、action=ALLOW
**And** 不存在该 5 元组的 ingress 策略
**When** egress 数据包从 192.168.1.10 到 10.0.0.20:5432
**Then** 系统必须(SHALL)匹配 egress 策略
**And** 数据包必须(SHALL)被允许
**When** ingress 数据包从 10.0.0.20:5432 到 192.168.1.10
**Then** 系统必须(SHALL)不匹配 egress 策略（方向不符）
**And** 必须(SHALL)应用默认策略

### Requirement: 策略键结构

策略键结构必须(SHALL)包含以下字段：

```c
struct policy_key {
    __u32 src_ip;       // 源 IP（网络字节序）
    __u32 dst_ip;       // 目标 IP（网络字节序）
    __u16 dst_port;     // 目标端口（主机字节序）
    __u8  protocol;     // 协议（6=TCP, 17=UDP, 1=ICMP, 0=ANY）
    __u8  direction;    // ✅ 新增: 方向（0=ANY, 1=INGRESS, 2=EGRESS）
} __attribute__((packed));
```

#### Scenario: 策略键正确填充

**Given** 一个 ingress 数据包：src=192.168.1.100、dst=10.0.0.5、dport=22、proto=TCP
**When** eBPF 程序构造 policy_key
**Then** policy_key.src_ip 必须(SHALL)为 192.168.1.100（网络字节序）
**And** policy_key.dst_ip 必须(SHALL)为 10.0.0.5（网络字节序）
**And** policy_key.dst_port 必须(SHALL)为 22
**And** policy_key.protocol 必须(SHALL)为 6（TCP）
**And** policy_key.direction 必须(SHALL)为 1（INGRESS）

### Requirement: 通配符策略的方向支持

通配符策略（wildcard_policy_map）也必须(SHALL)支持方向字段：

```c
struct wildcard_key {
    __u32 dst_ip;       // 目标 IP
    __u16 dst_port;     // 目标端口
    __u8  protocol;     // 协议
    __u8  direction;    // ✅ 新增: 方向
} __attribute__((packed));
```

#### Scenario: 通配符策略方向匹配

**Given** 存在通配符策略：dst=10.0.0.10、dport=80、proto=TCP、direction=EGRESS、action=DENY
**When** egress TCP 数据包发往 10.0.0.10:80
**Then** 通配符策略查找必须(SHALL)匹配（忽略源 IP/端口，但匹配方向）
**And** 数据包必须(SHALL)被拒绝

## ADDED Requirements

### Requirement: 双向策略独立性

Ingress 和 Egress 策略必须(SHALL)独立执行：
- 同一个 5 元组可以有不同的 ingress 和 egress 策略
- Ingress 策略不影响 egress 流量，反之亦然

#### Scenario: 同一 5 元组的双向不同策略

**Given** 存在两个策略：
  - 策略 1: dst=10.0.0.10、dport=80、direction=INGRESS、action=ALLOW
  - 策略 2: src=10.0.0.10、dport=ANY、direction=EGRESS、action=DENY
**When** ingress 数据包到达 10.0.0.10:80
**Then** 必须(SHALL)应用策略 1，数据包被允许
**When** egress 数据包从 10.0.0.10 发出
**Then** 必须(SHALL)应用策略 2，数据包被拒绝
**And** 两个方向的决策必须(SHALL)完全独立

### Requirement: 默认策略与方向

默认策略（当没有匹配的策略时）必须(SHALL)考虑方向：
- 如果配置了默认 DENY，必须(SHALL)分别拒绝 ingress 和 egress
- 统计必须(SHALL)分别记录 ingress 和 egress 的默认拒绝

#### Scenario: 默认拒绝 Egress

**Given** 配置了默认策略为 DENY
**And** 不存在匹配的 egress 策略
**When** egress 数据包发出
**Then** 数据包必须(SHALL)被拒绝
**And** STAT_EGRESS_DENIED 必须(SHALL)递增
**And** 流事件必须(SHALL)标记 direction=EGRESS

#### Scenario: 默认允许 Ingress

**Given** 配置了默认策略为 ALLOW
**And** 不存在匹配的 ingress 策略
**When** ingress 数据包到达
**Then** 数据包必须(SHALL)被允许
**And** STAT_INGRESS_PACKETS 必须(SHALL)递增

## 性能要求

- 方向字段的添加必须(SHALL)不影响查找性能（仍然是 O(1)）
- 双向策略匹配的额外开销必须(SHALL) < 1 微秒

#### Scenario: 方向感知匹配性能

**Given** policy_map 中有 1000 个策略
**When** 进行方向感知策略查找
**Then** 查找时间必须(SHALL) < 1 微秒
**And** 查找复杂度必须(SHALL)为 O(1)

## 向后兼容性

- 旧策略（不包含 direction）必须(SHALL)自动迁移为 direction=ANY
- API 必须(SHALL)支持不传 direction 参数（默认为 ANY）
- 现有的 5 元组匹配行为必须(SHALL)保持不变（通过 direction=ANY 实现）

---

**变更 ID**: add-tc-egress-support
**修改日期**: 2025-11-11
**状态**: Proposed
