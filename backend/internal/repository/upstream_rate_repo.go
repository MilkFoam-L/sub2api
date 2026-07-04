package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type upstreamRateRepository struct {
	db *sql.DB
}

func NewUpstreamRateRepository(db *sql.DB) service.UpstreamRateRepository {
	return &upstreamRateRepository{db: db}
}

func (r *upstreamRateRepository) CreateSource(ctx context.Context, source *service.UpstreamRateSource) error {
	const q = `
		INSERT INTO upstream_rate_sources (name, source_type, base_url, auth_mode, token_encrypted, recharge_multiplier, sync_interval_seconds, enabled, use_for_scheduling)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, q, source.Name, source.SourceType, source.BaseURL, source.AuthMode, source.TokenEncrypted, source.RechargeMultiplier, source.SyncIntervalSeconds, source.Enabled, source.UseForScheduling).
		Scan(&source.ID, &source.CreatedAt, &source.UpdatedAt)
}

func (r *upstreamRateRepository) UpdateSource(ctx context.Context, source *service.UpstreamRateSource) error {
	const q = `
		UPDATE upstream_rate_sources
		SET name=$2, source_type=$3, base_url=$4, auth_mode=$5, token_encrypted=$6, recharge_multiplier=$7,
		    sync_interval_seconds=$8, enabled=$9, use_for_scheduling=$10, updated_at=NOW()
		WHERE id=$1
		RETURNING updated_at
	`
	err := r.db.QueryRowContext(ctx, q, source.ID, source.Name, source.SourceType, source.BaseURL, source.AuthMode, source.TokenEncrypted, source.RechargeMultiplier, source.SyncIntervalSeconds, source.Enabled, source.UseForScheduling).
		Scan(&source.UpdatedAt)
	return translateUpstreamRateErr(err, service.ErrUpstreamRateSourceNotFound)
}

func (r *upstreamRateRepository) DeleteSource(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM upstream_rate_sources WHERE id=$1`, id)
	if err != nil {
		return err
	}
	return requireRowsAffected(res, service.ErrUpstreamRateSourceNotFound)
}

func (r *upstreamRateRepository) GetSource(ctx context.Context, id int64) (*service.UpstreamRateSource, error) {
	const q = `
		SELECT id, name, source_type, base_url, auth_mode, token_encrypted, recharge_multiplier, sync_interval_seconds,
		       enabled, use_for_scheduling, last_sync_at, last_sync_status, last_error, created_at, updated_at
		FROM upstream_rate_sources WHERE id=$1
	`
	row := r.db.QueryRowContext(ctx, q, id)
	return scanUpstreamRateSource(row)
}

func (r *upstreamRateRepository) ListSources(ctx context.Context) ([]*service.UpstreamRateSource, error) {
	const q = `
		SELECT id, name, source_type, base_url, auth_mode, token_encrypted, recharge_multiplier, sync_interval_seconds,
		       enabled, use_for_scheduling, last_sync_at, last_sync_status, last_error, created_at, updated_at
		FROM upstream_rate_sources ORDER BY id DESC
	`
	return r.listSources(ctx, q)
}

func (r *upstreamRateRepository) ListEnabledSources(ctx context.Context) ([]*service.UpstreamRateSource, error) {
	const q = `
		SELECT id, name, source_type, base_url, auth_mode, token_encrypted, recharge_multiplier, sync_interval_seconds,
		       enabled, use_for_scheduling, last_sync_at, last_sync_status, last_error, created_at, updated_at
		FROM upstream_rate_sources WHERE enabled = TRUE ORDER BY id DESC
	`
	return r.listSources(ctx, q)
}

func (r *upstreamRateRepository) listSources(ctx context.Context, q string) ([]*service.UpstreamRateSource, error) {
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]*service.UpstreamRateSource, 0)
	for rows.Next() {
		source, err := scanUpstreamRateSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, source)
	}
	return out, rows.Err()
}

