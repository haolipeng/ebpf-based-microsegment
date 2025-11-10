# k8s-workload-sync Specification

## Purpose
定义 Kubernetes 工作负载同步的规范，使用 Informer 模式实时监听 Pod 和 Service 资源变化，并自动同步到工作负载管理系统。

## ADDED Requirements

### Requirement: Pod Informer 初始化

系统必须(SHALL)创建和启动 Pod Informer 以监听 Pod 资源变化。

#### Scenario: Informer 启动和初始化

**Given** Kubernetes 客户端已成功初始化
**And** 工作负载管理器已就绪
**When** 系统启动 Pod Informer
**Then** Informer 必须(SHALL)成功连接到 API Server
**And** Informer 必须(SHALL)执行初始的 List 操作获取现有 Pod
**And** Informer 必须(SHALL)启动 Watch 流监听后续变化
**And** 系统必须(SHALL)记录日志：`"Pod Informer started, initial sync completed"`

#### Scenario: Namespace 过滤配置

**Given** 系统配置包含 Namespace 白名单/黑名单
**When** 创建 Pod Informer
**Then** Informer 必须(SHALL)使用 FieldSelector 或 LabelSelector 过滤 Namespace
**And** 如果配置了 `include` 列表，必须(SHALL)仅监听列表中的 Namespace
**And** 如果配置了 `exclude` 列表，必须(SHALL)排除列表中的 Namespace
**And** 系统必须(SHALL)记录过滤配置：`"Pod Informer watching namespaces: [list]"`

#### Scenario: Informer 本地缓存

**Given** Pod Informer 正在运行
**When** 接收到 Pod 事件
**Then** Informer 必须(SHALL)首先更新本地缓存
**And** 缓存必须(SHALL)维护最新的 ResourceVersion
**And** 系统应该(SHOULD)能够从缓存快速查询 Pod 信息而不调用 API Server

#### Scenario: Informer 重新同步（Resync）

**Given** Pod Informer 正在运行
**When** 达到配置的 Resync 周期（默认 30 分钟）
**Then** Informer 必须(SHALL)触发全量重新同步
**And** 系统必须(SHALL)对所有缓存的 Pod 触发 Update 事件
**And** 系统必须(SHALL)记录日志：`"Pod Informer resync triggered, processing N pods"`

### Requirement: Pod 事件处理

系统必须(SHALL)为 Pod 的 Add/Update/Delete 事件注册处理器。

#### Scenario: Pod Add 事件处理

**Given** Pod Informer 正在运行
**When** 新 Pod 被创建（收到 Add 事件）
**Then** 系统必须(SHALL)提取 Pod 的以下信息：
- Namespace
- Name
- UID
- Pod IP
- Node Name
- Labels
- Annotations
- Status (Running, Pending, etc.)

**And** 系统必须(SHALL)将 Pod 转换为工作负载对象
**And** 工作负载 ID 必须(SHALL)使用格式：`k8s:<namespace>:<uid>`
**And** 系统必须(SHALL)调用 WorkloadManager.CreateWorkload() 或 UpsertWorkload()
**And** 系统必须(SHALL)应用标签映射规则
**And** 系统必须(SHALL)记录日志：`"Pod added: <namespace>/<name> (UID: <uid>, IP: <ip>)"`

#### Scenario: Pod Update 事件处理

**Given** Pod 已存在于系统中
**When** Pod 被更新（收到 Update 事件）
**Then** 系统必须(SHALL)比较 oldPod 和 newPod
**And** 如果标签变化，必须(SHALL)更新工作负载标签
**And** 如果 IP 变化（Pod 重建），必须(SHALL)更新工作负载 IP
**And** 如果状态变化（Running → Terminating），必须(SHALL)更新工作负载状态
**And** 系统必须(SHALL)调用 WorkloadManager.UpdateWorkload()
**And** 系统必须(SHALL)记录日志：`"Pod updated: <namespace>/<name>, changes: [labels|ip|status]"`

#### Scenario: Pod Delete 事件处理

**Given** Pod 已存在于系统中
**When** Pod 被删除（收到 Delete 事件）
**Then** 系统必须(SHALL)调用 WorkloadManager.DeleteWorkload() 使用 workload ID `k8s:<namespace>:<uid>`
**And** 系统必须(SHALL)从所有组中移除该工作负载
**And** 系统必须(SHALL)触发策略重新编译（如果有受影响的策略规则）
**And** 系统必须(SHALL)记录日志：`"Pod deleted: <namespace>/<name> (UID: <uid>)"`

#### Scenario: 忽略无 IP 的 Pod

**Given** 收到 Pod Add 或 Update 事件
**When** Pod 状态为 Pending 且 Pod IP 为空
**Then** 系统必须(SHALL)跳过该 Pod 不创建工作负载
**And** 系统必须(SHALL)记录 Debug 日志：`"Skipping Pod without IP: <namespace>/<name>"`
**And** 当 Pod 获得 IP 后（后续 Update 事件），必须(SHALL)创建工作负载

