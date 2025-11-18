# Map Access Optimization Specification

## Capability Overview

Optimize eBPF map access patterns to reduce latency overhead from configuration lookups, stats updates, and cache queries.

## ADDED Requirements

### Requirement: Configuration Caching

The system MUST cache frequently-accessed configuration in program variables to eliminate redundant map lookups.

#### Scenario: Fragment configuration caching

**Given** fragment handling is enabled
**When** processing fragmented packets
**Then** fragment mode MUST be read from program variable, not map
**And** configuration updates MUST invalidate cache
**And** latency overhead MUST be reduced by 3-5μs per fragmented packet

**Implementation:**
```c
// Global config cache (updated by user-space on program reload)
volatile const struct {
    __u8  frag_mode;
    __u8  nat_enabled;
    __u16 reserved;
} config_cache = {
    .frag_mode = FRAG_MODE_NORMAL,
    .nat_enabled = 0,
};

static __always_inline __u8 get_frag_mode(void) {
    return config_cache.frag_mode;
}
```

#### Scenario: NAT configuration caching

**Given** NAT detection is optionally enabled
**When** processing new sessions
**Then** NAT enabled status MUST be read from program variable
**AND** NAT detection MUST be skipped when disabled
**And** latency MUST improve by 5-8μs per new session when NAT disabled

#### Scenario: Configuration update mechanism

**Given** user-space wants to update configuration
**When** configuration change is applied
**Then** eBPF program MUST be reloaded with new config values
**And** in-flight packets MUST not see inconsistent state
**And** update MUST complete within 100ms

---

### Requirement: Selective Feature Checks

The system MUST skip unnecessary feature checks when features are disabled.

#### Scenario: Fragment check bypass

**Given** fragment mode is DISABLED
**When** processing packets
**Then** all fragment detection logic MUST be skipped
**And** 99.9% of packets MUST avoid fragment code path
**And** latency MUST be reduced by 2-3μs per packet

**Implementation:**
```c
// Early bailout for disabled features
if (config_cache.frag_mode == FRAG_MODE_DISABLED) [[likely]] {
    goto skip_fragment_check;
}

// Fragment handling only when enabled
if (is_ipv4_fragment(iph)) {
    // ...
}

skip_fragment_check:
// Continue processing
```

#### Scenario: NAT detection bypass

**Given** NAT support is disabled in configuration
**When** processing new sessions
**Then** conntrack lookup MUST be skipped
**And** original_key MUST equal current key without lookup
**And** latency overhead MUST be eliminated

---

### Requirement: Batch Map Updates

The system MUST combine multiple map update operations where possible.

#### Scenario: Combined stats update

**Given** multiple related statistics need updating
**When** processing packets
**Then** stats MUST be grouped into struct
**And** single map update MUST replace multiple calls
**And** map access count MUST be reduced by at least 40%

**Before:**
```c
update_stats(STATS_TOTAL_PACKETS);
update_stats(STATS_INGRESS_PACKETS);
update_stats(STATS_ALLOWED_PACKETS);
```

**After:**
```c
struct stats_batch {
    __u64 total;
    __u64 ingress;
    __u64 allowed;
};

// Batch update every 16th packet
if ((session->packets & 0xF) == 0) {
    struct stats_batch batch = {1, 1, 1};
    update_stats_batched(&batch);
}
```

#### Scenario: Ring buffer batching

**Given** flow events need submission to user-space
**When** events are generated
**Then** non-critical events MUST be batched
**And** critical events (policy deny) MUST be immediate
**And** ring buffer pressure MUST be reduced

---

## MODIFIED Requirements

### Requirement: Stats Map Architecture

The stats collection architecture MUST be optimized for Per-CPU efficiency.

#### Scenario: Per-CPU array optimization

**Given** global statistics are collected
**When** packets are processed on multiple CPUs
**Then** each CPU MUST have independent stats array
**And** no locks MUST be required
**And** cache line bouncing MUST be minimized

#### Scenario: Stats aggregation

**Given** user-space needs global statistics
**When** stats are queried
**Then** Per-CPU values MUST be summed
**And** aggregation MUST happen in user-space
**And** kernel overhead MUST be zero

---

### Requirement: Fragment State Cache Management

Fragment state caching MUST minimize lookup overhead.

#### Scenario: First fragment caching

**Given** a first fragment is processed
**When** policy decision is made
**Then** fragment state MUST be cached with flow key + policy action
**And** cache entry MUST include timestamp for timeout
**And** cache operation MUST complete in <1μs

#### Scenario: Subsequent fragment lookup

**Given** a subsequent fragment arrives
**When** fragment is processed
**Then** cache lookup MUST complete in <2μs
**And** cache hit MUST return policy without re-matching
**And** cache miss MUST safely deny packet

#### Scenario: Cache timeout cleanup

