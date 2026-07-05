package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const gatewaySchedulingSettingsCacheTTL = 60 * time.Second

var gatewaySchedulingSettingKeys = []string{
	SettingKeyGatewaySchedulingPreferredAccountID,
	SettingKeyGatewaySchedulingPreferredAccountByGroupID,
	SettingKeyGatewaySchedulingScoreWeightLoad,
	SettingKeyGatewaySchedulingScoreWeightQueue,
	SettingKeyGatewaySchedulingScoreWeightDebt,
	SettingKeyGatewaySchedulingScoreWeightErrorRate,
	SettingKeyGatewaySchedulingScoreWeightLatency,
	SettingKeyGatewaySchedulingScoreWeightRateMultiplier,
	SettingKeyGatewaySchedulingScoreWeightQuotaRisk,
	SettingKeyGatewaySchedulingLatencyBaselineMS,
	SettingKeyGatewaySchedulingQuotaRiskThreshold,
	SettingKeyGatewaySchedulingMaxScorePenalty,
	SettingKeyGatewaySchedulingStickySessionMode,
	SettingKeyGatewaySchedulingStickyEscapeScoreRatio,
	SettingKeyGatewaySchedulingStickyEscapeLoadRate,
	SettingKeyGatewaySchedulingActiveProbeAutoPause,
	SettingKeyGatewaySchedulingActiveProbeFailureThreshold,
	SettingKeyGatewaySchedulingActiveProbePauseDuration,
	SettingKeyGatewaySchedulingActiveProbePauseDurationMax,
	SettingKeyGatewaySchedulingSlowStartEnabled,
	SettingKeyGatewaySchedulingSlowStartDuration,
	SettingKeyGatewaySchedulingSlowStartPenalty,
	SettingKeyGatewaySchedulingUpstreamRateEnabled,
	SettingKeyGatewaySchedulingUpstreamRateStaleTTLSeconds,
	SettingKeyGatewaySchedulingUpstreamRateRateWeight,
	SettingKeyGatewaySchedulingUpstreamRateHealthWeight,
	SettingKeyGatewaySchedulingUpstreamRateMinSuccessRate,
}

func (s *SettingService) GetGatewaySchedulingConfig(ctx context.Context) (config.GatewaySchedulingConfig, error) {
	base := defaultGatewaySchedulingConfig()
	if s != nil && s.cfg != nil {
		base = normalizeGatewaySchedulingConfig(s.cfg.Gateway.Scheduling)
	}
	if s == nil || s.settingRepo == nil {
		return base, nil
	}
	if cached, ok := s.gatewaySchedulingSettingsCache.Load().(*cachedGatewaySchedulingSettings); ok && cached != nil {
		if time.Now().UnixNano() < cached.expiresAt {
			return cached.cfg, nil
		}
	}
	settings, err := s.settingRepo.GetMultiple(ctx, gatewaySchedulingSettingKeys)
	if err != nil {
		return base, fmt.Errorf("get gateway scheduling settings: %w", err)
	}
	merged, err := applyGatewaySchedulingSettings(base, settings)
	if err != nil {
		return base, err
	}
	s.gatewaySchedulingSettingsCache.Store(&cachedGatewaySchedulingSettings{cfg: merged, expiresAt: time.Now().Add(gatewaySchedulingSettingsCacheTTL).UnixNano()})
	return merged, nil
}

func (s *SettingService) UpdateGatewaySchedulingConfig(ctx context.Context, cfg config.GatewaySchedulingConfig) error {
	if s == nil || s.settingRepo == nil {
		return nil
	}
	normalized, err := validateGatewaySchedulingSettingsConfig(normalizeGatewaySchedulingConfig(cfg))
	if err != nil {
		return err
	}
	updates := gatewaySchedulingConfigToMap(normalized)
	if err := s.settingRepo.SetMultiple(ctx, updates); err != nil {
		return err
	}
	s.gatewaySchedulingSettingsCache.Store(&cachedGatewaySchedulingSettings{cfg: normalized, expiresAt: time.Now().Add(gatewaySchedulingSettingsCacheTTL).UnixNano()})
	return nil
}

