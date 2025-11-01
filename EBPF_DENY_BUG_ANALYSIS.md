# eBPF DENY Policy Bug - Root Cause Analysis

**Date**: 2025-11-01
**Severity**: CRITICAL (Security Vulnerability)
**Status**: Root Cause Identified

---

## Problem Summary

The E2E test `TestE2E_DenyPolicy` fails because **DENY policies do not block traffic**. Traffic is allowed despite an explicit DENY policy being configured and present in the eBPF map.

### Evidence
```
DENY policy added: 10.100.0.1:0 -> 10.100.0.2:8080 proto=tcp action=deny
Policy exists in map: true
Connection result: true (should be false) ❌
Final stats: total=3 allowed=4 denied=0
```

---

## Root Cause

### The Problem: Wildcard Source Port Not Supported

The eBPF policy lookup uses **exact 5-tuple matching**:

```c
// src/bpf/tc_microsegment.bpf.c:115-117
static __always_inline struct policy_value *lookup_policy(struct flow_key *key) {
    struct policy_value *policy = bpf_map_lookup_elem(&policy_map, key);
    // ...
}
```

This performs an **exact hash table lookup** requiring ALL fields to match:
- src_ip
- dst_ip
- src_port ⬅️ **THIS IS THE PROBLEM**
- dst_port
- protocol

### How Policies Are Created

When we create a policy with `SrcPort: 0`, it means "match ANY source port":

```go
// test/e2e/policy_enforcement_test.go:38
denyPolicy := &policy.Policy{
    SrcIP:    "10.100.0.1",
    DstIP:    "10.100.0.2",
    SrcPort:  0,  // ⬅️ Intended to mean "ANY source port"
    DstPort:  8080,
    Protocol: "tcp",
    Action:   "deny",
}
```

This policy gets stored in the eBPF map with key:
```
{src_ip: 10.100.0.1, dst_ip: 10.100.0.2, src_port: 0, dst_port: 8080, proto: 6}
```

### How Real Traffic Works

When a TCP client connects, it uses a **random ephemeral source port** (e.g., 54321):

```
Actual packet flow:
Client: 10.100.0.1:54321 → Server: 10.100.0.2:8080
                  ^^^^^^
                  Random ephemeral port
```

The eBPF program extracts the actual 5-tuple:
```
{src_ip: 10.100.0.1, dst_ip: 10.100.0.2, src_port: 54321, dst_port: 8080, proto: 6}
```

### The Mismatch

```
Policy in map:     src_port = 0
Actual packet:     src_port = 54321
                              ^^^
                              NO MATCH!
```

Since `0 ≠ 54321`, the hash table lookup **fails**. The policy is not found, so the eBPF program uses the **default ALLOW** behavior.

**Result**: Traffic passes through despite the DENY policy.

---

## Why This Happens

### Design Limitation

The current implementation uses a **HASH map** for exact matching:

```c
struct {
    __uint(type, BPF_MAP_TYPE_HASH);  // ⬅️ Exact match only
    __uint(max_entries, MAX_ENTRIES_POLICY);
    __type(key, struct policy_key);
    __type(value, struct policy_value);
} policy_map SEC(".maps");
```

Hash maps require **exact key matches**. There's no concept of "wildcard" or "don't care" fields.

### Why Tests Pass Inconsistently

- `TestE2E_AllowPolicy` ✅ **PASSES** - ALLOW is the default, so even though policy lookup fails, traffic is allowed (correct by accident)
- `TestE2E_DenyPolicy` ❌ **FAILS** - Policy lookup fails, defaults to ALLOW instead of DENY
- `TestE2E_NoPolicy` ✅ **PASSES** - Confirms default is ALLOW
- `TestE2E_PolicyPriority` ✅ **PASSES** - Both policies fail to match, uses default ALLOW

---

## Solutions

### Option 1: Multiple Map Entries (Quick Fix)

