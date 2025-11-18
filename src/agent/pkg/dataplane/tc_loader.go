// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"errors"
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// TCLoader 负责加载和管理 TC (Traffic Control) eBPF 程序
type TCLoader struct {
	mode         DataPlaneMode      // ModeTCX 或 ModeLegacyTC
	iface        string             // 网卡名称
	ifaceIdx     int                // 网卡索引
	objs         *bpfObjects        // eBPF 对象
	ingressLink  link.Link          // TCX ingress link (kernel >= 6.6)
	egressLink   link.Link          // TCX egress link (kernel >= 6.6) - 新增
	ingressFilter *netlink.BpfFilter // Legacy TC ingress filter (kernel < 6.6)
	egressFilter  *netlink.BpfFilter // Legacy TC egress filter (kernel < 6.6) - 新增
	pinConfig    *MapPinConfig      // Map pinning 配置
}

// NewTCLoader 创建一个新的 TC 加载器
func NewTCLoader(mode DataPlaneMode, iface string, ifaceIdx int) (*TCLoader, error) {
	if mode != ModeTCX && mode != ModeLegacyTC {
		return nil, fmt.Errorf("invalid mode for TCLoader: %v (must be TCX or LegacyTC)", mode)
	}

	return &TCLoader{
		mode:      mode,
		iface:     iface,
		ifaceIdx:  ifaceIdx,
		pinConfig: DefaultMapPinConfig(), // 使用默认 Map Pinning 配置
	}, nil
}

// Load 加载并附加 TC eBPF 程序到网卡
func (l *TCLoader) Load() error {
	// 1. 确保 pin 目录存在
	if err := EnsurePinPath(l.pinConfig.PinPath); err != nil {
		log.Warnf("Failed to ensure pin path (continuing anyway): %v", err)
	}

	// 2. 加载 eBPF 对象,使用 Map Pinning
	objs := &bpfObjects{}
	opts := &ebpf.CollectionOptions{
		Maps: ebpf.MapOptions{
			// 设置 pin 路径,cilium/ebpf 会自动处理 LIBBPF_PIN_BY_NAME 的 map
			PinPath: l.pinConfig.PinPath,
		},
	}

	if err := loadBpfObjects(objs, opts); err != nil {
		return fmt.Errorf("loading eBPF objects: %w", err)
	}
	l.objs = objs

	log.Debugf("eBPF objects loaded successfully (with Map Pinning)")
	log.Infof("✓ Pinned maps to: %s", l.pinConfig.PinPath)

	// 3. 根据模式附加 TC 程序 (双向: ingress + egress)
	switch l.mode {
	case ModeTCX:
		// 先附加 ingress
		if err := l.attachTCXIngress(); err != nil {
			return err
		}
		// 再附加 egress (如果失败,清理 ingress)
		if err := l.attachTCXEgress(); err != nil {
			l.detachTCXIngress() // 清理已附加的 ingress
			return err
		}
		return nil
	case ModeLegacyTC:
		// 先附加 ingress
		if err := l.attachLegacyTCIngress(); err != nil {
			return err
		}
		// 再附加 egress (如果失败,清理 ingress)
		if err := l.attachLegacyTCEgress(); err != nil {
			l.detachLegacyTCIngress() // 清理已附加的 ingress
			return err
		}
		return nil
	default:
		l.objs.Close()
		return fmt.Errorf("unsupported TC mode: %v", l.mode)
	}
}

// attachTCXIngress 使用 TCX 附加 Ingress 程序 (kernel >= 6.6)
func (l *TCLoader) attachTCXIngress() error {
	ingressLink, err := link.AttachTCX(link.TCXOptions{
		Interface: l.ifaceIdx,
		Program:   l.objs.TcMicrosegmentFilter,
		Attach:    ebpf.AttachTCXIngress,
	})
	if err != nil {
		l.objs.Close()
		return fmt.Errorf("attaching TCX ingress: %w", err)
	}

	l.ingressLink = ingressLink
	log.Infof("✓ TC ingress program attached to %s (TCX mode)", l.iface)
	return nil
}

