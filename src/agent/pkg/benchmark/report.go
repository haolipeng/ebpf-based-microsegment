// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: benchmark results
// output: formatted benchmark reports
// pos: benchmark report generator - if file updated, must sync with this header comment and pkg/benchmark/CLAUDE.md
package benchmark

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

// BenchmarkReport contains all information for a performance benchmark report
type BenchmarkReport struct {
	TestName      string
	TestDate      time.Time
	Environment   EnvironmentInfo
	Results       []BenchmarkResult
	Baseline      *BenchmarkResult // Optional baseline for comparison
	Summary       string
	Recommendations []string
}

// EnvironmentInfo contains system and environment information
type EnvironmentInfo struct {
	KernelVersion string
	OS            string
	Arch          string
	CPU           string
	NumCPU        int
	GoVersion     string
	Hostname      string
}

// BenchmarkResult contains results for a specific test scenario
type BenchmarkResult struct {
	ScenarioName  string
	PolicyCount   int
	Stats         *Stats
	Comparison    *StatComparison // Comparison with baseline (if available)
	Notes         string
}

// GetEnvironmentInfo collects current environment information
func GetEnvironmentInfo() EnvironmentInfo {
	hostname, _ := os.Hostname()

	// Try to read kernel version
	kernelVersion := "Unknown"
	if data, err := os.ReadFile("/proc/version"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 3 {
			kernelVersion = parts[2]
		}
	}

	// Try to read CPU info
	cpuInfo := "Unknown"
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					cpuInfo = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}

	return EnvironmentInfo{
		KernelVersion: kernelVersion,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		CPU:           cpuInfo,
		NumCPU:        runtime.NumCPU(),
		GoVersion:     runtime.Version(),
		Hostname:      hostname,
	}
}

