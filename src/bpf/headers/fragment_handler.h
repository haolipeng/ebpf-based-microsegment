// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Fragment Handler - Core fragment processing logic
 *
 * This header file implements the fragment processing workflow:
 * 1. Detect if packet is fragmented (IPv4/IPv6)
 * 2. For first fragment: Extract 5-tuple, match policy, cache result
 * 3. For subsequent fragments: Look up cached policy, apply action
 * 4. Handle different fragment modes (STRICT/NORMAL/PERMISSIVE)
 * 5. Update fragment statistics
 *
 * Prerequisites:
 * - Include fragment_tracking.h for data structures
 * - Include policy_lookup.h for policy matching functions
 * - Define fragment maps in the main BPF program
 */

#ifndef __FRAGMENT_HANDLER_H__
#define __FRAGMENT_HANDLER_H__

#include "fragment_tracking.h"
#include "common_types.h"

/* Fragment processing result codes */
#define FRAG_RESULT_NOT_FRAGMENT    0  // Not a fragment, continue normal processing
#define FRAG_RESULT_FIRST_FRAGMENT  1  // First fragment detected, processed
#define FRAG_RESULT_SUBSEQUENT_OK   2  // Subsequent fragment, allowed
#define FRAG_RESULT_DENIED          3  // Fragment denied by policy or mode
#define FRAG_RESULT_ERROR           4  // Processing error

/* update_frag_stats - Update fragment statistics
 *
 * @stats_map: Fragment statistics map pointer
 * @key: Statistics key (enum frag_stats_key)
 *
 * Atomically increments the per-CPU counter for the given statistic.
 */
static __always_inline void update_frag_stats(
	void *stats_map,
	__u32 key)
{
	if (!stats_map)
		return;

	__u64 *value = bpf_map_lookup_elem(stats_map, &key);
	if (value) {
		__sync_fetch_and_add(value, 1);
	}
}

/* handle_ipv4_fragment - Process IPv4 fragmented packet
 *
 * @iph: IPv4 header pointer
 * @data_end: End of packet data
 * @flow_key: Flow key extracted from packet (may be incomplete for fragments)
 * @policy_action: Output - policy action to apply
 * @frag_state_map: Fragment state tracking map
 * @frag_config_map: Fragment configuration map
 * @frag_stats_map: Fragment statistics map
 *
 * Returns:
 *   FRAG_RESULT_* code indicating how to proceed
 *
 * Processing Logic:
 * 1. Check if packet is a fragment
 * 2. If first fragment:
 *    - Extract complete 5-tuple (has L4 headers)
 *    - Match against policy (caller provides action)
 *    - Cache flow key and policy action
 *    - Return FIRST_FRAGMENT (caller decides based on mode)
 * 3. If subsequent fragment:
 *    - Look up cached policy from first fragment
 *    - Apply cached policy action
 *    - Return SUBSEQUENT_OK or DENIED
 * 4. Update statistics
 */
