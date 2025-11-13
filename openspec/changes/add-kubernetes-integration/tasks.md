# 实施任务 (MVP 版本)

> **注**: 本任务列表已简化为 MVP 版本,专注于核心功能实现。从 40+ 任务减少到 9 个核心任务,预计时间从 100-120 小时减少到约 22 小时 (3 个工作日)。

---

## ✅ 阶段 1：Kubernetes API 客户端基础 - 已完成

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

### 任务 1.6：实现 Namespace 过滤 ✅
- ✅ 添加 namespace 过滤配置（include/exclude 列表）
- ✅ 在 Informer 创建时实现过滤逻辑
- ✅ 支持通配符模式（例如：`prod-*`）
- ✅ 添加 namespace 过滤配置验证

**文件**：`src/agent/pkg/k8s/namespace_filter.go`
**测试**：`src/agent/pkg/k8s/namespace_filter_test.go`
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - Namespace 过滤配置
**预估时间**：3 小时
**状态**：✅ 已完成

### 任务 1.7：实现标签映射规则 ✅
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
- ✅ 自动添加元数据标签：`k8s.namespace`、`k8s.pod.name`、`k8s.node.name`

**文件**：`src/agent/pkg/k8s/converter.go`
**测试**：`src/agent/pkg/k8s/converter_test.go`
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - 标签映射规则
**预估时间**：4 小时
**状态**：✅ 已完成

---

## 🎯 阶段 2：Pod 同步核心功能 (MVP 必须)

### 任务 2.1：创建 Pod Informer
- 创建 `src/agent/pkg/k8s/informer.go`
- 实现 `NewPodInformer(client *Client, namespace string) cache.SharedIndexInformer`
- 配置 Informer 的 ListWatch 监听 Pod
- 设置 IndexInformer 支持 namespace 索引
- 配置重新同步间隔（默认 30 分钟）
- 集成 namespace 过滤器

**文件**：`src/agent/pkg/k8s/informer.go`
**测试**：`src/agent/pkg/k8s/informer_test.go`（使用 fake clientset）
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - Pod Informer 初始化
**预估时间**：4 小时
**状态**：⏳ 待完成

### 任务 2.2：实现 Pod 事件处理器
- 创建 `src/agent/pkg/k8s/pod_handler.go`
- 实现 `OnAdd(obj interface{})` 处理器
- 实现 `OnUpdate(oldObj, newObj interface{})` 处理器
- 实现 `OnDelete(obj interface{})` 处理器
- 提取 Pod 元数据：Namespace、Name、UID、IP、Node、Labels
- 添加基础错误恢复（panic recovery）
- 添加结构化日志记录

**文件**：`src/agent/pkg/k8s/pod_handler.go`
**测试**：`src/agent/pkg/k8s/pod_handler_test.go`
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - Pod 事件处理
**预估时间**：4 小时
**状态**：⏳ 待完成

### 任务 2.3：完善 Pod 到 Workload 转换
- 使用现有的 `src/agent/pkg/k8s/converter.go`
- 确保 `PodToWorkload(pod *corev1.Pod) (*workload.Workload, error)` 正常工作
- 生成 workload ID：`k8s:<namespace>:<uid>`
- 处理无 IP 的 Pod（返回 nil，跳过）
- 可选：处理 HostNetwork Pod（配置是否包含）

**文件**：`src/agent/pkg/k8s/converter.go`（已存在，需验证和集成）
**测试**：`src/agent/pkg/k8s/converter_test.go`（已存在）
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - Pod 到 Workload 转换
**预估时间**：2 小时
**状态**：⏳ 待完成（基础代码已存在，需集成）

---

## 🔌 阶段 3：Workload 集成 (MVP 必须)

### 任务 3.1：集成 Workload Manager
- 在 pod_handler 中添加 Workload Manager 依赖
- 实现工作负载创建：`workloadMgr.CreateWorkload(wl)`
- 实现工作负载更新：`workloadMgr.UpdateWorkload(wl)`
- 实现工作负载删除：`workloadMgr.DeleteWorkload(id)`
- 添加基础错误处理

**文件**：更新 `src/agent/pkg/k8s/pod_handler.go`
**测试**：`src/agent/pkg/k8s/pod_handler_test.go`（使用 mock Workload Manager）
**预估时间**：2 小时
**状态**：⏳ 待完成

---

## 🛡️ 阶段 4：基础容错 (MVP 必须)

### 任务 4.1：实现基础重连机制
- 为 Watch 重连添加简单重试逻辑
- 处理 "410 Gone" ResourceVersion 过期
- 在 ResourceVersion 过期时实现完整重新同步
- 添加基础连接状态日志

**文件**：`src/agent/pkg/k8s/reconnect.go`
**测试**：`src/agent/pkg/k8s/reconnect_test.go`
**规范**：[k8s-api-client](./specs/k8s-api-client/spec.md) - 网络故障重试
**规范**：[k8s-workload-sync](./specs/k8s-workload-sync/spec.md) - Watch 连接中断
**预估时间**：2 小时
**状态**：⏳ 待完成

### 任务 4.2：优雅降级
- 支持在禁用 K8s 集成时运行
- 优雅处理 RBAC 权限错误
- 记录警告但不崩溃
- 即使个别处理器失败也继续处理事件

