-- +goose Up

-- +goose StatementBegin
DO $$ BEGIN
  CREATE TYPE job_status AS ENUM ('queued', 'processing', 'completed', 'failed');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS analysis_jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id    UUID        NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    status        job_status  NOT NULL DEFAULT 'queued',
    priority      INTEGER     NOT NULL DEFAULT 0,
    retry_count   INTEGER     NOT NULL DEFAULT 0,
    worker_id     VARCHAR(100),
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    error_message TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_analysis_jobs_status    ON analysis_jobs (status);
CREATE INDEX IF NOT EXISTS idx_analysis_jobs_priority  ON analysis_jobs (priority DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_analysis_jobs_article   ON analysis_jobs (article_id);

CREATE TABLE IF NOT EXISTS analysis_results (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id      UUID        NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    summary         TEXT,
    business_impact TEXT,
    urgency         INTEGER     CHECK (urgency BETWEEN 1 AND 10),
    confidence      REAL        CHECK (confidence BETWEEN 0 AND 1),
    reasoning       TEXT,
    raw_json        JSONB,
    llm_provider    VARCHAR(50),
    llm_model       VARCHAR(100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_analysis_results_article UNIQUE (article_id)
);

CREATE INDEX IF NOT EXISTS idx_analysis_results_article ON analysis_results (article_id);

-- +goose Down

DROP TABLE IF EXISTS analysis_results CASCADE;
DROP TABLE IF EXISTS analysis_jobs CASCADE;
DROP TYPE IF EXISTS job_status CASCADE;