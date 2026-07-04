package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayLegacyFallbackUsesPreferredAccountAndRecordsLog(t *testing.T) {
	oldLogService := DefaultSchedulingLogService
	DefaultSchedulingLogService = NewSchedulingLogService(10)
	defer func() { DefaultSchedulingLogService = oldLogService }()

	cfg := testWeightedP2CConfig()
	cfg.PreferredAccountID = 2
	svc := &GatewayService{}
	accounts := []*Account{
		{ID: 1, Name: "normal", Priority: 0, Status: StatusActive, Schedulable: true, Type: AccountTypeAPIKey, Concurrency: 1},
		{ID: 2, Name: "preferred", Priority: 0, Status: StatusActive, Schedulable: true, Type: AccountTypeAPIKey, Concurrency: 1},
	}

	selection, ok, err := svc.tryAcquireByLegacyOrder(context.Background(), accounts, nil, "", false, cfg, "anthropic", "claude-test")

	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, selection)
	require.Equal(t, int64(2), selection.Account.ID)
	logs := DefaultSchedulingLogService.List(10)
	require.Len(t, logs, 1)
	require.Equal(t, int64(2), logs[0].AccountID)
	require.True(t, logs[0].PreferredHit)
	require.Equal(t, "legacy_fallback_selected", logs[0].Reason)
}
