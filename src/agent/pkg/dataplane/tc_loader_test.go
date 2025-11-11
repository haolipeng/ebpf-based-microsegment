// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"testing"
)

func TestNewTCLoader(t *testing.T) {
	tests := []struct {
		name      string
		mode      DataPlaneMode
		iface     string
		ifaceIdx  int
		expectErr bool
	}{
		{
			name:      "valid TCX mode",
			mode:      ModeTCX,
			iface:     "eth0",
			ifaceIdx:  1,
			expectErr: false,
		},
		{
			name:      "valid LegacyTC mode",
			mode:      ModeLegacyTC,
			iface:     "eth0",
			ifaceIdx:  1,
			expectErr: false,
		},
		{
			name:      "invalid mode - Native XDP",
			mode:      ModeNativeXDP,
			iface:     "eth0",
			ifaceIdx:  1,
			expectErr: true,
		},
		{
			name:      "invalid mode - Generic XDP",
			mode:      ModeGenericXDP,
			iface:     "eth0",
			ifaceIdx:  1,
			expectErr: true,
		},
		{
			name:      "invalid mode - Unknown",
			mode:      ModeUnknown,
			iface:     "eth0",
			ifaceIdx:  1,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader, err := NewTCLoader(tt.mode, tt.iface, tt.ifaceIdx)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				if loader != nil {
					t.Errorf("Expected nil loader but got %v", loader)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if loader == nil {
					t.Errorf("Expected loader but got nil")
				}
				if loader != nil {
					if loader.mode != tt.mode {
						t.Errorf("Expected mode %v but got %v", tt.mode, loader.mode)
					}
					if loader.iface != tt.iface {
						t.Errorf("Expected iface %s but got %s", tt.iface, loader.iface)
					}
					if loader.ifaceIdx != tt.ifaceIdx {
						t.Errorf("Expected ifaceIdx %d but got %d", tt.ifaceIdx, loader.ifaceIdx)
					}
				}
			}
		})
	}
}

func TestTCLoaderGetMode(t *testing.T) {
	tests := []struct {
		name string
		mode DataPlaneMode
	}{
		{
			name: "TCX mode",
			mode: ModeTCX,
		},
		{
			name: "LegacyTC mode",
			mode: ModeLegacyTC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader, err := NewTCLoader(tt.mode, "eth0", 1)
			if err != nil {
				t.Fatalf("Failed to create loader: %v", err)
			}

			if loader.GetMode() != tt.mode {
				t.Errorf("Expected mode %v but got %v", tt.mode, loader.GetMode())
			}
		})
	}
}

func TestTCLoaderGetMaps_NotLoaded(t *testing.T) {
	loader, err := NewTCLoader(ModeTCX, "eth0", 1)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	// GetMaps should fail when eBPF objects are not loaded
	maps, err := loader.GetMaps()
	if err == nil {
		t.Error("Expected error when getting maps without loading, but got nil")
	}
	if maps != nil {
		t.Errorf("Expected nil maps but got %v", maps)
	}
}

func TestIsFileExistsError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "file exists string",
			err:      &testError{msg: "file exists"},
			expected: true,
		},
		{
			name:     "other error",
			err:      &testError{msg: "some other error"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFileExistsError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v but got %v", tt.expected, result)
			}
		})
	}
}

// testError is a helper type for testing error messages
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// 注意: 测试 Load() 和 Unload() 方法需要实际的网卡和 root 权限
// 这些测试应该在集成测试中进行,而不是单元测试
// 这里只测试基本的创建和验证逻辑
