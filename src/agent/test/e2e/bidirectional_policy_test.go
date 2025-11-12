// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package e2e

import (
	"testing"
	"time"

	"github.com/ebpf-microsegment/src/agent/pkg/policy"
	"github.com/ebpf-microsegment/src/agent/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_BidirectionalPolicy_IngressAllowEgressDeny 测试 Ingress 允许但 Egress 拒绝的场景
//
// 场景描述:
// - Ingress ALLOW: 允许 Client → Server:8080
// - Egress DENY: 拒绝 Server → Client (响应流量)
//
// 预期行为:
// - Client 可以连接到 Server (Ingress 允许)
// - 但 Server 无法发送响应 (Egress 拒绝)
// - 从 Client 角度看,连接会超时或失败
func TestE2E_BidirectionalPolicy_IngressAllowEgressDeny(t *testing.T) {
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	env, err := NewE2ETestEnv(t)
	require.NoError(t, err, "Failed to create test environment")
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	t.Logf("场景: Ingress ALLOW + Egress DENY")
	t.Logf("  Client(%s) → Server(%s):8080 [ALLOW]", clientIP, serverIP)
	t.Logf("  Server(%s) → Client(%s) [DENY]", serverIP, clientIP)

	// 启动 Server 侧的 TCP 服务器
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err, "Failed to start TCP server")
	defer server.Stop()

	// 1. 创建 Ingress ALLOW 策略 (Client → Server)
	ingressAllowPolicy := &policy.Policy{
		RuleID:    8000,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  200,  // 更高优先级 (具体端口)
	}
	err = env.CreatePolicy(ingressAllowPolicy)
	require.NoError(t, err, "Failed to create ingress allow policy")

	// 2. 创建 Egress DENY 策略 (Server → Client,阻止响应)
	egressDenyPolicy := &policy.Policy{
		RuleID:    8001,
		SrcIP:     serverIP,
		DstIP:     clientIP,
		SrcPort:   0,  // 使用通配符,确保进入 wildcard_policy_map
		DstPort:   0,
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "egress",
		Priority:  100,
	}
	err = env.CreatePolicy(egressDenyPolicy)
	require.NoError(t, err, "Failed to create egress deny policy")

	// 记录统计
	statsBefore := env.GetStatistics()
	t.Logf("统计 (测试前): ingress_packets=%d, egress_denied=%d",
		statsBefore.IngressPackets, statsBefore.EgressDenied)

	// 3. 尝试从 Client 连接 Server
	// Ingress 应该允许,但 Egress 会阻止响应
	// 注意: TryConnect 会尝试建立连接并等待响应,因为响应被阻止,会超时失败
	connected := env.TryConnect(8080)

	// 等待统计更新
	time.Sleep(200 * time.Millisecond)

	statsAfter := env.GetStatistics()
	t.Logf("统计 (测试后): ingress_packets=%d, egress_denied=%d, denied_packets=%d",
		statsAfter.IngressPackets, statsAfter.EgressDenied, statsAfter.DeniedPackets)

	// 验证: Ingress 流量应该通过
	assert.Greater(t, statsAfter.IngressPackets, statsBefore.IngressPackets,
		"Ingress traffic should be processed")

	// 验证: Egress 流量应该被拒绝
	assert.Greater(t, statsAfter.EgressDenied, statsBefore.EgressDenied,
		"Egress response should be denied")

	// 从 Client 角度,连接应该失败 (因为响应被阻止)
	assert.False(t, connected,
		"Connection should fail because egress response is blocked")

	t.Log("✓ Ingress ALLOW + Egress DENY: 策略正确执行")
}

