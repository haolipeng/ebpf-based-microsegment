
// input: workload API requests (GET/POST/PUT/DELETE)
// output: workload JSON responses
// pos: workload API handlers - if file updated, must sync with this header comment and pkg/api/CLAUDE.md
package handlers

import (
	"fmt"
	"net"
	"net/http"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/api/models"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/workload"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// WorkloadHandler handles workload management requests
type WorkloadHandler struct {
	workloadManager *workload.Manager
}

// NewWorkloadHandler creates a new workload handler
func NewWorkloadHandler(wm *workload.Manager) *WorkloadHandler {
	return &WorkloadHandler{
		workloadManager: wm,
	}
}

// CreateWorkload handles POST /api/v1/workloads
func (h *WorkloadHandler) CreateWorkload(c *gin.Context) {
	var req models.WorkloadRequest

	// Bind and validate JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			http.StatusBadRequest,
			"validation_error",
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Parse IP addresses
	ips := make([]net.IP, 0, len(req.IPs))
	for _, ipStr := range req.IPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			c.JSON(http.StatusBadRequest, models.NewErrorResponse(
				http.StatusBadRequest,
				"validation_error",
				fmt.Sprintf("Invalid IP address: %s", ipStr),
				nil,
			))
			return
		}
		ips = append(ips, ip)
	}

	// Create workload
	// Use hostname as HostID if not provided
	hostID := req.HostID
	if hostID == "" {
		hostID = "default-host"
	}

	w := &workload.Workload{
		ID:     req.ID,
		Name:   req.Name,
		HostID: hostID,
		IPs:    ips,
		Labels: req.Labels,
	}

	// Add workload
	if err := h.workloadManager.CreateWorkload(w); err != nil {
		log.Errorf("Failed to add workload: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"workload_error",
			"Failed to add workload",
			err.Error(),
		))
		return
	}

	// Convert IPs back to strings for response
	ipStrs := make([]string, len(w.IPs))
	for i, ip := range w.IPs {
		ipStrs[i] = ip.String()
	}

	// Return created workload
	response := models.WorkloadResponse{
		ID:     w.ID,
		Name:   w.Name,
		HostID: w.HostID,
		IPs:    ipStrs,
		Labels: w.Labels,
	}

	c.JSON(http.StatusCreated, response)
}

// ListWorkloads handles GET /api/v1/workloads
func (h *WorkloadHandler) ListWorkloads(c *gin.Context) {
	// Get all workloads
	workloads, err := h.workloadManager.ListWorkloads()
	if err != nil {
		log.Errorf("Failed to list workloads: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"workload_error",
			"Failed to list workloads",
			err.Error(),
		))
		return
	}

	// Convert to response format
	var workloadResponses []models.WorkloadResponse
	for _, w := range workloads {
		// Convert IPs to strings
		ipStrs := make([]string, len(w.IPs))
		for i, ip := range w.IPs {
			ipStrs[i] = ip.String()
		}

		workloadResponses = append(workloadResponses, models.WorkloadResponse{
			ID:     w.ID,
			Name:   w.Name,
			HostID: w.HostID,
			IPs:    ipStrs,
			Labels: w.Labels,
		})
	}

	response := models.WorkloadListResponse{
		Workloads: workloadResponses,
		Count:     len(workloadResponses),
	}

	c.JSON(http.StatusOK, response)
}

// GetWorkload handles GET /api/v1/workloads/:id
func (h *WorkloadHandler) GetWorkload(c *gin.Context) {
	// Get workload ID from URL parameter
	workloadID := c.Param("id")

	// Get workload
	w, err := h.workloadManager.GetWorkload(workloadID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse(
			http.StatusNotFound,
			"not_found",
			fmt.Sprintf("Workload with ID %s not found", workloadID),
			err.Error(),
		))
		return
	}

	// Convert IPs to strings
	ipStrs := make([]string, len(w.IPs))
	for i, ip := range w.IPs {
		ipStrs[i] = ip.String()
	}

	response := models.WorkloadResponse{
		ID:     w.ID,
		Name:   w.Name,
		HostID: w.HostID,
		IPs:    ipStrs,
		Labels: w.Labels,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateWorkload handles PUT /api/v1/workloads/:id
func (h *WorkloadHandler) UpdateWorkload(c *gin.Context) {
	// Get workload ID from URL parameter
	workloadID := c.Param("id")

	var req models.WorkloadRequest

	// Bind and validate JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			http.StatusBadRequest,
			"validation_error",
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Ensure ID matches
	if req.ID != workloadID {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			http.StatusBadRequest,
			"validation_error",
			"Workload ID in URL does not match ID in request body",
			nil,
		))
		return
	}

	// Parse IP addresses
	ips := make([]net.IP, 0, len(req.IPs))
	for _, ipStr := range req.IPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			c.JSON(http.StatusBadRequest, models.NewErrorResponse(
				http.StatusBadRequest,
				"validation_error",
				fmt.Sprintf("Invalid IP address: %s", ipStr),
				nil,
			))
			return
		}
		ips = append(ips, ip)
	}

	// Use hostname as HostID if not provided
	hostID := req.HostID
	if hostID == "" {
		hostID = "default-host"
	}

	// Create updated workload
	w := &workload.Workload{
		ID:     req.ID,
		Name:   req.Name,
		HostID: hostID,
		IPs:    ips,
		Labels: req.Labels,
	}

	// Update workload
	if err := h.workloadManager.UpdateWorkload(w); err != nil {
		log.Errorf("Failed to update workload: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"workload_error",
			"Failed to update workload",
			err.Error(),
		))
		return
	}

	// Convert IPs to strings
	ipStrs := make([]string, len(w.IPs))
	for i, ip := range w.IPs {
		ipStrs[i] = ip.String()
	}

	// Return updated workload
	response := models.WorkloadResponse{
		ID:     w.ID,
		Name:   w.Name,
		HostID: w.HostID,
		IPs:    ipStrs,
		Labels: w.Labels,
	}

	c.JSON(http.StatusOK, response)
}

// DeleteWorkload handles DELETE /api/v1/workloads/:id
func (h *WorkloadHandler) DeleteWorkload(c *gin.Context) {
	// Get workload ID from URL parameter
	workloadID := c.Param("id")

	// Delete workload
	if err := h.workloadManager.DeleteWorkload(workloadID); err != nil {
		log.Errorf("Failed to delete workload: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"workload_error",
			"Failed to delete workload",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Workload with ID %s deleted successfully", workloadID),
	})
}
