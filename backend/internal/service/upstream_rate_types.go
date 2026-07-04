package service

import "time"

const (
	UpstreamRateSourceTypeSub2API = "sub2api"
	UpstreamRateSourceTypeNewAPI  = "newapi"

	UpstreamRateAuthModeNone        = "none"
	UpstreamRateAuthModeBearerToken = "bearer_token"

	UpstreamRateTargetTypeAccount = "account"
	UpstreamRateTargetTypeGroup   = "group"

	UpstreamRateRuleFirst = "first"
	UpstreamRateRuleAvg   = "avg"
	UpstreamRateRuleMin   = "min"
	UpstreamRateRuleMax   = "max"

	UpstreamRateHealthSuccess = "success"
	UpstreamRateHealthWarning = "warning"
	UpstreamRateHealthFailed  = "failed"
)

type UpstreamRateSource struct {
	ID                  int64
	Name                string
	SourceType          string
	BaseURL             string
	AuthMode            string
	TokenEncrypted      string
	Token               string
	TokenConfigured     bool
	RechargeMultiplier  float64
	SyncIntervalSeconds int
	Enabled             bool
	UseForScheduling    bool
	LastSyncAt          *time.Time
	LastSyncStatus      string
	LastError           string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type UpstreamRateSnapshot struct {
	ID                      int64
	SourceID                int64
	UpstreamGroupKey        string
	UpstreamGroupName       string
	RawRateMultiplier       float64
	EffectiveRateMultiplier float64
	ModelRatioJSON          string
	CompletionRatioJSON     string
	PeakRateEnabled         bool
	PeakRateMultiplier      float64
	Status                  string
	LatencyMS               *int
	FetchedAt               time.Time
	ExpiresAt               time.Time
	ErrorMessage            string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type UpstreamRateBinding struct {
	ID               int64
	SourceID         int64
	SourceName       string
	UpstreamGroupKey string
	TargetType       string
	TargetID         int64
	TargetName       string
	Mode             string
	Offset           float64
	ClampMin         *float64
	ClampMax         *float64
	Enabled          bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type UpstreamRateHealthCheck struct {
	ID           int64
	SourceID     int64
	CheckType    string
	Status       string
	HTTPStatus   *int
	LatencyMS    *int
	ErrorMessage string
	CheckedAt    time.Time
}

type UpstreamRateHealthRollup struct {
	SourceID            int64      `json:"source_id"`
	WindowSeconds       int        `json:"window_seconds"`
	SuccessCount        int        `json:"success_count"`
	FailureCount        int        `json:"failure_count"`
	TotalCount          int        `json:"total_count"`
	SuccessRate         float64    `json:"success_rate"`
	AvgLatencyMS        *int       `json:"avg_latency_ms"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	LastStatus          string     `json:"last_status"`
	LastError           string     `json:"last_error"`
	LastCheckedAt       *time.Time `json:"last_checked_at"`
}

type UpstreamRateOverviewItem struct {
	SourceID             int64
	SourceName           string
	SourceType           string
	BaseURL              string
	Enabled              bool
	UseForScheduling     bool
	TokenConfigured      bool
	LastSyncAt           *time.Time
	LastSyncStatus       string
	LastError            string
	SnapshotCount        int
	HealthSuccessRate1h  float64
	HealthAvgLatencyMS1h *int
	BindingCount         int
}

type UpstreamRateSignalSnapshot struct {
	AccountID               int64
	EffectiveRateMultiplier float64
	SuccessRate             float64
	SnapshotAt              time.Time
	ExpiresAt               time.Time
	SourceCount             int
	Stale                   bool
}

type UpstreamRateCreateSourceParams struct {
	Name                string
	SourceType          string
	BaseURL             string
	AuthMode            string
	Token               string
	RechargeMultiplier  float64
	SyncIntervalSeconds int
	Enabled             bool
	UseForScheduling    bool
}

type UpstreamRateUpdateSourceParams struct {
	Name                *string
	SourceType          *string
	BaseURL             *string
	AuthMode            *string
	Token               *string
	ClearToken          bool
	RechargeMultiplier  *float64
	SyncIntervalSeconds *int
	Enabled             *bool
	UseForScheduling    *bool
}

type UpstreamRateBindingParams struct {
	SourceID         int64
	UpstreamGroupKey string
	TargetType       string
	TargetID         int64
	Mode             string
	Offset           float64
	ClampMin         *float64
	ClampMax         *float64
	Enabled          bool
}

type UpstreamRateSyncResult struct {
	SourceID      int64                   `json:"source_id"`
	Status        string                  `json:"status"`
	LatencyMS     int                     `json:"latency_ms"`
	SnapshotCount int                     `json:"snapshot_count"`
	Error         string                  `json:"error,omitempty"`
	Snapshots     []*UpstreamRateSnapshot `json:"snapshots,omitempty"`
}
