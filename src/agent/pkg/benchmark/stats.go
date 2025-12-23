// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// input: benchmark test data
// output: performance statistics
// pos: benchmark statistics collector - if file updated, must sync with this header comment and pkg/benchmark/CLAUDE.md
package benchmark

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// Stats contains statistical metrics for performance benchmarking
type Stats struct {
	Count   int
	Min     time.Duration
	Max     time.Duration
	Mean    time.Duration
	Median  time.Duration // P50
	P95     time.Duration
	P99     time.Duration
	P999    time.Duration
	StdDev  time.Duration

	// Throughput metrics
	TotalDuration time.Duration
	Throughput    float64 // operations per second
}

// String returns a human-readable representation of the stats
func (s *Stats) String() string {
	return fmt.Sprintf(
		"Count: %d, Min: %v, Max: %v, Mean: %v, P50: %v, P95: %v, P99: %v, P999: %v, StdDev: %v, Throughput: %.2f ops/s",
		s.Count, s.Min, s.Max, s.Mean, s.Median, s.P95, s.P99, s.P999, s.StdDev, s.Throughput,
	)
}

// CalculateStats computes statistical metrics from a slice of durations
func CalculateStats(samples []time.Duration) *Stats {
	if len(samples) == 0 {
		return &Stats{}
	}

	// Sort samples for percentile calculation
	sorted := make([]time.Duration, len(samples))
	copy(sorted, samples)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	stats := &Stats{
		Count: len(samples),
		Min:   sorted[0],
		Max:   sorted[len(sorted)-1],
	}

	// Calculate mean
	var sum time.Duration
	for _, d := range samples {
		sum += d
	}
	stats.Mean = sum / time.Duration(len(samples))

	// Calculate percentiles
	stats.Median = percentile(sorted, 50)
	stats.P95 = percentile(sorted, 95)
	stats.P99 = percentile(sorted, 99)
	stats.P999 = percentile(sorted, 99.9)

	// Calculate standard deviation
	var variance float64
	meanNs := float64(stats.Mean.Nanoseconds())
	for _, d := range samples {
		diff := float64(d.Nanoseconds()) - meanNs
		variance += diff * diff
	}
	variance /= float64(len(samples))
	stats.StdDev = time.Duration(math.Sqrt(variance))

	// Calculate total duration
	stats.TotalDuration = sum

	// Calculate throughput
	if stats.TotalDuration > 0 {
		stats.Throughput = float64(len(samples)) / stats.TotalDuration.Seconds()
	}

	return stats
}

// percentile calculates the nth percentile from sorted samples
func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}

	// Linear interpolation between closest ranks
	rank := p / 100.0 * float64(len(sorted)-1)
	lowerIndex := int(math.Floor(rank))
	upperIndex := int(math.Ceil(rank))

	if lowerIndex == upperIndex {
		return sorted[lowerIndex]
	}

	lowerValue := float64(sorted[lowerIndex].Nanoseconds())
	upperValue := float64(sorted[upperIndex].Nanoseconds())
	fraction := rank - float64(lowerIndex)

	interpolated := lowerValue + fraction*(upperValue-lowerValue)
	return time.Duration(interpolated)
}

// CompareStats compares two sets of statistics and returns the percentage difference
type StatComparison struct {
	Baseline *Stats
	Current  *Stats

	MeanDiff    float64 // Percentage difference
	MedianDiff  float64
	P95Diff     float64
	P99Diff     float64

	Improved   bool // True if current is better than baseline
	Regressed  bool // True if current is worse than baseline (> threshold)
	Threshold  float64 // Regression threshold (e.g., 0.10 for 10%)
}

// CompareWithBaseline compares current stats with baseline and detects regressions
func CompareWithBaseline(baseline, current *Stats, regressionThreshold float64) *StatComparison {
	comp := &StatComparison{
		Baseline:  baseline,
		Current:   current,
		Threshold: regressionThreshold,
	}

	if baseline.Mean > 0 {
		comp.MeanDiff = (float64(current.Mean-baseline.Mean) / float64(baseline.Mean)) * 100
	}
	if baseline.Median > 0 {
		comp.MedianDiff = (float64(current.Median-baseline.Median) / float64(baseline.Median)) * 100
	}
	if baseline.P95 > 0 {
		comp.P95Diff = (float64(current.P95-baseline.P95) / float64(baseline.P95)) * 100
	}
	if baseline.P99 > 0 {
		comp.P99Diff = (float64(current.P99-baseline.P99) / float64(baseline.P99)) * 100
	}

	// Check if improved (negative difference means faster)
	comp.Improved = comp.MeanDiff < 0

	// Check if regressed (positive difference exceeds threshold)
	comp.Regressed = comp.MeanDiff > (regressionThreshold * 100)

	return comp
}

