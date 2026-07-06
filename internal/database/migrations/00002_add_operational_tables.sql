-- +goose Up

-- ============================================================
-- NEW ENUMS
-- ============================================================

-- +goose StatementBegin
DO $$ BEGIN
  CREATE TYPE analysis_status AS ENUM ('queued', 'processing', 'completed', 'failed', 'retry');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ BEGIN
  CREATE TYPE user_role AS ENUM ('admin', 'analyst', 'viewer');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- ============================================================
-- TABLE: fetch_logs
-- ============================================================

CREATE TABLE IF NOT EXISTS fetch_logs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id         UUID        NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    status            VARCHAR(20) NOT NULL DEFAULT 'success',
    articles_fetched  INTEGER     DEFAULT 0,
    articles_new      INTEGER     DEFAULT 0,
    error_message     TEXT,
    started_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_fetch_logs_source_id  ON fetch_logs (source_id);
CREATE INDEX idx_fetch_logs_status     ON fetch_logs (status);
CREATE INDEX idx_fetch_logs_started_at ON fetch_logs (started_at DESC);

-- ============================================================
-- TABLE: analysis_queue
-- ============================================================

CREATE TABLE IF NOT EXISTS analysis_queue (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id        UUID            NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    status            analysis_status NOT NULL DEFAULT 'queued',
    priority          INTEGER         DEFAULT 0,
    attempts          INTEGER         DEFAULT 0,
    max_attempts      INTEGER         DEFAULT 3,
    last_error        TEXT,
    scheduled_at      TIMESTAMPTZ,
    started_at        TIMESTAMPTZ,
    completed_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ     NOT NULL DEFAULT now(),

    CONSTRAINT uq_analysis_queue_article UNIQUE (article_id)
);

CREATE INDEX idx_analysis_queue_status     ON analysis_queue (status);
CREATE INDEX idx_analysis_queue_priority   ON analysis_queue (priority DESC, created_at ASC);
CREATE INDEX idx_analysis_queue_scheduled  ON analysis_queue (scheduled_at) WHERE status = 'queued';

-- ============================================================
-- TABLE: trend_signals (daily aggregation)
-- ============================================================