// GenerateMarkdownReport creates a Markdown-formatted performance report
func GenerateMarkdownReport(report *BenchmarkReport) string {
	var sb strings.Builder

	// Title
	sb.WriteString(fmt.Sprintf("# %s\n\n", report.TestName))
	sb.WriteString(fmt.Sprintf("**Test Date**: %s\n\n", report.TestDate.Format("2006-01-02 15:04:05")))

	// Environment Information
	sb.WriteString("## Test Environment\n\n")
	sb.WriteString(fmt.Sprintf("- **Hostname**: %s\n", report.Environment.Hostname))
	sb.WriteString(fmt.Sprintf("- **Kernel Version**: %s\n", report.Environment.KernelVersion))
	sb.WriteString(fmt.Sprintf("- **OS**: %s\n", report.Environment.OS))
	sb.WriteString(fmt.Sprintf("- **Architecture**: %s\n", report.Environment.Arch))
	sb.WriteString(fmt.Sprintf("- **CPU**: %s\n", report.Environment.CPU))
	sb.WriteString(fmt.Sprintf("- **CPU Cores**: %d\n", report.Environment.NumCPU))
	sb.WriteString(fmt.Sprintf("- **Go Version**: %s\n\n", report.Environment.GoVersion))

	// Results Table
	sb.WriteString("## Benchmark Results\n\n")
	sb.WriteString("| Scenario | Policy Count | P50 (μs) | P95 (μs) | P99 (μs) | Mean (μs) | Max (μs) | Throughput (ops/s) |\n")
	sb.WriteString("|----------|--------------|----------|----------|----------|-----------|----------|-------------------|\n")

	for _, result := range report.Results {
		sb.WriteString(fmt.Sprintf("| %s | %d | %.2f | %.2f | %.2f | %.2f | %.2f | %.0f |\n",
			result.ScenarioName,
			result.PolicyCount,
			float64(result.Stats.Median.Nanoseconds())/1000.0,   // Convert to microseconds
			float64(result.Stats.P95.Nanoseconds())/1000.0,
			float64(result.Stats.P99.Nanoseconds())/1000.0,
			float64(result.Stats.Mean.Nanoseconds())/1000.0,
			float64(result.Stats.Max.Nanoseconds())/1000.0,
			result.Stats.Throughput,
		))
	}
	sb.WriteString("\n")

	// Detailed Statistics
	sb.WriteString("## Detailed Statistics\n\n")
	for _, result := range report.Results {
		sb.WriteString(fmt.Sprintf("### %s (%d policies)\n\n", result.ScenarioName, result.PolicyCount))
		sb.WriteString(fmt.Sprintf("- **Samples**: %d\n", result.Stats.Count))
		sb.WriteString(fmt.Sprintf("- **Min**: %v\n", result.Stats.Min))
		sb.WriteString(fmt.Sprintf("- **Max**: %v\n", result.Stats.Max))
		sb.WriteString(fmt.Sprintf("- **Mean**: %v\n", result.Stats.Mean))
		sb.WriteString(fmt.Sprintf("- **Median (P50)**: %v\n", result.Stats.Median))
		sb.WriteString(fmt.Sprintf("- **P95**: %v\n", result.Stats.P95))
		sb.WriteString(fmt.Sprintf("- **P99**: %v\n", result.Stats.P99))
		sb.WriteString(fmt.Sprintf("- **P99.9**: %v\n", result.Stats.P999))
		sb.WriteString(fmt.Sprintf("- **Std Dev**: %v\n", result.Stats.StdDev))
		sb.WriteString(fmt.Sprintf("- **Throughput**: %.2f ops/s\n", result.Stats.Throughput))

		if result.Notes != "" {
			sb.WriteString(fmt.Sprintf("\n**Notes**: %s\n", result.Notes))
		}
		sb.WriteString("\n")
	}

	// Comparison with Baseline
	if report.Baseline != nil {
		sb.WriteString("## Comparison with Baseline\n\n")
		sb.WriteString("| Scenario | Mean Diff | Median Diff | P95 Diff | P99 Diff | Status |\n")
		sb.WriteString("|----------|-----------|-------------|----------|----------|--------|\n")

		for _, result := range report.Results {
			if result.Comparison != nil {
				status := "NEUTRAL"
				emoji := "➖"
				if result.Comparison.Improved {
					status = "IMPROVED"
					emoji = "✅"
				} else if result.Comparison.Regressed {
					status = "REGRESSED"
					emoji = "⚠️"
				}

				sb.WriteString(fmt.Sprintf("| %s | %+.1f%% | %+.1f%% | %+.1f%% | %+.1f%% | %s %s |\n",
					result.ScenarioName,
					result.Comparison.MeanDiff,
					result.Comparison.MedianDiff,
					result.Comparison.P95Diff,
					result.Comparison.P99Diff,
					status,
					emoji,
				))
			}
		}
		sb.WriteString("\n")
	}

	// Summary
	if report.Summary != "" {
		sb.WriteString("## Summary\n\n")
		sb.WriteString(report.Summary)
		sb.WriteString("\n\n")
	}

	// Analysis and Observations
	sb.WriteString("## Analysis\n\n")
	sb.WriteString(generateAnalysis(report))

	// Recommendations
	if len(report.Recommendations) > 0 {
		sb.WriteString("## Recommendations\n\n")
		for i, rec := range report.Recommendations {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, rec))
		}
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString("---\n\n")
	sb.WriteString("*This report was automatically generated by the eBPF microsegmentation benchmark tool.*\n")

	return sb.String()
}

