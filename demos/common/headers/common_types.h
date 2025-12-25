// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Common types for eBPF demo programs (Simplified version)
 *
 * This is a simplified version of src/bpf/headers/common_types.h
 * Simplifications:
 * - IPv4 only (32-bit IP addresses instead of 128-bit)
 * - No VLAN support
 * - No process-level policy
 * - Reduced features for learning purposes
 */

#ifndef __COMMON_TYPES_DEMO_H__
#define __COMMON_TYPES_DEMO_H__

// Map size limits
#define MAX_ENTRIES_SESSION 100000
#define MAX_ENTRIES_POLICY 10000
#define MAX_ENTRIES_WILDCARD_POLICY 1000

// Session timeout configuration (in nanoseconds)
#define DEFAULT_TCP_TIMEOUT_NS      (300ULL * 1000000000ULL)  // 5 minutes
#define DEFAULT_UDP_TIMEOUT_NS      (60ULL * 1000000000ULL)   // 1 minute
#define DEFAULT_ICMP_TIMEOUT_NS     (30ULL * 1000000000ULL)   // 30 seconds

// 5-tuple flow key for session tracking (IPv4 only - simplified)
struct flow_key {
    __u32 src_ip;      // Source IP address (32 bits for IPv4)
    __u32 dst_ip;      // Destination IP address (32 bits for IPv4)
    __u16 src_port;    // Source port
    __u16 dst_port;    // Destination port
    __u8  protocol;    // Protocol (TCP=6, UDP=17, ICMP=1)
    __u8  pad[3];      // Padding for alignment
} __attribute__((packed));

// TCP state tracking (standard TCP FSM)
enum tcp_state {
    TCP_STATE_CLOSED = 0,
    TCP_STATE_SYN_SENT,
    TCP_STATE_SYN_RECV,
    TCP_STATE_ESTABLISHED,
    TCP_STATE_FIN_WAIT1,
    TCP_STATE_FIN_WAIT2,
    TCP_STATE_CLOSE_WAIT,
    TCP_STATE_CLOSING,
    TCP_STATE_LAST_ACK,
    TCP_STATE_TIME_WAIT,
};

// Policy action
enum policy_action {
    POLICY_ACTION_ALLOW = 0,
    POLICY_ACTION_DENY = 1,
};

// Policy direction
enum policy_direction {
    POLICY_DIR_ANY = 0,       // Match both ingress and egress
    POLICY_DIR_INGRESS = 1,   // Match only ingress traffic
    POLICY_DIR_EGRESS = 2,    // Match only egress traffic
};

// Session value stored in LRU_HASH map (simplified version)
struct session_value {
    __u64 created_ts;         // Session creation timestamp (nanoseconds)
    __u64 last_seen_ts;       // Last packet timestamp
    __u64 packets_to_server;  // Packets from client to server
    __u64 packets_to_client;  // Packets from server to client
    __u64 bytes_to_server;    // Bytes from client to server
    __u64 bytes_to_client;    // Bytes from server to client

    __u8  tcp_state;          // TCP state machine (enum tcp_state)
    __u8  policy_action;      // Matched policy action (enum policy_action)
    __u16 pad;                // Padding for alignment
    __u32 rule_id;            // Matched rule ID
} __attribute__((packed));

// Policy key for exact matching (IPv4 only - simplified)
struct policy_key {
    __u32 src_ip;      // Source IP address
    __u32 dst_ip;      // Destination IP address
    __u16 src_port;    // Source port
    __u16 dst_port;    // Destination port
    __u8  protocol;    // Protocol (TCP/UDP/ICMP)
    __u8  direction;   // Policy direction (enum policy_direction)
    __u16 pad;         // Padding for alignment
} __attribute__((packed));

// Policy value
struct policy_value {
    __u8  action;      // Policy action (enum policy_action)
    __u8  pad;         // Padding
    __u16 priority;    // Policy priority (higher = more important)
    __u32 rule_id;     // Rule ID for tracking
    __u64 hit_count;   // Number of times this policy was matched
} __attribute__((packed));

// Wildcard policy for matching with wildcards (0 = match any)
// Used in array map for linear searching (IPv4 only - simplified)
struct wildcard_policy {
    __u32 src_ip;          // Source IP address
    __u32 src_ip_mask;     // Mask for source IP (0xffffffff = exact, 0 = any)
    __u32 dst_ip;          // Destination IP address
    __u32 dst_ip_mask;     // Mask for destination IP (0xffffffff = exact, 0 = any)
    __u16 src_port;        // 0 = any port
    __u16 dst_port;        // 0 = any port
    __u8  protocol;        // 0 = any protocol
    __u8  action;          // Policy action (enum policy_action)
    __u8  direction;       // Policy direction (enum policy_direction)
    __u8  pad;             // Padding for alignment
    __u16 priority;        // Policy priority (higher = more important)
    __u16 pad2;            // Padding
    __u32 rule_id;         // Rule ID (0 = empty slot)
} __attribute__((packed));

// Statistics counters
enum stats_key {
    STATS_TOTAL_PACKETS = 0,
    STATS_ALLOWED_PACKETS,
    STATS_DENIED_PACKETS,
    STATS_NEW_SESSIONS,
    STATS_CLOSED_SESSIONS,
    STATS_ACTIVE_SESSIONS,
    STATS_POLICY_HITS,
    STATS_POLICY_MISSES,
    // Direction-specific statistics
    STATS_INGRESS_PACKETS,
    STATS_EGRESS_PACKETS,
    // Protocol-specific statistics
    STATS_TCP_PACKETS,
    STATS_UDP_PACKETS,
    STATS_ICMP_PACKETS,
    // TCP-specific statistics
    STATS_TCP_SYN,
    STATS_TCP_FIN,
    STATS_TCP_RST,
    // Error statistics
    STATS_PARSE_ERRORS,
    STATS_MAX,
};

// Flow event types
enum flow_event_type {
    FLOW_EVENT_NEW = 0,      // New connection established
    FLOW_EVENT_UPDATE = 1,   // Connection active/updated
    FLOW_EVENT_CLOSED = 2,   // Connection closed
};

// Flow direction
enum flow_direction {
    FLOW_DIRECTION_INGRESS = 0,
    FLOW_DIRECTION_EGRESS = 1,
};

// Flow event for reporting to control plane (IPv4 only - simplified)
struct flow_event {
    // 5-tuple identification
    __u32 src_ip;      // Source IP address
    __u32 dst_ip;      // Destination IP address
    __u16 src_port;
    __u16 dst_port;

    // Packet metadata
    __u8  protocol;
    __u8  event_type;  // enum flow_event_type
    __u8  direction;   // enum flow_direction
    __u8  tcp_flags;   // TCP flags (SYN, FIN, RST, etc.)

    // Traffic statistics
    __u64 packet_count;   // Total packets in this flow
    __u64 byte_count;     // Total bytes in this flow
    __u64 timestamp_ns;   // Event timestamp in nanoseconds

    // TCP state
    __u8  tcp_state;      // TCP state (enum tcp_state)
    __u8  pad[3];         // Padding

    // Policy context
    __u32 policy_id;      // Matched policy/rule ID
    __u8  policy_action;  // enum policy_action
    __u8  pad2[3];        // Padding
} __attribute__((packed));

#endif /* __COMMON_TYPES_DEMO_H__ */
