package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	ws "github.com/ebpf-microsegment/src/server/pkg/websocket"
	commonpb "github.com/ebpf-microsegment/src/proto/common"
	flowpb "github.com/ebpf-microsegment/src/proto/flow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewFlowStreamHandler tests the creation of a new FlowStreamHandler
func TestNewFlowStreamHandler(t *testing.T) {
	hub := ws.NewHub()
	handler := NewFlowStreamHandler(hub)

	assert.NotNil(t, handler)
	assert.Equal(t, hub, handler.hub)
	assert.NotNil(t, handler.upgrader)
}

// TestFlowStreamHandler_RegisterRoutes tests that routes are properly registered
func TestFlowStreamHandler_RegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	hub := ws.NewHub()
	handler := NewFlowStreamHandler(hub)

	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	// Get registered routes
	routes := router.Routes()

	// Check that the WebSocket stream route is registered
	hasStreamRoute := false
	hasStatsRoute := false

	for _, route := range routes {
		if route.Method == "GET" && route.Path == "/api/v1/flows/stream" {
			hasStreamRoute = true
		}
		if route.Method == "GET" && route.Path == "/api/v1/flows/stream/stats" {
			hasStatsRoute = true
		}
	}

	assert.True(t, hasStreamRoute, "WebSocket stream route should be registered")
	assert.True(t, hasStatsRoute, "Stats route should be registered")
}

// TestFlowStreamHandler_GetStats tests the GetStats endpoint
func TestFlowStreamHandler_GetStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	hub := ws.NewHub()
	handler := NewFlowStreamHandler(hub)

	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	// Make request to stats endpoint
	req, _ := http.NewRequest("GET", "/api/v1/flows/stream/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var stats ws.HubStats
	err := json.Unmarshal(w.Body.Bytes(), &stats)
	require.NoError(t, err)

	// Initial stats should be zero
	assert.Equal(t, 0, stats.ConnectedClients)
	assert.Equal(t, int64(0), stats.TotalMessages)
	assert.Equal(t, int64(0), stats.DroppedMessages)
}

// TestFlowStreamHandler_GetStats_WithActivity tests stats after hub activity
func TestFlowStreamHandler_GetStats_WithActivity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	hub := ws.NewHub()
	handler := NewFlowStreamHandler(hub)

	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	// Start the hub in background
	go hub.Run()

	// Create a mock client (don't actually connect WebSocket, just simulate registration)
	mockClient := &ws.Client{
		Conn:         nil, // We won't use the actual connection
		Send:         make(chan []byte, 256),
		Filters:      nil,
		Hub:          hub,
		ID:           "test-client",
		LastActivity: time.Now(),
	}

	// Register the client
	hub.Register(mockClient)

	// Give hub time to process registration
	time.Sleep(50 * time.Millisecond)

	// Make request to stats endpoint
	req, _ := http.NewRequest("GET", "/api/v1/flows/stream/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var stats ws.HubStats
	err := json.Unmarshal(w.Body.Bytes(), &stats)
	require.NoError(t, err)

	// Should show one connected client
	assert.Equal(t, 1, stats.ConnectedClients)
	assert.Equal(t, int64(1), stats.TotalConnections)

	// Unregister the client
	hub.Unregister(mockClient)
	time.Sleep(50 * time.Millisecond)

	// Check stats again
	req2, _ := http.NewRequest("GET", "/api/v1/flows/stream/stats", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	err = json.Unmarshal(w2.Body.Bytes(), &stats)
	require.NoError(t, err)

	assert.Equal(t, 0, stats.ConnectedClients)
	assert.Equal(t, int64(1), stats.TotalDisconnects)
}

// TestFlowStreamHandler_WebSocketUpgrade tests WebSocket connection upgrade
func TestFlowStreamHandler_WebSocketUpgrade(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := ws.NewHub()
	handler := NewFlowStreamHandler(hub)

	// Start the hub
	go hub.Run()

	// Create test server
	router := gin.New()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	server := httptest.NewServer(router)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/flows/stream"

	// Connect WebSocket client
	wsConn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	defer wsConn.Close()

	// Give time for client registration
	time.Sleep(50 * time.Millisecond)

	// Check hub stats to verify client was registered
	stats := hub.GetStats()
	assert.Equal(t, 1, stats.ConnectedClients)
	assert.Equal(t, int64(1), stats.TotalConnections)
}

// TestFlowStreamHandler_MessageBroadcast tests flow message broadcasting
func TestFlowStreamHandler_MessageBroadcast(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := ws.NewHub()
	handler := NewFlowStreamHandler(hub)

	// Start the hub
	go hub.Run()

	// Create test server
	router := gin.New()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	server := httptest.NewServer(router)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/flows/stream"

	// Connect WebSocket client
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer wsConn.Close()

	// Give time for client registration
	time.Sleep(50 * time.Millisecond)

	// Create a test flow
	testFlow := &flowpb.Flow{
		Id:        12345,
		AgentId:   "agent-1",
		SrcIp:     "192.168.1.10",
		DstIp:     "192.168.1.20",
		SrcPort:   12345,
		DstPort:   80,
		Protocol:  commonpb.Protocol(6), // TCP
		StartTime: time.Now().UnixNano(),
	}

	// Broadcast the flow
	hub.Broadcast(testFlow)

	// Set read deadline
	wsConn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Read the message from WebSocket
	_, message, err := wsConn.ReadMessage()
	require.NoError(t, err)

	// Parse the received flow
	var receivedFlow flowpb.Flow
	err = json.Unmarshal(message, &receivedFlow)
	require.NoError(t, err)

	// Verify the flow data
	assert.Equal(t, testFlow.Id, receivedFlow.Id)
	assert.Equal(t, testFlow.AgentId, receivedFlow.AgentId)
	assert.Equal(t, testFlow.SrcIp, receivedFlow.SrcIp)
	assert.Equal(t, testFlow.DstIp, receivedFlow.DstIp)
	assert.Equal(t, testFlow.SrcPort, receivedFlow.SrcPort)
	assert.Equal(t, testFlow.DstPort, receivedFlow.DstPort)
}

// TestFlowStreamHandler_FilterUpdate tests client filter updates
func TestFlowStreamHandler_FilterUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := ws.NewHub()
	handler := NewFlowStreamHandler(hub)

	// Start the hub
	go hub.Run()

	// Create test server
	router := gin.New()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	server := httptest.NewServer(router)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/flows/stream"

	// Connect WebSocket client
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer wsConn.Close()

	// Give time for client registration
	time.Sleep(50 * time.Millisecond)

	// Send filter update
	sourceIP := "192.168.1.10"
	filter := ws.FlowFilter{
		SourceIP: &sourceIP,
	}

	filterJSON, err := json.Marshal(filter)
	require.NoError(t, err)

	err = wsConn.WriteMessage(websocket.TextMessage, filterJSON)
	require.NoError(t, err)

	// Give time for filter processing
	time.Sleep(50 * time.Millisecond)

	// Create flows - one matching, one not matching
	matchingFlow := &flowpb.Flow{
		Id:        1001,
		AgentId:   "agent-1",
		SrcIp:     "192.168.1.10", // Matches filter
		DstIp:     "192.168.1.20",
		SrcPort:   12345,
		DstPort:   80,
		Protocol:  commonpb.Protocol(6),
		StartTime: time.Now().UnixNano(),
	}

	nonMatchingFlow := &flowpb.Flow{
		Id:        1002,
		AgentId:   "agent-1",
		SrcIp:     "192.168.1.30", // Does not match filter
		DstIp:     "192.168.1.20",
		SrcPort:   12346,
		DstPort:   80,
		Protocol:  commonpb.Protocol(6),
		StartTime: time.Now().UnixNano(),
	}

	// Broadcast both flows
	hub.Broadcast(nonMatchingFlow)
	hub.Broadcast(matchingFlow)

	// Set read deadline
	wsConn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Should only receive the matching flow
	_, message, err := wsConn.ReadMessage()
	require.NoError(t, err)

	var receivedFlow flowpb.Flow
	err = json.Unmarshal(message, &receivedFlow)
	require.NoError(t, err)

	// Verify we received the matching flow
	assert.Equal(t, uint64(1001), receivedFlow.Id)
	assert.Equal(t, "192.168.1.10", receivedFlow.SrcIp)

	// Set a short read deadline to verify no more messages
	wsConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err = wsConn.ReadMessage()
	// Should timeout because non-matching flow was filtered out
	assert.Error(t, err)
}

