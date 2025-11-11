# XDP 和 TC 程序的 Map 共享设计

## 概述

本文档解释了在 XDP/TC 双模式架构中,为什么某些 eBPF Map 需要共享,而另一些需要保持独立。

## Map 共享策略总结

| Map 类型 | 是否共享 | 共享方式 | 原因 |
|---------|---------|---------|------|
| **policy_map** | ✅ 共享 | Map Pinning | 策略必须一致,只读访问无冲突 |
| **wildcard_policy_map** | ✅ 共享 | Map Pinning | 策略必须一致,只读访问无冲突 |
| **stats_map** | ✅ 共享 | Map Pinning | 统计需要合并,Per-CPU 支持并发 |
| **session_map** | ❌ 独立 | - | 数据包方向不同,避免冲突 |
| **flow_events** | ❌ 独立 | - | Ring Buffer 技术限制,性能隔离 |

---

## 1. session_map 为何独立?

### 1.1 数据包处理路径不同

```
数据包流向:
┌─────────────┐
│ 网卡 (NIC)  │
└──────┬──────┘
       │
       ├──> XDP 程序 (驱动层)      ← XDP session_map
       │
       ├──> TC Ingress 程序 (网络栈) ← TC session_map
       │
       v
   内核网络栈
```

**关键差异**: XDP 和 TC 看到的数据包方向不同:
- **XDP**: 只能处理 **ingress (入站)** 数据包
- **TC**: 可以同时处理 **ingress 和 egress (出站)** 数据包

### 1.2 流键冲突问题

**场景**: 客户端 → 服务器的 TCP 连接

```c
// XDP 看到的请求数据包:
struct flow_key xdp_key = {
    .src_ip   = 192.168.1.100,  // 客户端
    .dst_ip   = 10.0.0.1,       // 服务器
    .src_port = 50000,
    .dst_port = 80,
    .protocol = IPPROTO_TCP
};

// TC 看到的响应数据包 (反向):
struct flow_key tc_key = {
    .src_ip   = 10.0.0.1,       // 服务器
    .dst_ip   = 192.168.1.100,  // 客户端
    .src_port = 80,
    .dst_port = 50000,
    .protocol = IPPROTO_TCP
};
```

**问题**: 如果共享 session_map,会导致:

1. **创建两个不同的会话记录**:
   ```c
   // XDP 创建的会话
   session_map[xdp_key] = {
       .packets_to_server = 100,
       .packets_to_client = 0,
       // ...
   };

   // TC 创建的会话 (不同的 key!)
   session_map[tc_key] = {
       .packets_to_server = 0,
       .packets_to_client = 100,  // 实际是同一个连接!
       // ...
   };
   ```

2. **统计数据混乱**:
   - 同一个 TCP 连接被计数两次
   - `packets_to_server` 和 `packets_to_client` 计数错误
   - 无法正确追踪双向流量

3. **内存浪费**:
   - 每个连接占用双倍 session_map 空间
   - LRU 淘汰效率降低

### 1.3 并发访问性能问题

```c
// XDP 在 CPU 0 上处理入站数据包
XDP (CPU 0): bpf_map_lookup_elem(&session_map, &xdp_key)
             session->packets_to_server += 1;

// TC 在 CPU 1 上同时处理出站数据包
TC (CPU 1):  bpf_map_lookup_elem(&session_map, &tc_key)
             session->packets_to_client += 1;
```

**性能影响**:

1. **缓存行争用 (Cache Line Contention)**:
   - XDP 和 TC 在不同 CPU 核心上运行
   - 共享 Map 会导致频繁的缓存失效 (Cache Invalidation)
   - 性能下降 20-30%

2. **更新冲突风险**:
   ```c
   // 如果 XDP 和 TC 同时更新同一个 session
   // (虽然 key 不同,但可能 hash 碰撞)

   // XDP 线程:
   session->last_seen_ts = get_timestamp_ns();  // T1
   session->packets_to_server += 1;             // T2

   // TC 线程 (同时):
   session->last_seen_ts = get_timestamp_ns();  // T1
   session->packets_to_client += 1;             // T2

   // 可能导致丢失更新 (Lost Update)
   ```

