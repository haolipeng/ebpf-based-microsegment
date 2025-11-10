# k8s-api-client Specification

## Purpose
定义 Kubernetes API 客户端的规范，支持 in-cluster 和 out-of-cluster 配置，提供可靠的 Kubernetes 资源访问能力。

## ADDED Requirements

### Requirement: Kubernetes 客户端初始化

系统必须(SHALL)支持创建和配置 Kubernetes API 客户端，支持多种配置模式。

#### Scenario: In-Cluster 配置模式

**Given** Agent 部署在 Kubernetes 集群内（DaemonSet 或 Deployment）
**And** ServiceAccount token 和 CA 证书存在于 `/var/run/secrets/kubernetes.io/serviceaccount/`
**When** 系统初始化 K8s 客户端使用 in-cluster 配置
**Then** 客户端必须(SHALL)成功连接到 Kubernetes API Server
**And** 客户端必须(SHALL)使用 ServiceAccount 的身份进行认证
**And** 客户端必须(SHALL)记录日志："Kubernetes client initialized in in-cluster mode"

#### Scenario: Out-of-Cluster 配置模式（开发环境）

**Given** Agent 运行在 Kubernetes 集群外部
**And** Kubeconfig 文件存在于 `~/.kube/config` 或 `KUBECONFIG` 环境变量指定的路径
**When** 系统初始化 K8s 客户端使用 out-of-cluster 配置
**Then** 客户端必须(SHALL)成功读取 Kubeconfig 文件
**And** 客户端必须(SHALL)使用 Kubeconfig 中的 current-context
**And** 客户端必须(SHALL)记录日志："Kubernetes client initialized from kubeconfig: <path>"

#### Scenario: 自动检测配置模式

**Given** 系统启动时未明确指定配置模式
**When** 系统尝试初始化 K8s 客户端
**Then** 系统必须(SHALL)首先尝试 in-cluster 配置
**And** 如果 in-cluster 配置失败，必须(SHALL)回退到 out-of-cluster 配置
**And** 系统必须(SHALL)记录使用的配置模式

#### Scenario: 客户端初始化失败处理

**Given** K8s 客户端配置不可用（无 ServiceAccount 且无 Kubeconfig）
**When** 系统尝试初始化 K8s 客户端
**Then** 系统必须(SHALL)返回清晰的错误消息
**And** 系统必须(SHALL)支持在无 K8s 集成模式下继续运行
**And** 系统必须(SHALL)记录警告日志："Kubernetes integration disabled: <reason>"

### Requirement: Kubernetes API 健康检查

系统必须(SHALL)实现 Kubernetes API Server 的健康检查机制。

#### Scenario: API Server 连通性检查

**Given** K8s 客户端已初始化
**When** 系统执行健康检查
**Then** 系统必须(SHALL)向 API Server 发送 GET /api/v1 请求
**And** 如果请求成功，必须(SHALL)返回健康状态
**And** 如果请求失败，必须(SHALL)返回错误状态和详细错误信息

#### Scenario: 定期健康检查

**Given** K8s 集成已启用
**When** 系统运行中
**Then** 系统必须(SHALL)每 30 秒执行一次健康检查
**And** 如果健康检查连续失败 3 次，必须(SHALL)记录错误日志
**And** 系统必须(SHALL)尝试重新初始化客户端

### Requirement: RBAC 权限验证

系统必须(SHALL)验证 ServiceAccount 是否具有必要的 RBAC 权限。

#### Scenario: 权限检查 - Pod 资源

**Given** K8s 客户端已初始化
**When** 系统启动时验证权限
**Then** 系统必须(SHALL)执行 SelfSubjectAccessReview 检查 Pod.list 权限
**And** 系统必须(SHALL)执行 SelfSubjectAccessReview 检查 Pod.watch 权限
**And** 如果权限不足，必须(SHALL)记录清晰的错误消息并提供 RBAC 配置建议

#### Scenario: 权限检查 - Service 资源

**Given** K8s 客户端已初始化
**When** 系统启动时验证权限
**Then** 系统必须(SHALL)执行 SelfSubjectAccessReview 检查 Service.list 权限
**And** 系统必须(SHALL)执行 SelfSubjectAccessReview 检查 Service.watch 权限
**And** 如果权限不足，必须(SHALL)记录警告但允许系统继续运行（部分功能禁用）

### Requirement: 客户端配置选项

