# VSCode 调试指南

本文档说明如何在 VSCode 中调试 eBPF Microsegmentation 项目。

## 📋 前置要求

### 1. 安装必需的 VSCode 扩展

打开 VSCode,按 `Ctrl+Shift+X` 打开扩展面板,搜索并安装:

- **Go** (`golang.go`) - Go 语言支持
- **C/C++** (`ms-vscode.cpptools`) - C/C++ 语言支持 (用于 eBPF 代码)
- **vscode-proto3** (`zxh404.vscode-proto3`) - Protocol Buffers 语法高亮
- **YAML** (`redhat.vscode-yaml`) - YAML 配置文件支持

或者在 VSCode 中按 `Ctrl+Shift+P`,输入 "Extensions: Show Recommended Extensions" 安装推荐的扩展。

### 2. 安装 Go 调试工具 Delve

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

### 3. 确保有 root 权限

eBPF 程序需要 root 权限运行,因此 VSCode 需要以 root 身份启动调试器。

## 🚀 调试配置说明

项目提供了以下调试配置 (在 `.vscode/launch.json` 中):

### 1. Debug Agent (Standalone Mode)
调试 TC 模式的 Agent (使用 `config/agent-tc.yaml`)

**使用方法**:
1. 按 `F5` 或选择 "Run and Debug" 面板
2. 选择 "Debug Agent (Standalone Mode)"
3. 点击绿色播放按钮开始调试

### 2. Debug Agent (XDP Mode)
调试 XDP 模式的 Agent (使用 `config/agent-xdp.yaml`)

**使用方法**:
1. 选择 "Debug Agent (XDP Mode)"
2. 点击播放按钮开始调试

### 3. Debug Agent (Custom Config)
使用自定义配置文件调试 Agent

**使用方法**:
1. 选择 "Debug Agent (Custom Config)"
2. 输入配置文件路径 (如 `/path/to/custom-config.yaml`)
3. 开始调试

### 4. Debug Server
调试控制平面 Server (使用 `config/server.yaml`)

**使用方法**:
1. 选择 "Debug Server"
2. 开始调试

### 5. Attach to Running Agent
附加到已运行的 Agent 进程进行调试

**使用方法**:
1. 先手动启动 Agent: `sudo ./bin/microsegment-agent -c config/agent-tc.yaml`
2. 选择 "Attach to Running Agent"
3. 从进程列表中选择 `microsegment-agent` 进程

## 🔧 调试步骤示例

### 调试 Agent 启动流程

1. **设置断点**:
   - 打开 `src/agent/cmd/main.go`
   - 在第 56 行 (`dp, err := dataplane.New(cfg.Interface)`) 设置断点
   - 点击行号左侧即可设置红色断点

2. **启动调试**:
   - 按 `F5` 或选择调试配置 "Debug Agent (Standalone Mode)"
   - 程序会自动编译 eBPF 代码 (通过 `preLaunchTask: build-bpf`)
   - 程序在断点处暂停

3. **单步调试**:
   - `F10` - Step Over (单步跳过)
   - `F11` - Step Into (单步进入函数)
   - `Shift+F11` - Step Out (跳出当前函数)
   - `F5` - Continue (继续执行)

4. **查看变量**:
   - 左侧 "Variables" 面板显示所有局部变量和全局变量
   - 悬停在变量名上查看值
   - 在 "Watch" 面板添加表达式监视

### 调试数据平面逻辑

1. **设置断点**:
   ```
   src/agent/pkg/dataplane/dataplane.go:123 (New 函数)
   src/agent/pkg/dataplane/tc.go:45 (AttachTC 函数)
   src/agent/pkg/dataplane/xdp.go:45 (AttachXDP 函数)
   ```

2. **查看 eBPF Map**:
   - 在 `dataplane.go` 中的 map 操作处设置断点
   - 查看 `objs.PolicyMap`, `objs.StatsMap` 等变量
   - 使用 Debug Console 执行命令查看 map 内容

### 调试策略加载

1. **设置断点**:
   ```
   src/agent/pkg/policy/manager.go:58 (AddPolicy 函数)
   src/agent/pkg/policy/manager.go:150 (loadPolicyToDataPlane 函数)
   ```

2. **单步执行**:
   - 观察策略如何转换为 eBPF map 条目
   - 查看 `policy_key` 和 `policy_value` 结构体的值

## 📊 调试技巧

### 1. 使用 Debug Console

在调试过程中,可以在 "Debug Console" 中执行 Go 表达式:

```go
// 查看变量
p cfg.Interface

// 调用函数
call dp.GetStats()

// 查看结构体字段
p objs.SessionMap.MaxEntries
```

### 2. 条件断点

右键点击断点,选择 "Edit Breakpoint",设置条件:

```go
// 仅当错误发生时暂停
err != nil

// 仅当特定端口时暂停
key.DstPort == 80
```

