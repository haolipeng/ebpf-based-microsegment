// input: kernel version, eBPF feature probes, network interface capabilities
// output: detected data plane mode (TCX/Legacy TC/Native XDP/Generic XDP)
// pos: data plane capability detection and mode selection - if file updated, must sync with this header comment and pkg/dataplane/CLAUDE.md
package dataplane

import (
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/features"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// DataPlaneMode 数据平面模式
type DataPlaneMode int

const (
	ModeUnknown DataPlaneMode = iota
	ModeNativeXDP             // Native XDP（驱动支持，最快）
	ModeGenericXDP            // Generic XDP（内核回退，较快）
	ModeTCX                   // TCX（kernel >= 6.6）
	ModeLegacyTC              // Legacy TC（kernel >= 4.18）
)

// String 返回模式名称
func (m DataPlaneMode) String() string {
	switch m {
	case ModeNativeXDP:
		return "Native XDP"
	case ModeGenericXDP:
		return "Generic XDP"
	case ModeTCX:
		return "TCX"
	case ModeLegacyTC:
		return "Legacy TC"
	default:
		return "Unknown"
	}
}

// Capabilities 系统数据平面能力
type Capabilities struct {
	// 内核版本信息
	KernelVersion string
	KernelMajor   int
	KernelMinor   int

	// XDP 支持
	SupportsXDP       bool // 是否支持 XDP 程序类型
	SupportsNativeXDP bool // 是否支持 Native XDP（驱动级别）
	SupportsGenericXDP bool // 是否支持 Generic XDP

	// TC 支持
	SupportsTCX      bool // 是否支持 TCX (kernel >= 6.6)
	SupportsLegacyTC bool // 是否支持 Legacy TC

	// 网卡信息
	InterfaceName string
	InterfaceIndex int
}

// DetectCapabilities 检测系统数据平面能力
func DetectCapabilities(ifaceName string) (*Capabilities, error) {
	caps := &Capabilities{
		InterfaceName: ifaceName,
	}

	// 1. 检测内核版本
	var uname unix.Utsname
	if err := unix.Uname(&uname); err != nil {
		return nil, fmt.Errorf("failed to get kernel version: %w", err)
	}

	// 解析内核版本字符串
	caps.KernelVersion = unix.ByteSliceToString(uname.Release[:])
	fmt.Sscanf(caps.KernelVersion, "%d.%d", &caps.KernelMajor, &caps.KernelMinor)

	log.Debugf("Detected kernel version: %s (%d.%d)",
		caps.KernelVersion, caps.KernelMajor, caps.KernelMinor)

	// 2. 检测 XDP 程序类型支持
	caps.SupportsXDP = features.HaveProgramType(ebpf.XDP) == nil
	if caps.SupportsXDP {
		log.Debug("✓ XDP program type supported")
	} else {
		log.Debug("✗ XDP program type not supported")
	}

	// 3. 检测 TCX 支持（kernel >= 6.6）
	// TCX 是 kernel 6.6 引入的新一代 TC hook
	caps.SupportsTCX = caps.KernelMajor > 6 ||
		(caps.KernelMajor == 6 && caps.KernelMinor >= 6)

	if caps.SupportsTCX {
		log.Debug("✓ TCX supported (kernel >= 6.6)")
	} else {
		log.Debug("✗ TCX not supported (kernel < 6.6)")
	}

	// 4. Legacy TC 支持检查（kernel >= 4.18）
	caps.SupportsLegacyTC = caps.KernelMajor > 4 ||
		(caps.KernelMajor == 4 && caps.KernelMinor >= 18)

	if caps.SupportsLegacyTC {
		log.Debug("✓ Legacy TC supported")
	} else {
		log.Warn("✗ Legacy TC not supported - kernel too old")
	}

	// 5. 检测 Native XDP 驱动支持
	// 通过实际尝试附加测试程序来验证
	if caps.SupportsXDP {
		// 测试 Native XDP (驱动级别支持)
		caps.SupportsNativeXDP = testNativeXDPSupport(ifaceName)

		// 测试 Generic XDP (内核回退实现)
		caps.SupportsGenericXDP = testGenericXDPSupport(ifaceName)

		if caps.SupportsNativeXDP {
			log.Debugf("✓ Native XDP supported on %s", ifaceName)
		} else {
			log.Debugf("✗ Native XDP not supported on %s (driver limitation)", ifaceName)
		}

		if caps.SupportsGenericXDP {
			log.Debugf("✓ Generic XDP supported on %s", ifaceName)
		} else {
			log.Debugf("✗ Generic XDP not supported on %s", ifaceName)
		}
	}

	return caps, nil
}

// ModeConfig 模式选择配置
type ModeConfig struct {
	// ForceMode 强制使用指定模式（0 表示自动选择）
	ForceMode DataPlaneMode

	// PreferXDP 优先使用 XDP（在自动模式下）
	PreferXDP bool

	// AllowGenericXDP 允许使用 Generic XDP 回退
	AllowGenericXDP bool
}

// SelectBestMode 根据系统能力和配置选择最佳模式
func SelectBestMode(caps *Capabilities, config *ModeConfig) DataPlaneMode {
	// 用户强制指定模式
	if config.ForceMode != ModeUnknown {
		if validateMode(config.ForceMode, caps) {
			log.Infof("Using forced mode: %v", config.ForceMode)
			return config.ForceMode
		}
		log.Warnf("Forced mode %v not available, falling back to auto-select", config.ForceMode)
	}

	// 自动选择模式（性能优先）
	log.Debug("Auto-selecting best dataplane mode...")

	// 1. 优先尝试 XDP（如果配置允许）
	if config.PreferXDP {
		// 1.1 尝试 Native XDP（最佳性能）
		if caps.SupportsNativeXDP {
			log.Info("Selected mode: Native XDP (best performance)")
			return ModeNativeXDP
		}

		// 1.2 回退到 Generic XDP
		if config.AllowGenericXDP && caps.SupportsGenericXDP {
			log.Info("Selected mode: Generic XDP (good performance)")
			return ModeGenericXDP
		}
	}

	// 2. 回退到 TC
	if caps.SupportsTCX {
		log.Info("Selected mode: TCX (kernel >= 6.6)")
		return ModeTCX
	}

	if caps.SupportsLegacyTC {
		log.Info("Selected mode: Legacy TC (fallback)")
		return ModeLegacyTC
	}

	// 3. 无可用模式
	log.Error("No suitable dataplane mode available")
	return ModeUnknown
}

// validateMode 验证指定模式是否可用
func validateMode(mode DataPlaneMode, caps *Capabilities) bool {
	switch mode {
	case ModeNativeXDP:
		return caps.SupportsNativeXDP
	case ModeGenericXDP:
		return caps.SupportsGenericXDP
	case ModeTCX:
		return caps.SupportsTCX
	case ModeLegacyTC:
		return caps.SupportsLegacyTC
	default:
		return false
	}
}

// IsXDPMode 判断是否为 XDP 模式
func IsXDPMode(mode DataPlaneMode) bool {
	return mode == ModeNativeXDP || mode == ModeGenericXDP
}

// IsTCMode 判断是否为 TC 模式
func IsTCMode(mode DataPlaneMode) bool {
	return mode == ModeTCX || mode == ModeLegacyTC
}
