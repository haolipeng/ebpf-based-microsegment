# eBPF Performance Optimization Design

## Architecture Overview

The optimization strategy focuses on three critical paths in the eBPF data plane:

```
┌─────────────────────────────────────────────────┐
│         Packet Processing Pipeline              │
├─────────────────────────────────────────────────┤
│                                                 │
│  1. FAST PATH (99%+ packets)                   │
│     ┌─────────────────────────────────────┐   │
│     │ Session Lookup → Stats → Action     │   │
│     │ Target: <5μs                         │   │
│     └─────────────────────────────────────┘   │
│                                                 │
│  2. SLOW PATH (New sessions)                   │
│     ┌─────────────────────────────────────┐   │
│     │ Policy Match → Create Session       │   │
│     │ Target: <20μs                        │   │
│     └─────────────────────────────────────┘   │
│                                                 │
│  3. FRAGMENT PATH (Rare)                       │
│     ┌─────────────────────────────────────┐   │
│     │ Fragment Detect → Cache Lookup      │   │
│     │ Target: <15μs                        │   │
│     └─────────────────────────────────────┘   │
│                                                 │
└─────────────────────────────────────────────────┘
```

## Phase 1: Hot Path Optimization

### 1.1 Early Return Pattern

**Current Problem:**
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
        // HOT PATH - but after 3 function calls and 2 map updates!
        ...
    }
}
```

**Optimized Approach:**
```c
SEC("tc")
int tc_microsegment_filter(struct __sk_buff *skb) {
    struct flow_key key = {0};

    // FAST PATH: Minimize work before session lookup
    if (extract_flow_key(skb, &key) < 0) [[unlikely]] {
        return TC_ACT_OK;
    }

    struct session_value *session = bpf_map_lookup_elem(&session_map, &key);
    if (session) [[likely]] {
        // HOT PATH - optimized to 2 instructions before lookup
        return handle_existing_session(skb, &key, session);
    }

    // SLOW PATH - infrequent, can afford overhead
    return handle_new_session(skb, &key);
}
```

**Gains:**
- Reduce hot path instruction count by 60%
- Eliminate unnecessary stats updates before session check
- Improve I-cache utilization

### 1.2 Stats Batching

**Current Problem:**
```c
// Every packet updates multiple stats
update_stats(STATS_TOTAL_PACKETS);
update_stats(STATS_INGRESS_PACKETS);  // or EGRESS
session->packets_to_server += 1;
session->bytes_to_server += skb->len;
```

**Optimized Approach:**
```c
// Batch stats update every N packets
static __always_inline void update_stats_batch(
    struct session_value *session,
    __u32 len,
    __u8 direction
) {
    session->packets_to_server++;
    session->bytes_to_server += len;

    // Update global stats only every 16th packet
    if ((session->packets_to_server & 0xF) == 0) {
        update_stats(STATS_TOTAL_PACKETS);
        update_stats(direction == INGRESS ? STATS_INGRESS : STATS_EGRESS);
    }
}
```

**Gains:**
- Reduce Per-CPU array contention by 93%
- Lower cache pressure
- Maintain statistical accuracy (±0.1% error acceptable)

### 1.3 Inline Optimization Strategy

**Guidelines:**
```c
// Always inline: Small, hot functions (<20 instructions)
static __always_inline __u8 get_policy_action(struct session_value *s) {
    return s->policy_action;
}

// Never inline: Large, cold functions
static __noinline void handle_tcp_state_transition(...) {
    // Complex logic, rarely called
}

// Let compiler decide: Medium complexity
static inline bool is_fragmented(...) {
    // Compiler chooses based on call sites
}
```

## Phase 2: Map Access Optimization

### 2.1 Configuration Caching

**Current Problem:**
```c
// Every fragment check reads config
struct frag_config *config = bpf_map_lookup_elem(&frag_config_map, &config_key);
__u8 mode = config ? config->mode : FRAG_MODE_NORMAL;
```

**Optimized Approach:**
```c
// Cache config in program global variable
volatile const struct frag_config CONFIG_CACHE = {0};

