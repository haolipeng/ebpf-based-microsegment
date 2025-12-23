// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
//
// input: TCP header flags (SYN/ACK/FIN/RST), current session state
// output: updated TCP state, connection status (established/closed)
// pos: bpf/headers - TCP FSM for stateful connection tracking
//
/* TCP 状态机实现
 *
 * 实现标准 TCP 状态转换逻辑,用于准确追踪 TCP 连接生命周期
 *
 * TCP 状态转换图:
 *
 * 客户端视角:
 *   CLOSED -> SYN_SENT (发送 SYN) -> ESTABLISHED (收到 SYN+ACK, 发送 ACK)
 *   ESTABLISHED -> FIN_WAIT1 (发送 FIN) -> FIN_WAIT2 (收到 ACK)
 *   FIN_WAIT2 -> TIME_WAIT (收到 FIN, 发送 ACK)
 *
 * 服务端视角:
 *   CLOSED -> SYN_RECV (收到 SYN, 发送 SYN+ACK) -> ESTABLISHED (收到 ACK)
 *   ESTABLISHED -> CLOSE_WAIT (收到 FIN, 发送 ACK) -> LAST_ACK (发送 FIN)
 *   LAST_ACK -> CLOSED (收到 ACK)
 *
 * 同时关闭:
 *   ESTABLISHED -> FIN_WAIT1 (发送 FIN) -> CLOSING (收到 FIN, 发送 ACK)
 *   CLOSING -> TIME_WAIT (收到 ACK)
 *
 * RST 处理:
 *   任何状态 + RST -> CLOSED
 *
 * 前置要求:
 * - 包含 common_types.h (定义 enum tcp_state)
 * - 包含 vmlinux.h (定义 struct tcphdr)
 */

#ifndef __TCP_STATE_MACHINE_H__
#define __TCP_STATE_MACHINE_H__

// TCP 标志位定义
#define TCP_FLAG_FIN  0x01
#define TCP_FLAG_SYN  0x02
#define TCP_FLAG_RST  0x04
#define TCP_FLAG_PSH  0x08
#define TCP_FLAG_ACK  0x10
#define TCP_FLAG_URG  0x20

/* 从 tcphdr 提取 TCP 标志位
 *
 * @tcph: TCP 头指针
 * 返回: 8位标志字节 (FIN=0x01, SYN=0x02, RST=0x04, ACK=0x10 等)
 */
static __always_inline __u8 get_tcp_flags(struct tcphdr *tcph)
{
	__u8 flags = 0;

	if (tcph->fin) flags |= TCP_FLAG_FIN;
	if (tcph->syn) flags |= TCP_FLAG_SYN;
	if (tcph->rst) flags |= TCP_FLAG_RST;
	if (tcph->psh) flags |= TCP_FLAG_PSH;
	if (tcph->ack) flags |= TCP_FLAG_ACK;
	if (tcph->urg) flags |= TCP_FLAG_URG;

	return flags;
}

/* TCP 状态机转换 - 客户端方向 (发出的数据包)
 *
 * @current_state: 当前 TCP 状态
 * @flags: TCP 标志位
 * 返回: 新的 TCP 状态
 *
 * 处理从客户端发出的数据包引起的状态转换
 */
static __always_inline __u8 tcp_state_transition_outbound(
	__u8 current_state,
	__u8 flags)
{
	// RST 标志 - 直接关闭连接
	if (flags & TCP_FLAG_RST) {
		return TCP_STATE_CLOSED;
	}

	switch (current_state) {
	case TCP_STATE_CLOSED:
		// CLOSED -> SYN_SENT: 发送 SYN (建立连接)
		if (flags & TCP_FLAG_SYN) {
			return TCP_STATE_SYN_SENT;
		}
		break;

	case TCP_STATE_SYN_SENT:
		// SYN_SENT -> ESTABLISHED: 收到 SYN+ACK 后发送 ACK
		// 注意: 这个转换通常由入站数据包触发,但为了完整性保留
		if ((flags & (TCP_FLAG_SYN | TCP_FLAG_ACK)) == (TCP_FLAG_SYN | TCP_FLAG_ACK)) {
			// 同时发起连接的情况: SYN_SENT + (SYN+ACK) -> SYN_RECV
			return TCP_STATE_SYN_RECV;
		}
		break;

	case TCP_STATE_ESTABLISHED:
		// ESTABLISHED -> FIN_WAIT1: 主动关闭,发送 FIN
		if (flags & TCP_FLAG_FIN) {
			return TCP_STATE_FIN_WAIT1;
		}
		break;

	case TCP_STATE_FIN_WAIT1:
		// FIN_WAIT1 -> CLOSING: 收到对方 FIN (同时关闭)
		// 注意: 这个转换通常由入站数据包触发
		if (flags & TCP_FLAG_FIN) {
			return TCP_STATE_CLOSING;
		}
		break;

	case TCP_STATE_CLOSE_WAIT:
		// CLOSE_WAIT -> LAST_ACK: 被动关闭,发送 FIN
		if (flags & TCP_FLAG_FIN) {
			return TCP_STATE_LAST_ACK;
		}
		break;

	default:
		// 其他状态保持不变
		break;
	}

	return current_state;
}

