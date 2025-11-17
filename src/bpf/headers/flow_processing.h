// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* 流处理逻辑 - 共享头文件
 *
 * 这个头文件包含数据包解析和流键提取的核心逻辑,可以被 TC 和 XDP 程序共享使用
 *
 * 主要功能:
 * - parse_ethernet(): 解析以太网头
 * - parse_ipv4(): 解析 IPv4 头
 * - parse_tcp(): 解析 TCP 头
 * - parse_udp(): 解析 UDP 头
 * - extract_flow_key_from_packet(): 从原始数据包提取流键
 *
 * 前置要求 (必须在包含此头文件之前完成):
 * 1. 包含 common_types.h 定义基础类型 (struct flow_key)
 * 2. 包含 vmlinux.h 或相关头文件定义网络协议结构
 */

#ifndef __FLOW_PROCESSING_H__
#define __FLOW_PROCESSING_H__

// VLAN 802.1Q and 802.1ad constants
#define ETH_P_8021Q  0x8100  // 802.1Q VLAN Extended Header
#define ETH_P_8021AD 0x88A8  // 802.1ad Service VLAN tag identifier (QinQ)

// Note: vlan_hdr is already defined in vmlinux.h, so we don't redefine it

/* parse_ethernet - Parse Ethernet header (with VLAN support)
 *
 * @data: Packet start pointer
 * @data_end: Packet end pointer
 * @eth_proto: Output parameter - Ethernet protocol type (network byte order)
 * @vlan_id: Output parameter - VLAN ID (0 if no VLAN)
 *
 * Returns: IP header start pointer, NULL on failure
 *
 * Functionality:
 * 1. Validate Ethernet header boundary
 * 2. Extract protocol type (h_proto)
 * 3. Handle VLAN tags (802.1Q and 802.1ad/QinQ)
 * 4. Return IP layer start position
 */
static __always_inline void *parse_ethernet(
	void *data,
	void *data_end,
	__u16 *eth_proto,
	__u16 *vlan_id)
{
	struct ethhdr *eth = data;
	void *next_header = (void *)(eth + 1);

	// Boundary check: ensure enough space to read Ethernet header
	if (next_header > data_end)
		return NULL;

	// Extract protocol type (network byte order)
	*eth_proto = eth->h_proto;
	*vlan_id = 0;  // Default: no VLAN

	// Check for VLAN tags (802.1Q or 802.1ad)
	if (*eth_proto == bpf_htons(ETH_P_8021Q) ||
	    *eth_proto == bpf_htons(ETH_P_8021AD)) {
		struct vlan_hdr *vlan = next_header;

		// Boundary check for VLAN header
		if ((void *)(vlan + 1) > data_end)
			return NULL;

		// Extract VLAN ID (lower 12 bits of TCI)
		*vlan_id = bpf_ntohs(vlan->h_vlan_TCI) & 0x0FFF;

		// Update protocol and next header position
		*eth_proto = vlan->h_vlan_encapsulated_proto;
		next_header = (void *)(vlan + 1);

		// Handle QinQ (802.1ad with nested 802.1Q)
		// Only extract outer VLAN ID for now
		if (*eth_proto == bpf_htons(ETH_P_8021Q)) {
			struct vlan_hdr *inner_vlan = next_header;

			// Boundary check for inner VLAN header
			if ((void *)(inner_vlan + 1) > data_end)
				return NULL;

			// For QinQ, we keep the outer VLAN ID
			// (Could be extended to track both outer and inner)
			*eth_proto = inner_vlan->h_vlan_encapsulated_proto;
			next_header = (void *)(inner_vlan + 1);
		}
	}

	// Return next layer protocol start position
	return next_header;
}

/* ipv4_to_ipv4_mapped_ipv6 - Convert IPv4 address to IPv4-mapped IPv6 format
 *
 * @ipv4: IPv4 address in network byte order (32 bits)
 * @ipv6: Output parameter - IPv6 address array (128 bits = 4 x 32 bits)
 *
 * Functionality:
 * Converts IPv4 address to IPv4-mapped IPv6 format (::ffff:a.b.c.d)
 * Result: [0, 0, 0xffff0000, ipv4_addr] in host byte order
 */
static __always_inline void ipv4_to_ipv4_mapped_ipv6(__u32 ipv4, __u32 ipv6[4])
{
	ipv6[0] = 0;
	ipv6[1] = 0;
	ipv6[2] = bpf_htonl(0x0000ffff);  // ::ffff prefix in network byte order
	ipv6[3] = ipv4;  // IPv4 address (already in network byte order)
}

