# 贡献指南

感谢您对 eBPF 微隔离系统的关注！我们欢迎各种形式的贡献。

## 如何贡献

### 报告问题

- [报告 Bug](https://github.com/your-org/ebpf-based-microsegment/issues/new?template=bug_report.md)
- [请求新功能](https://github.com/your-org/ebpf-based-microsegment/issues/new?template=feature_request.md)
- [文档问题](https://github.com/your-org/ebpf-based-microsegment/issues/new?template=docs_issue.md)

### 贡献代码

1. Fork 仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 开发文档

### [开发环境搭建](development-setup.md)

设置本地开发环境：

- 安装依赖
- 克隆仓库
- 编译项目
- 运行测试

### [代码规范](code-style.md)

代码风格和规范：

- Go 代码规范
- C (eBPF) 代码规范
- 注释规范
- 命名约定

### [提交指南](pull-requests.md)

提交 PR 的最佳实践：

- Commit 消息格式
- PR 描述模板
- 代码审查流程
- CI/CD 流程

### [测试指南](testing.md)

编写和运行测试：

- 单元测试
- 集成测试
- E2E 测试
- 性能测试

### [发布流程](release.md)

版本发布流程：

- 版本号规范
- 发布检查清单
- 变更日志

## 项目结构

```
ebpf-based-microsegment/
├── src/
│   ├── agent/           # Go Agent 代码
│   │   ├── cmd/         # 命令行入口
│   │   └── pkg/         # 核心包
│   ├── bpf/             # eBPF C 代码
│   └── server/          # 控制平面服务器
├── docs/                # 文档
├── deploy/              # 部署配置
├── tests/               # 测试代码
└── scripts/             # 构建脚本
```

## 行为准则

请阅读我们的 [行为准则](code-of-conduct.md)，确保营造友好、包容的社区环境。

## 获取帮助

- GitHub Discussions
- Slack 频道
- 邮件列表
