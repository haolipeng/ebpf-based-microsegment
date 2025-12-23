[根目录](../../CLAUDE.md) > **src/bpf**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# BPF 模块

eBPF 内核程序，实现高性能数据包过滤、会话跟踪和策略执行。

## 核心职责

| 职责 | 说明 | 相关文件 |
|------|------|----------|
| 数据包过滤 | TC/XDP Hook 实现报文过滤 | tc_microsegment.bpf.c, xdp_microsegment.bpf.c |
| 会话跟踪 | LRU Hash Map 跟踪连接状态 | [headers/common_types.h](headers/CLAUDE.md) |
| 策略匹配 | 精确匹配 + 通配符匹配 | [headers/policy_match.h](headers/CLAUDE.md) |
| TCP 状态机 | 有状态连接过滤 | [headers/tcp_state_machine.h](headers/CLAUDE.md) |
| 事件上报 | Ring Buffer 上报流事件 | [headers/flow_processing.h](headers/CLAUDE.md) |
| 进程监控 | Tracepoint 捕获进程信息 | process_monitor.bpf.c |

## 文件清单

| 文件 | 职责 | Hook 类型 |
|------|------|-----------|
| tc_microsegment.bpf.c | TC 数据包过滤器（主程序） | `SEC("tc")` Ingress/Egress |
| xdp_microsegment.bpf.c | XDP 高性能过滤器 | `SEC("xdp")` Ingress only |
| process_monitor.bpf.c | 进程执行监控 | `SEC("tracepoint/sched/sched_process_exec")` |

## 子目录索引

| 子目录 | 职责 | 文档 |
|--------|------|------|
| **headers** | 共享数据类型、策略匹配、TCP 状态机 | [→](headers/CLAUDE.md) |

## 核心 eBPF Maps

| Map 名称 | 类型 | 用途 |
|----------|------|------|
| session_map | LRU_HASH | 会话状态缓存（热路径 <1μs） |
| policy_map | HASH | 精确匹配策略 |
| wildcard_policy_map | ARRAY | 通配符策略（CIDR/端口范围） |
| default_policy | ARRAY | 全局默认策略 |
| stats_map | PERCPU_ARRAY | Per-CPU 无锁统计 |
| flow_events | RINGBUF | 流事件上报 |
| conntrack_map | HASH | NAT 连接跟踪 |

## 策略匹配流程

```
1. 检查 session_map → 命中则返回缓存 action（热路径）
2. 精确匹配 policy_map（Hash 查找）
3. 遍历 wildcard_policy_map（CIDR/端口匹配）
4. 返回 default_policy
```

## 特性标志

| 标志 | 说明 | 默认值 |
|------|------|--------|
| `DEBUG_MODE` | 启用 bpf_printk 调试日志 | 0 |
| `ENABLE_IP_FRAGMENT_HANDLING` | 处理 IP 分片 | 1 |
| `ENABLE_NAT_SUPPORT` | NAT 检测支持 | 1 |

## 调试命令

```bash
# 查看 eBPF 程序
sudo bpftool prog show

# 查看 Maps 内容
sudo bpftool map dump name policy_map

# 查看调试日志
sudo cat /sys/kernel/debug/tracing/trace_pipe
```

## 构建

```bash
# 生产构建
make build-production

# 调试构建（启用 DEBUG_MODE）
make build-debug

# 重新生成 Go 绑定
make bpf
```

---

**最后更新**: 2025-12-23
