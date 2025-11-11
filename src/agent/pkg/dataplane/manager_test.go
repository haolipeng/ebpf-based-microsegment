// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"testing"
)

// TestNewManager 测试数据平面管理器创建
func TestNewManager(t *testing.T) {
	tests := []struct {
		name      string
		iface     string
		config    *ModeConfig
		expectErr bool
	}{
		{
			name:      "valid interface with nil config",
			iface:     "lo",
			config:    nil,
			expectErr: false,
		},
		{
			name:  "valid interface with custom config",
			iface: "lo",
			config: &ModeConfig{
				ForceMode:       ModeTCX,
				PreferXDP:       false,
				AllowGenericXDP: true,
			},
			expectErr: false,
		},
		{
			name:      "invalid interface",
			iface:     "nonexistent999",
			config:    nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := NewManager(tt.iface, tt.config)

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

			if mgr == nil {
				t.Error("Expected non-nil manager")
				return
			}

			// 验证字段
			if mgr.iface != tt.iface {
				t.Errorf("Expected iface %s, got %s", tt.iface, mgr.iface)
			}

			if mgr.caps == nil {
				t.Error("Expected non-nil capabilities")
			}

			if mgr.config == nil {
				t.Error("Expected non-nil config")
			}

			if mgr.currentMode != ModeUnknown {
				t.Errorf("Expected initial mode Unknown, got %v", mgr.currentMode)
			}

			if mgr.loader != nil {
				t.Error("Expected nil loader initially")
			}

			// 验证默认配置
			if tt.config == nil {
				if mgr.config.ForceMode != ModeUnknown {
					t.Errorf("Expected default ForceMode Unknown, got %v", mgr.config.ForceMode)
				}
				if mgr.config.PreferXDP {
					t.Error("Expected default PreferXDP false")
				}
				if !mgr.config.AllowGenericXDP {
					t.Error("Expected default AllowGenericXDP true")
				}
			}
		})
	}
}

// TestDataPlaneManagerGetMode 测试获取模式
func TestManagerGetMode(t *testing.T) {
	mgr, err := NewManager("lo", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	mode := mgr.GetMode()
	if mode != ModeUnknown {
		t.Errorf("Expected initial mode Unknown, got %v", mode)
	}
}

// TestDataPlaneManagerGetCapabilities 测试获取系统能力
func TestManagerGetCapabilities(t *testing.T) {
	mgr, err := NewManager("lo", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	caps := mgr.GetCapabilities()
	if caps == nil {
		t.Error("Expected non-nil capabilities")
		return
	}

	// 验证能力检测结果
	if caps.InterfaceName != "lo" {
		t.Errorf("Expected interface name 'lo', got %s", caps.InterfaceName)
	}

	if caps.KernelVersion == "" {
		t.Error("Expected non-empty kernel version")
	}

	if caps.KernelMajor == 0 {
		t.Error("Expected non-zero kernel major version")
	}
}

// TestDataPlaneManagerIsLoaded 测试加载状态检查
func TestManagerIsLoaded(t *testing.T) {
	mgr, err := NewManager("lo", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// 初始状态应该是未加载
	if mgr.IsLoaded() {
		t.Error("Expected IsLoaded to be false initially")
	}
}

// TestDataPlaneManagerGetMapsBeforeLoad 测试在加载前获取 Maps
func TestManagerGetMapsBeforeLoad(t *testing.T) {
	mgr, err := NewManager("lo", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	maps, err := mgr.GetMaps()
	if err == nil {
		t.Error("Expected error when getting maps before load")
	}

	if maps != nil {
		t.Errorf("Expected nil maps before load, got %v", maps)
	}
}

// TestDataPlaneManagerUnloadBeforeLoad 测试在加载前卸载
func TestManagerUnloadBeforeLoad(t *testing.T) {
	mgr, err := NewManager("lo", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Unload 在未加载时应该成功 (幂等性)
	err = mgr.Unload()
	if err != nil {
		t.Errorf("Unexpected error when unloading before load: %v", err)
	}
}

// TestDataPlaneManagerReloadBeforeLoad 测试在加载前重载
func TestManagerReloadBeforeLoad(t *testing.T) {
	mgr, err := NewManager("lo", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Reload 在未加载时应该失败
	err = mgr.Reload()
	if err == nil {
		t.Error("Expected error when reloading before load")
	}
}

// TestDataPlaneManagerSwitchModeBeforeLoad 测试在加载前切换模式
func TestManagerSwitchModeBeforeLoad(t *testing.T) {
	mgr, err := NewManager("lo", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// 获取系统支持的模式
	caps := mgr.GetCapabilities()
	var targetMode DataPlaneMode
	if caps.SupportsTCX {
		targetMode = ModeTCX
	} else if caps.SupportsLegacyTC {
		targetMode = ModeLegacyTC
	} else {
		t.Skip("No TC mode available for testing")
	}

	// SwitchMode 在未加载时应该尝试加载到目标模式
	// 但由于没有 root 权限,可能会失败,这是预期的
	err = mgr.SwitchMode(targetMode)
	// 不验证错误,因为这需要 root 权限
	t.Logf("SwitchMode result: %v", err)
}

// TestDataPlaneManagerSwitchModeToUnsupported 测试切换到不支持的模式
func TestManagerSwitchModeToUnsupported(t *testing.T) {
	mgr, err := NewManager("lo", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// 尝试切换到一个肯定不支持的模式
	// 假设我们找一个当前系统不支持的模式
	caps := mgr.GetCapabilities()

	var unsupportedMode DataPlaneMode
	if !caps.SupportsNativeXDP {
		unsupportedMode = ModeNativeXDP
	} else if !caps.SupportsGenericXDP {
		unsupportedMode = ModeGenericXDP
	} else if !caps.SupportsTCX {
		unsupportedMode = ModeTCX
	} else {
		t.Skip("Cannot find unsupported mode for testing")
	}

	err = mgr.SwitchMode(unsupportedMode)
	if err == nil {
		t.Error("Expected error when switching to unsupported mode")
	}
}

// TestModeConfigDefaults 测试默认配置
func TestModeConfigDefaults(t *testing.T) {
	mgr, err := NewManager("lo", nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	config := mgr.config
	if config.ForceMode != ModeUnknown {
		t.Errorf("Expected default ForceMode Unknown, got %v", config.ForceMode)
	}

	if config.PreferXDP {
		t.Error("Expected default PreferXDP false (TC is more stable)")
	}

	if !config.AllowGenericXDP {
		t.Error("Expected default AllowGenericXDP true")
	}
}

// TestModeConfigCustom 测试自定义配置
func TestModeConfigCustom(t *testing.T) {
	customConfig := &ModeConfig{
		ForceMode:       ModeTCX,
		PreferXDP:       true,
		AllowGenericXDP: false,
	}

	mgr, err := NewManager("lo", customConfig)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	config := mgr.config
	if config.ForceMode != ModeTCX {
		t.Errorf("Expected ForceMode TCX, got %v", config.ForceMode)
	}

	if !config.PreferXDP {
		t.Error("Expected PreferXDP true")
	}

	if config.AllowGenericXDP {
		t.Error("Expected AllowGenericXDP false")
	}
}

// 注意: 测试 Load()、Unload()、SwitchMode() 和 Reload() 方法需要实际的网卡和 root 权限
// 这些测试应该在集成测试中进行,而不是单元测试
// 这里只测试基本的创建、验证和错误处理逻辑
