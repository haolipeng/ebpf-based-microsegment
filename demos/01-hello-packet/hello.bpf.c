// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Demo 1: Hello Packet Counter
 *
 * This is the simplest eBPF program for learning TC (Traffic Control) Hook.
 *
 * What it does:
 * - Counts every packet passing through the network interface
 * - Stores the count in an ARRAY map
 * - Allows all packets to pass through (TC_ACT_OK)
 *
 * Learning objectives:
 * 1. Understand TC Hook and SEC("tc") section
 * 2. Learn eBPF Map basics (ARRAY type)
 * 3. Understand packet context (struct __sk_buff)
 * 4. Learn return values (TC_ACT_OK vs TC_ACT_SHOT)
 * 5. Use bpf_printk() for debugging
 */

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

// License declaration (required for eBPF programs)
char LICENSE[] SEC("license") = "GPL";

// TC action codes
#define TC_ACT_OK    0  // Allow packet to pass
#define TC_ACT_SHOT  2  // Drop packet

// Map to store packet counter
// - Type: BPF_MAP_TYPE_ARRAY (fixed-size array, indexed by integer)
// - Max entries: 1 (we only need a single counter)
// - Key: __u32 (array index, always 0 in our case)
// - Value: __u64 (64-bit counter to avoid overflow)
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, __u64);
} packet_counter SEC(".maps");

/* TC eBPF program entry point
 *
 * @skb: Packet context (struct __sk_buff)
 *      - Contains packet metadata: length, protocol, etc.
 *      - Provides access to packet data via data/data_end pointers
 *
 * Returns:
 *      - TC_ACT_OK (0): Allow packet to continue
 *      - TC_ACT_SHOT (2): Drop packet
 *
 * This function is called for EVERY packet that passes through the interface
 * where this eBPF program is attached (ingress or egress).
 */
SEC("tc")
int tc_hello_packet(struct __sk_buff *skb)
{
    // Map key (always 0 for single counter)
    __u32 key = 0;

    // Lookup the counter from map
    // bpf_map_lookup_elem() returns a pointer to the value, or NULL if not found
    __u64 *count = bpf_map_lookup_elem(&packet_counter, &key);

    if (count) {
        // Atomically increment the counter
        // __sync_fetch_and_add() is used to safely increment in concurrent environment
        // (multiple CPU cores may execute this program simultaneously)
        __sync_fetch_and_add(count, 1);

        // Print debug message (visible in /sys/kernel/debug/tracing/trace_pipe)
        // Note: bpf_printk() has performance overhead, use sparingly in production
        bpf_printk("Hello! Packet #%llu received (len=%d bytes)\n",
                   *count, skb->len);
    }

    // Allow packet to pass through
    // Returning TC_ACT_OK means "do nothing special, let packet continue"
    return TC_ACT_OK;
}
