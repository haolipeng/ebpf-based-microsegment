// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* NAT (Network Address Translation) support for microsegmentation
 *
 * This header provides NAT detection and original address restoration
 * to enable correct policy matching in Docker, Kubernetes, and other
 * NAT environments.
 *
 * Supported scenarios:
 * - Docker bridge network (SNAT)
 * - Kubernetes ClusterIP Service (DNAT)
 * - Kubernetes NodePort Service (DNAT + SNAT)
 * - iptables MASQUERADE
 */

#ifndef __NAT_SUPPORT_H__
#define __NAT_SUPPORT_H__

#include "common_types.h"

// NAT type constants
#define NAT_TYPE_NONE   0  // No NAT detected
#define NAT_TYPE_SNAT   1  // Source NAT (e.g., Docker bridge)
#define NAT_TYPE_DNAT   2  // Destination NAT (e.g., K8s Service)
#define NAT_TYPE_BOTH   3  // Both SNAT and DNAT (e.g., K8s NodePort)

// NAT match mode - controls how policies are matched
#define NAT_MATCH_MODE_ORIGINAL    0  // Match using original addresses (before NAT)
#define NAT_MATCH_MODE_TRANSLATED  1  // Match using translated addresses (after NAT)
#define NAT_MATCH_MODE_BOTH        2  // Try both original and translated addresses

// NAT conntrack status flags (from Linux nf_conntrack)
#define CONNTRACK_STATUS_SEEN_REPLY  0x00000002
#define CONNTRACK_STATUS_ASSURED     0x00000004
#define CONNTRACK_STATUS_CONFIRMED   0x00000008
#define CONNTRACK_STATUS_SRC_NAT     0x00000010  // SNAT applied
#define CONNTRACK_STATUS_DST_NAT     0x00000020  // DNAT applied
#define CONNTRACK_STATUS_NAT_MASK    0x00000030  // SNAT | DNAT

// Map size configurations
#define MAX_CONNTRACK_ENTRIES  200000  // 200K concurrent NAT connections
#define MAX_NAT_STATS_ENTRIES  16      // NAT statistics counters

// Conntrack cache key (post-NAT addresses)
// This represents the packet as seen on the wire after NAT transformation
struct conntrack_key {
    __u32 src_ip[4];      // Source IP (post-NAT)
    __u32 dst_ip[4];      // Destination IP (post-NAT)
    __u16 src_port;       // Source port (post-NAT)
    __u16 dst_port;       // Destination port (post-NAT)
    __u8  protocol;       // L4 protocol (TCP/UDP/ICMP)
    __u8  ip_version;     // 4 = IPv4, 6 = IPv6
    __u16 pad;            // Padding for alignment
} __attribute__((packed));

// Conntrack cache entry (original and reply tuples)
// Stores both original and reply direction information for bidirectional matching
struct conntrack_entry {
    struct flow_key original_tuple;  // Original 5-tuple (pre-NAT)
    struct flow_key reply_tuple;     // Reply direction 5-tuple

    __u64 timestamp;      // Last update timestamp (nanoseconds)
    __u32 status;         // Conntrack status flags
    __u8  nat_type;       // NAT_TYPE_* constant
    __u8  pad[3];         // Padding for alignment
} __attribute__((packed));

// NAT configuration
// Controls NAT detection behavior and policy matching mode
struct nat_config {
    __u8  match_mode;        // NAT_MATCH_MODE_* constant
    __u8  enable_cache;      // Enable conntrack cache lookup
    __u8  enable_bpf_helper; // Enable BPF conntrack helper (kernel >= 5.18)
    __u8  log_events;        // Log NAT detection events
    __u32 reserved[4];       // Reserved for future use
} __attribute__((packed));

// NAT statistics counters
// Tracks NAT detection performance and behavior
enum nat_stats_key {
    NAT_STATS_TOTAL_LOOKUPS = 0,      // Total NAT lookups attempted
    NAT_STATS_CACHE_HITS,             // Cache hits (fast path)
    NAT_STATS_CACHE_MISSES,           // Cache misses (slow path)
    NAT_STATS_BPF_HELPER_SUCCESS,     // BPF helper successful lookups
    NAT_STATS_BPF_HELPER_FAILED,      // BPF helper failures
    NAT_STATS_SNAT_DETECTED,          // SNAT detections
    NAT_STATS_DNAT_DETECTED,          // DNAT detections
    NAT_STATS_BOTH_DETECTED,          // Both SNAT and DNAT
    NAT_STATS_NO_NAT_DETECTED,        // No NAT detected
    NAT_STATS_RESTORE_SUCCESS,        // Successfully restored original address
    NAT_STATS_RESTORE_FAILED,         // Failed to restore original address
    NAT_STATS_MAX,                    // Total number of stats
};

