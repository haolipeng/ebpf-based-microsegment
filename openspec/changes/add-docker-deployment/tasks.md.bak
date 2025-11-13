# 任务列表: Docker 容器化部署

## 阶段 1: Dockerfile 创建 (1 天)

### 1.1 创建 Server Dockerfile
- [ ] 创建 `deploy/docker/Dockerfile.server`
- [ ] 实现多阶段构建 (builder + runtime)
- [ ] 使用 Alpine Linux 作为运行时基础镜像
- [ ] 配置非 root 用户运行
- [ ] 暴露 8080 (HTTP) 和 9090 (gRPC) 端口
- [ ] 添加健康检查指令
- [ ] 验证: 构建成功,镜像大小 < 50MB
- [ ] 验证: 容器可正常启动并响应请求

### 1.2 创建 Agent Dockerfile
- [ ] 创建 `deploy/docker/Dockerfile.agent`
- [ ] 使用 Ubuntu 22.04 作为基础镜像 (eBPF 兼容性)
- [ ] 安装 libbpf, bpftool 等 eBPF 工具
- [ ] 配置 eBPF 程序和 CO-RE 字节码复制
- [ ] 暴露 8081 (API) 端口
- [ ] 验证: 构建成功,镜像大小 < 200MB
- [ ] 验证: 容器可加载 eBPF 程序 (特权模式)

### 1.3 创建 .dockerignore
- [ ] 创建 `.dockerignore` 文件
- [ ] 排除 .git, tests, docs 等不必要文件
- [ ] 排除构建产物和临时文件
- [ ] 排除 IDE 配置文件
- [ ] 优化构建速度和上下文大小
- [ ] 验证: 构建上下文大小减小

## 阶段 2: Docker Compose 配置 (0.5 天)

### 2.1 创建主 Docker Compose 文件
- [ ] 创建 `deploy/docker/docker-compose.yml`
- [ ] 配置 PostgreSQL 服务 (postgres:15-alpine)
- [ ] 配置健康检查 (pg_isready)
- [ ] 配置 named volume (postgres_data)
- [ ] 验证: PostgreSQL 服务可正常启动

### 2.2 配置 Server 服务
- [ ] 在 docker-compose.yml 中添加 Server 服务
- [ ] 配置依赖 PostgreSQL (depends_on + condition: service_healthy)
- [ ] 配置环境变量 (数据库连接、端口、日志级别)
- [ ] 配置端口映射 (8080:8080, 9090:9090)
- [ ] 配置健康检查 (wget /health)
- [ ] 配置 restart 策略 (unless-stopped)
- [ ] 验证: Server 服务可正常启动并连接数据库

### 2.3 配置 Agent 服务
- [ ] 在 docker-compose.yml 中添加 Agent 服务
- [ ] 配置依赖 Server (depends_on + condition: service_healthy)
- [ ] 配置 privileged: true
- [ ] 配置 network_mode: host
- [ ] 配置 pid: host
- [ ] 配置 /sys/fs/bpf 挂载 (rw)
- [ ] 配置 /sys/kernel/debug 挂载 (ro)
- [ ] 配置环境变量 (Server URL、API 端口)
- [ ] 配置 restart 策略 (unless-stopped)
- [ ] 验证: Agent 服务可正常启动并加载 eBPF 程序

### 2.4 配置 Web UI 服务 (可选)
- [ ] 在 docker-compose.yml 中添加 Web UI 服务
- [ ] 配置端口映射 (3000:3000)
- [ ] 配置源代码挂载
- [ ] 配置环境变量 (API URL)
- [ ] 配置依赖 Server
- [ ] 验证: Web UI 可访问

### 2.5 配置网络和卷
- [ ] 定义 bridge network (microsegment)
- [ ] 配置 PostgreSQL volume (postgres_data)
- [ ] 验证: 网络隔离正常工作
- [ ] 验证: 数据持久化正常工作

## 阶段 3: 开发环境配置 (0.5 天)

### 3.1 创建开发环境 Override
- [ ] 创建 `deploy/docker/docker-compose.dev.yml`
- [ ] Server 服务配置源代码卷挂载 (src/server)
- [ ] Agent 服务配置源代码卷挂载 (src/agent)
- [ ] 配置使用 builder stage (包含开发工具)
- [ ] 配置 debug 日志级别
- [ ] 配置调试端口暴露
- [ ] 修改 command 为 `go run` 实现热重载
- [ ] 验证: 开发环境可正常启动
- [ ] 验证: 代码修改后自动重载

