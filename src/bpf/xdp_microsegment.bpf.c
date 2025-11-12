// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* XDP eBPF program for microsegmentation with session tracking
 *
 * 这个 XDP 程序在网卡驱动层处理数据包,提供最高性能的网络策略执行
 * 与 TC 程序共享策略数据 (通过 Map Pinning),确保策略一致性
 */

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>
#include <bpf/bpf_endian.h>

// XDP action codes
#define XDP_ABORTED 0  // 错误,丢弃数据包
#define XDP_DROP 1     // 丢弃数据包
#define XDP_PASS 2     // 允许数据包继续传递到网络栈
#define XDP_TX 3       // 从同一网卡发送数据包
#define XDP_REDIRECT 4 // 重定向到其他网卡

// Ethernet protocol types
#define ETH_P_IP 0x0800

// Debug mode - disable for production to reduce latency
#define DEBUG_MODE 0

#include "headers/common_types.h"

char LICENSE[] SEC("license") = "GPL";

// ========== eBPF Maps 定义 ==========

// Session tracking map - LRU_HASH for automatic eviction
// NOTE: XDP 和 TC 各自维护独立的会话表,不使用 pinning
struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__uint(max_entries, MAX_ENTRIES_SESSION);
	__type(key, struct flow_key);
	__type(value, struct session_value);
} session_map SEC(".maps");

// Policy map for exact 5-tuple matching
// PINNED: XDP 和 TC 共享策略数据
struct {
	__uint(type, BPF_MAP_TYPE_HASH);
	__uint(max_entries, MAX_ENTRIES_POLICY);
	__type(key, struct policy_key);
	__type(value, struct policy_value);
	__uint(pinning, LIBBPF_PIN_BY_NAME);  // 按名称固定到 /sys/fs/bpf/
} policy_map SEC(".maps");

// Wildcard policy map for policies with wildcards (0 = any)
// Uses ARRAY for linear search (slower but supports wildcards)
// PINNED: XDP 和 TC 共享策略数据
struct {
	__uint(type, BPF_MAP_TYPE_ARRAY);
	__uint(max_entries, MAX_ENTRIES_WILDCARD_POLICY);
	__type(key, __u32);  // index
	__type(value, struct wildcard_policy);
	__uint(pinning, LIBBPF_PIN_BY_NAME);  // 按名称固定到 /sys/fs/bpf/
} wildcard_policy_map SEC(".maps");

// Statistics map (Per-CPU for lock-free updates)
// PINNED: XDP 和 TC 共享统计数据
struct {
	__uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
	__uint(max_entries, STATS_MAX);
	__type(key, __u32);
	__type(value, __u64);
	__uint(pinning, LIBBPF_PIN_BY_NAME);  // 按名称固定到 /sys/fs/bpf/
} stats_map SEC(".maps");

// Ring buffer for flow events to user-space
// NOTE: XDP 有自己独立的 ring buffer (与 TC 分开)
struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 256 * 1024);  // 256KB ring buffer
} flow_events SEC(".maps");

// ========== Helper Functions ==========

// Helper: Update statistics counter (optimized - no error checking for speed)
static __always_inline void update_stats(__u32 key) {
	__u64 *count = bpf_map_lookup_elem(&stats_map, &key);
	if (count) {
		// Direct increment for per-CPU array (no atomic needed)
		*count += 1;
	}
}

// Include shared policy matching logic
#include "headers/policy_match.h"

// Include shared flow processing logic
#include "headers/flow_processing.h"

// Helper: Get current timestamp in nanoseconds
static __always_inline __u64 get_timestamp_ns() {
	return bpf_ktime_get_ns();
}

// Helper: Extract flow key from packet (XDP-specific wrapper)
// XDP 使用 struct xdp_md,提供 data 和 data_end 指针
static __always_inline int extract_flow_key(struct xdp_md *ctx, struct flow_key *key) {
	void *data = (void *)(long)ctx->data;
	void *data_end = (void *)(long)ctx->data_end;

	// 调用通用的流键提取函数 (在 flow_processing.h 中定义)
	return extract_flow_key_from_packet(data, data_end, key);
}

