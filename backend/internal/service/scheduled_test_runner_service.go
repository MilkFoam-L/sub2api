package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/robfig/cron/v3"
)

const scheduledTestDefaultMaxWorkers = 10

type scheduledTestAccountTester interface {
	RunTestBackground(ctx context.Context, accountID int64, modelID string) (*ScheduledTestResult, error)
}

type scheduledTestAccountRecoverer interface {
	RecoverAccountAfterSuccessfulTest(ctx context.Context, accountID int64) (*SuccessfulTestRecoveryResult, error)
}

type scheduledTestAccountPauser interface {
	SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error
}

// AccountSlowStartMarker marks accounts that should receive slow-start scheduling penalties.
type AccountSlowStartMarker interface {
	MarkAccountSlowStart(accountID int64, duration time.Duration)
}

// ScheduledTestRunnerService periodically scans due test plans and executes them.
type ScheduledTestRunnerService struct {
	planRepo           ScheduledTestPlanRepository
	scheduledSvc       *ScheduledTestService
	accountTester      scheduledTestAccountTester
	rateLimitRecoverer scheduledTestAccountRecoverer
	accountPauser      scheduledTestAccountPauser
	slowStartMarker    AccountSlowStartMarker
	cfg                *config.Config

	cron      *cron.Cron
	startOnce sync.Once
	stopOnce  sync.Once
}

// NewScheduledTestRunnerService creates a new runner.
func NewScheduledTestRunnerService(
	planRepo ScheduledTestPlanRepository,
	scheduledSvc *ScheduledTestService,
	accountTestSvc *AccountTestService,
	rateLimitSvc *RateLimitService,
	accountPauser scheduledTestAccountPauser,
	slowStartMarker AccountSlowStartMarker,
	cfg *config.Config,
) *ScheduledTestRunnerService {
	return &ScheduledTestRunnerService{
		planRepo:           planRepo,
		scheduledSvc:       scheduledSvc,
		accountTester:      accountTestSvc,
		rateLimitRecoverer: rateLimitSvc,
		accountPauser:      accountPauser,
		slowStartMarker:    slowStartMarker,
		cfg:                cfg,
	}
}

// Start begins the cron ticker (every minute).
func (s *ScheduledTestRunnerService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		loc := time.Local
		if s.cfg != nil {
			if parsed, err := time.LoadLocation(s.cfg.Timezone); err == nil && parsed != nil {
				loc = parsed
			}
		}

		c := cron.New(cron.WithParser(scheduledTestCronParser), cron.WithLocation(loc))
		_, err := c.AddFunc("* * * * *", func() { s.runScheduled() })
		if err != nil {
			logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] not started (invalid schedule): %v", err)
			return
		}
		s.cron = c
		s.cron.Start()
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] started (tick=every minute)")
	})
}

// Stop gracefully shuts down the cron scheduler.
func (s *ScheduledTestRunnerService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.cron != nil {
			ctx := s.cron.Stop()
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
				logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] cron stop timed out")
			}
		}
	})
}

func (s *ScheduledTestRunnerService) runScheduled() {
	// Delay 10s so execution lands at ~:10 of each minute instead of :00.
	time.Sleep(10 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	now := time.Now()
	plans, err := s.planRepo.ListDue(ctx, now)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] ListDue error: %v", err)
		return
	}
	if len(plans) == 0 {
		return
	}

	logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] found %d due plans", len(plans))

	sem := make(chan struct{}, scheduledTestDefaultMaxWorkers)
	var wg sync.WaitGroup

	for _, plan := range plans {
		sem <- struct{}{}
		wg.Add(1)
		go func(p *ScheduledTestPlan) {
			defer wg.Done()
			defer func() { <-sem }()
			s.runOnePlan(ctx, p)
		}(plan)
	}

	wg.Wait()
}

