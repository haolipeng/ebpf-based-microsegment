# 文档索引

本目录包含项目的用户文档、开发指南和参考资料。

## 📂 目录结构

```
docs/
├── guides/          # 📖 操作指南（入门、部署、故障排查）
├── tutorials/       # 📚 动手教程
├── references/      # 📋 参考文档（API、架构）
├── specs/           # 📝 技术规格（设计方案）
├── research/        # 🔬 研究分析（NeuVector、ZFW、Cilium）
├── learning/        # 📖 学习资料
├── contributing/    # 🤝 贡献指南
└── archive/         # 📦 归档文档
```

---

## 🚀 快速开始

| 文档 | 说明 |
|------|------|
| [项目 README](../README.md) | 项目简介和快速开始 |
| [构建指南](guides/getting-started/build.md) | 详细的编译和运行说明 |
| [部署指南](guides/deployment/README.md) | 生产环境部署 |
| [运维手册](guides/operations/README.md) | 日常运维操作指南 |
| [E2E 测试](guides/getting-started/e2e-testing.md) | 端到端测试框架 |
| [故障排查](guides/troubleshooting/README.md) | 常见问题解决 |

---

## 📖 学习路线

### 6周学习指南
从零到精通的 eBPF 微隔离学习路径：

| 周次 | 主题 | 链接 |
|------|------|------|
| Week 1 | 环境和基础 | [week1](learning/weekly-guide/week1-environment-and-basics.md) |
| Week 2 | 基础框架 | [week2](learning/weekly-guide/week2-basic-framework.md) |
| Week 3 | 用户态控制 | [week3](learning/weekly-guide/week3-userspace-control.md) |
| Week 4 | 高级功能 | [week4](learning/weekly-guide/week4-advanced-features.md) |
| Week 5 | 测试优化 | [week5](learning/weekly-guide/week5-testing-optimization.md) |
| Week 6 | 生产部署 | [week6](learning/weekly-guide/week6-production-deployment.md) |

- [eBPF 知识点](learning/ebpf-knowledge.md) - eBPF 核心概念
- [前端学习计划](learning/frontend-learning-plan-3weeks.md) - 3周前端开发计划

---

## 📋 参考文档

### API 和配置
| 文档 | 说明 |
|------|------|
| [REST API 参考](references/api/rest-api.md) | 完整 REST API 文档 |
| [配置参考](references/config.md) | Agent/Server 配置详解 |
| [CLI 参考](references/cli/README.md) | 命令行使用指南 |
| [Label-based Policy API](references/api/label-based-policies.md) | 策略 API 文档 |

### 架构和性能
| 文档 | 说明 |
|------|------|
| [架构概述](references/architecture/overview.md) | 系统架构设计 |
| [前端架构](references/architecture/frontend.md) | Web UI 架构 |
| [性能参考](references/performance.md) | 性能指标 |
| [性能优化指南](references/performance/) | 详细优化文档 |

---

## 📝 技术规格

### 网络功能
| 文档 | 说明 |
|------|------|
| [NAT 支持](specs/networking/nat-support.md) | NAT 实现方案 |
| [IP 分片处理](specs/networking/fragment-handling.md) | 分片处理进度 |
| [TCP 重组](specs/networking/tcp-reassembly-solutions.md) | TCP 重组方案 |
| [应用层检测](specs/networking/app-layer-detection.md) | L7 协议检测 |

### 策略引擎
| 文档 | 说明 |
|------|------|
| [策略引擎分析](specs/policies/policy-engine-analysis.md) | 引擎架构分析 |
| [通配符优化](specs/policies/wildcard-optimization.md) | 通配符策略优化 |
| [会话超时](specs/policies/session-timeout.md) | 会话管理设计 |

### 其他
| 文档 | 说明 |
|------|------|
| [路线图](specs/roadmap.md) | 项目发展规划 |
| [进程监控](specs/features/process-monitoring.md) | 进程感知设计 |
| [标签获取](specs/features/label-acquisition.md) | 自动标签获取 |

---

## 🔬 研究分析

### NeuVector 分析
**完整的 NeuVector 技术分析文档集** → [research/neuvector/](research/neuvector/)

| 文档 | 说明 | 推荐度 |
|------|------|--------|
| [dp 编译指南](research/neuvector/neuvector-dp-build-guide.md) | 数据平面编译 | ⭐⭐⭐ |
| [Agent-dp 通信](research/neuvector/neuvector-dp-agent-communication.md) | 通信机制 | ⭐⭐⭐⭐⭐ |
| [策略分发流程](research/neuvector/neuvector-agent-dp-policy-flow.md) | 策略同步 | ⭐⭐⭐⭐⭐ |
| [FQDN 实现](research/neuvector/neuvector-fqdn-implementation.md) | 域名过滤 | ⭐⭐⭐⭐ |
| [网络拓扑](research/neuvector/neuvector-network-topology-implementation.md) | 拓扑实现 | ⭐⭐⭐⭐⭐ |
| [诊断工具](research/neuvector/dp-diagnostic-tool.md) | DP 诊断 | ⭐⭐⭐ |

### ZFW 分析
**完整的 ZFW 技术分析文档集** → [research/zfw/](research/zfw/)

| 文档 | 说明 | 推荐度 |
|------|------|--------|
| [架构分析](research/zfw/zfw-architecture-analysis.md) | eBPF 架构 | ⭐⭐⭐⭐⭐ |
| [技术图表](research/zfw/zfw-technical-diagrams.md) | 关键图表 | ⭐⭐⭐⭐⭐ |
| [深度剖析](research/zfw/zfw-deep-dive.md) | 高级功能 | ⭐⭐⭐⭐⭐ |
| [快速参考](research/zfw/zfw-quick-reference.md) | 速查手册 | ⭐⭐⭐⭐ |

### Cilium 研究
- [Cilium Identity 机制](research/cilium/) - 身份认证研究

---

## 🤝 贡献指南

| 文档 | 说明 |
|------|------|
| [贡献入门](contributing/index.md) | 如何参与项目 |
| [图表指南](contributing/diagrams-guide.md) | 如何绘制项目图表 |
| [代码搜索](contributing/code-search/) | 代码搜索工具 |

---

## 📦 归档文档

历史任务总结和已完成工作记录 → [archive/](archive/)

---

*最后更新：2024-12-24*
