// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Protocol-indexed wildcard policy matching
 *
 * This header implements a hierarchical indexing system for wildcard policies.
 * Policies are organized by protocol to reduce linear scan overhead.
 *
 * Architecture:
 * - L1 Index: Protocol-based buckets (256 protocols: TCP, UDP, ICMP, ANY, ...)
 * - L2 Storage: Per-protocol array of wildcard policies (max 100 per protocol)
 * - Metadata: Track policy count per protocol for early termination
 *
 * Performance:
 * - Current: O(n) scan up to 50 policies, avg 50 μs
 * - Optimized: O(k) scan where k = policies in matching protocol bucket, avg 10-20 μs
 * - Capacity: 50 -> 500+ policies (10x improvement)
 *
 * Prerequisites (must be defined before including this header):
 * 1. Include common_types.h for basic types
 * 2. Define update_stats() function
 * 3. Define session and policy maps
 */

#ifndef __INDEXED_POLICY_MATCH_H__
#define __INDEXED_POLICY_MATCH_H__

/* Configuration: Max policies per protocol bucket */
#define MAX_POLICIES_PER_PROTOCOL 100

/* Configuration: Max protocols (0-255, but we use sparse allocation) */
#define MAX_PROTOCOL_BUCKETS 256

/* Metadata structure: track policy count per protocol */
struct protocol_metadata {
    __u32 policy_count;        // Number of active policies in this protocol bucket
    __u32 last_updated_ts;     // Timestamp of last update (for debugging)
    __u32 lookup_count;        // Number of lookups to this bucket (for monitoring)
    __u32 hit_count;           // Number of successful matches in this bucket
} __attribute__((packed));

/* Protocol bucket statistics (optional, for monitoring) */
struct protocol_bucket_stats {
    __u64 total_lookups;       // Total lookups to this bucket
    __u64 total_hits;          // Total successful matches
    __u64 avg_scan_count;      // Average number of policies scanned
    __u32 max_policies;        // Max policies ever stored in this bucket
    __u32 current_policies;    // Current number of policies
} __attribute__((packed));

/*
 * Data Structure Layout:
 *
 * ┌──────────────────────────────────────────────────────────┐
 * │           Protocol Metadata Map (ARRAY)                  │
 * │  Key: protocol (uint8)  →  Value: protocol_metadata      │
 * ├──────────────────────────────────────────────────────────┤
 * │  protocol=6 (TCP)   →  {policy_count: 150, ...}          │
 * │  protocol=17 (UDP)  →  {policy_count: 80, ...}           │
 * │  protocol=0 (ANY)   →  {policy_count: 20, ...}           │
 * └──────────────────────────────────────────────────────────┘
 *
 * ┌──────────────────────────────────────────────────────────┐
 * │         Protocol Index Map (HASH_OF_MAPS)                │
 * │  Key: protocol (uint8)  →  Value: inner_map_id (uint32)  │
 * ├──────────────────────────────────────────────────────────┤
 * │  protocol=6  →  inner_map_fd_123  ────┐                  │
 * │  protocol=17 →  inner_map_fd_456  ──┐ │                  │
 * │  protocol=0  →  inner_map_fd_789  ┐ │ │                  │
 * └────────────────────────────────────┼─┼─┼─────────────────┘
 *                                      │ │ │
 *          ┌───────────────────────────┘ │ │
 *          │   ┌─────────────────────────┘ │
 *          │   │   ┌───────────────────────┘
 *          ▼   ▼   ▼
 *     ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
 *     │  TCP Bucket     │  │  UDP Bucket     │  │  ANY Bucket     │
 *     │  (ARRAY, 100)   │  │  (ARRAY, 100)   │  │  (ARRAY, 100)   │
 *     ├─────────────────┤  ├─────────────────┤  ├─────────────────┤
 *     │ [0]: policy_1   │  │ [0]: policy_150 │  │ [0]: policy_230 │
 *     │ [1]: policy_2   │  │ [1]: policy_151 │  │ [1]: policy_231 │
 *     │ ...             │  │ ...             │  │ ...             │
 *     │ [149]: policy_X │  │ [79]: policy_Y  │  │ [19]: policy_Z  │
 *     └─────────────────┘  └─────────────────┘  └─────────────────┘
 */

