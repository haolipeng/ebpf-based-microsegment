// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Common types shared between eBPF and user-space programs */

#ifndef __COMMON_TYPES_H__
#define __COMMON_TYPES_H__

#define MAX_ENTRIES_SESSION 100000
#define MAX_ENTRIES_POLICY 10000
#define MAX_ENTRIES_WILDCARD_POLICY 1000

// Session timeout configuration (in nanoseconds)
// These are default values, can be overridden via timeout_config_map
#define DEFAULT_TCP_TIMEOUT_NS      (300ULL * 1000000000ULL)  // 5 minutes
#define DEFAULT_UDP_TIMEOUT_NS      (60ULL * 1000000000ULL)   // 1 minute
#define DEFAULT_ICMP_TIMEOUT_NS     (30ULL * 1000000000ULL)   // 30 seconds
#define DEFAULT_OTHER_TIMEOUT_NS    (60ULL * 1000000000ULL)   // 1 minute

// Session timeout scan configuration
#define TIMEOUT_SCAN_BATCH_SIZE     100  // Max sessions to check per scan cycle

// 5-tuple flow key for session tracking (IPv4/IPv6 + VLAN support)
// All IP addresses are stored in 128-bit format:
// - IPv6: native format
// - IPv4: IPv4-mapped IPv6 format (::ffff:a.b.c.d)
struct flow_key {
    __u32 src_ip[4];   // Source IP address (128 bits = 4 x 32 bits)
    __u32 dst_ip[4];   // Destination IP address (128 bits = 4 x 32 bits)
    __u16 src_port;
    __u16 dst_port;
    __u8  protocol;
    __u8  ip_version;  // 4 = IPv4, 6 = IPv6
    __u16 vlan_id;     // VLAN ID (0 = no VLAN, 1-4094 = VLAN ID)
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

// Connection flags for enhanced tracking
#define CONN_FLAG_NAT           0x01  // Connection uses NAT
#define CONN_FLAG_ESTABLISHED   0x02  // TCP connection established
#define CONN_FLAG_RETRANS       0x04  // Retransmission detected
#define CONN_FLAG_WINDOW_FULL   0x08  // TCP window full detected
#define CONN_FLAG_VLAN          0x10  // VLAN tagged traffic

// Session value stored in LRU_HASH map (enhanced connection tracking)
struct session_value {
    __u64 created_ts;         // Session creation timestamp (nanoseconds)
    __u64 last_seen_ts;       // Last packet timestamp
    __u64 packets_to_server;  // Packets from client to server
    __u64 packets_to_client;  // Packets from server to client
    __u64 bytes_to_server;    // Bytes from client to server
    __u64 bytes_to_client;    // Bytes from server to client

    // Enhanced TCP tracking
    __u32 tcp_seq_client;     // Last TCP sequence number from client
    __u32 tcp_seq_server;     // Last TCP sequence number from server
    __u32 tcp_ack_client;     // Last TCP acknowledgment from client
    __u32 tcp_ack_server;     // Last TCP acknowledgment from server
    __u16 tcp_window_size;    // TCP window size (last seen)
    __u8  tcp_retrans_count;  // TCP retransmission count

