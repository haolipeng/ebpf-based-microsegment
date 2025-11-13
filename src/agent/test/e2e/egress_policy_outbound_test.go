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

// TestE2E_EgressOutboundDeny 测试 Egress DENY 策略阻止 Server 主动发起的出站连接
//
// 测试拓扑:
//   [Client NS]  <---  [Server NS with eBPF]
//   TCP Server        主动发起连接 (应被拒绝)
//
// 场景描述:
// - eBPF 程序附加在 Server 的 veth-server 接口上
// - Client 运行 TCP Server 监听端口
// - Server 尝试主动连接 Client (Egress 流量)
// - Egress DENY 策略应该阻止这个连接
func TestE2E_EgressOutboundDeny(t *testing.T) {
	// 跳过非 root 测试
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	// 创建测试环境
	env, err := NewE2ETestEnv(t)
	require.NoError(t, err, "Failed to create test environment")
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	t.Logf("Test topology: Client(%s) <--- Server(%s)", clientIP, serverIP)

	// 在 Client 侧启动 TCP Server
	server, err := env.StartTCPServerOnClient(9000)
	require.NoError(t, err, "Failed to start TCP server on client")
	defer server.Stop()

	// 创建 Egress DENY 策略
	// 阻止 Server (src) 主动连接 Client (dst)
	egressDenyPolicy := &policy.Policy{
		RuleID:    5000,
		SrcIP:     serverIP,  // Server 作为源
		DstIP:     clientIP,  // Client 作为目标
		SrcPort:   0,         // 任意源端口
		DstPort:   9000,      // Client 监听的端口
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "egress",  // Egress 方向
		Priority:  100,
	}

	err = env.CreatePolicy(egressDenyPolicy)
	require.NoError(t, err, "Failed to create egress deny policy")

	// 记录统计
	statsBefore := env.GetStatistics()
	t.Logf("统计 (策略创建前): egress_denied=%d, egress_packets=%d",
		statsBefore.EgressDenied, statsBefore.EgressPackets)

	// 从 Server 尝试连接 Client
	// 这应该被 Egress DENY 策略阻止
	connected := env.TryConnectFromServer(9000)
	assert.False(t, connected, "Egress connection should be blocked by policy")

	// 等待统计更新
	time.Sleep(200 * time.Millisecond)

	// 验证统计
	statsAfter := env.GetStatistics()
	t.Logf("统计 (测试后): egress_denied=%d, egress_packets=%d, denied_packets=%d",
		statsAfter.EgressDenied, statsAfter.EgressPackets, statsAfter.DeniedPackets)

	// 验证 Egress 流量被拒绝
	assert.Greater(t, statsAfter.EgressDenied, statsBefore.EgressDenied,
		"Egress denied counter should increase")

	// 验证总拒绝数增加
	assert.Greater(t, statsAfter.DeniedPackets, statsBefore.DeniedPackets,
		"Total denied packets should increase")

	t.Logf("✓ Egress DENY 策略成功阻止了 Server 主动发起的出站连接")
	t.Logf("  Egress denied: %d packets", statsAfter.EgressDenied-statsBefore.EgressDenied)
}

