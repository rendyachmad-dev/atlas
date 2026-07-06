package article

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SourceArticleCount holds article count per source.
type SourceArticleCount struct {
	SourceID   string
	SourceName string
	Total      int
	New        int
	Analyzed   int
}

// FetchLogSummary holds recent fetch log data for display.
type FetchLogSummary struct {
	SourceName      string
	Status          string
	ArticlesFetched int
	ArticlesNew     int
	ErrorMessage    *string
	StartedAt       time.Time
	CompletedAt     *time.Time
}

// Article represents a row from the articles table.
type Article struct {
	ID          string     `json:"id"`
	SourceID    string     `json:"source_id"`
	ExternalID  *string    `json:"external_id,omitempty"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Author      *string    `json:"author,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Content     *string    `json:"content,omitempty"`
	ContentHash string     `json:"content_hash"`
	Language    string     `json:"language"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// FetchLog represents a row from the fetch_logs table.
type FetchLog struct {
	ID              string     `json:"id"`
	SourceID        string     `json:"source_id"`
	Status          string     `json:"status"`
	ArticlesFetched int        `json:"articles_fetched"`
	ArticlesNew     int        `json:"articles_new"`
	ErrorMessage    *string    `json:"error_message,omitempty"`
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// Insert inserts a new article. If the URL already exists, it's a no-op.
// Returns the article ID, or empty string if the article already existed.
func Insert(ctx context.Context, pool *pgxpool.Pool, a *Article) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO articles (source_id, external_id, title, url, author,
		                      published_at, content, content_hash, language, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (url) DO NOTHING
		RETURNING id
	`,
		a.SourceID, a.ExternalID, a.Title, a.URL, a.Author,
		a.PublishedAt, a.Content, a.ContentHash, a.Language, a.Status,
	).Scan(&id)

	if err != nil {
		// No rows means the URL already existed — not an error.
		return "", nil
	}
	return id, nil
}

// CreateFetchLog inserts a new fetch log with status 'running'.
func CreateFetchLog(ctx context.Context, pool *pgxpool.Pool, sourceID string) (*FetchLog, error) {
	fl := &FetchLog{}
	err := pool.QueryRow(ctx, `
		INSERT INTO fetch_logs (source_id, status, started_at)
		VALUES ($1, 'running', now())
		RETURNING id, source_id, status, articles_fetched, articles_new,
		          error_message, started_at, completed_at, created_at
	`, sourceID).Scan(
		&fl.ID, &fl.SourceID, &fl.Status, &fl.ArticlesFetched, &fl.ArticlesNew,
		&fl.ErrorMessage, &fl.StartedAt, &fl.CompletedAt, &fl.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create fetch log: %w", err)
	}
	return fl, nil
}

// CompleteFetchLog updates a fetch log with final status and counts.
// CountArticlesBySource returns article counts grouped by source.
func CountArticlesBySource(ctx context.Context, pool *pgxpool.Pool) ([]SourceArticleCount, error) {
	rows, err := pool.Query(ctx, `
		SELECT s.id, s.name,
		       COUNT(a.id) AS total,
		       COUNT(*) FILTER (WHERE a.status = 'new') AS new,
		       COUNT(*) FILTER (WHERE a.status = 'analyzed') AS analyzed
		FROM sources s
		LEFT JOIN articles a ON a.source_id = s.id
		GROUP BY s.id, s.name
		ORDER BY s.name
	`)
	if err != nil {
		return nil, fmt.Errorf("count articles: %w", err)
	}
	defer rows.Close()

	var results []SourceArticleCount
	for rows.Next() {
		var r SourceArticleCount
		if err := rows.Scan(&r.SourceID, &r.SourceName, &r.Total, &r.New, &r.Analyzed); err != nil {
			return nil, fmt.Errorf("scan count: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// ListRecentFetchLogs returns the most recent fetch logs, ordered by time DESC.
func ListRecentFetchLogs(ctx context.Context, pool *pgxpool.Pool, limit int) ([]FetchLogSummary, error) {
	rows, err := pool.Query(ctx, `
		SELECT s.name, fl.status, fl.articles_fetched, fl.articles_new,
		       fl.error_message, fl.started_at, fl.completed_at
		FROM fetch_logs fl
		JOIN sources s ON s.id = fl.source_id
		ORDER BY fl.started_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list fetch logs: %w", err)
	}
	defer rows.Close()

	var logs []FetchLogSummary
	for rows.Next() {
		var l FetchLogSummary
		if err := rows.Scan(&l.SourceName, &l.Status, &l.ArticlesFetched, &l.ArticlesNew,
			&l.ErrorMessage, &l.StartedAt, &l.CompletedAt); err != nil {
			return nil, fmt.Errorf("scan fetch log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func CompleteFetchLog(ctx context.Context, pool *pgxpool.Pool, id, status string, fetched, new int, errMsg *string) error {
	_, err := pool.Exec(ctx, `
		UPDATE fetch_logs
		SET status = $1, articles_fetched = $2, articles_new = $3,
		    error_message = $4, completed_at = now()
		WHERE id = $5
	`, status, fetched, new, errMsg, id)
	if err != nil {
		return fmt.Errorf("complete fetch log: %w", err)
	}
	return nil
}