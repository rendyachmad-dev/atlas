package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"atlas/configs"
	"atlas/internal/database"
	"atlas/internal/intelligence"
)

func runAnalyze() {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	once := fs.Bool("once", false, "Process available jobs once and exit")
	enqueue := fs.Bool("enqueue", false, "Enqueue new articles before processing")
	max := fs.Int("max", 0, "Max jobs to process (0 = all available)")
	mock := fs.Bool("mock", false, "Use MockProvider instead of Anthropic API")
	priority := fs.Int("priority", 5, "Priority for enqueued jobs")
	daemon := fs.Bool("daemon", false, "Run in continuous daemon mode")
	interval := fs.Duration("interval", 60*time.Second, "Poll interval in daemon mode")
	help := fs.Bool("help", false, "Show help")

	fs.Usage = func() {
		fmt.Println("Usage: atlas analyze [flags]")
		fmt.Println()
		fmt.Println("Analyze articles via LLM (Anthropic Claude) or MockProvider.")
		fmt.Println()
		fmt.Println("Flags:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  atlas analyze --once --mock              Process all queued jobs with mock data")
		fmt.Println("  atlas analyze --once --enqueue           Enqueue + process with real LLM")
		fmt.Println("  atlas analyze --once --enqueue --max 5   Enqueue then process 5 articles")
		fmt.Println("  atlas analyze --daemon                   Run as continuous daemon")
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

	var provider intelligence.IntelligenceProvider
	if *mock {
		provider = intelligence.NewMockProvider(true)
		fmt.Println("Using MockProvider (deterministic fake data)")
	} else {
		if cfg.AnthropicAPIKey == "" {
			log.Fatal("ANTHROPIC_API_KEY not set. Use --mock to use MockProvider, or set ANTHROPIC_API_KEY env var.")
		}
		provider = intelligence.NewAnthropicProvider(cfg.AnthropicAPIKey, cfg.AnthropicModel)
		fmt.Printf("Using AnthropicProvider (model=%s)\n", cfg.AnthropicModel)
	}

	svc := intelligence.NewService(pool, provider)

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
		// Default: process once
		processed, err := svc.RunAll(ctx, *max)
		if err != nil {
			log.Fatalf("run: %v", err)
		}
		fmt.Printf("Processed %d jobs.\n", processed)
	}
}