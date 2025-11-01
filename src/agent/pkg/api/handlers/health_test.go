// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ebpf-microsegment/src/agent/pkg/api/models"
	"github.com/ebpf-microsegment/src/agent/pkg/dataplane"
	"github.com/ebpf-microsegment/src/agent/pkg/policy"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// MockDataPlane is a mock implementation of DataPlane for testing
type MockDataPlane struct {
	statistics dataplane.Statistics
}

func NewMockDataPlane() *MockDataPlane {
	return &MockDataPlane{
		statistics: dataplane.Statistics{
			TotalPackets:   1000,
			AllowedPackets: 800,
			DeniedPackets:  200,
			NewSessions:    50,
			ClosedSessions: 30,
			ActiveSessions: 20,
			PolicyHits:     900,
			PolicyMisses:   100,
		},
	}
}

func (m *MockDataPlane) GetStatistics() dataplane.Statistics {
	return m.statistics
}

func (m *MockDataPlane) SetStatistics(stats dataplane.Statistics) {
	m.statistics = stats
}

// MockPolicyManagerForHealth is a simple mock for health testing
type MockPolicyManagerForHealth struct {
	policies []policy.Policy
	err      error
}

func NewMockPolicyManagerForHealth() *MockPolicyManagerForHealth {
	return &MockPolicyManagerForHealth{
		policies: []policy.Policy{
			{
				RuleID:   1,
				SrcIP:    "192.168.1.100",
				DstIP:    "10.0.0.1",
				Protocol: "tcp",
				Action:   "allow",
			},
			{
				RuleID:   2,
				SrcIP:    "192.168.1.0/24",
				DstIP:    "10.0.0.2",
				Protocol: "udp",
				Action:   "deny",
			},
		},
	}
}

func (m *MockPolicyManagerForHealth) AddPolicy(p *policy.Policy) error {
	return nil
}

func (m *MockPolicyManagerForHealth) DeletePolicy(p *policy.Policy) error {
	return nil
}

func (m *MockPolicyManagerForHealth) ListPolicies() ([]policy.Policy, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.policies, m.err
}

func (m *MockPolicyManagerForHealth) SetPolicies(policies []policy.Policy) {
	m.policies = policies
}

func (m *MockPolicyManagerForHealth) SetError(err error) {
	m.err = err
}

// setupHealthTestRouter creates a test router with health handler
func setupHealthTestRouter(dp *MockDataPlane, pm *MockPolicyManagerForHealth) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := NewHealthHandler(dp, pm)

	router.GET("/api/v1/health", handler.GetHealth)
	router.GET("/api/v1/status", handler.GetStatus)

	return router
}

// TestGetHealth_Success tests the basic health check endpoint
func TestGetHealth_Success(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlane()
	mockPM := NewMockPolicyManagerForHealth()
	router := setupHealthTestRouter(mockDP, mockPM)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response.Status)
	assert.Equal(t, "API server is healthy", response.Message)

	// Verify the response is valid JSON
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

// TestGetHealth_ResponseFormat tests the health response format
func TestGetHealth_ResponseFormat(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlane()
	mockPM := NewMockPolicyManagerForHealth()
	router := setupHealthTestRouter(mockDP, mockPM)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify required fields are present
	assert.NotEmpty(t, response.Status)
	assert.NotEmpty(t, response.Message)

	// Verify JSON structure
	var jsonMap map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &jsonMap)
	assert.NoError(t, err)
	assert.Contains(t, jsonMap, "status")
	assert.Contains(t, jsonMap, "message")
}

// TestGetHealth_ConsistentResponse tests that health check returns consistent results
func TestGetHealth_ConsistentResponse(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlane()
	mockPM := NewMockPolicyManagerForHealth()
	router := setupHealthTestRouter(mockDP, mockPM)

	// Execute multiple times
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)

		var response models.HealthResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "ok", response.Status)
	}
}

// TestGetStatus_Success tests successful status retrieval
func TestGetStatus_Success(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlane()
	mockPM := NewMockPolicyManagerForHealth()

	// Set start time to a known value for testing
	originalStartTime := startTime
	startTime = time.Now().Add(-1 * time.Hour) // Simulate 1 hour uptime
	defer func() { startTime = originalStartTime }()

	router := setupHealthTestRouter(mockDP, mockPM)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.StatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify overall status
	assert.Equal(t, "ok", response.Status)
	assert.Equal(t, "0.1.0", response.Version)
	assert.Equal(t, "lo", response.Interface)

	// Verify data plane status
	assert.Equal(t, "running", response.DataPlane.Status)
	assert.Equal(t, "Data plane is operational", response.DataPlane.Message)

	// Verify API status
	assert.Equal(t, "running", response.API.Status)
	assert.Equal(t, "API server is operational", response.API.Message)

	// Verify statistics
	assert.NotNil(t, response.Statistics)
	assert.Equal(t, uint64(1000), response.Statistics.TotalPackets)
	assert.Equal(t, uint64(800), response.Statistics.AllowedPackets)
	assert.Equal(t, uint64(200), response.Statistics.DeniedPackets)
	assert.Equal(t, uint64(50), response.Statistics.NewSessions)
	assert.Equal(t, uint64(30), response.Statistics.ClosedSessions)
	assert.Equal(t, uint64(20), response.Statistics.ActiveSessions)
	assert.Equal(t, uint64(900), response.Statistics.PolicyHits)
	assert.Equal(t, uint64(100), response.Statistics.PolicyMisses)

	// Verify policy count
	assert.Equal(t, 2, response.PolicyCount)

	// Verify uptime (should be around 3600 seconds ± a few seconds)
	assert.InDelta(t, 3600, response.Uptime, 5)
}

