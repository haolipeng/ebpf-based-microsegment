// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/policy"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/testutil"
	"github.com/stretchr/testify/require"
)

// TestE2E_Performance_Baseline 基准性能测试 - 无策略
//
// 测试场景: 不加载任何策略，测量基准性能
// 指标: 吞吐量、连接建立时间、统计更新
func TestE2E_Performance_Baseline(t *testing.T) {
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	env, err := NewE2ETestEnv(t)
	require.NoError(t, err)
	defer env.Cleanup()

	// 启动 Server
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err)
	defer server.Stop()

	t.Log("=== 基准性能测试 (无策略) ===")

	// 测试 1: 单次连接延迟
	statsBefore := env.GetStatistics()
	start := time.Now()

	connected := env.TryConnect(8080)
	require.True(t, connected, "Baseline connection should succeed")

	connectionTime := time.Since(start)
	time.Sleep(100 * time.Millisecond)

	statsAfter := env.GetStatistics()

	t.Logf("单次连接时间: %v", connectionTime)
	t.Logf("统计: total_packets=%d, allowed=%d, policy_misses=%d",
		statsAfter.TotalPackets-statsBefore.TotalPackets,
		statsAfter.AllowedPackets-statsBefore.AllowedPackets,
		statsAfter.PolicyMisses-statsBefore.PolicyMisses)

	// 测试 2: 批量连接吞吐量
	t.Log("\n=== 批量连接测试 (100 次) ===")
	stats2Before := env.GetStatistics()
	batchStart := time.Now()

	successCount := 0
	for i := 0; i < 100; i++ {
		if env.TryConnect(8080) {
			successCount++
		}
	}

	batchDuration := time.Since(batchStart)
	time.Sleep(200 * time.Millisecond)
	stats2After := env.GetStatistics()

	throughput := float64(successCount) / batchDuration.Seconds()
	avgLatency := batchDuration / time.Duration(successCount)

	t.Logf("批量连接: %d/%d 成功", successCount, 100)
	t.Logf("总耗时: %v", batchDuration)
	t.Logf("平均延迟: %v", avgLatency)
	t.Logf("吞吐量: %.2f connections/sec", throughput)
	t.Logf("统计增量: total_packets=%d, allowed=%d",
		stats2After.TotalPackets-stats2Before.TotalPackets,
		stats2After.AllowedPackets-stats2Before.AllowedPackets)

	// 保存基准数据
	t.Logf("\n=== 基准性能指标 ===")
	t.Logf("单次连接延迟: %v", connectionTime)
	t.Logf("批量平均延迟: %v", avgLatency)
	t.Logf("连接吞吐量: %.2f conn/s", throughput)
}

// TestE2E_Performance_WithIngressPolicy 测试 Ingress 策略的性能影响
//
// 测试场景: 添加 Ingress ALLOW 策略
// 对比: 与 Baseline 对比，测量策略匹配开销
func TestE2E_Performance_WithIngressPolicy(t *testing.T) {
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	env, err := NewE2ETestEnv(t)
	require.NoError(t, err)
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	// 创建 Ingress ALLOW 策略
	ingressPolicy := &policy.Policy{
		RuleID:    40000,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  100,
	}
	err = env.CreatePolicy(ingressPolicy)
	require.NoError(t, err)

	// 启动 Server
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err)
	defer server.Stop()

	t.Log("=== Ingress 策略性能测试 ===")

	// 测试 1: 单次连接延迟
	statsBefore := env.GetStatistics()
	start := time.Now()

	connected := env.TryConnect(8080)
	require.True(t, connected, "Connection should succeed with ALLOW policy")

	connectionTime := time.Since(start)
	time.Sleep(100 * time.Millisecond)

	statsAfter := env.GetStatistics()

	t.Logf("单次连接时间: %v", connectionTime)
	t.Logf("统计: ingress_packets=%d, allowed=%d, policy_hits=%d",
		statsAfter.IngressPackets-statsBefore.IngressPackets,
		statsAfter.AllowedPackets-statsBefore.AllowedPackets,
		statsAfter.PolicyHits-statsBefore.PolicyHits)

	// 测试 2: 批量连接吞吐量
	t.Log("\n=== 批量连接测试 (100 次) ===")
	stats2Before := env.GetStatistics()
	batchStart := time.Now()

	successCount := 0
	for i := 0; i < 100; i++ {
		if env.TryConnect(8080) {
			successCount++
		}
	}

	batchDuration := time.Since(batchStart)
	time.Sleep(200 * time.Millisecond)
	stats2After := env.GetStatistics()

	throughput := float64(successCount) / batchDuration.Seconds()
	avgLatency := batchDuration / time.Duration(successCount)

	t.Logf("批量连接: %d/%d 成功", successCount, 100)
	t.Logf("总耗时: %v", batchDuration)
	t.Logf("平均延迟: %v", avgLatency)
	t.Logf("吞吐量: %.2f connections/sec", throughput)
	t.Logf("统计增量: ingress_packets=%d, policy_hits=%d",
		stats2After.IngressPackets-stats2Before.IngressPackets,
		stats2After.PolicyHits-stats2Before.PolicyHits)

	t.Logf("\n=== Ingress 策略性能指标 ===")
	t.Logf("单次连接延迟: %v", connectionTime)
	t.Logf("批量平均延迟: %v", avgLatency)
	t.Logf("连接吞吐量: %.2f conn/s", throughput)
}

