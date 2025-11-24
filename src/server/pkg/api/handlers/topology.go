package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/topology"
	"github.com/sirupsen/logrus"
)

// TopologyHandler handles topology-related HTTP requests.
type TopologyHandler struct {
	manager *topology.Manager
}

// NewTopologyHandler creates a new TopologyHandler.
func NewTopologyHandler(manager *topology.Manager) *TopologyHandler {
	return &TopologyHandler{
		manager: manager,
	}
}

// RegisterRoutes registers topology API routes.
func (h *TopologyHandler) RegisterRoutes(rg *gin.RouterGroup) {
	topo := rg.Group("/topology")
	{
		topo.GET("", h.GetTopology)
		topo.GET("/nodes", h.GetNodes)
		topo.GET("/nodes/:id", h.GetNodeDetail)
		topo.GET("/edges", h.GetEdges)
		topo.GET("/edges/:src/:dst", h.GetEdgeDetail)
		topo.GET("/stats", h.GetStats)
	}
}

// GetTopology handles GET /api/v1/topology - Get complete topology graph.
func (h *TopologyHandler) GetTopology(c *gin.Context) {
	filter := h.parseFilter(c)

	topo := h.manager.GetTopology(filter)
	if topo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get topology",
		})
		return
	}

	c.JSON(http.StatusOK, topo)
}

// GetNodes handles GET /api/v1/topology/nodes - Get all nodes.
func (h *TopologyHandler) GetNodes(c *gin.Context) {
	filter := h.parseFilter(c)

	topo := h.manager.GetTopology(filter)
	if topo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get nodes",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes":       topo.Nodes,
		"total_count": len(topo.Nodes),
	})
}

// GetNodeDetail handles GET /api/v1/topology/nodes/:id - Get node details.
func (h *TopologyHandler) GetNodeDetail(c *gin.Context) {
	nodeID := c.Param("id")
	if nodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Node ID is required",
		})
		return
	}

	detail := h.manager.GetNodeDetail(nodeID)
	if detail == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Node not found",
		})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// GetEdges handles GET /api/v1/topology/edges - Get all edges.
func (h *TopologyHandler) GetEdges(c *gin.Context) {
	filter := h.parseFilter(c)

	topo := h.manager.GetTopology(filter)
	if topo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get edges",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"edges":       topo.Edges,
		"total_count": len(topo.Edges),
	})
}

// GetEdgeDetail handles GET /api/v1/topology/edges/:src/:dst - Get edge details.
func (h *TopologyHandler) GetEdgeDetail(c *gin.Context) {
	srcID := c.Param("src")
	dstID := c.Param("dst")

	if srcID == "" || dstID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Source and destination IDs are required",
		})
		return
	}

	detail := h.manager.GetEdgeDetail(srcID, dstID)
	if detail == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Edge not found",
		})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// GetStats handles GET /api/v1/topology/stats - Get topology statistics.
func (h *TopologyHandler) GetStats(c *gin.Context) {
	stats := h.manager.GetStats()
	if stats == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get stats",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// parseFilter parses query parameters into a TopologyFilter.
func (h *TopologyHandler) parseFilter(c *gin.Context) *topology.TopologyFilter {
	filter := &topology.TopologyFilter{
		IncludeExternal: true, // Default to include external
	}

	// Parse namespace
	if ns := c.Query("namespace"); ns != "" {
		filter.Namespace = ns
	}

	// Parse include_external
	if extStr := c.Query("include_external"); extStr != "" {
		if ext, err := strconv.ParseBool(extStr); err == nil {
			filter.IncludeExternal = ext
		}
	}

	// Parse min_flow_count
	if minStr := c.Query("min_flow_count"); minStr != "" {
		if min, err := strconv.Atoi(minStr); err == nil {
			filter.MinFlowCount = min
		}
	}

	// Parse policy_action
	if action := c.Query("policy_action"); action != "" {
		filter.PolicyAction = action
	}

	// Parse group_by
	if groupBy := c.Query("group_by"); groupBy != "" {
		filter.GroupBy = groupBy
	}

	// Parse time range
	filter.TimeRange = h.parseTimeRange(c)

	// Parse node_types
	if nodeTypes := c.QueryArray("node_type"); len(nodeTypes) > 0 {
		filter.NodeTypes = make([]topology.NodeType, 0, len(nodeTypes))
		for _, t := range nodeTypes {
			filter.NodeTypes = append(filter.NodeTypes, topology.NodeType(t))
		}
	}

	// Parse labels (format: label=key:value)
	if labels := c.QueryArray("label"); len(labels) > 0 {
		filter.Labels = make(map[string]string)
		for _, label := range labels {
			if k, v, ok := parseLabel(label); ok {
				filter.Labels[k] = v
			}
		}
	}

	return filter
}

// parseTimeRange parses time range from query parameters.
func (h *TopologyHandler) parseTimeRange(c *gin.Context) *topology.TimeRange {
	// Check for duration-based range (e.g., time_range=1h)
	if rangeStr := c.Query("time_range"); rangeStr != "" {
		duration, err := time.ParseDuration(rangeStr)
		if err == nil {
			now := time.Now()
			return &topology.TimeRange{
				Start: now.Add(-duration),
				End:   now,
			}
		}
		logrus.Warnf("Invalid time_range duration: %s", rangeStr)
	}

	// Check for explicit start/end times
	var startTime, endTime time.Time

	if startStr := c.Query("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = t
		}
	}

	if endStr := c.Query("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = t
		}
	}

	if !startTime.IsZero() || !endTime.IsZero() {
		if startTime.IsZero() {
			startTime = time.Now().Add(-24 * time.Hour)
		}
		if endTime.IsZero() {
			endTime = time.Now()
		}
		return &topology.TimeRange{
			Start: startTime,
			End:   endTime,
		}
	}

	return nil
}

// parseLabel parses a label string in the format "key:value".
func parseLabel(label string) (key, value string, ok bool) {
	for i := 0; i < len(label); i++ {
		if label[i] == ':' {
			return label[:i], label[i+1:], true
		}
	}
	return "", "", false
}
