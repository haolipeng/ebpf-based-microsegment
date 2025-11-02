# Capability: 控制平面 API

## Purpose

控制平面 API 能力提供 RESTful HTTP API，用于管理微隔离系统的策略、监控系统状态和配置参数。它使运维人员能够通过标准 HTTP 接口动态管理 eBPF 数据平面，而无需重新编译或重新加载程序。

## Context

控制平面 API 是微隔离系统的管理界面。它连接数据平面（eBPF 程序和映射）、策略管理器（策略 CRUD）和外部系统（编排器、SIEM、监控仪表板）。

API 必须是：
- 快速响应（< 100ms 用于策略操作）
- 线程安全（支持并发请求）
- 用户友好（RESTful 设计，JSON 格式）
- 可靠（优雅错误处理）

## ADDED Requirements

### Requirement: HTTP API 服务器

系统必须(SHALL)提供 HTTP API 服务器，使用 Gin 框架，在可配置端口上运行（默认 :8080），支持优雅关闭，最小化数据平面性能影响（< 1% CPU）。

#### Scenario: API 服务器启动

**Given** 微隔离代理正在启动
**When** 使用 `--enable-api` 标志启动
**Then** API 服务器必须(SHALL)在配置的端口上启动
**And** 必须(SHALL)日志记录 "API server started on http://host:port"
**And** 必须(SHALL)响应 HTTP 请求

#### Scenario: 优雅关闭

**Given** API 服务器正在运行并处理请求
**When** 收到 SIGINT 或 SIGTERM 信号
**Then** API 服务器必须(SHALL)停止接受新请求
**And** 必须(SHALL)等待正在进行的请求完成（最多 30 秒）
**And** 必须(SHALL)日志记录 "API server stopped gracefully"

### Requirement: 健康检查端点

系统必须(SHALL)提供健康检查端点：GET /api/v1/health（简单健康检查）和 GET /api/v1/status（详细状态）。

#### Scenario: 简单健康检查

**Given** API 服务器正在运行
**When** 客户端请求 GET /api/v1/health
**Then** 必须(SHALL)返回 HTTP 200 OK
**And** 响应必须(SHALL)包含 status 和 uptime 字段

#### Scenario: 详细状态检查

**Given** API 服务器和数据平面正在运行
**When** 客户端请求 GET /api/v1/status
**Then** 必须(SHALL)返回 HTTP 200 OK
**And** 响应必须(SHALL)包含服务器状态、数据平面状态和策略数量

### Requirement: 策略管理 API

系统必须(SHALL)提供完整的策略 CRUD API，包括创建（POST）、列出（GET）、获取单个（GET/:id）、更新（PUT/:id）和删除（DELETE/:id）策略的端点。

#### Scenario: 创建新策略

**Given** API 服务器正在运行
**When** 客户端 POST 到 /api/v1/policies 包含有效的策略 JSON
**Then** 必须(SHALL)返回 HTTP 201 Created
**And** 策略必须(SHALL)添加到 eBPF 映射
**And** 策略必须(SHALL)持久化到存储

#### Scenario: 创建策略 - 无效输入

**Given** API 服务器正在运行
**When** 客户端 POST 无效的策略 JSON（缺少必需字段）
**Then** 必须(SHALL)返回 HTTP 400 Bad Request
**And** 响应必须(SHALL)包含验证错误详情

#### Scenario: 列出所有策略

**Given** 系统中存在多个策略
**When** 客户端请求 GET /api/v1/policies
**Then** 必须(SHALL)返回 HTTP 200 OK
**And** 响应必须(SHALL)包含所有策略的数组

#### Scenario: 获取特定策略

**Given** rule_id=100 的策略存在
**When** 客户端请求 GET /api/v1/policies/100
**Then** 必须(SHALL)返回 HTTP 200 OK
**And** 响应必须(SHALL)包含 rule_id=100 的策略

#### Scenario: 获取不存在的策略

**Given** rule_id=999 的策略不存在
**When** 客户端请求 GET /api/v1/policies/999
**Then** 必须(SHALL)返回 HTTP 404 Not Found

#### Scenario: 更新现有策略

**Given** rule_id=100 的策略存在
**When** 客户端 PUT 到 /api/v1/policies/100 包含更新的策略
**Then** 必须(SHALL)返回 HTTP 200 OK
**And** 策略必须(SHALL)在 eBPF 映射中更新

#### Scenario: 删除策略

**Given** rule_id=100 的策略存在
**When** 客户端 DELETE /api/v1/policies/100
**Then** 必须(SHALL)返回 HTTP 200 OK
**And** 策略必须(SHALL)从 eBPF 映射和存储中删除

### Requirement: 统计信息 API

系统必须(SHALL)提供统计信息查询端点：GET /api/v1/stats（所有统计）、GET /api/v1/stats/packets（数据包统计）、GET /api/v1/stats/sessions（会话统计）和 GET /api/v1/stats/policies（策略统计）。

#### Scenario: 获取所有统计信息

