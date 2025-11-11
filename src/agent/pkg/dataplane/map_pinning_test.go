// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"testing"
)

func TestDefaultMapPinConfig(t *testing.T) {
	config := DefaultMapPinConfig()

	if config.PinPath != pinBasePath {
		t.Errorf("Expected PinPath %s, got %s", pinBasePath, config.PinPath)
	}

	if config.UnpinOnClose {
		t.Error("Expected UnpinOnClose to be false by default")
	}
}

func TestGetPinnedMapPath(t *testing.T) {
	tests := []struct {
		name     string
		pinPath  string
		mapName  string
		expected string
	}{
		{
			name:     "standard path",
			pinPath:  "/sys/fs/bpf/microsegment",
			mapName:  "policy_map",
			expected: "/sys/fs/bpf/microsegment/policy_map",
		},
		{
			name:     "custom path",
			pinPath:  "/tmp/test",
			mapName:  "test_map",
			expected: "/tmp/test/test_map",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPinnedMapPath(tt.pinPath, tt.mapName)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestEnsureBPFFS(t *testing.T) {
	// 注意: 这个测试需要在有 BPF FS 的环境中运行
	// 在大多数现代 Linux 系统上,/sys/fs/bpf 应该存在

	err := EnsureBPFFS()
	if err != nil {
		t.Skipf("BPF filesystem not available: %v (this is expected in some environments)", err)
	}
}

func TestIsPinned(t *testing.T) {
	// 测试不存在的 map
	result := IsPinned("/nonexistent/path", "nonexistent_map")
	if result {
		t.Error("Expected IsPinned to return false for non-existent map")
	}
}

// 注意: 以下测试需要 root 权限和实际的 BPF 环境
// 这些测试在 CI 环境或没有权限的环境中会被跳过

func TestEnsurePinPath(t *testing.T) {
	// 使用临时路径进行测试
	// 注意: 这需要 /sys/fs/bpf 存在
	err := EnsureBPFFS()
	if err != nil {
		t.Skipf("BPF filesystem not available, skipping test: %v", err)
		return
	}

	// 测试创建子目录
	testPath := "/sys/fs/bpf/test_microsegment_temp"
	err = EnsurePinPath(testPath)
	if err != nil {
		t.Skipf("Cannot create pin path (may need root): %v", err)
		return
	}

	// 清理
	// 注意: 这里不强制清理,避免测试失败
	_ = CleanupPinnedMaps(testPath)
}

func TestListPinnedMaps(t *testing.T) {
	// 测试不存在的目录
	maps, err := ListPinnedMaps("/nonexistent/path")
	if err != nil {
		t.Errorf("Expected no error for non-existent directory, got: %v", err)
	}
	if maps != nil {
		t.Errorf("Expected nil for non-existent directory, got: %v", maps)
	}
}
