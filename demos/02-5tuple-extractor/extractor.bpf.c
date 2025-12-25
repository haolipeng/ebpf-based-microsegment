// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Demo 2: 5-Tuple Extractor
 *
 * This program demonstrates packet parsing and 5-tuple extraction.
 *
 * What it does:
 * - Parses Ethernet, IPv4, TCP/UDP headers
 * - Extracts 5-tuple: (src_ip, dst_ip, src_port, dst_port, protocol)
 * - Prints flow information using bpf_printk()
 * - Demonstrates proper boundary checking (required by eBPF verifier)
 *
 * Learning objectives:
 * 1. Understand packet structure (Ethernet → IP → TCP/UDP)
 * 2. Learn boundary checking patterns (data_end validation)
 * 3. Use helper functions from common headers
 * 4. Work with network byte order (bpf_ntohs, bpf_htonl)
 * 5. Debug with bpf_printk() to visualize packet contents
 */

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

// Include our common headers
#include "../common/headers/common_types.h"
#include "../common/headers/flow_processing.h"

char LICENSE[] SEC("license") = "GPL";

// TC action codes
#define TC_ACT_OK    0
#define TC_ACT_SHOT  2

// Map to count parsed packets by protocol
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 256);  // One entry per protocol number (0-255)
    __type(key, __u32);        // Protocol number
    __type(value, __u64);      // Packet count
} protocol_counter SEC(".maps");

/* Helper function to convert IP address to string format (for printing)
 *
 * Note: bpf_printk() supports %pI4 format specifier for IPv4 addresses,
 * so we can print IP addresses directly.
 */
static __always_inline void print_flow_info(struct flow_key *key)
{
    // Print 5-tuple information
    // %pI4 = print IPv4 address in dotted-decimal notation
    // %d = decimal integer
    bpf_printk("[FLOW] Proto=%d  %pI4:%d",
               key->protocol,
               &key->src_ip,
               bpf_ntohs(key->src_port));  // Convert from network to host byte order

    bpf_printk("      → %pI4:%d",
               &key->dst_ip,
               bpf_ntohs(key->dst_port));
}

/* Main TC eBPF program
 *
 * This program intercepts every packet and attempts to extract the 5-tuple.
 * If successful, it prints the flow information and updates protocol statistics.
 */
SEC("tc")
int tc_extract_5tuple(struct __sk_buff *skb)
{
    // Packet data pointers
    void *data = (void *)(unsigned long)skb->data;
    void *data_end = (void *)(unsigned long)skb->data_end;

    // Flow key to store extracted 5-tuple
    struct flow_key key = {0};

    // Extract 5-tuple from packet
    // This function is defined in ../common/headers/flow_processing.h
    // It handles:
    // 1. Ethernet header parsing
    // 2. IPv4 header parsing
    // 3. TCP/UDP header parsing
    // 4. Boundary checking at every step
    if (extract_flow_key_from_packet(data, data_end, &key) < 0) {
        // Failed to parse packet (could be non-IPv4, malformed, etc.)
        // Just allow it to pass through without logging
        return TC_ACT_OK;
    }

    // Successfully extracted 5-tuple!
    // Print it for debugging
    print_flow_info(&key);

    // Update protocol statistics
    __u32 proto_key = (__u32)key.protocol;
    __u64 *count = bpf_map_lookup_elem(&protocol_counter, &proto_key);
    if (count) {
        __sync_fetch_and_add(count, 1);
    }

    // Allow packet to continue
    return TC_ACT_OK;
}
