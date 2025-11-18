# Hot Path Optimization Specification

## Capability Overview

Optimize the eBPF data plane hot path (existing session processing) to achieve <10μs latency overhead for 99% of packets.

## ADDED Requirements

### Requirement: Early Return Pattern

The eBPF filter MUST minimize instructions executed before session map lookup.

#### Scenario: Fast session lookup for existing flows

**Given** a packet arrives for an existing session
**When** the eBPF filter processes the packet
**Then** the session lookup MUST occur within the first 10 instructions
**And** no unnecessary stats updates MUST precede the lookup
**And** direction detection MUST be deferred to post-lookup processing

#### Scenario: Session cache hit handling

**Given** a session exists in the session map
**When** the session lookup succeeds
**Then** the filter MUST return the policy action within 60 total instructions
**And** stats updates MUST be batched where possible
**And** TCP state updates MUST be conditional on protocol type

---

### Requirement: Batched Statistics Updates

The system MUST reduce per-packet statistics overhead through batching.

#### Scenario: Global stats batching

**Given** packets are being processed at high rate
**When** updating global statistics
**Then** updates MUST occur only every Nth packet (configurable, default 16)
**And** session-level counters MUST be updated on every packet
**And** statistical accuracy MUST be within 1% of actual values

#### Scenario: Stats accuracy verification

**Given** a benchmark with 1,000,000 packets
**When** global stats are collected
**Then** the reported count MUST be within 10,000 of actual (1% error)
**And** throughput MUST improve by at least 20% compared to per-packet updates

---

### Requirement: Inline Function Optimization

The system MUST use strategic inline directives to optimize code path performance.

#### Scenario: Always inline for hot functions

**Given** a function is called in the critical hot path
**When** the function has fewer than 20 instructions
**Then** it MUST be marked `__always_inline`
**And** it MUST be inlined at all call sites
**And** verifier complexity MUST remain within limits

#### Scenario: Never inline for cold functions

**Given** a function handles rare error cases
**When** the function has more than 100 instructions
**Then** it MUST be marked `__noinline`
**And** it MUST reduce I-cache pressure
**And** overall program size MUST be optimized

---

### Requirement: Branch Prediction Hints

The system MUST use compiler hints to optimize branch prediction.

#### Scenario: Likely session cache hit

**Given** session lookup is performed
**When** a session exists (common case)
**Then** the branch MUST be marked with `[[likely]]`
**And** CPU branch predictor MUST be optimized for this path

#### Scenario: Unlikely error conditions

**Given** packet parsing can fail
**When** errors occur (rare case)
**Then** error paths MUST be marked with `[[unlikely]]`
**And** hot path code layout MUST be optimized

---

## MODIFIED Requirements

### Requirement: Session Value Structure Layout

The `session_value` structure MUST be reordered for optimal cache utilization.

#### Scenario: Hot fields in first cache line

**Given** the session structure is accessed on every packet
**When** hot fields are positioned in memory
**Then** `policy_action` MUST be at offset 0
**And** `state` and `tcp_state` MUST follow within 8 bytes
**And** `last_seen_ts` MUST be within the first 16 bytes
**And** all hot fields MUST fit within 64 bytes (one cache line)

**Before:**
```c
struct session_value {
    __u64 created_ts;       // +0
    __u64 last_seen_ts;     // +8
    __u64 packets_to_server;// +16
    __u64 packets_to_client;// +24
    __u64 bytes_to_server;  // +32
    __u64 bytes_to_client;  // +40
    __u32 tcp_seq_client;   // +48
    __u32 tcp_seq_server;   // +52
    __u8  state;            // +56
    __u8  tcp_state;        // +57
    __u8  policy_action;    // +58 (cache miss!)
    ...
};
```

**After:**
```c
struct session_value {
    __u8  policy_action;    // +0  (HOT: first byte)
    __u8  state;            // +1
    __u8  tcp_state;        // +2
    __u8  flags;            // +3
    __u32 pad;              // +4  (alignment)
    __u64 last_seen_ts;     // +8  (HOT: updated every packet)
    __u64 packets_to_server;// +16 (HOT)
    __u64 bytes_to_server;  // +24 (HOT)
    __u64 packets_to_client;// +32
    __u64 bytes_to_client;  // +40
    __u64 created_ts;       // +48 (COLD)
    __u32 tcp_seq_client;   // +56 (COLD)
    __u32 tcp_seq_server;   // +60 (COLD)
    ...
};
```

#### Scenario: Cache efficiency measurement

**Given** the optimized structure layout
**When** processing 1M packets
**Then** cache misses MUST be reduced by at least 30%
**And** memory bandwidth usage MUST decrease

---

### Requirement: Main Filter Function Structure

The `tc_microsegment_filter()` function MUST be refactored for minimal hot path overhead.

#### Scenario: Simplified main function

