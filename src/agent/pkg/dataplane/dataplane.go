// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"errors"
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/ringbuf"
	log "github.com/sirupsen/logrus"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/flow"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/session"
)

//go:generate sh -c "go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags \"-O2 -g -Wall ${BPF_CFLAGS}\" -target amd64 bpf ../../../bpf/tc_microsegment.bpf.c -- -I../../../bpf -I../../../../vmlinux/x86"
//go:generate sh -c "go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags \"-O2 -g -Wall ${BPF_CFLAGS}\" -target amd64 xdpbpf ../../../bpf/xdp_microsegment.bpf.c -- -I../../../bpf -I../../../../vmlinux/x86"
//go:generate sh -c "go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags \"-O2 -g -Wall ${BPF_CFLAGS}\" -target amd64 processbpf ../../../bpf/process_monitor.bpf.c -- -I../../../bpf -I../../../../vmlinux/x86"

// eBPF 统计指标索引常量
// 这些常量必须与 eBPF C 代码保持严格一致
// 参考: src/bpf/headers/common_types.h 中的 enum stats_key
//
// 注意：修改这些值时，必须同步更新 C 代码中的定义！
const (
	StatsTotalPackets   uint32 = 0  // STATS_TOTAL_PACKETS
	StatsAllowedPackets uint32 = 1  // STATS_ALLOWED_PACKETS
	StatsDeniedPackets  uint32 = 2  // STATS_DENIED_PACKETS
	StatsNewSessions    uint32 = 3  // STATS_NEW_SESSIONS
	StatsClosedSessions uint32 = 4  // STATS_CLOSED_SESSIONS
	StatsActiveSessions uint32 = 5  // STATS_ACTIVE_SESSIONS
	StatsPolicyHits     uint32 = 6  // STATS_POLICY_HITS
	StatsPolicyMisses   uint32 = 7  // STATS_POLICY_MISSES
	StatsIngressPackets uint32 = 8  // STATS_INGRESS_PACKETS (Direction-specific)
	StatsEgressPackets  uint32 = 9  // STATS_EGRESS_PACKETS  (Direction-specific)
	StatsIngressDenied  uint32 = 10 // STATS_INGRESS_DENIED  (Direction-specific)
	StatsEgressDenied   uint32 = 11 // STATS_EGRESS_DENIED   (Direction-specific)
	// Protocol-specific statistics (12-16)
	StatsIPv4Packets uint32 = 12 // STATS_IPV4_PACKETS
	StatsIPv6Packets uint32 = 13 // STATS_IPV6_PACKETS
	StatsTCPPackets  uint32 = 14 // STATS_TCP_PACKETS
	StatsUDPPackets  uint32 = 15 // STATS_UDP_PACKETS
	StatsICMPPackets uint32 = 16 // STATS_ICMP_PACKETS
	// VLAN statistics (17-18)
	StatsVLANPackets uint32 = 17 // STATS_VLAN_PACKETS
	StatsQinQPackets uint32 = 18 // STATS_QINQ_PACKETS
	// TCP-specific statistics (19-22)
	StatsTCPSyn     uint32 = 19 // STATS_TCP_SYN
	StatsTCPFin     uint32 = 20 // STATS_TCP_FIN
	StatsTCPRst     uint32 = 21 // STATS_TCP_RST
	StatsTCPRetrans uint32 = 22 // STATS_TCP_RETRANS
	// Error statistics (23-24)
	StatsParseErrors uint32 = 23 // STATS_PARSE_ERRORS
	StatsRingbufFull uint32 = 24 // STATS_RINGBUF_FULL
	StatsMax         uint32 = 25 // STATS_MAX - 总的统计项数量
)

// DataPlane 管理 eBPF 数据平面
// 高级 API，封装了统计、监控等功能
type DataPlane struct {
	manager        *Manager                 // 数据平面管理器 (底层)
	rbReader       *ringbuf.Reader          // Ring buffer reader
	timeoutManager *session.TimeoutManager  // Session timeout manager (optional)
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
	// Direction-specific statistics (for egress support)
	IngressPackets uint64
	EgressPackets  uint64
	IngressDenied  uint64
	EgressDenied   uint64
	// Ring buffer statistics
	RingBufferFull uint64 // Number of events dropped due to ring buffer full
}

