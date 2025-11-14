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

/* parse_ethernet - 解析以太网头
 *
 * @data: 数据包起始指针
 * @data_end: 数据包结束指针
 * @eth_proto: 输出参数 - 以太网协议类型 (网络字节序)
 *
 * 返回: IP 头起始指针,失败返回 NULL
 *
 * 功能:
 * 1. 验证以太网头边界
 * 2. 提取协议类型 (h_proto)
 * 3. 返回 IP 层起始位置
 */
static __always_inline void *parse_ethernet(
	void *data,
	void *data_end,
	__u16 *eth_proto)
{
	struct ethhdr *eth = data;

	// 边界检查: 确保有足够空间读取以太网头
	if ((void *)(eth + 1) > data_end)
		return NULL;

	// 提取协议类型 (已经是网络字节序)
	*eth_proto = eth->h_proto;

	// 返回下一层协议的起始位置
	return (void *)(eth + 1);
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
 * 2. 提取源/目标 IP 地址
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

	// 提取 IP 地址 (网络字节序)
	key->src_ip = iph->saddr;
	key->dst_ip = iph->daddr;
	key->protocol = iph->protocol;

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

/* extract_flow_key_from_packet - 从原始数据包提取流键
 *
 * @data: 数据包起始指针
 * @data_end: 数据包结束指针
 * @key: 输出参数 - 提取的流键 (5-tuple)
 *
 * 返回: 0 成功, -1 失败
 *
 * 功能:
 * 1. 解析以太网头,验证是 IPv4
 * 2. 解析 IPv4 头,提取 IP 地址和协议
 * 3. 根据协议类型解析传输层 (TCP/UDP)
 * 4. 对于其他协议 (ICMP 等),端口设置为 0
 *
 * 支持的协议:
 * - IPv4 over Ethernet
 * - TCP (IPPROTO_TCP = 6)
 * - UDP (IPPROTO_UDP = 17)
 * - ICMP 和其他协议 (端口为 0)
 *
 * 不支持:
 * - IPv6 (未来可扩展)
 * - VLAN 标签 (未来可扩展)
 * - 分片包 (需要上层处理)
 */
static __always_inline int extract_flow_key_from_packet(
	void *data,
	void *data_end,
	struct flow_key *key)
{
	__u16 eth_proto;
	void *l3;
	void *l4;

	// 1. 解析以太网头
	l3 = parse_ethernet(data, data_end, &eth_proto);
	if (!l3)
		return -1;

	// 2. 仅处理 IPv4
	// ETH_P_IP = 0x0800,需要转换为网络字节序
	if (eth_proto != bpf_htons(ETH_P_IP))
		return -1;

	// 3. 解析 IPv4 头
	l4 = parse_ipv4((struct iphdr *)l3, data_end, key);
	if (!l4)
		return -1;

	// 4. 根据协议类型解析传输层
	if (key->protocol == IPPROTO_TCP) {
		// TCP - 不提取标志 (传 NULL)
		if (parse_tcp((struct tcphdr *)l4, data_end, key, NULL) < 0)
			return -1;
	} else if (key->protocol == IPPROTO_UDP) {
		// UDP
		if (parse_udp((struct udphdr *)l4, data_end, key) < 0)
			return -1;
	} else {
		// ICMP 或其他协议 - 端口设置为 0
		key->src_port = 0;
		key->dst_port = 0;
	}

	return 0;
}

#endif /* __FLOW_PROCESSING_H__ */
