package handlers

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ebpf-microsegment/src/agent/pkg/api/models"
	"github.com/ebpf-microsegment/src/agent/pkg/workload"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupWorkloadTest(t *testing.T) (*WorkloadHandler, *workload.Manager, func()) {
	// Create temporary database
	dbPath := "/tmp/test_workload_api_" + t.Name() + ".db"

	storage, err := workload.NewSQLiteWorkloadStorage(dbPath)
	require.NoError(t, err)

	manager := workload.NewManager(storage)
	handler := NewWorkloadHandler(manager)

	cleanup := func() {
		storage.Close()
		os.Remove(dbPath)
	}

	return handler, manager, cleanup
}

func TestCreateWorkload(t *testing.T) {
	handler, _, cleanup := setupWorkloadTest(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/workloads", handler.CreateWorkload)

	tests := []struct {
		name           string
		requestBody    models.WorkloadRequest
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "valid workload",
			requestBody: models.WorkloadRequest{
				ID:   "test-workload-1",
				Name: "Test Workload 1",
				IPs:  []string{"10.0.1.10"},
				Labels: map[string]string{
					"app": "test",
					"env": "dev",
				},
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response models.WorkloadResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "test-workload-1", response.ID)
				assert.Equal(t, "Test Workload 1", response.Name)
				assert.Equal(t, []string{"10.0.1.10"}, response.IPs)
				assert.Equal(t, "test", response.Labels["app"])
			},
		},
		{
			name: "missing required fields",
			requestBody: models.WorkloadRequest{
				ID: "test-workload-2",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response models.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "validation_error", response.Error)
			},
		},
		{
			name: "invalid IP address",
			requestBody: models.WorkloadRequest{
				ID:   "test-workload-3",
				Name: "Test Workload 3",
				IPs:  []string{"invalid-ip"},
				Labels: map[string]string{
					"app": "test",
				},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response models.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response.Message, "Invalid IP address")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, "/workloads", bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestListWorkloads(t *testing.T) {
	handler, manager, cleanup := setupWorkloadTest(t)
	defer cleanup()

	// Create test workloads
	workloads := []*workload.Workload{
		{
			ID:     "workload-1",
			Name:   "Workload 1",
			HostID: "test-host",
			IPs:    []net.IP{net.ParseIP("10.0.1.10")},
			Labels: map[string]string{
				"app": "web",
			},
		},
		{
			ID:     "workload-2",
			Name:   "Workload 2",
			HostID: "test-host",
			IPs:    []net.IP{net.ParseIP("10.0.1.11")},
			Labels: map[string]string{
				"app": "api",
			},
		},
	}

	for _, w := range workloads {
		err := manager.CreateWorkload(w)
		require.NoError(t, err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/workloads", handler.ListWorkloads)

	req, err := http.NewRequest(http.MethodGet, "/workloads", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.WorkloadListResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 2, response.Count)
	assert.Len(t, response.Workloads, 2)
}

func TestGetWorkload(t *testing.T) {
	handler, manager, cleanup := setupWorkloadTest(t)
	defer cleanup()

	// Create test workload
	testWorkload := &workload.Workload{
		ID:     "test-get-1",
		Name:   "Test Get Workload",
		HostID: "test-host",
		IPs:    []net.IP{net.ParseIP("10.0.1.20")},
		Labels: map[string]string{
			"app": "test",
		},
	}
	err := manager.CreateWorkload(testWorkload)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/workloads/:id", handler.GetWorkload)

	tests := []struct {
		name           string
		workloadID     string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "existing workload",
			workloadID:     "test-get-1",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response models.WorkloadResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "test-get-1", response.ID)
				assert.Equal(t, "Test Get Workload", response.Name)
			},
		},
		{
			name:           "non-existent workload",
			workloadID:     "non-existent",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response models.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "not_found", response.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/workloads/"+tt.workloadID, nil)
			require.NoError(t, err)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestUpdateWorkload(t *testing.T) {
	handler, manager, cleanup := setupWorkloadTest(t)
	defer cleanup()

	// Create initial workload
	initialWorkload := &workload.Workload{
		ID:     "test-update-1",
		Name:   "Initial Name",
		HostID: "test-host",
		IPs:    []net.IP{net.ParseIP("10.0.1.30")},
		Labels: map[string]string{
			"version": "v1",
		},
	}
	err := manager.CreateWorkload(initialWorkload)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/workloads/:id", handler.UpdateWorkload)

	tests := []struct {
		name           string
		workloadID     string
		requestBody    models.WorkloadRequest
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:       "valid update",
			workloadID: "test-update-1",
			requestBody: models.WorkloadRequest{
				ID:   "test-update-1",
				Name: "Updated Name",
				IPs:  []string{"10.0.1.30", "10.0.1.31"},
				Labels: map[string]string{
					"version": "v2",
				},
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response models.WorkloadResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "Updated Name", response.Name)
				assert.Len(t, response.IPs, 2)
				assert.Equal(t, "v2", response.Labels["version"])
			},
		},
		{
			name:       "mismatched ID",
			workloadID: "test-update-1",
			requestBody: models.WorkloadRequest{
				ID:   "different-id",
				Name: "Name",
				IPs:  []string{"10.0.1.30"},
				Labels: map[string]string{
					"app": "test",
				},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response models.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response.Message, "does not match")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPut, "/workloads/"+tt.workloadID, bytes.NewBuffer(body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestDeleteWorkload(t *testing.T) {
	handler, manager, cleanup := setupWorkloadTest(t)
	defer cleanup()

	// Create test workload
	testWorkload := &workload.Workload{
		ID:     "test-delete-1",
		Name:   "Test Delete",
		HostID: "test-host",
		IPs:    []net.IP{net.ParseIP("10.0.1.40")},
		Labels: map[string]string{
			"app": "test",
		},
	}
	err := manager.CreateWorkload(testWorkload)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.DELETE("/workloads/:id", handler.DeleteWorkload)

	// Test successful deletion
	req, err := http.NewRequest(http.MethodDelete, "/workloads/test-delete-1", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["message"], "deleted successfully")

	// Verify workload was deleted
	_, err = manager.GetWorkload("test-delete-1")
	assert.Error(t, err)
}