### 1.4 当前架构的设计决策

在当前的实现中,**不会同时运行 XDP 和 TC**:

```go
// src/agent/pkg/dataplane/dataplane.go
func New(iface string) (*DataPlane, error) {
    // 选择最佳模式
    mode := SelectBestMode(caps, config)

    if IsXDPMode(mode) {
        // 只加载 XDP 程序
        xdpLoader, err := NewXDPLoader(mode, iface, ifaceIdx)
        xdpLoader.Load()
    } else if IsTCMode(mode) {
        // 只加载 TC 程序
        tcLoader, err := NewTCLoader(mode, iface, ifaceIdx)
        tcLoader.Load()
    }
}
```

**设计原则**:
- **互斥运行**: 同一时间只有 XDP 或 TC,不会并存
- **简化实现**: 避免复杂的会话同步逻辑
- **性能优化**: 独立的 session_map 避免跨 CPU 竞争

**优点**:
- ✅ 无数据不一致风险
- ✅ 实现简单,易于维护
- ✅ 性能最优

**缺点**:
- ❌ 切换模式时会话数据丢失 (可接受,因为切换很少发生)

---

## 2. flow_events (Ring Buffer) 为何独立?

### 2.1 Ring Buffer 的技术限制

```c
// Ring Buffer 定义
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);  // 256KB
} flow_events SEC(".maps");
```

**BPF Ring Buffer 的内部结构**:

```
Ring Buffer 内存布局:
┌─────────────────────────────────────────┐
│ producer (生产者指针)                    │ ← XDP 或 TC 写入位置
├─────────────────────────────────────────┤
│ consumer (消费者指针)                    │ ← 用户空间读取位置
├─────────────────────────────────────────┤
│ data[] (256KB 循环缓冲区)                │
│   ┌──────┬──────┬──────┬──────┐        │
│   │ evt1 │ evt2 │ evt3 │ ...  │        │
│   └──────┴──────┴──────┴──────┘        │
└─────────────────────────────────────────┘
```

**技术限制**: Ring Buffer 不支持多生产者:

```c
// Ring Buffer 内部的写入逻辑 (简化版)
static int ringbuf_reserve(struct ringbuf_map *rb, size_t size) {
    // 读取当前 producer 指针
    u64 producer = atomic_read(&rb->producer);

    // 计算新位置
    u64 new_producer = producer + size;

    // 更新 producer 指针
    atomic_set(&rb->producer, new_producer);  // ⚠️ 竞争条件!

    return producer;  // 返回保留的位置
}
```

**问题**: 如果 XDP 和 TC 同时写入:

```c
// XDP (CPU 0) 和 TC (CPU 1) 同时调用 ringbuf_reserve()

// 时刻 T1:
XDP: producer = 1000  // 读取
TC:  producer = 1000  // 读取 (相同!)

// 时刻 T2:
XDP: new_producer = 1000 + 100 = 1100
TC:  new_producer = 1000 + 200 = 1200

// 时刻 T3:
XDP: atomic_set(&rb->producer, 1100)  // 写入
TC:  atomic_set(&rb->producer, 1200)  // 覆盖! ⚠️

// 结果: XDP 的事件被覆盖,数据损坏
```

**解决方案**: 使用独立的 Ring Buffer,避免多生产者问题。

### 2.2 事件来源区分

如果共享 Ring Buffer,无法区分事件来源:

```c
struct flow_event {
    // 5-tuple
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8  protocol;

    // 元数据
    __u8  event_type;     // NEW, UPDATE, CLOSED
    __u8  direction;      // INGRESS, EGRESS

    // ⚠️ 缺少: 来自 XDP 还是 TC?
};
```

**问题**:
- 调试困难: 无法知道事件来自哪个数据平面
- 监控混乱: XDP 和 TC 的统计混在一起
- 性能分析受限: 无法单独分析 XDP/TC 的性能

**独立 Ring Buffer 的好处**:

