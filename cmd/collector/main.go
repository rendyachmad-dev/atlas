package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"atlas/configs"
	"atlas/internal/database"
	"atlas/internal/logger"
	"atlas/internal/scheduler"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := configs.Load()
	logger.Setup(cfg.LogLevel)

	log.Printf("starting Atlas RSS Collector — log_level=%s port=%s", cfg.LogLevel, cfg.Port)

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()
	log.Println("database connected")

	if err := database.RunMigrations(ctx, pool); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	log.Println("Atlas RSS Collector — ready")

	// Start scheduler in background.
	go scheduler.Run(ctx, pool, 60*time.Second)

	// Wait for SIGINT or SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("signal received: %s — shutting down", sig)
	cancel()

	time.Sleep(500 * time.Millisecond)
	log.Println("shutdown complete")
}