# eBPF Policy Lookup Benchmark Guide

This guide explains how to run and interpret the eBPF wildcard policy lookup performance benchmarks.

## Overview

The benchmark framework tests the performance of eBPF wildcard policy matching under different scenarios:
- **Worst Case**: Matching policy is at the end of the list (tests full linear scan)
- **Best Case**: Matching policy is first in the list (tests early termination)
- **Average Case**: Matching policy is in the middle (tests typical performance)

## Prerequisites

### System Requirements
- Linux kernel >= 4.15 (eBPF support)
- Root/sudo access (for eBPF operations)
- Go >= 1.21
- Network namespace support

### Build Requirements
```bash
# Install dependencies
go mod download

# Compile eBPF programs
make bpf
```

## Running Benchmarks

### Quick Start

Run all policy lookup benchmarks:
```bash
cd src/agent
sudo go test -bench=BenchmarkPolicyLookup ./test/benchmark -benchtime=10s
```

### Individual Scenarios

**Test with 10 policies:**
```bash
sudo go test -bench=BenchmarkPolicyLookup_10Policies ./test/benchmark -benchtime=10s
```

**Test with 50 policies:**
```bash
sudo go test -bench=BenchmarkPolicyLookup_50Policies ./test/benchmark -benchtime=10s
```

**Test with 100 policies:**
```bash
sudo go test -bench=BenchmarkPolicyLookup_100Policies ./test/benchmark -benchtime=10s
```

**Test different scenarios:**
```bash
# Worst case (default)
sudo go test -bench=BenchmarkPolicyLookup_50Policies ./test/benchmark

# Best case
sudo go test -bench=BenchmarkPolicyLookup_BestCase_50Policies ./test/benchmark

# Average case
sudo go test -bench=BenchmarkPolicyLookup_AverageCase_50Policies ./test/benchmark
```

### Generate Comprehensive Report

Generate a full performance report with statistics:
```bash
cd src/agent
sudo go test -v -run=TestPolicyLookup_GenerateReport ./test/benchmark
```

This will create `docs/performance/BENCHMARK_BASELINE.md` with detailed performance analysis.

## Understanding Output

### Benchmark Output Format

```
BenchmarkPolicyLookup_50Policies-4    1000    12345 ns/op    p50_μs:10.5    p95_μs:15.2    p99_μs:18.9
```

Fields:
- `BenchmarkPolicyLookup_50Policies`: Test name
- `-4`: Number of CPU cores
- `1000`: Number of iterations
- `12345 ns/op`: Average time per operation (nanoseconds)
- `p50_μs`: Median latency (microseconds)
- `p95_μs`: 95th percentile latency
- `p99_μs`: 99th percentile latency

### Performance Metrics

**Key Metrics:**
- **P50 (Median)**: Typical performance
- **P95**: Performance for 95% of requests
- **P99**: Worst-case for 99% of requests
- **Mean**: Average latency
- **Throughput**: Operations per second

**What to Look For:**
- P99/P50 ratio < 2.0: Very consistent performance ✅
- P99/P50 ratio < 5.0: Good consistency
- P99/P50 ratio > 5.0: High variance, investigate outliers ⚠️

## Benchmark Scenarios

### Scenario 1: Worst Case (Default)
**Purpose**: Test maximum latency when all policies must be scanned

**Policy Configuration:**
- Policies 0 to N-2: Non-matching (different CIDR ranges)
- Policy N-1: Catch-all (matches any traffic)

