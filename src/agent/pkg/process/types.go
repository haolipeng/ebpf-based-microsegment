
// input: N/A (type definition)
// output: process data structures
// pos: process type definitions - if file updated, must sync with this header comment and pkg/process/CLAUDE.md
package process

import "time"

// ProcessInfo represents complete process information with cache metadata
type ProcessInfo struct {
	PID          uint32    `json:"pid"`
	Comm         string    `json:"comm"`          // Process command name (16 chars max)
	Path         string    `json:"path"`          // Full executable path from /proc/<pid>/exe
	ContainerID  string    `json:"container_id"`  // Container ID (if containerized)
	ExecTime     uint64    `json:"exec_time"`     // Process start timestamp (nanoseconds)
	CachedTime   time.Time `json:"cached_time"`   // When this info was cached
	PathResolved bool      `json:"path_resolved"` // Whether path resolution succeeded
}

// ProcessEvent represents the raw event from eBPF ring buffer (process_events)
// This structure must match the kernel-side process_event struct in process_monitor.h
type ProcessEvent struct {
	PID         uint32   // Process ID
	Comm        [16]byte // Process command name (null-terminated)
	ExecTime    uint64   // Execution timestamp (nanoseconds)
	ContainerID [64]byte // Container ID (null-terminated, extracted in userspace)
	Flags       uint32   // Event flags (reserved for future use)
}

// ToProcessInfo converts ProcessEvent to ProcessInfo with basic fields
// Path resolution and full enrichment happens separately
func (e *ProcessEvent) ToProcessInfo() *ProcessInfo {
	return &ProcessInfo{
		PID:          e.PID,
		Comm:         nullTerminatedString(e.Comm[:]),
		ContainerID:  nullTerminatedString(e.ContainerID[:]),
		ExecTime:     e.ExecTime,
		CachedTime:   time.Now(),
		PathResolved: false, // Will be set to true after path resolution
	}
}

// nullTerminatedString converts a null-terminated byte array to a Go string
func nullTerminatedString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// IsValid checks if ProcessInfo has valid data
func (p *ProcessInfo) IsValid() bool {
	return p.PID > 0 && p.ExecTime > 0
}

// IsExpired checks if the cached process info has exceeded TTL
func (p *ProcessInfo) IsExpired(ttl time.Duration) bool {
	return time.Since(p.CachedTime) > ttl
}
