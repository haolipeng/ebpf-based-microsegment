# OpenSpec 指南

AI 编码助手使用 OpenSpec 进行规范驱动开发的指南。

## TL;DR 快速检查清单

- 搜索现有工作：`openspec spec list --long`, `openspec list`（仅用 `rg` 进行全文搜索）
- 决定范围：新能力 vs 修改现有能力
- 选择唯一的 `change-id`：kebab-case，动词引导（`add-`, `update-`, `remove-`, `refactor-`）
- 脚手架：`proposal.md`、`tasks.md`、`design.md`（仅在需要时），以及每个受影响能力的增量规范
- 编写增量：使用 `## ADDED|MODIFIED|REMOVED|RENAMED Requirements`；每个需求至少包含一个 `#### Scenario:`
- 验证：`openspec validate [change-id] --strict` 并修复问题
- 请求批准：在提案获得批准之前不要开始实施

## 三阶段工作流

### 阶段 1：创建变更
在以下情况下创建提案：
- 添加功能或特性
- 进行破坏性更改（API、模式）
- 更改架构或模式
- 优化性能（改变行为）
- 更新安全模式

触发器（示例）：
- "帮我创建变更提案"
- "帮我规划变更"
- "帮我创建提案"
- "我想创建规范提案"
- "我想创建规范"

宽松匹配指导：
- 包含以下之一：`proposal`、`change`、`spec`
- 以及以下之一：`create`、`plan`、`make`、`start`、`help`

跳过提案的情况：
- Bug 修复（恢复预期行为）
- 拼写错误、格式化、注释
- 依赖项更新（非破坏性）
- 配置更改
- 针对现有行为的测试

**工作流**
1. 查看 `openspec/project.md`、`openspec list` 和 `openspec list --specs` 以了解当前上下文。
2. 选择唯一的动词引导的 `change-id`，并在 `openspec/changes/<id>/` 下创建脚手架：`proposal.md`、`tasks.md`、可选的 `design.md` 和规范增量。
3. 使用 `## ADDED|MODIFIED|REMOVED Requirements` 起草规范增量，每个需求至少包含一个 `#### Scenario:`。
4. 运行 `openspec validate <id> --strict` 并在分享提案之前解决所有问题。

### 阶段 2：实施变更
将这些步骤作为待办事项跟踪，逐一完成。
1. **阅读 proposal.md** - 理解正在构建什么
2. **阅读 design.md**（如果存在）- 审查技术决策
3. **阅读 tasks.md** - 获取实施清单
4. **按顺序实施任务** - 按顺序完成
5. **确认完成** - 确保 `tasks.md` 中的每一项在更新状态之前都已完成
6. **更新清单** - 完成所有工作后，将每个任务设置为 `- [x]`，使列表反映现实
7. **批准关卡** - 在提案经过审查和批准之前不要开始实施

### 阶段 3：归档变更
部署后，创建单独的 PR：
- 移动 `changes/[name]/` → `changes/archive/YYYY-MM-DD-[name]/`
- 如果能力发生变化，更新 `specs/`
- 对于仅工具的更改使用 `openspec archive <change-id> --skip-specs --yes`（始终明确传递 change ID）
- 运行 `openspec validate --strict` 以确认归档的变更通过检查

## 任何任务之前

**上下文检查清单：**
- [ ] 阅读 `specs/[capability]/spec.md` 中的相关规范
- [ ] 检查 `changes/` 中的待处理更改是否有冲突
- [ ] 阅读 `openspec/project.md` 了解约定
- [ ] 运行 `openspec list` 查看活跃的更改
- [ ] 运行 `openspec list --specs` 查看现有能力

**创建规范之前：**
- 始终检查能力是否已存在
- 优先修改现有规范而不是创建重复项
- 使用 `openspec show [spec]` 审查当前状态
- 如果请求不明确，在搭建脚手架之前提出 1-2 个澄清性问题

