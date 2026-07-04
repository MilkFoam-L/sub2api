package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type upstreamHTTPPayload struct {
	status int
	body   []byte
}

func (s *UpstreamRateService) collectSub2API(ctx context.Context, source *UpstreamRateSource) ([]*UpstreamRateSnapshot, int, error) {
	available, status, err := s.getUpstreamJSON(ctx, source, "/api/v1/groups/available")
	if err != nil {
		return nil, status, err
	}
	rates, rateStatus, err := s.getUpstreamJSON(ctx, source, "/api/v1/groups/rates")
	if err != nil {
		return nil, rateStatus, err
	}
	userRates := parseSub2APIRateMap(rates.body)
	groups, err := parseSub2APIAvailableGroups(available.body)
	if err != nil {
		return nil, status, err
	}
	multiplier := rechargeMultiplierValue(source)
	snapshots := make([]*UpstreamRateSnapshot, 0, len(groups))
	for _, group := range groups {
		raw := group.RateMultiplier
		if user, ok := userRates[group.Key]; ok && user > 0 {
			raw = user
		}
		if raw <= 0 {
			raw = 1
		}
		snapshots = append(snapshots, &UpstreamRateSnapshot{
			UpstreamGroupKey:        group.Key,
			UpstreamGroupName:       group.Name,
			RawRateMultiplier:       raw,
			EffectiveRateMultiplier: raw / multiplier,
			PeakRateEnabled:         group.PeakRateEnabled,
			PeakRateMultiplier:      group.PeakRateMultiplier,
		})
	}
	return sortedUpstreamSnapshots(snapshots), status, nil
}

func (s *UpstreamRateService) collectNewAPI(ctx context.Context, source *UpstreamRateSource) ([]*UpstreamRateSnapshot, int, error) {
	payload, status, err := s.getUpstreamJSON(ctx, source, "/api/ratio_config")
	if err != nil {
		return nil, status, err
	}
	groupRates, modelJSON, completionJSON, err := parseNewAPIRatioConfig(payload.body)
	if err != nil {
		return nil, status, err
	}
	multiplier := rechargeMultiplierValue(source)
	snapshots := make([]*UpstreamRateSnapshot, 0, len(groupRates))
	for key, raw := range groupRates {
		if raw <= 0 {
			raw = 1
		}
		snapshots = append(snapshots, &UpstreamRateSnapshot{
			UpstreamGroupKey:        key,
			UpstreamGroupName:       key,
			RawRateMultiplier:       raw,
			EffectiveRateMultiplier: raw / multiplier,
			ModelRatioJSON:          modelJSON,
			CompletionRatioJSON:     completionJSON,
		})
	}
	return sortedUpstreamSnapshots(snapshots), status, nil
}

func (s *UpstreamRateService) getUpstreamJSON(ctx context.Context, source *UpstreamRateSource, path string) (*upstreamHTTPPayload, int, error) {
	base, err := url.Parse(strings.TrimRight(source.BaseURL, "/"))
	if err != nil {
		return nil, 0, err
	}
	base.Path = strings.TrimRight(base.Path, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	if source.AuthMode == UpstreamRateAuthModeBearerToken && strings.TrimSpace(source.Token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(source.Token))
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if readErr != nil {
		return nil, resp.StatusCode, readErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("upstream %s returned HTTP %d", path, resp.StatusCode)
	}
	return &upstreamHTTPPayload{status: resp.StatusCode, body: body}, resp.StatusCode, nil
}

type sub2APIGroupRate struct {
	Key                string
	Name               string
	RateMultiplier     float64
	PeakRateEnabled    bool
	PeakRateMultiplier float64
}

func parseSub2APIAvailableGroups(body []byte) ([]sub2APIGroupRate, error) {
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	rows := unwrapArrayPayload(raw)
	if len(rows) == 0 {
		return nil, fmt.Errorf("no upstream groups found")
	}
	out := make([]sub2APIGroupRate, 0, len(rows))
	for _, row := range rows {
		m, ok := row.(map[string]any)
		if !ok {
			continue
		}
		key := firstString(m, "id", "group_id", "key", "name")
		if key == "" {
			continue
		}
		name := firstString(m, "name", "group_name")
		if name == "" {
			name = key
		}
		rate := firstFloat(m, "rate_multiplier", "rate", "ratio")
		if rate <= 0 {
			rate = 1
		}
		peak := firstFloat(m, "peak_rate_multiplier")
		if peak <= 0 {
			peak = 1
		}
		out = append(out, sub2APIGroupRate{Key: key, Name: name, RateMultiplier: rate, PeakRateEnabled: firstBool(m, "peak_rate_enabled"), PeakRateMultiplier: peak})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no parsable upstream groups found")
	}
	return out, nil
}

func parseSub2APIRateMap(body []byte) map[string]float64 {
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return map[string]float64{}
	}
	if m, ok := unwrapObjectPayload(raw); ok {
		out := make(map[string]float64, len(m))
		for key, value := range m {
			if rate := anyToFloat(value); rate > 0 {
				out[key] = rate
			}
		}
		return out
	}
	return map[string]float64{}
}

func parseNewAPIRatioConfig(body []byte) (map[string]float64, string, string, error) {
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, "", "", err
	}
	obj, ok := unwrapObjectPayload(raw)
	if !ok {
		return nil, "", "", fmt.Errorf("ratio_config response is not an object")
	}
	groupObj, _ := unwrapObjectPayload(firstAny(obj, "GroupRatio", "group_ratio", "groupRatio"))
	if len(groupObj) == 0 {
		return nil, "", "", fmt.Errorf("GroupRatio is empty")
	}
	groupRates := make(map[string]float64, len(groupObj))
	for key, value := range groupObj {
		if rate := anyToFloat(value); rate > 0 {
			groupRates[key] = rate
		}
	}
	modelJSON := marshalJSONString(firstAny(obj, "ModelRatio", "model_ratio", "modelRatio"))
	completionJSON := marshalJSONString(firstAny(obj, "CompletionRatio", "completion_ratio", "completionRatio"))
	return groupRates, modelJSON, completionJSON, nil
}

func unwrapArrayPayload(raw any) []any {
	switch v := raw.(type) {
	case []any:
		return v
	case map[string]any:
		for _, key := range []string{"data", "groups", "items"} {
			if arr, ok := v[key].([]any); ok {
				return arr
			}
		}
	}
	return nil
}

func unwrapObjectPayload(raw any) (map[string]any, bool) {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	for _, key := range []string{"data", "result"} {
		if nested, ok := m[key].(map[string]any); ok {
			return nested, true
		}
	}
	return m, true
}

func firstAny(m map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value
		}
	}
	return nil
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			switch v := value.(type) {
			case string:
				if strings.TrimSpace(v) != "" {
					return strings.TrimSpace(v)
				}
			case float64:
				return strconv.FormatInt(int64(v), 10)
			case int:
				return strconv.Itoa(v)
			}
		}
	}
	return ""
}

func firstFloat(m map[string]any, keys ...string) float64 {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			if f := anyToFloat(value); f != 0 {
				return f
			}
		}
	}
	return 0
}

func firstBool(m map[string]any, keys ...string) bool {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			if b, ok := value.(bool); ok {
				return b
			}
		}
	}
	return false
}

func anyToFloat(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f
	default:
		return 0
	}
}

func marshalJSONString(value any) string {
	if value == nil {
		return ""
	}
	b, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(b)
}
