// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: network interface name, mode config (TC/XDP mode selection)
// output: eBPF program loader instance, eBPF maps (policy_map, session_map, stats_map)
// pos: data plane lifecycle manager - if file updated, must sync with this header comment and pkg/dataplane/CLAUDE.md
package dataplane

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// Loader 数据平面加载器接口
// 统一管理 TC 和 XDP 两种数据平面实现
type Loader interface {
	// Load 加载并附加 eBPF 程序
	Load() error

	// Unload 卸载 eBPF 程序
	Unload() error

	// GetMaps 获取 eBPF Map 引用
	GetMaps() (*DataPlaneMaps, error)

	// GetMode 获取当前运行模式
	GetMode() DataPlaneMode
}

// Manager 数据平面管理器
// 负责根据系统能力和用户配置选择和管理数据平面
type Manager struct {
	// 配置
	iface    string        // 网卡名称
	ifaceIdx int           // 网卡索引
	config   *ModeConfig   // 模式配置
	caps     *Capabilities // 系统能力

	// 当前加载的数据平面
	currentMode DataPlaneMode // 当前模式
	loader      Loader        // 当前加载器 (TCLoader 或 XDPLoader)
}

// NewManager 创建数据平面管理器
//
// 参数:
//   - ifaceName: 网卡名称
//   - config: 模式配置 (可选,传 nil 使用默认配置)
//
// 返回:
//   - *Manager: 管理器实例
//   - error: 错误信息
func NewManager(ifaceName string, config *ModeConfig) (*Manager, error) {
	// 1. 获取网卡索引
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("getting interface %s: %w", ifaceName, err)
	}
	ifaceIdx := link.Attrs().Index

	// 2. 检测系统能力
	caps, err := DetectCapabilities(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("detecting capabilities: %w", err)
	}
	caps.InterfaceIndex = ifaceIdx

	// 3. 使用默认配置 (如果未提供)
	if config == nil {
		config = &ModeConfig{
			ForceMode:       ModeUnknown, // 自动选择
			PreferXDP:       false,       // 默认使用 TC (更稳定)
			AllowGenericXDP: true,        // 允许 Generic XDP 回退
		}
	}

	return &Manager{
		iface:       ifaceName,
		ifaceIdx:    ifaceIdx,
		config:      config,
		caps:        caps,
		currentMode: ModeUnknown,
		loader:      nil,
	}, nil
}

// Load 加载数据平面
//
// 根据系统能力和配置自动选择最佳模式,创建并加载对应的 loader
//
// 返回:
//   - error: 错误信息
func (m *Manager) Load() error {
	// 1. 如果已经加载,先卸载
	if m.loader != nil {
		log.Warn("DataPlane already loaded, unloading first")
		if err := m.Unload(); err != nil {
			return fmt.Errorf("unloading existing dataplane: %w", err)
		}
	}

	// 2. 选择最佳模式
	mode := SelectBestMode(m.caps, m.config)
	if mode == ModeUnknown {
		return fmt.Errorf("no suitable dataplane mode available")
	}

	log.Infof("Loading dataplane in mode: %v on interface %s", mode, m.iface)

	// 3. 创建ebpf程序对应的 loader加载器
	var loader Loader
	var err error

	if IsTCMode(mode) {
		// TC 模式 (TCX 或 Legacy TC)
		loader, err = NewTCLoader(mode, m.iface, m.ifaceIdx)
		if err != nil {
			return fmt.Errorf("creating TC loader: %w", err)
		}
	} else if IsXDPMode(mode) {
		// XDP 模式 (Native 或 Generic)
		loader, err = NewXDPLoader(mode, m.iface, m.ifaceIdx)
		if err != nil {
			return fmt.Errorf("creating XDP loader: %w", err)
		}
	} else {
		return fmt.Errorf("unsupported mode: %v", mode)
	}

	// 4. 加载 eBPF 程序
	if err := loader.Load(); err != nil {
		return fmt.Errorf("loading %v dataplane: %w", mode, err)
	}

	// 5. 保存状态
	m.loader = loader
	m.currentMode = mode

	log.Infof("✓ DataPlane loaded successfully in %v mode", mode)
	return nil
}

