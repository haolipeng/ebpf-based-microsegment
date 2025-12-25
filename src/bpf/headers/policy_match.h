// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
//
// input: flow_key, policy maps (exact IP + CIDR/wildcard)
// output: matched policy action (allow/deny), rule_id
// pos: bpf/headers - policy matching logic shared by TC and XDP programs
//
/* 策略匹配逻辑 - 共享头文件
 *
 * 这个头文件包含策略匹配的核心逻辑,可以被 TC 和 XDP 程序共享使用
 *
 * 主要功能:
 * - matches_wildcard(): 检查流是否匹配通配符策略
 * - lookup_policy_action(): 执行策略查找 (精确 IP 匹配 + CIDR/通配符匹配)
 *
 * 前置要求 (必须在包含此头文件之前完成):
 * 1. 包含 common_types.h 定义基础类型
 * 2. 定义以下 eBPF Maps:
 *    - ipaddr_policy_map: 精确 IP 地址匹配策略表 (HASH map)
 *    - ipcidr_policy_map: CIDR/通配符策略表 (ARRAY map)
 * 3. 定义 update_stats() 函数用于更新统计计数
 */

#ifndef __POLICY_MATCH_H__
#define __POLICY_MATCH_H__

/* Maximum number of wildcard policy entries to scan
 * Reduced from 100 to 50 to improve performance
 * This value should be aligned with server-side compact storage implementation
 */
#define MAX_WILDCARD_LOOP 50

/* matches_wildcard - 检查流是否匹配通配符策略 (IPv4/IPv6 support)
 *
 * @key: 流的五元组键
 * @wildcard: 通配符策略规则
 * @direction: 数据包方向 (POLICY_DIR_INGRESS 或 POLICY_DIR_EGRESS)
 *
 * 返回: true 如果流匹配策略,否则 false
 *
 * 匹配逻辑:
 * 1. 源 IP 匹配: (key.src_ip & mask) == (wildcard.src_ip & mask) (逐 32位比较)
 * 2. 目标 IP 匹配: (key.dst_ip & mask) == (wildcard.dst_ip & mask) (逐 32位比较)
 * 3. 源端口匹配: port == 0 (任意) 或精确匹配
 * 4. 目标端口匹配: port == 0 (任意) 或精确匹配
 * 5. 协议匹配: protocol == 0 (任意) 或精确匹配
 * 6. 方向匹配: direction == ANY 或匹配当前方向
 */
static __always_inline bool matches_wildcard(
	struct flow_key *key,
	struct wildcard_policy *wildcard,
	__u8 direction)
{
	// 方向匹配检查
	// POLICY_DIR_ANY (0) 匹配所有方向
	if (wildcard->direction != POLICY_DIR_ANY &&
	    wildcard->direction != direction)
		return false;

	// 源 IP 匹配检查 (IPv6 format: 4 x 32-bit words)
	// 使用掩码支持 CIDR 范围匹配
	#pragma unroll
	for (int i = 0; i < 4; i++) {
		if ((key->src_ip[i] & wildcard->src_ip_mask[i]) !=
		    (wildcard->src_ip[i] & wildcard->src_ip_mask[i]))
			return false;
	}

	// 目标 IP 匹配检查 (IPv6 format: 4 x 32-bit words)
	#pragma unroll
	for (int i = 0; i < 4; i++) {
		if ((key->dst_ip[i] & wildcard->dst_ip_mask[i]) !=
		    (wildcard->dst_ip[i] & wildcard->dst_ip_mask[i]))
			return false;
	}

	// 源端口匹配 (0 表示匹配任意端口)
	if (wildcard->src_port != 0 && key->src_port != wildcard->src_port)
		return false;

	// 目标端口匹配 (0 表示匹配任意端口)
	if (wildcard->dst_port != 0 && key->dst_port != wildcard->dst_port)
		return false;

	// 协议匹配 (0 表示匹配任意协议)
	if (wildcard->protocol != 0 && key->protocol != wildcard->protocol)
		return false;

	// VLAN ID 匹配 (0 表示匹配任意 VLAN)
	if (wildcard->vlan_id != 0 && key->vlan_id != wildcard->vlan_id)
		return false;

	return true;
}

