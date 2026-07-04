package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpstreamRateCollectorSub2APIMergesUserRates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/groups/available":
			_, _ = w.Write([]byte(`{"data":[{"id":1,"name":"default","rate_multiplier":1.2},{"id":2,"name":"vip","rate_multiplier":0.8,"peak_rate_enabled":true,"peak_rate_multiplier":1.5}]}`))
		case "/api/v1/groups/rates":
			require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
			_, _ = w.Write([]byte(`{"data":{"1":0.6}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	svc := NewUpstreamRateService(nil, nil)
	svc.SetHTTPClient(server.Client())
	snapshots, _, err := svc.collectSub2API(context.Background(), &UpstreamRateSource{
		Name: "上游", SourceType: UpstreamRateSourceTypeSub2API, BaseURL: server.URL,
		AuthMode: UpstreamRateAuthModeBearerToken, Token: "token", RechargeMultiplier: 2, SyncIntervalSeconds: 60,
	})

	require.NoError(t, err)
	require.Len(t, snapshots, 2)
	require.Equal(t, "1", snapshots[0].UpstreamGroupKey)
	require.Equal(t, 0.6, snapshots[0].RawRateMultiplier)
	require.Equal(t, 0.3, snapshots[0].EffectiveRateMultiplier)
	require.Equal(t, "2", snapshots[1].UpstreamGroupKey)
	require.Equal(t, 0.8, snapshots[1].RawRateMultiplier)
	require.True(t, snapshots[1].PeakRateEnabled)
}

func TestUpstreamRateCollectorNewAPIParsesGroupRatio(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/ratio_config", r.URL.Path)
		_, _ = w.Write([]byte(`{"data":{"GroupRatio":{"default":1.4,"vip":"0.7"},"ModelRatio":{"gpt-4o":2},"CompletionRatio":{"gpt-4o":3}}}`))
	}))
	defer server.Close()

	svc := NewUpstreamRateService(nil, nil)
	svc.SetHTTPClient(server.Client())
	snapshots, _, err := svc.collectNewAPI(context.Background(), &UpstreamRateSource{
		Name: "NewAPI", SourceType: UpstreamRateSourceTypeNewAPI, BaseURL: server.URL,
		AuthMode: UpstreamRateAuthModeNone, RechargeMultiplier: 1, SyncIntervalSeconds: 60,
	})

	require.NoError(t, err)
	require.Len(t, snapshots, 2)
	require.Equal(t, "default", snapshots[0].UpstreamGroupKey)
	require.Equal(t, 1.4, snapshots[0].EffectiveRateMultiplier)
	require.Contains(t, snapshots[0].ModelRatioJSON, "gpt-4o")
}

func TestApplyUpstreamRateRuleModesOffsetClamp(t *testing.T) {
	min := 0.5
	max := 1.0
	require.Equal(t, 0.9, applyUpstreamRateRule([]float64{0.8, 1.4}, UpstreamRateRuleAvg, -0.2, &min, &max))
	require.Equal(t, 0.8, applyUpstreamRateRule([]float64{0.8, 1.4}, UpstreamRateRuleMin, 0, nil, nil))
	require.Equal(t, 1.0, applyUpstreamRateRule([]float64{0.8, 1.4}, UpstreamRateRuleMax, 0, nil, &max))
	require.Equal(t, 1.0, applyUpstreamRateRule(nil, UpstreamRateRuleAvg, 0, nil, nil))
}