// Unload 卸载数据平面
//
// 返回:
//   - error: 错误信息
func (m *Manager) Unload() error {
	if m.loader == nil {
		log.Debug("DataPlane not loaded, nothing to unload")
		return nil
	}

	log.Infof("Unloading dataplane (mode: %v)", m.currentMode)

	if err := m.loader.Unload(); err != nil {
		return fmt.Errorf("unloading dataplane: %w", err)
	}

	m.loader = nil
	m.currentMode = ModeUnknown

	log.Info("✓ DataPlane unloaded successfully")
	return nil
}

// GetMaps 获取 eBPF Map 引用
//
// 返回:
//   - *DataPlaneMaps: Map 集合
//   - error: 错误信息
func (m *Manager) GetMaps() (*DataPlaneMaps, error) {
	if m.loader == nil {
		return nil, fmt.Errorf("dataplane not loaded")
	}

	return m.loader.GetMaps()
}

// GetMode 获取当前运行模式
//
// 返回:
//   - DataPlaneMode: 当前模式
func (m *Manager) GetMode() DataPlaneMode {
	return m.currentMode
}

// GetCapabilities 获取系统能力
//
// 返回:
//   - *Capabilities: 系统能力
func (m *Manager) GetCapabilities() *Capabilities {
	return m.caps
}

// SwitchMode 切换数据平面模式
//
// 此方法会:
//  1. 卸载当前数据平面
//  2. 切换到新模式
//  3. 重新加载数据平面
//
// 注意:
//   - 切换过程中会短暂中断流量过滤
//   - 会话表不会迁移 (TC 和 XDP 各自维护独立会话表)
//   - 策略数据通过 Map Pinning 自动共享
//
// 参数:
//   - newMode: 目标模式
//
// 返回:
//   - error: 错误信息
func (m *Manager) SwitchMode(newMode DataPlaneMode) error {
	// 1. 验证新模式是否可用
	if !validateMode(newMode, m.caps) {
		return fmt.Errorf("mode %v not supported on this system", newMode)
	}

	// 2. 如果已经是目标模式,无需切换
	if m.currentMode == newMode {
		log.Infof("Already in mode %v, no switch needed", newMode)
		return nil
	}

	log.Infof("Switching dataplane mode: %v -> %v", m.currentMode, newMode)

	// 3. 卸载当前数据平面
	if err := m.Unload(); err != nil {
		return fmt.Errorf("unloading current dataplane: %w", err)
	}

	// 4. 强制使用新模式
	oldForceMode := m.config.ForceMode
	m.config.ForceMode = newMode

	// 5. 加载新数据平面
	err := m.Load()

	// 6. 恢复配置
	m.config.ForceMode = oldForceMode

	if err != nil {
		return fmt.Errorf("loading new dataplane: %w", err)
	}

	log.Infof("✓ DataPlane switched to %v mode successfully", newMode)
	return nil
}

// Reload 重新加载数据平面
//
// 保持相同模式,重新加载 eBPF 程序
// 用于更新 eBPF 程序或恢复错误状态
//
// 返回:
//   - error: 错误信息
func (m *Manager) Reload() error {
	currentMode := m.currentMode

	if currentMode == ModeUnknown {
		return fmt.Errorf("cannot reload: dataplane not loaded")
	}

	log.Infof("Reloading dataplane (mode: %v)", currentMode)

	// 卸载并重新加载
	if err := m.Unload(); err != nil {
		return fmt.Errorf("unloading for reload: %w", err)
	}

	// 强制使用相同模式
	oldForceMode := m.config.ForceMode
	m.config.ForceMode = currentMode

	err := m.Load()

	m.config.ForceMode = oldForceMode

	if err != nil {
		return fmt.Errorf("reloading dataplane: %w", err)
	}

	log.Infof("✓ DataPlane reloaded successfully in %v mode", currentMode)
	return nil
}

// IsLoaded 检查数据平面是否已加载
//
// 返回:
//   - bool: 是否已加载
func (m *Manager) IsLoaded() bool {
	return m.loader != nil && m.currentMode != ModeUnknown
}
