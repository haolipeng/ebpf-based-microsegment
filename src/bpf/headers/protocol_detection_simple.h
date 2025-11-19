// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Simplified Application Layer Protocol Detection
 *
 * This is a simplified version that only detects HTTP and HTTPS/TLS
 * to avoid eBPF verifier complexity issues.
 *
 * Supported protocols:
 * - HTTP (GET, POST, PUT methods)
 * - HTTPS/TLS (ClientHello detection)
 */

#ifndef __PROTOCOL_DETECTION_SIMPLE_H__
#define __PROTOCOL_DETECTION_SIMPLE_H__

// Maximum payload bytes to inspect
#define MAX_INSPECT_BYTES 32

// Minimum confidence threshold
#define MIN_CONFIDENCE_THRESHOLD 80

/* Check if pointer is within packet bounds */
static __always_inline bool is_safe_access(void *ptr, void *data_end, __u32 size)
{
    return (ptr + size <= data_end);
}

/* Detect HTTP protocol (simplified)
 * Only checks for GET, POST, PUT, HTTP/1
 */
static __always_inline bool detect_http_simple(
    void *payload_start,
    void *data_end,
    __u32 inspect_len,
    __u8 *confidence,
    __u16 *flags)
{
    if (inspect_len < 4) {
        return false;
    }

    unsigned char *p = (unsigned char *)payload_start;

    // Bounds check
    if (!is_safe_access(p, data_end, 4)) {
        return false;
    }

    // Check for HTTP methods (GET, POST, PUT)
    if ((p[0] == 'G' && p[1] == 'E' && p[2] == 'T' && p[3] == ' ') ||
        (p[0] == 'P' && p[1] == 'O' && p[2] == 'S' && p[3] == 'T') ||
        (p[0] == 'P' && p[1] == 'U' && p[2] == 'T' && p[3] == ' ')) {
        *confidence = 90;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_REQUEST;
        return true;
    }

    // Check for HTTP response (HTTP/1)
    if (inspect_len >= 8 && is_safe_access(p, data_end, 8)) {
        if (p[0] == 'H' && p[1] == 'T' && p[2] == 'T' && p[3] == 'P' &&
            p[4] == '/' && p[5] == '1') {
            *confidence = 90;
            *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_RESPONSE;
            return true;
        }
    }

    return false;
}

/* Detect TLS/HTTPS protocol (simplified)
 * Only checks TLS record header
 */
static __always_inline bool detect_tls_simple(
    void *payload_start,
    void *data_end,
    __u32 inspect_len,
    __u8 *confidence,
    __u16 *flags)
{
    if (inspect_len < 3) {
        return false;
    }

    unsigned char *p = (unsigned char *)payload_start;

    // Bounds check
    if (!is_safe_access(p, data_end, 3)) {
        return false;
    }

    // TLS record header: [Content Type (1)] [Version (2)]
    // Content Type: 0x16 = Handshake, 0x17 = Application Data
    // Version: 0x0301 = TLS 1.0, 0x0302 = TLS 1.1, 0x0303 = TLS 1.2, 0x0304 = TLS 1.3
    __u8 content_type = p[0];
    __u8 version_major = p[1];
    __u8 version_minor = p[2];

    // Check for TLS Handshake (0x16) or Application Data (0x17)
    if ((content_type == 0x16 || content_type == 0x17) &&
        version_major == 0x03 &&
        (version_minor >= 0x01 && version_minor <= 0x04)) {
        *confidence = 95;
        *flags = PROTO_FLAG_ENCRYPTED | PROTO_FLAG_BINARY;
        if (content_type == 0x16) {
            *flags |= PROTO_FLAG_REQUEST; // Handshake is usually request
        }
        return true;
    }

    return false;
}

/* Main protocol detection function (simplified) */
static __always_inline int detect_application_protocol_simple(
    struct __sk_buff *skb,
    struct flow_key *key,
    struct session_value *session,
    void *data_end)
{
    // Only detect if not already detected with high confidence
    if (session->proto_confidence >= MIN_CONFIDENCE_THRESHOLD) {
        return 0;
    }

    // Limit detection attempts (max 3 packets to reduce overhead)
    __u64 total_packets = session->packets_to_server + session->packets_to_client;
    if (total_packets > 3) {
        return 0;
    }

    // Only handle TCP for now
    if (key->protocol != IPPROTO_TCP) {
        return 0;
    }

    // Extract packet data
    void *data = (void *)(long)skb->data;
    struct ethhdr *eth = data;

    if (!is_safe_access(eth, data_end, sizeof(*eth))) {
        return -1;
    }

    void *ip_hdr = (void *)(eth + 1);
    void *l4_hdr = NULL;

    // Parse IP header
    if (key->ip_version == 4) {
        struct iphdr *iph = (struct iphdr *)ip_hdr;
        if (!is_safe_access(iph, data_end, sizeof(*iph))) {
            return -1;
        }
        l4_hdr = (void *)iph + (iph->ihl * 4);
    } else if (key->ip_version == 6) {
        struct ipv6hdr *ip6h = (struct ipv6hdr *)ip_hdr;
        if (!is_safe_access(ip6h, data_end, sizeof(*ip6h))) {
            return -1;
        }
        l4_hdr = (void *)(ip6h + 1);
    } else {
        return -1;
    }

    // Parse TCP header
    struct tcphdr *tcph = (struct tcphdr *)l4_hdr;
    if (!is_safe_access(tcph, data_end, sizeof(*tcph))) {
        return -1;
    }

    // Calculate TCP header length
    __u8 tcp_hdr_len = tcph->doff * 4;
    if (tcp_hdr_len < sizeof(*tcph)) {
        return -1;
    }

    // Get payload
    void *payload = (void *)tcph + tcp_hdr_len;
    if (payload >= data_end) {
        return 0; // No payload
    }

    __u32 payload_len = (__u32)((long)data_end - (long)payload);
    if (payload_len == 0) {
        return 0;
    }

    // Limit inspection length
    __u32 inspect_len = payload_len < MAX_INSPECT_BYTES ? payload_len : MAX_INSPECT_BYTES;

    __u8 confidence = 0;
    __u16 proto_flags = 0;

    // Try TLS first (more specific signature)
    if (detect_tls_simple(payload, data_end, inspect_len, &confidence, &proto_flags)) {
        session->app_protocol = APP_PROTO_HTTPS;
        session->proto_confidence = confidence;
        session->proto_flags = proto_flags;
        session->proto_first_seen_ts = (__u32)(session->last_seen_ts / 1000000000ULL);
        session->proto_payload_bytes = (__u16)inspect_len;
        return 0;
    }

    // Try HTTP
    if (detect_http_simple(payload, data_end, inspect_len, &confidence, &proto_flags)) {
        session->app_protocol = APP_PROTO_HTTP;
        session->proto_confidence = confidence;
        session->proto_flags = proto_flags;
        session->proto_first_seen_ts = (__u32)(session->last_seen_ts / 1000000000ULL);
        session->proto_payload_bytes = (__u16)inspect_len;
        return 0;
    }

    return 0;
}

#endif /* __PROTOCOL_DETECTION_SIMPLE_H__ */
