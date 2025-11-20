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
#define ETH_P_IPV6 0x86DD

// Feature flags with default values (can be overridden via -D flags)

// Debug mode - disable for production to reduce latency
#ifndef DEBUG_MODE
#define DEBUG_MODE 0
#endif

// Enable IP fragment handling
// Set to 1 to enable fragment detection and tracking (recommended for production)
// Set to 0 to disable fragment handling (reduces complexity by ~2000 instructions)
#ifndef ENABLE_IP_FRAGMENT_HANDLING
#define ENABLE_IP_FRAGMENT_HANDLING 1
#endif

// Enable NAT (Network Address Translation) support
// Set to 1 to enable NAT detection and address restoration (recommended for Docker/K8s)
// Set to 0 to disable NAT support (reduces complexity by ~1500 instructions)
#ifndef ENABLE_NAT_SUPPORT
#define ENABLE_NAT_SUPPORT 1
#endif

// Enable protocol-indexed wildcard lookup
// Set to 1 to use indexed lookup (better performance for 200+ policies)
// Set to 0 to use legacy linear scan (simpler, works for < 50 policies)
#ifndef USE_INDEXED_LOOKUP
#define USE_INDEXED_LOOKUP 1
#endif

#include "headers/common_types.h"
#include "headers/tcp_state_machine.h"
#include "headers/process_monitor.h"  // Issue #47: Process monitoring support

// Conditionally include headers based on feature flags
#if ENABLE_IP_FRAGMENT_HANDLING
#include "headers/fragment_tracking.h"
#endif

#if ENABLE_NAT_SUPPORT
#include "headers/nat_support.h"
#endif

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

// Protocol offset map for indexed wildcard lookup
// Maps protocol number to segment descriptor (start index + count)
// PINNED: TC 和 XDP 共享索引数据
// Note: struct protocol_segment is defined in common_types.h
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 256);  // 256 possible protocol numbers (0-255)
    __type(key, __u32);        // protocol number
    __type(value, struct protocol_segment);
    __uint(pinning, LIBBPF_PIN_BY_NAME);  // 按名称固定到 /sys/fs/bpf/
} protocol_offset_map SEC(".maps");

// Statistics map (Per-CPU for lock-free updates)
// PINNED: TC 和 XDP 共享统计数据
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, STATS_MAX);
    __type(key, __u32);
    __type(value, __u64);
    __uint(pinning, LIBBPF_PIN_BY_NAME);  // 按名称固定到 /sys/fs/bpf/
} stats_map SEC(".maps");

// Timeout configuration map (shared between TC and XDP)
// PINNED: Allows user-space to configure session timeout values
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, TIMEOUT_CONFIG_MAX);
    __type(key, __u32);  // enum timeout_config_key
    __type(value, __u64); // timeout in nanoseconds
    __uint(pinning, LIBBPF_PIN_BY_NAME);  // 按名称固定到 /sys/fs/bpf/
} timeout_config_map SEC(".maps");

// Ring buffer for flow events to user-space
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);  // 256KB ring buffer
} flow_events SEC(".maps");

// Issue #47: process_info_map is defined in process_monitor.h (included above)
// No need for extern declaration here - it's already available

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

// Include process-aware policy matching (Issue #47)
#include "headers/process_policy_match.h"

// Include indexed policy matching (optional, controlled by USE_INDEXED_LOOKUP)
#if USE_INDEXED_LOOKUP
#include "headers/indexed_policy_match_v2.h"
#endif

// Include flow processing logic (packet parsing)
#include "headers/flow_processing.h"

// Include fragment processing logic (after flow_processing.h)
#include "headers/fragment_handler.h"

// Helper: Get current timestamp in nanoseconds
static __always_inline __u64 get_timestamp_ns() {
    return bpf_ktime_get_ns();
}

