# Capability: 策略匹配

## Purpose

策略匹配能力使用 eBPF 哈希映射提供高性能的网络安全策略查找和匹配。它支持基于 5 元组的精确匹配，具有 O(1) 查找复杂度，实现线速策略执行以进行微隔离。

## Context

策略匹配是微隔离系统的核心决策组件。它根据网络流的 5 元组特征确定应对其应用哪个操作（ALLOW、DENY、LOG）。匹配引擎必须是：
- 极快（< 1 微秒查找时间）
- 可扩展（支持 10,000+ 策略）
- 准确（精确的 5 元组匹配）
- 高效（最小的内存占用）
## Requirements
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

### Requirement: 策略操作

系统必须(SHALL)支持三种策略操作：
- **ALLOW**：允许流量（TC_ACT_OK）
- **DENY**：丢弃流量（TC_ACT_SHOT）
- **LOG**：允许流量但生成审计日志

#### Scenario: ALLOW 策略应用

**Given** dst=10.0.0.5:443、proto=TCP 的策略，action=ALLOW
**When** 匹配的数据包到达
**Then** 数据包必须(SHALL)被允许通过（TC_ACT_OK）
**And** 统计计数器 STATS_ALLOWED_PACKETS 必须(SHALL)增加

#### Scenario: DENY 策略应用

**Given** dst=192.168.1.100:22、proto=TCP 的策略，action=DENY
**When** 匹配的数据包到达
**Then** 数据包必须(SHALL)被丢弃（TC_ACT_SHOT）
**And** 统计计数器 STATS_DENIED_PACKETS 必须(SHALL)增加
**And** 必须(SHALL)通过环形缓冲区向用户空间发送流事件

#### Scenario: LOG 策略应用

**Given** dst=10.0.0.10:8080、proto=TCP 的策略，action=LOG
**When** 匹配的数据包到达
**Then** 数据包必须(SHALL)被允许通过（TC_ACT_OK）
**And** 必须(SHALL)向用户空间发送流事件以进行审计日志记录

### Requirement: 策略优先级

每个策略必须(SHALL)有一个优先级字段（0-65535，数字越小优先级越高）。
- 目前用于元数据和未来的规则排序
- 在当前实现（精确匹配）中，第一个匹配获胜

#### Scenario: 策略优先级元数据

**Given** 两个策略具有相同的 5 元组但不同的优先级
**When** 插入策略
**Then** 系统必须(SHALL)执行精确匹配语义
**And** 优先级必须(SHALL)可用于未来的分层匹配

### Requirement: 策略命中计数

系统必须(SHALL)为每个策略维护命中计数器以跟踪使用情况：
- 每次策略匹配时计数器递增
- 计数器存储在 policy_value.hit_count 中
- 用户空间可以查询命中计数以进行分析

#### Scenario: 策略命中计数器更新

**Given** 策略的 hit_count = 100
**When** 匹配的数据包到达
**Then** hit_count 必须(SHALL)递增到 101
**And** STATS_POLICY_HITS 全局计数器必须(SHALL)递增

#### Scenario: 策略使用分析

**Given** 安装了多个策略
**When** 用户空间代理查询策略统计信息
**Then** 它必须(SHALL)能够识别：
- 最常匹配的策略（高 hit_count）
- 未使用的策略（hit_count = 0）
- 随时间推移的策略有效性

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

### Requirement: 策略 CRUD 操作

用户空间代理必须(SHALL)支持对策略的完整 CRUD 操作：
- **Create**：向 eBPF 映射添加新策略
- **Read**：从 eBPF 映射列出所有策略
- **Update**：修改现有策略
- **Delete**：从 eBPF 映射删除策略

#### Scenario: 通过 API 添加策略

**Given** 用户空间代理正在运行
**When** 通过 PolicyManager.AddPolicy() 添加新策略
**Then** 策略必须(SHALL)插入到 eBPF policy_map 中
**And** 操作必须(SHALL)返回成功

#### Scenario: 删除策略

**Given** 存在具有特定 5 元组的策略
**When** 使用匹配的 5 元组调用 PolicyManager.DeletePolicy()
**Then** 策略必须(SHALL)从 eBPF policy_map 中删除
**And** 未来的流必须(SHALL NOT)匹配此策略