func scanUpstreamRateSource(scanner interface{ Scan(...any) error }) (*service.UpstreamRateSource, error) {
	source := &service.UpstreamRateSource{}
	var lastSync sql.NullTime
	err := scanner.Scan(&source.ID, &source.Name, &source.SourceType, &source.BaseURL, &source.AuthMode, &source.TokenEncrypted, &source.RechargeMultiplier, &source.SyncIntervalSeconds, &source.Enabled, &source.UseForScheduling, &lastSync, &source.LastSyncStatus, &source.LastError, &source.CreatedAt, &source.UpdatedAt)
	if err != nil {
		return nil, translateUpstreamRateErr(err, service.ErrUpstreamRateSourceNotFound)
	}
	if lastSync.Valid {
		source.LastSyncAt = &lastSync.Time
	}
	source.TokenConfigured = strings.TrimSpace(source.TokenEncrypted) != ""
	return source, nil
}

func (r *upstreamRateRepository) UpdateSourceSyncStatus(ctx context.Context, id int64, status string, lastSyncAt *time.Time, lastError string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE upstream_rate_sources SET last_sync_at=$2, last_sync_status=$3, last_error=$4, updated_at=NOW() WHERE id=$1`, id, lastSyncAt, status, lastError)
	return err
}

func (r *upstreamRateRepository) UpsertSnapshots(ctx context.Context, snapshots []*service.UpstreamRateSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	const q = `
		INSERT INTO upstream_rate_snapshots (
		    source_id, upstream_group_key, upstream_group_name, raw_rate_multiplier, effective_rate_multiplier,
		    model_ratio_json, completion_ratio_json, peak_rate_enabled, peak_rate_multiplier, status, latency_ms,
		    fetched_at, expires_at, error_message
		) VALUES ($1,$2,$3,$4,$5,NULLIF($6,'')::jsonb,NULLIF($7,'')::jsonb,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (source_id, upstream_group_key) DO UPDATE SET
		    upstream_group_name=EXCLUDED.upstream_group_name,
		    raw_rate_multiplier=EXCLUDED.raw_rate_multiplier,
		    effective_rate_multiplier=EXCLUDED.effective_rate_multiplier,
		    model_ratio_json=EXCLUDED.model_ratio_json,
		    completion_ratio_json=EXCLUDED.completion_ratio_json,
		    peak_rate_enabled=EXCLUDED.peak_rate_enabled,
		    peak_rate_multiplier=EXCLUDED.peak_rate_multiplier,
		    status=EXCLUDED.status,
		    latency_ms=EXCLUDED.latency_ms,
		    fetched_at=EXCLUDED.fetched_at,
		    expires_at=EXCLUDED.expires_at,
		    error_message=EXCLUDED.error_message,
		    updated_at=NOW()
	`
	for _, snapshot := range snapshots {
		if snapshot == nil {
			continue
		}
		_, err := tx.ExecContext(ctx, q, snapshot.SourceID, snapshot.UpstreamGroupKey, snapshot.UpstreamGroupName, snapshot.RawRateMultiplier, snapshot.EffectiveRateMultiplier, snapshot.ModelRatioJSON, snapshot.CompletionRatioJSON, snapshot.PeakRateEnabled, snapshot.PeakRateMultiplier, snapshot.Status, snapshot.LatencyMS, snapshot.FetchedAt, snapshot.ExpiresAt, snapshot.ErrorMessage)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *upstreamRateRepository) ListLatestSnapshots(ctx context.Context, sourceID int64) ([]*service.UpstreamRateSnapshot, error) {
	const q = `
		SELECT id, source_id, upstream_group_key, upstream_group_name, raw_rate_multiplier, effective_rate_multiplier,
		       COALESCE(model_ratio_json::text,''), COALESCE(completion_ratio_json::text,''), peak_rate_enabled, peak_rate_multiplier,
		       status, latency_ms, fetched_at, expires_at, error_message, created_at, updated_at
		FROM upstream_rate_snapshots
		WHERE source_id=$1
		ORDER BY upstream_group_key ASC
	`
	rows, err := r.db.QueryContext(ctx, q, sourceID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanSnapshotRows(rows)
}

func scanSnapshotRows(rows *sql.Rows) ([]*service.UpstreamRateSnapshot, error) {
	out := make([]*service.UpstreamRateSnapshot, 0)
	for rows.Next() {
		snapshot := &service.UpstreamRateSnapshot{}
		var latency sql.NullInt64
		if err := rows.Scan(&snapshot.ID, &snapshot.SourceID, &snapshot.UpstreamGroupKey, &snapshot.UpstreamGroupName, &snapshot.RawRateMultiplier, &snapshot.EffectiveRateMultiplier, &snapshot.ModelRatioJSON, &snapshot.CompletionRatioJSON, &snapshot.PeakRateEnabled, &snapshot.PeakRateMultiplier, &snapshot.Status, &latency, &snapshot.FetchedAt, &snapshot.ExpiresAt, &snapshot.ErrorMessage, &snapshot.CreatedAt, &snapshot.UpdatedAt); err != nil {
			return nil, err
		}
		if latency.Valid {
			v := int(latency.Int64)
			snapshot.LatencyMS = &v
		}
		out = append(out, snapshot)
	}
	return out, rows.Err()
}

func (r *upstreamRateRepository) CreateBinding(ctx context.Context, binding *service.UpstreamRateBinding) error {
	const q = `
		INSERT INTO upstream_rate_bindings (source_id, upstream_group_key, target_type, target_id, mode, offset_value, clamp_min, clamp_max, enabled)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, q, binding.SourceID, binding.UpstreamGroupKey, binding.TargetType, binding.TargetID, binding.Mode, binding.Offset, binding.ClampMin, binding.ClampMax, binding.Enabled).
		Scan(&binding.ID, &binding.CreatedAt, &binding.UpdatedAt)
}