### 3. 日志点 (Logpoint)

右键点击行号,选择 "Add Logpoint",输入:

```
策略 {policy.RuleID} 已加载: {policy.SrcIP} -> {policy.DstIP}
```

程序执行到此处时会输出日志,但不会暂停。

### 4. 查看 eBPF 程序状态

在 Debug Console 中执行系统命令:

```bash
# 查看加载的 eBPF 程序
!sudo bpftool prog list

# 查看 eBPF map
!sudo bpftool map list

# 查看统计数据
!sudo bpftool map dump name stats_map
```

## ⚙️ 构建任务

项目提供了以下构建任务 (在 `.vscode/tasks.json` 中):

### 运行构建任务

按 `Ctrl+Shift+B` 或 `Ctrl+Shift+P` -> "Tasks: Run Build Task"

- **build-bpf**: 编译 eBPF 代码
- **build-agent**: 编译 Agent (自动先编译 eBPF)
- **build-server**: 编译 Server
- **build-all**: 编译所有组件
- **test-agent**: 运行单元测试
- **clean**: 清理编译产物

### 自动化工作流

调试配置已配置 `preLaunchTask: build-bpf`,每次启动调试时会自动编译 eBPF 代码,确保代码最新。

## 🐛 常见问题

### 1. "permission denied" 错误

**原因**: eBPF 需要 root 权限

**解决**:
- 确保调试配置中 `"asRoot": true` 已设置
- 或手动以 root 启动 VSCode: `sudo code --user-data-dir=/root/.vscode`

### 2. "could not launch process: not an executable file"

**原因**: 程序未编译或编译失败

**解决**:
1. 运行构建任务: `Ctrl+Shift+B` -> "build-agent"
2. 检查编译错误
3. 确保 `bin/microsegment-agent` 存在

### 3. 断点变灰色 (未激活)

**原因**: 编译时可能优化了代码

**解决**:
- 确保使用 debug 模式编译 (`mode: "debug"` 已在 launch.json 中设置)
- 或在代码中添加 `runtime.Breakpoint()` 强制断点

### 4. "CGO_ENABLED must be set to 1"

**原因**: eBPF 需要 CGO 支持

**解决**:
- 已在 launch.json 中设置 `"env": {"CGO_ENABLED": "1"}`
- 或手动设置环境变量: `export CGO_ENABLED=1`

### 5. 无法查看某些变量

**原因**: Delve 默认限制变量深度和大小

**解决**:
- 已在 settings.json 中配置 `go.delveConfig.dlvLoadConfig`
- 或在 Debug Console 中使用 `dlv` 命令手动查看:
  ```
  dlv config substitutePath
  ```

## 📚 进阶调试

### 远程调试

如果需要在远程机器上调试:

1. **在远程机器启动 Delve 服务器**:
   ```bash
   sudo dlv exec ./bin/microsegment-agent \
     --headless \
     --listen=:2345 \
     --api-version=2 \
     --accept-multiclient \
     -- -c config/agent-tc.yaml
   ```

2. **在本地 VSCode 添加远程调试配置**:
   ```json
   {
       "name": "Remote Debug Agent",
       "type": "go",
       "request": "attach",
       "mode": "remote",
       "remotePath": "${workspaceFolder}",
       "port": 2345,
       "host": "192.168.1.100"
   }
   ```

### 性能分析

在调试配置中添加 profiling:

```json
{
    "name": "Debug Agent with Profiling",
    "type": "go",
    "request": "launch",
    "mode": "debug",
    "program": "${workspaceFolder}/src/agent/cmd",
    "buildFlags": "-race",  // 启用竞态检测
    "env": {
        "GODEBUG": "gctrace=1"  // 启用 GC 跟踪
    }
}
```

## 🎯 推荐工作流

1. **开发新功能时**:
   - 编写代码
   - 设置断点
   - `F5` 启动调试
   - 单步执行验证逻辑
   - 修改代码后按 `Shift+F5` 停止,然后 `F5` 重新启动

2. **排查 Bug 时**:
   - 复现问题
   - 在可疑位置设置断点
   - 使用条件断点缩小范围
   - 查看变量和调用堆栈
   - 使用 Logpoint 记录执行路径

3. **性能优化时**:
   - 使用 profiling 工具
   - 在热点代码设置断点
   - 检查数据结构大小和分配
   - 使用 `go test -bench` 进行基准测试

## 📖 相关资源

- [VSCode Go 调试文档](https://github.com/golang/vscode-go/blob/master/docs/debugging.md)
- [Delve 调试器文档](https://github.com/go-delve/delve/tree/master/Documentation)
- [eBPF 开发指南](https://ebpf.io/what-is-ebpf/)

---

**提示**: 本调试配置已针对 eBPF 项目优化,包含自动编译、root 权限、CGO 支持等必要设置。祝调试顺利! 🎉
