# eBPF + TC 可行性分析 - 架构设计

## 1. 整体架构

### 1.1 系统架构图

```
┌──────────────────────────────────────────────────────────┐
│                    容器网络命名空间                          │
│  ┌──────────────┐                                           │
│  │ Application  │                                           │
│  └──────┬───────┘                                           │
│         │                                                    │
│    ┌────▼────┐                                              │
│    │  eth0   │                                              │
│    └────┬────┘                                              │
└─────────┼──────────────────────────────────────────────────┘
          │ veth pair
┌─────────▼──────────────────────────────────────────────────┐
│                    主机网络命名空间                          │
│                                                              │
│  ┌────────────┐              ┌────────────┐                │
│  │  veth-in   │              │  veth-ex   │                │
│  └─────┬──────┘              └──────┬─────┘                │
│        │                            │                       │
│   ┌────▼─────────┐          ┌──────▼──────┐               │
│   │ TC Ingress   │          │ TC Egress   │               │
│   │ eBPF Program │          │ eBPF Program│               │
│   └────┬─────────┘          └──────┬──────┘               │
│        │                            │                       │
│        └────────────┬───────────────┘                       │
│                     │                                        │
│            ┌────────▼────────┐                              │
│            │  Linux Bridge   │                              │
│            └────────┬────────┘                              │
│                     │                                        │
│                ┌────▼─────┐                                 │
│                │ Physical │                                 │
│                │   NIC    │                                 │
│                └──────────┘                                 │
│                                                              │
│  ┌──────────────────────────────────────────────────┐          │
│  │          eBPF Maps (内核空间)                 │          │
│  │  ┌──────────────┐  ┌──────────────┐          │          │
│  │  │ Policy Map   │  │ Session Map  │          │          │
│  │  │ (HASH)       │  │ (LRU_HASH)   │          │          │
│  │  └──────────────┘  └──────────────┘          │          │
│  │  ┌──────────────┐  ┌──────────────┐          │          │
│  │  │ IP Range Map │  │  Stats Map   │          │          │
│  │  │ (LPM_TRIE)   │  │ (PERCPU_ARR) │          │          │
│  │  └──────────────┘  └──────────────┘          │          │
│  └──────────────────────────────────────────────────┘          │
└─────────────────────┬────────────────────────────────┬─┘
                      │                                    │
                      │ bpf syscall                        │
┌─────────────────────▼────────────────────────────────────▼─┐
│                    用户态控制平面                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │ Policy Mgmt  │  │   Monitor    │  │  DPI Engine  │    │
│  │   Service    │  │   Service    │  │  (Optional)  │    │
│  └──────────────┘  └──────────────┘  └──────────────┘    │
└──────────────────────────────────────────────────────────┘
```

### 1.2 数据流向

#### 出站流量（Egress）
```
Container eth0 → veth-in → TC Ingress eBPF
                              ↓
                    [策略查找 + 会话检查]
                              ↓
                    ┌─────────┴─────────┐
                    │                   │
                 ALLOW                 DROP
                    │                   │
                    ↓                   ↓
              veth-ex → Bridge      丢弃数据包
                    ↓
              物理网卡
```

#### 入站流量（Ingress）
```
物理网卡 → Bridge → veth-ex → TC Egress eBPF
                                  ↓
                      [策略查找 + 会话检查]
                                  ↓
                        ┌─────────┴─────────┐
                        │                   │
                     ALLOW                 DROP
                        │                   │
                        ↓                   ↓
                  veth-in → eth0        丢弃数据包
                        ↓
                   Container
```

---

## 2. eBPF数据结构设计

### 2.1 策略Map（Policy Map）

