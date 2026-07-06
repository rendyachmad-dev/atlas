-- +goose Up

CREATE TABLE IF NOT EXISTS stakeholders (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code         VARCHAR(50)  NOT NULL,
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    is_active    BOOLEAN      DEFAULT true,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_stakeholders_code UNIQUE (code)
);

CREATE INDEX IF NOT EXISTS idx_stakeholders_is_active ON stakeholders (is_active);

CREATE TRIGGER trg_stakeholders_updated_at
    BEFORE UPDATE ON stakeholders
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- +goose Down

DROP TRIGGER IF EXISTS trg_stakeholders_updated_at ON stakeholders;
DROP TABLE IF EXISTS stakeholders CASCADE;