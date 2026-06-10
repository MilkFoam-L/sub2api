package handler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const privacyFilterFailureMessage = "隐私过滤服务暂时不可用，请稍后重试"

type privacyFilterApplyError struct {
	status  int
	code    string
	message string
	err     error
}

func userPrivacyFilterEnabled(apiKey *service.APIKey) bool {
	return apiKey != nil && apiKey.User != nil && apiKey.User.PrivacyFilterEnabled
}

func applyUserPrivacyFilterBody(
	ctx context.Context,
	reqLog *zap.Logger,
	apiKey *service.APIKey,
	protocol string,
	body []byte,
	client service.PrivacyFilterClient,
	failClosed bool,
) ([]byte, *privacyFilterApplyError) {
	if !userPrivacyFilterEnabled(apiKey) {
		return body, nil
	}
	result, err := service.RedactPrivacyFilterBody(ctx, protocol, body, client)
	if err != nil {
		if reqLog != nil {
			reqLog.Warn("privacy_filter.apply_failed", zap.Error(err), zap.String("protocol", protocol), zap.Bool("fail_closed", failClosed))
		}
		if !failClosed {
			return body, nil
		}
		return nil, &privacyFilterApplyError{
			status:  http.StatusServiceUnavailable,
			code:    "privacy_filter_error",
			message: privacyFilterFailureMessage,
			err:     err,
		}
	}
	if result == nil || !result.Changed {
		return body, nil
	}
	if reqLog != nil {
		reqLog.Info("privacy_filter.body_redacted",
			zap.String("protocol", protocol),
			zap.Int("hit_count", result.HitCount),
			zap.Strings("entity_types", result.EntityTypes),
		)
	}
	return result.Body, nil
}

func resetRequestBody(c *gin.Context, body []byte) {
	if c == nil || c.Request == nil {
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	c.Request.ContentLength = int64(len(body))
	c.Request.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
}

func newGatewayPrivacyFilterClient(cfg *config.Config) service.PrivacyFilterClient {
	if cfg == nil {
		return service.NewPrivacyFilterHTTPClient(config.GatewayPrivacyFilterConfig{})
	}
	return service.NewPrivacyFilterHTTPClient(cfg.Gateway.PrivacyFilter)
}

func gatewayPrivacyFilterFailClosed(cfg *config.Config) bool {
	if cfg == nil {
		return true
	}
	privacyCfg := cfg.Gateway.PrivacyFilter
	if privacyCfg.BaseURL == "" && privacyCfg.TimeoutMS == 0 {
		return true
	}
	return privacyCfg.FailClosed
}

func privacyFilterApplyErrorMessage(applyErr *privacyFilterApplyError) string {
	if applyErr == nil {
		return ""
	}
	if applyErr.message != "" {
		return applyErr.message
	}
	if applyErr.err != nil {
		return fmt.Sprintf("privacy filter failed: %v", applyErr.err)
	}
	return privacyFilterFailureMessage
}
