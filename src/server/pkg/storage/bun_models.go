package storage

import (
	"time"

	"github.com/uptrace/bun"
)

// BunFlow represents a row in the flows table for Bun operations.
type BunFlow struct {
	bun.BaseModel `bun:"table:flows,alias:f"`

	ID           int64             `bun:"id,pk,autoincrement"`
	AgentID      string            `bun:"agent_id"`
	SrcIP        string            `bun:"src_ip"`
	DstIP        string            `bun:"dst_ip"`
	SrcPort      int32             `bun:"src_port"`
	DstPort      int32             `bun:"dst_port"`
	Protocol     int32             `bun:"protocol"`
	Direction    int32             `bun:"direction"`
	PacketCount  int64             `bun:"packet_count"`
	ByteCount    int64             `bun:"byte_count"`
	PolicyID     *int32            `bun:"policy_id"`
	PolicyAction *int32            `bun:"policy_action"`
	State        *int32            `bun:"state"`
	TimestampNS  int64             `bun:"timestamp_ns"`
	StartTime    *time.Time        `bun:"start_time"`
	EndTime      *time.Time        `bun:"end_time"`
	LastSeen     *time.Time        `bun:"last_seen"`
	SourceLabels map[string]string `bun:"source_labels,type:jsonb"` // Native JSONB support!
	DestLabels   map[string]string `bun:"dest_labels,type:jsonb"`   // Native JSONB support!
	CreatedAt    time.Time         `bun:"created_at,default:current_timestamp"`

	// Process info fields
	SrcPID         *uint32 `bun:"src_pid"`
	SrcPPID        *uint32 `bun:"src_ppid"`
	SrcUID         *uint32 `bun:"src_uid"`
	SrcGID         *uint32 `bun:"src_gid"`
	SrcComm        *string `bun:"src_comm"`
	SrcExePath     *string `bun:"src_exe_path"`
	SrcCmdline     *string `bun:"src_cmdline"`
	SrcContainerID *string `bun:"src_container_id"`
	DstPID         *uint32 `bun:"dst_pid"`
	DstComm        *string `bun:"dst_comm"`
}

// BunSecurityAlert represents a row in the security_alerts table for Bun operations.
type BunSecurityAlert struct {
	bun.BaseModel `bun:"table:security_alerts,alias:sa"`

	ID          int64             `bun:"id,pk,autoincrement"`
	AlertID     string            `bun:"alert_id,unique,notnull"`
	Level       int32             `bun:"level,notnull"`
	Type        int32             `bun:"type,notnull"`
	PID         *uint32           `bun:"pid"`
	PPID        *uint32           `bun:"ppid"`
	UID         *uint32           `bun:"uid"`
	GID         *uint32           `bun:"gid"`
	Comm        *string           `bun:"comm"`
	ExePath     *string           `bun:"exe_path"`
	ContainerID *string           `bun:"container_id"`
	FlowID      *int64            `bun:"flow_id"`
	SrcIP       *string           `bun:"src_ip"`
	DstIP       *string           `bun:"dst_ip"`
	DstPort     *uint32           `bun:"dst_port"`
	Protocol    *string           `bun:"protocol"`
	Reason      string            `bun:"reason,type:text,notnull"`
	Metadata    map[string]string `bun:"metadata,type:jsonb"` // Native JSONB support!
	Timestamp   int64             `bun:"timestamp,notnull"`
	CreatedAt   time.Time         `bun:"created_at,default:current_timestamp"`
	AgentID     *string           `bun:"agent_id"`
}

// BunAgent represents a row in the agents table for Bun operations.
type BunAgent struct {
	bun.BaseModel `bun:"table:agents,alias:a"`

	ID           int64             `bun:"id,pk,autoincrement"`
	AgentID      string            `bun:"agent_id,unique,notnull"`
	Hostname     string            `bun:"hostname"`
	IPAddress    string            `bun:"ip_address"`
	Version      string            `bun:"version"`
	Status       int32             `bun:"status"`
	Labels       map[string]string `bun:"labels,type:jsonb"`
	LastHeartbeat *time.Time       `bun:"last_heartbeat"`
	CreatedAt    time.Time         `bun:"created_at,default:current_timestamp"`
	UpdatedAt    time.Time         `bun:"updated_at,default:current_timestamp"`
}

// BunPolicy represents a row in the policies table for Bun operations.
type BunPolicy struct {
	bun.BaseModel `bun:"table:policies,alias:p"`

	ID             int64             `bun:"id,pk,autoincrement"`
	Name           string            `bun:"name,notnull"`
	Description    string            `bun:"description"`
	Priority       int32             `bun:"priority"`
	Action         int32             `bun:"action"`
	Direction      int32             `bun:"direction"`
	Protocol       *int32            `bun:"protocol"`
	SrcLabels      map[string]string `bun:"src_labels,type:jsonb"`
	DstLabels      map[string]string `bun:"dst_labels,type:jsonb"`
	SrcPorts       []int32           `bun:"src_ports,array"`  // Native array support!
	DstPorts       []int32           `bun:"dst_ports,array"`  // Native array support!
	Enabled        bool              `bun:"enabled,default:true"`
	CreatedAt      time.Time         `bun:"created_at,default:current_timestamp"`
	UpdatedAt      time.Time         `bun:"updated_at,default:current_timestamp"`

	// Process-based policy fields
	ProcessName    *string `bun:"process_name"`
	ProcessPath    *string `bun:"process_path"`
	ProcessHash    *string `bun:"process_hash"`
}

// BunPolicyStat represents policy enforcement statistics
type BunPolicyStat struct {
	bun.BaseModel `bun:"table:policy_stats,alias:ps"`

	ID          int64     `bun:"id,pk,autoincrement"`
	PolicyID    int64     `bun:"policy_id,notnull"`
	AgentID     string    `bun:"agent_id,notnull"`
	MatchCount  int64     `bun:"match_count,default:0"`
	AllowCount  int64     `bun:"allow_count,default:0"`
	DenyCount   int64     `bun:"deny_count,default:0"`
	LogCount    int64     `bun:"log_count,default:0"`
	BytesTotal  int64     `bun:"bytes_total,default:0"`
	LastMatchAt *time.Time `bun:"last_match_at"`
	CreatedAt   time.Time  `bun:"created_at,default:current_timestamp"`
	UpdatedAt   time.Time  `bun:"updated_at,default:current_timestamp"`
}
