package rss

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
)

// FetchFeed fetches an RSS/Atom feed from url and returns parsed entries.
func FetchFeed(ctx context.Context, url string) (*gofeed.Feed, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch request: %w", err)
	}
	req.Header.Set("User-Agent", "AtlasRSS/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, resp.Status)
	}

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("feed parse: %w", err)
	}

	return feed, nil
}