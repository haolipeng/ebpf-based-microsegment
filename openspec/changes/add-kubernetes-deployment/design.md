# 设计文档: Kubernetes 部署 (测试环境)

## 架构概览

本文档描述 eBPF 微分段系统在 Kubernetes 测试环境的基础部署架构。

## 系统组件

### 部署架构

```
┌─────────────────────────────────────────────────────────┐
│              Kubernetes Cluster (测试环境)                │
│                                                           │
│  Namespace: microsegment                                 │
│  ┌───────────────────────────────────────────────────┐  │
│  │  ConfigMap: microsegment-config (基础配置)         │  │
│  └───────────────────────────────────────────────────┘  │
│                          ↓                               │
│  ┌───────────────────────────────────────────────────┐  │
│  │  PostgreSQL Deployment (1 replica)                │  │
│  │  ┌─────────────────┐                              │  │
│  │  │  postgres-pod   │                              │  │
│  │  └─────────────────┘                              │  │
│  │  Service: microsegment-postgres (ClusterIP)       │  │
│  └───────────────────────────────────────────────────┘  │
│                          ↓                               │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Server Deployment (1 replica)                    │  │
│  │  ┌─────────────────┐                              │  │
│  │  │  server-pod     │                              │  │
│  │  └─────────────────┘                              │  │
│  │  Service: microsegment-server (ClusterIP)         │  │
│  └───────────────────────────────────────────────────┘  │
│                          ↓                               │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Agent DaemonSet (每节点一个)                      │  │
│  │  Node1        Node2        Node3                  │  │
│  │  ┌──────┐    ┌──────┐    ┌──────┐                │  │
│  │  │agent │    │agent │    │agent │                │  │
│  │  └──────┘    └──────┘    └──────┘                │  │
│  │  ServiceAccount + RBAC                            │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## 关键设计决策

### 1. 简化配置用于测试

**原因**: 测试环境优先考虑简单性和快速部署
**配置**:
- 所有服务使用单副本 (PostgreSQL, Server)
- 不配置持久化存储 (数据丢失可接受)
- 使用 ClusterIP Service (集群内访问)

### 2. Agent 使用 DaemonSet

**原因**: Agent 需要在每个节点收集网络流量
**配置**:
- DaemonSet 确保每个节点运行一个 Agent
- hostNetwork: true - 访问主机网络
- hostPID: true - 访问主机进程信息
- privileged: true - 加载 eBPF 程序

### 3. 最小化 RBAC 权限

**配置**:
- ServiceAccount: microsegment-agent
- ClusterRole: 仅授予必要的 API 访问权限
- ClusterRoleBinding: 绑定角色到 ServiceAccount

### 4. 配置使用 ConfigMap

**原因**: 简化配置管理
**配置**:
- ConfigMap 存储基础配置
- 环境变量方式注入配置

## 资源清单设计

### 文件结构

```
deploy/kubernetes/
├── namespace.yaml          # Namespace 定义
├── rbac.yaml              # RBAC 配置
├── configmap.yaml         # ConfigMap 配置
├── postgres.yaml          # PostgreSQL Deployment + Service
├── server.yaml            # Server Deployment + Service
└── agent.yaml             # Agent DaemonSet
```

### 关键配置项

#### Agent DaemonSet
- **特权容器**: privileged: true (加载 eBPF 程序)
- **主机访问**: hostNetwork, hostPID
- **卷挂载**: /sys/fs/bpf (BPF 文件系统)

#### Server Deployment
- **环境变量**: 数据库连接信息
- **Service**: ClusterIP 类型,供 Agent 连接

#### PostgreSQL Deployment
- **环境变量**: 数据库初始化配置
- **Service**: ClusterIP 类型,供 Server 连接

## 部署流程

1. 创建 Namespace
2. 应用 RBAC 配置
3. 应用 ConfigMap
4. 部署 PostgreSQL
5. 部署 Server
6. 部署 Agent
7. 验证所有 Pod 运行正常

## 验证测试

1. **Pod 状态检查**: 所有 Pod 为 Running 状态
2. **eBPF 加载检查**: Agent 成功加载 eBPF 程序
3. **连接性检查**: Agent 可连接 Server
4. **功能检查**: 基本的流量捕获功能正常

## 限制和注意事项

1. **无持久化**: 数据库数据不持久化,Pod 重启后丢失
2. **无高可用**: 所有服务单副本,不适合生产环境
3. **安全性**: Agent 使用特权容器,仅适用于测试环境
4. **外部访问**: 无 Ingress,仅支持集群内访问

## 未来扩展

测试环境验证通过后,可考虑:
1. 生产级配置 (StatefulSet, 多副本)
2. 持久化存储 (PVC)
3. Ingress 配置
4. 监控和日志集成