// TestFlowStreamHandler_MultipleClients tests multiple WebSocket clients
func TestFlowStreamHandler_MultipleClients(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := ws.NewHub()
	handler := NewFlowStreamHandler(hub)

	// Start the hub
	go hub.Run()

	// Create test server
	router := gin.New()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	server := httptest.NewServer(router)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/flows/stream"

	// Connect multiple WebSocket clients
	client1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer client1.Close()

	client2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer client2.Close()

	client3, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer client3.Close()

	// Give time for client registration
	time.Sleep(100 * time.Millisecond)

	// Check hub stats
	stats := hub.GetStats()
	assert.Equal(t, 3, stats.ConnectedClients)
	assert.Equal(t, int64(3), stats.TotalConnections)

	// Broadcast a flow
	testFlow := &flowpb.Flow{
		Id:        2001,
		AgentId:   "agent-1",
		SrcIp:     "192.168.1.10",
		DstIp:     "192.168.1.20",
		SrcPort:   12345,
		DstPort:   80,
		Protocol:  commonpb.Protocol(6),
		StartTime: time.Now().UnixNano(),
	}

	hub.Broadcast(testFlow)

	// All clients should receive the message
	clients := []*websocket.Conn{client1, client2, client3}
	for i, client := range clients {
		client.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, message, err := client.ReadMessage()
		require.NoError(t, err, "Client %d should receive message", i+1)

		var receivedFlow flowpb.Flow
		err = json.Unmarshal(message, &receivedFlow)
		require.NoError(t, err)
		assert.Equal(t, uint64(2001), receivedFlow.Id)
	}

	// Close one client
	client2.Close()
	time.Sleep(100 * time.Millisecond)

	// Check stats again
	stats = hub.GetStats()
	assert.Equal(t, 2, stats.ConnectedClients)
	assert.Equal(t, int64(1), stats.TotalDisconnects)
}

// TestFlowStreamHandler_InvalidUpgrade tests invalid WebSocket upgrade requests
func TestFlowStreamHandler_InvalidUpgrade(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	hub := ws.NewHub()
	handler := NewFlowStreamHandler(hub)

	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	// Make a regular HTTP request (not WebSocket upgrade)
	req, _ := http.NewRequest("GET", "/api/v1/flows/stream", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should fail with bad request (upgrade required)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
