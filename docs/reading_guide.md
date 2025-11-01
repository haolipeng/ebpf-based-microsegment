# 📖 项目文档阅读指南

> 帮助你快速熟悉代码和项目架构的最佳阅读路径

---

## 🎯 根据你的目标选择路径

### 路径 1: 快速了解项目（30 分钟）⚡
**适合**：第一次接触项目，想快速了解"这是什么"

1. **[README_CN.md](../README_CN.md)** (5 分钟)
   - 项目背景和功能需求
   - 快速了解要做什么

2. **[README.md](../README.md)** (5 分钟)
   - 项目特性和架构图
   - Quick Start 命令

3. **[docs/microsegmentation-mvp-implementation-plan.md](microsegmentation-mvp-implementation-plan.md)** (10 分钟)
   - MVP 8 周实施计划
   - 里程碑和优先级

4. **[IMPLEMENTATION_SUMMARY.md](../IMPLEMENTATION_SUMMARY.md)** (10 分钟)
   - 当前进度和已完成功能
   - 下一步计划

---

### 路径 2: 理解系统架构（1-2 小时）🏗️
**适合**：需要了解技术架构和设计决策

#### 第一步：总体架构（30 分钟）

1. **[docs/ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md)** ⭐⭐⭐⭐⭐
   - **34KB，最重要的架构文档**
   - 数据平面架构（eBPF + TC）
   - 控制平面架构（Agent）
   - 组件交互流程
   - 数据结构设计

#### 第二步：深入数据平面（30 分钟）

2. **[design-docs/architecture/tc-mode-microsegmentation.md](../design-docs/architecture/tc-mode-microsegmentation.md)**
   - TC 模式技术细节
   - eBPF 程序设计
   - Hook 点选择

3. **[docs/PERFORMANCE.md](PERFORMANCE.md)**
   - 性能指标和优化
   - 热路径分析
   - 性能测试结果

#### 第三步：理解控制流程（30 分钟）

4. **[design-docs/architecture/design.md](../design-docs/architecture/design.md)**
   - 策略下发流程
   - 会话跟踪机制
   - 统计上报机制

---

### 路径 3: 学习参考实现（2-3 小时）📚
**适合**：想学习业界最佳实践，了解 NeuVector 实现

#### NeuVector 学习路径（推荐顺序）

1. **[docs/neuvector-dp-agent-communication.md](neuvector-dp-agent-communication.md)** (30 分钟) ⭐⭐⭐⭐⭐
   - **40KB，必读**
   - Agent 和 dp 如何通信
   - Unix Socket + JSON/Binary 协议
   - 30+ 控制命令详解
   - 完整代码示例

2. **[docs/neuvector-agent-dp-policy-flow.md](neuvector-agent-dp-policy-flow.md)** (40 分钟) ⭐⭐⭐⭐⭐
   - **46KB，必读**
   - Controller → Agent → dp → Kernel 完整流程
   - 策略如何下发和执行
   - 数据结构转换
   - 4 个完整场景示例

3. **[docs/neuvector-fqdn-implementation.md](neuvector-fqdn-implementation.md)** (30 分钟) ⭐⭐⭐⭐
   - **31KB**
   - FQDN 域名过滤实现
   - DNS 拦截机制
   - 双向映射表设计

4. **[docs/neuvector-dp-build-guide.md](neuvector-dp-build-guide.md)** (20 分钟) ⭐⭐⭐
   - **12KB**
   - dp 组件编译指南
   - 依赖库解析
   - 如何编译和运行

5. **[docs/dp-diagnostic-tool.md](dp-diagnostic-tool.md)** (20 分钟) ⭐⭐⭐
   - **15KB**
   - diag.py 诊断工具
   - 如何手动测试 dp
   - 模拟策略下发

---

### 路径 4: 深入代码实现（持续学习）💻
**适合**：准备开始编码或修改代码

#### 阶段 1: 环境搭建（1 天）

1. **[docs/BUILD_GUIDE.md](BUILD_GUIDE.md)** (30 分钟)
   - 完整的编译步骤
   - 依赖安装
   - 常见问题

2. **[docs/TROUBLESHOOTING.md](TROUBLESHOOTING.md)** (30 分钟)
   - 故障排查指南
   - 常见错误和解决方案

3. **实际操作**：编译和运行项目

#### 阶段 2: 跟着周计划学习（6 周）

**[docs/weekly-guide/](weekly-guide/)** - 6 周完整学习路径

按顺序阅读：

1. **[week1-environment-and-basics.md](weekly-guide/week1-environment-and-basics.md)**
   - eBPF 基础概念
   - TC Hook 原理
   - 开发环境配置

