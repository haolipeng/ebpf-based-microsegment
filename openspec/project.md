# 项目上下文

## Purpose
面向容器/虚拟机环境的基于 eBPF 的网络微隔离系统。该项目实现了一个 TC (Traffic Control) eBPF 程序，用于内核级数据包过滤和策略执行，提供高性能的网络安全性，开销最小。

**目标：**
- 构建生产就绪的 eBPF 微隔离解决方案
- 通过为期 6 周的实际实施学习 eBPF 编程
- 实现数据包处理 <10μs 的延迟开销
- 支持 100,000+ 并发会话

## Tech Stack

### 核心技术
- **eBPF (Extended Berkeley Packet Filter)** - 内核级数据包处理
- **libbpf 1.x** - 现代 eBPF 加载器和骨架框架
- **C** - eBPF 程序和用户空间控制平面
- **TC (Traffic Control)** - Linux 流量控制子系统（ingress/egress hooks）
- **Clang/LLVM** - eBPF 程序编译
- **bpftool** - eBPF 内省和调试

### 支持工具
- **Go**（可选）- 用户空间控制程序的替代方案
- **Prometheus** - 指标收集和监控
- **Grafana** - 可视化仪表板
- **Python** - 测试和自动化脚本

### 开发环境
- Linux Kernel ≥ 5.10（用于 BTF 和 CO-RE 支持）
- Ubuntu 22.04+ 或同等版本
- VSCode 与 C/C++ 扩展

## 项目约定

### 代码风格

**eBPF 程序 (C)：**
- 文件命名：`*.bpf.c` 用于 eBPF 代码，`*.c` 用于用户空间
- 使用 `SEC("tc")` 表示 TC 程序
- 始终包含边界检查：`if ((void *)(hdr + 1) > data_end)`
- 优先使用内联函数实现代码复用（对 eBPF 验证器友好）
- 注释 eBPF 辅助函数的使用

**用户空间程序 (C)：**
- 使用 libbpf skeleton：从 `xxx.bpf.o` 自动生成 `xxx.skel.h`
- 加载器模式：`xxx_loader.c` 用于程序生命周期管理
- 信号处理以实现优雅关闭（SIGINT/SIGTERM）
- 始终检查错误并提供有意义的消息

**命名约定：**
- Maps：`policy_map`、`session_map`、`stats_map`（snake_case）
- Structs：`struct policy_key`、`struct tcp_state`（snake_case）
- Functions：`track_session()`、`match_policy()`（snake_case，动词引导）
- Constants：`MAX_ENTRIES`、`TC_ACT_OK`（UPPER_CASE）

### 架构模式

**eBPF 程序结构：**
```
src/bpf/          # eBPF 内核程序
  ├── hello.bpf.c         # TC 程序
  ├── parse_packet.bpf.c
  └── microsegment.bpf.c

src/user/         # 用户空间加载器
  ├── hello_loader.c
  └── microsegment_loader.c
```

**关键模式：**
- **基于骨架的加载**：使用 `bpf_object__open_and_load()` 自动加载 map/程序
- **TC 附加**：使用 `bpf_tc_hook_create()` + `bpf_tc_attach()` 实现 libbpf TC API
- **Map pinning**：将 maps 固定到 `/sys/fs/bpf/` 以实现跨进程访问
- **优雅清理**：使用信号处理器 + `bpf_tc_detach()` + `bpf_tc_hook_destroy()`

**数据流：**
```
Packet → TC Ingress Hook → eBPF Program → Policy Check → Session Tracking → Action (PASS/DROP)
```

### 测试策略

**单元测试：**
- 使用模拟数据测试单个 eBPF 辅助函数
- 验证 map 操作（插入、查找、删除）
- 测试边界条件和边缘情况

**功能测试：**
- 端到端数据包过滤场景
- 通过 CLI 进行策略 CRUD 操作
- 会话跟踪准确性（TCP/UDP）
- 多协议支持（IPv4/IPv6）

**性能测试：**
- 吞吐量测量（iperf3）
- 延迟测量（ping RTT）
- 连接容量（100k+ 会话）
- CPU 开销监控

**压力测试：**
- 高数据包速率（>1M pps）
- 连接波动（快速打开/关闭）
- Map 压力（接近 max_entries）

