// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* IPCache data structures for IP to identity mapping */

#ifndef __IPCACHE_H__
#define __IPCACHE_H__

// Maximum entries in the IPCache map
#define MAX_ENTRIES_IPCACHE 65536

// IPCache key structure for LPM trie
// Uses longest prefix match for efficient CIDR lookups
struct ipcache_key {
    __u32 prefixlen;     // Number of bits in the prefix (for LPM)
    __u8  ip[16];        // IPv4/IPv6 address (IPv4 stored as IPv4-mapped IPv6)
} __attribute__((packed));

// IPCache value structure
struct ipcache_value {
    __u32 identity;      // Security identity ID
    __u8  pad[4];        // Padding for alignment
} __attribute__((packed));

// IPCache map definition
// Uses LPM_TRIE for efficient longest-prefix matching
struct {
    __uint(type, BPF_MAP_TYPE_LPM_TRIE);
    __uint(max_entries, MAX_ENTRIES_IPCACHE);
    __type(key, struct ipcache_key);
    __type(value, struct ipcache_value);
    __uint(pinning, LIBBPF_PIN_BY_NAME);
    __uint(map_flags, BPF_F_NO_PREALLOC);
} ipcache_map SEC(".maps");

// Helper function to lookup identity from IPCache
// Returns the identity ID or 0 (IdentityUnknown) if not found
static __always_inline __u32 ipcache_lookup(__u32 *ip, __u8 ip_version) {
    struct ipcache_key key = {};

    if (ip_version == 4) {
        // IPv4: Create IPv4-mapped IPv6 key
        // Prefix length: /32 IPv4 = /128 IPv6 (32 + 96 = 128)
        key.prefixlen = 128;
        // First 10 bytes are 0
        key.ip[10] = 0xff;
        key.ip[11] = 0xff;
        // Copy IPv4 address (stored in ip[3] for IPv4)
        __builtin_memcpy(&key.ip[12], &ip[3], 4);
    } else {
        // IPv6: Use full address
        key.prefixlen = 128;
        __builtin_memcpy(key.ip, ip, 16);
    }

    struct ipcache_value *val = bpf_map_lookup_elem(&ipcache_map, &key);
    if (val) {
        return val->identity;
    }

    // TODO: For true LPM support, we would need to try shorter prefixes
    // However, BPF_MAP_TYPE_LPM_TRIE handles this automatically
    return 0;  // IdentityUnknown
}

// Helper function to lookup identity for IPv4 address (single u32)
static __always_inline __u32 ipcache_lookup_ipv4(__u32 ipv4_le) {
    __u32 ip[4] = {0, 0, 0, ipv4_le};
    return ipcache_lookup(ip, 4);
}

#endif /* __IPCACHE_H__ */
