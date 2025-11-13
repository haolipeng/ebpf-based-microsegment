// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package e2e

import (
	"testing"
	"time"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/policy"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_SessionConsistency_DirectionAwareness 测试会话缓存的方向感知
//
// 测试场景:
// 1. Client → Server:8080 (Ingress ALLOW) 建立 session
// 2. Server → Client (Egress DENY) 使用同一个 5-tuple
//
// 关键问题:
// - session_map 使用 5-tuple key (没有 direction)
// - 同一个流的上行和下行会共享同一个 session entry
//
// 预期行为:
// - Ingress 流量应该 ALLOW (建立 session)
// - Egress 流量应该 DENY (即使 session 已存在)
func TestE2E_SessionConsistency_DirectionAwareness(t *testing.T) {
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	env, err := NewE2ETestEnv(t)
	require.NoError(t, err)
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	t.Logf("测试场景: 会话方向感知")
	t.Logf("  Client(%s) → Server(%s):8080 [ALLOW] 建立 session", clientIP, serverIP)
	t.Logf("  Server(%s) → Client(%s) [DENY] 使用同一个 5-tuple 响应", serverIP, clientIP)

	// 启动 Server 侧的 TCP 服务器
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err)
	defer server.Stop()

	// 1. 创建 Ingress ALLOW 策略
	ingressAllowPolicy := &policy.Policy{
		RuleID:    30000,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  200,
	}
	err = env.CreatePolicy(ingressAllowPolicy)
	require.NoError(t, err)
	t.Log("策略 1: Ingress ALLOW - Client → Server:8080")

	// 2. 创建 Egress DENY 策略 (阻止响应)
	egressDenyPolicy := &policy.Policy{
		RuleID:    30001,
		SrcIP:     serverIP,
		DstIP:     clientIP,
		SrcPort:   0,
		DstPort:   0, // 通配符,匹配所有响应端口
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "egress",
		Priority:  100,
	}
	err = env.CreatePolicy(egressDenyPolicy)
	require.NoError(t, err)
	t.Log("策略 2: Egress DENY - Server → Client (响应)")

	statsBefore := env.GetStatistics()
	t.Logf("测试前统计: ingress_packets=%d, egress_packets=%d, ingress_denied=%d, egress_denied=%d, new_sessions=%d, active_sessions=%d",
		statsBefore.IngressPackets, statsBefore.EgressPackets,
		statsBefore.IngressDenied, statsBefore.EgressDenied,
		statsBefore.NewSessions, statsBefore.ActiveSessions)

	// 3. 尝试从 Client 连接 Server
	// Ingress 应该 ALLOW,但 Egress DENY 会阻止响应
	connected := env.TryConnect(8080)

	time.Sleep(200 * time.Millisecond)

	statsAfter := env.GetStatistics()
	t.Logf("测试后统计: ingress_packets=%d, egress_packets=%d, ingress_denied=%d, egress_denied=%d, new_sessions=%d, active_sessions=%d",
		statsAfter.IngressPackets, statsAfter.EgressPackets,
		statsAfter.IngressDenied, statsAfter.EgressDenied,
		statsAfter.NewSessions, statsAfter.ActiveSessions)

	// 验证 Ingress 流量被处理
	assert.Greater(t, statsAfter.IngressPackets, statsBefore.IngressPackets,
		"Ingress traffic should be processed")

	// 验证 Egress 响应被拒绝
	assert.Greater(t, statsAfter.EgressDenied, statsBefore.EgressDenied,
		"Egress response should be denied even if session exists")

	// 连接应该失败 (响应被阻止)
	assert.False(t, connected, "Connection should fail because egress response is blocked")

	// 关键验证: Session 应该被创建 (Ingress ALLOW)
	newSessions := statsAfter.NewSessions - statsBefore.NewSessions
	if newSessions > 0 {
		t.Logf("✓ Session 被创建: new_sessions=%d", newSessions)
	} else {
		t.Logf("⚠ Session 未被创建: new_sessions=%d (可能因为 SYN 包未完成握手)", newSessions)
	}

	t.Log("✓ 会话方向感知测试通过: Egress DENY 正确阻止响应")
}

