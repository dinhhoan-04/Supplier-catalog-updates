-- Migration: 001_init.sql
-- Description: Create initial schema for Catalog Sync system

CREATE TABLE IF NOT EXISTS products (
    id VARCHAR(64) PRIMARY KEY,
    sku VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    price_cents BIGINT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    supplier_updated_at TIMESTAMPTZ NOT NULL,
    withdrawn_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_products_sku ON products(sku);
CREATE INDEX IF NOT EXISTS idx_products_status ON products(status);

CREATE TABLE IF NOT EXISTS catalog_sync_jobs (
    id VARCHAR(64) PRIMARY KEY,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    scenario VARCHAR(64) NOT NULL DEFAULT 'standard',
    total_batches INT NOT NULL DEFAULT 0,
    processed_batches INT NOT NULL DEFAULT 0,
    total_items INT NOT NULL DEFAULT 0,
    created_count INT NOT NULL DEFAULT 0,
    updated_count INT NOT NULL DEFAULT 0,
    withdrawn_count INT NOT NULL DEFAULT 0,
    skipped_count INT NOT NULL DEFAULT 0,
    error_count INT NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    interrupted_at TIMESTAMPTZ,
    error_summary TEXT
);

CREATE INDEX IF NOT EXISTS idx_jobs_status ON catalog_sync_jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_started_at ON catalog_sync_jobs(started_at DESC);

CREATE TABLE IF NOT EXISTS job_batch_checkpoints (
    id SERIAL PRIMARY KEY,
    job_id VARCHAR(64) NOT NULL REFERENCES catalog_sync_jobs(id) ON DELETE CASCADE,
    batch_index INT NOT NULL,
    items_count INT NOT NULL DEFAULT 0,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status VARCHAR(32) NOT NULL DEFAULT 'COMMITTED',
    CONSTRAINT uq_job_batch UNIQUE (job_id, batch_index)
);

CREATE INDEX IF NOT EXISTS idx_checkpoints_job_id ON job_batch_checkpoints(job_id);

CREATE TABLE IF NOT EXISTS catalog_change_logs (
    id SERIAL PRIMARY KEY,
    job_id VARCHAR(64) NOT NULL REFERENCES catalog_sync_jobs(id) ON DELETE CASCADE,
    batch_index INT NOT NULL,
    sku VARCHAR(64) NOT NULL,
    change_type VARCHAR(32) NOT NULL, -- CREATED, UPDATED, WITHDRAWN, SKIPPED_OUTDATED, SKIPPED_IDENTICAL
    diff_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_change_logs_job_id ON catalog_change_logs(job_id);
CREATE INDEX IF NOT EXISTS idx_change_logs_sku ON catalog_change_logs(sku);

CREATE TABLE IF NOT EXISTS catalog_update_issues (
    id SERIAL PRIMARY KEY,
    job_id VARCHAR(64) NOT NULL REFERENCES catalog_sync_jobs(id) ON DELETE CASCADE,
    batch_index INT NOT NULL,
    sku VARCHAR(64),
    error_code VARCHAR(64) NOT NULL,
    message TEXT NOT NULL,
    raw_item JSONB,
    resolved BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_issues_job_id ON catalog_update_issues(job_id);
CREATE INDEX IF NOT EXISTS idx_issues_resolved ON catalog_update_issues(resolved);
