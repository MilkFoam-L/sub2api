package service

import (
	"math"
	mathrand "math/rand"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type schedulerStickyDecision struct {
	useSticky bool
	reason    string
}

func defaultGatewaySchedulingConfig() config.GatewaySchedulingConfig {
	return config.GatewaySchedulingConfig{
		StickySessionMaxWaiting:  3,
		StickySessionWaitTimeout: 120 * time.Second,
		FallbackWaitTimeout:      30 * time.Second,
		FallbackMaxWaiting:       100,
		Algorithm:                config.GatewaySchedulingAlgorithmWeightedP2C,
		P2CChoiceCount:           2,
		SelectionDebtTTLMS:       5000,
		SelectionDebtWeight:      1,
		WaitPenalty:              1,
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
		StickySessionMode:      config.GatewayStickySessionModeSoft,
		StickyEscapeScoreRatio: 1.25,
		StickyEscapeLoadRate:   75,
		LoadBatchEnabled:       true,
	}
}

func normalizeGatewaySchedulingConfig(cfg config.GatewaySchedulingConfig) config.GatewaySchedulingConfig {
	cfg.Algorithm = strings.ToLower(strings.TrimSpace(cfg.Algorithm))
	cfg.StickySessionMode = strings.ToLower(strings.TrimSpace(cfg.StickySessionMode))
	if cfg.Algorithm == "" {
		cfg.Algorithm = config.GatewaySchedulingAlgorithmWeightedP2C
	}
	if cfg.P2CChoiceCount <= 0 {
		cfg.P2CChoiceCount = 2
	}
	if cfg.StickySessionMode == "" {
		cfg.StickySessionMode = config.GatewayStickySessionModeSoft
	}
	if cfg.StickyEscapeScoreRatio == 0 {
		cfg.StickyEscapeScoreRatio = 1.25
	}
	cfg.ScoreWeights = normalizeGatewaySchedulingScoreWeights(cfg.ScoreWeights)
	if cfg.LatencyBaselineMS <= 0 {
		cfg.LatencyBaselineMS = 15000
	}
	if cfg.QuotaRiskThreshold <= 0 || cfg.QuotaRiskThreshold > 1 {
		cfg.QuotaRiskThreshold = 0.2
	}
	if cfg.MaxScorePenalty <= 0 {
		cfg.MaxScorePenalty = 5
	}
	if cfg.ActiveProbe.FailureThreshold <= 0 {
		cfg.ActiveProbe.FailureThreshold = 3
	}
	if cfg.ActiveProbe.PauseDuration <= 0 {
		cfg.ActiveProbe.PauseDuration = 10 * time.Minute
	}
	if cfg.ActiveProbe.PauseDurationMax <= 0 || cfg.ActiveProbe.PauseDurationMax < cfg.ActiveProbe.PauseDuration {
		cfg.ActiveProbe.PauseDurationMax = time.Hour
	}
	if cfg.SlowStart.Duration <= 0 {
		cfg.SlowStart.Duration = 5 * time.Minute
	}
	if cfg.SlowStart.Penalty == 0 {
		cfg.SlowStart.Penalty = 1
	}
	return cfg
}

func normalizeGatewaySchedulingScoreWeights(weights config.GatewaySchedulingScoreWeights) config.GatewaySchedulingScoreWeights {
	if weights.Load == 0 && weights.Queue == 0 && weights.Debt == 0 && weights.ErrorRate == 0 && weights.Latency == 0 && weights.RateMultiplier == 0 && weights.QuotaRisk == 0 {
		return config.GatewaySchedulingScoreWeights{
			Load:           1,
			Queue:          1,
			Debt:           1,
			ErrorRate:      0.8,
			Latency:        0.4,
			RateMultiplier: 0.6,
			QuotaRisk:      0.3,
		}
	}
	return weights
}

func schedulerAccountCapacity(account *Account) float64 {
	if account == nil {
		return 1
	}
	capacity := account.EffectiveLoadFactor()
	if capacity <= 0 {
		return 1
	}
	return float64(capacity)
}

func schedulerAccountCost(item accountWithLoad, selectionDebt int, cfg config.GatewaySchedulingConfig) float64 {
	cfg = normalizeGatewaySchedulingConfig(cfg)
	weights := cfg.ScoreWeights

	load := 0.0
	if item.loadInfo != nil {
		load += float64(item.loadInfo.CurrentConcurrency) * weights.Load
		load += float64(item.loadInfo.WaitingCount) * cfg.WaitPenalty * weights.Queue
		if load == 0 && item.loadInfo.LoadRate > 0 {
			load = schedulerAccountCapacity(item.account) * float64(item.loadInfo.LoadRate) / 100 * weights.Load
		}
	}
	if selectionDebt > 0 && cfg.SelectionDebtWeight > 0 && weights.Debt > 0 {
		load += float64(selectionDebt) * cfg.SelectionDebtWeight * weights.Debt
	}
	capacity := schedulerAccountCapacity(item.account)
	cost := load / capacity
	cost += schedulerCapPenalty(item.selectionScorePenalty, cfg.MaxScorePenalty)
	cost += schedulerSoftPenalty(item.runtimeStatsPenalty(weights, cfg))
	cost += schedulerSoftPenalty(schedulerRateMultiplierPenalty(item.account, weights.RateMultiplier))
	cost += schedulerSoftPenalty(schedulerQuotaRiskPenalty(item.account, weights.QuotaRisk, cfg.QuotaRiskThreshold))
	if math.IsNaN(cost) || math.IsInf(cost, 0) || cost < 0 {
		return math.MaxFloat64
	}
	return cost
}

func (item accountWithLoad) runtimeStatsPenalty(weights config.GatewaySchedulingScoreWeights, cfg config.GatewaySchedulingConfig) float64 {
	if item.runtimeStats == nil {
		return 0
	}
	penalty := item.runtimeStats.ErrorRate * weights.ErrorRate
	if item.runtimeStats.HasLatency && weights.Latency > 0 && cfg.LatencyBaselineMS > 0 {
		ratio := item.runtimeStats.LatencyMs/float64(cfg.LatencyBaselineMS) - 1
		if ratio > 0 {
			penalty += ratio * weights.Latency
		}
	}
	if cfg.SlowStart.Enabled && cfg.SlowStart.Penalty > 0 && item.runtimeStats.HasSlowStart {
		if remaining := time.Until(item.runtimeStats.SlowStartUntil); remaining > 0 {
			ratio := 1.0
			if cfg.SlowStart.Duration > 0 && remaining < cfg.SlowStart.Duration {
				ratio = remaining.Seconds() / cfg.SlowStart.Duration.Seconds()
			}
			penalty += ratio * cfg.SlowStart.Penalty
		}
	}
	return schedulerCapPenalty(penalty, cfg.MaxScorePenalty)
}

func schedulerRateMultiplierPenalty(account *Account, weight float64) float64 {
	if account == nil || weight <= 0 {
		return 0
	}
	multiplier := account.BillingRateMultiplier()
	if multiplier <= 1 {
		return 0
	}
	return (multiplier - 1) * weight
}

func schedulerQuotaRiskPenalty(account *Account, weight float64, threshold float64) float64 {
	if account == nil || weight <= 0 || threshold <= 0 {
		return 0
	}
	risk := schedulerQuotaRiskRatio(account, threshold)
	if risk <= 0 {
		return 0
	}
	return risk * weight
}

func schedulerQuotaRiskRatio(account *Account, threshold float64) float64 {
	if account == nil {
		return 0
	}
	maxRisk := 0.0
	maxRisk = math.Max(maxRisk, quotaRiskForLimit(account.GetQuotaUsed(), account.GetQuotaLimit(), threshold))
	if !account.IsDailyQuotaPeriodExpired() {
		maxRisk = math.Max(maxRisk, quotaRiskForLimit(account.GetQuotaDailyUsed(), account.GetQuotaDailyLimit(), threshold))
	}
	if !account.IsWeeklyQuotaPeriodExpired() {
		maxRisk = math.Max(maxRisk, quotaRiskForLimit(account.GetQuotaWeeklyUsed(), account.GetQuotaWeeklyLimit(), threshold))
	}
	return maxRisk
}

func quotaRiskForLimit(used float64, limit float64, threshold float64) float64 {
	if limit <= 0 || threshold <= 0 || used < 0 {
		return 0
	}
	remainingRatio := (limit - used) / limit
	if remainingRatio >= threshold {
		return 0
	}
	if remainingRatio < 0 {
		remainingRatio = 0
	}
	return (threshold - remainingRatio) / threshold
}

func schedulerSoftPenalty(penalty float64) float64 {
	if math.IsNaN(penalty) || math.IsInf(penalty, 0) || penalty < 0 {
		return 0
	}
	return penalty
}

func schedulerCapPenalty(penalty float64, maxPenalty float64) float64 {
	penalty = schedulerSoftPenalty(penalty)
	if maxPenalty > 0 && penalty > maxPenalty {
		return maxPenalty
	}
	return penalty
}

func buildWeightedP2CSelectionOrder(accounts []accountWithLoad, selectionDebts map[int64]int, preferOAuth bool, cfg config.GatewaySchedulingConfig) []accountWithLoad {
	if len(accounts) == 0 {
		return nil
	}
	cfg = normalizeGatewaySchedulingConfig(cfg)
	if cfg.Algorithm == config.GatewaySchedulingAlgorithmLegacyLRU {
		return buildLegacyLRUSelectionOrder(accounts, preferOAuth)
	}

	pool := append([]accountWithLoad(nil), accounts...)
	order := make([]accountWithLoad, 0, len(pool))
	for len(pool) > 0 {
		minPriority := pool[0].account.Priority
		for _, item := range pool[1:] {
			if item.account.Priority < minPriority {
				minPriority = item.account.Priority
			}
		}

		layerIdxs := make([]int, 0, len(pool))
		for idx, item := range pool {
			if item.account.Priority == minPriority {
				layerIdxs = append(layerIdxs, idx)
			}
		}
		selectedPoolIdx := selectWeightedP2CIndex(pool, layerIdxs, selectionDebts, preferOAuth, cfg)
		order = append(order, pool[selectedPoolIdx])
		pool = append(pool[:selectedPoolIdx], pool[selectedPoolIdx+1:]...)
	}
	return order
}

func buildLegacyLRUSelectionOrder(accounts []accountWithLoad, preferOAuth bool) []accountWithLoad {
	pool := append([]accountWithLoad(nil), accounts...)
	order := make([]accountWithLoad, 0, len(pool))
	for len(pool) > 0 {
		candidates := filterByMinPriority(pool)
		candidates = filterByMinLoadRate(candidates)
		selected := selectByLRU(candidates, preferOAuth)
		if selected == nil || selected.account == nil {
			break
		}
		selectedID := selected.account.ID
		for i, item := range pool {
			if item.account.ID == selectedID {
				order = append(order, item)
				pool = append(pool[:i], pool[i+1:]...)
				break
			}
		}
	}
	return order
}

func selectWeightedP2CIndex(pool []accountWithLoad, candidateIdxs []int, debts map[int64]int, preferOAuth bool, cfg config.GatewaySchedulingConfig) int {
	if len(candidateIdxs) == 1 {
		return candidateIdxs[0]
	}
	choiceCount := cfg.P2CChoiceCount
	if choiceCount <= 0 {
		choiceCount = 2
	}
	if choiceCount > len(candidateIdxs) {
		choiceCount = len(candidateIdxs)
	}

	choices := make([]int, 0, choiceCount)
	used := make(map[int]struct{}, choiceCount)
	for len(choices) < choiceCount {
		idx := weightedRandomCandidateIndex(pool, candidateIdxs, used)
		used[idx] = struct{}{}
		choices = append(choices, idx)
	}

	best := choices[0]
	bestCost := schedulerAccountCost(pool[best], debts[pool[best].account.ID], cfg)
	for _, idx := range choices[1:] {
		cost := schedulerAccountCost(pool[idx], debts[pool[idx].account.ID], cfg)
		if cost < bestCost || (cost == bestCost && isSchedulerTieBetter(pool[idx], pool[best], preferOAuth)) {
			best = idx
			bestCost = cost
		}
	}
	return best
}

func weightedRandomCandidateIndex(pool []accountWithLoad, candidateIdxs []int, used map[int]struct{}) int {
	total := 0.0
	for _, idx := range candidateIdxs {
		if _, ok := used[idx]; ok {
			continue
		}
		total += schedulerAccountCapacity(pool[idx].account)
	}
	if total <= 0 {
		for _, idx := range candidateIdxs {
			if _, ok := used[idx]; !ok {
				return idx
			}
		}
		return candidateIdxs[0]
	}
	r := mathrand.Float64() * total
	accum := 0.0
	for _, idx := range candidateIdxs {
		if _, ok := used[idx]; ok {
			continue
		}
		accum += schedulerAccountCapacity(pool[idx].account)
		if r <= accum {
			return idx
		}
	}
	for _, idx := range candidateIdxs {
		if _, ok := used[idx]; !ok {
			return idx
		}
	}
	return candidateIdxs[len(candidateIdxs)-1]
}

func isSchedulerTieBetter(left accountWithLoad, right accountWithLoad, preferOAuth bool) bool {
	if preferOAuth && left.account.Type != right.account.Type {
		return left.account.Type == AccountTypeOAuth
	}
	return mathrand.Intn(2) == 0
}

func shouldUseStickyAccountForScheduling(sticky accountWithLoad, candidates []accountWithLoad, debts map[int64]int, cfg config.GatewaySchedulingConfig) schedulerStickyDecision {
	cfg = normalizeGatewaySchedulingConfig(cfg)
	switch cfg.StickySessionMode {
	case config.GatewayStickySessionModeOff:
		return schedulerStickyDecision{useSticky: false, reason: "off"}
	case config.GatewayStickySessionModeStrict:
		return schedulerStickyDecision{useSticky: true, reason: "strict"}
	}
	if sticky.account == nil {
		return schedulerStickyDecision{useSticky: false, reason: "missing"}
	}
	if sticky.loadInfo != nil && cfg.StickyEscapeLoadRate > 0 && sticky.loadInfo.LoadRate >= cfg.StickyEscapeLoadRate {
		return schedulerStickyDecision{useSticky: false, reason: "load_rate"}
	}
	if len(candidates) == 0 || cfg.StickyEscapeScoreRatio <= 0 {
		return schedulerStickyDecision{useSticky: true, reason: "soft"}
	}

	stickyCost := schedulerAccountCost(sticky, debts[sticky.account.ID], cfg)
	bestCost := stickyCost
	for _, candidate := range candidates {
		if candidate.account == nil || candidate.account.Priority != sticky.account.Priority {
			continue
		}
		cost := schedulerAccountCost(candidate, debts[candidate.account.ID], cfg)
		if cost < bestCost {
			bestCost = cost
		}
	}
	if stickyCost > bestCost*cfg.StickyEscapeScoreRatio {
		return schedulerStickyDecision{useSticky: false, reason: "score"}
	}
	return schedulerStickyDecision{useSticky: true, reason: "soft"}
}

func accountSelectionDebtTTL(cfg config.GatewaySchedulingConfig) time.Duration {
	cfg = normalizeGatewaySchedulingConfig(cfg)
	if cfg.SelectionDebtTTLMS <= 0 {
		return 0
	}
	return time.Duration(cfg.SelectionDebtTTLMS) * time.Millisecond
}

func accountIDsFromLoadItems(items []accountWithLoad) []int64 {
	ids := make([]int64, 0, len(items))
	seen := make(map[int64]struct{}, len(items))
	for _, item := range items {
		if item.account == nil || item.account.ID <= 0 {
			continue
		}
		if _, ok := seen[item.account.ID]; ok {
			continue
		}
		seen[item.account.ID] = struct{}{}
		ids = append(ids, item.account.ID)
	}
	return ids
}

func sortAccountWithLoadByCost(accounts []accountWithLoad, debts map[int64]int, cfg config.GatewaySchedulingConfig) []accountWithLoad {
	ordered := append([]accountWithLoad(nil), accounts...)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.account.Priority != b.account.Priority {
			return a.account.Priority < b.account.Priority
		}
		return schedulerAccountCost(a, debts[a.account.ID], cfg) < schedulerAccountCost(b, debts[b.account.ID], cfg)
	})
	return ordered
}