// TestE2E_Performance_WithBidirectionalPolicy 测试双向策略的性能影响
//
// 测试场景: 添加 Ingress + Egress 策略
// 对比: 测量双向策略匹配的开销
func TestE2E_Performance_WithBidirectionalPolicy(t *testing.T) {
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	env, err := NewE2ETestEnv(t)
	require.NoError(t, err)
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	// 创建 Ingress ALLOW 策略
	ingressPolicy := &policy.Policy{
		RuleID:    41000,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  200,
	}
	err = env.CreatePolicy(ingressPolicy)
	require.NoError(t, err)

	// 创建 Egress ALLOW 策略
	egressPolicy := &policy.Policy{
		RuleID:    41001,
		SrcIP:     serverIP,
		DstIP:     clientIP,
		SrcPort:   0,
		DstPort:   0,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "egress",
		Priority:  100,
	}
	err = env.CreatePolicy(egressPolicy)
	require.NoError(t, err)

	// 启动 Server
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err)
	defer server.Stop()

	t.Log("=== 双向策略性能测试 ===")

	// 测试 1: 单次连接延迟
	statsBefore := env.GetStatistics()
	start := time.Now()

	connected := env.TryConnect(8080)
	require.True(t, connected, "Connection should succeed with bidirectional ALLOW")

	connectionTime := time.Since(start)
	time.Sleep(100 * time.Millisecond)

	statsAfter := env.GetStatistics()

	t.Logf("单次连接时间: %v", connectionTime)
	t.Logf("统计: ingress_packets=%d, egress_packets=%d, allowed=%d",
		statsAfter.IngressPackets-statsBefore.IngressPackets,
		statsAfter.EgressPackets-statsBefore.EgressPackets,
		statsAfter.AllowedPackets-statsBefore.AllowedPackets)

	// 测试 2: 批量连接吞吐量
	t.Log("\n=== 批量连接测试 (100 次) ===")
	stats2Before := env.GetStatistics()
	batchStart := time.Now()

	successCount := 0
	for i := 0; i < 100; i++ {
		if env.TryConnect(8080) {
			successCount++
		}
	}

	batchDuration := time.Since(batchStart)
	time.Sleep(200 * time.Millisecond)
	stats2After := env.GetStatistics()

	throughput := float64(successCount) / batchDuration.Seconds()
	avgLatency := batchDuration / time.Duration(successCount)

	t.Logf("批量连接: %d/%d 成功", successCount, 100)
	t.Logf("总耗时: %v", batchDuration)
	t.Logf("平均延迟: %v", avgLatency)
	t.Logf("吞吐量: %.2f connections/sec", throughput)
	t.Logf("统计增量: ingress=%d, egress=%d, new_sessions=%d",
		stats2After.IngressPackets-stats2Before.IngressPackets,
		stats2After.EgressPackets-stats2Before.EgressPackets,
		stats2After.NewSessions-stats2Before.NewSessions)

	t.Logf("\n=== 双向策略性能指标 ===")
	t.Logf("单次连接延迟: %v", connectionTime)
	t.Logf("批量平均延迟: %v", avgLatency)
	t.Logf("连接吞吐量: %.2f conn/s", throughput)
}

