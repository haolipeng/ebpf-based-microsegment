[上级索引](../CLAUDE.md) > **dataplane**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# dataplane

## 架构定位

eBPF 程序生命周期管理器 | 输入: 网卡名称、eBPF 程序路径、模式配置（TC/XDP） | 输出: 加载的 eBPF 程序、Pin 的 Maps（policy_map、session_map、stats_map 等）

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| manager.go | 数据平面管理器，选择和管理 TC/XDP 加载器 | `NewManager()`, `Load()`, `Unload()`, `GetMaps()` |
| tc_loader.go | TC（Traffic Control）方式加载 eBPF 程序 | `NewTCLoader()`, `Load()`, `AttachIngress()`, `AttachEgress()` |
| xdp_loader.go | XDP（eXpress Data Path）方式加载 eBPF 程序 | `NewXDPLoader()`, `AttachXDP()` |
| xdp_test_loader.go | XDP 测试模式加载器，使用 BPF_PROG_TEST_RUN | `NewXDPTestLoader()`, `RunTest()` |
| mode_detect.go | 检测系统支持的数据平面模式（TCX/Legacy TC/XDP） | `DetectCapabilities()`, `SelectMode()` |
| capability.go | 系统能力检测（内核版本、eBPF 特性） | `CheckCapabilities()` |
| map_pinning.go | eBPF Map Pin/Unpin 到文件系统 | `PinMaps()`, `UnpinMaps()`, `LoadPinnedMap()` |
| interface.go | 网卡操作接口（查询、配置） | `GetInterfaceIndex()`, `SetupInterface()` |
| nat.go | NAT 相关逻辑（需要 conntrack 支持） | `EnableNAT()`, `DisableNAT()` |
| fragment.go | IP 分片处理逻辑 | `EnableFragmentHandling()` |
| bpf_x86_bpfel.go | Cilium eBPF 生成的 TC eBPF 程序加载代码 | `loadBpf()`, `bpfObjects` |
| xdpbpf_x86_bpfel.go | XDP eBPF 程序加载代码 | `loadXdpbpf()` |
| processbpf_x86_bpfel.go | 进程监控 eBPF 程序加载代码 | `loadProcessbpf()` |
| bpftest/packet_builder.go | eBPF 测试数据包构建工具 | `BuildTCPPacket()`, `BuildUDPPacket()` |
| bpftest/context_builder.go | eBPF 测试上下文构建 | `BuildTestContext()` |
| bpftest/runner.go | eBPF 单元测试运行器 | `RunBPFTest()`, `ValidateResult()` |

## 核心功能

- **多模式支持**: TC（TCX/Legacy）、XDP、测试模式
- **自动检测**: 根据内核能力自动选择最优模式
- **Map 管理**: Pin Maps 到 /sys/fs/bpf/ 用于持久化和跨程序共享
- **热重载**: 支持不丢流的程序更新
- **测试框架**: 使用 BPF_PROG_TEST_RUN 进行单元测试

## eBPF Maps

| Map 名称 | 类型 | 用途 |
|---------|------|------|
| policy_map | Hash | 精确匹配策略（五元组 -> 动作） |
| wildcard_policy_map | Hash | 通配符策略（CIDR/端口范围） |
| session_map | LRU Hash | 会话跟踪（五元组 -> 会话状态） |
| stats_map | Per-CPU Array | 统计数据（包数、字节数、丢包数） |
| conntrack_cache_map | LRU Hash | NAT 映射缓存 |
| fragment_state_map | Hash | IP 分片状态跟踪 |
