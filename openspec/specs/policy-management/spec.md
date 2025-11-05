# policy-management Specification

## Purpose
TBD - created by archiving change add-control-plane-api. Update Purpose after archive.
## Requirements
### Requirement: 线程安全策略访问
策略管理系统必须(SHALL)为并发操作提供线程安全访问。

#### Scenario: 并发策略读取
- **WHEN** 多个 goroutine 同时读取策略
- **THEN** 所有读取必须(MUST)成功完成
- **AND** 所有读取必须(MUST)返回一致的数据

#### Scenario: 并发策略写入
- **WHEN** 多个 goroutine 同时更新策略
- **THEN** 所有更新必须(MUST)原子地应用
- **AND** 不得(MUST NOT)丢失任何更新

#### Scenario: 读写同步
- **WHEN** 在更新发生时正在读取策略
- **THEN** 读取不得(MUST NOT)看到部分更新
- **AND** 读取不得(MUST NOT)不必要地阻塞

---

### Requirement: 策略生命周期管理
策略管理器必须(SHALL)跟踪策略的完整生命周期。

#### Scenario: 策略创建时间戳
- **WHEN** 创建新策略
- **THEN** 必须(MUST)记录创建时间戳
- **AND** 时间戳必须(MUST)可通过 API 访问

#### Scenario: 策略更新时间戳
- **WHEN** 更新策略
- **THEN** 必须(MUST)记录最后更新时间戳
- **AND** 保留原始创建时间戳

---

### Requirement: 策略 ID 管理
策略管理器必须(SHALL)分配和管理唯一的策略标识符。

#### Scenario: 自动生成策略 ID
- **WHEN** 创建策略时未指定显式 ID
- **THEN** 系统必须(MUST)生成唯一 ID
- **AND** 必须(MUST)将 ID 返回给客户端

#### Scenario: 显式策略 ID
- **WHEN** 使用显式 ID 创建策略
- **THEN** 如果尚未使用，则使用提供的 ID
- **WHEN** ID 已被使用
- **THEN** 返回冲突错误

#### Scenario: ID 持久化
- **THEN** 策略 ID 必须(MUST)在 API 重启后保持稳定
- **AND** ID 必须(MUST)在系统内唯一

---

### Requirement: 策略验证
策略管理器必须(SHALL)在应用之前验证所有策略规则。

#### Scenario: 必需字段验证
- **WHEN** 策略缺少必需字段
- **THEN** 返回列出缺少字段的验证错误

#### Scenario: IP 地址格式验证
- **WHEN** 策略包含无效的 IP 地址格式
- **THEN** 返回验证错误

#### Scenario: CIDR 表示法验证
- **WHEN** 策略使用 CIDR 表示法（例如，"10.0.0.0/24"）
- **THEN** 验证网络地址和前缀长度
- **WHEN** CIDR 无效
- **THEN** 返回验证错误

#### Scenario: 端口范围验证
- **WHEN** 策略端口超出 0-65535 范围
- **THEN** 返回验证错误

#### Scenario: 协议验证
- **WHEN** 策略协议不是"tcp"、"udp"、"icmp"或"any"
- **THEN** 返回验证错误

#### Scenario: 操作验证
- **WHEN** 策略操作不是"allow"、"deny"或"log"
- **THEN** 返回验证错误

---

### Requirement: 策略查询和过滤
策略管理器必须(SHALL)支持查询和过滤策略。

#### Scenario: 列出所有策略
- **WHEN** 查询所有策略
- **THEN** 返回活跃策略的完整列表
- **AND** 包括所有策略元数据

#### Scenario: 按源 IP 过滤
- **WHEN** 使用源 IP 过滤器查询策略
- **THEN** 仅返回匹配的策略

#### Scenario: 按操作过滤
- **WHEN** 使用操作过滤器查询策略
- **THEN** 仅返回具有指定操作的策略

#### Scenario: 按优先级排序
- **WHEN** 查询策略
- **THEN** 允许按优先级排序（升序/降序）

### Requirement: 基于组的策略规则（Group-Based Policy Rules）
系统 SHALL 支持在工作负载组之间定义策略规则。

#### Scenario: 使用组创建 PolicyRule
- **WHEN** POST /api/v1/policy-rules 包含 from_group 和 to_group
- **THEN** 必须返回 201 Created
- **AND** 必须验证两个组都存在
- **AND** 必须存储策略规则

