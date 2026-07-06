-- +goose Up

ALTER TABLE immutable_trends RENAME TO trends;

ALTER INDEX immutable_trends_pkey              RENAME TO trends_pkey;
ALTER INDEX idx_immutable_trends_confidence    RENAME TO idx_trends_confidence;
ALTER INDEX idx_immutable_trends_seen          RENAME TO idx_trends_seen;
ALTER INDEX idx_immutable_trends_status        RENAME TO idx_trends_status;

-- +goose Down

ALTER TABLE trends RENAME TO immutable_trends;

ALTER INDEX trends_pkey              RENAME TO immutable_trends_pkey;
ALTER INDEX idx_trends_confidence    RENAME TO idx_immutable_trends_confidence;
ALTER INDEX idx_trends_seen          RENAME TO idx_immutable_trends_seen;
ALTER INDEX idx_trends_status        RENAME TO idx_immutable_trends_status;