For each wildcard src_port policy, create **multiple entries** in the map with common ephemeral ports:

```go
// When SrcPort=0, expand to multiple policies
commonPorts := []uint16{32768, 32769, ..., 65535}  // All ephemeral ports
for _, port := range commonPorts {
    key := policy_key{
        src_ip: policy.SrcIP,
        dst_ip: policy.DstIP,
        src_port: port,  // Specific port
        dst_port: policy.DstPort,
        protocol: policy.Protocol,
    }
    bpf_map_update_elem(&policy_map, &key, &value)
}
```

**Pros**:
- Quick to implement
- No eBPF changes needed

**Cons**:
- Huge memory overhead (32K+ entries per wildcard policy)
- Slow policy updates
- Not scalable

### Option 2: LPM Trie for IP Matching + Linear Search for Ports (Hybrid Approach)

```c
// Use LPM trie for IP prefix matching
struct {
    __uint(type, BPF_MAP_TYPE_LPM_TRIE);
    __type(key, struct ip_prefix_key);
    __type(value, struct port_rules);
} ip_policy_map SEC(".maps");

// port_rules contains array of port rules with wildcards
struct port_rules {
    struct port_rule rules[MAX_RULES_PER_IP];
    __u32 count;
};

struct port_rule {
    __u16 src_port;       // 0 = wildcard
    __u16 src_port_mask;  // 0xFFFF = exact, 0x0000 = wildcard
    __u16 dst_port;
    __u16 dst_port_mask;
    __u8  protocol;
    __u8  action;
};
```

**Pros**:
- Proper wildcard support
- CIDR support for IP ranges
- Reasonable memory usage

**Cons**:
- Requires significant eBPF code changes
- Linear search for ports (slower)
- More complex implementation

### Option 3: Separate Wildcard Map + Priority Lookup (Recommended)

Use two maps: one for exact matches (fast path), one for wildcards (slow path):

```c
// Fast path: Exact 5-tuple match
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, struct flow_key);
    __type(value, struct policy_value);
} exact_policy_map SEC(".maps");

// Slow path: Wildcard matching
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, MAX_WILDCARD_POLICIES);
    __type(key, __u32);  // index
    __type(value, struct wildcard_policy);
} wildcard_policy_map SEC(".maps");

struct wildcard_policy {
    __u32 src_ip;
    __u32 src_ip_mask;  // 0xFFFFFFFF = exact, 0 = any
    __u32 dst_ip;
    __u32 dst_ip_mask;
    __u16 src_port;      // 0 = any
    __u16 dst_port;      // 0 = any
    __u8  protocol;      // 0 = any
    __u8  action;
    __u16 priority;
};
```

**Lookup logic**:
```c
// 1. Try exact match (fast path - most traffic)
policy = bpf_map_lookup_elem(&exact_policy_map, &key);
if (policy) return policy;

// 2. Linear search wildcards (slow path - first packet only)
for (i = 0; i < MAX_WILDCARD_POLICIES; i++) {
    wildcard = bpf_map_lookup_elem(&wildcard_policy_map, &i);
    if (wildcard && matches_wildcard(&key, wildcard)) {
        return wildcard;
    }
}
```

**Pros**:
- Fast path unchanged (exact matches still fast)
- Proper wildcard support
- Moderate complexity
- Works with session caching (slow path only for first packet)

**Cons**:
- Need to maintain two maps
- Linear search for wildcards (but cached in session map)

### Option 4: Session-Based Caching (Current Mitigation)

**Current behavior**: Once a session is created, the policy action is cached in the session map. So even though the first packet lookup fails, subsequent packets use the cached action.

**Problem**: The **first packet** still fails lookup and uses default ALLOW, so the session is created with action=ALLOW instead of DENY.

