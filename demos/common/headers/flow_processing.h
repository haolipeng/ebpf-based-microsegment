// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Flow processing logic for demo programs (Simplified - IPv4 only)
 *
 * This is a simplified version of src/bpf/headers/flow_processing.h
 * Simplifications:
 * - IPv4 only (removed IPv6 support)
 * - No VLAN support
 * - Simplified error handling
 *
 * Main functions:
 * - parse_ethernet(): Parse Ethernet header
 * - parse_ipv4(): Parse IPv4 header
 * - parse_tcp(): Parse TCP header
 * - parse_udp(): Parse UDP header
 * - extract_flow_key_from_packet(): Extract 5-tuple from packet
 *
 * Prerequisites (must be included before this header):
 * 1. Include common_types.h for struct flow_key
 * 2. Include vmlinux.h or relevant headers for network protocol structures
 */

#ifndef __FLOW_PROCESSING_DEMO_H__
#define __FLOW_PROCESSING_DEMO_H__

/* parse_ethernet - Parse Ethernet header (simplified - no VLAN)
 *
 * @data: Packet start pointer
 * @data_end: Packet end pointer
 * @eth_proto: Output parameter - Ethernet protocol type (network byte order)
 *
 * Returns: IP header start pointer, NULL on failure
 *
 * Functionality:
 * 1. Validate Ethernet header boundary
 * 2. Extract protocol type (h_proto)
 * 3. Return IP layer start position
 */
static __always_inline void *parse_ethernet(
	void *data,
	void *data_end,
	__u16 *eth_proto)
{
	struct ethhdr *eth = data;
	void *next_header = (void *)(eth + 1);

	// Boundary check: ensure enough space to read Ethernet header
	if (next_header > data_end)
		return NULL;

	// Extract protocol type (network byte order)
	*eth_proto = eth->h_proto;

	// Return next layer protocol start position
	return next_header;
}

/* parse_ipv4 - Parse IPv4 header
 *
 * @iph: IPv4 header pointer
 * @data_end: Packet end pointer
 * @key: Output parameter - flow key, fills source/destination IP and protocol fields
 *
 * Returns: Transport layer start pointer, NULL on failure
 *
 * Functionality:
 * 1. Validate IPv4 header boundary
 * 2. Extract source/destination IP addresses
 * 3. Extract protocol type (TCP/UDP/ICMP etc.)
 * 4. Calculate transport layer start position (considering IP options)
 */
static __always_inline void *parse_ipv4(
	struct iphdr *iph,
	void *data_end,
	struct flow_key *key)
{
	// Boundary check: ensure enough space to read basic IP header (20 bytes)
	if ((void *)(iph + 1) > data_end)
		return NULL;

	// Extract IP addresses
	key->src_ip = iph->saddr;
	key->dst_ip = iph->daddr;
	key->protocol = iph->protocol;

	// Calculate transport layer start position
	// iph->ihl is IP header length in 4-byte units
	// Example: ihl=5 means 20 bytes (no options), ihl=6 means 24 bytes (with options)
	void *l4 = (void *)iph + (iph->ihl * 4);

	// Boundary check: ensure transport layer start is within packet
	if (l4 > data_end)
		return NULL;

	return l4;
}

/* parse_tcp - Parse TCP header
 *
 * @tcph: TCP header pointer
 * @data_end: Packet end pointer
 * @key: Output parameter - flow key, fills source/destination port fields
 * @tcp_flags: Output parameter - TCP flags (optional, pass NULL if not needed)
 *
 * Returns: 0 on success, -1 on failure
 *
 * Functionality:
 * 1. Validate TCP header boundary
 * 2. Extract source/destination ports (network byte order)
 * 3. Extract TCP flags (FIN, SYN, RST, ACK etc.)
 */
