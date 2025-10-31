package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ebpf-microsegment/src/agent/pkg/api/models"
	"github.com/ebpf-microsegment/src/agent/pkg/policy"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// PolicyHandler handles policy management requests
type PolicyHandler struct{
	policyManager *policy.PolicyManager
}

// NewPolicyHandler creates a new policy handler
func NewPolicyHandler(pm *policy.PolicyManager) *PolicyHandler {
	return &PolicyHandler{
		policyManager: pm,
	}
}

// CreatePolicy handles POST /api/v1/policies
func (h *PolicyHandler) CreatePolicy(c *gin.Context) {
	var req models.PolicyRequest

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

	// Convert to internal policy format
	p := &policy.Policy{
		RuleID:   req.RuleID,
		SrcIP:    req.SrcIP,
		DstIP:    req.DstIP,
		SrcPort:  req.SrcPort,
		DstPort:  req.DstPort,
		Protocol: req.Protocol,
		Action:   req.Action,
		Priority: req.Priority,
	}

	// Add policy
	if err := h.policyManager.AddPolicy(p); err != nil {
		log.Errorf("Failed to add policy: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"policy_error",
			"Failed to add policy",
			err.Error(),
		))
		return
	}

	// Return created policy
	response := models.PolicyResponse{
		RuleID:   p.RuleID,
		SrcIP:    p.SrcIP,
		DstIP:    p.DstIP,
		SrcPort:  p.SrcPort,
		DstPort:  p.DstPort,
		Protocol: p.Protocol,
		Action:   p.Action,
		Priority: p.Priority,
	}

	c.JSON(http.StatusCreated, response)
}

// ListPolicies handles GET /api/v1/policies
func (h *PolicyHandler) ListPolicies(c *gin.Context) {
	// Get all policies
	policies, err := h.policyManager.ListPolicies()
	if err != nil {
		log.Errorf("Failed to list policies: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"policy_error",
			"Failed to list policies",
			err.Error(),
		))
		return
	}

	// Convert to response format
	var policyResponses []models.PolicyResponse
	for _, p := range policies {
		policyResponses = append(policyResponses, models.PolicyResponse{
			RuleID:   p.RuleID,
			SrcIP:    p.SrcIP,
			DstIP:    p.DstIP,
			SrcPort:  p.SrcPort,
			DstPort:  p.DstPort,
			Protocol: p.Protocol,
			Action:   p.Action,
			Priority: p.Priority,
		})
	}

	response := models.PolicyListResponse{
		Policies: policyResponses,
		Count:    len(policyResponses),
	}

	c.JSON(http.StatusOK, response)
}

// GetPolicy handles GET /api/v1/policies/:id
func (h *PolicyHandler) GetPolicy(c *gin.Context) {
	// Get rule ID from URL parameter
	ruleIDStr := c.Param("id")
	ruleID, err := strconv.ParseUint(ruleIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			http.StatusBadRequest,
			"validation_error",
			"Invalid rule ID",
			err.Error(),
		))
		return
	}

	// Get all policies and find the matching one
	policies, err := h.policyManager.ListPolicies()
	if err != nil {
		log.Errorf("Failed to list policies: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"policy_error",
			"Failed to retrieve policy",
			err.Error(),
		))
		return
	}

	// Find policy with matching rule ID
	for _, p := range policies {
		if p.RuleID == uint32(ruleID) {
			response := models.PolicyResponse{
				RuleID:   p.RuleID,
				SrcIP:    p.SrcIP,
				DstIP:    p.DstIP,
				SrcPort:  p.SrcPort,
				DstPort:  p.DstPort,
				Protocol: p.Protocol,
				Action:   p.Action,
				Priority: p.Priority,
			}
			c.JSON(http.StatusOK, response)
			return
		}
	}

	// Policy not found
	c.JSON(http.StatusNotFound, models.NewErrorResponse(
		http.StatusNotFound,
		"not_found",
		fmt.Sprintf("Policy with rule ID %d not found", ruleID),
		nil,
	))
}

// UpdatePolicy handles PUT /api/v1/policies/:id
func (h *PolicyHandler) UpdatePolicy(c *gin.Context) {
	// Get rule ID from URL parameter
	ruleIDStr := c.Param("id")
	ruleID, err := strconv.ParseUint(ruleIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			http.StatusBadRequest,
			"validation_error",
			"Invalid rule ID",
			err.Error(),
		))
		return
	}

	var req models.PolicyRequest

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

	// Ensure rule ID matches
	if req.RuleID != uint32(ruleID) {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			http.StatusBadRequest,
			"validation_error",
			"Rule ID in URL does not match rule ID in request body",
			nil,
		))
		return
	}

	// Convert to internal policy format
	p := &policy.Policy{
		RuleID:   req.RuleID,
		SrcIP:    req.SrcIP,
		DstIP:    req.DstIP,
		SrcPort:  req.SrcPort,
		DstPort:  req.DstPort,
		Protocol: req.Protocol,
		Action:   req.Action,
		Priority: req.Priority,
	}

	// Delete old policy first
	if err := h.policyManager.DeletePolicy(p); err != nil {
		// If delete fails, policy might not exist
		log.Warnf("Failed to delete old policy during update: %v", err)
	}

	// Add updated policy
	if err := h.policyManager.AddPolicy(p); err != nil {
		log.Errorf("Failed to update policy: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"policy_error",
			"Failed to update policy",
			err.Error(),
		))
		return
	}

	// Return updated policy
	response := models.PolicyResponse{
		RuleID:   p.RuleID,
		SrcIP:    p.SrcIP,
		DstIP:    p.DstIP,
		SrcPort:  p.SrcPort,
		DstPort:  p.DstPort,
		Protocol: p.Protocol,
		Action:   p.Action,
		Priority: p.Priority,
	}

	c.JSON(http.StatusOK, response)
}

// DeletePolicy handles DELETE /api/v1/policies/:id
func (h *PolicyHandler) DeletePolicy(c *gin.Context) {
	// Get rule ID from URL parameter
	ruleIDStr := c.Param("id")
	ruleID, err := strconv.ParseUint(ruleIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			http.StatusBadRequest,
			"validation_error",
			"Invalid rule ID",
			err.Error(),
		))
		return
	}

	// Get all policies to find the one to delete
	policies, err := h.policyManager.ListPolicies()
	if err != nil {
		log.Errorf("Failed to list policies: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"policy_error",
			"Failed to retrieve policy for deletion",
			err.Error(),
		))
		return
	}

	// Find policy with matching rule ID
	var policyToDelete *policy.Policy
	for i := range policies {
		if policies[i].RuleID == uint32(ruleID) {
			policyToDelete = &policies[i]
			break
		}
	}

	if policyToDelete == nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse(
			http.StatusNotFound,
			"not_found",
			fmt.Sprintf("Policy with rule ID %d not found", ruleID),
			nil,
		))
		return
	}

	// Delete policy
	if err := h.policyManager.DeletePolicy(policyToDelete); err != nil {
		log.Errorf("Failed to delete policy: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"policy_error",
			"Failed to delete policy",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Policy with rule ID %d deleted successfully", ruleID),
	})
}

