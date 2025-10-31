# Capability: 控制平面 API

## ADDED Requirements

### Requirement: HTTP API 服务器
系统必须(SHALL)为控制平面操作提供 HTTP API 服务器。

#### Scenario: API 服务器成功启动
- **WHEN** 代理在启用 API 的情况下启动
- **THEN** HTTP 服务器必须(MUST)监听配置的端口（默认 8080）
- **AND** 所有端点必须(MUST)可访问

#### Scenario: API 服务器优雅关闭
- **WHEN** 代理收到 SIGTERM 或 SIGINT
- **THEN** API 服务器必须(MUST)完成正在进行的请求
- **AND** 拒绝新请求
- **AND** 在 30 秒内关闭

---

### Requirement: 策略管理端点
API 必须(SHALL)为策略 CRUD 操作提供 RESTful 端点。

#### Scenario: 通过 API 创建策略
- **WHEN** POST /api/v1/policies 并附带有效的策略 JSON
- **THEN** 返回 201 Created 及策略详情
- **AND** 策略必须(MUST)添加到 eBPF policy_map
- **AND** 返回包含策略 URL 的 Location 头

#### Scenario: 列出所有策略
- **WHEN** GET /api/v1/policies
- **THEN** 返回 200 OK 及所有策略的数组
- **AND** 响应时间必须(MUST) < 100ms

#### Scenario: 获取特定策略
- **WHEN** GET /api/v1/policies/:id 并附带有效 ID
- **THEN** 返回 200 OK 及策略详情
- **WHEN** 策略 ID 不存在
- **THEN** 返回 404 Not Found

#### Scenario: 更新策略
- **WHEN** PUT /api/v1/policies/:id 并附带有效更新
- **THEN** 返回 200 OK 及更新后的策略
- **AND** eBPF policy_map 必须(MUST)被更新
- **WHEN** 策略 ID 不存在
- **THEN** 返回 404 Not Found

#### Scenario: 删除策略
- **WHEN** DELETE /api/v1/policies/:id
- **THEN** 返回 204 No Content
- **AND** 策略必须(MUST)从 eBPF policy_map 中移除
- **WHEN** 策略 ID 不存在
- **THEN** 返回 404 Not Found

#### Scenario: 无效的策略数据
- **WHEN** POST/PUT 附带无效 JSON 或缺少必需字段
- **THEN** 返回 400 Bad Request 及验证错误

#### Scenario: 重复策略
- **WHEN** 创建与现有策略具有相同 5 元组的策略
- **THEN** 返回 409 Conflict 及错误消息

---

### Requirement: 统计端点
API 必须(SHALL)提供端点以查询数据平面统计信息。

#### Scenario: 获取所有统计信息
- **WHEN** GET /api/v1/stats
- **THEN** 返回 200 OK 及完整统计信息
- **AND** 包括数据包计数器（total、allowed、denied）
- **AND** 包括会话计数器（new、active、closed）
- **AND** 包括策略计数器（hits、misses）
- **AND** 响应时间必须(MUST) < 50ms

#### Scenario: 获取数据包统计信息
- **WHEN** GET /api/v1/stats/packets
- **THEN** 返回 200 OK 及数据包特定统计信息

#### Scenario: 获取会话统计信息
- **WHEN** GET /api/v1/stats/sessions
- **THEN** 返回 200 OK 及会话特定统计信息

#### Scenario: 获取策略统计信息
- **WHEN** GET /api/v1/stats/policies
- **THEN** 返回 200 OK 及策略特定统计信息

---

### Requirement: 健康检查端点
API 必须(SHALL)提供用于监控的健康检查端点。

#### Scenario: 基本健康检查
- **WHEN** GET /api/v1/health
- **THEN** 如果系统健康则返回 200 OK
- **AND** 如果数据平面未运行则返回 503 Service Unavailable

#### Scenario: 详细状态
- **WHEN** GET /api/v1/status
- **THEN** 返回 200 OK 及详细系统状态
- **AND** 包括数据平面状态
- **AND** 包括已加载的 eBPF 程序
- **AND** 包括运行时间

---

### Requirement: 配置端点
API 必须(SHALL)提供端点以查看和更新配置。

#### Scenario: 获取当前配置
- **WHEN** GET /api/v1/config
- **THEN** 返回 200 OK 及当前配置

