package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestPrivacyFilterHTTPClientRedactBatchUsesBatchEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/redact/batch", r.URL.Path)
		var req struct {
			Texts []string `json:"texts"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, []string{"email a@b.com", "token sk-test"}, req.Texts)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"redacted":"email [邮箱]","hit":true,"count":1,"entities":[{"type":"email"}],"elapsed_ms":0.1},
			{"redacted":"token [密钥]","hit":true,"count":1,"entities":[{"type":"secret"}],"elapsed_ms":0.2}
		]`))
	}))
	defer server.Close()

	client := NewPrivacyFilterHTTPClient(config.GatewayPrivacyFilterConfig{BaseURL: server.URL, TimeoutMS: 500})
	results, err := client.RedactBatch(t.Context(), []string{"email a@b.com", "token sk-test"})

	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, "email [邮箱]", results[0].Redacted)
	require.True(t, results[0].Hit)
	require.Equal(t, 1, results[0].Count)
	require.Equal(t, "secret", results[1].Entities[0].Type)
}

func TestPrivacyFilterHTTPClientRedactBatchAcceptsWrappedResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/redact/batch", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"redacted":"email [邮箱]","hit":true,"count":1,"entities":[{"type":"email"}],"elapsed_ms":0.1}]}`))
	}))
	defer server.Close()

	client := NewPrivacyFilterHTTPClient(config.GatewayPrivacyFilterConfig{BaseURL: server.URL, TimeoutMS: 500})
	results, err := client.RedactBatch(t.Context(), []string{"email a@b.com"})

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "email [邮箱]", results[0].Redacted)
}

func TestPrivacyFilterHTTPClientRedactBatchFallsBackToSingleEndpointForTexts(t *testing.T) {
	var singleRequests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redact/batch":
			http.NotFound(w, r)
		case "/redact":
			var req struct {
				Text string `json:"text"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			singleRequests = append(singleRequests, req.Text)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"redacted":"[redacted] ` + req.Text + `","hit":true,"count":1,"entities":[{"type":"email"}],"elapsed_ms":0.1}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewPrivacyFilterHTTPClient(config.GatewayPrivacyFilterConfig{BaseURL: server.URL, TimeoutMS: 500})
	results, err := client.RedactBatch(t.Context(), []string{"email a@b.com", "token sk-test"})

	require.NoError(t, err)
	require.Equal(t, []string{"email a@b.com", "token sk-test"}, singleRequests)
	require.Len(t, results, 2)
	require.Equal(t, "[redacted] email a@b.com", results[0].Redacted)
	require.Equal(t, "[redacted] token sk-test", results[1].Redacted)
}

func TestPrivacyFilterHTTPClientRedactBatchReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "downstream unavailable", http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewPrivacyFilterHTTPClient(config.GatewayPrivacyFilterConfig{BaseURL: server.URL, TimeoutMS: 500})
	_, err := client.RedactBatch(t.Context(), []string{"hello"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "status 502")
}

func TestPrivacyFilterHTTPClientRedactBatchHonorsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewPrivacyFilterHTTPClient(config.GatewayPrivacyFilterConfig{BaseURL: server.URL, TimeoutMS: 10})
	_, err := client.RedactBatch(t.Context(), []string{"hello"})

	require.Error(t, err)
}