static __always_inline int handle_ipv4_fragment(
	struct iphdr *iph,
	void *data_end,
	struct flow_key *flow_key,
	__u8 *policy_action,
	void *frag_state_map,
	void *frag_config_map,
	void *frag_stats_map)
{
	// Check if packet is fragmented
	if (!is_ipv4_fragment(iph)) {
		return FRAG_RESULT_NOT_FRAGMENT;
	}

	// Update IPv4 fragment statistics
	__u32 stat_key = FRAG_STAT_IPV4_FRAGMENTS;
	update_frag_stats(frag_stats_map, stat_key);

	// Extract fragment key for cache lookup
	struct frag_key fkey = {0};
	extract_ipv4_frag_key(iph, &fkey);

	// Get fragment configuration
	struct frag_config *config = NULL;
	__u32 config_key = 0;
	if (frag_config_map) {
		config = bpf_map_lookup_elem(frag_config_map, &config_key);
	}

	// Default to NORMAL mode if no config
	__u8 mode = config ? config->mode : FRAG_MODE_NORMAL;

	// STRICT mode: Deny all fragments
	if (mode == FRAG_MODE_STRICT) {
		*policy_action = POLICY_ACTION_DENY;
		stat_key = FRAG_STAT_FRAGMENTS_DENIED;
		update_frag_stats(frag_stats_map, stat_key);
		return FRAG_RESULT_DENIED;
	}

	// Check if this is the first fragment
	if (is_ipv4_first_fragment(iph)) {
		// First fragment: has L4 headers, complete 5-tuple available
		// Policy action is already determined by caller
		// Cache the flow key and policy action for subsequent fragments

		stat_key = FRAG_STAT_FIRST_FRAGMENTS;
		update_frag_stats(frag_stats_map, stat_key);

		if (frag_state_map && flow_key) {
			struct frag_value fval = {0};

			// Copy complete flow key
			__builtin_memcpy(&fval.complete_key, flow_key, sizeof(struct flow_key));

			// Store policy action (provided by caller after policy lookup)
			fval.policy_action = *policy_action;

			// Timestamp for timeout
			fval.timestamp = bpf_ktime_get_ns();

			// Update fragment state map
			bpf_map_update_elem(frag_state_map, &fkey, &fval, BPF_ANY);
		}

		// Return FIRST_FRAGMENT, caller will decide based on policy action
		if (*policy_action == POLICY_ACTION_ALLOW) {
			stat_key = FRAG_STAT_FRAGMENTS_ALLOWED;
			update_frag_stats(frag_stats_map, stat_key);
			return FRAG_RESULT_FIRST_FRAGMENT;
		} else {
			stat_key = FRAG_STAT_FRAGMENTS_DENIED;
			update_frag_stats(frag_stats_map, stat_key);
			return FRAG_RESULT_DENIED;
		}
	}

	// Subsequent fragment: no L4 headers, look up cached policy
	if (is_ipv4_subsequent_fragment(iph)) {
		stat_key = FRAG_STAT_SUBSEQUENT_FRAGMENTS;
		update_frag_stats(frag_stats_map, stat_key);

		// Look up fragment state
		struct frag_value *fval = NULL;
		if (frag_state_map) {
			fval = bpf_map_lookup_elem(frag_state_map, &fkey);
		}

		if (fval) {
			// Cache hit: use cached policy action
			stat_key = FRAG_STAT_CACHE_HITS;
			update_frag_stats(frag_stats_map, stat_key);

			*policy_action = fval->policy_action;

			// In NORMAL mode: deny subsequent fragments
			// In PERMISSIVE mode: allow subsequent fragments if first was allowed
			if (mode == FRAG_MODE_NORMAL) {
				*policy_action = POLICY_ACTION_DENY;
				stat_key = FRAG_STAT_FRAGMENTS_DENIED;
				update_frag_stats(frag_stats_map, stat_key);
				return FRAG_RESULT_DENIED;
			} else if (mode == FRAG_MODE_PERMISSIVE) {
				if (fval->policy_action == POLICY_ACTION_ALLOW) {
					stat_key = FRAG_STAT_FRAGMENTS_ALLOWED;
					update_frag_stats(frag_stats_map, stat_key);
					return FRAG_RESULT_SUBSEQUENT_OK;
				} else {
					stat_key = FRAG_STAT_FRAGMENTS_DENIED;
					update_frag_stats(frag_stats_map, stat_key);
					return FRAG_RESULT_DENIED;
				}
			}
		} else {
			// Cache miss: first fragment not seen or timed out
			stat_key = FRAG_STAT_CACHE_MISSES;
			update_frag_stats(frag_stats_map, stat_key);

			// In NORMAL/PERMISSIVE mode without first fragment: deny for safety
			*policy_action = POLICY_ACTION_DENY;
			stat_key = FRAG_STAT_FRAGMENTS_DENIED;
			update_frag_stats(frag_stats_map, stat_key);
			return FRAG_RESULT_DENIED;
		}
	}

	return FRAG_RESULT_ERROR;
}

/* handle_ipv6_fragment - Process IPv6 fragmented packet
 *
 * @ip6h: IPv6 base header pointer
 * @data_end: End of packet data
 * @nexthdr: Next header value (from IPv6 base header or extension header)
 * @flow_key: Flow key extracted from packet (may be incomplete for fragments)
 * @policy_action: Output - policy action to apply
 * @frag_state_map: Fragment state tracking map
 * @frag_config_map: Fragment configuration map
 * @frag_stats_map: Fragment statistics map
 *
 * Returns:
 *   FRAG_RESULT_* code indicating how to proceed
 *
 * Note: This implementation assumes the fragment header immediately
 * follows the IPv6 base header. Full implementation should walk
 * through all extension headers to find the fragment header.
 */
