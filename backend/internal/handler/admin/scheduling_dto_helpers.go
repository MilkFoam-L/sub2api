package admin

import (
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
)

func gatewaySchedulingToDTO(cfg config.GatewaySchedulingConfig) dto.GatewaySchedulingSettings {
	if strings.TrimSpace(cfg.Credential.Strategy) == "" {
		cfg.Credential.Strategy = config.GatewaySchedulingCredentialStrategyBalanced
		cfg.Credential.FallbackEnabled = true
	}
	return dto.GatewaySchedulingSettings{
		PreferredAccountID:        cfg.PreferredAccountID,
		PreferredAccountByGroupID: cfg.PreferredAccountByGroupID,
		ScoreWeights: dto.GatewaySchedulingScoreWeights{
			Load:           cfg.ScoreWeights.Load,
			Queue:          cfg.ScoreWeights.Queue,
			Debt:           cfg.ScoreWeights.Debt,
			ErrorRate:      cfg.ScoreWeights.ErrorRate,
			Latency:        cfg.ScoreWeights.Latency,
			RateMultiplier: cfg.ScoreWeights.RateMultiplier,
			QuotaRisk:      cfg.ScoreWeights.QuotaRisk,
		},
		LatencyBaselineMS:      cfg.LatencyBaselineMS,
		QuotaRiskThreshold:     cfg.QuotaRiskThreshold,
		MaxScorePenalty:        cfg.MaxScorePenalty,
		StickySessionMode:      cfg.StickySessionMode,
		StickyEscapeScoreRatio: cfg.StickyEscapeScoreRatio,
		StickyEscapeLoadRate:   cfg.StickyEscapeLoadRate,
		ActiveProbe: dto.GatewaySchedulingActiveProbeSettings{
			AutoPauseEnabled: cfg.ActiveProbe.AutoPauseEnabled,
			FailureThreshold: cfg.ActiveProbe.FailureThreshold,
			PauseDuration:    cfg.ActiveProbe.PauseDuration.String(),
			PauseDurationMax: cfg.ActiveProbe.PauseDurationMax.String(),
		},
		SlowStart: dto.GatewaySchedulingSlowStartSettings{
			Enabled:  cfg.SlowStart.Enabled,
			Duration: cfg.SlowStart.Duration.String(),
			Penalty:  cfg.SlowStart.Penalty,
		},
		UpstreamRate: dto.GatewaySchedulingUpstreamRateSettings{
			Enabled:         cfg.UpstreamRate.Enabled,
			StaleTTLSeconds: cfg.UpstreamRate.StaleTTLSeconds,
			RateWeight:      cfg.UpstreamRate.RateWeight,
			HealthWeight:    cfg.UpstreamRate.HealthWeight,
			MinSuccessRate:  cfg.UpstreamRate.MinSuccessRate,
		},
		Credential: dto.GatewaySchedulingCredentialSettings{
			Strategy:        cfg.Credential.Strategy,
			FallbackEnabled: cfg.Credential.FallbackEnabled,
		},
	}
}

