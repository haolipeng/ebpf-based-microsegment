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

// TestE2E_Simple_IngressDeny 简单测试 Ingress DENY 策略
func TestE2E_Simple_IngressDeny(t *testing.T) {
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	env, err := NewE2ETestEnv(t)
	require.NoError(t, err)
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	// 启动 Server
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err)
	defer server.Stop()

	// 创建简单的 Ingress DENY 策略
	denyPolicy := &policy.Policy{
		RuleID:    12000,
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

	t.Logf("策略: Ingress DENY - %s → %s:8080", clientIP, serverIP)

	// 记录统计 (策略创建后)
	statsBefore := env.GetStatistics()
	t.Logf("测试前统计: ingress_packets=%d, ingress_denied=%d, denied=%d",
		statsBefore.IngressPackets, statsBefore.IngressDenied, statsBefore.DeniedPackets)

	// 尝试连接
	connected := env.TryConnect(8080)

	// 等待统计更新
	time.Sleep(200 * time.Millisecond)

	statsAfter := env.GetStatistics()
	t.Logf("测试后统计: ingress_packets=%d, ingress_denied=%d, denied=%d",
		statsAfter.IngressPackets, statsAfter.IngressDenied, statsAfter.DeniedPackets)

	// 验证连接被阻止
	assert.False(t, connected, "Connection should be denied")

	// 验证统计
	assert.Greater(t, statsAfter.IngressPackets, statsBefore.IngressPackets,
		"Ingress packets should increase")
	assert.Greater(t, statsAfter.IngressDenied, statsBefore.IngressDenied,
		"Ingress denied counter should increase")
	assert.Greater(t, statsAfter.DeniedPackets, statsBefore.DeniedPackets,
		"Total denied packets should increase")

	t.Log("✓ Ingress DENY 策略正常工作")
}

// TestE2E_Simple_EgressDeny 简单测试 Egress DENY 策略
func TestE2E_Simple_EgressDeny(t *testing.T) {
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	env, err := NewE2ETestEnv(t)
	require.NoError(t, err)
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	// 启动 Client 侧的服务器
	clientServer, err := env.StartTCPServerOnClient(9000)
	require.NoError(t, err)
	defer clientServer.Stop()

	// 创建简单的 Egress DENY 策略
	denyPolicy := &policy.Policy{
		RuleID:    13000,
		SrcIP:     serverIP,
		DstIP:     clientIP,
		SrcPort:   0,
		DstPort:   9000,
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "egress",
		Priority:  100,
	}

	err = env.CreatePolicy(denyPolicy)
	require.NoError(t, err)

	t.Logf("策略: Egress DENY - %s → %s:9000", serverIP, clientIP)

	// 记录统计 (策略创建后)
	statsBefore := env.GetStatistics()
	t.Logf("测试前统计: egress_packets=%d, egress_denied=%d, denied=%d",
		statsBefore.EgressPackets, statsBefore.EgressDenied, statsBefore.DeniedPackets)

	// 尝试从 Server 连接 Client
	connected := env.TryConnectFromServer(9000)

	// 等待统计更新
	time.Sleep(200 * time.Millisecond)

	statsAfter := env.GetStatistics()
	t.Logf("测试后统计: egress_packets=%d, egress_denied=%d, denied=%d",
		statsAfter.EgressPackets, statsAfter.EgressDenied, statsAfter.DeniedPackets)

	// 验证连接被阻止
	assert.False(t, connected, "Connection should be denied")

	// 验证统计
	assert.Greater(t, statsAfter.EgressPackets, statsBefore.EgressPackets,
		"Egress packets should increase")
	assert.Greater(t, statsAfter.EgressDenied, statsBefore.EgressDenied,
		"Egress denied counter should increase")
	assert.Greater(t, statsAfter.DeniedPackets, statsBefore.DeniedPackets,
		"Total denied packets should increase")

	t.Log("✓ Egress DENY 策略正常工作")
}
