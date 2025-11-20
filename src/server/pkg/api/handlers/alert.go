package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/storage"
)

// AlertHandler handles Security Alert-related HTTP requests
type AlertHandler struct {
	alertStorage *storage.AlertStorage
}

// NewAlertHandler creates a new AlertHandler
func NewAlertHandler(alertStorage *storage.AlertStorage) *AlertHandler {
	return &AlertHandler{
		alertStorage: alertStorage,
	}
}

// RegisterRoutes registers Alert API routes to the router
func (h *AlertHandler) RegisterRoutes(rg *gin.RouterGroup) {
	alerts := rg.Group("/alerts")
	{
		alerts.GET("", h.ListAlerts)
		alerts.GET("/:id", h.GetAlert)
		alerts.GET("/stats", h.GetAlertStats)
		// Acknowledge endpoint can be added later
		// alerts.PUT("/:id/acknowledge", h.AcknowledgeAlert)
	}
}

// ListAlerts handles GET /api/v1/alerts - Query alert list with pagination and filtering
func (h *AlertHandler) ListAlerts(c *gin.Context) {
	// Parse query parameters
	opts := &storage.AlertQueryOptions{
		Page:     1,
		PageSize: 50,
	}

	// Parse page and page_size
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			opts.Page = page
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 && pageSize <= 100 {
			opts.PageSize = pageSize
		}
	}

	// Parse level filter
	if levelStr := c.Query("level"); levelStr != "" {
		// Convert level string to int (info=0, warning=1, critical=2)
		var level int32
		switch levelStr {
		case "info", "INFO", "ALERT_INFO":
			level = 0
		case "warning", "WARNING", "ALERT_WARNING":
			level = 1
		case "critical", "CRITICAL", "ALERT_CRITICAL":
			level = 2
		default:
			if l, err := strconv.ParseInt(levelStr, 10, 32); err == nil {
				level = int32(l)
			}
		}
		opts.Level = &level
	}

	// Parse type filter
	if typeStr := c.Query("type"); typeStr != "" {
		if t, err := strconv.ParseInt(typeStr, 10, 32); err == nil {
			alertType := int32(t)
			opts.Type = &alertType
		}
	}

	// Parse process_path filter
	if processPath := c.Query("process_path"); processPath != "" {
		opts.ProcessPath = &processPath
	}

	// Parse time range (default: last 24 hours)
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

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

	startTimeNs := startTime.UnixNano()
	endTimeNs := endTime.UnixNano()
	opts.StartTime = &startTimeNs
	opts.EndTime = &endTimeNs

	// Query alerts from storage
	alerts, total, err := h.alertStorage.QueryAlerts(c.Request.Context(), opts)
	if err != nil {
		logrus.Errorf("Failed to query alerts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to query alerts",
		})
		return
	}

	// Build response
	c.JSON(http.StatusOK, gin.H{
		"alerts":      alerts,
		"total_count": total,
		"page":        opts.Page,
		"page_size":   opts.PageSize,
	})
}

// GetAlert handles GET /api/v1/alerts/:id - Get single alert record
func (h *AlertHandler) GetAlert(c *gin.Context) {
	alertID := c.Param("id")

	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "alert_id is required",
		})
		return
	}

	// Query single alert by ID
	alert, err := h.alertStorage.GetAlertByID(c.Request.Context(), alertID)
	if err != nil {
		if err.Error() == "alert not found: "+alertID {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Alert not found",
			})
			return
		}
		logrus.Errorf("Failed to get alert: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get alert",
		})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// GetAlertStats handles GET /api/v1/alerts/stats - Get alert statistics summary
func (h *AlertHandler) GetAlertStats(c *gin.Context) {
	// Parse time range parameter (default: 24h)
	timeWindow := c.DefaultQuery("time_window", "24h")

	var duration time.Duration
	switch timeWindow {
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	default:
		duration = 24 * time.Hour
	}

	endTime := time.Now()
	startTime := endTime.Add(-duration)

	// Override with explicit time range if provided
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

	startTimeNs := startTime.UnixNano()
	endTimeNs := endTime.UnixNano()

	// Query alert statistics
	stats, err := h.alertStorage.GetAlertStats(c.Request.Context(), startTimeNs, endTimeNs, timeWindow)
	if err != nil {
		logrus.Errorf("Failed to get alert stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get alert statistics",
		})
		return
	}

	// Build response
	c.JSON(http.StatusOK, gin.H{
		"by_level":      stats.ByLevel,
		"by_type":       stats.ByType,
		"top_processes": stats.TopProcesses,
		"timeline":      stats.Timeline,
		"time_window":   timeWindow,
		"start_time":    startTime.Format(time.RFC3339),
		"end_time":      endTime.Format(time.RFC3339),
	})
}