    __u8  state;              // Session state
    __u8  tcp_state;          // TCP state machine
    __u8  policy_action;      // Matched policy action
    __u8  flags;              // Connection flags (CONN_FLAG_*)
    __u8  pad;                // Padding for alignment
};

// Policy key for exact matching (IPv4/IPv6 + VLAN support)
// Added direction field for egress support while keeping full 5-tuple
// All IP addresses are stored in 128-bit format (same as flow_key)
struct policy_key {
    __u32 src_ip[4];      // Source IP address (128 bits)
    __u32 dst_ip[4];      // Destination IP address (128 bits)
    __u16 src_port;       // Keep for full 5-tuple matching
    __u16 dst_port;
    __u8  protocol;
    __u8  direction;      // enum policy_direction
    __u8  ip_version;     // 4 = IPv4, 6 = IPv6
    __u8  pad;            // Padding for alignment
    __u16 vlan_id;        // VLAN ID (0 = match any VLAN)
    __u16 pad2;           // Padding
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
// Used in array map for linear searching (IPv4/IPv6 + VLAN support)
// All IP addresses are stored in 128-bit format
struct wildcard_policy {
    __u32 src_ip[4];          // Source IP address (128 bits)
    __u32 src_ip_mask[4];     // Mask for source IP (all 1s = exact, all 0s = any)
    __u32 dst_ip[4];          // Destination IP address (128 bits)
    __u32 dst_ip_mask[4];     // Mask for destination IP (all 1s = exact, all 0s = any)
    __u16 src_port;           // 0 = any port
    __u16 dst_port;           // 0 = any port
    __u8  protocol;           // 0 = any protocol
    __u8  action;             // Policy action
    __u8  log_enabled;        // Enable logging
    __u8  direction;          // enum policy_direction
    __u8  ip_version;         // 4 = IPv4, 6 = IPv6
    __u8  pad[3];             // Padding for alignment
    __u16 priority;           // Policy priority (higher = more important)
    __u16 vlan_id;            // VLAN ID (0 = match any VLAN)
    __u32 rule_id;            // Rule ID (0 = empty slot)
} __attribute__((packed));

// Protocol segment descriptor for indexed wildcard policy lookup
// Used to track which range of wildcard_policy_map belongs to each protocol
struct protocol_segment {
    __u32 start_idx;           // Starting index in wildcard_policy_map
    __u32 policy_count;        // Number of policies in this segment
    __u32 reserved[2];         // Reserved for future use
} __attribute__((packed));

// Statistics counters (enhanced monitoring)
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
    // Protocol-specific statistics
    STATS_IPV4_PACKETS,
    STATS_IPV6_PACKETS,
    STATS_TCP_PACKETS,
    STATS_UDP_PACKETS,
    STATS_ICMP_PACKETS,
    // VLAN statistics
    STATS_VLAN_PACKETS,
    STATS_QINQ_PACKETS,
    // TCP-specific statistics
    STATS_TCP_SYN,
    STATS_TCP_FIN,
    STATS_TCP_RST,
    STATS_TCP_RETRANS,
    // Error statistics
    STATS_PARSE_ERRORS,
    STATS_RINGBUF_FULL,
    STATS_MAX,
};

// Timeout configuration keys for timeout_config_map
enum timeout_config_key {
    TIMEOUT_CONFIG_TCP = 0,    // TCP session timeout (nanoseconds)
    TIMEOUT_CONFIG_UDP = 1,    // UDP session timeout (nanoseconds)
    TIMEOUT_CONFIG_ICMP = 2,   // ICMP session timeout (nanoseconds)
    TIMEOUT_CONFIG_OTHER = 3,  // Other protocol timeout (nanoseconds)
    TIMEOUT_CONFIG_MAX,
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

// Flow event for reporting to control plane (IPv4/IPv6 + VLAN + enhanced tracking)
// All IP addresses are stored in 128-bit format
struct flow_event {
    // 5-tuple identification (36 bytes)
    __u32 src_ip[4];      // Source IP address (128 bits)
    __u32 dst_ip[4];      // Destination IP address (128 bits)
    __u16 src_port;
    __u16 dst_port;

    // Packet metadata (8 bytes)
    __u8  protocol;
    __u8  event_type;     // enum flow_event_type
    __u8  direction;      // enum flow_direction
    __u8  ip_version;     // 4 = IPv4, 6 = IPv6
    __u16 vlan_id;        // VLAN ID (0 = no VLAN)
    __u8  tcp_flags;      // TCP flags (SYN, FIN, RST, etc.)
    __u8  flags;          // Connection flags (CONN_FLAG_*)

    // Traffic statistics (24 bytes)
    __u64 packet_count;   // Total packets in this flow
    __u64 byte_count;     // Total bytes in this flow
    __u64 timestamp_ns;   // Event timestamp in nanoseconds

    // Enhanced TCP tracking (12 bytes)
    __u32 tcp_seq;        // TCP sequence number
    __u32 tcp_ack;        // TCP acknowledgment number
    __u16 tcp_window;     // TCP window size
    __u8  tcp_retrans;    // Retransmission count
    __u8  tcp_state;      // TCP state

    // Policy context (4 bytes)
    __u32 policy_id;      // Matched policy/rule ID
    __u8  policy_action;  // enum policy_action
    __u8  state;          // enum flow_state
    __u16 reserved;       // Reserved for future use
} __attribute__((packed));

#endif /* __COMMON_TYPES_H__ */

