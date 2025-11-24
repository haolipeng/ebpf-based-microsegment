# eBPF 知识点总结 - 微分割项目

本文档总结了该项目中使用的所有 eBPF 技术知识点，供学习和练习使用。

---

## 目录

1. [eBPF 程序类型](#1-ebpf-程序类型)
2. [eBPF Map 类型](#2-ebpf-map-类型)
3. [eBPF 辅助函数](#3-ebpf-辅助函数)
4. [核心数据结构](#4-核心数据结构)
5. [网络协议处理](#5-网络协议处理)
6. [用户态与内核态交互](#6-用户态与内核态交互)
7. [使用的 eBPF 库](#7-使用的-ebpf-库)
8. [Demo 练习建议](#8-demo-练习建议)

---

## 1. eBPF 程序类型

### 1.1 Tracepoint - 进程监控

**项目位置**: `src/bpf/process_monitor.bpf.c`

**程序定义**:
```c
SEC("tp/sched/sched_process_exec")
int trace_sched_process_exec(struct trace_event_raw_sched_process_exec *ctx)
```

**功能说明**:
- 在进程执行时触发（`execve`、`execveat` 系统调用）
- 捕获 PID、命令名称、执行时间、容器 ID
- 通过 Ring Buffer 推送进程事件到用户态

**Demo 练习**:
```c
// 简单的进程监控示例
SEC("tp/sched/sched_process_exec")
int trace_exec(struct trace_event_raw_sched_process_exec *ctx)
{
    __u64 pid_tgid = bpf_get_current_pid_tgid();
    __u32 pid = pid_tgid >> 32;

    char comm[16];
    bpf_get_current_comm(&comm, sizeof(comm));

    bpf_printk("Process exec: pid=%d, comm=%s\n", pid, comm);
    return 0;
}
```

### 1.2 XDP - 高性能数据包处理

**项目位置**: `src/bpf/xdp_microsegment.bpf.c`

**程序定义**:
```c
SEC("xdp")
int xdp_microsegment_prog(struct xdp_md *ctx)
```

**特性**:
| 特性 | 说明 |
|------|------|
| 处理位置 | 网卡驱动层（最高性能） |
| 方向 | 仅 INGRESS |
| 上下文 | `struct xdp_md` |
| 返回值 | `XDP_PASS`, `XDP_DROP`, `XDP_REDIRECT`, `XDP_TX`, `XDP_ABORTED` |

**Demo 练习**:
```c
SEC("xdp")
int xdp_drop_icmp(struct xdp_md *ctx)
{
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return XDP_PASS;

    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return XDP_PASS;

    struct iphdr *iph = (void *)(eth + 1);
    if ((void *)(iph + 1) > data_end)
        return XDP_PASS;

    // Drop ICMP packets
    if (iph->protocol == IPPROTO_ICMP)
        return XDP_DROP;

    return XDP_PASS;
}
```

### 1.3 TC (Traffic Control) - 流量控制

**项目位置**: `src/bpf/tc_microsegment.bpf.c`

**程序定义**:
```c
SEC("tc")
int tc_microsegment_filter(struct __sk_buff *skb)
```

**特性**:
| 特性 | 说明 |
|------|------|
| 处理位置 | 内核网络栈 |
| 方向 | INGRESS 和 EGRESS |
| 上下文 | `struct __sk_buff` |
| 返回值 | `TC_ACT_OK`, `TC_ACT_SHOT`, `TC_ACT_REDIRECT` |

**XDP vs TC 对比**:
| 特性 | XDP | TC |
|------|-----|-----|
| 性能 | 最高 | 较高 |
| 方向 | 仅入站 | 入站+出站 |
| 功能 | 简单 | 更丰富 |
| SKB | 无 | 有 |

---

## 2. eBPF Map 类型

### 2.1 LRU_HASH - 自动淘汰哈希表

**项目位置**: `src/bpf/xdp_microsegment.bpf.c:80-85`

```c
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 100000);
    __type(key, struct flow_key);
    __type(value, struct session_value);
} session_map SEC(".maps");
```

**用途**: 会话追踪，自动淘汰最久未使用的条目

**Demo 练习**:
```c
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 1024);
    __type(key, __u32);  // IP address
    __type(value, __u64); // packet count
} ip_count_map SEC(".maps");

// 使用示例
__u32 src_ip = iph->saddr;
__u64 *count = bpf_map_lookup_elem(&ip_count_map, &src_ip);
if (count) {
    (*count)++;
} else {
    __u64 init = 1;
    bpf_map_update_elem(&ip_count_map, &src_ip, &init, BPF_ANY);
}
```

### 2.2 HASH - 固定大小哈希表

**项目位置**: `src/bpf/xdp_microsegment.bpf.c:89-95`

```c
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, struct policy_key);
    __type(value, struct policy_value);
    __uint(pinning, LIBBPF_PIN_BY_NAME);  // Pin 到文件系统
} policy_map SEC(".maps");
```

**用途**: 精确 5 元组策略匹配

### 2.3 ARRAY - 数组

**项目位置**: `src/bpf/xdp_microsegment.bpf.c:101-106`

```c
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1000);
    __type(key, __u32);
    __type(value, struct wildcard_policy);
} wildcard_policy_map SEC(".maps");
```

**用途**: 通配符策略存储，支持线性扫描

### 2.4 PERCPU_ARRAY - 每 CPU 数组

**项目位置**: `src/bpf/xdp_microsegment.bpf.c:122-128`

```c
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 32);
    __type(key, __u32);
    __type(value, __u64);
} stats_map SEC(".maps");
```

**用途**: 统计计数器，每 CPU 一个副本，无锁竞争

**Demo 练习**:
```c
// 定义统计 key
enum stats_key {
    STATS_PACKETS = 0,
    STATS_BYTES = 1,
    STATS_MAX,
};

// 更新统计
static __always_inline void update_stats(__u32 key, __u64 value) {
    __u64 *counter = bpf_map_lookup_elem(&stats_map, &key);
    if (counter)
        *counter += value;
}
```

### 2.5 RINGBUF - 环形缓冲区

**项目位置**: `src/bpf/xdp_microsegment.bpf.c:142-145`

```c
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);  // 256KB
} flow_events SEC(".maps");
```

**用途**: 内核 → 用户态的高效事件推送

**Demo 练习**:
```c
struct event {
    __u32 pid;
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
};

// 推送事件
struct event *e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
if (!e)
    return 0;

e->pid = pid;
e->src_ip = src_ip;
// ... 填充其他字段

bpf_ringbuf_submit(e, 0);
```

### Map 类型对比表

| Map 类型 | 查找复杂度 | 适用场景 | 特性 |
|----------|-----------|----------|------|
| HASH | O(1) | 精确匹配 | 固定大小 |
| LRU_HASH | O(1) | 会话追踪 | 自动淘汰 |
| ARRAY | O(1) | 索引访问 | 预分配 |
| PERCPU_ARRAY | O(1) | 统计计数 | 无锁 |
| RINGBUF | O(1) | 事件推送 | 变长数据 |

---

## 3. eBPF 辅助函数

### 3.1 进程/时间相关

```c
// 获取当前 PID 和线程 ID
__u64 pid_tgid = bpf_get_current_pid_tgid();
__u32 pid = pid_tgid >> 32;  // 高 32 位是 PID
__u32 tid = pid_tgid & 0xFFFFFFFF;  // 低 32 位是 TID

// 获取当前进程名
char comm[16];
bpf_get_current_comm(&comm, sizeof(comm));

// 获取时间戳（纳秒）
__u64 ts = bpf_ktime_get_ns();
```

### 3.2 Map 操作

```c
// 查找元素
void *value = bpf_map_lookup_elem(&map, &key);

// 更新元素
// flags: BPF_ANY (插入或更新), BPF_NOEXIST (仅插入), BPF_EXIST (仅更新)
bpf_map_update_elem(&map, &key, &value, BPF_ANY);

// 删除元素
bpf_map_delete_elem(&map, &key);
```

### 3.3 字节序转换

```c
// 主机字节序 -> 网络字节序
__u16 net_port = bpf_htons(host_port);
__u32 net_ip = bpf_htonl(host_ip);

// 网络字节序 -> 主机字节序
__u16 host_port = bpf_ntohs(net_port);
__u32 host_ip = bpf_ntohl(net_ip);
```

### 3.4 Ring Buffer 操作

```c
// 预留空间（非阻塞）
struct event *e = bpf_ringbuf_reserve(&ringbuf, sizeof(*e), 0);
if (!e)
    return -1;  // 缓冲区满

// 填充数据
e->field1 = value1;

// 提交
bpf_ringbuf_submit(e, 0);

// 或者丢弃
bpf_ringbuf_discard(e, 0);
```

### 3.5 调试输出

```c
// 输出到 /sys/kernel/debug/tracing/trace_pipe
bpf_printk("Debug: pid=%d, ip=%pI4\n", pid, &ip);

// 查看输出
// sudo cat /sys/kernel/debug/tracing/trace_pipe
```

---

## 4. 核心数据结构

### 4.1 流键 (5 元组)

**项目位置**: `src/bpf/headers/common_types.h:25-33`

```c
struct flow_key {
    __u32 src_ip[4];   // 源 IP（IPv6 兼容格式）
    __u32 dst_ip[4];   // 目标 IP
    __u16 src_port;    // 源端口
    __u16 dst_port;    // 目标端口
    __u8  protocol;    // 协议 (TCP=6, UDP=17, ICMP=1)
    __u8  ip_version;  // 4 或 6
    __u16 vlan_id;     // VLAN ID
} __attribute__((packed));
```

### 4.2 会话值

**项目位置**: `src/bpf/headers/common_types.h:79-100`

```c
struct session_value {
    __u64 created_ts;         // 创建时间
    __u64 last_seen_ts;       // 最后活跃时间
    __u64 packets_to_server;  // 包计数
    __u64 bytes_to_server;    // 字节计数

    // TCP 状态追踪
    __u32 tcp_seq_client;
    __u32 tcp_ack_client;
    __u8  tcp_state;
    __u8  policy_action;      // 缓存的策略决策
};
```

### 4.3 通配符策略

**项目位置**: `src/bpf/headers/common_types.h:131-151`

```c
struct wildcard_policy {
    __u32 src_ip[4];
    __u32 src_ip_mask[4];     // 掩码匹配
    __u32 dst_ip[4];
    __u32 dst_ip_mask[4];
    __u16 src_port;           // 0 = 任意
    __u16 dst_port;
    __u8  protocol;           // 0 = 任意
    __u8  action;             // 允许/拒绝
    __u16 priority;           // 优先级
    char  process_name[16];   // 进程级策略
};
```

---

## 5. 网络协议处理

### 5.1 数据包解析

**项目位置**: `src/bpf/headers/flow_processing.h`

```c
// 解析以太网头
struct ethhdr *eth = data;
if ((void *)(eth + 1) > data_end)
    return -1;

// 处理 VLAN
__u16 eth_proto = eth->h_proto;
if (eth_proto == bpf_htons(ETH_P_8021Q)) {
    struct vlan_hdr *vlan = (void *)(eth + 1);
    if ((void *)(vlan + 1) > data_end)
        return -1;
    eth_proto = vlan->h_vlan_encapsulated_proto;
    // 数据偏移 +4 字节
}

// 解析 IP 头
struct iphdr *iph = (void *)(eth + 1) + vlan_offset;
if ((void *)(iph + 1) > data_end)
    return -1;

// 解析 TCP/UDP 头
if (iph->protocol == IPPROTO_TCP) {
    struct tcphdr *tcph = (void *)iph + (iph->ihl * 4);
    if ((void *)(tcph + 1) > data_end)
        return -1;
    // 提取端口
    __u16 src_port = bpf_ntohs(tcph->source);
    __u16 dst_port = bpf_ntohs(tcph->dest);
}
```

### 5.2 TCP 状态机

**项目位置**: `src/bpf/headers/tcp_state_machine.h`

```c
// TCP 状态枚举
enum tcp_state {
    TCP_STATE_CLOSED = 0,
    TCP_STATE_SYN_SENT,
    TCP_STATE_SYN_RECV,
    TCP_STATE_ESTABLISHED,
    TCP_STATE_FIN_WAIT1,
    TCP_STATE_FIN_WAIT2,
    TCP_STATE_CLOSE_WAIT,
    TCP_STATE_CLOSING,
    TCP_STATE_LAST_ACK,
    TCP_STATE_TIME_WAIT,
};

// TCP 标志提取
static __always_inline __u8 get_tcp_flags(struct tcphdr *tcph)
{
    __u8 flags = 0;
    if (tcph->fin) flags |= 0x01;
    if (tcph->syn) flags |= 0x02;
    if (tcph->rst) flags |= 0x04;
    if (tcph->psh) flags |= 0x08;
    if (tcph->ack) flags |= 0x10;
    if (tcph->urg) flags |= 0x20;
    return flags;
}

// 状态转换
static __always_inline __u8 tcp_state_transition(__u8 state, __u8 flags)
{
    // 三次握手
    // CLOSED + SYN -> SYN_RECV
    // SYN_RECV + ACK -> ESTABLISHED

    // 四次挥手
    // ESTABLISHED + FIN -> CLOSE_WAIT
    // ...
}
```

### 5.3 IP 分段处理

**项目位置**: `src/bpf/headers/fragment_tracking.h`

```c
// 分段类型
enum ipv4_frag_type {
    IPV4_FRAG_TYPE_NONE = 0,        // 非分段
    IPV4_FRAG_TYPE_FIRST = 1,       // 首分段
    IPV4_FRAG_TYPE_SUBSEQUENT = 2   // 后续分段
};

// 检测分段
#define IP_MF     0x2000  // More Fragments
#define IP_OFFSET 0x1FFF  // Fragment Offset

static __always_inline enum ipv4_frag_type check_fragment(struct iphdr *iph)
{
    __u16 frag_off = bpf_ntohs(iph->frag_off);

    if (!(frag_off & (IP_MF | IP_OFFSET)))
        return IPV4_FRAG_TYPE_NONE;

    if ((frag_off & IP_OFFSET) == 0)
        return IPV4_FRAG_TYPE_FIRST;

    return IPV4_FRAG_TYPE_SUBSEQUENT;
}
```

---

## 6. 用户态与内核态交互

### 6.1 Go 语言 Ring Buffer 接收

**项目位置**: `src/agent/pkg/flow/collector.go`

```go
import "github.com/cilium/ebpf/ringbuf"

// 创建 Reader
reader, err := ringbuf.NewReader(objs.FlowEvents)
if err != nil {
    log.Fatal(err)
}
defer reader.Close()

// 事件接收循环
for {
    record, err := reader.Read()
    if err != nil {
        if errors.Is(err, ringbuf.ErrClosed) {
            return
        }
        continue
    }

    // 解析事件
    event := (*FlowEvent)(unsafe.Pointer(&record.RawSample[0]))
    processEvent(event)
}
```

### 6.2 Map Pinning 共享

**项目位置**: `src/agent/pkg/dataplane/map_pinning.go`

```go
// 加载时指定 pin 路径
opts := &ebpf.CollectionOptions{
    Maps: ebpf.MapOptions{
        PinPath: "/sys/fs/bpf/microsegment/",
    },
}

// TC 和 XDP 共享同一个 policy_map
// 通过 __uint(pinning, LIBBPF_PIN_BY_NAME) 实现
```

### 6.3 程序加载和附加

```go
import (
    "github.com/cilium/ebpf"
    "github.com/cilium/ebpf/link"
)

// 加载 eBPF 对象
objs := &bpfObjects{}
if err := loadBpfObjects(objs, nil); err != nil {
    log.Fatal(err)
}

// 附加 XDP 程序
xdpLink, err := link.AttachXDP(link.XDPOptions{
    Program:   objs.XdpProg,
    Interface: ifIndex,
    Flags:     link.XDPDriverMode,  // 或 XDPGenericMode
})

// 附加 TC 程序 (kernel >= 6.6)
tcLink, err := link.AttachTCX(link.TCXOptions{
    Interface: ifIndex,
    Program:   objs.TcProg,
    Attach:    ebpf.AttachTCXIngress,
})
```

---

## 7. 使用的 eBPF 库

### 7.1 cilium/ebpf (Go)

```go
import (
    "github.com/cilium/ebpf"
    "github.com/cilium/ebpf/link"
    "github.com/cilium/ebpf/ringbuf"
    "github.com/cilium/ebpf/rlimit"
)

// 移除内存限制
rlimit.RemoveMemlock()

// 加载、附加、读取事件...
```

### 7.2 libbpf (C)

```c
#include <bpf/bpf_helpers.h>    // Map 和辅助函数
#include <bpf/bpf_core_read.h>  // CO-RE 支持
#include <bpf/bpf_endian.h>     // 字节序转换
```

### 7.3 vmlinux.h

自动生成的内核头文件，包含所有内核结构定义：

```bash
# 生成 vmlinux.h
bpftool btf dump file /sys/kernel/btf/vmlinux format c > vmlinux.h
```

```c
#include "vmlinux.h"  // 包含所有内核结构
```

---

## 8. Demo 练习建议

### 8.1 入门级练习

| 练习 | 知识点 | 难度 |
|------|--------|------|
| Hello World | Tracepoint + bpf_printk | ⭐ |
| 包计数器 | XDP + PERCPU_ARRAY | ⭐⭐ |
| 丢弃 ICMP | XDP + 数据包解析 | ⭐⭐ |

### 8.2 中级练习

| 练习 | 知识点 | 难度 |
|------|--------|------|
| 进程监控 | Tracepoint + RINGBUF | ⭐⭐⭐ |
| 简单防火墙 | XDP + HASH Map | ⭐⭐⭐ |
| 流量统计 | TC + LRU_HASH | ⭐⭐⭐ |

### 8.3 高级练习

| 练习 | 知识点 | 难度 |
|------|--------|------|
| TCP 状态追踪 | XDP + 状态机 | ⭐⭐⭐⭐ |
| 会话管理 | TC + LRU_HASH + RINGBUF | ⭐⭐⭐⭐ |
| NAT 穿透 | XDP + 连接追踪 | ⭐⭐⭐⭐⭐ |

### 8.4 推荐练习顺序

1. **Hello World** - 熟悉编译和加载流程
2. **包计数器** - 学习 Map 操作
3. **简单防火墙** - 学习数据包解析
4. **进程监控** - 学习 Ring Buffer
5. **TCP 状态追踪** - 学习状态机实现

### 8.5 编译命令

```bash
# 编译 BPF 程序
clang -O2 -g -target bpf \
    -D__TARGET_ARCH_x86 \
    -I/usr/include \
    -c my_prog.bpf.c -o my_prog.bpf.o

# 生成 Go skeleton
bpf2go -cc clang -cflags "-O2 -g" bpf my_prog.bpf.c
```

---

## 附录：项目文件结构

```
src/bpf/
├── process_monitor.bpf.c       # Tracepoint 程序
├── xdp_microsegment.bpf.c      # XDP 程序
├── tc_microsegment.bpf.c       # TC 程序
└── headers/
    ├── common_types.h          # 共享数据结构
    ├── tcp_state_machine.h     # TCP 状态机
    ├── nat_support.h           # NAT 支持
    ├── fragment_tracking.h     # 分段处理
    ├── flow_processing.h       # 数据包解析
    └── policy_match.h          # 策略匹配

src/agent/pkg/
├── dataplane/
│   ├── xdp_loader.go          # XDP 加载器
│   ├── tc_loader.go           # TC 加载器
│   └── map_pinning.go         # Map 共享
├── flow/
│   └── collector.go           # Ring Buffer 收集
└── process/
    └── monitor.go             # 进程监控
```

---

## 参考资源

- [eBPF 官方文档](https://ebpf.io/docs/)
- [cilium/ebpf 库](https://github.com/cilium/ebpf)
- [libbpf 库](https://github.com/libbpf/libbpf)
- [BPF 辅助函数参考](https://man7.org/linux/man-pages/man7/bpf-helpers.7.html)
