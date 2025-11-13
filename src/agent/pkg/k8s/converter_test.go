package k8s

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodToWorkload(t *testing.T) {
	tests := []struct {
		name        string
		pod         *corev1.Pod
		expectNil   bool
		expectError bool
		checkLabels map[string]string
	}{
		{
			name:        "nil pod should return error",
			pod:         nil,
			expectNil:   true,
			expectError: true,
		},
		{
			name: "pod without IP should return nil",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
					UID:       "12345",
				},
				Status: corev1.PodStatus{
					PodIP: "", // 没有 IP
				},
			},
			expectNil:   true,
			expectError: false,
		},
		{
			name: "pod with basic info should convert successfully",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
					UID:       "12345",
					Labels: map[string]string{
						"app": "my-app",
					},
				},
				Spec: corev1.PodSpec{
					NodeName: "node-1",
				},
				Status: corev1.PodStatus{
					PodIP: "10.0.1.100",
				},
			},
			expectNil:   false,
			expectError: false,
			checkLabels: map[string]string{
				"app":           "my-app",
				"k8s.namespace": "default",
				"k8s.pod.name":  "test-pod",
				"k8s.node.name": "node-1",
			},
		},
		{
			name: "pod with standard k8s labels should map correctly",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "web-pod",
					Namespace: "production",
					UID:       "67890",
					Labels: map[string]string{
						"app.kubernetes.io/name":      "nginx",
						"app.kubernetes.io/component": "frontend",
						"environment":                 "prod",
					},
				},
				Spec: corev1.PodSpec{
					NodeName: "node-2",
				},
				Status: corev1.PodStatus{
					PodIP: "10.0.2.50",
				},
			},
			expectNil:   false,
			expectError: false,
			checkLabels: map[string]string{
				"app":              "nginx",  // 从 app.kubernetes.io/name 映射
				"role":             "frontend", // 从 app.kubernetes.io/component 映射
				"env":              "prod",    // 从 environment 映射到 env
				"k8s.namespace":    "production",
				"k8s.pod.name":     "web-pod",
				"k8s.node.name":    "node-2",
			},
		},
		{
			name: "pod without node name should still work",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pending-pod",
					Namespace: "default",
					UID:       "11111",
				},
				Spec: corev1.PodSpec{
					NodeName: "", // 没有分配节点
				},
				Status: corev1.PodStatus{
					PodIP: "10.0.3.10",
				},
			},
			expectNil:   false,
			expectError: false,
			checkLabels: map[string]string{
				"k8s.namespace": "default",
				"k8s.pod.name":  "pending-pod",
				// k8s.node.name 不应该存在
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wl, err := PodToWorkload(tt.pod)

			// 检查错误
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// 检查是否返回 nil
			if tt.expectNil {
				if wl != nil {
					t.Errorf("Expected nil workload but got %+v", wl)
				}
				return
			}

			// 检查 workload 不为 nil
			if wl == nil {
				t.Fatal("Expected non-nil workload")
			}

			// 检查 Workload ID 格式
			expectedID := "k8s:" + tt.pod.Namespace + ":" + string(tt.pod.UID)
			if wl.ID != expectedID {
				t.Errorf("Expected workload ID %s but got %s", expectedID, wl.ID)
			}

			// 检查 Workload Name
			if wl.Name != tt.pod.Name {
				t.Errorf("Expected workload name %s but got %s", tt.pod.Name, wl.Name)
			}

			// 检查 IP
			if len(wl.IPs) != 1 {
				t.Errorf("Expected 1 IP but got %d", len(wl.IPs))
			} else if wl.IPs[0].String() != tt.pod.Status.PodIP {
				t.Errorf("Expected IP %s but got %s", tt.pod.Status.PodIP, wl.IPs[0].String())
			}

			// 检查标签
			if tt.checkLabels != nil {
				for key, expectedValue := range tt.checkLabels {
					actualValue, exists := wl.Labels[key]
					if !exists {
						t.Errorf("Expected label %s to exist", key)
					} else if actualValue != expectedValue {
						t.Errorf("Expected label %s=%s but got %s", key, expectedValue, actualValue)
					}
				}
			}
		})
	}
}

