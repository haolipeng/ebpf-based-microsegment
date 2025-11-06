package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/ebpf-microsegment/src/server/pkg/aggregator"
)

// AggregatorHandler handles aggregation and analysis requests
type AggregatorHandler struct {
	aggregator *aggregator.FlowAggregator
}

// NewAggregatorHandler creates a new AggregatorHandler
func NewAggregatorHandler(agg *aggregator.FlowAggregator) *AggregatorHandler {
	return &AggregatorHandler{
		aggregator: agg,
	}
}

// RegisterRoutes registers aggregator API routes to the router
func (h *AggregatorHandler) RegisterRoutes(rg *gin.RouterGroup) {
	agg := rg.Group("/aggregator")
	{
		agg.GET("/dependencies", h.GetDependencies)
		agg.GET("/top-talkers", h.GetTopTalkers)
		agg.GET("/stats", h.GetAggregatedStats)
	}
}

// GetDependencies handles GET /api/v1/aggregator/dependencies
func (h *AggregatorHandler) GetDependencies(c *gin.Context) {
	query := h.parseAggregationQuery(c)

	dependencies, err := h.aggregator.GetDependencies(c.Request.Context(), query)
	if err != nil {
		logrus.Errorf("Failed to get dependencies: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get dependencies",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"dependencies": dependencies,
		"group_by":     query.GroupBy,
		"time_range": gin.H{
			"start_time": query.TimeRange.StartTime.Format(time.RFC3339),
			"end_time":   query.TimeRange.EndTime.Format(time.RFC3339),
		},
	})
}

// GetTopTalkers handles GET /api/v1/aggregator/top-talkers
func (h *AggregatorHandler) GetTopTalkers(c *gin.Context) {
	query := h.parseAggregationQuery(c)
	query.IncludeTopTalkers = true

	topTalkers, err := h.aggregator.GetTopTalkers(c.Request.Context(), query)
	if err != nil {
		logrus.Errorf("Failed to get top talkers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get top talkers",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"top_talkers": topTalkers,
		"time_range": gin.H{
			"start_time": query.TimeRange.StartTime.Format(time.RFC3339),
			"end_time":   query.TimeRange.EndTime.Format(time.RFC3339),
		},
	})
}

// GetAggregatedStats handles GET /api/v1/aggregator/stats
func (h *AggregatorHandler) GetAggregatedStats(c *gin.Context) {
	query := h.parseAggregationQuery(c)

	// Check if top talkers should be included
	if includeTopTalkers := c.Query("include_top_talkers"); includeTopTalkers == "true" {
		query.IncludeTopTalkers = true
	}

	stats, err := h.aggregator.GetAggregatedStats(c.Request.Context(), query)
	if err != nil {
		logrus.Errorf("Failed to get aggregated stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get aggregated stats",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// parseAggregationQuery parses query parameters into AggregationQuery
func (h *AggregatorHandler) parseAggregationQuery(c *gin.Context) *aggregator.AggregationQuery {
	// Default time range: last 1 hour
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

	// Parse time range
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = t
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = t
		}
	}

	// Parse group_by (default: "app")
	groupBy := c.DefaultQuery("group_by", "app")

	// Parse top N (default: 10)
	topN := 10
	if topNStr := c.Query("top_n"); topNStr != "" {
		if n, err := strconv.Atoi(topNStr); err == nil && n > 0 {
			topN = n
		}
	}

	// Parse protocol filter
	var protocols []string
	if protocol := c.Query("protocol"); protocol != "" {
		protocols = append(protocols, protocol)
	}

	// Parse agent ID filter
	var agentIDs []string
	if agentID := c.Query("agent_id"); agentID != "" {
		agentIDs = append(agentIDs, agentID)
	}

	return &aggregator.AggregationQuery{
		TimeRange: aggregator.TimeRange{
			StartTime: startTime,
			EndTime:   endTime,
		},
		GroupBy:             groupBy,
		IncludeTopTalkers:   false, // Set by caller
		TopN:                topN,
		Protocols:           protocols,
		AgentIDs:            agentIDs,
	}
}
