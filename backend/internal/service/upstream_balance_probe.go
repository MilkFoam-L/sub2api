package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"golang.org/x/sync/errgroup"
)

const (
	UpstreamBalanceProbeExtraKey        = "upstream_balance_probe"
	UpstreamBalanceProbeEnabledExtraKey = "upstream_balance_probe_enabled"
)

type UpstreamBalanceProbeSnapshot struct {
	Status        string         `json:"status"`
	Data          map[string]any `json:"data,omitempty"`
	ReceivedAt    *time.Time     `json:"received_at,omitempty"`
	FreshUntil    *time.Time     `json:"fresh_until,omitempty"`
	LastAttemptAt time.Time      `json:"last_attempt_at"`
	NextProbeAt   time.Time      `json:"next_probe_at"`
	FailureCount  int            `json:"failure_count,omitempty"`
	HTTPStatus    int            `json:"http_status,omitempty"`
	LastError     string         `json:"last_error,omitempty"`
}

type UpstreamBalanceProbeResult struct {
	AccountID int64                         `json:"account_id"`
	Snapshot  *UpstreamBalanceProbeSnapshot `json:"snapshot,omitempty"`
	Error     string                        `json:"error,omitempty"`
}

type upstreamBalanceSnapshotWriter interface {
	UpdateUpstreamBalanceProbeSnapshot(context.Context, *Account, *UpstreamBalanceProbeSnapshot) error
}
type upstreamBalanceDueLister interface {
	ListDueUpstreamBalanceProbeAccounts(context.Context, time.Time, int) ([]Account, error)
}

func decodeUpstreamBalanceProbeSnapshot(extra map[string]any) *UpstreamBalanceProbeSnapshot {
	if extra == nil {
		return nil
	}
	raw, err := json.Marshal(extra[UpstreamBalanceProbeExtraKey])
	if err != nil {
		return nil
	}
	var s UpstreamBalanceProbeSnapshot
	if json.Unmarshal(raw, &s) != nil || s.Status == "" {
		return nil
	}
	return &s
}
func upstreamBalanceProbeEnabled(a *Account) bool {
	if a == nil || a.Extra == nil {
		return false
	}
	v, ok := a.Extra[UpstreamBalanceProbeEnabledExtraKey].(bool)
	return ok && v
}
func (s *UpstreamBillingProbeService) SetBalanceAccountEnabled(ctx context.Context, id int64, enabled bool) error {
	if s == nil || s.accountRepo == nil {
		return ErrUpstreamBillingProbeUnavailable
	}
	a, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !isUpstreamBillingProbeAccount(a) {
		return ErrUpstreamBillingProbeAccountInvalid
	}
	return s.accountRepo.UpdateExtra(ctx, id, map[string]any{UpstreamBalanceProbeEnabledExtraKey: enabled})
}
func (s *UpstreamBillingProbeService) ProbeBalanceAccount(ctx context.Context, id int64) (*UpstreamBalanceProbeSnapshot, error) {
	if s == nil || s.accountRepo == nil {
		return nil, ErrUpstreamBillingProbeUnavailable
	}
	settings, err := s.getSettings(ctx)
	if err != nil {
		return nil, err
	}
	return s.probeBalanceAccountWithMode(ctx, id, settings.BalanceIntervalMinutes, false)
}