```c
// 策略键：5元组
struct policy_key {
    __u32 src_ip;        // 源IP地址
    __u32 dst_ip;        // 目标IP地址
    __u16 src_port;      // 源端口
    __u16 dst_port;      // 目标端口
    __u8  protocol;      // 协议 (TCP=6, UDP=17)
    __u8  direction;     // 方向 (0=egress, 1=ingress)
    __u16 padding;       // 对齐填充
} __attribute__((packed));

// 策略值
struct policy_value {
    __u32 policy_id;     // 策略ID
    __u8  action;        // 动作：0=ALLOW, 1=DENY, 2=LOG
    __u8  flags;         // 标志位
    __u16 reserved;      // 预留
    __u64 hit_count;     // 命中计数
    __u64 last_hit;      // 最后命中时间
} __attribute__((packed));

// 策略Map定义
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 100000);  // 最多10万条策略
    __type(key, struct policy_key);
    __type(value, struct policy_value);
} policy_map SEC(".maps");
```

### 2.2 会话Map（Session Map）

```c
// 会话键：5元组（不含方向）
struct session_key {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8  protocol;
    __u8  padding[3];    // 对齐
} __attribute__((packed));

// TCP连接状态 (RFC 793)
enum tcp_state {
    TCP_NONE = 0,           // 初始状态
    TCP_SYN_SENT,           // 客户端发送SYN
    TCP_SYN_RECV,           // 服务端收到SYN,发送SYN-ACK
    TCP_ESTABLISHED,        // 连接建立
    TCP_FIN_WAIT1,          // 主动关闭,发送FIN
    TCP_FIN_WAIT2,          // 收到FIN的ACK
    TCP_TIME_WAIT,          // 等待2MSL
    TCP_CLOSE,              // 连接关闭
    TCP_CLOSE_WAIT,         // 被动关闭,收到FIN
    TCP_LAST_ACK,           // 发送最后的ACK
    TCP_CLOSING,            // 同时关闭
    TCP_MAX
};

// 会话标志位
#define SESSION_FLAG_INGRESS       0x0001  // 入站会话
#define SESSION_FLAG_TLS_DETECTED  0x0002  // 检测到TLS
#define SESSION_FLAG_HTTP_DETECTED 0x0004  // 检测到HTTP
#define SESSION_FLAG_DPI_REQUIRED  0x0008  // 需要DPI
#define SESSION_FLAG_POLICY_CACHED 0x0010  // 策略已缓存

// 会话值
struct session_value {
    __u64 session_id;       // 会话ID（创建时间戳）
    __u64 packets;          // 数据包计数
    __u64 bytes;            // 字节计数
    __u64 last_seen;        // 最后活跃时间（bpf_ktime_get_ns）
    __u32 policy_id;        // 关联的策略ID
    __u8  action;           // 缓存的动作
    __u8  state;            // TCP状态 (enum tcp_state)
    __u16 flags;            // 标志位
    __u32 seq_init;         // TCP初始序列号
    __u32 ack_init;         // TCP初始确认号
    __u16 parser;           // 应用层协议解析器ID
    __u8  policy_ver;       // 策略版本（用于检测策略更新）
    __u8  reserved;         // 保留字段
} __attribute__((packed));

// 会话Map定义（使用LRU自动淘汰）
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 1000000);  // 最多100万会话
    __type(key, struct session_key);
    __type(value, struct session_value);
    __uint(map_flags, BPF_F_NO_COMMON_LRU);  // Per-CPU LRU,避免全局锁竞争
} session_map SEC(".maps");

// TCP状态转换逻辑
static __always_inline int tcp_state_transition(
    struct session_value *sess,
    struct tcphdr *tcp,
    bool is_ingress)
{
    __u8 flags = tcp->syn | (tcp->ack << 1) | (tcp->fin << 2) | (tcp->rst << 3);

    switch (sess->state) {
    case TCP_NONE:
        if (flags == 0x1) {  // SYN
            sess->state = TCP_SYN_SENT;
            sess->seq_init = bpf_ntohl(tcp->seq);
            return 0;
        }
        break;

    case TCP_SYN_SENT:
        if (flags == 0x3 && is_ingress) {  // SYN-ACK
            sess->state = TCP_SYN_RECV;
            sess->ack_init = bpf_ntohl(tcp->ack_seq);
            return 0;
        }
        if (flags == 0x8) {  // RST
            sess->state = TCP_CLOSE;
            return 0;
        }
        break;

    case TCP_SYN_RECV:
        if (flags == 0x2) {  // ACK
            sess->state = TCP_ESTABLISHED;
            return 0;
        }
        break;

    case TCP_ESTABLISHED:
        if (flags == 0x5) {  // FIN-ACK
            sess->state = TCP_FIN_WAIT1;
            return 0;
        }
        if (flags == 0x8) {  // RST
            sess->state = TCP_CLOSE;
            return 0;
        }
        break;

    case TCP_FIN_WAIT1:
        if (flags == 0x2) {  // ACK
            sess->state = TCP_FIN_WAIT2;
            return 0;
        }
        if (flags == 0x5 && is_ingress) {  // FIN-ACK
            sess->state = TCP_CLOSING;
            return 0;
        }
        break;

    case TCP_FIN_WAIT2:
        if (flags == 0x5 && is_ingress) {  // FIN-ACK
            sess->state = TCP_TIME_WAIT;
            return 0;
        }
        break;

    case TCP_CLOSE_WAIT:
        if (flags == 0x5) {  // FIN-ACK
            sess->state = TCP_LAST_ACK;
            return 0;
        }
        break;

    case TCP_LAST_ACK:
        if (flags == 0x2) {  // ACK
            sess->state = TCP_CLOSE;
            return 0;
        }
        break;

    case TCP_CLOSING:
        if (flags == 0x2) {  // ACK
            sess->state = TCP_TIME_WAIT;
            return 0;
        }
        break;

    default:
        break;
    }

    return -1;  // 非法状态转换
}
```

