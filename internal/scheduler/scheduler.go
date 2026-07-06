package scheduler

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"time"

	"atlas/internal/article"
	"atlas/internal/rss"
	"atlas/internal/source"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mmcdole/gofeed"
)

// Run starts the fetch loop. It runs until ctx is cancelled.
// interval controls how often the full source list is checked.
func Run(ctx context.Context, pool *pgxpool.Pool, interval time.Duration) {
	if interval <= 0 {
		interval = 60 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run once immediately on start.
	doCycle(ctx, pool)

	for {
		select {
		case <-ctx.Done():
			log.Println("scheduler: shutting down")
			return
		case <-ticker.C:
			doCycle(ctx, pool)
		}
	}
}

func doCycle(ctx context.Context, pool *pgxpool.Pool) {
	sources, err := source.ListActive(ctx, pool)
	if err != nil {
		log.Printf("scheduler: list sources: %v", err)
		return
	}

	log.Printf("scheduler: cycle — %d active sources", len(sources))

	var totalFetched, totalNew int
	now := time.Now()
	for _, s := range sources {
		// Skip if last_fetch + fetch_interval hasn't elapsed yet.
		if s.LastFetch != nil {
			nextFetch := s.LastFetch.Add(time.Duration(s.FetchInterval) * time.Minute)
			if now.Before(nextFetch) {
				continue
			}
		}

		fetched, newCount := fetchSource(ctx, pool, &s)
		totalFetched += fetched
		totalNew += newCount
	}

	log.Printf("scheduler: cycle done — %d fetched, %d new", totalFetched, totalNew)
}

func fetchSource(ctx context.Context, pool *pgxpool.Pool, s *source.Source) (int, int) {
	log.Printf("scheduler: fetching %s (%s)", s.Name, s.URL)

	fl, err := article.CreateFetchLog(ctx, pool, s.ID)
	if err != nil {
		log.Printf("scheduler: create fetch log for %s: %v", s.Name, err)
		return 0, 0
	}

	feed, err := rss.FetchFeed(ctx, s.URL)
	if err != nil {
		errMsg := err.Error()
		log.Printf("scheduler: fetch %s: %v", s.Name, err)
		_ = article.CompleteFetchLog(ctx, pool, fl.ID, "failed", 0, 0, &errMsg)
		return 0, 0
	}

	var newCount int
	for _, entry := range feed.Items {
		a := mapFeedItem(entry, s)
		id, err := article.Insert(ctx, pool, a)
		if err != nil {
			log.Printf("scheduler: insert article %q: %v", truncate(entry.Title, 60), err)
			continue
		}
		if id != "" {
			newCount++
		}
	}

	totalFetched := len(feed.Items)
	log.Printf("scheduler: %s — %d entries, %d new", s.Name, totalFetched, newCount)

	_ = source.UpdateLastFetch(ctx, pool, s.ID)
	_ = article.CompleteFetchLog(ctx, pool, fl.ID, "success", totalFetched, newCount, nil)

	return totalFetched, newCount
}

func mapFeedItem(entry *gofeed.Item, s *source.Source) *article.Article {
	hash := sha256.Sum256([]byte(entry.Link + entry.Title))
	contentHash := fmt.Sprintf("%x", hash)

	var content *string
	if entry.Content != "" {
		content = &entry.Content
	} else if entry.Description != "" {
		content = &entry.Description
	}

	var author *string
	if entry.Author != nil && entry.Author.Name != "" {
		author = &entry.Author.Name
	}

	var externalID *string
	if entry.GUID != "" {
		externalID = &entry.GUID
	}

	return &article.Article{
		SourceID:    s.ID,
		ExternalID:  externalID,
		Title:       entry.Title,
		URL:         entry.Link,
		Author:      author,
		PublishedAt: entry.PublishedParsed,
		Content:     content,
		ContentHash: contentHash,
		Language:    s.Language,
		Status:      "new",
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}