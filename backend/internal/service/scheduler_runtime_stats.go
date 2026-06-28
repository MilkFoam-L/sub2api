package service

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

const schedulerRuntimeStatsAlpha = 0.2

type schedulerAccountRuntimeStat struct {
	errorRateEWMABits atomic.Uint64
	latencyEWMABits   atomic.Uint64
	consecutiveFails  atomic.Int64
	lastSuccessUnixMS atomic.Int64
	lastFailureUnixMS atomic.Int64
	slowStartUntilMS  atomic.Int64
}

type schedulerAccountRuntimeSnapshot struct {
	ErrorRate           float64
	LatencyMs           float64
	HasLatency          bool
	ConsecutiveFailures int
	LastSuccess         time.Time
	LastFailure         time.Time
	HasSlowStart        bool
	SlowStartUntil      time.Time
}

type schedulerAccountRuntimeStats struct {
	accounts sync.Map
	count    atomic.Int64
}

func newSchedulerAccountRuntimeStats() *schedulerAccountRuntimeStats {
	return &schedulerAccountRuntimeStats{}
}

func (s *schedulerAccountRuntimeStats) loadOrCreate(accountID int64) *schedulerAccountRuntimeStat {
	if s == nil || accountID <= 0 {
		return nil
	}
	if value, ok := s.accounts.Load(accountID); ok {
		stat, _ := value.(*schedulerAccountRuntimeStat)
		return stat
	}
	stat := &schedulerAccountRuntimeStat{}
	stat.latencyEWMABits.Store(math.Float64bits(math.NaN()))
	actual, loaded := s.accounts.LoadOrStore(accountID, stat)
	if loaded {
		stat, _ = actual.(*schedulerAccountRuntimeStat)
		return stat
	}
	s.count.Add(1)
	return stat
}

func (s *schedulerAccountRuntimeStats) report(accountID int64, success bool, latencyMs *int) {
	stat := s.loadOrCreate(accountID)
	if stat == nil {
		return
	}
	nowMS := time.Now().UnixMilli()
	if success {
		stat.lastSuccessUnixMS.Store(nowMS)
		stat.consecutiveFails.Store(0)
	} else {
		stat.lastFailureUnixMS.Store(nowMS)
		stat.consecutiveFails.Add(1)
	}

	errorSample := 1.0
	if success {
		errorSample = 0
	}
	updateSchedulerEWMA(&stat.errorRateEWMABits, errorSample, schedulerRuntimeStatsAlpha)

	if latencyMs != nil && *latencyMs > 0 {
		updateSchedulerEWMA(&stat.latencyEWMABits, float64(*latencyMs), schedulerRuntimeStatsAlpha)
	}
}

func (s *schedulerAccountRuntimeStats) markSlowStart(accountID int64, duration time.Duration) {
	if duration <= 0 {
		return
	}
	stat := s.loadOrCreate(accountID)
	if stat == nil {
		return
	}
	stat.slowStartUntilMS.Store(time.Now().Add(duration).UnixMilli())
}

func (s *schedulerAccountRuntimeStats) snapshot(accountID int64) *schedulerAccountRuntimeSnapshot {
	snapshot := &schedulerAccountRuntimeSnapshot{}
	if s == nil || accountID <= 0 {
		return snapshot
	}
	value, ok := s.accounts.Load(accountID)
	if !ok {
		return snapshot
	}
	stat, _ := value.(*schedulerAccountRuntimeStat)
	if stat == nil {
		return snapshot
	}
	snapshot.ErrorRate = clampSchedulerUnit(math.Float64frombits(stat.errorRateEWMABits.Load()))
	latency := math.Float64frombits(stat.latencyEWMABits.Load())
	if !math.IsNaN(latency) && latency > 0 {
		snapshot.LatencyMs = latency
		snapshot.HasLatency = true
	}
	snapshot.ConsecutiveFailures = int(stat.consecutiveFails.Load())
	if ms := stat.lastSuccessUnixMS.Load(); ms > 0 {
		snapshot.LastSuccess = time.UnixMilli(ms)
	}
	if ms := stat.lastFailureUnixMS.Load(); ms > 0 {
		snapshot.LastFailure = time.UnixMilli(ms)
	}
	if ms := stat.slowStartUntilMS.Load(); ms > 0 {
		until := time.UnixMilli(ms)
		if time.Now().Before(until) {
			snapshot.HasSlowStart = true
			snapshot.SlowStartUntil = until
		}
	}
	return snapshot
}

func updateSchedulerEWMA(bits *atomic.Uint64, sample float64, alpha float64) {
	if bits == nil || math.IsNaN(sample) || math.IsInf(sample, 0) {
		return
	}
	if alpha <= 0 || alpha > 1 {
		alpha = schedulerRuntimeStatsAlpha
	}
	for {
		oldBits := bits.Load()
		oldValue := math.Float64frombits(oldBits)
		newValue := sample
		if !math.IsNaN(oldValue) {
			newValue = alpha*sample + (1-alpha)*oldValue
		}
		if bits.CompareAndSwap(oldBits, math.Float64bits(newValue)) {
			return
		}
	}
}

func clampSchedulerUnit(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