// generateAnalysis generates automatic analysis based on benchmark results
func generateAnalysis(report *BenchmarkReport) string {
	var analysis strings.Builder

	// Analyze scaling behavior
	if len(report.Results) >= 2 {
		analysis.WriteString("### Scaling Behavior\n\n")

		// Compare latency increase as policy count grows
		firstResult := report.Results[0]
		lastResult := report.Results[len(report.Results)-1]

		policyRatio := float64(lastResult.PolicyCount) / float64(firstResult.PolicyCount)
		latencyRatio := float64(lastResult.Stats.Mean) / float64(firstResult.Stats.Mean)

		analysis.WriteString(fmt.Sprintf("- Policy count increased by %.1fx (%d → %d policies)\n",
			policyRatio, firstResult.PolicyCount, lastResult.PolicyCount))
		analysis.WriteString(fmt.Sprintf("- Mean latency increased by %.1fx (%v → %v)\n",
			latencyRatio, firstResult.Stats.Mean, lastResult.Stats.Mean))

		if latencyRatio < policyRatio {
			analysis.WriteString("- **Performance**: Sub-linear scaling detected ✅ (likely due to early termination optimization)\n")
		} else if latencyRatio > policyRatio {
			analysis.WriteString("- **Performance**: Super-linear scaling detected ⚠️ (may indicate cache effects or other overhead)\n")
		} else {
			analysis.WriteString("- **Performance**: Linear scaling (O(n)) as expected for linear scan\n")
		}
		analysis.WriteString("\n")
	}

	// Analyze latency distribution
	analysis.WriteString("### Latency Distribution\n\n")
	for _, result := range report.Results {
		p99ToMedianRatio := float64(result.Stats.P99) / float64(result.Stats.Median)

		analysis.WriteString(fmt.Sprintf("- **%s**: ", result.ScenarioName))
		if p99ToMedianRatio < 2.0 {
			analysis.WriteString("Very consistent performance (P99/P50 < 2.0) ✅\n")
		} else if p99ToMedianRatio < 5.0 {
			analysis.WriteString("Good consistency (P99/P50 < 5.0)\n")
		} else {
			analysis.WriteString(fmt.Sprintf("High variance (P99/P50 = %.1f) - investigate outliers ⚠️\n", p99ToMedianRatio))
		}
	}
	analysis.WriteString("\n")

	// Performance bottleneck identification
	analysis.WriteString("### Performance Bottlenecks\n\n")
	analysis.WriteString("1. **Wildcard Policy Matching**: Linear scan O(n) is the primary bottleneck\n")
	analysis.WriteString("2. **Early Termination**: Current optimization (Task #18) helps reduce worst-case latency\n")
	analysis.WriteString("3. **Future Optimization**: Consider LPM Trie (Phase 2) for O(log n) lookups\n\n")

	return analysis.String()
}

// SaveReport saves the report to a file
func SaveReport(report *BenchmarkReport, filename string) error {
	content := GenerateMarkdownReport(report)
	return os.WriteFile(filename, []byte(content), 0644)
}

// GenerateBaselineComparison generates a comparison report between current and baseline
func GenerateBaselineComparison(current, baseline *BenchmarkReport) string {
	var sb strings.Builder

	sb.WriteString("# Benchmark Comparison Report\n\n")
	sb.WriteString(fmt.Sprintf("**Current Test**: %s\n", current.TestDate.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("**Baseline Test**: %s\n\n", baseline.TestDate.Format("2006-01-02 15:04:05")))

	sb.WriteString("## Performance Changes\n\n")
	sb.WriteString("| Scenario | Mean Δ | P95 Δ | P99 Δ | Status |\n")
	sb.WriteString("|----------|--------|-------|-------|--------|\n")

	// Match scenarios and compare
	for _, currResult := range current.Results {
		for _, baseResult := range baseline.Results {
			if currResult.ScenarioName == baseResult.ScenarioName {
				comp := CompareWithBaseline(baseResult.Stats, currResult.Stats, 0.10) // 10% regression threshold

				status := "🟢 SAME"
				if comp.Improved {
					status = "✅ IMPROVED"
				} else if comp.Regressed {
					status = "⚠️ REGRESSED"
				}

				sb.WriteString(fmt.Sprintf("| %s | %+.1f%% | %+.1f%% | %+.1f%% | %s |\n",
					currResult.ScenarioName,
					comp.MeanDiff,
					comp.P95Diff,
					comp.P99Diff,
					status,
				))
				break
			}
		}
	}

	return sb.String()
}