```c
// SLOW PATH: New session - lookup policy
struct policy_value *policy = lookup_policy(&key);  // ⬅️ Fails for wildcard
__u8 action = POLICY_ACTION_ALLOW;  // ⬅️ Defaults to ALLOW

if (policy) {
    action = policy->action;  // ⬅️ Never reached for wildcards
}

// Create session with wrong action
create_session(&key, action, now, skb->len);  // ⬅️ Caches ALLOW
```

---

## Recommended Fix

Implement **Option 3: Dual Map Approach** with these steps:

### Step 1: Modify eBPF Data Structures

```c
// Add wildcard policy structure
struct wildcard_policy {
    __u32 src_ip;
    __u32 src_ip_mask;
    __u32 dst_ip;
    __u32 dst_ip_mask;
    __u16 src_port;  // 0 = wildcard
    __u16 dst_port;  // 0 = wildcard
    __u8  protocol;  // 0 = wildcard
    __u8  action;
    __u16 priority;
    __u32 rule_id;
} __attribute__((packed));

// Add wildcard policy map
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1000);
    __type(key, __u32);
    __type(value, struct wildcard_policy);
} wildcard_policy_map SEC(".maps");
```

### Step 2: Implement Wildcard Matching

```c
static __always_inline bool matches_wildcard(
    struct flow_key *key,
    struct wildcard_policy *wildcard)
{
    // IP matching with masks
    if ((key->src_ip & wildcard->src_ip_mask) !=
        (wildcard->src_ip & wildcard->src_ip_mask))
        return false;

    if ((key->dst_ip & wildcard->dst_ip_mask) !=
        (wildcard->dst_ip & wildcard->dst_ip_mask))
        return false;

    // Port matching (0 = wildcard)
    if (wildcard->src_port != 0 && key->src_port != wildcard->src_port)
        return false;

    if (wildcard->dst_port != 0 && key->dst_port != wildcard->dst_port)
        return false;

    // Protocol matching (0 = wildcard)
    if (wildcard->protocol != 0 && key->protocol != wildcard->protocol)
        return false;

    return true;
}

static __always_inline struct policy_value *lookup_policy_with_wildcard(
    struct flow_key *key)
{
    // Fast path: Try exact match first
    struct policy_value *policy = bpf_map_lookup_elem(&policy_map, key);
    if (policy) {
        policy->hit_count += 1;
        update_stats(STATS_POLICY_HITS);
        return policy;
    }

    // Slow path: Linear search wildcards
    #pragma unroll
    for (int i = 0; i < MAX_WILDCARD_POLICIES; i++) {
        __u32 idx = i;
        struct wildcard_policy *wildcard =
            bpf_map_lookup_elem(&wildcard_policy_map, &idx);

        if (!wildcard || wildcard->rule_id == 0)
            continue;  // Empty slot

        if (matches_wildcard(key, wildcard)) {
            update_stats(STATS_POLICY_HITS);
            // Note: Can't increment hit_count easily, would need helper
            return (struct policy_value*)wildcard;  // Cast for compatibility
        }
    }

    update_stats(STATS_POLICY_MISSES);
    return NULL;
}
```

### Step 3: Update Policy Manager

```go
// pkg/policy/policy.go
func (pm *PolicyManager) AddPolicy(p *Policy) error {
    // Check if this is a wildcard policy
    hasWildcard := p.SrcPort == 0 || p.SrcIP == "0.0.0.0/0" || ...

    if hasWildcard {
        // Add to wildcard map
        return pm.addWildcardPolicy(p)
    } else {
        // Add to exact match map (current behavior)
        return pm.addPolicyToMap(p)
    }
}

func (pm *PolicyManager) addWildcardPolicy(p *Policy) error {
    // Find empty slot in wildcard array
    for i := 0; i < MAX_WILDCARD_POLICIES; i++ {
        var existing WildcardPolicy
        key := uint32(i)
        err := pm.wildcardMap.Lookup(&key, &existing)

        if err != nil || existing.RuleID == 0 {
            // Empty slot found
            wildcard := WildcardPolicy{
                SrcIP:      ipToUint32(srcIP),
                SrcIPMask:  maskToUint32(srcMask),
                DstIP:      ipToUint32(dstIP),
                DstIPMask:  maskToUint32(dstMask),
                SrcPort:    p.SrcPort,
                DstPort:    p.DstPort,
                Protocol:   proto,
                Action:     action,
                Priority:   p.Priority,
                RuleID:     p.RuleID,
            }
            return pm.wildcardMap.Put(&key, &wildcard)
        }
    }

    return fmt.Errorf("wildcard policy map full")
}
```