2. **[week2-basic-framework.md](weekly-guide/week2-basic-framework.md)**
   - 数据平面框架
   - 会话跟踪实现
   - 策略匹配引擎

3. **[week3-userspace-control.md](weekly-guide/week3-userspace-control.md)**
   - 用户态 Agent 开发
   - eBPF Map 交互
   - Ring Buffer 事件处理

4. **[week4-advanced-features.md](weekly-guide/week4-advanced-features.md)**
   - FQDN 过滤
   - 统计聚合
   - 高级策略

5. **[week5-testing-optimization.md](weekly-guide/week5-testing-optimization.md)**
   - 单元测试
   - 性能测试
   - 优化技巧

6. **[week6-production-deployment.md](weekly-guide/week6-production-deployment.md)**
   - 生产部署
   - 监控告警
   - 运维管理

#### 阶段 3: 代码阅读顺序

**eBPF 数据平面代码**（按依赖关系）：

```
1. src/ebpf/headers/          # 头文件和数据结构定义
   ├── vmlinux.h              # 内核类型定义
   ├── types.h                # 自定义类型
   └── maps.h                 # Map 定义

2. src/ebpf/session.h         # 会话跟踪数据结构
   src/ebpf/session.c         # 会话跟踪逻辑

3. src/ebpf/policy.h          # 策略匹配数据结构
   src/ebpf/policy.c          # 策略匹配逻辑

4. src/ebpf/stats.h           # 统计数据结构
   src/ebpf/stats.c           # 统计收集逻辑

5. src/ebpf/microsegment.c    # 主入口（tc_ingress/egress）
```

**Go Agent 代码**（按功能模块）：

```
1. cmd/agent/main.go          # 程序入口

2. pkg/ebpf/
   ├── loader.go              # eBPF 程序加载
   ├── maps.go                # Map 操作封装
   └── events.go              # Ring Buffer 事件处理

3. pkg/policy/
   ├── manager.go             # 策略管理
   ├── types.go               # 策略数据结构
   └── matcher.go             # 策略匹配逻辑

4. pkg/session/
   ├── tracker.go             # 会话跟踪
   └── cache.go               # 会话缓存

5. pkg/stats/
   ├── collector.go           # 统计收集
   └── aggregator.go          # 统计聚合
```

---

### 路径 5: 参与开发流程（开发者必读）🔧
**适合**：准备提交代码或参与协作

1. **[docs/openspec-learning-guide.md](openspec-learning-guide.md)** (1 小时) ⭐⭐⭐⭐⭐
   - **46KB，开发流程必读**
   - OpenSpec 规范化开发
   - Proposal → Apply → Archive
   - 团队协作最佳实践

2. **[docs/OpenSpec-Workflow-Guide.md](OpenSpec-Workflow-Guide.md)** (30 分钟)
   - OpenSpec 快速参考
   - 常用命令和操作

3. **[docs/GO_DOC_STYLE_GUIDE.md](GO_DOC_STYLE_GUIDE.md)** (15 分钟)
   - Go 代码注释规范
   - 文档生成指南

4. **[docs/project-diagrams-guide.md](project-diagrams-guide.md)** (可选，30 分钟)
   - 如何绘制项目图表
   - 图表类型和工具

---

## 📊 文档重要性分级

### ⭐⭐⭐⭐⭐ 必读（优先级最高）

| 文档 | 大小 | 用途 | 阅读时间 |
|------|------|------|---------|
| [ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md) | 34KB | 系统架构总览 | 30 分钟 |
| [neuvector-dp-agent-communication.md](neuvector-dp-agent-communication.md) | 40KB | Agent-dp 通信机制 | 30 分钟 |
| [neuvector-agent-dp-policy-flow.md](neuvector-agent-dp-policy-flow.md) | 46KB | 策略完整流程 | 40 分钟 |
| [openspec-learning-guide.md](openspec-learning-guide.md) | 46KB | 开发流程规范 | 1 小时 |

### ⭐⭐⭐⭐ 重要（核心知识）

| 文档 | 大小 | 用途 | 阅读时间 |
|------|------|------|---------|
| [PERFORMANCE.md](PERFORMANCE.md) | 8.8KB | 性能指标和优化 | 20 分钟 |
| [neuvector-fqdn-implementation.md](neuvector-fqdn-implementation.md) | 31KB | FQDN 实现细节 | 30 分钟 |
| [BUILD_GUIDE.md](BUILD_GUIDE.md) | 5.7KB | 编译构建指南 | 20 分钟 |
| [weekly-guide/](weekly-guide/) | - | 6 周学习路径 | 6 周 |