func (s *UpstreamBillingProbeService) probeBalanceAccountWithMode(ctx context.Context, id int64, interval int, requireEnabled bool) (*UpstreamBalanceProbeSnapshot, error) {
	key := "balance:" + strconv.FormatInt(id, 10)
	value, err, _ := s.probeGroup.Do(key, func() (any, error) {
		select {
		case s.probeSlots <- struct{}{}:
			defer func() { <-s.probeSlots }()
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		a, err := s.accountRepo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if !isUpstreamBillingProbeAccount(a) {
			return nil, ErrUpstreamBillingProbeAccountInvalid
		}
		if requireEnabled {
			if !a.IsActive() || !upstreamBalanceProbeEnabled(a) {
				return nil, nil
			}
			if snapshot := decodeUpstreamBalanceProbeSnapshot(a.Extra); snapshot != nil &&
				!snapshot.NextProbeAt.IsZero() && s.currentTime().Before(snapshot.NextProbeAt) {
				return nil, nil
			}
		}
		return s.probeLoadedBalanceAccount(ctx, a, interval)
	})
	if err != nil || value == nil {
		return nil, err
	}
	snapshot, ok := value.(*UpstreamBalanceProbeSnapshot)
	if !ok {
		return nil, fmt.Errorf("invalid upstream balance probe result")
	}
	return snapshot, nil
}
func (s *UpstreamBillingProbeService) ProbeBalanceAccounts(ctx context.Context, ids []int64) []UpstreamBalanceProbeResult {
	if len(ids) > UpstreamBillingProbeMaxBatchSize {
		ids = ids[:UpstreamBillingProbeMaxBatchSize]
	}
	out := make([]UpstreamBalanceProbeResult, len(ids))
	var wg sync.WaitGroup
	for i, id := range ids {
		out[i].AccountID = id
		i, id := i, id
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, e := s.ProbeBalanceAccount(ctx, id)
			if e != nil {
				out[i].Error = safeProbeError(e)
			} else {
				out[i].Snapshot = v
			}
		}()
	}
	wg.Wait()
	return out
}
func (s *UpstreamBillingProbeService) RunBalanceDue(ctx context.Context) error {
	if s == nil || s.accountRepo == nil {
		return nil
	}
	settings, err := s.getSettings(ctx)
	if err != nil || !settings.BalanceEnabled {
		return err
	}
	release, acquired, err := s.tryAcquireLeaderLock(ctx, upstreamBillingProbeLeaderLockKey+":balance")
	if err != nil || !acquired {
		return err
	}
	defer release()
	cadenceRelease, acquired, err := s.tryAcquireLeaderLock(ctx, upstreamBalanceProbeLeaderLockKeyAt(s.currentTime()))
	if err != nil || !acquired {
		return err
	}
	defer releaseUpstreamBillingProbeLeaderLock(cadenceRelease, s.currentTime().Truncate(upstreamBillingProbeCycleInterval).Add(upstreamBillingProbeCycleInterval))

	var accounts []Account
	if l, ok := s.accountRepo.(upstreamBalanceDueLister); ok {
		accounts, err = l.ListDueUpstreamBalanceProbeAccounts(ctx, s.currentTime(), upstreamBillingProbeMaxPerCycle)
	} else {
		accounts, err = s.accountRepo.FindByExtraField(ctx, UpstreamBalanceProbeEnabledExtraKey, true)
	}
	if err != nil {
		return err
	}
	now := s.currentTime()
	var group errgroup.Group
	for i := range accounts {
		accountID := accounts[i].ID
		account := accounts[i]
		if !account.IsActive() || !isUpstreamBillingProbeAccount(&account) || !upstreamBalanceProbeEnabled(&account) {
			continue
		}
		if snap := decodeUpstreamBalanceProbeSnapshot(account.Extra); snap != nil && !snap.NextProbeAt.IsZero() && now.Before(snap.NextProbeAt) {
			continue
		}
		group.Go(func() error {
			if _, probeErr := s.probeBalanceAccountWithMode(ctx, accountID, settings.BalanceIntervalMinutes, true); probeErr != nil {
				logger.LegacyPrintf("service.upstream_balance_probe", "probe_due_failed: account_id=%d err=%v", accountID, probeErr)
			}
			return nil
		})
	}
	return group.Wait()
}

