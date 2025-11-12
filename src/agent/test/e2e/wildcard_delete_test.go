// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

package e2e

import (
	"testing"
	"time"

	"github.com/ebpf-microsegment/src/agent/pkg/policy"
	"github.com/stretchr/testify/require"
)

// TestE2E_WildcardPolicyDelete 测试 Wildcard 策略的删除功能
// 使用 DENY 策略来清晰验证删除效果 (默认策略是 ALLOW)
func TestE2E_WildcardPolicyDelete(t *testing.T) {
	// 创建测试环境
	env, err := NewE2ETestEnv(t)
	require.NoError(t, err, "Failed to create test environment")
	defer env.Cleanup()

	// 启动两个服务器(用于测试策略删除前后的行为)
	_, err = env.StartTCPServer(8080)
	require.NoError(t, err, "Failed to start TCP server on 8080")

	_, err = env.StartTCPServer(8081)
	require.NoError(t, err, "Failed to start TCP server on 8081")

	// 创建 wildcard DENY 策略 for port 8080 (source port = 0 是 wildcard)
	denyPolicy8080 := &policy.Policy{
		RuleID:    1000,
		SrcIP:     "10.100.0.1/32",
		DstIP:     "10.100.0.2/32",
		SrcPort:   0, // Wildcard
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "ingress",
		Priority:  100,
	}

	err = env.CreatePolicy(denyPolicy8080)
	require.NoError(t, err, "Failed to create wildcard deny policy for 8080")

	// 等待策略生效
	time.Sleep(200 * time.Millisecond)

	// 验证 8080 连接失败 (DENY), 8081 连接成功 (默认 ALLOW)
	success := env.TryConnect(8080)
	require.False(t, success, "Port 8080 should be DENIED")

	success = env.TryConnect(8081)
	require.True(t, success, "Port 8081 should be ALLOWED (default)")

	// 删除 8080 的 wildcard DENY 策略
	t.Log("删除 port 8080 的 wildcard DENY 策略...")
	err = env.DeletePolicy(denyPolicy8080)
	require.NoError(t, err, "Failed to delete wildcard policy for 8080")

	// 等待删除生效
	time.Sleep(200 * time.Millisecond)

	// 验证策略数量
	policies, err := env.ListPolicies()
	require.NoError(t, err, "Failed to list policies")
	t.Logf("当前精确策略数量: %d (wildcard 策略在单独的 map)", len(policies))

	// 验证 8080 现在可以访问 (DENY 策略已删除,使用默认 ALLOW)
	success = env.TryConnect(8080)
	require.True(t, success, "Port 8080 should be ALLOWED after deleting DENY policy")

	// 现在添加 8081 的 DENY 策略,验证删除后可以正常添加新策略
	denyPolicy8081 := &policy.Policy{
		RuleID:    1001,
		SrcIP:     "10.100.0.1/32",
		DstIP:     "10.100.0.2/32",
		SrcPort:   0, // Wildcard
		DstPort:   8081,
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "ingress",
		Priority:  100,
	}

	err = env.CreatePolicy(denyPolicy8081)
	require.NoError(t, err, "Failed to create wildcard deny policy for 8081")

	time.Sleep(200 * time.Millisecond)

	// 验证 8081 现在被拒绝
	success = env.TryConnect(8081)
	require.False(t, success, "Port 8081 should be DENIED after adding policy")

	t.Log("✓ Wildcard 策略删除功能正常工作")
	t.Log("  - 策略成功从 map 中删除")
	t.Log("  - 删除后流量恢复 (默认 ALLOW)")
	t.Log("  - 删除后可以正常添加新策略")
}

// TestE2E_WildcardPolicyMultipleAddDelete 测试多个 wildcard 策略的添加和删除
// 使用 DENY 策略来验证删除功能 (因为默认策略是 ALLOW)
func TestE2E_WildcardPolicyMultipleAddDelete(t *testing.T) {
	// 创建测试环境
	env, err := NewE2ETestEnv(t)
	require.NoError(t, err, "Failed to create test environment")
	defer env.Cleanup()

	// 启动服务器
	_, err = env.StartTCPServer(8080)
	require.NoError(t, err, "Failed to start TCP server")

	_, err = env.StartTCPServer(8081)
	require.NoError(t, err, "Failed to start TCP server")

	_, err = env.StartTCPServer(8082)
	require.NoError(t, err, "Failed to start TCP server")

	// 创建 3 个 wildcard DENY 策略
	policy1 := &policy.Policy{
		RuleID:    2000,
		SrcIP:     "10.100.0.1/32",
		DstIP:     "10.100.0.2/32",
		SrcPort:   0, // Wildcard
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "ingress",
		Priority:  100,
	}

	policy2 := &policy.Policy{
		RuleID:    2001,
		SrcIP:     "10.100.0.1/32",
		DstIP:     "10.100.0.2/32",
		SrcPort:   0, // Wildcard
		DstPort:   8081,
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "ingress",
		Priority:  100,
	}

	policy3 := &policy.Policy{
		RuleID:    2002,
		SrcIP:     "10.100.0.1/32",
		DstIP:     "10.100.0.2/32",
		SrcPort:   0, // Wildcard
		DstPort:   8082,
		Protocol:  "tcp",
		Action:    "deny",
		Direction: "ingress",
		Priority:  100,
	}

	// 添加所有 DENY 策略
	err = env.CreatePolicy(policy1)
	require.NoError(t, err, "Failed to create policy1")
	err = env.CreatePolicy(policy2)
	require.NoError(t, err, "Failed to create policy2")
	err = env.CreatePolicy(policy3)
	require.NoError(t, err, "Failed to create policy3")

	time.Sleep(200 * time.Millisecond)

	// 验证所有连接失败 (被 DENY)
	require.False(t, env.TryConnect(8080), "Port 8080 should be DENIED")
	require.False(t, env.TryConnect(8081), "Port 8081 should be DENIED")
	require.False(t, env.TryConnect(8082), "Port 8082 should be DENIED")

	// 删除中间的策略 (policy2)
	t.Log("删除 policy2 (port 8081 DENY)...")
	err = env.DeletePolicy(policy2)
	require.NoError(t, err, "Failed to delete policy2")

	time.Sleep(200 * time.Millisecond)

	// 验证 8080 和 8082 仍然被拒绝, 8081 现在可以访问 (默认 ALLOW)
	require.False(t, env.TryConnect(8080), "Port 8080 should still be DENIED")
	require.True(t, env.TryConnect(8081), "Port 8081 should be ALLOWED (policy deleted)")
	require.False(t, env.TryConnect(8082), "Port 8082 should still be DENIED")

	// 删除剩余策略
	t.Log("删除 policy1 和 policy3...")
	err = env.DeletePolicy(policy1)
	require.NoError(t, err, "Failed to delete policy1")
	err = env.DeletePolicy(policy3)
	require.NoError(t, err, "Failed to delete policy3")

	time.Sleep(200 * time.Millisecond)

	// 验证所有连接成功 (默认 ALLOW)
	require.True(t, env.TryConnect(8080), "Port 8080 should be ALLOWED (default)")
	require.True(t, env.TryConnect(8081), "Port 8081 should be ALLOWED (default)")
	require.True(t, env.TryConnect(8082), "Port 8082 should be ALLOWED (default)")

	t.Log("✓ 多个 wildcard 策略的添加和删除功能正常工作")
	t.Log("  - DENY 策略成功阻止连接")
	t.Log("  - 删除 DENY 策略后流量恢复 (默认 ALLOW)")
	t.Log("  - 策略独立删除,不影响其他策略")
}
