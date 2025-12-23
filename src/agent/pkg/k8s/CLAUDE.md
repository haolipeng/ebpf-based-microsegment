[上级索引](../CLAUDE.md) > **k8s**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# k8s

## 架构定位

Kubernetes 集成器 | 输入: K8s Pod 事件（Create/Update/Delete） | 输出: Workload 创建/更新/删除、标签同步

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| syncer.go | K8s 同步器主体，协调 Client、Informer、Handler | `NewSyncer()`, `Start()`, `Stop()` |
| client.go | K8s 客户端封装（In-Cluster / Kubeconfig） | `NewClient()`, `GetPod()`, `ListPods()` |
| informer.go | Pod Informer 工厂，监听 Pod 事件 | `NewPodInformer()`, `Run()` |
| pod_handler.go | Pod 事件处理器，转换 Pod 到 Workload | `OnAdd()`, `OnUpdate()`, `OnDelete()` |
| converter.go | Pod 对象转换为 Workload | `PodToWorkload()`, `ExtractLabels()` |
| namespace_filter.go | 命名空间过滤器（黑名单/白名单） | `ShouldInclude()`, `AddExclude()` |
| health.go | K8s 连接健康检查 | `IsHealthy()`, `Ping()` |
| rbac.go | RBAC 权限检查和建议 | `CheckPermissions()`, `GenerateRBAC()` |

## 核心功能

- **自动发现**: 监听 Pod 创建事件，自动注册为 Workload
- **标签同步**: 提取 Pod Labels 和 Annotations，应用到工作负载
- **命名空间过滤**: 仅监控指定命名空间（如排除 kube-system）
- **健康监控**: 检测 K8s API Server 连接状态
- **RBAC 支持**: 检查必要权限，生成 RBAC 清单

## 权限要求

需要以下 K8s RBAC 权限：
- **pods**: get, list, watch
- **namespaces**: get, list, watch（可选）

## 应用场景

- **K8s 环境**: 自动发现和管理 Pod 工作负载
- **标签策略**: 使用 Pod 标签定义网络策略
- **命名空间隔离**: 基于命名空间的网络分段
