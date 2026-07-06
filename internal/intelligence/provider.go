package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
)

// IntelligenceProvider defines the interface for analyzing articles.
// Implementations can be mock, OpenAI, Anthropic, Gemini, etc.
type IntelligenceProvider interface {
	// Analyze analyzes an article and returns structured analysis data.
	Analyze(ctx context.Context, article ArticleData) (*AnalysisResult, error)
}

// MockProvider returns deterministic fake analysis data.
// Used for testing the pipeline without calling any LLM.
type MockProvider struct {
	// seedSource is used to derive deterministic results per article.
	// When true, same article always returns same result.
	deterministic bool
}

// NewMockProvider creates a new MockProvider.
func NewMockProvider(deterministic bool) *MockProvider {
	return &MockProvider{deterministic: deterministic}
}

// Analyze returns deterministic fake analysis for the given article.
func (p *MockProvider) Analyze(ctx context.Context, article ArticleData) (*AnalysisResult, error) {
	// Derive seed from article data for deterministic results.
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

	data := MockAnalysisData{
		Summary:         p.pickSummary(rng, article),
		BusinessImpact:  p.pickBusinessImpact(rng),
		Urgency:         rng.Intn(10) + 1,
		Confidence:      float32(60+rng.Intn(35)) / 100.0,
		Reasoning:       fmt.Sprintf("Analysis based on article content from %s. Title: %q.", article.Source, article.Title),
		Sentiment:       p.pickSentiment(rng),
		TimeHorizon:     p.pickTimeHorizon(rng),
		RiskScore:       rng.Intn(10) + 1,
		InnovationScore: rng.Intn(10) + 1,
		MarketSizeScore: rng.Intn(10) + 1,
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("mock marshal: %w", err)
	}
	rawStr := string(raw)

	summary := data.Summary
	impact := data.BusinessImpact
	urgency := data.Urgency
	confidence := data.Confidence
	reasoning := data.Reasoning
	provider := "mock"
	model := "mock-v1"

	return &AnalysisResult{
		ArticleID:      article.ID,
		Summary:        &summary,
		BusinessImpact: &impact,
		Urgency:        &urgency,
		Confidence:     &confidence,
		Reasoning:      &reasoning,
		RawJSON:        &rawStr,
		LLMProvider:    &provider,
		LLMModel:       &model,
	}, nil
}

func (p *MockProvider) pickSummary(rng *rand.Rand, article ArticleData) string {
	summaries := []string{
		"Market shifting. Opportunity in fintech.",
		"AI regulation tightens. Compliance costs rise.",
		"New tech disrupts incumbents.",
		"Consumer demand growing. Supply chain needs optimization.",
		"Geopolitical risk creates volatility.",
		"Investment surge indicates sector confidence.",
		"Innovation gap creates competitive advantage.",
		"Cost pressures drive efficiency adoption.",
		"Regulatory change opens new segments.",
		"Partnership signals consolidation trend.",
	}
	return summaries[rng.Intn(len(summaries))]
}

func (p *MockProvider) pickBusinessImpact(rng *rand.Rand) string {
	impacts := []string{"Low", "Medium", "High"}
	return impacts[rng.Intn(len(impacts))]
}

func (p *MockProvider) pickSentiment(rng *rand.Rand) string {
	sentiments := []string{"positive", "negative", "neutral", "mixed"}
	return sentiments[rng.Intn(len(sentiments))]
}

func (p *MockProvider) pickTimeHorizon(rng *rand.Rand) string {
	horizons := []string{"short-term", "mid-term", "long-term"}
	return horizons[rng.Intn(len(horizons))]
}