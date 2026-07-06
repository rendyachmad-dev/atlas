-- +goose Up

CREATE TABLE IF NOT EXISTS industries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code         VARCHAR(50)  NOT NULL,
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    parent_id    UUID REFERENCES industries(id) ON DELETE SET NULL,
    is_active    BOOLEAN      DEFAULT true,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_industries_code UNIQUE (code)
);

CREATE INDEX IF NOT EXISTS idx_industries_parent_id  ON industries (parent_id);
CREATE INDEX IF NOT EXISTS idx_industries_is_active  ON industries (is_active);

CREATE TRIGGER trg_industries_updated_at
    BEFORE UPDATE ON industries
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- +goose Down

DROP TRIGGER IF EXISTS trg_industries_updated_at ON industries;
DROP TABLE IF EXISTS industries CASCADE;