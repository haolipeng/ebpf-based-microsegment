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
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall" -target amd64 xdpbpf ../../../bpf/xdp_microsegment.bpf.c -- -I../../../bpf -I../../../../vmlinux/x86

// DataPlane 管理 eBPF 数据平面
// 高级 API，封装了统计、监控等功能
type DataPlane struct {
	manager  *Manager         // 数据平面管理器 (底层)
	rbReader *ringbuf.Reader  // Ring buffer reader
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
// 自动检测系统能力并选择最佳模式
//
// 默认配置：优先 TC 模式（更稳定），支持自动选择 TCX 或 Legacy TC
// 如果需要 XDP，请使用 NewWithConfig() 并设置 PreferXDP: true
func New(iface string) (*DataPlane, error) {
	// 使用默认配置 (TC 优先，保持向后兼容)
	config := &ModeConfig{
		ForceMode:       ModeUnknown, // 自动选择
		PreferXDP:       false,       // 默认使用 TC (更稳定)
		AllowGenericXDP: true,        // 允许 Generic XDP 回退
	}

	return NewWithConfig(iface, config)
}

// NewWithConfig 使用自定义配置创建数据平面实例
//
// 参数:
//   - iface: 网卡名称
//   - config: 模式配置（传 nil 使用默认配置）
//
// 返回:
//   - *DataPlane: 数据平面实例
//   - error: 错误信息
func NewWithConfig(iface string, config *ModeConfig) (*DataPlane, error) {
	// 1. 创建 Manager
	manager, err := NewManager(iface, config)
	if err != nil {
		return nil, fmt.Errorf("creating manager: %w", err)
	}

	// 2. 加载数据平面
	if err := manager.Load(); err != nil {
		return nil, fmt.Errorf("loading dataplane: %w", err)
	}

	// 3. 获取 Maps
	maps, err := manager.GetMaps()
	if err != nil {
		manager.Unload()
		return nil, fmt.Errorf("getting maps: %w", err)
	}

	// 4. 创建 Ring Buffer Reader
	rbReader, err := ringbuf.NewReader(maps.FlowEventsRB)
	if err != nil {
		manager.Unload()
		return nil, fmt.Errorf("creating ring buffer reader: %w", err)
	}

	dp := &DataPlane{
		manager:  manager,
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

	// 卸载数据平面
	if dp.manager != nil {
		if err := dp.manager.Unload(); err != nil {
			errs = append(errs, fmt.Errorf("unloading dataplane: %w", err))
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

	// 获取 maps
	maps, err := dp.manager.GetMaps()
	if err != nil {
		log.Debugf("Failed to get maps: %v", err)
		return stats
	}

	// Helper function to read per-CPU array and sum values
	readStat := func(key uint32) uint64 {
		var values []uint64
		if err := maps.StatsMap.Lookup(&key, &values); err != nil {
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
	maps, err := dp.manager.GetMaps()
	if err != nil {
		log.Debugf("Failed to get maps: %v", err)
		return nil
	}
	return maps.SessionMap
}

// GetPolicyMap 返回策略 map 供外部访问
func (dp *DataPlane) GetPolicyMap() *ebpf.Map {
	maps, err := dp.manager.GetMaps()
	if err != nil {
		log.Debugf("Failed to get maps: %v", err)
		return nil
	}
	return maps.PolicyMap
}

// GetWildcardPolicyMap 返回通配符策略 map 供外部访问
func (dp *DataPlane) GetWildcardPolicyMap() *ebpf.Map {
	maps, err := dp.manager.GetMaps()
	if err != nil {
		log.Debugf("Failed to get maps: %v", err)
		return nil
	}
	return maps.WildcardPolicyMap
}

// GetMode 返回当前数据平面模式
func (dp *DataPlane) GetMode() DataPlaneMode {
	return dp.manager.GetMode()
}

// GetManager 返回底层的 Manager (用于高级操作)
func (dp *DataPlane) GetManager() *Manager {
	return dp.manager
}