### ⭐⭐⭐ 有用（参考资料）

| 文档 | 大小 | 用途 | 阅读时间 |
|------|------|------|---------|
| [neuvector-dp-build-guide.md](neuvector-dp-build-guide.md) | 12KB | NeuVector dp 编译 | 20 分钟 |
| [dp-diagnostic-tool.md](dp-diagnostic-tool.md) | 15KB | 诊断工具使用 | 20 分钟 |
| [TROUBLESHOOTING.md](TROUBLESHOOTING.md) | 15KB | 故障排查 | 按需查阅 |
| [zfw-architecture-analysis.md](zfw-architecture-analysis.md) | 33KB | ZFW 项目分析 | 30 分钟 |

### ⭐⭐ 可选（特定场景）

| 文档 | 大小 | 用途 | 阅读时间 |
|------|------|------|---------|
| [frontend-learning-plan-3weeks.md](frontend-learning-plan-3weeks.md) | 45KB | 前端开发计划 | 需要时阅读 |
| [project-diagrams-guide.md](project-diagrams-guide.md) | 95KB | 图表绘制指南 | 需要时阅读 |
| [OPTIMIZATION_SUMMARY.md](OPTIMIZATION_SUMMARY.md) | 7.1KB | 优化总结 | 需要时阅读 |

---

## 🎓 推荐学习路径（完整版）

### 第 1 天：快速入门
```
上午：
□ README_CN.md
□ README.md
□ microsegmentation-mvp-implementation-plan.md
□ IMPLEMENTATION_SUMMARY.md

下午：
□ ARCHITECTURE_OVERVIEW.md（重点阅读）
□ BUILD_GUIDE.md
□ 实际编译和运行项目
```

### 第 2-3 天：理解架构
```
第 2 天：
□ design-docs/architecture/tc-mode-microsegmentation.md
□ PERFORMANCE.md
□ neuvector-dp-agent-communication.md

第 3 天：
□ neuvector-agent-dp-policy-flow.md
□ neuvector-fqdn-implementation.md
□ dp-diagnostic-tool.md
```

### 第 4-5 天：代码阅读
```
第 4 天（eBPF 数据平面）：
□ 阅读 src/ebpf/headers/
□ 阅读 src/ebpf/session.h 和 session.c
□ 阅读 src/ebpf/policy.h 和 policy.c
□ 阅读 src/ebpf/microsegment.c

第 5 天（Go Agent）：
□ 阅读 cmd/agent/main.go
□ 阅读 pkg/ebpf/
□ 阅读 pkg/policy/
□ 阅读 pkg/session/
```

### 第 2-7 周：跟着周计划深入学习
```
□ Week 1: eBPF 基础和 TC Hook
□ Week 2: 会话跟踪和策略匹配
□ Week 3: 用户态 Agent 开发
□ Week 4: FQDN 和高级功能
□ Week 5: 测试和性能优化
□ Week 6: 生产部署和运维
```

### 持续学习：开发规范
```
□ openspec-learning-guide.md
□ OpenSpec-Workflow-Guide.md
□ GO_DOC_STYLE_GUIDE.md
```

---

## 💡 阅读建议

### 1. 先广后深
- ✅ 第一遍：快速浏览，了解大概
- ✅ 第二遍：重点阅读，标记不懂的地方
- ✅ 第三遍：结合代码，深入理解

### 2. 边读边实践
- ✅ 读完 BUILD_GUIDE 就编译运行
- ✅ 读完 ARCHITECTURE_OVERVIEW 就画架构图
- ✅ 读完 neuvector-agent-dp-policy-flow 就跟踪代码

### 3. 做笔记和总结
- ✅ 用自己的话总结核心概念
- ✅ 画出数据流程图
- ✅ 记录不懂的问题，逐个解决

### 4. 问题驱动
- ✅ 带着问题阅读：这个模块解决什么问题？
- ✅ 思考设计：为什么这样设计？有没有更好的方案？
- ✅ 验证理解：能否用一句话解释清楚？

### 5. 从整体到局部
```
项目整体架构
    ↓
各个模块的职责
    ↓
模块之间的交互
    ↓
具体的实现细节
    ↓
代码级别的理解
```

---

## 🔍 快速查找指南

### 我想了解...