// Helper: Extract flow key from packet
// TC 使用 struct __sk_buff,提供 data 和 data_end 指针
static __always_inline int extract_flow_key(struct __sk_buff *skb, struct flow_key *key) {
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    return extract_flow_key_from_packet(data, data_end, key);
}

// Helper: Update TCP state machine - IPv4/IPv6 support
// 更新 TCP 状态机
static __always_inline __u8 update_tcp_state(struct __sk_buff *skb, struct flow_key *key, __u8 current_state) {
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    // 仅处理 TCP 协议
    if (key->protocol != IPPROTO_TCP)
        return current_state;

    // 解析以太网头
    __u16 eth_proto;
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return current_state;
    eth_proto = eth->h_proto;

    void *tcph_ptr = NULL;

    // 根据 IP 版本解析不同的 IP 头
    if (eth_proto == bpf_htons(ETH_P_IP)) {
        // IPv4
        struct iphdr *iph = (void *)(eth + 1);
        if ((void *)(iph + 1) > data_end)
            return current_state;

        // 计算 TCP 头位置
        tcph_ptr = (void *)iph + (iph->ihl * 4);

    } else if (eth_proto == bpf_htons(ETH_P_IPV6)) {
        // IPv6
        struct ipv6hdr *ip6h = (void *)(eth + 1);
        if ((void *)(ip6h + 1) > data_end)
            return current_state;

        // TCP 头位置 (IPv6 固定 40 字节头部，不考虑扩展头)
        tcph_ptr = (void *)(ip6h + 1);

    } else {
        return current_state;
    }

    // 验证 TCP 头边界
    struct tcphdr *tcph = tcph_ptr;
    if ((void *)(tcph + 1) > data_end)
        return current_state;

    // 提取 TCP 标志
    __u8 flags = get_tcp_flags(tcph);

    // TC 程序无法可靠区分出站/入站数据包方向
    // 因此我们采用简化的状态转换:
    // - 如果看到 SYN: CLOSED -> SYN_RECV (假设是入站)
    // - 如果看到 SYN+ACK: SYN_SENT -> ESTABLISHED 或 SYN_RECV -> ESTABLISHED
    // - 如果看到 FIN: ESTABLISHED -> FIN_WAIT1/CLOSE_WAIT
    // - 如果看到 RST: 任何状态 -> CLOSED

    // 使用入站转换逻辑作为主要路径
    __u8 new_state = tcp_state_transition_inbound(current_state, flags);

    // 如果状态未变化,尝试出站转换 (覆盖更多场景)
    if (new_state == current_state) {
        new_state = tcp_state_transition_outbound(current_state, flags);
    }

    return new_state;
}

// Helper: Extract TCP details from packet (TC version)
// Returns 0 on success, -1 on failure
static __always_inline int extract_tcp_details(
    struct __sk_buff *skb,
    struct flow_key *key,
    __u32 *seq,
    __u32 *ack,
    __u16 *window,
    __u8 *flags)
{
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    // Only process TCP protocol
    if (key->protocol != IPPROTO_TCP)
        return -1;

    // Parse Ethernet header (skip VLAN if present)
    __u16 eth_proto, vlan_id;
    void *l3 = parse_ethernet(data, data_end, &eth_proto, &vlan_id);
    if (!l3)
        return -1;

    void *tcph_ptr = NULL;

    // Locate TCP header based on IP version
    if (eth_proto == bpf_htons(ETH_P_IP)) {
        struct iphdr *iph = l3;
        if ((void *)(iph + 1) > data_end)
            return -1;
        tcph_ptr = (void *)iph + (iph->ihl * 4);
    } else if (eth_proto == bpf_htons(ETH_P_IPV6)) {
        struct ipv6hdr *ip6h = l3;
        if ((void *)(ip6h + 1) > data_end)
            return -1;
        tcph_ptr = (void *)(ip6h + 1);
    } else {
        return -1;
    }

    // Validate TCP header
    struct tcphdr *tcph = tcph_ptr;
    if ((void *)(tcph + 1) > data_end)
        return -1;

    // Extract TCP fields
    *seq = bpf_ntohl(tcph->seq);
    *ack = bpf_ntohl(tcph->ack_seq);
    *window = bpf_ntohs(tcph->window);
    *flags = get_tcp_flags(tcph);

    return 0;
}