### 搜索指导
- 枚举规范：`openspec spec list --long`（或 `--json` 用于脚本）
- 枚举更改：`openspec list`（或 `openspec change list --json` - 已弃用但可用）
- 显示详细信息：
  - 规范：`openspec show <spec-id> --type spec`（使用 `--json` 进行过滤）
  - 变更：`openspec show <change-id> --json --deltas-only`
- 全文搜索（使用 ripgrep）：`rg -n "Requirement:|Scenario:" openspec/specs`

## 快速开始

### CLI 命令

```bash
# 基本命令
openspec list                  # 列出活跃的更改
openspec list --specs          # 列出规范
openspec show [item]           # 显示更改或规范
openspec validate [item]       # 验证更改或规范
openspec archive <change-id> [--yes|-y]   # 部署后归档（添加 --yes 用于非交互式运行）

# 项目管理
openspec init [path]           # 初始化 OpenSpec
openspec update [path]         # 更新指令文件

# 交互模式
openspec show                  # 提示选择
openspec validate              # 批量验证模式

# 调试
openspec show [change] --json --deltas-only
openspec validate [change] --strict
```

### 命令标志

- `--json` - 机器可读输出
- `--type change|spec` - 消除项目歧义
- `--strict` - 全面验证
- `--no-interactive` - 禁用提示
- `--skip-specs` - 归档时不更新规范
- `--yes`/`-y` - 跳过确认提示（非交互式归档）

## 目录结构

```
openspec/
├── project.md              # 项目约定
├── specs/                  # 当前真相 - 已构建的内容
│   └── [capability]/       # 单一聚焦能力
│       ├── spec.md         # 需求和场景
│       └── design.md       # 技术模式
├── changes/                # 提案 - 应该更改的内容
│   ├── [change-name]/
│   │   ├── proposal.md     # 为什么、什么、影响
│   │   ├── tasks.md        # 实施清单
│   │   ├── design.md       # 技术决策（可选；见标准）
│   │   └── specs/          # 增量更改
│   │       └── [capability]/
│   │           └── spec.md # ADDED/MODIFIED/REMOVED
│   └── archive/            # 已完成的更改
```

## 创建变更提案

### 决策树

```
新请求？
├─ 恢复规范行为的 Bug 修复？ → 直接修复
├─ 拼写/格式/注释？ → 直接修复
├─ 新功能/能力？ → 创建提案
├─ 破坏性更改？ → 创建提案
├─ 架构更改？ → 创建提案
└─ 不清楚？ → 创建提案（更安全）
```

### 提案结构

1. **创建目录：** `changes/[change-id]/`（kebab-case，动词引导，唯一）

2. **编写 proposal.md：**
```markdown
## Why
[1-2 句话说明问题/机会]

## What Changes
- [更改项目列表]
- [用 **BREAKING** 标记破坏性更改]

## Impact
- 受影响的规范：[列出能力]
- 受影响的代码：[关键文件/系统]
```

3. **创建规范增量：** `specs/[capability]/spec.md`
```markdown
## ADDED Requirements
### Requirement: 新功能
系统必须(SHALL)提供...

#### Scenario: 成功案例
- **WHEN** 用户执行操作
- **THEN** 预期结果

## MODIFIED Requirements
### Requirement: 现有功能
[完整的修改后需求]

## REMOVED Requirements
### Requirement: 旧功能
**Reason**: [为什么移除]
**Migration**: [如何处理]
```
如果影响多个能力，在 `changes/[change-id]/specs/<capability>/spec.md` 下创建多个增量文件——每个能力一个。

4. **创建 tasks.md：**
```markdown
## 1. 实施
- [ ] 1.1 创建数据库模式
- [ ] 1.2 实现 API 端点
- [ ] 1.3 添加前端组件
- [ ] 1.4 编写测试
```

