package handlers

import (
	"net/http"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/flow"
	"github.com/gin-gonic/gin"
)

// LifecycleHandler handles lifecycle management API requests
type LifecycleHandler struct {
	manager *flow.LifecycleManager
}

// NewLifecycleHandler creates a new lifecycle handler
func NewLifecycleHandler(manager *flow.LifecycleManager) *LifecycleHandler {
	return &LifecycleHandler{
		manager: manager,
	}
}

// GetLifecycleStats handles GET /api/v1/flows/lifecycle/stats
// Returns lifecycle management statistics including:
// - Total cleanup runs and flows deleted
// - Last cleanup time and result
// - Disk space usage and warnings
func (h *LifecycleHandler) GetLifecycleStats(c *gin.Context) {
	stats := h.manager.GetStats()
	c.JSON(http.StatusOK, stats)
}
