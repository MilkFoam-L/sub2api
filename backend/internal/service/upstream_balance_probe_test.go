package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

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

func TestParseNewAPITokenUsageResponseAllowsNegativeAvailable(t *testing.T) {
	data, err := parseNewAPITokenUsageResponse([]byte(`{
		"code":true,
		"data":{
			"object":"token_usage",
			"total_granted":100,
			"total_used":150,
			"total_available":-50,
			"unlimited_quota":false,
			"expires_at":0
		}
	}`), 50)

	require.NoError(t, err)
	require.Equal(t, float64(-50), data["raw_remaining"])
	require.Equal(t, float64(-1), data["remaining"])
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

func TestProbeBalanceRejectsNewAPIUnlimitedWithoutUserToken(t *testing.T) {
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
	require.Equal(t, UpstreamBillingProbeStatusFailed, snap.Status)
	require.Equal(t, "newapi_user_balance_unavailable", snap.LastError)
	require.Nil(t, snap.Data)
}

func TestParseNewAPIUserBalanceResponseAllowsNegativeBalance(t *testing.T) {
	data, err := parseNewAPIUserBalanceResponse([]byte(`{
		"success":true,
		"data":{"quota":-100,"used_quota":300}
	}`), 50)

	require.NoError(t, err)
	require.Equal(t, "newapi_user", data["source"])
	require.Equal(t, "wallet", data["mode"])
	require.Equal(t, float64(-100), data["raw_remaining"])
	require.Equal(t, float64(300), data["raw_used"])
	require.Equal(t, float64(-2), data["remaining"])
	require.Equal(t, float64(6), data["used"])
	require.NotContains(t, data, "raw_limit")
	require.NotContains(t, data, "limit")
	require.NotContains(t, data, "currency")
}

func TestParseNewAPIUserBalanceResponseRejectsInvalidResponses(t *testing.T) {
	for name, body := range map[string]string{
		"success false":    `{"success":false,"data":{"quota":1,"used_quota":0}}`,
		"missing quota":    `{"success":true,"data":{"used_quota":0}}`,
		"wrong quota type": `{"success":true,"data":{"quota":"1","used_quota":0}}`,
		"missing used":     `{"success":true,"data":{"quota":1}}`,
		"duplicate quota":  `{"success":true,"data":{"quota":1,"quota":2,"used_quota":0}}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseNewAPIUserBalanceResponse([]byte(body), 1)
			require.Error(t, err)
		})
	}
}

func TestParseNewAPIStatusQuotaPerUnitRejectsExplicitFailureAndInvalidValues(t *testing.T) {
	for _, body := range []string{
		`{"success":false,"data":{"quota_per_unit":500000}}`,
		`{"success":true,"data":{"quota_per_unit":0}}`,
		`{"success":true,"data":{"quota_per_unit":"500000"}}`,
	} {
		_, err := parseNewAPIQuotaPerUnitResponse([]byte(body))
		require.Error(t, err)
	}
}

func TestParseNewAPIStatusQuotaPerUnitSupportsLegacyShapes(t *testing.T) {
	for _, body := range []string{
		`{"data":{"quota_per_unit":500000}}`,
		`{"quota_per_unit":500000}`,
	} {
		quotaPerUnit, err := parseNewAPIQuotaPerUnitResponse([]byte(body))
		require.NoError(t, err)
		require.Equal(t, float64(500000), quotaPerUnit)
	}
}

func TestProbeBalanceUsesNewAPIUserCredentialsWithOptionalUserHeader(t *testing.T) {
	for _, tc := range []struct {
		name       string
		userID     string
		wantHeader string
	}{
		{name: "with user id", userID: "123", wantHeader: "123"},
		{name: "without user id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			credentials := map[string]any{"api_key": "model-key", "base_url": "https://upstream.example", "newapi_access_token": "dashboard-token"}
			if tc.userID != "" {
				credentials["newapi_user_id"] = tc.userID
			}
			account := &Account{ID: 80, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Credentials: credentials, Extra: map[string]any{UpstreamBalanceProbeEnabledExtraKey: true}}
			repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{80: account}}
			upstream := &upstreamBillingProbeHTTPStub{responseForURL: func(req *http.Request) (*http.Response, error) {
				body := `{"success":true,"data":{"quota":-100,"used_quota":300}}`
				if strings.HasSuffix(req.URL.Path, "/api/status") {
					body = `{"success":true,"data":{"quota_per_unit":50}}`
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
			}}
			svc := newUpstreamBillingProbeTestService(repo, upstream, &upstreamBillingProbeSettingRepo{})

			snap, err := svc.ProbeBalanceAccount(context.Background(), 80)

			require.NoError(t, err)
			require.Equal(t, UpstreamBillingProbeStatusOK, snap.Status)
			require.Equal(t, "newapi_user", snap.Data["source"])
			require.Equal(t, float64(-2), snap.Data["remaining"])
			require.Len(t, upstream.requests, 2)
			for _, req := range upstream.requests {
				require.NotEqual(t, "/v1/usage", req.URL.Path)
				require.NotEqual(t, "/api/usage/token/", req.URL.Path)
				if strings.HasSuffix(req.URL.Path, "/api/user/self") {
					require.Equal(t, "Bearer dashboard-token", req.Header.Get("Authorization"))
					require.Equal(t, tc.wantHeader, req.Header.Get("New-Api-User"))
				} else {
					require.Empty(t, req.Header.Get("Authorization"))
					require.Empty(t, req.Header.Get("New-Api-User"))
				}
			}
		})
	}
}

func TestProbeBalanceNewAPIUserFailuresDoNotFallback(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			account := &Account{ID: 81, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Credentials: map[string]any{"api_key": "model-key", "base_url": "https://upstream.example", "newapi_access_token": "dashboard-token"}, Extra: map[string]any{UpstreamBalanceProbeEnabledExtraKey: true}}
			repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{81: account}}
			upstream := &upstreamBillingProbeHTTPStub{responseForURL: func(req *http.Request) (*http.Response, error) {
				body := `{"success":true,"data":{"quota_per_unit":50}}`
				responseStatus := http.StatusOK
				if strings.HasSuffix(req.URL.Path, "/api/user/self") {
					responseStatus = status
					body = `{"message":"credential rejected"}`
				}
				return &http.Response{StatusCode: responseStatus, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
			}}
			svc := newUpstreamBillingProbeTestService(repo, upstream, &upstreamBillingProbeSettingRepo{})

			snap, err := svc.ProbeBalanceAccount(context.Background(), 81)

			require.NoError(t, err)
			require.Equal(t, UpstreamBillingProbeStatusFailed, snap.Status)
			require.Equal(t, "newapi_user_auth_failed", snap.LastError)
			require.NotContains(t, snap.LastError, "dashboard-token")
			for _, req := range upstream.requests {
				require.NotEqual(t, "/v1/usage", req.URL.Path)
				require.NotEqual(t, "/api/usage/token/", req.URL.Path)
			}
		})
	}
}

func TestProbeBalanceNewAPIUserClassifiesInvalidResponses(t *testing.T) {
	for _, tc := range []struct {
		name       string
		selfBody   string
		statusBody string
		wantError  string
	}{
		{name: "invalid self", selfBody: `{"success":true,"data":{"quota":"bad","used_quota":0}}`, statusBody: `{"success":true,"data":{"quota_per_unit":50}}`, wantError: "newapi_user_invalid_response"},
		{name: "invalid status", selfBody: `{"success":true,"data":{"quota":1,"used_quota":0}}`, statusBody: `{"success":false,"data":{"quota_per_unit":50}}`, wantError: "newapi_user_quota_unit_unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			account := &Account{ID: 82, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Credentials: map[string]any{"api_key": "model-key", "base_url": "https://upstream.example", "newapi_access_token": "dashboard-token"}, Extra: map[string]any{UpstreamBalanceProbeEnabledExtraKey: true}}
			repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{82: account}}
			upstream := &upstreamBillingProbeHTTPStub{responseForURL: func(req *http.Request) (*http.Response, error) {
				body := tc.selfBody
				if strings.HasSuffix(req.URL.Path, "/api/status") {
					body = tc.statusBody
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
			}}
			svc := newUpstreamBillingProbeTestService(repo, upstream, &upstreamBillingProbeSettingRepo{})
			snap, err := svc.ProbeBalanceAccount(context.Background(), 82)
			require.NoError(t, err)
			require.Equal(t, tc.wantError, snap.LastError)
			require.Nil(t, snap.Data)
		})
	}
}

func TestUpstreamProbeCycleDoesNotLetSlowBillingBlockBalanceRefresh(t *testing.T) {
	billingStarted := make(chan struct{})
	balanceStarted := make(chan struct{})
	releaseBilling := make(chan struct{})
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{
		90: {
			ID: 90, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
			Credentials: map[string]any{"api_key": "billing-key", "base_url": "https://billing.example"},
			Extra:       map[string]any{UpstreamBillingProbeEnabledExtraKey: true},
		},
		91: {
			ID: 91, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
			Credentials: map[string]any{"base_url": "https://balance.example", "newapi_access_token": "dashboard-token"},
			Extra:       map[string]any{UpstreamBalanceProbeEnabledExtraKey: true},
		},
	}}
	upstream := &upstreamBillingProbeHTTPStub{responseForURL: func(req *http.Request) (*http.Response, error) {
		body := `{"object":"sub2api.key_billing","schema_version":1,"billing_scope":"token","group_rate_multiplier":1,"resolved_rate_multiplier":1,"peak_rate_enabled":false,"effective_rate_multiplier":1,"observed_at":"2026-07-27T00:00:00Z"}`
		if req.URL.Host == "billing.example" {
			close(billingStarted)
			<-releaseBilling
		} else if strings.HasSuffix(req.URL.Path, "/api/user/self") {
			close(balanceStarted)
			body = `{"success":true,"data":{"quota":100,"used_quota":50}}`
		} else if strings.HasSuffix(req.URL.Path, "/api/status") {
			body = `{"success":true,"data":{"quota_per_unit":50}}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	}}
	svc := newUpstreamBillingProbeTestService(repo, upstream, &upstreamBillingProbeSettingRepo{})
	cycleDone := make(chan struct{})
	go func() {
		svc.runCycle(context.Background())
		close(cycleDone)
	}()

	select {
	case <-billingStarted:
	case <-time.After(time.Second):
		t.Fatal("billing probe did not start")
	}
	select {
	case <-balanceStarted:
	case <-time.After(time.Second):
		t.Fatal("balance refresh was blocked by the slow billing probe")
	}
	close(releaseBilling)
	select {
	case <-cycleDone:
	case <-time.After(2 * time.Second):
		t.Fatal("probe cycle did not finish")
	}

	snapshot := decodeUpstreamBalanceProbeSnapshot(repo.accounts[91].Extra)
	require.NotNil(t, snapshot)
	require.Equal(t, UpstreamBillingProbeStatusOK, snapshot.Status)
	require.Equal(t, float64(2), snapshot.Data["remaining"])
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
			body = `{"success":true,"data":{"quota_per_unit":50}}`
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
