package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"atlas/configs"
	"atlas/internal/database"
	"atlas/internal/extract"
	"atlas/internal/signal"

	"github.com/jackc/pgx/v5/pgxpool"
)

func runDetect() {
	fs := flag.NewFlagSet("detect", flag.ExitOnError)
	articleID := fs.String("article-id", "", "Process a specific article by ID")
	all := fs.Bool("all", false, "Process all articles with extraction results")
	max := fs.Int("max", 0, "Maximum articles to process (0 = all)")
	help := fs.Bool("help", false, "Show help")

	fs.Usage = func() {
		fmt.Println("Usage: atlas detect [flags]")
		fmt.Println()
		fmt.Println("Detect business signals from extraction results.")
		fmt.Println("Converts knowledge records into normalized signals (investment, regulation, partnership, etc.)")
		fmt.Println()
		fmt.Println("Flags:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  atlas detect --article-id <uuid>    Detect signals for a specific article")
		fmt.Println("  atlas detect --all                  Detect signals for all articles with results")
		fmt.Println("  atlas detect --all --max 10         Detect signals for up to 10 articles")
	}

	fs.Parse(os.Args[2:])

	if *help {
		fs.Usage()
		return
	}

	ctx := context.Background()
	cfg := configs.Load()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	detector := signal.NewRuleBasedDetector()
	signalStore := signal.NewPGStore(pool)
	pipeline := signal.NewPipeline(detector, signalStore)

	if *articleID != "" {
		detectSingleArticle(ctx, pool, pipeline, *articleID)
		return
	}

	if *all {
		detectAllArticles(ctx, pool, pipeline, *max)
		return
	}

	fs.Usage()
}

func detectSingleArticle(ctx context.Context, pool *pgxpool.Pool, pipeline *signal.Pipeline, articleID string) {
	store := extract.NewJobStore(pool)
	article, err := store.LoadArticleData(ctx, articleID)
	if err != nil {
		log.Fatalf("load article: %v", err)
	}

	var rawJSON *string
	err = pool.QueryRow(ctx, `
		SELECT raw_json FROM extraction_results WHERE article_id = $1
	`, articleID).Scan(&rawJSON)
	if err != nil {
		log.Fatalf("no extraction result for article %s: %v", articleID, err)
	}

	if rawJSON == nil || *rawJSON == "" {
		log.Fatalf("empty extraction result for article %s", articleID)
	}

	knowledge := &extract.Knowledge{}
	if err := json.Unmarshal([]byte(*rawJSON), knowledge); err != nil {
		log.Fatalf("parse knowledge: %v", err)
	}

	extract.NormalizeKnowledge(knowledge)

	fmt.Printf("Detecting signals for article %s...\n", shortID(articleID))
	fmt.Printf("  Title: %s\n", article.Title)

	count, err := pipeline.ProcessOne(ctx, articleID, knowledge)
	if err != nil {
		log.Fatalf("detect: %v", err)
	}
	fmt.Printf("  Signals detected: %d (new)\n", count)
}

func detectAllArticles(ctx context.Context, pool *pgxpool.Pool, pipeline *signal.Pipeline, max int) {
	query := `SELECT article_id FROM extraction_results ORDER BY created_at DESC`
	if max > 0 {
		query = fmt.Sprintf(`SELECT article_id FROM extraction_results ORDER BY created_at DESC LIMIT %d`, max)
	}

	rows, err := pool.Query(ctx, query)
	if err != nil {
		log.Fatalf("query articles: %v", err)
	}
	defer rows.Close()

	var articleIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Fatalf("scan article: %v", err)
		}
		articleIDs = append(articleIDs, id)
	}

	if len(articleIDs) == 0 {
		fmt.Println("No articles with extraction results found.")
		return
	}

	fmt.Printf("Processing %d articles...\n", len(articleIDs))

	total := 0
	for _, id := range articleIDs {
		detectSingleArticle(ctx, pool, pipeline, id)
		total++
	}
	fmt.Printf("Done. Processed %d articles.\n", total)
}