#### Scenario: 处理 HostNetwork Pod

**Given** Pod 使用 HostNetwork (spec.hostNetwork: true)
**When** 收到 Pod Add 事件
**Then** 系统应该(SHOULD)根据配置决定是否包含该 Pod
**And** 如果配置 `exclude_host_network: true`，必须(SHALL)跳过该 Pod
**And** 如果包含，必须(SHALL)在标签中添加 `network.mode: host`

### Requirement: 标签映射规则

系统必须(SHALL)将 Kubernetes 标签和注解映射到工作负载标签。

#### Scenario: 标准 Kubernetes 标签映射

**Given** Pod 包含 Kubernetes 推荐的标签
**When** 创建或更新工作负载
**Then** 系统必须(SHALL)应用以下映射规则：
- `app.kubernetes.io/name` → `app`
- `app.kubernetes.io/component` → `role`
- `app.kubernetes.io/version` → `version`
- `app.kubernetes.io/instance` → `instance`

**And** 如果 Pod 同时有旧标签（如 `app`），必须(SHALL)使用 `app.kubernetes.io/*` 标签优先

#### Scenario: 环境和位置标签映射

**Given** Pod 包含环境或位置相关标签
**When** 创建或更新工作负载
**Then** 系统必须(SHALL)应用以下映射规则：
- `environment` 或 `env` → `env`
- `topology.kubernetes.io/zone` → `loc` 或 `zone`
- `topology.kubernetes.io/region` → `region`

#### Scenario: 自定义标签透传

**Given** Pod 包含自定义标签（不在标准映射规则中）
**When** 创建或更新工作负载
**Then** 系统必须(SHALL)保留所有原始标签
**And** 标签键必须(SHALL)使用格式：`k8s.label.<original-key>`
**Example**: Pod label `team: backend` → Workload label `k8s.label.team: backend`

#### Scenario: 注解处理

**Given** Pod 包含特定注解
**When** 创建或更新工作负载
**Then** 系统应该(SHOULD)提取以下注解：
- `microsegment.io/labels` - 额外的工作负载标签（JSON 格式）
- `microsegment.io/groups` - 强制加入的组列表（逗号分隔）

**And** 如果 `microsegment.io/labels` 存在，必须(SHALL)解析 JSON 并合并到工作负载标签
**And** 如果解析失败，必须(SHALL)记录警告但不中断处理

#### Scenario: Namespace 作为标签

**Given** Pod 属于某个 Namespace
**When** 创建工作负载
**Then** 系统必须(SHALL)自动添加标签：
- `k8s.namespace: <namespace-name>`
- `k8s.pod.name: <pod-name>`
- `k8s.node.name: <node-name>`

### Requirement: Service Informer（可选功能）

如果启用 Service 同步，系统必须(SHALL)支持监听 Service 资源以发现服务端点。

#### Scenario: Service 到组的映射

**Given** Service Informer 已启用
**When** 新 Service 被创建
**Then** 系统应该(SHOULD)为该 Service 创建一个对应的组
**And** 组名必须(SHALL)使用格式：`k8s-svc:<namespace>:<service-name>`
**And** 组的标签选择器应该(SHOULD)基于 Service 的 selector

**Example**:
```yaml
Service:
  name: mysql
  namespace: production
  selector:
    app: mysql
    role: db
→ 创建组: k8s-svc:production:mysql
   Selector: app=mysql AND role=db
```

#### Scenario: Service 更新处理

**Given** Service 已存在
**When** Service 的 selector 被修改
**Then** 系统应该(SHOULD)更新对应组的标签选择器
**And** 系统应该(SHOULD)触发组成员重新计算
**And** 系统应该(SHOULD)触发相关策略重新编译

#### Scenario: Service 删除处理

**Given** Service 对应的组存在
**When** Service 被删除
**Then** 系统应该(SHOULD)删除对应的组
**And** 系统应该(SHOULD)检查是否有策略规则引用该组
**And** 如果有引用，必须(SHALL)记录警告：`"Service deleted but policy rules still reference group: <group-name>"`

### Requirement: 错误处理和容错

系统必须(SHALL)优雅处理 Informer 运行时错误。

#### Scenario: Watch 连接中断

**Given** Pod Informer 正在运行
**When** API Server 连接中断（网络故障、API Server 重启）
**Then** Informer 必须(SHALL)自动尝试重新建立 Watch 连接
**And** 必须(SHALL)使用指数退避重试（初始 1s, 最大 30s）
**And** 重连成功后，必须(SHALL)从上次的 ResourceVersion 继续 Watch
**And** 系统必须(SHALL)记录日志：`"Pod Informer reconnected after <duration>"`

#### Scenario: ResourceVersion 过期

