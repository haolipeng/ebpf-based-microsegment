# 代码搜索速查表

快速参考常用的代码搜索命令和模式。

---

## 🔍 grep 快速参考

### 基础搜索
```bash
grep -rn "pattern" .              # 递归搜索，显示行号
grep -rni "pattern" .             # 忽略大小写
grep -rnw "word" .                # 全词匹配
grep -rn "pattern" --include="*.go" .    # 只搜索 Go 文件
grep -rn "pattern" --exclude-dir="vendor" .    # 排除目录
```

### 上下文显示
```bash
grep -rn -A 3 "pattern" .         # 显示后 3 行
grep -rn -B 3 "pattern" .         # 显示前 3 行
grep -rn -C 3 "pattern" .         # 显示前后各 3 行
```

### 只显示信息
```bash
grep -rl "pattern" .              # 只显示文件名
grep -rc "pattern" .              # 只显示匹配计数
grep -rn "pattern" . | wc -l     # 统计匹配行数
```

---

## 📁 find 快速参考

### 按名称查找
```bash
find . -name "*.go"               # 查找所有 .go 文件
find . -iname "readme.md"         # 忽略大小写
find . -name "*test*"             # 包含 test 的文件
```

### 按类型查找
```bash
find . -type f                    # 只查找文件
find . -type d                    # 只查找目录
find . -type l                    # 只查找符号链接
```

### 排除路径
```bash
find . -name "*.go" -not -path "*/vendor/*"    # 排除 vendor
find . -name "*.go" ! -path "*/test/*"         # 排除 test
```

### 按时间查找
```bash
find . -mtime -7                  # 最近 7 天修改
find . -mmin -60                  # 最近 1 小时修改
```

### 执行操作
```bash
find . -name "*.go" -exec wc -l {} \;    # 统计行数
find . -name "*.tmp" -delete             # 删除文件（小心！）
```

---

## 🚀 ripgrep (rg) 快速参考

### 基础用法
```bash
rg "pattern"                      # 自动递归，彩色输出
rg -i "pattern"                   # 忽略大小写
rg -w "word"                      # 全词匹配
rg -t go "pattern"                # 只搜索 Go 文件
```

### 显示控制
```bash
rg -l "pattern"                   # 只显示文件名
rg -c "pattern"                   # 只显示计数
rg -C 3 "pattern"                 # 显示上下文 3 行
```

### 高级功能
```bash
rg --hidden "pattern"             # 包括隐藏文件
rg --no-ignore "pattern"          # 不使用 .gitignore
rg -U "pattern"                   # 多行匹配
rg --json "pattern"               # JSON 输出
```

---

## 🎯 常用搜索模式

### Go 代码搜索

#### 查找函数
```bash
# 所有函数定义
grep -Ern "^func " --include="*.go" .

# 特定函数
grep -Ern "^func NewReporter" --include="*.go" .

# 导出的函数（大写开头）
grep -Ern "^func [A-Z]" --include="*.go" .

# 方法（有接收者）
grep -Ern "^func \(.*\) " --include="*.go" .
```

#### 查找类型
```bash
# 所有结构体
grep -Ern "^type [a-zA-Z]+ struct" --include="*.go" .

# 所有接口
grep -Ern "^type [a-zA-Z]+ interface" --include="*.go" .

# 特定类型
grep -Ern "^type Reporter" --include="*.go" .
```

#### 查找导入
```bash
# 所有导入语句
grep -rh "^import" --include="*.go" .

# 特定包
grep -rn "github.com/sirupsen/logrus" --include="*.go" .

# 统计最常用的包
grep -rh "import" --include="*.go" . | grep -o '"[^"]*"' | sort | uniq -c | sort -rn | head -10
```

#### 查找模式
```bash
# 错误处理
grep -rn "if err != nil" --include="*.go" .

# goroutine
grep -rn "go func" --include="*.go" .

# defer 语句
grep -rn "defer " --include="*.go" .

# channel 操作
grep -rn " <- " --include="*.go" .
```

### 配置文件搜索

