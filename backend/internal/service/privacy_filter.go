package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	defaultPrivacyFilterBaseURL   = "http://127.0.0.1:8088"
	defaultPrivacyFilterTimeoutMS = 250
	maxPrivacyFilterTimeoutMS     = 5000
	maxPrivacyFilterResponseBytes = 4 * 1024 * 1024
)

type PrivacyFilterClient interface {
	RedactBatch(ctx context.Context, texts []string) ([]PrivacyFilterTextResult, error)
}

type PrivacyFilterTextResult struct {
	Redacted  string                `json:"redacted"`
	Hit       bool                  `json:"hit"`
	Count     int                   `json:"count"`
	Entities  []PrivacyFilterEntity `json:"entities"`
	ElapsedMS float64               `json:"elapsed_ms"`
}

type PrivacyFilterEntity struct {
	Type string `json:"type"`
}

type PrivacyFilterHTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

type privacyFilterHTTPStatusError struct {
	statusCode int
}

func (e *privacyFilterHTTPStatusError) Error() string {
	return fmt.Sprintf("privacy filter returned status %d", e.statusCode)
}

func NewPrivacyFilterHTTPClient(cfg config.GatewayPrivacyFilterConfig) *PrivacyFilterHTTPClient {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultPrivacyFilterBaseURL
	}
	timeoutMS := cfg.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = defaultPrivacyFilterTimeoutMS
	}
	if timeoutMS > maxPrivacyFilterTimeoutMS {
		timeoutMS = maxPrivacyFilterTimeoutMS
	}
	return &PrivacyFilterHTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutMS) * time.Millisecond,
		},
	}
}

func (c *PrivacyFilterHTTPClient) RedactBatch(ctx context.Context, texts []string) ([]PrivacyFilterTextResult, error) {
	if c == nil {
		return nil, fmt.Errorf("privacy filter client is nil")
	}
	if len(texts) == 0 {
		return nil, nil
	}
	results, err := c.redactBatch(ctx, texts)
	var statusErr *privacyFilterHTTPStatusError
	if err == nil || !errors.As(err, &statusErr) || (statusErr.statusCode != http.StatusNotFound && statusErr.statusCode != http.StatusMethodNotAllowed) {
		return results, err
	}

	fallbackResults := make([]PrivacyFilterTextResult, 0, len(texts))
	for _, text := range texts {
		single, singleErr := c.redactOne(ctx, text)
		if singleErr != nil {
			return nil, singleErr
		}
		fallbackResults = append(fallbackResults, single)
	}
	return fallbackResults, nil
}

func (c *PrivacyFilterHTTPClient) redactBatch(ctx context.Context, texts []string) ([]PrivacyFilterTextResult, error) {
	payload, err := json.Marshal(struct {
		Texts []string `json:"texts"`
	}{Texts: texts})
	if err != nil {
		return nil, fmt.Errorf("marshal privacy filter request: %w", err)
	}
	body, err := c.postJSON(ctx, "/redact/batch", payload)
	if err != nil {
		return nil, err
	}
	results, err := parsePrivacyFilterBatchResponse(body)
	if err != nil {
		return nil, err
	}
	if len(results) != len(texts) {
		return nil, fmt.Errorf("privacy filter returned %d results for %d texts", len(results), len(texts))
	}
	return results, nil
}

func (c *PrivacyFilterHTTPClient) redactOne(ctx context.Context, text string) (PrivacyFilterTextResult, error) {
	payload, err := json.Marshal(struct {
		Text string `json:"text"`
	}{Text: text})
	if err != nil {
		return PrivacyFilterTextResult{}, fmt.Errorf("marshal privacy filter request: %w", err)
	}
	body, err := c.postJSON(ctx, "/redact", payload)
	if err != nil {
		return PrivacyFilterTextResult{}, err
	}
	var result PrivacyFilterTextResult
	if err := json.Unmarshal(body, &result); err != nil {
		return PrivacyFilterTextResult{}, fmt.Errorf("parse privacy filter response: %w", err)
	}
	return result, nil
}

func (c *PrivacyFilterHTTPClient) postJSON(ctx context.Context, path string, payload []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create privacy filter request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call privacy filter: %w", err)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, maxPrivacyFilterResponseBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read privacy filter response: %w", err)
	}
	if len(body) > maxPrivacyFilterResponseBytes {
		return nil, fmt.Errorf("privacy filter response too large")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &privacyFilterHTTPStatusError{statusCode: resp.StatusCode}
	}
	return body, nil
}

func parsePrivacyFilterBatchResponse(body []byte) ([]PrivacyFilterTextResult, error) {
	var results []PrivacyFilterTextResult
	if err := json.Unmarshal(body, &results); err == nil {
		return results, nil
	}
	var wrapped struct {
		Results []PrivacyFilterTextResult `json:"results"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, fmt.Errorf("parse privacy filter response: %w", err)
	}
	return wrapped.Results, nil
}
