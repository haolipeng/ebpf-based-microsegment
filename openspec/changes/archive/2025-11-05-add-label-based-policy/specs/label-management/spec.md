# label-management 规范

## Purpose
本规范定义了基于标签的工作负载管理和基于组的策略抽象的需求。它引入了一个更高级别的策略模型，其中工作负载通过标签（键值对）进行标识，并通过标签选择器进行分组，从而实现语义化的策略规则，这些规则在工作负载 IP 地址变化时仍然有效。

## ADDED Requirements

### Requirement: 工作负载身份管理
系统 SHALL 维护独立于 IP 地址的工作负载持久身份。

#### Scenario: 使用唯一 ID 创建工作负载
- **WHEN** 注册新的工作负载时
- **THEN** 系统必须分配一个唯一的 workload ID
- **AND** 该 ID 必须在工作负载重启后保持不变
- **AND** 该 ID 必须独立于 IP 地址

#### Scenario: 工作负载 IP 地址变化
- **GIVEN** 一个 ID 为 "workload-123" 的工作负载的 IP 地址为 10.0.1.10
- **WHEN** 该工作负载使用新的 IP 地址 10.0.1.20 重启时
- **THEN** workload ID 必须保持为 "workload-123"
- **AND** 引用该工作负载的策略必须继续适用

#### Scenario: 每个工作负载多个 IP 地址
- **WHEN** 一个工作负载具有多个网络接口时
- **THEN** 系统必须存储所有 IP 地址
- **AND** 策略必须适用于所有 IP 地址

---

### Requirement: 标签系统
系统 SHALL 支持用于工作负载分类的任意键值标签。

#### Scenario: 标签键值存储
- **WHEN** 创建带有标签的工作负载时
- **THEN** 所有标签必须以键值对的形式存储
- **AND** 键和值都必须支持字母数字字符、连字符、下划线和点号

#### Scenario: 标签更新
- **WHEN** 工作负载标签被更新时
- **THEN** 系统必须记录更新时间戳
- **AND** 必须重新评估组成员资格

#### Scenario: 保留的标签前缀
- **WHEN** 标签使用保留的前缀时（例如 "system.", "k8s."）
- **THEN** 系统应该验证标签格式
- **AND** 可以拒绝无效的系统标签

#### Scenario: Illumio 四维标签
- **GIVEN** 推荐的标签维度：Role、App、Env、Location
- **WHEN** 工作负载使用这些维度时
- **THEN** 系统必须将它们视为标准标签
- **AND** 不得强制执行这些维度（它们是推荐的，而非必需的）

---

### Requirement: 自动标记
系统 SHALL 支持从工作负载元数据自动推断标签。

#### Scenario: 从容器镜像推断 Role
- **WHEN** 工作负载的镜像名称为 "nginx:1.21" 时
- **THEN** 自动标记器应该推断出标签 `role=web`
- **WHEN** 工作负载的镜像名称为 "mysql:8.0" 时
- **THEN** 自动标记器应该推断出标签 `role=db`

#### Scenario: 从监听端口推断 Role
- **WHEN** 工作负载监听端口 [80, 443] 时
- **THEN** 自动标记器应该推断出标签 `role=web`
- **WHEN** 工作负载监听端口 3306 时
- **THEN** 自动标记器应该推断出标签 `role=db`

#### Scenario: 手动标签覆盖自动标签
- **GIVEN** 自动标记器从镜像推断出 `role=web`
- **WHEN** 用户明确设置 `role=api` 时
- **THEN** 手动标签必须优先
- **AND** 自动推断的标签必须被丢弃

#### Scenario: 选择退出自动标记
- **WHEN** 创建工作负载时禁用了自动标记
- **THEN** 系统不得添加任何推断的标签
- **AND** 只能使用明确提供的标签

---

### Requirement: 标签验证
系统 SHALL 在存储前验证标签的键和值。

#### Scenario: 有效的标签格式
- **WHEN** 标签键为 "app.kubernetes.io/name"
- **AND** 值为 "frontend-v2.1" 时
- **THEN** 验证必须成功

#### Scenario: 无效的标签键字符
- **WHEN** 标签键包含空格或特殊字符时（例如 "my label!", "foo@bar"）
- **THEN** 验证必须失败并返回描述性错误

#### Scenario: 标签长度限制
- **WHEN** 标签键超过 253 个字符
- **OR** 标签值超过 63 个字符时
- **THEN** 验证必须失败并返回长度限制错误

#### Scenario: 空标签值
- **WHEN** 标签有键但值为空时（例如 `env=`）
- **THEN** 验证必须成功（允许空值）

---

### Requirement: 组定义
系统 SHALL 支持通过标签选择器定义组。