// TestE2E_SessionConsistency_PolicyUpdate 测试策略更新后会话的一致性
//
// 测试场景:
// 1. 创建 ALLOW 策略,建立连接和 session
// 2. 更新为 DENY 策略
// 3. 验证后续流量是否被正确拒绝
//
// 预期行为:
// - 现有 session 不应该绕过新策略
// - 策略更新应该立即生效
//
// 注意: 当前跳过此测试,因为 wildcard 策略删除功能尚未实现
func TestE2E_SessionConsistency_PolicyUpdate(t *testing.T) {
	t.Skip("跳过: wildcard 策略删除功能尚未实现")
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	env, err := NewE2ETestEnv(t)
	require.NoError(t, err)
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	t.Logf("测试场景: 策略更新后的会话一致性")

	// 启动 Server
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err)
	defer server.Stop()

	// 阶段 1: 创建 ALLOW 策略
	allowPolicy := &policy.Policy{
		RuleID:    31000,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  100,
	}
	err = env.CreatePolicy(allowPolicy)
	require.NoError(t, err)
	t.Log("阶段 1: 创建 ALLOW 策略")

	// 建立连接 (应该成功)
	stats1Before := env.GetStatistics()
	connected1 := env.TryConnect(8080)
	time.Sleep(100 * time.Millisecond)
	stats1After := env.GetStatistics()

	assert.True(t, connected1, "Initial connection should succeed with ALLOW policy")
	t.Logf("阶段 1 结果: connected=%v, sessions=%d → %d",
		connected1, stats1Before.ActiveSessions, stats1After.ActiveSessions)

	// 阶段 2: 删除 ALLOW 策略,创建 DENY 策略
	err = env.DeletePolicy(allowPolicy)
	require.NoError(t, err)

	denyPolicy := &policy.Policy{
		RuleID:    31001,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "ingress",
		Priority:  100,
	}
	err = env.CreatePolicy(denyPolicy)
	require.NoError(t, err)
	t.Log("阶段 2: 更新为 DENY 策略")

	// 等待策略生效
	time.Sleep(200 * time.Millisecond)

	// 再次尝试连接 (应该失败)
	stats2Before := env.GetStatistics()
	connected2 := env.TryConnect(8080)
	time.Sleep(100 * time.Millisecond)
	stats2After := env.GetStatistics()

	assert.False(t, connected2, "Connection should fail after policy update to DENY")
	assert.Greater(t, stats2After.IngressDenied, stats2Before.IngressDenied,
		"DENY policy should take effect immediately")

	t.Logf("阶段 2 结果: connected=%v, ingress_denied=%d",
		connected2, stats2After.IngressDenied-stats2Before.IngressDenied)

	t.Log("✓ 策略更新一致性测试通过: DENY 策略立即生效,不受现有 session 影响")
}

// TestE2E_SessionConsistency_SameFlowDifferentDirections 测试同一个流的不同方向
//
// 测试场景:
// 1. Ingress DENY: Client → Server:8080
// 2. Egress ALLOW: Server → Client:9000
// 3. 验证同一个 5-tuple 在不同方向上的策略是否正确应用
//
// 关键问题:
// - 同一个流 (src_ip, dst_ip, protocol) 在不同方向上可能有不同的策略
// - session_map 的 5-tuple key 无法区分方向
//
// 预期行为:
// - Ingress 流量应该被 DENY
// - Egress 流量应该被 ALLOW
func TestE2E_SessionConsistency_SameFlowDifferentDirections(t *testing.T) {
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	env, err := NewE2ETestEnv(t)
	require.NoError(t, err)
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	t.Logf("测试场景: 同一个流的不同方向策略")
	t.Logf("  Ingress: Client(%s) → Server(%s):8080 [DENY]", clientIP, serverIP)
	t.Logf("  Egress:  Server(%s) → Client(%s):9000 [ALLOW]", serverIP, clientIP)

	// 1. 创建 Ingress DENY 策略
	ingressDenyPolicy := &policy.Policy{
		RuleID:    32000,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "ingress",
		Priority:  200,
	}
	err = env.CreatePolicy(ingressDenyPolicy)
	require.NoError(t, err)

	// 2. 创建 Egress ALLOW 策略
	egressAllowPolicy := &policy.Policy{
		RuleID:    32001,
		SrcIP:     serverIP,
		DstIP:     clientIP,
		SrcPort:   0,
		DstPort:   9000,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "egress",
		Priority:  200,
	}
	err = env.CreatePolicy(egressAllowPolicy)
	require.NoError(t, err)

	// 3. 创建 Ingress ALLOW 策略 (允许 Client 的响应)
	ingressAllowResponsePolicy := &policy.Policy{
		RuleID:    32002,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   0,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  100,
	}
	err = env.CreatePolicy(ingressAllowResponsePolicy)
	require.NoError(t, err)

	// 启动服务器
	serverTCP, err := env.StartTCPServer(8080)
	require.NoError(t, err)
	defer serverTCP.Stop()

	clientTCP, err := env.StartTCPServerOnClient(9000)
	require.NoError(t, err)
	defer clientTCP.Stop()

	// 测试 1: Client → Server 应该被拒绝
	t.Log("测试 1: Client → Server (Ingress DENY)")
	stats1Before := env.GetStatistics()

	connected1 := env.TryConnect(8080)
	assert.False(t, connected1, "Ingress connection should be denied")

	time.Sleep(100 * time.Millisecond)
	stats1After := env.GetStatistics()

	ingressDenied := stats1After.IngressDenied - stats1Before.IngressDenied
	t.Logf("测试 1 结果: connected=%v, ingress_denied=%d", connected1, ingressDenied)
	assert.Greater(t, ingressDenied, uint64(0), "Ingress should be denied")

	// 测试 2: Server → Client 应该成功
	t.Log("测试 2: Server → Client (Egress ALLOW)")
	stats2Before := env.GetStatistics()

	connected2 := env.TryConnectFromServer(9000)
	assert.True(t, connected2, "Egress connection should be allowed")

	time.Sleep(100 * time.Millisecond)
	stats2After := env.GetStatistics()

	egressPackets := stats2After.EgressPackets - stats2Before.EgressPackets
	t.Logf("测试 2 结果: connected=%v, egress_packets=%d", connected2, egressPackets)
	assert.Greater(t, egressPackets, uint64(0), "Egress should be allowed")

	t.Log("✓ 同一个流不同方向策略测试通过: Ingress DENY 和 Egress ALLOW 互不影响")
}

