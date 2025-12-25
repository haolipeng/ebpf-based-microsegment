// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Enhanced Policy Matching with Identity Support
 *
 * This header provides identity-first policy matching:
 * 1. Query IPCache for source/destination identities
 * 2. Try identity-based policy matching
 * 3. Fall back to IP-based policy matching
 *
 * Prerequisites:
 * - Include common_types.h
 * - Include ipcache.h and identity_policy.h
 * - Include policy_match.h
 */

#ifndef __POLICY_MATCH_IDENTITY_H__
#define __POLICY_MATCH_IDENTITY_H__

// Feature flag to enable/disable identity-based policy matching
// Can be overridden via -D flag during compilation
#ifndef ENABLE_IDENTITY_POLICY
#define ENABLE_IDENTITY_POLICY 1
#endif

#if ENABLE_IDENTITY_POLICY

/* lookup_policy_action_with_identity - Identity-first policy lookup
 *
 * @key: Flow 5-tuple key
 * @direction: Packet direction (POLICY_DIR_INGRESS or POLICY_DIR_EGRESS)
 * @rule_id: Output parameter - matched rule ID
 *
 * Returns: Policy action (POLICY_ACTION_ALLOW, POLICY_ACTION_DENY, or POLICY_ACTION_LOG)
 *
 * Matching order:
 * 1. IPCache lookup for source and destination identities
 * 2. If both have identities, try identity-based policy
 * 3. If no identity match, fall back to IP-based policy
 */
static __always_inline __u8 lookup_policy_action_with_identity(
	struct flow_key *key,
	__u8 direction,
	__u32 *rule_id)
{
	__u32 src_identity = IDENTITY_UNKNOWN;
	__u32 dst_identity = IDENTITY_UNKNOWN;
	int identity_result;

	// Step 1: Lookup source identity from IPCache
	src_identity = ipcache_lookup(key->src_ip, key->ip_version);

	// Step 2: Lookup destination identity from IPCache
	dst_identity = ipcache_lookup(key->dst_ip, key->ip_version);

	// Step 3: If both have valid identities, try identity-based policy
	if (src_identity != IDENTITY_UNKNOWN && dst_identity != IDENTITY_UNKNOWN) {
		// Try identity policy match
		identity_result = match_identity_policy(
			src_identity,
			dst_identity,
			key->dst_port,
			key->protocol
		);

		if (identity_result > 0) {
			// Identity policy matched
			// identity_result: 1 = allow, 2 = deny
			*rule_id = 0; // TODO: Return actual rule_id from identity policy
			return (identity_result == 1) ? POLICY_ACTION_ALLOW : POLICY_ACTION_DENY;
		}
	}

	// Step 4: Fall back to IP-based policy matching
	return lookup_policy_action(key, direction, rule_id);
}

#else /* !ENABLE_IDENTITY_POLICY */

/* When identity policy is disabled, directly use IP-based policy */
static __always_inline __u8 lookup_policy_action_with_identity(
	struct flow_key *key,
	__u8 direction,
	__u32 *rule_id)
{
	return lookup_policy_action(key, direction, rule_id);
}

#endif /* ENABLE_IDENTITY_POLICY */

/* Reserved identity checks - useful for special handling */

/* is_host_traffic - Check if traffic is to/from the host
 *
 * @src_id: Source identity
 * @dst_id: Destination identity
 *
 * Returns: true if either endpoint is the host
 */
static __always_inline bool is_host_traffic(__u32 src_id, __u32 dst_id)
{
	return src_id == IDENTITY_HOST || dst_id == IDENTITY_HOST;
}

/* is_world_traffic - Check if traffic is to/from the world (external)
 *
 * @src_id: Source identity
 * @dst_id: Destination identity
 *
 * Returns: true if either endpoint is the world
 */
static __always_inline bool is_world_traffic(__u32 src_id, __u32 dst_id)
{
	return src_id == IDENTITY_WORLD || dst_id == IDENTITY_WORLD;
}

/* is_health_check - Check if traffic is a health check
 *
 * @src_id: Source identity
 * @dst_id: Destination identity
 *
 * Returns: true if either endpoint is a health check endpoint
 */
static __always_inline bool is_health_check(__u32 src_id, __u32 dst_id)
{
	return src_id == IDENTITY_HEALTH || dst_id == IDENTITY_HEALTH;
}

/* should_skip_policy - Check if traffic should bypass policy enforcement
 *
 * @src_id: Source identity
 * @dst_id: Destination identity
 *
 * Returns: true if traffic should be allowed without policy check
 *
 * This function can be customized based on requirements:
 * - Health checks typically bypass policy
 * - Host traffic may need special handling
 */
static __always_inline bool should_skip_policy(__u32 src_id, __u32 dst_id)
{
	// Health checks always allowed
	if (is_health_check(src_id, dst_id))
		return true;

	// Host traffic uses normal policy (can be changed if needed)
	// if (is_host_traffic(src_id, dst_id))
	//     return true;

	return false;
}

#endif /* __POLICY_MATCH_IDENTITY_H__ */