// attachTCXEgress 使用 TCX 附加 Egress 程序 (kernel >= 6.6)
func (l *TCLoader) attachTCXEgress() error {
	egressLink, err := link.AttachTCX(link.TCXOptions{
		Interface: l.ifaceIdx,
		Program:   l.objs.TcMicrosegmentFilter, // 使用同一个程序
		Attach:    ebpf.AttachTCXEgress,         // Egress 方向
	})
	if err != nil {
		return fmt.Errorf("attaching TCX egress: %w", err)
	}

	l.egressLink = egressLink
	log.Infof("✓ TC egress program attached to %s (TCX mode)", l.iface)
	return nil
}

// detachTCXIngress 分离 TCX Ingress hook
func (l *TCLoader) detachTCXIngress() error {
	if l.ingressLink != nil {
		if err := l.ingressLink.Close(); err != nil {
			return fmt.Errorf("detaching TCX ingress: %w", err)
		}
		l.ingressLink = nil
		log.Debugf("TCX ingress link closed")
	}
	return nil
}

// detachTCXEgress 分离 TCX Egress hook
func (l *TCLoader) detachTCXEgress() error {
	if l.egressLink != nil {
		if err := l.egressLink.Close(); err != nil {
			return fmt.Errorf("detaching TCX egress: %w", err)
		}
		l.egressLink = nil
		log.Debugf("TCX egress link closed")
	}
	return nil
}

// attachLegacyTCIngress 使用 netlink 附加 Ingress 程序 (kernel >= 4.18)
func (l *TCLoader) attachLegacyTCIngress() error {
	// 1. 获取 netlink interface
	nlLink, err := netlink.LinkByIndex(l.ifaceIdx)
	if err != nil {
		l.objs.Close()
		return fmt.Errorf("getting netlink interface: %w", err)
	}

	// 2. 创建 clsact qdisc (如果不存在)
	qdisc := &netlink.GenericQdisc{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: l.ifaceIdx,
			Handle:    netlink.MakeHandle(0xffff, 0),
			Parent:    netlink.HANDLE_CLSACT,
		},
		QdiscType: "clsact",
	}

	// 尝试添加 qdisc,忽略"文件已存在"错误
	if err := netlink.QdiscAdd(qdisc); err != nil {
		if !isFileExistsError(err) {
			l.objs.Close()
			return fmt.Errorf("adding clsact qdisc: %w", err)
		}
		log.Debugf("clsact qdisc already exists on %s", l.iface)
	} else {
		log.Debugf("Added clsact qdisc to %s", l.iface)
	}

	// 3. 清理旧的 ingress filter (来自之前的运行)
	existingFilters, err := netlink.FilterList(nlLink, netlink.HANDLE_MIN_INGRESS)
	if err == nil {
		for _, f := range existingFilters {
			if bpfFilter, ok := f.(*netlink.BpfFilter); ok {
				if bpfFilter.Name == "tc_microsegment_ingress" {
					netlink.FilterDel(bpfFilter)
					log.Debugf("Removed old ingress BPF filter from %s", l.iface)
				}
			}
		}
	}

	// 4. 附加 ingress BPF filter
	ingressFilter := &netlink.BpfFilter{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: l.ifaceIdx,
			Parent:    netlink.HANDLE_MIN_INGRESS,
			Handle:    1,
			Protocol:  unix.ETH_P_ALL,
			Priority:  1,
		},
		Fd:           l.objs.TcMicrosegmentFilter.FD(),
		Name:         "tc_microsegment_ingress",
		DirectAction: true,
	}

	if err := netlink.FilterAdd(ingressFilter); err != nil {
		l.objs.Close()
		return fmt.Errorf("attaching TC ingress filter: %w", err)
	}

	l.ingressFilter = ingressFilter
	log.Infof("✓ TC ingress program attached to %s (legacy netlink mode)", l.iface)
	return nil
}