**文件**：更新 `src/agent/cmd/agent/main.go`、`src/agent/pkg/k8s/client.go`
**规范**：[k8s-api-client](./specs/k8s-api-client/spec.md) - 客户端初始化失败处理
**预估时间**：2 小时
**状态**：⏳ 待完成

---

## 📝 阶段 5：基础测试和文档 (MVP 必须)

### 任务 5.1：核心功能单元测试
- 为 Informer、PodHandler、Converter 编写单元测试
- 使用 fake Kubernetes clientset 进行测试
- Mock Workload Manager
- 目标测试覆盖率：60%（核心路径）

**文件**：`*_test.go` 文件
**预估时间**：4 小时
**状态**：⏳ 待完成

### 任务 5.2：RBAC 配置清单
- 创建 RBAC 示例清单：
  - ServiceAccount
  - ClusterRole（Pod.list、Pod.watch）
  - ClusterRoleBinding
- 记录最小必需权限

**文件**：`deploy/kubernetes/rbac.yaml`
**预估时间**：1 小时
**状态**：⏳ 待完成

### 任务 5.3：基础配置文档
- 记录 K8s 集成配置选项
- 提供示例配置：
  - In-cluster DaemonSet 部署
  - Out-of-cluster 开发设置
  - Namespace 过滤示例

**文件**：`docs/kubernetes-integration.md`、`configs/agent-k8s-example.yaml`
**预估时间**：1 小时
**状态**：⏳ 待完成

---

## 📊 总结

### 时间估算

**MVP 版本总时间**：约 **22 小时** (3 个工作日)

- ✅ 阶段 1（已完成）：12.5 小时
- 🎯 阶段 2：10 小时
- 🔌 阶段 3：2 小时
- 🛡️ 阶段 4：4 小时
- 📝 阶段 5：6 小时

**对比原计划**：从 100-120 小时减少到 22 小时 (减少 82%)

### MVP 成功标准

- [x] ✅ K8s API Client 成功连接（in-cluster 和 out-of-cluster）
- [ ] ⏳ 监听和同步 Pod 的 Add/Update/Delete 事件
- [x] ✅ K8s 标签正确映射到工作负载标签
- [x] ✅ 支持 Namespace 过滤
- [ ] ⏳ 核心功能测试覆盖率 ≥ 60%
- [ ] ⏳ 基础文档（配置、RBAC）

### 已完成任务

- ✅ 任务 1.1：添加 client-go 依赖
- ✅ 任务 1.2：实现 K8s 客户端初始化
- ✅ 任务 1.3：实现健康检查
- ✅ 任务 1.4：实现 RBAC 权限检查
- ✅ 任务 1.5：配置集成
- ✅ 任务 1.6：实现 Namespace 过滤
- ✅ 任务 1.7：实现标签映射规则

**进度**：7/16 任务完成 (44%)

### 下一步任务（优先级排序）

1. **任务 2.1**：创建 Pod Informer（最高优先级）
2. **任务 2.2**：实现 Pod 事件处理器
3. **任务 2.3**：完善 Pod 到 Workload 转换
4. **任务 3.1**：集成 Workload Manager
5. **任务 4.1**：实现基础重连机制
6. **任务 4.2**：优雅降级
7. **任务 5.1**：核心功能单元测试
8. **任务 5.2**：RBAC 配置清单
9. **任务 5.3**：基础配置文档

---

## ❌ 已删除的非 MVP 功能

以下功能已从 MVP 版本中移除，可在后续版本中实现：

### 阶段 3（原）：高级标签处理
- ❌ 任务 3.2：实现注解处理（`microsegment.io/labels`）
- ❌ 任务 3.3：添加元数据标签（已在 converter.go 中实现基础版本）

### 阶段 4（原）：Service Informer（整个阶段）
- ❌ 任务 4.1：实现 Service Informer
- ❌ 任务 4.2：Service 到 Group 映射

### 阶段 5（原）：高级错误处理
- ❌ 任务 5.2：事件处理错误处理（重试队列、死信队列）

### 阶段 6（原）：性能优化（整个阶段）
- ❌ 任务 6.1：初始同步批量处理
- ❌ 任务 6.2：事件去重
- ❌ 任务 6.3：内存管理

### 阶段 7（原）：高级监控
- ❌ 任务 7.2：Prometheus 指标

### 阶段 8（原）：高级测试
- ❌ 任务 8.2：集成测试（使用 kind/minikube）
- ❌ 任务 8.3：端到端测试
- ❌ 任务 8.6：故障排查指南

**删除任务数**：31 个任务
**保留任务数**：16 个任务（7 个已完成 + 9 个待完成）

---

## 🎯 MVP 交付物检查清单

- [x] ✅ Kubernetes 客户端库集成
- [x] ✅ In-cluster 和 out-of-cluster 配置支持
- [x] ✅ RBAC 权限检查
- [x] ✅ Namespace 过滤
- [x] ✅ 标签映射规则（包括元数据标签）
- [ ] ⏳ Pod Informer 实现
- [ ] ⏳ Pod 事件处理（Add/Update/Delete）
- [ ] ⏳ Workload Manager 集成
- [ ] ⏳ 基础重连机制
- [ ] ⏳ 优雅降级处理
- [ ] ⏳ 核心功能单元测试
- [ ] ⏳ RBAC 配置清单
- [ ] ⏳ 基础使用文档
