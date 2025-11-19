# 提案: 添加 Kubernetes 部署配置 (测试环境)

## 概述

本提案旨在为 eBPF 微分段系统添加基础的 Kubernetes 部署支持,使系统能够在 K8s 集群中部署并进行测试验证。

## Why

当前 eBPF 微分段系统的 Server 和 Agent 组件已经实现并可在本地运行,但缺少 Kubernetes 部署配置,导致:

**问题**: 无法在 K8s 环境测试
- 没有基本的 Kubernetes 资源清单
- 无法利用 K8s 的容器编排能力
- 缺少 Agent DaemonSet 配置 (每节点一个)
- 无法验证系统在容器环境中的运行情况

**影响**:
- 无法在云原生环境进行测试
- 阻碍 K8s 部署方案的开发和验证

本提案通过提供基础的 Kubernetes 部署配置,使系统能够在 K8s 测试环境中运行。

## 动机

当前项目已经具备:
- 可运行的 Server 和 Agent 组件
- Docker 镜像构建能力 (来自 add-docker-deployment 提案)

但是缺少:
1. **基础 Kubernetes 资源清单**: Deployment, DaemonSet, Service
2. **RBAC 配置**: Agent 必要权限
3. **部署脚本**: 简单的部署和清理脚本
4. **部署文档**: 基础部署指南

## 目标

### 主要目标

1. **Server Deployment 配置**
   - 单副本 Deployment
   - 基本的环境变量配置
   - ClusterIP Service

2. **Agent DaemonSet 配置**
   - DaemonSet 资源 (每节点一个)
   - 特权容器配置
   - BPF 文件系统挂载
   - RBAC 权限配置

3. **PostgreSQL 部署**
   - 单副本 Deployment
   - ClusterIP Service

4. **部署脚本**
   - 一键部署脚本
   - 清理脚本

5. **基础文档**
   - 快速开始指南
   - 基础故障排查

### 非目标 (未来可扩展)

- 生产级高可用配置 (多副本、StatefulSet)
- Ingress 配置
- Kustomize 配置
- Helm Charts
- 监控和日志集成
- 高级安全配置

## 设计概要

### 资源架构

```
Namespace: microsegment
├── ConfigMap: microsegment-config (基础配置)
├── PostgreSQL
│   ├── Deployment: microsegment-postgres (1 replica)
│   └── Service: microsegment-postgres
├── Server
│   ├── Deployment: microsegment-server (1 replica)
│   └── Service: microsegment-server
└── Agent
    ├── ServiceAccount: microsegment-agent
    ├── ClusterRole: microsegment-agent
    ├── ClusterRoleBinding: microsegment-agent
    └── DaemonSet: microsegment-agent
```

### Agent DaemonSet 关键配置

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
        volumeMounts:
        - name: bpf
          mountPath: /sys/fs/bpf
      volumes:
      - name: bpf
        hostPath:
          path: /sys/fs/bpf
```

## 依赖关系

### 前置条件
- Server 和 Agent 组件已实现
- Docker 镜像可构建
- K8s 测试集群可用 (版本 >= 1.20)

### 阻塞项
- 无

## 成功标准

1. **功能完整性**
   - ✅ 可通过 kubectl apply 部署到 K8s 集群
   - ✅ Server 可正常启动
   - ✅ Agent 在每个节点运行

2. **可用性**
   - ✅ 提供快速开始文档
   - ✅ 提供一键部署脚本
   - ✅ Agent 可加载 eBPF 程序
   - ✅ Agent 可连接 Server

## 风险与缓解

### 风险: Agent 需要特权容器
- **描述**: Agent 需要 privileged: true
- **缓解**: 测试环境可接受,在文档中说明

## 后续工作

测试环境验证通过后,可考虑:
1. 生产级高可用配置
2. Helm Charts
3. 监控和日志集成

## 时间估算

**总计**: 1-2 天
