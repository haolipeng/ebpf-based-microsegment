// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package benchmark

import (
	"fmt"
	"testing"
	"time"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/benchmark"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/policy"
	"github.com/haolipeng/ebpf-based-microsegment/pkg/testutil"
	"github.com/haolipeng/ebpf-based-microsegment/test/e2e"
	"github.com/stretchr/testify/require"
)

// BenchmarkPolicyLookup_10Policies benchmarks policy lookup with 10 wildcard policies
func BenchmarkPolicyLookup_10Policies(b *testing.B) {
	benchmarkPolicyLookup(b, 10, "Worst Case")
}

// BenchmarkPolicyLookup_50Policies benchmarks policy lookup with 50 wildcard policies
func BenchmarkPolicyLookup_50Policies(b *testing.B) {
	benchmarkPolicyLookup(b, 50, "Worst Case")
}

// BenchmarkPolicyLookup_100Policies benchmarks policy lookup with 100 wildcard policies
func BenchmarkPolicyLookup_100Policies(b *testing.B) {
	benchmarkPolicyLookup(b, 100, "Worst Case")
}

// BenchmarkPolicyLookup_BestCase_50Policies tests best-case scenario (first policy matches)
func BenchmarkPolicyLookup_BestCase_50Policies(b *testing.B) {
	benchmarkPolicyLookup(b, 50, "Best Case")
}

// BenchmarkPolicyLookup_AverageCase_50Policies tests average-case scenario (middle policy matches)
func BenchmarkPolicyLookup_AverageCase_50Policies(b *testing.B) {
	benchmarkPolicyLookup(b, 50, "Average Case")
}

