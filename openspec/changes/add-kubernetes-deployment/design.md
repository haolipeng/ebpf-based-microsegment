# 设计文档: Kubernetes 部署

## 架构概览

本文档描述 eBPF 微分段系统的 Kubernetes 部署架构,涵盖资源清单设计、RBAC 配置、存储管理和服务编排。

## 系统组件

### 1. 部署架构

```
┌─────────────────────────────────────────────────────────────┐
│                  Kubernetes Cluster                          │
│                                                               │
│  Namespace: microsegment                                     │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  ConfigMap/Secret                                     │  │
│  │  ┌─────────────────┐  ┌─────────────────┐            │  │
│  │  │  microsegment-  │  │  microsegment-  │            │  │
│  │  │     config      │  │     secret      │            │  │
│  │  └─────────────────┘  └─────────────────┘            │  │
│  └───────────────────────────────────────────────────────┘  │
│                          ↓                                   │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  PostgreSQL StatefulSet                               │  │
│  │  ┌─────────────────┐  ┌─────────────────┐            │  │
│  │  │  postgres-0     │→ │  PVC: data-0    │            │  │
│  │  └─────────────────┘  └─────────────────┘            │  │
│  └───────────────────────────────────────────────────────┘  │
│                          ↓                                   │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Server Deployment (2 replicas)                       │  │
│  │  ┌─────────────────┐  ┌─────────────────┐            │  │
│  │  │  server-pod-1   │  │  server-pod-2   │            │  │
│  │  └─────────────────┘  └─────────────────┘            │  │
│  │  Service: microsegment-server (LoadBalancer)         │  │
│  │  Ingress: microsegment-ingress (可选)                 │  │
│  └───────────────────────────────────────────────────────┘  │
│                          ↓                                   │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Agent DaemonSet (每节点一个)                          │  │
│  │  Node1        Node2        Node3                      │  │
│  │  ┌──────┐    ┌──────┐    ┌──────┐                    │  │
│  │  │agent │    │agent │    │agent │                    │  │
│  │  │(priv)│    │(priv)│    │(priv)│                    │  │
│  │  └──────┘    └──────┘    └──────┘                    │  │
│  │  ServiceAccount: microsegment-agent                   │  │
│  │  ClusterRole + ClusterRoleBinding                     │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## 关键设计决策

### 1. Server 使用 Deployment (高可用)

**原因**: Server 是无状态服务,适合使用 Deployment 实现多副本
**配置**:
- 2 个副本实现高可用
- RollingUpdate 策略实现无中断更新
- PodAntiAffinity 确保副本分散到不同节点
- Liveness 和 Readiness probes 实现健康检查

### 2. Agent 使用 DaemonSet (每节点一个)

**原因**: Agent 需要在每个节点收集网络流量
**配置**:
- DaemonSet 确保每个节点运行一个 Agent
- hostNetwork: true 访问主机网络
- hostPID: true 访问主机进程
- privileged: true 加载 eBPF 程序

### 3. PostgreSQL 使用 StatefulSet

**原因**: 数据库需要持久化存储和稳定的网络标识
**配置**:
- StatefulSet 提供稳定的 Pod 名称和存储
- PVC 实现数据持久化
- Headless Service 提供稳定的 DNS 名称

### 4. 配置使用 ConfigMap 和 Secret

**原因**: 分离配置和代码,方便管理
**配置**:
- ConfigMap 存储非敏感配置 (server.yaml, agent.yaml)
- Secret 存储敏感信息 (数据库密码)
- 挂载为文件或环境变量

## 资源清单详细设计

由于篇幅限制,完整的清单文件将在 deploy/kubernetes/ 目录中提供。关键配置点:

1. **Namespace**: 资源隔离
2. **RBAC**: Agent 需要访问 nodes 和 pods 信息
3. **NetworkPolicy**: 网络隔离 (可选)
4. **ResourceQuota**: 资源配额限制 (可选)
5. **PodSecurityPolicy**: 安全策略 (可选,已废弃,使用 PodSecurity)

## 部署流程

1. 创建 Namespace
2. 应用 ConfigMap 和 Secret
3. 部署 PostgreSQL StatefulSet
4. 等待 PostgreSQL 就绪
5. 部署 Server Deployment
6. 创建 Server Service
7. 等待 Server 就绪
8. 创建 Agent RBAC
9. 部署 Agent DaemonSet
10. 创建 Ingress (可选)

## 监控和日志

- Prometheus 集成: /metrics 端点
- 日志: stdout/stderr → Kubernetes 日志系统
- 健康检查: liveness 和 readiness probes

## 安全考虑

1. **RBAC 最小权限**: Agent 仅授予必要权限
2. **Secret 加密**: 使用 K8s Secret 存储敏感信息
3. **Network Policy**: 限制 Pod 间网络访问
4. **Pod Security**: 限制特权容器使用

## 总结

本设计提供生产级别的 Kubernetes 部署方案,支持高可用、自动扩展、滚动更新和自愈能力。
