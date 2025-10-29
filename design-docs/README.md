# 设计文档 (Design Documents)

本目录包含项目前期的分析、设计和规划文档。

## 📂 目录结构

```
design-docs/
├── README.md                    # 本文件
├── architecture/                # 架构设计文档
│   ├── ebpf-tc-architecture.md # eBPF + TC 架构设计
│   ├── design.md               # 详细设计文档
│   └── tc-mode-microsegmentation.md # TC 模式微隔离分析
├── analysis/                    # 可行性分析文档
│   ├── ebpf-tc-comparison.md   # 方案对比分析
│   ├── ebpf-tc-risks.md        # 风险分析
│   └── ebpf-tc-feasibility-index.md # 可行性分析索引
├── implementation/              # 实施指南
│   └── ebpf-tc-implementation.md # 6周实施指南
├── CHANGES.md                   # 设计变更记录
└── REVIEW_REPORT.md             # 设计审查报告
```

## 📖 文档说明

### 架构设计 (architecture/)

#### [ebpf-tc-architecture.md](architecture/ebpf-tc-architecture.md)
完整的系统架构设计，包括：
- 整体架构图
- eBPF 程序设计
- TC Hook 点设计
- 数据流设计
- 组件交互设计

#### [design.md](architecture/design.md)
详细的技术设计文档，包括：
- 核心数据结构设计
- BPF Map 设计
- 策略匹配算法
- 会话跟踪机制

#### [tc-mode-microsegmentation.md](architecture/tc-mode-microsegmentation.md)
TC 模式微隔离的详细分析

### 可行性分析 (analysis/)

#### [ebpf-tc-comparison.md](analysis/ebpf-tc-comparison.md)
eBPF + TC 方案与其他方案的对比分析：
- vs iptables
- vs XDP
- vs Cilium
- 性能对比
- 功能对比

#### [ebpf-tc-risks.md](analysis/ebpf-tc-risks.md)
项目风险分析和缓解措施：
- 技术风险
- 性能风险
- 维护风险
- 运维风险

#### [ebpf-tc-feasibility-index.md](analysis/ebpf-tc-feasibility-index.md)
可行性分析总索引，链接到所有相关分析文档

### 实施指南 (implementation/)

#### [ebpf-tc-implementation.md](implementation/ebpf-tc-implementation.md)
6 周详细实施计划：
- 周度目标和交付物
- 每日任务分解
- 学习路径
- 技术参考

**注意**：更详细的学习指南在 `docs/weekly-guide/` 目录中。

## 📋 项目管理文档

### [CHANGES.md](CHANGES.md)
设计变更记录，追踪设计决策的演进

### [REVIEW_REPORT.md](REVIEW_REPORT.md)
设计审查报告，记录评审意见和改进措施

## 🔗 相关文档

- **OpenSpec 规格**: `openspec/specs/` - 正式的需求规格（待创建）
- **学习指南**: `docs/weekly-guide/` - 6 周学习计划
- **架构分析**: `docs/zfw-architecture-analysis.md` - ZFW 参考架构分析
- **项目上下文**: `openspec/project.md` - 项目约定和技术栈

## 📚 阅读顺序建议

### 第一次阅读（了解项目）
1. `analysis/ebpf-tc-feasibility-index.md` - 从总索引开始
2. `architecture/ebpf-tc-architecture.md` - 理解整体架构
3. `implementation/ebpf-tc-implementation.md` - 了解实施计划

### 深入了解（开始开发前）
1. `analysis/ebpf-tc-comparison.md` - 理解方案选择理由
2. `architecture/design.md` - 学习详细设计
3. `analysis/ebpf-tc-risks.md` - 了解潜在风险
4. `docs/weekly-guide/` - 按周学习实施

### 参考查阅（开发过程中）
- 架构问题 → `architecture/`
- 技术选型 → `analysis/ebpf-tc-comparison.md`
- 风险评估 → `analysis/ebpf-tc-risks.md`
- 实施进度 → `implementation/ebpf-tc-implementation.md`

## 🆚 design-docs/ vs openspec/

| 方面 | design-docs/ | openspec/ |
|------|-------------|-----------|
| **目的** | 前期分析、设计、规划 | 正式需求规格和变更管理 |
| **内容** | 架构图、可行性分析、技术决策 | Requirements + Scenarios |
| **阶段** | 项目启动前/设计阶段 | 开发过程中持续维护 |
| **格式** | 自由格式文档 | 结构化规格（OpenSpec 格式）|
| **受众** | 架构师、技术评审者 | 开发者、测试人员 |
| **变更** | 相对固定（设计完成后） | 持续演进（功能开发） |

## 📝 文档维护

- **设计阶段**：积极更新 `design-docs/`
- **开发阶段**：主要维护 `openspec/specs/`
- **设计变更**：记录到 `CHANGES.md`
- **重大决策**：更新对应的架构或分析文档

---

**最后更新**：2025-10-29
**维护者**：项目团队
