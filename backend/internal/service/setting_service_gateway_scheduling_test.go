package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type gatewaySchedulingSettingRepoStub struct {
	values map[string]string
	sets   map[string]string
}

func (s *gatewaySchedulingSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}
func (s *gatewaySchedulingSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}
func (s *gatewaySchedulingSettingRepoStub) Set(_ context.Context, key, value string) error {
	if s.sets == nil {
		s.sets = map[string]string{}
	}
	s.sets[key] = value
	return nil
}
func (s *gatewaySchedulingSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}
func (s *gatewaySchedulingSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	if s.sets == nil {
		s.sets = map[string]string{}
	}
	for key, value := range settings {
		s.sets[key] = value
	}
	return nil
}
func (s *gatewaySchedulingSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	out := map[string]string{}
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}
func (s *gatewaySchedulingSettingRepoStub) Delete(context.Context, string) error { return nil }

func baseGatewaySchedulingTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Gateway.Scheduling = config.GatewaySchedulingConfig{
		Algorithm:                 config.GatewaySchedulingAlgorithmWeightedP2C,
		P2CChoiceCount:            2,
		SelectionDebtTTLMS:        5000,
		SelectionDebtWeight:       1,
		WaitPenalty:               1,
		StickySessionMode:         config.GatewayStickySessionModeSoft,
		StickyEscapeScoreRatio:    1.25,
		StickyEscapeLoadRate:      75,
		LatencyBaselineMS:         15000,
		QuotaRiskThreshold:        0.2,
		MaxScorePenalty:           5,
		LoadBatchEnabled:          true,
		LoadBatchCacheTTLMS:       300,
		SnapshotMGetChunkSize:     500,
		SnapshotWriteChunkSize:    500,
		OutboxPollIntervalSeconds: 5,
		OutboxLagWarnSeconds:      15,
		OutboxLagRebuildSeconds:   60,
		OutboxLagRebuildFailures:  3,
		ScoreWeights:              config.GatewaySchedulingScoreWeights{Load: 1, Queue: 1, Debt: 1, ErrorRate: 0.8, Latency: 0.4, RateMultiplier: 0.6, QuotaRisk: 0.3},
		ActiveProbe:               config.GatewaySchedulingActiveProbeConfig{AutoPauseEnabled: true, FailureThreshold: 3, PauseDuration: 10 * time.Minute, PauseDurationMax: time.Hour},
		SlowStart:                 config.GatewaySchedulingSlowStartConfig{Enabled: true, Duration: 5 * time.Minute, Penalty: 1},
	}
	return cfg
}

func TestSettingServiceGatewaySchedulingConfigFallsBackToConfig(t *testing.T) {
	svc := NewSettingService(&gatewaySchedulingSettingRepoStub{values: map[string]string{}}, baseGatewaySchedulingTestConfig())

	got, err := svc.GetGatewaySchedulingConfig(context.Background())

	require.NoError(t, err)
	require.Equal(t, 0.4, got.ScoreWeights.Latency)
	require.Equal(t, 15000, got.LatencyBaselineMS)
	require.Equal(t, 10*time.Minute, got.ActiveProbe.PauseDuration)
	require.Equal(t, config.GatewayStickySessionModeSoft, got.StickySessionMode)
	require.Equal(t, int64(0), got.PreferredAccountID)
	require.Empty(t, got.PreferredAccountByGroupID)
}

func TestSettingServiceGatewaySchedulingConfigUsesDatabaseOverrides(t *testing.T) {
	repo := &gatewaySchedulingSettingRepoStub{values: map[string]string{
		SettingKeyGatewaySchedulingPreferredAccountID:          "42",
		SettingKeyGatewaySchedulingPreferredAccountByGroupID:   "{\"1\":99}",
		SettingKeyGatewaySchedulingScoreWeightLatency:          "0.9",
		SettingKeyGatewaySchedulingScoreWeightRateMultiplier:   "1.1",
		SettingKeyGatewaySchedulingLatencyBaselineMS:           "9000",
		SettingKeyGatewaySchedulingStickySessionMode:           config.GatewayStickySessionModeStrict,
		SettingKeyGatewaySchedulingActiveProbeFailureThreshold: "5",
		SettingKeyGatewaySchedulingActiveProbePauseDuration:    "15m",
		SettingKeyGatewaySchedulingActiveProbePauseDurationMax: "2h",
		SettingKeyGatewaySchedulingSlowStartEnabled:            "false",
		SettingKeyGatewaySchedulingSlowStartDuration:           "7m",
		SettingKeyGatewaySchedulingSlowStartPenalty:            "1.7",
	}}
	svc := NewSettingService(repo, baseGatewaySchedulingTestConfig())

	got, err := svc.GetGatewaySchedulingConfig(context.Background())

	require.NoError(t, err)
	require.Equal(t, int64(42), got.PreferredAccountID)
	require.Equal(t, map[int64]int64{1: 99}, got.PreferredAccountByGroupID)
	require.Equal(t, 0.9, got.ScoreWeights.Latency)
	require.Equal(t, 1.1, got.ScoreWeights.RateMultiplier)
	require.Equal(t, 9000, got.LatencyBaselineMS)
	require.Equal(t, config.GatewayStickySessionModeStrict, got.StickySessionMode)
	require.Equal(t, 5, got.ActiveProbe.FailureThreshold)
	require.Equal(t, 15*time.Minute, got.ActiveProbe.PauseDuration)
	require.Equal(t, 2*time.Hour, got.ActiveProbe.PauseDurationMax)
	require.False(t, got.SlowStart.Enabled)
	require.Equal(t, 7*time.Minute, got.SlowStart.Duration)
	require.Equal(t, 1.7, got.SlowStart.Penalty)
}

