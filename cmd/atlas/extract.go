package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"atlas/configs"
	"atlas/internal/database"
	"atlas/internal/extract"

	"github.com/jackc/pgx/v5/pgxpool"
)

func runExtract() {
	fs := flag.NewFlagSet("extract", flag.ExitOnError)
	once := fs.Bool("once", false, "Process available jobs once and exit")
	enqueue := fs.Bool("enqueue", false, "Enqueue new articles before processing")
	max := fs.Int("max", 0, "Max jobs to process (0 = all available)")
	mock := fs.Bool("mock", false, "Use MockProvider instead of Anthropic API")
	priority := fs.Int("priority", 5, "Priority for enqueued jobs")
	daemon := fs.Bool("daemon", false, "Run in continuous daemon mode")
	interval := fs.Duration("interval", 60*time.Second, "Poll interval in daemon mode")
	articleID := fs.String("article-id", "", "Process a specific article by ID (bypasses job queue)")
	help := fs.Bool("help", false, "Show help")

	fs.Usage = func() {
		fmt.Println("Usage: atlas extract [flags]")
		fmt.Println()
		fmt.Println("Extract taxonomy entities from articles via LLM — topics, industries, countries, technologies, organizations.")
		fmt.Println()
		fmt.Println("Flags:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  atlas extract --once --mock               Process all queued jobs with mock data")
		fmt.Println("  atlas extract --once --enqueue            Enqueue + process with real LLM")
		fmt.Println("  atlas extract --once --enqueue --max 5    Enqueue then process 5 articles")
		fmt.Println("  atlas extract --daemon                    Run as continuous daemon")
		fmt.Println("  atlas extract --mock --article-id <uuid>  Extract a single article directly")
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

	provider, err := extract.New(extract.ProviderConfig{
		Mock:     *mock,
		Provider: cfg.LLMProvider,
		APIKey:   cfg.LLMAPIKey,
		Model:    cfg.LLMModel,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Single article mode — bypass job queue entirely.
	if *articleID != "" {
		extractSingleArticle(ctx, pool, provider, *articleID, cfg.AnthropicAPIKey, cfg.AnthropicModel)
		return
	}

	svc := extract.NewService(pool, provider, cfg.AnthropicAPIKey, cfg.AnthropicModel)

	if *enqueue {
		n, err := svc.Enqueue(ctx, *priority)
		if err != nil {
			log.Fatalf("enqueue: %v", err)
		}
		if n == 0 {
			fmt.Println("No new articles to enqueue.")
		} else {
			fmt.Printf("Enqueued %d articles.\n", n)
		}
	}

	if *daemon {
		fmt.Printf("Starting daemon mode — poll interval %s\n", *interval)
		svc.RunDaemon(ctx, *interval)
	} else if *once {
		processed, err := svc.RunAll(ctx, *max)
		if err != nil {
			log.Fatalf("run: %v", err)
		}
		fmt.Printf("Processed %d jobs.\n", processed)
	} else {
		processed, err := svc.RunAll(ctx, *max)
		if err != nil {
			log.Fatalf("run: %v", err)
		}
		fmt.Printf("Processed %d jobs.\n", processed)
	}
}

// extractSingleArticle processes a single article directly without the job queue.
func extractSingleArticle(ctx context.Context, pool *pgxpool.Pool, provider extract.Provider, articleID, apiKey, model string) {
	fmt.Printf("Extracting article %s...\n", shortID(articleID))

	store := extract.NewJobStore(pool)
	article, err := store.LoadArticleData(ctx, articleID)
	if err != nil {
		log.Fatalf("load article: %v", err)
	}

	// First attempt.
	k, err := provider.Analyze(*article)
	if err != nil {
		log.Fatalf("extract: %v", err)
	}

	// Validate.
	if err := extract.ValidateKnowledge(k); err != nil {
		fmt.Printf("Validation failed (attempt 1): %v\n", err)
		fmt.Println("Retrying...")

		k, err = provider.Analyze(*article)
		if err != nil {
			log.Fatalf("extract retry: %v", err)
		}

		if err := extract.ValidateKnowledge(k); err != nil {
			log.Fatalf("validation failed after retry: %v", err)
		}
	}

	// Resolve and insert into M2M junction tables.
	if err := extract.ResolveAndInsert(ctx, pool, articleID, k); err != nil {
		log.Fatalf("resolve and insert: %v", err)
	}

	// Save result.
	rawJSON, _ := json.Marshal(k)
	rawStr := string(rawJSON)
	providerName := "mock"
	modelName := "mock-v1"
	if apiKey != "" {
		providerName = "anthropic"
		modelName = model
	}
	result := &extract.ExtractionResult{
		ArticleID:   articleID,
		RawJSON:     &rawStr,
		LLMProvider: &providerName,
		LLMModel:    &modelName,
	}
	if err := store.SaveResult(ctx, result); err != nil {
		log.Fatalf("save result: %v", err)
	}

	fmt.Println("Extraction complete.")
	printExtractionResult(k, providerName, modelName)
}

func printExtractionResult(k *extract.Knowledge, provider, model string) {
	fmt.Println()
	fmt.Println("━━━ Extracted Entities ━━━")
	fmt.Printf("Provider: %s  Model: %s\n", provider, model)
	fmt.Println()

	printEntityList("Topics", k.Topics)
	printEntityList("Industries", k.Industries)
	printEntityList("Countries", k.Countries)
	printEntityList("Technologies", k.Technologies)
	printEntityList("Organizations", k.Organizations)
}

func printEntityList(label string, ents []extract.Entity) {
	if len(ents) == 0 {
		fmt.Printf("  %s: (none)\n", label)
		return
	}
	fmt.Printf("  %s:\n", label)
	for _, e := range ents {
		fmt.Printf("    - %s  (%.0f%%)\n", e.Code, e.Confidence*100)
	}
	fmt.Println()
}