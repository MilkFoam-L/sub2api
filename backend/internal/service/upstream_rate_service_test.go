package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type upstreamRateTestEncryptor struct{}

func (upstreamRateTestEncryptor) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}
func (upstreamRateTestEncryptor) Decrypt(ciphertext string) (string, error) { return ciphertext, nil }

type upstreamRateMemoryRepo struct {
	source *UpstreamRateSource
}

func (r *upstreamRateMemoryRepo) CreateSource(ctx context.Context, source *UpstreamRateSource) error {
	source.ID = 1
	source.TokenConfigured = source.TokenEncrypted != ""
	r.source = source
	return nil
}
func (r *upstreamRateMemoryRepo) UpdateSource(ctx context.Context, source *UpstreamRateSource) error {
	r.source = source
	return nil
}
func (r *upstreamRateMemoryRepo) DeleteSource(ctx context.Context, id int64) error { return nil }
func (r *upstreamRateMemoryRepo) GetSource(ctx context.Context, id int64) (*UpstreamRateSource, error) {
	return r.source, nil
}
func (r *upstreamRateMemoryRepo) ListSources(ctx context.Context) ([]*UpstreamRateSource, error) {
	return []*UpstreamRateSource{r.source}, nil
}
func (r *upstreamRateMemoryRepo) ListEnabledSources(ctx context.Context) ([]*UpstreamRateSource, error) {
	return []*UpstreamRateSource{r.source}, nil
}
func (r *upstreamRateMemoryRepo) UpdateSourceSyncStatus(ctx context.Context, id int64, status string, lastSyncAt *time.Time, lastError string) error {
	return nil
}
func (r *upstreamRateMemoryRepo) UpsertSnapshots(ctx context.Context, snapshots []*UpstreamRateSnapshot) error {
	return nil
}
func (r *upstreamRateMemoryRepo) ListLatestSnapshots(ctx context.Context, sourceID int64) ([]*UpstreamRateSnapshot, error) {
	return nil, nil
}
func (r *upstreamRateMemoryRepo) CreateBinding(ctx context.Context, binding *UpstreamRateBinding) error {
	return nil
}
func (r *upstreamRateMemoryRepo) UpdateBinding(ctx context.Context, binding *UpstreamRateBinding) error {
	return nil
}
func (r *upstreamRateMemoryRepo) DeleteBinding(ctx context.Context, id int64) error { return nil }
func (r *upstreamRateMemoryRepo) GetBinding(ctx context.Context, id int64) (*UpstreamRateBinding, error) {
	return nil, nil
}
func (r *upstreamRateMemoryRepo) ListBindings(ctx context.Context) ([]*UpstreamRateBinding, error) {
	return nil, nil
}
func (r *upstreamRateMemoryRepo) InsertHealthCheck(ctx context.Context, check *UpstreamRateHealthCheck) error {
	return nil
}
func (r *upstreamRateMemoryRepo) ComputeHealthRollups(ctx context.Context, window time.Duration) (map[int64]UpstreamRateHealthRollup, error) {
	return nil, nil
}
func (r *upstreamRateMemoryRepo) ListOverview(ctx context.Context, window time.Duration) ([]*UpstreamRateOverviewItem, error) {
	return nil, nil
}
func (r *upstreamRateMemoryRepo) ListAccountSignals(ctx context.Context, now time.Time, staleTTL time.Duration) (map[int64]UpstreamRateSignalSnapshot, error) {
	return nil, nil
}

func TestUpstreamRateServiceCreateSourceEncryptsAndSanitizesToken(t *testing.T) {
	repo := &upstreamRateMemoryRepo{}
	svc := NewUpstreamRateService(repo, upstreamRateTestEncryptor{})

	created, err := svc.CreateSource(context.Background(), UpstreamRateCreateSourceParams{
		Name: "源", SourceType: UpstreamRateSourceTypeSub2API, BaseURL: "https://example.com", AuthMode: UpstreamRateAuthModeBearerToken,
		Token: "secret", RechargeMultiplier: 1, SyncIntervalSeconds: 60, Enabled: true,
	})

	require.NoError(t, err)
	require.True(t, created.TokenConfigured)
	require.Empty(t, created.Token)
	require.Empty(t, created.TokenEncrypted)
	require.Equal(t, "enc:secret", repo.source.TokenEncrypted)
}