CREATE TABLE IF NOT EXISTS trend_signals (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_date           DATE        NOT NULL,
    topic_keywords        TEXT[],
    avg_urgency           DECIMAL(3,1),
    avg_innovation        DECIMAL(3,1),
    avg_risk              DECIMAL(3,1),
    avg_sentiment         DECIMAL(3,1),
    avg_confidence        DECIMAL(3,1),
    total_articles        INTEGER     DEFAULT 0,
    source_diversity      INTEGER     DEFAULT 0,
    language_distribution JSONB,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_trend_signals_date   ON trend_signals (signal_date DESC);
CREATE INDEX idx_trend_signals_topic  ON trend_signals USING GIN (topic_keywords);

-- ============================================================
-- TABLE: trend_article_links (M2M: trends ⇄ articles)
-- ============================================================

CREATE TABLE IF NOT EXISTS trend_article_links (
    trend_id        UUID        NOT NULL REFERENCES immutable_trends(id) ON DELETE CASCADE,
    article_id      UUID        NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    relevance_score REAL        CHECK (relevance_score BETWEEN 0 AND 1),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (trend_id, article_id)
);

CREATE INDEX idx_trend_article_links_article ON trend_article_links (article_id);

-- ============================================================
-- TABLE: opportunity_trends (M2M: opportunities ⇄ trends)
-- ============================================================

CREATE TABLE IF NOT EXISTS opportunity_trends (
    opportunity_id  UUID        NOT NULL REFERENCES opportunities(id) ON DELETE CASCADE,
    trend_id        UUID        NOT NULL REFERENCES immutable_trends(id) ON DELETE CASCADE,
    strength        REAL        CHECK (strength BETWEEN 0 AND 1),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (opportunity_id, trend_id)
);

CREATE INDEX idx_opportunity_trends_trend ON opportunity_trends (trend_id);

-- ============================================================
-- TABLE: tags (hierarchical)
-- ============================================================

CREATE TABLE IF NOT EXISTS tags (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    parent_id   UUID REFERENCES tags(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_tags_name UNIQUE (name)
);

CREATE INDEX idx_tags_parent_id ON tags (parent_id);

-- ============================================================
-- TABLE: article_tags (M2M: articles ⇄ tags)
-- ============================================================

CREATE TABLE IF NOT EXISTS article_tags (
    article_id  UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    tag_id      UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,

    PRIMARY KEY (article_id, tag_id)
);

CREATE INDEX idx_article_tags_tag ON article_tags (tag_id);

-- ============================================================
-- TABLE: opportunity_tags (M2M: opportunities ⇄ tags)
-- ============================================================

CREATE TABLE IF NOT EXISTS opportunity_tags (
    opportunity_id  UUID NOT NULL REFERENCES opportunities(id) ON DELETE CASCADE,
    tag_id          UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,

    PRIMARY KEY (opportunity_id, tag_id)
);

CREATE INDEX idx_opportunity_tags_tag ON opportunity_tags (tag_id);

-- ============================================================
-- TABLE: users
-- ============================================================

CREATE TABLE IF NOT EXISTS users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       VARCHAR(255) NOT NULL,
    name        VARCHAR(255),
    role        user_role    NOT NULL DEFAULT 'viewer',
    preferences JSONB,
    last_login  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_users_email UNIQUE (email)
);

CREATE INDEX idx_users_role ON users (role);

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- ============================================================
-- TABLE: user_bookmarks
-- ============================================================

CREATE TABLE IF NOT EXISTS user_bookmarks (
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    article_id      UUID REFERENCES articles(id) ON DELETE CASCADE,
    opportunity_id  UUID REFERENCES opportunities(id) ON DELETE CASCADE,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_user_bookmark_article     ON user_bookmarks (user_id, article_id) WHERE article_id IS NOT NULL;
CREATE UNIQUE INDEX uq_user_bookmark_opportunity ON user_bookmarks (user_id, opportunity_id) WHERE opportunity_id IS NOT NULL;
CREATE INDEX idx_user_bookmarks_user             ON user_bookmarks (user_id);
CREATE INDEX idx_user_bookmarks_article          ON user_bookmarks (article_id);
CREATE INDEX idx_user_bookmarks_opp              ON user_bookmarks (opportunity_id);

-- ============================================================
-- TABLE: collaboration_comments
-- ============================================================

CREATE TABLE IF NOT EXISTS collaboration_comments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id  UUID        NOT NULL REFERENCES opportunities(id) ON DELETE CASCADE,
    author_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content         TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_collab_comments_opportunity ON collaboration_comments (opportunity_id);
CREATE INDEX idx_collab_comments_author      ON collaboration_comments (author_id);

CREATE TRIGGER trg_collab_comments_updated_at
    BEFORE UPDATE ON collaboration_comments
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- ============================================================
-- TABLE: dashboard_widgets
-- ============================================================

CREATE TABLE IF NOT EXISTS dashboard_widgets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    widget_type VARCHAR(50)  NOT NULL,
    config      JSONB,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_dashboard_widgets_user ON dashboard_widgets (user_id, widget_type);

CREATE TRIGGER trg_dashboard_widgets_updated_at
    BEFORE UPDATE ON dashboard_widgets
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- +goose Down

DROP TABLE IF EXISTS dashboard_widgets CASCADE;
DROP TABLE IF EXISTS collaboration_comments CASCADE;
DROP TABLE IF EXISTS user_bookmarks CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS opportunity_tags CASCADE;
DROP TABLE IF EXISTS article_tags CASCADE;
DROP TABLE IF EXISTS tags CASCADE;
DROP TABLE IF EXISTS opportunity_trends CASCADE;
DROP TABLE IF EXISTS trend_article_links CASCADE;
DROP TABLE IF EXISTS trend_signals CASCADE;
DROP TABLE IF EXISTS analysis_queue CASCADE;
DROP TABLE IF EXISTS fetch_logs CASCADE;

DROP TYPE IF EXISTS user_role CASCADE;
DROP TYPE IF EXISTS analysis_status CASCADE;