package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ebpf-microsegment/src/agent/pkg/api/models"
	"github.com/ebpf-microsegment/src/agent/pkg/policy"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPolicyManager is a mock implementation of PolicyManager for testing
type MockPolicyManager struct {
	mock.Mock
}

func (m *MockPolicyManager) AddPolicy(p *policy.Policy) error {
	args := m.Called(p)
	return args.Error(0)
}

func (m *MockPolicyManager) DeletePolicy(p *policy.Policy) error {
	args := m.Called(p)
	return args.Error(0)
}

func (m *MockPolicyManager) ListPolicies() ([]policy.Policy, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]policy.Policy), args.Error(1)
}

// setupTestRouter creates a test router with the policy handler
func setupTestRouter(mockPM *MockPolicyManager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := NewPolicyHandler(mockPM)

	// Register routes
	api := router.Group("/api/v1")
	{
		api.POST("/policies", handler.CreatePolicy)
		api.GET("/policies", handler.ListPolicies)
		api.GET("/policies/:id", handler.GetPolicy)
		api.PUT("/policies/:id", handler.UpdatePolicy)
		api.DELETE("/policies/:id", handler.DeletePolicy)
	}

	return router
}

// TestCreatePolicy_Success tests successful policy creation
func TestCreatePolicy_Success(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Mock expectations
	mockPM.On("AddPolicy", mock.AnythingOfType("*policy.Policy")).Return(nil)

	// Prepare request
	reqBody := models.PolicyRequest{
		RuleID:   1,
		SrcIP:    "192.168.1.100",
		DstIP:    "10.0.0.1",
		SrcPort:  0,
		DstPort:  80,
		Protocol: "tcp",
		Action:   "allow",
		Priority: 100,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/policies", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.PolicyResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), response.RuleID)
	assert.Equal(t, "192.168.1.100", response.SrcIP)
	assert.Equal(t, "10.0.0.1", response.DstIP)
	assert.Equal(t, uint16(80), response.DstPort)
	assert.Equal(t, "tcp", response.Protocol)
	assert.Equal(t, "allow", response.Action)

	mockPM.AssertExpectations(t)
}

// TestCreatePolicy_InvalidJSON tests policy creation with invalid JSON
func TestCreatePolicy_InvalidJSON(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Prepare invalid JSON
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/policies", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Equal(t, "validation_error", response.Error)
}

// TestCreatePolicy_MissingRequiredFields tests validation
func TestCreatePolicy_MissingRequiredFields(t *testing.T) {
	testCases := []struct {
		name    string
		request map[string]interface{}
	}{
		{
			name: "missing rule_id",
			request: map[string]interface{}{
				"src_ip":   "192.168.1.100",
				"dst_ip":   "10.0.0.1",
				"protocol": "tcp",
				"action":   "allow",
			},
		},
		{
			name: "missing src_ip",
			request: map[string]interface{}{
				"rule_id":  1,
				"dst_ip":   "10.0.0.1",
				"protocol": "tcp",
				"action":   "allow",
			},
		},
		{
			name: "missing dst_ip",
			request: map[string]interface{}{
				"rule_id":  1,
				"src_ip":   "192.168.1.100",
				"protocol": "tcp",
				"action":   "allow",
			},
		},
		{
			name: "missing protocol",
			request: map[string]interface{}{
				"rule_id": 1,
				"src_ip":  "192.168.1.100",
				"dst_ip":  "10.0.0.1",
				"action":  "allow",
			},
		},
		{
			name: "missing action",
			request: map[string]interface{}{
				"rule_id":  1,
				"src_ip":   "192.168.1.100",
				"dst_ip":   "10.0.0.1",
				"protocol": "tcp",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			mockPM := new(MockPolicyManager)
			router := setupTestRouter(mockPM)

			// Prepare request
			jsonBody, _ := json.Marshal(tc.request)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/policies", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// Execute
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// TestCreatePolicy_InvalidProtocol tests invalid protocol value
func TestCreatePolicy_InvalidProtocol(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Prepare request with invalid protocol
	reqBody := map[string]interface{}{
		"rule_id":  1,
		"src_ip":   "192.168.1.100",
		"dst_ip":   "10.0.0.1",
		"protocol": "invalid",
		"action":   "allow",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/policies", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCreatePolicy_InvalidAction tests invalid action value
func TestCreatePolicy_InvalidAction(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Prepare request with invalid action
	reqBody := map[string]interface{}{
		"rule_id":  1,
		"src_ip":   "192.168.1.100",
		"dst_ip":   "10.0.0.1",
		"protocol": "tcp",
		"action":   "invalid",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/policies", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCreatePolicy_PolicyManagerError tests error from policy manager
func TestCreatePolicy_PolicyManagerError(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Mock expectations - return error
	mockPM.On("AddPolicy", mock.AnythingOfType("*policy.Policy")).Return(errors.New("failed to add policy"))

	// Prepare request
	reqBody := models.PolicyRequest{
		RuleID:   1,
		SrcIP:    "192.168.1.100",
		DstIP:    "10.0.0.1",
		Protocol: "tcp",
		Action:   "allow",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/policies", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "policy_error", response.Error)

	mockPM.AssertExpectations(t)
}

// TestListPolicies_Success tests successful policy listing
func TestListPolicies_Success(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Mock data
	policies := []policy.Policy{
		{
			RuleID:   1,
			SrcIP:    "192.168.1.100",
			DstIP:    "10.0.0.1",
			DstPort:  80,
			Protocol: "tcp",
			Action:   "allow",
			Priority: 100,
		},
		{
			RuleID:   2,
			SrcIP:    "192.168.1.200",
			DstIP:    "10.0.0.2",
			DstPort:  443,
			Protocol: "tcp",
			Action:   "deny",
			Priority: 200,
		},
	}

	// Mock expectations
	mockPM.On("ListPolicies").Return(policies, nil)

	// Prepare request
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/policies", nil)

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PolicyListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 2, response.Count)
	assert.Len(t, response.Policies, 2)
	assert.Equal(t, uint32(1), response.Policies[0].RuleID)
	assert.Equal(t, uint32(2), response.Policies[1].RuleID)

	mockPM.AssertExpectations(t)
}

// TestListPolicies_Empty tests listing with no policies
func TestListPolicies_Empty(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Mock expectations - return empty list
	mockPM.On("ListPolicies").Return([]policy.Policy{}, nil)

	// Prepare request
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/policies", nil)

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PolicyListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Count)
	assert.Len(t, response.Policies, 0)

	mockPM.AssertExpectations(t)
}

// TestListPolicies_Error tests error from policy manager
func TestListPolicies_Error(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Mock expectations - return error
	mockPM.On("ListPolicies").Return(nil, errors.New("failed to list policies"))

	// Prepare request
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/policies", nil)

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mockPM.AssertExpectations(t)
}

// TestGetPolicy_Success tests successful policy retrieval
func TestGetPolicy_Success(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Mock data
	policies := []policy.Policy{
		{
			RuleID:   1,
			SrcIP:    "192.168.1.100",
			DstIP:    "10.0.0.1",
			DstPort:  80,
			Protocol: "tcp",
			Action:   "allow",
			Priority: 100,
		},
	}

	// Mock expectations
	mockPM.On("ListPolicies").Return(policies, nil)

	// Prepare request
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/policies/1", nil)

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PolicyResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), response.RuleID)
	assert.Equal(t, "192.168.1.100", response.SrcIP)

	mockPM.AssertExpectations(t)
}

// TestGetPolicy_NotFound tests policy not found
func TestGetPolicy_NotFound(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Mock expectations - return empty list
	mockPM.On("ListPolicies").Return([]policy.Policy{}, nil)

	// Prepare request
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/policies/999", nil)

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "not_found", response.Error)

	mockPM.AssertExpectations(t)
}

// TestGetPolicy_InvalidID tests invalid rule ID
func TestGetPolicy_InvalidID(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Prepare request with invalid ID
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/policies/invalid", nil)

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "validation_error", response.Error)
}

// TestUpdatePolicy_Success tests successful policy update
func TestUpdatePolicy_Success(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Mock expectations
	mockPM.On("DeletePolicy", mock.AnythingOfType("*policy.Policy")).Return(nil)
	mockPM.On("AddPolicy", mock.AnythingOfType("*policy.Policy")).Return(nil)

	// Prepare request
	reqBody := models.PolicyRequest{
		RuleID:   1,
		SrcIP:    "192.168.1.100",
		DstIP:    "10.0.0.1",
		DstPort:  443,
		Protocol: "tcp",
		Action:   "deny",
		Priority: 200,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/policies/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PolicyResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), response.RuleID)
	assert.Equal(t, uint16(443), response.DstPort)
	assert.Equal(t, "deny", response.Action)

	mockPM.AssertExpectations(t)
}

// TestUpdatePolicy_RuleIDMismatch tests rule ID mismatch
func TestUpdatePolicy_RuleIDMismatch(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Prepare request with mismatched rule ID
	reqBody := models.PolicyRequest{
		RuleID:   2,
		SrcIP:    "192.168.1.100",
		DstIP:    "10.0.0.1",
		Protocol: "tcp",
		Action:   "allow",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/policies/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "does not match")
}

// TestUpdatePolicy_AddError tests error when adding updated policy
func TestUpdatePolicy_AddError(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Mock expectations
	mockPM.On("DeletePolicy", mock.AnythingOfType("*policy.Policy")).Return(nil)
	mockPM.On("AddPolicy", mock.AnythingOfType("*policy.Policy")).Return(errors.New("failed to add"))

	// Prepare request
	reqBody := models.PolicyRequest{
		RuleID:   1,
		SrcIP:    "192.168.1.100",
		DstIP:    "10.0.0.1",
		Protocol: "tcp",
		Action:   "allow",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/policies/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mockPM.AssertExpectations(t)
}

// TestDeletePolicy_Success tests successful policy deletion
func TestDeletePolicy_Success(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Mock data
	policies := []policy.Policy{
		{
			RuleID:   1,
			SrcIP:    "192.168.1.100",
			DstIP:    "10.0.0.1",
			Protocol: "tcp",
			Action:   "allow",
		},
	}

	// Mock expectations
	mockPM.On("ListPolicies").Return(policies, nil)
	mockPM.On("DeletePolicy", mock.AnythingOfType("*policy.Policy")).Return(nil)

	// Prepare request
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/policies/1", nil)

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "deleted successfully")

	mockPM.AssertExpectations(t)
}

// TestDeletePolicy_NotFound tests deleting non-existent policy
func TestDeletePolicy_NotFound(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Mock expectations - return empty list
	mockPM.On("ListPolicies").Return([]policy.Policy{}, nil)

	// Prepare request
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/policies/999", nil)

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "not_found", response.Error)

	mockPM.AssertExpectations(t)
}

// TestDeletePolicy_InvalidID tests deleting with invalid ID
func TestDeletePolicy_InvalidID(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Prepare request with invalid ID
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/policies/invalid", nil)

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestDeletePolicy_DeleteError tests error from policy manager during deletion
func TestDeletePolicy_DeleteError(t *testing.T) {
	// Setup
	mockPM := new(MockPolicyManager)
	router := setupTestRouter(mockPM)

	// Mock data
	policies := []policy.Policy{
		{
			RuleID:   1,
			SrcIP:    "192.168.1.100",
			DstIP:    "10.0.0.1",
			Protocol: "tcp",
			Action:   "allow",
		},
	}

	// Mock expectations
	mockPM.On("ListPolicies").Return(policies, nil)
	mockPM.On("DeletePolicy", mock.AnythingOfType("*policy.Policy")).Return(errors.New("failed to delete"))

	// Prepare request
	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/policies/1", nil)

	// Execute
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	mockPM.AssertExpectations(t)
}

// TestPolicyHandler_AllProtocols tests all valid protocol values
func TestPolicyHandler_AllProtocols(t *testing.T) {
	protocols := []string{"tcp", "udp", "icmp", "any"}

	for _, proto := range protocols {
		t.Run(fmt.Sprintf("protocol_%s", proto), func(t *testing.T) {
			// Setup
			mockPM := new(MockPolicyManager)
			router := setupTestRouter(mockPM)

			// Mock expectations
			mockPM.On("AddPolicy", mock.AnythingOfType("*policy.Policy")).Return(nil)

			// Prepare request
			reqBody := models.PolicyRequest{
				RuleID:   1,
				SrcIP:    "192.168.1.100",
				DstIP:    "10.0.0.1",
				Protocol: proto,
				Action:   "allow",
			}

			jsonBody, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/policies", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// Execute
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, http.StatusCreated, w.Code)

			mockPM.AssertExpectations(t)
		})
	}
}

// TestPolicyHandler_AllActions tests all valid action values
func TestPolicyHandler_AllActions(t *testing.T) {
	actions := []string{"allow", "deny", "log"}

	for _, action := range actions {
		t.Run(fmt.Sprintf("action_%s", action), func(t *testing.T) {
			// Setup
			mockPM := new(MockPolicyManager)
			router := setupTestRouter(mockPM)

			// Mock expectations
			mockPM.On("AddPolicy", mock.AnythingOfType("*policy.Policy")).Return(nil)

			// Prepare request
			reqBody := models.PolicyRequest{
				RuleID:   1,
				SrcIP:    "192.168.1.100",
				DstIP:    "10.0.0.1",
				Protocol: "tcp",
				Action:   action,
			}

			jsonBody, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/policies", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			// Execute
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, http.StatusCreated, w.Code)

			var response models.PolicyResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, action, response.Action)

			mockPM.AssertExpectations(t)
		})
	}
}
