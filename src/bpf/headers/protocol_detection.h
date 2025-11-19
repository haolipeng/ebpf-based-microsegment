// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Application Layer Protocol Detection (Stage 1: First Packet Only)
 *
 * Implements lightweight protocol detection for common application protocols
 * by analyzing payload signatures in the first few packets.
 *
 * Supported protocols:
 * - HTTP (GET, POST, PUT, DELETE, HEAD, OPTIONS, PATCH)
 * - HTTPS/TLS (ClientHello detection)
 * - DNS (Query/Response detection)
 * - SSH (Banner exchange)
 * - MySQL (Handshake packet)
 * - Redis (RESP protocol commands)
 *
 * Detection strategy:
 * - First packet only (90%+ coverage)
 * - Minimal overhead (<10μs per packet)
 * - No TCP reassembly (Stage 1 implementation)
 *
 * Prerequisites:
 * - Include common_types.h (defines enum app_protocol, PROTO_FLAG_*)
 * - Include vmlinux.h (defines struct tcphdr, struct udphdr)
 */

#ifndef __PROTOCOL_DETECTION_H__
#define __PROTOCOL_DETECTION_H__

// Maximum payload bytes to inspect (keep small for performance)
#define MAX_INSPECT_BYTES 64

// Minimum confidence threshold for protocol detection
#define MIN_CONFIDENCE_THRESHOLD 80

/* Check if pointer is within packet bounds
 *
 * @ptr: Pointer to check
 * @data_end: End of packet data
 * @size: Number of bytes to check
 * Returns: true if safe to access, false otherwise
 */
static __always_inline bool is_safe_access(void *ptr, void *data_end, __u32 size)
{
    return (ptr + size <= data_end);
}

/* Extract TCP payload pointer and length
 *
 * @skb: Socket buffer (TC context)
 * @tcph: TCP header pointer
 * @data_end: End of packet data
 * @payload_start: Output - pointer to payload start
 * @payload_len: Output - payload length
 * Returns: 0 on success, -1 on error
 */
static __always_inline int extract_tcp_payload(
    struct __sk_buff *skb,
    struct tcphdr *tcph,
    void *data_end,
    void **payload_start,
    __u32 *payload_len)
{
    // Verify TCP header is accessible
    if (!is_safe_access(tcph, data_end, sizeof(*tcph))) {
        return -1;
    }

    // Calculate TCP header length (doff is in 32-bit words)
    __u8 tcp_hdr_len = tcph->doff * 4;
    if (tcp_hdr_len < sizeof(*tcph)) {
        return -1; // Invalid header length
    }

    // Calculate payload start
    void *payload = (void *)tcph + tcp_hdr_len;
    if (payload >= data_end) {
        *payload_len = 0;
        return 0; // No payload (valid for ACK-only packets)
    }

    // Calculate payload length
    __u32 len = (__u32)((long)data_end - (long)payload);

    *payload_start = payload;
    *payload_len = len;
    return 0;
}

/* Detect HTTP protocol
 *
 * Looks for HTTP methods and version strings in payload.
 * Common patterns:
 * - Request: "GET ", "POST ", "PUT ", "DELETE ", "HEAD ", "OPTIONS ", "PATCH "
 * - Response: "HTTP/1.0", "HTTP/1.1", "HTTP/2"
 *
 * @payload_start: Start of payload
 * @data_end: End of packet data
 * @inspect_len: Number of bytes to inspect
 * @confidence: Output - detection confidence (0-100)
 * @flags: Output - protocol feature flags
 * Returns: true if HTTP detected
 */