/* lookup_policy_action - 查找流的策略动作
 *
 * @key: 流的五元组键
 * @direction: 数据包方向 (POLICY_DIR_INGRESS 或 POLICY_DIR_EGRESS)
 * @rule_id: 输出参数 - 匹配的规则 ID
 *
 * 返回: 策略动作 (POLICY_ACTION_ALLOW 或 POLICY_ACTION_DENY)
 *
 * 查找策略 (支持方向感知):
 * 1. 快速路径: 精确匹配 - O(1) hash 查找 policy_map
 *    a. 先尝试匹配方向特定的策略 (direction=INGRESS/EGRESS)
 *    b. 如果没有匹配,回退到双向策略 (direction=ANY)
 * 2. 慢速路径: 通配符匹配 - 线性扫描 wildcard_policy_map
 *    - 如果有多个匹配,选择优先级最高的
 * 3. 默认策略: 如果都不匹配,允许通过 (POLICY_ACTION_ALLOW)
 */
static __always_inline __u8 lookup_policy_action(
	struct flow_key *key,
	__u8 direction,
	__u32 *rule_id)
{
	// ===== 快速路径: 精确匹配 =====
	// O(1) hash 查找,处理绝大多数常见情况

	// 构造 policy_key (包含方向、VLAN，IPv4/IPv6 support)
	struct policy_key pkey = {
		.src_port = key->src_port,
		.dst_port = key->dst_port,
		.protocol = key->protocol,
		.direction = direction,  // 先尝试方向特定的策略
		.ip_version = key->ip_version,
		.vlan_id = key->vlan_id,
		.pad = 0,
		.pad2 = 0,
	};

	// Copy IP addresses (4 x 32-bit words for IPv6)
	#pragma unroll
	for (int i = 0; i < 4; i++) {
		pkey.src_ip[i] = key->src_ip[i];
		pkey.dst_ip[i] = key->dst_ip[i];
	}

	// 1. 尝试匹配方向特定的策略 (INGRESS 或 EGRESS)
	struct policy_value *policy = bpf_map_lookup_elem(&ipaddr_policy_map, &pkey);
	if (policy) {
		// 更新命中计数 (用于策略使用统计)
		policy->hit_count += 1;
		update_stats(STATS_POLICY_HITS);

		*rule_id = policy->rule_id;
		return policy->action;
	}

	// 2. 如果没有匹配,回退到双向策略 (direction=ANY)
	pkey.direction = POLICY_DIR_ANY;
	policy = bpf_map_lookup_elem(&ipaddr_policy_map, &pkey);
	if (policy) {
		policy->hit_count += 1;
		update_stats(STATS_POLICY_HITS);

		*rule_id = policy->rule_id;
		return policy->action;
	}

	// ===== 慢速路径: 通配符匹配 =====
	// 线性扫描所有通配符策略,支持 IP 范围、端口范围等复杂规则
	struct wildcard_policy *wildcard;
	struct wildcard_policy *best_match = NULL;
	__u8 best_priority = 0;

	// 使用 #pragma unroll 展开循环,满足 eBPF 验证器要求
	// 最多扫描 MAX_WILDCARD_LOOP (50) 条通配符策略
	#pragma unroll
	for (__u32 i = 0; i < MAX_WILDCARD_LOOP; i++) {
		__u32 idx = i;
		if (idx >= MAX_ENTRIES_WILDCARD_POLICY)
			break;

		wildcard = bpf_map_lookup_elem(&ipcidr_policy_map, &idx);

		if (!wildcard)
			continue;

		// 早停优化: 遇到空槽位立即停止扫描
		// 假设策略紧凑存储(由 Server 端保证),空槽位后没有有效策略
		if (wildcard->rule_id == 0)
			break;

		// 检查是否匹配 (包含方向匹配)
		if (!matches_wildcard(key, wildcard, direction))
			continue;

		// 如果匹配,选择优先级最高的策略
		// 优先级越大越优先 (例如: priority=100 > priority=50)
		if (!best_match || wildcard->priority > best_priority) {
			best_match = wildcard;
			best_priority = wildcard->priority;
		}
	}

	// 如果找到通配符匹配
	if (best_match) {
		update_stats(STATS_POLICY_HITS);
		*rule_id = best_match->rule_id;
		return best_match->action;
	}

	// ===== 默认策略: 允许 =====
	// 如果没有任何策略匹配,默认允许通过
	// 这是"默认允许"策略,适合内部网络环境
	// 如果需要"默认拒绝",可以修改为 POLICY_ACTION_DENY
	update_stats(STATS_POLICY_MISSES);
	*rule_id = 0;
	return POLICY_ACTION_ALLOW;
}

#endif /* __POLICY_MATCH_H__ */