// New 创建一个新的数据平面实例
// 自动检测系统能力并选择最佳模式
//
// 默认配置：优先 TC 模式（更稳定），支持自动选择 TCX 或 Legacy TC
// 如果需要 XDP，请使用 NewWithConfig() 并设置 PreferXDP: true
func New(iface string) (*DataPlane, error) {
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
		_ = manager.Unload()
		return nil, fmt.Errorf("getting maps: %w", err)
	}

	// 4. 创建 Ring Buffer Reader
	rbReader, err := ringbuf.NewReader(maps.FlowEventsRB)
	if err != nil {
		_ = manager.Unload()
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

	// Stop timeout manager first
	if dp.timeoutManager != nil {
		if err := dp.timeoutManager.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("stopping timeout manager: %w", err))
		}
	}

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

// statMapping defines the relationship between eBPF map keys and Statistics struct fields
type statMapping struct {
	key   uint32
	field *uint64
}

// GetStatistics retrieves and aggregates statistics from the eBPF data plane
// It reads per-CPU values from the stats map and sums them up
func (dp *DataPlane) GetStatistics() Statistics {
	stats := Statistics{}

	// Get eBPF maps
	maps, err := dp.manager.GetMaps()
	if err != nil {
		log.Debugf("Failed to get eBPF maps: %v", err)
		return stats
	}

	// Define mappings between eBPF keys and struct fields
	// This approach is more maintainable and type-safe than reflection
	mappings := []statMapping{
		{StatsTotalPackets, &stats.TotalPackets},
		{StatsAllowedPackets, &stats.AllowedPackets},
		{StatsDeniedPackets, &stats.DeniedPackets},
		{StatsNewSessions, &stats.NewSessions},
		{StatsClosedSessions, &stats.ClosedSessions},
		{StatsActiveSessions, &stats.ActiveSessions},
		{StatsPolicyHits, &stats.PolicyHits},
		{StatsPolicyMisses, &stats.PolicyMisses},
		{StatsIngressPackets, &stats.IngressPackets},
		{StatsEgressPackets, &stats.EgressPackets},
		{StatsIngressDenied, &stats.IngressDenied},
		{StatsEgressDenied, &stats.EgressDenied},
		{StatsRingbufFull, &stats.RingBufferFull},
	}

	// Read and aggregate per-CPU statistics
	for _, m := range mappings {
		*m.field = dp.readAndSumPerCPUStat(maps.StatsMap, m.key)
	}

	return stats
}

// readAndSumPerCPUStat reads per-CPU array values and returns their sum
// Returns 0 if the lookup fails (graceful degradation)
func (dp *DataPlane) readAndSumPerCPUStat(statsMap *ebpf.Map, key uint32) uint64 {
	var perCPUValues []uint64

	if err := statsMap.Lookup(&key, &perCPUValues); err != nil {
		log.Debugf("Failed to lookup stat key %d: %v", key, err)
		return 0
	}

	var sum uint64
	for _, val := range perCPUValues {
		sum += val
	}

	return sum
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

		// Parse flow event using the dedicated parser
		event, err := flow.ParseFlowEvent(record.RawSample)
		if err != nil {
			log.Warnf("Failed to parse flow event: %v", err)
			continue
		}

		// Convert to Flow structure (handles IPv4/IPv6 conversion)
		flowData := event.ToFlow()

		// Handle different event types
		switch event.EventType {
		case flow.FlowEventNew:
			log.Infof("[FLOW NEW] %s:%d -> %s:%d proto=%s dir=%s action=%s packets=%d bytes=%d",
				flowData.SourceIP, event.SrcPort,
				flowData.DestIP, event.DstPort,
				event.Protocol, event.Direction, event.PolicyAction,
				event.PacketCount, event.ByteCount)

		case flow.FlowEventClosed:
			log.Infof("[FLOW CLOSED] %s:%d -> %s:%d proto=%s dir=%s packets=%d bytes=%d",
				flowData.SourceIP, event.SrcPort,
				flowData.DestIP, event.DstPort,
				event.Protocol, event.Direction,
				event.PacketCount, event.ByteCount)

		case flow.FlowEventTimeout:
			log.Infof("[FLOW TIMEOUT] %s:%d -> %s:%d proto=%s dir=%s packets=%d bytes=%d",
				flowData.SourceIP, event.SrcPort,
				flowData.DestIP, event.DstPort,
				event.Protocol, event.Direction,
				event.PacketCount, event.ByteCount)

		case flow.FlowEventUpdate:
			log.Debugf("[FLOW UPDATE] %s:%d -> %s:%d proto=%s packets=%d bytes=%d",
				flowData.SourceIP, event.SrcPort,
				flowData.DestIP, event.DstPort,
				event.Protocol,
				event.PacketCount, event.ByteCount)

		default:
			log.Warnf("[FLOW UNKNOWN] Unknown event type %d for %s:%d -> %s:%d",
				event.EventType,
				flowData.SourceIP, event.SrcPort,
				flowData.DestIP, event.DstPort)
		}
	}
}

