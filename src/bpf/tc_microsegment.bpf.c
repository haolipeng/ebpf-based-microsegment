// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* TC eBPF program for microsegmentation with session tracking */

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_endian.h>

// TC action codes
#define TC_ACT_OK 0
#define TC_ACT_SHOT 2

// Ethernet protocol types
#define ETH_P_IP 0x0800

// Debug mode - disable for production to reduce latency
#define DEBUG_MODE 0

#include "headers/common_types.h"

char LICENSE[] SEC("license") = "GPL";

// Session tracking map - LRU_HASH for automatic eviction
// NOTE: TC 和 XDP 各自维护独立的会话表,不使用 pinning
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, MAX_ENTRIES_SESSION);
    __type(key, struct flow_key);
    __type(value, struct session_value);
} session_map SEC(".maps");

// Policy map for exact 5-tuple matching
// PINNED: TC 和 XDP 共享策略数据
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_ENTRIES_POLICY);
    __type(key, struct policy_key);
    __type(value, struct policy_value);
    __uint(pinning, LIBBPF_PIN_BY_NAME);  // 按名称固定到 /sys/fs/bpf/
} policy_map SEC(".maps");

// Wildcard policy map for policies with wildcards (0 = any)
// Uses ARRAY for linear search (slower but supports wildcards)
// PINNED: TC 和 XDP 共享策略数据
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, MAX_ENTRIES_WILDCARD_POLICY);
    __type(key, __u32);  // index
    __type(value, struct wildcard_policy);
    __uint(pinning, LIBBPF_PIN_BY_NAME);  // 按名称固定到 /sys/fs/bpf/
} wildcard_policy_map SEC(".maps");

// Statistics map (Per-CPU for lock-free updates)
// PINNED: TC 和 XDP 共享统计数据
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, STATS_MAX);
    __type(key, __u32);
    __type(value, __u64);
    __uint(pinning, LIBBPF_PIN_BY_NAME);  // 按名称固定到 /sys/fs/bpf/
} stats_map SEC(".maps");

// Ring buffer for flow events to user-space
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);  // 256KB ring buffer
} flow_events SEC(".maps");

// Helper: Update statistics counter (optimized - no error checking for speed)
static __always_inline void update_stats(__u32 key) {
    __u64 *count = bpf_map_lookup_elem(&stats_map, &key);
    if (count) {
        // Direct increment for per-CPU array (no atomic needed)
        *count += 1;
    }
}

// Include policy matching logic (requires update_stats to be defined first)
#include "headers/policy_match.h"

// Include flow processing logic (packet parsing)
#include "headers/flow_processing.h"

// Helper: Get current timestamp in nanoseconds
static __always_inline __u64 get_timestamp_ns() {
    return bpf_ktime_get_ns();
}

// Helper: Extract flow key from packet (TC-specific wrapper)
// TC 使用 struct __sk_buff,提供 data 和 data_end 指针
static __always_inline int extract_flow_key(struct __sk_buff *skb, struct flow_key *key) {
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    // 调用通用的流键提取函数 (在 flow_processing.h 中定义)
    return extract_flow_key_from_packet(data, data_end, key);
}

// Helper: Check if TCP connection is closing (FIN or RST)
// 检查 TCP 连接是否正在关闭 (FIN 或 RST 标志)
static __always_inline bool is_tcp_closing(struct __sk_buff *skb, struct flow_key *key) {
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    // 仅处理 TCP 协议
    if (key->protocol != IPPROTO_TCP)
        return false;

    // 解析以太网头
    __u16 eth_proto;
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return false;
    eth_proto = eth->h_proto;

    if (eth_proto != bpf_htons(ETH_P_IP))
        return false;

    // 解析 IPv4 头
    struct iphdr *iph = (void *)(eth + 1);
    if ((void *)(iph + 1) > data_end)
        return false;

    // 计算 TCP 头位置
    void *tcph_ptr = (void *)iph + (iph->ihl * 4);
    struct tcphdr *tcph = tcph_ptr;

    if ((void *)(tcph + 1) > data_end)
        return false;

    // 检查 FIN 或 RST 标志
    return (tcph->fin || tcph->rst);
}