### 2.3 统计Map（Per-CPU Array）

```c
// 统计类型枚举
enum stat_type {
    STAT_TOTAL_PACKETS = 0,
    STAT_ALLOWED_PACKETS,
    STAT_DENIED_PACKETS,
    STAT_TOTAL_BYTES,
    STAT_ALLOWED_BYTES,
    STAT_DENIED_BYTES,
    STAT_NEW_SESSIONS,
    STAT_POLICY_LOOKUPS,
    STAT_SESSION_HITS,
    STAT_MAP_FULL_DROPS,        // Map满时丢弃的包数
    STAT_LRU_EVICTIONS,         // LRU淘汰次数
    STAT_TCP_SYN_FLOODS,        // SYN Flood攻击次数
    STAT_INVALID_STATE_TRANS,   // 非法TCP状态转换
    STAT_MAX
};

// 统计Map定义
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, STAT_MAX);
    __type(key, __u32);
    __type(value, __u64);
} stats_map SEC(".maps");

// 辅助函数：更新统计
static __always_inline void update_stats(__u32 type, __u64 val)
{
    __u64 *counter = bpf_map_lookup_elem(&stats_map, &type);
    if (counter)
        __sync_fetch_and_add(counter, val);
}
```

### 2.4 Map压力监控

```c
// Map压力监控结构
struct map_pressure {
    __u64 total_sessions;       // 当前会话总数（估算）
    __u64 evictions;            // LRU淘汰次数
    __u64 insertion_failures;   // 插入失败次数
    __u64 last_check;           // 最后检查时间
    __u32 pressure_level;       // 压力等级 (0-100)
    __u32 syn_sent_count;       // TCP_SYN_SENT状态的会话数
};

// Map压力监控Map
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct map_pressure);
} map_pressure_map SEC(".maps");

// 压力等级定义
#define PRESSURE_NORMAL   0   // < 70% 容量
#define PRESSURE_WARNING  1   // 70-90% 容量
#define PRESSURE_CRITICAL 2   // > 90% 容量

// 检查Map压力
static __always_inline int check_map_pressure(void)
{
    __u32 key = 0;
    struct map_pressure *pressure = bpf_map_lookup_elem(&map_pressure_map, &key);
    if (!pressure)
        return PRESSURE_NORMAL;

    // 如果Map使用率 > 90%,返回紧急状态
    if (pressure->total_sessions > 900000) {
        pressure->pressure_level = PRESSURE_CRITICAL;
        return PRESSURE_CRITICAL;
    }

    // 如果Map使用率 > 70%,返回警告状态
    if (pressure->total_sessions > 700000) {
        pressure->pressure_level = PRESSURE_WARNING;
        return PRESSURE_WARNING;
    }

    pressure->pressure_level = PRESSURE_NORMAL;
    return PRESSURE_NORMAL;
}

// SYN Flood防护：限制SYN_SENT状态的会话数
#define MAX_SYN_SENT_SESSIONS 10000

static __always_inline bool check_syn_flood(void)
{
    __u32 key = 0;
    struct map_pressure *pressure = bpf_map_lookup_elem(&map_pressure_map, &key);
    if (!pressure)
        return false;

    if (pressure->syn_sent_count > MAX_SYN_SENT_SESSIONS) {
        update_stats(STAT_TCP_SYN_FLOODS, 1);
        return true;  // 可能遭受SYN Flood攻击
    }

    return false;
}

// Map满载时的降级策略
static __always_inline int handle_map_full_scenario(
    struct __sk_buff *skb,
    struct session_key *key,
    struct session_value *val,
    struct policy_value *policy)
{
    // 尝试插入新会话
    int ret = bpf_map_update_elem(&session_map, key, val, BPF_NOEXIST);

    if (ret == -ENOSPC || ret == -E2BIG) {
        // Map已满,记录统计
        update_stats(STAT_MAP_FULL_DROPS, 1);

        // 降级策略1: 如果是白名单流量,允许通过但不建会话
        if (policy && policy->action == POLICY_ACTION_ALLOW) {
            // 允许通过,但不跟踪会话
            return TC_ACT_OK;
        }

        // 降级策略2: 如果是基础设施流量(DNS, ICMP),允许
        if (key->protocol == IPPROTO_ICMP ||
            key->dst_port == 53) {  // DNS
            return TC_ACT_OK;
        }

        // 降级策略3: 其他流量默认拒绝
        return TC_ACT_SHOT;
    }

    // 插入成功
    return ret;
}
```