**Given** 数据平面正在处理流量
**When** 客户端请求 GET /api/v1/stats
**Then** 必须(SHALL)返回 HTTP 200 OK
**And** 响应必须(SHALL)包含 total_packets、allowed_packets、denied_packets、new_sessions、active_sessions、policy_hits 和 policy_misses

#### Scenario: 获取数据包统计

**Given** 数据平面已处理数据包
**When** 客户端请求 GET /api/v1/stats/packets
**Then** 必须(SHALL)返回 HTTP 200 OK
**And** 响应必须(SHALL)包含 total、allowed 和 denied 计数

### Requirement: 输入验证

系统必须(SHALL)验证所有 API 输入，包括必需字段存在、字段类型正确、值在有效范围内和 IP 地址格式有效。

#### Scenario: 验证必需字段

**Given** 策略 JSON 缺少 dst_ip 字段
**When** 客户端尝试创建策略
**Then** 必须(SHALL)返回 HTTP 400 Bad Request
**And** 响应必须(SHALL)列出缺少的字段

#### Scenario: 验证 IP 地址格式

**Given** 策略包含无效的 IP 地址
**When** 客户端尝试创建策略
**Then** 必须(SHALL)返回 HTTP 400 Bad Request
**And** 响应必须(SHALL)指示 IP 格式无效

### Requirement: 错误处理

系统必须(SHALL)提供一致的错误响应，包括统一的错误 JSON 格式、适当的 HTTP 状态码（400/404/409/500）、详细的错误消息和错误日志记录。

#### Scenario: 统一错误格式

**Given** API 请求失败
**When** 返回错误响应
**Then** 响应必须(SHALL)包含 error 和 details 字段

#### Scenario: HTTP 状态码映射

**Given** 各种错误条件
**When** API 返回错误
**Then** 必须(SHALL)使用适当的状态码：400（输入验证失败）、404（资源不存在）、409（规则 ID 冲突）、500（服务器错误）

### Requirement: 并发安全

系统必须(SHALL)支持并发 API 请求，包括线程安全的策略访问、无数据竞争、一致的数据视图和适当的锁定机制。

#### Scenario: 并发策略创建

**Given** 两个客户端同时创建不同的策略
**When** 请求并发到达
**Then** 两个策略都必须(SHALL)成功创建
**And** 不得(SHALL NOT)发生数据竞争

#### Scenario: 并发读写

**Given** 一个客户端正在读取策略列表，另一个客户端正在添加新策略
**When** 操作并发发生
**Then** 读取必须(SHALL)返回一致的快照
**And** 写入必须(SHALL)成功完成
**And** 不得(SHALL NOT)发生死锁

### Requirement: 中间件

系统必须(SHALL)提供 HTTP 中间件：Recovery（捕获 panic）、Logger（记录请求/响应）和 CORS（支持跨域请求）。

#### Scenario: Panic 恢复

**Given** 处理器函数发生 panic
**When** 处理请求
**Then** 必须(SHALL)捕获 panic 并返回 HTTP 500
**And** 服务器不得(SHALL NOT)崩溃

#### Scenario: 请求日志记录

**Given** 客户端发送 API 请求
**When** 请求被处理
**Then** 必须(SHALL)记录请求方法、路径、响应状态码和处理时间

### Requirement: 性能

系统必须(SHALL)满足性能要求：策略 CRUD 操作 < 10ms、统计查询 < 50ms、API 服务器 CPU 开销 < 1%、无数据平面性能影响。

#### Scenario: 策略创建性能

**Given** API 服务器正在运行
**When** 客户端创建新策略
**Then** 响应时间必须(SHALL) < 10ms
**And** 数据平面数据包处理不得(SHALL NOT)延迟

#### Scenario: 统计查询性能

**Given** 数据平面正在高负载下运行
**When** 客户端查询统计信息
**Then** 响应时间必须(SHALL) < 50ms

## Data Structures

**策略请求/响应**：包含 rule_id、src_ip、dst_ip、src_port、dst_port、protocol、action 和 priority 字段。

**统计响应**：包含 total_packets、allowed_packets、denied_packets、new_sessions、active_sessions、policy_hits 和 policy_misses 字段。

**健康响应**：包含 status、uptime、dataplane 和 policies 字段。

**错误响应**：包含 error 和 details 字段。

## Implementation Notes

- API 服务器在单独的 goroutine 中运行
- 使用 Gin 框架进行路由和中间件
- PolicyManager 内置线程安全（sync.RWMutex）
- 所有端点返回 JSON（Content-Type: application/json）
- 支持通过命令行标志配置（--api-host、--api-port、--enable-api）
- 优雅关闭等待正在进行的请求（30 秒超时）

## Performance Characteristics

- 策略 CRUD: < 10ms 平均响应时间
- 统计查询: < 50ms 平均响应时间
- CPU 开销: < 1% 在空闲时
- 内存: < 10MB
- 并发: 支持数百个并发请求

## Related Capabilities

- **Policy Management** - API 公开策略 CRUD 操作
- **Session Tracking** - API 返回会话统计
- **Statistics Reporting** - API 公开所有统计端点
- **Dataplane Performance** - API 对数据平面影响最小
