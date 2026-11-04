package service

import (
	"math"
	"time"
)

// computeDashboardHealthScore computes a 0-100 health score from the metrics returned by the dashboard overview.
//
// Design goals:
// - Backend-owned scoring (UI only displays).
// - Layered scoring: Business Health (70%) + Infrastructure Health (30%)
// - Avoids double-counting (e.g., DB failure affects both infra and business metrics)
// - Conservative + stable: penalize clear degradations; avoid overreacting to missing/idle data.
func computeDashboardHealthScore(now time.Time, overview *OpsDashboardOverview) int {
	return computeDashboardHealthScoreWithThresholds(now, overview, nil)
}

// computeDashboardHealthScoreWithThresholds scores business health against the operator's metric
// thresholds (the same ones that turn the dashboard cards red). A metric at its threshold scores 0;
// an error rate at a tenth of its threshold, or a TTFT P99 at a third of its threshold, scores 100.
// A missing threshold falls back to the built-in curve: errors 1% -> 100, 10% -> 0; TTFT 1s -> 100, 3s -> 0.
func computeDashboardHealthScoreWithThresholds(now time.Time, overview *OpsDashboardOverview, thresholds *OpsMetricThresholds) int {
	if overview == nil {
		return 0
	}

	// Idle/no-data: avoid showing a "bad" score when there is no traffic.
	// UI can still render a gray/idle state based on QPS + error rate.
	if overview.RequestCountSLA <= 0 && overview.RequestCountTotal <= 0 && overview.ErrorCountTotal <= 0 {
		return 100
	}

	businessHealth := computeBusinessHealthWithThresholds(overview, thresholds)
	infraHealth := computeInfraHealth(now, overview)

	// Weighted combination: 70% business + 30% infrastructure
	score := businessHealth*0.7 + infraHealth*0.3
	return int(math.Round(clampFloat64(score, 0, 100)))
}

// computeBusinessHealth calculates business health score (0-100) on the built-in curves.
func computeBusinessHealth(overview *OpsDashboardOverview) float64 {
	return computeBusinessHealthWithThresholds(overview, nil)
}

// computeBusinessHealthWithThresholds calculates business health score (0-100)
// Components: Error Rate (50%) + TTFT (50%)
func computeBusinessHealthWithThresholds(overview *OpsDashboardOverview, thresholds *OpsMetricThresholds) float64 {
	const (
		defaultErrorRatePercentMax = 10.0
		defaultTTFTp99MsMax        = 3000.0
	)
	requestErrMax, upstreamErrMax, ttftMax := defaultErrorRatePercentMax, defaultErrorRatePercentMax, defaultTTFTp99MsMax
	if thresholds != nil {
		requestErrMax = positiveOr(thresholds.RequestErrorRatePercentMax, requestErrMax)
		upstreamErrMax = positiveOr(thresholds.UpstreamErrorRatePercentMax, upstreamErrMax)
		ttftMax = positiveOr(thresholds.TTFTp99MsMax, ttftMax)
	}

	// Error rate score: each rate against its own threshold, the worse one counts.
	errorPct := clampFloat64(overview.ErrorRate*100, 0, 100)
	upstreamPct := clampFloat64(overview.UpstreamErrorRate*100, 0, 100)
	errorScore := math.Min(
		linearScore(errorPct, requestErrMax/10, requestErrMax),
		linearScore(upstreamPct, upstreamErrMax/10, upstreamErrMax),
	)

	// TTFT score: time to first token is critical for user experience.
	ttftScore := 100.0
	if overview.TTFT.P99 != nil {
		ttftScore = linearScore(float64(*overview.TTFT.P99), ttftMax/3, ttftMax)
	}

	// Weighted combination: 50% error rate + 50% TTFT
	return errorScore*0.5 + ttftScore*0.5
}

// linearScore is 100 at or below good, 0 at or above bad, and linear in between.
func linearScore(value, good, bad float64) float64 {
	if value <= good {
		return 100
	}
	if value >= bad {
		return 0
	}
	return (bad - value) / (bad - good) * 100
}

func positiveOr(v *float64, fallback float64) float64 {
	if v != nil && *v > 0 {
		return *v
	}
	return fallback
}

// computeInfraHealth calculates infrastructure health score (0-100)
// Components: Storage (40%) + Compute Resources (30%) + Background Jobs (30%)
func computeInfraHealth(now time.Time, overview *OpsDashboardOverview) float64 {
	// Storage score: DB critical, Redis less critical
	storageScore := 100.0
	if overview.SystemMetrics != nil {
		if overview.SystemMetrics.DBOK != nil && !*overview.SystemMetrics.DBOK {
			storageScore = 0 // DB failure is critical
		} else if overview.SystemMetrics.RedisOK != nil && !*overview.SystemMetrics.RedisOK {
			storageScore = 50 // Redis failure is degraded but not critical
		}
	}

	// Compute resources score: CPU + Memory
	computeScore := 100.0
	if overview.SystemMetrics != nil {
		cpuScore := 100.0
		if overview.SystemMetrics.CPUUsagePercent != nil {
			cpuPct := clampFloat64(*overview.SystemMetrics.CPUUsagePercent, 0, 100)
			if cpuPct > 80 {
				if cpuPct <= 100 {
					cpuScore = (100 - cpuPct) / 20 * 100
				} else {
					cpuScore = 0
				}
			}
		}

		memScore := 100.0
		if overview.SystemMetrics.MemoryUsagePercent != nil {
			memPct := clampFloat64(*overview.SystemMetrics.MemoryUsagePercent, 0, 100)
			if memPct > 85 {
				if memPct <= 100 {
					memScore = (100 - memPct) / 15 * 100
				} else {
					memScore = 0
				}
			}
		}

		computeScore = (cpuScore + memScore) / 2
	}

	// Background jobs score
	jobScore := 100.0
	failedJobs := 0
	totalJobs := 0
	for _, hb := range overview.JobHeartbeats {
		if hb == nil {
			continue
		}
		totalJobs++
		if hb.LastErrorAt != nil && (hb.LastSuccessAt == nil || hb.LastErrorAt.After(*hb.LastSuccessAt)) {
			failedJobs++
		} else if hb.LastSuccessAt != nil && now.Sub(*hb.LastSuccessAt) > 15*time.Minute {
			failedJobs++
		}
	}
	if totalJobs > 0 && failedJobs > 0 {
		jobScore = (1 - float64(failedJobs)/float64(totalJobs)) * 100
	}

	// Weighted combination
	return storageScore*0.4 + computeScore*0.3 + jobScore*0.3
}

func clampFloat64(v float64, min float64, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
