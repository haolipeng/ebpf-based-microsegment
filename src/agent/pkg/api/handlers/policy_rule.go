
// input: policy rule API requests (GET/POST/PUT/DELETE)
// output: policy rule JSON responses
// pos: policy rule API handlers - if file updated, must sync with this header comment and pkg/api/CLAUDE.md
package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/api/models"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/policy"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// PolicyRuleHandler handles policy rule management requests
type PolicyRuleHandler struct {
	policyManager *policy.PolicyManager
}

// NewPolicyRuleHandler creates a new policy rule handler
func NewPolicyRuleHandler(pm *policy.PolicyManager) *PolicyRuleHandler {
	return &PolicyRuleHandler{
		policyManager: pm,
	}
}

// CreatePolicyRule handles POST /api/v1/policy-rules
func (h *PolicyRuleHandler) CreatePolicyRule(c *gin.Context) {
	var req models.PolicyRuleRequest

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

	// Convert port ranges
	ports := make([]policy.PortRange, len(req.Ports))
	for i, p := range req.Ports {
		ports[i] = policy.PortRange{
			Start:    p.Start,
			End:      p.End,
			Protocol: p.Protocol,
		}
	}

	// Create policy rule
	rule := &policy.PolicyRule{
		Name:        req.Name,
		Description: req.Description,
		FromGroup:   req.FromGroup,
		ToGroup:     req.ToGroup,
		Ports:       ports,
		Action:      req.Action,
		Priority:    req.Priority,
		Enabled:     req.Enabled,
	}

	// Add policy rule (this will compile and load to eBPF)
	if err := h.policyManager.AddPolicyRule(rule); err != nil {
		log.Errorf("Failed to add policy rule: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"policy_rule_error",
			"Failed to add policy rule",
			err.Error(),
		))
		return
	}

	// Convert ports to response format
	portResponses := make([]models.PortRangeResponse, len(rule.Ports))
	for i, p := range rule.Ports {
		portResponses[i] = models.PortRangeResponse{
			Start:    p.Start,
			End:      p.End,
			Protocol: p.Protocol,
		}
	}

	// Return created policy rule
	response := models.PolicyRuleResponse{
		ID:          rule.ID,
		Name:        rule.Name,
		Description: rule.Description,
		FromGroup:   rule.FromGroup,
		ToGroup:     rule.ToGroup,
		Ports:       portResponses,
		Action:      rule.Action,
		Priority:    rule.Priority,
		Enabled:     rule.Enabled,
		CreatedAt:   rule.CreatedAt,
		UpdatedAt:   rule.UpdatedAt,
	}

	c.JSON(http.StatusCreated, response)
}

// ListPolicyRules handles GET /api/v1/policy-rules
func (h *PolicyRuleHandler) ListPolicyRules(c *gin.Context) {
	// Get all policy rules
	rules, err := h.policyManager.ListPolicyRules()
	if err != nil {
		log.Errorf("Failed to list policy rules: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"policy_rule_error",
			"Failed to list policy rules",
			err.Error(),
		))
		return
	}

	// Convert to response format
	var ruleResponses []models.PolicyRuleResponse
	for _, rule := range rules {
		// Convert ports
		portResponses := make([]models.PortRangeResponse, len(rule.Ports))
		for i, p := range rule.Ports {
			portResponses[i] = models.PortRangeResponse{
				Start:    p.Start,
				End:      p.End,
				Protocol: p.Protocol,
			}
		}

		ruleResponses = append(ruleResponses, models.PolicyRuleResponse{
			ID:          rule.ID,
			Name:        rule.Name,
			Description: rule.Description,
			FromGroup:   rule.FromGroup,
			ToGroup:     rule.ToGroup,
			Ports:       portResponses,
			Action:      rule.Action,
			Priority:    rule.Priority,
			Enabled:     rule.Enabled,
			CreatedAt:   rule.CreatedAt,
			UpdatedAt:   rule.UpdatedAt,
		})
	}

	response := models.PolicyRuleListResponse{
		Rules: ruleResponses,
		Count: len(ruleResponses),
	}

	c.JSON(http.StatusOK, response)
}

