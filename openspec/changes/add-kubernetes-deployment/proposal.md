# 提案: 添加 Kubernetes 部署配置

## 概述

本提案旨在为 eBPF 微分段系统添加 Kubernetes 部署支持,包括 Deployment、DaemonSet、ConfigMap、Secret、Service 等 K8s 资源清单和部署脚本,以支持在 Kubernetes 集群中的生产级部署。

## Why

当前 eBPF 微分段系统的 Server 和 Agent 组件已经实现并可运行,但缺少 Kubernetes 部署配置,导致:

**问题 1: 无法在 K8s 集群部署**
- 没有标准的 Kubernetes 资源清单
- 无法利用 K8s 的编排能力 (自动调度、滚动更新、自愈)
- 缺少 Server 高可用配置 (多副本)
- 缺少 Agent DaemonSet 配置 (每节点一个)

**问题 2: 配置管理不标准**
- 无法使用 ConfigMap 和 Secret 管理配置
- 配置变更需要重新构建镜像
- 敏感信息管理不安全
- 多环境配置管理困难

**问题 3: 缺少服务发现和负载均衡**
- 无法使用 K8s Service 提供服务发现
- 缺少负载均衡配置
- 无法通过 Ingress 暴露服务
- 服务间通信配置复杂

**问题 4: 运维能力不足**
- 无法使用 kubectl 管理服务
- 缺少健康检查和就绪检查
- 无法滚动更新和回滚
- 缺少资源限制和监控集成

**业务影响**:
- 阻碍在云原生环境的推广
- 无法满足大规模生产部署需求
- 降低系统可用性和可靠性
- 增加运维复杂度和成本

本提案通过提供完整的 Kubernetes 部署配置,解决上述问题,使系统能够在 K8s 集群中稳定、高效运行。

## 动机

当前项目已经具备:
- 可运行的 Server 和 Agent 组件
- Docker 镜像构建能力 (来自 add-docker-deployment 提案)
- 配置文件支持 (YAML 格式)

但是缺少:
1. **Kubernetes 资源清单**: Deployment, DaemonSet, Service, ConfigMap, Secret, Ingress
2. **RBAC 配置**: ServiceAccount, ClusterRole, ClusterRoleBinding (Agent 需要)
3. **持久化存储**: PostgreSQL StatefulSet 和 PVC
4. **配置管理**: ConfigMap 和 Secret 管理配置和密钥
5. **部署工具**: K8s 部署脚本和 Kustomize 配置
6. **部署文档**: 详细的 K8s 部署指南

这些缺失导致:
- 无法在 Kubernetes 环境标准化部署
- 无法利用云原生的优势
- 大规模部署困难
- 运维管理复杂

## 目标

### 主要目标

1. **Server Deployment 配置**
   - Deployment 资源 (2 副本实现高可用)
   - Liveness 和 Readiness probes
   - 资源请求和限制
   - 反亲和性规则 (副本分散到不同节点)
   - Service 和 Ingress 配置

2. **Agent DaemonSet 配置**
   - DaemonSet 资源 (每节点一个)
   - 特权容器配置 (privileged: true)
   - 主机网络和 PID 命名空间 (hostNetwork, hostPID)
   - BPF 文件系统挂载
   - RBAC 权限配置

3. **配置和密钥管理**
   - ConfigMap 管理配置文件
   - Secret 管理敏感信息
   - 环境变量注入

4. **PostgreSQL 部署**
   - StatefulSet 配置
   - PersistentVolumeClaim
   - Service 配置

5. **部署自动化**
   - K8s 部署脚本
   - Kustomize 配置
   - 健康检查脚本

6. **文档和测试**
   - K8s 部署文档
   - 部署流程测试验证

### 非目标

- Docker 容器化配置 (在 add-docker-deployment 提案中处理)
- Systemd 服务配置 (在 add-systemd-deployment 提案中处理)
- Helm Charts (可作为未来扩展)
- Kubernetes Operator (可作为未来扩展)
- 多集群部署

## 设计概要

### 1. 资源架构

