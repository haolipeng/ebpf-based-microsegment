# 实施任务：服务端生产环境加固

**Change ID**: `add-server-production-hardening`
**创建时间**: 2025-11-06
**预计工作量**: 4-7 天
**当前状态**: 进行中
**进度**: 2/19 任务完成 (阶段 6: CI/CD 流水线已完成)

---

## 任务概览

| 阶段 | 任务数 | 预计时间 | 状态 |
|------|--------|----------|------|
| 阶段 1: 完善 HTTP APIs | 3 个任务 | 1 天 | ⬜ 未开始 |
| 阶段 2: Manager 层 | 3 个任务 | 1 天 | ⬜ 未开始 |
| 阶段 3: Docker 部署 | 3 个任务 | 1 天 | ⬜ 未开始 |
| 阶段 4: 可观测性 | 3 个任务 | 1 天 | ⬜ 未开始 |
| 阶段 5: 安全性 | 3 个任务 | 1.5 天 | ⬜ 未开始 |
| 阶段 6: CI/CD 流水线 | 2 个任务 | 1 天 | ✅ **已完成** |
| 阶段 7: 文档 | 2 个任务 | 0.5 天 | ⬜ 未开始 |
| **总计** | **19 个任务** | **7 天** | **10.5% 完成** |

---

## 当前状态说明

**最后更新**: 2025-11-07

### 已完成内容 (来自 Server MVP)
- ✅ 基础 HTTP API 框架 (Gin)
- ✅ 基础 gRPC 服务
- ✅ PostgreSQL 存储层
- ✅ Flow 数据采集 API
- ✅ WebSocket 实时流推送
- ✅ 基础健康检查端点
- ✅ Agent 注册与心跳

### 待实施内容
- ❌ 策略 CRUD API (仅有 GET,缺 POST/PUT/DELETE)
- ❌ Agent 详情 API (仅有列表,缺详情端点)
- ❌ Manager 业务逻辑层
- ❌ Docker 容器化部署
- ❌ Prometheus 指标
- ❌ TLS 支持
- ❌ 限流与认证
- ❌ CI/CD 自动化

---

## 阶段 1: 完善 HTTP APIs (1 天)

### 任务 1.1: 实现策略 CRUD API
- [ ] 创建 `pkg/api/handlers/policy.go`
- [ ] 实现 POST /api/v1/policies (创建策略)
- [ ] 实现 PUT /api/v1/policies/:id (更新策略)
- [ ] 实现 DELETE /api/v1/policies/:id (删除策略)
- [ ] 添加输入验证中间件
- [ ] 为所有端点添加 HTTP 测试
- [ ] 更新 API 文档

### 任务 1.2: 实现 Agent 详情 API
- [ ] 更新 `cmd/main.go` 中的 agent 路由
- [ ] 实现 GET /api/v1/agents/:id
- [ ] 返回 agent 详情及最新指标
- [ ] 返回 agent 连接状态
- [ ] 为端点添加 HTTP 测试
- [ ] 更新 API 文档

### 任务 1.3: 统一错误处理
- [ ] 创建 `pkg/api/errors.go`
- [ ] 定义标准错误响应格式
- [ ] 实现错误处理中间件
- [ ] 将存储层错误映射到 HTTP 状态码
- [ ] 添加带关联 ID 的错误日志

---

## 阶段 2: Manager 层 (1 天)

### 任务 2.1: 实现 AgentManager
- [ ] 创建 `pkg/manager/agent_manager.go`
- [ ] 实现 `OnAgentRegistered()`
- [ ] 实现 `OnAgentUnregistered()`
- [ ] 实现 `GetPendingCommands()`
- [ ] 实现 `SendCommand()`
- [ ] 与 AgentService 集成
- [ ] 添加单元测试

### 任务 2.2: 实现 PolicyDistributor
- [ ] 创建 `pkg/manager/policy_distributor.go`
- [ ] 实现 `OnPolicyCreated()` 广播
- [ ] 实现 `OnPolicyUpdated()` 广播
- [ ] 实现 `OnPolicyDeleted()` 广播
- [ ] 与 PolicyService 集成
- [ ] 添加单元测试

### 任务 2.3: 实现 HealthMonitor
- [ ] 创建 `pkg/manager/health_monitor.go`
- [ ] 实现后台心跳检查
- [ ] 超时后标记 agent 为不健康
- [ ] 实现 Start() 和 Stop()
- [ ] 与 main.go 启动流程集成
- [ ] 添加单元测试

---

## 阶段 3: Docker 部署 (1 天)

### 任务 3.1: 创建 Dockerfile
- [ ] 创建 `src/server/Dockerfile`
- [ ] 使用多阶段构建 (builder + runtime)
- [ ] 优化镜像大小 (<500MB)
- [ ] 添加健康检查
- [ ] 添加非 root 用户
- [ ] 测试镜像构建

### 任务 3.2: 创建 docker-compose
- [ ] 创建 `deployments/docker-compose.yml`
- [ ] 添加带 TimescaleDB 的 postgres 服务
- [ ] 添加带健康检查的 server 服务
- [ ] 添加数据持久化的卷挂载
- [ ] 添加环境变量配置
- [ ] 添加网络配置
- [ ] 测试完整技术栈启动

