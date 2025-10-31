# 变更提案：实施 eBPF 会话跟踪系统

**状态**：已归档（回顾性）  
**实施时间**：2025-10-30  
**Change ID**: implement-session-tracking

## Why

网络微隔离需要高效的会话状态跟踪以避免每个数据包的策略查找。如果没有会话缓存，每个数据包都需要昂贵的映射查找进行策略评估，从而严重影响性能。

## What Changes

- 实施基于 LRU_HASH 映射的会话跟踪系统
- 添加 5 元组流密钥（src_ip、dst_ip、src_port、dst_port、protocol）
- 创建会话值结构，包含时间戳、数据包/字节计数器和缓存的策略操作
- 用于内存管理的自动 LRU 驱逐（最大 100k 条目）
- 会话状态机（NEW、ESTABLISHED、CLOSED）
- TCP 状态跟踪（SYN_SENT、ESTABLISHED、FIN_WAIT 等）

## Impact

**受影响的规范：**
- `data-plane/packet-processing`（新增）
- `data-plane/session-tracking`（新增）

**受影响的代码：**
- `src/bpf/tc_microsegment.bpf.c` - 主 TC eBPF 程序
- `src/bpf/headers/common_types.h` - 数据结构定义
- `src/agent/pkg/dataplane/dataplane.go` - Go 绑定

**性能影响：**
- 热路径：现有会话的单次映射查找
- 缓存命中率：> 99%
- 会话容量：100,000 个并发连接

