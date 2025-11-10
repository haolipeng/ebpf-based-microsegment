# 任务列表: Systemd 服务集成

## 阶段 1: Service 文件创建 (0.5 天)

### 1.1 创建 Server Systemd Service
- [ ] 创建 `deploy/systemd/microsegment-server.service`
- [ ] 配置 [Unit] 段 (Description, Documentation, After, Wants, Requires)
- [ ] 配置依赖 postgresql.service
- [ ] 添加 ConditionPathExists 检查
- [ ] 配置 [Service] 段 (Type, User, Group, ExecStart)
- [ ] 配置 User/Group 为非 root (microsegment)
- [ ] 配置 ExecStartPre 验证配置
- [ ] 配置 Restart=on-failure, RestartSec=5s
- [ ] 配置 StartLimitBurst 和 StartLimitInterval
- [ ] 配置环境变量和 EnvironmentFile
- [ ] 配置安全加固选项 (NoNewPrivileges, ProtectSystem 等)
- [ ] 配置文件系统权限 (ReadWritePaths, ReadOnlyPaths)
- [ ] 配置资源限制 (LimitNOFILE, LimitNPROC)
- [ ] 配置日志输出 (StandardOutput=journal, SyslogIdentifier)
- [ ] 配置 [Install] 段 (WantedBy=multi-user.target)
- [ ] 验证: service 文件语法正确 (`systemd-analyze verify`)

### 1.2 创建 Agent Systemd Service
- [ ] 创建 `deploy/systemd/microsegment-agent.service`
- [ ] 配置 [Unit] 段
- [ ] 配置依赖 microsegment-server.service
- [ ] 添加 ConditionPathExists 检查 (包括 /sys/fs/bpf)
- [ ] 配置 [Service] 段
- [ ] 配置 User=root (eBPF 需要)
- [ ] 配置 ExecStartPre 验证配置和 BPF 文件系统
- [ ] 配置 ExecStopPost 清理 eBPF 程序
- [ ] 配置 Restart 策略
- [ ] 配置 AmbientCapabilities (CAP_SYS_ADMIN, CAP_NET_ADMIN, CAP_BPF, CAP_PERFMON, CAP_NET_RAW)
- [ ] 配置 SecureBits=keep-caps
- [ ] 配置最小化安全限制 (ProtectHome, PrivateTmp)
- [ ] 配置资源限制 (LimitNOFILE, LimitMEMLOCK=infinity)
- [ ] 配置日志输出
- [ ] 配置 [Install] 段
- [ ] 验证: service 文件语法正确

## 阶段 2: 配置文件模板 (0.5 天)

### 2.1 创建 Systemd Server 配置
- [ ] 创建 `deploy/config/systemd/server.yaml`
- [ ] 配置 HTTP 服务 (host, port)
- [ ] 配置 gRPC 服务 (host, port)
- [ ] 配置数据库连接
- [ ] 配置日志 (level: info, format: json)
- [ ] 配置文件路径 (使用系统标准路径)
- [ ] 验证: 配置可正常加载

### 2.2 创建 Systemd Agent 配置
- [ ] 创建 `deploy/config/systemd/agent.yaml`
- [ ] 配置 Server 连接 URL
- [ ] 配置 API 端口
- [ ] 配置 eBPF 程序路径
- [ ] 配置日志
- [ ] 验证: 配置可正常加载

### 2.3 创建环境变量模板
- [ ] 创建 `deploy/config/systemd/server.env.example`
- [ ] 定义常用环境变量
- [ ] 添加变量说明注释
- [ ] 创建 `deploy/config/systemd/agent.env.example`
- [ ] 验证: 环境变量可正常加载

## 阶段 3: 安装脚本 (1 天)