5. **在需要时创建 design.md：**
如果以下任一情况适用，则创建 `design.md`；否则省略：
- 跨领域更改（多个服务/模块）或新的架构模式
- 新的外部依赖或重要的数据模型更改
- 安全性、性能或迁移复杂性
- 在编码之前需要技术决策来消除歧义

最小 `design.md` 骨架：
```markdown
## Context
[背景、约束、利益相关者]

## Goals / Non-Goals
- 目标：[...]
- 非目标：[...]

## Decisions
- 决策：[什么和为什么]
- 考虑的替代方案：[选项 + 理由]

## Risks / Trade-offs
- [风险] → 缓解措施

## Migration Plan
[步骤、回滚]

## Open Questions
- [...]
```

## 规范文件格式

### 关键：场景格式

**正确**（使用 #### 标题）：
```markdown
#### Scenario: 用户登录成功
- **WHEN** 提供有效凭据
- **THEN** 返回 JWT 令牌
```

**错误**（不要使用项目符号或粗体）：
```markdown
- **Scenario: 用户登录**  ❌
**Scenario**: 用户登录     ❌
### Scenario: 用户登录      ❌
```

每个需求必须(MUST)至少有一个场景。

### 需求措辞
- 使用 SHALL/MUST 表示规范性需求（除非有意非规范性，否则避免 should/may）

### 增量操作

- `## ADDED Requirements` - 新能力
- `## MODIFIED Requirements` - 行为更改
- `## REMOVED Requirements` - 废弃功能
- `## RENAMED Requirements` - 名称更改

标题使用 `trim(header)` 匹配 - 忽略空格。

#### 何时使用 ADDED vs MODIFIED
- ADDED：引入可以作为需求独立存在的新能力或子能力。当更改是正交的（例如，添加"斜杠命令配置"）而不是改变现有需求的语义时，首选 ADDED。
- MODIFIED：更改现有需求的行为、范围或验收标准。始终粘贴完整的、更新的需求内容（标题 + 所有场景）。归档器将用您在此处提供的内容替换整个需求；部分增量将丢失先前的详细信息。
- RENAMED：仅当名称更改时使用。如果您还更改了行为，请使用 RENAMED（名称）加上 MODIFIED（内容）引用新名称。

常见陷阱：使用 MODIFIED 添加新关注点而不包括先前的文本。这会在归档时导致详细信息丢失。如果您没有明确更改现有需求，请改为在 ADDED 下添加新需求。

正确编写 MODIFIED 需求：
1) 在 `openspec/specs/<capability>/spec.md` 中找到现有需求。
2) 复制整个需求块（从 `### Requirement: ...` 到其场景）。
3) 将其粘贴到 `## MODIFIED Requirements` 下并编辑以反映新行为。
4) 确保标题文本完全匹配（空格不敏感）并保持至少一个 `#### Scenario:`。

RENAMED 示例：
```markdown
## RENAMED Requirements
- FROM: `### Requirement: 登录`
- TO: `### Requirement: 用户认证`
```

## 故障排查

### 常见错误

**"Change must have at least one delta"**
- 检查 `changes/[name]/specs/` 是否存在 .md 文件
- 验证文件是否有操作前缀（## ADDED Requirements）

**"Requirement must have at least one scenario"**
- 检查场景是否使用 `#### Scenario:` 格式（4 个井号）
- 不要使用项目符号或粗体作为场景标题

**静默场景解析失败**
- 需要精确格式：`#### Scenario: 名称`
- 使用以下方式调试：`openspec show [change] --json --deltas-only`

### 验证提示

```bash
# 始终使用严格模式进行全面检查
openspec validate [change] --strict

# 调试增量解析
openspec show [change] --json | jq '.deltas'

# 检查特定需求
openspec show [spec] --json -r 1
```

## 快乐路径脚本

