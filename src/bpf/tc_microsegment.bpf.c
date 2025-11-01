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
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, MAX_ENTRIES_SESSION);
    __type(key, struct flow_key);
    __type(value, struct session_value);
} session_map SEC(".maps");

// Policy map for exact 5-tuple matching
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_ENTRIES_POLICY);
    __type(key, struct policy_key);
    __type(value, struct policy_value);
} policy_map SEC(".maps");

// Wildcard policy map for policies with wildcards (0 = any)
// Uses ARRAY for linear search (slower but supports wildcards)
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, MAX_ENTRIES_WILDCARD_POLICY);
    __type(key, __u32);  // index
    __type(value, struct wildcard_policy);
} wildcard_policy_map SEC(".maps");

// Statistics map (Per-CPU for lock-free updates)
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, STATS_MAX);
    __type(key, __u32);
    __type(value, __u64);
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

// Helper: Get current timestamp in nanoseconds
static __always_inline __u64 get_timestamp_ns() {
    return bpf_ktime_get_ns();
}

// Helper: Extract flow key from packet
static __always_inline int extract_flow_key(struct __sk_buff *skb, struct flow_key *key) {
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;
    
    // Parse Ethernet header
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return -1;
    
    // Only handle IPv4 for now
    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return -1;
    
    // Parse IP header
    struct iphdr *iph = (void *)(eth + 1);
    if ((void *)(iph + 1) > data_end)
        return -1;
    
    key->src_ip = iph->saddr;
    key->dst_ip = iph->daddr;
    key->protocol = iph->protocol;
    
    // Parse transport layer
    void *l4 = (void *)iph + (iph->ihl * 4);
    
    if (iph->protocol == IPPROTO_TCP) {
        struct tcphdr *tcph = l4;
        if ((void *)(tcph + 1) > data_end)
            return -1;
        key->src_port = tcph->source;
        key->dst_port = tcph->dest;
    } else if (iph->protocol == IPPROTO_UDP) {
        struct udphdr *udph = l4;
        if ((void *)(udph + 1) > data_end)
            return -1;
        key->src_port = udph->source;
        key->dst_port = udph->dest;
    } else {
        // ICMP or other protocols
        key->src_port = 0;
        key->dst_port = 0;
    }
    
    return 0;
}

// Helper: Check if flow matches wildcard policy
static __always_inline bool matches_wildcard(
    struct flow_key *key,
    struct wildcard_policy *wildcard)
{
    // IP matching with masks
    if ((key->src_ip & wildcard->src_ip_mask) !=
        (wildcard->src_ip & wildcard->src_ip_mask))
        return false;

    if ((key->dst_ip & wildcard->dst_ip_mask) !=
        (wildcard->dst_ip & wildcard->dst_ip_mask))
        return false;

    // Port matching (0 = wildcard, matches any)
    if (wildcard->src_port != 0 && key->src_port != wildcard->src_port)
        return false;

    if (wildcard->dst_port != 0 && key->dst_port != wildcard->dst_port)
        return false;

    // Protocol matching (0 = wildcard, matches any)
    if (wildcard->protocol != 0 && key->protocol != wildcard->protocol)
        return false;

    return true;
}

// Helper: Lookup policy with wildcard support
// Fast path: Try exact match first (most common)
// Slow path: Linear search wildcard policies (only for first packet)
static __always_inline __u8 lookup_policy_action(struct flow_key *key, __u32 *rule_id) {
    // FAST PATH: Try exact match first (O(1) hash lookup)
    struct policy_value *policy = bpf_map_lookup_elem(&policy_map, key);
    if (policy) {
        // Increment hit count (simple increment, not atomic for speed)
        policy->hit_count += 1;
        update_stats(STATS_POLICY_HITS);
        *rule_id = policy->rule_id;
        return policy->action;
    }

    // SLOW PATH: Linear search wildcard policies
    // Use #pragma unroll to help verifier, limit iterations
    struct wildcard_policy *wildcard = NULL;
    struct wildcard_policy *best_match = NULL;
    __u16 best_priority = 0;

    // Search for matching wildcard policies
    // Limit to reasonable number to pass verifier
    #pragma unroll
    for (__u32 i = 0; i < 100; i++) {
        __u32 idx = i;
        if (idx >= MAX_ENTRIES_WILDCARD_POLICY)
            break;

        wildcard = bpf_map_lookup_elem(&wildcard_policy_map, &idx);
        if (!wildcard)
            continue;

        // Skip empty slots (rule_id == 0)
        if (wildcard->rule_id == 0)
            continue;

        // Check if this policy matches
        if (matches_wildcard(key, wildcard)) {
            // Select highest priority match
            if (!best_match || wildcard->priority > best_priority) {
                best_match = wildcard;
                best_priority = wildcard->priority;
            }
        }
    }

    if (best_match) {
        update_stats(STATS_POLICY_HITS);
        *rule_id = best_match->rule_id;
        return best_match->action;
    }

    update_stats(STATS_POLICY_MISSES);
    *rule_id = 0;
    return POLICY_ACTION_ALLOW;  // Default allow if no policy matches
}

// Helper: Create new session (optimized - minimal initialization)
static __always_inline int create_session(struct flow_key *key, __u8 action, __u64 ts, __u32 packet_len) {
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
        
        // Only send events for DENY or if explicitly logging
        if (action == POLICY_ACTION_DENY || action == POLICY_ACTION_LOG) {
            struct flow_event *event = bpf_ringbuf_reserve(&flow_events, sizeof(*event), 0);
            if (event) {
                event->key = *key;
                event->timestamp = ts;
                event->packets = 1;
                event->bytes = packet_len;
                event->action = action;
                event->event_type = 0;  // new session
                bpf_ringbuf_submit(event, 0);
            }
        }
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
    
    // Update total packets counter
    update_stats(STATS_TOTAL_PACKETS);
    
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
        
        // Fast enforcement check
        if (action == POLICY_ACTION_DENY) {
            update_stats(STATS_DENIED_PACKETS);
#if DEBUG_MODE
            bpf_printk("DENY: %pI4:%d -> %pI4:%d (cached)\n",
                       &key.src_ip, bpf_ntohs(key.src_port),
                       &key.dst_ip, bpf_ntohs(key.dst_port));
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
    __u8 action = lookup_policy_action(&key, &matched_rule_id);

#if DEBUG_MODE
    if (matched_rule_id != 0) {
        bpf_printk("Policy %d matched: %pI4:%d -> %pI4:%d action=%d\n",
                   matched_rule_id,
                   &key.src_ip, bpf_ntohs(key.src_port),
                   &key.dst_ip, bpf_ntohs(key.dst_port),
                   action);
    }
#endif
    
    // Create new session with policy action (includes first packet stats)
    create_session(&key, action, now, skb->len);
    
    // Enforce policy
    if (action == POLICY_ACTION_DENY) {
        update_stats(STATS_DENIED_PACKETS);
#if DEBUG_MODE
        bpf_printk("DENY: %pI4:%d -> %pI4:%d (new)\n",
                   &key.src_ip, bpf_ntohs(key.src_port),
                   &key.dst_ip, bpf_ntohs(key.dst_port));
#endif
        return TC_ACT_SHOT;  // Drop packet
    }
    
    update_stats(STATS_ALLOWED_PACKETS);
    return TC_ACT_OK;  // Allow packet
}