/* parse_ipv4 - 解析 IPv4 头
 *
 * @iph: IPv4 头指针
 * @data_end: 数据包结束指针
 * @key: 输出参数 - 流键,填充源/目标 IP 和协议字段
 *
 * 返回: 传输层起始指针,失败返回 NULL
 *
 * 功能:
 * 1. 验证 IPv4 头边界
 * 2. 提取源/目标 IP 地址 (转换为 IPv4-mapped IPv6 格式)
 * 3. 提取协议类型 (TCP/UDP/ICMP 等)
 * 4. 计算传输层起始位置 (考虑 IP 选项)
 */
static __always_inline void *parse_ipv4(
	struct iphdr *iph,
	void *data_end,
	struct flow_key *key)
{
	// 边界检查: 确保有足够空间读取基本 IP 头 (20 字节)
	if ((void *)(iph + 1) > data_end)
		return NULL;

	// 提取 IP 地址并转换为 IPv4-mapped IPv6 格式
	ipv4_to_ipv4_mapped_ipv6(iph->saddr, key->src_ip);
	ipv4_to_ipv4_mapped_ipv6(iph->daddr, key->dst_ip);
	key->protocol = iph->protocol;
	key->ip_version = 4;

	// 计算传输层起始位置
	// iph->ihl 是 IP 头长度,单位是 4 字节
	// 例如: ihl=5 表示 20 字节 (无选项), ihl=6 表示 24 字节 (有选项)
	void *l4 = (void *)iph + (iph->ihl * 4);

	// 边界检查: 确保传输层起始位置在数据包范围内
	if (l4 > data_end)
		return NULL;

	return l4;
}

/* parse_tcp - 解析 TCP 头
 *
 * @tcph: TCP 头指针
 * @data_end: 数据包结束指针
 * @key: 输出参数 - 流键,填充源/目标端口字段
 * @tcp_flags: 输出参数 - TCP 标志 (可选,传 NULL 表示不需要)
 *
 * 返回: 0 成功, -1 失败
 *
 * 功能:
 * 1. 验证 TCP 头边界
 * 2. 提取源/目标端口 (网络字节序)
 * 3. 提取 TCP 标志 (FIN, SYN, RST, ACK 等)
 */
static __always_inline int parse_tcp(
	struct tcphdr *tcph,
	void *data_end,
	struct flow_key *key,
	__u8 *tcp_flags)
{
	// 边界检查: 确保有足够空间读取基本 TCP 头 (20 字节)
	if ((void *)(tcph + 1) > data_end)
		return -1;

	// 提取端口 (网络字节序)
	key->src_port = tcph->source;
	key->dst_port = tcph->dest;

	// 提取 TCP 标志 (如果需要)
	if (tcp_flags) {
		// 从 tcph 中提取标志位
		// FIN=0x01, SYN=0x02, RST=0x04, ACK=0x10 等
		*tcp_flags = 0;
		if (tcph->fin) *tcp_flags |= 0x01;
		if (tcph->syn) *tcp_flags |= 0x02;
		if (tcph->rst) *tcp_flags |= 0x04;
		if (tcph->psh) *tcp_flags |= 0x08;
		if (tcph->ack) *tcp_flags |= 0x10;
	}

	return 0;
}

/* parse_udp - 解析 UDP 头
 *
 * @udph: UDP 头指针
 * @data_end: 数据包结束指针
 * @key: 输出参数 - 流键,填充源/目标端口字段
 *
 * 返回: 0 成功, -1 失败
 *
 * 功能:
 * 1. 验证 UDP 头边界
 * 2. 提取源/目标端口 (网络字节序)
 */
static __always_inline int parse_udp(
	struct udphdr *udph,
	void *data_end,
	struct flow_key *key)
{
	// 边界检查: 确保有足够空间读取 UDP 头 (8 字节)
	if ((void *)(udph + 1) > data_end)
		return -1;

	// 提取端口 (网络字节序)
	key->src_port = udph->source;
	key->dst_port = udph->dest;

	return 0;
}

/* parse_ipv6 - Parse IPv6 header
 *
 * @ip6h: IPv6 header pointer
 * @data_end: End of packet data pointer
 * @key: Output parameter - flow key, fills source/destination IP and protocol fields
 * @nexthdr: Output parameter - next header type (for extension headers)
 *
 * Returns: Transport layer start pointer, NULL on failure
 *
 * Functionality:
 * 1. Validate IPv6 header boundary
 * 2. Extract source/destination IPv6 addresses (128 bits)
 * 3. Extract next header type (TCP/UDP or extension header)
 * 4. Calculate transport layer start position
 *
 * Note: This basic implementation does not handle IPv6 extension headers.
 * Extension headers (Hop-by-Hop, Routing, Fragment, etc.) are not processed.
 * The nexthdr field will contain the immediate next header value.
 */
