package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestGatewayPrivacyFilterFailClosedDefaultsToTrueForZeroConfig(t *testing.T) {
	require.True(t, gatewayPrivacyFilterFailClosed(nil))
	require.True(t, gatewayPrivacyFilterFailClosed(&config.Config{}))
}

func TestGatewayPrivacyFilterFailClosedUsesExplicitConfig(t *testing.T) {
	require.False(t, gatewayPrivacyFilterFailClosed(&config.Config{
		Gateway: config.GatewayConfig{
			PrivacyFilter: config.GatewayPrivacyFilterConfig{
				BaseURL:   "http://privacy-filter.local",
				TimeoutMS: 250,
			},
		},
	}))
	require.True(t, gatewayPrivacyFilterFailClosed(&config.Config{
		Gateway: config.GatewayConfig{
			PrivacyFilter: config.GatewayPrivacyFilterConfig{
				BaseURL:    "http://privacy-filter.local",
				TimeoutMS:  250,
				FailClosed: true,
			},
		},
	}))
}
