# Implementation Tasks

## Phase 0: Preparation & Baseline (Week 1)

### Task 1: Setup Performance Measurement Infrastructure
**Priority**: P0 (Blocker)
**Effort**: 2 days

- [ ] Create benchmark harness script (`scripts/perf-benchmark.sh`)
  - iperf3 integration for throughput
  - Custom latency measurement tool
  - Packet generator (pktgen or similar)
- [ ] Add eBPF timestamp instrumentation
  - Add per-path latency tracking
  - Create histogram map for latency distribution
- [ ] Setup flamegraph tooling
  - perf record/report scripts
  - bpf-perf integration
- [ ] Document baseline metrics
  - Current latency (p50, p99, p999)
  - Throughput at various packet sizes
  - CPU utilization profile
  - Instruction count per path

**Validation**: Can run `./scripts/perf-benchmark.sh` and get reproducible results

**Dependencies**: None

---

### Task 2: Profile Current Implementation
**Priority**: P0
**Effort**: 1 day

- [ ] Run perf analysis on TC program
  - Identify hot functions
  - Measure map access overhead
  - Analyze cache misses
- [ ] Use bpftool to analyze program stats
  - Run time per invocation
  - Invocation count
  - Stack usage
- [ ] Create flamegraph of data plane
- [ ] Document top 10 bottlenecks

**Validation**: Have flamegraph and bottleneck analysis document

**Dependencies**: Task 1

---

## Phase 1: Hot Path Optimization (Week 2-3)

### Task 3: Implement Early Return Pattern
**Priority**: P0 (Critical Path)
**Effort**: 2 days

- [ ] Refactor `tc_microsegment_filter()` main function
  - Move session lookup immediately after flow key extraction
  - Defer direction detection to slow path
  - Defer stats updates to batch function
- [ ] Add `[[likely]]` / `[[unlikely]]` hints
  - Mark session hit as likely
  - Mark flow key extraction failure as unlikely
- [ ] Create `handle_existing_session_fast()` inline function
  - Policy action check
  - Stats update (batched)
  - TCP state update (if needed)
  - Return action
- [ ] Measure improvement
  - Compare instruction count before/after
  - Measure latency reduction

**Validation**:
- Fast path instruction count reduced by >50%
- All tests pass
- Latency improves by >30%

**Dependencies**: Task 2

---

### Task 4: Implement Stats Batching
**Priority**: P1
**Effort**: 1.5 days

- [ ] Modify stats update logic
  - Add batching threshold (every 16th packet)
  - Update session counters on every packet
  - Update global stats on batch boundary
- [ ] Verify statistical accuracy
  - Test with high packet rates
  - Ensure <1% error margin
- [ ] Update user-space stats collection
  - Account for batching in calculations
  - Update API documentation

**Validation**:
- Per-CPU contention reduced by >90%
- Stats accuracy within 1%
- Throughput improves by >20%

**Dependencies**: Task 3

---

### Task 5: Optimize Inline Strategy
**Priority**: P1
**Effort**: 1 day

- [ ] Audit all function inline directives
  - `__always_inline` for <20 instruction functions
  - `__noinline` for >100 instruction functions
  - Remove directive for 20-100 instruction functions
- [ ] Test verifier complexity
  - Ensure all programs still load
  - Monitor instruction count
- [ ] Measure I-cache efficiency
  - Use perf to check instruction cache misses

**Validation**:
- Verifier accepts all programs
- Instruction cache misses reduced by >15%
- Code size within limits

**Dependencies**: None (parallel with Task 3-4)

---

### Task 6: Optimize Struct Field Layout
**Priority**: P2
**Effort**: 1 day

- [ ] Reorder `struct session_value` fields
  - Move hot fields (`policy_action`, `last_seen_ts`) to start
  - Group related fields together
  - Ensure 8-byte alignment
- [ ] Reorder `struct flow_key` if needed
- [ ] Update all code accessing structs
- [ ] Verify padding and alignment
  - Use pahole or similar tool
  - Ensure single cache line for hot fields (64 bytes)

