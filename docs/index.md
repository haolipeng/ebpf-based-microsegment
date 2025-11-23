# eBPF 微隔离系统文档

欢迎使用 eBPF 微隔离系统文档！本文档将帮助您了解、部署和使用基于 eBPF 的容器网络微隔离解决方案。

## 文档导航

### [入门指南](getting-started/index.md)

快速开始使用 eBPF 微隔离系统，包括系统要求、安装步骤和基本概念介绍。

- [概述](getting-started/overview.md) - 系统介绍和核心特性
- [快速开始](getting-started/quickstart.md) - 5 分钟快速体验
- [核心概念](getting-started/concepts.md) - 理解关键概念

### [教程](tutorials/index.md)

通过实践学习如何使用各项功能，包括基础操作和高级场景。

**基础教程**:
- [创建第一个策略](tutorials/basic/first-policy.md)
- [流量监控入门](tutorials/basic/flow-monitoring.md)
- [会话追踪](tutorials/basic/session-tracking.md)

**高级教程**:
- [Kubernetes 集成](tutorials/advanced/kubernetes-integration.md)
- [自定义策略规则](tutorials/advanced/custom-policies.md)
- [性能调优](tutorials/advanced/performance-tuning.md)

### [参考文档](docs/index.md)

完整的功能参考文档，涵盖所有配置选项和 API。

- [安装指南](docs/install/index.md) - Docker、Kubernetes、二进制安装
- [架构设计](docs/architecture/index.md) - 系统架构和数据流
- [事件系统](docs/events/index.md) - 事件类型和处理
- [策略系统](docs/policies/index.md) - 策略定义和管理
- [CLI 参数](docs/cli-flags/index.md) - 命令行参数参考
- [数据源](docs/data-sources/index.md) - 内置和自定义数据源
- [集成](docs/integrations/index.md) - 第三方系统集成

### [贡献指南](contributing/index.md)

了解如何参与项目开发和贡献代码。

- [开发环境搭建](contributing/development-setup.md)
- [代码规范](contributing/code-style.md)
- [提交指南](contributing/pull-requests.md)

---

## 快速链接

| 资源 | 链接 |
|------|------|
| GitHub 仓库 | [ebpf-based-microsegment](https://github.com/your-org/ebpf-based-microsegment) |
| 问题追踪 | [Issues](https://github.com/your-org/ebpf-based-microsegment/issues) |
| 构建状态 | ![Build](https://img.shields.io/badge/build-passing-brightgreen) |

## 系统要求

- **Linux 内核**: 5.4+ (推荐 5.10+)
- **架构**: x86_64, arm64
- **依赖**: clang, llvm, libbpf

## 许可证

本项目采用 Apache 2.0 许可证。