// Helper: Push flow event to user-space via Ring Buffer (Enhanced with VLAN and TCP tracking)
// Returns 0 on success, -1 on failure
// Issue #47: Added proc_info parameter for process-level policy support
static __always_inline int push_flow_event(
    struct __sk_buff *skb,
    struct flow_key *key,
    __u64 timestamp_ns,
    __u64 packet_count,
    __u64 byte_count,
    __u8 event_type,
    __u8 policy_action,
    __u32 policy_id,
    __u8 state,
    __u8 direction,
    __u8 tcp_state,
    __u8 conn_flags,
    struct process_match_info *proc_info)  // Issue #47: Process information
{
    // Reserve space in ring buffer (non-blocking)
    struct flow_event *event = bpf_ringbuf_reserve(&flow_events, sizeof(*event), 0);
    if (!event) {
        // Ring buffer full - event dropped (silent failure for performance)
        return -1;
    }

    // Populate event fields - IPv4/IPv6 + VLAN support
    #pragma unroll
    for (int i = 0; i < 4; i++) {
        event->src_ip[i] = key->src_ip[i];
        event->dst_ip[i] = key->dst_ip[i];
    }
    event->src_port = key->src_port;
    event->dst_port = key->dst_port;
    event->protocol = key->protocol;

    // Event metadata
    event->event_type = event_type;
    event->direction = direction;
    event->ip_version = key->ip_version;
    event->vlan_id = key->vlan_id;
    event->flags = conn_flags;

    // Extract TCP details if TCP protocol
    __u32 tcp_seq = 0, tcp_ack = 0;
    __u16 tcp_window = 0;
    __u8 tcp_flags = 0;
    if (key->protocol == IPPROTO_TCP) {
        extract_tcp_details(skb, key, &tcp_seq, &tcp_ack, &tcp_window, &tcp_flags);
    }
    event->tcp_flags = tcp_flags;

    // Statistics
    event->packet_count = packet_count;
    event->byte_count = byte_count;
    event->timestamp_ns = timestamp_ns;

    // Enhanced TCP tracking
    event->tcp_seq = tcp_seq;
    event->tcp_ack = tcp_ack;
    event->tcp_window = tcp_window;
    event->tcp_retrans = 0;  // Will be updated by session tracking
    event->tcp_state = tcp_state;

    // Policy context
    event->policy_id = policy_id;
    event->policy_action = policy_action;
    event->state = state;
    event->reserved = 0;

    // Issue #47: Fill process context fields
    if (proc_info) {
        fill_flow_event_process_fields(event, proc_info);
    } else {
        // No process info available - zero out process fields
        event->process_name[0] = '\0';
        event->pid = 0;
        event->container_id[0] = '\0';
        event->process_exec_time = 0;
    }

    // Submit to ring buffer (non-blocking, will not fail)
    bpf_ringbuf_submit(event, 0);

    return 0;
}

