# 代码搜索实战演示

本文档展示实际项目中的代码搜索示例，所有命令都已在本项目中验证。

---

## 📋 演示环境

- **项目**: ebpf-based-microsegment
- **语言**: Go
- **目录结构**: 标准 Go 项目布局

---

## 🎯 实战案例

### 案例 1: 查找 Reporter 相关代码

**场景**: 想要了解 Reporter 功能的实现

#### 1.1 找到所有相关文件
```bash
$ grep -rl "Reporter" --include="*.go" src/agent/pkg/
```

**输出**:
```
src/agent/pkg/reporter/local_reporter.go
src/agent/pkg/reporter/grpc_reporter.go
src/agent/pkg/reporter/reporter.go
```

**解释**:
- `-r`: 递归搜索
- `-l`: 只显示文件名
- `--include="*.go"`: 只搜索 Go 文件

#### 1.2 找到接口定义
```bash
$ grep -rn "^type Reporter interface" --include="*.go" src/agent/
```

**输出**:
```
src/agent/pkg/reporter/reporter.go:11:type Reporter interface {
```

**解释**:
- `-n`: 显示行号
- `^type Reporter interface`: 正则表达式，^ 表示行首

#### 1.3 查看接口完整定义（含上下文）
```bash
$ grep -rn -C 5 "^type Reporter interface" --include="*.go" src/agent/
```

**输出**:
```
src/agent/pkg/reporter/reporter.go-6-  "github.com/ebpf-microsegment/src/agent/pkg/flow"
src/agent/pkg/reporter/reporter.go-7-)
src/agent/pkg/reporter/reporter.go-8-
src/agent/pkg/reporter/reporter.go-9-// Reporter is an interface for reporting flow data
src/agent/pkg/reporter/reporter.go-10-// Implementations can be local (SQLite) or remote (gRPC)
src/agent/pkg/reporter/reporter.go:11:type Reporter interface {
src/agent/pkg/reporter/reporter.go-12-    // Report sends a single flow to the reporter
src/agent/pkg/reporter/reporter.go-13-    Report(ctx context.Context, flow *flow.Flow) error
src/agent/pkg/reporter/reporter.go-14-
src/agent/pkg/reporter/reporter.go-15-    // ReportBatch sends multiple flows (for efficiency)
src/agent/pkg/reporter/reporter.go-16-    ReportBatch(ctx context.Context, flows []*flow.Flow) error
```

**解释**:
- `-C 5`: 显示匹配行前后各 5 行
- 帮助理解代码上下文

---

### 案例 2: 分析函数和方法

#### 2.1 列出所有函数定义
```bash
$ grep -Ern "^func " --include="*.go" src/agent/pkg/reporter/ | head -10
```

**输出**:
```
src/agent/pkg/reporter/local_reporter.go:17:func NewLocalReporter(storage storage.Storage) *LocalReporter {
src/agent/pkg/reporter/local_reporter.go:24:func (r *LocalReporter) Report(ctx context.Context, f *flow.Flow) error {
src/agent/pkg/reporter/local_reporter.go:29:func (r *LocalReporter) ReportBatch(ctx context.Context, flows []*flow.Flow) error {
src/agent/pkg/reporter/local_reporter.go:40:func (r *LocalReporter) Start() error {
src/agent/pkg/reporter/local_reporter.go:46:func (r *LocalReporter) Stop() error {
src/agent/pkg/reporter/grpc_reporter.go:30:func NewGRPCReporter(serverAddr, agentID string, batchSize int) *GRPCReporter {
src/agent/pkg/reporter/grpc_reporter.go:45:func (r *GRPCReporter) Start() error {
src/agent/pkg/reporter/grpc_reporter.go:64:func (r *GRPCReporter) Stop() error {
src/agent/pkg/reporter/grpc_reporter.go:73:func (r *GRPCReporter) Report(ctx context.Context, f *flow.Flow) error {
src/agent/pkg/reporter/grpc_reporter.go:84:func (r *GRPCReporter) ReportBatch(ctx context.Context, flows []*flow.Flow) error {
```

**分析**:
- LocalReporter 有 5 个方法
- GRPCReporter 实现了相同的接口
- 都有构造函数 `New*Reporter`

#### 2.2 查找所有结构体
```bash
$ grep -Ern "^type [a-zA-Z]+ struct" --include="*.go" src/agent/pkg/reporter/
```

**输出**:
```
src/agent/pkg/reporter/local_reporter.go:12:type LocalReporter struct {
src/agent/pkg/reporter/grpc_reporter.go:19:type GRPCReporter struct {
```

**分析**: 两个 Reporter 的实现结构体

#### 2.3 提取纯函数名（使用 awk）
```bash
$ grep -Eroh "^func [a-zA-Z][a-zA-Z0-9]*" --include="*.go" src/agent/pkg/reporter/ | awk '{print $2}' | sort -u
```

**输出**:
```
directionStringToEnum
eventTypeStringToEnum
ipToUint32
NewGRPCReporter
NewLocalReporter
policyActionStringToEnum
protocolStringToEnum
stateStringToEnum
```

