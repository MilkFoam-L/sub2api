package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func schedulerIntPtr(v int) *int { return &v }

func schedulerFloatPtr(v float64) *float64 { return &v }

func makeWeightedPolicyAccount(id int64, priority int, concurrency int, loadFactor int, current int, waiting int) accountWithLoad {
	account := &Account{
		ID:          id,
		Priority:    priority,
		Concurrency: concurrency,
		Status:      StatusActive,
		Schedulable: true,
		Type:        AccountTypeAPIKey,
	}
	if loadFactor > 0 {
		account.LoadFactor = schedulerIntPtr(loadFactor)
	}
	max := account.EffectiveLoadFactor()
	loadRate := 0
	if max > 0 {
		loadRate = (current + waiting) * 100 / max
	}
	return accountWithLoad{
		account: account,
		loadInfo: &AccountLoadInfo{
			AccountID:          id,
			CurrentConcurrency: current,
			WaitingCount:       waiting,
			LoadRate:           loadRate,
		},
	}
}

func testWeightedP2CConfig() config.GatewaySchedulingConfig {
	return config.GatewaySchedulingConfig{
		Algorithm:           config.GatewaySchedulingAlgorithmWeightedP2C,
		P2CChoiceCount:      2,
		SelectionDebtTTLMS:  5000,
		SelectionDebtWeight: 1,
		WaitPenalty:         1,
		ScoreWeights: config.GatewaySchedulingScoreWeights{
			Load:           1,
			Queue:          1,
			Debt:           1,
			ErrorRate:      0.8,
			Latency:        0.4,
			RateMultiplier: 0.6,
			QuotaRisk:      0.3,
		},
		LatencyBaselineMS:      15000,
		QuotaRiskThreshold:     0.2,
		MaxScorePenalty:        5,
		SlowStart:              config.GatewaySchedulingSlowStartConfig{Enabled: true, Duration: 5 * time.Minute, Penalty: 1},
		StickySessionMode:      config.GatewayStickySessionModeSoft,
		StickyEscapeScoreRatio: 1.25,
		StickyEscapeLoadRate:   75,
	}
}

func TestSchedulerPolicyCostUsesLoadFactorAsCapacityWeight(t *testing.T) {
	cfg := testWeightedP2CConfig()
	lowCapacity := makeWeightedPolicyAccount(1, 0, 1, 1, 1, 0)
	highCapacity := makeWeightedPolicyAccount(2, 0, 1, 10, 1, 0)

	lowCost := schedulerAccountCost(lowCapacity, 0, cfg)
	highCost := schedulerAccountCost(highCapacity, 0, cfg)

	require.Greater(t, lowCost, highCost, "同样并发下，更高 UI 负载因子应该代表更大容量、更低调度成本")
}

func TestSchedulerPolicyCostFallsBackToLoadRate(t *testing.T) {
	cfg := testWeightedP2CConfig()
	highLoad := makeWeightedPolicyAccount(1, 0, 1, 5, 0, 0)
	lowLoad := makeWeightedPolicyAccount(2, 0, 1, 5, 0, 0)
	highLoad.loadInfo.LoadRate = 80
	lowLoad.loadInfo.LoadRate = 20

	highCost := schedulerAccountCost(highLoad, 0, cfg)
	lowCost := schedulerAccountCost(lowLoad, 0, cfg)

	require.Greater(t, highCost, lowCost, "批量负载只返回 LoadRate 时也应优先低负载账号")
}

func TestSchedulerPolicySelectionDebtPenalizesRecentlySelectedAccount(t *testing.T) {
	cfg := testWeightedP2CConfig()
	accounts := []accountWithLoad{
		makeWeightedPolicyAccount(1, 0, 1, 10, 0, 0),
		makeWeightedPolicyAccount(2, 0, 1, 10, 0, 0),
	}
	debts := map[int64]int{1: 3}

	order := buildWeightedP2CSelectionOrder(accounts, debts, false, cfg)

	require.NotEmpty(t, order)
	require.Equal(t, int64(2), order[0].account.ID, "selection debt 应该让刚选过的账号让路")
}

func TestSchedulerPolicyPreservesPriorityLayering(t *testing.T) {
	cfg := testWeightedP2CConfig()
	accounts := []accountWithLoad{
		makeWeightedPolicyAccount(1, 1, 1, 100, 0, 0),
		makeWeightedPolicyAccount(2, 0, 1, 1, 0, 0),
	}

	order := buildWeightedP2CSelectionOrder(accounts, nil, false, cfg)

	require.NotEmpty(t, order)
	require.Equal(t, int64(2), order[0].account.ID, "priority 仍应先于负载因子容量权重")
}

