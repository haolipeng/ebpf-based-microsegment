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
	mode      DataPlaneMode     // ModeTCX 或 ModeLegacyTC
	iface     string            // 网卡名称
	ifaceIdx  int               // 网卡索引
	objs      *bpfObjects       // eBPF 对象
	tcLink    link.Link         // TCX link (kernel >= 6.6)
	tcFilter  *netlink.BpfFilter // Legacy TC filter (kernel < 6.6)
	pinConfig *MapPinConfig     // Map pinning 配置
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

	// 3. 根据模式附加 TC 程序
	switch l.mode {
	case ModeTCX:
		return l.attachTCX()
	case ModeLegacyTC:
		return l.attachLegacyTC()
	default:
		l.objs.Close()
		return fmt.Errorf("unsupported TC mode: %v", l.mode)
	}
}

// attachTCX 使用 TCX 附加 TC 程序 (kernel >= 6.6)
func (l *TCLoader) attachTCX() error {
	tcLink, err := link.AttachTCX(link.TCXOptions{
		Interface: l.ifaceIdx,
		Program:   l.objs.TcMicrosegmentFilter,
		Attach:    ebpf.AttachTCXIngress,
	})
	if err != nil {
		l.objs.Close()
		return fmt.Errorf("attaching TCX program: %w", err)
	}

	l.tcLink = tcLink
	log.Infof("✓ TC program attached to %s ingress (TCX mode, kernel >= 6.6)", l.iface)
	return nil
}

// attachLegacyTC 使用 netlink 附加 TC 程序 (kernel >= 4.18)
func (l *TCLoader) attachLegacyTC() error {
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

	// 3. 清理旧的 BPF filter (来自之前的运行)
	existingFilters, err := netlink.FilterList(nlLink, netlink.HANDLE_MIN_INGRESS)
	if err == nil {
		for _, f := range existingFilters {
			if bpfFilter, ok := f.(*netlink.BpfFilter); ok {
				if bpfFilter.Name == "tc_microsegment_filter" {
					netlink.FilterDel(bpfFilter)
					log.Debugf("Removed old BPF filter from %s", l.iface)
				}
			}
		}
	}

	// 4. 附加 BPF filter
	filter := &netlink.BpfFilter{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: l.ifaceIdx,
			Parent:    netlink.HANDLE_MIN_INGRESS,
			Handle:    1,
			Protocol:  unix.ETH_P_ALL,
			Priority:  1,
		},
		Fd:           l.objs.TcMicrosegmentFilter.FD(),
		Name:         "tc_microsegment_filter",
		DirectAction: true,
	}

	if err := netlink.FilterAdd(filter); err != nil {
		l.objs.Close()
		return fmt.Errorf("attaching TC filter: %w", err)
	}

	l.tcFilter = filter
	log.Infof("✓ TC program attached to %s ingress (legacy netlink mode, kernel >= 4.18)", l.iface)
	return nil
}

// Unload 卸载 TC eBPF 程序
func (l *TCLoader) Unload() error {
	var errs []error

	// 1. 清理 TC 附加 (TCX 或 legacy)
	if l.mode == ModeLegacyTC && l.tcFilter != nil {
		// Legacy netlink-based TC cleanup
		if err := netlink.FilterDel(l.tcFilter); err != nil {
			errs = append(errs, fmt.Errorf("removing TC filter: %w", err))
		} else {
			log.Debugf("TC filter removed from %s", l.iface)
		}
		l.tcFilter = nil
	} else if l.mode == ModeTCX && l.tcLink != nil {
		// TCX cleanup
		if err := l.tcLink.Close(); err != nil {
			errs = append(errs, fmt.Errorf("detaching TCX program: %w", err))
		}
		l.tcLink = nil
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

	log.Debugf("TC loader unloaded successfully")
	return nil
}

// GetMaps 返回 eBPF Map 引用
func (l *TCLoader) GetMaps() (*DataPlaneMaps, error) {
	if l.objs == nil {
		return nil, fmt.Errorf("eBPF objects not loaded")
	}

	return &DataPlaneMaps{
		SessionMap:        l.objs.SessionMap,
		PolicyMap:         l.objs.PolicyMap,
		WildcardPolicyMap: l.objs.WildcardPolicyMap,
		StatsMap:          l.objs.StatsMap,
		FlowEventsRB:      l.objs.FlowEvents,
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
