package intelligence

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkerID is a unique identifier for a worker instance.
var WorkerID = fmt.Sprintf("worker-%d", time.Now().UnixNano())

// JobStore handles database operations for analysis jobs.
type JobStore struct {
	pool *pgxpool.Pool
}

// NewJobStore creates a new JobStore.
func NewJobStore(pool *pgxpool.Pool) *JobStore {
	return &JobStore{pool: pool}
}

// ClaimNextJob picks the next queued job, marks it processing, and returns it.
func (s *JobStore) ClaimNextJob(ctx context.Context, workerID string) (*AnalysisJob, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Use SKIP LOCKED to avoid worker contention.
	var job AnalysisJob
	err = tx.QueryRow(ctx, `
		UPDATE analysis_jobs
		SET status = 'processing',
		    worker_id = $1,
		    started_at = now(),
		    retry_count = retry_count + 1
		WHERE id = (
			SELECT id FROM analysis_jobs
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

// MarkArticleAnalyzed updates the article status to 'analyzed'.
func (s *JobStore) MarkArticleAnalyzed(ctx context.Context, articleID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE articles SET status = 'analyzed' WHERE id = $1`, articleID)
	if err != nil {
		return fmt.Errorf("mark article analyzed: %w", err)
	}
	return nil
}

// CompleteJob marks a job as completed.
func (s *JobStore) CompleteJob(ctx context.Context, jobID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE analysis_jobs
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
		UPDATE analysis_jobs
		SET status = 'failed', finished_at = now(), error_message = $2
		WHERE id = $1
	`, jobID, errMsg)
	if err != nil {
		return fmt.Errorf("fail job: %w", err)
	}
	return nil
}

// SaveResult inserts or updates an analysis result for an article.
func (s *JobStore) SaveResult(ctx context.Context, result *AnalysisResult) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO analysis_results
			(article_id, summary, business_impact, urgency, confidence,
			 reasoning, raw_json, llm_provider, llm_model)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (article_id) DO UPDATE SET
			summary = EXCLUDED.summary,
			business_impact = EXCLUDED.business_impact,
			urgency = EXCLUDED.urgency,
			confidence = EXCLUDED.confidence,
			reasoning = EXCLUDED.reasoning,
			raw_json = EXCLUDED.raw_json,
			llm_provider = EXCLUDED.llm_provider,
			llm_model = EXCLUDED.llm_model
	`, result.ArticleID, result.Summary, result.BusinessImpact,
		result.Urgency, result.Confidence, result.Reasoning,
		result.RawJSON, result.LLMProvider, result.LLMModel)
	if err != nil {
		return fmt.Errorf("save result: %w", err)
	}
	return nil
}

// LoadArticleData loads the minimal article data needed for analysis.
func (s *JobStore) LoadArticleData(ctx context.Context, articleID string) (*ArticleData, error) {
	var a ArticleData
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

// EnqueueNewArticles creates analysis jobs for articles that don't have one yet.
func (s *JobStore) EnqueueNewArticles(ctx context.Context, priority int) (int, error) {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO analysis_jobs (article_id, status, priority)
		SELECT a.id, 'queued', $1::int
		FROM articles a
		WHERE a.status = 'new'
		  AND NOT EXISTS (
			SELECT 1 FROM analysis_jobs j WHERE j.article_id = a.id
		  )
		ORDER BY a.created_at ASC
	`, priority)
	if err != nil {
		return 0, fmt.Errorf("enqueue articles: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// Worker processes analysis jobs using the given provider.
type Worker struct {
	store    *JobStore
	provider IntelligenceProvider
}

// NewWorker creates a new Worker.
func NewWorker(store *JobStore, provider IntelligenceProvider) *Worker {
	return &Worker{store: store, provider: provider}
}

// RunOnce processes a single job. Returns true if a job was processed.
func (w *Worker) RunOnce(ctx context.Context) (bool, error) {
	job, err := w.store.ClaimNextJob(ctx, WorkerID)
	if err != nil {
		return false, nil // no job available
	}

	log.Printf("intelligence: processing job %s — article %s", job.ID, job.ArticleID)

	article, err := w.store.LoadArticleData(ctx, job.ArticleID)
	if err != nil {
		errMsg := fmt.Sprintf("load article: %v", err)
		log.Printf("intelligence: %s", errMsg)
		_ = w.store.FailJob(ctx, job.ID, errMsg)
		return true, nil
	}

	result, err := w.provider.Analyze(ctx, *article)
	if err != nil {
		errMsg := fmt.Sprintf("analyze: %v", err)
		log.Printf("intelligence: %s", errMsg)
		_ = w.store.FailJob(ctx, job.ID, errMsg)
		return true, nil
	}

	if err := w.store.SaveResult(ctx, result); err != nil {
		errMsg := fmt.Sprintf("save result: %v", err)
		log.Printf("intelligence: %s", errMsg)
		_ = w.store.FailJob(ctx, job.ID, errMsg)
		return true, nil
	}

	if err := w.store.CompleteJob(ctx, job.ID); err != nil {
		log.Printf("intelligence: complete job: %v", err)
		return true, nil
	}

	if err := w.store.MarkArticleAnalyzed(ctx, job.ArticleID); err != nil {
		log.Printf("intelligence: mark article analyzed: %v", err)
	}

	log.Printf("intelligence: job %s completed — article %s analyzed", job.ID, job.ArticleID)
	return true, nil
}

// Run processes jobs in a loop until ctx is cancelled or no more jobs.
func (w *Worker) Run(ctx context.Context, maxJobs int) (int, error) {
	processed := 0
	for {
		select {
		case <-ctx.Done():
			return processed, ctx.Err()
		default:
		}

		if maxJobs > 0 && processed >= maxJobs {
			return processed, nil
		}

		ok, err := w.RunOnce(ctx)
		if err != nil {
			return processed, err
		}
		if !ok {
			return processed, nil
		}
		processed++
	}
}