#### Scenario: PolicyRule 引用不存在的组
- **WHEN** 创建引用不存在的组的策略规则时
- **THEN** 必须返回 400 Bad Request
- **AND** 必须在错误消息中标识缺失的组

#### Scenario: PolicyRule 包含端口范围
- **WHEN** 策略规则指定多个端口范围时
- **THEN** 必须支持包含 protocol、start 和 end 的端口范围
- **AND** 必须验证：0 ≤ start ≤ end ≤ 65535

#### Scenario: PolicyRule 动作类型
- **THEN** 必须支持以下动作："allow"、"deny"、"log"
- **AND** 必须验证动作是支持的类型之一

#### Scenario: PolicyRule 优先级
- **WHEN** 创建带有优先级的策略规则时
- **THEN** 必须支持优先级值 0-1000
- **AND** 更高优先级的规则必须首先编译

#### Scenario: PolicyRule 启用/禁用
- **WHEN** 策略规则被禁用时（enabled=false）
- **THEN** 不得编译为 IP 规则
- **AND** 现有的已编译规则必须被删除
- **WHEN** 重新启用时
- **THEN** 必须重新编译并应用规则

---

### Requirement: 策略编译（Policy Compilation）
系统 SHALL 将基于组的策略规则编译为基于 IP 的规则。

#### Scenario: 笛卡尔积扩展
- **GIVEN** from_group 有 3 个工作负载（IPs: 10.0.1.1, 10.0.1.2, 10.0.1.3）
- **AND** to_group 有 2 个工作负载（IPs: 10.0.2.1, 10.0.2.2）
- **WHEN** 编译策略规则时
- **THEN** 必须生成 3×2=6 条基于 IP 的策略
- **AND** 每条已编译策略必须映射一个源 IP 到一个目标 IP

#### Scenario: 多端口扩展
- **GIVEN** 策略规则有 2 个端口范围
- **AND** from_group 有 2 个工作负载，to_group 有 2 个工作负载
- **WHEN** 编译时
- **THEN** 必须生成 2×2×2=8 条基于 IP 的策略（N_src × N_dst × N_ports）

#### Scenario: 编译性能目标
- **GIVEN** from_group 中有 10 个工作负载，to_group 中有 10 个工作负载
- **WHEN** 编译策略规则时（100 条 IP 规则）
- **THEN** 编译必须在 <500ms 内完成

#### Scenario: 大型组警告
- **GIVEN** from_group 或 to_group 有 >100 个成员
- **WHEN** 编译策略规则时
- **THEN** 必须记录关于潜在策略膨胀的警告
- **AND** 应当包含预计的已编译规则数量

---

### Requirement: 溯源跟踪（Provenance Tracking）
系统 SHALL 跟踪已编译策略与源策略规则之间的关系。

#### Scenario: 源策略关联
- **WHEN** 策略规则被编译为基于 IP 的策略时
- **THEN** 每条已编译策略必须记录：
  - source_policy_id（PolicyRule ID）
  - from_group 名称
  - to_group 名称
  - from_workload_id（源工作负载）
  - to_workload_id（目标工作负载）

#### Scenario: 按源查询已编译策略
- **WHEN** GET /api/v1/policy-rules/{id}/compiled
- **THEN** 必须返回 200 OK
- **AND** 必须返回此规则的所有已编译 IP 策略

#### Scenario: 追溯已编译策略到源
- **WHEN** GET /api/v1/compiled-policies/{id}/source
- **THEN** 必须返回 200 OK
- **AND** 必须返回生成此已编译策略的源 PolicyRule

#### Scenario: 溯源持久化
- **THEN** 溯源数据必须持久化到数据库
- **AND** 必须在系统重启后保留

---

### Requirement: 策略重新编译（Policy Recompilation）
系统 SHALL 在依赖项变更时重新编译策略。

#### Scenario: 工作负载标签变更触发重新编译
- **GIVEN** 策略规则从组 "web" 到组 "db"
- **AND** 工作负载 "wl-1" 是组 "web" 的成员
- **WHEN** 工作负载 "wl-1" 的标签被更新并离开组 "web" 时
- **THEN** 必须重新编译所有引用组 "web" 的策略
- **AND** 必须删除涉及 "wl-1" 的已编译策略

#### Scenario: 组选择器变更触发重新编译
- **GIVEN** 策略规则从组 "api" 到组 "cache"
- **WHEN** 组 "api" 的选择器被更新时
- **THEN** 必须重新编译所有引用组 "api" 的策略
- **AND** 必须更新已编译策略以反映新的成员关系