func (s *ScheduledTestRunnerService) runOnePlan(ctx context.Context, plan *ScheduledTestPlan) {
	if plan == nil || s == nil || s.accountTester == nil {
		return
	}
	result, err := s.accountTester.RunTestBackground(ctx, plan.AccountID, plan.ModelID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d RunTestBackground error: %v", plan.ID, err)
		return
	}

	if err := s.scheduledSvc.SaveResult(ctx, plan.ID, plan.MaxResults, result); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d SaveResult error: %v", plan.ID, err)
	}

	// Auto-recover account if test succeeded and auto_recover is enabled.
	if result.Status == "success" && plan.AutoRecover {
		s.tryRecoverAccount(ctx, plan.AccountID, plan.ID)
	} else if result.Status != "success" {
		s.tryPauseAccountAfterConsecutiveProbeFailures(ctx, plan)
	}

	nextRun, err := computeNextRun(plan.CronExpression, time.Now())
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d computeNextRun error: %v", plan.ID, err)
		return
	}

	if err := s.planRepo.UpdateAfterRun(ctx, plan.ID, time.Now(), nextRun); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d UpdateAfterRun error: %v", plan.ID, err)
	}
}

func (s *ScheduledTestRunnerService) schedulingConfig() config.GatewaySchedulingConfig {
	if s == nil || s.cfg == nil {
		return normalizeGatewaySchedulingConfig(config.GatewaySchedulingConfig{})
	}
	return normalizeGatewaySchedulingConfig(s.cfg.Gateway.Scheduling)
}

func (s *ScheduledTestRunnerService) tryPauseAccountAfterConsecutiveProbeFailures(ctx context.Context, plan *ScheduledTestPlan) {
	if s == nil || plan == nil || s.scheduledSvc == nil || s.accountPauser == nil {
		return
	}
	cfg := s.schedulingConfig().ActiveProbe
	if !cfg.AutoPauseEnabled {
		return
	}
	limit := cfg.FailureThreshold
	if limit <= 0 {
		return
	}
	lookback := plan.MaxResults
	if lookback < limit {
		lookback = limit
	}
	results, err := s.scheduledSvc.ListResults(ctx, plan.ID, lookback)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d ListResults error: %v", plan.ID, err)
		return
	}
	consecutiveFailures := countConsecutiveScheduledTestFailures(results)
	if consecutiveFailures < limit {
		return
	}
	cooldown := scheduledProbePauseDuration(cfg, consecutiveFailures)
	until := time.Now().Add(cooldown)
	reason := formatScheduledProbePauseReason(plan.ID, consecutiveFailures, cooldown)
	if err := s.accountPauser.SetTempUnschedulable(ctx, plan.AccountID, until, reason); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d temp-unschedule account=%d failed: %v", plan.ID, plan.AccountID, err)
	}
}

func countConsecutiveScheduledTestFailures(results []*ScheduledTestResult) int {
	count := 0
	for _, result := range results {
		if result == nil || result.Status == "success" {
			break
		}
		count++
	}
	return count
}

func scheduledProbePauseDuration(cfg config.GatewaySchedulingActiveProbeConfig, consecutiveFailures int) time.Duration {
	if consecutiveFailures < cfg.FailureThreshold {
		return 0
	}
	multiplier := consecutiveFailures - cfg.FailureThreshold + 1
	if multiplier < 1 {
		multiplier = 1
	}
	duration := time.Duration(multiplier) * cfg.PauseDuration
	if cfg.PauseDurationMax > 0 && duration > cfg.PauseDurationMax {
		return cfg.PauseDurationMax
	}
	return duration
}

func formatScheduledProbePauseReason(planID int64, consecutiveFailures int, cooldown time.Duration) string {
	return fmt.Sprintf("scheduled active probe failed: plan=%d consecutive_failures=%d auto temp-unschedule %s", planID, consecutiveFailures, cooldown)
}

// tryRecoverAccount attempts to recover an account from recoverable runtime state.
func (s *ScheduledTestRunnerService) tryRecoverAccount(ctx context.Context, accountID int64, planID int64) {
	if s.rateLimitRecoverer == nil {
		return
	}

	recovery, err := s.rateLimitRecoverer.RecoverAccountAfterSuccessfulTest(ctx, accountID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover failed: %v", planID, err)
		return
	}
	if recovery == nil {
		return
	}

	if recovery.ClearedError {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover: account=%d recovered from error status", planID, accountID)
	}
	if recovery.ClearedRateLimit {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover: account=%d cleared rate-limit/runtime state", planID, accountID)
	}
	if (recovery.ClearedError || recovery.ClearedRateLimit) && s.slowStartMarker != nil && s.schedulingConfig().SlowStart.Enabled {
		s.slowStartMarker.MarkAccountSlowStart(accountID, s.schedulingConfig().SlowStart.Duration)
	}
}
