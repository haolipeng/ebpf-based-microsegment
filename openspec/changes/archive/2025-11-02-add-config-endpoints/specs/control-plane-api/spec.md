# control-plane-api Specification Delta

## ADDED Requirements

### Requirement: 配置管理 API

系统必须(SHALL)提供配置管理 API，包括 GET /api/v1/config（查询当前配置）和 PUT /api/v1/config（更新运行时配置），支持动态调整日志级别和统计间隔，无需重启服务。

#### Scenario: 获取当前配置

**Given** API 服务器正在运行
**When** 客户端请求 GET /api/v1/config
**Then** 必须(SHALL)返回 HTTP 200 OK
**And** 响应必须(SHALL)包含 interface、log_level、stats_interval、api_host 和 api_port 字段
**And** 所有字段必须(SHALL)反映当前运行时配置

#### Scenario: 更新日志级别

**Given** API 服务器以 info 日志级别运行
**When** 客户端发送 PUT /api/v1/config 请求，body 为 {"log_level": "debug"}
**Then** 必须(SHALL)返回 HTTP 200 OK
**And** 响应必须(SHALL)显示 log_level 为 "debug"
**And** 日志系统必须(SHALL)立即切换到 debug 级别
**And** 必须(SHALL)记录日志级别变更信息

#### Scenario: 更新统计间隔

**Given** API 服务器以 5 秒统计间隔运行
**When** 客户端发送 PUT /api/v1/config 请求，body 为 {"stats_interval": 10}
**Then** 必须(SHALL)返回 HTTP 200 OK
**And** 响应必须(SHALL)显示 stats_interval 为 10
**And** 必须(SHALL)记录统计间隔变更信息

#### Scenario: 同时更新多个字段

**Given** API 服务器正在运行
**When** 客户端发送 PUT /api/v1/config 请求，body 为 {"log_level": "warn", "stats_interval": 15}
**Then** 必须(SHALL)返回 HTTP 200 OK
**And** 响应必须(SHALL)显示 log_level 为 "warn"
**And** 响应必须(SHALL)显示 stats_interval 为 15
**And** 两个配置必须(SHALL)同时生效

#### Scenario: 无效的日志级别

**Given** API 服务器正在运行
**When** 客户端发送 PUT /api/v1/config 请求，body 为 {"log_level": "invalid"}
**Then** 必须(SHALL)返回 HTTP 400 Bad Request
**And** 错误消息必须(SHALL)指示 "Invalid request format"
**And** 当前日志级别必须(SHALL)保持不变

#### Scenario: 统计间隔超出范围 - 过低

**Given** API 服务器正在运行
**When** 客户端发送 PUT /api/v1/config 请求，body 为 {"stats_interval": 0}
**Then** 必须(SHALL)返回 HTTP 400 Bad Request
**And** 错误消息必须(SHALL)指示验证失败
**And** 当前统计间隔必须(SHALL)保持不变

#### Scenario: 统计间隔超出范围 - 过高

**Given** API 服务器正在运行
**When** 客户端发送 PUT /api/v1/config 请求，body 为 {"stats_interval": 301}
**Then** 必须(SHALL)返回 HTTP 400 Bad Request
**And** 错误消息必须(SHALL)指示验证失败
**And** 当前统计间隔必须(SHALL)保持不变

#### Scenario: 未提供任何字段

**Given** API 服务器正在运行
**When** 客户端发送 PUT /api/v1/config 请求，body 为空 JSON 对象 {}
**Then** 必须(SHALL)返回 HTTP 400 Bad Request
**And** 错误消息必须(SHALL)指示 "No configuration fields provided"

#### Scenario: 无效的 JSON 格式

**Given** API 服务器正在运行
**When** 客户端发送 PUT /api/v1/config 请求，body 为格式错误的 JSON
**Then** 必须(SHALL)返回 HTTP 400 Bad Request
**And** 错误消息必须(SHALL)指示 "Invalid request format"

#### Scenario: 只读字段保护

**Given** API 服务器在接口 "lo"、主机 "127.0.0.1"、端口 8080 上运行
**When** 客户端发送任何有效的 PUT /api/v1/config 请求
**Then** interface、api_host 和 api_port 字段必须(SHALL)保持不变
**And** 响应必须(SHALL)返回原始值

### Requirement: 配置验证规则

系统必须(SHALL)对配置更新请求执行严格验证，log_level 只允许 debug、info、warn、error，stats_interval 必须在 1-300 秒范围内，拒绝无效值并返回明确的错误消息。

#### Scenario: 日志级别枚举验证

**Given** API 服务器正在运行
**When** 客户端尝试设置 log_level 为非 (debug|info|warn|error) 的值
**Then** 必须(SHALL)返回 HTTP 400 Bad Request
**And** 必须(SHALL)拒绝该配置变更

#### Scenario: 统计间隔范围验证

**Given** API 服务器正在运行
**When** 客户端尝试设置 stats_interval < 1 或 > 300
**Then** 必须(SHALL)返回 HTTP 400 Bad Request
**And** 必须(SHALL)拒绝该配置变更

#### Scenario: 验证失败时的回滚

**Given** API 服务器以 info 日志级别运行
**When** 客户端发送包含无效日志级别的配置更新
**Then** 验证失败后，日志级别必须(SHALL)保持为 info
**And** 必须(SHALL)不应用任何部分更改