### 3.1 创建安装脚本
- [ ] 创建 `deploy/scripts/install-systemd.sh`
- [ ] 实现 root 权限检查
- [ ] 实现前置条件检查 (Systemd, PostgreSQL)
- [ ] 实现创建 microsegment 用户和组
- [ ] 检查用户是否已存在
- [ ] 使用 --system 标志创建系统用户
- [ ] 设置 home 目录和 shell
- [ ] 实现安装二进制文件步骤
- [ ] 使用 install 命令设置正确权限
- [ ] 复制到 /usr/local/bin/
- [ ] 实现安装配置文件步骤
- [ ] 创建 /etc/microsegment/ 目录
- [ ] 复制配置文件并设置权限
- [ ] 实现创建数据目录步骤
- [ ] 创建 /var/lib/microsegment/{server,agent}
- [ ] 设置正确的所有者和权限
- [ ] 实现安装 service 文件步骤
- [ ] 复制到 /etc/systemd/system/
- [ ] 执行 systemctl daemon-reload
- [ ] 实现启用和启动服务步骤
- [ ] systemctl enable 设置开机自启
- [ ] systemctl start 启动服务
- [ ] 实现验证安装步骤
- [ ] 检查服务运行状态
- [ ] 输出日志 (如果失败)
- [ ] 添加错误处理和回滚机制
- [ ] 添加彩色输出和进度提示
- [ ] 验证: 脚本可在干净环境成功安装

### 3.2 创建卸载脚本
- [ ] 创建 `deploy/scripts/uninstall-systemd.sh`
- [ ] 实现 root 权限检查
- [ ] 实现停止服务步骤 (agent → server 顺序)
- [ ] 实现禁用服务步骤
- [ ] 实现删除 service 文件步骤
- [ ] 实现删除二进制文件步骤
- [ ] 实现删除配置文件步骤 (可选,带确认)
- [ ] 实现删除数据目录步骤 (可选,带确认)
- [ ] 实现删除用户和组步骤 (可选,带确认)
- [ ] 执行 systemctl daemon-reload
- [ ] 添加错误处理
- [ ] 验证: 脚本可完全清理安装

### 3.3 创建健康检查脚本
- [ ] 创建 `deploy/scripts/health-check-systemd.sh`
- [ ] 检查 Server 服务状态 (systemctl is-active)
- [ ] 检查 Agent 服务状态
- [ ] 检查 PostgreSQL 服务状态
- [ ] 检查 Server HTTP API (/health)
- [ ] 检查 Server gRPC 端口可访问
- [ ] 检查 Agent API 端口可访问
- [ ] 检查 Agent eBPF 程序加载状态
- [ ] 返回正确的退出码 (0=健康, 1=不健康)
- [ ] 输出详细的检查结果
- [ ] 添加彩色输出
- [ ] 验证: 可准确检测系统健康状态

### 3.4 创建日志查看脚本
- [ ] 创建 `deploy/scripts/logs-systemd.sh`
- [ ] 支持查看 Server 或 Agent 日志
- [ ] 使用 journalctl 命令
- [ ] 支持 --follow 参数实时查看
- [ ] 支持 --lines 参数限制行数
- [ ] 支持 --since 参数时间范围
- [ ] 支持 --priority 参数过滤日志级别
- [ ] 添加使用帮助
- [ ] 验证: 可方便查看日志

## 阶段 4: 文档编写 (0.5 天)

### 4.1 创建 Systemd 部署文档
- [ ] 创建 `deploy/systemd/README.md`
- [ ] 编写前置要求 (Systemd 版本, PostgreSQL)
- [ ] 编写快速开始步骤
- [ ] 使用安装脚本安装
- [ ] 手工安装步骤 (作为备选)
- [ ] 编写服务管理命令
- [ ] 启动、停止、重启、重载
- [ ] 启用和禁用开机自启
- [ ] 编写配置说明
- [ ] 配置文件位置和格式
- [ ] 环境变量使用
- [ ] 配置修改后如何重载
- [ ] 编写日志查看方法
- [ ] journalctl 常用命令
- [ ] 日志级别和过滤
- [ ] 日志轮转配置
- [ ] 编写服务依赖说明
- [ ] PostgreSQL → Server → Agent
- [ ] 依赖失败的处理
- [ ] 编写安全说明
- [ ] Server 非 root 用户运行
- [ ] Agent 权限要求和风险
- [ ] 安全加固选项说明
- [ ] 编写故障排查指南
- [ ] 常见问题及解决方法
- [ ] 调试技巧
- [ ] 编写性能调优建议
- [ ] 资源限制调整
- [ ] 优先级配置

### 4.2 创建配置参考文档
- [ ] 在 deploy/systemd/README.md 中添加配置章节
- [ ] 列出所有配置项及说明
- [ ] 提供配置示例
- [ ] 说明环境变量覆盖机制

