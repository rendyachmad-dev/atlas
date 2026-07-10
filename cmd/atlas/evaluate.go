package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"atlas/configs"
	"atlas/internal/evaluate"
	"atlas/internal/extract"
)

func runEvaluate() {
	fs := flag.NewFlagSet("evaluate", flag.ExitOnError)
	dataset := fs.String("dataset", "testdata/golden-sample.json", "Path to golden dataset JSON file")
	mock := fs.Bool("mock", true, "Use MockProvider instead of real LLM")
	format := fs.String("format", "table", "Output format: table, json")
	help := fs.Bool("help", false, "Show help")

	fs.Usage = func() {
		fmt.Println("Usage: atlas evaluate [flags]")
		fmt.Println()
		fmt.Println("Evaluate extraction pipeline quality against a labeled golden dataset.")
		fmt.Println()
		fmt.Println("Metrics: Industry/Topic/Country/Technology/Organization Accuracy,")
		fmt.Println("         Entity Precision/Recall/F1, JSON Validation Rate, Avg Processing Time.")
		fmt.Println()
		fmt.Println("Flags:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  atlas evaluate                                   Evaluate with MockProvider (default)")
		fmt.Println("  atlas evaluate --mock=false                     Evaluate with real Anthropic API")
		fmt.Println("  atlas evaluate --dataset testdata/custom.json   Evaluate with custom dataset")
		fmt.Println("  atlas evaluate --format json                    Output as JSON")
	}

	fs.Parse(os.Args[2:])

	if *help {
		fs.Usage()
		return
	}

	cfg := configs.Load()

	// Load golden dataset.
	ds, err := evaluate.LoadDataset(*dataset)
	if err != nil {
		log.Fatalf("load dataset: %v", err)
	}
	fmt.Printf("Loaded dataset %q — %d records\n", ds.Name, len(ds.Records))

	// Create provider.
	provider, err := extract.New(extract.ProviderConfig{
		Mock:     *mock,
		Provider: cfg.LLMProvider,
		APIKey:   cfg.LLMAPIKey,
		Model:    cfg.LLMModel,
	})
	if err != nil {
		log.Fatal(err)
	}

	model := cfg.LLMModel
	if providerName := providerName(provider); providerName == "mock" {
		model = "mock-v1"
	}
	fmt.Printf("Provider: %s  Model: %s\n\n", providerName(provider), model)

	// Run evaluation.
	result, err := evaluate.Evaluate(provider, ds, model)
	if err != nil {
		log.Fatalf("evaluate: %v", err)
	}

	// Output.
	switch *format {
	case "json":
		jsonStr, err := evaluate.JSONReport(result)
		if err != nil {
			log.Fatalf("json report: %v", err)
		}
		fmt.Println(jsonStr)
	default:
		fmt.Print(evaluate.Report(result))
	}
}

func providerName(p extract.Provider) string {
	switch p.(type) {
	case *extract.MockProvider:
		return "mock"
	case *extract.AnthropicExtractor:
		return "anthropic"
	case *extract.GroqExtractor:
		return "groq"
	default:
		return "unknown"
	}
}