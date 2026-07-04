CREATE TABLE IF NOT EXISTS upstream_rate_sources (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    source_type VARCHAR(20) NOT NULL CHECK (source_type IN ('sub2api', 'newapi')),
    base_url VARCHAR(500) NOT NULL,
    auth_mode VARCHAR(20) NOT NULL DEFAULT 'bearer_token' CHECK (auth_mode IN ('none', 'bearer_token')),
    token_encrypted TEXT NOT NULL DEFAULT '',
    recharge_multiplier DECIMAL(14,6) NOT NULL DEFAULT 1.0,
    sync_interval_seconds INTEGER NOT NULL DEFAULT 300,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    use_for_scheduling BOOLEAN NOT NULL DEFAULT FALSE,
    last_sync_at TIMESTAMPTZ NULL,
    last_sync_status VARCHAR(20) NOT NULL DEFAULT '',
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_upstream_rate_sources_enabled ON upstream_rate_sources(enabled);
CREATE INDEX IF NOT EXISTS idx_upstream_rate_sources_type ON upstream_rate_sources(source_type);

CREATE TABLE IF NOT EXISTS upstream_rate_snapshots (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES upstream_rate_sources(id) ON DELETE CASCADE,
    upstream_group_key VARCHAR(200) NOT NULL,
    upstream_group_name VARCHAR(200) NOT NULL DEFAULT '',
    raw_rate_multiplier DECIMAL(14,6) NOT NULL DEFAULT 1.0,
    effective_rate_multiplier DECIMAL(14,6) NOT NULL DEFAULT 1.0,
    model_ratio_json JSONB NULL,
    completion_ratio_json JSONB NULL,
    peak_rate_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    peak_rate_multiplier DECIMAL(14,6) NOT NULL DEFAULT 1.0,
    status VARCHAR(20) NOT NULL DEFAULT 'success',
    latency_ms INTEGER NULL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_id, upstream_group_key)
);

CREATE INDEX IF NOT EXISTS idx_upstream_rate_snapshots_source ON upstream_rate_snapshots(source_id);
CREATE INDEX IF NOT EXISTS idx_upstream_rate_snapshots_expires_at ON upstream_rate_snapshots(expires_at);

CREATE TABLE IF NOT EXISTS upstream_rate_bindings (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES upstream_rate_sources(id) ON DELETE CASCADE,
    upstream_group_key VARCHAR(200) NOT NULL,
    target_type VARCHAR(20) NOT NULL CHECK (target_type IN ('account', 'group')),
    target_id BIGINT NOT NULL,
    mode VARCHAR(20) NOT NULL DEFAULT 'first' CHECK (mode IN ('first', 'avg', 'min', 'max')),
    offset_value DECIMAL(14,6) NOT NULL DEFAULT 0,
    clamp_min DECIMAL(14,6) NULL,
    clamp_max DECIMAL(14,6) NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_upstream_rate_bindings_source ON upstream_rate_bindings(source_id);
CREATE INDEX IF NOT EXISTS idx_upstream_rate_bindings_target ON upstream_rate_bindings(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_upstream_rate_bindings_enabled ON upstream_rate_bindings(enabled);

CREATE TABLE IF NOT EXISTS upstream_rate_health_checks (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES upstream_rate_sources(id) ON DELETE CASCADE,
    check_type VARCHAR(40) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('success', 'warning', 'failed')),
    http_status INTEGER NULL,
    latency_ms INTEGER NULL,
    error_message TEXT NOT NULL DEFAULT '',
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_upstream_rate_health_source_time ON upstream_rate_health_checks(source_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_upstream_rate_health_window ON upstream_rate_health_checks(checked_at DESC);