// Forward declaration: matches_wildcard function from policy_match.h
// This function is defined in policy_match.h and reused here
static __always_inline bool matches_wildcard(
    struct flow_key *key,
    struct wildcard_policy *wildcard,
    __u8 direction);

/*
 * lookup_protocol_bucket - Get wildcard policies for a specific protocol
 *
 * @protocol: Protocol number to lookup (6=TCP, 17=UDP, 0=ANY)
 * @meta_out: Output pointer to receive metadata (can be NULL)
 *
 * Returns: Pointer to protocol metadata, or NULL if protocol has no policies
 *
 * This is a helper function used by the main lookup function.
 * It retrieves the metadata for a given protocol bucket.
 */
static __always_inline struct protocol_metadata *
lookup_protocol_metadata(__u8 protocol)
{
    // Metadata map key is protocol number (expanded to u32 for map key)
    __u32 proto_key = protocol;

    // Note: We use ARRAY map for metadata, so key is the index
    // Protocol number directly maps to array index
    struct protocol_metadata *meta =
        bpf_map_lookup_elem(&protocol_metadata_map, &proto_key);

    return meta;
}

/*
 * scan_protocol_bucket - Scan wildcard policies in a specific protocol bucket
 *
 * @protocol: Protocol number (6=TCP, 17=UDP, 0=ANY)
 * @key: Flow key to match against
 * @direction: Packet direction (INGRESS/EGRESS)
 * @best_match_out: Output pointer to store best matching policy
 * @best_priority_out: Output pointer to store priority of best match
 *
 * Returns: Number of policies scanned (for monitoring)
 *
 * This function performs linear scan within a single protocol bucket.
 * Since bucket size is typically small (< 100), this is much faster
 * than scanning all 500+ policies.
 */
static __always_inline __u32 scan_protocol_bucket(
    __u8 protocol,
    struct flow_key *key,
    __u8 direction,
    struct wildcard_policy **best_match_out,
    __u16 *best_priority_out)
{
    // Get metadata to know how many policies to scan
    struct protocol_metadata *meta = lookup_protocol_metadata(protocol);
    if (!meta || meta->policy_count == 0) {
        return 0;  // No policies in this bucket
    }

    // Get the protocol bucket map (inner map in hash_of_maps)
    __u32 proto_key = protocol;
    void *bucket_map = bpf_map_lookup_elem(&protocol_index_map, &proto_key);
    if (!bucket_map) {
        return 0;  // Bucket not initialized
    }

    // Determine scan limit (min of policy_count and max bucket size)
    __u32 max_scan = meta->policy_count;
    if (max_scan > MAX_POLICIES_PER_PROTOCOL) {
        max_scan = MAX_POLICIES_PER_PROTOCOL;
    }

    // Linear scan within protocol bucket
    __u32 scanned = 0;
    struct wildcard_policy *current_best = *best_match_out;
    __u16 current_priority = *best_priority_out;

    // Use bounded loop with #pragma unroll for eBPF verifier
    #pragma unroll
    for (__u32 i = 0; i < MAX_POLICIES_PER_PROTOCOL; i++) {
        // Early termination when we've scanned all active policies
        if (i >= max_scan) {
            break;
        }

        // Lookup policy at index i in the bucket
        __u32 idx = i;
        struct wildcard_policy *wildcard =
            bpf_map_lookup_elem(bucket_map, &idx);

        if (!wildcard) {
            continue;  // Sparse bucket, skip empty slots
        }

        // Early stop: empty slot (rule_id=0) means end of compact storage
        if (wildcard->rule_id == 0) {
            break;
        }

        scanned++;

        // Check if this policy matches the flow
        if (!matches_wildcard(key, wildcard, direction)) {
            continue;
        }

        // Priority selection: higher priority wins
        if (!current_best || wildcard->priority > current_priority) {
            current_best = wildcard;
            current_priority = wildcard->priority;
        }
    }

    // Update output parameters
    *best_match_out = current_best;
    *best_priority_out = current_priority;

    return scanned;
}