系统必须(SHALL)支持灵活的客户端配置。

#### Scenario: 配置 API Server 地址

**Given** 系统配置文件包含 `kubernetes.api_server` 配置项
**When** 系统初始化 K8s 客户端
**Then** 客户端必须(SHALL)使用配置的 API Server 地址覆盖默认值

#### Scenario: 配置请求超时

**Given** 系统配置文件包含 `kubernetes.timeout` 配置项
**When** 系统向 API Server 发送请求
**Then** 请求必须(SHALL)使用配置的超时时间（默认 30 秒）
**And** 如果超时，必须(SHALL)返回明确的超时错误

#### Scenario: 配置 QPS 和 Burst

**Given** 系统配置文件包含 `kubernetes.qps` 和 `kubernetes.burst` 配置项
**When** 系统初始化 K8s 客户端
**Then** 客户端必须(SHALL)应用 QPS 限制（默认 5 QPS）
**And** 客户端必须(SHALL)应用 Burst 限制（默认 10 Burst）
**And** 系统必须(SHALL)遵守 Rate Limiting 避免 API Server 过载

### Requirement: 错误处理和重试机制

系统必须(SHALL)实现健壮的错误处理和重试机制。

#### Scenario: 网络故障重试

**Given** K8s 客户端已初始化
**When** 向 API Server 发送请求遇到网络错误（Connection refused, Timeout）
**Then** 系统必须(SHALL)使用指数退避重试（初始 1s, 最大 30s）
**And** 系统必须(SHALL)最多重试 5 次
**And** 系统必须(SHALL)记录每次重试的日志

#### Scenario: API 错误处理

**Given** K8s 客户端已初始化
**When** API Server 返回错误（401 Unauthorized, 403 Forbidden, 404 Not Found）
**Then** 系统必须(SHALL)根据错误类型采取不同的处理策略：
- 401/403: 记录错误并停止重试（需要修复 RBAC）
- 404: 记录警告但不重试（资源不存在）
- 5xx: 使用指数退避重试

**And** 系统必须(SHALL)在日志中包含完整的错误上下文（HTTP 状态码、错误消息、请求路径）

### Requirement: 日志和监控集成

系统必须(SHALL)提供详细的日志记录和监控指标。

#### Scenario: 结构化日志记录

**Given** K8s 客户端执行任何操作
**When** 记录日志
**Then** 日志必须(SHALL)包含以下字段：
- `component`: "k8s-client"
- `operation`: 操作类型（init, list, watch, get）
- `resource`: 资源类型（Pod, Service）
- `duration_ms`: 请求耗时（毫秒）

**And** 错误日志必须(SHALL)包含 `error` 字段和堆栈跟踪

#### Scenario: 监控指标（可选，未来实现）

**Given** Prometheus 集成已启用
**When** K8s 客户端执行操作
**Then** 系统应该(SHOULD)记录以下指标：
- `k8s_api_requests_total{operation, status}` - 请求总数
- `k8s_api_request_duration_seconds{operation}` - 请求延迟
- `k8s_api_errors_total{operation, error_type}` - 错误总数

### Requirement: 配置文件示例

系统必须(SHALL)提供清晰的配置文件示例。

#### Scenario: Agent 配置文件包含 K8s 集成配置

**Given** 系统提供配置文件模板
**Then** 配置文件必须(SHALL)包含以下 Kubernetes 相关配置：

```yaml
# Kubernetes integration (optional)
kubernetes:
  # Enable Kubernetes integration
  # If false, the agent will not attempt to connect to Kubernetes API
  enabled: true

  # Configuration mode: auto, in-cluster, kubeconfig
  # - auto: Try in-cluster first, fallback to kubeconfig
  # - in-cluster: Use ServiceAccount (for DaemonSet deployments)
  # - kubeconfig: Use kubeconfig file
  config_mode: auto

  # Kubeconfig file path (only used when config_mode is kubeconfig)
  # Default: ~/.kube/config or $KUBECONFIG
  kubeconfig_path: ""

  # API Server address override (optional)
  # If not set, will use the address from kubeconfig or in-cluster service
  api_server: ""

  # Request timeout in seconds
  timeout: 30

  # Rate limiting
  qps: 5      # Queries per second
  burst: 10   # Burst size

  # Health check interval in seconds
  health_check_interval: 30
```

**And** 配置文件必须(SHALL)包含注释说明每个配置项的用途