// Helper: Create new session (optimized - minimal initialization)
// Issue #47: Added proc_info parameter for process-level policy support
static __always_inline int create_session(struct __sk_buff *skb, struct flow_key *key, __u8 action, __u64 ts, __u32 packet_len, __u32 rule_id, __u8 direction, struct process_match_info *proc_info) {
    // 初始化 TCP 状态 (根据第一个数据包的标志)
    __u8 initial_tcp_state = TCP_STATE_CLOSED;
    if (key->protocol == IPPROTO_TCP) {
        // 从 CLOSED 状态开始转换,模拟收到第一个数据包
        initial_tcp_state = update_tcp_state(skb, key, TCP_STATE_CLOSED);
    }

    struct session_value new_session = {
        .created_ts = ts,
        .last_seen_ts = ts,
        .packets_to_server = 1,       // First packet
        .packets_to_client = 0,
        .bytes_to_server = packet_len, // First packet bytes
        .bytes_to_client = 0,
        .state = SESSION_STATE_NEW,
        .tcp_state = initial_tcp_state,  // 使用计算得到的 TCP 状态
        .policy_action = action,
        .flags = 0,
    };

    int ret = bpf_map_update_elem(&session_map, key, &new_session, BPF_NOEXIST);
    if (ret == 0) {
        update_stats(STATS_NEW_SESSIONS);
        update_stats(STATS_ACTIVE_SESSIONS);  // 增加活跃会话计数

        // Push flow event for all new connections (ALLOW, DENY, LOG)
        // Control plane will handle filtering based on configuration
        // Issue #47: Include process information for new connections
        push_flow_event(
            skb,                    // sk_buff context
            key,
            ts,
            1,                      // First packet
            packet_len,             // First packet bytes
            FLOW_EVENT_NEW,         // New connection
            action,                 // Policy action
            rule_id,                // Matched rule ID
            FLOW_STATE_ACTIVE,      // Initial state
            direction,              // Actual packet direction (ingress/egress)
            initial_tcp_state,      // TCP state
            0,                      // Connection flags (initial state)
            proc_info               // Issue #47: Process information
        );
    }

    return ret;
}

#if ENABLE_IP_FRAGMENT_HANDLING
/* process_ip_fragment - Unified IPv4/IPv6 fragment processing
 *
 * Extracts fragment detection logic into a dedicated function to keep
 * the main processing flow clean and readable.
 *
 * @skb: Socket buffer context
 * @key: Flow key (may be incomplete for fragments)
 * @action: Output - policy action determined after lookup (for first fragments)
 * @is_first_fragment: Output - set to true if this is a first fragment
 *
 * Returns:
 *   TC_ACT_OK   - Allow packet (non-fragment or allowed first fragment)
 *   TC_ACT_SHOT - Drop packet (denied fragment or subsequent fragment)
 *   -1          - Not a fragment, continue normal processing
 *
 * Fragment Processing Logic:
 * 1. Detect if packet is fragmented (IPv4 or IPv6)
 * 2. For non-fragments: return -1 (caller handles normal flow)
 * 3. For first fragments: set is_first_fragment flag, return -1 (caller does policy lookup and caching)
 * 4. For subsequent fragments: look up cached policy and enforce immediately
 */