// attachLegacyTCEgress 使用 netlink 附加 Egress 程序 (kernel >= 4.18)
func (l *TCLoader) attachLegacyTCEgress() error {
	// 1. 获取 netlink interface
	nlLink, err := netlink.LinkByIndex(l.ifaceIdx)
	if err != nil {
		return fmt.Errorf("getting netlink interface: %w", err)
	}

	// 2. 确保 clsact qdisc 存在 (应该已在 attachLegacyTCIngress 中创建)
	// 这里不重复创建,只是为了防御性编程

	// 3. 清理旧的 egress filter (来自之前的运行)
	existingFilters, err := netlink.FilterList(nlLink, netlink.HANDLE_MIN_EGRESS)
	if err == nil {
		for _, f := range existingFilters {
			if bpfFilter, ok := f.(*netlink.BpfFilter); ok {
				if bpfFilter.Name == "tc_microsegment_egress" {
					netlink.FilterDel(bpfFilter)
					log.Debugf("Removed old egress BPF filter from %s", l.iface)
				}
			}
		}
	}

	// 4. 附加 egress BPF filter
	egressFilter := &netlink.BpfFilter{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: l.ifaceIdx,
			Parent:    netlink.HANDLE_MIN_EGRESS, // Egress parent
			Handle:    2,                         // 不同的 handle (避免冲突)
			Protocol:  unix.ETH_P_ALL,
			Priority:  1,
		},
		Fd:           l.objs.TcMicrosegmentFilter.FD(), // 使用同一个程序
		Name:         "tc_microsegment_egress",
		DirectAction: true,
	}

	if err := netlink.FilterAdd(egressFilter); err != nil {
		return fmt.Errorf("attaching TC egress filter: %w", err)
	}

	l.egressFilter = egressFilter
	log.Infof("✓ TC egress program attached to %s (legacy netlink mode)", l.iface)
	return nil
}

// detachLegacyTCIngress 删除 Legacy TC Ingress filter
func (l *TCLoader) detachLegacyTCIngress() error {
	if l.ingressFilter != nil {
		if err := netlink.FilterDel(l.ingressFilter); err != nil {
			return fmt.Errorf("removing TC ingress filter: %w", err)
		}
		l.ingressFilter = nil
		log.Debugf("TC ingress filter removed from %s", l.iface)
	}
	return nil
}

// detachLegacyTCEgress 删除 Legacy TC Egress filter
func (l *TCLoader) detachLegacyTCEgress() error {
	if l.egressFilter != nil {
		if err := netlink.FilterDel(l.egressFilter); err != nil {
			return fmt.Errorf("removing TC egress filter: %w", err)
		}
		l.egressFilter = nil
		log.Debugf("TC egress filter removed from %s", l.iface)
	}
	return nil
}

// Unload 卸载 TC eBPF 程序 (双向: egress + ingress)
func (l *TCLoader) Unload() error {
	var errs []error

	// 1. 清理 TC 附加 (TCX 或 legacy) - 分别卸载 egress 和 ingress
	if l.mode == ModeLegacyTC {
		// Legacy netlink-based TC cleanup
		// 先卸载 egress
		if err := l.detachLegacyTCEgress(); err != nil {
			errs = append(errs, err)
		}
		// 再卸载 ingress
		if err := l.detachLegacyTCIngress(); err != nil {
			errs = append(errs, err)
		}
	} else if l.mode == ModeTCX {
		// TCX cleanup
		// 先卸载 egress
		if err := l.detachTCXEgress(); err != nil {
			errs = append(errs, err)
		}
		// 再卸载 ingress
		if err := l.detachTCXIngress(); err != nil {
			errs = append(errs, err)
		}
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

	log.Debugf("TC loader unloaded successfully (ingress + egress)")
	return nil
}

// GetMaps 返回 eBPF Map 引用
func (l *TCLoader) GetMaps() (*DataPlaneMaps, error) {
	if l.objs == nil {
		return nil, fmt.Errorf("eBPF objects not loaded")
	}

	return &DataPlaneMaps{
		SessionMap:         l.objs.SessionMap,
		PolicyMap:          l.objs.PolicyMap,
		WildcardPolicyMap:  l.objs.WildcardPolicyMap,
		ProtocolOffsetMap:  l.objs.ProtocolOffsetMap,
		StatsMap:           l.objs.StatsMap,
		FlowEventsRB:       l.objs.FlowEvents,
		ConntrackCacheMap:  l.objs.ConntrackCacheMap,
		NATConfigMap:       l.objs.NatConfigMap,
		NATStatsMap:        l.objs.NatStatsMap,
		TimeoutConfigMap:   l.objs.TimeoutConfigMap,
	}, nil
}

// GetMode 返回当前 TC 模式
func (l *TCLoader) GetMode() DataPlaneMode {
	return l.mode
}

// isFileExistsError 检查错误是否为"文件已存在"
func isFileExistsError(err error) bool {
	if err == nil {
		return false
	}
	// 检查 EEXIST 错误 (文件已存在)
	return err.Error() == "file exists" ||
		err.Error() == unix.EEXIST.Error() ||
		errors.Is(err, unix.EEXIST)
}
