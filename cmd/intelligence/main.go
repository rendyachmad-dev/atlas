package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"atlas/configs"
	"atlas/internal/database"
	"atlas/internal/intelligence"
)

func main() {
	enqueue := flag.Bool("enqueue", false, "Enqueue articles before processing")
	priority := flag.Int("priority", 5, "Priority for enqueued jobs")
	max := flag.Int("max", 0, "Max jobs to process (0 = all available)")
	help := flag.Bool("help", false, "Show help")

	flag.Usage = func() {
		fmt.Println("Usage: atlas intelligence run [flags]")
		fmt.Println()
		fmt.Println("Process analysis jobs using MockProvider.")
		fmt.Println()
		fmt.Println("Flags:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  atlas intelligence run                      Process all queued jobs")
		fmt.Println("  atlas intelligence run -enqueue             Enqueue + process all")
		fmt.Println("  atlas intelligence run -max 5               Process 5 jobs")
		fmt.Println("  atlas intelligence run -enqueue -priority 1 Enqueue with low priority")
	}

	flag.Parse()

	if *help || (len(os.Args) > 1 && os.Args[1] == "help") {
		flag.Usage()
		return
	}

	ctx := context.Background()
	cfg := configs.Load()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	provider := intelligence.NewMockProvider(true)
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

	processed, err := svc.RunAll(ctx, *max)
	if err != nil {
		log.Fatalf("run: %v", err)
	}

	fmt.Printf("Processed %d jobs.\n", processed)
}