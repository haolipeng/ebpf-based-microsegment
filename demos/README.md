# eBPF 微隔离学习 Demo

本目录包含 8 个渐进式 eBPF Demo,通过动手实践帮助新人理解项目的核心技术栈。

## 🎯 学习目标

通过完成这些 Demo,你将掌握:
- eBPF TC Hook 数据包过滤
- 5-tuple 流键提取和数据包解析
- Hash/LRU Map 策略匹配和会话跟踪
- TCP 状态机和连接生命周期管理
- Ring Buffer 事件上报机制
- Cilium eBPF Go 库和用户态控制
- 生产级微隔离数据平面架构

## 📚 学习路径

| Demo | 学习时间 | 难度 | 核心技术 |
|------|---------|------|---------|
| [01-hello-packet](./01-hello-packet/) | 2-3h | ⭐☆☆☆☆ | TC Hook, ARRAY Map, 包计数 |
| [02-5tuple-extractor](./02-5tuple-extractor/) | 4-6h | ⭐⭐☆☆☆ | 数据包解析, 边界检查, bpf_printk |
| [03-hash-policy](./03-hash-policy/) | 4-6h | ⭐⭐☆☆☆ | HASH Map, 策略匹配, Go 控制 |
| [04-lru-session](./04-lru-session/) | 5-7h | ⭐⭐⭐☆☆ | LRU_HASH, 热路径优化, 会话跟踪 |
| [05-tcp-state](./05-tcp-state/) | 6-8h | ⭐⭐⭐☆☆ | TCP 状态机, Ring Buffer, 事件监听 |
| [06-wildcard-policy](./06-wildcard-policy/) | 5-7h | ⭐⭐⭐☆☆ | CIDR 匹配, 端口范围, 优先级 |
| [07-userspace-controller](./07-userspace-controller/) | 8-10h | ⭐⭐⭐⭐☆ | Cilium eBPF, TC 管理, Per-CPU 统计 |
| [08-full-pipeline](./08-full-pipeline/) | 10-12h | ⭐⭐⭐⭐⭐ | 完整流水线, HTTP API, 模块化架构 |

**总学习时间**: 约 60 小时 (1-2 周全职学习)

## 🚀 快速开始

### 1. 环境准备

```bash
# 检查环境
cd demos/common/scripts
./setup_env.sh

# 需要的工具:
# - Linux Kernel 5.10+ (建议 6.x)
# - clang/llvm 14+
# - libbpf-dev
# - bpftool
# - Go 1.21+
```

### 2. 开始第一个 Demo

```bash
cd demos/01-hello-packet
make              # 编译 eBPF 程序
sudo make load    # 加载到内核
./test.sh         # 运行测试
sudo make unload  # 卸载程序
```

### 3. 查看 Demo 教程

每个 Demo 目录下都有详细的 README.md,包含:
- 学习目标和前置知识
- 核心概念讲解
- 代码逐行解读
- 运行和测试指南
- 常见问题和排错
- 下一步建议

## 📁 目录结构

```
demos/
├── README.md                 # 本文件 - 学习路径总览
├── common/                   # 共享组件
│   ├── headers/             # 简化的头文件 (复用自主项目)
│   └── scripts/             # 通用脚本
├── 01-hello-packet/         # Demo 1: 最简单的包计数器
├── 02-5tuple-extractor/     # Demo 2: 5-tuple 提取和打印
├── 03-hash-policy/          # Demo 3: Hash Map 策略匹配
├── 04-lru-session/          # Demo 4: LRU 会话跟踪
├── 05-tcp-state/            # Demo 5: TCP 状态机
├── 06-wildcard-policy/      # Demo 6: 通配符策略
├── 07-userspace-controller/ # Demo 7: 用户态控制器
└── 08-full-pipeline/        # Demo 8: 完整流水线
```

## 🔧 简化说明

为了降低学习难度,这些 Demo 做了以下简化:
- ✅ **仅支持 IPv4** (主项目同时支持 IPv6)
- ❌ **跳过 VLAN** (主项目支持 802.1Q/802.1ad)
- ❌ **跳过 IP 分片** (主项目有完整分片处理)
- ❌ **跳过 NAT 检测** (主项目支持 Conntrack 集成)
- ❌ **跳过进程级策略** (主项目支持进程名匹配)

简化后的代码更专注于核心概念,学完 Demo 后可以阅读主项目源码了解完整实现。

## 📖 学习建议

### 按顺序学习
Demo 设计为渐进式,每个 Demo 都会引入 1-2 个新概念,建议按顺序完成。

### 动手实践
不要只是阅读代码,一定要:
1. 亲自编译和运行
2. 修改参数观察效果
3. 使用 bpftool 查看 Maps
4. 阅读内核日志调试