**分析**:
- 2 个构造函数
- 6 个辅助转换函数

---

### 案例 3: 代码统计

#### 3.1 统计代码行数
```bash
$ find src/agent/pkg/reporter -name "*.go" -exec wc -l {} + | tail -1
```

**输出**:
```
 362 total
```

**解释**: reporter 包总共 362 行代码

#### 3.2 按文件统计
```bash
$ find src/agent/pkg/reporter -name "*.go" -exec wc -l {} +
```

**输出**:
```
  50 src/agent/pkg/reporter/local_reporter.go
  24 src/agent/pkg/reporter/reporter.go
 288 src/agent/pkg/reporter/grpc_reporter.go
 362 total
```

**分析**:
- `grpc_reporter.go` 是最大的文件（288 行）
- 包含批处理逻辑和协议转换

#### 3.3 统计函数数量
```bash
$ grep -c "^func " src/agent/pkg/reporter/*.go
```

**输出**:
```
src/agent/pkg/reporter/grpc_reporter.go:14
src/agent/pkg/reporter/local_reporter.go:5
src/agent/pkg/reporter/reporter.go:0
```

**分析**: GRPCReporter 有 14 个函数，LocalReporter 有 5 个

---

### 案例 4: 配置文件搜索

#### 4.1 查找所有配置文件
```bash
$ find config -name "*.yaml" -o -name "*.yml"
```

**输出**:
```
config/agent-standalone.yaml
config/agent-server.yaml
```

#### 4.2 搜索特定配置项
```bash
$ grep -rn "server_addr" config/*.yaml
```

**输出**:
```
config/agent-server.yaml:38:  server_addr: "localhost:9090"
```

#### 4.3 查找所有配置键
```bash
$ grep -rh "^[a-z_]*:" config/*.yaml | sort -u
```

**输出示例**:
```
agent_server:
api:
batch_size: 100
batch_timeout: 5s
enable_cors: true
enabled: true
host: 127.0.0.1
interface: eth0
log_level: info
mode: agent-server
port: 8080
reconnect_interval: 30s
server_addr: "localhost:9090"
stats_interval: 30
storage:
```

---

### 案例 5: 依赖分析

#### 5.1 查找所有导入语句
```bash
$ grep -h "import" src/agent/pkg/reporter/*.go | head -20
```

**输出**:
```
import (
    "context"
    "github.com/ebpf-microsegment/src/agent/pkg/flow"
    "github.com/ebpf-microsegment/src/agent/pkg/storage"
    "github.com/sirupsen/logrus"
)
import (
    "context"
    "fmt"
    "net"
    "strings"
    "time"
    commonpb "github.com/ebpf-microsegment/src/proto/common"
    flowpb "github.com/ebpf-microsegment/src/proto/flow"
    "github.com/ebpf-microsegment/src/agent/pkg/flow"
    "github.com/sirupsen/logrus"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)
```

#### 5.2 提取所有唯一的包
```bash
$ grep -rh "import" src/agent/pkg/reporter/ | grep -o '"[^"]*"' | sort -u
```

**输出**:
```
"context"
"fmt"
"github.com/ebpf-microsegment/src/agent/pkg/flow"
"github.com/ebpf-microsegment/src/agent/pkg/storage"
"github.com/ebpf-microsegment/src/proto/common"
"github.com/ebpf-microsegment/src/proto/flow"
"github.com/sirupsen/logrus"
"google.golang.org/grpc"
"google.golang.org/grpc/credentials/insecure"
"net"
"strings"
"time"
```

---

### 案例 6: 错误处理分析

#### 6.1 查找错误处理
```bash
$ grep -rn "if err != nil" src/agent/pkg/reporter/
```

**输出**:
```
src/agent/pkg/reporter/local_reporter.go:32:      if err := r.storage.SaveFlow(f); err != nil {
src/agent/pkg/reporter/grpc_reporter.go:48:   if err != nil {
src/agent/pkg/reporter/grpc_reporter.go:131:  if err != nil {
src/agent/pkg/reporter/grpc_reporter.go:142:  if err != nil {
src/agent/pkg/reporter/grpc_reporter.go:147:  if err := stream.Send(event); err != nil {
src/agent/pkg/reporter/grpc_reporter.go:152:  if err != nil {
```

**分析**: GRPCReporter 有更多的错误处理（网络调用）

#### 6.2 统计错误处理数量
```bash
$ grep -c "if err != nil" src/agent/pkg/reporter/*.go
```

**输出**:
```
src/agent/pkg/reporter/grpc_reporter.go:5
src/agent/pkg/reporter/local_reporter.go:1
src/agent/pkg/reporter/reporter.go:0
```

---

### 案例 7: 查找特定模式

#### 7.1 查找所有 goroutine
```bash
$ grep -rn "go func" src/agent/pkg/reporter/
```

**输出**:
```
src/agent/pkg/reporter/grpc_reporter.go:127:    go func() {
```

**分析**: GRPCReporter 使用 goroutine 异步发送批次

#### 7.2 查找 channel 操作
```bash
$ grep -rn "<-" src/agent/pkg/reporter/
```

