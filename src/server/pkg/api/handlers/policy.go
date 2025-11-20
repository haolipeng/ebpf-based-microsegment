package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	commonpb "github.com/haolipeng/ebpf-based-microsegment/api/proto/common"
	policypb "github.com/haolipeng/ebpf-based-microsegment/api/proto/policy"
	"github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/storage"
)

// PolicyHandler 处理策略相关的 HTTP 请求
type PolicyHandler struct {
	storage *storage.PolicyStorage
}

// NewPolicyHandler 创建新的 PolicyHandler
func NewPolicyHandler(storage *storage.PolicyStorage) *PolicyHandler {
	return &PolicyHandler{storage: storage}
}

// RegisterRoutes 注册策略相关的路由
func (h *PolicyHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/policies", h.ListPolicies)
	r.POST("/policies", h.CreatePolicy)
	r.PUT("/policies/:id", h.UpdatePolicy)
	r.DELETE("/policies/:id", h.DeletePolicy)
}

// ListPolicies 获取所有策略
func (h *PolicyHandler) ListPolicies(c *gin.Context) {
	policies, version, err := h.storage.GetAllPolicies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"policies": policies,
		"version":  version,
	})
}

// CreatePolicyRequest 创建策略请求
type CreatePolicyRequest struct {
	SrcIP        string             `json:"src_ip"`
	DstIP        string             `json:"dst_ip"`
	SrcPort      uint32             `json:"src_port"`
	DstPort      uint32             `json:"dst_port"`
	Protocol     int32              `json:"protocol"`
	Action       int32              `json:"action" binding:"required,min=0,max=2"`    // 0=ALLOW, 1=DENY, 2=LOG
	Priority     uint32             `json:"priority" binding:"required,min=0,max=100"`
	SourceLabels map[string]string  `json:"source_labels"`
	DestLabels   map[string]string  `json:"dest_labels"`
	Description  string             `json:"description"`
	ProcessName  string             `json:"process_name" binding:"omitempty,max=255"`
	ProcessPath  string             `json:"process_path" binding:"omitempty,max=1024"`
	MatchMode    int32              `json:"match_mode" binding:"omitempty,min=0,max=2"` // 0=EXACT, 1=PREFIX, 2=WILDCARD
}

// CreatePolicy 创建新策略
func (h *PolicyHandler) CreatePolicy(c *gin.Context) {
	var req CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 创建策略对象
	policy := &policypb.Policy{
		SrcIp:        req.SrcIP,
		DstIp:        req.DstIP,
		SrcPort:      req.SrcPort,
		DstPort:      req.DstPort,
		Protocol:     commonpb.Protocol(req.Protocol),
		Action:       commonpb.PolicyAction(req.Action),
		Priority:     req.Priority,
		SourceLabels: req.SourceLabels,
		DestLabels:   req.DestLabels,
		Description:  req.Description,
		ProcessName:  req.ProcessName,
		ProcessPath:  req.ProcessPath,
		MatchMode:    policypb.ProcessMatchMode(req.MatchMode),
	}

	// 保存到数据库
	if err := h.storage.CreatePolicy(c.Request.Context(), policy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create policy: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// UpdatePolicy 更新策略
func (h *PolicyHandler) UpdatePolicy(c *gin.Context) {
	// 获取策略 ID
	idStr := c.Param("id")
	ruleID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid policy ID"})
		return
	}

	var req CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 创建更新的策略对象
	policy := &policypb.Policy{
		RuleId:       uint32(ruleID),
		SrcIp:        req.SrcIP,
		DstIp:        req.DstIP,
		SrcPort:      req.SrcPort,
		DstPort:      req.DstPort,
		Protocol:     commonpb.Protocol(req.Protocol),
		Action:       commonpb.PolicyAction(req.Action),
		Priority:     req.Priority,
		SourceLabels: req.SourceLabels,
		DestLabels:   req.DestLabels,
		Description:  req.Description,
		ProcessName:  req.ProcessName,
		ProcessPath:  req.ProcessPath,
		MatchMode:    policypb.ProcessMatchMode(req.MatchMode),
	}

	// 更新数据库
	if err := h.storage.UpdatePolicy(c.Request.Context(), policy); err != nil {
		if err.Error() == "policy not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update policy: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// DeletePolicy 删除策略
func (h *PolicyHandler) DeletePolicy(c *gin.Context) {
	// 获取策略 ID
	idStr := c.Param("id")
	ruleID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid policy ID"})
		return
	}

	// 删除策略
	if err := h.storage.DeletePolicy(c.Request.Context(), uint32(ruleID)); err != nil {
		if err.Error() == "policy not found: "+idStr {
			c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete policy: " + err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