#### Scenario: 使用选择器创建组
- **WHEN** 创建名为 "web-prod" 的组
- **AND** 选择器为 `[{key: "role", operator: "=", values: ["web"]}, {key: "env", operator: "=", values: ["prod"]}]` 时
- **THEN** 该组必须被存储
- **AND** 选择器必须定义组成员资格

#### Scenario: 组名称唯一性
- **WHEN** 创建与已存在组同名的组时
- **THEN** 系统必须返回冲突错误
- **AND** 不得覆盖现有组

#### Scenario: 选择器操作符
- **THEN** 系统必须支持以下操作符：
  - `=`（等于）
  - `!=`（不等于）
  - `in`（值在列表中）
  - `not-in`（值不在列表中）
  - `exists`（键存在）
  - `not-exists`（键不存在）

#### Scenario: 多个选择器（AND 逻辑）
- **WHEN** 组有多个选择器时
- **THEN** 工作负载必须匹配所有选择器才能成为成员（AND 逻辑）

---

### Requirement: 组成员解析
系统 SHALL 根据工作负载标签动态解析组成员资格。

#### Scenario: 等于操作符匹配
- **GIVEN** 组的选择器为 `{key: "role", operator: "=", values: ["web"]}`
- **WHEN** 工作负载具有标签 `role=web` 时
- **THEN** 工作负载必须是该组的成员

#### Scenario: 不等于操作符匹配
- **GIVEN** 组的选择器为 `{key: "env", operator: "!=", values: ["prod"]}`
- **WHEN** 工作负载具有标签 `env=dev` 时
- **THEN** 工作负载必须是该组的成员
- **WHEN** 工作负载具有标签 `env=prod` 时
- **THEN** 工作负载不得是该组的成员

#### Scenario: In 操作符匹配
- **GIVEN** 组的选择器为 `{key: "app", operator: "in", values: ["frontend", "backend"]}`
- **WHEN** 工作负载具有标签 `app=frontend` 时
- **THEN** 工作负载必须是该组的成员
- **WHEN** 工作负载具有标签 `app=database` 时
- **THEN** 工作负载不得是该组的成员

#### Scenario: Exists 操作符匹配
- **GIVEN** 组的选择器为 `{key: "version", operator: "exists"}`
- **WHEN** 工作负载具有标签 `version=1.2.3` 时
- **THEN** 工作负载必须是该组的成员
- **WHEN** 工作负载没有 "version" 标签时
- **THEN** 工作负载不得是该组的成员

#### Scenario: 动态成员更新
- **GIVEN** 工作负载不是组 "web-prod" 的成员
- **WHEN** 工作负载标签被更新以匹配组选择器时
- **THEN** 工作负载必须成为该组的成员
- **AND** 成员资格变更必须触发策略重新编译

#### Scenario: 工作负载状态过滤
- **WHEN** 解析组成员资格时
- **THEN** 必须仅包含处于 "running" 状态的工作负载
- **AND** 必须排除已停止、已暂停或已终止的工作负载

---

### Requirement: 组成员查询性能
系统 SHALL 高效地解析组成员资格以供运营使用。

#### Scenario: 小组解析性能
- **GIVEN** 100 个工作负载和 10 个组
- **WHEN** 解析所有组的成员资格时
- **THEN** 总解析时间必须小于 100ms

#### Scenario: 大组解析性能
- **GIVEN** 1000 个工作负载和 50 个组
- **WHEN** 解析所有组的成员资格时
- **THEN** 总解析时间必须小于 1000ms

#### Scenario: 成员资格缓存
- **WHEN** 组成员资格被解析时
- **THEN** 系统可以缓存结果
- **AND** 当工作负载标签发生变化时必须使缓存失效
- **AND** 当组选择器发生变化时必须使缓存失效

---

### Requirement: 工作负载 CRUD 操作
系统 SHALL 提供用于创建、读取、更新和删除工作负载的 API。

#### Scenario: 创建工作负载
- **WHEN** POST /api/v1/workloads 带有有效的工作负载数据时
- **THEN** 必须返回 201 Created
- **AND** 必须返回已创建的工作负载及其分配的 ID

#### Scenario: 列出工作负载
- **WHEN** GET /api/v1/workloads
- **THEN** 必须返回 200 OK
- **AND** 必须返回所有工作负载的数组
- **AND** 必须包含每个工作负载的所有标签

#### Scenario: 通过 ID 获取工作负载
- **WHEN** GET /api/v1/workloads/{id} 带有有效 ID 时
- **THEN** 必须返回 200 OK
- **AND** 必须返回工作负载详情
- **WHEN** GET /api/v1/workloads/{id} 带有不存在的 ID 时
- **THEN** 必须返回 404 Not Found