**Given** fragment entries have timeout configured
**When** cleanup runs
**Then** expired entries MUST be removed
**And** cleanup MUST not block packet processing
**And** memory MUST be reclaimed

---

## Performance Acceptance Criteria

### Requirement: Map Access Latency

Map access operations MUST meet latency targets.

#### Scenario: Configuration read latency

**Given** configuration is cached in program variable
**When** configuration is accessed
**Then** read latency MUST be <100ns (memory access)
**And** no map lookup MUST occur
**And** performance MUST be consistent

#### Scenario: Stats update latency

**Given** stats are updated with batching
**When** batch threshold is not met
**Then** no map access MUST occur
**And** session counters MUST update in <50ns
**And** latency variance MUST be low

---

### Requirement: Map Access Count Reduction

The number of map accesses per packet MUST be minimized.

#### Scenario: Hot path map access count

**Given** packet hits existing session
**When** fast path is executed
**Then** exactly 1 map lookup MUST occur (session lookup)
**And** zero map updates MUST occur on most packets
**And** stats updates MUST be batched

#### Scenario: Slow path map access count

**Given** packet creates new session
**When** slow path is executed
**Then** maximum 4 map operations MUST occur:
  - 1 session lookup (miss)
  - 1 policy lookup
  - 1 session create
  - 1 stats update (batched)

---

## Cache Efficiency Requirements

### Requirement: Cache Line Optimization

Map value structures MUST be optimized for cache line efficiency.

#### Scenario: Single cache line access

**Given** frequently accessed map values
**When** values are read or updated
**Then** all hot fields MUST fit in 64 bytes
**And** cache line splits MUST be avoided
**And** cache misses MUST be minimized

#### Scenario: False sharing prevention

**Given** Per-CPU stats arrays
**When** multiple CPUs access stats
**Then** each CPU's data MUST be cache-line aligned
**And** false sharing MUST not occur
**And** cache coherency traffic MUST be minimized

---

## Monitoring Requirements

### Requirement: Map Access Metrics

The system MUST expose metrics for map access patterns.

#### Scenario: Map operation counting

**Given** the eBPF filter is running
**When** packets are processed
**Then** map lookups per second MUST be tracked
**And** map updates per second MUST be tracked
**And** map access latency distribution MUST be available

#### Scenario: Cache hit rate tracking

**Given** configuration or fragment caching is active
**When** metrics are collected
**Then** cache hit rate MUST be >99%
**And** cache misses MUST be logged
**And** trends MUST be monitorable

---

## Correctness Requirements

### Requirement: Configuration Consistency

Configuration caching MUST not introduce race conditions.

#### Scenario: Atomic configuration updates

**Given** configuration is being updated
**When** eBPF program is reloaded
**Then** old and new config MUST not be mixed
**And** in-flight packets MUST see consistent state
**And** no packets MUST be dropped during update

#### Scenario: Config cache validation

**Given** cached configuration values
**When** program starts or reloads
**Then** cached values MUST match map values
**And** validation MUST occur before packet processing
**And** inconsistencies MUST be detected

---

### Requirement: Stats Accuracy with Batching

Batched statistics MUST maintain acceptable accuracy.

#### Scenario: Batching accuracy bounds

**Given** stats are batched every 16 packets
**When** global stats are collected
**Then** error MUST be <1% of actual count
**And** error MUST be bounded and deterministic
**And** accuracy MUST be documented

#### Scenario: High-rate accuracy

**Given** packet rate >1M pps
**When** stats batching is active
**Then** reported stats MUST still be within 1% accuracy
**And** no integer overflow MUST occur
**And** Per-CPU summing MUST be correct

---

## Migration Requirements

### Requirement: Backward Compatibility

Optimizations MUST maintain compatibility with existing deployments.

#### Scenario: Graceful fallback

**Given** configuration caching is not supported
**When** running on older kernel/compiler
**Then** system MUST fall back to map-based config
**And** functionality MUST be preserved
**And** performance MUST degrade gracefully

#### Scenario: Feature detection

**Given** advanced map features are available
**When** program initializes
**Then** feature support MUST be detected
**And** optimal code path MUST be chosen
**And** unsupported features MUST be disabled

---

## Testing Requirements

### Requirement: Map Access Testing

Optimizations MUST be validated through testing.

#### Scenario: Cache consistency testing

**Given** configuration cache is implemented
**When** stress tested with config changes
**Then** cache MUST remain consistent
**And** no stale values MUST be observed
**And** updates MUST be atomic

#### Scenario: Stats accuracy testing

**Given** batched stats implementation
**When** tested with known packet counts
**Then** reported stats MUST match actual within error bounds
**And** Per-CPU aggregation MUST be correct
**And** edge cases (overflow, etc.) MUST be handled

#### Scenario: Performance regression testing

**Given** map optimizations are implemented
**When** benchmarks are run
**Then** map access count MUST be reduced by >40%
**And** latency MUST improve by >30%
**And** no functional regression MUST occur
