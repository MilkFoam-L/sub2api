package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSchedulerAccountRuntimeStatsReportsErrorRateEWMA(t *testing.T) {
	stats := newSchedulerAccountRuntimeStats()

	stats.report(101, false, nil)
	stats.report(101, false, nil)

	snapshot := stats.snapshot(101)
	require.Greater(t, snapshot.ErrorRate, 0.0)
	require.LessOrEqual(t, snapshot.ErrorRate, 1.0)
	require.Equal(t, 2, snapshot.ConsecutiveFailures)

	stats.report(101, true, nil)
	snapshot = stats.snapshot(101)
	require.Zero(t, snapshot.ConsecutiveFailures)
}

func TestSchedulerAccountRuntimeStatsReportsLatencyEWMA(t *testing.T) {
	stats := newSchedulerAccountRuntimeStats()
	firstLatency := 1000
	secondLatency := 3000

	stats.report(202, true, &firstLatency)
	stats.report(202, true, &secondLatency)

	snapshot := stats.snapshot(202)
	require.True(t, snapshot.HasLatency)
	require.Greater(t, snapshot.LatencyMs, float64(firstLatency))
	require.Less(t, snapshot.LatencyMs, float64(secondLatency))
}

func TestSchedulerAccountRuntimeStatsColdAccountHasNoPenaltySignals(t *testing.T) {
	stats := newSchedulerAccountRuntimeStats()

	snapshot := stats.snapshot(303)

	require.Zero(t, snapshot.ErrorRate)
	require.False(t, snapshot.HasLatency)
	require.Zero(t, snapshot.ConsecutiveFailures)
}

func TestSchedulerAccountRuntimeStatsMarksSlowStart(t *testing.T) {
	stats := newSchedulerAccountRuntimeStats()

	stats.markSlowStart(404, 5*time.Minute)

	snapshot := stats.snapshot(404)
	require.True(t, snapshot.HasSlowStart)
	require.True(t, snapshot.SlowStartUntil.After(time.Now().Add(4*time.Minute)))
}