#### Scenario: 更新配置
- **WHEN** PUT /api/v1/config 并附带有效更新
- **THEN** 返回 200 OK 及更新后的配置
- **AND** 配置必须(MUST)立即应用
- **WHEN** 配置无效
- **THEN** 返回 400 Bad Request 及验证错误

---

### Requirement: 错误处理
API 必须(SHALL)在所有端点上提供一致的错误响应。

#### Scenario: 结构化错误响应
- **WHEN** 任何端点返回错误
- **THEN** 响应必须(MUST)包含错误对象，包含：
  - `code`：机器可读的错误代码
  - `message`：人类可读的描述
  - `details`：附加上下文（可选）

#### Scenario: 错误状态码
- **THEN** API 必须(MUST)使用适当的 HTTP 状态码：
  - 200：成功
  - 201：已创建
  - 204：无内容
  - 400：错误请求（验证错误）
  - 404：未找到
  - 409：冲突
  - 500：内部服务器错误
  - 503：服务不可用

---

### Requirement: 请求验证
API 必须(SHALL)在处理之前验证所有传入请求。

#### Scenario: 输入验证
- **WHEN** 请求包含无效的数据类型
- **THEN** 返回 400 Bad Request 及特定字段错误
- **WHEN** 缺少必需字段
- **THEN** 返回 400 Bad Request 列出缺少的字段

#### Scenario: IP 地址验证
- **WHEN** 策略包含无效的 IP 地址
- **THEN** 返回 400 Bad Request 及"无效 IP 地址"错误

#### Scenario: 端口验证
- **WHEN** 策略包含 0-65535 范围之外的端口
- **THEN** 返回 400 Bad Request 及"无效端口"错误

#### Scenario: 协议验证
- **WHEN** 策略包含不支持的协议
- **THEN** 返回 400 Bad Request 及支持的协议列表

---

### Requirement: 并发访问安全
API 必须(SHALL)安全地处理并发请求，不会出现数据损坏。

#### Scenario: 并发策略更新
- **WHEN** 多个客户端同时更新策略
- **THEN** 所有更新必须(MUST)原子地应用
- **AND** 不得(MUST NOT)丢失任何策略更新

#### Scenario: 读写并发
- **WHEN** 客户端在更新进行中时读取策略
- **THEN** 读取必须(MUST)返回一致的数据
- **AND** 读取不得(MUST NOT)不必要地阻塞写入

---

### Requirement: 性能
API 必须(SHALL)保持高性能，不影响数据平面。

#### Scenario: API 延迟目标
- **WHEN** 执行任何策略 CRUD 操作
- **THEN** 响应时间必须(MUST) < 10ms（p95）
- **WHEN** 查询统计信息
- **THEN** 响应时间必须(MUST) < 50ms（p95）

#### Scenario: 无数据平面影响
- **WHEN** API 服务器运行时
- **THEN** 数据平面数据包处理延迟不得(MUST NOT)增加
- **AND** CPU 开销必须(MUST) < 1% 额外开销

#### Scenario: 并发请求处理
- **WHEN** API 每秒接收 1000 个请求
- **THEN** 所有请求必须(MUST)无错误地处理
- **AND** 响应时间必须(MUST)保持在目标范围内

---

### Requirement: API 文档
API 必须(SHALL)提供全面的文档。

#### Scenario: OpenAPI 规范
- **THEN** API 必须(MUST)提供 OpenAPI 3.0 规范
- **AND** 规范必须(MUST)在 /api/docs/openapi.json 提供

#### Scenario: 交互式文档
- **THEN** API 必须(MUST)在 /api/docs 提供 Swagger UI
- **AND** 允许交互式测试端点

---

### Requirement: CORS 支持
API 必须(SHALL)为 Web 客户端支持跨源资源共享。

#### Scenario: CORS 头
- **WHEN** API 接收 OPTIONS 请求
- **THEN** 返回适当的 CORS 头
- **AND** 允许配置的源（默认：localhost）

---

### Requirement: 请求日志记录
API 必须(SHALL)记录所有请求以进行调试和审计。

#### Scenario: 访问日志记录
- **WHEN** 接收任何 API 请求
- **THEN** 记录请求方法、路径、状态码和延迟
- **AND** 包括用于关联的请求 ID