#### Scenario: 工作负载删除触发清理
- **GIVEN** 工作负载 "wl-5" 涉及已编译策略
- **WHEN** 工作负载 "wl-5" 被删除时
- **THEN** 必须删除所有涉及 "wl-5" 的已编译策略
- **AND** 必须清理溯源记录

#### Scenario: 增量重新编译
- **WHEN** 重新编译策略规则时
- **THEN** 应当仅重新生成已变更的已编译策略
- **AND** 不应当重新编译未变更的策略（优化）

---

### Requirement: PolicyRule CRUD 操作
系统 SHALL 提供用于管理基于组的策略规则的 API。

#### Scenario: 创建 PolicyRule
- **WHEN** POST /api/v1/policy-rules 包含有效数据
- **THEN** 必须返回 201 Created
- **AND** 必须自动编译规则
- **AND** 必须返回已创建的带有分配 ID 的 PolicyRule

#### Scenario: 列出 PolicyRules
- **WHEN** GET /api/v1/policy-rules
- **THEN** 必须返回 200 OK
- **AND** 必须返回所有策略规则的数组
- **AND** 必须包含 from_group、to_group、ports、action、priority

#### Scenario: 按 ID 获取 PolicyRule
- **WHEN** GET /api/v1/policy-rules/{id}
- **THEN** 必须返回 200 OK
- **AND** 必须返回完整的规则详情
- **AND** 可以包含已编译策略的数量

#### Scenario: 更新 PolicyRule
- **WHEN** PUT /api/v1/policy-rules/{id}
- **THEN** 必须返回 200 OK
- **AND** 必须删除旧的已编译策略
- **AND** 必须使用新设置重新编译
- **AND** 必须将新的已编译策略应用到 eBPF maps

#### Scenario: 删除 PolicyRule
- **WHEN** DELETE /api/v1/policy-rules/{id}
- **THEN** 必须返回 204 No Content
- **AND** 必须删除所有已编译策略
- **AND** 必须从 eBPF maps 中删除规则
- **AND** 必须清理溯源记录

---

### Requirement: 手动编译触发器（Manual Compilation Trigger）
系统 SHALL 允许通过 API 手动触发策略编译。

#### Scenario: 触发单个规则的编译
- **WHEN** POST /api/v1/policy-rules/{id}/compile
- **THEN** 必须返回 200 OK
- **AND** 必须重新编译指定的规则
- **AND** 必须返回生成的已编译策略数量

#### Scenario: 触发所有规则的编译
- **WHEN** POST /api/v1/policy-rules/compile-all
- **THEN** 必须返回 200 OK
- **AND** 必须重新编译所有已启用的策略规则
- **AND** 必须返回摘要：总规则数、总已编译策略数

#### Scenario: 编译幂等性
- **WHEN** 在没有变更的情况下多次编译同一规则时
- **THEN** 必须产生相同的已编译策略
- **AND** 不得在数据库中创建重复项

---

### Requirement: 已编译策略生命周期（Compiled Policy Lifecycle）
系统 SHALL 管理已编译的基于 IP 的策略的生命周期。

#### Scenario: 已编译策略创建
- **WHEN** PolicyRule 被编译时
- **THEN** 已编译策略必须存储在现有的 `policies` 表中
- **AND** 必须为每条已编译策略分配唯一的 rule_id
- **AND** 溯源必须存储在 `policy_compilation` 表中

#### Scenario: 已编译策略格式
- **THEN** 已编译策略必须使用与手动基于 IP 的策略相同的格式：
  - rule_id（唯一）
  - src_ip（CIDR 表示法）
  - dst_ip（CIDR 表示法）
  - dst_port
  - protocol
  - action
  - priority

#### Scenario: 已编译策略应用到数据平面
- **WHEN** 创建已编译策略时
- **THEN** 必须调用现有的 PolicyManager.AddPolicy()
- **AND** 必须更新 eBPF maps
- **AND** 数据平面必须强制执行规则

#### Scenario: 已编译策略删除
- **WHEN** 源 PolicyRule 被删除或重新编译时
- **THEN** 必须从 `policies` 表中删除已编译策略
- **AND** 必须更新 eBPF maps
- **AND** 必须清理 `policy_compilation` 记录

---

### Requirement: PolicyRule 验证
系统 SHALL 在编译前验证策略规则。

