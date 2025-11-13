package handlers

import (
	"fmt"
	"net/http"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/api/models"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/groups"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// GroupHandler handles group management requests
type GroupHandler struct {
	groupManager *groups.GroupManager
}

// NewGroupHandler creates a new group handler
func NewGroupHandler(gm *groups.GroupManager) *GroupHandler {
	return &GroupHandler{
		groupManager: gm,
	}
}

// CreateGroup handles POST /api/v1/groups
func (h *GroupHandler) CreateGroup(c *gin.Context) {
	var req models.GroupRequest

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

	// Create group object
	group := groups.NewGroup(req.Name)

	// Build label selectors from MatchLabels
	if len(req.MatchLabels) > 0 {
		for key, value := range req.MatchLabels {
			selector := groups.NewEqualSelector(key, value)
			group.AddSelector(selector)
		}
	}

	// Create group
	if err := h.groupManager.CreateGroup(group); err != nil {
		log.Errorf("Failed to create group: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"group_error",
			"Failed to create group",
			err.Error(),
		))
		return
	}

	// Resolve members to get count
	members, _ := h.groupManager.ResolveGroupMembers(req.Name)
	memberIDs := make([]string, len(members))
	for i, m := range members {
		memberIDs[i] = m.ID
	}

	// Return created group
	response := models.GroupResponse{
		Name:        group.Name,
		MatchLabels: req.MatchLabels,
		MemberIDs:   memberIDs,
		MemberCount: len(memberIDs),
		IsStatic:    false,
	}

	c.JSON(http.StatusCreated, response)
}

// ListGroups handles GET /api/v1/groups
func (h *GroupHandler) ListGroups(c *gin.Context) {
	// Get all groups
	groupList, err := h.groupManager.ListGroups()
	if err != nil {
		log.Errorf("Failed to list groups: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"group_error",
			"Failed to list groups",
			err.Error(),
		))
		return
	}

	// Convert to response format
	var groupResponses []models.GroupResponse
	for _, g := range groupList {
		// Resolve members to get count
		members, _ := h.groupManager.ResolveGroupMembers(g.Name)
		memberIDs := make([]string, len(members))
		for i, m := range members {
			memberIDs[i] = m.ID
		}

		// Extract MatchLabels from selectors (simplified - assumes OpEqual selectors)
		matchLabels := make(map[string]string)
		for _, sel := range g.Selectors {
			if sel.Operator == groups.OpEqual && len(sel.Values) > 0 {
				matchLabels[sel.Key] = sel.Values[0]
			}
		}

		groupResponses = append(groupResponses, models.GroupResponse{
			Name:        g.Name,
			MatchLabels: matchLabels,
			MemberIDs:   memberIDs,
			MemberCount: len(memberIDs),
			IsStatic:    false,
		})
	}

	response := models.GroupListResponse{
		Groups: groupResponses,
		Count:  len(groupResponses),
	}

	c.JSON(http.StatusOK, response)
}

// GetGroup handles GET /api/v1/groups/:name
func (h *GroupHandler) GetGroup(c *gin.Context) {
	// Get group name from URL parameter
	groupName := c.Param("name")

	// Get group
	group, err := h.groupManager.GetGroup(groupName)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse(
			http.StatusNotFound,
			"not_found",
			fmt.Sprintf("Group %s not found", groupName),
			err.Error(),
		))
		return
	}

	// Resolve members
	members, _ := h.groupManager.ResolveGroupMembers(groupName)
	memberIDs := make([]string, len(members))
	for i, m := range members {
		memberIDs[i] = m.ID
	}

	// Extract MatchLabels from selectors
	matchLabels := make(map[string]string)
	for _, sel := range group.Selectors {
		if sel.Operator == groups.OpEqual && len(sel.Values) > 0 {
			matchLabels[sel.Key] = sel.Values[0]
		}
	}

	response := models.GroupResponse{
		Name:        group.Name,
		MatchLabels: matchLabels,
		MemberIDs:   memberIDs,
		MemberCount: len(memberIDs),
		IsStatic:    false,
	}

	c.JSON(http.StatusOK, response)
}

// GetGroupMembers handles GET /api/v1/groups/:name/members
func (h *GroupHandler) GetGroupMembers(c *gin.Context) {
	// Get group name from URL parameter
	groupName := c.Param("name")

	// Check if group exists
	_, err := h.groupManager.GetGroup(groupName)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse(
			http.StatusNotFound,
			"not_found",
			fmt.Sprintf("Group %s not found", groupName),
			err.Error(),
		))
		return
	}

	// Resolve members
	members, err := h.groupManager.ResolveGroupMembers(groupName)
	if err != nil {
		log.Errorf("Failed to resolve group members: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"group_error",
			"Failed to resolve group members",
			err.Error(),
		))
		return
	}

	// Convert members to response format
	var memberResponses []models.WorkloadResponse
	for _, m := range members {
		// Convert IPs to strings
		ipStrs := make([]string, len(m.IPs))
		for i, ip := range m.IPs {
			ipStrs[i] = ip.String()
		}

		memberResponses = append(memberResponses, models.WorkloadResponse{
			ID:     m.ID,
			Name:   m.Name,
			IPs:    ipStrs,
			Labels: m.Labels,
		})
	}

	response := models.GroupMembersResponse{
		GroupName: groupName,
		Members:   memberResponses,
		Count:     len(memberResponses),
	}

	c.JSON(http.StatusOK, response)
}

// UpdateGroup handles PUT /api/v1/groups/:name
func (h *GroupHandler) UpdateGroup(c *gin.Context) {
	// Get group name from URL parameter
	groupName := c.Param("name")

	var req models.GroupRequest

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

	// Ensure name matches
	if req.Name != groupName {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			http.StatusBadRequest,
			"validation_error",
			"Group name in URL does not match name in request body",
			nil,
		))
		return
	}

	// Get existing group to preserve created_at
	existingGroup, err := h.groupManager.GetGroup(groupName)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse(
			http.StatusNotFound,
			"not_found",
			fmt.Sprintf("Group %s not found", groupName),
			err.Error(),
		))
		return
	}

	// Update selectors
	existingGroup.Selectors = []groups.LabelSelector{}
	if len(req.MatchLabels) > 0 {
		for key, value := range req.MatchLabels {
			selector := groups.NewEqualSelector(key, value)
			existingGroup.AddSelector(selector)
		}
	}

	// Update group
	if err := h.groupManager.UpdateGroup(existingGroup); err != nil {
		log.Errorf("Failed to update group: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"group_error",
			"Failed to update group",
			err.Error(),
		))
		return
	}

	// Resolve members
	members, _ := h.groupManager.ResolveGroupMembers(req.Name)
	memberIDs := make([]string, len(members))
	for i, m := range members {
		memberIDs[i] = m.ID
	}

	// Return updated group
	response := models.GroupResponse{
		Name:        existingGroup.Name,
		MatchLabels: req.MatchLabels,
		MemberIDs:   memberIDs,
		MemberCount: len(memberIDs),
		IsStatic:    false,
	}

	c.JSON(http.StatusOK, response)
}

// DeleteGroup handles DELETE /api/v1/groups/:name
func (h *GroupHandler) DeleteGroup(c *gin.Context) {
	// Get group name from URL parameter
	groupName := c.Param("name")

	// Delete group
	if err := h.groupManager.DeleteGroup(groupName); err != nil {
		log.Errorf("Failed to delete group: %v", err)
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			http.StatusInternalServerError,
			"group_error",
			"Failed to delete group",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Group %s deleted successfully", groupName),
	})
}