### 2.5 IP范围匹配优化（LPM Trie）

```c
// LPM Trie键结构
struct lpm_key {
    __u32 prefixlen;    // 前缀长度 (0-32)
    __u32 ip;           // IP地址
} __attribute__((packed));

// IP范围策略值
struct ip_range_value {
    __u32 policy_id;
    __u8  action;
    __u8  priority;     // 优先级 (值越小优先级越高)
    __u16 reserved;
} __attribute__((packed));

// IP范围Map定义
struct {
    __uint(type, BPF_MAP_TYPE_LPM_TRIE);
    __uint(max_entries, 10000);  // 最多1万个IP段
    __uint(map_flags, BPF_F_NO_PREALLOC);
    __type(key, struct lpm_key);
    __type(value, struct ip_range_value);
} ip_range_map SEC(".maps");

// 混合策略查找：优先精确匹配,再IP段匹配
static __always_inline struct policy_value *
lookup_policy_optimized(struct packet_info *pkt)
{
    struct policy_key exact_key = {
        .src_ip = pkt->sip,
        .dst_ip = pkt->dip,
        .src_port = pkt->sport,
        .dst_port = pkt->dport,
        .protocol = pkt->proto,
    };

    // 快速路径1: 精确匹配 (Hash O(1), ~50ns)
    struct policy_value *val = bpf_map_lookup_elem(&policy_map, &exact_key);
    if (val)
        return val;

    // 快速路径2: IP范围匹配 (LPM Trie O(log n), ~200-500ns)
    struct lpm_key lpm_key = {
        .prefixlen = 32,
        .ip = pkt->dip,
    };
    struct ip_range_value *range_val = bpf_map_lookup_elem(&ip_range_map, &lpm_key);
    if (range_val) {
        // 将IP段策略转换为标准策略格式
        static struct policy_value range_policy;
        range_policy.policy_id = range_val->policy_id;
        range_policy.action = range_val->action;
        return &range_policy;
    }

    // 慢速路径3: 默认策略
    return &default_policy;
}

// 性能对比
// - 精确匹配: ~50ns, 适用于点对点策略
// - IP段匹配: ~200-500ns, 适用于网段策略
// - 线性扫描: 不可用 (eBPF Verifier不允许无界循环)
```