#### Scenario: 必填字段验证
- **WHEN** 创建 PolicyRule 时
- **THEN** 必须验证必填字段：name、from_group、to_group、action
- **AND** 如果缺少任何字段，必须返回 400 Bad Request

#### Scenario: 组存在性验证
- **WHEN** 创建 PolicyRule 时
- **THEN** 必须验证 from_group 存在
- **AND** 必须验证 to_group 存在
- **AND** 如果任一组缺失，必须返回 400

#### Scenario: 端口范围验证
- **WHEN** PolicyRule 包含端口时
- **THEN** 每个端口范围必须有有效的 protocol（"tcp"、"udp"、"icmp"）
- **AND** 必须满足 0 ≤ start ≤ end ≤ 65535
- **AND** 对于无效的端口范围必须返回 400

#### Scenario: 动作验证
- **WHEN** PolicyRule 指定动作时
- **THEN** 动作必须是 "allow"、"deny" 或 "log"
- **AND** 对于无效的动作必须返回 400

#### Scenario: 优先级范围验证
- **WHEN** PolicyRule 指定优先级时
- **THEN** 优先级必须在 0-1000 之间
- **AND** 对于超出范围的优先级必须返回 400

---

### Requirement: 策略编译错误处理（Policy Compilation Error Handling）
系统 SHALL 优雅地处理编译错误。

#### Scenario: 编译期间组为空
- **WHEN** 编译具有空组（零成员）的规则时
- **THEN** 必须成功（返回 200）
- **AND** 必须生成零条已编译策略
- **AND** 应当记录关于空组的警告

#### Scenario: 编译期间组被删除
- **WHEN** 组在其策略正在编译时被删除
- **THEN** 编译必须失败并返回错误
- **AND** 不得创建部分已编译策略

#### Scenario: eBPF Map 更新失败
- **WHEN** 已编译策略无法写入 eBPF maps 时
- **THEN** 必须回滚编译
- **AND** 不得在数据库中持久化已编译策略
- **AND** 必须返回 500 Internal Server Error

---

### Requirement: PolicyRule 元数据
系统 SHALL 跟踪策略规则的元数据。

#### Scenario: PolicyRule 时间戳
- **WHEN** 创建 PolicyRule 时
- **THEN** 必须记录 created_at 时间戳
- **WHEN** 更新时
- **THEN** 必须记录 updated_at 时间戳
- **AND** 必须保留原始的 created_at

#### Scenario: PolicyRule 描述
- **WHEN** 创建 PolicyRule 时
- **THEN** 可以包含可选的 description 字段
- **AND** description 必须在查询中被存储并返回

#### Scenario: 编译时间戳
- **WHEN** PolicyRule 被编译时
- **THEN** 每条已编译策略必须记录 compiled_at 时间戳
- **AND** 时间戳必须为 UTC

---

### Requirement: 策略查询和过滤（Policy Query and Filtering）
系统 SHALL 支持查询和过滤策略规则。

#### Scenario: 按组过滤
- **WHEN** GET /api/v1/policy-rules?from_group=web
- **THEN** 必须仅返回 from_group="web" 的规则
- **WHEN** GET /api/v1/policy-rules?to_group=db
- **THEN** 必须仅返回 to_group="db" 的规则

#### Scenario: 按动作过滤
- **WHEN** GET /api/v1/policy-rules?action=deny
- **THEN** 必须仅返回 action="deny" 的规则

#### Scenario: 按启用状态过滤
- **WHEN** GET /api/v1/policy-rules?enabled=true
- **THEN** 必须仅返回已启用的规则
- **WHEN** GET /api/v1/policy-rules?enabled=false
- **THEN** 必须仅返回已禁用的规则

#### Scenario: 按优先级排序
- **WHEN** GET /api/v1/policy-rules?sort=priority
- **THEN** 必须返回按优先级排序的规则（默认降序）

---

### Requirement: 并发 PolicyRule 操作（Concurrent PolicyRule Operations）
系统 SHALL 安全地处理并发策略规则操作。

#### Scenario: 并发规则创建
- **WHEN** 同时创建多个策略规则时
- **THEN** 每个规则必须独立编译
- **AND** 所有规则必须成功

#### Scenario: 并发编译
- **WHEN** 同一 PolicyRule 被并发编译时
- **THEN** 编译必须序列化
- **AND** 不得创建重复的已编译策略

#### Scenario: 更新期间的并发重新编译
- **WHEN** PolicyRule 在重新编译时被更新
- **THEN** 更新必须排队
- **AND** 最终状态必须反映最后一次更新

