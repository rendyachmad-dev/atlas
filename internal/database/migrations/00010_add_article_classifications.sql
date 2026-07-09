-- +goose Up

CREATE TABLE IF NOT EXISTS article_classifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id      UUID        NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    industry_id     UUID        REFERENCES industries(id) ON DELETE SET NULL,
    industry_name   VARCHAR(255),
    industry_confidence REAL    CHECK (industry_confidence BETWEEN 0 AND 1),
    technology_id   UUID        REFERENCES technologies(id) ON DELETE SET NULL,
    technology_name VARCHAR(255),
    technology_confidence REAL CHECK (technology_confidence BETWEEN 0 AND 1),
    region_id       UUID        REFERENCES regions(id) ON DELETE SET NULL,
    region_name     VARCHAR(255),
    region_confidence REAL    CHECK (region_confidence BETWEEN 0 AND 1),
    stakeholder_id  UUID        REFERENCES stakeholders(id) ON DELETE SET NULL,
    stakeholder_name VARCHAR(255),
    stakeholder_confidence REAL CHECK (stakeholder_confidence BETWEEN 0 AND 1),
    overall_confidence REAL   CHECK (overall_confidence BETWEEN 0 AND 1),
    llm_provider    VARCHAR(50),
    llm_model       VARCHAR(100),
    raw_json        JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_article_classification UNIQUE (article_id)
);

CREATE INDEX IF NOT EXISTS idx_article_classifications_article   ON article_classifications (article_id);
CREATE INDEX IF NOT EXISTS idx_article_classifications_industry  ON article_classifications (industry_id);
CREATE INDEX IF NOT EXISTS idx_article_classifications_technology ON article_classifications (technology_id);
CREATE INDEX IF NOT EXISTS idx_article_classifications_region    ON article_classifications (region_id);
CREATE INDEX IF NOT EXISTS idx_article_classifications_stakeholder ON article_classifications (stakeholder_id);

CREATE TRIGGER trg_article_classifications_updated_at
    BEFORE UPDATE ON article_classifications
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- +goose Down

DROP TRIGGER IF EXISTS trg_article_classifications_updated_at ON article_classifications;
DROP TABLE IF EXISTS article_classifications CASCADE;