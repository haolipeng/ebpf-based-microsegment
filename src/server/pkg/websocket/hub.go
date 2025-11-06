package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	flowpb "github.com/ebpf-microsegment/src/proto/flow"
	log "github.com/sirupsen/logrus"
)

// Client represents a WebSocket client connection
type Client struct {
	// WebSocket connection
	Conn *websocket.Conn

	// Buffered channel for outbound messages
	Send chan []byte

	// Client subscription filters
	Filters *FlowFilter

	// Hub reference
	Hub *Hub

	// Unique client ID
	ID string

	// Last activity timestamp
	LastActivity time.Time
}

// FlowFilter defines subscription filters for a WebSocket client
type FlowFilter struct {
	// Protocol filter (e.g., "TCP", "UDP", "ICMP")
	Protocol *string `json:"protocol,omitempty"`

	// Source IP filter
	SourceIP *string `json:"source_ip,omitempty"`

	// Destination IP filter
	DestIP *string `json:"dest_ip,omitempty"`

	// Agent ID filter
	AgentID *string `json:"agent_id,omitempty"`

	// Policy action filter (e.g., "ALLOW", "DENY")
	PolicyAction *string `json:"policy_action,omitempty"`

	// Flow state filter (e.g., "ACTIVE", "CLOSED")
	State *string `json:"state,omitempty"`

	// Source labels filter
	SourceLabels map[string]string `json:"source_labels,omitempty"`

	// Destination labels filter
	DestLabels map[string]string `json:"dest_labels,omitempty"`
}

// Hub maintains the set of active WebSocket clients and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from FlowService
	broadcast chan *flowpb.Flow

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe operations
	mutex sync.RWMutex

	// Message buffer size
	messageBufferSize int

	// Statistics
	stats HubStats
}

// HubStats tracks WebSocket hub statistics
type HubStats struct {
	ConnectedClients int   `json:"connected_clients"`
	TotalMessages    int64 `json:"total_messages"`
	DroppedMessages  int64 `json:"dropped_messages"`
	TotalConnections int64 `json:"total_connections"`
	TotalDisconnects int64 `json:"total_disconnects"`
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:           make(map[*Client]bool),
		broadcast:         make(chan *flowpb.Flow, 256), // Buffer for 256 messages
		register:          make(chan *Client),
		unregister:        make(chan *Client),
		messageBufferSize: 256,
		stats:             HubStats{},
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	log.Info("WebSocket hub started")

	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.stats.ConnectedClients = len(h.clients)
			h.stats.TotalConnections++
			h.mutex.Unlock()
			log.Infof("WebSocket client registered: %s (total: %d)", client.ID, len(h.clients))

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				h.stats.ConnectedClients = len(h.clients)
				h.stats.TotalDisconnects++
			}
			h.mutex.Unlock()
			log.Infof("WebSocket client unregistered: %s (total: %d)", client.ID, len(h.clients))

		case flow := <-h.broadcast:
			h.mutex.RLock()
			clientCount := len(h.clients)
			h.mutex.RUnlock()

			if clientCount == 0 {
				continue
			}

			// Serialize message once
			message, err := json.Marshal(flow)
			if err != nil {
				log.Errorf("Failed to marshal flow for WebSocket: %v", err)
				continue
			}

			h.stats.TotalMessages++

			// Broadcast to all matching clients
			h.mutex.RLock()
			for client := range h.clients {
				// Check if flow matches client's filter
				if !h.matchesFilter(flow, client.Filters) {
					continue
				}

				// Non-blocking send
				select {
				case client.Send <- message:
					// Message sent successfully
				default:
					// Client buffer full, drop message
					h.stats.DroppedMessages++
					log.Warnf("Dropped message for slow client %s", client.ID)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// Broadcast sends a flow to all connected WebSocket clients
func (h *Hub) Broadcast(flow *flowpb.Flow) {
	// Non-blocking send to avoid blocking the gRPC handler
	select {
	case h.broadcast <- flow:
		// Message queued successfully
	default:
		// Broadcast channel full, drop message
		h.stats.DroppedMessages++
		log.Warn("WebSocket broadcast channel full, message dropped")
	}
}

// Register adds a new client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// GetStats returns current hub statistics
func (h *Hub) GetStats() HubStats {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.stats
}

// matchesFilter checks if a flow matches the client's subscription filter
func (h *Hub) matchesFilter(flow *flowpb.Flow, filter *FlowFilter) bool {
	if filter == nil {
		// No filter, accept all flows
		return true
	}

	// Check protocol filter
	if filter.Protocol != nil && flow.Protocol.String() != *filter.Protocol {
		return false
	}

	// Check source IP filter
	if filter.SourceIP != nil && flow.SrcIp != *filter.SourceIP {
		return false
	}

	// Check destination IP filter
	if filter.DestIP != nil && flow.DstIp != *filter.DestIP {
		return false
	}

	// Check agent ID filter
	if filter.AgentID != nil && flow.AgentId != *filter.AgentID {
		return false
	}

	// Check policy action filter
	if filter.PolicyAction != nil && flow.PolicyAction.String() != *filter.PolicyAction {
		return false
	}

	// Check state filter
	if filter.State != nil && flow.State.String() != *filter.State {
		return false
	}

	// Check source labels filter
	if len(filter.SourceLabels) > 0 {
		if !matchesLabels(flow.SourceLabels, filter.SourceLabels) {
			return false
		}
	}

	// Check destination labels filter
	if len(filter.DestLabels) > 0 {
		if !matchesLabels(flow.DestLabels, filter.DestLabels) {
			return false
		}
	}

	return true
}

// matchesLabels checks if flow labels match the filter labels
func matchesLabels(flowLabels, filterLabels map[string]string) bool {
	for key, value := range filterLabels {
		if flowLabel, ok := flowLabels[key]; !ok || flowLabel != value {
			return false
		}
	}
	return true
}

// Constants for WebSocket configuration
const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		c.LastActivity = time.Now()
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("WebSocket error for client %s: %v", c.ID, err)
			}
			break
		}

		// Parse filter update from client
		var filter FlowFilter
		if err := json.Unmarshal(message, &filter); err != nil {
			log.Warnf("Invalid filter message from client %s: %v", c.ID, err)
			continue
		}

		// Update client filter
		c.Filters = &filter
		log.Debugf("Client %s updated filter: %+v", c.ID, filter)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
