# 构建指南

本文档介绍如何编译和运行 eBPF 微隔离项目。

## 📋 前置要求

### 系统要求
- Linux Kernel ≥ 5.10（支持 BTF 和 eBPF）
- Ubuntu 22.04+ 或其他 Linux 发行版
- 至少 4GB RAM
- root 权限（用于加载 eBPF 程序）

### 开发工具
```bash
# 安装必要工具（Ubuntu/Debian）
sudo apt-get update
sudo apt-get install -y \
    clang \
    llvm \
    libbpf-dev \
    linux-headers-$(uname -r) \
    build-essential \
    golang-1.21 \
    git

# 验证安装
clang --version  # >= 11
go version       # >= 1.21
```

## 🚀 快速开始

### 1. 下载依赖

```bash
# 下载 Go 模块依赖
make deps

# 或手动执行
go mod download
go mod tidy
```

### 2. 生成 eBPF Go 绑定

```bash
# 使用 bpf2go 编译 C 代码并生成 Go 绑定
make bpf
```

**这一步做了什么？**
- 将 `src/bpf/tc_microsegment.bpf.c` 编译为 eBPF 字节码（.o 文件）
- 自动生成 Go 绑定代码（bpf_*.go）
- 嵌入字节码到 Go 包中

**生成的文件位置：**
```
src/agent/pkg/dataplane/
├── bpf_bpfel_x86.go          # x86_64 架构的 Go 绑定
├── bpf_bpfel_x86.o           # x86_64 eBPF 字节码
├── bpf_bpfel_arm64.go        # arm64 架构的 Go 绑定（如果生成）
└── bpf_bpfel_arm64.o         # arm64 eBPF 字节码（如果生成）
```

### 3. 编译 Agent

```bash
# 编译用户态 Go 程序
make agent
```

**输出：**
```
bin/microsegment-agent         # 可执行文件
```

### 4. 运行 Agent

```bash
# 在 loopback 接口运行（测试）
make run

# 或手动运行
sudo ./bin/microsegment-agent --interface lo --log-level debug

# 在生产接口运行
sudo ./bin/microsegment-agent --interface eth0 --log-level info
```

## 🛠️ Makefile 命令参考

### 构建命令

| 命令 | 说明 |
|------|------|
| `make all` | 构建所有（bpf + agent），默认目标 |
| `make bpf` | 生成 eBPF Go 绑定 |
| `make agent` | 编译 Agent 二进制文件 |
| `make clean` | 清理所有构建产物 |

### 依赖管理

| 命令 | 说明 |
|------|------|
| `make deps` | 下载 Go 依赖 |

### 代码质量

| 命令 | 说明 |
|------|------|
| `make fmt` | 格式化 Go 代码 |
| `make lint` | 运行代码检查器（需要 golangci-lint） |

### 测试命令

| 命令 | 说明 |
|------|------|
| `make test` | 运行单元测试 |
| `make test-integration` | 运行集成测试（需要 sudo） |

### 运行和安装

| 命令 | 说明 |
|------|------|
| `make run` | 运行 Agent（需要 sudo） |
| `make install` | 安装到 /usr/local/bin（需要 sudo） |

### 帮助

| 命令 | 说明 |
|------|------|
| `make help` | 显示所有可用命令 |

## 📝 完整构建流程

```bash
# 1. 克隆项目
git clone https://github.com/haolipeng/ebpf-based-microsegment.git
cd ebpf-based-microsegment

# 2. 安装系统依赖
sudo apt-get install -y clang llvm libbpf-dev linux-headers-$(uname -r) golang-1.21

# 3. 下载 Go 依赖
make deps

# 4. 生成 eBPF 绑定
make bpf

# 5. 编译 Agent
make agent

# 6. 运行（需要 root）
sudo ./bin/microsegment-agent --interface lo --log-level debug
```

## 🔧 常见问题

### 1. `make bpf` 失败：找不到 bpf2go

**错误信息：**
```
go: github.com/cilium/ebpf/cmd/bpf2go: no such file or directory
```

**解决方案：**
```bash
# 安装 bpf2go 工具
go install github.com/cilium/ebpf/cmd/bpf2go@latest

# 确保 $GOPATH/bin 在 PATH 中
export PATH=$PATH:$(go env GOPATH)/bin
```

### 2. `make bpf` 失败：找不到 vmlinux.h

**错误信息：**
```
fatal error: 'vmlinux.h' file not found
```

**解决方案：**
```bash
# 生成 vmlinux.h（BTF 类型信息）
sudo bpftool btf dump file /sys/kernel/btf/vmlinux format c > vmlinux/x86/vmlinux.h

# 或使用现有的（项目已包含）
ls vmlinux/x86/vmlinux.h
```

### 3. 运行 Agent 失败：权限拒绝

**错误信息：**
```
Failed to attach TC program: operation not permitted
```

**解决方案：**
```bash
# 必须使用 sudo 运行
sudo ./bin/microsegment-agent --interface lo
```

### 4. 编译 eBPF 程序失败：内核头文件缺失

**错误信息：**
```
fatal error: linux/bpf.h: No such file or directory
```

**解决方案：**
```bash
# 安装内核头文件
sudo apt-get install linux-headers-$(uname -r)
```

### 5. Go 版本过低

**错误信息：**
```
go.mod requires go >= 1.21
```

**解决方案：**
```bash
# 安装更新版本的 Go
sudo add-apt-repository ppa:longsleep/golang-backports
sudo apt-get update
sudo apt-get install golang-1.21

# 或从官网下载：https://go.dev/dl/
```

## 🧪 测试和验证

### 验证 eBPF 程序已加载

```bash
# 查看已加载的 eBPF 程序
sudo bpftool prog list | grep tc_microsegment

# 查看 Map
sudo bpftool map list
```

### 查看 eBPF 日志

```bash
# 实时查看内核日志（bpf_printk 输出）
sudo cat /sys/kernel/debug/tracing/trace_pipe
```

### 生成测试流量

```bash
# 终端1：运行 Agent
sudo ./bin/microsegment-agent --interface lo --log-level debug

# 终端2：生成流量
ping 127.0.0.1
curl http://127.0.0.1:8080

# 终端3：查看日志
sudo cat /sys/kernel/debug/tracing/trace_pipe
```

## 📦 交叉编译

如果需要为不同架构编译：

```bash
# 为 ARM64 编译
GOARCH=arm64 make agent

# 为 x86_64 编译
GOARCH=amd64 make agent
```

## 🐳 Docker 构建（未来）

```bash
# 使用 Docker 构建（避免环境依赖）
docker build -t microsegment-agent .
docker run --privileged --net=host microsegment-agent
```

## 📚 参考资料

- [Cilium eBPF 文档](https://pkg.go.dev/github.com/cilium/ebpf)
- [bpf2go 使用指南](https://pkg.go.dev/github.com/cilium/ebpf/cmd/bpf2go)
- [eBPF 官方文档](https://ebpf.io/)
- [项目 README](../README.md)

---

*最后更新：2025-10-30*

