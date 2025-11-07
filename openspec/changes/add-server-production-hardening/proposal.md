# 提案：服务端生产环境加固

**Change ID**: `add-server-production-hardening`
**创建时间**: 2025-11-06
**状态**: 提案
**优先级**: P2 (中等 - 生产就绪)
**预计工作量**: 4-7 天

---

## 为什么

微隔离服务端 MVP (已归档为 `2025-11-06-add-server-component`) 提供了核心功能,但缺少生产部署所需的特性:

**当前缺陷**:
- **API 不完整**: 策略 CRUD (仅实现了 GET,缺少 POST/PUT/DELETE)
- **API 不完整**: Agent 详情视图 (仅实现了列表,缺少 GET /:id)
- **无 Manager 层**: 业务逻辑分散在各个服务中
- **无部署方案**: 没有 Docker 镜像或 docker-compose
- **无 CI/CD**: 没有自动化构建/部署流水线
- **无监控**: 没有 Prometheus 指标或健康检查
- **无安全性**: 没有 TLS、认证或限流

**生产环境需求**:
- 完整的 REST APIs 用于运维管理
- 适当的关注点分离 (Manager 层)
- 容器化部署与编排
- 可观测性 (指标、日志、追踪)
- 安全加固
- 自动化部署流水线

## 变更内容

添加生产就绪的特性和加固措施:

### 1. 完善 HTTP APIs
- **策略 CRUD**: 添加 POST/PUT/DELETE 端点
- **Agent 管理**: 添加 GET /agents/:id 返回详细指标
- **错误处理**: 标准化错误响应
- **输入验证**: 请求验证中间件

### 2. Manager 层 (业务逻辑)
- **AgentManager**: 集中式 agent 生命周期管理
- **PolicyDistributor**: 策略变更广播逻辑
- **HealthMonitor**: 后台 agent 健康检查

### 3. 部署与编排
- **Dockerfile**: 多阶段 Docker 镜像
- **docker-compose.yml**: 完整技术栈编排 (server + postgres)
- **Helm charts** (可选): Kubernetes 部署
- **环境配置**: 开发/预发/生产配置

### 4. 可观测性
- **Prometheus 指标**: 请求计数、延迟、错误率
- **结构化日志**: 带关联 ID 的 JSON 日志
- **健康检查**: 存活和就绪探针
- **Grafana 仪表板** (可选): 预构建仪表板

### 5. 安全加固
- **TLS 支持**: HTTPS 和 gRPC TLS
- **API 认证**: 基于 token 的认证 (可选 mTLS)
- **限流**: 按 IP 和按 agent 的速率限制
- **输入清理**: SQL 注入防护

### 6. CI/CD 流水线
- **GitHub Actions**: 构建、测试和发布工作流
- **Docker Registry**: 推送镜像到仓库
- **版本标记**: 语义化版本控制
- **发布自动化**: 自动生成发布说明

---

## 成功标准

- [ ] 策略 CRUD APIs 完全可用 (POST/PUT/DELETE)
- [ ] Agent 详情 API 返回指标和状态
- [ ] Manager 层实现业务逻辑
- [ ] Docker 镜像构建成功 (<500MB)
- [ ] docker-compose 启动完整技术栈
- [ ] 在 /metrics 暴露 Prometheus 指标
- [ ] TLS 配置文档化且可测试
- [ ] CI/CD 流水线构建并发布镜像
- [ ] 生产部署指南文档化

---

## 依赖

- 需要 server MVP (已归档)
- 需要测试提案 (用于验证)
- 核心特性无新的外部依赖

---

## 范围

**包含范围**:
- 完善 HTTP APIs
- Manager 层分离
- Docker 部署
- 基础可观测性 (指标、日志)
- CI/CD 流水线
- 安全配置

**不包含范围**:
- 高级特性 (多租户、RBAC)
- 云平台特定部署 (AWS/GCP 特定配置)
- 高级监控 (分布式追踪)
- 负载均衡配置
- 备份/恢复流程