func TestSchedulerPolicyPreferredAccountWinsWithinPriorityLayer(t *testing.T) {
	cfg := testWeightedP2CConfig()
	cfg.PreferredAccountID = 2
	cfg.P2CChoiceCount = 1
	accounts := []accountWithLoad{
		makeWeightedPolicyAccount(1, 0, 1, 10, 0, 0),
		makeWeightedPolicyAccount(2, 0, 1, 10, 9, 0),
		makeWeightedPolicyAccount(3, 0, 1, 10, 0, 0),
	}

	order := buildWeightedP2CSelectionOrder(accounts, nil, false, cfg)

	require.NotEmpty(t, order)
	require.Equal(t, int64(2), order[0].account.ID, "优先账号在同 priority 候选层内应临时置顶")
}

func TestSchedulerPolicyPreferredAccountByGroupOnlyAppliesCurrentGroup(t *testing.T) {
	cfg := testWeightedP2CConfig()
	cfg.PreferredAccountByGroupID = map[int64]int64{1: 2, 2: 3}
	accounts := []accountWithLoad{
		makeWeightedPolicyAccount(1, 0, 1, 10, 0, 0),
		makeWeightedPolicyAccount(2, 0, 1, 10, 9, 0),
		makeWeightedPolicyAccount(3, 0, 1, 10, 8, 0),
	}

	groupOneOrder := buildWeightedP2CSelectionOrder(accounts, nil, false, schedulingConfigForGroup(cfg, 1))
	groupTwoOrder := buildWeightedP2CSelectionOrder(accounts, nil, false, schedulingConfigForGroup(cfg, 2))
	missingGroupOrder := buildWeightedP2CSelectionOrder(accounts, nil, false, schedulingConfigForGroup(cfg, 3))

	require.Equal(t, int64(2), groupOneOrder[0].account.ID)
	require.Equal(t, int64(3), groupTwoOrder[0].account.ID)
	require.NotEqual(t, int64(2), missingGroupOrder[0].account.ID, "未配置分组不应继承其他分组优先账号")
}

func TestSchedulerPolicyPreferredAccountDoesNotBypassPriorityLayer(t *testing.T) {
	cfg := testWeightedP2CConfig()
	cfg.PreferredAccountID = 1
	accounts := []accountWithLoad{
		makeWeightedPolicyAccount(1, 5, 1, 1, 0, 0),
		makeWeightedPolicyAccount(2, 0, 1, 100, 0, 0),
	}

	order := buildWeightedP2CSelectionOrder(accounts, nil, false, cfg)

	require.NotEmpty(t, order)
	require.Equal(t, int64(2), order[0].account.ID, "优先账号不能绕过更高优先级层")
}

func TestSchedulerPolicySoftStickyEscapesOverloadedAccount(t *testing.T) {
	cfg := testWeightedP2CConfig()
	sticky := makeWeightedPolicyAccount(1, 0, 1, 10, 8, 0)
	best := makeWeightedPolicyAccount(2, 0, 1, 10, 0, 0)

	decision := shouldUseStickyAccountForScheduling(sticky, []accountWithLoad{sticky, best}, nil, cfg)

	require.False(t, decision.useSticky)
	require.Equal(t, "load_rate", decision.reason)
}

func TestSchedulerPolicyStrictStickyKeepsOverloadedAccount(t *testing.T) {
	cfg := testWeightedP2CConfig()
	cfg.StickySessionMode = config.GatewayStickySessionModeStrict
	sticky := makeWeightedPolicyAccount(1, 0, 1, 10, 8, 0)
	best := makeWeightedPolicyAccount(2, 0, 1, 10, 0, 0)

	decision := shouldUseStickyAccountForScheduling(sticky, []accountWithLoad{sticky, best}, nil, cfg)

	require.True(t, decision.useSticky)
	require.Equal(t, "strict", decision.reason)
}

func TestSchedulerPolicyCostPrefersLowerRateMultiplier(t *testing.T) {
	cfg := testWeightedP2CConfig()
	cfg.ScoreWeights.RateMultiplier = 1
	cheap := makeWeightedPolicyAccount(1, 0, 1, 10, 0, 0)
	expensive := makeWeightedPolicyAccount(2, 0, 1, 10, 0, 0)
	cheap.account.RateMultiplier = schedulerFloatPtr(0.5)
	expensive.account.RateMultiplier = schedulerFloatPtr(2.0)

	cheapCost := schedulerAccountCost(cheap, 0, cfg)
	expensiveCost := schedulerAccountCost(expensive, 0, cfg)

	require.Less(t, cheapCost, expensiveCost, "同等负载下低倍率账号应该拥有更低调度成本")
}

