package handlers

import (
	"net/http"
	"time"

	"github.com/ebpf-microsegment/src/agent/pkg/api/models"
	"github.com/ebpf-microsegment/src/agent/pkg/dataplane"
	"github.com/ebpf-microsegment/src/agent/pkg/policy"
	"github.com/gin-gonic/gin"
)

var startTime = time.Now()

// HealthHandler handles health check requests
type HealthHandler struct {
	dataPlane     dataplane.DataPlaneInterface
	policyManager policy.Manager
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(dp dataplane.DataPlaneInterface, pm policy.Manager) *HealthHandler {
	return &HealthHandler{
		dataPlane:     dp,
		policyManager: pm,
	}
}

// GetHealth handles GET /api/v1/health
// Simple health check endpoint
func (h *HealthHandler) GetHealth(c *gin.Context) {
	response := models.HealthResponse{
		Status:  "ok",
		Message: "API server is healthy",
	}

	c.JSON(http.StatusOK, response)
}

// GetStatus handles GET /api/v1/status
// Detailed status endpoint with data plane information
func (h *HealthHandler) GetStatus(c *gin.Context) {
	// Get data plane statistics
	stats := h.dataPlane.GetStatistics()

	// Get policy count
	policies, err := h.policyManager.ListPolicies()
	policyCount := len(policies)

	// Determine data plane status
	dataPlaneStatus := models.DataPlaneStatus{
		Status:  "running",
		Message: "Data plane is operational",
	}

	// Check if we're receiving packets (basic health check)
	if stats.TotalPackets == 0 {
		dataPlaneStatus.Status = "idle"
		dataPlaneStatus.Message = "Data plane is idle (no packets processed)"
	}

	// Determine overall status
	overallStatus := "ok"
	if err != nil {
		overallStatus = "degraded"
	}

	// Build response
	response := models.StatusResponse{
		Status:    overallStatus,
		Version:   "0.1.0", // TODO: Get from build info
		Interface: "lo",    // TODO: Get from config
		DataPlane: dataPlaneStatus,
		API: models.APIStatus{
			Status:  "running",
			Message: "API server is operational",
		},
		Statistics: &models.StatisticsResponse{
			TotalPackets:   stats.TotalPackets,
			AllowedPackets: stats.AllowedPackets,
			DeniedPackets:  stats.DeniedPackets,
			NewSessions:    stats.NewSessions,
			ClosedSessions: stats.ClosedSessions,
			ActiveSessions: stats.ActiveSessions,
			PolicyHits:     stats.PolicyHits,
			PolicyMisses:   stats.PolicyMisses,
		},
		PolicyCount: policyCount,
		Uptime:      int64(time.Since(startTime).Seconds()),
	}

	c.JSON(http.StatusOK, response)
}

