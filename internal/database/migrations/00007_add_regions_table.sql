-- +goose Up

CREATE TABLE IF NOT EXISTS regions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code         VARCHAR(50)  NOT NULL,
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    parent_id    UUID REFERENCES regions(id) ON DELETE SET NULL,
    is_active    BOOLEAN      DEFAULT true,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_regions_code UNIQUE (code)
);

CREATE INDEX IF NOT EXISTS idx_regions_parent_id  ON regions (parent_id);
CREATE INDEX IF NOT EXISTS idx_regions_is_active  ON regions (is_active);

CREATE TRIGGER trg_regions_updated_at
    BEFORE UPDATE ON regions
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- +goose Down

DROP TRIGGER IF EXISTS trg_regions_updated_at ON regions;
DROP TABLE IF EXISTS regions CASCADE;