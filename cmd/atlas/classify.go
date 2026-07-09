package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"

	"atlas/configs"
	"atlas/internal/classify"
	"atlas/internal/database"
)

func runClassify() {
	fs := flag.NewFlagSet("classify", flag.ExitOnError)
	mock := fs.Bool("mock", false, "Use MockClassifier instead of real LLM")
	force := fs.Bool("force", false, "Re-classify even if already classified")
	help := fs.Bool("help", false, "Show help")

	fs.Usage = func() {
		fmt.Println("Usage: atlas classify [flags] <article-id>")
		fmt.Println()
		fmt.Println("Classify an article by UUID — industry, technology, country, stakeholder.")
		fmt.Println()
		fmt.Println("Flags:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  atlas classify <uuid>                          Classify with real LLM")
		fmt.Println("  atlas classify --mock <uuid>                   Classify with mock data")
		fmt.Println("  atlas classify --force <uuid>                  Re-classify, ignoring cache")
	}

	fs.Parse(os.Args[2:])

	if *help {
		fs.Usage()
		return
	}

	// Remaining args after flags = article ID.
	args := fs.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: article-id is required\n\n")
		fs.Usage()
		os.Exit(1)
	}
	articleID := args[0]

	ctx := context.Background()
	cfg := configs.Load()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	var classifier classify.Classifier

	if *mock {
		classifier = classify.NewMock(pool)
		fmt.Println("Using MockClassifier (deterministic fake data)")
	} else {
		if cfg.AnthropicAPIKey == "" {
			log.Fatal("ANTHROPIC_API_KEY not set. Use --mock to test without API key, or set ANTHROPIC_API_KEY env var.")
		}
		classifier = classify.New(pool, cfg.AnthropicAPIKey, cfg.AnthropicModel)
	}

	if *force {
		// Drop existing classification so it re-runs.
		_, err := pool.Exec(ctx, `DELETE FROM article_classifications WHERE article_id = $1`, articleID)
		if err != nil {
			log.Fatalf("force reset: %v", err)
		}
		log.Printf("Force mode: deleted existing classification for %s", shortID(articleID))
	}

	result, err := classifier.ClassifyByID(ctx, articleID)
	if err != nil {
		log.Fatalf("classify: %v", err)
	}

	printClassification(result)
}

func printClassification(c *classify.Classification) {
	fmt.Printf("\nArticle #%s\n\n", shortID(c.ArticleID))

	fmt.Println("Industry")
	fmt.Println(c.IndustryName)
	fmt.Printf("  Confidence  %.0f%%\n\n", c.IndustryConfidence*100)

	fmt.Println("Technology")
	fmt.Println(c.TechnologyName)
	fmt.Printf("  Confidence  %.0f%%\n\n", c.TechnologyConfidence*100)

	fmt.Println("Country")
	fmt.Println(c.RegionName)
	fmt.Printf("  Confidence  %.0f%%\n\n", c.RegionConfidence*100)

	fmt.Println("Stakeholder")
	fmt.Println(c.StakeholderName)
	fmt.Printf("  Confidence  %.0f%%\n\n", c.StakeholderConfidence*100)

	fmt.Println("━━━ Summary ━━━")
	fmt.Printf("Overall Confidence  %.0f%%\n", c.OverallConfidence*100)
	fmt.Printf("Model               %s\n", c.LLMModel)
}

func shortID(uuid string) string {
	if len(uuid) < 8 {
		return uuid
	}
	for i := 0; i < len(uuid) && i < 36; i++ {
		if uuid[i] == '-' {
			first := uuid[:i]
			if len(first) > 6 {
				first = first[:6]
			}
			return first + "..."
		}
	}
	return uuid[:int(math.Min(8, float64(len(uuid))))] + "..."
}