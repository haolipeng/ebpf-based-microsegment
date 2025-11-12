// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Common types shared between eBPF and user-space programs */

#ifndef __COMMON_TYPES_H__
#define __COMMON_TYPES_H__

#define MAX_ENTRIES_SESSION 100000
#define MAX_ENTRIES_POLICY 10000
#define MAX_ENTRIES_WILDCARD_POLICY 1000

// 5-tuple flow key for session tracking
struct flow_key {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8  protocol;
    __u8  pad[3];  // Padding for alignment
} __attribute__((packed));

// Session state tracking
enum session_state {
    SESSION_STATE_NEW = 0,
    SESSION_STATE_ESTABLISHED,
    SESSION_STATE_CLOSING,
    SESSION_STATE_CLOSED,
};

// TCP state tracking
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
    POLICY_ACTION_DENY,
    POLICY_ACTION_LOG,
};

// Policy direction (for egress support)
enum policy_direction {
    POLICY_DIR_ANY = 0,       // Match both ingress and egress
    POLICY_DIR_INGRESS = 1,   // Match only ingress traffic
    POLICY_DIR_EGRESS = 2,    // Match only egress traffic
};

// Session value stored in LRU_HASH map
struct session_value {
    __u64 created_ts;         // Session creation timestamp (nanoseconds)
    __u64 last_seen_ts;       // Last packet timestamp
    __u64 packets_to_server;  // Packets from client to server
    __u64 packets_to_client;  // Packets from server to client
    __u64 bytes_to_server;    // Bytes from client to server
    __u64 bytes_to_client;    // Bytes from server to client
    __u8  state;              // Session state
    __u8  tcp_state;          // TCP state machine
    __u8  policy_action;      // Matched policy action
    __u8  flags;              // Session flags
    __u32 pad;                // Padding
};

// Policy key for exact matching
// Added direction field for egress support while keeping full 5-tuple
struct policy_key {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;       // Keep for full 5-tuple matching
    __u16 dst_port;
    __u8  protocol;
    __u8  direction;      // enum policy_direction
    __u16 pad;            // Padding for alignment (16 bytes total)
} __attribute__((packed));

// Policy value
struct policy_value {
    __u8  action;             // Policy action
    __u8  log_enabled;        // Enable logging
    __u16 priority;           // Policy priority
    __u32 rule_id;            // Rule ID for tracking
    __u64 hit_count;          // Number of times this policy was matched
};

// Wildcard policy for matching with wildcards (0 = match any)
// Used in array map for linear searching
struct wildcard_policy {
    __u32 src_ip;
    __u32 src_ip_mask;        // 0xFFFFFFFF = exact, 0x00000000 = any
    __u32 dst_ip;
    __u32 dst_ip_mask;        // 0xFFFFFFFF = exact, 0x00000000 = any
    __u16 src_port;           // 0 = any port
    __u16 dst_port;           // 0 = any port
    __u8  protocol;           // 0 = any protocol
    __u8  action;             // Policy action
    __u8  log_enabled;        // Enable logging
    __u8  direction;          // enum policy_direction
    __u16 priority;           // Policy priority (higher = more important)
    __u16 pad;                // Padding
    __u32 rule_id;            // Rule ID (0 = empty slot)
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
    // Direction-specific statistics (for egress support)
    STATS_INGRESS_PACKETS,
    STATS_EGRESS_PACKETS,
    STATS_INGRESS_DENIED,
    STATS_EGRESS_DENIED,
    STATS_MAX,
};

// Flow event types
enum flow_event_type {
    FLOW_EVENT_NEW = 0,      // New connection established
    FLOW_EVENT_UPDATE = 1,   // Connection active/updated
    FLOW_EVENT_CLOSED = 2,   // Connection closed
    FLOW_EVENT_TIMEOUT = 3,  // Connection timeout
};

// Flow direction
enum flow_direction {
    FLOW_DIRECTION_INGRESS = 0,
    FLOW_DIRECTION_EGRESS = 1,
    FLOW_DIRECTION_UNKNOWN = 2,
};

// Flow state
enum flow_state {
    FLOW_STATE_ACTIVE = 0,
    FLOW_STATE_CLOSED = 1,
    FLOW_STATE_TIMEOUT = 2,
};

// Flow event for reporting to control plane (48 bytes, optimized for Ring Buffer)
struct flow_event {
    // 5-tuple identification (12 bytes)
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;

    // Packet metadata (4 bytes)
    __u8  protocol;
    __u8  event_type;     // enum flow_event_type
    __u8  direction;      // enum flow_direction
    __u8  padding;        // Padding for alignment

    // Traffic statistics (24 bytes)
    __u64 packet_count;   // Total packets in this flow
    __u64 byte_count;     // Total bytes in this flow
    __u64 timestamp_ns;   // Event timestamp in nanoseconds

    // Policy context (8 bytes)
    __u32 policy_id;      // Matched policy/rule ID
    __u8  policy_action;  // enum policy_action
    __u8  state;          // enum flow_state
    __u16 reserved;       // Reserved for future use
} __attribute__((packed));

#endif /* __COMMON_TYPES_H__ */