// Helper: Push flow event to user-space via Ring Buffer
// Returns 0 on success, -1 on failure
static __always_inline int push_flow_event(
    struct flow_key *key,
    __u64 timestamp_ns,
    __u64 packet_count,
    __u64 byte_count,
    __u8 event_type,
    __u8 policy_action,
    __u32 policy_id,
    __u8 state,
    __u8 direction)
{
    // Reserve space in ring buffer (non-blocking)
    struct flow_event *event = bpf_ringbuf_reserve(&flow_events, sizeof(*event), 0);
    if (!event) {
        // Ring buffer full - event dropped (silent failure for performance)
        return -1;
    }

    // Populate event fields
    event->src_ip = key->src_ip;
    event->dst_ip = key->dst_ip;
    event->src_port = key->src_port;
    event->dst_port = key->dst_port;
    event->protocol = key->protocol;

    event->event_type = event_type;
    event->direction = direction;
    event->padding = 0;

    event->packet_count = packet_count;
    event->byte_count = byte_count;
    event->timestamp_ns = timestamp_ns;

    event->policy_id = policy_id;
    event->policy_action = policy_action;
    event->state = state;
    event->reserved = 0;

    // Submit to ring buffer (non-blocking, will not fail)
    bpf_ringbuf_submit(event, 0);

    return 0;
}

// Helper: Create new session (optimized - minimal initialization)
static __always_inline int create_session(struct flow_key *key, __u8 action, __u64 ts, __u32 packet_len, __u32 rule_id, __u8 direction) {
    struct session_value new_session = {
        .created_ts = ts,
        .last_seen_ts = ts,
        .packets_to_server = 1,       // First packet
        .packets_to_client = 0,
        .bytes_to_server = packet_len, // First packet bytes
        .bytes_to_client = 0,
        .state = SESSION_STATE_NEW,
        .tcp_state = TCP_STATE_CLOSED,
        .policy_action = action,
        .flags = 0,
    };

    int ret = bpf_map_update_elem(&session_map, key, &new_session, BPF_NOEXIST);
    if (ret == 0) {
        update_stats(STATS_NEW_SESSIONS);
        update_stats(STATS_ACTIVE_SESSIONS);  // 增加活跃会话计数

        // Push flow event for all new connections (ALLOW, DENY, LOG)
        // Control plane will handle filtering based on configuration
        push_flow_event(
            key,
            ts,
            1,                      // First packet
            packet_len,             // First packet bytes
            FLOW_EVENT_NEW,         // New connection
            action,                 // Policy action
            rule_id,                // Matched rule ID
            FLOW_STATE_ACTIVE,      // Initial state
            direction               // Actual packet direction (ingress/egress)
        );
    }

    return ret;
}