// TestE2E_BidirectionalPolicy_IngressDenyEgressAllow 测试 Ingress 拒绝但 Egress 允许的场景
//
// 场景描述:
// - Ingress DENY: 拒绝 Client → Server
// - Egress ALLOW: 允许 Server → Client (主动出站)
//
// 预期行为:
// - Client 无法连接到 Server (Ingress 拒绝)
// - Server 可以主动连接 Client (Egress 允许)
func TestE2E_BidirectionalPolicy_IngressDenyEgressAllow(t *testing.T) {
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	env, err := NewE2ETestEnv(t)
	require.NoError(t, err, "Failed to create test environment")
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	t.Logf("场景: Ingress DENY + Egress ALLOW")
	t.Logf("  Client(%s) → Server(%s):8080 [DENY]", clientIP, serverIP)
	t.Logf("  Server(%s) → Client(%s):9000 [ALLOW]", serverIP, clientIP)

	// 1. 创建 Ingress DENY 策略 (Client → Server)
	ingressDenyPolicy := &policy.Policy{
		RuleID:    9000,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "ingress",
		Priority:  200,  // 更高优先级,确保覆盖通配符策略
	}
	err = env.CreatePolicy(ingressDenyPolicy)
	require.NoError(t, err, "Failed to create ingress deny policy")

	// 2. 创建 Egress ALLOW 策略 (Server → Client)
	egressAllowPolicy := &policy.Policy{
		RuleID:    9001,
		SrcIP:     serverIP,
		DstIP:     clientIP,
		SrcPort:   0,
		DstPort:   9000,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "egress",
		Priority:  200,  // 更高优先级 (具体端口)
	}
	err = env.CreatePolicy(egressAllowPolicy)
	require.NoError(t, err, "Failed to create egress allow policy")

	// 3. 创建 Ingress ALLOW 策略 (允许 Client 的响应回来)
	ingressAllowResponsePolicy := &policy.Policy{
		RuleID:    9002,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,  // 使用通配符,确保进入 wildcard_policy_map
		DstPort:   0,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  100,
	}
	err = env.CreatePolicy(ingressAllowResponsePolicy)
	require.NoError(t, err, "Failed to create ingress allow response policy")

	// 启动 Server 侧的 TCP 服务器 (用于测试 Ingress DENY)
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err, "Failed to start TCP server")
	defer server.Stop()

	// 启动 Client 侧的 TCP 服务器 (用于测试 Egress ALLOW)
	clientServer, err := env.StartTCPServerOnClient(9000)
	require.NoError(t, err, "Failed to start TCP server on client")
	defer clientServer.Stop()

	// 测试 1: Client → Server 应该被拒绝 (Ingress DENY)
	t.Log("测试 1: Client → Server (应该被拒绝)")
	stats1Before := env.GetStatistics()

	connected1 := env.TryConnect(8080)
	assert.False(t, connected1, "Ingress connection should be denied")

	time.Sleep(100 * time.Millisecond)
	stats1After := env.GetStatistics()

	t.Logf("测试 1 统计: ingress_denied=%d",
		stats1After.IngressDenied-stats1Before.IngressDenied)
	assert.Greater(t, stats1After.IngressDenied, stats1Before.IngressDenied,
		"Ingress should be denied")

	// 测试 2: Server → Client 应该成功 (Egress ALLOW)
	t.Log("测试 2: Server → Client (应该成功)")
	stats2Before := env.GetStatistics()

	connected2 := env.TryConnectFromServer(9000)
	assert.True(t, connected2, "Egress connection should be allowed")

	time.Sleep(100 * time.Millisecond)
	stats2After := env.GetStatistics()

	t.Logf("测试 2 统计: egress_packets=%d",
		stats2After.EgressPackets-stats2Before.EgressPackets)
	assert.Greater(t, stats2After.EgressPackets, stats2Before.EgressPackets,
		"Egress should be allowed")

	t.Log("✓ Ingress DENY + Egress ALLOW: 策略正确执行")
}

