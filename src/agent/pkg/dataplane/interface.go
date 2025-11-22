// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"github.com/cilium/ebpf"
)

// DataPlaneInterface defines the operations for data plane management.
// This interface is useful for testing and dependency injection.
type DataPlaneInterface interface {
	GetStatistics() Statistics
}

// DataPlaneMaps 封装所有数据平面 eBPF Map 的引用
type DataPlaneMaps struct {
	SessionMap        *ebpf.Map // 会话表
	PolicyMap         *ebpf.Map // 策略表
	WildcardPolicyMap *ebpf.Map // 通配符策略表
	ProtocolOffsetMap *ebpf.Map // 协议偏移表（用于索引查找）
	StatsMap          *ebpf.Map // 统计表
	FlowEventsRB      *ebpf.Map // 流事件 Ring Buffer
	ProcessEventsRB   *ebpf.Map // 进程事件 Ring Buffer
	ProcessInfoMap    *ebpf.Map // 进程信息缓存表
	ConntrackCacheMap *ebpf.Map // Conntrack缓存表（NAT支持）
	NATConfigMap      *ebpf.Map // NAT配置表
	NATStatsMap       *ebpf.Map // NAT统计表
	TimeoutConfigMap  *ebpf.Map // 超时配置表
	FragStateMap      *ebpf.Map // 分片状态表（分片跟踪）
	FragConfigMap     *ebpf.Map // 分片配置表
	FragStatsMap      *ebpf.Map // 分片统计表
}

// Ensure DataPlane implements DataPlaneInterface
var _ DataPlaneInterface = (*DataPlane)(nil)
