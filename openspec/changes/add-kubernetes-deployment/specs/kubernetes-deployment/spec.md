# kubernetes-deployment 规范

**规范 ID**: `kubernetes-deployment`
**版本**: v0.1.0
**状态**: 草案

## Purpose

本规范定义 eBPF 微分段系统的 Kubernetes 部署要求,包括 Deployment、DaemonSet、Service、ConfigMap、Secret 等资源清单和部署工具。

## ADDED Requirements

### Requirement: Server Deployment 配置

系统必须(SHALL)提供 Server 组件的 Kubernetes Deployment 配置,支持高可用部署。

#### Scenario: Server Deployment 创建

**Given** 存在 Server Deployment 清单文件
**When** 执行 `kubectl apply -f deployment.yaml`
**Then** 必须(SHALL)创建 Deployment 资源
**And** Deployment 必须(SHALL)配置 2 个副本
**And** 必须(SHALL)配置 liveness probe
**And** 必须(SHALL)配置 readiness probe
**And** 必须(SHALL)配置资源请求和限制
**And** 必须(SHALL)配置 PodAntiAffinity

#### Scenario: Server 滚动更新

**Given** Server Deployment 正在运行
**When** 更新镜像版本
**Then** 必须(SHALL)执行滚动更新
**And** 必须(SHALL)保持至少 1 个副本可用
**And** 更新过程必须(SHALL)无中断

### Requirement: Agent DaemonSet 配置

系统必须(SHALL)提供 Agent 组件的 Kubernetes DaemonSet 配置,在每个节点运行。

#### Scenario: Agent DaemonSet 创建

**Given** 存在 Agent DaemonSet 清单文件
**When** 执行 `kubectl apply -f daemonset.yaml`
**Then** 必须(SHALL)创建 DaemonSet 资源
**And** 每个节点必须(SHALL)运行一个 Agent Pod
**And** Pod 必须(SHALL)配置 hostNetwork: true
**And** Pod 必须(SHALL)配置 hostPID: true
**And** Pod 必须(SHALL)配置 privileged: true
**And** Pod 必须(SHALL)挂载 /sys/fs/bpf

#### Scenario: Agent 节点扩展

**Given** Agent DaemonSet 正在运行
**When** 集群添加新节点
**Then** 新节点必须(SHALL)自动部署 Agent Pod

### Requirement: RBAC 配置

系统必须(SHALL)提供 Agent 所需的 RBAC 配置。

#### Scenario: Agent RBAC 创建

**Given** 存在 RBAC 清单文件
**When** 执行 kubectl apply 命令
**Then** 必须(SHALL)创建 ServiceAccount
**And** 必须(SHALL)创建 ClusterRole
**And** ClusterRole 必须(SHALL)授予访问 nodes 权限
**And** ClusterRole 必须(SHALL)授予访问 pods 权限
**And** 必须(SHALL)创建 ClusterRoleBinding

### Requirement: ConfigMap 和 Secret

系统必须(SHALL)使用 ConfigMap 和 Secret 管理配置和密钥。

#### Scenario: ConfigMap 配置

**Given** 存在 ConfigMap 清单文件
**When** 部署 Server 或 Agent
**Then** ConfigMap 必须(SHALL)包含配置文件
**And** 配置文件必须(SHALL)挂载到 Pod
**And** Pod 必须(SHALL)能读取配置

#### Scenario: Secret 密钥管理

**Given** 存在 Secret 清单文件
**Then** Secret 必须(SHALL)存储敏感信息 (数据库密码)
**And** Secret 值必须(SHALL)使用 base64 编码
**And** Secret 必须(SHALL)注入到 Pod 环境变量

### Requirement: Service 和 Ingress

系统必须(SHALL)提供 Service 配置用于服务发现和负载均衡。

#### Scenario: Server Service 创建

**Given** Server Deployment 正在运行
**When** 创建 Service
**Then** Service 必须(SHALL)暴露端口 8080 和 9090
**And** Service 必须(SHALL)负载均衡到所有 Server Pods
**And** Service 必须(SHALL)可通过 DNS 名称访问

#### Scenario: Ingress 配置 (可选)

**Given** 存在 Ingress 清单文件
**When** 部署 Ingress
**Then** Ingress 必须(SHALL)配置路由规则
**And** 必须(SHALL)支持通过域名访问 Server

### Requirement: 持久化存储

系统必须(SHALL)为 PostgreSQL 配置持久化存储。

#### Scenario: PostgreSQL StatefulSet

**Given** 存在 PostgreSQL StatefulSet 清单
**When** 部署 StatefulSet
**Then** 必须(SHALL)创建 StatefulSet 资源
**And** 必须(SHALL)配置 PersistentVolumeClaim
**And** 数据必须(SHALL)持久化到 PVC

### Requirement: 健康检查

系统必须(SHALL)配置 Pod 健康检查。

#### Scenario: Liveness 和 Readiness Probes

**Given** Server Deployment 已配置
**Then** 必须(SHALL)配置 livenessProbe
**And** livenessProbe 必须(SHALL)检查 /health 端点
**And** 必须(SHALL)配置 readinessProbe
**And** readinessProbe 必须(SHALL)检查 /ready 端点
**And** 探测失败时必须(SHALL)重启或移出 Service

### Requirement: 资源管理

系统必须(SHALL)配置资源请求和限制。

#### Scenario: 资源配置

**Given** Deployment 或 DaemonSet 清单
**Then** 必须(SHALL)配置 resources.requests
**And** 必须(SHALL)配置 resources.limits
**And** 请求值必须(SHALL)小于等于限制值

### Requirement: 部署自动化

系统必须(SHALL)提供 Kubernetes 部署脚本。

#### Scenario: 一键部署

**Given** 存在 deploy-k8s.sh 脚本
**When** 执行该脚本
**Then** 脚本必须(SHALL)检查 kubectl 环境
**And** 脚本必须(SHALL)创建所有必需资源
**And** 脚本必须(SHALL)等待 Pods 就绪
**And** 脚本必须(SHALL)验证部署成功
**And** 脚本失败时必须(SHALL)返回非零退出码

### Requirement: Kustomize 支持

系统必须(SHALL)提供 Kustomize 配置。

#### Scenario: Kustomize 部署

**Given** 存在 kustomization.yaml 文件
**When** 执行 `kubectl apply -k`
**Then** 必须(SHALL)生成并应用所有资源
**And** 必须(SHALL)支持环境特定的 overlay

### Requirement: 部署文档

系统必须(SHALL)提供详细的 Kubernetes 部署文档。

#### Scenario: Kubernetes 部署指南

**Given** 存在 deploy/kubernetes/README.md 文档
**Then** 文档必须(SHALL)包含前置要求
**And** 文档必须(SHALL)包含快速开始步骤
**And** 文档必须(SHALL)包含 RBAC 权限说明
**And** 文档必须(SHALL)包含 Agent 特权说明
**And** 文档必须(SHALL)包含配置定制方法
**And** 文档必须(SHALL)包含升级和回滚步骤
**And** 文档必须(SHALL)包含故障排查指南