---

## 3. 核心处理流程

### 3.1 完整数据包处理流程

```c
// 主处理函数
SEC("tc_ingress")
int tc_microsegment_ingress(struct __sk_buff *skb)
{
    // 1. 解析以太网头
    struct ethhdr *eth;
    if (skb->protocol != bpf_htons(ETH_P_IP))
        return TC_ACT_OK;  // 非IP包,放行

    // 2. 解析IP头
    struct iphdr *iph;
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    eth = data;
    if ((void *)(eth + 1) > data_end)
        return TC_ACT_SHOT;

    iph = (void *)(eth + 1);
    if ((void *)(iph + 1) > data_end)
        return TC_ACT_SHOT;

    // 3. 构建会话键
    struct session_key sess_key = {};
    sess_key.src_ip = iph->saddr;
    sess_key.dst_ip = iph->daddr;
    sess_key.protocol = iph->protocol;

    // 4. 解析L4层（TCP/UDP）
    struct tcphdr *tcp;
    struct udphdr *udp;

    if (iph->protocol == IPPROTO_TCP) {
        tcp = (void *)iph + (iph->ihl << 2);
        if ((void *)(tcp + 1) > data_end)
            return TC_ACT_SHOT;

        sess_key.src_port = bpf_ntohs(tcp->source);
        sess_key.dst_port = bpf_ntohs(tcp->dest);

    } else if (iph->protocol == IPPROTO_UDP) {
        udp = (void *)iph + (iph->ihl << 2);
        if ((void *)(udp + 1) > data_end)
            return TC_ACT_SHOT;

        sess_key.src_port = bpf_ntohs(udp->source);
        sess_key.dst_port = bpf_ntohs(udp->dest);

    } else {
        // ICMP等其他协议
        sess_key.src_port = 0;
        sess_key.dst_port = 0;
    }

    // 5. 查找会话
    struct session_value *sess = bpf_map_lookup_elem(&session_map, &sess_key);

    if (sess) {
        // ===== 快速路径: 会话命中 =====
        update_stats(STAT_SESSION_HITS, 1);

        // 更新会话统计
        __sync_fetch_and_add(&sess->packets, 1);
        __sync_fetch_and_add(&sess->bytes, skb->len);
        sess->last_seen = bpf_ktime_get_ns();

        // TCP状态跟踪
        if (iph->protocol == IPPROTO_TCP) {
            if (tcp_state_transition(sess, tcp, true) < 0) {
                update_stats(STAT_INVALID_STATE_TRANS, 1);
            }
        }

        // 应用缓存的策略
        if (sess->action == POLICY_ACTION_ALLOW) {
            update_stats(STAT_ALLOWED_PACKETS, 1);
            update_stats(STAT_ALLOWED_BYTES, skb->len);
            return TC_ACT_OK;
        } else {
            update_stats(STAT_DENIED_PACKETS, 1);
            return TC_ACT_SHOT;
        }

    } else {
        // ===== 慢速路径: 新会话 =====
        update_stats(STAT_NEW_SESSIONS, 1);

        // 6. 检查Map压力
        int pressure = check_map_pressure();
        if (pressure == PRESSURE_CRITICAL) {
            // Map接近满载,进入降级模式
            // (详见 handle_map_full_scenario 函数)
        }

        // 7. SYN Flood检测
        if (iph->protocol == IPPROTO_TCP && tcp->syn && !tcp->ack) {
            if (check_syn_flood()) {
                return TC_ACT_SHOT;  // 丢弃可疑SYN包
            }
        }

        // 8. 策略查找
        update_stats(STAT_POLICY_LOOKUPS, 1);

        struct packet_info pkt_info = {
            .sip = sess_key.src_ip,
            .dip = sess_key.dst_ip,
            .sport = sess_key.src_port,
            .dport = sess_key.dst_port,
            .proto = sess_key.protocol,
        };

        struct policy_value *policy = lookup_policy_optimized(&pkt_info);
        if (!policy)
            return TC_ACT_SHOT;  // 无策略,默认拒绝

        // 9. 创建会话
        struct session_value new_sess = {
            .session_id = bpf_ktime_get_ns(),
            .packets = 1,
            .bytes = skb->len,
            .last_seen = bpf_ktime_get_ns(),
            .policy_id = policy->policy_id,
            .action = policy->action,
            .state = TCP_NONE,
            .flags = SESSION_FLAG_POLICY_CACHED,
        };

        // TCP初始状态
        if (iph->protocol == IPPROTO_TCP && tcp->syn && !tcp->ack) {
            new_sess.state = TCP_SYN_SENT;
            new_sess.seq_init = bpf_ntohl(tcp->seq);
        }

        // 10. 插入会话到Map
        int ret = handle_map_full_scenario(skb, &sess_key, &new_sess, policy);
        if (ret == TC_ACT_SHOT)
            return TC_ACT_SHOT;

        // 11. 执行策略动作
        if (policy->action == POLICY_ACTION_ALLOW) {
            update_stats(STAT_ALLOWED_PACKETS, 1);
            update_stats(STAT_ALLOWED_BYTES, skb->len);
            return TC_ACT_OK;
        } else {
            update_stats(STAT_DENIED_PACKETS, 1);
            return TC_ACT_SHOT;
        }
    }
}
```