static __noinline int process_ip_fragment(
    struct __sk_buff *skb,
    struct flow_key *key,
    __u8 *action,
    bool *is_first_fragment)
{
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    // Parse Ethernet header
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end) {
        return TC_ACT_OK;  // Invalid packet
    }
    __u16 eth_proto = eth->h_proto;

    // IPv4 Fragment Detection and Handling
    if (eth_proto == bpf_htons(ETH_P_IP)) {
        struct iphdr *iph = (struct iphdr *)(eth + 1);
        if ((void *)(iph + 1) > data_end) {
            return TC_ACT_OK;  // Invalid packet
        }

        // Check if packet is fragmented (single optimized call)
        enum ipv4_frag_type frag_type = get_ipv4_frag_type(iph);

        if (frag_type != IPV4_FRAG_TYPE_NONE) {
            if (frag_type == IPV4_FRAG_TYPE_FIRST) {
                // First fragment: has L4 headers, continue to policy matching
                // Set flag so caller can cache policy decision
                *is_first_fragment = true;
                update_frag_stats(&frag_stats_map, FRAG_STAT_TOTAL);
                return -1;  // Continue normal processing
            } else {  // IPV4_FRAG_TYPE_SUBSEQUENT
                // Subsequent fragment: no L4 headers, look up cached policy
                struct frag_key fkey = {0};
                extract_ipv4_frag_key(iph, &fkey);
                update_frag_stats(&frag_stats_map, FRAG_STAT_TOTAL);

                struct frag_value *fval = bpf_map_lookup_elem(&frag_state_map, &fkey);
                if (fval) {
                    // Cache hit: use cached policy action
                    __u32 config_key = 0;
                    struct frag_config *config = bpf_map_lookup_elem(&frag_config_map, &config_key);
                    __u8 mode = config ? config->mode : FRAG_MODE_NORMAL;

                    if (mode == FRAG_MODE_NORMAL) {
                        // NORMAL mode: deny subsequent fragments
                        update_frag_stats(&frag_stats_map, FRAG_STAT_DENIED);
                        update_stats(STATS_DENIED_PACKETS);
                        return TC_ACT_SHOT;
                    } else if (mode == FRAG_MODE_PERMISSIVE && fval->policy_action == POLICY_ACTION_ALLOW) {
                        // PERMISSIVE mode: allow if first fragment was allowed
                        update_frag_stats(&frag_stats_map, FRAG_STAT_ALLOWED);
                        update_stats(STATS_ALLOWED_PACKETS);
                        return TC_ACT_OK;
                    } else {
                        // Deny otherwise
                        update_frag_stats(&frag_stats_map, FRAG_STAT_DENIED);
                        update_stats(STATS_DENIED_PACKETS);
                        return TC_ACT_SHOT;
                    }
                } else {
                    // Cache miss: first fragment not seen or timed out, deny for safety
                    update_frag_stats(&frag_stats_map, FRAG_STAT_DENIED);
                    update_stats(STATS_DENIED_PACKETS);
                    return TC_ACT_SHOT;
                }
            }
        }
    }
    // IPv6 Fragment Detection and Handling
    else if (eth_proto == bpf_htons(ETH_P_IPV6)) {
        struct ipv6hdr *ip6h = (struct ipv6hdr *)(eth + 1);
        if ((void *)(ip6h + 1) > data_end) {
            return TC_ACT_OK;  // Invalid packet
        }

        // Check if packet has fragment extension header
        if (is_ipv6_fragment(ip6h->nexthdr)) {
            struct ipv6_frag_hdr *frag_hdr = (struct ipv6_frag_hdr *)(ip6h + 1);
            if ((void *)(frag_hdr + 1) > data_end) {
                return TC_ACT_OK;  // Invalid packet
            }

            if (is_ipv6_first_fragment(frag_hdr)) {
                // First fragment: continue to policy matching
                *is_first_fragment = true;
                update_frag_stats(&frag_stats_map, FRAG_STAT_TOTAL);
                return -1;  // Continue normal processing
            } else if (is_ipv6_subsequent_fragment(frag_hdr)) {
                // Subsequent fragment: look up cached policy
                struct frag_key fkey = {0};
                extract_ipv6_frag_key(ip6h, frag_hdr, &fkey);
                update_frag_stats(&frag_stats_map, FRAG_STAT_TOTAL);

                struct frag_value *fval = bpf_map_lookup_elem(&frag_state_map, &fkey);
                if (fval) {
                    // Cache hit: use cached policy action
                    __u32 config_key = 0;
                    struct frag_config *config = bpf_map_lookup_elem(&frag_config_map, &config_key);
                    __u8 mode = config ? config->mode : FRAG_MODE_NORMAL;

                    if (mode == FRAG_MODE_NORMAL) {
                        update_frag_stats(&frag_stats_map, FRAG_STAT_DENIED);
                        update_stats(STATS_DENIED_PACKETS);
                        return TC_ACT_SHOT;
                    } else if (mode == FRAG_MODE_PERMISSIVE && fval->policy_action == POLICY_ACTION_ALLOW) {
                        update_frag_stats(&frag_stats_map, FRAG_STAT_ALLOWED);
                        update_stats(STATS_ALLOWED_PACKETS);
                        return TC_ACT_OK;
                    } else {
                        update_frag_stats(&frag_stats_map, FRAG_STAT_DENIED);
                        update_stats(STATS_DENIED_PACKETS);
                        return TC_ACT_SHOT;
                    }
                } else {
                    // Cache miss: first fragment not seen or timed out, deny for safety
                    update_frag_stats(&frag_stats_map, FRAG_STAT_DENIED);
                    update_stats(STATS_DENIED_PACKETS);
                    return TC_ACT_SHOT;
                }
            }
        }
    }

    // Not a fragment
    return -1;
}
#endif /* ENABLE_IP_FRAGMENT_HANDLING */

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

        // 更新 TCP 状态机 (仅对 TCP 协议)
        if (key.protocol == IPPROTO_TCP) {
            __u8 old_tcp_state = session->tcp_state;
            session->tcp_state = update_tcp_state(skb, &key, old_tcp_state);

            // 检测 TCP 连接关闭状态
            if (is_tcp_state_closing(session->tcp_state) &&
                session->state != SESSION_STATE_CLOSING &&
                session->state != SESSION_STATE_CLOSED) {
                // 标记会话为关闭状态
                session->state = SESSION_STATE_CLOSING;
                // 增加已关闭会话计数
                update_stats(STATS_CLOSED_SESSIONS);

                // Push connection closed event to control plane
                push_flow_event(
                    skb,                // sk_buff context
                    &key,
                    session->last_seen_ts,
                    session->packets_to_server + session->packets_to_client,
                    session->bytes_to_server + session->bytes_to_client,
                    FLOW_EVENT_CLOSED,
                    session->policy_action,
                    0,                  // policy_id not tracked in session
                    SESSION_STATE_CLOSING,
                    direction,
                    session->tcp_state, // TCP state
                    session->flags,     // Connection flags
                    NULL                // Issue #47: No proc_info in HOT PATH (performance)
                );

#if DEBUG_MODE
                bpf_printk("TCP state transition: %pI4:%d -> %pI4:%d (%d -> %d)\n",
                           &key.src_ip, bpf_ntohs(key.src_port),
                           &key.dst_ip, bpf_ntohs(key.dst_port),
                           old_tcp_state, session->tcp_state);
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

#if ENABLE_IP_FRAGMENT_HANDLING
    // Fragment Detection: Handle fragmented packets (IPv4/IPv6)
    // - Non-fragments: continue to policy matching
    // - First fragments: continue to policy matching, cache result after
    // - Subsequent fragments: use cached policy (handled in process_ip_fragment)
    bool is_first_fragment = false;
    __u8 frag_action = POLICY_ACTION_DENY;  // Temporary for fragment processing

    int frag_result = process_ip_fragment(skb, &key, &frag_action, &is_first_fragment);
    if (frag_result != -1) {
        // Fragment was handled (subsequent fragment or error), return immediately
        return frag_result;
    }
    // If frag_result == -1: not a fragment or first fragment, continue to policy matching
#endif /* ENABLE_IP_FRAGMENT_HANDLING */

    // NAT Detection: Restore original addresses for policy matching
    // This enables correct policy matching in Docker/Kubernetes environments
    struct flow_key original_key = key;  // Default to current key
    __u8 nat_type = NAT_TYPE_NONE;

    detect_nat_and_restore_with_maps(
        skb,                       // Context (TC __sk_buff)
        false,                     // is_xdp = false (this is TC)
        &key,                      // Current flow key (post-NAT)
        &original_key,             // Output: original flow key (pre-NAT)
        &nat_type,                 // Output: NAT type detected
        &nat_config_map,           // NAT configuration map
        &conntrack_cache_map,      // Conntrack cache map
        &nat_stats_map             // NAT statistics map
    );

    // Issue #47: Get current process information for process-level policy matching
    struct process_match_info proc_info = {0};
    get_current_process_info(&proc_info);
    lookup_process_cache(&proc_info, &process_info_map);

    // Use original addresses for policy matching (pre-NAT addresses)
    // This ensures policies match the actual source/destination, not NAT'd addresses
    // Issue #47: Use process-aware policy matching
    __u8 action = lookup_policy_action_with_process(
        &original_key, &proc_info, direction, &matched_rule_id,
        &policy_map, &wildcard_policy_map);

    // Note: INDEX lookup is disabled for now to support process policies
    // Future optimization: extend indexed lookup to support process fields
// #if USE_INDEXED_LOOKUP
//     __u8 action = lookup_policy_action_indexed(&original_key, direction, &matched_rule_id);
// #else
//     __u8 action = lookup_policy_action(&original_key, direction, &matched_rule_id);
// #endif

#if DEBUG_MODE
    // Log NAT detection if NAT is present
    if (nat_type != NAT_TYPE_NONE) {
        bpf_printk("NAT detected: type=%d, orig=%pI4:%d->%pI4:%d, nat=%pI4:%d->%pI4:%d\n",
                   nat_type,
                   &original_key.src_ip[3], bpf_ntohs(original_key.src_port),
                   &original_key.dst_ip[3], bpf_ntohs(original_key.dst_port),
                   &key.src_ip[3], bpf_ntohs(key.src_port),
                   &key.dst_ip[3], bpf_ntohs(key.dst_port));
    }

    if (matched_rule_id != 0) {
        bpf_printk("Policy %d matched: %pI4:%d -> %pI4:%d action=%d dir=%d NAT=%d\n",
                   matched_rule_id,
                   &original_key.src_ip[3], bpf_ntohs(original_key.src_port),
                   &original_key.dst_ip[3], bpf_ntohs(original_key.dst_port),
                   action,
                   direction,
                   nat_type);
    }
#endif

    // Create new session with policy action (includes first packet stats)
    // Issue #47: Pass process information to create_session
    create_session(skb, &key, action, now, skb->len, matched_rule_id, direction, &proc_info);

#if ENABLE_IP_FRAGMENT_HANDLING
    // Cache fragment state for first fragments (after policy lookup)
    if (is_first_fragment) {
        void *data = (void *)(long)skb->data;
        void *data_end = (void *)(long)skb->data_end;
        struct ethhdr *eth = data;
        if ((void *)(eth + 1) <= data_end) {
            __u16 eth_proto = eth->h_proto;
            struct frag_key fkey = {0};
            bool cache_success = false;

            // Extract fragment key based on IP version
            if (eth_proto == bpf_htons(ETH_P_IP)) {
                struct iphdr *iph = (struct iphdr *)(eth + 1);
                if ((void *)(iph + 1) <= data_end) {
                    extract_ipv4_frag_key(iph, &fkey);
                    cache_success = true;
                }
            } else if (eth_proto == bpf_htons(ETH_P_IPV6)) {
                struct ipv6hdr *ip6h = (struct ipv6hdr *)(eth + 1);
                struct ipv6_frag_hdr *frag_hdr = (struct ipv6_frag_hdr *)(ip6h + 1);
                if ((void *)(frag_hdr + 1) <= data_end) {
                    extract_ipv6_frag_key(ip6h, frag_hdr, &fkey);
                    cache_success = true;
                }
            }

            // Cache policy decision for subsequent fragments
            if (cache_success) {
                struct frag_value fval = {0};
                __builtin_memcpy(&fval.complete_key, &key, sizeof(struct flow_key));
                fval.policy_action = action;
                fval.timestamp = now;
                bpf_map_update_elem(&frag_state_map, &fkey, &fval, BPF_ANY);

                // Update statistics
                if (action == POLICY_ACTION_ALLOW) {
                    update_frag_stats(&frag_stats_map, FRAG_STAT_ALLOWED);
                } else {
                    update_frag_stats(&frag_stats_map, FRAG_STAT_DENIED);
                }
            }
        }
    }
#endif /* ENABLE_IP_FRAGMENT_HANDLING */

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
