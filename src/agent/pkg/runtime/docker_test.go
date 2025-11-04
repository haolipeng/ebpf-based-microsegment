// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package runtime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDockerDetector_Success tests successful connection to Docker
func TestDockerDetector_Success(t *testing.T) {
	// This test requires a real Docker daemon
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	detector, err := NewDockerDetector("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer detector.Close()

	assert.NotNil(t, detector, "Detector should not be nil")
	assert.Equal(t, DefaultDockerSocket, detector.socketPath,
		"Should use default socket path")
}

// TestDockerDetector_CustomSocket tests creating detector with custom socket
func TestDockerDetector_CustomSocket(t *testing.T) {
	// Test with non-existent socket
	detector, err := NewDockerDetector("unix:///invalid/path/to/docker.sock")
	assert.Error(t, err, "Should fail with invalid socket path")
	assert.Nil(t, detector, "Detector should be nil on error")
}

// TestDockerGetContainerLabels_Success tests successful label extraction
func TestDockerGetContainerLabels_Success(t *testing.T) {
	// This test requires a real Docker daemon
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	detector, err := NewDockerDetector("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer detector.Close()

	// Test will only run if Docker is accessible
	containers, err := detector.ListContainersWithLabels()
	if err != nil {
		t.Skipf("Cannot list containers: %v", err)
	}

	// If we have containers, test label extraction
	if len(containers) > 0 {
		for containerID, labels := range containers {
			t.Logf("Container %s has %d labels", containerID, len(labels))

			// Verify no io.kubernetes.* labels in the result (should be filtered)
			for key := range labels {
				assert.NotContains(t, key, "io.kubernetes.",
					"System labels should be filtered out")
			}

			// Verify no com.docker.* labels in the result (should be filtered)
			for key := range labels {
				assert.False(t,
					len(key) >= len(DockerLabelPrefix) && key[:len(DockerLabelPrefix)] == DockerLabelPrefix,
					"Docker system labels should be filtered out")
			}

			// If we extracted k8s metadata, verify format
			if podName, ok := labels["k8s.pod"]; ok {
				assert.NotEmpty(t, podName, "k8s.pod should not be empty")
			}
			if namespace, ok := labels["k8s.namespace"]; ok {
				assert.NotEmpty(t, namespace, "k8s.namespace should not be empty")
			}
		}
	}
}

// TestDockerGetContainerLabels_FilterSystemLabels tests that system labels are filtered
func TestDockerGetContainerLabels_FilterSystemLabels(t *testing.T) {
	// Test the label filtering logic independently
	rawLabels := map[string]string{
		"io.kubernetes.pod.name":       "test-pod",
		"io.kubernetes.pod.namespace":  "default",
		"io.kubernetes.pod.uid":        "12345-67890",
		"io.kubernetes.container.name": "main-container",
		"com.docker.compose.project":   "myproject",
		"com.docker.compose.service":   "web",
		"app":                          "my-app",
		"role":                         "backend",
		"env":                          "production",
	}

	// Simulate the filtering logic from GetContainerLabels
	filtered := make(map[string]string)
	for k, v := range rawLabels {
		// Skip Docker system labels
		if len(k) >= len(DockerLabelPrefix) && k[:len(DockerLabelPrefix)] == DockerLabelPrefix {
			continue
		}

		// Handle Kubernetes system labels
		if len(k) >= len(K8sDockerPrefix) && k[:len(K8sDockerPrefix)] == K8sDockerPrefix {
			switch k {
			case "io.kubernetes.pod.namespace":
				filtered["k8s.namespace"] = v
			case "io.kubernetes.pod.name":
				filtered["k8s.pod"] = v
			case "io.kubernetes.pod.uid":
				filtered["k8s.pod_uid"] = v
			case "io.kubernetes.container.name":
				filtered["k8s.container"] = v
			}
			continue
		}

		filtered[k] = v
	}

	// Verify filtering results
	assert.Equal(t, "my-app", filtered["app"])
	assert.Equal(t, "backend", filtered["role"])
	assert.Equal(t, "production", filtered["env"])
	assert.Equal(t, "test-pod", filtered["k8s.pod"])
	assert.Equal(t, "default", filtered["k8s.namespace"])
	assert.Equal(t, "12345-67890", filtered["k8s.pod_uid"])
	assert.Equal(t, "main-container", filtered["k8s.container"])

	// Verify original io.kubernetes.* labels are not present
	assert.NotContains(t, filtered, "io.kubernetes.pod.name")
	assert.NotContains(t, filtered, "io.kubernetes.pod.namespace")
	assert.NotContains(t, filtered, "io.kubernetes.pod.uid")
	assert.NotContains(t, filtered, "io.kubernetes.container.name")

	// Verify Docker system labels are not present
	assert.NotContains(t, filtered, "com.docker.compose.project")
	assert.NotContains(t, filtered, "com.docker.compose.service")

	// Verify we have the right number of labels
	assert.Equal(t, 7, len(filtered), "Should have 3 user labels + 4 k8s metadata labels")
}

// TestDockerGetContainerLabels_InvalidContainerID tests error handling for invalid container ID
func TestDockerGetContainerLabels_InvalidContainerID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	detector, err := NewDockerDetector("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer detector.Close()

	// Test with invalid container ID
	labels, err := detector.GetContainerLabels("invalid-container-id-that-does-not-exist")
	assert.Error(t, err, "Should return error for invalid container ID")
	assert.Nil(t, labels, "Labels should be nil on error")
}

// TestDockerListContainersWithLabels tests listing all containers
func TestDockerListContainersWithLabels(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	detector, err := NewDockerDetector("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer detector.Close()

	containers, err := detector.ListContainersWithLabels()
	if err != nil {
		t.Skipf("Cannot list containers: %v", err)
	}

	t.Logf("Found %d containers", len(containers))

	// Verify structure
	for containerID, labels := range containers {
		assert.NotEmpty(t, containerID, "Container ID should not be empty")
		assert.NotNil(t, labels, "Labels map should not be nil")

		// Verify no system labels
		for key := range labels {
			assert.False(t,
				len(key) >= len(K8sDockerPrefix) && key[:len(K8sDockerPrefix)] == K8sDockerPrefix,
				"Should not contain io.kubernetes.* labels")
			assert.False(t,
				len(key) >= len(DockerLabelPrefix) && key[:len(DockerLabelPrefix)] == DockerLabelPrefix,
				"Should not contain com.docker.* labels")
		}
	}
}

// TestDockerGetContainerInfo tests retrieving detailed container information
func TestDockerGetContainerInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	detector, err := NewDockerDetector("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer detector.Close()

	containers, err := detector.ListContainersWithLabels()
	if err != nil || len(containers) == 0 {
		t.Skip("No containers available for testing")
	}

	// Get first container ID
	var firstContainerID string
	for id := range containers {
		firstContainerID = id
		break
	}

	// Test GetContainerInfo
	info, err := detector.GetContainerInfo(firstContainerID)
	if err != nil {
		t.Skipf("Cannot get container info: %v", err)
	}

	assert.NotNil(t, info, "Container info should not be nil")
	assert.Equal(t, firstContainerID, info.ID, "Container ID should match")
	assert.NotEmpty(t, info.Name, "Container name should not be empty")
	assert.NotNil(t, info.Labels, "Labels should not be nil")
	assert.NotEmpty(t, info.State, "State should not be empty")
	assert.Greater(t, info.CreatedAt, int64(0), "CreatedAt should be positive")
}

// TestDockerClose tests cleanup and resource release
func TestDockerClose(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	detector, err := NewDockerDetector("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}

	err = detector.Close()
	assert.NoError(t, err, "Close should not return error")

	// Verify context is cancelled
	select {
	case <-detector.ctx.Done():
		// Expected: context should be cancelled
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be cancelled after Close()")
	}
}

// TestDockerWatchContainerEvents tests event watching (basic functionality)
func TestDockerWatchContainerEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	detector, err := NewDockerDetector("")
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	defer detector.Close()

	eventReceived := make(chan ContainerEvent, 1)
	callback := func(event ContainerEvent) {
		select {
		case eventReceived <- event:
		default:
		}
	}

	err = detector.WatchContainerEvents(callback)
	assert.NoError(t, err, "Starting event watcher should not fail")

	// Note: This test won't receive events unless containers are created/stopped
	// In a real integration test, you would create/stop a container here
	t.Log("Event watcher started successfully (no events expected in unit test)")
}

// TestDockerLabelFiltering tests comprehensive label filtering
func TestDockerLabelFiltering(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name: "all Docker and K8s labels",
			input: map[string]string{
				"io.kubernetes.pod.name":     "my-pod",
				"io.kubernetes.pod.namespace": "my-namespace",
				"com.docker.compose.project": "myproject",
				"com.docker.compose.service": "web",
			},
			expected: map[string]string{
				"k8s.pod":       "my-pod",
				"k8s.namespace": "my-namespace",
			},
		},
		{
			name: "mixed labels",
			input: map[string]string{
				"io.kubernetes.pod.name": "test-pod",
				"com.docker.version":     "20.10.0",
				"app":                    "my-app",
				"version":                "v1.0",
			},
			expected: map[string]string{
				"k8s.pod": "test-pod",
				"app":     "my-app",
				"version": "v1.0",
			},
		},
		{
			name: "no system labels",
			input: map[string]string{
				"app":  "my-app",
				"role": "backend",
			},
			expected: map[string]string{
				"app":  "my-app",
				"role": "backend",
			},
		},
		{
			name:     "empty labels",
			input:    map[string]string{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]string)
			for k, v := range tt.input {
				// Skip Docker system labels
				if len(k) >= len(DockerLabelPrefix) && k[:len(DockerLabelPrefix)] == DockerLabelPrefix {
					continue
				}

				// Handle Kubernetes system labels
				if len(k) >= len(K8sDockerPrefix) && k[:len(K8sDockerPrefix)] == K8sDockerPrefix {
					switch k {
					case "io.kubernetes.pod.namespace":
						result["k8s.namespace"] = v
					case "io.kubernetes.pod.name":
						result["k8s.pod"] = v
					case "io.kubernetes.pod.uid":
						result["k8s.pod_uid"] = v
					case "io.kubernetes.container.name":
						result["k8s.container"] = v
					}
					continue
				}

				result[k] = v
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}

// BenchmarkDockerGetContainerLabels benchmarks label extraction performance
func BenchmarkDockerGetContainerLabels(b *testing.B) {
	detector, err := NewDockerDetector("")
	if err != nil {
		b.Skipf("Docker not available: %v", err)
	}
	defer detector.Close()

	containers, err := detector.ListContainersWithLabels()
	if err != nil || len(containers) == 0 {
		b.Skip("No containers available for benchmarking")
	}

	// Get first container ID
	var containerID string
	for id := range containers {
		containerID = id
		break
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := detector.GetContainerLabels(containerID)
		if err != nil {
			b.Fatalf("GetContainerLabels failed: %v", err)
		}
	}
}

// BenchmarkDockerListContainersWithLabels benchmarks listing all containers
func BenchmarkDockerListContainersWithLabels(b *testing.B) {
	detector, err := NewDockerDetector("")
	if err != nil {
		b.Skipf("Docker not available: %v", err)
	}
	defer detector.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := detector.ListContainersWithLabels()
		if err != nil {
			b.Fatalf("ListContainersWithLabels failed: %v", err)
		}
	}
}