### 3.2 创建环境变量模板
- [ ] 创建 `deploy/docker/.env.example`
- [ ] 定义数据库密码变量
- [ ] 定义日志级别变量
- [ ] 定义端口变量
- [ ] 添加变量说明注释
- [ ] 验证: 环境变量可正常加载

## 阶段 4: 配置文件模板 (0.5 天)

### 4.1 创建 Server Docker 配置
- [ ] 创建 `deploy/config/docker/server.yaml`
- [ ] 配置 HTTP 服务 (host: 0.0.0.0, port: 8080)
- [ ] 配置 gRPC 服务 (host: 0.0.0.0, port: 9090)
- [ ] 配置数据库连接 (使用环境变量)
- [ ] 配置日志 (level: info, format: json)
- [ ] 验证: 配置可正常加载

### 4.2 创建 Agent Docker 配置
- [ ] 创建 `deploy/config/docker/agent.yaml`
- [ ] 配置 Server 连接 URL (使用环境变量)
- [ ] 配置 API 端口 (8081)
- [ ] 配置 eBPF 程序路径
- [ ] 配置日志 (level: info, format: json)
- [ ] 验证: 配置可正常加载

### 4.3 创建配置说明文档
- [ ] 创建 `deploy/config/README.md`
- [ ] 说明 Docker 配置文件结构
- [ ] 说明环境变量覆盖机制
- [ ] 提供配置示例
- [ ] 说明密钥管理最佳实践

## 阶段 5: 部署脚本 (1 天)

### 5.1 创建 Docker 部署脚本
- [ ] 创建 `deploy/scripts/deploy-docker.sh`
- [ ] 实现检查 Docker 环境步骤
- [ ] 实现检查 Docker Compose 环境步骤
- [ ] 实现验证配置文件步骤
- [ ] 实现构建镜像步骤 (docker-compose build)
- [ ] 实现启动 PostgreSQL 步骤
- [ ] 实现等待数据库就绪步骤
- [ ] 实现运行数据库迁移步骤
- [ ] 实现启动所有服务步骤 (docker-compose up -d)
- [ ] 实现健康检查步骤
- [ ] 实现输出访问信息步骤
- [ ] 添加错误处理和退出码
- [ ] 添加彩色输出和进度提示
- [ ] 验证: 脚本可一键部署完整系统

### 5.2 创建健康检查脚本
- [ ] 创建 `deploy/scripts/health-check-docker.sh`
- [ ] 检查容器运行状态 (docker-compose ps)
- [ ] 检查 PostgreSQL 健康状态
- [ ] 检查 Server HTTP API (/health)
- [ ] 检查 Server gRPC 端口
- [ ] 检查 Agent API 端口
- [ ] 检查 Agent eBPF 程序加载状态
- [ ] 返回正确的退出码 (0=健康, 1=不健康)
- [ ] 输出详细的检查结果
- [ ] 添加超时机制
- [ ] 验证: 可准确检测系统健康状态

### 5.3 创建停止脚本
- [ ] 创建 `deploy/scripts/stop-docker.sh`
- [ ] 实现停止所有容器 (docker-compose down)
- [ ] 支持 --volumes 参数删除数据
- [ ] 支持 --rmi 参数删除镜像
- [ ] 验证: 可正确清理环境

### 5.4 创建日志查看脚本
- [ ] 创建 `deploy/scripts/logs-docker.sh`
- [ ] 支持查看指定服务日志
- [ ] 支持 --follow 参数实时查看
- [ ] 支持 --tail 参数限制行数
- [ ] 验证: 可方便查看日志

## 阶段 6: 健康检查端点实现 (0.5 天)

### 6.1 实现 Server 健康检查端点
- [ ] 检查 `src/server/pkg/api/handlers/health.go` 是否存在
- [ ] 如不存在,创建健康检查处理器
- [ ] 实现 `GET /health` 端点
- [ ] 检查数据库连接
- [ ] 检查 gRPC 服务状态
- [ ] 返回 JSON 格式响应
- [ ] 实现 `GET /ready` 端点 (就绪检查)
- [ ] 实现 `GET /version` 端点 (版本信息)
- [ ] 在 router.go 中注册路由
- [ ] 验证: 端点返回正确的状态和数据

### 6.2 添加版本信息注入
- [ ] 修改 Dockerfile.server
- [ ] 使用 --build-arg 传递版本信息
- [ ] 使用 -ldflags 注入版本到二进制
- [ ] 验证: 版本信息正确显示

## 阶段 7: 文档编写 (0.5 天)

