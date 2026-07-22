package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSub2APIUsageResponseQuotaLimited(t *testing.T) {
	data, err := parseSub2APIUsageResponse([]byte(`{
		"mode":"quota_limited",
		"isValid":true,
		"quota":{"limit":10.5,"used":3.25,"remaining":7.25,"unit":"USD"},
		"remaining":7.25,
		"unit":"USD"
	}`))

	require.NoError(t, err)
	require.Equal(t, "sub2api", data["source"])
	require.Equal(t, "quota_limited", data["mode"])
	require.Equal(t, "USD", data["currency"])
	require.Equal(t, 10.5, data["limit"])
	require.Equal(t, 3.25, data["used"])
	require.Equal(t, 7.25, data["remaining"])
	require.Equal(t, false, data["unlimited"])
}

func TestParseSub2APIUsageResponseWallet(t *testing.T) {
	data, err := parseSub2APIUsageResponse([]byte(`{
		"mode":"unrestricted",
		"planName":"钱包余额",
		"isValid":true,
		"balance":4.5,
		"remaining":4.5,
		"unit":"USD"
	}`))

	require.NoError(t, err)
	require.Equal(t, "wallet", data["mode"])
	require.Equal(t, 4.5, data["remaining"])
	require.Equal(t, 4.5, data["balance"])
	require.Equal(t, "USD", data["currency"])
}

func TestParseSub2APIUsageResponseRejectsUnknownShape(t *testing.T) {
	_, err := parseSub2APIUsageResponse([]byte(`{"mode":"unrestricted","remaining":1}`))

	require.Error(t, err)
	require.Contains(t, err.Error(), "unit")
}

func TestParseNewAPITokenUsageResponseConvertsQuota(t *testing.T) {
	data, err := parseNewAPITokenUsageResponse([]byte(`{
		"code":true,
		"data":{
			"object":"token_usage",
			"total_granted":1000000,
			"total_used":300000,
			"total_available":700000,
			"unlimited_quota":false,
			"expires_at":0
		}
	}`), 500000)

	require.NoError(t, err)
	require.Equal(t, "newapi", data["source"])
	require.Equal(t, "token_quota", data["mode"])
	require.Equal(t, "USD", data["currency"])
	require.Equal(t, float64(2), data["limit"])
	require.Equal(t, float64(0.6), data["used"])
	require.Equal(t, float64(1.4), data["remaining"])
	require.Equal(t, float64(700000), data["raw_remaining"])
	require.Equal(t, float64(500000), data["quota_per_unit"])
	require.Equal(t, false, data["unlimited"])
}

func TestParseNewAPITokenUsageResponsePreservesUnlimitedQuota(t *testing.T) {
	data, err := parseNewAPITokenUsageResponse([]byte(`{
		"code":true,
		"data":{
			"object":"token_usage",
			"total_granted":0,
			"total_used":12,
			"total_available":0,
			"unlimited_quota":true,
			"expires_at":0
		}
	}`), 500000)

	require.NoError(t, err)
	require.Equal(t, true, data["unlimited"])
	require.Equal(t, "unlimited", data["mode"])
	require.NotContains(t, data, "remaining")
}

func TestParseNewAPIStatusQuotaPerUnit(t *testing.T) {
	quotaPerUnit, err := parseNewAPIQuotaPerUnitResponse([]byte(`{
		"success":true,
		"data":{"quota_per_unit":500000}
	}`))

	require.NoError(t, err)
	require.Equal(t, float64(500000), quotaPerUnit)
}

func TestParseNewAPITokenUsageResponseRejectsInvalidQuotaPerUnit(t *testing.T) {
	_, err := parseNewAPITokenUsageResponse([]byte(`{
		"code":true,
		"data":{"object":"token_usage","total_granted":1,"total_used":0,"total_available":1}
	}`), 0)

	require.Error(t, err)
	require.Contains(t, err.Error(), "quota_per_unit")
}

func TestParseSub2APIUsageResponseRejectsDuplicateFields(t *testing.T) {
	_, err := parseSub2APIUsageResponse([]byte(`{"mode":"unrestricted","remaining":1,"remaining":2,"unit":"USD"}`))

	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate")
}

func TestParseNewAPITokenUsageResponseRequiresSuccessCode(t *testing.T) {
	_, err := parseNewAPITokenUsageResponseRaw([]byte(`{"data":{"object":"token_usage","total_granted":1,"total_used":0,"total_available":1,"unlimited_quota":false}}`), 1)

	require.Error(t, err)
	require.Contains(t, err.Error(), "code")
}