func upstreamBalanceProbeLeaderLockKeyAt(now time.Time) string {
	return fmt.Sprintf("%s:balance:%d", upstreamBillingProbeLeaderLockKey, now.Unix()/int64(upstreamBillingProbeCycleInterval/time.Second))
}
func (s *UpstreamBillingProbeService) probeLoadedBalanceAccount(ctx context.Context, a *Account, interval int) (*UpstreamBalanceProbeSnapshot, error) {
	now := s.currentTime().UTC()
	fail := func(reason string, status int) (*UpstreamBalanceProbeSnapshot, error) {
		prev := decodeUpstreamBalanceProbeSnapshot(a.Extra)
		n := 1
		if prev != nil {
			n = prev.FailureCount + 1
		}
		snap := &UpstreamBalanceProbeSnapshot{Status: UpstreamBillingProbeStatusFailed, LastAttemptAt: now, NextProbeAt: now.Add(nextProbeDelay(interval, 0)), FailureCount: n, HTTPStatus: status, LastError: reason}
		if w, ok := s.accountRepo.(upstreamBalanceSnapshotWriter); ok {
			return snap, w.UpdateUpstreamBalanceProbeSnapshot(ctx, a, snap)
		}
		return nil, ErrUpstreamBillingProbeUnavailable
	}
	if s.accountTestService == nil || s.accountTestService.httpUpstream == nil {
		return fail("transport_unavailable", 0)
	}
	key := a.GetOpenAIApiKey()
	if key == "" {
		return fail("missing_api_key", 0)
	}
	base := a.GetOpenAIBaseURL()
	if base == "" {
		base = "https://api.openai.com"
	}
	base, err := s.accountTestService.validateUpstreamBaseURL(base)
	if err != nil {
		return fail("invalid_base_url", 0)
	}
	proxyURL := ""
	if a.ProxyID != nil {
		if a.Proxy == nil || a.Proxy.ID != *a.ProxyID {
			return fail("proxy_unavailable", 0)
		}
		proxyURL = a.Proxy.URL()
	}
	var tlsProfile *tlsfingerprint.Profile
	if s.accountTestService.tlsFPProfileService != nil {
		tlsProfile = s.accountTestService.tlsFPProfileService.ResolveTLSProfile(a)
	}
	resp, body, e := s.doProbeRequest(ctx, a, tlsProfile, proxyURL, key, http.MethodGet, buildOpenAIEndpointURL(base, "/v1/usage"), upstreamBillingProbeMaxBodyBytes, true)
	if e != nil {
		return fail(e.lastError, statusCodeOf(resp))
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		data, pe := parseSub2APIUsageResponse(body)
		if pe != nil {
			return fail("invalid_response", resp.StatusCode)
		}
		return s.saveBalance(ctx, a, now, interval, data, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusMethodNotAllowed {
		return fail("http_error", resp.StatusCode)
	}
	tokenURL, urlErr := buildNewAPIControlEndpointURL(base, "/api/usage/token/")
	if urlErr != nil {
		return fail("invalid_base_url", 0)
	}
	tr, tb, te := s.doProbeRequest(ctx, a, tlsProfile, proxyURL, key, http.MethodGet, tokenURL, upstreamBillingProbeMaxBodyBytes, true)
	if te != nil {
		return fail(te.lastError, statusCodeOf(tr))
	}
	if tr.StatusCode < 200 || tr.StatusCode >= 300 {
		return fail(classifyNewAPIBalanceHTTPError(tr.StatusCode), tr.StatusCode)
	}
	statusURL, urlErr := buildNewAPIControlEndpointURL(base, "/api/status")
	if urlErr != nil {
		return fail("invalid_base_url", 0)
	}
	sr, sb, se := s.doProbeRequest(ctx, a, tlsProfile, proxyURL, key, http.MethodGet, statusURL, upstreamBillingProbeMaxBodyBytes, false)
	quota := 0.0
	if se == nil && sr.StatusCode >= 200 && sr.StatusCode < 300 {
		quota, _ = parseNewAPIQuotaPerUnitResponse(sb)
	}
	data, pe := parseNewAPITokenUsageResponseRaw(tb, quota)
	if pe != nil {
		return fail("newapi_invalid_response", tr.StatusCode)
	}
	return s.saveBalance(ctx, a, now, interval, data, tr.StatusCode)
}
func parseNewAPITokenUsageResponseRaw(body []byte, quota float64) (map[string]any, error) {
	var p map[string]any
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, err
	}
	if code, present := p["code"]; present {
		if ok, valid := code.(bool); !valid || !ok {
			return nil, fmt.Errorf("NewAPI token usage response is not successful")
		}
	}
	d, ok := p["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing data")
	}
	if object, ok := d["object"].(string); !ok || object != "token_usage" {
		return nil, fmt.Errorf("unsupported object")
	}
	u, ok := d["unlimited_quota"].(bool)
	if !ok {
		return nil, fmt.Errorf("missing unlimited_quota")
	}
	g, ge := requiredFiniteBalanceNumber(d, "total_granted")
	used, ue := requiredFiniteBalanceNumber(d, "total_used")
	av, ae := requiredFiniteBalanceNumber(d, "total_available")
	if ge != nil || ue != nil || ae != nil {
		return nil, fmt.Errorf("invalid quota")
	}
	r := map[string]any{"source": "newapi", "raw_unit": "quota", "raw_limit": g, "raw_used": used, "raw_remaining": av, "unlimited": u, "mode": "token_quota"}
	if u {
		r["mode"] = "unlimited"
		return r, nil
	}
	if quota > 0 {
		r["currency"] = "USD"
		r["quota_per_unit"] = quota
		r["limit"] = g / quota
		r["used"] = used / quota
		r["remaining"] = av / quota
	}
	if expiresAt, ok := d["expires_at"].(float64); ok {
		if !isFiniteNonNegative(expiresAt) {
			return nil, fmt.Errorf("invalid expires_at")
		}
		if expiresAt > 0 {
			r["expires_at_unix"] = expiresAt
		}
	}
	return r, nil
}
func (s *UpstreamBillingProbeService) saveBalance(ctx context.Context, a *Account, now time.Time, interval int, data map[string]any, status int) (*UpstreamBalanceProbeSnapshot, error) {
	snap := &UpstreamBalanceProbeSnapshot{Status: UpstreamBillingProbeStatusOK, Data: data, ReceivedAt: &now, FreshUntil: probeTimePtr(now.Add(2 * time.Duration(interval) * time.Minute)), LastAttemptAt: now, NextProbeAt: now.Add(nextProbeDelay(interval, 0)), HTTPStatus: status}
	w, ok := s.accountRepo.(upstreamBalanceSnapshotWriter)
	if !ok {
		return nil, ErrUpstreamBillingProbeUnavailable
	}
	return snap, w.UpdateUpstreamBalanceProbeSnapshot(ctx, a, snap)
}

