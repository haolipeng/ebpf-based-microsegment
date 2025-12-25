# 代码搜索文档汇总

本目录包含完整的代码搜索教程和实践指南，帮助你快速掌握使用命令行工具搜索和分析代码的技能。

---

## 📚 文档清单

### 1. [架构指南](code-architecture-guide.md) ⭐ 推荐首读
**推荐指数**: ⭐⭐⭐⭐⭐
**适合人群**: 新加入项目的开发者
**长度**: ~15,000 字

**内容包括**:
- 项目三层架构详解 (Web UI / Server / Agent)
- 目录结构和模块职责
- 关键数据结构 (Protobuf, eBPF)
- 学习路径建议
- 代码搜索速查

**何时阅读**: 第一次接触项目或需要理解架构时

---

### 2. [完整教程](code-search-guide.md)
**推荐指数**: ⭐⭐⭐⭐⭐
**适合人群**: 初学者到高级用户
**长度**: ~15,000 字

**内容包括**:
- grep, find, awk, ripgrep 详细使用说明
- 50+ 实用示例
- 正则表达式教程
- 组合命令技巧
- 性能优化建议
- 自定义脚本示例

**何时阅读**: 第一次学习命令行搜索或需要系统了解时

---

### 3. [快速参考卡](code-search-cheatsheet.md)
**推荐指数**: ⭐⭐⭐⭐⭐  
**适合人群**: 所有用户  
**长度**: 速查表

**内容包括**:
- 常用命令速查
- Go 代码搜索模式
- 实用别名
- 性能对比表

**何时阅读**: 日常工作中快速查找命令时

---

### 4. [实战演示](code-search-demo.md)
**推荐指数**: ⭐⭐⭐⭐⭐  
**适合人群**: 实践学习者  
**长度**: ~5,000 字

**内容包括**:
- 8 个真实案例
- 所有示例都已验证
- 详细的输出和解释
- 组合技巧
- 即用型别名

**何时阅读**: 想看实际应用效果时

---

## 🚀 快速开始

### 5 分钟入门

```bash
# 1. 搜索文件中的文本
grep -rn "pattern" --include="*.go" .

# 2. 查找文件
find . -name "*.go"

# 3. 组合使用
find . -name "*.go" | xargs grep -l "pattern"

# 4. 使用现代工具（推荐）
rg "pattern"  # 需要先安装 ripgrep
```

### 安装推荐工具

```bash
# Ubuntu/Debian
sudo apt install ripgrep

# macOS
brew install ripgrep

# 验证安装
rg --version
```

---

## 📖 学习路径

### 第一天: 基础
1. 阅读 [快速参考卡](code-search-cheatsheet.md) 的 "基础搜索" 部分
2. 尝试 [实战演示](code-search-demo.md) 中的案例 1-3
3. 练习 5 个基础命令

**目标**: 能用 grep 和 find 进行基本搜索

### 第二天: 进阶
1. 阅读 [完整教程](code-search-guide.md) 的 "正则表达式" 部分
2. 尝试 [实战演示](code-search-demo.md) 中的案例 4-6
3. 学习管道组合

**目标**: 能组合多个命令解决复杂问题

### 第三天: 高级
1. 阅读 [完整教程](code-search-guide.md) 的 "高级技巧" 部分
2. 安装并学习 ripgrep
3. 创建自己的别名和脚本

**目标**: 高效搜索，创建自动化工具

---

## 💡 最佳实践

### DO ✅

```bash
# ✅ 使用 -n 显示行号
grep -rn "pattern" .

# ✅ 指定文件类型
grep -rn "pattern" --include="*.go" .

# ✅ 排除 vendor 目录
grep -rn "pattern" --exclude-dir="vendor" .

# ✅ 使用上下文查看
grep -rn -C 3 "pattern" .

# ✅ 优先使用 ripgrep
rg "pattern"
```

### DON'T ❌

```bash
# ❌ 不指定文件类型（太慢）
grep -r "pattern" .

# ❌ 不排除无关目录
grep -rn "pattern" .  # 会搜索 vendor, node_modules 等

# ❌ 没有行号（难以定位）
grep -r "pattern" .

# ❌ 不使用引号（可能出错）
grep -rn pattern .  # 特殊字符会有问题
```

---

## 🎯 常见问题

### Q1: grep 太慢怎么办？

**A**: 使用 ripgrep (rg)，速度快 5-10 倍：
```bash
# 安装
sudo apt install ripgrep  # Ubuntu
brew install ripgrep      # macOS

# 使用
rg "pattern"  # 自动排除 .git, vendor 等
```

### Q2: 如何搜索时忽略大小写？

**A**: 使用 `-i` 选项：
```bash
grep -rni "reporter" .
rg -i "reporter"
```

### Q3: 如何只显示文件名？

**A**: 使用 `-l` 选项：
```bash
grep -rl "pattern" .
rg -l "pattern"
```

### Q4: 如何搜索多个模式？

**A**: 使用 `-E` 和 `|`：
```bash
grep -Ern "Reporter|reporter" .
rg "Reporter|reporter"
```

### Q5: 如何排除测试文件？

**A**: 使用 `--exclude`：
```bash
grep -rn "pattern" --include="*.go" --exclude="*_test.go" .
rg "pattern" -g '!*_test.go'
```

---

## 🔧 工具对比

| 特性 | grep | ripgrep (rg) | ag | ack |
|------|------|--------------|----|----|
| 速度 | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| 易用性 | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| 功能 | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| 默认行为 | 需配置 | 智能 | 智能 | 智能 |
| 彩色输出 | 需启用 | 默认 | 默认 | 默认 |
| .gitignore | 不支持 | 支持 | 支持 | 部分支持 |

**推荐**: ripgrep (rg) - 速度最快，功能最全

---

## 📝 实用别名

把这些添加到 `~/.bashrc` 或 `~/.zshrc`:

```bash
# 搜索 Go 代码
alias gog='grep -rn --include="*.go"'

# 查找函数定义
findfunc() {
    grep -Ern "^func.*$1" --include="*.go" .
}

# 查找类型定义
findtype() {
    grep -Ern "^type $1" --include="*.go" .
}

# 统计代码行数
alias countgo='find . -name "*.go" -not -path "*/vendor/*" -exec wc -l {} + | tail -1'

# 查找 TODO
alias todos='grep -rn "TODO" --include="*.go" .'

# 使用 ripgrep（如果已安装）
alias rgg='rg -t go'
```

**使用方法**:
```bash
source ~/.bashrc  # 重新加载

# 使用别名
gog "Reporter"
findfunc NewReporter
findtype GRPCReporter
countgo
todos
```

---

## 🎓 推荐资源

### 官方文档
- [grep 手册](https://www.gnu.org/software/grep/manual/)
- [ripgrep GitHub](https://github.com/BurntSushi/ripgrep)
- [awk 教程](https://www.gnu.org/software/gawk/manual/)

### 在线工具
- [regex101](https://regex101.com/) - 正则表达式测试
- [explainshell](https://explainshell.com/) - 命令解释

### 视频教程
- YouTube: "Linux grep tutorial"
- YouTube: "ripgrep tutorial"

---

## 💬 反馈和贡献

如果你发现错误或有改进建议：
1. 查看现有文档
2. 尝试相关示例
3. 提出问题或建议

---

## 📄 许可证

本文档遵循项目许可证。

---

**最后更新**: 2025-11-22
**维护者**: Claude Code Assistant
**版本**: 2.0.0
