# kubernetes-deployment Specification

## Purpose

Defines the Kubernetes deployment configuration for the eBPF microsegment system in testing environments. This specification covers basic Kubernetes resource manifests, RBAC configuration, deployment scripts, and documentation needed to deploy and run the system in a Kubernetes cluster for testing and validation purposes.
## Requirements
### Requirement: Server Deployment 配置

系统必须(SHALL)提供 Server 组件的 Kubernetes Deployment 配置。

#### Scenario: Server Deployment 创建

**Given** 存在 Server Deployment 清单文件
**When** 执行 `kubectl apply -f server.yaml`
**Then** 必须(SHALL)创建 Deployment 资源
**And** 必须(SHALL)配置环境变量(数据库连接信息)
**And** 必须(SHALL)配置资源请求和限制
**And** 必须(SHALL)创建 ClusterIP Service

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

### Requirement: ConfigMap 配置

系统必须(SHALL)使用 ConfigMap 管理基础配置。

#### Scenario: ConfigMap 配置

**Given** 存在 ConfigMap 清单文件
**When** 部署 Server 或 Agent
**Then** ConfigMap 必须(SHALL)包含 server.yaml 和 agent.yaml 配置
**And** 配置必须(SHALL)包含数据库连接信息和服务端点

### Requirement: PostgreSQL 部署

系统必须(SHALL)提供 PostgreSQL 数据库的 Kubernetes 部署配置。

#### Scenario: PostgreSQL Deployment

**Given** 存在 PostgreSQL Deployment 清单
**When** 部署 PostgreSQL
**Then** 必须(SHALL)创建 Deployment 资源(单副本用于测试)
**And** 必须(SHALL)创建 ClusterIP Service
**And** 必须(SHALL)配置环境变量(POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD)

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
**And** 脚本必须(SHALL)按顺序创建所有必需资源(namespace, RBAC, ConfigMap, PostgreSQL, Server, Agent)
**And** 脚本必须(SHALL)等待 Deployment 和 DaemonSet 就绪
**And** 脚本失败时必须(SHALL)返回非零退出码

#### Scenario: 清理部署

**Given** 存在 undeploy-k8s.sh 脚本
**When** 执行该脚本
**Then** 脚本必须(SHALL)删除所有部署的资源
**And** 脚本必须(SHALL)提供删除 namespace 的选项

### Requirement: 部署文档

系统必须(SHALL)提供详细的 Kubernetes 部署文档。

#### Scenario: Kubernetes 部署指南

**Given** 存在 deploy/kubernetes/README.md 文档
**Then** 文档必须(SHALL)包含前置要求(K8s 版本, kubectl, Docker 镜像)
**And** 文档必须(SHALL)包含快速开始步骤(一键部署和手动部署)
**And** 文档必须(SHALL)包含 RBAC 权限说明
**And** 文档必须(SHALL)包含 Agent 特权容器说明
**And** 文档必须(SHALL)包含配置修改方法
**And** 文档必须(SHALL)包含故障排查指南
**And** 文档必须(SHALL)说明测试环境限制(无持久化、单副本、无高可用)

