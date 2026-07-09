-- +goose Up
-- ============================================================
-- Extraction Job Queue + Results (Knowledge Extraction Engine)
-- ============================================================

-- Reuses job_status enum from 00008_add_intelligence_tables.sql

CREATE TABLE IF NOT EXISTS extraction_jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id    UUID        NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    status        job_status  NOT NULL DEFAULT 'queued',
    priority      INTEGER     NOT NULL DEFAULT 0,
    retry_count   INTEGER     NOT NULL DEFAULT 0,
    worker_id     VARCHAR(100),
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    error_message TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_extraction_jobs_article UNIQUE (article_id)
);

CREATE INDEX IF NOT EXISTS idx_extraction_jobs_status   ON extraction_jobs (status);
CREATE INDEX IF NOT EXISTS idx_extraction_jobs_priority ON extraction_jobs (priority DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_extraction_jobs_article  ON extraction_jobs (article_id);

CREATE TABLE IF NOT EXISTS extraction_results (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id    UUID        NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    raw_json      JSONB,
    llm_provider  VARCHAR(50),
    llm_model     VARCHAR(100),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_extraction_results_article UNIQUE (article_id)
);

CREATE INDEX IF NOT EXISTS idx_extraction_results_article ON extraction_results (article_id);

-- +goose Down

DROP TABLE IF EXISTS extraction_results CASCADE;
DROP TABLE IF EXISTS extraction_jobs CASCADE;