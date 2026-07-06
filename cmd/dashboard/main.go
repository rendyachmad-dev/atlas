package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	"atlas/configs"
	"atlas/internal/article"
	"atlas/internal/database"
	"atlas/internal/source"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()
	cfg := configs.Load()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "sources":
			printSources(ctx, pool)
		case "logs":
			limit := 20
			if len(os.Args) > 2 {
				fmt.Sscanf(os.Args[2], "%d", &limit)
			}
			printLogs(ctx, pool, limit)
		case "help", "-h", "--help":
			printHelp()
		default:
			fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", os.Args[1])
			printHelp()
			os.Exit(1)
		}
		return
	}

	// Default: full dashboard
	printDashboard(ctx, pool)
}

func printHelp() {
	fmt.Println("Usage: dashboard [subcommand]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  (no args)   Show full dashboard")
	fmt.Println("  sources     Show sources table")
	fmt.Println("  logs [N]    Show N most recent fetch logs (default 20)")
	fmt.Println("  help        Show this help")
}

func printDashboard(ctx context.Context, pool *pgxpool.Pool) {
	stats, err := source.GetDashboardStats(ctx, pool)
	if err != nil {
		log.Fatalf("stats: %v", err)
	}

	articles, err := article.CountArticlesBySource(ctx, pool)
	if err != nil {
		log.Fatalf("articles: %v", err)
	}

	logs, err := article.ListRecentFetchLogs(ctx, pool, 10)
	if err != nil {
		log.Fatalf("logs: %v", err)
	}

	printHeader("Atlas RSS Collector — Dashboard")
	fmt.Println()

	// System stats
	printHeader("System Stats")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "Sources:\t%d total\t%d active\n", stats.TotalSources, stats.ActiveSources)
	fmt.Fprintf(w, "Health:\t%d healthy\t%d degraded\t%d down\n",
		stats.HealthySources, stats.DegradedSources, stats.DownSources)
	fmt.Fprintf(w, "Articles:\t%d\n", stats.TotalArticles)
	fmt.Fprintf(w, "Fetch Logs:\t%d\n", stats.TotalFetchLogs)
	if stats.LatestFetchAt != nil {
		fmt.Fprintf(w, "Last Fetch:\t%s\n", stats.LatestFetchAt.Format(time.RFC3339))
	}
	w.Flush()
	fmt.Println()

	// Sources table
	printHeader("Sources")
	w = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "NAME\tTYPE\tHEALTH\tFAILURES\tARTICLES\tLAST FETCH\n")
	fmt.Fprintf(w, "----\t----\t------\t--------\t--------\t----------\n")

	sources, err := source.ListActive(ctx, pool)
	if err != nil {
		log.Fatalf("list sources: %v", err)
	}

	articleMap := make(map[string]article.SourceArticleCount)
	for _, a := range articles {
		articleMap[a.SourceID] = a
	}

	for _, s := range sources {
		healthColor := healthColor(s.HealthStatus)
		lastFetch := "-"
		if s.LastFetch != nil {
			lastFetch = s.LastFetch.Format("15:04 Jan 02")
		}
		ac := articleMap[s.ID]
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%s\n",
			s.Name, s.Type, healthColor, s.ConsecutiveFailures, ac.Total, lastFetch)
	}
	w.Flush()
	fmt.Println()

	// Recent logs
	printHeader("Recent Fetch Logs")
	w = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "SOURCE\tSTATUS\tFETCHED\tNEW\tTIME\tERROR\n")
	fmt.Fprintf(w, "------\t------\t-------\t---\t----\t-----\n")
	for _, l := range logs {
		errMsg := ""
		if l.ErrorMessage != nil {
			errMsg = truncate(*l.ErrorMessage, 40)
		}
		statusColor := l.Status
		if l.Status == "failed" {
			statusColor = "✗ " + l.Status
		} else if l.Status == "success" {
			statusColor = "✓ " + l.Status
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\t%s\n",
			l.SourceName, statusColor, l.ArticlesFetched, l.ArticlesNew,
			l.StartedAt.Format("15:04 Jan 02"), errMsg)
	}
	w.Flush()
	fmt.Println()

	fmt.Println("Subcommands: dashboard sources | logs [N]")
}

func printSources(ctx context.Context, pool *pgxpool.Pool) {
	sources, err := source.ListActive(ctx, pool)
	if err != nil {
		log.Fatalf("list sources: %v", err)
	}

	articles, err := article.CountArticlesBySource(ctx, pool)
	if err != nil {
		log.Fatalf("articles: %v", err)
	}

	articleMap := make(map[string]article.SourceArticleCount)
	for _, a := range articles {
		articleMap[a.SourceID] = a
	}

	printHeader("Sources")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "NAME\tTYPE\tHEALTH\tFAILURES\tURL\tARTICLES\tLAST FETCH\n")
	fmt.Fprintf(w, "----\t----\t------\t--------\t---\t--------\t----------\n")
	for _, s := range sources {
		lastFetch := "-"
		if s.LastFetch != nil {
			lastFetch = s.LastFetch.Format(time.RFC3339)
		}
		ac := articleMap[s.ID]
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%d\t%s\n",
			s.Name, s.Type, s.HealthStatus, s.ConsecutiveFailures,
			truncate(s.URL, 50), ac.Total, lastFetch)
	}
	w.Flush()
}

func printLogs(ctx context.Context, pool *pgxpool.Pool, limit int) {
	logs, err := article.ListRecentFetchLogs(ctx, pool, limit)
	if err != nil {
		log.Fatalf("logs: %v", err)
	}

	printHeader(fmt.Sprintf("Recent Fetch Logs (last %d)", limit))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "SOURCE\tSTATUS\tFETCHED\tNEW\tSTARTED\tCOMPLETED\tERROR\n")
	fmt.Fprintf(w, "------\t------\t-------\t---\t-------\t---------\t-----\n")

	for _, l := range logs {
		completed := "-"
		if l.CompletedAt != nil {
			completed = l.CompletedAt.Format("15:04:05 Jan 02")
		}
		errMsg := ""
		if l.ErrorMessage != nil {
			errMsg = truncate(*l.ErrorMessage, 50)
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\t%s\t%s\n",
			l.SourceName, l.Status, l.ArticlesFetched, l.ArticlesNew,
			l.StartedAt.Format("15:04:05 Jan 02"), completed, errMsg)
	}
	w.Flush()
}

func printHeader(title string) {
	fmt.Println("━━━ " + title + " ━━━")
}

func healthColor(status string) string {
	switch status {
	case "healthy":
		return "✓ healthy"
	case "degraded":
		return "⚠ degraded"
	case "down":
		return "✗ down"
	default:
		return status
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}