// Main TC program (optimized for minimal latency)
SEC("tc")
int tc_microsegment_filter(struct __sk_buff *skb) {
    struct flow_key key = {0};

    // Extract flow key from packet (fast path)
    if (extract_flow_key(skb, &key) < 0) {
        return TC_ACT_OK;  // Pass non-IP packets
    }

    // 检测数据包方向
    // ingress_ifindex != 0 表示 ingress (从外部进入)
    // ingress_ifindex == 0 表示 egress (从内部发出)
    __u8 direction = (skb->ingress_ifindex != 0) ? POLICY_DIR_INGRESS : POLICY_DIR_EGRESS;

    // Update total packets counter
    update_stats(STATS_TOTAL_PACKETS);

    // Update direction-specific counters
    if (direction == POLICY_DIR_INGRESS) {
        update_stats(STATS_INGRESS_PACKETS);
    } else {
        update_stats(STATS_EGRESS_PACKETS);
    }

    // Fast path: Lookup existing session (most common case)
    struct session_value *session = bpf_map_lookup_elem(&session_map, &key);

    if (session) {
        // HOT PATH: Existing session - use cached policy decision
        // This is the most performance-critical path (>99% of packets)

        __u8 action = session->policy_action;

        // Update session stats (inline for speed)
        session->last_seen_ts = get_timestamp_ns();
        session->packets_to_server += 1;
        session->bytes_to_server += skb->len;

        // 检测 TCP 连接关闭 (FIN 或 RST 标志)
        // 只在会话未标记为关闭时检测,避免重复计数
        if (session->state != SESSION_STATE_CLOSING &&
            session->state != SESSION_STATE_CLOSED) {
            if (is_tcp_closing(skb, &key)) {
                // 标记会话为关闭状态
                session->state = SESSION_STATE_CLOSING;
                // 增加已关闭会话计数
                update_stats(STATS_CLOSED_SESSIONS);
                // 减少活跃会话计数 (通过 Go 端周期性校正更准确)
#if DEBUG_MODE
                bpf_printk("TCP closing: %pI4:%d -> %pI4:%d\n",
                           &key.src_ip, bpf_ntohs(key.src_port),
                           &key.dst_ip, bpf_ntohs(key.dst_port));
#endif
            }
        }

        // Fast enforcement check
        if (action == POLICY_ACTION_DENY) {
            update_stats(STATS_DENIED_PACKETS);

            // Update direction-specific deny counters
            if (direction == POLICY_DIR_INGRESS) {
                update_stats(STATS_INGRESS_DENIED);
            } else {
                update_stats(STATS_EGRESS_DENIED);
            }

#if DEBUG_MODE
            bpf_printk("DENY: %pI4:%d -> %pI4:%d (cached, dir=%d)\n",
                       &key.src_ip, bpf_ntohs(key.src_port),
                       &key.dst_ip, bpf_ntohs(key.dst_port),
                       direction);
#endif
            return TC_ACT_SHOT;  // Drop packet
        }

        update_stats(STATS_ALLOWED_PACKETS);
        return TC_ACT_OK;  // Allow packet
    }

    // SLOW PATH: New session - lookup policy with wildcard support
    // This happens less frequently, so more overhead is acceptable

    __u64 now = get_timestamp_ns();
    __u32 matched_rule_id = 0;
    __u8 action = lookup_policy_action(&key, direction, &matched_rule_id);

#if DEBUG_MODE
    if (matched_rule_id != 0) {
        bpf_printk("Policy %d matched: %pI4:%d -> %pI4:%d action=%d dir=%d\n",
                   matched_rule_id,
                   &key.src_ip, bpf_ntohs(key.src_port),
                   &key.dst_ip, bpf_ntohs(key.dst_port),
                   action,
                   direction);
    }
#endif

    // Create new session with policy action (includes first packet stats)
    create_session(&key, action, now, skb->len, matched_rule_id, direction);

    // Enforce policy
    if (action == POLICY_ACTION_DENY) {
        update_stats(STATS_DENIED_PACKETS);

        // Update direction-specific deny counters
        if (direction == POLICY_DIR_INGRESS) {
            update_stats(STATS_INGRESS_DENIED);
        } else {
            update_stats(STATS_EGRESS_DENIED);
        }

#if DEBUG_MODE
        bpf_printk("DENY: %pI4:%d -> %pI4:%d (new, dir=%d)\n",
                   &key.src_ip, bpf_ntohs(key.src_port),
                   &key.dst_ip, bpf_ntohs(key.dst_port),
                   direction);
#endif
        return TC_ACT_SHOT;  // Drop packet
    }

    update_stats(STATS_ALLOWED_PACKETS);
    return TC_ACT_OK;  // Allow packet
}
