# 实施任务

## 阶段 1：Kubernetes API 客户端基础（第 1 周）✅ **已完成**

### 任务 1.1：添加 client-go 依赖 ✅
- ✅ 添加 `k8s.io/client-go` v0.31.0+ 到 go.mod
- ✅ 添加 `k8s.io/api` v0.31.0+
- ✅ 添加 `k8s.io/apimachinery` v0.31.0+
- ✅ 运行 `go mod tidy` 并验证无冲突

**交付物**：更新的 go.mod 和 go.sum
**预估时间**：30 分钟
**状态**：✅ 已完成

### 任务 1.2：实现 K8s 客户端初始化 ✅
- ✅ 创建 `src/agent/pkg/k8s/client.go`
- ✅ 实现 `NewClient(config *Config) (*Client, error)` 函数
- ✅ 支持 in-cluster 配置（使用 `rest.InClusterConfig()`）
- ✅ 支持 out-of-cluster 配置（使用 `clientcmd.BuildConfigFromFlags()`）
- ✅ 实现自动检测模式（先尝试 in-cluster，失败则回退到 kubeconfig）

**文件**：`src/agent/pkg/k8s/client.go`
**测试**：`src/agent/pkg/k8s/client_test.go`
**规范**：[k8s-api-client](./specs/k8s-api-client/spec.md) - Kubernetes 客户端初始化
**预估时间**：4 小时
**状态**：✅ 已完成

### 任务 1.3：实现健康检查 ✅
- ✅ 添加 `HealthCheck() error` 方法到 Client
- ✅ 发送 GET /api/v1 请求验证 API Server 连通性
- ✅ 实现定期健康检查（可配置间隔，默认 30 秒）
- ✅ 添加指数退避重连逻辑

**文件**：`src/agent/pkg/k8s/health.go`
**测试**：`src/agent/pkg/k8s/health_test.go`
**规范**：[k8s-api-client](./specs/k8s-api-client/spec.md) - Kubernetes API 健康检查
**预估时间**：3 小时
**状态**：✅ 已完成

### 任务 1.4：实现 RBAC 权限检查 ✅
- ✅ 添加 `CheckPermissions() error` 方法
- ✅ 使用 `SelfSubjectAccessReview` 验证 Pod.list 和 Pod.watch 权限
- ✅ 检查 Service.list 和 Service.watch 权限
- ✅ 返回清晰的错误消息和 RBAC 配置建议

**文件**：`src/agent/pkg/k8s/rbac.go`
**测试**：`src/agent/pkg/k8s/rbac_test.go`
**规范**：[k8s-api-client](./specs/k8s-api-client/spec.md) - RBAC 权限验证
**预估时间**：3 小时
**状态**：✅ 已完成

### 任务 1.5：配置集成 ✅
- ✅ 在 `src/agent/pkg/config/config.go` 创建 `K8sConfig` 结构体
- ✅ 添加配置字段：enabled、config_mode、kubeconfig_path、api_server、timeout、qps、burst
- ✅ 实现配置验证
- ✅ 更新 Agent 初始化逻辑，条件性启用 K8s 集成

**文件**：`src/agent/pkg/config/config.go`、`src/agent/cmd/agent/main.go`
**规范**：[k8s-api-client](./specs/k8s-api-client/spec.md) - 客户端配置选项
**预估时间**：2 小时
**状态**：✅ 已完成

---

## 阶段 2：Pod Informer 和事件处理（第 2 周）

### 任务 2.1：创建 Pod Informer
- 创建 `src/agent/pkg/k8s/informer.go`
- 实现 `NewPodInformer(client *Client, namespace string) cache.SharedIndexInformer`
- 配置 Informer 的 ListWatch 监听 Pod
- 设置 IndexInformer 支持 namespace 索引
- 配置重新同步间隔（默认 30 分钟）

**文件**：`src/agent/pkg/k8s/informer.go`
**测试**：`src/agent/pkg/k8s/informer_test.go`（使用 fake clientset）
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - Pod Informer 初始化
**预估时间**：4 小时

### 任务 2.2：实现 Namespace 过滤 ✅
- ✅ 添加 namespace 过滤配置（include/exclude 列表）
- ✅ 在 Informer 创建时实现过滤逻辑
- ✅ 支持通配符模式（例如：`prod-*`）
- ✅ 添加 namespace 过滤配置验证