func TestMapPodLabels(t *testing.T) {
	tests := []struct {
		name           string
		podLabels      map[string]string
		expectedLabels map[string]string
	}{
		{
			name:           "nil labels should return empty map",
			podLabels:      nil,
			expectedLabels: map[string]string{},
		},
		{
			name:           "empty labels should return empty map",
			podLabels:      map[string]string{},
			expectedLabels: map[string]string{},
		},
		{
			name: "standard k8s labels should be mapped",
			podLabels: map[string]string{
				"app.kubernetes.io/name":      "nginx",
				"app.kubernetes.io/component": "web",
			},
			expectedLabels: map[string]string{
				"app":  "nginx",
				"role": "web",
			},
		},
		{
			name: "custom labels should be preserved",
			podLabels: map[string]string{
				"team": "backend",
				"tier": "database",
			},
			expectedLabels: map[string]string{
				"team": "backend",
				"tier": "database",
			},
		},
		{
			name: "mixed standard and custom labels",
			podLabels: map[string]string{
				"app.kubernetes.io/name": "redis",
				"tier":                   "cache",
				"version":                "6.2",
			},
			expectedLabels: map[string]string{
				"app":     "redis",
				"tier":    "cache",
				"version": "6.2",
			},
		},
		{
			name: "environment label mappings",
			podLabels: map[string]string{
				"environment": "production",
			},
			expectedLabels: map[string]string{
				"env": "production",
			},
		},
		{
			name: "env label should map to env",
			podLabels: map[string]string{
				"env": "staging",
			},
			expectedLabels: map[string]string{
				"env": "staging",
			},
		},
		{
			name: "topology zone should map to loc",
			podLabels: map[string]string{
				"topology.kubernetes.io/zone": "us-west-1a",
			},
			expectedLabels: map[string]string{
				"loc": "us-west-1a",
			},
		},
		{
			name: "topology region should map to region",
			podLabels: map[string]string{
				"topology.kubernetes.io/region": "us-west-1",
			},
			expectedLabels: map[string]string{
				"region": "us-west-1",
			},
		},
		{
			name: "version label mappings",
			podLabels: map[string]string{
				"app.kubernetes.io/version": "v1.2.3",
			},
			expectedLabels: map[string]string{
				"version": "v1.2.3",
			},
		},
		{
			name: "instance label mappings",
			podLabels: map[string]string{
				"app.kubernetes.io/instance": "my-release",
			},
			expectedLabels: map[string]string{
				"instance": "my-release",
			},
		},
		{
			name: "priority test - standard label over simplified",
			podLabels: map[string]string{
				"app.kubernetes.io/name": "nginx",
				"app":                    "apache", // 应该被忽略
			},
			expectedLabels: map[string]string{
				"app": "nginx", // 使用标准标签的值
			},
		},
		{
			name: "priority test - environment over env",
			podLabels: map[string]string{
				"environment": "prod",
				"env":         "dev", // 应该被忽略
			},
			expectedLabels: map[string]string{
				"env": "prod", // 使用 environment 的值
			},
		},
		{
			name: "comprehensive label mapping",
			podLabels: map[string]string{
				"app.kubernetes.io/name":        "myapp",
				"app.kubernetes.io/component":   "api",
				"app.kubernetes.io/version":     "2.0.0",
				"app.kubernetes.io/instance":    "prod-instance",
				"environment":                   "production",
				"topology.kubernetes.io/zone":   "us-east-1b",
				"topology.kubernetes.io/region": "us-east-1",
				"team":                          "platform",
				"owner":                         "john",
			},
			expectedLabels: map[string]string{
				"app":      "myapp",
				"role":     "api",
				"version":  "2.0.0",
				"instance": "prod-instance",
				"env":      "production",
				"loc":      "us-east-1b",
				"region":   "us-east-1",
				"team":     "platform",
				"owner":    "john",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: tt.podLabels,
				},
			}

			result := mapPodLabels(pod)

			// 检查结果数量
			if len(result) != len(tt.expectedLabels) {
				t.Errorf("Expected %d labels but got %d. Result: %+v", len(tt.expectedLabels), len(result), result)
			}

			// 检查每个标签
			for key, expectedValue := range tt.expectedLabels {
				actualValue, exists := result[key]
				if !exists {
					t.Errorf("Expected label %s to exist", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected label %s=%s but got %s", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestMapPodLabels_ExportedFunction(t *testing.T) {
	// 测试导出的 MapPodLabels 函数
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string]string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: map[string]string{},
		},
		{
			name: "standard labels",
			input: map[string]string{
				"app.kubernetes.io/name": "test-app",
				"environment":            "dev",
			},
			expected: map[string]string{
				"app": "test-app",
				"env": "dev",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapPodLabels(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d labels but got %d", len(tt.expected), len(result))
			}

			for key, expectedValue := range tt.expected {
				actualValue, exists := result[key]
				if !exists {
					t.Errorf("Expected label %s to exist", key)
				} else if actualValue != expectedValue {
					t.Errorf("Expected label %s=%s but got %s", key, expectedValue, actualValue)
				}
			}
		})
	}
}