```bash
# 查找配置文件
find . -name "*.yaml" -o -name "*.yml" -o -name "*.json"

# 搜索配置项
grep -rn "server_addr" --include="*.yaml" .

# 查找端口配置
grep -Ern "port.*:" --include="*.yaml" .
```

### 文档搜索

```bash
# 查找 TODO
grep -rn "TODO" --include="*.go" --include="*.md" .

# 查找 FIXME
grep -rn "FIXME" .

# 查找代码块
grep -rn "^```" --include="*.md" .

# 查找链接
grep -Ern '\[.*\]\(.*\)' --include="*.md" .
```

---

## 🔧 组合命令模式

### 查找并统计
```bash
# 统计 Go 文件总行数
find . -name "*.go" -exec wc -l {} + | tail -1

# 统计每个目录的文件数
find . -name "*.go" | awk -F/ '{print $2}' | sort | uniq -c

# 统计最大的 10 个文件
find . -name "*.go" -exec wc -l {} + | sort -rn | head -10
```

### 查找并提取
```bash
# 提取所有函数名
grep -Eroh "^func [a-zA-Z][a-zA-Z0-9]*" --include="*.go" . | awk '{print $2}' | sort -u

# 提取所有结构体名
grep -Eroh "^type [a-zA-Z]+ struct" --include="*.go" . | awk '{print $2}' | sort -u

# 提取所有导入的包
grep -rh "import" --include="*.go" . | grep -o '"[^"]*"' | sort -u
```

### 查找并过滤
```bash
# 查找但排除测试文件
grep -rn "Reporter" --include="*.go" --exclude="*_test.go" .

# 查找但排除 vendor
find . -name "*.go" -not -path "*/vendor/*" | xargs grep -l "Reporter"

# 查找最近修改的文件中的模式
find . -name "*.go" -mtime -7 -exec grep -l "Reporter" {} \;
```

---

## 💡 实用技巧

### 创建别名

添加到 `~/.bashrc` 或 `~/.zshrc`:

```bash
# 搜索 Go 文件
alias gog='grep -rn --include="*.go"'

# 搜索但排除测试
alias gognt='grep -rn --include="*.go" --exclude="*_test.go"'

# 查找函数定义
findfunc() {
    grep -Ern "^func $1" --include="*.go" .
}

# 统计代码行数
alias countgo='find . -name "*.go" -not -path "*/vendor/*" -exec wc -l {} + | tail -1'
```

### 正则表达式常用模式

```bash
# IP 地址
grep -Ern '[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}'

# 邮箱
grep -Ern '[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}'

# URL
grep -Ern 'https?://[a-zA-Z0-9./?=_-]*'

# UUID
grep -Ern '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}'
```

---

## 📊 性能对比

| 工具 | 速度 | 功能 | 推荐度 |
|------|------|------|--------|
| ripgrep (rg) | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 强烈推荐 |
| ag | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | 推荐 |
| grep | ⭐⭐⭐ | ⭐⭐⭐⭐ | 基础必备 |
| ack | ⭐⭐⭐ | ⭐⭐⭐ | 可选 |

---

## 🎓 学习路径

### 初级
1. 掌握 `grep -rn`
2. 学会使用 `--include` 和 `--exclude`
3. 熟悉基础正则表达式

### 中级
1. 掌握 `find` 命令
2. 学会管道组合
3. 使用 ripgrep (rg)

### 高级
1. 掌握复杂正则表达式
2. 编写搜索脚本
3. 性能优化技巧

---

## 📚 相关文档

- [详细指南](code-search-guide.md) - 完整的搜索教程
- [会话记录](SESSION-INDEX.md) - 实战案例

---

**快速提示**:
- 使用 `rg` 替代 `grep` 可以提升 5-10 倍速度
- 善用 `--exclude-dir` 避免搜索 vendor/node_modules
- 复杂搜索时先用简单模式验证，再添加过滤条件
- 使用 `-C 3` 查看上下文帮助理解代码

---

**最后更新**: 2025-11-05
