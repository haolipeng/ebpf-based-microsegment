package handlers

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/api/models"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/groups"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/workload"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGroupTest(t *testing.T) (*GroupHandler, *groups.GroupManager, *workload.Manager, func()) {
	// Create temporary databases
	workloadDBPath := "/tmp/test_group_api_workload_" + t.Name() + ".db"
	groupDBPath := "/tmp/test_group_api_group_" + t.Name() + ".db"

	workloadStorage, err := workload.NewSQLiteWorkloadStorage(workloadDBPath)
	require.NoError(t, err)

	groupStorage, err := groups.NewSQLiteGroupStorage(groupDBPath)
	require.NoError(t, err)

	workloadMgr := workload.NewManager(workloadStorage)
	groupMgr := groups.NewGroupManager(groupStorage, workloadMgr)
	handler := NewGroupHandler(groupMgr)

	cleanup := func() {
		workloadStorage.Close()
		groupStorage.Close()
		os.Remove(workloadDBPath)
		os.Remove(groupDBPath)
	}

	return handler, groupMgr, workloadMgr, cleanup
}

func TestCreateGroup(t *testing.T) {
	handler, _, workloadMgr, cleanup := setupGroupTest(t)
	defer cleanup()

	// Create some workloads for testing
	w1 := &workload.Workload{
		ID:   "web-1",
		HostID: "test-host",
		Name: "Web 1",
		IPs:  []net.IP{net.ParseIP("10.0.1.10")},
		Labels: map[string]string{
			"tier": "frontend",
			"app":  "web",
		},
	}
	err := workloadMgr.CreateWorkload(w1)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/groups", handler.CreateGroup)

	tests := []struct {
		name           string
		requestBody    models.GroupRequest
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "valid group",
			requestBody: models.GroupRequest{
				Name: "frontend-group",
				MatchLabels: map[string]string{
					"tier": "frontend",
				},
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response models.GroupResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "frontend-group", response.Name)
				assert.Equal(t, "frontend", response.MatchLabels["tier"])
				assert.Equal(t, 1, response.MemberCount)
			},
		},
		{
			name: "missing name",
			requestBody: models.GroupRequest{
				MatchLabels: map[string]string{
					"tier": "backend",
				},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response models.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "validation_error", response.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, "/groups", bytes.NewBuffer(body))
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

func TestListGroups(t *testing.T) {
	handler, groupMgr, _, cleanup := setupGroupTest(t)
	defer cleanup()

	// Create test groups
	group1 := groups.NewGroup("group-1")
	group1.AddSelector(groups.NewEqualSelector("tier", "frontend"))
	err := groupMgr.CreateGroup(group1)
	require.NoError(t, err)

	group2 := groups.NewGroup("group-2")
	group2.AddSelector(groups.NewEqualSelector("tier", "backend"))
	err = groupMgr.CreateGroup(group2)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/groups", handler.ListGroups)

	req, err := http.NewRequest(http.MethodGet, "/groups", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.GroupListResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 2, response.Count)
	assert.Len(t, response.Groups, 2)
}

func TestGetGroup(t *testing.T) {
	handler, groupMgr, workloadMgr, cleanup := setupGroupTest(t)
	defer cleanup()

	// Create workload
	w1 := &workload.Workload{
		ID:   "web-1",
		HostID: "test-host",
		Name: "Web 1",
		IPs:  []net.IP{net.ParseIP("10.0.1.10")},
		Labels: map[string]string{
			"app": "web",
		},
	}
	err := workloadMgr.CreateWorkload(w1)
	require.NoError(t, err)

	// Create group
	group := groups.NewGroup("test-group")
	group.AddSelector(groups.NewEqualSelector("app", "web"))
	err = groupMgr.CreateGroup(group)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/groups/:name", handler.GetGroup)

	tests := []struct {
		name           string
		groupName      string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "existing group",
			groupName:      "test-group",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response models.GroupResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "test-group", response.Name)
				assert.Equal(t, 1, response.MemberCount)
			},
		},
		{
			name:           "non-existent group",
			groupName:      "non-existent",
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
			req, err := http.NewRequest(http.MethodGet, "/groups/"+tt.groupName, nil)
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

func TestGetGroupMembers(t *testing.T) {
	handler, groupMgr, workloadMgr, cleanup := setupGroupTest(t)
	defer cleanup()

	// Create workloads
	w1 := &workload.Workload{
		ID:   "web-1",
		HostID: "test-host",
		Name: "Web 1",
		IPs:  []net.IP{net.ParseIP("10.0.1.10")},
		Labels: map[string]string{
			"app": "web",
		},
	}
	w2 := &workload.Workload{
		ID:   "web-2",
		HostID: "test-host",
		Name: "Web 2",
		IPs:  []net.IP{net.ParseIP("10.0.1.11")},
		Labels: map[string]string{
			"app": "web",
		},
	}
	err := workloadMgr.CreateWorkload(w1)
	require.NoError(t, err)
	err = workloadMgr.CreateWorkload(w2)
	require.NoError(t, err)

	// Create group
	group := groups.NewGroup("web-group")
	group.AddSelector(groups.NewEqualSelector("app", "web"))
	err = groupMgr.CreateGroup(group)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/groups/:name/members", handler.GetGroupMembers)

	req, err := http.NewRequest(http.MethodGet, "/groups/web-group/members", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.GroupMembersResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "web-group", response.GroupName)
	assert.Equal(t, 2, response.Count)
	assert.Len(t, response.Members, 2)
}

func TestUpdateGroup(t *testing.T) {
	handler, groupMgr, workloadMgr, cleanup := setupGroupTest(t)
	defer cleanup()

	// Create workloads
	w1 := &workload.Workload{
		ID:   "web-1",
		HostID: "test-host",
		Name: "Web 1",
		IPs:  []net.IP{net.ParseIP("10.0.1.10")},
		Labels: map[string]string{
			"app": "web",
			"env": "prod",
		},
	}
	w2 := &workload.Workload{
		ID:   "web-2",
		HostID: "test-host",
		Name: "Web 2",
		IPs:  []net.IP{net.ParseIP("10.0.1.11")},
		Labels: map[string]string{
			"app": "web",
			"env": "dev",
		},
	}
	err := workloadMgr.CreateWorkload(w1)
	require.NoError(t, err)
	err = workloadMgr.CreateWorkload(w2)
	require.NoError(t, err)

	// Create initial group
	group := groups.NewGroup("web-group")
	group.AddSelector(groups.NewEqualSelector("app", "web"))
	err = groupMgr.CreateGroup(group)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/groups/:name", handler.UpdateGroup)

	// Update to narrow selector
	requestBody := models.GroupRequest{
		Name: "web-group",
		MatchLabels: map[string]string{
			"app": "web",
			"env": "prod",
		},
	}
	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, "/groups/web-group", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.GroupResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, 1, response.MemberCount) // Only prod workload
}

func TestDeleteGroup(t *testing.T) {
	handler, groupMgr, _, cleanup := setupGroupTest(t)
	defer cleanup()

	// Create test group
	group := groups.NewGroup("test-delete-group")
	group.AddSelector(groups.NewEqualSelector("app", "test"))
	err := groupMgr.CreateGroup(group)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.DELETE("/groups/:name", handler.DeleteGroup)

	req, err := http.NewRequest(http.MethodDelete, "/groups/test-delete-group", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["message"], "deleted successfully")

	// Verify group was deleted
	_, err = groupMgr.GetGroup("test-delete-group")
	assert.Error(t, err)
}