/* TCP 状态机转换 - 服务端方向 (接收的数据包)
 *
 * @current_state: 当前 TCP 状态
 * @flags: TCP 标志位
 * 返回: 新的 TCP 状态
 *
 * 处理从服务端接收的数据包引起的状态转换
 */
static __always_inline __u8 tcp_state_transition_inbound(
	__u8 current_state,
	__u8 flags)
{
	// RST 标志 - 直接关闭连接
	if (flags & TCP_FLAG_RST) {
		return TCP_STATE_CLOSED;
	}

	switch (current_state) {
	case TCP_STATE_CLOSED:
		// CLOSED -> SYN_RECV: 收到 SYN (被动打开)
		if (flags & TCP_FLAG_SYN) {
			return TCP_STATE_SYN_RECV;
		}
		break;

	case TCP_STATE_SYN_SENT:
		// SYN_SENT -> ESTABLISHED: 收到 SYN+ACK
		if ((flags & (TCP_FLAG_SYN | TCP_FLAG_ACK)) == (TCP_FLAG_SYN | TCP_FLAG_ACK)) {
			return TCP_STATE_ESTABLISHED;
		}
		// SYN_SENT -> SYN_RECV: 同时发起连接,收到 SYN
		if ((flags & TCP_FLAG_SYN) && !(flags & TCP_FLAG_ACK)) {
			return TCP_STATE_SYN_RECV;
		}
		break;

	case TCP_STATE_SYN_RECV:
		// SYN_RECV -> ESTABLISHED: 收到 ACK (三次握手完成)
		if ((flags & TCP_FLAG_ACK) && !(flags & TCP_FLAG_FIN)) {
			return TCP_STATE_ESTABLISHED;
		}
		break;

	case TCP_STATE_ESTABLISHED:
		// ESTABLISHED -> CLOSE_WAIT: 收到 FIN (被动关闭)
		if (flags & TCP_FLAG_FIN) {
			return TCP_STATE_CLOSE_WAIT;
		}
		break;

	case TCP_STATE_FIN_WAIT1:
		// FIN_WAIT1 -> FIN_WAIT2: 收到 ACK (对方确认我方 FIN)
		if ((flags & TCP_FLAG_ACK) && !(flags & TCP_FLAG_FIN)) {
			return TCP_STATE_FIN_WAIT2;
		}
		// FIN_WAIT1 -> CLOSING: 收到 FIN (同时关闭)
		if (flags & TCP_FLAG_FIN) {
			return TCP_STATE_CLOSING;
		}
		// FIN_WAIT1 -> TIME_WAIT: 收到 FIN+ACK (快速关闭)
		if ((flags & (TCP_FLAG_FIN | TCP_FLAG_ACK)) == (TCP_FLAG_FIN | TCP_FLAG_ACK)) {
			return TCP_STATE_TIME_WAIT;
		}
		break;

	case TCP_STATE_FIN_WAIT2:
		// FIN_WAIT2 -> TIME_WAIT: 收到 FIN
		if (flags & TCP_FLAG_FIN) {
			return TCP_STATE_TIME_WAIT;
		}
		break;

	case TCP_STATE_CLOSING:
		// CLOSING -> TIME_WAIT: 收到 ACK (同时关闭完成)
		if (flags & TCP_FLAG_ACK) {
			return TCP_STATE_TIME_WAIT;
		}
		break;

	case TCP_STATE_LAST_ACK:
		// LAST_ACK -> CLOSED: 收到 ACK (被动关闭完成)
		if (flags & TCP_FLAG_ACK) {
			return TCP_STATE_CLOSED;
		}
		break;

	default:
		// 其他状态保持不变 (TIME_WAIT, CLOSE_WAIT)
		break;
	}

	return current_state;
}

/* 简化的 TCP 状态转换 (不区分方向)
 *
 * @current_state: 当前 TCP 状态
 * @flags: TCP 标志位
 * @is_outbound: true 表示出站数据包, false 表示入站数据包
 * 返回: 新的 TCP 状态
 *
 * 根据数据包方向调用相应的状态转换函数
 */
static __always_inline __u8 tcp_state_transition(
	__u8 current_state,
	__u8 flags,
	bool is_outbound)
{
	if (is_outbound) {
		return tcp_state_transition_outbound(current_state, flags);
	} else {
		return tcp_state_transition_inbound(current_state, flags);
	}
}

/* 检查 TCP 连接是否正在关闭或已关闭
 *
 * @tcp_state: TCP 状态
 * 返回: true 表示连接正在关闭或已关闭
 */
static __always_inline bool is_tcp_state_closing(__u8 tcp_state)
{
	return (tcp_state == TCP_STATE_FIN_WAIT1 ||
		tcp_state == TCP_STATE_FIN_WAIT2 ||
		tcp_state == TCP_STATE_CLOSE_WAIT ||
		tcp_state == TCP_STATE_CLOSING ||
		tcp_state == TCP_STATE_LAST_ACK ||
		tcp_state == TCP_STATE_TIME_WAIT ||
		tcp_state == TCP_STATE_CLOSED);
}

/* 检查 TCP 连接是否已建立
 *
 * @tcp_state: TCP 状态
 * 返回: true 表示连接已建立
 */
static __always_inline bool is_tcp_state_established(__u8 tcp_state)
{
	return (tcp_state == TCP_STATE_ESTABLISHED);
}

#endif /* __TCP_STATE_MACHINE_H__ */
