// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Fragment Tracking and Reassembly Support
 *
 * This header file implements IP fragment detection and tracking logic
 * to handle fragmented packets in the microsegmentation system.
 *
 * Problem:
 * - Fragmented packets cannot be parsed for complete 5-tuple
 * - Only the first fragment contains L4 headers (TCP/UDP ports)
 * - Subsequent fragments lack port information for policy matching
 * - Fragments can be used to bypass security policies
 *
 * Solution:
 * - Detect IP fragmentation (IPv4 and IPv6)
 * - Track first fragment's complete flow key and policy action
 * - Associate subsequent fragments with first fragment's policy
 * - Implement fragment timeout and cleanup
 * - Support three handling modes: STRICT, NORMAL, PERMISSIVE
 *
 * Prerequisites:
 * - Include common_types.h for flow_key structure
 * - Include vmlinux.h for IP header structures
 */

#ifndef __FRAGMENT_TRACKING_H__
#define __FRAGMENT_TRACKING_H__

#include "common_types.h"

// Fragment handling modes
#define FRAG_MODE_STRICT      0  // Drop all fragments (safest, may break some apps)
#define FRAG_MODE_NORMAL      1  // Allow first fragment if policy matches, drop subsequent fragments (recommended)
#define FRAG_MODE_PERMISSIVE  2  // Allow first fragment if policy matches, allow subsequent fragments (least safe)

// IPv4 fragment flags and offset
#define IP_MF  0x2000  // More Fragments flag (network byte order)
#define IP_OFFSET 0x1FFF  // Fragment offset mask (network byte order)

// IPv6 fragment header
#define NEXTHDR_FRAGMENT 44  // IPv6 Fragment extension header

// Fragment timeout (in nanoseconds)
#define FRAG_TIMEOUT_NS (30ULL * 1000000000ULL)  // 30 seconds

/* Fragment Key - Identifies a fragmented datagram
 *
 * Used as key in fragment tracking map to identify fragments
 * belonging to the same original datagram.
 *
 * IPv4: src_ip + dst_ip + identification + protocol
 * IPv6: src_ip + dst_ip + fragment_id + protocol
 */
struct frag_key {
	__u32 src_ip[4];      // Source IP (IPv4-mapped or IPv6)
	__u32 dst_ip[4];      // Destination IP (IPv4-mapped or IPv6)
	__u32 frag_id;        // IPv4: identification field, IPv6: fragment ID
	__u8  protocol;       // IP protocol (TCP/UDP/ICMP)
	__u8  ip_version;     // 4 or 6
	__u16 reserved;       // Padding for alignment
};

/* Fragment Value - Stores first fragment information
 *
 * Cached information from the first fragment:
 * - Complete flow key (5-tuple with ports)
 * - Policy action determined for first fragment
 * - Timestamp for timeout cleanup
 */
struct frag_value {
	struct flow_key complete_key;  // Complete 5-tuple from first fragment
	__u64 timestamp;               // Fragment arrival timestamp (bpf_ktime_get_ns)
	__u8  policy_action;           // Policy action for this flow (ALLOW/DENY)
	__u8  reserved[7];             // Padding for alignment
};

/* Fragment Configuration - Global fragment handling settings
 *
 * Controls how the system handles fragmented packets.
 */
struct frag_config {
	__u8  mode;            // Fragment handling mode (FRAG_MODE_*)
	__u8  log_events;      // Log fragment events (0 = disabled, 1 = enabled)
	__u16 reserved;        // Padding
	__u64 timeout_ns;      // Fragment timeout in nanoseconds
	__u64 reserved2[2];    // Reserved for future use
};

/* Fragment Statistics - Per-CPU counters
 *
 * Statistics keys (must match enum in Go code)
 */
enum frag_stats_key {
	FRAG_STAT_FIRST_FRAGMENTS = 0,   // First fragments processed
	FRAG_STAT_SUBSEQUENT_FRAGMENTS,  // Subsequent fragments processed
	FRAG_STAT_FRAGMENTS_ALLOWED,     // Fragments allowed through
	FRAG_STAT_FRAGMENTS_DENIED,      // Fragments denied
	FRAG_STAT_FRAGMENTS_TIMEOUT,     // Fragments timed out
	FRAG_STAT_CACHE_HITS,            // Fragment cache hits
	FRAG_STAT_CACHE_MISSES,          // Fragment cache misses
	FRAG_STAT_IPV4_FRAGMENTS,        // IPv4 fragments detected
	FRAG_STAT_IPV6_FRAGMENTS,        // IPv6 fragments detected
	FRAG_STAT_MAX
};

/* IPv4 Fragment Type Classification
 *
 * Enumeration for efficient fragment type detection.
 * This enum is used by get_ipv4_frag_type() to classify packets
 * in a single pass, avoiding multiple bpf_ntohs() calls.
 */
