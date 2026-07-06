package article

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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