func TestSchedulerPolicyCostPenalizesUnhealthyRuntimeStats(t *testing.T) {
	cfg := testWeightedP2CConfig()
	cfg.ScoreWeights.ErrorRate = 1
	healthy := makeWeightedPolicyAccount(1, 0, 1, 10, 0, 0)
	unhealthy := makeWeightedPolicyAccount(2, 0, 1, 10, 0, 0)
	healthy.runtimeStats = &schedulerAccountRuntimeSnapshot{ErrorRate: 0.05}
	unhealthy.runtimeStats = &schedulerAccountRuntimeSnapshot{ErrorRate: 0.8}

	healthyCost := schedulerAccountCost(healthy, 0, cfg)
	unhealthyCost := schedulerAccountCost(unhealthy, 0, cfg)

	require.Greater(t, unhealthyCost, healthyCost, "错误率更高的账号应该被健康评分惩罚")
}

func TestSchedulerPolicyCostPenalizesHighLatencyRuntimeStats(t *testing.T) {
	cfg := testWeightedP2CConfig()
	cfg.ScoreWeights.Latency = 1
	cfg.LatencyBaselineMS = 1000
	fast := makeWeightedPolicyAccount(1, 0, 1, 10, 0, 0)
	slow := makeWeightedPolicyAccount(2, 0, 1, 10, 0, 0)
	fast.runtimeStats = &schedulerAccountRuntimeSnapshot{LatencyMs: 500, HasLatency: true}
	slow.runtimeStats = &schedulerAccountRuntimeSnapshot{LatencyMs: 4000, HasLatency: true}

	fastCost := schedulerAccountCost(fast, 0, cfg)
	slowCost := schedulerAccountCost(slow, 0, cfg)

	require.Greater(t, slowCost, fastCost, "延迟 EWMA 更高的账号应该被评分惩罚")
}

func TestSchedulerPolicyCostPenalizesQuotaRiskBeforeHardLimit(t *testing.T) {
	cfg := testWeightedP2CConfig()
	cfg.ScoreWeights.QuotaRisk = 1
	cfg.QuotaRiskThreshold = 0.2
	lowRisk := makeWeightedPolicyAccount(1, 0, 1, 10, 0, 0)
	highRisk := makeWeightedPolicyAccount(2, 0, 1, 10, 0, 0)
	lowRisk.account.Extra = map[string]any{"quota_limit": 100.0, "quota_used": 20.0}
	highRisk.account.Extra = map[string]any{"quota_limit": 100.0, "quota_used": 95.0}

	lowRiskCost := schedulerAccountCost(lowRisk, 0, cfg)
	highRiskCost := schedulerAccountCost(highRisk, 0, cfg)

	require.Greater(t, highRiskCost, lowRiskCost, "额度接近耗尽但未超限的账号应该被软惩罚")
	require.False(t, highRisk.account.IsQuotaExceeded(), "测试前提：高风险账号尚未触发硬超限过滤")
}

func TestSchedulerPolicyCostPenalizesSlowStartAccount(t *testing.T) {
	cfg := testWeightedP2CConfig()
	stable := makeWeightedPolicyAccount(1, 0, 1, 10, 0, 0)
	recovering := makeWeightedPolicyAccount(2, 0, 1, 10, 0, 0)
	recovering.runtimeStats = &schedulerAccountRuntimeSnapshot{HasSlowStart: true, SlowStartUntil: time.Now().Add(5 * time.Minute)}

	stableCost := schedulerAccountCost(stable, 0, cfg)
	recoveringCost := schedulerAccountCost(recovering, 0, cfg)

	require.Greater(t, recoveringCost, stableCost, "刚恢复的账号应该在 slow-start 窗口内被软惩罚")
}

func TestSchedulerPolicyCostIgnoresExpiredSlowStart(t *testing.T) {
	cfg := testWeightedP2CConfig()
	stable := makeWeightedPolicyAccount(1, 0, 1, 10, 0, 0)
	recovered := makeWeightedPolicyAccount(2, 0, 1, 10, 0, 0)
	recovered.runtimeStats = &schedulerAccountRuntimeSnapshot{HasSlowStart: true, SlowStartUntil: time.Now().Add(-time.Minute)}

	stableCost := schedulerAccountCost(stable, 0, cfg)
	recoveredCost := schedulerAccountCost(recovered, 0, cfg)

	require.Equal(t, stableCost, recoveredCost, "slow-start 过期后不应继续惩罚账号")
}