// String returns a human-readable comparison
func (sc *StatComparison) String() string {
	status := "NEUTRAL"
	if sc.Improved {
		status = "IMPROVED ✅"
	} else if sc.Regressed {
		status = "REGRESSED ⚠️"
	}

	return fmt.Sprintf(
		"Status: %s, Mean: %+.1f%%, Median: %+.1f%%, P95: %+.1f%%, P99: %+.1f%%",
		status, sc.MeanDiff, sc.MedianDiff, sc.P95Diff, sc.P99Diff,
	)
}

// LatencyHistogram represents a histogram of latency samples
type LatencyHistogram struct {
	Buckets []HistogramBucket
}

// HistogramBucket represents a single bucket in the histogram
type HistogramBucket struct {
	Min   time.Duration
	Max   time.Duration
	Count int
}

// CreateHistogram creates a latency histogram from samples
func CreateHistogram(samples []time.Duration, numBuckets int) *LatencyHistogram {
	if len(samples) == 0 || numBuckets <= 0 {
		return &LatencyHistogram{}
	}

	// Find min and max
	minVal := samples[0]
	maxVal := samples[0]
	for _, s := range samples {
		if s < minVal {
			minVal = s
		}
		if s > maxVal {
			maxVal = s
		}
	}

	// Create buckets
	bucketSize := (maxVal - minVal) / time.Duration(numBuckets)
	if bucketSize == 0 {
		bucketSize = 1
	}

	histogram := &LatencyHistogram{
		Buckets: make([]HistogramBucket, numBuckets),
	}

	for i := 0; i < numBuckets; i++ {
		histogram.Buckets[i] = HistogramBucket{
			Min: minVal + time.Duration(i)*bucketSize,
			Max: minVal + time.Duration(i+1)*bucketSize,
		}
	}

	// Count samples in each bucket
	for _, sample := range samples {
		for i := range histogram.Buckets {
			if sample >= histogram.Buckets[i].Min && sample < histogram.Buckets[i].Max {
				histogram.Buckets[i].Count++
				break
			}
			// Handle the last bucket edge case
			if i == len(histogram.Buckets)-1 && sample >= histogram.Buckets[i].Min {
				histogram.Buckets[i].Count++
			}
		}
	}

	return histogram
}

// AggregateStats aggregates multiple Stats into a single summary
type AggregatedStats struct {
	Runs       int
	AvgMean    time.Duration
	AvgMedian  time.Duration
	AvgP95     time.Duration
	AvgP99     time.Duration
	MinOfMeans time.Duration
	MaxOfMeans time.Duration
}

// AggregateMultipleStats combines statistics from multiple benchmark runs
func AggregateMultipleStats(allStats []*Stats) *AggregatedStats {
	if len(allStats) == 0 {
		return &AggregatedStats{}
	}

	agg := &AggregatedStats{
		Runs: len(allStats),
	}

	var sumMean, sumMedian, sumP95, sumP99 time.Duration
	agg.MinOfMeans = allStats[0].Mean
	agg.MaxOfMeans = allStats[0].Mean

	for _, s := range allStats {
		sumMean += s.Mean
		sumMedian += s.Median
		sumP95 += s.P95
		sumP99 += s.P99

		if s.Mean < agg.MinOfMeans {
			agg.MinOfMeans = s.Mean
		}
		if s.Mean > agg.MaxOfMeans {
			agg.MaxOfMeans = s.Mean
		}
	}

	agg.AvgMean = sumMean / time.Duration(len(allStats))
	agg.AvgMedian = sumMedian / time.Duration(len(allStats))
	agg.AvgP95 = sumP95 / time.Duration(len(allStats))
	agg.AvgP99 = sumP99 / time.Duration(len(allStats))

	return agg
}
