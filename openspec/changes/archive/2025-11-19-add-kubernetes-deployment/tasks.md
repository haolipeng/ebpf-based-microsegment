# 任务列表: Kubernetes 部署配置 (测试环境)

## 概述

本任务列表聚焦于实现在 Kubernetes 集群上部署和测试 eBPF 微分段系统的基本能力。

## 阶段 1: 基础部署配置

### 1.1 创建基础资源
- [x] 创建 `deploy/kubernetes/namespace.yaml` - 定义 microsegment namespace
- [x] 创建 `deploy/kubernetes/rbac.yaml` - Agent RBAC 权限配置
- [x] 创建 `deploy/kubernetes/configmap.yaml` - 基础配置

### 1.2 PostgreSQL 部署
- [x] 创建 `deploy/kubernetes/postgres.yaml` - 简单的 PostgreSQL Deployment (单副本)
- [x] 配置 Service 用于集群内访问

### 1.3 Server 部署
- [x] 创建 `deploy/kubernetes/server.yaml` - Server Deployment (单副本)
- [x] 配置基本的环境变量和数据库连接
- [x] 创建 ClusterIP Service

### 1.4 Agent 部署
- [x] 创建 `deploy/kubernetes/agent.yaml` - Agent DaemonSet
- [x] 配置必要的特权和主机访问权限
- [x] 配置 eBPF 文件系统挂载

## 阶段 2: 部署脚本

### 2.1 基础部署脚本
- [x] 创建 `deploy/scripts/deploy-k8s.sh` - 一键部署脚本
- [x] 检查 kubectl 可用性
- [x] 按顺序部署所有组件
- [x] 等待资源就绪

### 2.2 清理脚本
- [x] 创建 `deploy/scripts/undeploy-k8s.sh` - 清理部署资源

## 阶段 3: 基础文档

### 3.1 部署文档
- [x] 创建 `deploy/kubernetes/README.md`
- [x] 前置条件说明
- [x] 快速开始步骤
- [x] 基本故障排查指南

## 总计

- **总任务数**: 16 个核心任务
- **预估工时**: 1-2 天
- **目标**: 实现基本的 K8s 部署和测试能力

## 注意事项

- 此版本专注于测试环境,不考虑生产级别的高可用、监控等特性
- 所有组件使用单副本简化配置
- 不包含 Ingress、Kustomize、高级安全配置等复杂特性
- 后续可根据需要逐步增强
