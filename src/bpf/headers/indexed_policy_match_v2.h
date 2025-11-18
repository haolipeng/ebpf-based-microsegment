// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Segmented Protocol Index for Wildcard Policy Matching
 *
 * This is a simplified version of protocol-indexed matching that uses
 * a segmented approach instead of hash_of_maps for easier implementation.
 *
 * Architecture:
 * - Single large ARRAY map (wildcard_policy_map) with 1000 slots
 * - Protocol offset map tracks which segment belongs to which protocol
 * - Policies are stored in contiguous segments per protocol
 *
 * Example Layout:
 * ┌────────────────────────────────────────────┐
 * │  Slot Range  │  Protocol  │  Policy Count │
 * ├────────────────────────────────────────────┤
 * │  0-149       │  TCP (6)   │  150          │
 * │  150-229     │  UDP (17)  │  80           │
 * │  230-249     │  ANY (0)   │  20           │
 * │  250-299     │  ICMP (1)  │  50           │
 * │  300-999     │  (unused)  │  0            │
 * └────────────────────────────────────────────┘
 *
 * Performance:
 * - Current: Scan 50 policies (5% of 1000 capacity)
 * - Optimized: Scan only protocol-specific segment (e.g., 150 TCP policies)
 * - Effective capacity: 500+ policies across all protocols
 *
 * Prerequisites:
 * 1. Include common_types.h
 * 2. Define wildcard_policy_map (existing)
 * 3. Define protocol_offset_map (new)
 * 4. Define update_stats() function
 */

#ifndef __INDEXED_POLICY_MATCH_V2_H__
#define __INDEXED_POLICY_MATCH_V2_H__

/* Maximum policies per protocol segment */
#define MAX_POLICIES_PER_PROTOCOL 200

// Note: struct protocol_segment is defined in common_types.h

// Forward declaration: matches_wildcard from policy_match.h
static __always_inline bool matches_wildcard(
    struct flow_key *key,
    struct wildcard_policy *wildcard,
    __u8 direction);

/*
 * scan_protocol_segment - Scan wildcard policies in a protocol segment
 *
 * @segment: Protocol segment descriptor
 * @key: Flow key to match
 * @direction: Packet direction
 * @best_match: Output pointer for best matching policy
 * @best_priority: Output pointer for best priority
 *
 * Returns: Number of policies scanned
 *
 * This function scans only the policies belonging to a specific protocol,
 * significantly reducing the search space compared to full linear scan.
 */
static __always_inline __u32 scan_protocol_segment(
    struct protocol_segment *segment,
    struct flow_key *key,
    __u8 direction,
    struct wildcard_policy **best_match,
    __u16 *best_priority)
{
    if (!segment || segment->policy_count == 0) {
        return 0;
    }

    __u32 start = segment->start_idx;
    __u32 count = segment->policy_count;
    __u32 scanned = 0;

    // Cap scan count to prevent verifier issues
    if (count > MAX_POLICIES_PER_PROTOCOL) {
        count = MAX_POLICIES_PER_PROTOCOL;
    }

    struct wildcard_policy *current_best = *best_match;
    __u16 current_priority = *best_priority;

    // Bounded loop for eBPF verifier
    #pragma unroll
    for (__u32 i = 0; i < MAX_POLICIES_PER_PROTOCOL; i++) {
        if (i >= count) {
            break;
        }

        __u32 idx = start + i;

        // Bounds check for safety
        if (idx >= MAX_ENTRIES_WILDCARD_POLICY) {
            break;
        }

        struct wildcard_policy *wildcard =
            bpf_map_lookup_elem(&wildcard_policy_map, &idx);

        if (!wildcard) {
            continue;
        }

        // Early stop: empty slot means end of segment
        if (wildcard->rule_id == 0) {
            break;
        }

        scanned++;

        // Match check
        if (!matches_wildcard(key, wildcard, direction)) {
            continue;
        }

        // Priority selection
        if (!current_best || wildcard->priority > current_priority) {
            current_best = wildcard;
            current_priority = wildcard->priority;
        }
    }

    *best_match = current_best;
    *best_priority = current_priority;

    return scanned;
}

/*
 * lookup_policy_action_indexed - Protocol-indexed policy lookup
 *
 * @key: Flow key
 * @direction: Packet direction
 * @rule_id: Output matched rule ID
 *
 * Returns: Policy action (ALLOW/DENY/LOG)
 *
 * Lookup strategy:
 * 1. Fast path: Exact match (unchanged)
 * 2. Indexed slow path:
 *    a. Get protocol segment for flow's protocol
 *    b. Scan only that segment (much smaller than full scan)
 *    c. Also scan ANY protocol segment
 * 3. Default: ALLOW
 */
static __always_inline __u8 lookup_policy_action_indexed(
    struct flow_key *key,
    __u8 direction,
    __u32 *rule_id)
{
    // ===== Fast Path: Exact Match =====
    struct policy_key pkey = {
        .src_port = key->src_port,
        .dst_port = key->dst_port,
        .protocol = key->protocol,
        .direction = direction,
        .ip_version = key->ip_version,
        .vlan_id = key->vlan_id,
        .pad = 0,
        .pad2 = 0,
    };

    #pragma unroll
    for (int i = 0; i < 4; i++) {
        pkey.src_ip[i] = key->src_ip[i];
        pkey.dst_ip[i] = key->dst_ip[i];
    }

    // Try direction-specific exact match
    struct policy_value *policy = bpf_map_lookup_elem(&policy_map, &pkey);
    if (policy) {
        policy->hit_count += 1;
        update_stats(STATS_POLICY_HITS);
        *rule_id = policy->rule_id;
        return policy->action;
    }

    // Try bidirectional exact match
    pkey.direction = POLICY_DIR_ANY;
    policy = bpf_map_lookup_elem(&policy_map, &pkey);
    if (policy) {
        policy->hit_count += 1;
        update_stats(STATS_POLICY_HITS);
        *rule_id = policy->rule_id;
        return policy->action;
    }

    // ===== Slow Path: Protocol-Indexed Wildcard Match =====
    struct wildcard_policy *best_match = NULL;
    __u16 best_priority = 0;

    // Get protocol-specific segment
    __u32 proto_key = key->protocol;
    struct protocol_segment *segment =
        bpf_map_lookup_elem(&protocol_offset_map, &proto_key);

    if (segment && segment->policy_count > 0) {
        scan_protocol_segment(segment, key, direction, &best_match, &best_priority);
    }

    // Also scan ANY protocol segment (protocol=0)
    __u32 any_proto = 0;
    segment = bpf_map_lookup_elem(&protocol_offset_map, &any_proto);

    if (segment && segment->policy_count > 0) {
        scan_protocol_segment(segment, key, direction, &best_match, &best_priority);
    }

    if (best_match) {
        update_stats(STATS_POLICY_HITS);
        *rule_id = best_match->rule_id;
        return best_match->action;
    }

    // Default: ALLOW
    update_stats(STATS_POLICY_MISSES);
    *rule_id = 0;
    return POLICY_ACTION_ALLOW;
}

#endif /* __INDEXED_POLICY_MATCH_V2_H__ */
