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