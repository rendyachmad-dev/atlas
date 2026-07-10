package extract

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GroqExtractor calls the Groq API (OpenAI-compatible) to extract taxonomy entities.
type GroqExtractor struct {
	apiKey string
	model  string
}

// NewGroqExtractor creates a new GroqExtractor.
// Default model is llama-3.3-70b-versatile (fast, good at structured output).
func NewGroqExtractor(apiKey, model string) *GroqExtractor {
	if model == "" {
		model = "llama-3.3-70b-versatile"
	}
	return &GroqExtractor{
		apiKey: apiKey,
		model:  model,
	}
}

// Analyze sends article content to Groq API and returns extracted knowledge.
func (p *GroqExtractor) Analyze(article Article) (*Knowledge, error) {
	entities, err := callGroqLLM(context.Background(), p.apiKey, p.model, article)
	if err != nil {
		return nil, fmt.Errorf("groq analyze: %w", err)
	}
	return entities, nil
}

// callGroqLLM sends article content to the Groq API (OpenAI-compatible) and returns extracted knowledge.
func callGroqLLM(ctx context.Context, apiKey, model string, article Article) (*Knowledge, error) {
	userContent := fmt.Sprintf("Title: %s\nSource: %s\nAuthor: %s\n\nContent:\n%s",
		article.Title, article.Source, nullableStr(article.Author), truncateContent(article.Content, 8000))

	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt()},
			{"role": "user", "content": userContent},
		},
		"max_tokens":  2048,
		"temperature": 0.1, // low temp for consistent structured output
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
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

	// OpenAI-compatible response format
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response choices")
	}

	text := apiResp.Choices[0].Message.Content
	var k Knowledge
	if err := json.Unmarshal([]byte(text), &k); err != nil {
		return nil, fmt.Errorf("parse LLM JSON: %w (raw: %s)", err, truncateContent(text, 200))
	}

	NormalizeKnowledge(&k)

	return &k, nil
}

// Ensure GroqExtractor implements Provider.
var _ Provider = (*GroqExtractor)(nil)