func (r *upstreamRateRepository) UpdateBinding(ctx context.Context, binding *service.UpstreamRateBinding) error {
	const q = `
		UPDATE upstream_rate_bindings SET source_id=$2, upstream_group_key=$3, target_type=$4, target_id=$5, mode=$6,
		    offset_value=$7, clamp_min=$8, clamp_max=$9, enabled=$10, updated_at=NOW()
		WHERE id=$1 RETURNING updated_at
	`
	err := r.db.QueryRowContext(ctx, q, binding.ID, binding.SourceID, binding.UpstreamGroupKey, binding.TargetType, binding.TargetID, binding.Mode, binding.Offset, binding.ClampMin, binding.ClampMax, binding.Enabled).
		Scan(&binding.UpdatedAt)
	return translateUpstreamRateErr(err, service.ErrUpstreamRateBindingNotFound)
}

func (r *upstreamRateRepository) DeleteBinding(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM upstream_rate_bindings WHERE id=$1`, id)
	if err != nil {
		return err
	}
	return requireRowsAffected(res, service.ErrUpstreamRateBindingNotFound)
}

func (r *upstreamRateRepository) GetBinding(ctx context.Context, id int64) (*service.UpstreamRateBinding, error) {
	const q = `
		SELECT b.id, b.source_id, s.name, b.upstream_group_key, b.target_type, b.target_id,
		       COALESCE(a.name, g.name, ''), b.mode, b.offset_value, b.clamp_min, b.clamp_max, b.enabled, b.created_at, b.updated_at
		FROM upstream_rate_bindings b
		JOIN upstream_rate_sources s ON s.id=b.source_id
		LEFT JOIN accounts a ON b.target_type='account' AND a.id=b.target_id
		LEFT JOIN groups g ON b.target_type='group' AND g.id=b.target_id
		WHERE b.id=$1
	`
	return scanUpstreamRateBinding(r.db.QueryRowContext(ctx, q, id))
}

func (r *upstreamRateRepository) ListBindings(ctx context.Context) ([]*service.UpstreamRateBinding, error) {
	const q = `
		SELECT b.id, b.source_id, s.name, b.upstream_group_key, b.target_type, b.target_id,
		       COALESCE(a.name, g.name, ''), b.mode, b.offset_value, b.clamp_min, b.clamp_max, b.enabled, b.created_at, b.updated_at
		FROM upstream_rate_bindings b
		JOIN upstream_rate_sources s ON s.id=b.source_id
		LEFT JOIN accounts a ON b.target_type='account' AND a.id=b.target_id
		LEFT JOIN groups g ON b.target_type='group' AND g.id=b.target_id
		ORDER BY b.id DESC
	`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]*service.UpstreamRateBinding, 0)
	for rows.Next() {
		binding, err := scanUpstreamRateBinding(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, binding)
	}
	return out, rows.Err()
}

func scanUpstreamRateBinding(scanner interface{ Scan(...any) error }) (*service.UpstreamRateBinding, error) {
	binding := &service.UpstreamRateBinding{}
	var clampMin, clampMax sql.NullFloat64
	err := scanner.Scan(&binding.ID, &binding.SourceID, &binding.SourceName, &binding.UpstreamGroupKey, &binding.TargetType, &binding.TargetID, &binding.TargetName, &binding.Mode, &binding.Offset, &clampMin, &clampMax, &binding.Enabled, &binding.CreatedAt, &binding.UpdatedAt)
	if err != nil {
		return nil, translateUpstreamRateErr(err, service.ErrUpstreamRateBindingNotFound)
	}
	if clampMin.Valid {
		binding.ClampMin = &clampMin.Float64
	}
	if clampMax.Valid {
		binding.ClampMax = &clampMax.Float64
	}
	return binding, nil
}

func (r *upstreamRateRepository) InsertHealthCheck(ctx context.Context, check *service.UpstreamRateHealthCheck) error {
	if check == nil {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO upstream_rate_health_checks (source_id, check_type, status, http_status, latency_ms, error_message, checked_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, check.SourceID, check.CheckType, check.Status, check.HTTPStatus, check.LatencyMS, check.ErrorMessage, check.CheckedAt)
	return err
}

func (r *upstreamRateRepository) ComputeHealthRollups(ctx context.Context, window time.Duration) (map[int64]service.UpstreamRateHealthRollup, error) {
	seconds := int(window.Seconds())
	if seconds <= 0 {
		seconds = 3600
	}
	const q = `
		SELECT source_id,
		       COUNT(*) FILTER (WHERE status IN ('success','warning')) AS ok,
		       COUNT(*) FILTER (WHERE status = 'failed') AS failed,
		       COUNT(*) AS total,
		       CASE WHEN COUNT(latency_ms) > 0 THEN AVG(latency_ms)::float8 ELSE NULL END AS avg_latency,
		       (ARRAY_AGG(status ORDER BY checked_at DESC))[1] AS last_status,
		       (ARRAY_AGG(error_message ORDER BY checked_at DESC))[1] AS last_error,
		       MAX(checked_at) AS last_checked_at
		FROM upstream_rate_health_checks
		WHERE checked_at >= NOW() - ($1::int || ' seconds')::interval
		GROUP BY source_id
	`
	rows, err := r.db.QueryContext(ctx, q, seconds)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[int64]service.UpstreamRateHealthRollup{}
	for rows.Next() {
		rollup := service.UpstreamRateHealthRollup{WindowSeconds: seconds}
		var avg sql.NullFloat64
		var lastChecked sql.NullTime
		if err := rows.Scan(&rollup.SourceID, &rollup.SuccessCount, &rollup.FailureCount, &rollup.TotalCount, &avg, &rollup.LastStatus, &rollup.LastError, &lastChecked); err != nil {
			return nil, err
		}
		if rollup.TotalCount > 0 {
			rollup.SuccessRate = float64(rollup.SuccessCount) / float64(rollup.TotalCount)
		}
		if avg.Valid {
			v := int(avg.Float64)
			rollup.AvgLatencyMS = &v
		}
		if lastChecked.Valid {
			rollup.LastCheckedAt = &lastChecked.Time
		}
		out[rollup.SourceID] = rollup
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	failures, err := r.consecutiveFailures(ctx)
	if err == nil {
		for sourceID, count := range failures {
			rollup := out[sourceID]
			rollup.ConsecutiveFailures = count
			out[sourceID] = rollup
		}
	}
	return out, nil
}

func (r *upstreamRateRepository) consecutiveFailures(ctx context.Context) (map[int64]int, error) {
	const q = `
		WITH ranked AS (
		  SELECT source_id, status, checked_at,
		         SUM(CASE WHEN status <> 'failed' THEN 1 ELSE 0 END) OVER (PARTITION BY source_id ORDER BY checked_at DESC) AS ok_seen
		  FROM upstream_rate_health_checks
		  WHERE checked_at >= NOW() - INTERVAL '24 hours'
		)
		SELECT source_id, COUNT(*) FROM ranked WHERE ok_seen = 0 AND status = 'failed' GROUP BY source_id
	`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[int64]int{}
	for rows.Next() {
		var sourceID int64
		var count int
		if err := rows.Scan(&sourceID, &count); err != nil {
			return nil, err
		}
		out[sourceID] = count
	}
	return out, rows.Err()
}

func (r *upstreamRateRepository) ListOverview(ctx context.Context, window time.Duration) ([]*service.UpstreamRateOverviewItem, error) {
	rollups, err := r.ComputeHealthRollups(ctx, window)
	if err != nil {
		return nil, err
	}
	const q = `
		SELECT s.id, s.name, s.source_type, s.base_url, s.enabled, s.use_for_scheduling,
		       (s.token_encrypted <> '') AS token_configured, s.last_sync_at, s.last_sync_status, s.last_error,
		       COUNT(DISTINCT snap.id) AS snapshot_count,
		       COUNT(DISTINCT b.id) AS binding_count
		FROM upstream_rate_sources s
		LEFT JOIN upstream_rate_snapshots snap ON snap.source_id=s.id
		LEFT JOIN upstream_rate_bindings b ON b.source_id=s.id
		GROUP BY s.id
		ORDER BY s.id DESC
	`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]*service.UpstreamRateOverviewItem, 0)
	for rows.Next() {
		item := &service.UpstreamRateOverviewItem{}
		var lastSync sql.NullTime
		if err := rows.Scan(&item.SourceID, &item.SourceName, &item.SourceType, &item.BaseURL, &item.Enabled, &item.UseForScheduling, &item.TokenConfigured, &lastSync, &item.LastSyncStatus, &item.LastError, &item.SnapshotCount, &item.BindingCount); err != nil {
			return nil, err
		}
		if lastSync.Valid {
			item.LastSyncAt = &lastSync.Time
		}
		if rollup, ok := rollups[item.SourceID]; ok {
			item.HealthSuccessRate1h = rollup.SuccessRate
			item.HealthAvgLatencyMS1h = rollup.AvgLatencyMS
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *upstreamRateRepository) ListAccountSignals(ctx context.Context, now time.Time, staleTTL time.Duration) (map[int64]service.UpstreamRateSignalSnapshot, error) {
	if staleTTL <= 0 {
		staleTTL = 10 * time.Minute
	}
	const q = `
		WITH direct AS (
		  SELECT b.target_id AS account_id, b.mode, b.offset_value, b.clamp_min, b.clamp_max,
		         snap.effective_rate_multiplier::float8 AS rate, snap.fetched_at, snap.expires_at, s.id AS source_id
		  FROM upstream_rate_bindings b
		  JOIN upstream_rate_sources s ON s.id=b.source_id AND s.enabled AND s.use_for_scheduling
		  JOIN upstream_rate_snapshots snap ON snap.source_id=b.source_id AND snap.upstream_group_key=b.upstream_group_key
		  WHERE b.enabled AND b.target_type='account'
		), group_bound AS (
		  SELECT ag.account_id, b.mode, b.offset_value, b.clamp_min, b.clamp_max,
		         snap.effective_rate_multiplier::float8 AS rate, snap.fetched_at, snap.expires_at, s.id AS source_id
		  FROM upstream_rate_bindings b
		  JOIN upstream_rate_sources s ON s.id=b.source_id AND s.enabled AND s.use_for_scheduling
		  JOIN upstream_rate_snapshots snap ON snap.source_id=b.source_id AND snap.upstream_group_key=b.upstream_group_key
		  JOIN account_groups ag ON b.target_type='group' AND ag.group_id=b.target_id
		  WHERE b.enabled
		), all_rates AS (
		  SELECT * FROM direct
		  UNION ALL
		  SELECT * FROM group_bound
		)
		SELECT account_id, mode, offset_value::float8, clamp_min::float8, clamp_max::float8,
		       ARRAY_AGG(rate ORDER BY source_id) AS rates,
		       MAX(fetched_at) AS snapshot_at,
		       MIN(expires_at) AS expires_at,
		       COUNT(DISTINCT source_id) AS source_count
		FROM all_rates
		GROUP BY account_id, mode, offset_value, clamp_min, clamp_max
	`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[int64]service.UpstreamRateSignalSnapshot{}
	health, _ := r.ComputeHealthRollups(ctx, time.Hour)
	for rows.Next() {
		var accountID int64
		var mode string
		var offset float64
		var clampMin, clampMax sql.NullFloat64
		var rates []float64
		var snapshotAt, expiresAt time.Time
		var sourceCount int
		if err := rows.Scan(&accountID, &mode, &offset, &clampMin, &clampMax, pq.Array(&rates), &snapshotAt, &expiresAt, &sourceCount); err != nil {
			return nil, err
		}
		var minPtr, maxPtr *float64
		if clampMin.Valid {
			minPtr = &clampMin.Float64
		}
		if clampMax.Valid {
			maxPtr = &clampMax.Float64
		}
		rate := serviceApplyUpstreamRateRule(rates, mode, offset, minPtr, maxPtr)
		successRate := 1.0
		if sourceCount > 0 && len(health) > 0 {
			// 当前 SQL 聚合未携带 source_id 明细到最终行；保守使用所有源平均健康度。
			sum := 0.0
			count := 0
			for _, rollup := range health {
				if rollup.TotalCount > 0 {
					sum += rollup.SuccessRate
					count++
				}
			}
			if count > 0 {
				successRate = sum / float64(count)
			}
		}
		stale := now.After(expiresAt) || now.Sub(snapshotAt) > staleTTL
		out[accountID] = service.UpstreamRateSignalSnapshot{AccountID: accountID, EffectiveRateMultiplier: rate, SuccessRate: successRate, SnapshotAt: snapshotAt, ExpiresAt: expiresAt, SourceCount: sourceCount, Stale: stale}
	}
	return out, rows.Err()
}

func serviceApplyUpstreamRateRule(values []float64, mode string, offset float64, clampMin, clampMax *float64) float64 {
	// repository 包不能调用 service 包未导出的 applyUpstreamRateRule，这里保持同样规则。
	valid := make([]float64, 0, len(values))
	for _, value := range values {
		if value > 0 {
			valid = append(valid, value)
		}
	}
	if len(valid) == 0 {
		return 1
	}
	result := valid[0]
	switch mode {
	case service.UpstreamRateRuleAvg:
		sum := 0.0
		for _, value := range valid {
			sum += value
		}
		result = sum / float64(len(valid))
	case service.UpstreamRateRuleMin:
		for _, value := range valid[1:] {
			if value < result {
				result = value
			}
		}
	case service.UpstreamRateRuleMax:
		for _, value := range valid[1:] {
			if value > result {
				result = value
			}
		}
	}
	result += offset
	if clampMin != nil && result < *clampMin {
		result = *clampMin
	}
	if clampMax != nil && result > *clampMax {
		result = *clampMax
	}
	if result <= 0 {
		return 1
	}
	return result
}

func translateUpstreamRateErr(err error, notFound error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return notFound
	}
	return err
}

func requireRowsAffected(res sql.Result, notFound error) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return notFound
	}
	return nil
}
