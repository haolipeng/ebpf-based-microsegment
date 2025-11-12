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

// TestTCLoaderStructure 验证 TCLoader 结构体是否正确初始化
func TestTCLoaderStructure(t *testing.T) {
	loader, err := NewTCLoader(ModeTCX, "eth0", 1)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	// 验证初始状态
	if loader.ingressLink != nil {
		t.Error("Expected ingressLink to be nil before Load()")
	}
	if loader.egressLink != nil {
		t.Error("Expected egressLink to be nil before Load()")
	}
	if loader.ingressFilter != nil {
		t.Error("Expected ingressFilter to be nil before Load()")
	}
	if loader.egressFilter != nil {
		t.Error("Expected egressFilter to be nil before Load()")
	}
	if loader.objs != nil {
		t.Error("Expected objs to be nil before Load()")
	}
	if loader.pinConfig == nil {
		t.Error("Expected pinConfig to be initialized")
	}
}

// TestTCLoaderModeValidation 验证模式验证逻辑
func TestTCLoaderModeValidation(t *testing.T) {
	invalidModes := []DataPlaneMode{
		ModeUnknown,
		ModeNativeXDP,
		ModeGenericXDP,
	}

	for _, mode := range invalidModes {
		t.Run(mode.String(), func(t *testing.T) {
			loader, err := NewTCLoader(mode, "eth0", 1)
			if err == nil {
				t.Errorf("Expected error for mode %v but got nil", mode)
			}
			if loader != nil {
				t.Errorf("Expected nil loader for invalid mode %v", mode)
			}
		})
	}

	validModes := []DataPlaneMode{
		ModeTCX,
		ModeLegacyTC,
	}

	for _, mode := range validModes {
		t.Run(mode.String(), func(t *testing.T) {
			loader, err := NewTCLoader(mode, "eth0", 1)
			if err != nil {
				t.Errorf("Unexpected error for valid mode %v: %v", mode, err)
			}
			if loader == nil {
				t.Errorf("Expected loader for valid mode %v", mode)
			}
		})
	}
}

// TestDetachMethodsIdempotency 验证 detach 方法的幂等性
func TestDetachMethodsIdempotency(t *testing.T) {
	loader, err := NewTCLoader(ModeTCX, "eth0", 1)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	// 测试 detachTCXIngress 幂等性 (多次调用不应报错)
	t.Run("detachTCXIngress idempotency", func(t *testing.T) {
		if err := loader.detachTCXIngress(); err != nil {
			t.Errorf("First detachTCXIngress failed: %v", err)
		}
		if err := loader.detachTCXIngress(); err != nil {
			t.Errorf("Second detachTCXIngress failed: %v", err)
		}
		if loader.ingressLink != nil {
			t.Error("Expected ingressLink to be nil after detach")
		}
	})

	// 测试 detachTCXEgress 幂等性
	t.Run("detachTCXEgress idempotency", func(t *testing.T) {
		if err := loader.detachTCXEgress(); err != nil {
			t.Errorf("First detachTCXEgress failed: %v", err)
		}
		if err := loader.detachTCXEgress(); err != nil {
			t.Errorf("Second detachTCXEgress failed: %v", err)
		}
		if loader.egressLink != nil {
			t.Error("Expected egressLink to be nil after detach")
		}
	})
}

// TestDetachLegacyTCIdempotency 验证 Legacy TC detach 方法的幂等性
func TestDetachLegacyTCIdempotency(t *testing.T) {
	loader, err := NewTCLoader(ModeLegacyTC, "eth0", 1)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	// 测试 detachLegacyTCIngress 幂等性
	t.Run("detachLegacyTCIngress idempotency", func(t *testing.T) {
		if err := loader.detachLegacyTCIngress(); err != nil {
			t.Errorf("First detachLegacyTCIngress failed: %v", err)
		}
		if err := loader.detachLegacyTCIngress(); err != nil {
			t.Errorf("Second detachLegacyTCIngress failed: %v", err)
		}
		if loader.ingressFilter != nil {
			t.Error("Expected ingressFilter to be nil after detach")
		}
	})

	// 测试 detachLegacyTCEgress 幂等性
	t.Run("detachLegacyTCEgress idempotency", func(t *testing.T) {
		if err := loader.detachLegacyTCEgress(); err != nil {
			t.Errorf("First detachLegacyTCEgress failed: %v", err)
		}
		if err := loader.detachLegacyTCEgress(); err != nil {
			t.Errorf("Second detachLegacyTCEgress failed: %v", err)
		}
		if loader.egressFilter != nil {
			t.Error("Expected egressFilter to be nil after detach")
		}
	})
}

// TestUnloadIdempotency 验证 Unload 方法的幂等性
func TestUnloadIdempotency(t *testing.T) {
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

			// Unload 应该可以多次调用而不报错
			if err := loader.Unload(); err != nil {
				t.Errorf("First Unload failed: %v", err)
			}
			if err := loader.Unload(); err != nil {
				t.Errorf("Second Unload failed: %v", err)
			}

			// 验证清理完成
			if loader.objs != nil {
				t.Error("Expected objs to be nil after Unload")
			}
			if loader.ingressLink != nil {
				t.Error("Expected ingressLink to be nil after Unload")
			}
			if loader.egressLink != nil {
				t.Error("Expected egressLink to be nil after Unload")
			}
			if loader.ingressFilter != nil {
				t.Error("Expected ingressFilter to be nil after Unload")
			}
			if loader.egressFilter != nil {
				t.Error("Expected egressFilter to be nil after Unload")
			}
		})
	}
}

// 注意: 测试 Load() 和实际 attach/detach 方法需要实际的网卡和 root 权限
// 这些测试应该在集成测试中进行,而不是单元测试
// 这里只测试基本的创建、验证和幂等性逻辑
