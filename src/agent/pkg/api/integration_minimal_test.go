// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// Package api provides minimal integration tests demonstrating the test framework.
// Full integration tests requiring actual eBPF maps are better suited for end-to-end testing.
package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ebpf-microsegment/src/agent/pkg/api/handlers"
	"github.com/ebpf-microsegment/src/agent/pkg/api/models"
	"github.com/ebpf-microsegment/src/agent/pkg/dataplane"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test environment for API integration tests
type MinimalTestEnv struct {
	Router *gin.Engine
	MockDP *MockDataPlaneForAPI
}

// MockDataPlaneForAPI provides a minimal mock for API testing
type MockDataPlaneForAPI struct {
	stats dataplane.Statistics
}

func (m *MockDataPlaneForAPI) GetStatistics() dataplane.Statistics {
	return m.stats
}

func (m *MockDataPlaneForAPI) SetStatistics(stats dataplane.Statistics) {
	m.stats = stats
}

// NewMinimalTestEnv creates a test environment for API-level integration tests
func NewMinimalTestEnv(t *testing.T) *MinimalTestEnv {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	mockDP := &MockDataPlaneForAPI{
		stats: dataplane.Statistics{},
	}

	// Create handlers with mock
	healthHandler := handlers.NewHealthHandler(mockDP, nil)
	statsHandler := handlers.NewStatisticsHandler(mockDP)

	// Setup routes
	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", healthHandler.GetHealth)
		v1.GET("/stats", statsHandler.GetAllStats)
		v1.GET("/stats/packets", statsHandler.GetPacketStats)
		v1.GET("/stats/sessions", statsHandler.GetSessionStats)
		v1.GET("/stats/policies", statsHandler.GetPolicyStats)
	}

	return &MinimalTestEnv{
		Router: router,
		MockDP: mockDP,
	}
}

// TestIntegration_API_Health tests the health endpoint integration
func TestIntegration_API_Health(t *testing.T) {
	env := NewMinimalTestEnv(t)

	// Test health check
	w := performRequest(env.Router, "GET", "/api/v1/health", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "ok", response.Status)
}

// TestIntegration_API_Statistics tests statistics endpoint integration
func TestIntegration_API_Statistics(t *testing.T) {
	env := NewMinimalTestEnv(t)

	// Set test statistics
	env.MockDP.SetStatistics(dataplane.Statistics{
		TotalPackets:   1000,
		AllowedPackets: 800,
		DeniedPackets:  200,
		PolicyHits:     950,
		PolicyMisses:   50,
	})

	// Test all stats endpoint
	w := performRequest(env.Router, "GET", "/api/v1/stats", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var stats models.StatisticsResponse
	err := json.Unmarshal(w.Body.Bytes(), &stats)
	require.NoError(t, err)
	assert.Equal(t, uint64(1000), stats.TotalPackets)
	assert.Equal(t, uint64(800), stats.AllowedPackets)
	assert.Equal(t, uint64(200), stats.DeniedPackets)
}

// TestIntegration_API_PacketStats tests packet statistics with rate calculations
func TestIntegration_API_PacketStats(t *testing.T) {
	env := NewMinimalTestEnv(t)

	// Set statistics for rate calculation
	env.MockDP.SetStatistics(dataplane.Statistics{
		TotalPackets:   2000,
		AllowedPackets: 1600, // 80%
		DeniedPackets:  400,  // 20%
	})

	w := performRequest(env.Router, "GET", "/api/v1/stats/packets", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var packetStats models.PacketStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &packetStats)
	require.NoError(t, err)

	assert.Equal(t, uint64(2000), packetStats.TotalPackets)
	assert.InDelta(t, 80.0, packetStats.AllowRate, 0.01)
	assert.InDelta(t, 20.0, packetStats.DenyRate, 0.01)
}

// TestIntegration_API_PolicyStats tests policy statistics with hit rate
func TestIntegration_API_PolicyStats(t *testing.T) {
	env := NewMinimalTestEnv(t)

	// Set policy statistics (95% hit rate)
	env.MockDP.SetStatistics(dataplane.Statistics{
		PolicyHits:   950,
		PolicyMisses: 50,
	})

	w := performRequest(env.Router, "GET", "/api/v1/stats/policies", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var policyStats models.PolicyStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &policyStats)
	require.NoError(t, err)

	assert.Equal(t, uint64(950), policyStats.PolicyHits)
	assert.Equal(t, uint64(50), policyStats.PolicyMisses)
	assert.InDelta(t, 95.0, policyStats.HitRate, 0.01)
}

// TestIntegration_API_ZeroStatistics tests handling of zero values
func TestIntegration_API_ZeroStatistics(t *testing.T) {
	env := NewMinimalTestEnv(t)

	// Don't set statistics (all zeros)
	w := performRequest(env.Router, "GET", "/api/v1/stats/packets", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var packetStats models.PacketStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &packetStats)
	require.NoError(t, err)

	// Rates should be 0 when total is 0
	assert.Equal(t, float64(0), packetStats.AllowRate)
	assert.Equal(t, float64(0), packetStats.DenyRate)
}

// Helper function to perform HTTP requests
func performRequest(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if body != nil {
		jsonData, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonData)
	}

	req, _ := http.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
