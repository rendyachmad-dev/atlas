package source

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DashboardStats holds system-level stats for the dashboard.
type DashboardStats struct {
	TotalSources       int
	ActiveSources      int
	HealthySources     int
	DegradedSources    int
	DownSources        int
	TotalArticles      int
	TotalFetchLogs     int
	LatestFetchAt      *time.Time
}

// SourceStats aggregates health status counts.
type HealthCount struct {
	HealthStatus string
	Count        int
}

// Source represents a row from the sources table.
type Source struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	Type               string     `json:"type"`
	Category           *string    `json:"category,omitempty"`
	URL                string     `json:"url"`
	Language           string     `json:"language"`
	Country            string     `json:"country"`
	IsActive           bool       `json:"is_active"`
	FetchInterval      int        `json:"fetch_interval"`
	LastFetch          *time.Time `json:"last_fetch,omitempty"`
	LastSuccessAt      *time.Time `json:"last_success_at,omitempty"`
	LastErrorAt        *time.Time `json:"last_error_at,omitempty"`
	LastHTTPStatus     *int       `json:"last_http_status,omitempty"`
	LastContentType    *string    `json:"last_content_type,omitempty"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	HealthStatus       string     `json:"health_status"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

const sourceColumns = `id, name, type, category, url, language, country,
	is_active, fetch_interval, last_fetch,
	last_success_at, last_error_at, last_http_status, last_content_type,
	consecutive_failures, health_status,
	created_at, updated_at`

// ListActive returns all sources where is_active = true.
func ListActive(ctx context.Context, pool *pgxpool.Pool) ([]Source, error) {
	rows, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT %s FROM sources
		WHERE is_active = true
		ORDER BY name
	`, sourceColumns))
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}
	defer rows.Close()

	var sources []Source
	for rows.Next() {
		s, err := scanSource(rows)
		if err != nil {
			return nil, err
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

// RecordSuccess updates health fields after a successful fetch.
func RecordSuccess(ctx context.Context, pool *pgxpool.Pool, sourceID string, httpStatus int, contentType string) error {
	_, err := pool.Exec(ctx, `
		UPDATE sources SET
			last_fetch = now(),
			last_success_at = now(),
			last_http_status = $2,
			last_content_type = $3,
			consecutive_failures = 0,
			health_status = 'healthy'
		WHERE id = $1
	`, sourceID, httpStatus, contentType)
	if err != nil {
		return fmt.Errorf("record success: %w", err)
	}
	return nil
}

// RecordFailure updates health fields after a failed fetch.
// It increments consecutive_failures and updates health_status accordingly.
func RecordFailure(ctx context.Context, pool *pgxpool.Pool, sourceID string, httpStatus *int, errMsg string) error {
	_, err := pool.Exec(ctx, `
		UPDATE sources SET
			last_fetch = now(),
			last_error_at = now(),
			last_http_status = $2,
			last_content_type = NULL,
			consecutive_failures = consecutive_failures + 1,
			health_status = CASE
				WHEN consecutive_failures + 1 >= 5 THEN 'down'
				WHEN consecutive_failures + 1 >= 2 THEN 'degraded'
				ELSE 'degraded'
			END
		WHERE id = $1
	`, sourceID, httpStatus)
	if err != nil {
		return fmt.Errorf("record failure: %w", err)
	}
	return nil
}

// ResetHealth resets health tracking for a source (e.g., after manual fix).
func ResetHealth(ctx context.Context, pool *pgxpool.Pool, sourceID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE sources SET
			consecutive_failures = 0,
			health_status = 'healthy'
		WHERE id = $1
	`, sourceID)
	if err != nil {
		return fmt.Errorf("reset health: %w", err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

// GetDashboardStats returns aggregate counts for the dashboard.
func GetDashboardStats(ctx context.Context, pool *pgxpool.Pool) (DashboardStats, error) {
	var s DashboardStats
	err := pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM sources) AS total_sources,
			(SELECT COUNT(*) FROM sources WHERE is_active = true) AS active_sources,
			(SELECT COUNT(*) FROM sources WHERE health_status = 'healthy') AS healthy_sources,
			(SELECT COUNT(*) FROM sources WHERE health_status = 'degraded') AS degraded_sources,
			(SELECT COUNT(*) FROM sources WHERE health_status = 'down') AS down_sources,
			(SELECT COUNT(*) FROM articles) AS total_articles,
			(SELECT COUNT(*) FROM fetch_logs) AS total_fetch_logs,
			(SELECT MAX(started_at) FROM fetch_logs) AS latest_fetch_at
	`).Scan(
		&s.TotalSources, &s.ActiveSources,
		&s.HealthySources, &s.DegradedSources, &s.DownSources,
		&s.TotalArticles, &s.TotalFetchLogs, &s.LatestFetchAt,
	)
	if err != nil {
		return s, fmt.Errorf("dashboard stats: %w", err)
	}
	return s, nil
}

func scanSource(row scanner) (Source, error) {
	var s Source
	err := row.Scan(
		&s.ID, &s.Name, &s.Type, &s.Category, &s.URL,
		&s.Language, &s.Country, &s.IsActive, &s.FetchInterval,
		&s.LastFetch,
		&s.LastSuccessAt, &s.LastErrorAt, &s.LastHTTPStatus,
		&s.LastContentType, &s.ConsecutiveFailures, &s.HealthStatus,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return s, fmt.Errorf("scan source: %w", err)
	}
	return s, nil
}