---

### Requirement: 与现有策略系统集成（Integration with Existing Policy System）
基于标签的策略系统 SHALL 与现有的基于 IP 的策略无缝集成。

#### Scenario: 策略类型共存
- **WHEN** 系统同时具有基于 IP 的策略（手动）和已编译策略（来自 PolicyRules）时
- **THEN** 两者必须存储在同一 `policies` 表中
- **AND** 两者必须由数据平面强制执行

#### Scenario: 区分策略类型
- **WHEN** 查询已编译策略时
- **THEN** 必须能够通过溯源表与手动 IP 策略区分
- **WHEN** 查询手动 IP 策略时
- **THEN** 不得出现在 policy_compilation 表中

#### Scenario: 优先级处理
- **WHEN** 手动策略和已编译策略都适用于同一流量时
- **THEN** priority 字段必须决定优先级
- **AND** 更高优先级必须获胜，无论策略类型如何

#### Scenario: API 向后兼容性
- **THEN** 现有的 /api/v1/policies 端点必须继续工作
- **AND** 必须返回手动和已编译策略
- **AND** 客户端无需更改即可支持基于标签的策略

---

### Requirement: 策略编译可观测性（Policy Compilation Observability）
系统 SHALL 提供对编译过程的可见性。

#### Scenario: 编译状态查询
- **WHEN** GET /api/v1/policy-rules/{id}
- **THEN** 响应应当包含：
  - last_compiled_at 时间戳
  - compiled_policy_count（生成的 IP 规则数量）
  - compilation_status（"success"、"pending"、"failed"）

#### Scenario: 编译日志
- **WHEN** 编译策略规则时
- **THEN** 必须记录：规则 ID、组大小、生成的已编译策略数量
- **AND** 必须记录大规模扩展的警告（>100 条策略）

#### Scenario: 编译错误详情
- **WHEN** 编译失败时
- **THEN** 必须记录详细的错误消息
- **AND** 必须包含：规则 ID、涉及的组、错误原因
- **AND** 错误必须返回给 API 调用者

---

### Requirement: 数据库模式扩展（Database Schema Extensions）
系统 SHALL 扩展数据库模式以支持策略规则和编译。

#### Scenario: PolicyRule 表模式
- **THEN** 必须包含表 `policy_rules`，包含以下列：
  - id（主键，自增）
  - name（唯一，非空）
  - description（可选）
  - from_group（外键到 groups.name）
  - to_group（外键到 groups.name）
  - ports（JSON 数组）
  - protocols（JSON 数组）
  - action（非空）
  - priority（非空，默认 100）
  - enabled（布尔值，默认 true）
  - created_at（时间戳）
  - updated_at（时间戳）

#### Scenario: Policy Compilation 表模式
- **THEN** 必须包含表 `policy_compilation`，包含以下列：
  - compiled_policy_id（外键到 policies.rule_id）
  - source_policy_id（外键到 policy_rules.id）
  - from_group（文本）
  - to_group（文本）
  - from_workload_id（文本）
  - to_workload_id（文本）
  - compiled_at（时间戳）

#### Scenario: 外键约束
- **THEN** policy_rules 中的 from_group 和 to_group 必须具有 ON DELETE CASCADE
- **AND** compiled_policy_id 必须具有 ON DELETE CASCADE
- **AND** 删除 PolicyRule 必须级联删除已编译策略

---

### Requirement: PolicyRule 原子操作（PolicyRule Atomic Operations）
系统 SHALL 确保策略规则操作的原子性。

#### Scenario: 原子规则创建和编译
- **WHEN** 创建 PolicyRule 时
- **THEN** 规则创建和编译必须是原子的
- **AND** 如果编译失败，规则不得持久化
- **AND** 如果规则创建失败，不得创建已编译策略

#### Scenario: 原子规则更新和重新编译
- **WHEN** 更新 PolicyRule 时
- **THEN** 必须删除旧的已编译策略
- **AND** 必须持久化规则更新
- **AND** 必须创建新的已编译策略
- **AND** 所有步骤必须是原子的（事务）

#### Scenario: 原子规则删除
- **WHEN** 删除 PolicyRule 时
- **THEN** 必须从 eBPF maps 中删除已编译策略
- **AND** 必须从数据库中删除已编译策略
- **AND** 必须删除 PolicyRule
- **AND** 所有步骤必须是原子的