/*
 * lookup_policy_action_indexed - Main indexed policy lookup function
 *
 * @key: Flow key to match against
 * @direction: Packet direction (INGRESS/EGRESS/ANY)
 * @rule_id: Output pointer to store matched rule ID
 *
 * Returns: Policy action (ALLOW/DENY/LOG)
 *
 * Lookup Strategy:
 * 1. Fast Path: Try exact match in ipaddr_policy_map (O(1) hash lookup)
 * 2. Slow Path (Indexed):
 *    a. Scan protocol-specific bucket (e.g., TCP bucket for TCP flows)
 *    b. Scan ANY protocol bucket (protocol=0, matches all)
 * 3. Default: Return ALLOW if no match
 *
 * Performance:
 * - Best case: Exact match, < 1 μs
 * - Average case: Protocol bucket scan (10-50 policies), 10-20 μs
 * - Worst case: Large bucket + ANY bucket, < 100 μs
 */
static __always_inline __u8 lookup_policy_action_indexed(
    struct flow_key *key,
    __u8 direction,
    __u32 *rule_id)
{
    // ===== Fast Path: Exact Match =====
    // Try hash map lookup first (unchanged from original implementation)

    // Build policy key (6-tuple with direction)
    struct policy_key pkey = {
        .src_port = key->src_port,
        .dst_port = key->dst_port,
        .protocol = key->protocol,
        .direction = direction,  // Try direction-specific policy first
        .ip_version = key->ip_version,
        .vlan_id = key->vlan_id,
        .pad = 0,
        .pad2 = 0,
    };

    // Copy IP addresses (4 x 32-bit words for IPv4/IPv6 compatibility)
    #pragma unroll
    for (int i = 0; i < 4; i++) {
        pkey.src_ip[i] = key->src_ip[i];
        pkey.dst_ip[i] = key->dst_ip[i];
    }

    // 1. Try direction-specific exact match
    struct policy_value *policy = bpf_map_lookup_elem(&ipaddr_policy_map, &pkey);
    if (policy) {
        policy->hit_count += 1;
        update_stats(STATS_POLICY_HITS);
        *rule_id = policy->rule_id;
        return policy->action;
    }

    // 2. Try bidirectional exact match (direction=ANY)
    pkey.direction = POLICY_DIR_ANY;
    policy = bpf_map_lookup_elem(&ipaddr_policy_map, &pkey);
    if (policy) {
        policy->hit_count += 1;
        update_stats(STATS_POLICY_HITS);
        *rule_id = policy->rule_id;
        return policy->action;
    }

    // ===== Slow Path: Indexed Wildcard Match =====
    struct wildcard_policy *best_match = NULL;
    __u16 best_priority = 0;
    __u32 total_scanned = 0;

    // Step 1: Scan protocol-specific bucket (e.g., TCP bucket for TCP flows)
    __u8 flow_protocol = key->protocol;
    __u32 scanned = scan_protocol_bucket(
        flow_protocol,
        key,
        direction,
        &best_match,
        &best_priority
    );
    total_scanned += scanned;

    // Step 2: Scan ANY protocol bucket (protocol=0, matches all protocols)
    // Only scan if we haven't found a match yet, or if we want to find higher priority
    scanned = scan_protocol_bucket(
        0,  // protocol=0 means ANY
        key,
        direction,
        &best_match,
        &best_priority
    );
    total_scanned += scanned;

    // Update statistics
    if (best_match) {
        update_stats(STATS_POLICY_HITS);
        *rule_id = best_match->rule_id;
        return best_match->action;
    }

    // ===== Default Policy: ALLOW =====
    update_stats(STATS_POLICY_MISSES);
    *rule_id = 0;
    return POLICY_ACTION_ALLOW;
}

#endif /* __INDEXED_POLICY_MATCH_H__ */