**Expected Behavior:**
- Linear scaling O(n) with policy count
- Tests effectiveness of early termination optimization (Task #18)

### Scenario 2: Best Case
**Purpose**: Test minimum latency with immediate match

**Policy Configuration:**
- Policy 0: Catch-all (matches any traffic)
- Policies 1 to N-1: Non-matching

**Expected Behavior:**
- Constant time O(1)
- Validates early termination works correctly

### Scenario 3: Average Case
**Purpose**: Test typical performance

**Policy Configuration:**
- Policies 0 to N/2-1: Non-matching
- Policy N/2: Catch-all
- Policies N/2+1 to N-1: Non-matching

**Expected Behavior:**
- ~50% of worst-case latency
- Represents realistic mixed workload

## Performance Baselines

### Target Performance (After Task #18 Optimization)

| Policy Count | P50 (μs) | P95 (μs) | P99 (μs) | Notes |
|--------------|----------|----------|----------|-------|
| 10           | < 2.0    | < 3.0    | < 4.0    | Excellent |
| 50           | < 7.0    | < 10.0   | < 12.0   | Good |
| 100          | < 15.0   | < 20.0   | < 25.0   | Acceptable |

**Performance Goals:**
- Linear scaling: Latency should scale linearly with policy count
- Early termination: 50% improvement when matching policy is first
- Consistency: P99/P50 ratio < 3.0

## Troubleshooting

### Common Issues

**Issue**: "E2E requirements not met"
```
Solution: Ensure you have root/sudo access and network namespace support
```

**Issue**: Benchmark hangs or takes too long
```
Solution: Reduce benchtime (default is 10s)
sudo go test -bench=Benchmark... -benchtime=3s
```

**Issue**: High variance in results
```
Solution:
1. Ensure system is not under load
2. Disable CPU frequency scaling
3. Run multiple times and average results
```

### System Optimization for Benchmarking

**Disable CPU frequency scaling:**
```bash
# Set CPU governor to performance mode
sudo cpupower frequency-set --governor performance

# Disable turbo boost
echo 0 | sudo tee /sys/devices/system/cpu/cpufreq/boost
```

**Isolate CPU cores (advanced):**
```bash
# Add to kernel boot parameters
isolcpus=2,3

# Pin benchmark to isolated cores
taskset -c 2,3 go test -bench=...
```

## Continuous Integration

### Adding to CI/CD

**GitHub Actions Example:**
```yaml
- name: Performance Benchmark
  run: |
    cd src/agent
    sudo go test -bench=BenchmarkPolicyLookup \
      ./test/benchmark \
      -benchtime=5s \
      | tee benchmark_results.txt

    # Compare with baseline
    if [ -f baseline.txt ]; then
      go run tools/compare_benchmarks.go \
        --baseline baseline.txt \
        --current benchmark_results.txt \
        --threshold 10%
    fi
```

## Interpreting Results

### Example Analysis

**Scenario**: 50 policies, worst case

```
Samples:    1000
P50 (μs):   5.5
P95 (μs):   8.2
P99 (μs):   9.5
Mean (μs):  6.1
Max (μs):   11.0
Throughput: 163,934.43 ops/s
```

**Analysis:**
1. **Good Performance**: P50 = 5.5μs is well within 7μs target
2. **Consistent**: P99/P50 = 1.73 (< 2.0) shows very consistent performance
3. **Early Stop Working**: Latency suggests early termination is effective
4. **Throughput**: ~164k ops/s is excellent for policy lookup

### Performance Regression Detection

**Baseline Comparison:**
```
| Metric | Baseline | Current | Change | Status |
|--------|----------|---------|--------|--------|
| Mean   | 6.0μs    | 5.5μs   | -8.3%  | ✅ IMPROVED |
| P95    | 8.5μs    | 8.2μs   | -3.5%  | ✅ IMPROVED |
| P99    | 10.0μs   | 9.5μs   | -5.0%  | ✅ IMPROVED |
```

**Regression Threshold**: > 10% increase triggers warning ⚠️

## Related Documentation

- [Task #18: eBPF Loop Optimization](../../.claude/epics/wildcard-policy-performance-optimization/18.md)
- [Performance Baseline Report](BENCHMARK_BASELINE.md)
- [Wildcard Policy Performance PRD](../../.claude/prds/wildcard-policy-performance-optimization.md)

## Contact

For questions or issues with benchmarking:
- Check existing GitHub issues
- Review benchmark code in `src/agent/test/benchmark/`
- Consult performance documentation in `docs/performance/`
