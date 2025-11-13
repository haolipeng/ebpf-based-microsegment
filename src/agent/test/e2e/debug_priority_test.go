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

// TestE2E_Debug_PriorityConflict 调试优先级冲突问题
//
// 复现场景:
// - 策略 1: Ingress DENY, Client → Server:8080, priority=200 (更具体)
// - 策略 2: Ingress ALLOW, Client → Server:*, priority=100 (通配符)
//
// 预期: 策略 1 优先级更高,应该 DENY
func TestE2E_Debug_PriorityConflict(t *testing.T) {
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

	// 策略 1: Ingress DENY (具体端口,高优先级)
	denyPolicy := &policy.Policy{
		RuleID:    20000,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "ingress",
		Priority:  200,
	}

	err = env.CreatePolicy(denyPolicy)
	require.NoError(t, err)
	t.Logf("策略 1: Ingress DENY - %s:* → %s:8080 (priority=200)", clientIP, serverIP)

	// 策略 2: Ingress ALLOW (通配符,低优先级)
	allowPolicy := &policy.Policy{
		RuleID:    20001,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   0,  // 任意端口
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  100,
	}

	err = env.CreatePolicy(allowPolicy)
	require.NoError(t, err)
	t.Logf("策略 2: Ingress ALLOW - %s:* → %s:* (priority=100)", clientIP, serverIP)

	// 记录统计
	statsBefore := env.GetStatistics()
	t.Logf("测试前统计: ingress_packets=%d, ingress_denied=%d, allowed=%d",
		statsBefore.IngressPackets, statsBefore.IngressDenied, statsBefore.AllowedPackets)

	// 尝试连接 (应该被 DENY 策略阻止,因为优先级更高)
	connected := env.TryConnect(8080)

	time.Sleep(200 * time.Millisecond)

	statsAfter := env.GetStatistics()
	t.Logf("测试后统计: ingress_packets=%d, ingress_denied=%d, allowed=%d",
		statsAfter.IngressPackets, statsAfter.IngressDenied, statsAfter.AllowedPackets)

	// 验证
	t.Logf("连接结果: connected=%v (应该是 false)", connected)
	assert.False(t, connected, "Connection should be denied by高优先级策略")

	if statsAfter.IngressPackets > statsBefore.IngressPackets {
		t.Logf("✓ Ingress 数据包被处理了")
	} else {
		t.Errorf("✗ 没有 Ingress 数据包!这说明数据包根本没到达 eBPF")
	}

	if statsAfter.IngressDenied > statsBefore.IngressDenied {
		t.Logf("✓ Ingress DENY 生效")
	} else {
		t.Errorf("✗ Ingress DENY 没生效 (denied=%d)", statsAfter.IngressDenied-statsBefore.IngressDenied)
	}

	if statsAfter.AllowedPackets > statsBefore.AllowedPackets {
		t.Errorf("✗ 数据包被 ALLOW 了!优先级没有正确工作")
	} else {
		t.Logf("✓ 数据包没有被 ALLOW")
	}
}