static __always_inline int handle_ipv6_fragment(
	struct ipv6hdr *ip6h,
	void *data_end,
	__u8 nexthdr,
	struct flow_key *flow_key,
	__u8 *policy_action,
	void *frag_state_map,
	void *frag_config_map,
	void *frag_stats_map)
{
	// Check if packet has fragment extension header
	if (!is_ipv6_fragment(nexthdr)) {
		return FRAG_RESULT_NOT_FRAGMENT;
	}

	// Update IPv6 fragment statistics
	__u32 stat_key = FRAG_STAT_IPV6_FRAGMENTS;
	update_frag_stats(frag_stats_map, stat_key);

	// Parse fragment extension header
	struct ipv6_frag_hdr *frag_hdr = (struct ipv6_frag_hdr *)(ip6h + 1);
	if ((void *)(frag_hdr + 1) > data_end) {
		return FRAG_RESULT_ERROR;
	}

	// Extract fragment key for cache lookup
	struct frag_key fkey = {0};
	extract_ipv6_frag_key(ip6h, frag_hdr, &fkey);

	// Get fragment configuration
	struct frag_config *config = NULL;
	__u32 config_key = 0;
	if (frag_config_map) {
		config = bpf_map_lookup_elem(frag_config_map, &config_key);
	}

	// Default to NORMAL mode if no config
	__u8 mode = config ? config->mode : FRAG_MODE_NORMAL;

	// STRICT mode: Deny all fragments
	if (mode == FRAG_MODE_STRICT) {
		*policy_action = POLICY_ACTION_DENY;
		stat_key = FRAG_STAT_FRAGMENTS_DENIED;
		update_frag_stats(frag_stats_map, stat_key);
		return FRAG_RESULT_DENIED;
	}

	// Check if this is the first fragment
	if (is_ipv6_first_fragment(frag_hdr)) {
		// First fragment: has L4 headers, complete 5-tuple available

		stat_key = FRAG_STAT_FIRST_FRAGMENTS;
		update_frag_stats(frag_stats_map, stat_key);

		if (frag_state_map && flow_key) {
			struct frag_value fval = {0};

			// Copy complete flow key
			__builtin_memcpy(&fval.complete_key, flow_key, sizeof(struct flow_key));

			// Store policy action
			fval.policy_action = *policy_action;

			// Timestamp for timeout
			fval.timestamp = bpf_ktime_get_ns();

			// Update fragment state map
			bpf_map_update_elem(frag_state_map, &fkey, &fval, BPF_ANY);
		}

		// Return based on policy action
		if (*policy_action == POLICY_ACTION_ALLOW) {
			stat_key = FRAG_STAT_FRAGMENTS_ALLOWED;
			update_frag_stats(frag_stats_map, stat_key);
			return FRAG_RESULT_FIRST_FRAGMENT;
		} else {
			stat_key = FRAG_STAT_FRAGMENTS_DENIED;
			update_frag_stats(frag_stats_map, stat_key);
			return FRAG_RESULT_DENIED;
		}
	}

	// Subsequent fragment: no L4 headers, look up cached policy
	if (is_ipv6_subsequent_fragment(frag_hdr)) {
		stat_key = FRAG_STAT_SUBSEQUENT_FRAGMENTS;
		update_frag_stats(frag_stats_map, stat_key);

		// Look up fragment state
		struct frag_value *fval = NULL;
		if (frag_state_map) {
			fval = bpf_map_lookup_elem(frag_state_map, &fkey);
		}

		if (fval) {
			// Cache hit
			stat_key = FRAG_STAT_CACHE_HITS;
			update_frag_stats(frag_stats_map, stat_key);

			*policy_action = fval->policy_action;

			// In NORMAL mode: deny subsequent fragments
			// In PERMISSIVE mode: allow subsequent fragments if first was allowed
			if (mode == FRAG_MODE_NORMAL) {
				*policy_action = POLICY_ACTION_DENY;
				stat_key = FRAG_STAT_FRAGMENTS_DENIED;
				update_frag_stats(frag_stats_map, stat_key);
				return FRAG_RESULT_DENIED;
			} else if (mode == FRAG_MODE_PERMISSIVE) {
				if (fval->policy_action == POLICY_ACTION_ALLOW) {
					stat_key = FRAG_STAT_FRAGMENTS_ALLOWED;
					update_frag_stats(frag_stats_map, stat_key);
					return FRAG_RESULT_SUBSEQUENT_OK;
				} else {
					stat_key = FRAG_STAT_FRAGMENTS_DENIED;
					update_frag_stats(frag_stats_map, stat_key);
					return FRAG_RESULT_DENIED;
				}
			}
		} else {
			// Cache miss
			stat_key = FRAG_STAT_CACHE_MISSES;
			update_frag_stats(frag_stats_map, stat_key);

			// Deny without first fragment
			*policy_action = POLICY_ACTION_DENY;
			stat_key = FRAG_STAT_FRAGMENTS_DENIED;
			update_frag_stats(frag_stats_map, stat_key);
			return FRAG_RESULT_DENIED;
		}
	}

	return FRAG_RESULT_ERROR;
}

#endif /* __FRAGMENT_HANDLER_H__ */
