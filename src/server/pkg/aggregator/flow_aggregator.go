package aggregator

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
)

// FlowAggregator performs aggregation and analysis on flow data
type FlowAggregator struct {
	db *sql.DB
}

// NewFlowAggregator creates a new FlowAggregator
func NewFlowAggregator(db *sql.DB) *FlowAggregator {
	return &FlowAggregator{db: db}
}

// GetDependencies returns application dependencies based on label aggregation
func (a *FlowAggregator) GetDependencies(ctx context.Context, query *AggregationQuery) ([]*FlowDependency, error) {
	// Build WHERE clause
	whereClause, args := a.buildWhereClause(query)
	argIdx := len(args) + 1

	// SQL query for label-based dependencies
	sqlQuery := fmt.Sprintf(`
		SELECT
			COALESCE(source_labels->>'%s', 'unknown') as source_label,
			COALESCE(dest_labels->>'%s', 'unknown') as dest_label,
			COUNT(*) as flow_count,
			SUM(byte_count) as total_bytes,
			SUM(packet_count) as total_packets,
			array_agg(DISTINCT protocol) as protocols,
			AVG(EXTRACT(EPOCH FROM (COALESCE(created_at, NOW()) - created_at)) * 1000) as avg_duration_ms
		FROM flows
		%s
		  AND source_labels ? $%d
		  AND dest_labels ? $%d
		GROUP BY source_label, dest_label
		ORDER BY flow_count DESC
		LIMIT 100
	`, query.GroupBy, query.GroupBy, whereClause, argIdx, argIdx+1)

	args = append(args, query.GroupBy, query.GroupBy)

	rows, err := a.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query dependencies: %w", err)
	}
	defer rows.Close()

	dependencies := []*FlowDependency{}
	for rows.Next() {
		var dep FlowDependency
		var protocolsJSON []byte
		var avgDuration sql.NullFloat64

		err := rows.Scan(
			&dep.SourceLabel,
			&dep.DestLabel,
			&dep.FlowCount,
			&dep.TotalBytes,
			&dep.TotalPackets,
			&protocolsJSON,
			&avgDuration,
		)
		if err != nil {
			logrus.Warnf("Failed to scan dependency: %v", err)
			continue
		}

		// Parse protocols array
		json.Unmarshal(protocolsJSON, &dep.Protocols)

		if avgDuration.Valid {
			dep.AvgDuration = avgDuration.Float64
		}

		dependencies = append(dependencies, &dep)
	}

	return dependencies, nil
}

// GetTopTalkers returns the top N endpoints by bytes/packets/flows
func (a *FlowAggregator) GetTopTalkers(ctx context.Context, query *AggregationQuery) (*TopTalkersResult, error) {
	whereClause, args := a.buildWhereClause(query)
	topN := query.TopN
	if topN == 0 {
		topN = 10 // Default to top 10
	}

	result := &TopTalkersResult{}

	// Get top talkers by bytes
	byBytes, err := a.getTopTalkersByMetric(ctx, "byte_count", whereClause, args, topN)
	if err != nil {
		return nil, fmt.Errorf("failed to get top talkers by bytes: %w", err)
	}
	result.ByBytes = byBytes

	// Get top talkers by packets
	byPackets, err := a.getTopTalkersByMetric(ctx, "packet_count", whereClause, args, topN)
	if err != nil {
		return nil, fmt.Errorf("failed to get top talkers by packets: %w", err)
	}
	result.ByPackets = byPackets

	// Get top talkers by flow count
	byFlowCount, err := a.getTopTalkersByFlowCount(ctx, whereClause, args, topN)
	if err != nil {
		return nil, fmt.Errorf("failed to get top talkers by flow count: %w", err)
	}
	result.ByFlowCount = byFlowCount

	return result, nil
}

// getTopTalkersByMetric gets top talkers sorted by a specific metric (bytes or packets)
func (a *FlowAggregator) getTopTalkersByMetric(ctx context.Context, metric, whereClause string, args []interface{}, topN int) ([]*TopTalker, error) {
	// Query both source and destination endpoints
	sqlQuery := fmt.Sprintf(`
		WITH source_stats AS (
			SELECT
				src_ip as ip_address,
				'source' as direction,
				SUM(%s) as total_metric,
				SUM(byte_count) as total_bytes,
				SUM(packet_count) as total_packets,
				COUNT(*) as flow_count,
				source_labels as labels
			FROM flows
			%s
			GROUP BY src_ip, source_labels
		),
		dest_stats AS (
			SELECT
				dst_ip as ip_address,
				'destination' as direction,
				SUM(%s) as total_metric,
				SUM(byte_count) as total_bytes,
				SUM(packet_count) as total_packets,
				COUNT(*) as flow_count,
				dest_labels as labels
			FROM flows
			%s
			GROUP BY dst_ip, dest_labels
		),
		combined AS (
			SELECT * FROM source_stats
			UNION ALL
			SELECT * FROM dest_stats
		)
		SELECT
			ip_address,
			direction,
			SUM(total_metric) as total_metric,
			SUM(total_bytes) as total_bytes,
			SUM(total_packets) as total_packets,
			SUM(flow_count) as flow_count,
			(array_agg(labels ORDER BY total_metric DESC))[1] as labels
		FROM combined
		GROUP BY ip_address, direction
		ORDER BY total_metric DESC
		LIMIT $%d
	`, metric, whereClause, metric, whereClause, len(args)+1)

	args = append(args, topN)

	rows, err := a.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query top talkers: %w", err)
	}
	defer rows.Close()

	talkers := []*TopTalker{}
	for rows.Next() {
		var talker TopTalker
		var labelsJSON []byte
		var totalMetric int64

		err := rows.Scan(
			&talker.IPAddress,
			&talker.Direction,
			&totalMetric,
			&talker.TotalBytes,
			&talker.TotalPackets,
			&talker.FlowCount,
			&labelsJSON,
		)
		if err != nil {
			logrus.Warnf("Failed to scan top talker: %v", err)
			continue
		}

		// Parse labels
		if len(labelsJSON) > 0 {
			talker.Labels = make(map[string]string)
			json.Unmarshal(labelsJSON, &talker.Labels)
		}

		talkers = append(talkers, &talker)
	}

	return talkers, nil
}