// NAT statistics value
struct nat_stats_value {
    __u64 count;  // Counter value
};

//
// eBPF Map Definitions (controlled by ENABLE_NAT_SUPPORT macro)
//

#ifdef ENABLE_NAT_SUPPORT

// NAT conntrack cache map
// Stores NAT connection tracking information for address restoration
// PINNED: Shared between TC and XDP programs
// Note: This map is only defined when ENABLE_NAT_SUPPORT=1
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, MAX_CONNTRACK_ENTRIES);
    __type(key, struct conntrack_key);
    __type(value, struct conntrack_entry);
    __uint(pinning, LIBBPF_PIN_BY_NAME);  // Pin to /sys/fs/bpf/
} conntrack_cache_map SEC(".maps");

// NAT configuration map
// Controls NAT detection behavior and policy matching mode
// PINNED: Shared between TC and XDP programs
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);  // Always 0 (single config entry)
    __type(value, struct nat_config);
    __uint(pinning, LIBBPF_PIN_BY_NAME);  // Pin to /sys/fs/bpf/
} nat_config_map SEC(".maps");

// NAT statistics map
// Tracks NAT detection performance and behavior
// PINNED: Shared between TC and XDP programs
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, NAT_STATS_MAX);
    __type(key, __u32);  // enum nat_stats_key
    __type(value, struct nat_stats_value);
    __uint(pinning, LIBBPF_PIN_BY_NAME);  // Pin to /sys/fs/bpf/
} nat_stats_map SEC(".maps");

#endif /* ENABLE_NAT_SUPPORT */

//
// Helper function declarations
//

// Convert flow_key to conntrack_key (for cache lookup)
static __always_inline void flow_key_to_conntrack_key(
    const struct flow_key *flow,
    struct conntrack_key *ct_key)
{
    if (!flow || !ct_key) {
        return;
    }

    // Copy IP addresses
    __builtin_memcpy(ct_key->src_ip, flow->src_ip, sizeof(ct_key->src_ip));
    __builtin_memcpy(ct_key->dst_ip, flow->dst_ip, sizeof(ct_key->dst_ip));

    // Copy ports and protocol
    ct_key->src_port = flow->src_port;
    ct_key->dst_port = flow->dst_port;
    ct_key->protocol = flow->protocol;
    ct_key->ip_version = flow->ip_version;
    ct_key->pad = 0;
}

// Check if conntrack status indicates NAT
static __always_inline __u8 get_nat_type_from_status(__u32 status)
{
    __u8 nat_type = NAT_TYPE_NONE;

    if ((status & CONNTRACK_STATUS_SRC_NAT) && (status & CONNTRACK_STATUS_DST_NAT)) {
        nat_type = NAT_TYPE_BOTH;
    } else if (status & CONNTRACK_STATUS_SRC_NAT) {
        nat_type = NAT_TYPE_SNAT;
    } else if (status & CONNTRACK_STATUS_DST_NAT) {
        nat_type = NAT_TYPE_DNAT;
    }

    return nat_type;
}

// Compare two flow keys
static __always_inline bool flow_keys_equal(
    const struct flow_key *a,
    const struct flow_key *b)
{
    if (!a || !b) {
        return false;
    }

    // Compare IP addresses
    for (int i = 0; i < 4; i++) {
        if (a->src_ip[i] != b->src_ip[i] || a->dst_ip[i] != b->dst_ip[i]) {
            return false;
        }
    }

    // Compare ports and protocol
    return (a->src_port == b->src_port &&
            a->dst_port == b->dst_port &&
            a->protocol == b->protocol &&
            a->ip_version == b->ip_version);
}

// Increment NAT statistics counter
// Note: This function requires access to nat_stats_map, which should be
// defined in the main eBPF program
static __always_inline void increment_nat_stat(void *stats_map, enum nat_stats_key key)
{
    if (!stats_map) {
        return;
    }

    __u32 stat_key = (__u32)key;
    struct nat_stats_value *value = bpf_map_lookup_elem(stats_map, &stat_key);
    if (value) {
        __sync_fetch_and_add(&value->count, 1);
    } else {
        // Initialize if not exists
        struct nat_stats_value init_value = { .count = 1 };
        bpf_map_update_elem(stats_map, &stat_key, &init_value, BPF_ANY);
    }
}