// GetFlowRingBuffer 返回流事件的 ring buffer reader
// 这允许外部组件 (如 flow.Collector) 读取流事件
func (dp *DataPlane) GetFlowRingBuffer() *ringbuf.Reader {
	return dp.rbReader
}

// GetProcessRingBuffer returns process events ring buffer map
// This allows external components (like ProcessMonitor) to create ring buffer reader (Issue #48)
func (dp *DataPlane) GetProcessRingBuffer() *ebpf.Map {
	maps, err := dp.manager.GetMaps()
	if err != nil {
		log.Debugf("Failed to get maps: %v", err)
		return nil
	}
	return maps.ProcessEventsRB
}

// GetProcessInfoMap returns process info map for cache queries (Issue #46)
func (dp *DataPlane) GetProcessInfoMap() *ebpf.Map {
	maps, err := dp.manager.GetMaps()
	if err != nil {
		log.Debugf("Failed to get maps: %v", err)
		return nil
	}
	return maps.ProcessInfoMap
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

// GetProtocolOffsetMap 返回协议偏移 map 供索引策略管理器访问
func (dp *DataPlane) GetProtocolOffsetMap() *ebpf.Map {
	maps, err := dp.manager.GetMaps()
	if err != nil {
		log.Debugf("Failed to get maps: %v", err)
		return nil
	}
	return maps.ProtocolOffsetMap
}

// GetMode 返回当前数据平面模式
func (dp *DataPlane) GetMode() DataPlaneMode {
	return dp.manager.GetMode()
}

// GetManager 返回底层的 Manager (用于高级操作)
func (dp *DataPlane) GetManager() *Manager {
	return dp.manager
}

// GetMaps returns all eBPF maps from the data plane
func (dp *DataPlane) GetMaps() (*DataPlaneMaps, error) {
	return dp.manager.GetMaps()
}

// EnableSessionTimeout enables and starts the session timeout manager
// This should be called after the data plane is initialized
func (dp *DataPlane) EnableSessionTimeout(config session.SessionTimeoutConfig) error {
	// Get session map
	sessionMap := dp.GetSessionMap()
	if sessionMap == nil {
		return fmt.Errorf("session map not available")
	}

	// Create timeout manager
	dp.timeoutManager = session.NewTimeoutManager(sessionMap, config)

	// Start timeout manager
	if err := dp.timeoutManager.Start(); err != nil {
		return fmt.Errorf("failed to start timeout manager: %w", err)
	}

	log.Info("✓ Session timeout manager enabled")
	return nil
}

// GetTimeoutStats returns session timeout statistics
// Returns empty stats if timeout manager is not enabled
func (dp *DataPlane) GetTimeoutStats() session.SessionTimeoutStats {
	if dp.timeoutManager == nil {
		return session.SessionTimeoutStats{}
	}
	return dp.timeoutManager.GetStats()
}