// TestE2E_BidirectionalPolicy_BothAllow 测试双向都允许的场景
//
// 场景描述:
// - Ingress ALLOW: 允许 Client → Server
// - Egress ALLOW: 允许 Server → Client
//
// 预期行为:
// - 双向通信都应该成功
func TestE2E_BidirectionalPolicy_BothAllow(t *testing.T) {
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	env, err := NewE2ETestEnv(t)
	require.NoError(t, err, "Failed to create test environment")
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	t.Logf("场景: 双向 ALLOW")

	// 1. 创建 Ingress ALLOW 策略
	ingressAllowPolicy := &policy.Policy{
		RuleID:    10000,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  200,  // 更高优先级 (具体端口)
	}
	err = env.CreatePolicy(ingressAllowPolicy)
	require.NoError(t, err)

	// 2. 创建 Egress ALLOW 策略 (允许响应)
	egressAllowPolicy := &policy.Policy{
		RuleID:    10001,
		SrcIP:     serverIP,
		DstIP:     clientIP,
		SrcPort:   0,  // 使用通配符,确保进入 wildcard_policy_map
		DstPort:   0,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "egress",
		Priority:  100,
	}
	err = env.CreatePolicy(egressAllowPolicy)
	require.NoError(t, err)

	// 启动服务器
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err)
	defer server.Stop()

	statsBefore := env.GetStatistics()

	// 测试双向通信
	testData := []byte("Bidirectional test")
	err = env.SendTCPTraffic(8080, testData)
	assert.NoError(t, err, "Bidirectional communication should succeed")

	time.Sleep(200 * time.Millisecond)

	statsAfter := env.GetStatistics()
	t.Logf("统计: ingress_packets=%d, egress_packets=%d, allowed=%d",
		statsAfter.IngressPackets-statsBefore.IngressPackets,
		statsAfter.EgressPackets-statsBefore.EgressPackets,
		statsAfter.AllowedPackets-statsBefore.AllowedPackets)

	// 验证双向流量都通过
	assert.Greater(t, statsAfter.IngressPackets, statsBefore.IngressPackets,
		"Ingress traffic should be processed")
	assert.Greater(t, statsAfter.EgressPackets, statsBefore.EgressPackets,
		"Egress traffic should be processed")
	assert.Greater(t, statsAfter.AllowedPackets, statsBefore.AllowedPackets,
		"Traffic should be allowed")

	t.Log("✓ 双向 ALLOW: 通信正常")
}

// TestE2E_BidirectionalPolicy_BothDeny 测试双向都拒绝的场景
//
// 场景描述:
// - Ingress DENY: 拒绝 Client → Server
// - Egress DENY: 拒绝 Server → Client
//
// 预期行为:
// - 双向通信都应该失败
func TestE2E_BidirectionalPolicy_BothDeny(t *testing.T) {
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	env, err := NewE2ETestEnv(t)
	require.NoError(t, err, "Failed to create test environment")
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	t.Logf("场景: 双向 DENY")

	// 1. 创建 Ingress DENY 策略
	ingressDenyPolicy := &policy.Policy{
		RuleID:    11000,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "ingress",
		Priority:  200,  // 更高优先级 (具体端口)
	}
	err = env.CreatePolicy(ingressDenyPolicy)
	require.NoError(t, err)

	// 2. 创建 Egress DENY 策略
	egressDenyPolicy := &policy.Policy{
		RuleID:    11001,
		SrcIP:     serverIP,
		DstIP:     clientIP,
		SrcPort:   0,
		DstPort:   9100,
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "egress",
		Priority:  200,  // 更高优先级 (具体端口)
	}
	err = env.CreatePolicy(egressDenyPolicy)
	require.NoError(t, err)

	// 启动服务器
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err)
	defer server.Stop()

	clientServer, err := env.StartTCPServerOnClient(9100)
	require.NoError(t, err)
	defer clientServer.Stop()

	// 测试 1: Client → Server 应该失败
	t.Log("测试 1: Client → Server (应该被拒绝)")
	stats1Before := env.GetStatistics()

	connected1 := env.TryConnect(8080)
	assert.False(t, connected1, "Ingress should be denied")

	time.Sleep(100 * time.Millisecond)
	stats1After := env.GetStatistics()

	t.Logf("测试 1 统计: ingress_denied=%d",
		stats1After.IngressDenied-stats1Before.IngressDenied)
	assert.Greater(t, stats1After.IngressDenied, stats1Before.IngressDenied,
		"Ingress should be denied")

	// 测试 2: Server → Client 应该失败
	t.Log("测试 2: Server → Client (应该被拒绝)")
	stats2Before := env.GetStatistics()

	connected2 := env.TryConnectFromServer(9100)
	assert.False(t, connected2, "Egress should be denied")

	time.Sleep(100 * time.Millisecond)
	stats2After := env.GetStatistics()

	t.Logf("测试 2 统计: egress_denied=%d",
		stats2After.EgressDenied-stats2Before.EgressDenied)
	assert.Greater(t, stats2After.EgressDenied, stats2Before.EgressDenied,
		"Egress should be denied")

	t.Log("✓ 双向 DENY: 所有流量都被阻止")
}