func parseSub2APIUsageResponse(body []byte) (map[string]any, error) {
	var p map[string]any
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, err
	}
	if valid, present := p["isValid"]; present {
		if ok, typed := valid.(bool); !typed || !ok {
			return nil, fmt.Errorf("Sub2API usage response is not valid")
		}
	}
	mode, _ := p["mode"].(string)
	unit, _ := p["unit"].(string)
	if strings.ToUpper(strings.TrimSpace(unit)) != "USD" {
		return nil, fmt.Errorf("unit")
	}
	r := map[string]any{"source": "sub2api", "currency": "USD", "unlimited": false}
	if mode == "quota_limited" {
		q, ok := p["quota"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("quota")
		}
		for _, k := range []string{"limit", "used", "remaining"} {
			v, e := requiredFiniteBalanceNumber(q, k)
			if e != nil {
				return nil, e
			}
			r[k] = v
		}
		r["mode"] = mode
		return r, nil
	}
	if mode != "unrestricted" {
		return nil, fmt.Errorf("mode")
	}
	v, e := requiredFiniteBalanceNumber(p, "remaining")
	if e != nil {
		return nil, e
	}
	r["remaining"] = v
	if b, ok := p["balance"].(float64); ok {
		if !isFiniteNonNegative(b) {
			return nil, fmt.Errorf("balance")
		}
		r["balance"] = b
	}
	if subscription, ok := p["subscription"].(map[string]any); ok {
		r["mode"] = "subscription"
		r["subscription"] = sanitizeSub2APISubscription(subscription)
	} else {
		r["mode"] = "wallet"
	}
	return r, nil
}
func parseNewAPITokenUsageResponse(body []byte, quota float64) (map[string]any, error) {
	if quota <= 0 || math.IsNaN(quota) || math.IsInf(quota, 0) {
		return nil, fmt.Errorf("quota_per_unit")
	}
	return parseNewAPITokenUsageResponseRaw(body, quota)
}
func parseNewAPIQuotaPerUnitResponse(body []byte) (float64, error) {
	var p map[string]any
	if json.Unmarshal(body, &p) != nil {
		return 0, fmt.Errorf("status")
	}
	if d, ok := p["data"].(map[string]any); ok {
		if v, e := finitePositiveBalanceNumber(d, "quota_per_unit"); e == nil {
			return v, nil
		}
	}
	return finitePositiveBalanceNumber(p, "quota_per_unit")
}
func requiredFiniteBalanceNumber(m map[string]any, k string) (float64, error) {
	v, ok := m[k].(float64)
	if !ok || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0, fmt.Errorf("%s", k)
	}
	return v, nil
}
func finitePositiveBalanceNumber(m map[string]any, k string) (float64, error) {
	v, e := requiredFiniteBalanceNumber(m, k)
	if e != nil || v <= 0 {
		return 0, fmt.Errorf("%s", k)
	}
	return v, nil
}

func isFiniteNonNegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

func sanitizeSub2APISubscription(subscription map[string]any) map[string]any {
	result := make(map[string]any)
	for _, key := range []string{
		"daily_usage_usd", "weekly_usage_usd", "monthly_usage_usd",
		"daily_limit_usd", "weekly_limit_usd", "monthly_limit_usd",
		"weekly_window_start", "expires_at",
	} {
		if value, ok := subscription[key]; ok {
			result[key] = value
		}
	}
	return result
}

func classifyNewAPIBalanceHTTPError(status int) string {
	switch status {
	case http.StatusUnauthorized:
		return "newapi_auth_failed"
	case http.StatusForbidden:
		return "newapi_forbidden"
	case http.StatusTooManyRequests:
		return "newapi_rate_limited"
	default:
		return "newapi_http_error"
	}
}
