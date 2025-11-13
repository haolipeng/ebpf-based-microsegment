package models

import (
	"time"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/flow"
)

// FlowListResponse represents the response for listing flows
type FlowListResponse struct {
	Flows []*flow.Flow  `json:"flows"`
	Count int           `json:"count"`
	Query FlowQueryInfo `json:"query"`
}

// FlowQueryInfo contains query metadata
type FlowQueryInfo struct {
	Limit     int    `json:"limit"`
	Offset    int    `json:"offset"`
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
}

// FlowMetricsResponse represents collector metrics
type FlowMetricsResponse struct {
	EventsProcessed uint64  `json:"events_processed"`
	EventsDropped   uint64  `json:"events_dropped"`
	ActiveFlows     int     `json:"active_flows"`
	DropRate        float64 `json:"drop_rate_percent"`
}

// DependencyListResponse represents the response for dependencies
type DependencyListResponse struct {
	Dependencies []*flow.Dependency `json:"dependencies"`
	Count        int                `json:"count"`
	TimeRange    TimeRangeInfo      `json:"time_range"`
}

// TopTalkersResponse represents the response for top talkers
type TopTalkersResponse struct {
	TopTalkers []flow.IPStats `json:"top_talkers"`
	Count      int            `json:"count"`
	TimeRange  TimeRangeInfo  `json:"time_range"`
}

// TimeRangeInfo contains time range metadata
type TimeRangeInfo struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}