**文件**：`src/agent/pkg/k8s/namespace_filter.go`
**测试**：`src/agent/pkg/k8s/namespace_filter_test.go`
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - Namespace 过滤配置
**预估时间**：3 小时
**状态**：✅ 已完成

### 任务 2.3：实现 Pod 事件处理器
- 创建 `src/agent/pkg/k8s/pod_handler.go`
- 实现 `OnAdd(obj interface{})` 处理器
- 实现 `OnUpdate(oldObj, newObj interface{})` 处理器
- 实现 `OnDelete(obj interface{})` 处理器
- 提取 Pod 元数据：Namespace、Name、UID、IP、Node、Labels、Annotations

**文件**：`src/agent/pkg/k8s/pod_handler.go`
**测试**：`src/agent/pkg/k8s/pod_handler_test.go`
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - Pod 事件处理
**预估时间**：6 小时

### 任务 2.4：Pod 到 Workload 转换
- 创建 `src/agent/pkg/k8s/converter.go`
- 实现 `PodToWorkload(pod *corev1.Pod) (*workload.Workload, error)`
- 生成 workload ID：`k8s:<namespace>:<uid>`
- 处理无 IP 的 Pod（跳过或标记为 pending）
- 处理 HostNetwork Pod（可配置是否包含）

**文件**：`src/agent/pkg/k8s/converter.go`
**测试**：`src/agent/pkg/k8s/converter_test.go`
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - 忽略无 IP 的 Pod、处理 HostNetwork Pod
**预估时间**：4 小时

---

## 阶段 3：标签映射和同步（第 2-3 周）

### 任务 3.1：实现标签映射规则 ✅
- ✅ 创建标签映射功能（在 `src/agent/pkg/k8s/converter.go` 中实现）
- ✅ 实现标准 K8s 标签映射：
  - `app.kubernetes.io/name` → `app`
  - `app.kubernetes.io/component` → `role`
  - `app.kubernetes.io/version` → `version`
  - `app.kubernetes.io/instance` → `instance`
- ✅ 实现环境和位置映射：
  - `environment`/`env` → `env`
  - `topology.kubernetes.io/zone` → `loc`
  - `topology.kubernetes.io/region` → `region`
- ✅ 实现优先级机制和自定义标签保留
- ✅ 导出 `MapPodLabels()` 函数供外部使用

**文件**：`src/agent/pkg/k8s/converter.go`
**测试**：`src/agent/pkg/k8s/converter_test.go`
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - 标签映射规则
**预估时间**：4 小时
**状态**：✅ 已完成

### 任务 3.2：实现注解处理
- 解析 `microsegment.io/labels` 注解（JSON 格式）
- 解析 `microsegment.io/groups` 注解（逗号分隔列表）
- 将注解中的标签与 Pod 标签合并
- 优雅处理注解解析错误

**文件**：`src/agent/pkg/k8s/annotation_parser.go`
**测试**：`src/agent/pkg/k8s/annotation_parser_test.go`
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - 注解处理
**预估时间**：3 小时

### 任务 3.3：添加元数据标签
- 自动添加 `k8s.namespace: <namespace>`
- 自动添加 `k8s.pod.name: <pod-name>`
- 自动添加 `k8s.node.name: <node-name>`
- 使元数据标签可配置

**文件**：更新 `src/agent/pkg/k8s/label_mapper.go`
**测试**：更新 `src/agent/pkg/k8s/label_mapper_test.go`
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - Namespace 作为标签
**预估时间**：2 小时

### 任务 3.4：集成 Workload Manager
- 在 pod_handler 中添加 Workload Manager 依赖
- 实现工作负载创建：`workloadMgr.CreateWorkload(wl)`
- 实现工作负载更新：`workloadMgr.UpdateWorkload(wl)`
- 实现工作负载删除：`workloadMgr.DeleteWorkload(id)`
- 处理并发更新（如果可用则使用 Upsert）

**文件**：更新 `src/agent/pkg/k8s/pod_handler.go`
**测试**：`src/agent/pkg/k8s/integration_test.go`（与真实 Workload Manager 的集成测试）
**预估时间**：4 小时

---

## 阶段 4：Service Informer（可选，第 3 周）

