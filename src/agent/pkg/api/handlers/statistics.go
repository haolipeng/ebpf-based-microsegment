package handlers

import (
	"net/http"

	"github.com/ebpf-microsegment/src/agent/pkg/api/models"
	"github.com/ebpf-microsegment/src/agent/pkg/dataplane"
	"github.com/gin-gonic/gin"
)

// StatisticsHandler handles statistics requests
type StatisticsHandler struct {
	dataPlane dataplane.DataPlaneInterface
}

// NewStatisticsHandler creates a new statistics handler
func NewStatisticsHandler(dp dataplane.DataPlaneInterface) *StatisticsHandler {
	return &StatisticsHandler{
		dataPlane: dp,
	}
}

// GetAllStats handles GET /api/v1/stats
func (h *StatisticsHandler) GetAllStats(c *gin.Context) {
	stats := h.dataPlane.GetStatistics()

	response := models.StatisticsResponse{
		TotalPackets:   stats.TotalPackets,
		AllowedPackets: stats.AllowedPackets,
		DeniedPackets:  stats.DeniedPackets,
		NewSessions:    stats.NewSessions,
		ClosedSessions: stats.ClosedSessions,
		ActiveSessions: stats.ActiveSessions,
		PolicyHits:     stats.PolicyHits,
		PolicyMisses:   stats.PolicyMisses,
	}

	c.JSON(http.StatusOK, response)
}

// GetPacketStats handles GET /api/v1/stats/packets
func (h *StatisticsHandler) GetPacketStats(c *gin.Context) {
	stats := h.dataPlane.GetStatistics()

	// Calculate rates
	var allowRate, denyRate float64
	if stats.TotalPackets > 0 {
		allowRate = float64(stats.AllowedPackets) / float64(stats.TotalPackets) * 100
		denyRate = float64(stats.DeniedPackets) / float64(stats.TotalPackets) * 100
	}

	response := models.PacketStatsResponse{
		TotalPackets:   stats.TotalPackets,
		AllowedPackets: stats.AllowedPackets,
		DeniedPackets:  stats.DeniedPackets,
		AllowRate:      allowRate,
		DenyRate:       denyRate,
	}

	c.JSON(http.StatusOK, response)
}

// GetSessionStats handles GET /api/v1/stats/sessions
func (h *StatisticsHandler) GetSessionStats(c *gin.Context) {
	stats := h.dataPlane.GetStatistics()

	response := models.SessionStatsResponse{
		NewSessions:    stats.NewSessions,
		ClosedSessions: stats.ClosedSessions,
		ActiveSessions: stats.ActiveSessions,
	}

	c.JSON(http.StatusOK, response)
}

// GetPolicyStats handles GET /api/v1/stats/policies
func (h *StatisticsHandler) GetPolicyStats(c *gin.Context) {
	stats := h.dataPlane.GetStatistics()

	// Calculate hit rate
	var hitRate float64
	totalLookups := stats.PolicyHits + stats.PolicyMisses
	if totalLookups > 0 {
		hitRate = float64(stats.PolicyHits) / float64(totalLookups) * 100
	}

	response := models.PolicyStatsResponse{
		PolicyHits:   stats.PolicyHits,
		PolicyMisses: stats.PolicyMisses,
		HitRate:      hitRate,
	}

	c.JSON(http.StatusOK, response)
}