// User-space updates this via bpf_obj_get_info_by_fd() + bpf_obj_pin()
static __always_inline __u8 get_frag_mode(void) {
    return CONFIG_CACHE.mode ?: FRAG_MODE_NORMAL;
}
```

**Gains:**
- Eliminate config map lookup on every fragment
- Reduce from O(1) map access to O(1) memory read
- 3-5μs saved per fragmented packet

### 2.2 Selective Fragment Checking

**Current Problem:**
```c
// Check every packet for fragments
if (eth_proto == bpf_htons(ETH_P_IP)) {
    struct iphdr *iph = (struct iphdr *)(eth + 1);
    if (is_ipv4_fragment(iph)) {
        // Fragment handling...
    }
}
```

**Optimized Approach:**
```c
// Skip fragment check if not in fragment mode
if (CONFIG_CACHE.mode != FRAG_MODE_STRICT) [[likely]] {
    // Regular processing
    goto skip_fragment_check;
}

// Only check when fragments are expected
if (eth_proto == bpf_htons(ETH_P_IP)) {
    // Fragment handling...
}

skip_fragment_check:
// Continue normal path
```

**Gains:**
- Eliminate fragment overhead for 99.9% of packets
- Reduce code path complexity

### 2.3 NAT Fast Path

**Current Problem:**
```c
// Always call NAT detection
detect_nat_and_restore_with_maps(
    skb, false, &key, &original_key, &nat_type,
    &nat_config_map, &conntrack_cache_map, &nat_stats_map
);
```

**Optimized Approach:**
```c
// Check NAT config first
if (CONFIG_CACHE.nat_enabled) [[unlikely]] {
    detect_nat_and_restore_with_maps(...);
} else {
    // Fast path: no NAT overhead
    original_key = key;
}
```

## Phase 3: Code Structure Refactoring

### 3.1 Function Decomposition

**Before:**
```c
// 827 lines in tc_microsegment_filter()
SEC("tc")
int tc_microsegment_filter(struct __sk_buff *skb) {
    // Extract key
    // Check direction
    // Update stats
    // Lookup session
    // Handle fragments
    // Detect NAT
    // Match policy
    // Create session
    // Enforce action
    // ...
}
```

**After:**
```c
SEC("tc")
int tc_microsegment_filter(struct __sk_buff *skb) {
    struct flow_key key;
    if (!extract_flow_key_fast(skb, &key)) [[unlikely]] {
        return TC_ACT_OK;
    }

    struct session_value *session = bpf_map_lookup_elem(&session_map, &key);
    return session ?
        handle_existing_session_fast(skb, &key, session) :
        handle_new_session_slow(skb, &key);
}

// Separate functions for clarity
static __always_inline int handle_existing_session_fast(...) { }
static __noinline int handle_new_session_slow(...) { }
```

### 3.2 Struct Field Ordering

**Current:**
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
    __u8  state;            // +56 (cache miss!)
    __u8  tcp_state;        // +57
    __u8  policy_action;    // +58 (HOT FIELD!)
    ...
};
```

**Optimized:**
```c
struct session_value {
    __u8  policy_action;    // +0  (HOT: first cache line)
    __u8  state;            // +1
    __u8  tcp_state;        // +2
    __u8  flags;            // +3
    __u32 pad;              // +4  (alignment)
    __u64 last_seen_ts;     // +8  (HOT: updated every packet)
    __u64 packets_to_server;// +16
    __u64 bytes_to_server;  // +24
    // Cold fields in second cache line
    __u64 created_ts;       // +32
    __u64 packets_to_client;// +40
    ...
};
```

**Gains:**
- Single cache line access for hot path (64 bytes)
- Reduce memory bandwidth by 50%

## Phase 4: Advanced Optimizations

### 4.1 Tail Calls for Staged Processing