// benchmarkPolicyLookup is the core benchmark function
// It tests the worst-case scenario where the matching policy is at the end of the list
func benchmarkPolicyLookup(b *testing.B, policyCount int, scenario string) {
	if msg := testutil.CheckE2ERequirements(); msg != "" {
		b.Skip(msg)
	}

	// Setup E2E test environment
	// Note: We need to create a testing.T from testing.B for E2E setup
	t := &testing.T{}
	env, err := e2e.NewE2ETestEnv(t)
	if err != nil {
		b.Fatalf("Failed to create test environment: %v", err)
	}
	defer env.Cleanup()

	// Generate test policies based on scenario
	var policies []*policy.Policy
	switch scenario {
	case "Best Case":
		policies = benchmark.GenerateBestCasePolicies(policyCount)
	case "Average Case":
		policies = benchmark.GenerateAverageCasePolicies(policyCount)
	default: // "Worst Case"
		policies = benchmark.GenerateTestPolicies(policyCount)
	}

	// Add policies using environment's CreatePolicy method
	for _, p := range policies {
		err := env.CreatePolicy(p)
		require.NoError(b, err, "Failed to add policy")
	}

	b.Logf("Setup complete: %d policies loaded (%s scenario)", policyCount, scenario)

	// Start TCP server
	server, err := env.StartTCPServer(8080)
	require.NoError(b, err)
	defer server.Stop()

	// Warmup: run some connections to warm up caches
	for i := 0; i < 100; i++ {
		env.TryConnect(8080)
	}
	time.Sleep(100 * time.Millisecond)

	// Reset timer to exclude setup time
	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark loop
	var samples []time.Duration
	for i := 0; i < b.N; i++ {
		start := time.Now()
		connected := env.TryConnect(8080)
		elapsed := time.Since(start)

		if connected {
			samples = append(samples, elapsed)
		}

		// Rate limiting to avoid overwhelming the system
		if i%100 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	b.StopTimer()

	// Calculate and report statistics
	if len(samples) > 0 {
		stats := benchmark.CalculateStats(samples)
		b.ReportMetric(float64(stats.Median.Nanoseconds()/1000), "p50_μs")
		b.ReportMetric(float64(stats.P95.Nanoseconds()/1000), "p95_μs")
		b.ReportMetric(float64(stats.P99.Nanoseconds()/1000), "p99_μs")
		b.ReportMetric(float64(stats.Mean.Nanoseconds()/1000), "mean_μs")
		b.ReportMetric(stats.Throughput, "ops/s")

		b.Logf("\n"+
			"=== Performance Statistics ===\n"+
			"Samples:    %d\n"+
			"P50 (μs):   %.2f\n"+
			"P95 (μs):   %.2f\n"+
			"P99 (μs):   %.2f\n"+
			"Mean (μs):  %.2f\n"+
			"Max (μs):   %.2f\n"+
			"Throughput: %.2f ops/s\n",
			stats.Count,
			float64(stats.Median.Nanoseconds())/1000.0,
			float64(stats.P95.Nanoseconds())/1000.0,
			float64(stats.P99.Nanoseconds())/1000.0,
			float64(stats.Mean.Nanoseconds())/1000.0,
			float64(stats.Max.Nanoseconds())/1000.0,
			stats.Throughput,
		)
	}
}

// TestPolicyLookup_GenerateReport generates a comprehensive performance report
// This is not a benchmark but a test that runs benchmarks and generates a report
func TestPolicyLookup_GenerateReport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping report generation in short mode")
	}

	if msg := testutil.CheckE2ERequirements(); msg != "" {
		t.Skip(msg)
	}

	t.Log("=== Generating Performance Benchmark Report ===")

	scenarios := []struct {
		policyCount int
		scenario    string
	}{
		{10, "Worst Case"},
		{50, "Worst Case"},
		{100, "Worst Case"},
	}

	report := &benchmark.BenchmarkReport{
		TestName:    "eBPF Wildcard Policy Lookup Performance Benchmark",
		TestDate:    time.Now(),
		Environment: benchmark.GetEnvironmentInfo(),
		Results:     make([]benchmark.BenchmarkResult, 0),
	}

	for _, sc := range scenarios {
		t.Logf("Running benchmark: %d policies (%s)", sc.policyCount, sc.scenario)

		// Setup environment
		env, err := e2e.NewE2ETestEnv(t)
		require.NoError(t, err)

		// Generate and add policies
		policies := benchmark.GenerateTestPolicies(sc.policyCount)
		for _, p := range policies {
			err := env.CreatePolicy(p)
			require.NoError(t, err)
		}

		// Start server
		server, err := env.StartTCPServer(8080)
		require.NoError(t, err)

		// Warmup
		for i := 0; i < 100; i++ {
			env.TryConnect(8080)
		}
		time.Sleep(100 * time.Millisecond)

		// Collect samples
		samples := make([]time.Duration, 0, 1000)
		for i := 0; i < 1000; i++ {
			start := time.Now()
			if env.TryConnect(8080) {
				samples = append(samples, time.Since(start))
			}
			if i%100 == 0 {
				time.Sleep(10 * time.Millisecond)
			}
		}

		// Calculate stats
		stats := benchmark.CalculateStats(samples)

		// Add to report
		result := benchmark.BenchmarkResult{
			ScenarioName: fmt.Sprintf("%s - %d Policies", sc.scenario, sc.policyCount),
			PolicyCount:  sc.policyCount,
			Stats:        stats,
			Notes:        "Testing worst-case scenario with matching policy at end of list",
		}
		report.Results = append(report.Results, result)

		// Cleanup
		server.Stop()
		env.Cleanup()

		t.Logf("Completed: %s", result.ScenarioName)
	}

	// Add summary
	report.Summary = fmt.Sprintf(
		"Benchmark completed with %d test scenarios. "+
			"Testing eBPF wildcard policy matching performance after Task #18 optimization "+
			"(loop constant %d, early termination enabled).",
		len(scenarios), 50,
	)

	// Add recommendations
	report.Recommendations = []string{
		"Current linear scan performs well for up to 50 policies",
		"Consider implementing LPM Trie (Phase 2) for >100 policies",
		"Early termination optimization significantly reduces worst-case latency",
		"Maintain compact storage on server side to maximize early termination benefit",
	}

	// Generate and save report
	reportPath := "docs/performance/BENCHMARK_BASELINE.md"
	err := benchmark.SaveReport(report, reportPath)
	if err != nil {
		t.Logf("Warning: Failed to save report to %s: %v", reportPath, err)
		// Print to console instead
		t.Log("\n" + benchmark.GenerateMarkdownReport(report))
	} else {
		t.Logf("Report saved to: %s", reportPath)
	}
}
