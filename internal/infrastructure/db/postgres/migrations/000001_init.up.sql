CREATE TABLE IF NOT EXISTS sources (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    kind TEXT NOT NULL,
    description TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS runs (
    id BIGSERIAL PRIMARY KEY,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NULL,
    status TEXT NOT NULL,
    config_hash TEXT NOT NULL,
    config_snapshot_json JSONB NULL,
    error_message TEXT NULL
);

CREATE TABLE IF NOT EXISTS run_sources (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    source_id BIGINT NOT NULL REFERENCES sources(id) ON DELETE RESTRICT,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NULL,
    status TEXT NOT NULL,
    error_message TEXT NULL,
    effective_config_json JSONB NULL,
    CONSTRAINT uq_run_source UNIQUE (run_id, source_id)
);

CREATE TABLE IF NOT EXISTS datasets (
    id BIGSERIAL PRIMARY KEY,
    run_source_id BIGINT NOT NULL REFERENCES run_sources(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    dataset_key TEXT NOT NULL,
    name TEXT NOT NULL,
    location TEXT NOT NULL,
    comment TEXT NULL,
    row_count BIGINT NULL,
    discovered_at TIMESTAMPTZ NOT NULL,
    profile_status TEXT NOT NULL,
    profile_error TEXT NULL,
    metadata_json JSONB NULL,
    CONSTRAINT uq_dataset_key_per_run_source UNIQUE (run_source_id, dataset_key)
);

CREATE TABLE IF NOT EXISTS columns (
    id BIGSERIAL PRIMARY KEY,
    dataset_id BIGINT NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    original_type TEXT NOT NULL,
    normalized_type TEXT NOT NULL,
    is_nullable BOOLEAN NOT NULL,
    comment TEXT NULL,
    ordinal_position INTEGER NOT NULL,
    CONSTRAINT uq_column_name_per_dataset UNIQUE (dataset_id, name)
);

CREATE TABLE IF NOT EXISTS column_stats (
    id BIGSERIAL PRIMARY KEY,
    column_id BIGINT NOT NULL UNIQUE REFERENCES columns(id) ON DELETE CASCADE,
    non_null_count BIGINT NOT NULL,
    null_count BIGINT NOT NULL,
    distinct_count BIGINT NOT NULL,
    min_value_json JSONB NULL,
    max_value_json JSONB NULL
);

CREATE TABLE IF NOT EXISTS column_top_values (
    id BIGSERIAL PRIMARY KEY,
    column_stat_id BIGINT NOT NULL REFERENCES column_stats(id) ON DELETE CASCADE,
    rank INTEGER NOT NULL,
    value_json JSONB NOT NULL,
    occurrence_count BIGINT NOT NULL,
    CONSTRAINT uq_rank_per_column_stat UNIQUE (column_stat_id, rank)
);

CREATE TABLE IF NOT EXISTS sensitive_patterns (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    pattern TEXT NOT NULL,
    description TEXT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_run_sources_run_id ON run_sources(run_id);
CREATE INDEX IF NOT EXISTS idx_run_sources_source_id ON run_sources(source_id);
CREATE INDEX IF NOT EXISTS idx_datasets_run_source_id ON datasets(run_source_id);
CREATE INDEX IF NOT EXISTS idx_columns_dataset_id ON columns(dataset_id);
CREATE INDEX IF NOT EXISTS idx_column_top_values_column_stat_id ON column_top_values(column_stat_id);
