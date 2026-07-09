package repl

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"atlas/internal/article"
	"atlas/internal/source"

	"github.com/jackc/pgx/v5/pgxpool"
)

// REPL handles the interactive prompt.
type REPL struct {
	pool *pgxpool.Pool
}

// New creates a new REPL with the given DB pool.
func New(pool *pgxpool.Pool) *REPL {
	return &REPL{pool: pool}
}

// Start displays the welcome banner and enters the interactive loop.
func (r *REPL) Start() {
	ctx := context.Background()

	r.printWelcome(ctx)
	r.loop(ctx)
}

func (r *REPL) printWelcome(ctx context.Context) {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = "User"
	}
	now := time.Now()
	wib := now.In(time.FixedZone("WIB", 7*60*60))
	hour := wib.Hour()
	weekday := wib.Weekday().String()
	day := wib.Day()
	month := wib.Month()
	year := wib.Year()

	greeting := "Good Evening"
	if hour < 12 {
		greeting = "Good Morning"
	} else if hour < 18 {
		greeting = "Good Afternoon"
	}

	// Load stats
	stats, _ := article.GetWelcomeStats(ctx, r.pool)
	highest, _ := article.GetHighestConfidence(ctx, r.pool)
	rec, _ := article.GetRecommendedAction(ctx, r.pool)

	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println("                 Atlas Intelligence Engine v0.2")
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("%s, %s.\n", greeting, username)
	fmt.Println("Welcome back.")
	fmt.Println()
	fmt.Printf("Computer : %s\n", hostname)
	fmt.Printf("Date     : %s, %d %s %d\n", weekday, day, month, year)
	fmt.Printf("Time     : %02d:%02d WIB\n", wib.Hour(), wib.Minute())
	fmt.Println()
	fmt.Println("───────────────────────────────────────────────────────────────")
	fmt.Println()

	if stats != nil {
		fmt.Printf("Yesterday Atlas analyzed : %d articles\n", stats.AnalyzedYesterday)
		fmt.Printf("Detected                : %d emerging trends\n", stats.EmergingTrends)
		fmt.Printf("Generated               : %d business opportunities\n", stats.Opportunities)
	} else {
		fmt.Println("Yesterday Atlas analyzed : 0 articles")
		fmt.Println("Detected                : 0 emerging trends")
		fmt.Println("Generated               : 0 business opportunities")
	}
	fmt.Println()
	fmt.Println("───────────────────────────────────────────────────────────────")
	fmt.Println()

	if highest != nil {
		fmt.Println("Highest Confidence")
		fmt.Println()
		title := truncateString(highest.Title, 30)
		fmt.Printf("%-36s %3.0f%%\n", title, highest.Confidence*100)
	}

	fmt.Println()
	fmt.Println("Recommended Action")
	fmt.Println()

	if rec != nil && rec.Summary != "" {
		fmt.Printf("Review %s. %s\n", rec.Title, rec.Summary)
	} else {
		fmt.Println("No analyzed articles yet. Run 'atlas analyze' first.")
	}

	fmt.Println()
	fmt.Println("───────────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("Type \"help\" to view available commands.")
	fmt.Println()
}

func (r *REPL) loop(ctx context.Context) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Atlas > ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		cmd := parts[0]
		args := parts[1:]

		switch cmd {
		case "exit", "quit", "q":
			fmt.Println("Goodbye.")
			return
		case "help", "?":
			r.printHelp()
		case "sources":
			r.printSources(ctx)
		case "logs":
			limit := 10
			if len(args) > 0 {
				fmt.Sscanf(args[0], "%d", &limit)
			}
			r.printLogs(ctx, limit)
		case "reports":
			limit := 5
			if len(args) > 0 {
				fmt.Sscanf(args[0], "%d", &limit)
			}
			r.printReports(ctx, limit)
		case "classify":
			if len(args) < 1 {
				fmt.Println("Usage: classify <article-uuid>")
				fmt.Println("       !atlas classify --mock <uuid>")
			} else {
				fmt.Printf("Run '!atlas classify --mock %s' in terminal\n", args[0])
			}
		case "status":
			r.printStatus(ctx)
		case "analyze":
			fmt.Println("Run 'atlas analyze --once --mock' in another terminal.")
		case "extract":
			fmt.Println("Run 'atlas extract --once --mock' in another terminal.")
		case "evaluate":
			fmt.Println("Run 'atlas evaluate --mock' in another terminal.")
		case "fetch":
			fmt.Println("Run 'atlas fetch' in another terminal.")
		case "clear", "cls":
			fmt.Print("\033[H\033[2J")
		default:
			fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", cmd)
		}
	}
}

