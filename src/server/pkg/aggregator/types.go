package aggregator

import (
	"time"
)

// FlowDependency represents a dependency between two workloads
type FlowDependency struct {
	SourceLabel  string   `json:"source_label"`
	DestLabel    string   `json:"dest_label"`
	FlowCount    int64    `json:"flow_count"`
	TotalBytes   int64    `json:"total_bytes"`
	TotalPackets int64    `json:"total_packets"`
	Protocols    []string `json:"protocols"`
	AvgDuration  float64  `json:"avg_duration_ms"`
}

// TopTalker represents a high-traffic endpoint
type TopTalker struct {
	IPAddress    string  `json:"ip_address"`
	Hostname     string  `json:"hostname,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	TotalBytes   int64   `json:"total_bytes"`
	TotalPackets int64   `json:"total_packets"`
	FlowCount    int64   `json:"flow_count"`
	Direction    string  `json:"direction"` // "source", "destination", or "both"
}

// AggregationResult represents aggregated flow statistics
type AggregationResult struct {
	GroupBy      string                 `json:"group_by"`
	TimeRange    TimeRange              `json:"time_range"`
	Dependencies []*FlowDependency      `json:"dependencies"`
	TopTalkers   *TopTalkersResult      `json:"top_talkers,omitempty"`
	Summary      *AggregationSummary    `json:"summary"`
}

// TopTalkersResult contains top talkers by different criteria
type TopTalkersResult struct {
	ByBytes     []*TopTalker `json:"by_bytes"`
	ByPackets   []*TopTalker `json:"by_packets"`
	ByFlowCount []*TopTalker `json:"by_flow_count"`
}

// AggregationSummary contains overall statistics
type AggregationSummary struct {
	TotalFlows      int64   `json:"total_flows"`
	TotalBytes      int64   `json:"total_bytes"`
	TotalPackets    int64   `json:"total_packets"`
	UniqueEndpoints int64   `json:"unique_endpoints"`
	AvgFlowDuration float64 `json:"avg_flow_duration_ms"`
}

// TimeRange represents a time range for aggregation
type TimeRange struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// AggregationQuery defines parameters for aggregation
type AggregationQuery struct {
	TimeRange       TimeRange
	GroupBy         string   // Label key to group by (e.g., "app", "env", "tier")
	IncludeTopTalkers bool
	TopN            int      // Number of top talkers to return
	Protocols       []string // Filter by protocols
	AgentIDs        []string // Filter by agent IDs
}
