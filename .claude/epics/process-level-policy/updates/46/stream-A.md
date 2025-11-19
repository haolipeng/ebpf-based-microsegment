---
issue: 46
stream: Data Structure Definition
agent: general-purpose
started: 2025-11-19T13:01:31Z
completed: 2025-11-19T21:08:00Z
status: completed
---

# Stream A: Data Structure Definition

## Summary
Successfully created process monitoring data structures and extended flow_event with process context fields.

## Scope
Define all data structures needed for process monitoring:
- ✅ `process_cache_entry` - eBPF map storage
- ✅ `process_event` - Ring buffer events
- ✅ `process_info_map` - LRU Hash map definition (10000 entries)
- ✅ `process_events` - Ring Buffer definition (256KB)
- ✅ Extend `flow_event` with process fields

## Files Modified
- ✅ `src/bpf/headers/process_monitor.h` (new, 118 lines)
- ✅ `src/bpf/headers/common_types.h` (modified, added process context)

## Completed Tasks

### 1. Created `src/bpf/headers/process_monitor.h`
- Defined `struct process_cache_entry` for eBPF map storage
  - Fields: comm[16], exec_time, container_id[64]
  - Size: 80 bytes (packed, aligned)

- Defined `struct process_event` for ring buffer
  - Fields: pid, comm[16], exec_time, container_id[64], flags
  - Size: 96 bytes (packed, aligned)

- Defined `process_info_map` (LRU Hash)
  - Type: BPF_MAP_TYPE_LRU_HASH
  - Max entries: 10000
  - Key: __u32 (PID), Value: struct process_cache_entry

- Defined `process_events` (Ring Buffer)
  - Type: BPF_MAP_TYPE_RINGBUF
  - Size: 256KB (~2700 events buffered)

### 2. Extended `struct flow_event` in common_types.h
Added process context fields (92 bytes total):
- `char process_name[16]` - Process command name
- `__u32 pid` - Process ID (0 if not available)
- `char container_id[64]` - Container ID
- `__u64 process_exec_time` - Process execution timestamp

### 3. Compilation Verification
- ✅ TC eBPF program compilation: SUCCESS
- ✅ XDP eBPF program compilation: SUCCESS
- ✅ No compiler warnings or errors

## Design Decisions

1. **Separate structures for cache and events**:
   - Cache entry is minimal for fast TC/XDP lookup
   - Event structure includes PID for userspace processing

2. **Fixed-size arrays**:
   - No pointers (eBPF verifier requirement)
   - Container ID: 64 bytes (sufficient for Docker/K8s)
   - Comm: 16 bytes (matches task_struct->comm)

3. **LRU eviction strategy**:
   - Automatic cleanup, no manual GC needed
   - 10000 entries for typical workloads

4. **Ring buffer sizing**:
   - 256KB handles burst process creation
   - ~2700 events buffered

## Integration Notes

### For Stream B (Tracepoint Program)
- Import `process_monitor.h`
- Use `bpf_map_update_elem()` for `process_info_map`
- Use `bpf_ringbuf_output()` for `process_events`

### For Stream C (Container ID)
- Implement `extract_container_id_from_cgroup()`
- Parse Docker/K8s cgroup formats
- Write to `process_event.container_id`

### For TC/XDP Programs
- Include `process_monitor.h`
- Lookup PID in `process_info_map`
- Copy to `flow_event` before sending to userspace

## Testing Results
```bash
# Compilation tests
clang -target bpf -O2 -g -Wall -I. -I./headers -I../../vmlinux/x86 \
  -c tc_microsegment.bpf.c -o /tmp/test_tc.o
# SUCCESS

clang -target bpf -O2 -g -Wall -I. -I./headers -I../../vmlinux/x86 \
  -c xdp_microsegment.bpf.c -o /tmp/test_xdp.o
# SUCCESS
```

## Completion Criteria Met
- ✅ Both header files created/modified
- ✅ Data structures properly defined
- ✅ Compilation verified without errors
- ✅ English comments throughout
- ✅ Followed existing code patterns
- ✅ Progress file updated

## Status
**COMPLETED** - Ready for Stream B and C to proceed