//
// BPF Conntrack Helper Support (Kernel >= 5.18)
//
// The following section implements NAT detection using kernel's BPF conntrack
// helpers, which provide direct access to netfilter connection tracking.
//

#ifdef HAVE_BPF_CT_LOOKUP

// Forward declaration of kernel conntrack structure (opaque to BPF)
struct nf_conn;

// BPF conntrack lookup options
struct bpf_ct_opts {
    s32 netns_id;     // Network namespace ID (-1 = current)
    s32 error;        // Error code from lookup
    u8  l4proto;      // L4 protocol (IPPROTO_TCP/UDP)
    u8  dir;          // Direction (0 = original, 1 = reply)
    u8  reserved[2];  // Reserved for alignment
} __attribute__((aligned(4)));

// Socket tuple for CT lookup (matches kernel struct bpf_sock_tuple)
struct bpf_sock_tuple {
    union {
        struct {
            __be32 saddr;  // Source address (network byte order)
            __be32 daddr;  // Destination address (network byte order)
            __be16 sport;  // Source port (network byte order)
            __be16 dport;  // Destination port (network byte order)
        } ipv4;
        struct {
            __be32 saddr[4];  // Source address (network byte order)
            __be32 daddr[4];  // Destination address (network byte order)
            __be16 sport;     // Source port (network byte order)
            __be16 dport;     // Destination port (network byte order)
        } ipv6;
    };
};

// BPF kfunc declarations (kernel >= 5.18)
// These are kernel functions exposed to BPF via BTF

extern struct nf_conn *bpf_skb_ct_lookup(
    struct __sk_buff *skb_ctx,
    struct bpf_sock_tuple *tuple,
    u32 tuple_size,
    struct bpf_ct_opts *opts,
    u32 opts_size) __ksym;

extern struct nf_conn *bpf_xdp_ct_lookup(
    struct xdp_md *xdp_ctx,
    struct bpf_sock_tuple *tuple,
    u32 tuple_size,
    struct bpf_ct_opts *opts,
    u32 opts_size) __ksym;

extern void bpf_ct_release(struct nf_conn *ct) __ksym;

// Extract original tuple from conntrack entry
// Returns true if NAT is detected and original address differs from current
static __always_inline bool extract_original_tuple_from_ct(
    struct nf_conn *ct,
    const struct flow_key *current_key,
    struct flow_key *original_key,
    __u8 *nat_type)
{
    if (!ct || !current_key || !original_key || !nat_type) {
        return false;
    }

    // Note: Direct access to nf_conn structure requires BTF and CO-RE
    // For now, we use a simplified approach that relies on user-space sync
    // Full implementation would use bpf_core_read() to access nf_conn fields

    // TODO: Implement BTF-based nf_conn field access when kernel support is verified
    // This would require:
    // 1. BTF info for struct nf_conn
    // 2. CO-RE relocations for field offsets
    // 3. bpf_core_read() to safely access kernel structures

    // For Day 2 implementation, we mark this as not yet fully implemented
    // and rely on the cache-based lookup as primary path
    return false;
}

// Lookup NAT information using BPF conntrack helper (for TC programs)
static __always_inline bool lookup_conntrack_bpf_helper_tc(
    struct __sk_buff *skb,
    const struct flow_key *current_key,
    struct flow_key *original_key,
    __u8 *nat_type)
{
    if (!skb || !current_key || !original_key || !nat_type) {
        return false;
    }

    struct bpf_sock_tuple tuple = {};
    struct bpf_ct_opts opts = {
        .netns_id = -1,  // Current namespace
        .error = 0,
        .l4proto = current_key->protocol,
        .dir = 0,  // Original direction
    };

    struct nf_conn *ct = NULL;
    u32 tuple_size = 0;

    // Construct tuple based on IP version
    if (current_key->ip_version == 4) {
        // IPv4: use last element of ip array
        tuple.ipv4.saddr = bpf_htonl(current_key->src_ip[3]);
        tuple.ipv4.daddr = bpf_htonl(current_key->dst_ip[3]);
        tuple.ipv4.sport = bpf_htons(current_key->src_port);
        tuple.ipv4.dport = bpf_htons(current_key->dst_port);
        tuple_size = sizeof(tuple.ipv4);
    } else if (current_key->ip_version == 6) {
        // IPv6: use all 4 elements
        #pragma unroll
        for (int i = 0; i < 4; i++) {
            tuple.ipv6.saddr[i] = bpf_htonl(current_key->src_ip[i]);
            tuple.ipv6.daddr[i] = bpf_htonl(current_key->dst_ip[i]);
        }
        tuple.ipv6.sport = bpf_htons(current_key->src_port);
        tuple.ipv6.dport = bpf_htons(current_key->dst_port);
        tuple_size = sizeof(tuple.ipv6);
    } else {
        return false;
    }

    // Perform conntrack lookup
    ct = bpf_skb_ct_lookup(skb, &tuple, tuple_size, &opts, sizeof(opts));
    if (!ct) {
        return false;
    }

    // Extract original tuple and detect NAT
    bool success = extract_original_tuple_from_ct(ct, current_key, original_key, nat_type);

    // Release conntrack reference
    bpf_ct_release(ct);

    return success;
}

