package storage

import (
	"time"

	"gorm.io/datatypes"
)

// Flow represents a row in the flows table for GORM operations.
type Flow struct {
	ID           int64             `gorm:"column:id;primaryKey;autoIncrement"`
	AgentID      string            `gorm:"column:agent_id"`
	SrcIP        string            `gorm:"column:src_ip"`
	DstIP        string            `gorm:"column:dst_ip"`
	SrcPort      int32             `gorm:"column:src_port"`
	DstPort      int32             `gorm:"column:dst_port"`
	Protocol     int32             `gorm:"column:protocol"`
	Direction    int32             `gorm:"column:direction"`
	PacketCount  int64             `gorm:"column:packet_count"`
	ByteCount    int64             `gorm:"column:byte_count"`
	PolicyID     *int32            `gorm:"column:policy_id"`
	PolicyAction *int32            `gorm:"column:policy_action"`
	State        *int32            `gorm:"column:state"`
	TimestampNS  int64             `gorm:"column:timestamp_ns"`
	StartTime    *time.Time        `gorm:"column:start_time"`
	EndTime      *time.Time        `gorm:"column:end_time"`
	LastSeen     *time.Time        `gorm:"column:last_seen"`
	SourceLabels datatypes.JSONMap `gorm:"column:source_labels"`
	DestLabels   datatypes.JSONMap `gorm:"column:dest_labels"`
	CreatedAt    time.Time         `gorm:"column:created_at"`
}

// TableName overrides the default pluralized table name.
func (Flow) TableName() string {
	return "flows"
}

// SecurityAlert represents a row in the security_alerts table for GORM operations.
type SecurityAlert struct {
	ID          int64             `gorm:"column:id;primaryKey;autoIncrement"`
	AlertID     string            `gorm:"column:alert_id;uniqueIndex;not null"`
	Level       int32             `gorm:"column:level;not null"`
	Type        int32             `gorm:"column:type;not null"`
	PID         *uint32           `gorm:"column:pid"`
	PPID        *uint32           `gorm:"column:ppid"`
	UID         *uint32           `gorm:"column:uid"`
	GID         *uint32           `gorm:"column:gid"`
	Comm        *string           `gorm:"column:comm;size:16"`
	ExePath     *string           `gorm:"column:exe_path;size:1024"`
	ContainerID *string           `gorm:"column:container_id;size:64"`
	FlowID      *int64            `gorm:"column:flow_id"`
	SrcIP       *string           `gorm:"column:src_ip"`
	DstIP       *string           `gorm:"column:dst_ip"`
	DstPort     *uint32           `gorm:"column:dst_port"`
	Protocol    *string           `gorm:"column:protocol;size:16"`
	Reason      string            `gorm:"column:reason;type:text;not null"`
	Metadata    datatypes.JSONMap `gorm:"column:metadata;type:jsonb"`
	Timestamp   int64             `gorm:"column:timestamp;not null"`
	CreatedAt   time.Time         `gorm:"column:created_at;autoCreateTime"`
	AgentID     *string           `gorm:"column:agent_id;size:64"`
}

// TableName overrides the default pluralized table name.
func (SecurityAlert) TableName() string {
	return "security_alerts"
}