### 任务 4.1：实现 Service Informer
- 创建 `src/agent/pkg/k8s/service_informer.go`
- 实现 `NewServiceInformer(client *Client) cache.SharedIndexInformer`
- 设置 Service 事件处理器

**文件**：`src/agent/pkg/k8s/service_informer.go`
**测试**：`src/agent/pkg/k8s/service_informer_test.go`
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - Service Informer
**预估时间**：3 小时

### 任务 4.2：Service 到 Group 映射
- 实现 Service 事件处理器
- 为每个 Service 创建 Group：`k8s-svc:<namespace>:<service-name>`
- 将 Service selector 转换为 Group 标签选择器
- 集成 Group Manager

**文件**：`src/agent/pkg/k8s/service_handler.go`
**测试**：`src/agent/pkg/k8s/service_handler_test.go`
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - Service 到组的映射
**预估时间**：4 小时

---

## 阶段 5：错误处理和容错（第 3-4 周）

### 任务 5.1：实现重试和重连
- 为 Watch 重连添加指数退避（初始 1s，最大 30s）
- 处理 "410 Gone" ResourceVersion 过期
- 在 ResourceVersion 过期时实现完整重新同步
- 添加连接状态跟踪

**文件**：`src/agent/pkg/k8s/error_handler.go`
**测试**：`src/agent/pkg/k8s/error_handler_test.go`
**规范**：[k8s-api-client](./specs/k8s-api-client/spec.md) - 网络故障重试
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - Watch 连接中断、ResourceVersion 过期
**预估时间**：4 小时

### 任务 5.2：事件处理错误处理
- 用错误恢复包装事件处理器
- 为失败事件添加重试队列
- 实现永久失败事件的死信队列
- 记录详细错误上下文（HTTP 状态码、错误消息、请求路径）

**文件**：更新 `src/agent/pkg/k8s/pod_handler.go`
**测试**：`src/agent/pkg/k8s/error_recovery_test.go`
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - 事件处理失败
**预估时间**：4 小时

### 任务 5.3：优雅降级
- 支持在禁用 K8s 集成时运行
- 优雅处理 RBAC 权限错误
- 即使个别处理器失败也继续处理事件

**文件**：更新 `src/agent/cmd/agent/main.go`、`src/agent/pkg/k8s/client.go`
**规范**：[k8s-api-client](./specs/k8s-api-client/spec.md) - 客户端初始化失败处理
**预估时间**：2 小时

---

## 阶段 6：性能优化（第 4 周）

### 任务 6.1：初始同步批量处理
- 在初始同步期间实现批量工作负载插入
- 限制并发 Goroutine 数量（默认 10 个 worker）
- 为大规模同步操作添加进度日志

**文件**：`src/agent/pkg/k8s/batch_processor.go`
**测试**：`src/agent/pkg/k8s/batch_processor_test.go`
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - 批量事件处理
**预估时间**：3 小时

### 任务 6.2：事件去重
- 实现带去重的事件队列
- 合并同一 Pod 的连续 Update 事件
- 只处理最新版本

**文件**：`src/agent/pkg/k8s/event_queue.go`
**测试**：`src/agent/pkg/k8s/event_queue_test.go`
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - 事件去重
**预估时间**：3 小时

### 任务 6.3：内存管理
- 配置 Informer 缓存大小限制
- 实现定期缓存清理
- 监控内存使用

**文件**：更新 `src/agent/pkg/k8s/informer.go`
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - 内存使用限制
**预估时间**：2 小时

---

## 阶段 7：可观测性和监控（第 4 周）

### 任务 7.1：结构化日志
- 实现结构化日志，包含必需字段：
  - `component`、`resource`、`event_type`、`namespace`、`name`、`uid`、`duration_ms`
- 使用 logrus 或 zap 的 JSON 格式化器
- 添加日志级别配置

**文件**：更新所有 K8s 处理器
**规范**：[k8s-api-client](./specs/k8s-api-client/spec.md) - 结构化日志记录
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - 同步状态日志
**预估时间**：3 小时