```go
// 用户空间可以分别订阅
type DataPlane struct {
    xdpEvents  *ringbuf.Reader  // XDP 事件
    tcEvents   *ringbuf.Reader  // TC 事件
}

// 根据模式选择
func (dp *DataPlane) GetFlowEvents() *ringbuf.Reader {
    if dp.mode == ModeNativeXDP || dp.mode == ModeGenericXDP {
        return dp.xdpEvents
    } else {
        return dp.tcEvents
    }
}

// 或者分别监控
func (dp *DataPlane) MonitorXDPEvents() { /* ... */ }
func (dp *DataPlane) MonitorTCEvents()  { /* ... */ }
```

### 2.3 性能隔离

```
┌──────────────────┐
│ XDP Ring Buffer  │ ← 高频事件 (100k events/s)
│   (256KB)        │
└──────────────────┘

┌──────────────────┐
│ TC Ring Buffer   │ ← 中频事件 (10k events/s)
│   (256KB)        │
└──────────────────┘
```

**独立 Ring Buffer 的性能优势**:

1. **性能隔离**:
   ```
   场景: XDP 处理高频 DDoS 攻击流量

   如果共享 Ring Buffer:
   - XDP 产生大量 DENY 事件
   - Ring Buffer 快速填满
   - TC 的正常事件被丢弃 ❌

   如果独立 Ring Buffer:
   - XDP Ring Buffer 满了,只丢弃 XDP 事件
   - TC Ring Buffer 不受影响 ✅
   ```

2. **大小可调优**:
   ```c
   // 可以为 XDP 和 TC 配置不同大小

   // XDP: 高频,需要更大缓冲区
   struct {
       __uint(type, BPF_MAP_TYPE_RINGBUF);
       __uint(max_entries, 1024 * 1024);  // 1MB
   } xdp_flow_events SEC(".maps");

   // TC: 中频,较小缓冲区即可
   struct {
       __uint(type, BPF_MAP_TYPE_RINGBUF);
       __uint(max_entries, 256 * 1024);   // 256KB
   } tc_flow_events SEC(".maps");
   ```

3. **错误隔离**:
   - XDP 的 Ring Buffer 满了 → 只丢弃 XDP 事件
   - TC 的 Ring Buffer 满了 → 只丢弃 TC 事件
   - 互不影响

---

## 3. 策略 Map 为何要共享?

### 3.1 策略一致性

```
用户配置策略: "拒绝 192.168.1.100 访问 80 端口"

场景 1: XDP 处理请求
  192.168.1.100:50000 → 10.0.0.1:80
  查询 policy_map → DENY ✅

场景 2: TC 处理请求
  192.168.1.100:50001 → 10.0.0.1:80
  查询 policy_map → DENY ✅

如果不共享:
  - 需要维护两份策略 (XDP 和 TC)
  - 同步问题: XDP 更新了,TC 没更新 → 不一致 ❌
  - 内存浪费: 策略数据存储两次
```

**Map Pinning 机制**:

```c
// TC 程序
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(pinning, LIBBPF_PIN_BY_NAME);  // 固定到 /sys/fs/bpf/
    __type(key, struct policy_key);
    __type(value, struct policy_value);
} policy_map SEC(".maps");

// XDP 程序 (相同定义)
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(pinning, LIBBPF_PIN_BY_NAME);  // 固定到同一位置!
    __type(key, struct policy_key);
    __type(value, struct policy_value);
} policy_map SEC(".maps");
```

**文件系统视图**:

```bash
/sys/fs/bpf/microsegment/
├── policy_map              # TC 和 XDP 都访问这个 map
├── wildcard_policy_map     # TC 和 XDP 都访问这个 map
└── stats_map               # TC 和 XDP 都访问这个 map
```

### 3.2 只读访问,无冲突

**策略在数据平面是只读的**:

```c
// XDP/TC 只查询策略,不修改
static __always_inline __u8 lookup_policy_action(
    struct flow_key *key,
    __u32 *rule_id)
{
    // 读取策略 (只读操作)
    struct policy_value *policy = bpf_map_lookup_elem(&policy_map, key);

    if (policy) {
        // 更新命中计数 (这是可以的,因为是统计信息)
        policy->hit_count += 1;  // Per-policy counter

        *rule_id = policy->rule_id;
        return policy->action;
    }

    // 没有修改策略本身!
    return POLICY_ACTION_ALLOW;
}
```