static __always_inline int parse_tcp(
	struct tcphdr *tcph,
	void *data_end,
	struct flow_key *key,
	__u8 *tcp_flags)
{
	// Boundary check: ensure enough space to read basic TCP header (20 bytes)
	if ((void *)(tcph + 1) > data_end)
		return -1;

	// Extract ports (network byte order)
	key->src_port = tcph->source;
	key->dst_port = tcph->dest;

	// Extract TCP flags (if needed)
	if (tcp_flags) {
		// Extract flag bits from tcph
		// FIN=0x01, SYN=0x02, RST=0x04, ACK=0x10 etc.
		*tcp_flags = 0;
		if (tcph->fin) *tcp_flags |= 0x01;
		if (tcph->syn) *tcp_flags |= 0x02;
		if (tcph->rst) *tcp_flags |= 0x04;
		if (tcph->psh) *tcp_flags |= 0x08;
		if (tcph->ack) *tcp_flags |= 0x10;
	}

	return 0;
}

/* parse_udp - Parse UDP header
 *
 * @udph: UDP header pointer
 * @data_end: Packet end pointer
 * @key: Output parameter - flow key, fills source/destination port fields
 *
 * Returns: 0 on success, -1 on failure
 *
 * Functionality:
 * 1. Validate UDP header boundary
 * 2. Extract source/destination ports (network byte order)
 */
static __always_inline int parse_udp(
	struct udphdr *udph,
	void *data_end,
	struct flow_key *key)
{
	// Boundary check: ensure enough space to read UDP header (8 bytes)
	if ((void *)(udph + 1) > data_end)
		return -1;

	// Extract ports (network byte order)
	key->src_port = udph->source;
	key->dst_port = udph->dest;

	return 0;
}

/* extract_flow_key_from_packet - Extract flow key from raw packet (IPv4 only)
 *
 * @data: Packet start pointer
 * @data_end: Packet end pointer
 * @key: Output parameter - extracted flow key (5-tuple)
 *
 * Returns: 0 on success, -1 on failure
 *
 * Functionality:
 * 1. Parse Ethernet header, validate IPv4
 * 2. Parse IP header, extract IP addresses and protocol
 * 3. Parse transport layer based on protocol type (TCP/UDP)
 * 4. For other protocols (ICMP etc.), set ports to 0
 *
 * Supported:
 * - IPv4 over Ethernet (ETH_P_IP = 0x0800)
 * - TCP (IPPROTO_TCP = 6)
 * - UDP (IPPROTO_UDP = 17)
 * - ICMP and other protocols (ports = 0)
 *
 * Not supported:
 * - IPv6 (simplified demo)
 * - VLAN (simplified demo)
 * - Fragmented packets (need upper layer handling)
 */
static __always_inline int extract_flow_key_from_packet(
	void *data,
	void *data_end,
	struct flow_key *key)
{
	__u16 eth_proto;
	void *l3;
	void *l4;
	__u8 protocol;

	// 1. Parse Ethernet header
	l3 = parse_ethernet(data, data_end, &eth_proto);
	if (!l3)
		return -1;

	// 2. Check if it's IPv4
	if (eth_proto != bpf_htons(ETH_P_IP)) {
		// Not IPv4 - unsupported in demo
		return -1;
	}

	// 3. Parse IPv4 header
	l4 = parse_ipv4((struct iphdr *)l3, data_end, key);
	if (!l4)
		return -1;

	protocol = key->protocol;

	// 4. Parse transport layer based on protocol type
	if (protocol == IPPROTO_TCP) {
		// TCP - don't extract flags (pass NULL)
		if (parse_tcp((struct tcphdr *)l4, data_end, key, NULL) < 0)
			return -1;
	} else if (protocol == IPPROTO_UDP) {
		// UDP
		if (parse_udp((struct udphdr *)l4, data_end, key) < 0)
			return -1;
	} else {
		// ICMP or other protocols - set ports to 0
		key->src_port = 0;
		key->dst_port = 0;
	}

	return 0;
}

#endif /* __FLOW_PROCESSING_DEMO_H__ */
