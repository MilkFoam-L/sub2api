package service

import "strings"

func isUpstreamInsufficientBalanceError(body []byte) bool {
	switch strings.ToLower(strings.TrimSpace(extractUpstreamErrorCode(body))) {
	case "insufficient_balance", "insufficient_user_quota":
		return true
	default:
		return false
	}
}
