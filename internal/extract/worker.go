package extract

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkerID is a unique identifier for a worker instance.
var WorkerID = fmt.Sprintf("extract-worker-%d", time.Now().UnixNano())

// Worker processes extraction jobs using the given provider.
type Worker struct {
	store    *JobStore
	provider Provider
	pool     *pgxpool.Pool
	apiKey   string
	model    string
}

// NewWorker creates a new Worker.
func NewWorker(store *JobStore, provider Provider, pool *pgxpool.Pool, apiKey, model string) *Worker {
	return &Worker{
		store:    store,
		provider: provider,
		pool:     pool,
		apiKey:   apiKey,
		model:    model,
	}
}

// RunOnce processes a single job. Returns true if a job was processed.
func (w *Worker) RunOnce(ctx context.Context) (bool, error) {
	job, err := w.store.ClaimNextJob(ctx, WorkerID)
	if err != nil {
		return false, nil // no job available
	}

	log.Printf("extract: processing job %s — article %s", job.ID, job.ArticleID)

	article, err := w.store.LoadArticleData(ctx, job.ArticleID)
	if err != nil {
		errMsg := fmt.Sprintf("load article: %v", err)
		log.Printf("extract: %s", errMsg)
		_ = w.store.FailJob(ctx, job.ID, errMsg)
		return true, nil
	}

	// First attempt.
	k, err := w.provider.Analyze(*article)
	if err != nil {
		errMsg := fmt.Sprintf("extract: %v", err)
		log.Printf("extract: %s", errMsg)
		_ = w.store.FailJob(ctx, job.ID, errMsg)
		return true, nil
	}

	// Validate.
	if err := ValidateKnowledge(k); err != nil {
		log.Printf("extract: validation failed (attempt 1) — %v", err)

		// Retry once.
		k, err = w.provider.Analyze(*article)
		if err != nil {
			errMsg := fmt.Sprintf("extract retry: %v", err)
			log.Printf("extract: %s", errMsg)
			_ = w.store.FailJob(ctx, job.ID, errMsg)
			return true, nil
		}

		// Validate again.
		if err := ValidateKnowledge(k); err != nil {
			errMsg := fmt.Sprintf("validation failed after retry: %v", err)
			log.Printf("extract: %s", errMsg)
			_ = w.store.FailJob(ctx, job.ID, errMsg)
			return true, nil
		}
	}

	// Resolve taxonomy codes and insert into M2M junction tables.
	if err := ResolveAndInsert(ctx, w.pool, job.ArticleID, k); err != nil {
		errMsg := fmt.Sprintf("resolve and insert: %v", err)
		log.Printf("extract: %s", errMsg)
		_ = w.store.FailJob(ctx, job.ID, errMsg)
		return true, nil
	}

	// Save raw result.
	rawJSON, _ := json.Marshal(k)
	rawStr := string(rawJSON)
	providerName := "mock"
	modelName := "mock-v1"
	if w.apiKey != "" {
		providerName = "anthropic"
		modelName = w.model
	}
	result := &ExtractionResult{
		ArticleID:   job.ArticleID,
		RawJSON:     &rawStr,
		LLMProvider: &providerName,
		LLMModel:    &modelName,
	}
	if err := w.store.SaveResult(ctx, result); err != nil {
		errMsg := fmt.Sprintf("save result: %v", err)
		log.Printf("extract: %s", errMsg)
		_ = w.store.FailJob(ctx, job.ID, errMsg)
		return true, nil
	}

	// Mark job completed.
	if err := w.store.CompleteJob(ctx, job.ID); err != nil {
		log.Printf("extract: complete job: %v", err)
		return true, nil
	}

	log.Printf("extract: job %s completed — article %s extracted", job.ID, shortID(job.ArticleID))
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

func shortID(id string) string {
	if len(id) < 8 {
		return id
	}
	return id[:8] + "..."
}