**测试命令：**
```bash
# 加载程序
sudo ./xxx_loader lo

# 运行测试
./tests/integration_test.sh
./tests/performance_test.sh

# 使用 bpftool 验证
sudo bpftool prog show
sudo bpftool map dump name policy_map
```

### Git 工作流

**分支：**
- `master` - 主开发分支
- 功能分支：`feature/add-tcp-state-machine`
- 没有单独的 `main` 分支（使用 `master`）

**提交约定：**
- 使用描述性提交消息
- 生成的提交包含 Claude Code 署名：
  ```
  Add TCP state machine tracking

  🤖 Generated with Claude Code
  Co-Authored-By: Claude <noreply@anthropic.com>
  ```

**Pull Request 流程：**
- 目前未强制执行（学习项目）
- 未来：对于生产部署需要 PR 批准

## 领域上下文

### eBPF 基础
- **Verifier**：eBPF 程序必须通过内核验证器（安全检查）
- **BPF Maps**：用于内核和用户空间之间状态共享的键值存储
- **TC Hooks**：流量控制的附加点（ingress/egress）
- **CO-RE (Compile Once, Run Everywhere)**：基于 BTF 的跨内核版本可移植性
- **Tail Calls**：跳转到另一个 eBPF 程序（用于程序分解）

### 网络概念
- **5-tuple**：(src_ip, dst_ip, src_port, dst_port, protocol) - 流标识符
- **Connection tracking**：跟踪 TCP/UDP 会话状态
- **Microsegmentation**：按工作负载进行细粒度的网络策略执行
- **TC return codes**：`TC_ACT_OK`（通过）、`TC_ACT_SHOT`（丢弃）、`TC_ACT_REDIRECT`

### 关键 eBPF 要求
- **边界检查**：必须(MUST)在访问数据包数据之前检查 `data_end`
- **验证器复杂性限制**：程序必须足够简单以供验证器处理
- **Map 大小限制**：根据预期规模规划 `max_entries`
- **栈限制**：每个 eBPF 程序 512 字节

## 重要约束

### 技术约束
- **内核版本**：需要 Linux ≥5.10 以支持 BTF 和现代 libbpf 功能
- **eBPF 复杂性**：程序必须通过验证器（无无界循环，栈有限）
- **性能**：目标是每个数据包 <10μs 的延迟开销
- **内存**：Map 条目消耗内核内存（仔细规划容量）

### 开发约束
- **以学习为重点**：代码应具有教育意义，最初不是为生产强化
- **渐进式**：在 6 周内逐步构建复杂性
- **文档**：为 eBPF 初学者解释所有概念

### 运营约束
- **优雅降级**：程序失败不应导致系统崩溃
- **可观察性**：必须通过 bpftool、Prometheus、日志提供可见性
- **热重载**：策略更新无需重新加载 eBPF 程序

## 外部依赖

### 必需的系统库
- **libbpf**（≥1.0）- eBPF 加载和骨架生成
- **libelf** - ELF 文件解析
- **zlib** - BTF 的压缩支持

### 开发工具
- **clang**（≥10）- 将 eBPF 程序编译为 BPF 字节码
- **llvm** - 用于 eBPF 的 LLVM 工具链
- **bpftool** - 检查和管理 eBPF 对象
- **tc** - Linux 流量控制实用程序

### 可选依赖
- **Prometheus** - 指标收集（第 6 周）
- **Grafana** - 监控仪表板（第 6 周）
- **iperf3** - 性能测试
- **netcat/curl** - 功能测试

### 参考项目
- **libbpf-bootstrap** - 现代 libbpf 开发的官方示例
- **ZFW (Zero Trust Firewall)** - 在 `docs/zfw-architecture-analysis.md` 中分析的参考架构
- **Cilium** - 生产级 eBPF 网络（灵感来源）

## 学习资源

### 主要文档
- `docs/weekly-guide/` - 6 周结构化学习计划
- `docs/zfw-architecture-analysis.md` - 生产级 eBPF 防火墙的深入分析
- `specs/ebpf-tc-implementation.md` - 实施规范

### 关键 eBPF 参考
- libbpf 文档：https://libbpf.readthedocs.io/
- Linux 内核 BPF 文档：https://www.kernel.org/doc/html/latest/bpf/
- Cilium BPF 参考指南：https://docs.cilium.io/en/stable/bpf/
