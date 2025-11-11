// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* 策略匹配逻辑 - 共享头文件
 *
 * 这个头文件包含策略匹配的核心逻辑,可以被 TC 和 XDP 程序共享使用
 *
 * 主要功能:
 * - matches_wildcard(): 检查流是否匹配通配符策略
 * - lookup_policy_action(): 执行策略查找 (精确匹配 + 通配符匹配)
 *
 * 前置要求 (必须在包含此头文件之前完成):
 * 1. 包含 common_types.h 定义基础类型
 * 2. 定义以下 eBPF Maps:
 *    - policy_map: 精确匹配策略表 (HASH map)
 *    - wildcard_policy_map: 通配符策略表 (ARRAY map)
 * 3. 定义 update_stats() 函数用于更新统计计数
 */

#ifndef __POLICY_MATCH_H__
#define __POLICY_MATCH_H__

/* matches_wildcard - 检查流是否匹配通配符策略
 *
 * @key: 流的五元组键
 * @wildcard: 通配符策略规则
 *
 * 返回: true 如果流匹配策略,否则 false
 *
 * 匹配逻辑:
 * 1. 源 IP 匹配: (key.src_ip & mask) == (wildcard.src_ip & mask)
 * 2. 目标 IP 匹配: (key.dst_ip & mask) == (wildcard.dst_ip & mask)
 * 3. 源端口匹配: port == 0 (任意) 或精确匹配
 * 4. 目标端口匹配: port == 0 (任意) 或精确匹配
 * 5. 协议匹配: protocol == 0 (任意) 或精确匹配
 */
static __always_inline bool matches_wildcard(
	struct flow_key *key,
	struct wildcard_policy *wildcard)
{
	// 源 IP 匹配检查
	// 使用掩码支持 CIDR 范围匹配 (例如 192.168.1.0/24)
	if ((key->src_ip & wildcard->src_ip_mask) !=
	    (wildcard->src_ip & wildcard->src_ip_mask))
		return false;

	// 目标 IP 匹配检查
	if ((key->dst_ip & wildcard->dst_ip_mask) !=
	    (wildcard->dst_ip & wildcard->dst_ip_mask))
		return false;

	// 源端口匹配 (0 表示匹配任意端口)
	if (wildcard->src_port != 0 && key->src_port != wildcard->src_port)
		return false;

	// 目标端口匹配 (0 表示匹配任意端口)
	if (wildcard->dst_port != 0 && key->dst_port != wildcard->dst_port)
		return false;

	// 协议匹配 (0 表示匹配任意协议)
	if (wildcard->protocol != 0 && key->protocol != wildcard->protocol)
		return false;

	return true;
}

/* lookup_policy_action - 查找流的策略动作
 *
 * @key: 流的五元组键
 * @rule_id: 输出参数 - 匹配的规则 ID
 *
 * 返回: 策略动作 (POLICY_ACTION_ALLOW 或 POLICY_ACTION_DENY)
 *
 * 查找策略:
 * 1. 快速路径: 精确匹配 - O(1) hash 查找 policy_map
 * 2. 慢速路径: 通配符匹配 - 线性扫描 wildcard_policy_map
 *    - 如果有多个匹配,选择优先级最高的
 * 3. 默认策略: 如果都不匹配,允许通过 (POLICY_ACTION_ALLOW)
 */
static __always_inline __u8 lookup_policy_action(struct flow_key *key, __u32 *rule_id)
{
	// ===== 快速路径: 精确匹配 =====
	// O(1) hash 查找,处理绝大多数常见情况
	struct policy_value *policy = bpf_map_lookup_elem(&policy_map, key);
	if (policy) {
		// 更新命中计数 (用于策略使用统计)
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
	// 最多支持 100 条通配符策略
	#pragma unroll
	for (__u32 i = 0; i < 100; i++) {
		__u32 idx = i;
		if (idx >= MAX_ENTRIES_WILDCARD_POLICY)
			break;

		wildcard = bpf_map_lookup_elem(&wildcard_policy_map, &idx);

		if (!wildcard)
			continue;

		// 跳过空槽位 (rule_id == 0 表示未使用)
		if (wildcard->rule_id == 0)
			continue;

		// 检查是否匹配
		if (!matches_wildcard(key, wildcard))
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