static __always_inline bool detect_http(
    void *payload_start,
    void *data_end,
    __u32 inspect_len,
    __u8 *confidence,
    __u16 *flags)
{
    if (inspect_len < 4) {
        return false;
    }

    char *data = (char *)payload_start;

    // Check for HTTP methods (request)
    if (is_safe_access(data, data_end, 4)) {
        // GET
        if (data[0] == 'G' && data[1] == 'E' && data[2] == 'T' && data[3] == ' ') {
            *confidence = 95;
            *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_REQUEST;
            return true;
        }
    }

    if (is_safe_access(data, data_end, 5)) {
        // POST
        if (data[0] == 'P' && data[1] == 'O' && data[2] == 'S' && data[3] == 'T' && data[4] == ' ') {
            *confidence = 95;
            *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_REQUEST;
            return true;
        }
        // HEAD
        if (data[0] == 'H' && data[1] == 'E' && data[2] == 'A' && data[3] == 'D' && data[4] == ' ') {
            *confidence = 95;
            *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_REQUEST;
            return true;
        }
    }

    if (is_safe_access(data, data_end, 6)) {
        // PATCH
        if (data[0] == 'P' && data[1] == 'A' && data[2] == 'T' && data[3] == 'C' &&
            data[4] == 'H' && data[5] == ' ') {
            *confidence = 95;
            *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_REQUEST;
            return true;
        }
    }

    if (is_safe_access(data, data_end, 7)) {
        // DELETE
        if (data[0] == 'D' && data[1] == 'E' && data[2] == 'L' && data[3] == 'E' &&
            data[4] == 'T' && data[5] == 'E' && data[6] == ' ') {
            *confidence = 95;
            *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_REQUEST;
            return true;
        }
    }

    if (is_safe_access(data, data_end, 8)) {
        // OPTIONS
        if (data[0] == 'O' && data[1] == 'P' && data[2] == 'T' && data[3] == 'I' &&
            data[4] == 'O' && data[5] == 'N' && data[6] == 'S' && data[7] == ' ') {
            *confidence = 95;
            *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_REQUEST;
            return true;
        }
    }

    // Check for HTTP response
    if (is_safe_access(data, data_end, 8)) {
        // HTTP/1.
        if (data[0] == 'H' && data[1] == 'T' && data[2] == 'T' && data[3] == 'P' &&
            data[4] == '/' && data[5] == '1' && data[6] == '.') {
            *confidence = 95;
            *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_RESPONSE;
            return true;
        }
    }

    return false;
}

/* Detect TLS/HTTPS protocol
 *
 * Looks for TLS record header in ClientHello or ServerHello.
 * TLS record structure:
 * - Byte 0: Content Type (0x16 = Handshake)
 * - Byte 1-2: Version (0x0301 = TLS 1.0, 0x0303 = TLS 1.2, 0x0304 = TLS 1.3)
 * - Byte 3-4: Record Length
 *
 * @payload_start: Start of payload
 * @data_end: End of packet data
 * @inspect_len: Number of bytes to inspect
 * @confidence: Output - detection confidence (0-100)
 * @flags: Output - protocol feature flags
 * Returns: true if TLS detected
 */
static __always_inline bool detect_tls(
    void *payload_start,
    void *data_end,
    __u32 inspect_len,
    __u8 *confidence,
    __u16 *flags)
{
    if (inspect_len < 5) {
        return false;
    }

    __u8 *data = (__u8 *)payload_start;

    if (!is_safe_access(data, data_end, 5)) {
        return false;
    }

    // Check TLS record header
    // Byte 0: Content Type (0x16 = Handshake, 0x17 = Application Data)
    // Byte 1-2: Version (0x0301-0x0304)
    __u8 content_type = data[0];
    __u8 version_major = data[1];
    __u8 version_minor = data[2];

    // TLS Handshake (ClientHello/ServerHello)
    if (content_type == 0x16 && version_major == 0x03 &&
        (version_minor >= 0x01 && version_minor <= 0x04)) {
        *confidence = 99;
        *flags = PROTO_FLAG_ENCRYPTED | PROTO_FLAG_BINARY;
        return true;
    }

    // TLS Application Data (encrypted payload)
    if (content_type == 0x17 && version_major == 0x03 &&
        (version_minor >= 0x01 && version_minor <= 0x04)) {
        *confidence = 95;
        *flags = PROTO_FLAG_ENCRYPTED | PROTO_FLAG_BINARY;
        return true;
    }

    return false;
}