func (r *REPL) printHelp() {
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  help                 Show this help")
	fmt.Println("  status               Show system dashboard")
	fmt.Println("  sources              List all sources")
	fmt.Println("  logs [N]             Show recent fetch logs (default 10)")
	fmt.Println("  reports [N]          Show intelligence reports (default 5)")
	fmt.Println("  classify <uuid>      Classify article (industry/tech/country/stakeholder)")
	fmt.Println("  evaluate             Evaluate extraction quality against golden dataset")
	fmt.Println("  extract              Run taxonomy extraction pipeline")
	fmt.Println("  analyze              Run article analysis pipeline")
	fmt.Println("  fetch                Run RSS collector")
	fmt.Println("  exit, quit, q        Exit")
	fmt.Println("  clear, cls           Clear screen")
	fmt.Println()
	fmt.Println("Tip: prefix '!' for shell commands, e.g. '!atlas source discover antaranews.com'")
	fmt.Println()
}

func (r *REPL) printStatus(ctx context.Context) {
	stats, err := source.GetDashboardStats(ctx, r.pool)
	if err != nil {
		log.Printf("stats: %v", err)
		return
	}

	fmt.Println()
	fmt.Println("━━━ Atlas Dashboard ━━━")
	fmt.Println()
	fmt.Printf("Sources: %d total, %d active\n", stats.TotalSources, stats.ActiveSources)
	fmt.Printf("Health: %d healthy, %d degraded, %d down\n",
		stats.HealthySources, stats.DegradedSources, stats.DownSources)
	fmt.Printf("Articles: %d\n", stats.TotalArticles)
	fmt.Printf("Fetch Logs: %d\n", stats.TotalFetchLogs)
	if stats.LatestFetchAt != nil {
		fmt.Printf("Last Fetch: %s\n", stats.LatestFetchAt.Format(time.RFC3339))
	}
	fmt.Println()
}

func (r *REPL) printSources(ctx context.Context) {
	sources, err := source.ListActive(ctx, r.pool)
	if err != nil {
		log.Printf("sources: %v", err)
		return
	}

	fmt.Println()
	fmt.Println("━━━ Sources ━━━")
	fmt.Println()
	for _, s := range sources {
		health := s.HealthStatus
		lastFetch := "-"
		if s.LastFetch != nil {
			lastFetch = s.LastFetch.Format("15:04 Jan 02")
		}
		fmt.Printf("  %s [%s] %s — last fetch: %s\n", s.Name, s.Type, health, lastFetch)
	}
	fmt.Println()
}

func (r *REPL) printLogs(ctx context.Context, limit int) {
	logs, err := article.ListRecentFetchLogs(ctx, r.pool, limit)
	if err != nil {
		log.Printf("logs: %v", err)
		return
	}

	fmt.Println()
	fmt.Println("━━━ Recent Fetch Logs ━━━")
	fmt.Println()
	for _, l := range logs {
		status := l.Status
		if status == "success" {
			status = "✓"
		} else if status == "failed" {
			status = "✗"
		}
		fmt.Printf("  %s %s — %d fetched, %d new [%s]\n",
			status, l.SourceName, l.ArticlesFetched, l.ArticlesNew,
			l.StartedAt.Format("15:04 Jan 02"))
	}
	fmt.Println()
}

func (r *REPL) printReports(ctx context.Context, limit int) {
	reports, err := article.ListReports(ctx, r.pool, limit)
	if err != nil {
		log.Printf("reports: %v", err)
		return
	}

	fmt.Println()
	fmt.Println("━━━ Intelligence Reports ━━━")
	fmt.Println()
	if len(reports) == 0 {
		fmt.Println("  No analyzed reports yet.")
		fmt.Println()
		return
	}
	for _, rep := range reports {
		urgency := "-"
		if rep.Urgency != nil {
			urgency = fmt.Sprintf("%d/10", *rep.Urgency)
		}
		summary := "-"
		if rep.SummaryShort != nil {
			summary = *rep.SummaryShort
		}
		fmt.Printf("  %s\n", rep.Title)
		fmt.Printf("  Urgency: %s | Source: %s\n", urgency, rep.SourceName)
		fmt.Printf("  %s\n", summary)
		fmt.Println()
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
