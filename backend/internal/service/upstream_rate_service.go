package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

var (
	ErrUpstreamRateSourceNotFound  = errors.New("upstream rate source not found")
	ErrUpstreamRateBindingNotFound = errors.New("upstream rate binding not found")
)

type UpstreamRateRepository interface {
	CreateSource(ctx context.Context, source *UpstreamRateSource) error
	UpdateSource(ctx context.Context, source *UpstreamRateSource) error
	DeleteSource(ctx context.Context, id int64) error
	GetSource(ctx context.Context, id int64) (*UpstreamRateSource, error)
	ListSources(ctx context.Context) ([]*UpstreamRateSource, error)
	ListEnabledSources(ctx context.Context) ([]*UpstreamRateSource, error)
	UpdateSourceSyncStatus(ctx context.Context, id int64, status string, lastSyncAt *time.Time, lastError string) error

	UpsertSnapshots(ctx context.Context, snapshots []*UpstreamRateSnapshot) error
	ListLatestSnapshots(ctx context.Context, sourceID int64) ([]*UpstreamRateSnapshot, error)

	CreateBinding(ctx context.Context, binding *UpstreamRateBinding) error
	UpdateBinding(ctx context.Context, binding *UpstreamRateBinding) error
	DeleteBinding(ctx context.Context, id int64) error
	GetBinding(ctx context.Context, id int64) (*UpstreamRateBinding, error)
	ListBindings(ctx context.Context) ([]*UpstreamRateBinding, error)

	InsertHealthCheck(ctx context.Context, check *UpstreamRateHealthCheck) error
	ComputeHealthRollups(ctx context.Context, window time.Duration) (map[int64]UpstreamRateHealthRollup, error)
	ListOverview(ctx context.Context, window time.Duration) ([]*UpstreamRateOverviewItem, error)
	ListAccountSignals(ctx context.Context, now time.Time, staleTTL time.Duration) (map[int64]UpstreamRateSignalSnapshot, error)
}

type UpstreamRateScheduler interface {
	Schedule(source *UpstreamRateSource)
	Unschedule(id int64)
}

type UpstreamRateSignalProvider interface {
	AccountSignals(ctx context.Context, now time.Time, staleTTL time.Duration) map[int64]UpstreamRateSignalSnapshot
}

type UpstreamRateService struct {
	repo      UpstreamRateRepository
	encryptor SecretEncryptor
	client    *http.Client
	scheduler UpstreamRateScheduler
	cache     atomic.Value // map[int64]UpstreamRateSignalSnapshot
}

var defaultUpstreamRateSignalProvider atomic.Value // UpstreamRateSignalProvider

func DefaultUpstreamRateSignalProvider() UpstreamRateSignalProvider {
	provider, _ := defaultUpstreamRateSignalProvider.Load().(UpstreamRateSignalProvider)
	return provider
}

