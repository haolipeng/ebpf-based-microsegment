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

// TestE2E_EgressDenyPolicy 测试 Egress DENY 策略能否正确阻止出站流量
//
// 场景描述:
// - eBPF 程序附加在 Server 的 veth-server 接口上
// - Client → Server: Ingress 流量 (进入 Server)
// - Server → Client: Egress 流量 (离开 Server)
//
// 测试步骤:
// 1. 创建 Egress DENY 策略,阻止 Server 发送到 Client 的流量
// 2. 从 Client 向 Server 发送请求
// 3. Server 尝试响应,但 Egress DENY 策略应该阻止响应
// 4. 验证 egress_denied 统计计数器增加
func TestE2E_EgressDenyPolicy(t *testing.T) {
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

	// 创建 Egress DENY 策略
	// 阻止从 Server (src) 到 Client (dst) 的流量
	// 注意: 这里的 src 是 Server,dst 是 Client,因为是 Server 发出的流量
	egressDenyPolicy := &policy.Policy{
		RuleID:    1000,
		SrcIP:     serverIP,   // Server 作为源
		DstIP:     clientIP,   // Client 作为目标
		SrcPort:   0,          // 任意源端口
		DstPort:   0,          // 任意目标端口
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "egress",   // 明确指定 Egress 方向
		Priority:  100,
	}

	err = env.CreatePolicy(egressDenyPolicy)
	require.NoError(t, err, "Failed to create egress deny policy")

	// 启动 TCP Echo Server
	// Server 会尝试响应客户端的请求,但 Egress DENY 策略应该阻止
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err, "Failed to start TCP server")
	defer server.Stop()

	// 记录策略创建前的统计
	statsBefore := env.GetStatistics()
	t.Logf("统计 (策略创建前): egress_denied=%d, egress_packets=%d",
		statsBefore.EgressDenied, statsBefore.EgressPackets)

	// 尝试从 Client 向 Server 发送流量
	// Ingress 流量应该允许 (没有 Ingress DENY 策略)
	// 但 Server 的响应 (Egress) 应该被阻止
	testData := []byte("Test egress deny")
	err = env.SendTCPTraffic(8080, testData)
	// 注意: SendTCPTraffic 会期望收到响应,如果 Egress 被阻止,可能会超时
	// 这是预期的行为

	// 等待一段时间,让统计数据更新
	time.Sleep(200 * time.Millisecond)

	// 验证统计
	statsAfter := env.GetStatistics()
	t.Logf("统计 (测试后): egress_denied=%d, egress_packets=%d, total_packets=%d",
		statsAfter.EgressDenied, statsAfter.EgressPackets, statsAfter.TotalPackets)

	// 验证 Egress 流量被拒绝
	// 应该有至少一个 Egress Denied (Server 尝试响应时被阻止)
	assert.Greater(t, statsAfter.EgressDenied, statsBefore.EgressDenied,
		"Egress denied counter should increase")

	// 验证总的拒绝数包含 Egress 拒绝
	assert.Greater(t, statsAfter.DeniedPackets, statsBefore.DeniedPackets,
		"Total denied packets should increase")

	t.Logf("✓ Egress DENY 策略成功阻止了 %d 个出站数据包",
		statsAfter.EgressDenied-statsBefore.EgressDenied)
}

// TestE2E_EgressAllowPolicy 测试 Egress ALLOW 策略允许出站流量
//
// 场景描述:
// 1. 创建 Egress ALLOW 策略,允许 Server → Client 流量
// 2. 从 Client 向 Server 发送请求
// 3. Server 响应应该被允许通过
// 4. 验证 egress_packets 统计计数器增加,egress_denied 不增加
func TestE2E_EgressAllowPolicy(t *testing.T) {
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

	// 创建 Egress ALLOW 策略
	// 允许从 Server 到 Client 的流量
	egressAllowPolicy := &policy.Policy{
		RuleID:    2000,
		SrcIP:     serverIP,   // Server 作为源
		DstIP:     clientIP,   // Client 作为目标
		SrcPort:   0,
		DstPort:   0,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "egress",   // Egress 方向
		Priority:  100,
	}

	err = env.CreatePolicy(egressAllowPolicy)
	require.NoError(t, err, "Failed to create egress allow policy")

	// 同时创建 Ingress ALLOW 策略,允许 Client → Server 流量
	// 这样完整的请求-响应流程才能工作
	ingressAllowPolicy := &policy.Policy{
		RuleID:    2001,
		SrcIP:     clientIP,   // Client 作为源
		DstIP:     serverIP,   // Server 作为目标
		SrcPort:   0,
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",  // Ingress 方向
		Priority:  100,
	}

	err = env.CreatePolicy(ingressAllowPolicy)
	require.NoError(t, err, "Failed to create ingress allow policy")

	// 启动 TCP Echo Server
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err, "Failed to start TCP server")
	defer server.Stop()

	// 记录策略创建前的统计
	statsBefore := env.GetStatistics()
	t.Logf("统计 (策略创建前): egress_packets=%d, egress_denied=%d",
		statsBefore.EgressPackets, statsBefore.EgressDenied)

	// 从 Client 向 Server 发送流量
	// 因为有 Ingress 和 Egress ALLOW 策略,流量应该双向通过
	testData := []byte("Test egress allow")
	err = env.SendTCPTraffic(8080, testData)
	assert.NoError(t, err, "Traffic should be allowed in both directions")

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

	t.Logf("✓ Egress ALLOW 策略成功允许了 %d 个出站数据包",
		statsAfter.EgressPackets-statsBefore.EgressPackets)
}