**Given** Informer 尝试从旧的 ResourceVersion 重新连接
**When** API Server 返回 "410 Gone" 错误
**Then** Informer 必须(SHALL)清空本地缓存
**And** 必须(SHALL)执行完整的 List 操作重新同步
**And** 系统必须(SHALL)记录警告：`"ResourceVersion expired, performing full resync"`

#### Scenario: 事件处理失败

**Given** 收到 Pod Add/Update/Delete 事件
**When** 事件处理器抛出异常（如数据库写入失败）
**Then** 系统必须(SHALL)记录详细错误日志
**And** 系统必须(SHALL)继续处理后续事件（不中断 Informer）
**And** 系统应该(SHOULD)将失败事件加入重试队列
**And** 系统必须(SHALL)在 Resync 时自动修复不一致状态

### Requirement: 性能和资源优化

系统必须(SHALL)优化资源使用避免过载。

#### Scenario: 批量事件处理

**Given** Informer 启动时收到大量 Pod 事件（如集群有 1000+ Pod）
**When** 处理初始同步事件
**Then** 系统应该(SHOULD)使用批量插入/更新数据库
**And** 系统应该(SHOULD)限制并发处理的 Goroutine 数量（默认 10）
**And** 系统必须(SHALL)记录进度：`"Pod sync progress: 100/1000 (10%)"`

#### Scenario: 事件去重

**Given** Informer 可能在短时间内收到同一 Pod 的多个 Update 事件
**When** 处理事件队列
**Then** 系统应该(SHOULD)合并相同 Pod 的连续 Update 事件
**And** 只处理最新版本的事件

#### Scenario: 内存使用限制

**Given** Informer 维护本地缓存
**When** 集群中 Pod 数量很大
**Then** 系统应该(SHOULD)配置 Informer 的缓存大小限制
**And** 系统应该(SHOULD)定期清理不再需要的缓存条目

### Requirement: 监控和可观测性

系统必须(SHALL)提供监控指标和日志。

#### Scenario: Informer 状态监控

**Given** Informer 正在运行
**When** 系统运行中
**Then** 系统必须(SHALL)暴露以下监控指标（如启用 Prometheus）：
- `k8s_informer_sync_total{resource}` - 同步次数
- `k8s_informer_event_total{resource, event_type}` - 事件总数（Add/Update/Delete）
- `k8s_informer_error_total{resource}` - 错误总数
- `k8s_informer_cache_size{resource}` - 缓存中的对象数量

#### Scenario: 同步状态日志

**Given** Informer 执行同步操作
**When** 记录日志
**Then** 日志必须(SHALL)包含以下字段：
- `component`: "k8s-workload-sync"
- `resource`: "Pod" 或 "Service"
- `event_type`: "Add" / "Update" / "Delete"
- `namespace`: Pod/Service 的命名空间
- `name`: Pod/Service 名称
- `uid`: Kubernetes UID
- `duration_ms`: 处理耗时

**Example**:
```json
{
  "level": "info",
  "component": "k8s-workload-sync",
  "resource": "Pod",
  "event_type": "Add",
  "namespace": "production",
  "name": "web-server-abc123",
  "uid": "12345678-1234-1234-1234-123456789abc",
  "ip": "10.0.1.100",
  "labels": {"app": "web", "role": "frontend"},
  "duration_ms": 12,
  "message": "Pod synchronized to workload system"
}
```

### Requirement: 配置示例

系统必须(SHALL)提供清晰的配置示例。

#### Scenario: Workload Sync 配置

**Given** 系统配置文件
**Then** 配置文件必须(SHALL)包含以下工作负载同步配置：

```yaml
# Kubernetes workload synchronization
kubernetes:
  workload_sync:
    # Enable Pod synchronization
    pod_sync:
      enabled: true

      # Namespace filtering
      namespaces:
        # Only watch these namespaces (empty = all)
        include: []
        # Exclude these namespaces
        exclude:
          - kube-system
          - kube-public
          - kube-node-lease

      # Skip Pods with hostNetwork
      exclude_host_network: true

      # Skip Pods without IP (Pending status)
      require_ip: true

      # Resync interval (full resync of all Pods)
      resync_interval: 30m

      # Concurrent workers for event processing
      workers: 10

    # Enable Service synchronization (creates Groups)
    service_sync:
      enabled: false  # Optional feature

      namespaces:
        include: []
        exclude:
          - kube-system
          - kube-public

      resync_interval: 30m

    # Label mapping rules
    label_mapping:
      # Standard Kubernetes labels → Workload labels
      standard:
        "app.kubernetes.io/name": "app"
        "app.kubernetes.io/component": "role"
        "app.kubernetes.io/version": "version"
        "app.kubernetes.io/instance": "instance"

      # Custom label prefix for unmapped labels
      custom_prefix: "k8s.label"

      # Always add these meta labels
      meta_labels:
        - "k8s.namespace"
        - "k8s.pod.name"
        - "k8s.node.name"
```

**And** 配置文件必须(SHALL)包含注释说明每个配置项的用途和默认值