func NewUpstreamRateService(repo UpstreamRateRepository, encryptor SecretEncryptor) *UpstreamRateService {
	svc := &UpstreamRateService{
		repo:      repo,
		encryptor: encryptor,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
	defaultUpstreamRateSignalProvider.Store(UpstreamRateSignalProvider(svc))
	return svc
}

func (s *UpstreamRateService) SetScheduler(scheduler UpstreamRateScheduler) {
	s.scheduler = scheduler
}

func (s *UpstreamRateService) SetHTTPClient(client *http.Client) {
	if client != nil {
		s.client = client
	}
}

func (s *UpstreamRateService) CreateSource(ctx context.Context, params UpstreamRateCreateSourceParams) (*UpstreamRateSource, error) {
	source, err := sourceFromCreateParams(params)
	if err != nil {
		return nil, err
	}
	if err := s.encryptSourceToken(source); err != nil {
		return nil, err
	}
	if err := s.repo.CreateSource(ctx, source); err != nil {
		return nil, err
	}
	if s.scheduler != nil {
		s.scheduler.Schedule(source)
	}
	return sanitizeUpstreamRateSource(source), nil
}

func (s *UpstreamRateService) UpdateSource(ctx context.Context, id int64, params UpstreamRateUpdateSourceParams) (*UpstreamRateSource, error) {
	if id <= 0 {
		return nil, fmt.Errorf("source id must be positive")
	}
	source, err := s.repo.GetSource(ctx, id)
	if err != nil {
		return nil, err
	}
	if params.Name != nil {
		source.Name = strings.TrimSpace(*params.Name)
	}
	if params.SourceType != nil {
		source.SourceType = strings.TrimSpace(*params.SourceType)
	}
	if params.BaseURL != nil {
		source.BaseURL = strings.TrimSpace(*params.BaseURL)
	}
	if params.AuthMode != nil {
		source.AuthMode = strings.TrimSpace(*params.AuthMode)
	}
	if params.RechargeMultiplier != nil {
		source.RechargeMultiplier = *params.RechargeMultiplier
	}
	if params.SyncIntervalSeconds != nil {
		source.SyncIntervalSeconds = *params.SyncIntervalSeconds
	}
	if params.Enabled != nil {
		source.Enabled = *params.Enabled
	}
	if params.UseForScheduling != nil {
		source.UseForScheduling = *params.UseForScheduling
	}
	if params.ClearToken {
		source.TokenEncrypted = ""
		source.Token = ""
	} else if params.Token != nil {
		source.Token = strings.TrimSpace(*params.Token)
		if err := s.encryptSourceToken(source); err != nil {
			return nil, err
		}
	}
	if err := validateUpstreamRateSource(source); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateSource(ctx, source); err != nil {
		return nil, err
	}
	if s.scheduler != nil {
		s.scheduler.Schedule(source)
	}
	return sanitizeUpstreamRateSource(source), nil
}

func (s *UpstreamRateService) DeleteSource(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("source id must be positive")
	}
	if err := s.repo.DeleteSource(ctx, id); err != nil {
		return err
	}
	if s.scheduler != nil {
		s.scheduler.Unschedule(id)
	}
	return nil
}

func (s *UpstreamRateService) GetSource(ctx context.Context, id int64) (*UpstreamRateSource, error) {
	source, err := s.repo.GetSource(ctx, id)
	if err != nil {
		return nil, err
	}
	return sanitizeUpstreamRateSource(source), nil
}

func (s *UpstreamRateService) ListSources(ctx context.Context) ([]*UpstreamRateSource, error) {
	sources, err := s.repo.ListSources(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*UpstreamRateSource, 0, len(sources))
	for _, source := range sources {
		out = append(out, sanitizeUpstreamRateSource(source))
	}
	return out, nil
}

func (s *UpstreamRateService) ListEnabledSources(ctx context.Context) ([]*UpstreamRateSource, error) {
	return s.repo.ListEnabledSources(ctx)
}

func (s *UpstreamRateService) CreateBinding(ctx context.Context, params UpstreamRateBindingParams) (*UpstreamRateBinding, error) {
	binding := bindingFromParams(params)
	if err := validateUpstreamRateBinding(binding); err != nil {
		return nil, err
	}
	if err := s.repo.CreateBinding(ctx, binding); err != nil {
		return nil, err
	}
	return binding, nil
}

func (s *UpstreamRateService) UpdateBinding(ctx context.Context, id int64, params UpstreamRateBindingParams) (*UpstreamRateBinding, error) {
	binding := bindingFromParams(params)
	binding.ID = id
	if err := validateUpstreamRateBinding(binding); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateBinding(ctx, binding); err != nil {
		return nil, err
	}
	return binding, nil
}

func (s *UpstreamRateService) DeleteBinding(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("binding id must be positive")
	}
	return s.repo.DeleteBinding(ctx, id)
}

func (s *UpstreamRateService) ListBindings(ctx context.Context) ([]*UpstreamRateBinding, error) {
	return s.repo.ListBindings(ctx)
}

