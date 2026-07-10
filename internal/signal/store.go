package signal

import (
	"context"
	"fmt"
	"time"

	"atlas/internal/extract"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStore persists signals to PostgreSQL.
type PGStore struct {
	pool *pgxpool.Pool
}

// NewPGStore creates a new PGStore.
func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

// InsertSignals inserts signals idempotently using ON CONFLICT DO NOTHING.
// Returns the count of newly inserted signals.
func (s *PGStore) InsertSignals(ctx context.Context, signals []Signal) (int, error) {
	if len(signals) == 0 {
		return 0, nil
	}

	// Resolve signal type IDs.
	typeIDCache := make(map[SignalTypeCode]string)
	for _, sig := range signals {
		if _, ok := typeIDCache[sig.SignalTypeCode]; ok {
			continue
		}
		id, err := s.GetSignalTypeID(ctx, sig.SignalTypeCode)
		if err != nil {
			return 0, fmt.Errorf("resolve signal type %s: %w", sig.SignalTypeCode, err)
		}
		typeIDCache[sig.SignalTypeCode] = id
	}

	inserted := 0
	for _, sig := range signals {
		signalTypeID := typeIDCache[sig.SignalTypeCode]
		tag, err := s.pool.Exec(ctx, `
			INSERT INTO signals (article_id, signal_type_id, entity_code, entity_category, confidence, detected_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (article_id, signal_type_id, entity_code) DO NOTHING
		`, sig.ArticleID, signalTypeID, sig.EntityCode, sig.EntityCategory, sig.Confidence, sig.DetectedAt)
		if err != nil {
			return inserted, fmt.Errorf("insert signal: %w", err)
		}
		inserted += int(tag.RowsAffected())
	}

	return inserted, nil
}

// GetSignalsByArticle returns all signals for a given article.
func (s *PGStore) GetSignalsByArticle(ctx context.Context, articleID string) ([]Signal, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT sig.id, sig.article_id, sig.signal_type_id, st.code,
		       sig.entity_code, sig.entity_category, sig.confidence,
		       sig.detected_at, sig.created_at
		FROM signals sig
		JOIN signal_types st ON st.id = sig.signal_type_id
		WHERE sig.article_id = $1
		ORDER BY sig.confidence DESC, sig.detected_at DESC
	`, articleID)
	if err != nil {
		return nil, fmt.Errorf("get signals: %w", err)
	}
	defer rows.Close()

	var out []Signal
	for rows.Next() {
		var s Signal
		if err := rows.Scan(
			&s.ID, &s.ArticleID, &s.SignalTypeID, &s.SignalTypeCode,
			&s.EntityCode, &s.EntityCategory, &s.Confidence,
			&s.DetectedAt, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan signal: %w", err)
		}
		out = append(out, s)
	}
	return out, nil
}

// GetSignalTypeID resolves a signal type code to its DB ID.
func (s *PGStore) GetSignalTypeID(ctx context.Context, code SignalTypeCode) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id FROM signal_types WHERE code = $1`, string(code)).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("signal_type %q not found: %w", code, err)
	}
	return id, nil
}

// Ensure PGStore implements Store.
var _ Store = (*PGStore)(nil)

// ── Service ──────────────────────────────────────────────────

// Pipeline orchestrates signal detection from knowledge records.
type Pipeline struct {
	detector Detector
	store    Store
}

// NewPipeline creates a new signal detection pipeline.
func NewPipeline(detector Detector, store Store) *Pipeline {
	return &Pipeline{
		detector: detector,
		store:    store,
	}
}

// ProcessOne runs detection on one knowledge record and stores the signals.
// Returns the number of new signals created.
func (p *Pipeline) ProcessOne(ctx context.Context, articleID string, knowledge *extract.Knowledge) (int, error) {
	signals, err := p.detector.Detect(ctx, articleID, knowledge)
	if err != nil {
		return 0, fmt.Errorf("detect: %w", err)
	}
	if len(signals) == 0 {
		return 0, nil
	}

	count, err := p.store.InsertSignals(ctx, signals)
	if err != nil {
		return 0, fmt.Errorf("insert signals: %w", err)
	}
	return count, nil
}

// timeNow is used for clock mocking in tests.
var timeNow = time.Now