func applyGatewaySchedulingDTO(base config.GatewaySchedulingConfig, payload *dto.GatewaySchedulingSettings) (config.GatewaySchedulingConfig, error) {
	if payload == nil {
		return base, nil
	}
	cfg := base
	cfg.PreferredAccountID = payload.PreferredAccountID
	cfg.PreferredAccountByGroupID = payload.PreferredAccountByGroupID
	if cfg.PreferredAccountByGroupID == nil {
		cfg.PreferredAccountByGroupID = map[int64]int64{}
	}
	cfg.ScoreWeights.Load = payload.ScoreWeights.Load
	cfg.ScoreWeights.Queue = payload.ScoreWeights.Queue
	cfg.ScoreWeights.Debt = payload.ScoreWeights.Debt
	cfg.ScoreWeights.ErrorRate = payload.ScoreWeights.ErrorRate
	cfg.ScoreWeights.Latency = payload.ScoreWeights.Latency
	cfg.ScoreWeights.RateMultiplier = payload.ScoreWeights.RateMultiplier
	cfg.ScoreWeights.QuotaRisk = payload.ScoreWeights.QuotaRisk
	cfg.LatencyBaselineMS = payload.LatencyBaselineMS
	cfg.QuotaRiskThreshold = payload.QuotaRiskThreshold
	cfg.MaxScorePenalty = payload.MaxScorePenalty
	cfg.StickySessionMode = payload.StickySessionMode
	cfg.StickyEscapeScoreRatio = payload.StickyEscapeScoreRatio
	cfg.StickyEscapeLoadRate = payload.StickyEscapeLoadRate
	cfg.ActiveProbe.AutoPauseEnabled = payload.ActiveProbe.AutoPauseEnabled
	cfg.ActiveProbe.FailureThreshold = payload.ActiveProbe.FailureThreshold
	pauseDuration, err := time.ParseDuration(strings.TrimSpace(payload.ActiveProbe.PauseDuration))
	if err != nil || pauseDuration <= 0 {
		return cfg, fmt.Errorf("gateway_scheduling.active_probe.pause_duration must be a positive duration")
	}
	pauseDurationMax, err := time.ParseDuration(strings.TrimSpace(payload.ActiveProbe.PauseDurationMax))
	if err != nil || pauseDurationMax <= 0 {
		return cfg, fmt.Errorf("gateway_scheduling.active_probe.pause_duration_max must be a positive duration")
	}
	cfg.ActiveProbe.PauseDuration = pauseDuration
	cfg.ActiveProbe.PauseDurationMax = pauseDurationMax
	cfg.SlowStart.Enabled = payload.SlowStart.Enabled
	slowStartDuration, err := time.ParseDuration(strings.TrimSpace(payload.SlowStart.Duration))
	if err != nil || slowStartDuration <= 0 {
		return cfg, fmt.Errorf("gateway_scheduling.slow_start.duration must be a positive duration")
	}
	cfg.SlowStart.Duration = slowStartDuration
	cfg.SlowStart.Penalty = payload.SlowStart.Penalty
	cfg.UpstreamRate.Enabled = payload.UpstreamRate.Enabled
	cfg.UpstreamRate.StaleTTLSeconds = payload.UpstreamRate.StaleTTLSeconds
	cfg.UpstreamRate.RateWeight = payload.UpstreamRate.RateWeight
	cfg.UpstreamRate.HealthWeight = payload.UpstreamRate.HealthWeight
	cfg.UpstreamRate.MinSuccessRate = payload.UpstreamRate.MinSuccessRate
	cfg.Credential.Strategy = strings.ToLower(strings.TrimSpace(payload.Credential.Strategy))
	if cfg.Credential.Strategy == "" {
		cfg.Credential.Strategy = config.GatewaySchedulingCredentialStrategyBalanced
	}
	cfg.Credential.FallbackEnabled = payload.Credential.FallbackEnabled
	if cfg.PreferredAccountID < 0 {
		return cfg, fmt.Errorf("gateway_scheduling.preferred_account_id must be non-negative")
	}
	for groupID, accountID := range cfg.PreferredAccountByGroupID {
		if groupID < 0 || accountID < 0 {
			return cfg, fmt.Errorf("gateway_scheduling.preferred_account_by_group_id must contain non-negative ids")
		}
	}
	if cfg.ScoreWeights.Load < 0 || cfg.ScoreWeights.Queue < 0 || cfg.ScoreWeights.Debt < 0 || cfg.ScoreWeights.ErrorRate < 0 || cfg.ScoreWeights.Latency < 0 || cfg.ScoreWeights.RateMultiplier < 0 || cfg.ScoreWeights.QuotaRisk < 0 {
		return cfg, fmt.Errorf("gateway_scheduling.score_weights must be non-negative")
	}
	if cfg.LatencyBaselineMS <= 0 || cfg.QuotaRiskThreshold < 0 || cfg.QuotaRiskThreshold > 1 || cfg.MaxScorePenalty < 0 {
		return cfg, fmt.Errorf("gateway_scheduling numeric limits are invalid")
	}
	if cfg.StickySessionMode != config.GatewayStickySessionModeStrict && cfg.StickySessionMode != config.GatewayStickySessionModeSoft && cfg.StickySessionMode != config.GatewayStickySessionModeOff {
		return cfg, fmt.Errorf("gateway_scheduling.sticky_session_mode is invalid")
	}
	if cfg.StickyEscapeScoreRatio < 1 || cfg.StickyEscapeLoadRate < 0 || cfg.StickyEscapeLoadRate > 100 {
		return cfg, fmt.Errorf("gateway_scheduling sticky escape settings are invalid")
	}
	if cfg.ActiveProbe.FailureThreshold <= 0 || cfg.ActiveProbe.PauseDurationMax < cfg.ActiveProbe.PauseDuration || cfg.SlowStart.Penalty < 0 {
		return cfg, fmt.Errorf("gateway_scheduling probe or slow-start settings are invalid")
	}
	if cfg.UpstreamRate.StaleTTLSeconds <= 0 || cfg.UpstreamRate.RateWeight < 0 || cfg.UpstreamRate.HealthWeight < 0 || cfg.UpstreamRate.MinSuccessRate < 0 || cfg.UpstreamRate.MinSuccessRate > 1 {
		return cfg, fmt.Errorf("gateway_scheduling upstream_rate settings are invalid")
	}
	switch cfg.Credential.Strategy {
	case config.GatewaySchedulingCredentialStrategyBalanced, config.GatewaySchedulingCredentialStrategyOAuthFirst, config.GatewaySchedulingCredentialStrategyAPIKeyFirst:
	default:
		return cfg, fmt.Errorf("gateway_scheduling credential strategy is invalid")
	}
	return cfg, nil
}