```
Namespace: microsegment
├── ConfigMap: microsegment-config
├── Secret: microsegment-secret
├── PostgreSQL
│   ├── StatefulSet: microsegment-postgres
│   ├── Service: microsegment-postgres
│   └── PVC: postgres-data
├── Server
│   ├── Deployment: microsegment-server (2 replicas)
│   ├── Service: microsegment-server
│   └── Ingress: microsegment-ingress (可选)
└── Agent
    ├── ServiceAccount: microsegment-agent
    ├── ClusterRole: microsegment-agent
    ├── ClusterRoleBinding: microsegment-agent
    └── DaemonSet: microsegment-agent
```

### 2. Server Deployment 关键配置

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: microsegment-server
spec:
  replicas: 2
  strategy:
    type: RollingUpdate
  selector:
    matchLabels:
      app: microsegment-server
  template:
    spec:
      containers:
      - name: server
        image: microsegment-server:latest
        ports:
        - containerPort: 8080
        - containerPort: 9090
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 1000m
            memory: 1Gi
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  app: microsegment-server
              topologyKey: kubernetes.io/hostname
```

### 3. Agent DaemonSet 关键配置

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: microsegment-agent
spec:
  selector:
    matchLabels:
      app: microsegment-agent
  template:
    spec:
      serviceAccountName: microsegment-agent
      hostNetwork: true
      hostPID: true
      containers:
      - name: agent
        image: microsegment-agent:latest
        securityContext:
          privileged: true
          capabilities:
            add:
            - SYS_ADMIN
            - NET_ADMIN
        volumeMounts:
        - name: bpf
          mountPath: /sys/fs/bpf
          mountPropagation: Bidirectional
      volumes:
      - name: bpf
        hostPath:
          path: /sys/fs/bpf
          type: Directory
```

## 影响的规范

本提案将添加新的 Kubernetes 部署规范:

1. **新规范 kubernetes-deployment**: 定义 K8s 部署要求

## 依赖关系

### 前置条件
- Server 和 Agent 组件已实现并可运行
- Docker 镜像已构建 (依赖 add-docker-deployment)
- 目标 K8s 集群可用 (版本 >= 1.20)

### 阻塞项
- 无

## 成功标准

1. **功能完整性**
   - ✅ 可通过 kubectl apply 部署到 K8s 集群
   - ✅ Server 以 2 副本运行 (高可用)
   - ✅ Agent 在每个节点运行 (DaemonSet)

2. **可用性**
   - ✅ 部署文档清晰,K8s 管理员可按指南完成部署
   - ✅ 提供一键部署脚本
   - ✅ 健康检查和就绪检查正常工作

3. **安全性**
   - ✅ 使用 RBAC 最小权限
   - ✅ 密钥通过 Secret 管理
   - ✅ Agent 特权配置有文档说明

4. **可靠性**
   - ✅ 滚动更新无中断
   - ✅ Pod 自动重启和自愈
   - ✅ 资源限制防止资源耗尽

## 风险与缓解

### 风险 1: Agent 特权容器的安全风险
- **描述**: Agent 需要 privileged: true,存在安全风险
- **缓解**:
  - 使用 RBAC 限制权限
  - 文档中明确说明安全影响
  - 考虑使用 SecurityContext capabilities 替代 privileged

### 风险 2: DaemonSet 资源消耗
- **描述**: 每个节点运行 Agent 可能消耗大量资源
- **缓解**:
  - 配置合理的资源请求和限制
  - 提供节点选择器,避免在不需要的节点运行
  - 监控资源使用

### 风险 3: 多副本 Server 的数据库连接
- **描述**: 多个 Server 副本可能导致数据库连接池耗尽
- **缓解**:
  - 配置合理的数据库连接池大小
  - 使用连接池复用
  - 监控数据库连接数

## 后续工作

本提案完成后,可以考虑:

1. **Helm Charts**: 提供 Helm Chart 简化部署
2. **Operator**: 实现 Kubernetes Operator 自动化运维
3. **监控集成**: Prometheus Operator, Grafana Dashboard
4. **日志集成**: Elasticsearch, Fluentd, Kibana
5. **多集群部署**: Federation 或多集群管理

## 时间估算

- **K8s 资源清单**: 2 天
- **RBAC 配置**: 0.5 天
- **部署脚本**: 1 天
- **Kustomize 配置**: 0.5 天
- **文档编写**: 1 天
- **测试验证**: 1 天

**总计**: 约 6 天

## 参考

- [Kubernetes 最佳实践](https://kubernetes.io/docs/concepts/configuration/)
- [DaemonSet 文档](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/)
- [RBAC 文档](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