// TestGetStatus_IdleDataPlane tests status when data plane is idle
func TestGetStatus_IdleDataPlane(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlane()
	mockPM := NewMockPolicyManagerForHealth()

	// Set statistics to zero packets
	mockDP.SetStatistics(dataplane.Statistics{
		TotalPackets:   0,
		AllowedPackets: 0,
		DeniedPackets:  0,
		NewSessions:    0,
		ClosedSessions: 0,
		ActiveSessions: 0,
		PolicyHits:     0,
		PolicyMisses:   0,
	})

	router := setupHealthTestRouter(mockDP, mockPM)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.StatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify data plane status shows idle
	assert.Equal(t, "idle", response.DataPlane.Status)
	assert.Equal(t, "Data plane is idle (no packets processed)", response.DataPlane.Message)

	// Overall status should still be ok
	assert.Equal(t, "ok", response.Status)
}

// TestGetStatus_PolicyManagerError tests status when policy manager has errors
func TestGetStatus_PolicyManagerError(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlane()
	mockPM := NewMockPolicyManagerForHealth()

	// Simulate policy manager error
	mockPM.SetError(fmt.Errorf("policy map error"))

	router := setupHealthTestRouter(mockDP, mockPM)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.StatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Overall status should be degraded
	assert.Equal(t, "degraded", response.Status)

	// Policy count should be 0 on error
	assert.Equal(t, 0, response.PolicyCount)
}

// TestGetStatus_NoPolicies tests status with no policies configured
func TestGetStatus_NoPolicies(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlane()
	mockPM := NewMockPolicyManagerForHealth()

	// Set no policies
	mockPM.SetPolicies([]policy.Policy{})

	router := setupHealthTestRouter(mockDP, mockPM)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.StatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Policy count should be 0
	assert.Equal(t, 0, response.PolicyCount)

	// Overall status should still be ok
	assert.Equal(t, "ok", response.Status)
}

// TestGetStatus_HighTraffic tests status with high traffic statistics
func TestGetStatus_HighTraffic(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlane()
	mockPM := NewMockPolicyManagerForHealth()

	// Set high traffic statistics
	mockDP.SetStatistics(dataplane.Statistics{
		TotalPackets:   1000000,
		AllowedPackets: 950000,
		DeniedPackets:  50000,
		NewSessions:    10000,
		ClosedSessions: 8000,
		ActiveSessions: 2000,
		PolicyHits:     990000,
		PolicyMisses:   10000,
	})

	router := setupHealthTestRouter(mockDP, mockPM)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.StatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify high traffic statistics
	assert.Equal(t, uint64(1000000), response.Statistics.TotalPackets)
	assert.Equal(t, uint64(950000), response.Statistics.AllowedPackets)
	assert.Equal(t, uint64(50000), response.Statistics.DeniedPackets)

	// Data plane should be running
	assert.Equal(t, "running", response.DataPlane.Status)
}

// TestGetStatus_ResponseStructure tests the complete status response structure
func TestGetStatus_ResponseStructure(t *testing.T) {
	// Setup
	mockDP := NewMockDataPlane()
	mockPM := NewMockPolicyManagerForHealth()

	router := setupHealthTestRouter(mockDP, mockPM)

	// Execute
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify JSON structure
	var jsonMap map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &jsonMap)
	assert.NoError(t, err)

	// Verify all required top-level fields
	assert.Contains(t, jsonMap, "status")
	assert.Contains(t, jsonMap, "version")
	assert.Contains(t, jsonMap, "interface")
	assert.Contains(t, jsonMap, "data_plane")
	assert.Contains(t, jsonMap, "api")
	assert.Contains(t, jsonMap, "statistics")
	assert.Contains(t, jsonMap, "policy_count")
	assert.Contains(t, jsonMap, "uptime_seconds")

	// Verify nested structures
	dataPlane := jsonMap["data_plane"].(map[string]interface{})
	assert.Contains(t, dataPlane, "status")
	assert.Contains(t, dataPlane, "message")

	api := jsonMap["api"].(map[string]interface{})
	assert.Contains(t, api, "status")
	assert.Contains(t, api, "message")

	statistics := jsonMap["statistics"].(map[string]interface{})
	assert.Contains(t, statistics, "total_packets")
	assert.Contains(t, statistics, "allowed_packets")
	assert.Contains(t, statistics, "denied_packets")
	assert.Contains(t, statistics, "new_sessions")
	assert.Contains(t, statistics, "closed_sessions")
	assert.Contains(t, statistics, "active_sessions")
	assert.Contains(t, statistics, "policy_hits")
	assert.Contains(t, statistics, "policy_misses")
}