**Given** packet processing entry point
**When** the filter function is called
**Then** it MUST extract flow key within 20 instructions
**And** it MUST perform session lookup within 30 total instructions
**And** it MUST branch to fast/slow path based on lookup result
**And** slow path logic MUST be extracted to separate function

**Before:**
```c
SEC("tc")
int tc_microsegment_filter(struct __sk_buff *skb) {
    struct flow_key key = {0};

    if (extract_flow_key(skb, &key) < 0) {
        return TC_ACT_OK;
    }

    __u8 direction = detect_direction(skb);
    update_stats(STATS_TOTAL_PACKETS);
    update_stats(direction == INGRESS ? STATS_INGRESS : STATS_EGRESS);

    struct session_value *session = bpf_map_lookup_elem(&session_map, &key);
    if (session) {
        // 100+ lines of hot path logic...
    }
    // 300+ lines of slow path logic...
}
```

**After:**
```c
SEC("tc")
int tc_microsegment_filter(struct __sk_buff *skb) {
    struct flow_key key = {0};

    if (extract_flow_key(skb, &key) < 0) [[unlikely]] {
        return TC_ACT_OK;
    }

    struct session_value *session = bpf_map_lookup_elem(&session_map, &key);
    if (session) [[likely]] {
        return handle_existing_session_fast(skb, &key, session);
    }

    return handle_new_session_slow(skb, &key);
}

static __always_inline int handle_existing_session_fast(
    struct __sk_buff *skb,
    struct flow_key *key,
    struct session_value *session
) {
    // Optimized hot path: <60 instructions
}

static __noinline int handle_new_session_slow(
    struct __sk_buff *skb,
    struct flow_key *key
) {
    // Slow path: full policy matching
}
```

---

## Performance Acceptance Criteria

### Requirement: Latency Targets

The optimized hot path MUST meet specific latency targets.

#### Scenario: Fast path latency measurement

**Given** a flow with existing session
**When** 1,000,000 packets are processed
**Then** the 50th percentile latency MUST be <5μs
**And** the 99th percentile latency MUST be <10μs
**And** the 99.9th percentile latency MUST be <15μs

#### Scenario: Instruction count verification

**Given** the optimized fast path code
**When** analyzed with bpftool
**Then** total instructions MUST be ≤60
**And** verifier complexity MUST be within kernel limits
**And** stack usage MUST be <512 bytes

---

### Requirement: Throughput Targets

The system MUST support high packet rates without degradation.

#### Scenario: High packet rate handling

**Given** a benchmark with sustained packet load
**When** processing 1,000,000 packets per second
**Then** no packets MUST be dropped due to eBPF processing
**And** CPU utilization MUST be <5% per core
**And** latency MUST remain within targets

---

## Testing Requirements

### Requirement: Performance Regression Testing

The system MUST include automated performance regression detection.

#### Scenario: Per-commit latency tracking

**Given** a new commit is merged
**When** benchmarks are run
**Then** latency MUST not regress by more than 5%
**And** throughput MUST not decrease by more than 5%
**And** CI MUST fail if regression detected

#### Scenario: Baseline comparison

**Given** the optimization implementation
**When** compared to baseline measurements
**Then** fast path latency MUST improve by at least 40%
**And** throughput MUST improve by at least 20%
**And** all functional tests MUST still pass

---

## Compatibility Requirements

### Requirement: Kernel Version Support

Optimizations MUST work across supported kernel versions.

#### Scenario: Minimum kernel version

**Given** kernel version 5.10 or later
**When** eBPF programs are loaded
**Then** all optimizations MUST function correctly
**And** verifier MUST accept the programs
**And** performance gains MUST be measurable

#### Scenario: Feature detection

**Given** advanced optimization features (e.g., `[[likely]]` hints)
**When** not supported by kernel/compiler
**Then** system MUST fall back gracefully
**And** basic optimizations MUST still apply
**And** functionality MUST be preserved

---

## Monitoring Requirements

### Requirement: Performance Metrics Export

The system MUST expose performance metrics for monitoring.

#### Scenario: Latency histogram export

**Given** the eBPF filter is running
**When** user-space queries metrics
**Then** latency histograms MUST be available
**And** percentile calculations MUST be accurate
**And** metrics MUST update in real-time

#### Scenario: Instruction count tracking

**Given** eBPF programs are loaded
**When** performance is monitored
**Then** instruction counts per path MUST be logged
**And** verifier statistics MUST be exported
**And** trends MUST be trackable over time

---

## Documentation Requirements

### Requirement: Optimization Documentation

All performance optimizations MUST be documented.

#### Scenario: Code comments

**Given** an optimization is implemented
**When** code is reviewed
**Then** rationale MUST be documented in comments
**And** performance impact MUST be quantified
**And** trade-offs MUST be explained

#### Scenario: Performance tuning guide

**Given** the optimization is complete
**When** users need to configure performance
**Then** a tuning guide MUST exist
**And** recommended settings MUST be provided
**And** troubleshooting tips MUST be included