#### Scenario: 更新工作负载标签
- **WHEN** PUT /api/v1/workloads/{id} 带有更新的标签时
- **THEN** 必须返回 200 OK
- **AND** 必须持久化标签更改
- **AND** 必须触发组成员资格重新评估

#### Scenario: 删除工作负载
- **WHEN** DELETE /api/v1/workloads/{id}
- **THEN** 必须返回 204 No Content
- **AND** 必须从所有组中移除该工作负载
- **AND** 如果该工作负载在活动策略中，必须触发策略重新编译

---

### Requirement: 组 CRUD 操作
系统 SHALL 提供用于管理组的 API。

#### Scenario: 创建组
- **WHEN** POST /api/v1/groups 带有有效选择器时
- **THEN** 必须返回 201 Created
- **AND** 必须返回已创建的组

#### Scenario: 获取组成员
- **WHEN** GET /api/v1/groups/{name}/members
- **THEN** 必须返回 200 OK
- **AND** 必须返回匹配选择器的工作负载数组
- **AND** 必须仅包含运行中的工作负载

#### Scenario: 更新组选择器
- **WHEN** PUT /api/v1/groups/{name} 带有更新的选择器时
- **THEN** 必须返回 200 OK
- **AND** 必须触发成员资格重新评估
- **AND** 必须为所有使用此组的规则触发策略重新编译

#### Scenario: 删除具有活动策略的组
- **WHEN** DELETE /api/v1/groups/{name}
- **AND** 该组被活动策略规则引用时
- **THEN** 必须返回 409 Conflict
- **AND** 必须列出使用该组的策略

#### Scenario: 删除未使用的组
- **WHEN** DELETE /api/v1/groups/{name}
- **AND** 该组未被任何策略引用时
- **THEN** 必须返回 204 No Content

---

### Requirement: 基于标签的策略集成
系统 SHALL 将基于标签的组与策略系统集成。

#### Scenario: 策略规则中的组引用
- **WHEN** 策略规则通过名称引用组时
- **THEN** 系统必须解析组成员资格
- **AND** 必须将规则展开为基于 IP 的策略

#### Scenario: 不存在的组引用
- **WHEN** 策略规则引用不存在的组时
- **THEN** 策略创建必须失败并返回错误
- **AND** 错误消息必须标识缺失的组

#### Scenario: 空组处理
- **WHEN** 策略规则引用成员为零的组时
- **THEN** 策略编译必须成功
- **AND** 必须生成零个已编译策略
- **AND** 应该记录关于空组的警告

---

### Requirement: 工作负载元数据存储
系统 SHALL 存储全面的工作负载元数据以提供运营可见性。

#### Scenario: 容器元数据
- **WHEN** 工作负载是容器时
- **THEN** 必须存储：镜像名称、容器 ID、命名空间（如果适用）

#### Scenario: 网络元数据
- **THEN** 必须存储：所有 IP 地址、MAC 地址、监听端口

#### Scenario: Kubernetes 元数据
- **WHEN** 工作负载是 Kubernetes pod 时
- **THEN** 应该存储：pod 名称、命名空间、服务名称
- **AND** 可以从 pod 标签自动填充标签

#### Scenario: 元数据时间戳
- **THEN** 必须记录：每个工作负载的 created_at、updated_at
- **AND** 时间戳必须采用 UTC 格式

---

### Requirement: 并发工作负载操作
系统 SHALL 安全地处理并发工作负载操作。

#### Scenario: 并发创建工作负载
- **WHEN** 同时创建多个工作负载时
- **THEN** 每个工作负载必须接收唯一的 ID
- **AND** 所有工作负载都必须正确持久化

#### Scenario: 并发标签更新
- **WHEN** 同一个工作负载被并发更新时
- **THEN** 更新必须被序列化
- **AND** 最终状态必须反映最后一次更新

#### Scenario: 并发成员资格解析
- **WHEN** 在工作负载更新期间解析组成员资格时
- **THEN** 解析不得看到部分标签更新
- **AND** 解析必须返回一致的结果

---

### Requirement: 向后兼容性
标签系统 SHALL 与现有的基于 IP 的策略系统共存。

#### Scenario: 混合策略类型
- **WHEN** 系统同时具有基于 IP 的策略和基于标签的策略时
- **THEN** 两种类型都必须被执行
- **AND** 基于 IP 的策略必须继续不变地工作

#### Scenario: 逐步迁移
- **WHEN** 运维人员从基于 IP 的策略迁移到基于标签的策略时
- **THEN** 必须同时支持两种策略类型
- **AND** 在添加基于标签的策略之前不得要求删除基于 IP 的策略

#### Scenario: 数据平面保持不变
- **THEN** eBPF 数据平面不得被修改
- **AND** 编译后的策略必须使用与基于 IP 的策略相同的格式
- **AND** 数据包处理延迟不得增加
