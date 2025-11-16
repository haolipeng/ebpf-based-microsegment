---
issue: 18
epic: wildcard-policy-performance-optimization
analyzed: 2025-11-16T13:06:00Z
complexity: simple
work_streams: 1
---

# Analysis: eBPF 循环常量化与早停优化

## Task Overview

Optimize wildcard policy matching loop in eBPF by:
1. Replacing hardcoded 100 with MAX_WILDCARD_LOOP constant (value: 50)
2. Implementing early termination when encountering empty slots
3. Ensuring eBPF verifier compliance

## File Analysis

### Primary File
- **src/bpf/headers/policy_match.h** (Lines 137-164)
  - Current loop: `for (__u32 i = 0; i < 100; i++)`
  - Contains wildcard policy matching logic
  - Needs constant definition and early stop logic

## Work Stream Decomposition

### Stream 1: Single-file Optimization (Primary)
**Agent Type**: general-purpose
**Estimated Time**: 2-4 hours
**Parallelizable**: No (single file, sequential changes)

**Scope**:
1. Add constant definition at file top
2. Replace hardcoded 100 with MAX_WILDCARD_LOOP
3. Add early termination logic after NULL check
4. Verify eBPF verifier passes

**Files**:
- src/bpf/headers/policy_match.h

**Implementation Steps**:
1. Define `#define MAX_WILDCARD_LOOP 50` near top of file
2. Change loop condition from `i < 100` to `i < MAX_WILDCARD_LOOP`
3. Add break statement when `wildcard->rule_id == 0`
4. Compile and verify with eBPF verifier
5. Run existing tests to ensure no regression

**Risks**:
- eBPF verifier may reject changes (low risk - simple modification)
- Early stop logic could miss valid entries (mitigated by checking rule_id == 0)

## Dependencies

**External**:
- None (standalone optimization)

**Internal**:
- None (this is Phase 1, Task 1)

## Testing Strategy

1. **Compilation Test**: Ensure eBPF program compiles
2. **Verifier Test**: Pass eBPF verifier validation
3. **Functional Test**: Verify policy matching still works correctly
4. **Performance Test**: Measure latency reduction (informal)

## Success Criteria

- [x] MAX_WILDCARD_LOOP constant defined
- [x] Loop uses constant instead of hardcoded value
- [x] Early stop logic implemented
- [x] eBPF verifier passes
- [x] No test regressions

## Notes

This is a straightforward optimization with minimal risk. The change is localized to a single file and involves simple constant replacement plus adding a break statement. No parallel work streams needed.
