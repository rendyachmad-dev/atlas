package source

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Source represents a row from the sources table.
type Source struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Type          string     `json:"type"`
	Category      *string    `json:"category,omitempty"`
	URL           string     `json:"url"`
	Language      string     `json:"language"`
	Country       string     `json:"country"`
	IsActive      bool       `json:"is_active"`
	FetchInterval int        `json:"fetch_interval"`
	LastFetch     *time.Time `json:"last_fetch,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ListActive returns all sources where is_active = true.
func ListActive(ctx context.Context, pool *pgxpool.Pool) ([]Source, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, name, type, category, url, language, country,
		       is_active, fetch_interval, last_fetch, created_at, updated_at
		FROM sources
		WHERE is_active = true
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}
	defer rows.Close()

	var sources []Source
	for rows.Next() {
		var s Source
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Type, &s.Category, &s.URL,
			&s.Language, &s.Country, &s.IsActive, &s.FetchInterval,
			&s.LastFetch, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan source: %w", err)
		}
		sources = append(sources, s)
	}
	return sources, rows.Err()
}

// UpdateLastFetch sets last_fetch to now for a source.
func UpdateLastFetch(ctx context.Context, pool *pgxpool.Pool, sourceID string) error {
	_, err := pool.Exec(ctx, `UPDATE sources SET last_fetch = now() WHERE id = $1`, sourceID)
	if err != nil {
		return fmt.Errorf("update source last_fetch: %w", err)
	}
	return nil
}