**Validation**:
- Hot fields in first cache line
- Cache misses reduced by >30%
- All tests pass

**Dependencies**: Task 3 (avoid conflicts)

---

## Phase 2: Map Access Optimization (Week 4)

### Task 7: Implement Configuration Caching
**Priority**: P1
**Effort**: 2 days

- [ ] Add global config cache variables
  - `volatile const` for fragment config
  - `volatile const` for NAT config
  - `volatile const` for timeout config
- [ ] Implement user-space config update mechanism
  - Update config cache on program reload
  - Use bpf_obj_get_info_by_fd() API
- [ ] Replace config map lookups with cache reads
  - Fragment mode check
  - NAT enabled check
- [ ] Measure improvement

**Validation**:
- Config map lookups eliminated
- Latency reduction of 3-5μs per check
- Config updates still work correctly

**Dependencies**: None

---

### Task 8: Selective Fragment Checking
**Priority**: P2
**Effort**: 1 day

- [ ] Add fragment mode check before fragment logic
  - Skip if mode is DISABLED
  - Quick check for STRICT mode
- [ ] Restructure code flow
  - Use goto labels for readability
  - Minimize code duplication
- [ ] Test with fragmented packets
  - Ensure correctness preserved
  - Verify performance improvement

**Validation**:
- Fragment overhead eliminated for non-fragment mode
- 99.9% of packets skip fragment checks
- Fragment tests still pass

**Dependencies**: Task 7 (config caching)

---

### Task 9: NAT Fast Path
**Priority**: P1
**Effort**: 1.5 days

- [ ] Add NAT enabled check before detection
  - Use config cache
  - Early bailout if NAT disabled
- [ ] Optimize conntrack cache lookup
  - Defer if NAT not needed
  - Combine with other map lookups if possible
- [ ] Measure NAT overhead savings

**Validation**:
- NAT overhead eliminated when disabled
- Latency improves by 5-8μs per new session
- NAT tests pass

**Dependencies**: Task 7 (config caching)

---

### Task 10: Batch Map Updates
**Priority**: P2
**Effort**: 1 day

- [ ] Combine multiple stats updates
  - Single map update call where possible
  - Use struct to group related stats
- [ ] Optimize flow event submission
  - Batch events in ring buffer
  - Defer non-critical events
- [ ] Measure improvement

**Validation**:
- Map update calls reduced by >40%
- Ring buffer efficiency improved
- Events still delivered reliably

**Dependencies**: Task 4 (stats batching)

---

## Phase 3: Code Structure Refactoring (Week 5)

### Task 11: Function Decomposition
**Priority**: P1
**Effort**: 2 days

- [ ] Split `tc_microsegment_filter()` into focused functions
  - `handle_existing_session_fast()` - hot path
  - `handle_new_session_slow()` - slow path
  - `handle_fragment_packet()` - fragment path
- [ ] Extract common logic into helpers
  - `extract_flow_key_fast()` - optimized extraction
  - `update_session_stats()` - batched stats
  - `enforce_policy_action()` - action enforcement
- [ ] Share code between TC and XDP
  - Move common logic to headers
  - Reduce duplication
- [ ] Measure instruction count reduction

**Validation**:
- Instruction count reduced by >20%
- Code maintainability improved
- All tests pass
- Verifier accepts programs

**Dependencies**: Tasks 3-10 (after optimizations stabilize)

---

### Task 12: Remove Debug Overhead
**Priority**: P2
**Effort**: 0.5 day

- [ ] Audit DEBUG_MODE usage
  - Ensure bpf_printk removed in production
  - Remove unnecessary conditional checks
- [ ] Add compile-time feature flags
  - Separate debug builds from production
  - Document flag usage
- [ ] Measure overhead of debug code

**Validation**:
- Production build has zero debug overhead
- Debug build still works for development
- Instruction count reduced by 5-10%

**Dependencies**: None

---

### Task 13: Optimize Common Paths
**Priority**: P2
**Effort**: 1 day