**策略修改由控制平面负责**:

```go
// 控制平面 (Agent) 更新策略
func (dp *DataPlane) UpdatePolicy(policy *Policy) error {
    key := PolicyKey{
        SrcIP:    policy.SrcIP,
        DstIP:    policy.DstIP,
        // ...
    }

    value := PolicyValue{
        Action:  policy.Action,
        RuleID:  policy.RuleID,
        // ...
    }

    // 更新共享的 policy_map
    // XDP 和 TC 都会立即看到新策略
    return dp.maps.PolicyMap.Update(&key, &value, ebpf.UpdateAny)
}
```

**无并发冲突**:
- ✅ XDP/TC 只读,控制平面写 → 无冲突
- ✅ 多个 CPU 同时读同一个策略 → 无问题 (缓存友好)
- ✅ 策略更新是原子操作 → 一致性保证

### 3.3 统计数据合并

```c
// stats_map 是 PERCPU_ARRAY
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(pinning, LIBBPF_PIN_BY_NAME);
    __type(key, __u32);
    __type(value, __u64);
} stats_map SEC(".maps");
```

**Per-CPU 特性**:

```
CPU 0: stats_map[STATS_TOTAL_PACKETS] = 1000  (XDP)
CPU 1: stats_map[STATS_TOTAL_PACKETS] = 500   (XDP)
CPU 2: stats_map[STATS_TOTAL_PACKETS] = 200   (TC)
CPU 3: stats_map[STATS_TOTAL_PACKETS] = 100   (TC)

用户空间读取:
  total = 1000 + 500 + 200 + 100 = 1800  ✅
```

**自动合并**:

```go
// 用户空间读取统计
func (dp *DataPlane) GetStatistics() Statistics {
    var values []uint64

    // 读取所有 CPU 的值
    key := uint32(STATS_TOTAL_PACKETS)
    dp.maps.StatsMap.Lookup(&key, &values)

    // 自动合并
    var total uint64
    for _, v := range values {
        total += v  // XDP + TC 的统计都包含在内
    }

    return Statistics{
        TotalPackets: total,
    }
}
```

**优势**:
- ✅ XDP 和 TC 的统计自动合并
- ✅ Per-CPU 设计,无锁操作
- ✅ 提供全局视图,便于监控

---

## 4. 未来优化方向

### 4.1 连接跟踪表 (Connection Tracking)

如果未来需要**同时运行 XDP 和 TC** (例如 XDP 处理入站,TC 处理出站):

```c
// 共享的连接跟踪表
struct conntrack_key {
    __u32 client_ip;     // 始终是发起连接的 IP
    __u32 server_ip;     // 始终是被连接的 IP
    __u16 client_port;   // 始终是发起连接的端口
    __u16 server_port;   // 始终是被连接的端口
    __u8  protocol;
} __attribute__((packed));

struct conntrack_value {
    __u64 created_ts;
    __u64 last_seen_ts;
    __u64 packets_c2s;   // client → server
    __u64 packets_s2c;   // server → client
    __u64 bytes_c2s;
    __u64 bytes_s2c;
    __u8  state;
    __u8  policy_action;
} __attribute__((packed));

// 共享的连接跟踪表
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(pinning, LIBBPF_PIN_BY_NAME);  // 共享!
    __type(key, struct conntrack_key);
    __type(value, struct conntrack_value);
} conntrack_map SEC(".maps");
```

**规范化流键**:

