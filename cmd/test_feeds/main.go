package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/mmcdole/gofeed"
)

func main() {
	urls := []string{
		"https://techcrunch.com/feed/",
		"https://openai.com/news/rss.xml",
		"https://www.cnbcindonesia.com/rss",
		"https://katadata.co.id/rss",
		"https://www.anthropic.com/blog/rss.xml",
	}
	client := &http.Client{Timeout: 15 * time.Second}
	exitCode := 0
	for _, u := range urls {
		req, _ := http.NewRequestWithContext(context.Background(), "GET", u, nil)
		req.Header.Set("User-Agent", "AtlasRSS/1.0")
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("✗ %s → http err: %v\n", u, err)
			exitCode = 1
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("→ %s HTTP %d (%d bytes)\n", u, resp.StatusCode, len(body))
		fp := gofeed.NewParser()
		feed, err := fp.ParseString(string(body))
		if err != nil {
			fmt.Printf("  ✗ parse: %v\n", err)
			exitCode = 1
		} else {
			fmt.Printf("  ✓ %d entries — %s\n", len(feed.Items), feed.Title)
		}
	}
	os.Exit(exitCode)
}
