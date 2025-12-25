// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Protocol-Indexed Policy Matching with Process Support (V3)
 *
 * This version extends V2 to support process-level policies (Issue #47).
 *
 * Architecture:
 * - Same segmented approach as V2
 * - Each protocol segment is divided into two sub-segments:
 *   1. Process-specific policies (stored at the end of segment)
 *   2. Network-only policies (stored at the beginning)
 *
 * Segment Layout:
 * ┌──────────────────────────────────────────────────────────┐
 * │  [start_idx ... start_idx + net_count)                   │
 * │  Network-only policies (process_name = "")               │
 * ├──────────────────────────────────────────────────────────┤
 * │  [start_idx + net_count ... start_idx + total_count)     │
 * │  Process-specific policies (process_name != "")          │
 * └──────────────────────────────────────────────────────────┘
 *
 * Where:
 *   net_count = policy_count - process_count
 *   total_count = policy_count
 *
 * Lookup Strategy:
 * 1. Fast path: Exact match (unchanged)
 * 2. Indexed slow path:
 *    a. Get protocol segment
 *    b. Scan process-specific sub-segment first (if proc info available)
 *    c. Scan network-only sub-segment
 *    d. Also scan ANY protocol segment
 *
 * Performance:
 * - Process match: ~2-5μs (scan 10-50 process policies)
 * - Network match: ~10-20μs (scan 50-200 network policies)
 * - Much better than O(1000) full linear scan
 */

#ifndef __INDEXED_POLICY_MATCH_V3_H__
#define __INDEXED_POLICY_MATCH_V3_H__

/* Maximum policies per protocol segment
 * Reduced values to fit within eBPF instruction limit
 * Total unrolled instructions ≈ (MAX_POLICIES + MAX_PROCESS) * ~100 per iteration
 */
#define MAX_POLICIES_PER_PROTOCOL_V3 25
#define MAX_PROCESS_POLICIES_PER_PROTOCOL 10

// Note: struct protocol_segment is defined in common_types.h
// Note: struct process_match_info is defined in common_types.h

// Forward declarations
static __always_inline bool matches_wildcard(
    struct flow_key *key,
    struct wildcard_policy *wildcard,
    __u8 direction);

static __always_inline bool matches_wildcard_with_process(
    struct flow_key *key,
    struct process_match_info *proc_info,
    struct wildcard_policy *wildcard,
    __u8 direction);

/*
 * scan_process_policies - Scan process-specific policies
 *
 * @start_idx: Starting index of process policies
 * @process_count: Number of process policies
 * @key: Flow key
 * @proc_info: Process information
 * @direction: Packet direction
 * @best_match: Output pointer for best matching policy
 * @best_priority: Output pointer for best priority
 *
 * Returns: Number of policies scanned
 */
static __always_inline __u32 scan_process_policies(
    __u32 start_idx,
    __u32 process_count,
    struct flow_key *key,
    struct process_match_info *proc_info,
    __u8 direction,
    struct wildcard_policy **best_match,
    __u16 *best_priority)
{
    if (process_count == 0 || !proc_info) {
        return 0;
    }

    __u32 scanned = 0;
    __u32 limit = process_count;
    if (limit > MAX_PROCESS_POLICIES_PER_PROTOCOL) {
        limit = MAX_PROCESS_POLICIES_PER_PROTOCOL;
    }

    struct wildcard_policy *current_best = *best_match;
    __u16 current_priority = *best_priority;

    // Bounded loop for eBPF verifier
    #pragma unroll
    for (__u32 i = 0; i < MAX_PROCESS_POLICIES_PER_PROTOCOL; i++) {
        if (i >= limit) {
            break;
        }

        __u32 idx = start_idx + i;
        if (idx >= MAX_ENTRIES_WILDCARD_POLICY) {
            break;
        }

        struct wildcard_policy *wildcard =
            bpf_map_lookup_elem(&ipcidr_policy_map, &idx);

        if (!wildcard || wildcard->rule_id == 0) {
            break;
        }

        scanned++;

        // Must have process_name for process policies
        if (wildcard->process_name[0] == '\0') {
            continue;
        }

        // Match check with process info
        if (!matches_wildcard_with_process(key, proc_info, wildcard, direction)) {
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
 * scan_network_policies - Scan network-only policies
 *
 * @start_idx: Starting index of network policies
 * @network_count: Number of network policies
 * @key: Flow key
 * @direction: Packet direction
 * @best_match: Output pointer for best matching policy
 * @best_priority: Output pointer for best priority
 *
 * Returns: Number of policies scanned
 */
static __always_inline __u32 scan_network_policies(
    __u32 start_idx,
    __u32 network_count,
    struct flow_key *key,
    __u8 direction,
    struct wildcard_policy **best_match,
    __u16 *best_priority)
{
    if (network_count == 0) {
        return 0;
    }

    __u32 scanned = 0;
    __u32 limit = network_count;
    if (limit > MAX_POLICIES_PER_PROTOCOL_V3) {
        limit = MAX_POLICIES_PER_PROTOCOL_V3;
    }

    struct wildcard_policy *current_best = *best_match;
    __u16 current_priority = *best_priority;

    // Bounded loop for eBPF verifier
    #pragma unroll
    for (__u32 i = 0; i < MAX_POLICIES_PER_PROTOCOL_V3; i++) {
        if (i >= limit) {
            break;
        }

        __u32 idx = start_idx + i;
        if (idx >= MAX_ENTRIES_WILDCARD_POLICY) {
            break;
        }

        struct wildcard_policy *wildcard =
            bpf_map_lookup_elem(&ipcidr_policy_map, &idx);

        if (!wildcard || wildcard->rule_id == 0) {
            break;
        }

        scanned++;

        // Network-only policies should not have process_name
        if (wildcard->process_name[0] != '\0') {
            continue;
        }

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
 * scan_protocol_segment_v3 - Scan protocol segment with process support
 *
 * @segment: Protocol segment descriptor
 * @key: Flow key
 * @proc_info: Process information (can be NULL)
 * @direction: Packet direction
 * @best_match: Output pointer for best matching policy
 * @best_priority: Output pointer for best priority
 *
 * Returns: Number of policies scanned
 */
static __always_inline __u32 scan_protocol_segment_v3(
    struct protocol_segment *segment,
    struct flow_key *key,
    struct process_match_info *proc_info,
    __u8 direction,
    struct wildcard_policy **best_match,
    __u16 *best_priority)
{
    if (!segment || segment->policy_count == 0) {
        return 0;
    }

    __u32 total_scanned = 0;
    __u32 network_count = segment->policy_count - segment->process_count;
    __u32 process_start_idx = segment->start_idx + network_count;

    // Phase 1: Scan process-specific policies (higher priority)
    if (proc_info && segment->process_count > 0) {
        total_scanned += scan_process_policies(
            process_start_idx,
            segment->process_count,
            key,
            proc_info,
            direction,
            best_match,
            best_priority
        );
    }

    // Phase 2: Scan network-only policies (if no process match)
    if (network_count > 0) {
        total_scanned += scan_network_policies(
            segment->start_idx,
            network_count,
            key,
            direction,
            best_match,
            best_priority
        );
    }

    return total_scanned;
}

/*
 * lookup_policy_action_indexed_v3 - Protocol-indexed policy lookup with process support
 *
 * @key: Flow key
 * @proc_info: Process information (can be NULL if not available)
 * @direction: Packet direction
 * @rule_id: Output matched rule ID
 *
 * Returns: Policy action (ALLOW/DENY/LOG)
 */
static __always_inline __u8 lookup_policy_action_indexed_v3(
    struct flow_key *key,
    struct process_match_info *proc_info,
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
    struct policy_value *policy = bpf_map_lookup_elem(&ipaddr_policy_map, &pkey);
    if (policy) {
        policy->hit_count += 1;
        update_stats(STATS_POLICY_HITS);
        *rule_id = policy->rule_id;
        return policy->action;
    }

    // Try bidirectional exact match
    pkey.direction = POLICY_DIR_ANY;
    policy = bpf_map_lookup_elem(&ipaddr_policy_map, &pkey);
    if (policy) {
        policy->hit_count += 1;
        update_stats(STATS_POLICY_HITS);
        *rule_id = policy->rule_id;
        return policy->action;
    }

    // ===== Slow Path: Protocol-Indexed Wildcard Match with Process Support =====
    struct wildcard_policy *best_match = NULL;
    __u16 best_priority = 0;

    // Get protocol-specific segment
    __u32 proto_key = key->protocol;
    struct protocol_segment *segment =
        bpf_map_lookup_elem(&protocol_offset_map, &proto_key);

    if (segment && segment->policy_count > 0) {
        scan_protocol_segment_v3(segment, key, proc_info, direction, &best_match, &best_priority);
    }

    // Also scan ANY protocol segment (protocol=0)
    __u32 any_proto = 0;
    segment = bpf_map_lookup_elem(&protocol_offset_map, &any_proto);

    if (segment && segment->policy_count > 0) {
        scan_protocol_segment_v3(segment, key, proc_info, direction, &best_match, &best_priority);
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

#endif /* __INDEXED_POLICY_MATCH_V3_H__ */
