# 代码搜索实践指南

本指南介绍如何使用命令行工具高效搜索和分析代码库，这些技巧可以帮助你快速定位代码、理解项目结构、查找引用关系。

---

## 目录

1. [基础工具介绍](#基础工具介绍)
2. [grep 搜索技巧](#grep-搜索技巧)
3. [find 查找文件](#find-查找文件)
4. [awk 文本处理](#awk-文本处理)
5. [组合使用技巧](#组合使用技巧)
6. [实战案例](#实战案例)
7. [高级技巧](#高级技巧)
8. [常用模式速查](#常用模式速查)

---

## 基础工具介绍

### 1. grep - 文本搜索

**用途**: 在文件中搜索匹配的文本模式

**基本语法**:
```bash
grep [选项] 模式 文件
```

**常用选项**:
- `-r` / `-R`: 递归搜索目录
- `-n`: 显示行号
- `-i`: 忽略大小写
- `-v`: 反向匹配（显示不匹配的行）
- `-l`: 只显示文件名
- `-c`: 只显示匹配行数
- `-A N`: 显示匹配行后 N 行
- `-B N`: 显示匹配行前 N 行
- `-C N`: 显示匹配行前后各 N 行
- `-E`: 使用扩展正则表达式
- `-w`: 全词匹配

### 2. find - 查找文件

**用途**: 在目录树中查找文件

**基本语法**:
```bash
find 路径 [选项] [表达式]
```

**常用选项**:
- `-name`: 按文件名查找
- `-iname`: 按文件名查找（忽略大小写）
- `-type f`: 查找文件
- `-type d`: 查找目录
- `-path`: 按路径模式查找
- `-mtime`: 按修改时间查找
- `-size`: 按文件大小查找

### 3. awk - 文本处理

**用途**: 强大的文本处理和数据提取工具

**基本语法**:
```bash
awk '模式 { 动作 }' 文件
```

### 4. ripgrep (rg) - 现代搜索工具

**用途**: 比 grep 更快的搜索工具（Claude Code 内部使用）

**安装**:
```bash
# Ubuntu/Debian
sudo apt install ripgrep

# macOS
brew install ripgrep
```

---

## grep 搜索技巧

### 基础搜索

#### 1. 在当前目录递归搜索
```bash
# 搜索包含 "Reporter" 的所有文件
grep -r "Reporter" .

# 带行号
grep -rn "Reporter" .

# 忽略大小写
grep -rni "reporter" .
```

#### 2. 搜索特定类型的文件
```bash
# 只搜索 .go 文件
grep -rn "Reporter" --include="*.go" .

# 排除特定文件类型
grep -rn "Reporter" --exclude="*.md" .

# 排除特定目录
grep -rn "Reporter" --exclude-dir="vendor" --exclude-dir="node_modules" .
```

#### 3. 显示上下文
```bash
# 显示匹配行前后 3 行
grep -rn -C 3 "Reporter" .

# 只显示后 5 行
grep -rn -A 5 "Reporter" .

# 只显示前 5 行
grep -rn -B 5 "Reporter" .
```

### 正则表达式搜索

#### 1. 基础正则
```bash
# 搜索以 "type" 开头的行
grep -rn "^type" --include="*.go" .

# 搜索以 "}" 结尾的行
grep -rn "}$" --include="*.go" .

# 搜索包含数字的行
grep -rn "[0-9]" --include="*.go" .
```

#### 2. 扩展正则（使用 -E）
```bash
# 搜索 "Reporter" 或 "reporter"
grep -Ern "Reporter|reporter" --include="*.go" .

# 搜索函数定义（func 开头）
grep -Ern "^func [a-zA-Z]+" --include="*.go" .

# 搜索接口定义
grep -Ern "^type [a-zA-Z]+ interface" --include="*.go" .

# 搜索结构体定义
grep -Ern "^type [a-zA-Z]+ struct" --include="*.go" .
```

#### 3. 全词匹配
```bash
# 只匹配完整的单词 "Report"（不匹配 "Reporter"）
grep -rnw "Report" --include="*.go" .
```

### 实用搜索模式

#### 1. 搜索函数定义
```bash
# 搜索所有函数定义
grep -Ern "^func " --include="*.go" . | head -20

# 搜索特定函数
grep -Ern "^func NewGRPCReporter" --include="*.go" .

# 搜索接收者方法
grep -Ern "^func \(.*\) " --include="*.go" .
```

#### 2. 搜索导入语句
```bash
# 搜索所有 import
grep -Ern "^import" --include="*.go" .

# 搜索特定包的导入
grep -Ern "github.com/sirupsen/logrus" --include="*.go" .
```

#### 3. 搜索注释
```bash
# 搜索单行注释
grep -rn "^//" --include="*.go" .

# 搜索多行注释开始
grep -rn "^/\*" --include="*.go" .

# 搜索 TODO 注释
grep -rn "// TODO" --include="*.go" .
```

#### 4. 搜索字符串字面量
```bash
# 搜索包含特定字符串的行
grep -rn '"reporter"' --include="*.go" .

# 搜索日志语句
grep -rn 'log\.' --include="*.go" .
```

---

## find 查找文件

### 按文件名查找

#### 1. 基础查找
```bash
# 查找所有 .go 文件
find . -name "*.go"

# 查找特定文件
find . -name "reporter.go"

# 忽略大小写
find . -iname "reporter.go"
```

#### 2. 使用通配符
```bash
# 查找所有包含 "test" 的 .go 文件
find . -name "*test*.go"

# 查找所有 _test.go 文件（单元测试）
find . -name "*_test.go"

# 查找所有 config 相关文件
find . -name "*config*"
```

#### 3. 排除目录
```bash
# 排除 vendor 目录
find . -name "*.go" -not -path "*/vendor/*"

# 排除多个目录
find . -name "*.go" -not -path "*/vendor/*" -not -path "*/node_modules/*"
```

### 按文件类型查找

```bash
# 只查找文件
find . -type f -name "*.go"

# 只查找目录
find . -type d -name "pkg"

# 查找符号链接
find . -type l
```

### 按时间查找

```bash
# 查找最近 7 天修改的文件
find . -type f -mtime -7

# 查找 7 天前修改的文件
find . -type f -mtime +7

# 查找最近 1 小时修改的文件
find . -type f -mmin -60
```

### 按大小查找

```bash
# 查找大于 1MB 的文件
find . -type f -size +1M

# 查找小于 100KB 的文件
find . -type f -size -100k

# 查找空文件
find . -type f -size 0
```

### 执行操作

```bash
# 对找到的文件执行命令
find . -name "*.go" -exec wc -l {} \;

# 删除找到的文件（危险！先用 -print 测试）
find . -name "*.tmp" -delete

# 查找并统计行数
find . -name "*.go" -exec wc -l {} + | tail -1
```

---

## awk 文本处理

### 基础用法

#### 1. 打印特定列
```bash
# 打印第一列
ls -l | awk '{print $1}'

# 打印第一列和第三列
ls -l | awk '{print $1, $3}'

# 打印最后一列
ls -l | awk '{print $NF}'
```

#### 2. 模式匹配
```bash
# 打印包含 "go" 的行
ls -l | awk '/go/ {print}'

# 打印第五列大于 1000 的行
ls -l | awk '$5 > 1000 {print}'

# 打印以 "d" 开头的行（目录）
ls -l | awk '/^d/ {print $NF}'
```

#### 3. 字段分隔符
```bash
# 使用冒号作为分隔符（如 /etc/passwd）
awk -F: '{print $1, $3}' /etc/passwd

# 使用多个分隔符
echo "a:b,c" | awk -F'[:,]' '{print $1, $2, $3}'
```

### 实用示例

#### 1. 统计代码行数
```bash
# 统计所有 .go 文件的总行数
find . -name "*.go" -exec wc -l {} + | awk '{sum+=$1} END {print sum}'

# 按目录统计
find . -name "*.go" | awk -F/ '{dir=$2; getline < $0; lines[dir]+=$1} END {for (d in lines) print d, lines[d]}'
```

#### 2. 提取信息
```bash
# 从 grep 输出中提取文件名
grep -rn "Reporter" --include="*.go" . | awk -F: '{print $1}' | sort -u

# 提取函数名
grep -Ern "^func " --include="*.go" . | awk '{print $2}' | cut -d'(' -f1 | sort -u
```

#### 3. 格式化输出
```bash
# 格式化 ls 输出
ls -l | awk '{printf "%-10s %10s %s\n", $3, $5, $NF}'

# 添加行号
cat file.go | awk '{printf "%4d: %s\n", NR, $0}'
```

---

## 组合使用技巧

### 管道组合

#### 1. grep + awk
```bash
# 搜索函数定义并提取函数名
grep -Ern "^func " --include="*.go" . | awk -F: '{print $1 ":" $2}' | awk '{print $2}' | sort -u

# 搜索并统计
grep -r "Reporter" --include="*.go" . | wc -l
```

#### 2. find + grep
```bash
# 在找到的文件中搜索
find . -name "*.go" -exec grep -Hn "Reporter" {} \;

# 更高效的方式
find . -name "*.go" | xargs grep -Hn "Reporter"
```

#### 3. find + awk
```bash
# 统计每个目录的文件数
find . -type f -name "*.go" | awk -F/ '{print $2}' | sort | uniq -c
```

### 复杂管道

#### 1. 代码分析
```bash
# 查找最大的 10 个文件
find . -type f -name "*.go" -exec wc -l {} + | sort -rn | head -10

# 统计每种文件类型的数量
find . -type f | awk -F. '{print $NF}' | sort | uniq -c | sort -rn
```

#### 2. 依赖分析
```bash
# 分析导入的包
grep -rh "^import" --include="*.go" . | sort -u

# 统计最常用的包
grep -rh "import" --include="*.go" . | grep -o '"[^"]*"' | sort | uniq -c | sort -rn | head -20
```

---

## 实战案例

### 案例 1: 查找函数定义和调用

```bash
# 1. 查找函数定义位置
grep -rn "^func NewGRPCReporter" --include="*.go" .

# 2. 查找函数调用位置
grep -rn "NewGRPCReporter(" --include="*.go" .

# 3. 统计调用次数
grep -rn "NewGRPCReporter(" --include="*.go" . | wc -l

# 4. 查看调用上下文
grep -rn -A 5 -B 5 "NewGRPCReporter(" --include="*.go" .
```

### 案例 2: 查找结构体和接口

```bash
# 1. 查找所有接口定义
grep -Ern "^type [a-zA-Z]+ interface" --include="*.go" .

# 2. 查找特定接口
grep -Ern "^type Reporter interface" --include="*.go" .

# 3. 查找实现该接口的结构体
grep -Ern "type.*Reporter" --include="*.go" .

# 4. 查找结构体方法
grep -Ern "func \(.*GRPCReporter\)" --include="*.go" .
```

### 案例 3: 分析配置文件

```bash
# 1. 查找所有 YAML 配置文件
find . -name "*.yaml" -o -name "*.yml"

# 2. 搜索特定配置项
grep -rn "server_addr" --include="*.yaml" .

# 3. 提取所有配置键
grep -rh ":" --include="*.yaml" . | awk -F: '{print $1}' | sort -u

# 4. 查找带注释的配置
grep -rn "^#" --include="*.yaml" .
```

### 案例 4: 代码质量检查

```bash
# 1. 查找 TODO 注释
grep -rn "TODO" --include="*.go" .

# 2. 查找 FIXME 注释
grep -rn "FIXME" --include="*.go" .

# 3. 查找可能的错误处理遗漏
grep -rn "if err != nil" --include="*.go" . | wc -l
grep -rn ":= .*()" --include="*.go" . | grep -v "if err" | head -20

# 4. 查找长函数（超过 50 行）
for file in $(find . -name "*.go"); do
    awk '/^func / {start=NR; func=$0} /^}/ && start {
        if (NR-start > 50) print FILENAME":"start":"func
    }' "$file"
done
```

### 案例 5: 依赖关系分析

```bash
# 1. 查找所有导入的外部包
grep -rh "^import" --include="*.go" . | \
    grep -o '"[^"]*"' | \
    grep -v "^\"github.com/ebpf-microsegment" | \
    sort -u

# 2. 统计最常用的标准库
grep -rh "import" --include="*.go" . | \
    grep -o '"[^"]*"' | \
    grep -v "github.com" | \
    sort | uniq -c | sort -rn | head -10

# 3. 查找内部包的引用关系
grep -rh "import" --include="*.go" . | \
    grep "github.com/ebpf-microsegment" | \
    sort | uniq -c | sort -rn
```

---

## 高级技巧

### 1. 使用 ripgrep (rg)

ripgrep 是现代的 grep 替代品，速度更快，功能更强。

```bash
# 基础搜索（自动递归，自动排除 .git 等）
rg "Reporter"

# 搜索特定类型文件
rg -t go "Reporter"

# 显示匹配的上下文
rg -C 3 "Reporter"

# 只显示文件名
rg -l "Reporter"

# 统计匹配数
rg -c "Reporter"

# 多行匹配
rg -U "func.*\{.*Reporter.*\}" --multiline

# JSON 输出（便于脚本处理）
rg --json "Reporter"
```

### 2. 使用 ag (The Silver Searcher)

```bash
# 基础搜索
ag "Reporter"

# 忽略大小写
ag -i "reporter"

# 只搜索 Go 文件
ag --go "Reporter"

# 显示上下文
ag -C 3 "Reporter"
```

### 3. 组合工具创建自定义命令

创建 `~/.bash_aliases` 或 `~/.zshrc`:

```bash
# 快速搜索 Go 文件
alias gog='grep -rn --include="*.go"'

# 查找函数定义
findfunc() {
    grep -Ern "^func $1" --include="*.go" .
}

# 查找接口定义
findinterface() {
    grep -Ern "^type $1 interface" --include="*.go" .
}

# 统计代码行数
countlines() {
    find . -name "*.go" -not -path "*/vendor/*" -exec wc -l {} + | tail -1
}

# 查找并排除测试文件
gognotest() {
    grep -rn --include="*.go" --exclude="*_test.go" "$@" .
}
```

### 4. 正则表达式速查

```bash
# 匹配 IP 地址
grep -Ern '[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}' .

# 匹配邮箱
grep -Ern '[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}' .

# 匹配 URL
grep -Ern 'https?://[a-zA-Z0-9./?=_-]*' .

# 匹配 UUID
grep -Ern '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}' .
```

---

## 常用模式速查

### Go 代码搜索

```bash
# 查找所有导出的函数（大写开头）
grep -Ern "^func [A-Z]" --include="*.go" .

# 查找所有私有函数（小写开头）
grep -Ern "^func [a-z]" --include="*.go" .

# 查找结构体字段
grep -Ern "^\s+[A-Z][a-zA-Z]*\s+[a-zA-Z]" --include="*.go" .

# 查找错误处理
grep -rn "if err != nil" --include="*.go" .

# 查找 defer 语句
grep -rn "defer " --include="*.go" .

# 查找 goroutine
grep -rn "go " --include="*.go" .

# 查找 channel 操作
grep -rn "<-" --include="*.go" .
```

### 配置文件搜索

```bash
# 查找所有配置文件
find . \( -name "*.yaml" -o -name "*.yml" -o -name "*.json" -o -name "*.toml" \)

# 搜索配置项
grep -rn "port:" --include="*.yaml" .

# 查找环境变量引用
grep -rn "\$[A-Z_]" --include="*.yaml" .
```

### 文档搜索

```bash
# 查找所有 Markdown 文件
find . -name "*.md"

# 搜索文档中的代码块
grep -rn "^```" --include="*.md" .

# 查找链接
grep -Ern '\[.*\]\(.*\)' --include="*.md" .

# 查找图片
grep -Ern '!\[.*\]\(.*\)' --include="*.md" .
```

---

## 性能优化技巧

### 1. 并行处理

```bash
# 使用 xargs 并行执行
find . -name "*.go" | xargs -P 4 grep -l "Reporter"

# 使用 GNU parallel（需要安装）
find . -name "*.go" | parallel grep -l "Reporter" {}
```

### 2. 限制搜索范围

```bash
# 只搜索特定目录
grep -rn "Reporter" src/agent/pkg/

# 排除大文件
find . -name "*.go" -size -1M -exec grep -l "Reporter" {} \;
```

### 3. 使用更快的工具

```bash
# ripgrep（最快）
rg "Reporter"

# ag（很快）
ag "Reporter"

# grep with PCRE（较快）
grep -rP "Reporter" .
```

---

## 实用脚本示例

### 1. 代码统计脚本

创建 `code-stats.sh`:

```bash
#!/bin/bash

echo "=== Code Statistics ==="
echo ""

echo "Go Files:"
find . -name "*.go" -not -path "*/vendor/*" | wc -l

echo ""
echo "Total Lines of Go Code:"
find . -name "*.go" -not -path "*/vendor/*" -exec wc -l {} + | tail -1

echo ""
echo "Test Files:"
find . -name "*_test.go" | wc -l

echo ""
echo "Top 10 Largest Files:"
find . -name "*.go" -not -path "*/vendor/*" -exec wc -l {} + | sort -rn | head -11 | tail -10

echo ""
echo "Package Count:"
find . -type d -name "pkg" | wc -l
```

### 2. 查找未使用的代码

创建 `find-unused.sh`:

```bash
#!/bin/bash

# 查找定义但未被调用的函数
for func in $(grep -Eroh "^func [a-zA-Z][a-zA-Z0-9]*" --include="*.go" . | awk '{print $2}'); do
    count=$(grep -r "$func(" --include="*.go" . | grep -v "^func $func" | wc -l)
    if [ $count -eq 0 ]; then
        echo "Potentially unused function: $func"
        grep -n "^func $func" --include="*.go" .
    fi
done
```

### 3. 依赖图生成

创建 `dep-graph.sh`:

```bash
#!/bin/bash

echo "digraph dependencies {"

find . -name "*.go" -not -path "*/vendor/*" | while read file; do
    package=$(dirname "$file" | sed 's/\.\///')
    grep -h "^import" "$file" | grep -o '"[^"]*"' | tr -d '"' | while read dep; do
        echo "  \"$package\" -> \"$dep\";"
    done
done

echo "}"
```

运行：`./dep-graph.sh | dot -Tpng > deps.png`

---

## 小结

### 最常用的命令

```bash
# 1. 递归搜索文本
grep -rn "pattern" --include="*.go" .

# 2. 查找文件
find . -name "*.go"

# 3. 组合搜索
find . -name "*.go" | xargs grep -l "pattern"

# 4. 使用 ripgrep（推荐）
rg -t go "pattern"

# 5. 查看上下文
grep -rn -C 3 "pattern" --include="*.go" .
```

### 推荐工作流

1. **快速搜索**: 使用 `rg` 或 `ag`
2. **详细分析**: 使用 `grep` 配合正则表达式
3. **文件定位**: 使用 `find`
4. **数据提取**: 使用 `awk`
5. **复杂操作**: 组合管道命令

### 学习建议

1. 从基础 `grep` 和 `find` 开始
2. 逐步学习正则表达式
3. 掌握管道组合
4. 尝试 `ripgrep` 等现代工具
5. 创建自己的别名和脚本

---

## 参考资源

- [grep 手册](https://www.gnu.org/software/grep/manual/)
- [ripgrep 文档](https://github.com/BurntSushi/ripgrep)
- [awk 教程](https://www.gnu.org/software/gawk/manual/)
- [正则表达式速查](https://regex101.com/)

---

**最后更新**: 2025-11-05
**作者**: Claude Code Assistant
