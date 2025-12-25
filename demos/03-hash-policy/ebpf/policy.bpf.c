// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Demo 3: Hash Policy Match
 *
 * This program demonstrates policy-based packet filtering using Hash Map.
 *
 * What it does:
 * - Looks up policy rules based on 5-tuple (exact match)
 * - Allows or denies packets according to policy
 * - Uses HASH map for O(1) policy lookup
 * - Tracks policy hit statistics
 *
 * Learning objectives:
 * 1. Understand HASH map for fast lookups
 * 2. Implement policy matching logic
 * 3. Use TC_ACT_SHOT to drop packets
 * 4. Learn userspace control (adding/removing policies via Go)
 * 5. Understand network security filtering basics
 */

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

#include "../common/headers/common_types.h"
#include "../common/headers/flow_processing.h"

char LICENSE[] SEC("license") = "GPL";

#define TC_ACT_OK    0
#define TC_ACT_SHOT  2

// Policy map: stores exact-match rules (5-tuple → action)
// Key: policy_key (src_ip, dst_ip, src_port, dst_port, protocol)
// Value: policy_value (action, priority, rule_id)
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_ENTRIES_POLICY);  // 10,000 policies
    __type(key, struct policy_key);
    __type(value, struct policy_value);
} policy_map SEC(".maps");

// Statistics counters
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, STATS_MAX);
    __type(key, __u32);
    __type(value, __u64);
} stats_map SEC(".maps");

/* Helper function to update statistics counter */
static __always_inline void update_stats(__u32 key)
{
    __u64 *count = bpf_map_lookup_elem(&stats_map, &key);
    if (count) {
        __sync_fetch_and_add(count, 1);
    }
}

/* Helper function to convert flow_key to policy_key
 *
 * We use the same 5-tuple for policy matching.
 * Direction is set to ANY for now (will be enhanced in later demos).
 */
static __always_inline void flow_to_policy_key(
    struct flow_key *flow,
    struct policy_key *policy)
{
    policy->src_ip = flow->src_ip;
    policy->dst_ip = flow->dst_ip;
    policy->src_port = flow->src_port;
    policy->dst_port = flow->dst_port;
    policy->protocol = flow->protocol;
    policy->direction = POLICY_DIR_ANY;
}

/* Main TC eBPF program */
SEC("tc")
int tc_policy_match(struct __sk_buff *skb)
{
    void *data = (void *)(unsigned long)skb->data;
    void *data_end = (void *)(unsigned long)skb->data_end;

    // Extract 5-tuple
    struct flow_key flow = {0};
    if (extract_flow_key_from_packet(data, data_end, &flow) < 0) {
        // Failed to parse - allow by default
        return TC_ACT_OK;
    }

    // Convert to policy key
    struct policy_key pkey = {0};
    flow_to_policy_key(&flow, &pkey);

    // Lookup policy in map
    struct policy_value *policy = bpf_map_lookup_elem(&policy_map, &pkey);

    // Update statistics
    update_stats(STATS_TOTAL_PACKETS);

    if (policy) {
        // Policy found - check action
        update_stats(STATS_POLICY_HITS);

        // Atomically increment hit count for this policy
        __sync_fetch_and_add(&policy->hit_count, 1);

        if (policy->action == POLICY_ACTION_DENY) {
            // DENY action - drop packet
            update_stats(STATS_DENIED_PACKETS);

            bpf_printk("[DENY] %pI4:%d → %pI4:%d proto=%d (rule_id=%u)\n",
                       &flow.src_ip, bpf_ntohs(flow.src_port),
                       &flow.dst_ip, bpf_ntohs(flow.dst_port),
                       flow.protocol, policy->rule_id);

            return TC_ACT_SHOT;  // Drop packet
        } else {
            // ALLOW action - permit packet
            update_stats(STATS_ALLOWED_PACKETS);

            bpf_printk("[ALLOW] %pI4:%d → %pI4:%d proto=%d (rule_id=%u)\n",
                       &flow.src_ip, bpf_ntohs(flow.src_port),
                       &flow.dst_ip, bpf_ntohs(flow.dst_port),
                       flow.protocol, policy->rule_id);

            return TC_ACT_OK;  // Allow packet
        }
    } else {
        // No policy found - allow by default (fail-open)
        update_stats(STATS_POLICY_MISSES);
        update_stats(STATS_ALLOWED_PACKETS);

        bpf_printk("[DEFAULT ALLOW] %pI4:%d → %pI4:%d proto=%d\n",
                   &flow.src_ip, bpf_ntohs(flow.src_port),
                   &flow.dst_ip, bpf_ntohs(flow.dst_port),
                   flow.protocol);

        return TC_ACT_OK;  // Default: allow
    }
}