```c
// XDP 和 TC 都使用相同的规范化逻辑
static __always_inline void normalize_flow_key(
    struct flow_key *key,
    struct conntrack_key *ct_key,
    bool *is_reverse)
{
    // 规则: 始终保持 client_ip < server_ip
    // (或者根据端口判断:小端口是服务器)

    if (key->dst_port == 80 || key->dst_port == 443) {
        // 正向: client → server
        ct_key->client_ip   = key->src_ip;
        ct_key->client_port = key->src_port;
        ct_key->server_ip   = key->dst_ip;
        ct_key->server_port = key->dst_port;
        *is_reverse = false;
    } else if (key->src_port == 80 || key->src_port == 443) {
        // 反向: server → client
        ct_key->client_ip   = key->dst_ip;
        ct_key->client_port = key->dst_port;
        ct_key->server_ip   = key->src_ip;
        ct_key->server_port = key->src_port;
        *is_reverse = true;
    } else {
        // 默认: 小 IP 是 client
        if (key->src_ip < key->dst_ip) {
            ct_key->client_ip = key->src_ip;
            ct_key->server_ip = key->dst_ip;
            // ...
            *is_reverse = false;
        } else {
            ct_key->client_ip = key->dst_ip;
            ct_key->server_ip = key->src_ip;
            // ...
            *is_reverse = true;
        }
    }
}
```

**使用示例**:

```c
// XDP 处理入站数据包: client → server
struct flow_key xdp_key = {
    .src_ip = 192.168.1.100, .dst_ip = 10.0.0.1,
    .src_port = 50000, .dst_port = 80
};

struct conntrack_key ct_key;
bool is_reverse;
normalize_flow_key(&xdp_key, &ct_key, &is_reverse);

// ct_key = { client: 192.168.1.100:50000, server: 10.0.0.1:80 }
// is_reverse = false

struct conntrack_value *ct = bpf_map_lookup_elem(&conntrack_map, &ct_key);
if (ct) {
    ct->packets_c2s += 1;  // client → server
}
```

```c
// TC 处理出站数据包: server → client (响应)
struct flow_key tc_key = {
    .src_ip = 10.0.0.1, .dst_ip = 192.168.1.100,
    .src_port = 80, .dst_port = 50000
};

normalize_flow_key(&tc_key, &ct_key, &is_reverse);

// ct_key = { client: 192.168.1.100:50000, server: 10.0.0.1:80 }
//           (相同的 key!)
// is_reverse = true

struct conntrack_value *ct = bpf_map_lookup_elem(&conntrack_map, &ct_key);
if (ct) {
    ct->packets_s2c += 1;  // server → client
}
```

**优势**:
- ✅ XDP 和 TC 访问同一个连接记录
- ✅ 双向流量统计准确
- ✅ 节省内存

### 4.2 事件聚合

虽然 Ring Buffer 保持独立,但可以在用户空间合并:

```go
// 用户空间事件聚合器
type EventAggregator struct {
    xdpReader  *ringbuf.Reader
    tcReader   *ringbuf.Reader
    merged     chan FlowEvent
    stop       chan struct{}
}

func NewEventAggregator(dp *DataPlane) *EventAggregator {
    return &EventAggregator{
        xdpReader: dp.GetXDPFlowRingBuffer(),
        tcReader:  dp.GetTCFlowRingBuffer(),
        merged:    make(chan FlowEvent, 1000),
        stop:      make(chan struct{}),
    }
}

func (a *EventAggregator) Start() {
    // 从 XDP Ring Buffer 读取
    go func() {
        for {
            record, err := a.xdpReader.Read()
            if err != nil {
                continue
            }

            event := parseFlowEvent(record.RawSample)
            event.Source = "XDP"  // 标记来源

            select {
            case a.merged <- event:
            case <-a.stop:
                return
            }
        }
    }()

    // 从 TC Ring Buffer 读取
    go func() {
        for {
            record, err := a.tcReader.Read()
            if err != nil {
                continue
            }

            event := parseFlowEvent(record.RawSample)
            event.Source = "TC"  // 标记来源

            select {
            case a.merged <- event:
            case <-a.stop:
                return
            }
        }
    }()
}

func (a *EventAggregator) Events() <-chan FlowEvent {
    return a.merged
}
```

**使用**:

```go
aggregator := NewEventAggregator(dataplane)
aggregator.Start()

for event := range aggregator.Events() {
    log.Printf("[%s] Flow: %s -> %s",
        event.Source,  // "XDP" 或 "TC"
        event.SrcAddr,
        event.DstAddr)
}
```

---

## 5. 实现代码参考

### 5.1 TC 程序 Map 定义

