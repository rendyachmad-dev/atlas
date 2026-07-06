-- rss-atlas: Database Schema
-- PostgreSQL 16+

-- ============================================================
-- ENUMS
-- ============================================================

DO $$ BEGIN
  CREATE TYPE source_type AS ENUM ('RSS', 'API', 'SCRAPER', 'MANUAL');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
  CREATE TYPE article_status AS ENUM ('new', 'analyzed', 'archived', 'error');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
  CREATE TYPE trend_status AS ENUM ('active', 'declining', 'emerging', 'archived');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ============================================================
-- TABLE: sources
-- ============================================================

CREATE TABLE IF NOT EXISTS sources (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    type            source_type NOT NULL,
    category        VARCHAR(100),
    url             TEXT NOT NULL,
    language        VARCHAR(10)  DEFAULT 'en',
    country         VARCHAR(10)  DEFAULT 'US',
    is_active       BOOLEAN      DEFAULT true,
    fetch_interval  INTEGER      DEFAULT 30,       -- minutes
    last_fetch      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_sources_type       ON sources (type);
CREATE INDEX idx_sources_category   ON sources (category);
CREATE INDEX idx_sources_is_active  ON sources (is_active);

-- ============================================================
-- TABLE: articles
-- ============================================================

CREATE TABLE IF NOT EXISTS articles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id       UUID           NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    external_id     VARCHAR(512),
    title           TEXT           NOT NULL,
    url             TEXT           NOT NULL,
    author          VARCHAR(255),
    published_at    TIMESTAMPTZ,
    content         TEXT,
    content_hash    VARCHAR(64),                  -- SHA-256 for dedup
    language        VARCHAR(10)   DEFAULT 'en',
    status          article_status DEFAULT 'new',
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),

    CONSTRAINT uq_articles_url UNIQUE (url)
);

CREATE INDEX idx_articles_source_id     ON articles (source_id);
CREATE INDEX idx_articles_status        ON articles (status);
CREATE INDEX idx_articles_published_at  ON articles (published_at DESC);
CREATE INDEX idx_articles_content_hash  ON articles (content_hash);
CREATE INDEX idx_articles_language      ON articles (language);

-- ============================================================
-- TABLE: ai_analysis
-- ============================================================

CREATE TABLE IF NOT EXISTS ai_analysis (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id        UUID        NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    summary_short     TEXT,                         -- 1 sentence
    summary_medium    TEXT,                         -- 1 paragraph
    summary_long      TEXT,                         -- full analysis
    sentiment         VARCHAR(20),                  -- positive / negative / neutral / mixed
    business_impact   TEXT,
    urgency           INTEGER     CHECK (urgency       BETWEEN 1 AND 10),
    confidence        REAL        CHECK (confidence    BETWEEN 0 AND 1),
    time_horizon      VARCHAR(20),                  -- short-term / mid-term / long-term
    risk_score        INTEGER     CHECK (risk_score    BETWEEN 1 AND 10),
    innovation_score  INTEGER     CHECK (innovation_score BETWEEN 1 AND 10),
    market_size_score INTEGER     CHECK (market_size_score BETWEEN 1 AND 10),
    generated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ai_analysis_article_id ON ai_analysis (article_id);
CREATE INDEX idx_ai_analysis_urgency    ON ai_analysis (urgency DESC);
CREATE INDEX idx_ai_analysis_confidence ON ai_analysis (confidence DESC);

-- ============================================================
-- TABLE: opportunities
-- ============================================================

CREATE TABLE IF NOT EXISTS opportunities (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id        UUID        NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    title             TEXT        NOT NULL,
    description       TEXT,
    customer          TEXT,                         -- target customer segment
    problem           TEXT,                         -- problem being solved
    solution          TEXT,                         -- proposed solution
    market_scope      VARCHAR(50),                  -- local / regional / national / global
    estimated_maturity VARCHAR(20),                 -- idea / prototype / early-stage / mature
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_opportunities_article_id ON opportunities (article_id);
CREATE INDEX idx_opportunities_maturity   ON opportunities (estimated_maturity);

-- ============================================================
-- TABLE: immutable_trends
-- ============================================================

CREATE TABLE IF NOT EXISTS immutable_trends (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    confidence      REAL        CHECK (confidence BETWEEN 0 AND 1),
    first_seen      TIMESTAMPTZ NOT NULL,
    last_seen       TIMESTAMPTZ NOT NULL,
    frequency       INTEGER     DEFAULT 1,          -- how many times observed
    status          trend_status DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_immutable_trends_status   ON immutable_trends (status);
CREATE INDEX idx_immutable_trends_seen     ON immutable_trends (last_seen DESC);
CREATE INDEX idx_immutable_trends_confidence ON immutable_trends (confidence DESC);

-- ============================================================
-- TRIGGER: auto-update updated_at
-- ============================================================

CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_sources_updated_at
    BEFORE UPDATE ON sources
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER trg_articles_updated_at
    BEFORE UPDATE ON articles
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();