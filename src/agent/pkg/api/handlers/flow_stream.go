package handlers

import (
	"net/http"
	"time"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/flow"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow connections from any origin (adjust for production)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// FlowStreamHandler handles WebSocket connections for real-time flow streaming
type FlowStreamHandler struct {
	hub *flow.Hub
}

// NewFlowStreamHandler creates a new flow stream handler
func NewFlowStreamHandler(hub *flow.Hub) *FlowStreamHandler {
	return &FlowStreamHandler{
		hub: hub,
	}
}

// HandleWebSocket handles GET /api/v1/flows/stream WebSocket upgrade
// Query parameters:
//   - protocol: Filter by protocol (TCP/UDP/ICMP)
//   - source_ip: Filter by source IP
//   - dest_ip: Filter by destination IP
//   - policy_action: Filter by policy action (ALLOW/DENY)
//   - state: Filter by state (ACTIVE/CLOSED)
//   - event_type: Filter by event type (NEW/UPDATE/CLOSED)
//
// Clients can also send filter updates via WebSocket messages in JSON format.
//
// Example WebSocket message for filter update:
//
//	{
//	  "protocol": "TCP",
//	  "policy_action": "DENY",
//	  "source_labels": {
//	    "app": "nginx"
//	  }
//	}
func (h *FlowStreamHandler) HandleWebSocket(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorf("Failed to upgrade WebSocket connection: %v", err)
		return
	}

	// Generate unique client ID
	clientID := uuid.New().String()

	// Parse initial filter from query parameters
	filter := h.parseFilterFromQuery(c)

	// Create client
	client := &flow.Client{
		Conn:         conn,
		Send:         make(chan []byte, 256),
		Filters:      filter,
		Hub:          h.hub,
		ID:           clientID,
		LastActivity: time.Now(),
	}

	// Register client with hub
	h.hub.Register(client)

	log.Infof("WebSocket client %s connected from %s", clientID, c.ClientIP())

	// Start client goroutines
	go client.WritePump()
	go client.ReadPump()
}

// GetHubStats handles GET /api/v1/flows/stream/stats
// Returns WebSocket hub statistics
func (h *FlowStreamHandler) GetHubStats(c *gin.Context) {
	stats := h.hub.GetStats()
	c.JSON(http.StatusOK, stats)
}

// parseFilterFromQuery parses flow filter from query parameters
func (h *FlowStreamHandler) parseFilterFromQuery(c *gin.Context) *flow.FlowFilter {
	filter := &flow.FlowFilter{}

	// Parse protocol filter
	if protocol := c.Query("protocol"); protocol != "" {
		filter.Protocol = &protocol
	}

	// Parse source IP filter
	if sourceIP := c.Query("source_ip"); sourceIP != "" {
		filter.SourceIP = &sourceIP
	}

	// Parse destination IP filter
	if destIP := c.Query("dest_ip"); destIP != "" {
		filter.DestIP = &destIP
	}

	// Parse policy action filter
	if policyAction := c.Query("policy_action"); policyAction != "" {
		filter.PolicyAction = &policyAction
	}

	// Parse state filter
	if state := c.Query("state"); state != "" {
		filter.State = &state
	}

	// Parse event type filter
	if eventType := c.Query("event_type"); eventType != "" {
		filter.EventType = &eventType
	}

	// Parse source labels (format: app=nginx,env=prod)
	if sourceLabels := c.Query("source_labels"); sourceLabels != "" {
		filter.SourceLabels = parseLabels(sourceLabels)
	}

	// Parse destination labels
	if destLabels := c.Query("dest_labels"); destLabels != "" {
		filter.DestLabels = parseLabels(destLabels)
	}

	log.Debugf("WebSocket client filter: %+v", filter)
	return filter
}

// parseLabels parses label string (format: key1=value1,key2=value2)
func parseLabels(labelStr string) map[string]string {
	labels := make(map[string]string)

	// Simple parsing (can be improved with proper CSV parser)
	// Format: app=nginx,env=prod
	pairs := splitLabels(labelStr)
	for _, pair := range pairs {
		kv := splitKeyValue(pair)
		if len(kv) == 2 {
			labels[kv[0]] = kv[1]
		}
	}

	return labels
}

// splitLabels splits label string by comma
func splitLabels(s string) []string {
	var result []string
	var current string

	for _, char := range s {
		if char == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}

// splitKeyValue splits key=value pair
func splitKeyValue(s string) []string {
	for i, char := range s {
		if char == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