```c
// src/bpf/tc_microsegment.bpf.c

// 独立的会话表
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, MAX_ENTRIES_SESSION);
    __type(key, struct flow_key);
    __type(value, struct session_value);
} session_map SEC(".maps");

// 共享的策略表
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_ENTRIES_POLICY);
    __type(key, struct policy_key);
    __type(value, struct policy_value);
    __uint(pinning, LIBBPF_PIN_BY_NAME);  // ← 共享
} policy_map SEC(".maps");

// 独立的事件缓冲区
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} flow_events SEC(".maps");
```

### 5.2 XDP 程序 Map 定义

```c
// src/bpf/xdp_microsegment.bpf.c

// 独立的会话表 (与 TC 不共享)
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, MAX_ENTRIES_SESSION);
    __type(key, struct flow_key);
    __type(value, struct session_value);
} session_map SEC(".maps");

// 共享的策略表 (与 TC 共享)
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_ENTRIES_POLICY);
    __type(key, struct policy_key);
    __type(value, struct policy_value);
    __uint(pinning, LIBBPF_PIN_BY_NAME);  // ← 共享
} policy_map SEC(".maps");

// 独立的事件缓冲区 (与 TC 不共享)
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} flow_events SEC(".maps");
```

### 5.3 Map Pinning 路径

```bash
# XDP 和 TC 共享的 Map 文件
$ ls -l /sys/fs/bpf/microsegment/
-rw------- 1 root root 0 policy_map           # 共享
-rw------- 1 root root 0 wildcard_policy_map  # 共享
-rw------- 1 root root 0 stats_map            # 共享

# 注意: session_map 和 flow_events 不会出现在这里
#       因为它们不使用 pinning,是各自独立的
```

---

## 6. 性能对比

### 6.1 共享 vs 独立 session_map

| 场景 | 共享 session_map | 独立 session_map |
|------|----------------|----------------|
| **内存使用** | 2x 会话数 (重复) | 1x 会话数 ✅ |
| **缓存性能** | 差 (跨 CPU 竞争) | 优秀 ✅ |
| **数据一致性** | 复杂 (需要同步) | 简单 ✅ |
| **实现复杂度** | 高 (需要连接跟踪) | 低 ✅ |

### 6.2 共享 vs 独立 flow_events

| 场景 | 共享 Ring Buffer | 独立 Ring Buffer |
|------|-----------------|-----------------|
| **并发写入** | 不支持 ❌ | 支持 ✅ |
| **事件区分** | 困难 ❌ | 简单 ✅ |
| **性能隔离** | 无 ❌ | 有 ✅ |
| **错误隔离** | 无 ❌ | 有 ✅ |

---

## 7. 总结

### 7.1 设计原则

1. **只读数据共享,可写数据独立**:
   - 策略 (只读) → 共享
   - 会话 (可写) → 独立

2. **全局统计共享,局部状态独立**:
   - stats_map (全局) → 共享 (Per-CPU)
   - session_map (局部) → 独立

3. **技术限制优先**:
   - Ring Buffer 不支持多生产者 → 独立
   - Map 支持并发读取 → 可以共享

### 7.2 当前架构的优势

- ✅ **实现简单**: XDP 和 TC 互斥运行,无复杂同步
- ✅ **性能最优**: 独立 session_map,无跨 CPU 竞争
- ✅ **策略一致**: Map Pinning 确保策略同步
- ✅ **易于调试**: 独立 Ring Buffer,事件来源清晰

### 7.3 未来扩展路径

如需同时运行 XDP 和 TC:
1. 引入共享的 `conntrack_map` (连接跟踪表)
2. 实现流键规范化算法
3. 用户空间聚合 XDP 和 TC 的事件流

---

## 参考资料

- [BPF Ring Buffer Documentation](https://docs.kernel.org/bpf/ringbuf.html)
- [BPF Map Types](https://docs.kernel.org/bpf/maps.html)
- [Map Pinning in libbpf](https://github.com/libbpf/libbpf)
- XDP/TC 双模式架构提案: `openspec/changes/add-xdp-tc-dual-mode/proposal.md`
