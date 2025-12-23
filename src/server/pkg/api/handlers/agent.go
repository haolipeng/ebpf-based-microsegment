// input: HTTP requests (GET /agents, GET /agents/:id), AgentStorage
// output: JSON responses with agent list or status
// pos: api/handlers - HTTP handler for Agent management endpoints

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/haolipeng/ebpf-based-microsegment/src/server/pkg/storage"
)

// AgentHandler 处理 Agent 相关的 HTTP 请求
type AgentHandler struct {
	storage *storage.AgentStorage
}

// NewAgentHandler 创建新的 AgentHandler
func NewAgentHandler(storage *storage.AgentStorage) *AgentHandler {
	return &AgentHandler{storage: storage}
}

// RegisterRoutes 注册 Agent 相关的路由
func (h *AgentHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/agents", h.ListAgents)
	r.GET("/agents/:id", h.GetAgent)
}

// ListAgents 获取所有 Agent
func (h *AgentHandler) ListAgents(c *gin.Context) {
	agents, err := h.storage.GetAllAgents(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agents)
}

// GetAgent 获取单个 Agent 详情
func (h *AgentHandler) GetAgent(c *gin.Context) {
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent ID is required"})
		return
	}

	agent, err := h.storage.GetAgentByID(c.Request.Context(), agentID)
	if err != nil {
		if err.Error() == "agent not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, agent)
}
