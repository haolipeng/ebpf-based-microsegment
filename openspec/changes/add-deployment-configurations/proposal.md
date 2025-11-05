# 提案: 添加部署配置

**变更 ID**: `add-deployment-configurations`
**提案日期**: 2025-11-04
**状态**: 提案中
**优先级**: P1
**预估工作量**: 3-4 天

---

## 📋 概述

提供生产级部署配置,包括 Docker、Kubernetes、Systemd 等多种部署方式,以及监控和告警配置。

## 🎯 目标

1. **Docker 部署** - Dockerfile 和 docker-compose.yml
2. **Kubernetes 部署** - DaemonSet (Agent) + Deployment (Server)
3. **Systemd 服务** - Linux 系统服务配置
4. **监控配置** - Prometheus + Grafana
5. **部署文档** - 完整的部署指南

## 🏗️ 核心配置

### Docker 部署

```yaml
# docker-compose.yml
version: '3.8'

services:
  postgres:
    image: timescale/timescaledb:latest-pg14
    environment:
      POSTGRES_DB: microsegment
      POSTGRES_USER: admin
      POSTGRES_PASSWORD: secret
    ports:
      - "5432:5432"

  server:
    build:
      context: .
      dockerfile: deployments/docker/Dockerfile.server
    depends_on:
      - postgres
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      DB_URL: "postgres://admin:secret@postgres:5432/microsegment"

  agent:
    build:
      context: .
      dockerfile: deployments/docker/Dockerfile.agent
    cap_add:
      - SYS_ADMIN  # Required for eBPF
    privileged: true
    network_mode: host
    environment:
      MODE: agent-server
      SERVER_URL: "grpc://server:9090"
```

### Kubernetes 部署

```yaml
# agent-daemonset.yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: microsegment-agent
spec:
  selector:
    matchLabels:
      app: microsegment-agent
  template:
    metadata:
      labels:
        app: microsegment-agent
    spec:
      hostNetwork: true
      hostPID: true
      containers:
      - name: agent
        image: microsegment-agent:latest
        securityContext:
          privileged: true
        env:
        - name: MODE
          value: "agent-server"
        - name: SERVER_URL
          value: "grpc://microsegment-server:9090"
```

### Prometheus 监控

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'microsegment-server'
    static_configs:
      - targets: ['server:8080']
    metrics_path: /metrics

  - job_name: 'microsegment-agents'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        regex: microsegment-agent
        action: keep
```

## ✅ 验收标准

- [ ] Docker 部署成功启动所有组件
- [ ] Kubernetes 部署成功
- [ ] Systemd 服务正常运行
- [ ] Prometheus 正常采集指标
- [ ] Grafana 仪表板可用
- [ ] 提供完整部署文档

## 🔗 依赖

**前置依赖**:
- add-server-component
- refactor-agent-for-remote-reporting

---

**提案人**: Claude Code
