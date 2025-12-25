# eBPF Performance Optimization - Change Proposal

## Quick Summary

Optimize the eBPF data plane to achieve <10μs latency overhead and support 100,000+ concurrent sessions through strategic code path optimization, map access improvements, and structural refactoring.

## Status

- **Status**: Proposed
- **Priority**: High (P0)
- **Target Version**: v0.2.0
- **Created**: 2025-11-18
- **Duration**: 6 weeks

## Problem

Current eBPF implementation has 15-20μs latency overhead (2x target) due to:
- Unnecessary work in hot path before session lookup
- Excessive map accesses and stats updates
- Suboptimal code structure and memory layout
- Non-selective feature checks

## Solution

Four-phase optimization strategy:
1. **Hot Path Optimization** - Early return, stats batching, inline optimization
2. **Map Access Optimization** - Config caching, selective checks, batch updates
3. **Code Structure Refactoring** - Function decomposition, shared code, cleanup
4. **Advanced Optimizations** - Tail calls, percpu variables (optional)

## Success Criteria

- ✓ Latency <10μs (p99)
- ✓ Throughput >1M pps
- ✓ Support 100K+ sessions
- ✓ CPU <5% at 100Kpps
- ✓ All tests pass

## Key Metrics

### Baseline (Before)
- Fast path: ~120 instructions, 12μs latency
- Map accesses per packet: 5+ (hot path)
- Stats updates per packet: 3
- Session scalability: 50K concurrent

### Target (After)
- Fast path: ~60 instructions, 5μs latency (-58%)
- Map accesses per packet: 1 (hot path) (-80%)
- Stats updates per packet: 0.06 (batched) (-98%)
- Session scalability: 100K+ concurrent

## Documents

- **[proposal.md](./proposal.md)** - Detailed problem statement and solution approach
- **[design.md](./design.md)** - Technical architecture and implementation patterns
- **[tasks.md](./tasks.md)** - 18 implementation tasks over 6 weeks
- **[specs/](./specs/)** - Formal requirements and acceptance criteria

## Capabilities

1. **[Hot Path Optimization](./specs/hot-path-optimization/spec.md)**
   - Early return pattern
   - Batched statistics
   - Strategic inlining
   - Branch prediction hints
   - Struct field reordering

2. **[Map Access Optimization](./specs/map-access-optimization/spec.md)**
   - Configuration caching
   - Selective feature checks
   - Batch map updates
   - Cache efficiency improvements

## Implementation Phases

### Phase 1: Hot Path (Critical) - Week 2-3
- Early return optimization
- Stats batching
- Inline strategy
- Struct layout optimization

**Target**: 40% latency reduction

### Phase 2: Map Access - Week 4
- Config caching
- Selective fragment/NAT checks
- Batch map operations

**Target**: 30% map overhead reduction

### Phase 3: Refactoring - Week 5
- Function decomposition
- Code sharing (TC/XDP)
- Debug overhead removal

**Target**: 20% instruction count reduction

### Phase 4: Advanced (Optional) - Week 5-6
- Tail call staging
- Percpu variables

**Target**: Additional 10-15% improvement

## Dependencies

**Required**:
- bpftool for measurement
- perf for profiling
- Test harness

**Recommended**:
- FlameGraph tools
- bpf-perf
- Kernel ≥5.15

## Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Verifier rejection | High | Incremental changes, multi-kernel testing |
| Correctness bugs | High | Extensive testing, formal verification |
| Maintainability loss | Medium | Documentation, preserve tests |
| Kernel incompatibility | Medium | Feature detection, graceful fallback |

## Alternatives Considered

- **XDP-Only**: Rejected (no egress support)
- **eBPF Offload**: Rejected (limited hardware support)
- **DPDK**: Rejected (out of scope for eBPF project)

## Validation Strategy

1. **Baseline Measurement** - Capture current metrics
2. **Incremental Optimization** - Phase-by-phase with validation
3. **Regression Prevention** - Automated benchmarks in CI
4. **Continuous Monitoring** - Per-commit latency tracking

## Timeline

| Week | Phase | Deliverable |
|------|-------|-------------|
| 1 | Baseline | Metrics, profiling, bottleneck analysis |
| 2-3 | Phase 1 | Hot path optimization, 40% latency reduction |
| 4 | Phase 2 | Map access optimization, 30% overhead reduction |
| 5 | Phase 3 | Refactoring, 20% instruction reduction |
| 6 | Validation | Testing, documentation, final metrics |

## How to Use This Proposal

1. **Review**: Read `proposal.md` for problem statement and approach
2. **Design**: Study `design.md` for technical architecture
3. **Implement**: Follow `tasks.md` for step-by-step work items
4. **Validate**: Check specs for requirements and acceptance criteria

## Related

- Project Goal: <10μs latency, 100K+ sessions (`openspec/project.md`)
- Architecture: `docs/specs/roadmap.md`
- Reference: ZFW analysis in `docs/research/zfw/`

## Contact

For questions or concerns about this proposal:
- Review design documents
- Check existing performance benchmarks
- Consult project maintainers
