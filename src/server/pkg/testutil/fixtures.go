package testutil

import (
	"fmt"
	"time"

	agentpb "github.com/ebpf-microsegment/src/proto/agent"
	commonpb "github.com/ebpf-microsegment/src/proto/common"
	flowpb "github.com/ebpf-microsegment/src/proto/flow"
	policypb "github.com/ebpf-microsegment/src/proto/policy"
)

// MockFlowEvent 创建一个模拟的流事件
// 用于单元测试和集成测试
func MockFlowEvent(opts ...FlowEventOption) *flowpb.FlowEvent {
	// 默认值
	event := &flowpb.FlowEvent{
		SrcIp:        IPToFixed32("10.0.1.10"),
		DstIp:        IPToFixed32("10.0.2.20"),
		SrcPort:      8080,
		DstPort:      443,
		Protocol:     commonpb.Protocol_TCP,
		EventType:    commonpb.FlowEventType_FLOW_NEW,
		Direction:    commonpb.FlowDirection_EGRESS,
		PacketCount:  100,
		ByteCount:    15000,
		TimestampNs:  uint64(time.Now().UnixNano()),
		PolicyId:     1,
		PolicyAction: commonpb.PolicyAction_ALLOW,
		State:        commonpb.FlowState_ESTABLISHED,
		AgentId:      "test-agent-001",
		SourceLabels: map[string]string{
			"app":  "web",
			"env":  "prod",
			"tier": "frontend",
		},
		DestLabels: map[string]string{
			"app":  "api",
			"env":  "prod",
			"tier": "backend",
		},
	}

	// 应用选项
	for _, opt := range opts {
		opt(event)
	}

	return event
}

// FlowEventOption 是用于自定义 FlowEvent 的函数选项
type FlowEventOption func(*flowpb.FlowEvent)

// WithSourceIP 设置源 IP 地址
func WithSourceIP(ip string) FlowEventOption {
	return func(e *flowpb.FlowEvent) {
		e.SrcIp = IPToFixed32(ip)
	}
}

// WithDestIP 设置目标 IP 地址
func WithDestIP(ip string) FlowEventOption {
	return func(e *flowpb.FlowEvent) {
		e.DstIp = IPToFixed32(ip)
	}
}

// WithPorts 设置源端口和目标端口
func WithPorts(srcPort, dstPort uint32) FlowEventOption {
	return func(e *flowpb.FlowEvent) {
		e.SrcPort = srcPort
		e.DstPort = dstPort
	}
}

// WithProtocol 设置协议类型
func WithProtocol(protocol commonpb.Protocol) FlowEventOption {
	return func(e *flowpb.FlowEvent) {
		e.Protocol = protocol
	}
}

// WithAgentID 设置 Agent ID
func WithAgentID(agentID string) FlowEventOption {
	return func(e *flowpb.FlowEvent) {
		e.AgentId = agentID
	}
}

// WithPolicyAction 设置策略动作
func WithPolicyAction(action commonpb.PolicyAction) FlowEventOption {
	return func(e *flowpb.FlowEvent) {
		e.PolicyAction = action
	}
}

// WithLabels 设置源和目标标签
func WithLabels(srcLabels, dstLabels map[string]string) FlowEventOption {
	return func(e *flowpb.FlowEvent) {
		e.SourceLabels = srcLabels
		e.DestLabels = dstLabels
	}
}

// MockPolicy 创建一个模拟的策略规则
func MockPolicy(opts ...PolicyOption) *policypb.Policy {
	now := time.Now().UnixNano()
	policy := &policypb.Policy{
		RuleId:      100,
		SrcIp:       "10.0.1.0/24",
		DstIp:       "10.0.2.0/24",
		SrcPort:     0, // any
		DstPort:     443,
		Protocol:    commonpb.Protocol_TCP,
		Action:      commonpb.PolicyAction_ALLOW,
		Priority:    10,
		CreatedAt:   now,
		UpdatedAt:   now,
		Description: "Test policy: Allow frontend to backend HTTPS",
		SourceLabels: map[string]string{
			"app":  "web",
			"tier": "frontend",
		},
		DestLabels: map[string]string{
			"app":  "api",
			"tier": "backend",
		},
	}

	for _, opt := range opts {
		opt(policy)
	}

	return policy
}

// PolicyOption 是用于自定义 Policy 的函数选项
type PolicyOption func(*policypb.Policy)

// WithRuleID 设置规则 ID
func WithRuleID(ruleID uint32) PolicyOption {
	return func(p *policypb.Policy) {
		p.RuleId = ruleID
	}
}

// WithPolicyCIDR 设置源和目标 CIDR
func WithPolicyCIDR(srcCIDR, dstCIDR string) PolicyOption {
	return func(p *policypb.Policy) {
		p.SrcIp = srcCIDR
		p.DstIp = dstCIDR
	}
}

// WithPolicyAction 设置策略动作（重用）
func WithPolicyActionPolicy(action commonpb.PolicyAction) PolicyOption {
	return func(p *policypb.Policy) {
		p.Action = action
	}
}

