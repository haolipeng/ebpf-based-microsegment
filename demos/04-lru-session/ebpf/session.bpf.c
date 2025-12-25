// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Demo 4: LRU Session Tracking - Hot path optimization with session cache */

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

#include "../common/headers/common_types.h"
#include "../common/headers/flow_processing.h"

char LICENSE[] SEC("license") = "GPL";

#define TC_ACT_OK    0
#define TC_ACT_SHOT  2

// LRU session map - automatically evicts least recently used entries
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, MAX_ENTRIES_SESSION);
    __type(key, struct flow_key);
    __type(value, struct session_value);
} session_map SEC(".maps");

// Policy map (same as Demo 3)
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_ENTRIES_POLICY);
    __type(key, struct policy_key);
    __type(value, struct policy_value);
} policy_map SEC(".maps");

// Statistics
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, STATS_MAX);
    __type(key, __u32);
    __type(value, __u64);
} stats_map SEC(".maps");

static __always_inline void update_stats(__u32 key)
{
    __u64 *count = bpf_map_lookup_elem(&stats_map, &key);
    if (count) __sync_fetch_and_add(count, 1);
}

SEC("tc")
int tc_session_tracking(struct __sk_buff *skb)
{
    void *data = (void *)(unsigned long)skb->data;
    void *data_end = (void *)(unsigned long)skb->data_end;

    struct flow_key flow = {0};
    if (extract_flow_key_from_packet(data, data_end, &flow) < 0)
        return TC_ACT_OK;

    __u64 now = bpf_ktime_get_ns();

    // HOT PATH: Check session cache
    struct session_value *session = bpf_map_lookup_elem(&session_map, &flow);
    if (session) {
        // Session hit - use cached decision (< 1μs)
        session->packets_to_server += 1;
        session->bytes_to_server += skb->len;
        session->last_seen_ts = now;
        
        update_stats(STATS_TOTAL_PACKETS);
        
        if (session->policy_action == POLICY_ACTION_ALLOW) {
            update_stats(STATS_ALLOWED_PACKETS);
            return TC_ACT_OK;
        } else {
            update_stats(STATS_DENIED_PACKETS);
            return TC_ACT_SHOT;
        }
    }

    // COLD PATH: New session - lookup policy (5-20μs)
    struct policy_key pkey = {
        .src_ip = flow.src_ip,
        .dst_ip = flow.dst_ip,
        .src_port = flow.src_port,
        .dst_port = flow.dst_port,
        .protocol = flow.protocol,
        .direction = POLICY_DIR_ANY,
    };

    struct policy_value *policy = bpf_map_lookup_elem(&policy_map, &pkey);
    __u8 action = policy ? policy->action : POLICY_ACTION_ALLOW;
    __u32 rule_id = policy ? policy->rule_id : 0;

    // Create new session entry
    struct session_value new_session = {
        .created_ts = now,
        .last_seen_ts = now,
        .packets_to_server = 1,
        .bytes_to_server = skb->len,
        .policy_action = action,
        .rule_id = rule_id,
    };

    bpf_map_update_elem(&session_map, &flow, &new_session, BPF_ANY);
    
    update_stats(STATS_TOTAL_PACKETS);
    update_stats(STATS_NEW_SESSIONS);
    
    bpf_printk("[NEW SESSION] %pI4:%d → %pI4:%d action=%s\n",
               &flow.src_ip, bpf_ntohs(flow.src_port),
               &flow.dst_ip, bpf_ntohs(flow.dst_port),
               action == POLICY_ACTION_ALLOW ? "ALLOW" : "DENY");

    if (action == POLICY_ACTION_ALLOW) {
        update_stats(STATS_ALLOWED_PACKETS);
        return TC_ACT_OK;
    } else {
        update_stats(STATS_DENIED_PACKETS);
        return TC_ACT_SHOT;
    }
}