enum ipv4_frag_type {
	IPV4_FRAG_TYPE_NONE = 0,       // Non-fragmented packet (MF=0, Offset=0)
	IPV4_FRAG_TYPE_FIRST,          // First fragment (MF=1, Offset=0)
	IPV4_FRAG_TYPE_SUBSEQUENT,     // Subsequent fragment (Offset>0)
};

/* get_ipv4_frag_type - Optimized IPv4 fragment type detection
 *
 * @iph: IPv4 header pointer
 *
 * Returns:
 *   IPV4_FRAG_TYPE_NONE       - Non-fragmented packet
 *   IPV4_FRAG_TYPE_FIRST      - First fragment (contains L4 headers)
 *   IPV4_FRAG_TYPE_SUBSEQUENT - Subsequent fragment (no L4 headers)
 *
 * Performance Optimization:
 * This function performs only ONE bpf_ntohs() call and determines
 * the fragment type in a single pass. This is significantly more
 * efficient than calling is_ipv4_fragment(), is_ipv4_first_fragment(),
 * and is_ipv4_subsequent_fragment() separately (which would call
 * bpf_ntohs() three times).
 *
 * Fragment Type Logic:
 * - Non-Fragment: MF=0 AND Offset=0
 * - First Fragment: MF=1 AND Offset=0
 * - Subsequent Fragment: Offset>0 (MF can be 0 or 1)
 *
 * Usage Example:
 *   enum ipv4_frag_type frag_type = get_ipv4_frag_type(iph);
 *   if (frag_type != IPV4_FRAG_TYPE_NONE) {
 *       if (frag_type == IPV4_FRAG_TYPE_FIRST) {
 *           // Handle first fragment
 *       } else {
 *           // Handle subsequent fragment
 *       }
 *   }
 */
static __always_inline enum ipv4_frag_type get_ipv4_frag_type(struct iphdr *iph)
{
	__u16 frag_off = bpf_ntohs(iph->frag_off);  // Single byte order conversion

	bool has_mf = frag_off & (IP_MF >> 8);      // More Fragments flag set?
	bool has_offset = frag_off & (IP_OFFSET >> 8);  // Fragment offset > 0?

	// Non-fragmented packet: MF=0 and Offset=0
	if (!has_mf && !has_offset) {
		return IPV4_FRAG_TYPE_NONE;
	}

	// First fragment: MF=1 and Offset=0
	if (has_mf && !has_offset) {
		return IPV4_FRAG_TYPE_FIRST;
	}

	// Subsequent fragment: Offset>0 (regardless of MF flag)
	return IPV4_FRAG_TYPE_SUBSEQUENT;
}

/* is_ipv4_fragment - Check if IPv4 packet is a fragment
 *
 * @iph: IPv4 header pointer
 *
 * Returns:
 *   true if packet is a fragment (MF flag set or offset > 0)
 *   false if packet is not fragmented
 *
 * IPv4 Fragmentation Detection:
 * - frag_off field contains flags (3 bits) + fragment offset (13 bits)
 * - Bit 0: Reserved (must be 0)
 * - Bit 1: DF (Don't Fragment)
 * - Bit 2: MF (More Fragments)
 * - Bits 3-15: Fragment Offset (in 8-byte units)
 *
 * Fragment Types:
 * - First Fragment: MF=1, Offset=0
 * - Middle Fragment: MF=1, Offset>0
 * - Last Fragment: MF=0, Offset>0
 * - Non-Fragment: MF=0, Offset=0
 */
static __always_inline bool is_ipv4_fragment(struct iphdr *iph)
{
	__u16 frag_off = bpf_ntohs(iph->frag_off);
	return (frag_off & (IP_MF >> 8)) || (frag_off & (IP_OFFSET >> 8));
}

/* is_ipv4_first_fragment - Check if this is the first fragment
 *
 * @iph: IPv4 header pointer
 *
 * Returns:
 *   true if this is the first fragment (offset = 0, MF = 1)
 *   false otherwise
 *
 * First fragment characteristics:
 * - Contains L4 header (TCP/UDP ports available)
 * - Fragment offset = 0
 * - More Fragments flag = 1
 */
static __always_inline bool is_ipv4_first_fragment(struct iphdr *iph)
{
	__u16 frag_off = bpf_ntohs(iph->frag_off);
	// First fragment: MF=1 and Offset=0
	return (frag_off & (IP_MF >> 8)) && !(frag_off & (IP_OFFSET >> 8));
}

/* is_ipv4_subsequent_fragment - Check if this is a subsequent fragment
 *
 * @iph: IPv4 header pointer
 *
 * Returns:
 *   true if this is a subsequent fragment (offset > 0)
 *   false otherwise
 *
 * Subsequent fragment characteristics:
 * - Does NOT contain L4 header (no port information)
 * - Fragment offset > 0
 * - MF flag may be 0 (last fragment) or 1 (middle fragment)
 */
