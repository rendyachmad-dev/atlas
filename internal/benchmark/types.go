package benchmark

import (
	"time"

	"atlas/internal/extract"
)

// ProviderRun configures one provider to benchmark.
type ProviderRun struct {
	Label    string // display name, e.g. "Groq (llama-3.3-70b)"
	Provider string // "mock", "anthropic", "groq"
	APIKey   string
	Model    string
	Mock     bool // shortcut for testing
}

// Result holds the evaluation result and metadata for one provider.
type Result struct {
	Label   string
	Config  ProviderRun
	Result  *evaluateResult
	Error   string
	Elapsed time.Duration
}

// CompareRow is one row in the comparison table.
type CompareRow struct {
	Label             string
	Provider          string
	Model             string
	OverallAccuracy   float64
	JSONValidRate     float64
	AvgProcessingTime time.Duration
	TotalTime         time.Duration
	// Per-category accuracy
	TopicAccuracy        float64
	IndustryAccuracy     float64
	CountryAccuracy      float64
	TechnologyAccuracy   float64
	OrganizationAccuracy float64
	// Per-category P/R/F1
	Categories map[string]CategoryMetrics
}

// Report is the full benchmark output.
type Report struct {
	DatasetName    string
	DatasetPath    string
	TotalRecords   int
	Runs           []Result
	CompareRows    []CompareRow
	RankedByAccuracy []CompareRow
	RankedBySpeed    []CompareRow
	Recommendations Recommendations
}

// Recommendations provides guidance on which provider to use for what.
type Recommendations struct {
	BestAccuracy string
	BestSpeed    string
	BestCost     string
	BestOverall  string
}

// evaluateResult is a simplified view of evaluate.EvalResult for the benchmark.
type evaluateResult struct {
	JSONValidationRate   float64
	AvgProcessingTime    time.Duration
	TotalProcessingTime  time.Duration
	IndustryAccuracy     float64
	TopicAccuracy        float64
	CountryAccuracy      float64
	TechnologyAccuracy   float64
	OrganizationAccuracy float64
	OverallAccuracy      float64
	Categories           map[string]CategoryMetrics
}

// CategoryMetrics mirrors evaluate.CategoryMetrics.
type CategoryMetrics struct {
	Precision float64
	Recall    float64
	F1        float64
	TP        int
	FP        int
	FN        int
}

// ProviderSet defines a named set of providers to benchmark.
// Pre-defined sets can be used for convenience.
type ProviderSet struct {
	Name string
	Runs []ProviderRun
}

// NewProviderSet creates a provider set.
func NewProviderSet(name string, runs []ProviderRun) ProviderSet {
	return ProviderSet{Name: name, Runs: runs}
}

// DefaultProviders returns a recommended set of providers to benchmark.
// Uses environment variables (LLM_API_KEY, LLM_PROVIDER) for credentials.
func DefaultProviders(apiKey, model string) []ProviderRun {
	return []ProviderRun{
		{
			Label:    "Mock",
			Mock:     true,
			Provider: "mock",
		},
		{
			Label:    "Groq (llama-3.3-70b)",
			Provider: "groq",
			APIKey:   apiKey,
			Model:    "llama-3.3-70b-versatile",
		},
		{
			Label:    "Groq (mixtral-8x7b)",
			Provider: "groq",
			APIKey:   apiKey,
			Model:    "mixtral-8x7b-32768",
		},
		{
			Label:    "Groq (gemma2-9b)",
			Provider: "groq",
			APIKey:   apiKey,
			Model:    "gemma2-9b-it",
		},
	}
}

// CreateProvider creates an extract.Provider from a ProviderRun config.
// Uses extract.New() internally for consistency with the rest of the system.
func CreateProvider(cfg ProviderRun) (extract.Provider, error) {
	return extract.New(extract.ProviderConfig{
		Mock:     cfg.Mock,
		Provider: cfg.Provider,
		APIKey:   cfg.APIKey,
		Model:    cfg.Model,
	})
}