### 参考主项目
完成 Demo 后,对比主项目源码:
- `src/bpf/tc_microsegment.bpf.c` - 完整 eBPF 程序
- `src/bpf/headers/` - 完整头文件库
- `src/agent/pkg/dataplane/` - 生产级用户态控制

### 遇到问题
1. 查看 Demo 的 README 常见问题部分
2. 使用 `dmesg` 查看内核错误
3. 使用 `bpftool prog show` 检查程序状态
4. 查看主项目文档 `docs/learning/weekly-guide/`

## 🎓 知识体系

### eBPF 基础
- **Hook 类型**: TC (流量控制), XDP (超高性能), Tracepoint (系统追踪)
- **返回值**: `TC_ACT_OK` (允许), `TC_ACT_SHOT` (丢弃)
- **边界检查**: `data_end` 验证 (eBPF 验证器要求)
- **辅助函数**: `bpf_map_lookup_elem`, `bpf_map_update_elem`, `bpf_printk`

### Map 类型
- **ARRAY**: 固定大小数组,索引快速访问
- **HASH**: 哈希表,O(1) 查找
- **LRU_HASH**: 自动淘汰最少使用的条目
- **RINGBUF**: 环形缓冲区,用于事件上报
- **PERCPU_ARRAY**: Per-CPU 数组,无锁统计

### 网络协议
- **以太网**: MAC 地址, EtherType, VLAN 标签
- **IPv4**: IP 地址, 协议号, 分片标志
- **TCP**: 源端口/目标端口, 标志位 (SYN/FIN/RST/ACK)
- **UDP**: 源端口/目标端口, 无状态

### 微隔离技术
- **5-tuple**: `(src_ip, dst_ip, src_port, dst_port, protocol)`
- **会话跟踪**: 双向流量统计,连接状态管理
- **策略匹配**: 精确匹配 (Hash) + 通配符匹配 (CIDR)
- **热路径优化**: 会话缓存 (<1μs) vs 策略查询 (5-20μs)

## 🔗 相关资源

### 主项目文档
- [项目 README](../README.md) - 项目总体介绍
- [学习指南](../docs/learning/weekly-guide/) - 6 周系统学习 (180-240 小时)
- [eBPF 知识点](../docs/learning/ebpf-knowledge.md) - 核心技术总结

### 外部资源
- [Cilium eBPF 文档](https://github.com/cilium/ebpf) - Go eBPF 库
- [libbpf 文档](https://github.com/libbpf/libbpf) - eBPF 用户态库
- [BPF 验证器文档](https://docs.kernel.org/bpf/verifier.html) - 理解验证器规则
- [eBPF by Example](https://ebpf.io/get-started/) - 官方入门教程

## ❓ 常见问题

### Q: 我需要什么基础知识?
**A**:
- 必须: C 语言基础, Linux 命令行
- 建议: 网络协议基础 (TCP/IP), Go 语言基础
- 加分: 内核基础知识, eBPF 概念

### Q: Demo 和主项目有什么区别?
**A**: Demo 是简化版,用于教学:
- 仅支持 IPv4 (主项目支持 IPv4/IPv6)
- 跳过高级特性 (NAT/分片/进程级策略)
- 代码结构更简单,注释更详细
- 专注核心概念,便于理解

### Q: 学完 Demo 能做什么?
**A**:
- 理解主项目的核心技术栈
- 阅读和修改主项目源码
- 实现自己的 eBPF 微隔离方案
- 具备 eBPF 生产实战基础

### Q: 遇到编译错误怎么办?
**A**:
1. 检查内核版本: `uname -r` (需要 5.10+)
2. 检查 clang 版本: `clang --version` (需要 14+)
3. 检查 BTF 支持: `ls /sys/kernel/btf/vmlinux`
4. 查看详细错误: `dmesg | tail -20`

### Q: eBPF 程序加载失败?
**A**:
1. 检查权限: 需要 root 或 CAP_BPF 权限
2. 检查验证器错误: `dmesg` 查看详细日志
3. 检查 Map 大小: 是否超过系统限制
4. 检查网卡状态: `ip link show`

## 🎯 学习成果

完成所有 Demo 后,你将:
- ✅ 掌握 eBPF TC Hook 数据包过滤
- ✅ 理解 5 种 Map 类型的使用场景
- ✅ 实现完整的会话跟踪和策略匹配
- ✅ 掌握 TCP 状态机原理
- ✅ 使用 Cilium eBPF 库开发用户态控制
- ✅ 构建生产级微隔离数据平面

预计学习时间: **60 小时** (1-2 周全职)

---

**开始你的 eBPF 学习之旅吧!** 🚀

建议从 [Demo 1: Hello Packet Counter](./01-hello-packet/README.md) 开始。
