package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"atlas/internal/discover"
)

func runSourceDiscover() {
	fs := flag.NewFlagSet("discover", flag.ExitOnError)
	batch := fs.String("batch", "", "File with list of URLs/domains to discover (one per line)")
	help := fs.Bool("help", false, "Show help")

	fs.Usage = func() {
		fmt.Println("Usage: atlas source discover [flags] <url-or-domain>")
		fmt.Println()
		fmt.Println("Probe a domain for RSS feeds and score it for Atlas.")
		fmt.Println()
		fmt.Println("Flags:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  atlas source discover antaranews.com              Discover single domain")
		fmt.Println("  atlas source discover https://example.com         Discover with full URL")
		fmt.Println("  atlas source discover --batch sources.txt         Batch discover from file")
	}

	fs.Parse(os.Args[3:]) // atlas source discover [flags] ...

	if *help {
		fs.Usage()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if *batch != "" {
		runBatchDiscover(ctx, *batch)
		return
	}

	args := fs.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: domain or URL required\n\n")
		fs.Usage()
		os.Exit(1)
	}

	result, err := discover.Discover(ctx, args[0])
	if err != nil {
		log.Fatalf("discover: %v", err)
	}

	printDiscoverResult(result)
}

func runBatchDiscover(ctx context.Context, filePath string) {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("open batch file: %v", err)
	}
	defer f.Close()

	var domains []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			domains = append(domains, line)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("read batch file: %v", err)
	}

	if len(domains) == 0 {
		log.Fatal("batch file is empty")
	}

	fmt.Printf("━━━ Batch Discovery: %d domains ━━━\n", len(domains))
	fmt.Println()

	for i, domain := range domains {
		fmt.Printf("[%d/%d] %s ... ", i+1, len(domains), domain)

		result, err := discover.Discover(ctx, domain)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			continue
		}

		if result.RSSAvailable {
			fmt.Printf("✓ RSS found | %s | %d articles | %s\n", result.Score, result.ArticleCount, result.Recommendation)
		} else {
			if result.Title != "" {
				fmt.Printf("✗ No RSS | %s | HTTP %d\n", result.Title, result.HTTPStatus)
			} else {
				fmt.Printf("✗ No RSS | HTTP %d\n", result.HTTPStatus)
			}
		}
	}
}

func printDiscoverResult(r *discover.DiscoverResult) {
	fmt.Println()
	fmt.Println("━━━ Source Discovery ━━━")
	fmt.Println()

	fmt.Printf("Domain:        %s\n", r.Domain)
	fmt.Printf("Source URL:    %s\n", r.SourceURL)

	if r.RSSAvailable {
		fmt.Printf("RSS URL:       %s\n", r.RSSURL)
		fmt.Printf("RSS Available: ✓ Yes\n")
	} else {
		fmt.Printf("RSS Available: ✗ No\n")
	}

	if r.Title != "" {
		fmt.Printf("Title:         %s\n", r.Title)
	}
	if r.Description != "" {
		fmt.Printf("Description:   %s\n", r.Description)
	}
	if len(r.Categories) > 0 {
		fmt.Printf("Categories:    %s\n", strings.Join(r.Categories, ", "))
	}
	if r.LastUpdated != nil {
		daysAgo := int(time.Since(*r.LastUpdated).Hours() / 24)
		label := fmt.Sprintf("%s (%d days ago)", r.LastUpdated.Format("2006-01-02"), daysAgo)
		if daysAgo == 0 {
			label = r.LastUpdated.Format("2006-01-02") + " (today)"
		}
		fmt.Printf("Last Updated:  %s\n", label)
	}
	if r.ArticleCount > 0 {
		fmt.Printf("Articles:      %d\n", r.ArticleCount)
	}
	if r.AvgArticlesPerDay > 0 {
		fmt.Printf("Avg/Day:       %.1f\n", r.AvgArticlesPerDay)
	}
	fmt.Printf("HTTP Status:   %d\n", r.HTTPStatus)
	fmt.Println()

	scoreLabel := r.Score
	switch r.Score {
	case "active":
		scoreLabel = "✓ Active"
	case "inactive":
		scoreLabel = "✗ Inactive"
	case "no_rss":
		scoreLabel = "✗ No RSS"
	case "moderate":
		scoreLabel = "~ Moderate"
	}
	fmt.Printf("Score:         %s\n", scoreLabel)

	recLabel := r.Recommendation
	switch r.Recommendation {
	case "layak":
		recLabel = "✓ Layak masuk Atlas"
	case "tidak_layak":
		recLabel = "✗ Tidak layak"
	}
	fmt.Printf("Recommend:     %s\n", recLabel)

	if r.Error != "" {
		fmt.Printf("Error:         %s\n", r.Error)
	}

	fmt.Println()
}