### 7.1 创建 Docker 部署文档
- [ ] 创建 `deploy/docker/README.md`
- [ ] 编写前置要求 (Docker, Docker Compose 版本)
- [ ] 编写快速开始步骤
- [ ] 编写配置说明 (环境变量、配置文件)
- [ ] 编写服务管理命令 (启动、停止、重启)
- [ ] 编写日志查看方法
- [ ] 编写数据备份和恢复方法
- [ ] 编写常见问题排查 (端口冲突、权限问题、网络问题)
- [ ] 添加架构图和命令示例

### 7.2 创建配置参考文档
- [ ] 在 deploy/docker/README.md 中添加配置章节
- [ ] 列出所有环境变量及说明
- [ ] 提供配置文件示例
- [ ] 说明配置优先级
- [ ] 说明密钥管理方法

### 7.3 创建安全说明文档
- [ ] 在 deploy/docker/README.md 中添加安全章节
- [ ] 说明 Agent 特权要求和风险
- [ ] 提供最小权限配置建议
- [ ] 说明网络隔离策略
- [ ] 说明密钥管理最佳实践

### 7.4 更新主 README
- [ ] 在主 README.md 中添加 Docker 部署章节
- [ ] 链接到详细的 Docker 部署文档
- [ ] 提供快速开始命令

## 阶段 8: 测试验证 (0.5 天)

### 8.1 Docker 镜像测试
- [ ] 在干净环境测试 Server 镜像构建
- [ ] 验证 Server 镜像大小 < 50MB
- [ ] 验证 Server 镜像安全扫描通过 (Trivy)
- [ ] 在干净环境测试 Agent 镜像构建
- [ ] 验证 Agent 镜像大小 < 200MB
- [ ] 验证 Agent 镜像安全扫描通过 (Trivy)

### 8.2 Docker Compose 部署测试
- [ ] 在干净环境测试完整部署
- [ ] 验证所有容器正常启动
- [ ] 验证所有健康检查通过
- [ ] 验证 Server API 可访问
- [ ] 验证 Agent eBPF 程序加载
- [ ] 验证数据持久化 (重启后数据保留)
- [ ] 验证容器重启机制 (手动停止后自动重启)

### 8.3 开发环境测试
- [ ] 使用 docker-compose.dev.yml 启动开发环境
- [ ] 验证源代码挂载生效
- [ ] 验证代码修改后自动重载
- [ ] 验证调试端口可访问

### 8.4 部署脚本测试
- [ ] 测试 deploy-docker.sh 一键部署
- [ ] 测试 health-check-docker.sh 健康检查
- [ ] 测试 stop-docker.sh 清理环境
- [ ] 测试 logs-docker.sh 日志查看
- [ ] 验证脚本错误处理正确

### 8.5 文档验证
- [ ] 让新用户按文档执行部署
- [ ] 收集反馈并改进文档
- [ ] 确保所有命令可执行
- [ ] 确保所有链接有效
- [ ] 确保截图和示例准确

## 阶段 9: 安全审查和优化 (0.5 天)

### 9.1 镜像安全审查
- [ ] 使用 Trivy 扫描 Server 镜像漏洞
- [ ] 使用 Trivy 扫描 Agent 镜像漏洞
- [ ] 修复发现的高危和中危漏洞
- [ ] 审查 Dockerfile 安全最佳实践
- [ ] 确保不包含敏感信息

### 9.2 配置安全审查
- [ ] 审查 docker-compose.yml 安全配置
- [ ] 确保密钥不硬编码
- [ ] 审查 Agent 特权配置必要性
- [ ] 验证网络隔离正确
- [ ] 审查卷挂载权限

### 9.3 性能优化
- [ ] 优化 Docker 镜像构建速度
- [ ] 优化镜像层缓存利用
- [ ] 优化 .dockerignore 减小上下文
- [ ] 测试构建时间并记录
- [ ] 优化容器资源配置 (内存、CPU 限制)

### 9.4 监控集成
- [ ] 在 docker-compose.yml 中添加 Prometheus 注解
- [ ] 验证 Server /metrics 端点可访问
- [ ] 验证 Agent /metrics 端点可访问
- [ ] 提供 Prometheus 配置示例 (可选)

## 总计

- **总任务数**: 106 任务
- **预估工时**: 4 天
- **阶段数**: 9 个阶段

## 里程碑

1. **M1 (第 1 天)**: Dockerfile 和 .dockerignore 完成
2. **M2 (第 1.5 天)**: Docker Compose 配置完成
3. **M3 (第 2.5 天)**: 配置文件和脚本完成
4. **M4 (第 3.5 天)**: 文档和健康检查完成
5. **M5 (第 4 天)**: 测试、安全审查和优化完成,提案可归档
