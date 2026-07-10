package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"atlas/configs"
	"atlas/internal/database"
	"atlas/internal/repl"
)

func main() {
	if len(os.Args) < 2 {
		// No args → interactive REPL mode
		startREPL()
		return
	}

	switch os.Args[1] {
	case "fetch":
		runFetch()
	case "analyze":
		runAnalyze()
	case "extract":
		runExtract()
	case "evaluate":
		runEvaluate()
	case "detect":
		runDetect()
	case "classify":
		runClassify()
	case "source":
		runSource()
	case "status":
		runStatus()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func startREPL() {
	ctx := context.Background()
	cfg := configs.Load()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	repl.New(pool).Start()
}

func runSource() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: atlas source <subcommand>\n\n")
		fmt.Fprintf(os.Stderr, "Subcommands:\n")
		fmt.Fprintf(os.Stderr, "  discover [flags] <url>   Probe a domain for RSS feeds\n")
		fmt.Fprintf(os.Stderr, "  help                     Show this help\n")
		os.Exit(1)
	}

	switch os.Args[2] {
	case "discover":
		runSourceDiscover()
	case "help", "-h", "--help":
		fmt.Println("Usage: atlas source <subcommand>")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  discover [flags] <url>   Probe a domain for RSS feeds and score it")
		fmt.Println()
		fmt.Println("discover flags:")
		fmt.Println("  --batch <file>  Batch discover from a file of URLs")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  atlas source discover antaranews.com")
		fmt.Println("  atlas source discover --batch sources.txt")
	default:
		fmt.Fprintf(os.Stderr, "unknown source subcommand: %s\n\n", os.Args[2])
		os.Exit(1)
	}
}