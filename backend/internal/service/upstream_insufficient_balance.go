package service

import (
	"fmt"
	"strings"
)

func isUpstreamInsufficientBalanceError(body []byte) bool {
	switch strings.ToLower(strings.TrimSpace(extractUpstreamErrorCode(body))) {
	case "insufficient_balance", "insufficient_user_quota":
		return true
	default:
		return false
	}
}

func buildUpstreamInsufficientBalanceErrorMessage(statusCode int, responseBody []byte) string {
	msg := "Upstream no balance (INSUFFICIENT_BALANCE): upstream account balance is insufficient"
	if upstreamMsg := strings.TrimSpace(sanitizeUpstreamErrorMessage(extractUpstreamErrorMessage(responseBody))); upstreamMsg != "" {
		msg = "Upstream no balance (INSUFFICIENT_BALANCE): " + truncateForLog([]byte(upstreamMsg), 512)
	}
	if statusCode > 0 {
		msg = fmt.Sprintf("%s | status=%d", msg, statusCode)
	}
	return msg
}