// Lookup NAT information using BPF conntrack helper (for XDP programs)
static __always_inline bool lookup_conntrack_bpf_helper_xdp(
    struct xdp_md *xdp,
    const struct flow_key *current_key,
    struct flow_key *original_key,
    __u8 *nat_type)
{
    if (!xdp || !current_key || !original_key || !nat_type) {
        return false;
    }

    struct bpf_sock_tuple tuple = {};
    struct bpf_ct_opts opts = {
        .netns_id = -1,  // Current namespace
        .error = 0,
        .l4proto = current_key->protocol,
        .dir = 0,  // Original direction
    };

    struct nf_conn *ct = NULL;
    u32 tuple_size = 0;

    // Construct tuple based on IP version
    if (current_key->ip_version == 4) {
        // IPv4: use last element of ip array
        tuple.ipv4.saddr = bpf_htonl(current_key->src_ip[3]);
        tuple.ipv4.daddr = bpf_htonl(current_key->dst_ip[3]);
        tuple.ipv4.sport = bpf_htons(current_key->src_port);
        tuple.ipv4.dport = bpf_htons(current_key->dst_port);
        tuple_size = sizeof(tuple.ipv4);
    } else if (current_key->ip_version == 6) {
        // IPv6: use all 4 elements
        #pragma unroll
        for (int i = 0; i < 4; i++) {
            tuple.ipv6.saddr[i] = bpf_htonl(current_key->src_ip[i]);
            tuple.ipv6.daddr[i] = bpf_htonl(current_key->dst_ip[i]);
        }
        tuple.ipv6.sport = bpf_htons(current_key->src_port);
        tuple.ipv6.dport = bpf_htons(current_key->dst_port);
        tuple_size = sizeof(tuple.ipv6);
    } else {
        return false;
    }

    // Perform conntrack lookup
    ct = bpf_xdp_ct_lookup(xdp, &tuple, tuple_size, &opts, sizeof(opts));
    if (!ct) {
        return false;
    }

    // Extract original tuple and detect NAT
    bool success = extract_original_tuple_from_ct(ct, current_key, original_key, nat_type);

    // Release conntrack reference
    bpf_ct_release(ct);

    return success;
}

#endif /* HAVE_BPF_CT_LOOKUP */

//
// Cache-based NAT Lookup (Fallback Method)
//
// This method relies on user-space synchronization of conntrack entries
// into the conntrack_cache_map. It's used as a fallback when BPF helpers
// are not available or when BTF access to nf_conn is not supported.
//

// Lookup NAT information from cache map
// Supports bidirectional lookup (original and reply directions)
static __always_inline bool lookup_conntrack_cache(
    void *cache_map,
    const struct flow_key *current_key,
    struct flow_key *original_key,
    __u8 *nat_type,
    bool *is_reply)
{
    if (!cache_map || !current_key || !original_key || !nat_type || !is_reply) {
        return false;
    }

    struct conntrack_key cache_key = {};

    // Convert current flow_key to conntrack_key
    flow_key_to_conntrack_key(current_key, &cache_key);

    // Try forward direction lookup (current flow)
    struct conntrack_entry *entry = bpf_map_lookup_elem(cache_map, &cache_key);
    if (entry) {
        // Found in forward direction - use original tuple
        *original_key = entry->original_tuple;
        *nat_type = entry->nat_type;
        *is_reply = false;
        return true;
    }

    // Try reverse direction lookup (reply flow)
    // Swap source and destination for reverse lookup
    struct conntrack_key reversed_key = {
        .src_port = cache_key.dst_port,
        .dst_port = cache_key.src_port,
        .protocol = cache_key.protocol,
        .ip_version = cache_key.ip_version,
        .pad = 0,
    };

    // Copy reversed IPs
    __builtin_memcpy(reversed_key.src_ip, cache_key.dst_ip, sizeof(reversed_key.src_ip));
    __builtin_memcpy(reversed_key.dst_ip, cache_key.src_ip, sizeof(reversed_key.dst_ip));

    entry = bpf_map_lookup_elem(cache_map, &reversed_key);
    if (entry) {
        // Found in reverse direction - use reply tuple
        *original_key = entry->reply_tuple;
        *nat_type = entry->nat_type;
        *is_reply = true;
        return true;
    }

    return false;
}

