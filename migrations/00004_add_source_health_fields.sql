-- +goose Up

ALTER TABLE sources
  ADD COLUMN IF NOT EXISTS last_success_at      TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS last_error_at        TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS last_http_status     INTEGER,
  ADD COLUMN IF NOT EXISTS last_content_type    VARCHAR(255),
  ADD COLUMN IF NOT EXISTS consecutive_failures INTEGER       DEFAULT 0,
  ADD COLUMN IF NOT EXISTS health_status        VARCHAR(20)    DEFAULT 'healthy';

CREATE INDEX IF NOT EXISTS idx_sources_health_status       ON sources (health_status);
CREATE INDEX IF NOT EXISTS idx_sources_consecutive_failures ON sources (consecutive_failures);

-- +goose Down

DROP INDEX IF EXISTS idx_sources_health_status;
DROP INDEX IF EXISTS idx_sources_consecutive_failures;

ALTER TABLE sources
  DROP COLUMN IF EXISTS last_success_at,
  DROP COLUMN IF EXISTS last_error_at,
  DROP COLUMN IF EXISTS last_http_status,
  DROP COLUMN IF EXISTS last_content_type,
  DROP COLUMN IF EXISTS consecutive_failures,
  DROP COLUMN IF EXISTS health_status;