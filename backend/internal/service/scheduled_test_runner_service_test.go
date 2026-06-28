package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type scheduledRunnerPlanRepoFake struct {
	plans       []*ScheduledTestPlan
	updateCalls []scheduledRunnerUpdateCall
}

type scheduledRunnerUpdateCall struct {
	id        int64
	lastRunAt time.Time
	nextRunAt time.Time
}

func (r *scheduledRunnerPlanRepoFake) Create(context.Context, *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	return nil, errors.New("unexpected Create")
}
func (r *scheduledRunnerPlanRepoFake) GetByID(context.Context, int64) (*ScheduledTestPlan, error) {
	return nil, errors.New("unexpected GetByID")
}
func (r *scheduledRunnerPlanRepoFake) ListByAccountID(context.Context, int64) ([]*ScheduledTestPlan, error) {
	return nil, errors.New("unexpected ListByAccountID")
}
func (r *scheduledRunnerPlanRepoFake) ListDue(context.Context, time.Time) ([]*ScheduledTestPlan, error) {
	return r.plans, nil
}
func (r *scheduledRunnerPlanRepoFake) Update(context.Context, *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	return nil, errors.New("unexpected Update")
}
func (r *scheduledRunnerPlanRepoFake) Delete(context.Context, int64) error {
	return errors.New("unexpected Delete")
}
func (r *scheduledRunnerPlanRepoFake) UpdateAfterRun(_ context.Context, id int64, lastRunAt time.Time, nextRunAt time.Time) error {
	r.updateCalls = append(r.updateCalls, scheduledRunnerUpdateCall{id: id, lastRunAt: lastRunAt, nextRunAt: nextRunAt})
	return nil
}

type scheduledRunnerResultRepoFake struct {
	mu      sync.Mutex
	results []*ScheduledTestResult
}

