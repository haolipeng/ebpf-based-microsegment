// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Identity-based policy structures and matching logic */

#ifndef __IDENTITY_POLICY_H__
#define __IDENTITY_POLICY_H__

#include "ipcache.h"

// Maximum entries in the identity policy map
#define MAX_ENTRIES_IDENTITY_POLICY 50000

// Reserved identity constants
// Must match the Go constants in src/agent/pkg/identity/types.go
#define IDENTITY_UNKNOWN        0
#define IDENTITY_HOST           1
#define IDENTITY_WORLD          2
#define IDENTITY_UNMANAGED      3
#define IDENTITY_HEALTH         4
#define IDENTITY_INIT           5
#define IDENTITY_REMOTE_NODE    6
#define IDENTITY_KUBE_APISERVER 7
#define IDENTITY_RESERVED_MAX   255

// Identity scope constants (high 8 bits)
#define IDENTITY_SCOPE_MASK       0xFF000000
#define IDENTITY_SCOPE_GLOBAL     0x00000000
#define IDENTITY_SCOPE_LOCAL      0x01000000
#define IDENTITY_SCOPE_REMOTE_NODE 0x02000000

// Identity policy key structure
// Matches on source identity, destination identity, port, and protocol
struct identity_policy_key {
    __u32 src_identity;    // Source security identity
    __u32 dst_identity;    // Destination security identity
    __u16 dst_port;        // Destination port (0 = any port)
    __u8  protocol;        // Protocol (0 = any protocol)
    __u8  pad;             // Padding for alignment
} __attribute__((packed));

// Identity policy value structure
struct identity_policy_value {
    __u8  action;          // Policy action (POLICY_ACTION_ALLOW, POLICY_ACTION_DENY, POLICY_ACTION_LOG)
    __u8  log_enabled;     // Enable logging for this policy
    __u16 priority;        // Policy priority (higher = more important)
    __u32 rule_id;         // Unique rule identifier for tracking
    __u64 hit_count;       // Number of times this policy was matched
} __attribute__((packed));

// Identity policy map definition
// Uses hash map for O(1) lookup
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, MAX_ENTRIES_IDENTITY_POLICY);
    __type(key, struct identity_policy_key);
    __type(value, struct identity_policy_value);
    __uint(pinning, LIBBPF_PIN_BY_NAME);
} identity_policy_map SEC(".maps");

// Helper function to check if an identity is reserved
static __always_inline bool identity_is_reserved(__u32 identity) {
    return identity <= IDENTITY_RESERVED_MAX;
}

// Helper function to check if an identity represents the host
static __always_inline bool identity_is_host(__u32 identity) {
    return identity == IDENTITY_HOST;
}

// Helper function to check if an identity represents the world (external)
static __always_inline bool identity_is_world(__u32 identity) {
    return identity == IDENTITY_WORLD;
}

// Helper function to check if traffic is intra-cluster
// (both source and destination have non-world identities)
static __always_inline bool is_intra_cluster(__u32 src_id, __u32 dst_id) {
    return src_id != IDENTITY_WORLD && dst_id != IDENTITY_WORLD &&
           src_id != IDENTITY_UNKNOWN && dst_id != IDENTITY_UNKNOWN;
}

// Match identity-based policy
// Returns: 0 = no match (fall through to IP policy), 1 = allow, 2 = deny
// This function tries multiple key combinations for flexibility:
// 1. Exact match (src_id, dst_id, port, proto)
// 2. Any port (src_id, dst_id, 0, proto)
// 3. Any protocol (src_id, dst_id, port, 0)
// 4. Any port and protocol (src_id, dst_id, 0, 0)
static __always_inline int match_identity_policy(
    __u32 src_identity,
    __u32 dst_identity,
    __u16 dst_port,
    __u8  protocol
) {
    struct identity_policy_key key = {
        .src_identity = src_identity,
        .dst_identity = dst_identity,
        .dst_port = dst_port,
        .protocol = protocol,
        .pad = 0,
    };

    struct identity_policy_value *val;

    // Try 1: Exact match
    val = bpf_map_lookup_elem(&identity_policy_map, &key);
    if (val) {
        // Update hit count (best effort, no error checking for performance)
        __sync_fetch_and_add(&val->hit_count, 1);
        return val->action == POLICY_ACTION_ALLOW ? 1 : 2;
    }

    // Try 2: Any port
    key.dst_port = 0;
    val = bpf_map_lookup_elem(&identity_policy_map, &key);
    if (val) {
        __sync_fetch_and_add(&val->hit_count, 1);
        return val->action == POLICY_ACTION_ALLOW ? 1 : 2;
    }

    // Try 3: Any protocol (restore port)
    key.dst_port = dst_port;
    key.protocol = 0;
    val = bpf_map_lookup_elem(&identity_policy_map, &key);
    if (val) {
        __sync_fetch_and_add(&val->hit_count, 1);
        return val->action == POLICY_ACTION_ALLOW ? 1 : 2;
    }

    // Try 4: Any port and protocol
    key.dst_port = 0;
    val = bpf_map_lookup_elem(&identity_policy_map, &key);
    if (val) {
        __sync_fetch_and_add(&val->hit_count, 1);
        return val->action == POLICY_ACTION_ALLOW ? 1 : 2;
    }

    // No match - fall through to IP-based policy
    return 0;
}

// Match policy using identity-first approach
// This function:
// 1. Looks up source and destination identities from IPCache
// 2. If both have identities, tries identity-based policy matching
// 3. Falls back to IP-based policy if no identity match
// Returns: 0 = fall through to IP policy, 1 = allow, 2 = deny
static __always_inline int match_policy_with_identity(
    struct flow_key *key
) {
    __u32 src_identity, dst_identity;

    // Lookup source identity from IPCache
    src_identity = ipcache_lookup(key->src_ip, key->ip_version);
    if (src_identity == IDENTITY_UNKNOWN) {
        // Unknown source - fall back to IP policy
        return 0;
    }

    // Lookup destination identity from IPCache
    dst_identity = ipcache_lookup(key->dst_ip, key->ip_version);
    if (dst_identity == IDENTITY_UNKNOWN) {
        // Unknown destination - fall back to IP policy
        return 0;
    }

    // Both have identities - try identity-based policy
    return match_identity_policy(
        src_identity,
        dst_identity,
        key->dst_port,
        key->protocol
    );
}

// Statistics counters for identity policy (extend stats_key enum)
// These can be added to the existing stats_map
#define STATS_IDENTITY_LOOKUPS      100  // IPCache lookups
#define STATS_IDENTITY_HITS         101  // Successful IPCache lookups
#define STATS_IDENTITY_MISSES       102  // Failed IPCache lookups
#define STATS_IDENTITY_POLICY_HITS  103  // Identity policy matches
#define STATS_IDENTITY_FALLBACKS    104  // Fell back to IP policy

#endif /* __IDENTITY_POLICY_H__ */