### 4.3 更新主 README
- [ ] 在主 README.md 中添加 Systemd 部署章节
- [ ] 链接到详细的 Systemd 部署文档
- [ ] 提供快速开始命令

## 阶段 5: 测试验证 (1 天)

### 5.1 Service 文件测试
- [ ] 在 Ubuntu 22.04 上测试 service 文件
- [ ] 使用 systemd-analyze verify 验证语法
- [ ] 测试 Server service 启动
- [ ] 测试 Agent service 启动
- [ ] 验证服务依赖关系正确
- [ ] 验证 PostgreSQL 停止时 Server 也停止
- [ ] 验证 Server 停止时 Agent 也停止
- [ ] 在 CentOS 8 上测试 service 文件
- [ ] 在 Debian 11 上测试 service 文件

### 5.2 安装脚本测试
- [ ] 在干净的 Ubuntu 22.04 测试安装
- [ ] 验证用户和组正确创建
- [ ] 验证文件权限正确
- [ ] 验证服务成功启动
- [ ] 验证服务开机自启启用
- [ ] 测试安装脚本错误处理
- [ ] PostgreSQL 未安装时
- [ ] 配置文件缺失时
- [ ] 测试卸载脚本
- [ ] 验证完全清理

### 5.3 服务运行时测试
- [ ] 测试 Server 服务自动重启
- [ ] 手动 kill 进程
- [ ] 验证 5 秒后自动重启
- [ ] 测试 Agent 服务自动重启
- [ ] 测试服务崩溃保护 (StartLimit)
- [ ] 快速重启 5 次触发限制
- [ ] 测试配置重载
- [ ] 修改配置文件
- [ ] systemctl reload
- [ ] 验证配置生效
- [ ] 测试系统重启后服务自启
- [ ] 测试日志输出到 journald
- [ ] 验证日志可通过 journalctl 查看

### 5.4 安全测试
- [ ] 验证 Server 以非 root 运行
- [ ] 检查进程用户
- [ ] 验证安全加固选项生效
- [ ] 检查 NoNewPrivileges
- [ ] 检查 ProtectSystem
- [ ] 验证 Agent Capabilities 配置
- [ ] 检查 CAP_SYS_ADMIN 等
- [ ] 测试文件系统权限
- [ ] 验证 /etc/microsegment/ 权限
- [ ] 验证 /var/lib/microsegment/ 权限

### 5.5 健康检查测试
- [ ] 测试 health-check-systemd.sh 脚本
- [ ] 服务正常时返回 0
- [ ] 服务异常时返回 1
- [ ] 测试各项检查逻辑
- [ ] 服务状态检查
- [ ] API 可用性检查

### 5.6 文档验证
- [ ] 让运维人员按文档执行部署
- [ ] 收集反馈并改进文档
- [ ] 确保所有命令可执行
- [ ] 确保所有步骤准确

## 阶段 6: 安全审查和优化 (0.5 天)

### 6.1 安全审查
- [ ] 审查 Server service 安全配置
- [ ] 确保非 root 运行
- [ ] 审查安全加固选项
- [ ] 审查 Agent service 权限配置
- [ ] 确认 Capabilities 最小化
- [ ] 检查不必要的权限
- [ ] 审查文件权限
- [ ] 确保配置文件权限正确 (640/600)
- [ ] 确保敏感信息保护
- [ ] 审查日志输出
- [ ] 确保不泄露敏感信息

### 6.2 性能优化
- [ ] 测试服务启动时间
- [ ] 优化 ExecStartPre 检查
- [ ] 测试资源使用情况
- [ ] 调整资源限制配置
- [ ] 测试服务重启性能
- [ ] 优化重启间隔

### 6.3 兼容性测试
- [ ] 测试不同 Systemd 版本兼容性
- [ ] Systemd 版本检查
- [ ] 测试不同 Linux 内核版本
- [ ] eBPF 特性检查
- [ ] 文档化兼容性要求

## 总计

- **总任务数**: 102 任务
- **预估工时**: 3 天
- **阶段数**: 6 个阶段

## 里程碑

1. **M1 (第 0.5 天)**: Service 文件和配置模板完成
2. **M2 (第 1.5 天)**: 所有脚本完成
3. **M3 (第 2 天)**: 文档完成
4. **M4 (第 3 天)**: 测试、安全审查和优化完成,提案可归档
