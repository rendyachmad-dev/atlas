package extract

import (
	"encoding/json"
	"fmt"
	"math/rand"
)

// Provider defines the interface for extracting taxonomy entities from articles.
type Provider interface {
	Analyze(article Article) (*Knowledge, error)
}

// ProviderConfig holds configuration for creating a Provider.
type ProviderConfig struct {
	Mock   bool
	APIKey string
	Model  string
}

// New creates a Provider based on config.
// Returns AnthropicExtractor by default, MockProvider when cfg.Mock is true.
// Errors if neither mock nor API key is set.
func New(cfg ProviderConfig) (Provider, error) {
	if cfg.Mock {
		return NewMockProvider(true), nil
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set — use MockProvider or set ANTHROPIC_API_KEY env var")
	}
	return NewAnthropicExtractor(cfg.APIKey, cfg.Model), nil
}

// MockProvider returns deterministic fake extraction data.
// Used for testing the pipeline without calling any LLM.
type MockProvider struct {
	deterministic bool
}

// NewMockProvider creates a new MockProvider.
func NewMockProvider(deterministic bool) *MockProvider {
	return &MockProvider{deterministic: deterministic}
}

// known codes that exist in the taxonomy seed data
var (
	mockTopicCodes = []string{
		"machine-learning", "fintech", "cybersecurity", "renewable-energy",
		"biotechnology", "cloud-computing", "ai-regulation", "supply-chain-mfg",
		"generative-ai", "blockchain", "healthtech", "edtech",
	}

	mockIndustryCodes = []string{
		"tech", "finance", "healthcare", "energy", "manufacturing", "agriculture",
	}

	mockCountryCodes = []string{
		"US", "CN", "ID", "SG", "JP", "DE", "GB", "IN",
	}

	mockTechnologyCodes = []string{
		"ai", "cloud", "blockchain", "cybersecurity", "iot", "fintech",
	}

	mockOrgCodes = []string{
		"microsoft", "google", "openai", "anthropic",
		"bank-mandiri", "goto", "united-nations",
	}
)

// Analyze returns deterministic fake taxonomy entities for the given article.
func (p *MockProvider) Analyze(article Article) (*Knowledge, error) {
	var seed int64
	if p.deterministic {
		h := 0
		for _, c := range article.ID + article.Title {
			h = h*31 + int(c)
		}
		seed = int64(h)
	} else {
		seed = rand.Int63()
	}
	rng := rand.New(rand.NewSource(seed))

	n := 3 // entities per category
	entities := &Knowledge{
		Topics:        pickMockEntities(rng, mockTopicCodes, n),
		Industries:    pickMockEntities(rng, mockIndustryCodes, 2),
		Countries:     pickMockEntities(rng, mockCountryCodes, 2),
		Technologies:  pickMockEntities(rng, mockTechnologyCodes, 2),
		Organizations: pickMockEntities(rng, mockOrgCodes, 2),
	}

	return entities, nil
}

func pickMockEntities(rng *rand.Rand, codes []string, n int) []Entity {
	if n > len(codes) {
		n = len(codes)
	}
	perm := rng.Perm(len(codes))
	entities := make([]Entity, n)
	for i := 0; i < n; i++ {
		entities[i] = Entity{
			Code:       codes[perm[i]],
			Confidence: float64(60+rng.Intn(35)) / 100.0,
		}
	}
	return entities
}

// Ensure MockProvider implements Provider.
var _ Provider = (*MockProvider)(nil)

// MockResultJSON returns the JSON representation of mock extraction data.
// Used for testing.
func MockResultJSON(k *Knowledge) string {
	b, _ := json.Marshal(k)
	return string(b)
}