---

## Implementation Plan

### Phase 1: Quick Validation (1-2 hours)
1. ✅ Confirm root cause with debug script
2. ✅ Document the issue
3. Add test with **exact src_port** to verify eBPF enforcement works

### Phase 2: Implement Wildcard Support (4-6 hours)
1. Add wildcard map to eBPF code
2. Implement wildcard matching logic
3. Update policy manager to route policies to correct map
4. Test with both exact and wildcard policies

### Phase 3: Optimize (2-3 hours)
1. Add priority-based sorting for wildcards
2. Optimize linear search with early termination
3. Add metrics for exact vs wildcard lookups
4. Performance testing

---

## Testing Strategy

### Test 1: Exact Match (Should Work Now)

```go
denyPolicy := &policy.Policy{
    SrcIP:    "10.100.0.1",
    DstIP:    "10.100.0.2",
    SrcPort:  12345,  // ⬅️ Exact port (not wildcard)
    DstPort:  8080,
    Protocol: "tcp",
    Action:   "deny",
}
```

This should block traffic from src_port=12345 (need to force client to use this port).

### Test 2: Wildcard After Fix

```go
denyPolicy := &policy.Policy{
    SrcIP:    "10.100.0.1",
    DstIP:    "10.100.0.2",
    SrcPort:  0,  // ⬅️ Wildcard (any port)
    DstPort:  8080,
    Protocol: "tcp",
    Action:   "deny",
}
```

After implementing wildcard support, this should block ALL traffic to dst_port=8080.

---

## Impact Assessment

### Current State
- ❌ **CRITICAL**: DENY policies with wildcard src_port do NOT work
- ❌ Microsegmentation cannot block traffic
- ❌ Security vulnerability in production
- ✅ ALLOW policies "work" (by accident - they're the default)
- ✅ eBPF program logic is correct (just lookup fails)

### After Fix
- ✅ Wildcard policies will work correctly
- ✅ DENY enforcement will function
- ✅ Microsegmentation will provide actual security
- ⚠️ Slight performance impact for wildcard lookups (first packet only)

---

## Alternative: Temporary Workaround for E2E Tests

Update E2E tests to use **destination IP-based policies** instead of port-based:

```go
// Instead of: deny traffic to port 8080
// Use: deny all traffic between these IPs

denyPolicy := &policy.Policy{
    SrcIP:    "10.100.0.1",
    DstIP:    "10.100.0.2",
    SrcPort:  0,
    DstPort:  0,  // ⬅️ Match any dst_port too
    Protocol: "tcp",
    Action:   "deny",
}
```

This still won't work because src_port is still a wildcard! The only way to make current tests work is to **force the client to use a specific src_port** or implement wildcard support.

---

## Conclusion

**Root Cause**: eBPF hash map requires exact 5-tuple match. Wildcard src_port (value=0) doesn't match actual ephemeral ports.

**Impact**: All DENY policies with wildcard src_port fail to match, defaulting to ALLOW.

**Fix**: Implement dual-map approach with wildcard support (4-6 hours).

**Workaround**: None practical without code changes.

**Priority**: CRITICAL - This is a security vulnerability that prevents the core microsegmentation functionality from working.

---

**Analysis by**: Claude (AI Assistant)
**Date**: 2025-11-01