```c
// Program split into stages
struct {
    __uint(type, BPF_MAP_TYPE_PROG_ARRAY);
    __uint(max_entries, 8);
    __type(key, __u32);
    __type(value, __u32);
} prog_array SEC(".maps");

enum {
    STAGE_FAST_PATH = 0,
    STAGE_SLOW_PATH = 1,
    STAGE_FRAGMENT  = 2,
};

SEC("tc")
int tc_main(struct __sk_buff *skb) {
    // Quick session check
    struct session_value *session = lookup_session_fast(skb);
    if (session) {
        bpf_tail_call(skb, &prog_array, STAGE_FAST_PATH);
    } else {
        bpf_tail_call(skb, &prog_array, STAGE_SLOW_PATH);
    }
    return TC_ACT_OK;  // Fallback
}

SEC("tc/fast")
int tc_fast_path(struct __sk_buff *skb) {
    // Optimized for existing sessions
}

SEC("tc/slow")
int tc_slow_path(struct __sk_buff *skb) {
    // Full policy matching
}
```

**Gains:**
- Reduce instruction count per program
- Bypass verifier complexity limits
- Better I-cache utilization

### 4.2 Percpu Variables

```c
// Replace stats map with percpu variables
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct global_stats);
} stats_cache SEC(".maps");

// Access pattern
struct global_stats *stats = bpf_map_lookup_elem(&stats_cache, &zero);
if (stats) {
    stats->total_packets++;  // No map update overhead
}
```

## Performance Budget

### Latency Breakdown (Target)

| Component | Current | Target | Optimization |
|-----------|---------|--------|--------------|
| Flow Key Extract | 2μs | 1μs | Inline, reduce checks |
| Session Lookup | 3μs | 2μs | Map optimization |
| Stats Update | 4μs | 0.5μs | Batching |
| Policy Action | 1μs | 0.5μs | Cache hit |
| Enforcement | 2μs | 1μs | Early return |
| **Total (Fast)** | **12μs** | **5μs** | **58% reduction** |

### Instruction Count Budget

| Path | Current | Target | Notes |
|------|---------|--------|-------|
| Fast Path | ~120 | ~60 | Critical |
| Slow Path | ~800 | ~600 | Acceptable |
| Fragment | ~200 | ~120 | Rare |

## Measurement Framework

### 1. eBPF Instrumentation

```c
// Timestamp tracking
static __always_inline __u64 measure_start(void) {
    return bpf_ktime_get_ns();
}

static __always_inline void measure_end(__u64 start, __u32 metric) {
    __u64 delta = bpf_ktime_get_ns() - start;
    update_histogram(metric, delta);
}

// Usage
__u64 t0 = measure_start();
// ... code ...
measure_end(t0, METRIC_FAST_PATH_LATENCY);
```

### 2. Continuous Monitoring

```bash
# Automated benchmarks
./scripts/perf-benchmark.sh \
    --baseline baseline.json \
    --threshold 0.95 \
    --metrics latency,throughput,cpu

# Per-commit tracking
git log --oneline | while read commit; do
    git checkout $commit
    ./run-benchmark.sh > "perf-$commit.json"
done
```

## Validation Criteria

### Functional Tests
- [ ] All existing unit tests pass
- [ ] E2E tests verify correctness
- [ ] Stress tests with 100K sessions
- [ ] Fragment handling preserved
- [ ] NAT detection accurate

### Performance Tests
- [ ] Latency <10μs (p99)
- [ ] Throughput >1M pps
- [ ] CPU usage <5% at 100Kpps
- [ ] Memory stable under load
- [ ] No packet drops

### Verifier Tests
- [ ] Programs load on kernel 5.10+
- [ ] Instruction count within limits
- [ ] Stack usage <512 bytes
- [ ] No unbounded loops
- [ ] All paths verified

## Rollback Plan

If performance regressions occur:

1. **Immediate**: Revert to baseline via feature flag
2. **Short-term**: Bisect to identify problematic commit
3. **Long-term**: Re-evaluate optimization strategy

## Documentation Updates

- Update `docs/specs/roadmap.md` with performance section
- Add performance tuning guide
- Document optimization techniques in code comments
- Create flamegraph interpretation guide