/* Detect DNS protocol (UDP-based)
 *
 * DNS header structure (12 bytes minimum):
 * - Bytes 0-1: Transaction ID
 * - Bytes 2-3: Flags (QR, Opcode, AA, TC, RD, RA, Z, RCODE)
 * - Bytes 4-5: Question count
 * - Bytes 6-7: Answer count
 * - Bytes 8-9: Authority count
 * - Bytes 10-11: Additional count
 *
 * @payload_start: Start of payload
 * @data_end: End of packet data
 * @inspect_len: Number of bytes to inspect
 * @confidence: Output - detection confidence (0-100)
 * @flags: Output - protocol feature flags
 * Returns: true if DNS detected
 */
static __always_inline bool detect_dns(
    void *payload_start,
    void *data_end,
    __u32 inspect_len,
    __u8 *confidence,
    __u16 *flags)
{
    if (inspect_len < 12) {
        return false;
    }

    __u8 *data = (__u8 *)payload_start;

    if (!is_safe_access(data, data_end, 12)) {
        return false;
    }

    // Check DNS flags (byte 2-3)
    __u16 flags_field = ((__u16)data[2] << 8) | data[3];
    __u8 qr = (flags_field >> 15) & 0x01;      // Query/Response bit
    __u8 opcode = (flags_field >> 11) & 0x0F;  // Opcode
    __u16 question_count = ((__u16)data[4] << 8) | data[5];

    // DNS query typically has:
    // - QR = 0 (query)
    // - Opcode = 0 (standard query)
    // - Question count > 0
    if (qr == 0 && opcode == 0 && question_count > 0 && question_count < 10) {
        *confidence = 90;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_BINARY | PROTO_FLAG_REQUEST;
        return true;
    }

    // DNS response typically has:
    // - QR = 1 (response)
    // - Opcode = 0
    if (qr == 1 && opcode == 0) {
        *confidence = 90;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_BINARY | PROTO_FLAG_RESPONSE;
        return true;
    }

    return false;
}

/* Detect SSH protocol
 *
 * SSH starts with protocol version exchange:
 * "SSH-2.0-" or "SSH-1.99-"
 *
 * @payload_start: Start of payload
 * @data_end: End of packet data
 * @inspect_len: Number of bytes to inspect
 * @confidence: Output - detection confidence (0-100)
 * @flags: Output - protocol feature flags
 * Returns: true if SSH detected
 */
static __always_inline bool detect_ssh(
    void *payload_start,
    void *data_end,
    __u32 inspect_len,
    __u8 *confidence,
    __u16 *flags)
{
    if (inspect_len < 8) {
        return false;
    }

    char *data = (char *)payload_start;

    if (!is_safe_access(data, data_end, 8)) {
        return false;
    }

    // Check for "SSH-2.0-" or "SSH-1.99-"
    if (data[0] == 'S' && data[1] == 'S' && data[2] == 'H' && data[3] == '-') {
        if ((data[4] == '2' && data[5] == '.' && data[6] == '0') ||
            (data[4] == '1' && data[5] == '.' && data[6] == '9' && data[7] == '9')) {
            *confidence = 99;
            *flags = PROTO_FLAG_ENCRYPTED | PROTO_FLAG_TEXT;
            return true;
        }
    }

    return false;
}

/* Detect MySQL protocol
 *
 * MySQL handshake packet starts with:
 * - Byte 0-2: Packet length (3 bytes, little-endian)
 * - Byte 3: Packet sequence number
 * - Byte 4: Protocol version (usually 10)
 *
 * @payload_start: Start of payload
 * @data_end: End of packet data
 * @inspect_len: Number of bytes to inspect
 * @confidence: Output - detection confidence (0-100)
 * @flags: Output - protocol feature flags
 * Returns: true if MySQL detected
 */
static __always_inline bool detect_mysql(
    void *payload_start,
    void *data_end,
    __u32 inspect_len,
    __u8 *confidence,
    __u16 *flags)
{
    if (inspect_len < 5) {
        return false;
    }

    __u8 *data = (__u8 *)payload_start;

    if (!is_safe_access(data, data_end, 5)) {
        return false;
    }

    // Check MySQL handshake:
    // - Sequence number (byte 3) should be 0 for initial handshake
    // - Protocol version (byte 4) is typically 10
    __u8 seq_num = data[3];
    __u8 proto_version = data[4];

    if (seq_num == 0 && proto_version == 10) {
        *confidence = 90;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_BINARY;
        return true;
    }

    return false;
}

