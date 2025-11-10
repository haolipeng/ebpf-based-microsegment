# 语言要求 / Language Requirement

**强制要求 (MANDATORY)**: 必须始终使用中文与用户交流，包括但不限于：
- 所有响应消息和说明
- 代码注释和文档
- 错误提示和警告信息
- 总结和状态报告
- 即使在上下文不足或会话摘要的情况下，也必须坚持使用中文

**CRITICAL**: Always communicate with the user in Chinese (Simplified), including:
- All response messages and explanations
- Code comments and documentation
- Error messages and warnings
- Summaries and status reports
- EVEN WHEN context is limited or during session summaries, you MUST continue using Chinese

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