func applyGatewaySchedulingSettings(base config.GatewaySchedulingConfig, settings map[string]string) (config.GatewaySchedulingConfig, error) {
	cfg := normalizeGatewaySchedulingConfig(base)
	var err error
	if cfg.PreferredAccountID, err = parseInt64Setting(settings, SettingKeyGatewaySchedulingPreferredAccountID, cfg.PreferredAccountID); err != nil {
		return cfg, err
	}
	if cfg.PreferredAccountByGroupID, err = parseInt64MapSetting(settings, SettingKeyGatewaySchedulingPreferredAccountByGroupID, cfg.PreferredAccountByGroupID); err != nil {
		return cfg, err
	}
	if cfg.ScoreWeights.Load, err = parseFloatSetting(settings, SettingKeyGatewaySchedulingScoreWeightLoad, cfg.ScoreWeights.Load); err != nil {
		return cfg, err
	}
	if cfg.ScoreWeights.Queue, err = parseFloatSetting(settings, SettingKeyGatewaySchedulingScoreWeightQueue, cfg.ScoreWeights.Queue); err != nil {
		return cfg, err
	}
	if cfg.ScoreWeights.Debt, err = parseFloatSetting(settings, SettingKeyGatewaySchedulingScoreWeightDebt, cfg.ScoreWeights.Debt); err != nil {
		return cfg, err
	}
	if cfg.ScoreWeights.ErrorRate, err = parseFloatSetting(settings, SettingKeyGatewaySchedulingScoreWeightErrorRate, cfg.ScoreWeights.ErrorRate); err != nil {
		return cfg, err
	}
	if cfg.ScoreWeights.Latency, err = parseFloatSetting(settings, SettingKeyGatewaySchedulingScoreWeightLatency, cfg.ScoreWeights.Latency); err != nil {
		return cfg, err
	}
	if cfg.ScoreWeights.RateMultiplier, err = parseFloatSetting(settings, SettingKeyGatewaySchedulingScoreWeightRateMultiplier, cfg.ScoreWeights.RateMultiplier); err != nil {
		return cfg, err
	}
	if cfg.ScoreWeights.QuotaRisk, err = parseFloatSetting(settings, SettingKeyGatewaySchedulingScoreWeightQuotaRisk, cfg.ScoreWeights.QuotaRisk); err != nil {
		return cfg, err
	}
	if cfg.LatencyBaselineMS, err = parseIntSetting(settings, SettingKeyGatewaySchedulingLatencyBaselineMS, cfg.LatencyBaselineMS); err != nil {
		return cfg, err
	}
	if cfg.QuotaRiskThreshold, err = parseFloatSetting(settings, SettingKeyGatewaySchedulingQuotaRiskThreshold, cfg.QuotaRiskThreshold); err != nil {
		return cfg, err
	}
	if cfg.MaxScorePenalty, err = parseFloatSetting(settings, SettingKeyGatewaySchedulingMaxScorePenalty, cfg.MaxScorePenalty); err != nil {
		return cfg, err
	}
	if mode := strings.TrimSpace(settings[SettingKeyGatewaySchedulingStickySessionMode]); mode != "" {
		cfg.StickySessionMode = strings.ToLower(mode)
	}
	if cfg.StickyEscapeScoreRatio, err = parseFloatSetting(settings, SettingKeyGatewaySchedulingStickyEscapeScoreRatio, cfg.StickyEscapeScoreRatio); err != nil {
		return cfg, err
	}
	if cfg.StickyEscapeLoadRate, err = parseIntSetting(settings, SettingKeyGatewaySchedulingStickyEscapeLoadRate, cfg.StickyEscapeLoadRate); err != nil {
		return cfg, err
	}
	if cfg.ActiveProbe.AutoPauseEnabled, err = parseBoolSetting(settings, SettingKeyGatewaySchedulingActiveProbeAutoPause, cfg.ActiveProbe.AutoPauseEnabled); err != nil {
		return cfg, err
	}
	if cfg.ActiveProbe.FailureThreshold, err = parseIntSetting(settings, SettingKeyGatewaySchedulingActiveProbeFailureThreshold, cfg.ActiveProbe.FailureThreshold); err != nil {
		return cfg, err
	}
	if cfg.ActiveProbe.PauseDuration, err = parseDurationSetting(settings, SettingKeyGatewaySchedulingActiveProbePauseDuration, cfg.ActiveProbe.PauseDuration); err != nil {
		return cfg, err
	}
	if cfg.ActiveProbe.PauseDurationMax, err = parseDurationSetting(settings, SettingKeyGatewaySchedulingActiveProbePauseDurationMax, cfg.ActiveProbe.PauseDurationMax); err != nil {
		return cfg, err
	}
	if cfg.SlowStart.Enabled, err = parseBoolSetting(settings, SettingKeyGatewaySchedulingSlowStartEnabled, cfg.SlowStart.Enabled); err != nil {
		return cfg, err
	}
	if cfg.SlowStart.Duration, err = parseDurationSetting(settings, SettingKeyGatewaySchedulingSlowStartDuration, cfg.SlowStart.Duration); err != nil {
		return cfg, err
	}
	if cfg.SlowStart.Penalty, err = parseFloatSetting(settings, SettingKeyGatewaySchedulingSlowStartPenalty, cfg.SlowStart.Penalty); err != nil {
		return cfg, err
	}
	if cfg.UpstreamRate.Enabled, err = parseBoolSetting(settings, SettingKeyGatewaySchedulingUpstreamRateEnabled, cfg.UpstreamRate.Enabled); err != nil {
		return cfg, err
	}
	if cfg.UpstreamRate.StaleTTLSeconds, err = parseIntSetting(settings, SettingKeyGatewaySchedulingUpstreamRateStaleTTLSeconds, cfg.UpstreamRate.StaleTTLSeconds); err != nil {
		return cfg, err
	}
	if cfg.UpstreamRate.RateWeight, err = parseFloatSetting(settings, SettingKeyGatewaySchedulingUpstreamRateRateWeight, cfg.UpstreamRate.RateWeight); err != nil {
		return cfg, err
	}
	if cfg.UpstreamRate.HealthWeight, err = parseFloatSetting(settings, SettingKeyGatewaySchedulingUpstreamRateHealthWeight, cfg.UpstreamRate.HealthWeight); err != nil {
		return cfg, err
	}
	if cfg.UpstreamRate.MinSuccessRate, err = parseFloatSetting(settings, SettingKeyGatewaySchedulingUpstreamRateMinSuccessRate, cfg.UpstreamRate.MinSuccessRate); err != nil {
		return cfg, err
	}
	return validateGatewaySchedulingSettingsConfig(cfg)
}

