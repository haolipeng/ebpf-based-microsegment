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
	SessionMap        *ebpf.Map      // 会话表
	PolicyMap         *ebpf.Map      // 策略表
	WildcardPolicyMap *ebpf.Map      // 通配符策略表
	StatsMap          *ebpf.Map      // 统计表
	FlowEventsRB      *ebpf.Map      // 流事件 Ring Buffer
}

// Ensure DataPlane implements DataPlaneInterface
var _ DataPlaneInterface = (*DataPlane)(nil)