### 任务 7.2：Prometheus 指标（可选）
- 添加 Prometheus 指标收集器：
  - `k8s_api_requests_total{operation, status}`
  - `k8s_api_request_duration_seconds{operation}`
  - `k8s_api_errors_total{operation, error_type}`
  - `k8s_informer_sync_total{resource}`
  - `k8s_informer_event_total{resource, event_type}`
  - `k8s_informer_cache_size{resource}`

**文件**：`src/agent/pkg/k8s/metrics.go`
**规范**：[k8s-api-client](./specs/k8s-api-client/spec.md) - 监控指标
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - Informer 状态监控
**预估时间**：4 小时

---

## 阶段 8：测试和文档（第 4-5 周）

### 任务 8.1：单元测试
- 为所有组件编写单元测试（目标 80% 覆盖率）
- 使用 fake Kubernetes clientset 进行测试
- Mock Workload Manager 和 Group Manager

**文件**：`*_test.go` 文件
**预估时间**：8 小时

### 任务 8.2：集成测试
- 使用 kind/minikube 集群创建集成测试
- 测试 Pod 生命周期：Add → Update → Delete
- 测试 Service 到 Group 同步
- 测试 namespace 过滤
- 测试 RBAC 权限错误

**文件**：`src/agent/pkg/k8s/integration_test.go`
**预估时间**：6 小时

### 任务 8.3：端到端测试
- 在测试集群中部署 Agent 作为 DaemonSet
- 创建测试 Pod 并验证工作负载创建
- 更新 Pod 标签并验证工作负载更新
- 删除 Pod 并验证清理
- 验证使用 K8s 来源工作负载的策略编译

**文件**：`test/e2e/k8s_integration_test.go`
**预估时间**：6 小时

### 任务 8.4：RBAC 配置文档
- 创建 RBAC 示例清单：
  - ServiceAccount
  - ClusterRole（Pod.list、Pod.watch、Service.list、Service.watch）
  - ClusterRoleBinding
- 记录最小必需权限
- 记录可选权限

**文件**：`deploy/kubernetes/rbac.yaml`、`docs/kubernetes-rbac.md`
**预估时间**：2 小时

### 任务 8.5：配置文档
- 记录所有 K8s 集成配置选项
- 为常见场景提供示例配置：
  - In-cluster DaemonSet 部署
  - Out-of-cluster 开发设置
  - 多 Namespace 过滤
  - Service 同步禁用

**文件**：`docs/kubernetes-integration.md`、`configs/agent-k8s-example.yaml`
**预估时间**：3 小时

### 任务 8.6：故障排查指南
- 记录常见问题和解决方案：
  - RBAC 权限错误
  - ServiceAccount token 未找到
  - API Server 连接问题
  - ResourceVersion 过期
- 添加诊断命令
- 添加 FAQ 部分

**文件**：`docs/kubernetes-troubleshooting.md`
**预估时间**：3 小时

---

## 总结

**总预估时间**：4-5 周（约 100-120 小时）

**关键路径**：
1. K8s API Client（任务 1.1-1.5）→ Pod Informer（任务 2.1-2.4）→ 标签映射（任务 3.1-3.4）→ 测试（任务 8.1-8.3）

**可选功能**（可延后）：
- Service Informer（阶段 4）
- Prometheus 指标（任务 7.2）

**风险缓解**：
- 仅从 in-cluster 配置开始，稍后添加 out-of-cluster
- 首先实现核心 Pod 同步，延后 Service 同步
- 使用 fake clientset 进行早期测试，无需真实集群
- 在基本功能运作之前延后性能优化

**成功标准**（来自 proposal.md）：
- [x] K8s API Client 成功连接（in-cluster 和 out-of-cluster）✅
- [ ] 监听和同步 Pod 的 Add/Update/Delete 事件
- [x] K8s 标签正确映射到工作负载标签 ✅
- [x] 支持 Namespace 过滤 ✅
- [ ] 测试覆盖率 ≥ 70%
- [ ] 完整文档（配置、RBAC、故障排查）

**已完成任务汇总**：
- ✅ 阶段 1：Kubernetes API 客户端基础（任务 1.1-1.5）
- ✅ 任务 2.2：Namespace 过滤
- ✅ 任务 3.1：标签映射规则

**下一步任务**：
- 任务 2.1：创建 Pod Informer
- 任务 2.3：实现 Pod 事件处理器
- 任务 2.4：Pod 到 Workload 转换（部分已完成，需集成）