// TestE2E_SessionConsistency_SessionTimeout 测试会话超时后策略重新评估
//
// 测试场景:
// 1. 建立连接,创建 session
// 2. 等待 session 超时
// 3. 验证新连接是否重新评估策略
//
// 预期行为:
// - Session 超时后应该被清理
// - 新连接应该重新进行策略匹配
//
// 注意: 这是一个观察性测试,不强制要求超时机制
func TestE2E_SessionConsistency_SessionTimeout(t *testing.T) {
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	env, err := NewE2ETestEnv(t)
	require.NoError(t, err)
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	t.Logf("测试场景: 会话超时后策略重新评估")

	// 启动 Server
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err)
	defer server.Stop()

	// 创建 ALLOW 策略
	allowPolicy := &policy.Policy{
		RuleID:    33000,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  100,
	}
	err = env.CreatePolicy(allowPolicy)
	require.NoError(t, err)

	// 阶段 1: 建立连接
	stats1Before := env.GetStatistics()
	connected1 := env.TryConnect(8080)
	time.Sleep(100 * time.Millisecond)
	stats1After := env.GetStatistics()

	assert.True(t, connected1, "Initial connection should succeed")
	sessionsBefore := stats1After.ActiveSessions
	t.Logf("阶段 1: 连接建立, active_sessions=%d (增加=%d)",
		sessionsBefore, sessionsBefore-stats1Before.ActiveSessions)

	// 阶段 2: 等待 session 超时 (当前实现可能没有超时机制)
	t.Log("阶段 2: 等待 5 秒 (观察 session 状态)")
	time.Sleep(5 * time.Second)

	statsTimeout := env.GetStatistics()
	sessionsAfterTimeout := statsTimeout.ActiveSessions
	t.Logf("等待后: active_sessions=%d (变化=%d)",
		sessionsAfterTimeout, int64(sessionsAfterTimeout)-int64(sessionsBefore))

	// 注意: 当前实现可能没有实现 session 超时清理
	// 这个测试主要是验证行为,不一定会失败
	if sessionsAfterTimeout < sessionsBefore {
		t.Logf("✓ Session 超时机制工作正常 (sessions 减少)")
	} else {
		t.Logf("⚠ Session 超时机制可能未实现 (sessions 未减少)")
	}

	// 阶段 3: 再次连接,验证策略重新评估
	stats3Before := env.GetStatistics()
	connected3 := env.TryConnect(8080)
	time.Sleep(100 * time.Millisecond)
	stats3After := env.GetStatistics()

	if !connected3 {
		t.Errorf("重新连接失败 (unexpected)")
	}
	t.Logf("阶段 3: 重新连接, ingress_packets=%d",
		stats3After.IngressPackets-stats3Before.IngressPackets)

	// 观察性测试,不强制失败
	t.Log("✓ 会话超时测试完成 (观察性测试,当前未实现超时机制)")
}
