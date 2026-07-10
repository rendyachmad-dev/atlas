package extract

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"atlas/internal/country"
	"atlas/internal/industry"
	"atlas/internal/organization"
	"atlas/internal/technology"
	"atlas/internal/topic"

	"github.com/jackc/pgx/v5/pgxpool"
)

// callLLM sends article content to the Anthropic Claude API and returns extracted knowledge.
func callLLM(ctx context.Context, apiKey, model string, article Article) (*Knowledge, error) {
	userContent := fmt.Sprintf("Title: %s\nSource: %s\nAuthor: %s\n\nContent:\n%s",
		article.Title, article.Source, nullableStr(article.Author), truncateContent(article.Content, 8000))

	reqBody := map[string]interface{}{
		"model":      model,
		"max_tokens": 2048,
		"system":     systemPrompt(),
		"messages": []map[string]string{
			{"role": "user", "content": userContent},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
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

	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty response content")
	}

	text := apiResp.Content[0].Text
	var k Knowledge
	if err := json.Unmarshal([]byte(text), &k); err != nil {
		return nil, fmt.Errorf("parse LLM JSON: %w (raw: %s)", err, truncateContent(text, 200))
	}

	// Initialize nil slices to empty.
	NormalizeKnowledge(&k)

	return &k, nil
}

func systemPrompt() string {
	return `You are an AI taxonomy classifier. Extract relevant taxonomy entities from the article.
Return ONLY valid JSON, no markdown, no explanation, no code fences.

The JSON must have exactly 5 keys, each an array of objects:
- "topics": array of {code: string, confidence: float}
- "industries": array of {code: string, confidence: float}
- "countries": array of {code: string, confidence: float} — ISO-3166 alpha-2 country codes
- "technologies": array of {code: string, confidence: float}
- "organizations": array of {code: string, confidence: float}

Confidence must be between 0.0 and 1.0.
Return empty arrays [] for categories with no matches.`
}

// ValidateKnowledge validates the extracted knowledge against the expected schema.
func ValidateKnowledge(k *Knowledge) error {
	checks := make([]string, 0)

	checkField := func(name string, entities []Entity) {
		for i, e := range entities {
			if e.Code == "" {
				checks = append(checks, fmt.Sprintf("%s[%d].code is empty string", name, i))
			}
			if e.Confidence < 0 || e.Confidence > 1 {
				checks = append(checks, fmt.Sprintf("%s[%d].confidence is %v, must be between 0 and 1", name, i, e.Confidence))
			}
		}
	}

	checkField("topics", k.Topics)
	checkField("industries", k.Industries)
	checkField("countries", k.Countries)
	checkField("technologies", k.Technologies)
	checkField("organizations", k.Organizations)

	if len(checks) > 0 {
		errMsg := "validation errors: "
		for i, c := range checks {
			if i > 0 {
				errMsg += "; "
			}
			errMsg += c
		}
		return errors.New(errMsg)
	}
	return nil
}

// ResolveAndInsert resolves entity codes to taxonomy IDs and inserts into M2M junction tables.
// Uses ON CONFLICT DO NOTHING to skip duplicates. Logs and skips unknown codes.
func ResolveAndInsert(ctx context.Context, pool *pgxpool.Pool, articleID string, k *Knowledge) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Topics
	for _, e := range k.Topics {
		t, err := topic.GetByCode(ctx, pool, e.Code)
		if err != nil {
			log.Printf("extract: unknown topic code %q — skipping", e.Code)
			continue
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO article_topics (article_id, topic_id)
			VALUES ($1, $2)
			ON CONFLICT (article_id, topic_id) DO NOTHING
		`, articleID, t.ID)
		if err != nil {
			return fmt.Errorf("insert topic %s: %w", e.Code, err)
		}
	}

	// Industries
	for _, e := range k.Industries {
		ind, err := industry.GetByCode(ctx, pool, e.Code)
		if err != nil {
			log.Printf("extract: unknown industry code %q — skipping", e.Code)
			continue
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO article_industries (article_id, industry_id)
			VALUES ($1, $2)
			ON CONFLICT (article_id, industry_id) DO NOTHING
		`, articleID, ind.ID)
		if err != nil {
			return fmt.Errorf("insert industry %s: %w", e.Code, err)
		}
	}

	// Countries
	for _, e := range k.Countries {
		c, err := country.GetByCode(ctx, pool, e.Code)
		if err != nil {
			log.Printf("extract: unknown country code %q — skipping", e.Code)
			continue
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO article_countries (article_id, country_id)
			VALUES ($1, $2)
			ON CONFLICT (article_id, country_id) DO NOTHING
		`, articleID, c.ID)
		if err != nil {
			return fmt.Errorf("insert country %s: %w", e.Code, err)
		}
	}

	// Technologies
	for _, e := range k.Technologies {
		tech, err := technology.GetByCode(ctx, pool, e.Code)
		if err != nil {
			log.Printf("extract: unknown technology code %q — skipping", e.Code)
			continue
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO article_technologies (article_id, technology_id)
			VALUES ($1, $2)
			ON CONFLICT (article_id, technology_id) DO NOTHING
		`, articleID, tech.ID)
		if err != nil {
			return fmt.Errorf("insert technology %s: %w", e.Code, err)
		}
	}

	// Organizations
	for _, e := range k.Organizations {
		org, err := organization.GetByCode(ctx, pool, e.Code)
		if err != nil {
			log.Printf("extract: unknown organization code %q — skipping", e.Code)
			continue
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO article_organizations (article_id, organization_id)
			VALUES ($1, $2)
			ON CONFLICT (article_id, organization_id) DO NOTHING
		`, articleID, org.ID)
		if err != nil {
			return fmt.Errorf("insert organization %s: %w", e.Code, err)
		}
	}

	return tx.Commit(ctx)
}

// NormalizeKnowledge ensures all slices are non-nil.
func NormalizeKnowledge(k *Knowledge) {
	if k.Topics == nil {
		k.Topics = []Entity{}
	}
	if k.Industries == nil {
		k.Industries = []Entity{}
	}
	if k.Countries == nil {
		k.Countries = []Entity{}
	}
	if k.Technologies == nil {
		k.Technologies = []Entity{}
	}
	if k.Organizations == nil {
		k.Organizations = []Entity{}
	}
}

func truncateContent(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func nullableStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}