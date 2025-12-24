# 文档索引

本目录包含项目的用户文档、开发指南和参考资料。

## 📚 文档导航

### 🚀 快速开始
- [项目 README](../README.md) - 项目简介和快速开始
- [构建指南](build_guide.md) - 详细的编译和运行说明
- [实施总结](../demos/IMPLEMENTATION_SUMMARY.md) - 当前进度和已完成工作

### 📖 学习路线
- [6周学习指南](learning/weekly-guide/) - 从零到精通的学习路径
  - [Week 1: 环境和基础](learning/weekly-guide/week1-environment-and-basics.md)
  - [Week 2: 基础框架](learning/weekly-guide/week2-basic-framework.md)
  - [Week 3: 用户态控制](learning/weekly-guide/week3-userspace-control.md)
  - [Week 4: 高级功能](learning/weekly-guide/week4-advanced-features.md)
  - [Week 5: 测试优化](learning/weekly-guide/week5-testing-optimization.md)
  - [Week 6: 生产部署](learning/weekly-guide/week6-production-deployment.md)

### 📋 项目规划
- [MVP 实施计划](microsegmentation-mvp-implementation-plan.md) - 8周 MVP 开发计划
- [前端学习计划](learning/frontend-learning-plan-3weeks.md) - 3周前端开发计划

### 🔧 操作指南
- [项目图表指南](project-diagrams-guide.md) - 如何绘制项目图表
- [DP 诊断工具](neuvector-analysis/dp-diagnostic-tool.md) - 数据平面诊断工具使用

### 📚 参考资料

#### NeuVector 源码分析
**完整的 NeuVector 技术分析文档集** → [neuvector-analysis/](neuvector-analysis/)

5 个深度分析文档（156KB，7800+ 行）:
- [dp 组件编译指南](neuvector-analysis/neuvector-dp-build-guide.md) (12KB)
- [Agent-dp 通信机制](neuvector-analysis/neuvector-dp-agent-communication.md) (40KB) ⭐⭐⭐⭐⭐
- [Agent-dp 策略分发流程](neuvector-analysis/neuvector-agent-dp-policy-flow.md) (46KB) ⭐⭐⭐⭐⭐
- [FQDN 域名过滤实现](neuvector-analysis/neuvector-fqdn-implementation.md) (31KB) ⭐⭐⭐⭐
- [网络拓扑图实现](neuvector-analysis/neuvector-network-topology-implementation.md) (41KB) ⭐⭐⭐⭐⭐

查看 [NeuVector 分析目录 README](neuvector-analysis/README.md) 了解详细导航和学习路径

#### ZFW (Zero Trust Firewall) 源码分析
**完整的 ZFW 技术分析文档集** → [zfw-analysis/](zfw-analysis/)

4 个深度分析文档（141.4KB，4400+ 行）:
- [ZFW eBPF 架构深度分析](zfw-analysis/zfw-architecture-analysis.md) (64KB) ⭐⭐⭐⭐⭐
- [ZFW 关键技术图表集](zfw-analysis/zfw-technical-diagrams.md) (34KB) ⭐⭐⭐⭐⭐ 🆕
- [ZFW 高级功能深度剖析](zfw-analysis/zfw-deep-dive.md) (35KB) ⭐⭐⭐⭐⭐ 🆕
- [ZFW 快速参考手册](zfw-analysis/zfw-quick-reference.md) (7.4KB) ⭐⭐⭐⭐

查看 [ZFW 分析目录 README](zfw-analysis/README.md) 了解详细导航和学习路径

### 🏗️ 设计文档
设计文档和架构决策记录位于 [`design-docs/`](../design-docs/) 目录：
- [架构设计](../design-docs/architecture/) - 系统架构和技术选型
- [可行性分析](../design-docs/analysis/) - 技术方案评估
- [实施细节](../design-docs/implementation/) - 详细实施指南

## 📂 文档组织原则

### docs/ - 用户和开发者文档
- **用户指南**：如何使用、配置、部署
- **开发指南**：如何构建、测试、贡献
- **学习资料**：教程、示例、最佳实践
- **参考资料**：其他项目分析、技术调研

### design-docs/ - 设计和决策文档
- **架构设计**：系统设计、组件交互
- **技术决策**：为什么这样设计、替代方案
- **实施计划**：详细的技术实现方案
- **变更记录**：设计演进历史

## 🤝 如何贡献文档

1. **用户文档**：放在 `docs/`
   - 面向使用者
   - 注重实用性和可操作性
   
2. **设计文档**：放在 `design-docs/`
   - 面向架构师和核心开发者
   - 注重技术深度和决策依据

3. **文档格式**：
   - 使用 Markdown 格式
   - 包含清晰的标题层次
   - 添加代码示例和图表
   - 保持文档更新

## 📝 文档模板

### 用户指南模板
```markdown
# 标题

## 概述
简要说明

## 前置要求
- 列出依赖

## 步骤
1. 第一步
2. 第二步

## 常见问题
### 问题1
解决方案

## 参考
- 相关链接
```

### 设计文档模板
参考 [ADR 模板](../design-docs/README.md)

---

*最后更新：2025-10-31*

