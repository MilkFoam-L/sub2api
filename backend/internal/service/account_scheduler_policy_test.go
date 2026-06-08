package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func schedulerIntPtr(v int) *int { return &v }

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
		Algorithm:              config.GatewaySchedulingAlgorithmWeightedP2C,
		P2CChoiceCount:         2,
		SelectionDebtTTLMS:     5000,
		SelectionDebtWeight:    1,
		WaitPenalty:            1,
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

func TestConcurrencyServiceSelectionDebtInMemoryFallback(t *testing.T) {
	svc := NewConcurrencyService(nil)
	ctx := context.Background()

	require.NoError(t, svc.RecordAccountSelection(ctx, 101, 50*time.Millisecond))
	debts, err := svc.GetAccountSelectionDebtBatch(ctx, []int64{101, 102})
	require.NoError(t, err)
	require.Equal(t, 1, debts[101])
	require.Equal(t, 0, debts[102])
}