func validateGatewaySchedulingSettingsConfig(cfg config.GatewaySchedulingConfig) (config.GatewaySchedulingConfig, error) {
	if cfg.PreferredAccountID < 0 {
		return cfg, fmt.Errorf("gateway.scheduling.preferred_account_id must be non-negative")
	}
	for groupID, accountID := range cfg.PreferredAccountByGroupID {
		if groupID < 0 || accountID < 0 {
			return cfg, fmt.Errorf("gateway.scheduling.preferred_account_by_group_id must contain non-negative ids")
		}
	}
	if cfg.ScoreWeights.Load < 0 || cfg.ScoreWeights.Queue < 0 || cfg.ScoreWeights.Debt < 0 || cfg.ScoreWeights.ErrorRate < 0 || cfg.ScoreWeights.Latency < 0 || cfg.ScoreWeights.RateMultiplier < 0 || cfg.ScoreWeights.QuotaRisk < 0 {
		return cfg, fmt.Errorf("gateway.scheduling.score_weights.* must be non-negative")
	}
	if cfg.ScoreWeights.Load+cfg.ScoreWeights.Queue+cfg.ScoreWeights.Debt+cfg.ScoreWeights.ErrorRate+cfg.ScoreWeights.Latency+cfg.ScoreWeights.RateMultiplier+cfg.ScoreWeights.QuotaRisk <= 0 {
		return cfg, fmt.Errorf("gateway.scheduling.score_weights must not all be zero")
	}
	if cfg.LatencyBaselineMS <= 0 {
		return cfg, fmt.Errorf("gateway.scheduling.latency_baseline_ms must be positive")
	}
	if cfg.QuotaRiskThreshold < 0 || cfg.QuotaRiskThreshold > 1 {
		return cfg, fmt.Errorf("gateway.scheduling.quota_risk_threshold must be between 0 and 1")
	}
	if cfg.MaxScorePenalty < 0 {
		return cfg, fmt.Errorf("gateway.scheduling.max_score_penalty must be non-negative")
	}
	switch cfg.StickySessionMode {
	case config.GatewayStickySessionModeStrict, config.GatewayStickySessionModeSoft, config.GatewayStickySessionModeOff:
	default:
		return cfg, fmt.Errorf("gateway.scheduling.sticky_session_mode must be one of %s|%s|%s", config.GatewayStickySessionModeStrict, config.GatewayStickySessionModeSoft, config.GatewayStickySessionModeOff)
	}
	if cfg.StickyEscapeScoreRatio < 1 {
		return cfg, fmt.Errorf("gateway.scheduling.sticky_escape_score_ratio must be >= 1")
	}
	if cfg.StickyEscapeLoadRate < 0 || cfg.StickyEscapeLoadRate > 100 {
		return cfg, fmt.Errorf("gateway.scheduling.sticky_escape_load_rate must be between 0 and 100")
	}
	if cfg.ActiveProbe.FailureThreshold <= 0 {
		return cfg, fmt.Errorf("gateway.scheduling.active_probe.failure_threshold must be positive")
	}
	if cfg.ActiveProbe.PauseDuration <= 0 {
		return cfg, fmt.Errorf("gateway.scheduling.active_probe.pause_duration must be positive")
	}
	if cfg.ActiveProbe.PauseDurationMax <= 0 {
		return cfg, fmt.Errorf("gateway.scheduling.active_probe.pause_duration_max must be positive")
	}
	if cfg.ActiveProbe.PauseDurationMax < cfg.ActiveProbe.PauseDuration {
		return cfg, fmt.Errorf("gateway.scheduling.active_probe.pause_duration_max must be >= pause_duration")
	}
	if cfg.SlowStart.Duration <= 0 {
		return cfg, fmt.Errorf("gateway.scheduling.slow_start.duration must be positive")
	}
	if cfg.SlowStart.Penalty < 0 {
		return cfg, fmt.Errorf("gateway.scheduling.slow_start.penalty must be non-negative")
	}
	if cfg.UpstreamRate.StaleTTLSeconds <= 0 {
		return cfg, fmt.Errorf("gateway.scheduling.upstream_rate.stale_ttl_seconds must be positive")
	}
	if cfg.UpstreamRate.RateWeight < 0 || cfg.UpstreamRate.HealthWeight < 0 {
		return cfg, fmt.Errorf("gateway.scheduling.upstream_rate weights must be non-negative")
	}
	if cfg.UpstreamRate.MinSuccessRate < 0 || cfg.UpstreamRate.MinSuccessRate > 1 {
		return cfg, fmt.Errorf("gateway.scheduling.upstream_rate.min_success_rate must be between 0 and 1")
	}
	return cfg, nil
}

