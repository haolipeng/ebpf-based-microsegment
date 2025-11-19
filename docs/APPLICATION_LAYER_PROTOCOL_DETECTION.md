# Application Layer Protocol Detection - Implementation Plan

## Executive Summary

This document outlines the design and implementation plan for adding **Application Layer Protocol Detection** capability to the eBPF-based microsegmentation system. This feature enables:

- **Protocol-aware policies**: Fine-grained rules based on application protocols (HTTP/HTTPS/DNS/SSH/MySQL/Redis, etc.)
- **Protocol abuse detection**: Identify protocol tunneling and misuse (SSH tunnels, DNS tunnels)
- **Enhanced visibility**: Protocol information in topology views and flow analytics
- **Security enforcement**: Block cleartext protocols, enforce encrypted communication

**Implementation Status**: Design Phase
**Estimated Effort**: 11-16 days
**Priority**: Medium-High (Business Value 🟡)

---

## Table of Contents

1. [Business Value](#business-value)
2. [Technical Architecture](#technical-architecture)
3. [Data Structure Design](#data-structure-design)
4. [Protocol Detection Algorithms](#protocol-detection-algorithms)
5. [Performance Optimization](#performance-optimization)
6. [User-Space API](#user-space-api)
7. [Implementation Roadmap](#implementation-roadmap)
8. [Technical Challenges](#technical-challenges)
9. [Testing Strategy](#testing-strategy)
10. [References](#references)

---

## 1. Business Value

### 1.1 Key Use Cases

**UC1: Protocol-Based Segmentation**
```yaml
# Example policy: Allow HTTPS only, deny HTTP
- name: "web-secure-only"
  from:
    workload: "external"
  to:
    workload: "web-tier"
  protocol: HTTPS
  action: ALLOW

- name: "web-deny-cleartext"
  from:
    workload: "external"
  to:
    workload: "web-tier"
  protocol: HTTP
  action: DENY
```

**UC2: Protocol Abuse Detection**
- Detect SSH on non-standard ports (e.g., port 8080)
- Identify DNS tunneling attempts
- Alert on unexpected protocols on known ports

**UC3: Compliance and Security**
- Enforce encryption for sensitive data (PCI DSS, HIPAA)
- Block cleartext database connections (MySQL, PostgreSQL)
- Audit protocol usage across the infrastructure

**UC4: Enhanced Observability**
- Topology maps showing protocol distribution
- Protocol-level traffic statistics
- Application dependency mapping

### 1.2 Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Protocol Detection Accuracy | >95% | Test suite validation |
| Detection Latency | <100μs | eBPF program execution time |
| False Positive Rate | <2% | Production monitoring |
| Supported Protocols | 10+ | HTTP, HTTPS, DNS, SSH, MySQL, Redis, Kafka, gRPC, PostgreSQL, MongoDB |
| Performance Impact | <5% | Packet processing throughput |

---

## 2. Technical Architecture

### 2.1 System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     eBPF Data Plane                         │
├─────────────────────────────────────────────────────────────┤
│  [Packet Ingress] → TC/XDP Hook                             │
│         ↓                                                    │
│  [L2/L3/L4 Parsing] (Existing)                              │
│         ↓                                                    │
│  [Session Lookup] session_map                               │
│         ↓                                                    │
│  ┌─────────────────────────────────────────┐                │
│  │   Application Layer Protocol Detection   │ ◄── NEW       │
│  ├─────────────────────────────────────────┤                │
│  │  Stage 1: Port-based Heuristics         │                │
│  │  Stage 2: Payload Signature Matching    │                │
│  │  Stage 3: Protocol State Machine        │                │
│  └─────────────────────────────────────────┘                │
│         ↓                                                    │
│  [Policy Matching] (Enhanced with Protocol Filter) ◄── NEW  │
│         ↓                                                    │
│  [Action Enforcement] ALLOW / DENY                           │
│         ↓                                                    │
│  [Statistics Update]                                         │
└─────────────────────────────────────────────────────────────┘
         ↓ (Ring Buffer Events)
┌─────────────────────────────────────────────────────────────┐
│                   User-Space Agent                          │
├─────────────────────────────────────────────────────────────┤
│  • Protocol Statistics Aggregation                          │
│  • Protocol Configuration Management                        │
│  • Anomaly Detection (Protocol Mismatch)                    │
│  • Event Reporting (Flow Events with Protocol Info)         │
│  • REST API (Protocol Query, Stats)                         │
└─────────────────────────────────────────────────────────────┘
         ↓
┌─────────────────────────────────────────────────────────────┐
│              Control Plane & Web UI                         │
├─────────────────────────────────────────────────────────────┤
│  • Protocol-based Policy Editor                             │
│  • Protocol Distribution Dashboard                          │
│  • Protocol Anomaly Alerts                                  │
│  • Application Dependency Map                               │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Data Flow

```
┌──────────────┐
│ TCP Packet   │
│ dst_port=443 │
└──────┬───────┘
       │
       ▼
┌────────────────────────────────────┐
│ Stage 1: Port Heuristics           │
│ → guess_protocol_by_port(443)      │
│ → Result: APP_PROTO_HTTPS (60%)    │
└────────┬───────────────────────────┘
         │ (Low confidence, continue)
         ▼
┌────────────────────────────────────┐
│ Stage 2: Payload Inspection        │
│ → Extract first 16 bytes            │
│ → Match TLS signature:              │
│   [0x16 0x03 0x03 ...] ✓            │
│ → Result: APP_PROTO_HTTPS (98%)    │
└────────┬───────────────────────────┘
         │ (High confidence, done)
         ▼
┌────────────────────────────────────┐
│ Update Session:                    │
│ session.app_protocol = HTTPS       │
│ session.proto_confidence = 98      │
│ session.proto_flags = ENCRYPTED    │
└────────┬───────────────────────────┘
         │
         ▼
┌────────────────────────────────────┐
│ Policy Matching (Enhanced)         │
│ → Match L3/L4 rules (5-tuple)       │
│ → Match L7 rules (protocol filter)  │
│   Policy: "Allow HTTPS only"        │
│   app_protocol == HTTPS ? ✓         │
└────────┬───────────────────────────┘
         │
         ▼
┌────────────────────────────────────┐
│ Action: ALLOW (TC_ACT_OK)          │
└────────────────────────────────────┘
```

### 2.3 eBPF Program Structure

**New/Modified Files**:

```
src/bpf/headers/
├── app_protocol_types.h         # Protocol enums and constants (NEW)
├── app_protocol_detection.h     # Detection functions (NEW)
├── app_protocol_http.h          # HTTP/HTTPS detector (NEW)
├── app_protocol_dns.h           # DNS detector (NEW)
├── app_protocol_ssh.h           # SSH detector (NEW)
├── app_protocol_mysql.h         # MySQL detector (NEW)
├── app_protocol_redis.h         # Redis detector (NEW)
├── common_types.h               # Add protocol fields to session_value (MODIFIED)
└── policy_match.h               # Add protocol filter to policy matching (MODIFIED)

src/bpf/
├── tc_microsegment.bpf.c        # Integrate protocol detection (MODIFIED)
└── xdp_microsegment.bpf.c       # Integrate protocol detection (MODIFIED)
```

**User-Space Files**:

```
src/agent/pkg/
├── protocol/                    # New package
│   ├── detector.go              # Protocol detection config/stats
│   ├── types.go                 # Go types for protocols
│   └── anomaly.go               # Protocol anomaly detection
├── dataplane/
│   └── manager.go               # Add protocol config APIs (MODIFIED)
├── flow/
│   └── storage.go               # Add protocol field to flow events (MODIFIED)
└── policy/
    └── compiler.go              # Add protocol filter compilation (MODIFIED)
```

---

## 3. Data Structure Design

### 3.1 Protocol Enumerations

**File**: `src/bpf/headers/app_protocol_types.h` (NEW)

```c
#ifndef __APP_PROTOCOL_TYPES_H__
#define __APP_PROTOCOL_TYPES_H__

// Application layer protocol identifiers
enum app_protocol {
    APP_PROTO_UNKNOWN = 0,

    // Web protocols
    APP_PROTO_HTTP = 1,
    APP_PROTO_HTTPS = 2,

    // Infrastructure
    APP_PROTO_DNS = 3,
    APP_PROTO_SSH = 4,

    // Databases
    APP_PROTO_MYSQL = 5,
    APP_PROTO_POSTGRESQL = 6,
    APP_PROTO_REDIS = 7,
    APP_PROTO_MONGODB = 8,

    // Messaging
    APP_PROTO_KAFKA = 9,
    APP_PROTO_RABBITMQ = 10,

    // RPC
    APP_PROTO_GRPC = 11,

    // Other
    APP_PROTO_FTP = 12,
    APP_PROTO_SMTP = 13,

    // Sentinel
    APP_PROTO_MAX = 100,
};

// Protocol feature flags
#define PROTO_FLAG_ENCRYPTED     0x0001  // TLS/SSL encrypted
#define PROTO_FLAG_CLEARTEXT     0x0002  // Plaintext protocol
#define PROTO_FLAG_BINARY        0x0004  // Binary protocol
#define PROTO_FLAG_TEXT          0x0008  // Text-based protocol
#define PROTO_FLAG_REQUEST       0x0010  // Request direction
#define PROTO_FLAG_RESPONSE      0x0020  // Response direction
#define PROTO_FLAG_TUNNEL        0x0040  // Potential tunneling detected

// Protocol detection configuration
struct proto_detect_config {
    __u8  enabled;               // Global enable/disable
    __u8  sampling_interval;     // 0 = every packet, N = every N packets
    __u16 max_payload_bytes;     // Max bytes to inspect (default: 128)
    __u8  confidence_threshold;  // Min confidence to report (0-100, default: 70)
    __u8  reserved[3];
};

#endif // __APP_PROTOCOL_TYPES_H__
```

### 3.2 Session Value Extension

**File**: `src/bpf/headers/common_types.h` (MODIFIED)

```c
// Add to existing session_value struct
struct session_value {
    // ===== Existing fields =====
    __u64 created_ts;
    __u64 last_seen_ts;
    __u64 packets_to_server;
    __u64 packets_to_client;
    __u64 bytes_to_server;
    __u64 bytes_to_client;

    __u32 tcp_seq_client;
    __u32 tcp_seq_server;
    __u32 tcp_ack_client;
    __u32 tcp_ack_server;
    __u16 tcp_window_size;
    __u8  tcp_retrans_count;

    __u8  state;
    __u8  tcp_state;
    __u8  policy_action;
    __u8  flags;

    // ===== NEW: Protocol detection fields =====
    __u8  app_protocol;          // enum app_protocol
    __u8  proto_confidence;      // 0-100
    __u16 proto_flags;           // PROTO_FLAG_* bitmask
    __u32 proto_first_seen_ts;   // First detection timestamp (seconds)
    __u32 proto_payload_bytes;   // Total payload bytes inspected

    __u8  pad[2];                // Padding for alignment
} __attribute__((packed));
```

**Size Analysis**:
- Existing: ~96 bytes
- New fields: 12 bytes
- Total: ~108 bytes (within reasonable limits for LRU map)

### 3.3 Policy Extension

**File**: `src/bpf/headers/common_types.h` (MODIFIED)

```c
// Add to existing wildcard_policy struct
struct wildcard_policy {
    // ===== Existing fields =====
    __u32 src_ip_mask[4];
    __u32 dst_ip_mask[4];
    __u32 src_ip[4];
    __u32 dst_ip[4];
    __u16 src_port_start;
    __u16 src_port_end;
    __u16 dst_port_start;
    __u16 dst_port_end;
    __u8  protocol;              // L4 protocol (TCP/UDP/ICMP)
    __u8  action;
    __u8  direction;
    __u16 vlan_id;

    // ===== NEW: Application protocol filter =====
    __u8  app_protocol;          // 0 = any, else specific protocol
    __u8  app_proto_match_mode;  // 0 = exact, 1 = exclude
    __u16 app_proto_flags_mask;  // Required flags (PROTO_FLAG_*)
    __u16 app_proto_flags_value; // Expected flag values

    __u32 rule_id;
    __u8  reserved[8];
} __attribute__((packed));
```

### 3.4 eBPF Maps

**File**: `src/bpf/tc_microsegment.bpf.c` (MODIFIED)

```c
// NEW: Protocol detection configuration
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct proto_detect_config);
    __uint(pinning, LIBBPF_PIN_BY_NAME);
} proto_detect_config_map SEC(".maps");

// NEW: Protocol statistics (per-protocol counters)
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 128);  // APP_PROTO_MAX + extra
    __type(key, __u32);        // protocol ID
    __type(value, __u64);      // packet count
    __uint(pinning, LIBBPF_PIN_BY_NAME);
} proto_stats_map SEC(".maps");
```

**Optional: Protocol Cache** (for optimization)

```c
// Cache for expensive protocol detections
struct app_proto_cache_entry {
    __u8  protocol;
    __u8  confidence;
    __u16 sample_count;
    __u32 last_update_ts;
};

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 10000);
    __type(key, struct flow_key);
    __type(value, struct app_proto_cache_entry);
} app_proto_cache SEC(".maps");
```

---

## 4. Protocol Detection Algorithms

### 4.1 Two-Stage Detection Strategy

**Stage 1: Port-Based Heuristics** (Fast Path)
- Lookup well-known port numbers
- Low confidence score (50-70%)
- Fallback for payload inspection

**Stage 2: Payload Signature Matching** (Accurate)
- Inspect first N bytes of payload
- Pattern matching for protocol signatures
- High confidence score (85-99%)

### 4.2 Core Detection Function

**File**: `src/bpf/headers/app_protocol_detection.h` (NEW)

```c
#ifndef __APP_PROTOCOL_DETECTION_H__
#define __APP_PROTOCOL_DETECTION_H__

#include "app_protocol_types.h"
#include "app_protocol_http.h"
#include "app_protocol_dns.h"
#include "app_protocol_ssh.h"
#include "app_protocol_mysql.h"
#include "app_protocol_redis.h"

// Port-based protocol guessing
static __always_inline __u8 guess_protocol_by_port(__u16 port, __u8 l4_proto)
{
    // Convert to host byte order for comparison
    __u16 port_h = bpf_ntohs(port);

    if (l4_proto == IPPROTO_TCP) {
        switch (port_h) {
        case 80:
        case 8080:
        case 8000:
            return APP_PROTO_HTTP;
        case 443:
        case 8443:
            return APP_PROTO_HTTPS;
        case 22:
            return APP_PROTO_SSH;
        case 3306:
            return APP_PROTO_MYSQL;
        case 5432:
            return APP_PROTO_POSTGRESQL;
        case 6379:
            return APP_PROTO_REDIS;
        case 27017:
            return APP_PROTO_MONGODB;
        case 9092:
            return APP_PROTO_KAFKA;
        case 21:
            return APP_PROTO_FTP;
        case 25:
        case 587:
            return APP_PROTO_SMTP;
        default:
            return APP_PROTO_UNKNOWN;
        }
    } else if (l4_proto == IPPROTO_UDP) {
        switch (port_h) {
        case 53:
            return APP_PROTO_DNS;
        default:
            return APP_PROTO_UNKNOWN;
        }
    }

    return APP_PROTO_UNKNOWN;
}

// Main protocol detection entry point
static __always_inline int detect_app_protocol(
    void *payload_start,
    void *data_end,
    struct flow_key *key,
    struct session_value *session,
    struct proto_detect_config *config)
{
    if (!config || !config->enabled)
        return 0;

    // Skip if already detected with high confidence
    if (session->proto_confidence >= 90)
        return 0;

    // Check sampling interval
    if (config->sampling_interval > 0) {
        __u64 pkt_count = session->packets_to_server + session->packets_to_client;
        if ((pkt_count % config->sampling_interval) != 0)
            return 0;
    }

    // Stage 1: Port heuristics (if not already done)
    if (session->app_protocol == APP_PROTO_UNKNOWN) {
        __u8 guessed = guess_protocol_by_port(key->dst_port, key->protocol);
        if (guessed != APP_PROTO_UNKNOWN) {
            session->app_protocol = guessed;
            session->proto_confidence = 60;  // Low confidence
            session->proto_first_seen_ts = bpf_ktime_get_ns() / 1000000000;
        }
    }

    // Stage 2: Payload inspection (if payload exists)
    if (!payload_start || payload_start >= data_end)
        return 0;

    __u32 payload_len = data_end - payload_start;
    if (payload_len == 0)
        return 0;

    // Limit inspection length
    __u32 inspect_len = payload_len;
    if (config->max_payload_bytes > 0 && inspect_len > config->max_payload_bytes)
        inspect_len = config->max_payload_bytes;

    // Update payload inspection counter
    session->proto_payload_bytes += inspect_len;

    __u8 detected_proto = APP_PROTO_UNKNOWN;
    __u8 confidence = 0;
    __u16 flags = 0;

    // Protocol-specific detection
    if (key->protocol == IPPROTO_TCP) {
        // Try HTTP detection
        if (detect_http(payload_start, data_end, inspect_len, &confidence, &flags)) {
            detected_proto = APP_PROTO_HTTP;
        }
        // Try HTTPS/TLS detection
        else if (detect_tls(payload_start, data_end, inspect_len, &confidence, &flags)) {
            detected_proto = APP_PROTO_HTTPS;
        }
        // Try SSH detection
        else if (detect_ssh(payload_start, data_end, inspect_len, &confidence, &flags)) {
            detected_proto = APP_PROTO_SSH;
        }
        // Try MySQL detection
        else if (detect_mysql(payload_start, data_end, inspect_len, &confidence, &flags)) {
            detected_proto = APP_PROTO_MYSQL;
        }
        // Try Redis detection
        else if (detect_redis(payload_start, data_end, inspect_len, &confidence, &flags)) {
            detected_proto = APP_PROTO_REDIS;
        }
        // More protocols...
    }
    else if (key->protocol == IPPROTO_UDP) {
        // Try DNS detection
        if (detect_dns(payload_start, data_end, inspect_len, &confidence, &flags)) {
            detected_proto = APP_PROTO_DNS;
        }
    }

    // Update session if detection succeeded
    if (detected_proto != APP_PROTO_UNKNOWN && confidence > session->proto_confidence) {
        session->app_protocol = detected_proto;
        session->proto_confidence = confidence;
        session->proto_flags = flags;

        if (session->proto_first_seen_ts == 0)
            session->proto_first_seen_ts = bpf_ktime_get_ns() / 1000000000;
    }

    return 0;
}

// Helper: Get payload start pointer
static __always_inline void *get_tcp_payload_start(
    struct tcphdr *tcph,
    void *data_end)
{
    if ((void *)(tcph + 1) > data_end)
        return NULL;

    // TCP header length in 32-bit words
    __u8 tcp_hdr_len = tcph->doff * 4;

    void *payload = (void *)tcph + tcp_hdr_len;

    if (payload >= data_end)
        return NULL;

    return payload;
}

static __always_inline void *get_udp_payload_start(
    struct udphdr *udph,
    void *data_end)
{
    if ((void *)(udph + 1) > data_end)
        return NULL;

    return (void *)(udph + 1);
}

#endif // __APP_PROTOCOL_DETECTION_H__
```

### 4.3 Protocol-Specific Detectors

#### 4.3.1 HTTP Detector

**File**: `src/bpf/headers/app_protocol_http.h` (NEW)

```c
#ifndef __APP_PROTOCOL_HTTP_H__
#define __APP_PROTOCOL_HTTP_H__

// HTTP method detection
static __always_inline bool detect_http(
    void *payload_start,
    void *data_end,
    __u32 inspect_len,
    __u8 *confidence,
    __u16 *flags)
{
    // Need at least 4 bytes
    if (payload_start + 4 > data_end)
        return false;

    char *data = (char *)payload_start;

    // Check HTTP methods (Request)
    bool is_http_request = false;

    // GET
    if (data[0] == 'G' && data[1] == 'E' && data[2] == 'T' && data[3] == ' ') {
        is_http_request = true;
        *confidence = 95;
    }
    // POST (need 5 bytes)
    else if (payload_start + 5 <= data_end &&
             data[0] == 'P' && data[1] == 'O' && data[2] == 'S' &&
             data[3] == 'T' && data[4] == ' ') {
        is_http_request = true;
        *confidence = 95;
    }
    // PUT (need 4 bytes)
    else if (data[0] == 'P' && data[1] == 'U' && data[2] == 'T' && data[3] == ' ') {
        is_http_request = true;
        *confidence = 95;
    }
    // HEAD (need 5 bytes)
    else if (payload_start + 5 <= data_end &&
             data[0] == 'H' && data[1] == 'E' && data[2] == 'A' &&
             data[3] == 'D' && data[4] == ' ') {
        is_http_request = true;
        *confidence = 95;
    }
    // DELETE (need 7 bytes)
    else if (payload_start + 7 <= data_end &&
             data[0] == 'D' && data[1] == 'E' && data[2] == 'L' &&
             data[3] == 'E' && data[4] == 'T' && data[5] == 'E' && data[6] == ' ') {
        is_http_request = true;
        *confidence = 95;
    }

    // Check HTTP response (need 8 bytes)
    bool is_http_response = false;
    if (payload_start + 8 <= data_end &&
        data[0] == 'H' && data[1] == 'T' && data[2] == 'T' && data[3] == 'P' &&
        data[4] == '/' && data[5] == '1' && data[6] == '.') {
        is_http_response = true;
        *confidence = 90;
    }

    if (is_http_request || is_http_response) {
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT;
        if (is_http_request)
            *flags |= PROTO_FLAG_REQUEST;
        else
            *flags |= PROTO_FLAG_RESPONSE;
        return true;
    }

    return false;
}

#endif // __APP_PROTOCOL_HTTP_H__
```

#### 4.3.2 TLS/HTTPS Detector

**File**: `src/bpf/headers/app_protocol_http.h` (add to same file)

```c
// TLS/SSL detection
static __always_inline bool detect_tls(
    void *payload_start,
    void *data_end,
    __u32 inspect_len,
    __u8 *confidence,
    __u16 *flags)
{
    // Need at least 5 bytes for TLS record header
    if (payload_start + 5 > data_end)
        return false;

    __u8 *data = (__u8 *)payload_start;

    // TLS Record Header:
    // - ContentType (1 byte): 0x16=Handshake, 0x17=Application Data, 0x14=ChangeCipherSpec
    // - Version (2 bytes): 0x0301=TLS1.0, 0x0302=TLS1.1, 0x0303=TLS1.2, 0x0304=TLS1.3
    // - Length (2 bytes)

    __u8 content_type = data[0];
    __u8 version_major = data[1];
    __u8 version_minor = data[2];

    // Check content type
    bool valid_content_type = (content_type == 0x14 ||  // ChangeCipherSpec
                               content_type == 0x15 ||  // Alert
                               content_type == 0x16 ||  // Handshake
                               content_type == 0x17);   // Application Data

    // Check version (TLS 1.0 - 1.3)
    bool valid_version = (version_major == 0x03 && version_minor >= 0x00 && version_minor <= 0x04);

    if (valid_content_type && valid_version) {
        *confidence = 98;
        *flags = PROTO_FLAG_ENCRYPTED | PROTO_FLAG_BINARY;

        // Distinguish request/response by handshake type
        if (content_type == 0x16 && payload_start + 6 <= data_end) {
            __u8 handshake_type = data[5];
            if (handshake_type == 0x01)  // ClientHello
                *flags |= PROTO_FLAG_REQUEST;
            else if (handshake_type == 0x02)  // ServerHello
                *flags |= PROTO_FLAG_RESPONSE;
        }

        return true;
    }

    return false;
}
```

#### 4.3.3 DNS Detector

**File**: `src/bpf/headers/app_protocol_dns.h` (NEW)

```c
#ifndef __APP_PROTOCOL_DNS_H__
#define __APP_PROTOCOL_DNS_H__

// DNS protocol detection
static __always_inline bool detect_dns(
    void *payload_start,
    void *data_end,
    __u32 inspect_len,
    __u8 *confidence,
    __u16 *flags)
{
    // Need at least 12 bytes for DNS header
    if (payload_start + 12 > data_end)
        return false;

    __u8 *data = (__u8 *)payload_start;

    // DNS Header (12 bytes):
    // - Transaction ID (2 bytes)
    // - Flags (2 bytes)
    // - Questions (2 bytes)
    // - Answer RRs (2 bytes)
    // - Authority RRs (2 bytes)
    // - Additional RRs (2 bytes)

    __u16 flags_field = (data[2] << 8) | data[3];
    __u16 questions = (data[4] << 8) | data[5];
    __u16 answers = (data[6] << 8) | data[7];

    // Extract key flag bits
    bool qr = (flags_field & 0x8000) != 0;     // 0=query, 1=response
    __u8 opcode = (flags_field >> 11) & 0x0F;  // Should be 0 for standard query

    // Validate DNS packet
    bool is_valid_dns = false;

    // Query validation
    if (!qr && questions > 0 && questions < 10 && opcode == 0) {
        is_valid_dns = true;
        *confidence = 85;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_BINARY | PROTO_FLAG_REQUEST;
    }
    // Response validation
    else if (qr && questions > 0 && questions < 10 && opcode == 0) {
        is_valid_dns = true;
        *confidence = 85;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_BINARY | PROTO_FLAG_RESPONSE;
    }

    // Check for potential DNS tunneling (unusually large questions/answers)
    if (is_valid_dns && (questions > 5 || answers > 20)) {
        *flags |= PROTO_FLAG_TUNNEL;
        *confidence = 70;  // Lower confidence, potential anomaly
    }

    return is_valid_dns;
}

#endif // __APP_PROTOCOL_DNS_H__
```

#### 4.3.4 SSH Detector

**File**: `src/bpf/headers/app_protocol_ssh.h` (NEW)

```c
#ifndef __APP_PROTOCOL_SSH_H__
#define __APP_PROTOCOL_SSH_H__

// SSH protocol detection
static __always_inline bool detect_ssh(
    void *payload_start,
    void *data_end,
    __u32 inspect_len,
    __u8 *confidence,
    __u16 *flags)
{
    // SSH version string: "SSH-2.0-" (8 bytes minimum)
    if (payload_start + 8 > data_end)
        return false;

    char *data = (char *)payload_start;

    // Check for SSH banner
    if (data[0] == 'S' && data[1] == 'S' && data[2] == 'H' && data[3] == '-' &&
        data[4] == '2' && data[5] == '.' && data[6] == '0' && data[7] == '-') {
        *confidence = 99;
        *flags = PROTO_FLAG_ENCRYPTED | PROTO_FLAG_TEXT;

        // SSH banner is usually sent by server first
        *flags |= PROTO_FLAG_RESPONSE;

        return true;
    }

    // Also check for SSH-1.x (legacy, less common)
    if (data[0] == 'S' && data[1] == 'S' && data[2] == 'H' && data[3] == '-' &&
        data[4] == '1' && data[5] == '.') {
        *confidence = 95;
        *flags = PROTO_FLAG_ENCRYPTED | PROTO_FLAG_TEXT | PROTO_FLAG_RESPONSE;
        return true;
    }

    return false;
}

#endif // __APP_PROTOCOL_SSH_H__
```

#### 4.3.5 MySQL Detector

**File**: `src/bpf/headers/app_protocol_mysql.h` (NEW)

```c
#ifndef __APP_PROTOCOL_MYSQL_H__
#define __APP_PROTOCOL_MYSQL_H__

// MySQL protocol detection
static __always_inline bool detect_mysql(
    void *payload_start,
    void *data_end,
    __u32 inspect_len,
    __u8 *confidence,
    __u16 *flags)
{
    // MySQL Initial Handshake Packet (Server Greeting):
    // - Packet length (3 bytes)
    // - Packet number (1 byte)
    // - Protocol version (1 byte): usually 10
    // - Server version string (null-terminated)

    if (payload_start + 5 > data_end)
        return false;

    __u8 *data = (__u8 *)payload_start;

    // Check protocol version (byte 4)
    __u8 protocol_version = data[4];

    if (protocol_version == 10) {
        // MySQL protocol version 10
        *confidence = 80;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_BINARY | PROTO_FLAG_RESPONSE;
        return true;
    }

    // Also check for MySQL client requests (COM_* commands)
    // Client request: length (3) + sequence (1) + command (1)
    if (payload_start + 5 <= data_end) {
        __u8 command = data[4];

        // Common MySQL commands
        if (command >= 0x00 && command <= 0x1F) {
            *confidence = 60;  // Lower confidence
            *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_BINARY | PROTO_FLAG_REQUEST;
            return true;
        }
    }

    return false;
}

#endif // __APP_PROTOCOL_MYSQL_H__
```

#### 4.3.6 Redis Detector

**File**: `src/bpf/headers/app_protocol_redis.h` (NEW)

```c
#ifndef __APP_PROTOCOL_REDIS_H__
#define __APP_PROTOCOL_REDIS_H__

// Redis protocol detection (RESP - REdis Serialization Protocol)
static __always_inline bool detect_redis(
    void *payload_start,
    void *data_end,
    __u32 inspect_len,
    __u8 *confidence,
    __u16 *flags)
{
    // RESP uses first byte as type indicator:
    // '+' : Simple String
    // '-' : Error
    // ':' : Integer
    // '$' : Bulk String
    // '*' : Array

    if (payload_start + 2 > data_end)
        return false;

    char *data = (char *)payload_start;
    char first_byte = data[0];

    bool is_redis = false;

    switch (first_byte) {
    case '+':  // Simple String (response)
        is_redis = true;
        *confidence = 85;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_RESPONSE;
        break;

    case '-':  // Error (response)
        is_redis = true;
        *confidence = 85;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_RESPONSE;
        break;

    case ':':  // Integer (response)
        is_redis = true;
        *confidence = 80;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_RESPONSE;
        break;

    case '$':  // Bulk String (can be request or response)
        is_redis = true;
        *confidence = 75;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT;
        break;

    case '*':  // Array (usually request: *3\r\n$3\r\nSET\r\n...)
        is_redis = true;
        *confidence = 85;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_REQUEST;
        break;

    default:
        return false;
    }

    // Validate RESP format: should be followed by digits and \r\n
    if (is_redis && payload_start + 3 <= data_end) {
        // Check for digit after type indicator
        if (data[1] < '0' || data[1] > '9') {
            // Special case: "+OK\r\n" or "-ERR ..."
            if (first_byte == '+' || first_byte == '-') {
                // Still valid
            } else {
                return false;
            }
        }
    }

    return is_redis;
}

#endif // __APP_PROTOCOL_REDIS_H__
```

### 4.4 Integration into TC/XDP Programs

**File**: `src/bpf/tc_microsegment.bpf.c` (MODIFIED)

```c
// Add includes
#include "headers/app_protocol_types.h"
#include "headers/app_protocol_detection.h"

// In main TC handler (after session creation)
SEC("tc/microsegment")
int tc_microsegment_handler(struct __sk_buff *skb)
{
    // ... existing L2/L3/L4 parsing ...

    // ... existing session lookup/creation ...

    // NEW: Protocol detection (after session creation)
    if (session && session->state == SESSION_STATE_ACTIVE) {
        // Load protocol detection config
        __u32 config_key = 0;
        struct proto_detect_config *proto_config =
            bpf_map_lookup_elem(&proto_detect_config_map, &config_key);

        // Get payload start
        void *payload = NULL;
        if (key.protocol == IPPROTO_TCP) {
            struct tcphdr *tcph = l4_header;
            payload = get_tcp_payload_start(tcph, data_end);
        } else if (key.protocol == IPPROTO_UDP) {
            struct udphdr *udph = l4_header;
            payload = get_udp_payload_start(udph, data_end);
        }

        // Perform protocol detection
        detect_app_protocol(payload, data_end, &key, session, proto_config);

        // Update protocol statistics
        if (session->app_protocol != APP_PROTO_UNKNOWN) {
            __u32 proto_key = session->app_protocol;
            __u64 *proto_count = bpf_map_lookup_elem(&proto_stats_map, &proto_key);
            if (proto_count) {
                __sync_fetch_and_add(proto_count, 1);
            }
        }
    }

    // ... existing policy matching (enhanced with protocol filter) ...

    // ... existing action enforcement ...
}
```

**Policy Matching Enhancement**:

**File**: `src/bpf/headers/policy_match.h` (MODIFIED)

```c
// Add protocol filter to wildcard policy matching
static __always_inline bool match_wildcard_policy(
    struct wildcard_policy *policy,
    struct flow_key *key,
    struct session_value *session,  // NEW parameter
    __u8 direction)
{
    // ... existing L3/L4 matching logic ...

    // NEW: Application protocol filter
    if (policy->app_protocol != 0) {
        // Check if session has protocol info
        if (!session || session->app_protocol == APP_PROTO_UNKNOWN) {
            // No protocol info yet, continue matching
        } else {
            // Match mode: 0 = exact, 1 = exclude
            if (policy->app_proto_match_mode == 0) {
                // Exact match
                if (session->app_protocol != policy->app_protocol)
                    return false;
            } else {
                // Exclude match
                if (session->app_protocol == policy->app_protocol)
                    return false;
            }
        }
    }

    // NEW: Protocol flags filter
    if (policy->app_proto_flags_mask != 0) {
        if (!session)
            return true;  // No session info, allow matching

        // Check if required flags are present
        __u16 required_flags = policy->app_proto_flags_value & policy->app_proto_flags_mask;
        __u16 actual_flags = session->proto_flags & policy->app_proto_flags_mask;

        if (actual_flags != required_flags)
            return false;
    }

    return true;
}
```

---

## 5. Performance Optimization

### 5.1 Optimization Strategies

**1. Lazy Detection**
```c
// Only detect on first few packets (e.g., first 3 packets)
#define PROTO_DETECT_MAX_PACKETS 3

if (session->packets_to_server + session->packets_to_client <= PROTO_DETECT_MAX_PACKETS) {
    detect_app_protocol(...);
}
```

**2. Confidence-Based Early Exit**
```c
// Stop detection once high confidence is reached
if (session->proto_confidence >= 90) {
    return;  // Already identified with high confidence
}
```

**3. Payload Length Limiting**
```c
// Inspect only first N bytes (configurable, default 128)
#define DEFAULT_MAX_PAYLOAD_INSPECT 128

__u32 inspect_len = min(payload_len, config->max_payload_bytes);
```

**4. Sampling**
```c
// Detect every N packets (configurable, default 1 = every packet)
if (config->sampling_interval > 0) {
    if ((packet_count % config->sampling_interval) != 0)
        return;
}
```

**5. Protocol Prioritization**
```c
// Check most common protocols first
// Order by likelihood: HTTP/HTTPS (80%), DNS (10%), others (10%)

if (detect_http(...)) return;      // Most common
if (detect_tls(...)) return;
if (detect_dns(...)) return;
// ... less common protocols
```

### 5.2 eBPF Verifier Optimization

**Instruction Count Reduction**:

```c
// Use helper macros to reduce code duplication
#define SAFE_READ_U8(ptr, end) \
    ((ptr) + 1 <= (end) ? *(__u8 *)(ptr) : 0)

#define SAFE_READ_U16(ptr, end) \
    ((ptr) + 2 <= (end) ? (*(__u8 *)(ptr) << 8) | *(__u8 *)((ptr) + 1) : 0)

// Use #pragma unroll for bounded loops
#pragma unroll
for (int i = 0; i < 8; i++) {
    // Loop body
}

// Use likely/unlikely hints
if (__builtin_expect(session->proto_confidence >= 90, 1)) {
    return;  // Fast path
}
```

**Tail Calls for Complex Detection**:

```c
// If a single protocol detector is too complex, use tail calls
enum {
    TAIL_CALL_DETECT_HTTP = 0,
    TAIL_CALL_DETECT_MYSQL = 1,
    // ...
};

struct {
    __uint(type, BPF_MAP_TYPE_PROG_ARRAY);
    __uint(max_entries, 10);
    __type(key, __u32);
    __type(value, __u32);
} proto_detector_progs SEC(".maps");

// In main program
if (session->app_protocol == APP_PROTO_UNKNOWN) {
    __u32 prog_id = TAIL_CALL_DETECT_HTTP;
    bpf_tail_call(skb, &proto_detector_progs, prog_id);
}

// Separate program for HTTP detection
SEC("tc/detect_http")
int detect_http_tail(struct __sk_buff *skb) {
    // Complex HTTP detection logic
    // ...
}
```

### 5.3 Performance Metrics

**Expected Overhead**:

| Scenario | Overhead | Notes |
|----------|----------|-------|
| No payload (TCP SYN) | <1μs | Port heuristic only |
| Payload inspection (128 bytes) | 10-50μs | Depends on protocol complexity |
| High confidence cached | <1μs | Skip detection |
| Sampling (every 10 packets) | <5μs avg | Amortized cost |

**Optimization Target**: <5% total packet processing overhead

---

## 6. User-Space API

### 6.1 Go Types

**File**: `src/agent/pkg/protocol/types.go` (NEW)

```go
package protocol

// AppProtocol represents an application layer protocol
type AppProtocol uint8

const (
    AppProtoUnknown AppProtocol = iota
    AppProtoHTTP
    AppProtoHTTPS
    AppProtoDNS
    AppProtoSSH
    AppProtoMySQL
    AppProtoPostgreSQL
    AppProtoRedis
    AppProtoMongoDB
    AppProtoKafka
    AppProtoRabbitMQ
    AppProtoGRPC
    AppProtoFTP
    AppProtoSMTP
    AppProtoMax = 100
)

func (p AppProtocol) String() string {
    switch p {
    case AppProtoHTTP:
        return "HTTP"
    case AppProtoHTTPS:
        return "HTTPS"
    case AppProtoDNS:
        return "DNS"
    case AppProtoSSH:
        return "SSH"
    case AppProtoMySQL:
        return "MySQL"
    case AppProtoPostgreSQL:
        return "PostgreSQL"
    case AppProtoRedis:
        return "Redis"
    case AppProtoMongoDB:
        return "MongoDB"
    case AppProtoKafka:
        return "Kafka"
    case AppProtoRabbitMQ:
        return "RabbitMQ"
    case AppProtoGRPC:
        return "gRPC"
    case AppProtoFTP:
        return "FTP"
    case AppProtoSMTP:
        return "SMTP"
    default:
        return "UNKNOWN"
    }
}

// ProtocolFlags represents protocol characteristics
type ProtocolFlags uint16

const (
    ProtoFlagEncrypted  ProtocolFlags = 0x0001
    ProtoFlagCleartext  ProtocolFlags = 0x0002
    ProtoFlagBinary     ProtocolFlags = 0x0004
    ProtoFlagText       ProtocolFlags = 0x0008
    ProtoFlagRequest    ProtocolFlags = 0x0010
    ProtoFlagResponse   ProtocolFlags = 0x0020
    ProtoFlagTunnel     ProtocolFlags = 0x0040
)

// DetectionConfig holds protocol detection configuration
type DetectionConfig struct {
    Enabled             bool     `json:"enabled"`
    SamplingInterval    uint8    `json:"sampling_interval"`    // 0 = every packet
    MaxPayloadBytes     uint16   `json:"max_payload_bytes"`    // Max bytes to inspect
    ConfidenceThreshold uint8    `json:"confidence_threshold"` // Min confidence (0-100)
    EnabledProtocols    []string `json:"enabled_protocols"`    // Filter specific protocols
}

// ProtocolStats represents statistics for a single protocol
type ProtocolStats struct {
    Protocol   string `json:"protocol"`
    Sessions   uint64 `json:"sessions"`
    Packets    uint64 `json:"packets"`
    Bytes      uint64 `json:"bytes"`
    Confidence uint8  `json:"avg_confidence"`
}

// SessionProtocolInfo holds protocol information for a session
type SessionProtocolInfo struct {
    Protocol       AppProtocol   `json:"protocol"`
    Confidence     uint8         `json:"confidence"`
    Flags          ProtocolFlags `json:"flags"`
    FirstSeenTS    uint32        `json:"first_seen_ts"`
    PayloadBytes   uint32        `json:"payload_bytes_inspected"`
}
```

### 6.2 Configuration API

**File**: `src/agent/pkg/protocol/detector.go` (NEW)

```go
package protocol

import (
    "encoding/binary"
    "fmt"

    "github.com/cilium/ebpf"
)

// Detector manages protocol detection configuration
type Detector struct {
    configMap *ebpf.Map
    statsMap  *ebpf.Map
    config    *DetectionConfig
}

// NewDetector creates a new protocol detector
func NewDetector(configMap, statsMap *ebpf.Map) *Detector {
    return &Detector{
        configMap: configMap,
        statsMap:  statsMap,
        config: &DetectionConfig{
            Enabled:             true,
            SamplingInterval:    1,  // Every packet
            MaxPayloadBytes:     128,
            ConfidenceThreshold: 70,
            EnabledProtocols:    []string{},  // All enabled
        },
    }
}

// SetConfig updates protocol detection configuration
func (d *Detector) SetConfig(cfg *DetectionConfig) error {
    // Validate config
    if cfg.MaxPayloadBytes > 1024 {
        return fmt.Errorf("max_payload_bytes too large (max: 1024)")
    }
    if cfg.ConfidenceThreshold > 100 {
        return fmt.Errorf("confidence_threshold out of range (0-100)")
    }

    // Convert to BPF format
    bpfConfig := struct {
        Enabled            uint8
        SamplingInterval   uint8
        MaxPayloadBytes    uint16
        ConfidenceThresh   uint8
        Reserved           [3]uint8
    }{
        Enabled:          boolToUint8(cfg.Enabled),
        SamplingInterval: cfg.SamplingInterval,
        MaxPayloadBytes:  cfg.MaxPayloadBytes,
        ConfidenceThresh: cfg.ConfidenceThreshold,
    }

    // Update map
    key := uint32(0)
    if err := d.configMap.Put(&key, &bpfConfig); err != nil {
        return fmt.Errorf("failed to update config map: %w", err)
    }

    d.config = cfg
    return nil
}

// GetConfig returns current configuration
func (d *Detector) GetConfig() *DetectionConfig {
    return d.config
}

// GetStats returns protocol statistics
func (d *Detector) GetStats() ([]ProtocolStats, error) {
    stats := []ProtocolStats{}

    // Iterate over all protocol IDs
    for protoID := AppProtoHTTP; protoID < AppProtoMax; protoID++ {
        key := uint32(protoID)
        var values []uint64

        if err := d.statsMap.Lookup(&key, &values); err != nil {
            continue  // Protocol not seen yet
        }

        // Sum per-CPU values
        total := uint64(0)
        for _, v := range values {
            total += v
        }

        if total == 0 {
            continue
        }

        stats = append(stats, ProtocolStats{
            Protocol: AppProtocol(protoID).String(),
            Packets:  total,
        })
    }

    return stats, nil
}

func boolToUint8(b bool) uint8 {
    if b {
        return 1
    }
    return 0
}
```

### 6.3 REST API Endpoints

**File**: `src/server/api/protocol_handler.go` (NEW)

```go
package api

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/yourusername/microsegment/src/agent/pkg/protocol"
)

type ProtocolHandler struct {
    detector *protocol.Detector
}

func NewProtocolHandler(detector *protocol.Detector) *ProtocolHandler {
    return &ProtocolHandler{detector: detector}
}

// GET /api/v1/protocols/config
func (h *ProtocolHandler) GetConfig(c *gin.Context) {
    config := h.detector.GetConfig()
    c.JSON(http.StatusOK, config)
}

// PUT /api/v1/protocols/config
func (h *ProtocolHandler) UpdateConfig(c *gin.Context) {
    var config protocol.DetectionConfig
    if err := c.BindJSON(&config); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if err := h.detector.SetConfig(&config); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Config updated"})
}

// GET /api/v1/protocols/stats
func (h *ProtocolHandler) GetStats(c *gin.Context) {
    stats, err := h.detector.GetStats()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, stats)
}

// RegisterRoutes registers protocol API routes
func (h *ProtocolHandler) RegisterRoutes(router *gin.RouterGroup) {
    protocols := router.Group("/protocols")
    {
        protocols.GET("/config", h.GetConfig)
        protocols.PUT("/config", h.UpdateConfig)
        protocols.GET("/stats", h.GetStats)
    }
}
```

---

## 7. Implementation Roadmap

### Phase 1: Foundation (3-4 days)

**Day 1: Data Structure Setup**
- [ ] Create `app_protocol_types.h` with protocol enums
- [ ] Extend `session_value` struct with protocol fields
- [ ] Extend `wildcard_policy` struct with protocol filters
- [ ] Create protocol detection config map
- [ ] Create protocol stats map
- [ ] Write unit tests for data structures

**Day 2: Core Detection Framework**
- [ ] Implement `app_protocol_detection.h` framework
- [ ] Implement `guess_protocol_by_port()` function
- [ ] Implement `detect_app_protocol()` entry point
- [ ] Add payload extraction helpers
- [ ] Integrate into TC program (skeleton)
- [ ] Integrate into XDP program (skeleton)
- [ ] Test basic port-based detection

**Day 3: Policy Engine Integration**
- [ ] Modify `policy_match.h` to support protocol filters
- [ ] Update policy compiler (Go) to handle protocol fields
- [ ] Update policy API to accept protocol constraints
- [ ] Write integration tests

### Phase 2: Protocol Detectors (4-5 days)

**Day 4: HTTP/HTTPS Detection**
- [ ] Implement `app_protocol_http.h`
  - [ ] HTTP method detection (GET, POST, PUT, DELETE, HEAD)
  - [ ] HTTP response detection (HTTP/1.x)
  - [ ] TLS/SSL handshake detection
  - [ ] TLS version identification
- [ ] Write unit tests
- [ ] Benchmark detection performance

**Day 5: DNS Detection**
- [ ] Implement `app_protocol_dns.h`
  - [ ] DNS header validation
  - [ ] Query/response differentiation
  - [ ] DNS tunneling heuristics
- [ ] Write unit tests
- [ ] Test with real DNS traffic

**Day 6: SSH and Database Protocols**
- [ ] Implement `app_protocol_ssh.h`
  - [ ] SSH banner detection (SSH-2.0)
- [ ] Implement `app_protocol_mysql.h`
  - [ ] MySQL handshake detection
  - [ ] Client command detection
- [ ] Implement `app_protocol_redis.h`
  - [ ] RESP protocol detection
- [ ] Write unit tests

**Day 7-8: Additional Protocols**
- [ ] Implement PostgreSQL detector
- [ ] Implement MongoDB detector (if needed)
- [ ] Implement Kafka detector (if needed)
- [ ] Implement gRPC detector (HTTP/2 + protocol buffers)
- [ ] Write comprehensive tests

### Phase 3: User-Space Integration (3-4 days)

**Day 9: Go API Implementation**
- [ ] Create `protocol` package
- [ ] Implement `types.go` (AppProtocol, flags, etc.)
- [ ] Implement `detector.go` (config management)
- [ ] Implement statistics aggregation
- [ ] Write Go unit tests

**Day 10: Agent Integration**
- [ ] Integrate protocol detector into Agent
- [ ] Update flow event structure to include protocol info
- [ ] Implement protocol-based anomaly detection
- [ ] Update session manager to expose protocol info
- [ ] Write integration tests

**Day 11: REST API and Server**
- [ ] Implement protocol REST API handlers
- [ ] Add protocol endpoints to server
- [ ] Update OpenAPI specification
- [ ] Test API endpoints

### Phase 4: Testing and Optimization (2-3 days)

**Day 12: Functional Testing**
- [ ] End-to-end testing with real traffic
  - [ ] HTTP/HTTPS workloads
  - [ ] DNS queries
  - [ ] SSH connections
  - [ ] Database connections (MySQL, Redis, PostgreSQL)
- [ ] Protocol-based policy testing
- [ ] Anomaly detection testing

**Day 13: Performance Testing**
- [ ] Benchmark eBPF program execution time
- [ ] Measure CPU overhead
- [ ] Measure memory usage
- [ ] Packet throughput testing (with/without detection)
- [ ] Optimize hot paths

**Day 14: Bug Fixes and Documentation**
- [ ] Fix bugs found during testing
- [ ] Write user documentation
- [ ] Create configuration examples
- [ ] Update architecture diagrams

---

## 8. Technical Challenges

### 8.1 eBPF Verifier Complexity

**Challenge**: Complex protocol detection logic may exceed eBPF instruction limit (1M instructions, but practical limit is lower).

**Solutions**:

1. **Simplify Detection Logic**
   - Focus on high-confidence signatures
   - Avoid deep payload inspection
   - Use fixed-size pattern matching

2. **Use Tail Calls**
   - Split complex detectors into separate programs
   - Chain them via `bpf_tail_call()`

3. **Offload to User-Space**
   - Perform initial detection in eBPF
   - Deep inspection in user-space (via ring buffer)

**Mitigation Plan**:
- Start with simple detectors
- Measure instruction count (`bpftool prog dump xlated`)
- Refactor if approaching limits

### 8.2 Payload Access Limitations

**Challenge**: eBPF has strict memory safety requirements. Accessing payload requires careful bounds checking.

**Solutions**:

1. **Strict Bounds Checking**
   ```c
   if (payload + offset + length > data_end)
       return false;
   ```

2. **Use Helper Macros**
   ```c
   #define SAFE_READ(ptr, end, len) \
       ((ptr) + (len) <= (end))
   ```

3. **Limit Inspection Length**
   - Default: 128 bytes
   - Configurable via map

**Mitigation Plan**:
- Test all detectors with verifier
- Use `#pragma unroll` for loops
- Keep inspection lengths small

### 8.3 Encrypted Traffic Detection

**Challenge**: TLS-encrypted payloads cannot be inspected for content.

**Solutions**:

1. **TLS Handshake Inspection**
   - Detect ClientHello/ServerHello
   - Extract SNI (Server Name Indication)
   - Identify cipher suites

2. **Port-Based Fallback**
   - Use port 443 → HTTPS
   - Lower confidence score

3. **Future Enhancement: eBPF SSL uprobes**
   - Hook into OpenSSL/BoringSSL functions
   - Access decrypted data (requires kernel 5.5+)

**Mitigation Plan**:
- Focus on TLS handshake detection
- Document limitations for encrypted protocols
- Plan future enhancement for uprobe-based inspection

### 8.4 Protocol Versioning and Variants

**Challenge**: Protocols have multiple versions (HTTP/1.1, HTTP/2, HTTP/3) and vendor-specific variants.

**Solutions**:

1. **Focus on Common Versions**
   - HTTP/1.1 (majority of traffic)
   - TLS 1.2/1.3
   - MySQL 5.x/8.x

2. **Generic Detection**
   - Detect protocol family (HTTP), not specific version
   - Version-specific detection as enhancement

3. **Extensible Design**
   - Easy to add new protocol variants
   - Version-specific flags in `proto_flags`

**Mitigation Plan**:
- Implement most common versions first
- Document supported versions
- Plan for incremental enhancement

### 8.5 Performance Impact

**Challenge**: Protocol detection adds overhead to packet processing.

**Solutions**:

1. **Lazy Detection**
   - Detect only on first few packets
   - Cache results in session

2. **Sampling**
   - Configurable sampling interval
   - Trade accuracy for performance

3. **Early Exit**
   - Stop detection once high confidence reached
   - Skip detection for already-identified sessions

4. **Payload Length Limiting**
   - Inspect only first N bytes (default 128)
   - Avoid reading entire packet

**Mitigation Plan**:
- Benchmark each detector
- Set performance targets (<100μs per packet)
- Implement optimizations iteratively

### 8.6 TCP Reassembly Necessity

**Challenge**: Application layer protocols (like HTTP) may be split across multiple TCP segments, requiring reassembly for accurate detection.

**Analysis**:

Protocol detection could fail in scenarios where the protocol signature spans multiple TCP segments:

```
Scenario: HTTP request split across TCP segments
┌─────────────────────────────────┐
│ TCP Segment 1 (Seq=100, Len=32)│
│ Payload: "GET /very/long/url/pa"│  ← Incomplete, cannot match
└─────────────────────────────────┘
         ↓
┌─────────────────────────────────┐
│ TCP Segment 2 (Seq=132, Len=30)│
│ Payload: "th HTTP/1.1\r\nHost: "│
└─────────────────────────────────┘
```

**Occurrence Probability Assessment**:

Based on analysis of typical network conditions and protocol characteristics:

| Protocol/Scenario | First Packet Detection | Reason |
|-------------------|------------------------|---------|
| HTTP GET (short URL) | **95%+** | Typical GET request < 200 bytes, MSS = 1460 bytes |
| HTTP POST (small body) | **90%+** | Headers usually fit in first segment |
| HTTPS/TLS ClientHello | **99%+** | ClientHello typically < 512 bytes |
| DNS Query (UDP) | **99.9%+** | No TCP segmentation, queries < 100 bytes |
| MySQL Handshake | **95%+** | Handshake packets < 200 bytes |
| Redis Commands | **98%+** | RESP protocol commands are small |
| Out-of-Order Arrival | **1-5%** | Rare in good network conditions |

**Current Codebase Capability**:

The project already tracks TCP sequence numbers in `session_value`:
```c
struct session_value {
    __u32 tcp_seq_client;     // Last TCP sequence number from client
    __u32 tcp_seq_server;     // Last TCP sequence number from server
    __u32 tcp_ack_client;     // Last TCP acknowledgment from client
    __u32 tcp_ack_server;     // Last TCP acknowledgment from server
    // ...
};
```

However, **these fields are currently unused** - no reassembly logic is implemented.

**Solutions (Progressive Implementation)**:

**✅ Recommended: Stage 1 - First Packet Only (Implement Now)**

Detect protocols only on the first data packet, without reassembly:

```c
// Detect on first payload-bearing packet only
if (session->packets_to_server == 1 ||  // First client data packet
    session->packets_to_client == 1) {  // First server data packet
    detect_app_protocol(payload, data_end, key, session, config);
}
```

**Advantages**:
- ✅ Simple implementation, no reassembly needed
- ✅ Covers 90%+ scenarios
- ✅ Low overhead (<50μs)
- ✅ Production-ready immediately

**Limitations**:
- ❌ Cannot detect protocols with headers split across segments
- ❌ Cannot handle out-of-order arrival

**⚠️ Optional: Stage 2 - Sequence Number Tracking**

Track sequence numbers to detect out-of-order packets, but don't cache data:

```c
static __always_inline bool tcp_seq_is_expected(
    struct session_value *session,
    __u32 tcp_seq,
    __u32 payload_len,
    bool is_client_to_server)
{
    if (is_client_to_server) {
        if (session->tcp_seq_client == 0) {
            session->tcp_seq_client = tcp_seq + payload_len;
            return true;
        }

        if (tcp_seq == session->tcp_seq_client) {
            session->tcp_seq_client += payload_len;
            return true;
        }

        // Out-of-order or retransmission
        session->proto_flags |= PROTO_FLAG_OUT_OF_ORDER;
        return false;
    }
    // Similar for server direction...
}
```

**Advantages**:
- ✅ Detect out-of-order scenarios
- ✅ Minimal overhead (sequence number comparison only)
- ✅ Provides observability metrics

**🔧 Future: Stage 3 - User-Space Reassembly (If Needed)**

If metrics show >10% undetected flows, implement reassembly in user-space:

```go
// src/agent/pkg/protocol/reassembler.go
type TCPReassembler struct {
    streams map[FlowKey]*TCPStream
}

func (r *TCPReassembler) ProcessPacket(pkt *Packet) {
    stream := r.getOrCreateStream(pkt.FlowKey)
    stream.AddSegment(pkt)

    if stream.IsReassembled() {
        protocol := DetectProtocol(stream.GetPayload())
        updateSessionProtocol(pkt.FlowKey, protocol)
    }
}
```

**Advantages**:
- ✅ Full reassembly capability
- ✅ No eBPF limitations
- ✅ Can use mature libraries (gopacket/reassembly)

**Limitations**:
- ❌ Higher latency (userspace processing)
- ❌ Only for flows that fail eBPF detection

**🚧 Discouraged: Stage 4 - eBPF Lightweight Reassembly**

Cache first 2-3 segments in eBPF (only if absolutely necessary):

```c
#define MAX_CACHED_SEGMENTS 3
#define MAX_SEGMENT_SIZE 128

struct tcp_segment_cache {
    __u32 seq[MAX_CACHED_SEGMENTS];
    __u8  data[MAX_CACHED_SEGMENTS][MAX_SEGMENT_SIZE];
    __u16 len[MAX_CACHED_SEGMENTS];
    __u8  count;
};
```

**Limitations**:
- ❌ High memory overhead (~400 bytes per flow)
- ❌ Complex implementation, may exceed verifier limits
- ❌ Only handles first few segments

**Performance Comparison**:

| Approach | Per-Packet Overhead | Memory Overhead | Coverage | Complexity |
|----------|---------------------|-----------------|----------|------------|
| Stage 1: First Packet Only | <10μs | 0 bytes | 90%+ | Low ✅ |
| Stage 2: + Seq Tracking | <20μs | 8 bytes/session | 90%+ | Low ✅ |
| Stage 3: User-Space Reassembly | 100-500μs | High (userspace) | 99%+ | Medium ⚠️ |
| Stage 4: eBPF Reassembly | 50-200μs | ~400 bytes/session | 95%+ | High ❌ |

**Recommended Implementation Strategy**:

1. **Phase 1**: Implement Stage 1 (first packet detection only)
2. **Phase 1.5**: Add Stage 2 (sequence number tracking)
3. **Monitor**: Collect metrics on detection success rate
4. **Evaluate**: If undetected flows >10%, consider Stage 3

**Monitoring Metrics**:

```c
enum stats_key {
    STATS_PROTO_DETECTED = 20,         // Successfully identified flows
    STATS_PROTO_UNKNOWN,               // Unidentified flows
    STATS_PROTO_OUT_OF_ORDER,          // Out-of-order packets
    STATS_PROTO_FIRST_PKT_TOO_SMALL,   // First packet too small
    STATS_PROTO_SPLIT_HEADER,          // Suspected split headers
};
```

**Decision Criteria**:
- If `PROTO_UNKNOWN` > 10% → Optimize detection algorithms
- If `PROTO_OUT_OF_ORDER` > 5% → Consider user-space reassembly
- If `PROTO_SPLIT_HEADER` > 5% → Consider eBPF lightweight reassembly

**Mitigation Plan**:
- Implement Stage 1 + Stage 2 initially (minimal overhead, 90%+ coverage)
- Add comprehensive metrics to monitor detection effectiveness
- Defer full reassembly until proven necessary by production data
- See [TCP_REASSEMBLY_ANALYSIS.md](./TCP_REASSEMBLY_ANALYSIS.md) for detailed analysis

---

## 9. Testing Strategy

### 9.1 Unit Testing

**eBPF Unit Tests** (using libbpf's test framework):

```c
// test_protocol_detection.c
void test_http_detection() {
    char http_get[] = "GET / HTTP/1.1\r\nHost: example.com\r\n";
    __u8 confidence = 0;
    __u16 flags = 0;

    bool result = detect_http(http_get, http_get + sizeof(http_get),
                              sizeof(http_get), &confidence, &flags);

    assert(result == true);
    assert(confidence >= 90);
    assert(flags & PROTO_FLAG_CLEARTEXT);
    assert(flags & PROTO_FLAG_REQUEST);
}

void test_tls_detection() {
    __u8 tls_handshake[] = {0x16, 0x03, 0x03, 0x00, 0x05, 0x01};
    __u8 confidence = 0;
    __u16 flags = 0;

    bool result = detect_tls(tls_handshake, tls_handshake + sizeof(tls_handshake),
                             sizeof(tls_handshake), &confidence, &flags);

    assert(result == true);
    assert(confidence >= 95);
    assert(flags & PROTO_FLAG_ENCRYPTED);
}
```

**Go Unit Tests**:

```go
func TestProtocolDetectorConfig(t *testing.T) {
    // Create test eBPF maps
    configMap := createTestMap(t, "proto_detect_config_map")
    statsMap := createTestMap(t, "proto_stats_map")

    detector := protocol.NewDetector(configMap, statsMap)

    // Test config update
    cfg := &protocol.DetectionConfig{
        Enabled:             true,
        SamplingInterval:    5,
        MaxPayloadBytes:     256,
        ConfidenceThreshold: 80,
    }

    err := detector.SetConfig(cfg)
    assert.NoError(t, err)

    // Verify config persisted
    readCfg := detector.GetConfig()
    assert.Equal(t, cfg.MaxPayloadBytes, readCfg.MaxPayloadBytes)
}
```

### 9.2 Integration Testing

**Test Environment Setup**:

```bash
# Start test containers
docker-compose -f test/docker-compose-proto.yml up -d

# Containers:
# - nginx (HTTP/HTTPS server)
# - mysql (MySQL server)
# - redis (Redis server)
# - dns-server (Bind9)
# - ssh-server (OpenSSH)
```

**Integration Test Cases**:

```go
func TestHTTPDetection(t *testing.T) {
    // Load eBPF programs
    agent := setupTestAgent(t)
    defer agent.Close()

    // Generate HTTP traffic
    client := &http.Client{}
    resp, err := client.Get("http://nginx-test/")
    require.NoError(t, err)
    defer resp.Body.Close()

    // Wait for detection
    time.Sleep(100 * time.Millisecond)

    // Query sessions
    sessions := agent.GetSessions()

    // Verify protocol detected
    found := false
    for _, sess := range sessions {
        if sess.DstPort == 80 {
            assert.Equal(t, protocol.AppProtoHTTP, sess.Protocol)
            assert.GreaterOrEqual(t, sess.Confidence, uint8(85))
            found = true
        }
    }
    assert.True(t, found, "HTTP session not found")
}

func TestTLSDetection(t *testing.T) {
    agent := setupTestAgent(t)
    defer agent.Close()

    // Generate HTTPS traffic
    client := &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
        },
    }
    resp, err := client.Get("https://nginx-test/")
    require.NoError(t, err)
    defer resp.Body.Close()

    // Verify protocol
    time.Sleep(100 * time.Millisecond)
    sessions := agent.GetSessions()

    found := false
    for _, sess := range sessions {
        if sess.DstPort == 443 {
            assert.Equal(t, protocol.AppProtoHTTPS, sess.Protocol)
            assert.GreaterOrEqual(t, sess.Confidence, uint8(95))
            found = true
        }
    }
    assert.True(t, found, "HTTPS session not found")
}
```

### 9.3 Performance Testing

**Benchmark Setup**:

```bash
# Use tcpreplay to replay PCAP files
sudo tcpreplay -i eth0 -M 10000 test/pcaps/mixed-traffic.pcap

# Monitor eBPF program performance
sudo bpftool prog profile id <prog_id> duration 10
```

**Performance Metrics**:

| Metric | Tool | Target |
|--------|------|--------|
| eBPF execution time | bpftool prog profile | <50μs p99 |
| CPU overhead | perf top | <5% |
| Packet drop rate | tc -s qdisc show | <0.1% |
| Detection accuracy | Test suite | >95% |

**Load Testing Script**:

```go
func BenchmarkProtocolDetection(b *testing.B) {
    agent := setupTestAgent(b)
    defer agent.Close()

    // Generate mixed traffic
    clients := []struct {
        protocol string
        endpoint string
    }{
        {"HTTP", "http://nginx-test/"},
        {"HTTPS", "https://nginx-test/"},
        {"DNS", "8.8.8.8:53"},
        {"SSH", "ssh-test:22"},
    }

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        for _, c := range clients {
            // Send request (implement per protocol)
            sendRequest(c.protocol, c.endpoint)
        }
    }

    b.StopTimer()

    // Report statistics
    stats := agent.GetStats()
    b.ReportMetric(float64(stats.TotalPackets)/b.Elapsed().Seconds(), "pps")
}
```

### 9.4 Acceptance Testing

**User Acceptance Criteria**:

1. ✅ **Functional Requirements**
   - [ ] Detects HTTP, HTTPS, DNS, SSH, MySQL, Redis with >90% accuracy
   - [ ] Protocol-based policies work correctly (allow/deny)
   - [ ] Protocol statistics are accurate
   - [ ] Configuration changes take effect immediately

2. ✅ **Performance Requirements**
   - [ ] Packet processing overhead <5%
   - [ ] Detection latency <100μs p99
   - [ ] No packet drops under normal load

3. ✅ **Usability Requirements**
   - [ ] REST API is intuitive
   - [ ] Configuration is easy to understand
   - [ ] Error messages are clear

4. ✅ **Security Requirements**
   - [ ] No information leakage in logs
   - [ ] Handles malformed packets gracefully
   - [ ] Resistant to DoS attacks

---

## 10. References

### 10.1 Internal References

- **Codebase Exploration Report**: See Task agent output (comprehensive analysis)
- **Existing Headers**:
  - `src/bpf/headers/common_types.h` - Core data structures
  - `src/bpf/headers/flow_processing.h` - Packet parsing
  - `src/bpf/headers/policy_match.h` - Policy engine
  - `src/bpf/headers/tcp_state_machine.h` - TCP state tracking
  - `src/bpf/headers/nat_support.h` - NAT detection
  - `src/bpf/headers/fragment_tracking.h` - IP fragmentation

- **NeuVector DPI Reference**:
  - `source-references/neuvector/dp/dpi/` - Full DPI implementation
  - `source-references/neuvector/dp/dpi/dpi_parser.c` - Parser framework
  - `source-references/neuvector/dp/dpi/parsers/` - Protocol detectors

### 10.2 External References

**eBPF Programming**:
- [Cilium eBPF Library](https://github.com/cilium/ebpf)
- [libbpf Documentation](https://libbpf.readthedocs.io/)
- [BPF and XDP Reference Guide](https://docs.cilium.io/en/latest/bpf/)

**Protocol Specifications**:
- [HTTP/1.1 RFC 7230](https://tools.ietf.org/html/rfc7230)
- [TLS 1.3 RFC 8446](https://tools.ietf.org/html/rfc8446)
- [DNS RFC 1035](https://tools.ietf.org/html/rfc1035)
- [SSH Protocol RFC 4253](https://tools.ietf.org/html/rfc4253)
- [MySQL Protocol](https://dev.mysql.com/doc/internals/en/client-server-protocol.html)
- [Redis RESP Protocol](https://redis.io/docs/reference/protocol-spec/)

**Similar Projects**:
- [Katran (Facebook's L4 load balancer)](https://github.com/facebookincubator/katran)
- [Cilium](https://github.com/cilium/cilium)
- [NeuVector](https://github.com/neuvector/neuvector)

---

## Appendix A: Configuration Examples

### Example 1: Enable Protocol Detection

```bash
# Via REST API
curl -X PUT http://localhost:8080/api/v1/protocols/config \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "sampling_interval": 1,
    "max_payload_bytes": 128,
    "confidence_threshold": 70
  }'
```

### Example 2: Protocol-Based Policy

```yaml
# Policy: Allow HTTPS only to web tier
policies:
  - name: "web-https-only"
    from:
      labels:
        app: "external"
    to:
      labels:
        tier: "web"
    rules:
      - protocol: "HTTPS"
        action: "ALLOW"
      - protocol: "HTTP"
        action: "DENY"
```

### Example 3: Detect Protocol Anomalies

```go
// User-space anomaly detection
func (d *Detector) DetectAnomalies(session *Session) []Anomaly {
    anomalies := []Anomaly{}

    // Check for protocol mismatch
    if session.DstPort == 80 && session.AppProtocol == protocol.AppProtoSSH {
        anomalies = append(anomalies, Anomaly{
            Type:        "protocol_mismatch",
            Description: "SSH detected on HTTP port (80)",
            Severity:    "HIGH",
        })
    }

    // Check for DNS tunneling
    if session.AppProtocol == protocol.AppProtoDNS &&
       session.ProtoFlags & protocol.ProtoFlagTunnel != 0 {
        anomalies = append(anomalies, Anomaly{
            Type:        "dns_tunneling",
            Description: "Potential DNS tunneling detected",
            Severity:    "MEDIUM",
        })
    }

    return anomalies
}
```

---

## Appendix B: Performance Tuning Guide

### Tuning Parameters

| Parameter | Default | Low Overhead | High Accuracy |
|-----------|---------|--------------|---------------|
| `sampling_interval` | 1 | 10 | 1 |
| `max_payload_bytes` | 128 | 64 | 256 |
| `confidence_threshold` | 70 | 80 | 60 |

### Optimization Tips

1. **For High-Throughput Environments**:
   - Increase sampling interval to 5-10
   - Reduce max payload bytes to 64
   - Increase confidence threshold to 85

2. **For Security-Focused Deployments**:
   - Keep sampling interval at 1
   - Increase max payload bytes to 256
   - Lower confidence threshold to 60

3. **For Resource-Constrained Systems**:
   - Disable less common protocol detectors
   - Use port-based heuristics only
   - Increase sampling interval

---

**Document Version**: 1.0
**Last Updated**: 2025-11-19
**Status**: Design Phase
**Next Review**: After Phase 1 completion