```bash
# 1) 探索当前状态
openspec spec list --long
openspec list
# 可选的全文搜索：
# rg -n "Requirement:|Scenario:" openspec/specs
# rg -n "^#|Requirement:" openspec/changes

# 2) 选择变更 id 并搭建脚手架
CHANGE=add-two-factor-auth
mkdir -p openspec/changes/$CHANGE/{specs/auth}
printf "## Why\n...\n\n## What Changes\n- ...\n\n## Impact\n- ...\n" > openspec/changes/$CHANGE/proposal.md
printf "## 1. Implementation\n- [ ] 1.1 ...\n" > openspec/changes/$CHANGE/tasks.md

# 3) 添加增量（示例）
cat > openspec/changes/$CHANGE/specs/auth/spec.md << 'EOF'
## ADDED Requirements
### Requirement: 双因素认证
用户必须(MUST)在登录期间提供第二个因素。

#### Scenario: 需要 OTP
- **WHEN** 提供有效凭据
- **THEN** 需要 OTP 质询
EOF

# 4) 验证
openspec validate $CHANGE --strict
```

## 多能力示例

```
openspec/changes/add-2fa-notify/
├── proposal.md
├── tasks.md
└── specs/
    ├── auth/
    │   └── spec.md   # ADDED: 双因素认证
    └── notifications/
        └── spec.md   # ADDED: OTP 邮件通知
```

auth/spec.md
```markdown
## ADDED Requirements
### Requirement: 双因素认证
...
```

notifications/spec.md
```markdown
## ADDED Requirements
### Requirement: OTP 邮件通知
...
```

## 最佳实践

### 简单优先
- 默认 <100 行新代码
- 单文件实现，直到证明不足
- 避免没有明确理由的框架
- 选择无聊的、经过验证的模式

### 复杂性触发器
仅在以下情况下增加复杂性：
- 性能数据显示当前解决方案太慢
- 具体的规模要求（>1000 用户，>100MB 数据）
- 需要抽象的多个经过验证的用例

### 清晰的引用
- 使用 `file.ts:42` 格式表示代码位置
- 将规范引用为 `specs/auth/spec.md`
- 链接相关更改和 PR

### 能力命名
- 使用动词-名词：`user-auth`、`payment-capture`
- 每个能力单一目的
- 10 分钟可理解性规则
- 如果描述需要"AND"则拆分

### 变更 ID 命名
- 使用 kebab-case，简短且描述性：`add-two-factor-auth`
- 优先使用动词引导的前缀：`add-`、`update-`、`remove-`、`refactor-`
- 确保唯一性；如果已使用，追加 `-2`、`-3` 等

## 工具选择指南

| 任务 | 工具 | 原因 |
|------|------|-----|
| 按模式查找文件 | Glob | 快速模式匹配 |
| 搜索代码内容 | Grep | 优化的正则表达式搜索 |
| 读取特定文件 | Read | 直接文件访问 |
| 探索未知范围 | Task | 多步骤调查 |

## 错误恢复

### 变更冲突
1. 运行 `openspec list` 查看活跃的更改
2. 检查规范是否重叠
3. 与更改所有者协调
4. 考虑合并提案

### 验证失败
1. 使用 `--strict` 标志运行
2. 检查 JSON 输出以获取详细信息
3. 验证规范文件格式
4. 确保场景格式正确

### 缺少上下文
1. 首先阅读 project.md
2. 检查相关规范
3. 审查最近的归档
4. 要求澄清

## 快速参考

### 阶段指示器
- `changes/` - 已提出，尚未构建
- `specs/` - 已构建和部署
- `archive/` - 已完成的更改

### 文件目的
- `proposal.md` - 为什么和什么
- `tasks.md` - 实施步骤
- `design.md` - 技术决策
- `spec.md` - 需求和行为

### CLI 要点
```bash
openspec list              # 正在进行什么？
openspec show [item]       # 查看详细信息
openspec validate --strict # 是否正确？
openspec archive <change-id> [--yes|-y]  # 标记完成（添加 --yes 用于自动化）
```

记住：规范是真相。更改是提案。保持它们同步。
