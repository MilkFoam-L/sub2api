//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type updateAccountCredsRepoStub struct {
	mockAccountRepoForGemini
	account                      *Account
	updateCalls                  int
	updateCredentialsFieldsCalls int
	lastCredentialFieldPatch     map[string]any
}

func (r *updateAccountCredsRepoStub) GetByID(ctx context.Context, id int64) (*Account, error) {
	return r.account, nil
}

func (r *updateAccountCredsRepoStub) Update(ctx context.Context, account *Account) error {
	r.updateCalls++
	r.account = account
	return nil
}

func (r *updateAccountCredsRepoStub) UpdateCredentialFields(ctx context.Context, id int64, updates map[string]any) error {
	r.updateCredentialsFieldsCalls++
	r.lastCredentialFieldPatch = shallowCopyMap(updates)
	if r.account.Credentials == nil {
		r.account.Credentials = map[string]any{}
	}
	for key, value := range updates {
		r.account.Credentials[key] = value
	}
	return nil
}

func TestUpdateAccount_PreservesSensitiveCredsWhenIncomingOmits(t *testing.T) {
	accountID := int64(202)
	repo := &updateAccountCredsRepoStub{
		account: &Account{
			ID:       accountID,
			Platform: PlatformAnthropic,
			Type:     AccountTypeOAuth,
			Status:   StatusActive,
			Credentials: map[string]any{
				"refresh_token": "rt-existing",
				"access_token":  "at-existing",
				"id_token":      "id-existing",
				"base_url":      "https://old.example.com",
			},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	// 模拟前端编辑：仅修改 base_url，没有传 token（脱敏后前端 spread 拿不到敏感键）
	updated, err := svc.UpdateAccount(context.Background(), accountID, &UpdateAccountInput{
		Credentials: map[string]any{
			"base_url": "https://new.example.com",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, 1, repo.updateCalls)

	// 敏感键应保留
	require.Equal(t, "rt-existing", repo.account.Credentials["refresh_token"])
	require.Equal(t, "at-existing", repo.account.Credentials["access_token"])
	require.Equal(t, "id-existing", repo.account.Credentials["id_token"])
	// 非敏感键被替换
	require.Equal(t, "https://new.example.com", repo.account.Credentials["base_url"])
}

func TestUpdateAccount_ExplicitNewTokenOverwrites(t *testing.T) {
	accountID := int64(203)
	repo := &updateAccountCredsRepoStub{
		account: &Account{
			ID:       accountID,
			Platform: PlatformAnthropic,
			Type:     AccountTypeOAuth,
			Status:   StatusActive,
			Credentials: map[string]any{
				"refresh_token": "rt-old",
				"api_key":       "sk-old",
			},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	updated, err := svc.UpdateAccount(context.Background(), accountID, &UpdateAccountInput{
		Credentials: map[string]any{
			"refresh_token": "rt-new",
			// api_key 没传 → 应保留旧值
		},
	})
	require.NoError(t, err)
	require.NotNil(t, updated)

	require.Equal(t, "rt-new", repo.account.Credentials["refresh_token"])
	require.Equal(t, "sk-old", repo.account.Credentials["api_key"])
}

func TestBuildAccountForCreateNormalizesNewAPIUserID(t *testing.T) {
	account, err := buildAccountForCreate(&CreateAccountInput{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":        "model-key",
			"newapi_user_id": " 123 ",
		},
	}, map[string]any{})

	require.NoError(t, err)
	require.Equal(t, "123", account.Credentials["newapi_user_id"])

	_, err = buildAccountForCreate(&CreateAccountInput{
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"newapi_user_id": "0"},
	}, map[string]any{})
	require.Error(t, err)
}

func TestUpdateAccountNormalizesAndValidatesNewAPIUserID(t *testing.T) {
	accountID := int64(206)
	repo := &updateAccountCredsRepoStub{account: &Account{
		ID: accountID, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive,
		Credentials: map[string]any{"api_key": "model-key", "newapi_access_token": "dashboard-existing"},
	}}
	svc := &adminServiceImpl{accountRepo: repo}

	updated, err := svc.UpdateAccount(context.Background(), accountID, &UpdateAccountInput{Credentials: map[string]any{
		"newapi_access_token": "",
		"newapi_user_id":      " 456 ",
	}})

	require.NoError(t, err)
	require.Equal(t, "456", updated.Credentials["newapi_user_id"])
	require.Equal(t, "dashboard-existing", updated.Credentials["newapi_access_token"])

	_, err = svc.UpdateAccount(context.Background(), accountID, &UpdateAccountInput{Credentials: map[string]any{"newapi_user_id": "-1"}})
	require.Error(t, err)
}

func TestUpdateAccount_EmptyCredentialsSkipsUpdate(t *testing.T) {
	accountID := int64(204)
	repo := &updateAccountCredsRepoStub{
		account: &Account{
			ID:       accountID,
			Platform: PlatformAnthropic,
			Type:     AccountTypeOAuth,
			Status:   StatusActive,
			Credentials: map[string]any{
				"refresh_token": "rt-existing",
			},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	_, err := svc.UpdateAccount(context.Background(), accountID, &UpdateAccountInput{
		Credentials: map[string]any{}, // len == 0 → 闸门跳过
		Name:        "renamed",
	})
	require.NoError(t, err)

	require.Equal(t, "rt-existing", repo.account.Credentials["refresh_token"], "空 credentials 不应触碰已有 token")
	require.Equal(t, "renamed", repo.account.Name)
}

func TestSetOpenAITeam401Retryable_UpdatesOnlyFlagForOpenAIOAuth(t *testing.T) {
	accountID := int64(205)
	repo := &updateAccountCredsRepoStub{
		account: &Account{
			ID:       accountID,
			Platform: PlatformOpenAI,
			Type:     AccountTypeOAuth,
			Status:   StatusActive,
			Credentials: map[string]any{
				"access_token":  "at-existing",
				"refresh_token": "rt-existing",
				"base_url":      "https://api.openai.com",
			},
		},
	}
	svc := &adminServiceImpl{accountRepo: repo}

	updated, err := svc.SetOpenAITeam401Retryable(context.Background(), accountID, true)

	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, 0, repo.updateCalls, "dedicated switch must not use whole-account Update")
	require.Equal(t, 1, repo.updateCredentialsFieldsCalls)
	require.Equal(t, map[string]any{"openai_team_401_retryable": true}, repo.lastCredentialFieldPatch)
	require.Equal(t, "at-existing", repo.account.Credentials["access_token"])
	require.Equal(t, "rt-existing", repo.account.Credentials["refresh_token"])
	require.Equal(t, true, repo.account.Credentials["openai_team_401_retryable"])
}

func TestSetOpenAITeam401Retryable_RejectsOtherAccountTypes(t *testing.T) {
	t.Run("openai_apikey", func(t *testing.T) {
		repo := &updateAccountCredsRepoStub{
			account: &Account{ID: 206, Platform: PlatformOpenAI, Type: AccountTypeAPIKey},
		}
		svc := &adminServiceImpl{accountRepo: repo}

		_, err := svc.SetOpenAITeam401Retryable(context.Background(), 206, true)

		require.Error(t, err)
		require.Equal(t, 0, repo.updateCredentialsFieldsCalls)
	})

	t.Run("non_openai_oauth", func(t *testing.T) {
		repo := &updateAccountCredsRepoStub{
			account: &Account{ID: 207, Platform: PlatformAntigravity, Type: AccountTypeOAuth},
		}
		svc := &adminServiceImpl{accountRepo: repo}

		_, err := svc.SetOpenAITeam401Retryable(context.Background(), 207, true)

		require.Error(t, err)
		require.Equal(t, 0, repo.updateCredentialsFieldsCalls)
	})
}
