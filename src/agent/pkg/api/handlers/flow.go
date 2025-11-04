package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ebpf-microsegment/src/agent/pkg/api/models"
	"github.com/ebpf-microsegment/src/agent/pkg/flow"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// FlowHandler handles flow-related API requests
type FlowHandler struct {
	collector  *flow.Collector
	storage    flow.Storage
	aggregator *flow.Aggregator
}

// NewFlowHandler creates a new flow handler
func NewFlowHandler(collector *flow.Collector, storage flow.Storage) *FlowHandler {
	return &FlowHandler{
		collector:  collector,
		storage:    storage,
		aggregator: flow.NewAggregator(storage),
	}
}

// ListFlows handles GET /api/v1/flows
// Query parameters:
//   - start_time: Start of time range (RFC3339 format)
//   - end_time: End of time range (RFC3339 format)
//   - source_ip: Filter by source IP
//   - dest_ip: Filter by destination IP
//   - protocol: Filter by protocol (TCP/UDP/ICMP)
//   - state: Filter by state (ACTIVE/CLOSED/TIMEOUT)
//   - direction: Filter by direction (INGRESS/EGRESS)
//   - policy_action: Filter by policy action (ALLOW/DENY/LOG)
//   - limit: Max results (default: 100)
//   - offset: Pagination offset (default: 0)
//   - sort_by: Field to sort by (default: start_time)
//   - sort_order: Sort order (asc/desc, default: desc)
func (h *FlowHandler) ListFlows(c *gin.Context) {
	query := &flow.FlowQuery{
		Limit:  100,
		Offset: 0,
		SortBy: "start_time",
		SortOrder: "desc",
	}

	// Parse time range
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "Invalid start_time format",
				Message: "start_time must be in RFC3339 format",
			})
			return
		}
		query.StartTime = &startTime
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "Invalid end_time format",
				Message: "end_time must be in RFC3339 format",
			})
			return
		}
		query.EndTime = &endTime
	}

	// Parse filters
	if sourceIP := c.Query("source_ip"); sourceIP != "" {
		query.SourceIP = &sourceIP
	}
	if destIP := c.Query("dest_ip"); destIP != "" {
		query.DestIP = &destIP
	}
	if protocol := c.Query("protocol"); protocol != "" {
		query.Protocol = &protocol
	}
	if state := c.Query("state"); state != "" {
		query.State = &state
	}
	if direction := c.Query("direction"); direction != "" {
		query.Direction = &direction
	}
	if policyAction := c.Query("policy_action"); policyAction != "" {
		query.PolicyAction = &policyAction
	}

	// Parse pagination
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 1000 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "Invalid limit",
				Message: "limit must be between 1 and 1000",
			})
			return
		}
		query.Limit = limit
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "Invalid offset",
				Message: "offset must be >= 0",
			})
			return
		}
		query.Offset = offset
	}

	// Parse sorting
	if sortBy := c.Query("sort_by"); sortBy != "" {
		query.SortBy = sortBy
	}
	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		if sortOrder != "asc" && sortOrder != "desc" {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "Invalid sort_order",
				Message: "sort_order must be 'asc' or 'desc'",
			})
			return
		}
		query.SortOrder = sortOrder
	}

	// Query flows from storage
	flows, err := h.storage.QueryFlows(query)
	if err != nil {
		log.Errorf("Failed to query flows: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to query flows",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.FlowListResponse{
		Flows: flows,
		Count: len(flows),
		Query: models.FlowQueryInfo{
			Limit:      query.Limit,
			Offset:     query.Offset,
			SortBy:     query.SortBy,
			SortOrder:  query.SortOrder,
		},
	})
}

// GetFlow handles GET /api/v1/flows/:id
func (h *FlowHandler) GetFlow(c *gin.Context) {
	flowID := c.Param("id")
	if flowID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Missing flow ID",
			Message: "Flow ID is required",
		})
		return
	}

	flow, err := h.storage.GetFlow(flowID)
	if err != nil {
		log.Errorf("Failed to get flow %s: %v", flowID, err)
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "Flow not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, flow)
}

