package signal

import (
	"context"
	"fmt"
	"time"

	"atlas/internal/extract"
)

// Detector converts knowledge records into business signals.
type Detector interface {
	// Detect converts one knowledge record into zero or more signals.
	Detect(ctx context.Context, articleID string, knowledge *extract.Knowledge) ([]Signal, error)
}

// RuleBasedDetector maps taxonomy entities to signal types using predefined rules.
type RuleBasedDetector struct {
	mappings []SignalTypeMapping
	now      func() time.Time
}

// NewRuleBasedDetector creates a detector with default mappings.
func NewRuleBasedDetector() *RuleBasedDetector {
	return &RuleBasedDetector{
		mappings: DefaultMappings(),
		now:      time.Now,
	}
}

// NewRuleBasedDetectorWithMappings creates a detector with custom mappings.
func NewRuleBasedDetectorWithMappings(mappings []SignalTypeMapping) *RuleBasedDetector {
	return &RuleBasedDetector{
		mappings: mappings,
		now:      time.Now,
	}
}

// Detect converts knowledge to signals using rule-based mappings.
func (d *RuleBasedDetector) Detect(ctx context.Context, articleID string, knowledge *extract.Knowledge) ([]Signal, error) {
	var signals []Signal
	now := d.now()

	// Process each entity category.
	signals = append(signals, d.detectCategory(knowledge.Topics, CatTopics, articleID, now)...)
	signals = append(signals, d.detectCategory(knowledge.Industries, CatIndustries, articleID, now)...)
	signals = append(signals, d.detectCategory(knowledge.Countries, CatCountries, articleID, now)...)
	signals = append(signals, d.detectCategory(knowledge.Technologies, CatTechnologies, articleID, now)...)
	signals = append(signals, d.detectCategory(knowledge.Organizations, CatOrganizations, articleID, now)...)

	return signals, nil
}

func (d *RuleBasedDetector) detectCategory(entities []extract.Entity, category, articleID string, now time.Time) []Signal {
	var signals []Signal
	seen := make(map[SignalTypeCode]bool) // one signal type per entity per article

	for _, entity := range entities {
		matched := false
		for _, m := range d.mappings {
			if m.EntityCategory != category {
				continue
			}
			// Skip if entity-specific mapping doesn't match.
			if m.EntityCode != "" && m.EntityCode != entity.Code {
				continue
			}
			// Skip if confidence too low.
			if entity.Confidence < m.MinConfidence {
				continue
			}
			// Dedup: one signal type per entity.
			key := m.SignalTypeCode
			if seen[key] {
				continue
			}
			seen[key] = true

			signals = append(signals, Signal{
				ArticleID:      articleID,
				SignalTypeCode: m.SignalTypeCode,
				EntityCode:     entity.Code,
				EntityCategory: category,
				Confidence:     entity.Confidence,
				DetectedAt:     now,
			})
			matched = true
		}

		// Fallback: if no specific mapping matched, try catch-all mappings.
		if !matched {
			for _, m := range d.mappings {
				if m.EntityCategory != category {
					continue
				}
				if m.EntityCode != "" {
					continue // not a catch-all
				}
				key := m.SignalTypeCode
				if seen[key] {
					continue
				}
				seen[key] = true

				signals = append(signals, Signal{
					ArticleID:      articleID,
					SignalTypeCode: m.SignalTypeCode,
					EntityCode:     entity.Code,
					EntityCategory: category,
					Confidence:     entity.Confidence * m.MinConfidence, // discount catch-all
					DetectedAt:     now,
				})
			}
		}
	}

	return signals
}

// Ensure implementations.
var _ Detector = (*RuleBasedDetector)(nil)

// ── Store ────────────────────────────────────────────────────

// Store handles signal persistence.
// Can be backed by DB or memory for testing.
type Store interface {
	// InsertSignals inserts signals idempotently. Returns count of new signals.
	InsertSignals(ctx context.Context, signals []Signal) (int, error)
	// GetSignalsByArticle returns all signals for an article.
	GetSignalsByArticle(ctx context.Context, articleID string) ([]Signal, error)
	// GetSignalTypeID resolves a signal type code to its DB ID.
	GetSignalTypeID(ctx context.Context, code SignalTypeCode) (string, error)
}

// MemoryStore is an in-memory store for testing.
type MemoryStore struct {
	types   map[SignalTypeCode]string // code → id
	signals []Signal
}

// NewMemoryStore creates an in-memory store with pre-loaded signal types.
func NewMemoryStore() *MemoryStore {
	types := make(map[SignalTypeCode]string)
	for i, m := range DefaultMappings() {
		// Use a deterministic ID per type code.
		id := fmt.Sprintf("st-%s", m.SignalTypeCode)
		types[m.SignalTypeCode] = id
		_ = i
	}
	return &MemoryStore{
		types: types,
	}
}

func (m *MemoryStore) InsertSignals(ctx context.Context, signals []Signal) (int, error) {
	// Dedup by (article_id, signal_type_id, entity_code).
	seen := make(map[string]bool)
	var newSignals []Signal
	for _, s := range signals {
		key := s.ArticleID + "|" + string(s.SignalTypeCode) + "|" + s.EntityCode
		if seen[key] {
			continue
		}
		seen[key] = true
		// Check existing.
		exists := false
		for _, existing := range m.signals {
			ek := existing.ArticleID + "|" + string(existing.SignalTypeCode) + "|" + existing.EntityCode
			if ek == key {
				exists = true
				break
			}
		}
		if exists {
			continue
		}
		s.ID = fmt.Sprintf("sig-%d", len(m.signals)+len(newSignals)+1)
		s.SignalTypeID = m.types[s.SignalTypeCode]
		newSignals = append(newSignals, s)
	}
	m.signals = append(m.signals, newSignals...)
	return len(newSignals), nil
}

func (m *MemoryStore) GetSignalsByArticle(ctx context.Context, articleID string) ([]Signal, error) {
	var out []Signal
	for _, s := range m.signals {
		if s.ArticleID == articleID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *MemoryStore) GetSignalTypeID(ctx context.Context, code SignalTypeCode) (string, error) {
	id, ok := m.types[code]
	if !ok {
		return "", fmt.Errorf("unknown signal type: %s", code)
	}
	return id, nil
}

// Ensure MemoryStore implements Store.
var _ Store = (*MemoryStore)(nil)