| 问题 | 推荐文档 |
|------|---------|
| 项目是做什么的？ | README_CN.md, README.md |
| 系统架构是什么样的？ | ARCHITECTURE_OVERVIEW.md |
| 如何编译运行？ | BUILD_GUIDE.md |
| 当前进度如何？ | IMPLEMENTATION_SUMMARY.md |
| 性能指标是多少？ | PERFORMANCE.md |
| NeuVector 是如何实现的？ | neuvector-agent-dp-policy-flow.md |
| Agent 和 dp 如何通信？ | neuvector-dp-agent-communication.md |
| FQDN 功能如何实现？ | neuvector-fqdn-implementation.md |
| 如何参与开发？ | openspec-learning-guide.md |
| 遇到问题怎么办？ | TROUBLESHOOTING.md |
| 如何学习 eBPF？ | weekly-guide/week1-environment-and-basics.md |
| 如何优化性能？ | PERFORMANCE.md, OPTIMIZATION_SUMMARY.md |
| 如何绘制图表？ | project-diagrams-guide.md |

---

## 📝 阅读进度追踪

你可以复制下面的清单，追踪自己的阅读进度：

```markdown
## 我的阅读进度

### 快速入门（第 1 天）
- [ ] README_CN.md
- [ ] README.md
- [ ] microsegmentation-mvp-implementation-plan.md
- [ ] IMPLEMENTATION_SUMMARY.md
- [ ] ARCHITECTURE_OVERVIEW.md ⭐
- [ ] BUILD_GUIDE.md
- [ ] 实际编译运行项目

### 架构理解（第 2-3 天）
- [ ] design-docs/architecture/tc-mode-microsegmentation.md
- [ ] PERFORMANCE.md
- [ ] neuvector-dp-agent-communication.md ⭐
- [ ] neuvector-agent-dp-policy-flow.md ⭐
- [ ] neuvector-fqdn-implementation.md
- [ ] dp-diagnostic-tool.md

### 代码阅读（第 4-5 天）
- [ ] eBPF 头文件和数据结构
- [ ] eBPF 会话跟踪代码
- [ ] eBPF 策略匹配代码
- [ ] eBPF 主程序代码
- [ ] Go Agent 入口代码
- [ ] Go Agent 各模块代码

### 周计划学习（第 2-7 周）
- [ ] Week 1: 环境和基础
- [ ] Week 2: 基础框架
- [ ] Week 3: 用户态控制
- [ ] Week 4: 高级功能
- [ ] Week 5: 测试优化
- [ ] Week 6: 生产部署

### 开发规范（持续）
- [ ] openspec-learning-guide.md ⭐
- [ ] OpenSpec-Workflow-Guide.md
- [ ] GO_DOC_STYLE_GUIDE.md
```

---

## 🎯 根据角色推荐

### 如果你是项目新人
**推荐路径**：快速了解 → 架构理解 → 周计划学习

**核心文档**：
1. README_CN.md
2. ARCHITECTURE_OVERVIEW.md
3. weekly-guide/ (6 周)

### 如果你是架构师
**推荐路径**：架构理解 → 参考实现 → 设计文档

**核心文档**：
1. ARCHITECTURE_OVERVIEW.md
2. neuvector-agent-dp-policy-flow.md
3. design-docs/architecture/

### 如果你是开发者
**推荐路径**：快速入门 → 代码阅读 → 开发规范

**核心文档**：
1. BUILD_GUIDE.md
2. ARCHITECTURE_OVERVIEW.md
3. openspec-learning-guide.md
4. 代码 + 周计划

### 如果你是测试工程师
**推荐路径**：快速入门 → 测试文档 → 故障排查

**核心文档**：
1. README.md
2. BUILD_GUIDE.md
3. weekly-guide/week5-testing-optimization.md
4. TROUBLESHOOTING.md

---

## 📌 总结：最重要的 5 个文档

如果时间有限，只读这 5 个文档：

1. **[ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md)** (34KB, 30 分钟)
   - 理解系统架构

2. **[neuvector-agent-dp-policy-flow.md](neuvector-agent-dp-policy-flow.md)** (46KB, 40 分钟)
   - 理解策略流程

3. **[neuvector-dp-agent-communication.md](neuvector-dp-agent-communication.md)** (40KB, 30 分钟)
   - 理解通信机制

4. **[BUILD_GUIDE.md](BUILD_GUIDE.md)** (5.7KB, 20 分钟)
   - 上手编译运行

5. **[openspec-learning-guide.md](openspec-learning-guide.md)** (46KB, 1 小时)
   - 参与团队开发

**总计阅读时间：3 小时**

读完这 5 个文档，你就能：
- ✅ 理解系统架构和设计
- ✅ 看懂代码的核心逻辑
- ✅ 编译运行项目
- ✅ 参与团队开发

---

**最后更新**: 2025-10-31
**维护者**: eBPF 微隔离项目组

---

**祝你学习顺利！如有问题，请查阅 [TROUBLESHOOTING.md](TROUBLESHOOTING.md) 或联系团队成员。**