// TestE2E_Performance_PolicyScaling 测试策略数量对性能的影响
//
// 测试场景: 添加多个 wildcard 策略，测量查找性能
// 目标: 验证 O(n) 线性扫描的性能影响
func TestE2E_Performance_PolicyScaling(t *testing.T) {
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	env, err := NewE2ETestEnv(t)
	require.NoError(t, err)
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	// 测试不同策略数量: 1, 10, 50, 100
	policyCounts := []int{1, 10, 50}

	for _, count := range policyCounts {
		t.Run(fmt.Sprintf("Policies_%d", count), func(t *testing.T) {
			// 清理之前的策略
			env.Cleanup()
			env, err = NewE2ETestEnv(t)
			require.NoError(t, err)
			defer env.Cleanup()

			// 创建多个 wildcard 策略
			baseRuleID := uint32(42000 + count*1000)
			for i := 0; i < count-1; i++ {
				// 创建不匹配的策略 (干扰项)
				dummyPolicy := &policy.Policy{
					RuleID:    baseRuleID + uint32(i),
					SrcIP:     "192.168.1.1", // 不匹配的 IP
					DstIP:     "192.168.1.2",
					SrcPort:   0,
					DstPort:   uint16(9000 + i),
					Protocol:  "tcp",
					Action:    "deny",
					Direction: "ingress",
					Priority:  100,
				}
				err = env.CreatePolicy(dummyPolicy)
				require.NoError(t, err)
			}

			// 创建最后一个匹配的策略
			matchPolicy := &policy.Policy{
				RuleID:    baseRuleID + uint32(count-1),
				SrcIP:     clientIP,
				DstIP:     serverIP,
				SrcPort:   0,
				DstPort:   8080,
				Protocol:  "tcp",
				Action:    "allow",
				Direction: "ingress",
				Priority:  200, // 高优先级
			}
			err = env.CreatePolicy(matchPolicy)
			require.NoError(t, err)

			// 启动 Server
			server, err := env.StartTCPServer(8080)
			require.NoError(t, err)
			defer server.Stop()

			t.Logf("=== 策略数量: %d ===", count)

			// 测试批量连接性能
			statsBefore := env.GetStatistics()
			batchStart := time.Now()

			successCount := 0
			iterations := 50 // 减少迭代次数以节省时间
			for i := 0; i < iterations; i++ {
				if env.TryConnect(8080) {
					successCount++
				}
			}

			batchDuration := time.Since(batchStart)
			time.Sleep(100 * time.Millisecond)
			statsAfter := env.GetStatistics()

			throughput := float64(successCount) / batchDuration.Seconds()
			avgLatency := batchDuration / time.Duration(successCount)

			t.Logf("批量连接: %d/%d 成功", successCount, iterations)
			t.Logf("平均延迟: %v", avgLatency)
			t.Logf("吞吐量: %.2f conn/s", throughput)
			t.Logf("统计: allowed=%d, policy_hits=%d",
				statsAfter.AllowedPackets-statsBefore.AllowedPackets,
				statsAfter.PolicyHits-statsBefore.PolicyHits)
		})
	}
}

// TestE2E_Performance_SessionCacheEfficiency 测试 Session 缓存效率
//
// 测试场景: 建立连接后重复发送数据，测量缓存命中性能
// 目标: 验证 session 缓存的性能提升
func TestE2E_Performance_SessionCacheEfficiency(t *testing.T) {
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	env, err := NewE2ETestEnv(t)
	require.NoError(t, err)
	defer env.Cleanup()

	clientIP := env.Network.GetClientIP()
	serverIP := env.Network.GetServerIP()

	// 创建策略
	ingressPolicy := &policy.Policy{
		RuleID:    43000,
		SrcIP:     clientIP,
		DstIP:     serverIP,
		SrcPort:   0,
		DstPort:   8080,
		Protocol:  "tcp",
		Action:    "allow",
		Direction: "ingress",
		Priority:  100,
	}
	err = env.CreatePolicy(ingressPolicy)
	require.NoError(t, err)

	// 启动 Server
	server, err := env.StartTCPServer(8080)
	require.NoError(t, err)
	defer server.Stop()

	t.Log("=== Session 缓存效率测试 ===")

	// 阶段 1: 首次连接 (冷启动，需要策略查找)
	t.Log("\n阶段 1: 首次连接 (冷启动)")
	stats1Before := env.GetStatistics()
	start1 := time.Now()

	connected := env.TryConnect(8080)
	require.True(t, connected)

	coldStartTime := time.Since(start1)
	time.Sleep(100 * time.Millisecond)
	stats1After := env.GetStatistics()

	t.Logf("冷启动延迟: %v", coldStartTime)
	t.Logf("统计: new_sessions=%d, policy_misses=%d",
		stats1After.NewSessions-stats1Before.NewSessions,
		stats1After.PolicyMisses-stats1Before.PolicyMisses)

	// 阶段 2: 后续连接 (热路径，session 缓存命中)
	t.Log("\n阶段 2: 后续连接 (热路径)")
	stats2Before := env.GetStatistics()
	batchStart := time.Now()

	successCount := 0
	iterations := 100
	for i := 0; i < iterations; i++ {
		if env.TryConnect(8080) {
			successCount++
		}
	}

	warmPathDuration := time.Since(batchStart)
	time.Sleep(100 * time.Millisecond)
	stats2After := env.GetStatistics()

	avgWarmLatency := warmPathDuration / time.Duration(successCount)
	throughput := float64(successCount) / warmPathDuration.Seconds()

	t.Logf("热路径平均延迟: %v", avgWarmLatency)
	t.Logf("吞吐量: %.2f conn/s", throughput)
	t.Logf("统计: new_sessions=%d, total_packets=%d",
		stats2After.NewSessions-stats2Before.NewSessions,
		stats2After.TotalPackets-stats2Before.TotalPackets)

	// 计算性能提升
	if avgWarmLatency > 0 && coldStartTime > 0 {
		speedup := float64(coldStartTime) / float64(avgWarmLatency)
		t.Logf("\n=== Session 缓存性能提升 ===")
		t.Logf("冷启动: %v", coldStartTime)
		t.Logf("热路径: %v", avgWarmLatency)
		t.Logf("加速比: %.2fx", speedup)
	}
}
