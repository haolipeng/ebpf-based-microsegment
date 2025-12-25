# Optimize eBPF Data Plane Performance

## Change Metadata
- **ID**: `optimize-ebpf-performance`
- **Type**: enhancement
- **Status**: proposed
- **Priority**: high
- **Created**: 2025-11-18
- **Target Version**: v0.2.0

## Problem Statement

The current eBPF data plane implementation has several performance bottlenecks that prevent it from achieving the target <10μs latency overhead and 100,000+ concurrent session support:

### Current Performance Issues

1. **Hot Path Inefficiencies**
   - Unnecessary map lookups in fast path (99%+ of packets)
   - Multiple stats updates causing cache thrashing
   - Redundant boundary checks
   - Suboptimal inline directives

2. **Map Access Patterns**
   - Session map lookups without early return optimization
   - Stats updates on every packet (Per-CPU contention)
   - Fragment state checks even for non-fragmented packets
   - NAT detection overhead on every new session

3. **Code Complexity**
   - Deep nesting in main filter function
   - Large inline functions causing instruction count bloat
   - Duplicated logic between TC and XDP programs
   - Excessive DEBUG_MODE conditional compilation

4. **Memory Access Patterns**
   - Non-sequential struct field access
   - Repeated map lookups for configuration
   - Unnecessary memcpy operations

### Impact

- **Latency**: Current overhead is estimated at 15-20μs per packet (2x target)
- **Throughput**: Suboptimal for high packet rates (>1M pps)
- **Scalability**: Session tracking becomes bottleneck at 50K+ concurrent connections
- **CPU**: Higher than necessary CPU utilization

## Proposed Solution

Implement a multi-phase optimization strategy targeting the eBPF data plane hot paths:

### Phase 1: Hot Path Optimization (Critical)

**Target**: Reduce fast path latency by 40% (to <10μs)

- **Early Return Optimization**: Minimize instructions before session cache hit
- **Stats Batching**: Reduce per-packet stats updates
- **Inline Optimization**: Strategic `__always_inline` placement
- **Branch Prediction Hints**: Use `__builtin_expect()` for common cases

### Phase 2: Map Access Optimization

**Target**: Reduce map lookup overhead by 30%

- **Configuration Caching**: Cache frequently-accessed config in stack
- **Batch Map Updates**: Combine multiple stats updates
- **Selective Fragment Checks**: Skip fragment logic for non-fragmented traffic
- **NAT Fast Path**: Early bailout for non-NAT traffic

### Phase 3: Code Structure Refactoring

**Target**: Reduce instruction count by 20%

- **Function Decomposition**: Split large functions into focused helpers
- **Common Code Extraction**: Share logic between TC/XDP
- **Conditional Compilation**: Remove debug overhead in production builds
- **Struct Packing**: Optimize memory layout for cache efficiency

### Phase 4: Advanced Optimizations (Optional)

**Target**: Further 10-15% improvement

- **Tail Calls**: Split processing into stages
- **Map-in-Map**: Optimize policy lookup structure
- **Percpu Variables**: Reduce stats map pressure
- **XDP Native Mode**: Leverage driver-level hooks

## Success Criteria

### Performance Metrics

- [ ] **Latency**: <10μs overhead per packet (99th percentile)
- [ ] **Throughput**: Handle >1M pps without packet loss
- [ ] **Sessions**: Support 100K+ concurrent connections
- [ ] **CPU**: <5% CPU usage at 100Kpps

### Quality Metrics

- [ ] All existing tests pass
- [ ] No regression in functionality
- [ ] Verifier complexity within limits
- [ ] Code maintainability preserved

### Measurement Tools

- **eBPF Metrics**: bpftool prog show (run_time_ns, run_cnt)
- **Latency**: Custom timestamp tracking in eBPF
- **Throughput**: iperf3 + netperf benchmarks
- **Profiling**: perf + FlameGraph analysis

## Implementation Strategy

### 1. Baseline Measurement

- Capture current performance metrics
- Identify hot spots using perf/bpftool
- Document instruction counts
- Profile map access patterns

### 2. Incremental Optimization

- Implement Phase 1 optimizations
- Measure and validate improvements
- Proceed to Phase 2 only if targets met
- Iterative refinement based on data

### 3. Regression Prevention

- Automated performance benchmarks in CI
- Per-commit latency tracking
- Verifier complexity monitoring
- Regular performance audits

## Dependencies

### Required
- bpftool for performance measurement
- perf for profiling
- Test harness for benchmarking

### Recommended
- FlameGraph tools for visualization
- bpf-perf for advanced profiling
- Kernel ≥5.15 for latest eBPF features

## Risks and Mitigation

### Technical Risks

1. **Verifier Rejection**
   - *Risk*: Optimizations may exceed verifier complexity limits
   - *Mitigation*: Incremental changes, test with multiple kernel versions

2. **Correctness Issues**
   - *Risk*: Performance optimizations introduce bugs
   - *Mitigation*: Extensive testing, formal verification where possible

3. **Maintainability**
   - *Risk*: Over-optimization reduces code readability
   - *Mitigation*: Document all optimizations, preserve tests

### Operational Risks

1. **Kernel Compatibility**
   - *Risk*: Optimizations work only on newer kernels
   - *Mitigation*: Feature detection, graceful fallback

2. **Production Impact**
   - *Risk*: Untested edge cases in optimized code
   - *Mitigation*: Gradual rollout, canary deployments

## Alternatives Considered

### 1. XDP-Only Approach
- **Pros**: Lower latency, earlier processing
- **Cons**: Less flexible, no egress support, driver dependency
- **Decision**: Keep TC for compatibility, optimize both

### 2. eBPF Offload
- **Pros**: Hardware acceleration
- **Cons**: Limited hardware support, vendor lock-in
- **Decision**: Not viable for general purpose solution

### 3. User-Space Fast Path (DPDK)
- **Pros**: Maximum performance potential
- **Cons**: Complexity, kernel bypass, not eBPF
- **Decision**: Out of scope for eBPF learning project

## References

- [Linux kernel BPF docs](https://www.kernel.org/doc/html/latest/bpf/bpf_design_QA.html)
- [Cilium Performance Guide](https://docs.cilium.io/en/stable/operations/performance/)
- [BPF Performance Tools](http://www.brendangregg.com/bpf-performance-tools-book.html)
- Project docs: `docs/specs/roadmap.md`
- ZFW Analysis: `docs/research/zfw/zfw-deep-dive.md`

## Timeline

- **Baseline & Design**: Week 1
- **Phase 1 Implementation**: Week 2-3
- **Phase 2 Implementation**: Week 4
- **Phase 3 Implementation**: Week 5
- **Validation & Documentation**: Week 6

Total Duration: 6 weeks (aligns with project learning timeline)