static __always_inline bool is_ipv4_subsequent_fragment(struct iphdr *iph)
{
	__u16 frag_off = bpf_ntohs(iph->frag_off);
	// Subsequent fragment: Offset > 0
	return (frag_off & (IP_OFFSET >> 8)) != 0;
}

/* extract_ipv4_frag_key - Extract fragment key from IPv4 header
 *
 * @iph: IPv4 header pointer
 * @frag_key: Output parameter - fragment key
 *
 * Extracts the fragment identification tuple for IPv4:
 * - Source IP
 * - Destination IP
 * - Identification field (16-bit ID for fragment reassembly)
 * - Protocol
 */
static __always_inline void extract_ipv4_frag_key(
	struct iphdr *iph,
	struct frag_key *key)
{
	// Convert IPv4 addresses to IPv4-mapped IPv6 format
	key->src_ip[0] = 0;
	key->src_ip[1] = 0;
	key->src_ip[2] = bpf_htonl(0x0000ffff);
	key->src_ip[3] = iph->saddr;

	key->dst_ip[0] = 0;
	key->dst_ip[1] = 0;
	key->dst_ip[2] = bpf_htonl(0x0000ffff);
	key->dst_ip[3] = iph->daddr;

	// IPv4 uses the identification field for fragment reassembly
	key->frag_id = bpf_ntohs(iph->id);
	key->protocol = iph->protocol;
	key->ip_version = 4;
	key->reserved = 0;
}

/* IPv6 Fragment Extension Header
 *
 * Defined in RFC 8200 Section 4.5
 * This structure matches the IPv6 fragment extension header format.
 */
struct ipv6_frag_hdr {
	__u8  nexthdr;          // Next header type
	__u8  reserved;         // Reserved (must be 0)
	__be16 frag_off;        // Fragment offset (13 bits) + flags (3 bits)
	__be32 identification;  // Fragment identification
};

/* is_ipv6_fragment - Check if IPv6 packet contains fragment header
 *
 * @nexthdr: Next header value from IPv6 base header
 *
 * Returns:
 *   true if nexthdr indicates Fragment extension header (44)
 *   false otherwise
 *
 * Note: This is a basic check. Full implementation should walk
 * through extension headers to find the fragment header.
 */
static __always_inline bool is_ipv6_fragment(__u8 nexthdr)
{
	return nexthdr == NEXTHDR_FRAGMENT;
}

/* is_ipv6_first_fragment - Check if this is the first IPv6 fragment
 *
 * @frag_hdr: IPv6 fragment extension header pointer
 *
 * Returns:
 *   true if this is the first fragment (offset = 0, M flag = 1)
 *   false otherwise
 */
static __always_inline bool is_ipv6_first_fragment(struct ipv6_frag_hdr *frag_hdr)
{
	__u16 frag_off = bpf_ntohs(frag_hdr->frag_off);
	// First fragment: M=1 (bit 0) and Offset=0 (bits 3-15)
	return (frag_off & 0x0001) && ((frag_off & 0xFFF8) == 0);
}

/* is_ipv6_subsequent_fragment - Check if this is a subsequent IPv6 fragment
 *
 * @frag_hdr: IPv6 fragment extension header pointer
 *
 * Returns:
 *   true if this is a subsequent fragment (offset > 0)
 *   false otherwise
 */
static __always_inline bool is_ipv6_subsequent_fragment(struct ipv6_frag_hdr *frag_hdr)
{
	__u16 frag_off = bpf_ntohs(frag_hdr->frag_off);
	// Subsequent fragment: Offset > 0 (bits 3-15)
	return (frag_off & 0xFFF8) != 0;
}

/* extract_ipv6_frag_key - Extract fragment key from IPv6 headers
 *
 * @ip6h: IPv6 base header pointer
 * @frag_hdr: IPv6 fragment extension header pointer
 * @frag_key: Output parameter - fragment key
 *
 * Extracts the fragment identification tuple for IPv6:
 * - Source IPv6 address (128 bits)
 * - Destination IPv6 address (128 bits)
 * - Fragment ID (32 bits)
 * - Next header (protocol after fragment header)
 */
static __always_inline void extract_ipv6_frag_key(
	struct ipv6hdr *ip6h,
	struct ipv6_frag_hdr *frag_hdr,
	struct frag_key *key)
{
	// Copy IPv6 addresses (128 bits = 4 x 32 bits)
	#pragma unroll
	for (int i = 0; i < 4; i++) {
		key->src_ip[i] = ip6h->saddr.in6_u.u6_addr32[i];
		key->dst_ip[i] = ip6h->daddr.in6_u.u6_addr32[i];
	}

	// IPv6 uses the identification field in fragment header
	key->frag_id = bpf_ntohl(frag_hdr->identification);
	key->protocol = frag_hdr->nexthdr;  // Protocol after fragment header
	key->ip_version = 6;
	key->reserved = 0;
}

#endif /* __FRAGMENT_TRACKING_H__ */