// getTopTalkersByFlowCount gets top talkers sorted by flow count
func (a *FlowAggregator) getTopTalkersByFlowCount(ctx context.Context, whereClause string, args []interface{}, topN int) ([]*TopTalker, error) {
	sqlQuery := fmt.Sprintf(`
		WITH source_stats AS (
			SELECT
				src_ip as ip_address,
				'source' as direction,
				COUNT(*) as flow_count,
				SUM(byte_count) as total_bytes,
				SUM(packet_count) as total_packets,
				source_labels as labels
			FROM flows
			%s
			GROUP BY src_ip, source_labels
		),
		dest_stats AS (
			SELECT
				dst_ip as ip_address,
				'destination' as direction,
				COUNT(*) as flow_count,
				SUM(byte_count) as total_bytes,
				SUM(packet_count) as total_packets,
				dest_labels as labels
			FROM flows
			%s
			GROUP BY dst_ip, dest_labels
		),
		combined AS (
			SELECT * FROM source_stats
			UNION ALL
			SELECT * FROM dest_stats
		)
		SELECT
			ip_address,
			direction,
			SUM(flow_count) as flow_count,
			SUM(total_bytes) as total_bytes,
			SUM(total_packets) as total_packets,
			(array_agg(labels ORDER BY flow_count DESC))[1] as labels
		FROM combined
		GROUP BY ip_address, direction
		ORDER BY flow_count DESC
		LIMIT $%d
	`, whereClause, whereClause, len(args)+1)

	args = append(args, topN)

	rows, err := a.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query top talkers by flow count: %w", err)
	}
	defer rows.Close()

	talkers := []*TopTalker{}
	for rows.Next() {
		var talker TopTalker
		var labelsJSON []byte

		err := rows.Scan(
			&talker.IPAddress,
			&talker.Direction,
			&talker.FlowCount,
			&talker.TotalBytes,
			&talker.TotalPackets,
			&labelsJSON,
		)
		if err != nil {
			logrus.Warnf("Failed to scan top talker: %v", err)
			continue
		}

		// Parse labels
		if len(labelsJSON) > 0 {
			talker.Labels = make(map[string]string)
			json.Unmarshal(labelsJSON, &talker.Labels)
		}

		talkers = append(talkers, &talker)
	}

	return talkers, nil
}

// GetAggregatedStats returns complete aggregation result
func (a *FlowAggregator) GetAggregatedStats(ctx context.Context, query *AggregationQuery) (*AggregationResult, error) {
	result := &AggregationResult{
		GroupBy:   query.GroupBy,
		TimeRange: query.TimeRange,
	}

	// Get dependencies
	deps, err := a.GetDependencies(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}
	result.Dependencies = deps

	// Get top talkers if requested
	if query.IncludeTopTalkers {
		topTalkers, err := a.GetTopTalkers(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed to get top talkers: %w", err)
		}
		result.TopTalkers = topTalkers
	}

	// Get summary statistics
	summary, err := a.getSummaryStats(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}
	result.Summary = summary

	return result, nil
}

// getSummaryStats returns summary statistics for the time range
func (a *FlowAggregator) getSummaryStats(ctx context.Context, query *AggregationQuery) (*AggregationSummary, error) {
	whereClause, args := a.buildWhereClause(query)

	sqlQuery := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_flows,
			SUM(byte_count) as total_bytes,
			SUM(packet_count) as total_packets,
			COUNT(DISTINCT src_ip) + COUNT(DISTINCT dst_ip) as unique_endpoints,
			AVG(EXTRACT(EPOCH FROM (COALESCE(created_at, NOW()) - created_at)) * 1000) as avg_duration_ms
		FROM flows
		%s
	`, whereClause)

	summary := &AggregationSummary{}
	var avgDuration sql.NullFloat64

	err := a.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&summary.TotalFlows,
		&summary.TotalBytes,
		&summary.TotalPackets,
		&summary.UniqueEndpoints,
		&avgDuration,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary stats: %w", err)
	}

	if avgDuration.Valid {
		summary.AvgFlowDuration = avgDuration.Float64
	}

	return summary, nil
}

// buildWhereClause builds SQL WHERE clause from query parameters
func (a *FlowAggregator) buildWhereClause(query *AggregationQuery) (string, []interface{}) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	// Time range filter
	if !query.TimeRange.StartTime.IsZero() && !query.TimeRange.EndTime.IsZero() {
		where += fmt.Sprintf(" AND timestamp_ns >= $%d AND timestamp_ns <= $%d", argIdx, argIdx+1)
		args = append(args, query.TimeRange.StartTime.UnixNano(), query.TimeRange.EndTime.UnixNano())
		argIdx += 2
	}

	// Protocol filter
	if len(query.Protocols) > 0 {
		where += fmt.Sprintf(" AND protocol = ANY($%d)", argIdx)
		args = append(args, query.Protocols)
		argIdx++
	}

	// Agent ID filter
	if len(query.AgentIDs) > 0 {
		where += fmt.Sprintf(" AND agent_id = ANY($%d)", argIdx)
		args = append(args, query.AgentIDs)
		argIdx++
	}

	return where, args
}
