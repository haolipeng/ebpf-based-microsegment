<!-- OPENSPEC:START -->
# OpenSpec Instructions

These instructions are for AI assistants working in this project.

**重要：所有 OpenSpec 相关的 Markdown 文件必须使用中文编写**
- 所有 `proposal.md`、`design.md`、`tasks.md` 文件必须使用中文
- 所有规范文件（`spec.md`）中的需求和场景必须使用中文
- 代码注释和文档字符串应使用中文
- 仅在以下情况使用英文：
  - 代码标识符（变量名、函数名、类型名）
  - API 端点路径
  - 技术术语的原文引用（可在括号中注明）
  - 命令行示例和代码块

Always open `@/openspec/AGENTS.md` when the request:
- Mentions planning or proposals (words like proposal, spec, change, plan)
- Introduces new capabilities, breaking changes, architecture shifts, or big performance/security work
- Sounds ambiguous and you need the authoritative spec before coding

Use `@/openspec/AGENTS.md` to learn:
- How to create and apply change proposals
- Spec format and conventions
- Project structure and guidelines

Keep this managed block so 'openspec update' can refresh the instructions.

<!-- OPENSPEC:END -->

---

# Git Repository Instructions

**IMPORTANT**: Do NOT automatically push commits to GitHub repository. Only create local commits. The user will manually push changes to the remote repository when ready.