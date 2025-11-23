package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	commonpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/common"
	flowpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/flow"
	"github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/storage"
)

// FlowHandler handles Flow-related HTTP requests
type FlowHandler struct {
	flowStorage *storage.FlowStorage
}

// NewFlowHandler creates a new FlowHandler
func NewFlowHandler(flowStorage *storage.FlowStorage) *FlowHandler {
	return &FlowHandler{
		flowStorage: flowStorage,
	}
}

// RegisterRoutes registers Flow API routes to the router
func (h *FlowHandler) RegisterRoutes(rg *gin.RouterGroup) {
	flows := rg.Group("/flows")
	{
		flows.GET("", h.ListFlows)
		flows.GET("/:id", h.GetFlow)
		flows.GET("/summary", h.GetFlowSummary)
		flows.GET("/dependencies", h.GetFlowDependencies)
	}
}

// ListFlows handles GET /api/v1/flows - Query flow list with pagination and filtering
func (h *FlowHandler) ListFlows(c *gin.Context) {
	// Parse query parameters
	query := &flowpb.FlowQuery{
		Limit:  100, // Default limit
		Offset: 0,
	}

	// Parse limit and offset
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.ParseUint(limitStr, 10, 32); err == nil {
			query.Limit = uint32(limit)
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.ParseUint(offsetStr, 10, 32); err == nil {
			query.Offset = uint32(offset)
		}
	}

	// Parse time range (default: last 1 hour)
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

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

	query.TimeRange = &commonpb.TimeRange{
		StartTime: startTime.UnixNano(),
		EndTime:   endTime.UnixNano(),
	}

	// Parse filters
	if agentID := c.Query("agent_id"); agentID != "" {
		query.AgentId = agentID
	}
	if sourceIP := c.Query("source_ip"); sourceIP != "" {
		query.SrcIp = sourceIP
	}
	if destIP := c.Query("dest_ip"); destIP != "" {
		query.DstIp = destIP
	}
	if protocol := c.Query("protocol"); protocol != "" {
		// Parse protocol string to enum (TCP=6, UDP=17, ICMP=1, etc.)
		switch protocol {
		case "tcp", "TCP":
			query.Protocol = commonpb.Protocol_PROTOCOL_TCP
		case "udp", "UDP":
			query.Protocol = commonpb.Protocol_PROTOCOL_UDP
		case "icmp", "ICMP":
			query.Protocol = commonpb.Protocol_PROTOCOL_ICMP
		}
	}

	// Parse label filters (JSON query)
	// source_labels={"app":"nginx"}
	// dest_labels={"env":"prod"}
	if sourceLabels := c.Query("source_labels"); sourceLabels != "" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(sourceLabels), &labels); err == nil {
			query.SourceLabels = labels
		} else {
			logrus.Warnf("Invalid source_labels JSON: %v", err)
		}
	}
	if destLabels := c.Query("dest_labels"); destLabels != "" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(destLabels), &labels); err == nil {
			query.DestLabels = labels
		} else {
			logrus.Warnf("Invalid dest_labels JSON: %v", err)
		}
	}

	// Query flows from storage
	flows, total, err := h.flowStorage.QueryFlows(c.Request.Context(), query)
	if err != nil {
		logrus.Errorf("Failed to query flows: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to query flows",
		})
		return
	}

	// Calculate pagination info
	hasMore := (query.Offset + query.Limit) < uint32(total)

	c.JSON(http.StatusOK, gin.H{
		"flows":       flows,
		"total_count": total,
		"limit":       query.Limit,
		"offset":      query.Offset,
		"has_more":    hasMore,
	})
}

// GetFlow handles GET /api/v1/flows/:id - Get single flow record
func (h *FlowHandler) GetFlow(c *gin.Context) {
	flowID := c.Param("id")

	if flowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "flow_id is required",
		})
		return
	}

	// Query single flow by ID
	flows, _, err := h.flowStorage.QueryFlows(c.Request.Context(), &flowpb.FlowQuery{
		Limit:  1,
		Offset: 0,
		// Note: FlowQuery doesn't have ID filter yet, this is a placeholder
		// In production, add ID field to FlowQuery or create GetFlowByID method
	})

	if err != nil {
		logrus.Errorf("Failed to get flow: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get flow",
		})
		return
	}

	if len(flows) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Flow not found",
		})
		return
	}

	c.JSON(http.StatusOK, flows[0])
}

// GetFlowSummary handles GET /api/v1/flows/summary - Get flow statistics summary
func (h *FlowHandler) GetFlowSummary(c *gin.Context) {
	// Parse time range (default: last 7 days)
	endTime := time.Now()
	startTime := endTime.Add(-7 * 24 * time.Hour)

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

	// Query flow summary (aggregated statistics)
	summary, err := h.flowStorage.GetFlowSummary(c.Request.Context(), startTime, endTime)
	if err != nil {
		logrus.Errorf("Failed to get flow summary: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get flow summary",
		})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetFlowDependencies handles GET /api/v1/flows/dependencies - Get application dependencies
func (h *FlowHandler) GetFlowDependencies(c *gin.Context) {
	// Parse time range (default: last 1 hour)
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

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

	// Parse group_by parameter (default: app label)
	groupBy := c.DefaultQuery("group_by", "app")

	// Query flow dependencies (label-based aggregation)
	dependencies, err := h.flowStorage.GetFlowDependencies(c.Request.Context(), startTime, endTime, groupBy)
	if err != nil {
		logrus.Errorf("Failed to get flow dependencies: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get flow dependencies",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"dependencies": dependencies,
		"group_by":     groupBy,
		"start_time":   startTime.Format(time.RFC3339),
		"end_time":     endTime.Format(time.RFC3339),
	})
}
