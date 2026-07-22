//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergePreservingSensitiveCreds_PreservesSensitiveWhenIncomingMissing(t *testing.T) {
	existing := map[string]any{
		"refresh_token": "rt-old",
		"access_token":  "at-old",
		"api_key":       "sk-old",
		"base_url":      "https://old.example.com",
	}
	incoming := map[string]any{
		"base_url":      "https://new.example.com",
		"model_mapping": map[string]any{"foo": "bar"},
	}

	out := MergePreservingSensitiveCreds(existing, incoming)

	require.Equal(t, "rt-old", out["refresh_token"], "incoming 没传 refresh_token，应保留 existing")
	require.Equal(t, "at-old", out["access_token"])
	require.Equal(t, "sk-old", out["api_key"])
	require.Equal(t, "https://new.example.com", out["base_url"], "非敏感键由 incoming 决定")
	require.Equal(t, map[string]any{"foo": "bar"}, out["model_mapping"])
}

func TestMergePreservingSensitiveCreds_PreservesBlankNewAPIAccessToken(t *testing.T) {
	existing := map[string]any{"newapi_access_token": "dashboard-existing", "newapi_user_id": "123"}
	incoming := map[string]any{"newapi_access_token": "", "newapi_user_id": "456"}

	out := MergePreservingSensitiveCreds(existing, incoming)

	require.Equal(t, "dashboard-existing", out["newapi_access_token"])
	require.Equal(t, "456", out["newapi_user_id"])
}

func TestMergePreservingSensitiveCreds_OverwritesWhenIncomingProvidesSensitive(t *testing.T) {
	existing := map[string]any{
		"refresh_token": "rt-old",
		"api_key":       "sk-old",
	}
	incoming := map[string]any{
		"refresh_token": "rt-new",
		// 显式没传 api_key —— 应保留
	}
	out := MergePreservingSensitiveCreds(existing, incoming)
	require.Equal(t, "rt-new", out["refresh_token"], "incoming 显式传入应覆盖")
	require.Equal(t, "sk-old", out["api_key"], "incoming 没传应保留")
}

func TestMergePreservingSensitiveCreds_DoesNotMutateInputs(t *testing.T) {
	existing := map[string]any{"refresh_token": "rt"}
	incoming := map[string]any{"base_url": "x"}

	_ = MergePreservingSensitiveCreds(existing, incoming)

	require.Equal(t, "rt", existing["refresh_token"])
	require.NotContains(t, existing, "base_url")
	require.Equal(t, "x", incoming["base_url"])
	require.NotContains(t, incoming, "refresh_token")
}

func TestMergePreservingSensitiveCreds_NilInputs(t *testing.T) {
	out := MergePreservingSensitiveCreds(nil, map[string]any{"base_url": "x"})
	require.Equal(t, "x", out["base_url"])
	require.NotContains(t, out, "refresh_token")

	out2 := MergePreservingSensitiveCreds(map[string]any{"refresh_token": "rt"}, nil)
	require.Equal(t, "rt", out2["refresh_token"])
}

func TestMergePreservingSensitiveCreds_NonSensitiveDeletionAllowed(t *testing.T) {
	existing := map[string]any{
		"refresh_token": "rt",
		"base_url":      "https://old",
		"project_id":    "p1",
	}
	incoming := map[string]any{
		"base_url": "https://new",
		// 不带 project_id —— 等同删除（非敏感键由 incoming 决定）
	}
	out := MergePreservingSensitiveCreds(existing, incoming)
	require.Equal(t, "rt", out["refresh_token"], "敏感键保留")
	require.Equal(t, "https://new", out["base_url"])
	require.NotContains(t, out, "project_id", "非敏感键 incoming 不传 = 删除")
}

func TestNewAPICredentialsAreClassifiedAndNormalized(t *testing.T) {
	require.True(t, IsSensitiveCredentialKey("newapi_access_token"))
	require.False(t, IsSensitiveCredentialKey("newapi_user_id"))

	credentials := map[string]any{"newapi_user_id": " 00123 "}
	require.NoError(t, NormalizeNewAPICredentials(credentials))
	require.Equal(t, "00123", credentials["newapi_user_id"])
	for _, value := range []any{"", "  "} {
		credentials["newapi_user_id"] = value
		require.NoError(t, NormalizeNewAPICredentials(credentials), "value=%v", value)
		require.Equal(t, "", credentials["newapi_user_id"])
	}
	for _, value := range []any{"0", "-1", "1.2", "abc", 123} {
		credentials["newapi_user_id"] = value
		require.Error(t, NormalizeNewAPICredentials(credentials), "value=%v", value)
	}
}

func TestIsSensitiveCredentialKey(t *testing.T) {
	require.True(t, IsSensitiveCredentialKey("refresh_token"))
	require.True(t, IsSensitiveCredentialKey("api_key"))
	require.True(t, IsSensitiveCredentialKey("private_key"))
	require.False(t, IsSensitiveCredentialKey("base_url"))
	require.False(t, IsSensitiveCredentialKey(""))
	require.False(t, IsSensitiveCredentialKey("model_mapping"))
}
