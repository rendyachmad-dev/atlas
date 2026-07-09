package extract

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// JobStore handles database operations for extraction jobs.
type JobStore struct {
	pool *pgxpool.Pool
}

// NewJobStore creates a new JobStore.
func NewJobStore(pool *pgxpool.Pool) *JobStore {
	return &JobStore{pool: pool}
}

// ClaimNextJob picks the next queued job, marks it processing, and returns it.
func (s *JobStore) ClaimNextJob(ctx context.Context, workerID string) (*ExtractionJob, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var job ExtractionJob
	err = tx.QueryRow(ctx, `
		UPDATE extraction_jobs
		SET status = 'processing',
		    worker_id = $1,
		    started_at = now(),
		    retry_count = retry_count + 1
		WHERE id = (
			SELECT id FROM extraction_jobs
			WHERE status = 'queued'
			ORDER BY priority DESC, created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, article_id, status, priority, retry_count,
		          worker_id, started_at, finished_at, error_message, created_at
	`, workerID).Scan(
		&job.ID, &job.ArticleID, &job.Status, &job.Priority, &job.RetryCount,
		&job.WorkerID, &job.StartedAt, &job.FinishedAt, &job.ErrorMessage, &job.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("claim job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &job, nil
}

// CompleteJob marks a job as completed.
func (s *JobStore) CompleteJob(ctx context.Context, jobID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE extraction_jobs
		SET status = 'completed', finished_at = now()
		WHERE id = $1
	`, jobID)
	if err != nil {
		return fmt.Errorf("complete job: %w", err)
	}
	return nil
}

// FailJob marks a job as failed with an error message.
func (s *JobStore) FailJob(ctx context.Context, jobID string, errMsg string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE extraction_jobs
		SET status = 'failed', finished_at = now(), error_message = $2
		WHERE id = $1
	`, jobID, errMsg)
	if err != nil {
		return fmt.Errorf("fail job: %w", err)
	}
	return nil
}

// SaveResult inserts or updates an extraction result for an article.
func (s *JobStore) SaveResult(ctx context.Context, result *ExtractionResult) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO extraction_results
			(article_id, raw_json, llm_provider, llm_model)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (article_id) DO UPDATE SET
			raw_json = EXCLUDED.raw_json,
			llm_provider = EXCLUDED.llm_provider,
			llm_model = EXCLUDED.llm_model
	`, result.ArticleID, result.RawJSON, result.LLMProvider, result.LLMModel)
	if err != nil {
		return fmt.Errorf("save result: %w", err)
	}
	return nil
}

// EnqueueNewArticles creates extraction jobs for articles that don't have one yet.
func (s *JobStore) EnqueueNewArticles(ctx context.Context, priority int) (int, error) {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO extraction_jobs (article_id, status, priority)
		SELECT a.id, 'queued', $1::int
		FROM articles a
		WHERE a.status = 'new'
		  AND NOT EXISTS (
			SELECT 1 FROM extraction_jobs j WHERE j.article_id = a.id
		  )
		ORDER BY a.created_at ASC
	`, priority)
	if err != nil {
		return 0, fmt.Errorf("enqueue articles: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// LoadArticleData loads the minimal article data needed for extraction.
func (s *JobStore) LoadArticleData(ctx context.Context, articleID string) (*Article, error) {
	var a Article
	err := s.pool.QueryRow(ctx, `
		SELECT a.id, a.title, COALESCE(a.content, ''), a.url, s.name, a.author
		FROM articles a
		JOIN sources s ON s.id = a.source_id
		WHERE a.id = $1
	`, articleID).Scan(&a.ID, &a.Title, &a.Content, &a.URL, &a.Source, &a.Author)
	if err != nil {
		return nil, fmt.Errorf("load article: %w", err)
	}
	return &a, nil
}