/* Detect Redis protocol (RESP)
 *
 * Redis RESP protocol uses specific first characters:
 * - '+' : Simple string
 * - '-' : Error
 * - ':' : Integer
 * - '$' : Bulk string
 * - '*' : Array
 *
 * Common commands start with '*' (array):
 * - "*3\r\n$3\r\nSET\r\n..." (SET command)
 * - "*2\r\n$3\r\nGET\r\n..." (GET command)
 *
 * @payload_start: Start of payload
 * @data_end: End of packet data
 * @inspect_len: Number of bytes to inspect
 * @confidence: Output - detection confidence (0-100)
 * @flags: Output - protocol feature flags
 * Returns: true if Redis detected
 */
static __always_inline bool detect_redis(
    void *payload_start,
    void *data_end,
    __u32 inspect_len,
    __u8 *confidence,
    __u16 *flags)
{
    if (inspect_len < 3) {
        return false;
    }

    char *data = (char *)payload_start;

    if (!is_safe_access(data, data_end, 3)) {
        return false;
    }

    // Check for RESP protocol markers
    char first_char = data[0];

    if (first_char == '*' || first_char == '$' || first_char == '+' ||
        first_char == '-' || first_char == ':') {
        // Additional validation: check for \r\n
        if (inspect_len >= 3) {
            // Look for \r\n within first few bytes
            #pragma unroll
            for (int i = 1; i < 16 && i < inspect_len - 1; i++) {
                if (is_safe_access(data + i, data_end, 2)) {
                    if (data[i] == '\r' && data[i + 1] == '\n') {
                        *confidence = 85;
                        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT;
                        return true;
                    }
                }
            }
        }
    }

    return false;
}

/* Main protocol detection function
 *
 * Attempts to detect application layer protocol from packet payload.
 * Uses first packet only (Stage 1 implementation).
 *
 * Detection order (optimized for common protocols):
 * 1. TLS/HTTPS (most common encrypted)
 * 2. HTTP (most common cleartext)
 * 3. DNS (UDP, very common)
 * 4. SSH (common encrypted)
 * 5. MySQL, Redis (database protocols)
 *
 * @skb: Socket buffer (TC context)
 * @key: Flow key (contains protocol, ports)
 * @session: Session value to update with detection results
 * @data_end: End of packet data
 * Returns: 0 on success, -1 on error
 */
