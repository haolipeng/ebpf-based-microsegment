# 任务列表: 最小化 Docker 部署

## 阶段 1: Dockerfile 创建 (2 小时)

### 1.1 创建 Server Dockerfile
- [ ] 创建 `deploy/docker/Dockerfile.server`
- [ ] 实现多阶段构建 (builder + runtime)
- [ ] 使用 Ubuntu 22.04 作为基础镜像
- [ ] 暴露 8080 (HTTP) 和 9090 (gRPC) 端口
- [ ] 配置启动命令
- [ ] 验证: 构建成功,容器可正常启动

**预估时间**: 1 小时

### 1.2 创建 Agent Dockerfile
- [ ] 创建 `deploy/docker/Dockerfile.agent`
- [ ] 实现多阶段构建 (builder + runtime)
- [ ] 使用 Ubuntu 22.04 作为基础镜像
- [ ] 安装必要的网络工具 (iproute2, iptables)
- [ ] 暴露 8081 (API) 端口
- [ ] 配置启动命令
- [ ] 验证: 构建成功,容器可正常启动

**预估时间**: 1 小时

## 阶段 2: Docker Compose 配置 (1 小时)

### 2.1 完善 Docker Compose 文件
- [ ] 将 `docker-compose.simple.yml` 重命名或创建新的 `docker-compose.yml`
- [ ] 添加 Server 服务定义
- [ ] 配置 Server 依赖 PostgreSQL (depends_on + healthcheck)
- [ ] 配置 Server 环境变量 (数据库连接)
- [ ] 配置 Server 端口映射 (8080:8080, 9090:9090)
- [ ] 配置 Server 配置文件挂载
- [ ] 验证: Server 服务可正常启动并连接数据库

**预估时间**: 0.5 小时

### 2.2 添加 Agent 服务
- [ ] 在 docker-compose.yml 中添加 Agent 服务
- [ ] 配置 Agent 依赖 Server (depends_on)
- [ ] 配置 privileged: true
- [ ] 配置 network_mode: host
- [ ] 配置 pid: host
- [ ] 配置 /sys/fs/bpf 挂载 (rw)
- [ ] 配置 Agent 环境变量 (Server URL)
- [ ] 配置 Agent 配置文件挂载
- [ ] 验证: Agent 服务可正常启动并连接 Server

**预估时间**: 0.5 小时

## 阶段 3: 配置文件 (1 小时)

### 3.1 创建 Server Docker 配置
- [ ] 创建 `config/docker-server.yaml`
- [ ] 配置数据库连接 (使用环境变量)
- [ ] 配置 API 监听地址
- [ ] 配置 gRPC 监听地址
- [ ] 配置日志级别
- [ ] 验证: 配置文件格式正确

**预估时间**: 0.5 小时

### 3.2 创建 Agent Docker 配置
- [ ] 创建 `config/docker-agent.yaml`
- [ ] 配置 Server 连接地址 (使用环境变量)
- [ ] 配置网络接口
- [ ] 配置 API 监听地址
- [ ] 配置日志级别
- [ ] 验证: 配置文件格式正确

**预估时间**: 0.5 小时

## 阶段 4: 部署文档 (可选,30 分钟)

### 4.1 创建 Docker 部署 README
- [ ] 创建 `deploy/docker/README.md`
- [ ] 添加部署前提条件
- [ ] 添加构建步骤说明
- [ ] 添加部署步骤说明 (`docker-compose up`)
- [ ] 添加验证步骤
- [ ] 添加常见问题说明

**预估时间**: 0.5 小时

## 阶段 5: 测试验证 (1 小时)

### 5.1 本地测试
- [ ] 构建 Server 镜像
- [ ] 构建 Agent 镜像
- [ ] 使用 docker-compose up 启动所有服务
- [ ] 验证 PostgreSQL 启动成功
- [ ] 验证 Server 启动成功并连接数据库
- [ ] 验证 Agent 启动成功并连接 Server
- [ ] 通过 Server API 创建测试策略
- [ ] 验证 Agent 可正常工作

**预估时间**: 1 小时

## 总结

**总预估时间**: 约 5 小时 (0.5 天)

**关键任务**:
1. Dockerfile 创建 (2 小时)
2. Docker Compose 配置 (1 小时)
3. 配置文件 (1 小时)
4. 测试验证 (1 小时)

**成功标准**:
- [ ] 可通过 `docker build` 成功构建 Server 和 Agent 镜像
- [ ] 可通过 `docker-compose up` 一键部署完整系统
- [ ] Server 可正常连接 PostgreSQL
- [ ] Agent 可正常连接 Server
- [ ] 可以进行基本的功能测试

**注意事项**:
- 本阶段不包含镜像优化 (大小、安全性等)
- 本阶段不包含开发环境配置
- 本阶段不包含自动化部署脚本
- 后续可以基于此基础版本进行优化和扩展
