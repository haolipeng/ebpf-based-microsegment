---
issue: 46
created: 2025-11-19T12:59:30Z
analysis_type: work_stream
---

# Issue #46 Work Stream Analysis

## Overview

Task: eBPF Tracepoint 和进程监控
Estimated effort: 80h (2 weeks)

## Parallel Work Streams

### Stream A: Data Structure Definition (Priority: HIGH, Can start immediately)
**Owner**: Backend/eBPF Engineer
**Estimated**: 12h
**Files**:
- `src/bpf/headers/process_monitor.h` (new)
- `src/bpf/headers/common_types.h` (modify - extend flow_event)

**Scope**:
- Define `process_cache_entry` structure
- Define `process_event` structure
- Define `process_info_map` (LRU Hash)
- Define `process_events` (Ring Buffer)
- Extend `flow_event` with process fields

**Dependencies**: None

**Conflicts**: None (new files, isolated structure changes)

---

### Stream B: Tracepoint Program Implementation (Priority: HIGH, Depends on A)
**Owner**: eBPF Specialist
**Estimated**: 40h
**Files**:
- `src/bpf/process_monitor.bpf.c` (new)

**Scope**:
- Implement `tp/sched/sched_process_exec` tracepoint handler
- Get process PID/comm/exec_time
- Cache to `process_info_map`
- Send events to `process_events` ring buffer
- Implement basic error handling

**Dependencies**: Stream A (needs data structures)

**Conflicts**: None

---

### Stream C: Container ID Extraction (Priority: MEDIUM, Depends on B)
**Owner**: eBPF Specialist + Container Expert
**Estimated**: 16h
**Files**:
- `src/bpf/process_monitor.bpf.c` (modify)
- `src/bpf/headers/process_monitor.h` (modify)

**Scope**:
- Implement `extract_container_id_from_cgroup()` helper
- Use BTF CO-RE to read task->cgroups
- Parse cgroup paths (Docker/Kubernetes/containerd)
- Extract container ID from path
- Handle edge cases (host processes, nested containers)

**Dependencies**: Stream B (needs tracepoint framework)

**Conflicts**: Stream B (same file, sequential)

---

### Stream D: Build Integration (Priority: MEDIUM, Can start immediately)
**Owner**: Build Engineer
**Estimated**: 8h
**Files**:
- `Makefile` or `src/bpf/Makefile`
- Build scripts
- `.gitignore` (if needed)

**Scope**:
- Add `process_monitor.bpf.c` to build targets
- Configure bpf2go code generation
- Ensure BTF/CO-RE compilation flags
- Update clean targets

**Dependencies**: None (can prepare in parallel)

**Conflicts**: None

---

### Stream E: Testing & Validation (Priority: LOW, Depends on A, B, C)
**Owner**: QA Engineer
**Estimated**: 20h
**Files**:
- `src/bpf/tests/process_monitor_test.c` (new, if applicable)
- Test scripts

**Scope**:
- Unit tests for tracepoint capture
- Container ID extraction accuracy tests
- Performance testing (capture rate > 95%)
- Integration testing with existing flow_event

**Dependencies**: All previous streams

**Conflicts**: None

---

## Execution Plan

**Phase 1 (Parallel Start)**:
- Launch Stream A (data structures) - immediate
- Launch Stream D (build config) - immediate

**Phase 2 (After A completes)**:
- Launch Stream B (tracepoint implementation)

**Phase 3 (After B completes)**:
- Launch Stream C (container ID extraction)

**Phase 4 (After A, B, C complete)**:
- Launch Stream E (testing)

**Critical Path**: A → B → C → E (76h on critical path)

**Parallel Savings**: Stream D runs in parallel, saves 8h
Total calendar time: ~68h with parallelization

## Coordination Notes

- Stream A and D can start immediately and run in parallel
- Stream B must wait for Stream A to define data structures
- Stream C must wait for Stream B to establish tracepoint framework
- All streams must complete before Stream E can fully execute
- No file conflicts between A and D (different files)
- Sequential dependency: A → B → C (same domain, building on each other)
