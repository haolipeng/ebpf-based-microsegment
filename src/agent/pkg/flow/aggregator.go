// input: flow events (same 5-tuple)
// output: aggregated flow statistics (packet count, byte count, duration)
// pos: flow event aggregator - if file updated, must sync with this header comment and pkg/flow/CLAUDE.md
package flow

import (
	"fmt"
	"time"
)

// Aggregator provides flow aggregation and dependency analysis
type Aggregator struct {
	storage Storage
}

// NewAggregator creates a new flow aggregator
func NewAggregator(storage Storage) *Aggregator {
	return &Aggregator{
		storage: storage,
	}
}

// GetDependencies analyzes flows to extract workload dependencies
// Returns aggregated dependencies based on labels within the specified time range
func (a *Aggregator) GetDependencies(startTime, endTime time.Time, minFlows int) ([]*Dependency, error) {
	// Query all flows in the time range
	query := &FlowQuery{
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     10000, // Large limit for aggregation
	}

	flows, err := a.storage.QueryFlows(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query flows: %w", err)
	}

	// Group flows by source/dest label combinations
	depMap := make(map[string]*Dependency)

	for _, flow := range flows {
		// Skip flows without labels
		if len(flow.SourceLabels) == 0 || len(flow.DestLabels) == 0 {
			continue
		}

		// Create dependency key from sorted labels
		depKey := makeDependencyKey(flow.SourceLabels, flow.DestLabels)

		dep, exists := depMap[depKey]
		if !exists {
			// Create new dependency
			dep = &Dependency{
				SourceLabels: copyLabels(flow.SourceLabels),
				DestLabels:   copyLabels(flow.DestLabels),
				FlowCount:    0,
				PacketCount:  0,
				ByteCount:    0,
				Protocols:    make([]string, 0),
				LastSeen:     flow.LastSeen,
			}
			depMap[depKey] = dep
		}

		// Aggregate statistics
		dep.FlowCount++
		dep.PacketCount += flow.PacketCount
		dep.ByteCount += flow.ByteCount

		// Update last seen
		if flow.LastSeen.After(dep.LastSeen) {
			dep.LastSeen = flow.LastSeen
		}

		// Track unique protocols
		if !contains(dep.Protocols, flow.Protocol) {
			dep.Protocols = append(dep.Protocols, flow.Protocol)
		}
	}

	// Filter by minimum flow count and convert to slice
	dependencies := make([]*Dependency, 0, len(depMap))
	for _, dep := range depMap {
		if dep.FlowCount >= int64(minFlows) {
			dependencies = append(dependencies, dep)
		}
	}

	return dependencies, nil
}

// GetTopTalkers returns the top N source IPs by traffic volume
func (a *Aggregator) GetTopTalkers(startTime, endTime time.Time, limit int, sortBy string) ([]IPStats, error) {
	// This would typically be implemented with a SQL query for efficiency
	// For now, we'll use the summary API
	summary, err := a.storage.GetFlowSummary(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}

	// Return top source IPs (already sorted by flow count)
	topN := limit
	if topN > len(summary.TopSourceIPs) {
		topN = len(summary.TopSourceIPs)
	}

	return summary.TopSourceIPs[:topN], nil
}

// GetProtocolDistribution returns protocol distribution statistics
func (a *Aggregator) GetProtocolDistribution(startTime, endTime time.Time) ([]ProtocolStats, error) {
	summary, err := a.storage.GetFlowSummary(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}

	return summary.TopProtocols, nil
}

// Helper: makeDependencyKey creates a unique key for a source/dest label combination
func makeDependencyKey(srcLabels, dstLabels map[string]string) string {
	// Use a canonical representation of labels for consistent grouping
	// Format: "app:nginx,env:prod|app:redis,env:prod"
	srcKey := labelsToString(srcLabels)
	dstKey := labelsToString(dstLabels)
	return fmt.Sprintf("%s|%s", srcKey, dstKey)
}

// Helper: labelsToString converts labels map to sorted string
func labelsToString(labels map[string]string) string {
	if len(labels) == 0 {
		return "unlabeled"
	}

	// For simplicity, use a single key-value pair (typically "app" label)
	// In production, you might want to sort all labels for consistency
	if app, ok := labels["app"]; ok {
		return fmt.Sprintf("app:%s", app)
	}
	if name, ok := labels["name"]; ok {
		return fmt.Sprintf("name:%s", name)
	}

	// Fallback to first label
	for k, v := range labels {
		return fmt.Sprintf("%s:%s", k, v)
	}

	return "unlabeled"
}

// Helper: copyLabels creates a copy of labels map
func copyLabels(labels map[string]string) map[string]string {
	copy := make(map[string]string, len(labels))
	for k, v := range labels {
		copy[k] = v
	}
	return copy
}

// Helper: contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