//
// Unified NAT Detection Interface
//
// This is the main entry point for NAT detection. It tries multiple methods
// in order of preference and returns the original addresses for policy matching.
//

// Detect NAT and restore original addresses (generic version for use with maps)
// This version takes map pointers as parameters for flexibility
static __always_inline bool detect_nat_and_restore_with_maps(
    void *ctx,
    bool is_xdp,
    const struct flow_key *current_key,
    struct flow_key *original_key,
    __u8 *nat_type,
    void *config_map,
    void *cache_map,
    void *stats_map)
{
    if (!ctx || !current_key || !original_key || !nat_type) {
        return false;
    }

    // Increment total lookups counter
    if (stats_map) {
        increment_nat_stat(stats_map, NAT_STATS_TOTAL_LOOKUPS);
    }

    // Check NAT configuration
    __u32 config_key = 0;
    struct nat_config *config = NULL;
    if (config_map) {
        config = bpf_map_lookup_elem(config_map, &config_key);
    }

    // If NAT is disabled or config not found, use current addresses
    if (!config || config->match_mode == NAT_MATCH_MODE_TRANSLATED) {
        *original_key = *current_key;
        *nat_type = NAT_TYPE_NONE;
        if (stats_map) {
            increment_nat_stat(stats_map, NAT_STATS_NO_NAT_DETECTED);
        }
        return true;
    }

    bool found = false;
    bool is_reply = false;

    #ifdef HAVE_BPF_CT_LOOKUP
    // Try BPF helper first (if enabled and supported)
    if (config->enable_bpf_helper) {
        if (is_xdp) {
            found = lookup_conntrack_bpf_helper_xdp((struct xdp_md *)ctx,
                current_key, original_key, nat_type);
        } else {
            found = lookup_conntrack_bpf_helper_tc((struct __sk_buff *)ctx,
                current_key, original_key, nat_type);
        }

        if (found) {
            if (stats_map) {
                increment_nat_stat(stats_map, NAT_STATS_BPF_HELPER_SUCCESS);
            }
        } else {
            if (stats_map) {
                increment_nat_stat(stats_map, NAT_STATS_BPF_HELPER_FAILED);
            }
        }
    }
    #endif

    // Fallback to cache lookup if BPF helper failed or not enabled
    if (!found && config->enable_cache && cache_map) {
        found = lookup_conntrack_cache(cache_map, current_key,
            original_key, nat_type, &is_reply);

        if (found) {
            if (stats_map) {
                increment_nat_stat(stats_map, NAT_STATS_CACHE_HITS);
            }
        } else {
            if (stats_map) {
                increment_nat_stat(stats_map, NAT_STATS_CACHE_MISSES);
            }
        }
    }

    // If no NAT info found, use current addresses
    if (!found) {
        *original_key = *current_key;
        *nat_type = NAT_TYPE_NONE;
        if (stats_map) {
            increment_nat_stat(stats_map, NAT_STATS_NO_NAT_DETECTED);
            increment_nat_stat(stats_map, NAT_STATS_RESTORE_FAILED);
        }
        return true;
    }

    // Update NAT type statistics
    if (stats_map) {
        if (*nat_type == NAT_TYPE_SNAT) {
            increment_nat_stat(stats_map, NAT_STATS_SNAT_DETECTED);
        } else if (*nat_type == NAT_TYPE_DNAT) {
            increment_nat_stat(stats_map, NAT_STATS_DNAT_DETECTED);
        } else if (*nat_type == NAT_TYPE_BOTH) {
            increment_nat_stat(stats_map, NAT_STATS_BOTH_DETECTED);
        }
        increment_nat_stat(stats_map, NAT_STATS_RESTORE_SUCCESS);
    }

    return true;
}

#endif /* __NAT_SUPPORT_H__ */
