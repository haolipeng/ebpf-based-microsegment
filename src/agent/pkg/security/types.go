package security

import (
	"time"
)

// AlertLevel represents the severity level of a security alert
type AlertLevel string

const (
	// AlertLevelInfo indicates informational alerts
	AlertLevelInfo AlertLevel = "INFO"

	// AlertLevelWarning indicates potential security concerns
	AlertLevelWarning AlertLevel = "WARNING"

	// AlertLevelCritical indicates critical security threats
	AlertLevelCritical AlertLevel = "CRITICAL"
)

// AlertType represents the type of security alert
type AlertType string

const (
	// AlertTypeSuspiciousPath indicates a process running from a suspicious directory
	AlertTypeSuspiciousPath AlertType = "SUSPICIOUS_PATH"

	// AlertTypePrivilegeEscalation indicates a privileged process from untrusted location
	AlertTypePrivilegeEscalation AlertType = "PRIVILEGE_ESCALATION"

	// AlertTypeNameMismatch indicates process name doesn't match executable
	AlertTypeNameMismatch AlertType = "NAME_MISMATCH"

	// AlertTypeAnomalousConnection indicates unexpected network behavior
	AlertTypeAnomalousConnection AlertType = "ANOMALOUS_CONNECTION"

	// AlertTypeDeletedExecutable indicates process running from deleted executable
	AlertTypeDeletedExecutable AlertType = "DELETED_EXECUTABLE"

	// AlertTypeHiddenExecutable indicates process running from hidden file
	AlertTypeHiddenExecutable AlertType = "HIDDEN_EXECUTABLE"
)

// PathCategory represents the classification of a process path
type PathCategory string

const (
	// PathCategorySystem indicates a system binary location
	PathCategorySystem PathCategory = "SYSTEM"

	// PathCategoryUser indicates a user-owned location
	PathCategoryUser PathCategory = "USER"

	// PathCategorySuspicious indicates a suspicious location
	PathCategorySuspicious PathCategory = "SUSPICIOUS"

	// PathCategoryUnknown indicates an unclassified location
	PathCategoryUnknown PathCategory = "UNKNOWN"
)

// SecurityAlert represents a security alert event
type SecurityAlert struct {
	// AlertID is the unique identifier for this alert
	AlertID string `json:"alert_id"`

	// Level is the severity level
	Level AlertLevel `json:"level"`

	// Type is the alert type
	Type AlertType `json:"type"`

	// ProcessInfo contains information about the suspicious process
	ProcessInfo ProcessInfo `json:"process_info"`

	// FlowInfo contains network flow information (optional)
	FlowInfo *FlowInfo `json:"flow_info,omitempty"`

	// Reason is a human-readable explanation
	Reason string `json:"reason"`

	// Timestamp is when the alert was generated
	Timestamp time.Time `json:"timestamp"`

	// Metadata contains additional context
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ProcessInfo contains information about a process
type ProcessInfo struct {
	PID         uint32 `json:"pid"`
	Comm        string `json:"comm"`
	Path        string `json:"path"`
	ContainerID string `json:"container_id,omitempty"`
	UID         uint32 `json:"uid,omitempty"`
}

// FlowInfo contains network flow information
type FlowInfo struct {
	SourceIP   string `json:"source_ip"`
	SourcePort uint16 `json:"source_port"`
	DestIP     string `json:"dest_ip"`
	DestPort   uint16 `json:"dest_port"`
	Protocol   string `json:"protocol"`
}

// PathValidationResult represents the result of path validation
type PathValidationResult struct {
	// Category is the path classification
	Category PathCategory `json:"category"`

	// IsSuspicious indicates if the path is suspicious
	IsSuspicious bool `json:"is_suspicious"`

	// Reasons contains the reasons for suspicion
	Reasons []string `json:"reasons,omitempty"`

	// Confidence is the confidence level (0-100)
	Confidence int `json:"confidence"`
}
