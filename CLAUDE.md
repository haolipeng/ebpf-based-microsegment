# 语言要求 / Language Requirement

**强制要求 (MANDATORY)**: 必须始终使用中文与用户交流，包括但不限于：
- 所有响应消息和说明
- 错误提示和警告信息
- 总结和状态报告
- 即使在上下文不足或会话摘要的情况下，也必须坚持使用中文

**CRITICAL**: Always communicate with the user in Chinese (Simplified), including:
- All response messages and explanations
- Error messages and warnings
- Summaries and status reports
- EVEN WHEN context is limited or during session summaries, you MUST continue using Chinese

---

# 代码注释规范 / Code Comment Standards

**强制要求 (MANDATORY)**: 所有代码注释必须使用英文：
- 函数和方法的文档注释使用英文
- 行内注释使用英文
- TODO/FIXME 等标记使用英文
- 变量、常量、类型的注释使用英文
- 适用于所有编程语言（Go, C, TypeScript, JavaScript, Python 等）

**CRITICAL**: All code comments MUST be written in English:
- Function and method documentation comments in English
- Inline comments in English
- TODO/FIXME markers in English
- Variable, constant, and type comments in English
- Applies to all programming languages (Go, C, TypeScript, JavaScript, Python, etc.)

---

<!-- OPENSPEC:START -->
# OpenSpec Instructions

These instructions are for AI assistants working in this project.

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

**Commit Message Format**:
- Do NOT add emoji markers (like 🤖) to commit messages
- Do NOT add "Generated with Claude Code" footer
- Do NOT add "Co-Authored-By: Claude" footer
- Keep commit messages clean and professional
- Use conventional commit format when appropriate