package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"atlas/configs"
	"atlas/internal/benchmark"
	"atlas/internal/evaluate"
)

func runBenchmark() {
	fs := flag.NewFlagSet("benchmark", flag.ExitOnError)
	dataset := fs.String("dataset", "testdata/golden-sample.json", "Path to golden dataset JSON file")
	providers := fs.String("providers", "mock,groq", "Comma-separated list of providers to benchmark (mock, anthropic, groq)")
	models := fs.String("models", "", "Comma-separated models per provider (overrides defaults)")
	format := fs.String("format", "table", "Output format: table")
	help := fs.Bool("help", false, "Show help")

	fs.Usage = func() {
		fmt.Println("Usage: atlas benchmark [flags]")
		fmt.Println()
		fmt.Println("Benchmark multiple LLM providers against a golden dataset.")
		fmt.Println("Reuses the evaluation framework to compare accuracy, speed, and reliability.")
		fmt.Println()
		fmt.Println("Flags:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  atlas benchmark                                      Benchmark mock + groq (default)")
		fmt.Println("  atlas benchmark --providers mock,groq,anthropic     Benchmark all providers")
		fmt.Println("  atlas benchmark --providers groq                    Benchmark groq only")
		fmt.Println("  atlas benchmark --models llama-3.3-70b-versatile    Single model override")
	}

	fs.Parse(os.Args[2:])

	if *help {
		fs.Usage()
		return
	}

	cfg := configs.Load()

	// Load dataset.
	ds, err := evaluate.LoadDataset(*dataset)
	if err != nil {
		log.Fatalf("load dataset: %v", err)
	}
	fmt.Printf("Loaded dataset %q — %d records\n", ds.Name, len(ds.Records))

	// Build provider runs from flags.
	providerList := parseList(*providers)
	modelList := parseList(*models)
	runs := buildProviderRuns(providerList, modelList, cfg)

	if len(runs) == 0 {
		log.Fatal("no providers to benchmark — use --providers flag")
	}

	fmt.Printf("Benchmarking %d providers: %s\n", len(runs), joinLabels(runs))
	fmt.Println()

	// Run benchmark.
	ctx := context.Background()
	runner := benchmark.NewRunner(ds, *dataset)
	report, err := runner.Run(ctx, runs)
	if err != nil {
		log.Fatalf("benchmark: %v", err)
	}

	// Output.
	switch *format {
	default:
		fmt.Print(benchmark.ReportText(report))
	}
}

// buildProviderRuns creates provider configs from user-specified lists.
func buildProviderRuns(providers, models []string, cfg *configs.Config) []benchmark.ProviderRun {
	var runs []benchmark.ProviderRun

	for _, name := range providers {
		switch name {
		case "mock":
			runs = append(runs, benchmark.ProviderRun{
				Label:    "Mock",
				Provider: "mock",
				Mock:     true,
			})

		case "anthropic":
			apiKey := cfg.LLMAPIKey
			if apiKey == "" {
				log.Printf("WARNING: skipping anthropic — LLM_API_KEY not set")
				continue
			}
			model := pickModel(name, models, "claude-sonnet-5")
			runs = append(runs, benchmark.ProviderRun{
				Label:    fmt.Sprintf("Anthropic (%s)", model),
				Provider: "anthropic",
				APIKey:   apiKey,
				Model:    model,
			})

		case "groq":
			apiKey := cfg.LLMAPIKey
			if apiKey == "" {
				log.Printf("WARNING: skipping groq — LLM_API_KEY not set")
				continue
			}
			models := []string{"llama-3.3-70b-versatile", "mixtral-8x7b-32768", "gemma2-9b-it"}
			if len(models) > 0 {
				models = strings.Split(pickModel(name, models, ""), ",")
				// If user specified models, use them directly
				if len(models) == 1 && models[0] != "" {
					runs = append(runs, benchmark.ProviderRun{
						Label:    fmt.Sprintf("Groq (%s)", models[0]),
						Provider: "groq",
						APIKey:   apiKey,
						Model:    models[0],
					})
					continue
				}
			}
			// Default: run all 3 Groq models.
			for _, m := range []string{"llama-3.3-70b-versatile", "mixtral-8x7b-32768", "gemma2-9b-it"} {
				runs = append(runs, benchmark.ProviderRun{
					Label:    fmt.Sprintf("Groq (%s)", m),
					Provider: "groq",
					APIKey:   apiKey,
					Model:    m,
				})
			}

		default:
			log.Printf("WARNING: unknown provider %q — skipping", name)
		}
	}

	return runs
}

// pickModel returns the model for a given provider from the user-specified models list.
func pickModel(provider string, models []string, fallback string) string {
	if len(models) == 0 {
		return fallback
	}
	// If only one model specified, use it for all providers.
	if len(models) == 1 {
		return models[0]
	}
	return fallback
}

func parseList(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func joinLabels(runs []benchmark.ProviderRun) string {
	labels := make([]string, len(runs))
	for i, r := range runs {
		labels[i] = r.Label
	}
	return strings.Join(labels, ", ")
}