// TestE2E_EgressOutboundAllow 测试 Egress ALLOW 策略允许 Server 主动发起的出站连接
//
// 场景描述:
// 1. 创建 Egress ALLOW 策略
// 2. Server 主动连接 Client
// 3. 连接应该成功
// 4. 验证 egress_packets 统计增加,egress_denied 不增加
func TestE2E_EgressOutboundAllow(t *testing.T) {
	// 跳过非 root 测试
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	// 创建测试环境
	env, err := NewE2ETestEnv(t)
	require.NoError(t, err, "Failed to create test environment")
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	t.Logf("Test topology: Client(%s) <--- Server(%s)", clientIP, serverIP)

	// 在 Client 侧启动 TCP Server
	server, err := env.StartTCPServerOnClient(9001)
	require.NoError(t, err, "Failed to start TCP server on client")
	defer server.Stop()

	// 创建 Egress ALLOW 策略
	// 允许 Server 主动连接 Client
	egressAllowPolicy := &policy.Policy{
		RuleID:    6000,
		SrcIP:     serverIP,
		DstIP:     clientIP,
		SrcPort:   0,
		DstPort:   9001,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "egress",
		Priority:  100,
	}

	err = env.CreatePolicy(egressAllowPolicy)
	require.NoError(t, err, "Failed to create egress allow policy")

	// 还需要创建 Ingress ALLOW 策略,允许 Client 的响应返回
	// (Client 的响应在 Server 看来是 Ingress)
	ingressAllowPolicy := &policy.Policy{
		RuleID:    6001,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   9001,
		DstPort:   0,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  100,
	}

	err = env.CreatePolicy(ingressAllowPolicy)
	require.NoError(t, err, "Failed to create ingress allow policy")

	// 记录统计
	statsBefore := env.GetStatistics()
	t.Logf("统计 (策略创建前): egress_packets=%d, egress_denied=%d",
		statsBefore.EgressPackets, statsBefore.EgressDenied)

	// 从 Server 连接 Client
	// 应该成功 (Egress ALLOW)
	connected := env.TryConnectFromServer(9001)
	assert.True(t, connected, "Egress connection should be allowed by policy")

	// 等待统计更新
	time.Sleep(200 * time.Millisecond)

	// 验证统计
	statsAfter := env.GetStatistics()
	t.Logf("统计 (测试后): egress_packets=%d, egress_denied=%d, allowed_packets=%d",
		statsAfter.EgressPackets, statsAfter.EgressDenied, statsAfter.AllowedPackets)

	// 验证 Egress 流量被允许
	assert.Greater(t, statsAfter.EgressPackets, statsBefore.EgressPackets,
		"Egress packets counter should increase")

	// 验证没有 Egress 拒绝
	assert.Equal(t, statsBefore.EgressDenied, statsAfter.EgressDenied,
		"Egress denied counter should not increase")

	// 验证总允许数增加
	assert.Greater(t, statsAfter.AllowedPackets, statsBefore.AllowedPackets,
		"Allowed packets should increase")

	t.Logf("✓ Egress ALLOW 策略成功允许了 Server 主动发起的出站连接")
	t.Logf("  Egress packets: %d", statsAfter.EgressPackets-statsBefore.EgressPackets)
}

// TestE2E_EgressOutboundDefaultBehavior 测试没有 Egress 策略时 Server 主动出站连接的默认行为
//
// 场景描述:
// 1. 不创建任何 Egress 策略
// 2. 创建 Ingress ALLOW 策略 (允许响应回来)
// 3. Server 主动连接 Client
// 4. 验证默认行为 (预期: ALLOW,因为没有明确的 DENY)
func TestE2E_EgressOutboundDefaultBehavior(t *testing.T) {
	// 跳过非 root 测试
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	// 创建测试环境
	env, err := NewE2ETestEnv(t)
	require.NoError(t, err, "Failed to create test environment")
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	t.Logf("Test topology: Client(%s) <--- Server(%s)", clientIP, serverIP)

	// 在 Client 侧启动 TCP Server
	server, err := env.StartTCPServerOnClient(9002)
	require.NoError(t, err, "Failed to start TCP server on client")
	defer server.Stop()

	// 创建 Ingress ALLOW 策略 (允许 Client 的响应)
	ingressAllowPolicy := &policy.Policy{
		RuleID:    7000,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   9002,
		DstPort:   0,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  100,
	}

	err = env.CreatePolicy(ingressAllowPolicy)
	require.NoError(t, err, "Failed to create ingress allow policy")

	// 记录统计
	statsBefore := env.GetStatistics()
	t.Logf("统计 (测试前): egress_packets=%d, egress_denied=%d",
		statsBefore.EgressPackets, statsBefore.EgressDenied)

	// 从 Server 连接 Client
	connected := env.TryConnectFromServer(9002)

	// 等待统计更新
	time.Sleep(200 * time.Millisecond)

	statsAfter := env.GetStatistics()
	t.Logf("统计 (测试后): egress_packets=%d, egress_denied=%d",
		statsAfter.EgressPackets, statsAfter.EgressDenied)

	// 文档化默认行为
	if connected {
		t.Log("✓ 默认行为: ALLOW (没有 Egress 策略 = 允许出站)")
		assert.Greater(t, statsAfter.EgressPackets, statsBefore.EgressPackets,
			"Egress packets should be processed")
		assert.Equal(t, statsBefore.EgressDenied, statsAfter.EgressDenied,
			"Egress should not be denied by default")
	} else {
		t.Log("✗ 默认行为: DENY (没有 Egress 策略 = 拒绝出站)")
		t.Log("  Note: This is unexpected. Default-deny behavior should be documented.")
		assert.Greater(t, statsAfter.EgressDenied, statsBefore.EgressDenied,
			"Egress should be denied by default if connection failed")
	}
}
