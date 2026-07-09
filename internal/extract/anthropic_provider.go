package extract

import (
	"context"
	"fmt"
)

// AnthropicExtractor calls the Anthropic Claude API to extract taxonomy entities.
type AnthropicExtractor struct {
	apiKey string
	model  string
}

// NewAnthropicExtractor creates a new AnthropicExtractor.
func NewAnthropicExtractor(apiKey, model string) *AnthropicExtractor {
	if model == "" {
		model = "claude-sonnet-5"
	}
	return &AnthropicExtractor{
		apiKey: apiKey,
		model:  model,
	}
}

// Analyze sends article content to Anthropic Claude and returns extracted knowledge.
func (p *AnthropicExtractor) Analyze(article Article) (*Knowledge, error) {
	entities, err := callLLM(context.Background(), p.apiKey, p.model, article)
	if err != nil {
		return nil, fmt.Errorf("anthropic analyze: %w", err)
	}
	return entities, nil
}

// Ensure AnthropicExtractor implements Provider.
var _ Provider = (*AnthropicExtractor)(nil)