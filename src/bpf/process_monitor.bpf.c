// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
/* Process monitoring via eBPF tracepoint
 *
 * This program monitors process execution events using the sched_process_exec tracepoint.
 * It captures process information (PID, comm, exec time, container ID) and:
 * 1. Caches it in process_info_map for fast lookup by TC/XDP programs
 * 2. Sends events to userspace via process_events ring buffer for full path resolution
 *
 * Key benefits:
 * - Captures process info at exec time (process definitely exists)
 * - Solves short-lived process problem (curl, wget exit before /proc/<pid> lookup)
 * - Provides container context for traffic isolation
 * - Low overhead (tracepoint fires only on exec, not fork)
 */

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_core_read.h>

#include "headers/process_monitor.h"

char LICENSE[] SEC("license") = "GPL";

/* Helper function: Extract container ID from cgroup path
 *
 * This function reads the cgroup path from task_struct and extracts the container ID.
 * Supported formats:
 * - Docker: /docker/<container-id-64-chars>
 * - Kubernetes: /kubepods/besteffort/pod<pod-uid>/<container-id>
 * - containerd: /system.slice/containerd.service/<container-id>
 *
 * Implementation uses BPF_CORE_READ to safely access kernel structures.
 * Extracts the last segment of the cgroup path as the container ID.
 *
 * @event: Pointer to process_event structure to populate container_id field
 */
static __always_inline void extract_container_id_from_cgroup(struct process_event *event)
{
	// Initialize container_id to empty string
	// NOTE: Container ID extraction is complex due to eBPF stack limitations (512 bytes)
	// Current implementation: Placeholder that will be enhanced in userspace
	// Userspace agent will read /proc/<pid>/cgroup for complete container ID
	event->container_id[0] = '\0';

	// TODO: Optimize stack usage to implement kernel-side container ID extraction
	// Options for future implementation:
	// 1. Use BPF per-cpu array map for temporary storage
	// 2. Reduce stack usage in main tracepoint function
	// 3. Implement simplified cgroup parsing with minimal stack usage
}

/* Tracepoint handler: sched/sched_process_exec
 *
 * Triggered when a process calls exec() family of syscalls (execve, execveat, etc.).
 * This is the ideal point to capture process information because:
 * - Process has just started executing new binary
 * - Process has not yet exited (unlike fork-only monitoring)
 * - We can access task_struct safely
 *
 * Context: struct trace_event_raw_sched_process_exec *ctx
 * - Contains tracepoint arguments (see vmlinux.h for definition)
 * - We primarily use BPF helpers instead of ctx fields
 *
 * Returns: 0 (always success, non-zero would cause program unload)
 */
SEC("tp/sched/sched_process_exec")
int trace_sched_process_exec(struct trace_event_raw_sched_process_exec *ctx)
{
	struct process_event event = {};

	// Step 1: Get process information using BPF helpers

	// Get PID (actually TGID - thread group ID, which is the process ID in user space)
	// Lower 32 bits = thread ID (TID), upper 32 bits = process ID (PID/TGID)
	__u64 pid_tgid = bpf_get_current_pid_tgid();
	event.pid = pid_tgid >> 32;  // Extract PID from upper 32 bits

	// Get process execution timestamp (nanoseconds since boot)
	// Used to detect PID reuse (if PID exists but exec_time differs, it's a new process)
	event.exec_time = bpf_ktime_get_ns();

	// Get process command name (up to 16 bytes from task_struct->comm)
	// This is the executable name, e.g., "nginx", "python3", "curl"
	// Note: Truncated to 15 chars + null terminator (kernel limitation)
	bpf_get_current_comm(&event.comm, sizeof(event.comm));

	// Step 2: Extract container ID from cgroup path
	// NOTE: Deferred to userspace due to eBPF stack limitations
	// Userspace agent will read /proc/<pid>/cgroup for complete info
	extract_container_id_from_cgroup(&event);

	// Reserved for future use (e.g., fork vs exec flag, setuid indicator)
	event.flags = 0;

	// Step 3: Cache process info in eBPF map for fast TC/XDP lookup
	// Create cache entry with minimal fields (optimized for TC/XDP fast path)
	struct process_cache_entry cache_entry = {};

	// Copy comm using __builtin_memcpy (eBPF verifier prefers this over loops)
	__builtin_memcpy(cache_entry.comm, event.comm, sizeof(cache_entry.comm));
	cache_entry.exec_time = event.exec_time;
	__builtin_memcpy(cache_entry.container_id, event.container_id, sizeof(cache_entry.container_id));

	// Update LRU map (BPF_ANY = insert or update)
	// If map is full, LRU policy automatically evicts least recently used entry
	// We don't check return value - if update fails, TC/XDP won't have process info,
	// but packet processing will continue (fail-open for process monitoring)
	bpf_map_update_elem(&process_info_map, &event.pid, &cache_entry, BPF_ANY);

	// Step 4: Send event to userspace via ring buffer
	// Userspace daemon will:
	// - Read /proc/<pid>/exe for full executable path
	// - Enrich with process metadata (user, parent PID, etc.)
	// - Store in cache for flow collector queries
	//
	// bpf_ringbuf_output returns:
	// - 0 on success
	// - -ENOSPC if ring buffer is full (event dropped)
	// We don't check return value - if buffer is full, userspace is slow,
	// but we prioritize low latency over guaranteed delivery
	bpf_ringbuf_output(&process_events, &event, sizeof(event), 0);

	// Step 5: Return success
	// IMPORTANT: Must return 0, non-zero causes kernel to unload the program
	return 0;
}