### 任务 3.3: 创建环境配置
- [ ] 创建 `deployments/env/dev.env`
- [ ] 创建 `deployments/env/staging.env`
- [ ] 创建 `deployments/env/prod.env.example`
- [ ] 编写配置选项文档
- [ ] 添加配置验证

---

## 阶段 4: 可观测性 (1 天)

### 任务 4.1: 添加 Prometheus 指标
- [ ] 添加 `github.com/prometheus/client_golang` 依赖
- [ ] 创建 `pkg/metrics/metrics.go`
- [ ] 添加 HTTP 请求指标 (计数、耗时、状态)
- [ ] 添加 gRPC 请求指标
- [ ] 添加数据库查询指标
- [ ] 暴露 /metrics 端点
- [ ] 测试指标采集

### 任务 4.2: 增强结构化日志
- [ ] 添加关联 ID 中间件
- [ ] 添加请求/响应日志
- [ ] 添加错误堆栈跟踪
- [ ] 按环境配置日志级别
- [ ] 编写日志格式文档

### 任务 4.3: 实现健康检查
- [ ] 增强 GET /health 端点
- [ ] 添加存活探针 (server running)
- [ ] 添加就绪探针 (DB connected)
- [ ] 添加依赖健康检查
- [ ] 返回详细健康状态

---

## 阶段 5: 安全性 (1.5 天)

### 任务 5.1: 添加 TLS 支持
- [ ] 在 config.yaml 中添加 TLS 配置
- [ ] 实现 HTTPS 服务器
- [ ] 实现 gRPC TLS
- [ ] 生成测试用的自签名证书
- [ ] 编写 TLS 设置文档
- [ ] 测试启用 TLS 的情况

### 任务 5.2: 实现 API 认证 (可选)
- [ ] 创建 `pkg/auth/` 包
- [ ] 实现基于 token 的认证
- [ ] 添加认证中间件
- [ ] 添加 /api/v1/auth/login 端点
- [ ] 编写认证流程文档
- [ ] 添加认证测试

### 任务 5.3: 添加限流
- [ ] 添加 `golang.org/x/time/rate` 依赖
- [ ] 创建 `pkg/middleware/ratelimit.go`
- [ ] 实现按 IP 限流
- [ ] 实现按 agent 限流
- [ ] 添加限流响应头
- [ ] 测试限流行为

---

## 阶段 6: CI/CD 流水线 (1 天)

### 任务 6.1: 创建 GitHub Actions 工作流
- [x] 创建 `.github/workflows/server-ci.yml`
- [x] 添加 build 任务 (编译 server)
- [x] 添加 test 任务 (运行测试及覆盖率)
- [x] 添加 lint 任务 (golangci-lint)
- [x] 添加安全扫描 (gosec)
- [x] 添加 Docker 构建任务
- [x] 配置工作流触发器 (push, PR, tags)

### 任务 6.2: 设置 Docker Registry 发布
- [x] 向工作流添加 Docker push 任务
- [x] 配置 GitHub Container Registry (ghcr.io)
- [x] 添加语义化版本标签
- [x] 添加 latest 标签
- [x] 添加分支和 SHA 标签
- [x] 添加自动 release 创建
- [x] 编写镜像拉取文档 (docs/docker-registry.md)
- [x] 增强 Dockerfile (多阶段构建、非root用户、健康检查)

---

## 阶段 7: 文档 (0.5 天)

### 任务 7.1: 更新服务端文档
- [ ] 更新 `src/server/README.md`
  - [ ] 添加生产部署章节
  - [ ] 添加 Docker 部署指南
  - [ ] 添加 TLS 配置指南
  - [ ] 添加监控设置指南
  - [ ] 添加故障排查章节
- [ ] 创建 `docs/deployment-guide.md`
- [ ] 创建 `docs/security-guide.md`

### 任务 7.2: 创建运维手册
- [ ] 创建 `docs/runbooks/startup.md`
- [ ] 创建 `docs/runbooks/backup-restore.md`
- [ ] 创建 `docs/runbooks/scaling.md`
- [ ] 创建 `docs/runbooks/troubleshooting.md`

---

## 验收标准

### API 完整性
- [ ] 所有策略 CRUD 操作正常工作
- [ ] Agent 详情 API 返回完整信息
- [ ] 错误响应遵循标准格式

### 部署
- [ ] Docker 镜像在 5 分钟内构建完成
- [ ] docker-compose 成功启动所有服务
- [ ] 健康检查通过
- [ ] 服务失败时自动重启

### 安全性
- [ ] TLS 在 HTTPS 和 gRPC 上正常工作
- [ ] 限流阻止过量请求
- [ ] 代码中无硬编码密钥

### CI/CD
- [ ] 每次提交都会运行工作流
- [ ] 自动发布 Docker 镜像
- [ ] 构建失败阻止合并

---

**预计总工作量**: 7 天
**依赖**: Server MVP (已归档), 测试提案 (可选)
**后续步骤**: 执行阶段 1 完善 APIs