// WithPriority 设置优先级
func WithPriority(priority uint32) PolicyOption {
	return func(p *policypb.Policy) {
		p.Priority = priority
	}
}

// MockRegisterRequest 创建模拟的 Agent 注册请求
func MockRegisterRequest(agentID string, opts ...RegisterOption) *agentpb.RegisterRequest {
	req := &agentpb.RegisterRequest{
		AgentId:       agentID,
		Hostname:      fmt.Sprintf("host-%s", agentID),
		Version:       "1.0.0",
		Interface:     "eth0",
		IpAddresses:   []string{"10.0.1.10", "192.168.1.100"},
		Os:            "Linux",
		KernelVersion: "5.15.0-58-generic",
		StartTime:     time.Now().UnixNano(),
		Capabilities: &agentpb.AgentCapabilities{
			FlowTracking:       true,
			PolicyEnforcement:  true,
			LabelBasedPolicies: true,
			FlowStreaming:      true,
			PolicyStats:        true,
			Ipv6Support:        false,
		},
	}

	for _, opt := range opts {
		opt(req)
	}

	return req
}

// RegisterOption 是用于自定义 RegisterRequest 的函数选项
type RegisterOption func(*agentpb.RegisterRequest)

// WithHostname 设置主机名
func WithHostname(hostname string) RegisterOption {
	return func(r *agentpb.RegisterRequest) {
		r.Hostname = hostname
	}
}

// WithVersion 设置版本
func WithVersion(version string) RegisterOption {
	return func(r *agentpb.RegisterRequest) {
		r.Version = version
	}
}

// WithIPAddresses 设置 IP 地址列表
func WithIPAddresses(ips ...string) RegisterOption {
	return func(r *agentpb.RegisterRequest) {
		r.IpAddresses = ips
	}
}

// MockHeartbeatRequest 创建模拟的心跳请求
func MockHeartbeatRequest(agentID string) *agentpb.HeartbeatRequest {
	return &agentpb.HeartbeatRequest{
		AgentId:   agentID,
		Timestamp: time.Now().UnixNano(),
		Metrics: &agentpb.AgentMetrics{
			CpuUsage:         25.5,
			MemoryUsage:      512 * 1024 * 1024, // 512MB
			PacketsProcessed: 10000,
			ActiveSessions:   150,
			FlowsReported:    500,
			ActivePolicies:   10,
		},
	}
}

// MockPolicyStatsReport 创建模拟的策略统计报告
func MockPolicyStatsReport(agentID string, ruleID uint32) *policypb.PolicyStatsReport {
	now := time.Now()
	return &policypb.PolicyStatsReport{
		AgentId:   agentID,
		Timestamp: now.UnixNano(),
		TimeRange: &commonpb.TimeRange{
			StartTime: now.Add(-1 * time.Minute).UnixNano(),
			EndTime:   now.UnixNano(),
		},
		PolicyStats: []*policypb.PolicyStats{
			{
				RuleId:        ruleID,
				PacketCount:   1000,
				ByteCount:     150000,
				FlowCount:     50,
				HitCount:      1000,
				LastMatchTime: now.UnixNano(),
			},
		},
	}
}

// BatchMockFlowEvents 批量创建模拟流事件
// 用于性能测试和批量插入测试
func BatchMockFlowEvents(count int, agentID string) []*flowpb.FlowEvent {
	events := make([]*flowpb.FlowEvent, count)
	for i := 0; i < count; i++ {
		events[i] = MockFlowEvent(
			WithAgentID(agentID),
			WithSourceIP(fmt.Sprintf("10.0.1.%d", (i%250)+1)),
			WithDestIP(fmt.Sprintf("10.0.2.%d", (i%250)+1)),
			WithPorts(uint32(8000+i%1000), uint32(443)),
		)
	}
	return events
}

// BatchMockPolicies 批量创建模拟策略规则
func BatchMockPolicies(count int) []*policypb.Policy {
	policies := make([]*policypb.Policy, count)
	for i := 0; i < count; i++ {
		policies[i] = MockPolicy(
			WithRuleID(uint32(100 + i)),
			WithPriority(uint32(10 + i)),
		)
	}
	return policies
}

// IPToFixed32 将 IP 字符串转换为 fixed32
// 用于 FlowEvent 的 src_ip 和 dst_ip 字段
func IPToFixed32(ip string) uint32 {
	// 简化版本：仅支持 IPv4
	// 生产代码应该使用 net.ParseIP
	var a, b, c, d uint32
	fmt.Sscanf(ip, "%d.%d.%d.%d", &a, &b, &c, &d)
	return (a << 24) | (b << 16) | (c << 8) | d
}

// Fixed32ToIP 将 fixed32 转换为 IP 字符串
func Fixed32ToIP(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		(ip>>24)&0xFF,
		(ip>>16)&0xFF,
		(ip>>8)&0xFF,
		ip&0xFF,
	)
}