// GetPolicyRule handles GET /api/v1/policy-rules/:id
func (h *PolicyRuleHandler) GetPolicyRule(c *gin.Context) {
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

	// Get policy rule
	rule, err := h.policyManager.GetPolicyRule(uint32(ruleID))
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse(
			http.StatusNotFound,
			"not_found",
			fmt.Sprintf("Policy rule with ID %d not found", ruleID),
			err.Error(),
		))
		return
	}

	// Convert ports
	portResponses := make([]models.PortRangeResponse, len(rule.Ports))
	for i, p := range rule.Ports {
		portResponses[i] = models.PortRangeResponse{
			Start:    p.Start,
			End:      p.End,
			Protocol: p.Protocol,
		}
	}

	response := models.PolicyRuleResponse{
		ID:          rule.ID,
		Name:        rule.Name,
		Description: rule.Description,
		FromGroup:   rule.FromGroup,
		ToGroup:     rule.ToGroup,
		Ports:       portResponses,
		Action:      rule.Action,
		Priority:    rule.Priority,
		Enabled:     rule.Enabled,
		CreatedAt:   rule.CreatedAt,
		UpdatedAt:   rule.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// GetCompiledPolicies handles GET /api/v1/policy-rules/:id/compiled
func (h *PolicyRuleHandler) GetCompiledPolicies(c *gin.Context) {
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

	// Get compiled policies for this rule
	compiledPolicies, err := h.policyManager.GetCompiledPoliciesForRule(uint32(ruleID))
	if err != nil {
		log.Errorf("Failed to get compiled policies: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"policy_rule_error",
			"Failed to get compiled policies",
			err.Error(),
		))
		return
	}

	// Convert to response format
	var policyResponses []models.CompiledPolicyResponse
	for _, cp := range compiledPolicies {
		policyResponses = append(policyResponses, models.CompiledPolicyResponse{
			RuleID:          cp.RuleID,
			SourceRuleID:    cp.SourceRuleID,
			SrcIP:           cp.SrcIP,
			DstIP:           cp.DstIP,
			DstPort:         cp.DstPort,
			Protocol:        cp.Protocol,
			Action:          cp.Action,
			Priority:        cp.Priority,
			FromGroup:       cp.FromGroup,
			ToGroup:         cp.ToGroup,
			FromWorkloadID:  cp.FromWorkloadID,
			ToWorkloadID:    cp.ToWorkloadID,
			CompilationTime: cp.CompilationTime.Format("2006-01-02T15:04:05Z07:00"),
			CompilerVersion: cp.CompilerVersion,
		})
	}

	response := models.CompiledPoliciesResponse{
		Policies: policyResponses,
		Count:    len(policyResponses),
	}

	c.JSON(http.StatusOK, response)
}

// UpdatePolicyRule handles PUT /api/v1/policy-rules/:id
func (h *PolicyRuleHandler) UpdatePolicyRule(c *gin.Context) {
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

	var req models.PolicyRuleRequest

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

	// Ensure rule ID matches (if provided in body)
	if req.ID != 0 && req.ID != uint32(ruleID) {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			http.StatusBadRequest,
			"validation_error",
			"Rule ID in URL does not match rule ID in request body",
			nil,
		))
		return
	}

	// Convert port ranges
	ports := make([]policy.PortRange, len(req.Ports))
	for i, p := range req.Ports {
		ports[i] = policy.PortRange{
			Start:    p.Start,
			End:      p.End,
			Protocol: p.Protocol,
		}
	}

	// Create updated policy rule
	rule := &policy.PolicyRule{
		ID:          uint32(ruleID),
		Name:        req.Name,
		Description: req.Description,
		FromGroup:   req.FromGroup,
		ToGroup:     req.ToGroup,
		Ports:       ports,
		Action:      req.Action,
		Priority:    req.Priority,
		Enabled:     req.Enabled,
	}

	// Update policy rule (this will recompile)
	if err := h.policyManager.UpdatePolicyRule(rule); err != nil {
		log.Errorf("Failed to update policy rule: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"policy_rule_error",
			"Failed to update policy rule",
			err.Error(),
		))
		return
	}

	// Get the updated rule to return current state
	updatedRule, err := h.policyManager.GetPolicyRule(uint32(ruleID))
	if err != nil {
		log.Errorf("Failed to retrieve updated policy rule: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"policy_rule_error",
			"Rule updated but failed to retrieve details",
			err.Error(),
		))
		return
	}

	// Convert ports
	portResponses := make([]models.PortRangeResponse, len(updatedRule.Ports))
	for i, p := range updatedRule.Ports {
		portResponses[i] = models.PortRangeResponse{
			Start:    p.Start,
			End:      p.End,
			Protocol: p.Protocol,
		}
	}

	// Return updated policy rule
	response := models.PolicyRuleResponse{
		ID:          updatedRule.ID,
		Name:        updatedRule.Name,
		Description: updatedRule.Description,
		FromGroup:   updatedRule.FromGroup,
		ToGroup:     updatedRule.ToGroup,
		Ports:       portResponses,
		Action:      updatedRule.Action,
		Priority:    updatedRule.Priority,
		Enabled:     updatedRule.Enabled,
		CreatedAt:   updatedRule.CreatedAt,
		UpdatedAt:   updatedRule.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// DeletePolicyRule handles DELETE /api/v1/policy-rules/:id
func (h *PolicyRuleHandler) DeletePolicyRule(c *gin.Context) {
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

	// Delete policy rule (this will remove from eBPF and invalidate compiled policies)
	if err := h.policyManager.DeletePolicyRule(uint32(ruleID)); err != nil {
		log.Errorf("Failed to delete policy rule: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"policy_rule_error",
			"Failed to delete policy rule",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Policy rule with ID %d deleted successfully", ruleID),
	})
}