static __always_inline int detect_application_protocol(
    struct __sk_buff *skb,
    struct flow_key *key,
    struct session_value *session,
    void *data_end)
{
    // Only detect once (skip if already detected with high confidence)
    if (session->proto_confidence >= MIN_CONFIDENCE_THRESHOLD) {
        return 0;
    }

    // Limit detection attempts (max 5 packets)
    __u64 total_packets = session->packets_to_server + session->packets_to_client;
    if (total_packets > 5) {
        return 0; // Give up after 5 packets
    }

    void *payload_start = NULL;
    __u32 payload_len = 0;

    // Extract payload based on protocol
    if (key->protocol == IPPROTO_TCP) {
        // TCP: extract payload from TCP header
        void *data = (void *)(long)skb->data;
        struct ethhdr *eth = data;

        if (!is_safe_access(eth, data_end, sizeof(*eth))) {
            return -1;
        }

        void *ip_hdr = (void *)(eth + 1);
        struct iphdr *iph = NULL;
        struct ipv6hdr *ip6h = NULL;
        void *tcp_hdr = NULL;

        if (key->ip_version == 4) {
            iph = (struct iphdr *)ip_hdr;
            if (!is_safe_access(iph, data_end, sizeof(*iph))) {
                return -1;
            }
            tcp_hdr = (void *)iph + (iph->ihl * 4);
        } else if (key->ip_version == 6) {
            ip6h = (struct ipv6hdr *)ip_hdr;
            if (!is_safe_access(ip6h, data_end, sizeof(*ip6h))) {
                return -1;
            }
            tcp_hdr = (void *)(ip6h + 1);
            // Note: IPv6 extension headers not handled in Stage 1
        } else {
            return -1;
        }

        struct tcphdr *tcph = (struct tcphdr *)tcp_hdr;
        if (extract_tcp_payload(skb, tcph, data_end, &payload_start, &payload_len) < 0) {
            return -1;
        }

        // No payload, skip detection
        if (payload_len == 0) {
            return 0;
        }

    } else if (key->protocol == IPPROTO_UDP) {
        // UDP: extract payload from UDP header
        void *data = (void *)(long)skb->data;
        struct ethhdr *eth = data;

        if (!is_safe_access(eth, data_end, sizeof(*eth))) {
            return -1;
        }

        void *ip_hdr = (void *)(eth + 1);
        struct iphdr *iph = NULL;
        struct ipv6hdr *ip6h = NULL;
        void *udp_hdr = NULL;

        if (key->ip_version == 4) {
            iph = (struct iphdr *)ip_hdr;
            if (!is_safe_access(iph, data_end, sizeof(*iph))) {
                return -1;
            }
            udp_hdr = (void *)iph + (iph->ihl * 4);
        } else if (key->ip_version == 6) {
            ip6h = (struct ipv6hdr *)ip_hdr;
            if (!is_safe_access(ip6h, data_end, sizeof(*ip6h))) {
                return -1;
            }
            udp_hdr = (void *)(ip6h + 1);
        } else {
            return -1;
        }

        struct udphdr *udph = (struct udphdr *)udp_hdr;
        if (!is_safe_access(udph, data_end, sizeof(*udph))) {
            return -1;
        }

        payload_start = (void *)(udph + 1);
        if (payload_start >= data_end) {
            return 0; // No payload
        }
        payload_len = (__u32)((long)data_end - (long)payload_start);
    } else {
        // Other protocols not supported
        return 0;
    }

    // Limit inspection length
    __u32 inspect_len = payload_len < MAX_INSPECT_BYTES ? payload_len : MAX_INSPECT_BYTES;

    // Try to detect protocol
    __u8 confidence = 0;
    __u16 proto_flags = 0;
    __u8 detected_proto = APP_PROTO_UNKNOWN;

    // TLS/HTTPS detection (most common encrypted)
    if (detect_tls(payload_start, data_end, inspect_len, &confidence, &proto_flags)) {
        detected_proto = APP_PROTO_HTTPS;
        goto update_session;
    }

    // HTTP detection (most common cleartext)
    if (detect_http(payload_start, data_end, inspect_len, &confidence, &proto_flags)) {
        detected_proto = APP_PROTO_HTTP;
        goto update_session;
    }

    // DNS detection (UDP, very common)
    if (key->protocol == IPPROTO_UDP &&
        detect_dns(payload_start, data_end, inspect_len, &confidence, &proto_flags)) {
        detected_proto = APP_PROTO_DNS;
        goto update_session;
    }

    // SSH detection
    if (detect_ssh(payload_start, data_end, inspect_len, &confidence, &proto_flags)) {
        detected_proto = APP_PROTO_SSH;
        goto update_session;
    }

    // MySQL detection
    if (detect_mysql(payload_start, data_end, inspect_len, &confidence, &proto_flags)) {
        detected_proto = APP_PROTO_MYSQL;
        goto update_session;
    }

    // Redis detection
    if (detect_redis(payload_start, data_end, inspect_len, &confidence, &proto_flags)) {
        detected_proto = APP_PROTO_REDIS;
        goto update_session;
    }

update_session:
    if (detected_proto != APP_PROTO_UNKNOWN) {
        // Update session with detection results
        session->app_protocol = detected_proto;
        session->proto_confidence = confidence;
        session->proto_flags = proto_flags;
        session->proto_first_seen_ts = (__u32)(session->last_seen_ts / 1000000000ULL);
        session->proto_payload_bytes = (__u16)inspect_len;
    }

    return 0;
}

#endif /* __PROTOCOL_DETECTION_H__ */