static __always_inline void *parse_ipv6(
	struct ipv6hdr *ip6h,
	void *data_end,
	struct flow_key *key,
	__u8 *nexthdr)
{
	// Boundary check: ensure enough space to read basic IPv6 header (40 bytes)
	if ((void *)(ip6h + 1) > data_end)
		return NULL;

	// Extract IPv6 addresses (128 bits = 4 x 32 bits)
	// ipv6hdr uses in6_addr which contains __u32 s6_addr32[4]
	#pragma unroll
	for (int i = 0; i < 4; i++) {
		key->src_ip[i] = ip6h->saddr.in6_u.u6_addr32[i];
		key->dst_ip[i] = ip6h->daddr.in6_u.u6_addr32[i];
	}

	// Extract next header and set IP version
	*nexthdr = ip6h->nexthdr;
	key->ip_version = 6;

	// Transport layer starts immediately after IPv6 header (40 bytes)
	// Note: This does not handle extension headers
	void *l4 = (void *)(ip6h + 1);

	// Boundary check: ensure transport layer start is within packet
	if (l4 > data_end)
		return NULL;

	return l4;
}

/* extract_flow_key_from_packet - Extract flow key from raw packet (IPv4/IPv6 + VLAN)
 *
 * @data: Packet start pointer
 * @data_end: Packet end pointer
 * @key: Output parameter - extracted flow key (5-tuple + VLAN)
 *
 * Returns: 0 on success, -1 on failure
 *
 * Functionality:
 * 1. Parse Ethernet header, validate IPv4 or IPv6 (with VLAN support)
 * 2. Parse IP header, extract IP addresses and protocol
 * 3. Parse transport layer based on protocol type (TCP/UDP)
 * 4. For other protocols (ICMP etc.), set ports to 0
 *
 * Supported:
 * - IPv4 over Ethernet (ETH_P_IP = 0x0800)
 * - IPv6 over Ethernet (ETH_P_IPV6 = 0x86DD)
 * - 802.1Q VLAN (ETH_P_8021Q = 0x8100)
 * - 802.1ad QinQ (ETH_P_8021AD = 0x88A8)
 * - TCP (IPPROTO_TCP = 6)
 * - UDP (IPPROTO_UDP = 17)
 * - ICMPv4/ICMPv6 and other protocols (ports = 0)
 *
 * Not supported:
 * - Fragmented packets (need upper layer handling)
 * - IPv6 extension headers (future enhancement)
 */
static __always_inline int extract_flow_key_from_packet(
	void *data,
	void *data_end,
	struct flow_key *key)
{
	__u16 eth_proto;
	__u16 vlan_id;
	void *l3;
	void *l4;
	__u8 protocol;

	// 1. Parse Ethernet header (with VLAN support)
	l3 = parse_ethernet(data, data_end, &eth_proto, &vlan_id);
	if (!l3)
		return -1;

	// Store VLAN ID in flow key
	key->vlan_id = vlan_id;

	// 2. 根据以太网类型处理 IPv4 或 IPv6
	if (eth_proto == bpf_htons(ETH_P_IP)) {
		// IPv4 处理
		l4 = parse_ipv4((struct iphdr *)l3, data_end, key);
		if (!l4)
			return -1;
		protocol = key->protocol;

	} else if (eth_proto == bpf_htons(ETH_P_IPV6)) {
		// IPv6 处理
		__u8 nexthdr;
		l4 = parse_ipv6((struct ipv6hdr *)l3, data_end, key, &nexthdr);
		if (!l4)
			return -1;

		// For IPv6, nexthdr indicates the next header type
		// Note: This does not handle extension headers
		protocol = nexthdr;
		key->protocol = protocol;

	} else {
		// 不支持的以太网协议类型 (VLAN, ARP, etc.)
		return -1;
	}

	// 3. 根据协议类型解析传输层
	if (protocol == IPPROTO_TCP) {
		// TCP - 不提取标志 (传 NULL)
		if (parse_tcp((struct tcphdr *)l4, data_end, key, NULL) < 0)
			return -1;
	} else if (protocol == IPPROTO_UDP) {
		// UDP
		if (parse_udp((struct udphdr *)l4, data_end, key) < 0)
			return -1;
	} else {
		// ICMP/ICMPv6 或其他协议 - 端口设置为 0
		key->src_port = 0;
		key->dst_port = 0;
	}

	return 0;
}

#endif /* __FLOW_PROCESSING_H__ */
