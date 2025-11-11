// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"testing"
)

// TestNewXDPLoader 测试 XDP 加载器的创建
func TestNewXDPLoader(t *testing.T) {
	tests := []struct {
		name      string
		mode      DataPlaneMode
		iface     string
		ifaceIdx  int
		expectErr bool
	}{
		{
			name:      "valid Native XDP mode",
			mode:      ModeNativeXDP,
			iface:     "eth0",
			ifaceIdx:  1,
			expectErr: false,
		},
		{
			name:      "valid Generic XDP mode",
			mode:      ModeGenericXDP,
			iface:     "eth0",
			ifaceIdx:  1,
			expectErr: false,
		},
		{
			name:      "invalid mode (TCX)",
			mode:      ModeTCX,
			iface:     "eth0",
			ifaceIdx:  1,
			expectErr: true,
		},
		{
			name:      "invalid mode (LegacyTC)",
			mode:      ModeLegacyTC,
			iface:     "eth0",
			ifaceIdx:  1,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader, err := NewXDPLoader(tt.mode, tt.iface, tt.ifaceIdx)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if loader == nil {
				t.Error("Expected non-nil loader")
				return
			}

			// 验证字段
			if loader.mode != tt.mode {
				t.Errorf("Expected mode %v, got %v", tt.mode, loader.mode)
			}

			if loader.iface != tt.iface {
				t.Errorf("Expected iface %s, got %s", tt.iface, loader.iface)
			}

			if loader.ifaceIdx != tt.ifaceIdx {
				t.Errorf("Expected ifaceIdx %d, got %d", tt.ifaceIdx, loader.ifaceIdx)
			}

			if loader.pinConfig == nil {
				t.Error("Expected non-nil pinConfig")
			}

			if loader.pinConfig.PinPath != pinBasePath {
				t.Errorf("Expected pinConfig.PinPath %s, got %s", pinBasePath, loader.pinConfig.PinPath)
			}
		})
	}
}

// TestXDPLoaderGetMode 测试 GetMode 方法
func TestXDPLoaderGetMode(t *testing.T) {
	tests := []struct {
		name         string
		mode         DataPlaneMode
		expectedMode DataPlaneMode
	}{
		{
			name:         "Native XDP mode",
			mode:         ModeNativeXDP,
			expectedMode: ModeNativeXDP,
		},
		{
			name:         "Generic XDP mode",
			mode:         ModeGenericXDP,
			expectedMode: ModeGenericXDP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader, err := NewXDPLoader(tt.mode, "eth0", 1)
			if err != nil {
				t.Fatalf("Failed to create loader: %v", err)
			}

			mode := loader.GetMode()
			if mode != tt.expectedMode {
				t.Errorf("Expected mode %v, got %v", tt.expectedMode, mode)
			}
		})
	}
}

// TestXDPLoaderGetMapsBeforeLoad 测试在加载前调用 GetMaps 的行为
func TestXDPLoaderGetMapsBeforeLoad(t *testing.T) {
	loader, err := NewXDPLoader(ModeNativeXDP, "eth0", 1)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	maps, err := loader.GetMaps()
	if err == nil {
		t.Error("Expected error when getting maps before load, but got none")
	}

	if maps != nil {
		t.Errorf("Expected nil maps before load, got: %v", maps)
	}
}

// TestXDPLoaderUnloadBeforeLoad 测试在加载前调用 Unload 的行为
func TestXDPLoaderUnloadBeforeLoad(t *testing.T) {
	loader, err := NewXDPLoader(ModeNativeXDP, "eth0", 1)
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	// Unload 在未加载时不应报错 (幂等性)
	err = loader.Unload()
	if err != nil {
		t.Errorf("Unexpected error when unloading before load: %v", err)
	}
}

// 注意: 测试 Load() 和 Unload() 方法需要实际的网卡和 root 权限
// 这些测试应该在集成测试中进行,而不是单元测试
// 这里只测试基本的创建和验证逻辑