### 3.2 数据包处理流程图

```
┌─────────────────┐
│  数据包到达TC   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  解析以太网头   │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │ IPv4?  │
    └───┬────┘
        │ No → 放行
        │ Yes
        ▼
┌─────────────────┐
│   解析IP头      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 解析传输层(TCP/UDP)│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 构造会话键(5元组)│
└────────┬────────┘
         │
         ▼
    ┌────────────┐
    │查找会话缓存│
    └─────┬──────┘
          │
     ┌────┴────┐
     │ 命中?   │
     └─┬────┬──┘
   Yes │    │ No
       │    │
       │    ▼
       │ ┌──────────┐
       │ │查找策略表│
       │ └─────┬────┘
       │       │
       │       ▼
       │  ┌────────┐
       │  │创建会话│
       │  └────┬───┘
       │       │
       └───────┘
           │
           ▼
      ┌─────────┐
      │更新统计 │
      └────┬────┘
           │
           ▼
      ┌─────────┐
      │执行动作 │
      └────┬────┘
           │
      ┌────┴────┐
      │         │
   ALLOW      DENY
      │         │
      ▼         ▼
  TC_ACT_OK  TC_ACT_SHOT
```

---

## 4. 关键优化点

### 4.1 会话缓存

**优势**：
- 首包查找策略，后续包直接使用缓存
- LRU自动淘汰，无需手动管理
- 大幅减少策略查找开销

**实现**：
```c
// 快速路径：会话命中
if (session) {
    // 直接使用缓存的action
    return session->action == ALLOW ? TC_ACT_OK : TC_ACT_SHOT;
}

// 慢速路径：新会话
// 1. 查找策略
// 2. 创建会话缓存
// 3. 执行动作
```

### 4.2 Per-CPU统计

**优势**：
- 无锁设计，零竞争
- 每个CPU独立计数
- 用户态聚合时汇总

**实现**：
```c
static __always_inline void update_stats(__u32 type, __u64 val) {
    __u64 *counter = bpf_map_lookup_elem(&stats_map, &type);
    if (counter) {
        __sync_fetch_and_add(counter, val);
    }
}
```

### 4.3 双向流量关联

**设计**：
- 出站流量创建会话（正向5元组）
- 入站流量使用反向5元组查找
- 同一会话双向流量共享统计

**实现**：
```c
// Ingress (出站): 正向
sess_key = {src_ip, dst_ip, src_port, dst_port, proto};

// Egress (入站): 反向
sess_key = {dst_ip, src_ip, dst_port, src_port, proto};
```

---

**下一步**：查看 [ebpf-tc-implementation.md](./ebpf-tc-implementation.md) 了解实施指南。