// GetFlowSummary handles GET /api/v1/flows/summary
// Query parameters:
//   - start_time: Start of time range (RFC3339 format, default: 1 hour ago)
//   - end_time: End of time range (RFC3339 format, default: now)
func (h *FlowHandler) GetFlowSummary(c *gin.Context) {
	// Default to last 1 hour
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

	// Parse custom time range if provided
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		t, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "Invalid start_time format",
				Message: "start_time must be in RFC3339 format",
			})
			return
		}
		startTime = t
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		t, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "Invalid end_time format",
				Message: "end_time must be in RFC3339 format",
			})
			return
		}
		endTime = t
	}

	// Validate time range
	if startTime.After(endTime) {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid time range",
			Message: "start_time must be before end_time",
		})
		return
	}

	// Get summary from storage
	summary, err := h.storage.GetFlowSummary(startTime, endTime)
	if err != nil {
		log.Errorf("Failed to get flow summary: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get flow summary",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetActiveFlows handles GET /api/v1/flows/active
// Returns currently active flows from the collector's in-memory cache
func (h *FlowHandler) GetActiveFlows(c *gin.Context) {
	flows := h.collector.GetActiveFlows()

	c.JSON(http.StatusOK, models.FlowListResponse{
		Flows: flows,
		Count: len(flows),
		Query: models.FlowQueryInfo{
			Limit:  len(flows),
			Offset: 0,
		},
	})
}

// GetCollectorMetrics handles GET /api/v1/flows/metrics
// Returns collector performance metrics
func (h *FlowHandler) GetCollectorMetrics(c *gin.Context) {
	processed, dropped, activeFlows := h.collector.GetMetrics()

	// Calculate drop rate
	var dropRate float64
	if processed > 0 {
		dropRate = float64(dropped) / float64(processed+dropped) * 100
	}

	c.JSON(http.StatusOK, models.FlowMetricsResponse{
		EventsProcessed: processed,
		EventsDropped:   dropped,
		ActiveFlows:     activeFlows,
		DropRate:        dropRate,
	})
}

// GetDependencies handles GET /api/v1/flows/dependencies
// Query parameters:
//   - start_time: Start of time range (RFC3339 format, default: 1 hour ago)
//   - end_time: End of time range (RFC3339 format, default: now)
//   - min_flows: Minimum flow count to include (default: 1)
func (h *FlowHandler) GetDependencies(c *gin.Context) {
	// Default to last 1 hour
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)
	minFlows := 1

	// Parse custom time range if provided
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		t, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "Invalid start_time format",
				Message: "start_time must be in RFC3339 format",
			})
			return
		}
		startTime = t
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		t, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "Invalid end_time format",
				Message: "end_time must be in RFC3339 format",
			})
			return
		}
		endTime = t
	}

	// Parse min_flows
	if minFlowsStr := c.Query("min_flows"); minFlowsStr != "" {
		mf, err := strconv.Atoi(minFlowsStr)
		if err != nil || mf < 1 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "Invalid min_flows",
				Message: "min_flows must be >= 1",
			})
			return
		}
		minFlows = mf
	}

	// Get dependencies
	dependencies, err := h.aggregator.GetDependencies(startTime, endTime, minFlows)
	if err != nil {
		log.Errorf("Failed to get dependencies: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get dependencies",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.DependencyListResponse{
		Dependencies: dependencies,
		Count:        len(dependencies),
		TimeRange: models.TimeRangeInfo{
			StartTime: startTime,
			EndTime:   endTime,
		},
	})
}

// GetTopTalkers handles GET /api/v1/flows/top-talkers
// Query parameters:
//   - start_time: Start of time range (RFC3339 format, default: 1 hour ago)
//   - end_time: End of time range (RFC3339 format, default: now)
//   - limit: Number of top talkers to return (default: 10)
func (h *FlowHandler) GetTopTalkers(c *gin.Context) {
	// Default to last 1 hour
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)
	limit := 10

	// Parse custom time range if provided
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		t, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "Invalid start_time format",
				Message: "start_time must be in RFC3339 format",
			})
			return
		}
		startTime = t
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		t, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "Invalid end_time format",
				Message: "end_time must be in RFC3339 format",
			})
			return
		}
		endTime = t
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil || l < 1 || l > 100 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "Invalid limit",
				Message: "limit must be between 1 and 100",
			})
			return
		}
		limit = l
	}

	// Get top talkers
	topTalkers, err := h.aggregator.GetTopTalkers(startTime, endTime, limit, "flow_count")
	if err != nil {
		log.Errorf("Failed to get top talkers: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get top talkers",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.TopTalkersResponse{
		TopTalkers: topTalkers,
		Count:      len(topTalkers),
		TimeRange: models.TimeRangeInfo{
			StartTime: startTime,
			EndTime:   endTime,
		},
	})
}