func TestSchedulerPolicyStrictStickyKeepsSlowStartAccount(t *testing.T) {
	cfg := testWeightedP2CConfig()
	cfg.StickySessionMode = config.GatewayStickySessionModeStrict
	sticky := makeWeightedPolicyAccount(1, 0, 1, 10, 0, 0)
	best := makeWeightedPolicyAccount(2, 0, 1, 10, 0, 0)
	sticky.runtimeStats = &schedulerAccountRuntimeSnapshot{HasSlowStart: true, SlowStartUntil: time.Now().Add(5 * time.Minute)}

	decision := shouldUseStickyAccountForScheduling(sticky, []accountWithLoad{sticky, best}, nil, cfg)

	require.True(t, decision.useSticky)
	require.Equal(t, "strict", decision.reason)
}

func TestSchedulerPolicySoftStickyEscapesSlowStartAccount(t *testing.T) {
	cfg := testWeightedP2CConfig()
	cfg.StickyEscapeScoreRatio = 1.1
	sticky := makeWeightedPolicyAccount(1, 0, 1, 10, 0, 0)
	best := makeWeightedPolicyAccount(2, 0, 1, 10, 0, 0)
	sticky.runtimeStats = &schedulerAccountRuntimeSnapshot{HasSlowStart: true, SlowStartUntil: time.Now().Add(5 * time.Minute)}

	decision := shouldUseStickyAccountForScheduling(sticky, []accountWithLoad{sticky, best}, nil, cfg)

	require.False(t, decision.useSticky)
	require.Equal(t, "score", decision.reason)
}

func TestSchedulerPolicySoftStickyEscapesCostlyAccount(t *testing.T) {
	cfg := testWeightedP2CConfig()
	cfg.ScoreWeights.RateMultiplier = 1
	cfg.StickyEscapeScoreRatio = 1.1
	sticky := makeWeightedPolicyAccount(1, 0, 1, 10, 0, 0)
	best := makeWeightedPolicyAccount(2, 0, 1, 10, 0, 0)
	sticky.account.RateMultiplier = schedulerFloatPtr(3.0)
	best.account.RateMultiplier = schedulerFloatPtr(0.5)

	decision := shouldUseStickyAccountForScheduling(sticky, []accountWithLoad{sticky, best}, nil, cfg)

	require.False(t, decision.useSticky)
	require.Equal(t, "score", decision.reason)
}

func TestSchedulerPolicyUpstreamRateSignalIsNeutralWhenDisabled(t *testing.T) {
	cfg := testWeightedP2CConfig()
	cfg.UpstreamRate.Enabled = false
	cfg.UpstreamRate.RateWeight = 1
	item := makeWeightedPolicyAccount(1, 0, 1, 10, 0, 0)
	item.upstreamRateSignal = &UpstreamRateSignalSnapshot{EffectiveRateMultiplier: 3, SuccessRate: 0.1}

	costWithSignal := schedulerAccountCost(item, 0, cfg)
	item.upstreamRateSignal = nil
	costWithoutSignal := schedulerAccountCost(item, 0, cfg)

	require.Equal(t, costWithoutSignal, costWithSignal)
}

func TestSchedulerPolicyUpstreamRateSignalAddsSoftPenalty(t *testing.T) {
	cfg := testWeightedP2CConfig()
	cfg.UpstreamRate.Enabled = true
	cfg.UpstreamRate.RateWeight = 1
	cfg.UpstreamRate.HealthWeight = 1
	cfg.UpstreamRate.MinSuccessRate = 0.9
	item := makeWeightedPolicyAccount(1, 0, 1, 10, 0, 0)
	base := schedulerAccountCost(item, 0, cfg)
	item.upstreamRateSignal = &UpstreamRateSignalSnapshot{EffectiveRateMultiplier: 1.5, SuccessRate: 0.5}

	withSignal := schedulerAccountCost(item, 0, cfg)

	require.Greater(t, withSignal, base)
}

func TestConcurrencyServiceSelectionDebtInMemoryFallback(t *testing.T) {
	svc := NewConcurrencyService(nil)
	ctx := context.Background()

	require.NoError(t, svc.RecordAccountSelection(ctx, 101, 50*time.Millisecond))
	debts, err := svc.GetAccountSelectionDebtBatch(ctx, []int64{101, 102})
	require.NoError(t, err)
	require.Equal(t, 1, debts[101])
	require.Equal(t, 0, debts[102])
}