// ========== Main XDP Program ==========

SEC("xdp")
int xdp_microsegment_prog(struct xdp_md *ctx) {
	struct flow_key key = {0};

	// 1. 提取流键 (五元组)
	if (extract_flow_key(ctx, &key) < 0) {
		// 解析失败 (非 IPv4 或其他协议) - 放行
		return XDP_PASS;
	}

	// 更新总数据包计数
	update_stats(STATS_TOTAL_PACKETS);

	// 2. 查询会话表 (快速路径)
	struct session_value *session = bpf_map_lookup_elem(&session_map, &key);

	if (session) {
		// ===== HOT PATH: 已存在会话 =====
		// 使用缓存的策略决策,这是最性能关键的路径 (>99% 的数据包)

		__u8 action = session->policy_action;

		// 更新会话统计
		__u64 now = get_timestamp_ns();
		session->last_seen_ts = now;
		session->packets_to_server += 1;

		// 执行策略动作
		if (action == POLICY_ACTION_DENY) {
			update_stats(STATS_DENIED_PACKETS);
#if DEBUG_MODE
			bpf_printk("XDP DENY: %pI4:%d -> %pI4:%d (cached)\n",
				   &key.src_ip, bpf_ntohs(key.src_port),
				   &key.dst_ip, bpf_ntohs(key.dst_port));
#endif
			return XDP_DROP;  // 丢弃数据包
		}

		update_stats(STATS_ALLOWED_PACKETS);
		return XDP_PASS;  // 允许数据包
	}

	// ===== SLOW PATH: 新会话 =====
	// 需要查询策略,这发生频率较低

	__u64 now = get_timestamp_ns();
	__u32 matched_rule_id = 0;

	// 3. 查询策略 (使用共享的策略匹配逻辑)
	// XDP 只能在 ingress 方向运行,所以方向固定为 INGRESS
	__u8 action = lookup_policy_action(&key, POLICY_DIR_INGRESS, &matched_rule_id);

#if DEBUG_MODE
	if (matched_rule_id != 0) {
		bpf_printk("XDP Policy %d matched: %pI4:%d -> %pI4:%d action=%d\n",
			   matched_rule_id,
			   &key.src_ip, bpf_ntohs(key.src_port),
			   &key.dst_ip, bpf_ntohs(key.dst_port),
			   action);
	}
#endif

	// 4. 创建新会话
	struct session_value new_session = {
		.created_ts = now,
		.last_seen_ts = now,
		.packets_to_server = 1,
		.packets_to_client = 0,
		.bytes_to_server = 0,  // XDP 不容易获取数据包长度
		.bytes_to_client = 0,
		.state = SESSION_STATE_NEW,
		.tcp_state = TCP_STATE_CLOSED,
		.policy_action = action,
		.flags = 0,
	};

	// 插入会话表
	int ret = bpf_map_update_elem(&session_map, &key, &new_session, BPF_NOEXIST);
	if (ret == 0) {
		update_stats(STATS_NEW_SESSIONS);
		// TODO: 推送流事件到 ring buffer (Task 3.3)
	}

	// 5. 执行策略动作
	if (action == POLICY_ACTION_DENY) {
		update_stats(STATS_DENIED_PACKETS);
#if DEBUG_MODE
		bpf_printk("XDP DENY: %pI4:%d -> %pI4:%d (new)\n",
			   &key.src_ip, bpf_ntohs(key.src_port),
			   &key.dst_ip, bpf_ntohs(key.dst_port));
#endif
		return XDP_DROP;  // 丢弃数据包
	}

	update_stats(STATS_ALLOWED_PACKETS);
	return XDP_PASS;  // 允许数据包
}
