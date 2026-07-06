package intelligence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AnthropicProvider analyzes articles using the Anthropic Claude API.
type AnthropicProvider struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// anthropicMessage represents a message in the Anthropic Messages API.
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicRequest is the request body for the Anthropic Messages API.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

// anthropicContentBlock holds the text content from the API response.
type anthropicContentBlock struct {
	Text string `json:"text"`
}

// anthropicResponse is the response from the Anthropic Messages API.
type anthropicResponse struct {
	ID      string                 `json:"id"`
	Model   string                 `json:"model"`
	Content []anthropicContentBlock `json:"content"`
	StopReason string              `json:"stop_reason"`
	Usage    struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// AnthropicResponseData holds the structured data returned by the LLM.
type AnthropicResponseData struct {
	Summary         string  `json:"summary"`
	BusinessImpact  string  `json:"business_impact"`
	Urgency         int     `json:"urgency"`
	Confidence      float32 `json:"confidence"`
	Reasoning       string  `json:"reasoning"`
	Sentiment       string  `json:"sentiment"`
	TimeHorizon     string  `json:"time_horizon"`
	RiskScore       int     `json:"risk_score"`
	InnovationScore int     `json:"innovation_score"`
	MarketSizeScore int     `json:"market_size_score"`
}

const anthropicAPIURL = "https://api.anthropic.com/v1/messages"
const anthropicAPIVersion = "2023-06-01"
const maxTokens = 1024

// NewAnthropicProvider creates a new AnthropicProvider.
func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	if model == "" {
		model = "claude-sonnet-5"
	}
	return &AnthropicProvider{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Analyze sends article content to the Anthropic Claude API and returns structured analysis.
func (p *AnthropicProvider) Analyze(ctx context.Context, article ArticleData) (*AnalysisResult, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Anthropic API key is empty")
	}

	systemPrompt := `You are an AI analyst that extracts structured business intelligence from news articles.
Analyze the article and return a JSON object with these fields:
- summary: a concise one-sentence summary of the article's key point
- business_impact: one of "Low", "Medium", "High" — how much this affects business strategy
- urgency: integer 1-10 (1=not urgent, 10=immediate action required)
- confidence: float 0.0-1.0 (how confident you are in this analysis)
- reasoning: a brief explanation of your analysis logic
- sentiment: one of "positive", "negative", "neutral", "mixed"
- time_horizon: one of "short-term", "mid-term", "long-term"
- risk_score: integer 1-10
- innovation_score: integer 1-10
- market_size_score: integer 1-10

Return ONLY valid JSON, no markdown formatting, no explanation outside the JSON.`

	userContent := fmt.Sprintf("Title: %s\nSource: %s\nAuthor: %s\n\nContent:\n%s",
		article.Title, article.Source,
		formatOptional(article.Author),
		article.Content)

	reqBody := anthropicRequest{
		Model:     p.model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: userContent},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)
	req.Header.Set("content-type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response JSON: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty response content")
	}

	// Parse the structured JSON from the LLM response text.
	text := apiResp.Content[0].Text
	var data AnthropicResponseData
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return nil, fmt.Errorf("parse LLM output JSON: %w (raw: %s)", err, truncateString(text, 200))
	}

	rawBytes, _ := json.Marshal(data)
	rawStr := string(rawBytes)

	provider := "anthropic"
	model := p.model
	summary := data.Summary
	impact := data.BusinessImpact
	urgency := data.Urgency
	confidence := data.Confidence
	reasoning := data.Reasoning

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

func formatOptional(s *string) string {
	if s == nil {
		return "N/A"
	}
	return *s
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}