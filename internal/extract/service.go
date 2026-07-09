package extract

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Service orchestrates the extraction pipeline.
type Service struct {
	store  *JobStore
	worker *Worker
	pool   *pgxpool.Pool
}

// NewService creates a new Service with the given provider.
func NewService(pool *pgxpool.Pool, provider Provider, apiKey, model string) *Service {
	store := NewJobStore(pool)
	worker := NewWorker(store, provider, pool, apiKey, model)
	return &Service{
		store:  store,
		worker: worker,
		pool:   pool,
	}
}

// Enqueue creates jobs for all unprocessed articles.
func (s *Service) Enqueue(ctx context.Context, priority int) (int, error) {
	count, err := s.store.EnqueueNewArticles(ctx, priority)
	if err != nil {
		return 0, fmt.Errorf("extract enqueue: %w", err)
	}
	log.Printf("extract: enqueued %d articles", count)
	return count, nil
}

// RunOnce processes a single job and returns whether one was processed.
func (s *Service) RunOnce(ctx context.Context) (bool, error) {
	return s.worker.RunOnce(ctx)
}

// RunAll processes all queued jobs sequentially.
func (s *Service) RunAll(ctx context.Context, maxJobs int) (int, error) {
	return s.worker.Run(ctx, maxJobs)
}

// RunDaemon runs the worker in a loop, polling for new jobs at the given interval.
func (s *Service) RunDaemon(ctx context.Context, interval time.Duration) {
	log.Printf("extract: daemon started — poll interval %s", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run once immediately.
	_, _ = s.RunAll(ctx, 0)

	for {
		select {
		case <-ctx.Done():
			log.Println("extract: daemon shutting down")
			return
		case <-ticker.C:
			processed, _ := s.RunAll(ctx, 0)
			if processed > 0 {
				log.Printf("extract: daemon processed %d jobs", processed)
			}
		}
	}
}