# 设计文档

本目录包含项目的架构设计、技术决策和实施细节。

## 📐 目录结构

```
design-docs/
├── architecture/          # 架构设计文档
├── analysis/             # 技术分析和可行性评估
├── implementation/       # 详细实施方案
├── CHANGES.md           # 设计变更记录
├── REVIEW_REPORT.md     # 设计评审报告
└── README.md            # 本文档
```

## 📚 文档索引

### 🏗️ 架构设计

#### [design.md](architecture/design.md)
**核心架构设计文档**
- DP (Data Plane) 微隔离系统架构
- 多线程模型和数据流
- 部署模式（TAP/TC/NFQ/ProxyMesh）
- 关键数据结构和算法

#### [ebpf-tc-architecture.md](architecture/ebpf-tc-architecture.md)
**eBPF + TC 技术架构**
- TC Hook 机制
- eBPF 程序设计
- Map 类型选择

#### [tc-mode-microsegmentation.md](architecture/tc-mode-microsegmentation.md)
**TC 模式微隔离**
- TC 流量控制原理
- 策略执行机制

### 🔍 技术分析

#### [ebpf-tc-feasibility-index.md](analysis/ebpf-tc-feasibility-index.md)
**eBPF + TC 可行性索引**
- 技术可行性评估
- 性能指标预测
- 风险评估

#### [ebpf-tc-comparison.md](analysis/ebpf-tc-comparison.md)
**eBPF + TC 对比分析**
- 与其他方案对比（XDP, iptables, nftables）
- 优劣势分析
- 适用场景

#### [ebpf-tc-risks.md](analysis/ebpf-tc-risks.md)
**eBPF + TC 风险分析**
- 技术风险
- 性能风险
- 兼容性风险
- 缓解策略

### 📝 实施方案

#### [ebpf-tc-implementation.md](implementation/ebpf-tc-implementation.md)
**详细实施指南（7187 行）**
- 6周实施计划
- 每日任务清单
- 代码示例和最佳实践
- 常见问题和解决方案

## 📋 变更管理

### [CHANGES.md](CHANGES.md)
设计变更记录，包括：
- 架构演进历史
- 重大技术决策
- 变更原因和影响

### [REVIEW_REPORT.md](REVIEW_REPORT.md)
设计评审报告，包括：
- 评审会议记录
- 发现的问题
- 改进建议
- 行动项

## 🎯 设计原则

### 1. 高性能优先
- 目标：<10μs 延迟
- 使用 eBPF 在内核级别处理数据包
- Per-CPU Map 无锁设计
- LRU 自动淘汰避免内存泄漏

### 2. 简单可靠
- 优先选择成熟技术
- 避免过度设计
- 渐进式架构演进

### 3. 云原生适配
- 容器友好
- Kubernetes 集成
- 动态配置更新

### 4. 可观测性
- 完整的统计指标
- 流量可视化
- 事件日志

## 🔄 设计演进历史

### v0.1 - 初始设计（2025-10）
- 基于 eBPF + TC 的数据平面
- LRU_HASH 会话跟踪
- 5元组策略匹配
- Go + Cilium eBPF 用户态

### v0.2 - 控制平面（计划中）
- RESTful API 服务
- 策略管理
- gRPC 通信

### v0.3 - 标签系统（计划中）
- 自动打标签
- 标签驱动策略

### v0.4 - 可视化（计划中）
- 应用依赖图
- 流量拓扑

## 📖 ADR (Architecture Decision Records)

我们使用 ADR 记录重要的架构决策：

### ADR-001: 选择 Cilium eBPF 而非 libbpf C
**日期**: 2025-10-30

**背景**:
项目初期使用 libbpf C 库，但遇到以下问题：
- 内存管理复杂
- 错误处理繁琐
- 与 Go 生态集成困难

**决策**:
迁移到 Cilium eBPF Go 库

**理由**:
- 纯 Go 实现，类型安全
- 更好的错误处理
- 与 Go 生态无缝集成
- 社区活跃，文档完善

**后果**:
- ✅ 开发效率提升
- ✅ 代码可维护性提高
- ✅ 单一二进制部署
- ⚠️ 需要重写用户态代码

### ADR-002: 使用 LRU_HASH 而非 HASH
**日期**: 2025-10-30

**背景**:
需要会话跟踪，但 Map 容量有限

**决策**:
使用 LRU_HASH Map

**理由**:
- 自动淘汰最少使用的会话
- 无需手动清理过期会话
- 防止内存耗尽

**后果**:
- ✅ 简化代码
- ✅ 防止内存泄漏
- ⚠️ 长连接可能被意外淘汰（可通过增大 max_entries 缓解）

## 🤝 如何贡献设计文档

### 新增设计文档
1. 确定文档类型（架构/分析/实施）
2. 放在对应目录
3. 更新本 README.md 索引
4. 提交 PR

### ADR 模板

```markdown
# ADR-XXX: 标题

**日期**: YYYY-MM-DD
**状态**: 提议中 | 已接受 | 已废弃 | 已替代

## 背景
描述问题和上下文

## 决策
我们决定...

## 理由
为什么这样决定：
- 原因1
- 原因2

## 替代方案
考虑过的其他方案：
- 方案A：优缺点
- 方案B：优缺点

## 后果
这个决策的影响：
- ✅ 正面影响
- ⚠️ 需要注意的点
- ❌ 负面影响

## 参考
- 相关文档链接
```

## 📚 相关资源

### 用户文档
参见 [`docs/`](../docs/) 目录

### 实施计划
- [MVP 实施计划](../docs/microsegmentation-mvp-implementation-plan.md)
- [实施总结](../IMPLEMENTATION_SUMMARY.md)

### 参考项目
- [NeuVector 架构分析](../docs/neuvector-dp-agent-communication.md)
- [ZFW 架构分析](../docs/zfw-architecture-analysis.md)

---

*最后更新：2025-10-30*