func TestSettingServiceGatewaySchedulingConfigRejectsInvalidOverrideSet(t *testing.T) {
	repo := &gatewaySchedulingSettingRepoStub{values: map[string]string{
		SettingKeyGatewaySchedulingScoreWeightLatency: "-1",
	}}
	svc := NewSettingService(repo, baseGatewaySchedulingTestConfig())

	_, err := svc.GetGatewaySchedulingConfig(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "gateway.scheduling.score_weights")
}

func TestSettingServiceUpdateSettingsWritesGatewaySchedulingSettings(t *testing.T) {
	repo := &gatewaySchedulingSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, baseGatewaySchedulingTestConfig())

	gatewayScheduling := baseGatewaySchedulingTestConfig().Gateway.Scheduling
	gatewayScheduling.PreferredAccountID = 0
	gatewayScheduling.PreferredAccountByGroupID = map[int64]int64{1: 99}
	gatewayScheduling.ScoreWeights.Load = 1.2
	gatewayScheduling.ScoreWeights.Queue = 1.3
	gatewayScheduling.ScoreWeights.Debt = 1.4
	gatewayScheduling.ScoreWeights.ErrorRate = 0.9
	gatewayScheduling.ScoreWeights.Latency = 0.8
	gatewayScheduling.ScoreWeights.RateMultiplier = 1.1
	gatewayScheduling.ScoreWeights.QuotaRisk = 0.4
	gatewayScheduling.LatencyBaselineMS = 12000
	gatewayScheduling.QuotaRiskThreshold = 0.15
	gatewayScheduling.MaxScorePenalty = 4.5
	gatewayScheduling.StickySessionMode = config.GatewayStickySessionModeOff
	gatewayScheduling.StickyEscapeScoreRatio = 1.5
	gatewayScheduling.StickyEscapeLoadRate = 80
	gatewayScheduling.ActiveProbe.AutoPauseEnabled = true
	gatewayScheduling.ActiveProbe.FailureThreshold = 4
	gatewayScheduling.ActiveProbe.PauseDuration = 12 * time.Minute
	gatewayScheduling.ActiveProbe.PauseDurationMax = 45 * time.Minute
	gatewayScheduling.SlowStart.Enabled = true
	gatewayScheduling.SlowStart.Duration = 6 * time.Minute
	gatewayScheduling.SlowStart.Penalty = 1.2

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		GatewayScheduling: gatewayScheduling,
	})

	require.NoError(t, err)
	require.Equal(t, "0", repo.sets[SettingKeyGatewaySchedulingPreferredAccountID])
	require.JSONEq(t, `{"1":99}`, repo.sets[SettingKeyGatewaySchedulingPreferredAccountByGroupID])
	require.Equal(t, "0.80000000", repo.sets[SettingKeyGatewaySchedulingScoreWeightLatency])
	require.Equal(t, "12000", repo.sets[SettingKeyGatewaySchedulingLatencyBaselineMS])
	require.Equal(t, config.GatewayStickySessionModeOff, repo.sets[SettingKeyGatewaySchedulingStickySessionMode])
	require.Equal(t, "12m0s", repo.sets[SettingKeyGatewaySchedulingActiveProbePauseDuration])
	require.Equal(t, "6m0s", repo.sets[SettingKeyGatewaySchedulingSlowStartDuration])
}

func TestSettingServiceUpdateSettingsWithAuthSourceDefaultsSkipsGatewaySchedulingSettings(t *testing.T) {
	repo := &gatewaySchedulingSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, baseGatewaySchedulingTestConfig())
	gatewayScheduling := baseGatewaySchedulingTestConfig().Gateway.Scheduling
	gatewayScheduling.PreferredAccountID = 456
	gatewayScheduling.PreferredAccountByGroupID = map[int64]int64{2: 456}
	gatewayScheduling.ScoreWeights.Latency = 0.55

	err := svc.UpdateSettingsWithAuthSourceDefaults(context.Background(), &SystemSettings{
		GatewayScheduling: gatewayScheduling,
	}, &AuthSourceDefaultSettings{})

	require.NoError(t, err)
	require.NotContains(t, repo.sets, SettingKeyGatewaySchedulingPreferredAccountID)
	require.NotContains(t, repo.sets, SettingKeyGatewaySchedulingPreferredAccountByGroupID)
	require.NotContains(t, repo.sets, SettingKeyGatewaySchedulingScoreWeightLatency)
}

func TestSettingServiceUpdateGatewaySchedulingConfigOnlyWritesSchedulingSettings(t *testing.T) {
	repo := &gatewaySchedulingSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, baseGatewaySchedulingTestConfig())
	cfg := baseGatewaySchedulingTestConfig().Gateway.Scheduling
	cfg.PreferredAccountID = 0
	cfg.PreferredAccountByGroupID = map[int64]int64{3: 123}
	cfg.ScoreWeights.Latency = 0.75

	err := svc.UpdateGatewaySchedulingConfig(context.Background(), cfg)

	require.NoError(t, err)
	require.Equal(t, "0", repo.sets[SettingKeyGatewaySchedulingPreferredAccountID])
	require.JSONEq(t, `{"3":123}`, repo.sets[SettingKeyGatewaySchedulingPreferredAccountByGroupID])
	require.Equal(t, "0.75000000", repo.sets[SettingKeyGatewaySchedulingScoreWeightLatency])
	require.NotContains(t, repo.sets, SettingKeySiteName)
}
