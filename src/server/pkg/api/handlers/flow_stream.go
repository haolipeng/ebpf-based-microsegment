package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	ws "github.com/ebpf-microsegment/src/server/pkg/websocket"
	log "github.com/sirupsen/logrus"
)

// FlowStreamHandler handles WebSocket connections for real-time flow streaming
type FlowStreamHandler struct {
	hub      *ws.Hub
	upgrader websocket.Upgrader
}

// NewFlowStreamHandler creates a new FlowStreamHandler
func NewFlowStreamHandler(hub *ws.Hub) *FlowStreamHandler {
	return &FlowStreamHandler{
		hub: hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for now (should be restricted in production)
				return true
			},
		},
	}
}

// RegisterRoutes registers WebSocket routes to the router
func (h *FlowStreamHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/flows/stream", h.HandleWebSocket)
	rg.GET("/flows/stream/stats", h.GetStats)
}

// HandleWebSocket handles WebSocket upgrade and client connection
func (h *FlowStreamHandler) HandleWebSocket(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorf("Failed to upgrade WebSocket connection: %v", err)
		return
	}

	// Create new client
	client := &ws.Client{
		Conn:         conn,
		Send:         make(chan []byte, 256),
		Filters:      nil, // No filter by default (accept all flows)
		Hub:          h.hub,
		ID:           uuid.New().String(),
		LastActivity: time.Now(),
	}

	// Register client with hub
	h.hub.Register(client)

	log.Infof("WebSocket client connected: %s from %s", client.ID, c.Request.RemoteAddr)

	// Start client goroutines
	go client.WritePump()
	go client.ReadPump()
}

// GetStats returns WebSocket hub statistics
func (h *FlowStreamHandler) GetStats(c *gin.Context) {
	stats := h.hub.GetStats()
	c.JSON(http.StatusOK, stats)
}