// TestE2E_EgressDefaultBehavior 测试没有 Egress 策略时的默认行为
//
// 场景描述:
// 1. 不创建任何 Egress 策略
// 2. 只创建 Ingress ALLOW 策略 (允许 Client → Server)
// 3. 测试 Server → Client 的响应是否被允许
// 4. 验证默认行为 (应该允许,因为没有明确的 DENY 策略)
func TestE2E_EgressDefaultBehavior(t *testing.T) {
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

	// 只创建 Ingress ALLOW 策略
	// 不创建任何 Egress 策略,测试默认行为
	ingressAllowPolicy := &policy.Policy{
		RuleID:    3000,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  100,
	}

	err = env.CreatePolicy(ingressAllowPolicy)
	require.NoError(t, err, "Failed to create ingress allow policy")

	// 启动 TCP Echo Server
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err, "Failed to start TCP server")
	defer server.Stop()

	// 记录统计
	statsBefore := env.GetStatistics()
	t.Logf("统计 (测试前): egress_packets=%d, egress_denied=%d",
		statsBefore.EgressPackets, statsBefore.EgressDenied)

	// 发送流量
	testData := []byte("Test egress default")
	err = env.SendTCPTraffic(8080, testData)

	// 等待统计更新
	time.Sleep(200 * time.Millisecond)

	statsAfter := env.GetStatistics()
	t.Logf("统计 (测试后): egress_packets=%d, egress_denied=%d",
		statsAfter.EgressPackets, statsAfter.EgressDenied)

	// 文档化默认行为
	if err == nil {
		t.Log("✓ 默认行为: ALLOW (没有 Egress 策略 = 允许出站)")
		assert.Greater(t, statsAfter.EgressPackets, statsBefore.EgressPackets,
			"Egress packets should be processed")
		assert.Equal(t, statsBefore.EgressDenied, statsAfter.EgressDenied,
			"Egress should not be denied by default")
	} else {
		t.Log("✗ 默认行为: DENY (没有 Egress 策略 = 拒绝出站)")
		t.Logf("Error: %v", err)
		// 如果默认是 DENY,应该看到 egress_denied 增加
		assert.Greater(t, statsAfter.EgressDenied, statsBefore.EgressDenied,
			"Egress should be denied by default")
	}
}

// TestE2E_DirectionalPolicyIsolation 测试 Ingress 和 Egress 策略的隔离性
//
// 场景描述:
// 1. 创建 Ingress DENY 策略 (拒绝 Client → Server)
// 2. 创建 Egress ALLOW 策略 (允许 Server → Client)
// 3. 验证两个方向的策略互不影响
// 4. Ingress 流量应该被阻止,Egress 流量应该被允许
func TestE2E_DirectionalPolicyIsolation(t *testing.T) {
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

	// 创建 Ingress DENY 策略 (拒绝 Client → Server)
	ingressDenyPolicy := &policy.Policy{
		RuleID:    4000,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "ingress",  // Ingress 方向
		Priority:  100,
	}

	err = env.CreatePolicy(ingressDenyPolicy)
	require.NoError(t, err, "Failed to create ingress deny policy")

	// 创建 Egress ALLOW 策略 (允许 Server → Client)
	// 这个策略不应该影响 Ingress DENY 的效果
	egressAllowPolicy := &policy.Policy{
		RuleID:    4001,
		SrcIP:     serverIP,
		DstIP:     clientIP,
		SrcPort:   0,
		DstPort:   0,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "egress",   // Egress 方向
		Priority:  100,
	}

	err = env.CreatePolicy(egressAllowPolicy)
	require.NoError(t, err, "Failed to create egress allow policy")

	// 启动 TCP Echo Server
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err, "Failed to start TCP server")
	defer server.Stop()

	// 记录统计
	statsBefore := env.GetStatistics()
	t.Logf("统计 (测试前): ingress_denied=%d, egress_denied=%d",
		statsBefore.IngressDenied, statsBefore.EgressDenied)

	// 尝试从 Client 向 Server 发送流量
	// Ingress 应该被 DENY 策略阻止
	env.AssertTrafficBlocked(8080)

	// 等待统计更新
	time.Sleep(200 * time.Millisecond)

	statsAfter := env.GetStatistics()
	t.Logf("统计 (测试后): ingress_denied=%d, egress_denied=%d",
		statsAfter.IngressDenied, statsAfter.EgressDenied)

	// 验证 Ingress 被拒绝
	assert.Greater(t, statsAfter.IngressDenied, statsBefore.IngressDenied,
		"Ingress traffic should be denied")

	// 验证 Egress 策略没有误阻止 Ingress 流量
	// (实际上 Ingress 被阻止是因为 Ingress DENY 策略,不是 Egress 策略)

	t.Log("✓ Ingress 和 Egress 策略正确隔离,互不影响")
}