func (s *UpstreamRateService) TestSource(ctx context.Context, id int64) (*UpstreamRateSyncResult, error) {
	source, err := s.loadSourceWithToken(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.collectAndRecord(ctx, source, false)
}

func (s *UpstreamRateService) SyncSource(ctx context.Context, id int64) (*UpstreamRateSyncResult, error) {
	source, err := s.loadSourceWithToken(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.collectAndRecord(ctx, source, true)
}

func (s *UpstreamRateService) SyncLoadedSource(ctx context.Context, source *UpstreamRateSource) (*UpstreamRateSyncResult, error) {
	if source == nil {
		return nil, ErrUpstreamRateSourceNotFound
	}
	if source.Token == "" && source.TokenEncrypted != "" {
		plain, err := s.decryptToken(source.TokenEncrypted)
		if err != nil {
			return nil, err
		}
		source.Token = plain
	}
	return s.collectAndRecord(ctx, source, true)
}

func (s *UpstreamRateService) Overview(ctx context.Context) ([]*UpstreamRateOverviewItem, error) {
	return s.repo.ListOverview(ctx, time.Hour)
}

func (s *UpstreamRateService) LatestSnapshots(ctx context.Context, sourceID int64) ([]*UpstreamRateSnapshot, error) {
	return s.repo.ListLatestSnapshots(ctx, sourceID)
}

func (s *UpstreamRateService) Health(ctx context.Context) (map[int64]UpstreamRateHealthRollup, error) {
	return s.repo.ComputeHealthRollups(ctx, time.Hour)
}

func (s *UpstreamRateService) AccountSignals(ctx context.Context, now time.Time, staleTTL time.Duration) map[int64]UpstreamRateSignalSnapshot {
	if cached, ok := s.cache.Load().(map[int64]UpstreamRateSignalSnapshot); ok && cached != nil {
		return cloneUpstreamRateSignals(cached)
	}
	// 请求热路径只读取内存快照，不在缓存缺失时回库或访问外部上游。
	// 后台同步和手动同步会通过 refreshSignalCache 刷新该缓存。
	return map[int64]UpstreamRateSignalSnapshot{}
}

func (s *UpstreamRateService) refreshSignalCache(ctx context.Context) {
	if s == nil || s.repo == nil {
		return
	}
	signals, err := s.repo.ListAccountSignals(ctx, time.Now(), 10*time.Minute)
	if err == nil {
		s.cache.Store(cloneUpstreamRateSignals(signals))
	}
}

func (s *UpstreamRateService) collectAndRecord(ctx context.Context, source *UpstreamRateSource, persist bool) (*UpstreamRateSyncResult, error) {
	start := time.Now()
	snapshots, httpStatus, err := s.collect(ctx, source)
	latency := int(time.Since(start) / time.Millisecond)
	if latency < 0 {
		latency = 0
	}
	status := UpstreamRateHealthSuccess
	lastErr := ""
	if err != nil {
		status = UpstreamRateHealthFailed
		lastErr = err.Error()
	}
	latencyPtr := latency
	health := &UpstreamRateHealthCheck{
		SourceID:     source.ID,
		CheckType:    "sync",
		Status:       status,
		LatencyMS:    &latencyPtr,
		ErrorMessage: lastErr,
		CheckedAt:    time.Now(),
	}
	if httpStatus > 0 {
		health.HTTPStatus = &httpStatus
	}
	_ = s.repo.InsertHealthCheck(ctx, health)
	now := time.Now()
	if persist {
		if err != nil {
			_ = s.repo.UpdateSourceSyncStatus(ctx, source.ID, status, &now, lastErr)
		} else {
			for _, snapshot := range snapshots {
				snapshot.SourceID = source.ID
				snapshot.Status = UpstreamRateHealthSuccess
				snapshot.LatencyMS = &latencyPtr
				snapshot.FetchedAt = now
				snapshot.ExpiresAt = now.Add(time.Duration(normalizeSyncInterval(source.SyncIntervalSeconds)*2) * time.Second)
			}
			if upsertErr := s.repo.UpsertSnapshots(ctx, snapshots); upsertErr != nil {
				err = upsertErr
				status = UpstreamRateHealthFailed
				lastErr = upsertErr.Error()
			}
			_ = s.repo.UpdateSourceSyncStatus(ctx, source.ID, status, &now, lastErr)
			s.refreshSignalCache(ctx)
		}
	}
	result := &UpstreamRateSyncResult{SourceID: source.ID, Status: status, LatencyMS: latency, SnapshotCount: len(snapshots), Error: lastErr, Snapshots: snapshots}
	return result, err
}

func (s *UpstreamRateService) collect(ctx context.Context, source *UpstreamRateSource) ([]*UpstreamRateSnapshot, int, error) {
	if err := validateUpstreamRateSource(source); err != nil {
		return nil, 0, err
	}
	switch source.SourceType {
	case UpstreamRateSourceTypeSub2API:
		return s.collectSub2API(ctx, source)
	case UpstreamRateSourceTypeNewAPI:
		return s.collectNewAPI(ctx, source)
	default:
		return nil, 0, fmt.Errorf("unsupported upstream source type %q", source.SourceType)
	}
}

func (s *UpstreamRateService) loadSourceWithToken(ctx context.Context, id int64) (*UpstreamRateSource, error) {
	source, err := s.repo.GetSource(ctx, id)
	if err != nil {
		return nil, err
	}
	if source.TokenEncrypted != "" {
		plain, err := s.decryptToken(source.TokenEncrypted)
		if err != nil {
			return nil, err
		}
		source.Token = plain
	}
	return source, nil
}

func (s *UpstreamRateService) encryptSourceToken(source *UpstreamRateSource) error {
	if source == nil || strings.TrimSpace(source.Token) == "" {
		return nil
	}
	if s.encryptor == nil {
		source.TokenEncrypted = source.Token
		return nil
	}
	ciphertext, err := s.encryptor.Encrypt(source.Token)
	if err != nil {
		return err
	}
	source.TokenEncrypted = ciphertext
	return nil
}

func (s *UpstreamRateService) decryptToken(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if s.encryptor == nil {
		return ciphertext, nil
	}
	return s.encryptor.Decrypt(ciphertext)
}

func sourceFromCreateParams(params UpstreamRateCreateSourceParams) (*UpstreamRateSource, error) {
	source := &UpstreamRateSource{
		Name:                strings.TrimSpace(params.Name),
		SourceType:          strings.TrimSpace(params.SourceType),
		BaseURL:             strings.TrimSpace(params.BaseURL),
		AuthMode:            strings.TrimSpace(params.AuthMode),
		Token:               strings.TrimSpace(params.Token),
		RechargeMultiplier:  params.RechargeMultiplier,
		SyncIntervalSeconds: params.SyncIntervalSeconds,
		Enabled:             params.Enabled,
		UseForScheduling:    params.UseForScheduling,
	}
	if source.AuthMode == "" {
		source.AuthMode = UpstreamRateAuthModeBearerToken
	}
	if source.RechargeMultiplier == 0 {
		source.RechargeMultiplier = 1
	}
	if source.SyncIntervalSeconds == 0 {
		source.SyncIntervalSeconds = 300
	}
	return source, validateUpstreamRateSource(source)
}

func bindingFromParams(params UpstreamRateBindingParams) *UpstreamRateBinding {
	mode := strings.TrimSpace(params.Mode)
	if mode == "" {
		mode = UpstreamRateRuleFirst
	}
	return &UpstreamRateBinding{
		SourceID:         params.SourceID,
		UpstreamGroupKey: strings.TrimSpace(params.UpstreamGroupKey),
		TargetType:       strings.TrimSpace(params.TargetType),
		TargetID:         params.TargetID,
		Mode:             mode,
		Offset:           params.Offset,
		ClampMin:         params.ClampMin,
		ClampMax:         params.ClampMax,
		Enabled:          params.Enabled,
	}
}

func validateUpstreamRateSource(source *UpstreamRateSource) error {
	if source == nil {
		return fmt.Errorf("source is required")
	}
	if strings.TrimSpace(source.Name) == "" {
		return fmt.Errorf("source name is required")
	}
	switch strings.TrimSpace(source.SourceType) {
	case UpstreamRateSourceTypeSub2API, UpstreamRateSourceTypeNewAPI:
	default:
		return fmt.Errorf("source_type must be sub2api or newapi")
	}
	parsed, err := url.Parse(strings.TrimSpace(source.BaseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("base_url must be a valid absolute URL")
	}
	switch strings.TrimSpace(source.AuthMode) {
	case UpstreamRateAuthModeNone, UpstreamRateAuthModeBearerToken:
	default:
		return fmt.Errorf("auth_mode must be none or bearer_token")
	}
	if source.RechargeMultiplier < 0 {
		return fmt.Errorf("recharge_multiplier must be non-negative")
	}
	if source.SyncIntervalSeconds < 15 || source.SyncIntervalSeconds > 86400 {
		return fmt.Errorf("sync_interval_seconds must be between 15 and 86400")
	}
	return nil
}

func validateUpstreamRateBinding(binding *UpstreamRateBinding) error {
	if binding == nil {
		return fmt.Errorf("binding is required")
	}
	if binding.SourceID <= 0 {
		return fmt.Errorf("source_id must be positive")
	}
	if strings.TrimSpace(binding.UpstreamGroupKey) == "" {
		return fmt.Errorf("upstream_group_key is required")
	}
	switch strings.TrimSpace(binding.TargetType) {
	case UpstreamRateTargetTypeAccount, UpstreamRateTargetTypeGroup:
	default:
		return fmt.Errorf("target_type must be account or group")
	}
	if binding.TargetID <= 0 {
		return fmt.Errorf("target_id must be positive")
	}
	switch strings.TrimSpace(binding.Mode) {
	case UpstreamRateRuleFirst, UpstreamRateRuleAvg, UpstreamRateRuleMin, UpstreamRateRuleMax:
	default:
		return fmt.Errorf("mode must be first, avg, min, or max")
	}
	if binding.ClampMin != nil && *binding.ClampMin <= 0 {
		return fmt.Errorf("clamp_min must be positive")
	}
	if binding.ClampMax != nil && *binding.ClampMax <= 0 {
		return fmt.Errorf("clamp_max must be positive")
	}
	if binding.ClampMin != nil && binding.ClampMax != nil && *binding.ClampMax < *binding.ClampMin {
		return fmt.Errorf("clamp_max must be >= clamp_min")
	}
	return nil
}

func sanitizeUpstreamRateSource(source *UpstreamRateSource) *UpstreamRateSource {
	if source == nil {
		return nil
	}
	copy := *source
	copy.TokenConfigured = strings.TrimSpace(source.TokenEncrypted) != "" || strings.TrimSpace(source.Token) != ""
	copy.Token = ""
	copy.TokenEncrypted = ""
	return &copy
}

func normalizeSyncInterval(seconds int) int {
	if seconds < 15 {
		return 300
	}
	return seconds
}

func rechargeMultiplierValue(source *UpstreamRateSource) float64 {
	if source == nil || source.RechargeMultiplier <= 0 || math.IsNaN(source.RechargeMultiplier) || math.IsInf(source.RechargeMultiplier, 0) {
		return 1
	}
	return source.RechargeMultiplier
}

func applyUpstreamRateRule(values []float64, mode string, offset float64, clampMin, clampMax *float64) float64 {
	valid := make([]float64, 0, len(values))
	for _, value := range values {
		if value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0) {
			valid = append(valid, value)
		}
	}
	if len(valid) == 0 {
		return 1
	}
	result := valid[0]
	switch mode {
	case UpstreamRateRuleAvg:
		sum := 0.0
		for _, value := range valid {
			sum += value
		}
		result = sum / float64(len(valid))
	case UpstreamRateRuleMin:
		result = valid[0]
		for _, value := range valid[1:] {
			if value < result {
				result = value
			}
		}
	case UpstreamRateRuleMax:
		result = valid[0]
		for _, value := range valid[1:] {
			if value > result {
				result = value
			}
		}
	case UpstreamRateRuleFirst:
		// keep first
	default:
		// keep first
	}
	result += offset
	if clampMin != nil && result < *clampMin {
		result = *clampMin
	}
	if clampMax != nil && result > *clampMax {
		result = *clampMax
	}
	if result <= 0 || math.IsNaN(result) || math.IsInf(result, 0) {
		return 1
	}
	return math.Round(result*10000) / 10000
}

func cloneUpstreamRateSignals(src map[int64]UpstreamRateSignalSnapshot) map[int64]UpstreamRateSignalSnapshot {
	out := make(map[int64]UpstreamRateSignalSnapshot, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func sortedUpstreamSnapshots(snapshots []*UpstreamRateSnapshot) []*UpstreamRateSnapshot {
	out := append([]*UpstreamRateSnapshot(nil), snapshots...)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].UpstreamGroupKey < out[j].UpstreamGroupKey
	})
	return out
}
