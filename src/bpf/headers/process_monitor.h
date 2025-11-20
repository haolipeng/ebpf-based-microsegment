// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Process monitoring data structures for eBPF tracepoint
 *
 * This header defines data structures for process monitoring via eBPF tracepoint.
 * It provides:
 * - process_cache_entry: Cached process information in eBPF map (for fast lookup)
 * - process_event: Process events sent to userspace via ring buffer
 * - process_info_map: LRU hash map for caching process information (10000 entries)
 * - process_events: Ring buffer for sending process events to userspace (256KB)
 *
 * Prerequisites:
 * - Include vmlinux.h (for kernel types and BPF helpers)
 * - Include bpf/bpf_helpers.h (for BPF map macros)
 *
 * Usage:
 * This header is designed to be used by:
 * 1. process_monitor.bpf.c - Tracepoint program that captures process exec events
 * 2. TC/XDP programs - Fast lookup of process information via process_info_map
 * 3. Userspace programs - Reading process events from process_events ring buffer
 */

#ifndef __PROCESS_MONITOR_H__
#define __PROCESS_MONITOR_H__

/* Process cache entry stored in eBPF map
 *
 * This structure contains minimal process information for fast lookup by TC/XDP programs.
 * Design considerations:
 * - Keep size small for efficient memory usage (aligned to 80 bytes)
 * - No pointers (eBPF verifier requirement for map values)
 * - Fixed-size arrays for compatibility across kernel versions
 *
 * Map key: PID (__u32)
 * Map type: BPF_MAP_TYPE_LRU_HASH (automatic eviction of least recently used entries)
 */
struct process_cache_entry {
	char comm[16];              // Process command name (from task_struct->comm)
	__u64 exec_time;            // Process execution timestamp (nanoseconds since boot)
	char container_id[64];      // Container ID extracted from cgroup path
} __attribute__((packed));

/* Process event sent to userspace via ring buffer
 *
 * This structure contains complete process information for userspace processing.
 * Design considerations:
 * - Include PID (not in cache entry since it's the map key)
 * - Match cache entry fields for consistency
 * - Add flags field for future extensibility
 * - Total size: 4 + 16 + 8 + 64 + 4 = 96 bytes (aligned)
 *
 * Userspace use cases:
 * - Full executable path resolution (/proc/<pid>/exe)
 * - Process metadata enrichment (user, group, parent PID)
 * - Audit logging and forensics
 */
struct process_event {
	__u32 pid;                  // Process ID (TGID from bpf_get_current_pid_tgid)
	char comm[16];              // Process command name (from task_struct->comm)
	__u64 exec_time;            // Execution timestamp (nanoseconds since boot)
	char container_id[64];      // Container ID extracted from cgroup path
	__u32 flags;                // Reserved for future use (e.g., fork/exec flags)
} __attribute__((packed));

/* eBPF Maps */

/* process_info_map - LRU hash map for caching process information
 *
 * Type: BPF_MAP_TYPE_LRU_HASH
 * - Automatically evicts least recently used entries when full
 * - No manual cleanup required
 * - O(1) lookup time for TC/XDP fast path
 *
 * Max entries: 10000
 * - Sufficient for typical workloads (thousands of concurrent processes)
 * - Memory usage: 10000 * (4 + 80) = ~820KB
 *
 * Key: __u32 (PID)
 * Value: struct process_cache_entry
 *
 * Access pattern:
 * - Write: tracepoint program on sched_process_exec
 * - Read: TC/XDP programs during packet processing
 */
struct {
	__uint(type, BPF_MAP_TYPE_LRU_HASH);
	__uint(max_entries, 10000);
	__type(key, __u32);         // PID
	__type(value, struct process_cache_entry);
} process_info_map SEC(".maps");

/* process_events - Ring buffer for sending process events to userspace
 *
 * Type: BPF_MAP_TYPE_RINGBUF
 * - Lock-free, high-performance circular buffer
 * - Supports variable-sized events (though we use fixed size)
 * - Userspace can poll/epoll for new events
 *
 * Max entries: 256KB (262144 bytes)
 * - Can buffer ~2700 process events (96 bytes each)
 * - Sufficient for burst of process creation
 *
 * Event: struct process_event
 *
 * Access pattern:
 * - Write: tracepoint program on sched_process_exec
 * - Read: userspace daemon via bpf_ringbuf__consume()
 */
struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 256 * 1024);  // 256KB
} process_events SEC(".maps");

#endif /* __PROCESS_MONITOR_H__ */