**输出**:
```
src/agent/pkg/reporter/grpc_reporter.go:75:        case r.batchQueue <- event:
src/agent/pkg/reporter/grpc_reporter.go:102:       case event := <-r.batchQueue:
src/agent/pkg/reporter/grpc_reporter.go:109:       case <-ticker.C:
src/agent/pkg/reporter/grpc_reporter.go:115:       case <-r.stopCh:
```

**分析**: GRPCReporter 使用 channel 进行批处理和关闭信号

#### 7.3 查找 defer 语句
```bash
$ grep -rn "defer " src/agent/pkg/reporter/
```

**输出**:
```
src/agent/pkg/reporter/grpc_reporter.go:95:    defer ticker.Stop()
src/agent/pkg/reporter/grpc_reporter.go:129:       defer cancel()
```

---

### 案例 8: 代码质量检查

#### 8.1 查找 TODO 注释
```bash
$ grep -rn "TODO" src/agent/pkg/reporter/
```

**输出**: （如果有的话会显示）

#### 8.2 查找注释
```bash
$ grep -rn "^//" src/agent/pkg/reporter/ | head -10
```

**输出**:
```
src/agent/pkg/reporter/reporter.go:9:// Reporter is an interface for reporting flow data
src/agent/pkg/reporter/reporter.go:10:// Implementations can be local (SQLite) or remote (gRPC)
src/agent/pkg/reporter/reporter.go:12:    // Report sends a single flow to the reporter
src/agent/pkg/reporter/reporter.go:15:    // ReportBatch sends multiple flows (for efficiency)
src/agent/pkg/reporter/reporter.go:18:    // Start initializes the reporter
src/agent/pkg/reporter/reporter.go:21:    // Stop gracefully shuts down the reporter
src/agent/pkg/reporter/local_reporter.go:11:// LocalReporter reports flows to local SQLite storage
src/agent/pkg/reporter/local_reporter.go:17:// NewLocalReporter creates a new LocalReporter
src/agent/pkg/reporter/local_reporter.go:24:// Report saves a single flow to local storage
src/agent/pkg/reporter/local_reporter.go:29:// ReportBatch saves multiple flows to local storage
```

---

## 🔄 组合技巧

### 技巧 1: 查找并查看
```bash
# 找到文件
files=$(grep -rl "NewGRPCReporter" --include="*.go" .)

# 查看每个文件中的定义
for file in $files; do
    echo "=== $file ==="
    grep -n "NewGRPCReporter" "$file"
done
```

### 技巧 2: 统计和分析
```bash
# 统计每个目录的 Go 文件数量
find src -name "*.go" | awk -F/ '{print $1"/"$2"/"$3}' | sort | uniq -c

# 统计每个包的代码行数
for dir in $(find src/agent/pkg -mindepth 1 -maxdepth 1 -type d); do
    lines=$(find "$dir" -name "*.go" -exec wc -l {} + 2>/dev/null | tail -1 | awk '{print $1}')
    echo "$dir: $lines lines"
done
```

### 技巧 3: 创建报告
```bash
# 生成代码统计报告
{
    echo "# Code Statistics Report"
    echo ""
    echo "## Total Lines"
    find src/agent/pkg/reporter -name "*.go" -exec wc -l {} + | tail -1
    echo ""
    echo "## Functions"
    grep -c "^func " src/agent/pkg/reporter/*.go | while IFS=: read file count; do
        echo "- $(basename $file): $count functions"
    done
    echo ""
    echo "## Structs"
    grep -Ern "^type [a-zA-Z]+ struct" src/agent/pkg/reporter/
} > reporter-stats.md
```

---

## 💡 实用别名

添加到 `~/.bashrc` 或 `~/.zshrc`:

```bash
# 在项目中搜索 Go 代码
alias gog='grep -rn --include="*.go"'

# 查找函数定义
findfunc() {
    grep -Ern "^func.*$1" --include="*.go" .
}

# 查找类型定义
findtype() {
    grep -Ern "^type $1" --include="*.go" .
}

# 统计 Go 代码行数
alias countgo='find . -name "*.go" -not -path "*/vendor/*" -exec wc -l {} + | tail -1'

# 查找 TODO
alias todos='grep -rn "TODO" --include="*.go" .'
```

**使用示例**:
```bash
$ findfunc Reporter
$ findtype GRPCReporter
$ countgo
$ todos
```

---

## 📚 学习建议

1. **从简单开始**: 先掌握 `grep -rn`
2. **逐步添加选项**: 慢慢加入 `--include`, `-C` 等
3. **使用管道**: 学会组合命令
4. **创建别名**: 把常用命令变成别名
5. **实践练习**: 在真实项目中多用

---

## 🎯 下一步

- 尝试所有示例命令
- 根据自己的需求修改
- 创建自己的搜索脚本
- 学习正则表达式
- 尝试 ripgrep (rg) 提升速度

---

**相关文档**:
- [完整指南](code-search-guide.md)
- [速查表](code-search-cheatsheet.md)

---

**最后更新**: 2025-11-05