func TestParseNewAPITokenUsageResponseRejectsDuplicateFields(t *testing.T) {
	_, err := parseNewAPITokenUsageResponseRaw([]byte(`{"code":true,"data":{"object":"token_usage","total_granted":1,"total_used":0,"total_available":1,"total_available":2,"unlimited_quota":false}}`), 1)

	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate")
}

func TestProbeBalanceFailsClosedWhenNewAPIStatusCannotConvertQuota(t *testing.T) {
	account := &Account{ID: 78, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Credentials: map[string]any{"api_key": "sk-secret", "base_url": "https://upstream.example"}, Extra: map[string]any{UpstreamBalanceProbeEnabledExtraKey: true}}
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{78: account}}
	upstream := &upstreamBillingProbeHTTPStub{responseForURL: func(req *http.Request) (*http.Response, error) {
		status := http.StatusOK
		body := `{"code":true,"data":{"object":"token_usage","total_granted":100,"total_used":25,"total_available":75,"unlimited_quota":false}}`
		if strings.HasSuffix(req.URL.Path, "/v1/usage") {
			status = http.StatusNotFound
			body = `{}`
		} else if strings.HasSuffix(req.URL.Path, "/api/status") {
			status = http.StatusInternalServerError
			body = `{}`
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	}}
	svc := newUpstreamBillingProbeTestService(repo, upstream, &upstreamBillingProbeSettingRepo{})
	snap, err := svc.ProbeBalanceAccount(context.Background(), 78)

	require.NoError(t, err)
	require.Equal(t, UpstreamBillingProbeStatusFailed, snap.Status)
	require.Equal(t, "newapi_quota_unit_unavailable", snap.LastError)
	require.Nil(t, snap.Data)
}

func TestProbeBalanceAllowsNewAPIUnlimitedWithoutStatusConversion(t *testing.T) {
	account := &Account{ID: 79, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Credentials: map[string]any{"api_key": "sk-secret", "base_url": "https://upstream.example"}, Extra: map[string]any{UpstreamBalanceProbeEnabledExtraKey: true}}
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{79: account}}
	upstream := &upstreamBillingProbeHTTPStub{responseForURL: func(req *http.Request) (*http.Response, error) {
		body := `{"code":true,"data":{"object":"token_usage","total_granted":0,"total_used":0,"total_available":0,"unlimited_quota":true}}`
		status := http.StatusOK
		if strings.HasSuffix(req.URL.Path, "/v1/usage") {
			status = http.StatusNotFound
			body = `{}`
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	}}
	svc := newUpstreamBillingProbeTestService(repo, upstream, &upstreamBillingProbeSettingRepo{})
	snap, err := svc.ProbeBalanceAccount(context.Background(), 79)

	require.NoError(t, err)
	require.Equal(t, UpstreamBillingProbeStatusOK, snap.Status)
	require.Equal(t, true, snap.Data["unlimited"])
}

func TestProbeBalanceFallsBackToNewAPIOnlyAfter404(t *testing.T) {
	account := &Account{ID: 77, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Credentials: map[string]any{"api_key": "sk-secret", "base_url": "https://upstream.example"}, Extra: map[string]any{UpstreamBalanceProbeEnabledExtraKey: true}}
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{77: account}}
	upstream := &upstreamBillingProbeHTTPStub{responseForURL: func(req *http.Request) (*http.Response, error) {
		body := `{"mode":"quota_limited","quota":{"limit":2,"used":1,"remaining":1},"unit":"USD"}`
		status := http.StatusOK
		if strings.HasSuffix(req.URL.Path, "/v1/usage") {
			status = http.StatusNotFound
			body = `{}`
		}
		if strings.HasSuffix(req.URL.Path, "/api/usage/token/") {
			body = `{"code":true,"data":{"object":"token_usage","total_granted":100,"total_used":25,"total_available":75,"unlimited_quota":false}}`
		}
		if strings.HasSuffix(req.URL.Path, "/api/status") {
			body = `{"data":{"quota_per_unit":50}}`
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	}}
	svc := newUpstreamBillingProbeTestService(repo, upstream, &upstreamBillingProbeSettingRepo{})
	snap, err := svc.ProbeBalanceAccount(context.Background(), 77)
	require.NoError(t, err)
	require.Equal(t, "newapi", snap.Data["source"])
	require.Equal(t, 1.5, snap.Data["remaining"])
	for _, req := range upstream.requests {
		require.NotContains(t, req.URL.String(), "sk-secret")
	}
}
