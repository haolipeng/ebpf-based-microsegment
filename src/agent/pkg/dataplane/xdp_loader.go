// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"errors"
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	log "github.com/sirupsen/logrus"
)

// XDPLoader 负责加载和管理 XDP (eXpress Data Path) eBPF 程序
type XDPLoader struct {
	mode      DataPlaneMode  // ModeNativeXDP 或 ModeGenericXDP
	iface     string         // 网卡名称
	ifaceIdx  int            // 网卡索引
	objs      *xdpbpfObjects // eBPF 对象
	xdpLink   link.Link      // XDP link
	pinConfig *MapPinConfig  // Map pinning 配置
}

// NewXDPLoader 创建一个新的 XDP 加载器
//
// 参数:
//   - mode: XDP 模式 (ModeNativeXDP 或 ModeGenericXDP)
//   - iface: 网卡名称
//   - ifaceIdx: 网卡索引
//
// 返回:
//   - *XDPLoader: XDP 加载器实例
//   - error: 错误信息
func NewXDPLoader(mode DataPlaneMode, iface string, ifaceIdx int) (*XDPLoader, error) {
	if mode != ModeNativeXDP && mode != ModeGenericXDP {
		return nil, fmt.Errorf("invalid mode for XDPLoader: %v (must be NativeXDP or GenericXDP)", mode)
	}

	return &XDPLoader{
		mode:      mode,
		iface:     iface,
		ifaceIdx:  ifaceIdx,
		pinConfig: DefaultMapPinConfig(), // 使用默认 Map Pinning 配置
	}, nil
}

// Load 加载并附加 XDP eBPF 程序到网卡
//
// 步骤:
//  1. 确保 BPF pin 目录存在
//  2. 加载 eBPF 对象,使用 Map Pinning 共享策略数据
//  3. 根据模式附加 XDP 程序 (Native 或 Generic)
//
// 返回:
//   - error: 错误信息
func (l *XDPLoader) Load() error {
	// 1. 确保 pin 目录存在
	if err := EnsurePinPath(l.pinConfig.PinPath); err != nil {
		log.Warnf("Failed to ensure pin path (continuing anyway): %v", err)
	}

	// 2. 加载 eBPF 对象,使用 Map Pinning
	objs := &xdpbpfObjects{}
	opts := &ebpf.CollectionOptions{
		Maps: ebpf.MapOptions{
			// 设置 pin 路径,cilium/ebpf 会自动处理 LIBBPF_PIN_BY_NAME 的 map
			// 如果 map 已经被 pin (例如由 TC 程序创建),则复用现有 map
			PinPath: l.pinConfig.PinPath,
		},
	}

	if err := loadXdpbpfObjects(objs, opts); err != nil {
		return fmt.Errorf("loading XDP eBPF objects: %w", err)
	}
	l.objs = objs

	log.Debugf("XDP eBPF objects loaded successfully (with Map Pinning)")
	log.Infof("✓ Pinned maps to: %s", l.pinConfig.PinPath)

	// 3. 根据模式附加 XDP 程序
	switch l.mode {
	case ModeNativeXDP:
		return l.attachXDPNative()
	case ModeGenericXDP:
		return l.attachXDPGeneric()
	default:
		l.objs.Close()
		return fmt.Errorf("unsupported XDP mode: %v", l.mode)
	}
}

// attachXDPNative 使用 Native XDP 模式附加程序 (driver-level, 最高性能)
//
// Native XDP 要求:
//   - 网卡驱动支持 XDP (大多数现代网卡支持)
//   - Kernel >= 4.8
//
// 优势:
//   - 最低延迟 (数据包在驱动层处理)
//   - 最高性能 (避免 skb 分配)
//
// 返回:
//   - error: 错误信息
func (l *XDPLoader) attachXDPNative() error {
	xdpLink, err := link.AttachXDP(link.XDPOptions{
		Program:   l.objs.XdpMicrosegmentProg,
		Interface: l.ifaceIdx,
		Flags:     link.XDPDriverMode, // Native XDP (driver-level)
	})
	if err != nil {
		l.objs.Close()
		return fmt.Errorf("attaching Native XDP program: %w", err)
	}

	l.xdpLink = xdpLink
	log.Infof("✓ XDP program attached to %s (Native mode, driver-level)", l.iface)
	return nil
}

// attachXDPGeneric 使用 Generic XDP 模式附加程序 (kernel-level fallback)
//
// Generic XDP 说明:
//   - 适用于所有网卡 (不需要驱动支持)
//   - 在网络栈较高层处理数据包
//   - 性能低于 Native XDP,但高于 TC
//   - Kernel >= 4.12
//
// 使用场景:
//   - 网卡驱动不支持 XDP
//   - 测试和开发环境
//   - 虚拟化环境 (某些虚拟网卡)
//
// 返回:
//   - error: 错误信息
func (l *XDPLoader) attachXDPGeneric() error {
	xdpLink, err := link.AttachXDP(link.XDPOptions{
		Program:   l.objs.XdpMicrosegmentProg,
		Interface: l.ifaceIdx,
		Flags:     link.XDPGenericMode, // Generic XDP (kernel-level fallback)
	})
	if err != nil {
		l.objs.Close()
		return fmt.Errorf("attaching Generic XDP program: %w", err)
	}

	l.xdpLink = xdpLink
	log.Infof("✓ XDP program attached to %s (Generic mode, kernel-level)", l.iface)
	return nil
}

// Unload 卸载 XDP eBPF 程序
//
// 步骤:
//  1. 分离 XDP 程序
//  2. 关闭 eBPF 对象
//
// 注意:
//   - 不会取消 Map Pinning (保留用于 TC/XDP 共享)
//   - 如需清理 pinned maps,使用 CleanupPinnedMaps()
//
// 返回:
//   - error: 错误信息
func (l *XDPLoader) Unload() error {
	var errs []error

	// 1. 分离 XDP 程序
	if l.xdpLink != nil {
		if err := l.xdpLink.Close(); err != nil {
			errs = append(errs, fmt.Errorf("detaching XDP program: %w", err))
		} else {
			log.Debugf("XDP program detached from %s", l.iface)
		}
		l.xdpLink = nil
	}

	// 2. 关闭 eBPF 对象
	if l.objs != nil {
		l.objs.Close()
		l.objs = nil
	}

	// 注意: 我们不在这里取消 Map Pinning
	// Pinned maps 需要保留以供 TC 和 XDP 程序共享
	// 如果需要清理 pinned maps,使用 CleanupPinnedMaps() 函数

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	log.Debugf("XDP loader unloaded successfully")
	return nil
}

// GetMaps 返回 eBPF Map 引用
//
// 返回:
//   - *DataPlaneMaps: Map 集合
//   - error: 错误信息
func (l *XDPLoader) GetMaps() (*DataPlaneMaps, error) {
	if l.objs == nil {
		return nil, fmt.Errorf("XDP eBPF objects not loaded")
	}

	return &DataPlaneMaps{
		SessionMap:        l.objs.SessionMap,
		PolicyMap:         l.objs.PolicyMap,
		WildcardPolicyMap: l.objs.WildcardPolicyMap,
		StatsMap:          l.objs.StatsMap,
		FlowEventsRB:      l.objs.FlowEvents,
	}, nil
}

// GetMode 返回当前 XDP 模式
func (l *XDPLoader) GetMode() DataPlaneMode {
	return l.mode
}