func (r *scheduledRunnerResultRepoFake) Create(_ context.Context, result *ScheduledTestResult) (*ScheduledTestResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copyResult := *result
	copyResult.ID = int64(len(r.results) + 1)
	r.results = append([]*ScheduledTestResult{&copyResult}, r.results...)
	return &copyResult, nil
}
func (r *scheduledRunnerResultRepoFake) ListByPlanID(_ context.Context, planID int64, limit int) ([]*ScheduledTestResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*ScheduledTestResult, 0, limit)
	for _, result := range r.results {
		if result.PlanID != planID {
			continue
		}
		copyResult := *result
		out = append(out, &copyResult)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
func (r *scheduledRunnerResultRepoFake) PruneOldResults(context.Context, int64, int) error {
	return nil
}

type scheduledRunnerAccountTesterFake struct {
	results []*ScheduledTestResult
	calls   []int64
}

func (f *scheduledRunnerAccountTesterFake) RunTestBackground(_ context.Context, accountID int64, modelID string) (*ScheduledTestResult, error) {
	f.calls = append(f.calls, accountID)
	if len(f.results) == 0 {
		return nil, errors.New("missing fake result")
	}
	result := f.results[0]
	f.results = f.results[1:]
	return result, nil
}

type scheduledRunnerAccountPauserFake struct {
	calls []scheduledRunnerPauseCall
}

type scheduledRunnerPauseCall struct {
	accountID int64
	until     time.Time
	reason    string
}

func (f *scheduledRunnerAccountPauserFake) SetTempUnschedulable(_ context.Context, id int64, until time.Time, reason string) error {
	f.calls = append(f.calls, scheduledRunnerPauseCall{accountID: id, until: until, reason: reason})
	return nil
}

type scheduledRunnerRecovererFake struct {
	calls []int64
}

func (f *scheduledRunnerRecovererFake) RecoverAccountAfterSuccessfulTest(_ context.Context, accountID int64) (*SuccessfulTestRecoveryResult, error) {
	f.calls = append(f.calls, accountID)
	return &SuccessfulTestRecoveryResult{ClearedRateLimit: true}, nil
}

type scheduledRunnerSlowStartMarkerFake struct {
	calls []scheduledRunnerSlowStartCall
}

type scheduledRunnerSlowStartCall struct {
	accountID int64
	duration  time.Duration
}

func (f *scheduledRunnerSlowStartMarkerFake) MarkAccountSlowStart(accountID int64, duration time.Duration) {
	f.calls = append(f.calls, scheduledRunnerSlowStartCall{accountID: accountID, duration: duration})
}

func newScheduledRunnerTestService(resultRepo *scheduledRunnerResultRepoFake, tester *scheduledRunnerAccountTesterFake, pauser *scheduledRunnerAccountPauserFake, recoverer *scheduledRunnerRecovererFake, marker *scheduledRunnerSlowStartMarkerFake) (*ScheduledTestRunnerService, *scheduledRunnerPlanRepoFake) {
	planRepo := &scheduledRunnerPlanRepoFake{plans: []*ScheduledTestPlan{{
		ID:             11,
		AccountID:      101,
		ModelID:        "claude-sonnet-4-5",
		CronExpression: "* * * * *",
		Enabled:        true,
		MaxResults:     10,
		AutoRecover:    true,
	}}}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.ActiveProbe.AutoPauseEnabled = true
	cfg.Gateway.Scheduling.ActiveProbe.FailureThreshold = 3
	cfg.Gateway.Scheduling.ActiveProbe.PauseDuration = time.Minute
	cfg.Gateway.Scheduling.ActiveProbe.PauseDurationMax = 10 * time.Minute
	cfg.Gateway.Scheduling.SlowStart.Enabled = true
	cfg.Gateway.Scheduling.SlowStart.Duration = 5 * time.Minute
	cfg.Gateway.Scheduling.SlowStart.Penalty = 1
	return &ScheduledTestRunnerService{
		planRepo:           planRepo,
		scheduledSvc:       NewScheduledTestService(planRepo, resultRepo),
		accountTester:      tester,
		rateLimitRecoverer: recoverer,
		accountPauser:      pauser,
		slowStartMarker:    marker,
		cfg:                cfg,
	}, planRepo
}

func scheduledRunnerFailureResult() *ScheduledTestResult {
	now := time.Now()
	return &ScheduledTestResult{Status: "error", ErrorMessage: "probe failed", StartedAt: now, FinishedAt: now.Add(time.Second), LatencyMs: 1000}
}

func scheduledRunnerSuccessResult() *ScheduledTestResult {
	now := time.Now()
	return &ScheduledTestResult{Status: "success", ResponseText: "ok", StartedAt: now, FinishedAt: now.Add(time.Second), LatencyMs: 1000}
}

func TestScheduledTestRunnerActiveProbeDoesNotPauseBeforeThreshold(t *testing.T) {
	resultRepo := &scheduledRunnerResultRepoFake{}
	tester := &scheduledRunnerAccountTesterFake{results: []*ScheduledTestResult{scheduledRunnerFailureResult()}}
	pauser := &scheduledRunnerAccountPauserFake{}
	runner, planRepo := newScheduledRunnerTestService(resultRepo, tester, pauser, &scheduledRunnerRecovererFake{}, &scheduledRunnerSlowStartMarkerFake{})

	runner.runOnePlan(context.Background(), planRepo.plans[0])

	require.Len(t, tester.calls, 1)
	require.Empty(t, pauser.calls)
	require.Len(t, planRepo.updateCalls, 1)
}

func TestScheduledTestRunnerActiveProbePausesAfterConsecutiveFailures(t *testing.T) {
	resultRepo := &scheduledRunnerResultRepoFake{}
	tester := &scheduledRunnerAccountTesterFake{results: []*ScheduledTestResult{
		scheduledRunnerFailureResult(),
		scheduledRunnerFailureResult(),
		scheduledRunnerFailureResult(),
	}}
	pauser := &scheduledRunnerAccountPauserFake{}
	runner, planRepo := newScheduledRunnerTestService(resultRepo, tester, pauser, &scheduledRunnerRecovererFake{}, &scheduledRunnerSlowStartMarkerFake{})

	runner.runOnePlan(context.Background(), planRepo.plans[0])
	runner.runOnePlan(context.Background(), planRepo.plans[0])
	runner.runOnePlan(context.Background(), planRepo.plans[0])

	require.Len(t, tester.calls, 3)
	require.Len(t, pauser.calls, 1)
	require.Equal(t, int64(101), pauser.calls[0].accountID)
	require.Contains(t, pauser.calls[0].reason, "plan=11")
	require.Contains(t, pauser.calls[0].reason, "consecutive_failures=3")
	require.True(t, pauser.calls[0].until.After(time.Now()))
}

func TestScheduledTestRunnerActiveProbePauseDurationGrowsWithConsecutiveFailures(t *testing.T) {
	resultRepo := &scheduledRunnerResultRepoFake{}
	tester := &scheduledRunnerAccountTesterFake{results: []*ScheduledTestResult{
		scheduledRunnerFailureResult(),
		scheduledRunnerFailureResult(),
		scheduledRunnerFailureResult(),
		scheduledRunnerFailureResult(),
	}}
	pauser := &scheduledRunnerAccountPauserFake{}
	runner, planRepo := newScheduledRunnerTestService(resultRepo, tester, pauser, &scheduledRunnerRecovererFake{}, &scheduledRunnerSlowStartMarkerFake{})

	runner.runOnePlan(context.Background(), planRepo.plans[0])
	runner.runOnePlan(context.Background(), planRepo.plans[0])
	runner.runOnePlan(context.Background(), planRepo.plans[0])
	runner.runOnePlan(context.Background(), planRepo.plans[0])

	require.Len(t, pauser.calls, 2)
	firstCooldown := time.Until(pauser.calls[0].until)
	secondCooldown := time.Until(pauser.calls[1].until)
	require.Greater(t, secondCooldown, firstCooldown+30*time.Second, "连续失败超过阈值后暂停时长应该递增")
}

func TestScheduledTestRunnerSuccessfulProbeRecoversAndMarksSlowStart(t *testing.T) {
	resultRepo := &scheduledRunnerResultRepoFake{}
	tester := &scheduledRunnerAccountTesterFake{results: []*ScheduledTestResult{scheduledRunnerSuccessResult()}}
	recoverer := &scheduledRunnerRecovererFake{}
	marker := &scheduledRunnerSlowStartMarkerFake{}
	runner, planRepo := newScheduledRunnerTestService(resultRepo, tester, &scheduledRunnerAccountPauserFake{}, recoverer, marker)

	runner.runOnePlan(context.Background(), planRepo.plans[0])

	require.Equal(t, []int64{101}, recoverer.calls)
	require.Len(t, marker.calls, 1)
	require.Equal(t, int64(101), marker.calls[0].accountID)
	require.Equal(t, 5*time.Minute, marker.calls[0].duration)
}