func parseFloatSetting(settings map[string]string, key string, fallback float64) (float64, error) {
	raw := strings.TrimSpace(settings[key])
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback, fmt.Errorf("%s must be a number", key)
	}
	return value, nil
}

func parseIntSetting(settings map[string]string, key string, fallback int) (int, error) {
	raw := strings.TrimSpace(settings[key])
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback, fmt.Errorf("%s must be an integer", key)
	}
	return value, nil
}

func parseInt64Setting(settings map[string]string, key string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(settings[key])
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback, fmt.Errorf("%s must be an integer", key)
	}
	return value, nil
}

func parseBoolSetting(settings map[string]string, key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(settings[key])
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback, fmt.Errorf("%s must be a boolean", key)
	}
	return value, nil
}

func parseInt64MapSetting(settings map[string]string, key string, fallback map[int64]int64) (map[int64]int64, error) {
	raw := strings.TrimSpace(settings[key])
	if raw == "" {
		return cloneInt64Map(fallback), nil
	}
	var value map[int64]int64
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return cloneInt64Map(fallback), fmt.Errorf("%s must be a json object", key)
	}
	return cloneInt64Map(value), nil
}

func cloneInt64Map(input map[int64]int64) map[int64]int64 {
	if len(input) == 0 {
		return map[int64]int64{}
	}
	out := make(map[int64]int64, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func formatInt64MapSetting(value map[int64]int64) string {
	if len(value) == 0 {
		return "{}"
	}
	cleaned := make(map[int64]int64, len(value))
	for groupID, accountID := range value {
		if groupID >= 0 && accountID > 0 {
			cleaned[groupID] = accountID
		}
	}
	if len(cleaned) == 0 {
		return "{}"
	}
	data, err := json.Marshal(cleaned)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func parseDurationSetting(settings map[string]string, key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(settings[key])
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback, fmt.Errorf("%s must be a duration", key)
	}
	return value, nil
}

func gatewaySchedulingSettingsToMap(settings *SystemSettings) map[string]string {
	cfg := defaultGatewaySchedulingConfig()
	if settings != nil {
		cfg = normalizeGatewaySchedulingConfig(settings.GatewayScheduling)
	}
	return gatewaySchedulingConfigToMap(cfg)
}

func gatewaySchedulingConfigToMap(cfg config.GatewaySchedulingConfig) map[string]string {
	cfg = normalizeGatewaySchedulingConfig(cfg)
	return map[string]string{
		SettingKeyGatewaySchedulingPreferredAccountID:          strconv.FormatInt(cfg.PreferredAccountID, 10),
		SettingKeyGatewaySchedulingPreferredAccountByGroupID:   formatInt64MapSetting(cfg.PreferredAccountByGroupID),
		SettingKeyGatewaySchedulingScoreWeightLoad:             strconv.FormatFloat(cfg.ScoreWeights.Load, 'f', 8, 64),
		SettingKeyGatewaySchedulingScoreWeightQueue:            strconv.FormatFloat(cfg.ScoreWeights.Queue, 'f', 8, 64),
		SettingKeyGatewaySchedulingScoreWeightDebt:             strconv.FormatFloat(cfg.ScoreWeights.Debt, 'f', 8, 64),
		SettingKeyGatewaySchedulingScoreWeightErrorRate:        strconv.FormatFloat(cfg.ScoreWeights.ErrorRate, 'f', 8, 64),
		SettingKeyGatewaySchedulingScoreWeightLatency:          strconv.FormatFloat(cfg.ScoreWeights.Latency, 'f', 8, 64),
		SettingKeyGatewaySchedulingScoreWeightRateMultiplier:   strconv.FormatFloat(cfg.ScoreWeights.RateMultiplier, 'f', 8, 64),
		SettingKeyGatewaySchedulingScoreWeightQuotaRisk:        strconv.FormatFloat(cfg.ScoreWeights.QuotaRisk, 'f', 8, 64),
		SettingKeyGatewaySchedulingLatencyBaselineMS:           strconv.Itoa(cfg.LatencyBaselineMS),
		SettingKeyGatewaySchedulingQuotaRiskThreshold:          strconv.FormatFloat(cfg.QuotaRiskThreshold, 'f', 8, 64),
		SettingKeyGatewaySchedulingMaxScorePenalty:             strconv.FormatFloat(cfg.MaxScorePenalty, 'f', 8, 64),
		SettingKeyGatewaySchedulingStickySessionMode:           cfg.StickySessionMode,
		SettingKeyGatewaySchedulingStickyEscapeScoreRatio:      strconv.FormatFloat(cfg.StickyEscapeScoreRatio, 'f', 8, 64),
		SettingKeyGatewaySchedulingStickyEscapeLoadRate:        strconv.Itoa(cfg.StickyEscapeLoadRate),
		SettingKeyGatewaySchedulingActiveProbeAutoPause:        strconv.FormatBool(cfg.ActiveProbe.AutoPauseEnabled),
		SettingKeyGatewaySchedulingActiveProbeFailureThreshold: strconv.Itoa(cfg.ActiveProbe.FailureThreshold),
		SettingKeyGatewaySchedulingActiveProbePauseDuration:    cfg.ActiveProbe.PauseDuration.String(),
		SettingKeyGatewaySchedulingActiveProbePauseDurationMax: cfg.ActiveProbe.PauseDurationMax.String(),
		SettingKeyGatewaySchedulingSlowStartEnabled:            strconv.FormatBool(cfg.SlowStart.Enabled),
		SettingKeyGatewaySchedulingSlowStartDuration:           cfg.SlowStart.Duration.String(),
		SettingKeyGatewaySchedulingSlowStartPenalty:            strconv.FormatFloat(cfg.SlowStart.Penalty, 'f', 8, 64),
		SettingKeyGatewaySchedulingUpstreamRateEnabled:         strconv.FormatBool(cfg.UpstreamRate.Enabled),
		SettingKeyGatewaySchedulingUpstreamRateStaleTTLSeconds: strconv.Itoa(cfg.UpstreamRate.StaleTTLSeconds),
		SettingKeyGatewaySchedulingUpstreamRateRateWeight:      strconv.FormatFloat(cfg.UpstreamRate.RateWeight, 'f', 8, 64),
		SettingKeyGatewaySchedulingUpstreamRateHealthWeight:    strconv.FormatFloat(cfg.UpstreamRate.HealthWeight, 'f', 8, 64),
		SettingKeyGatewaySchedulingUpstreamRateMinSuccessRate:  strconv.FormatFloat(cfg.UpstreamRate.MinSuccessRate, 'f', 8, 64),
	}
}
