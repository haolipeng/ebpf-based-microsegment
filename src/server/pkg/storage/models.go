package storage

import (
	"time"
)

// Flow represents a row in the flows table.
// This legacy struct is kept for backward compatibility with tests.
type Flow struct {
	ID           int64             `json:"id"`
	AgentID      string            `json:"agent_id"`
	SrcIP        string            `json:"src_ip"`
	DstIP        string            `json:"dst_ip"`
	SrcPort      int32             `json:"src_port"`
	DstPort      int32             `json:"dst_port"`
	Protocol     int32             `json:"protocol"`
	Direction    int32             `json:"direction"`
	PacketCount  int64             `json:"packet_count"`
	ByteCount    int64             `json:"byte_count"`
	PolicyID     *int32            `json:"policy_id"`
	PolicyAction *int32            `json:"policy_action"`
	State        *int32            `json:"state"`
	TimestampNS  int64             `json:"timestamp_ns"`
	StartTime    *time.Time        `json:"start_time"`
	EndTime      *time.Time        `json:"end_time"`
	LastSeen     *time.Time        `json:"last_seen"`
	SourceLabels map[string]string `json:"source_labels"`
	DestLabels   map[string]string `json:"dest_labels"`
	CreatedAt    time.Time         `json:"created_at"`
}

// SecurityAlert represents a row in the security_alerts table.
// This legacy struct is kept for backward compatibility with tests.
type SecurityAlert struct {
	ID          int64             `json:"id"`
	AlertID     string            `json:"alert_id"`
	Level       int32             `json:"level"`
	Type        int32             `json:"type"`
	PID         *uint32           `json:"pid"`
	PPID        *uint32           `json:"ppid"`
	UID         *uint32           `json:"uid"`
	GID         *uint32           `json:"gid"`
	Comm        *string           `json:"comm"`
	ExePath     *string           `json:"exe_path"`
	ContainerID *string           `json:"container_id"`
	FlowID      *int64            `json:"flow_id"`
	SrcIP       *string           `json:"src_ip"`
	DstIP       *string           `json:"dst_ip"`
	DstPort     *uint32           `json:"dst_port"`
	Protocol    *string           `json:"protocol"`
	Reason      string            `json:"reason"`
	Metadata    map[string]string `json:"metadata"`
	Timestamp   int64             `json:"timestamp"`
	CreatedAt   time.Time         `json:"created_at"`
	AgentID     *string           `json:"agent_id"`
}
