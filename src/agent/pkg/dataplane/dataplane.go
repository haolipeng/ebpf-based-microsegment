// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/ringbuf"
	log "github.com/sirupsen/logrus"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall" -target amd64 bpf ../../../bpf/tc_microsegment.bpf.c -- -I../../../bpf -I../../../../vmlinux/x86

// DataPlane 管理 eBPF 数据平面
type DataPlane struct {
	iface    string          // 网卡名称
	ifaceIdx int             // 网卡索引
	mode     DataPlaneMode   // 数据平面模式
	tcLoader *TCLoader       // TC 加载器
	maps     *DataPlaneMaps  // eBPF Map 引用
	rbReader *ringbuf.Reader // Ring buffer reader
}

// Statistics 保存数据包处理统计信息
type Statistics struct {
	TotalPackets   uint64
	AllowedPackets uint64
	DeniedPackets  uint64
	NewSessions    uint64
	ClosedSessions uint64
	ActiveSessions uint64
	PolicyHits     uint64
	PolicyMisses   uint64
}

// New 创建一个新的数据平面实例
// 自动检测系统能力并选择最佳模式 (TCX 或 Legacy TC)
func New(iface string) (*DataPlane, error) {
	// 获取网卡索引
	ifaceObj, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, fmt.Errorf("interface %s not found: %w", iface, err)
	}

	// 检测系统能力
	caps, err := DetectCapabilities(iface)
	if err != nil {
		return nil, fmt.Errorf("detecting capabilities: %w", err)
	}

	// 选择模式 (目前只支持 TC,优先 TCX)
	config := &ModeConfig{
		ForceMode:       ModeUnknown, // 自动选择
		PreferXDP:       false,       // 目前不使用 XDP
		AllowGenericXDP: false,
	}

	// 选择最佳模式
	mode := SelectBestMode(caps, config)

	// 如果选择了 XDP 模式,暂时回退到 TC (因为 XDP 还未实现)
	if IsXDPMode(mode) {
		log.Warn("XDP mode selected but not yet implemented, falling back to TC")
		if caps.SupportsTCX {
			mode = ModeTCX
		} else {
			mode = ModeLegacyTC
		}
	}

	// 验证选择的模式
	if !IsTCMode(mode) {
		return nil, fmt.Errorf("no suitable TC mode available (mode=%v)", mode)
	}

	log.Infof("Selected dataplane mode: %v", mode)

	// 创建 TC 加载器
	tcLoader, err := NewTCLoader(mode, iface, ifaceObj.Index)
	if err != nil {
		return nil, fmt.Errorf("creating TC loader: %w", err)
	}

	// 加载 TC 程序
	if err := tcLoader.Load(); err != nil {
		return nil, fmt.Errorf("loading TC program: %w", err)
	}

	// 获取 Map 引用
	maps, err := tcLoader.GetMaps()
	if err != nil {
		tcLoader.Unload()
		return nil, fmt.Errorf("getting maps: %w", err)
	}

	// 创建 Ring Buffer Reader
	rbReader, err := ringbuf.NewReader(maps.FlowEventsRB)
	if err != nil {
		tcLoader.Unload()
		return nil, fmt.Errorf("creating ring buffer reader: %w", err)
	}

	dp := &DataPlane{
		iface:    iface,
		ifaceIdx: ifaceObj.Index,
		mode:     mode,
		tcLoader: tcLoader,
		maps:     maps,
		rbReader: rbReader,
	}

	log.Info("✓ Data plane initialized")
	return dp, nil
}

// Close 清理数据平面资源
func (dp *DataPlane) Close() error {
	var errs []error

	// 关闭 Ring Buffer Reader
	if dp.rbReader != nil {
		if err := dp.rbReader.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing ring buffer reader: %w", err))
		}
	}

	// 卸载 TC 程序
	if dp.tcLoader != nil {
		if err := dp.tcLoader.Unload(); err != nil {
			errs = append(errs, fmt.Errorf("unloading TC loader: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	log.Info("Data plane closed successfully")
	return nil
}

// GetStatistics 获取当前数据包处理统计信息
func (dp *DataPlane) GetStatistics() Statistics {
	stats := Statistics{}

	// Helper function to read per-CPU array and sum values
	readStat := func(key uint32) uint64 {
		var values []uint64
		if err := dp.maps.StatsMap.Lookup(&key, &values); err != nil {
			log.Debugf("Failed to lookup stat key %d: %v", key, err)
			return 0
		}

		var total uint64
		for _, v := range values {
			total += v
		}
		return total
	}

	stats.TotalPackets = readStat(0)
	stats.AllowedPackets = readStat(1)
	stats.DeniedPackets = readStat(2)
	stats.NewSessions = readStat(3)
	stats.ClosedSessions = readStat(4)
	stats.ActiveSessions = readStat(5)
	stats.PolicyHits = readStat(6)
	stats.PolicyMisses = readStat(7)

	return stats
}

// MonitorFlowEvents 持续读取和处理 ring buffer 中的流事件
func (dp *DataPlane) MonitorFlowEvents() {
	log.Info("Starting flow event monitoring")

	for {
		record, err := dp.rbReader.Read()
		if err != nil {
			if errors.Is(err, ringbuf.ErrClosed) {
				log.Info("Ring buffer closed")
				return
			}
			log.Errorf("Reading from ring buffer: %v", err)
			continue
		}

		// 解析流事件 - 简单的结构体解析
		if len(record.RawSample) < 32 {
			log.Warn("Received incomplete flow event")
			continue
		}

		// 手动解析流事件结构
		// 解析流键 (5-tuple)
		srcIP := binary.LittleEndian.Uint32(record.RawSample[0:4])
		dstIP := binary.LittleEndian.Uint32(record.RawSample[4:8])
		srcPort := binary.LittleEndian.Uint16(record.RawSample[8:10])
		dstPort := binary.LittleEndian.Uint16(record.RawSample[10:12])
		protocol := record.RawSample[12]

		srcIPStr := intToIP(srcIP)
		dstIPStr := intToIP(dstIP)

		log.Infof("[FLOW EVENT] %s:%d -> %s:%d proto=%d",
			srcIPStr, srcPort,
			dstIPStr, dstPort,
			protocol)
	}
}

// intToIP 将 uint32 IP 转换为 net.IP
func intToIP(ip uint32) net.IP {
	return net.IPv4(byte(ip), byte(ip>>8), byte(ip>>16), byte(ip>>24))
}

// GetFlowRingBuffer 返回流事件的 ring buffer reader
// 这允许外部组件 (如 flow.Collector) 读取流事件
func (dp *DataPlane) GetFlowRingBuffer() *ringbuf.Reader {
	return dp.rbReader
}

// GetSessionMap 返回会话 map 供外部访问
func (dp *DataPlane) GetSessionMap() *ebpf.Map {
	return dp.maps.SessionMap
}

// GetPolicyMap 返回策略 map 供外部访问
func (dp *DataPlane) GetPolicyMap() *ebpf.Map {
	return dp.maps.PolicyMap
}

// GetWildcardPolicyMap 返回通配符策略 map 供外部访问
func (dp *DataPlane) GetWildcardPolicyMap() *ebpf.Map {
	return dp.maps.WildcardPolicyMap
}

// GetMode 返回当前数据平面模式
func (dp *DataPlane) GetMode() DataPlaneMode {
	return dp.mode
}