#### Scenario: 列出所有策略

**Given** 策略映射中存在多个策略
**When** 调用 PolicyManager.ListPolicies()
**Then** 必须(SHALL)返回所有策略的列表
**And** 每个策略必须(SHALL)包括其 5 元组、操作、优先级和 hit_count

### Requirement: 默认策略行为

系统必须(SHALL)在没有策略匹配时应用默认操作：
- 默认操作：ALLOW（开发期间为安全起见而失败打开）
- 可配置用于生产部署（通常失败关闭）

#### Scenario: 无匹配策略

**Given** 特定 5 元组不存在策略
**When** 具有该 5 元组的数据包到达
**Then** 必须(SHALL)应用默认操作（ALLOW）
**And** STATS_POLICY_MISSES 计数器必须(SHALL)递增
**And** 必须(SHALL)创建会话，policy_action=ALLOW

### Requirement: 策略规则 ID 跟踪

每个策略必须(SHALL)有一个唯一的 rule_id 用于跟踪和审计：
- 用户分配的 32 位无符号整数
- 用于日志和事件中的关联
- 帮助识别哪个策略触发了操作

#### Scenario: 事件中的策略规则 ID

**Given** rule_id=1001 和 action=DENY 的策略
**When** 匹配的数据包到达
**Then** 发送到用户空间的流事件必须(SHALL)包含 rule_id=1001
**And** 审计日志必须(SHALL)引用 rule_id=1001

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

## Data Structures

### 策略键结构

```c
struct policy_key {
    __u32 src_ip;      // 源 IP 地址（ANY 为 0.0.0.0）
    __u32 dst_ip;      // 目标 IP 地址
    __u16 src_port;    // 源端口（ANY 为 0）
    __u16 dst_port;    // 目标端口
    __u8  protocol;    // 协议（6=TCP、17=UDP、0=ANY）
    __u8  pad[3];      // 对齐填充
} __attribute__((packed));
```

### 策略值结构

```c
struct policy_value {
    __u8  action;        // 策略操作（ALLOW/DENY/LOG）
    __u8  log_enabled;   // 启用日志记录（0=禁用、1=启用）
    __u16 priority;      // 策略优先级（数字越小 = 优先级越高）
    __u32 rule_id;       // 用于跟踪的规则 ID
    __u64 hit_count;     // 策略匹配次数
};
```

### 策略操作枚举

```c
enum policy_action {
    POLICY_ACTION_ALLOW = 0,
    POLICY_ACTION_DENY  = 1,
    POLICY_ACTION_LOG   = 2,
};
```

## Implementation Notes

- 策略映射实现为 `BPF_MAP_TYPE_HASH`（标准哈希映射）
- 最大容量：10,000 个策略（可通过 `MAX_ENTRIES_POLICY` 配置）
- 策略匹配仅在第一个数据包（新会话）时发生
- 后续数据包使用 session_value 中的缓存决策
- 当前实现仅支持 IPv4；计划支持 IPv6
- 仅精确匹配；计划在未来版本中支持 CIDR/范围匹配

## Performance Characteristics

- **查找延迟**：< 1 微秒（平均）
- **查找复杂度**：O(1) 哈希映射查找
- **每个策略的内存**：40 字节
- **总内存**：~400 KB（10,000 个策略）
- **并发访问**：读取无锁

## 性能优化策略

策略匹配引擎针对常见情况进行了优化：
1. **第一个数据包**：完整策略查找（较慢路径）
2. **后续数据包**：会话中的缓存决策（快速路径）

这意味着：
- 新会话创建：~2-3 微秒（包括策略查找）
- 已建立会话处理：< 1 微秒（无策略查找）

对于具有长期连接的典型工作负载，>99% 的数据包使用快速路径。

## Related Capabilities

- **Session Tracking**：在会话状态中缓存策略决策
- **Policy Enforcement**：执行匹配的策略操作
- **Statistics Reporting**：跟踪策略命中和未命中
- **Dataplane Performance**：策略缓存对性能至关重要