- [ ] Identify duplicate code between TC and XDP
- [ ] Extract to shared header functions
- [ ] Ensure consistent behavior
- [ ] Test both TC and XDP

**Validation**:
- Code duplication reduced by >60%
- TC and XDP have identical logic
- Both programs pass tests

**Dependencies**: Task 11

---

## Phase 4: Advanced Optimizations (Week 5-6, Optional)

### Task 14: Tail Call Implementation (Optional)
**Priority**: P3 (Nice to have)
**Effort**: 3 days

- [ ] Design tail call architecture
  - Fast path program
  - Slow path program
  - Fragment program
- [ ] Implement prog_array map
- [ ] Split existing code into stages
- [ ] Handle tail call failures gracefully
- [ ] Test extensively

**Validation**:
- Verifier complexity per program reduced
- Overall latency comparable or better
- Fallback path works correctly

**Dependencies**: Task 11 (refactoring first)

---

### Task 15: Percpu Variable Optimization (Optional)
**Priority**: P3
**Effort**: 2 days

- [ ] Replace stats map with percpu variables
- [ ] Implement user-space aggregation
- [ ] Test performance improvement
- [ ] Ensure compatibility

**Validation**:
- Stats updates faster
- Accuracy preserved
- Lower cache contention

**Dependencies**: Task 4, 10 (stats optimization)

---

## Validation & Testing (Week 6)

### Task 16: Comprehensive Performance Testing
**Priority**: P0
**Effort**: 2 days

- [ ] Run full benchmark suite
  - Latency tests (p50, p99, p999)
  - Throughput tests (various packet sizes)
  - Scalability tests (10K, 50K, 100K sessions)
  - CPU utilization profiling
- [ ] Compare against baseline
  - Document improvements
  - Identify any regressions
  - Validate success criteria met
- [ ] Stress testing
  - High packet rate (>1M pps)
  - Connection churn
  - Memory pressure
  - Long-running stability

**Validation**:
- All performance targets met
- No functional regressions
- System stable under load

**Dependencies**: All optimization tasks

---

### Task 17: Regression Test Suite
**Priority**: P0
**Effort**: 1 day

- [ ] Run all existing unit tests
- [ ] Run E2E integration tests
- [ ] Run fragment handling tests
- [ ] Run NAT tests
- [ ] Run TCP state machine tests
- [ ] Verify test coverage

**Validation**:
- 100% existing tests pass
- No new bugs introduced
- Test coverage maintained

**Dependencies**: All optimization tasks

---

### Task 18: Documentation Updates
**Priority**: P1
**Effort**: 1 day

- [ ] Update `docs/specs/roadmap.md`
  - Add performance optimization section
  - Document achieved metrics
  - Update architecture diagrams
- [ ] Create performance tuning guide
  - Explain optimization techniques
  - Provide configuration guidance
  - Include troubleshooting tips
- [ ] Update code comments
  - Document optimization rationale
  - Explain complex patterns
  - Add performance notes
- [ ] Create flamegraph interpretation guide
  - How to read flamegraphs
  - Common bottlenecks
  - Optimization opportunities

**Validation**:
- Documentation complete and accurate
- Guides helpful for users
- Code well-commented

**Dependencies**: Task 16 (final metrics)

---

## Summary

**Total Tasks**: 18
**Critical Path**: Tasks 1 → 2 → 3 → 4 → 6 → 16 → 17
**Estimated Duration**: 6 weeks
**Success Criteria**:
- Latency <10μs (99th percentile) ✓
- Throughput >1M pps ✓
- 100K+ concurrent sessions ✓
- All tests pass ✓

## Parallelization Opportunities

**Week 2**: Tasks 3, 5 (different developers)
**Week 4**: Tasks 7, 9 (different focus areas)
**Week 5**: Tasks 11, 12, 13 (different codebases)

## Risk Mitigation

- Start each phase with measurement
- Validate after each task
- Keep optimization incremental
- Maintain rollback capability
- Test extensively before merging
