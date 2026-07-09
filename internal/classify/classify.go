package classify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Classification holds the output of classifying a single article.
type Classification struct {
	ArticleID             string
	IndustryName          string
	IndustryConfidence    float32
	TechnologyName        string
	TechnologyConfidence  float32
	RegionName            string
	RegionConfidence      float32
	StakeholderName       string
	StakeholderConfidence float32
	OverallConfidence     float32
	LLMProvider           string
	LLMModel              string
	RawJSON               string
}

// Classifier defines the interface for classifying articles.
type Classifier interface {
	ClassifyByID(ctx context.Context, articleID string) (*Classification, error)
}

// LLMClassifier calls an LLM to classify articles.
type LLMClassifier struct {
	apiKey string
	model  string
	pool   *pgxpool.Pool
	client *http.Client
}

// New creates a new LLMClassifier.
func New(pool *pgxpool.Pool, apiKey, model string) *LLMClassifier {
	if model == "" {
		model = "claude-sonnet-5"
	}
	return &LLMClassifier{
		apiKey: apiKey,
		model:  model,
		pool:   pool,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

// classifyResult is the JSON structure the LLM returns.
type classifyResult struct {
	Industry              string  `json:"industry"`
	IndustryConfidence    float32 `json:"industry_confidence"`
	Technology            string  `json:"technology"`
	TechnologyConfidence  float32 `json:"technology_confidence"`
	Country               string  `json:"country"`
	CountryConfidence     float32 `json:"country_confidence"`
	Stakeholder           string  `json:"stakeholder"`
	StakeholderConfidence float32 `json:"stakeholder_confidence"`
	OverallConfidence     float32 `json:"overall_confidence"`
}

// articleData is the minimal article data loaded from DB.
type articleData struct {
	ID      string
	Title   string
	Content string
	Source  string
}

// ClassifyByID loads an article, classifies it via LLM, stores the result, and returns it.
func (c *LLMClassifier) ClassifyByID(ctx context.Context, articleID string) (*Classification, error) {
	// Check if classification already exists.
	existing, err := c.getExisting(ctx, articleID)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Load article data.
	article, err := c.loadArticle(ctx, articleID)
	if err != nil {
		return nil, fmt.Errorf("load article: %w", err)
	}

	// Call LLM.
	result, err := c.callLLM(ctx, article)
	if err != nil {
		return nil, fmt.Errorf("llm classify: %w", err)
	}

	// Store in DB.
	if err := c.store(ctx, articleID, result); err != nil {
		return nil, fmt.Errorf("store classification: %w", err)
	}

	return result, nil
}

func (c *LLMClassifier) getExisting(ctx context.Context, articleID string) (*Classification, error) {
	row := c.pool.QueryRow(ctx, `
		SELECT article_id,
		       COALESCE(industry_name, ''),
		       COALESCE(industry_confidence, 0),
		       COALESCE(technology_name, ''),
		       COALESCE(technology_confidence, 0),
		       COALESCE(region_name, ''),
		       COALESCE(region_confidence, 0),
		       COALESCE(stakeholder_name, ''),
		       COALESCE(stakeholder_confidence, 0),
		       COALESCE(overall_confidence, 0),
		       COALESCE(llm_provider, ''),
		       COALESCE(llm_model, ''),
		       COALESCE(raw_json, '{}')
		FROM article_classifications
		WHERE article_id = $1
	`, articleID)

	var cls Classification
	err := row.Scan(
		&cls.ArticleID,
		&cls.IndustryName, &cls.IndustryConfidence,
		&cls.TechnologyName, &cls.TechnologyConfidence,
		&cls.RegionName, &cls.RegionConfidence,
		&cls.StakeholderName, &cls.StakeholderConfidence,
		&cls.OverallConfidence,
		&cls.LLMProvider, &cls.LLMModel, &cls.RawJSON,
	)
	if err != nil {
		return nil, err
	}
	return &cls, nil
}

func (c *LLMClassifier) loadArticle(ctx context.Context, articleID string) (*articleData, error) {
	var a articleData
	err := c.pool.QueryRow(ctx, `
		SELECT a.id, a.title, COALESCE(a.content, ''), s.name
		FROM articles a
		JOIN sources s ON s.id = a.source_id
		WHERE a.id = $1
	`, articleID).Scan(&a.ID, &a.Title, &a.Content, &a.Source)
	if err != nil {
		return nil, fmt.Errorf("query article %s: %w", articleID, err)
	}
	return &a, nil
}

func (c *LLMClassifier) callLLM(ctx context.Context, article *articleData) (*Classification, error) {
	systemPrompt := `You are an AI classifier. Given a news article, classify it into the following categories.

Return ONLY valid JSON, no markdown formatting, no explanation. The JSON must have these fields:
- industry: the primary industry this article relates to (e.g. Healthcare, Finance, Energy, Technology, Agriculture, Education, Manufacturing, Transportation, Retail, Cybersecurity, Government)
- industry_confidence: float 0.0-1.0
- technology: the primary technology domain (e.g. Artificial Intelligence, Computer Vision, NLP/LLM, Cloud Computing, Cybersecurity, Blockchain, IoT, Robotics, Fintech, Biotechnology, Renewable Energy, Electric Vehicles, 5G/Telecom, Edge Computing, Quantum Computing, Data & Analytics)
- technology_confidence: float 0.0-1.0
- country: the primary country or region (e.g. United States, Indonesia, Global, Europe, Asia, China, Japan)
- country_confidence: float 0.0-1.0
- stakeholder: the primary stakeholder affected (e.g. Enterprise, Government, Consumer, Startup, Healthcare Provider, Hospital, University, Investor, Media, Regulator, Small Business, Financial Institution, Manufacturer)
- stakeholder_confidence: float 0.0-1.0
- overall_confidence: float 0.0-1.0 representing how confident you are in the overall classification

Use the article title and content to determine the best matches. Be specific.`

	userContent := fmt.Sprintf("Title: %s\nSource: %s\n\nContent:\n%s",
		article.Title, article.Source, truncateContent(article.Content, 8000))

	reqBody := map[string]interface{}{
		"model":      c.model,
		"max_tokens": 1024,
		"system":     systemPrompt,
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
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := c.client.Do(req)
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

	var result classifyResult
	text := apiResp.Content[0].Text
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("parse LLM JSON: %w (raw: %s)", err, truncateContent(text, 200))
	}

	rawBytes, _ := json.Marshal(result)
	provider := "anthropic"

	return &Classification{
		ArticleID:             article.ID,
		IndustryName:          result.Industry,
		IndustryConfidence:    clampConfidence(result.IndustryConfidence),
		TechnologyName:        result.Technology,
		TechnologyConfidence:  clampConfidence(result.TechnologyConfidence),
		RegionName:            result.Country,
		RegionConfidence:      clampConfidence(result.CountryConfidence),
		StakeholderName:       result.Stakeholder,
		StakeholderConfidence: clampConfidence(result.StakeholderConfidence),
		OverallConfidence:     clampConfidence(result.OverallConfidence),
		LLMProvider:           provider,
		LLMModel:              c.model,
		RawJSON:               string(rawBytes),
	}, nil
}

func (c *LLMClassifier) store(ctx context.Context, articleID string, cls *Classification) error {
	_, err := c.pool.Exec(ctx, `
		INSERT INTO article_classifications
			(article_id, industry_name, industry_confidence,
			 technology_name, technology_confidence,
			 region_name, region_confidence,
			 stakeholder_name, stakeholder_confidence,
			 overall_confidence, llm_provider, llm_model, raw_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (article_id) DO UPDATE SET
			industry_name = EXCLUDED.industry_name,
			industry_confidence = EXCLUDED.industry_confidence,
			technology_name = EXCLUDED.technology_name,
			technology_confidence = EXCLUDED.technology_confidence,
			region_name = EXCLUDED.region_name,
			region_confidence = EXCLUDED.region_confidence,
			stakeholder_name = EXCLUDED.stakeholder_name,
			stakeholder_confidence = EXCLUDED.stakeholder_confidence,
			overall_confidence = EXCLUDED.overall_confidence,
			llm_provider = EXCLUDED.llm_provider,
			llm_model = EXCLUDED.llm_model,
			raw_json = EXCLUDED.raw_json
	`,
		articleID,
		cls.IndustryName, cls.IndustryConfidence,
		cls.TechnologyName, cls.TechnologyConfidence,
		cls.RegionName, cls.RegionConfidence,
		cls.StakeholderName, cls.StakeholderConfidence,
		cls.OverallConfidence,
		cls.LLMProvider, cls.LLMModel, cls.RawJSON,
	)
	return err
}

func clampConfidence(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func truncateContent(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// --- MockClassifier ---

// MockClassifier returns deterministic fake classification data.
type MockClassifier struct {
	pool       *pgxpool.Pool
	deterministic bool
}

// NewMock creates a new MockClassifier.
func NewMock(pool *pgxpool.Pool) *MockClassifier {
	return &MockClassifier{pool: pool, deterministic: true}
}

func (m *MockClassifier) ClassifyByID(ctx context.Context, articleID string) (*Classification, error) {
	// Check cache first.
	existing, err := m.getExisting(ctx, articleID)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Load article to get a seed.
	article, err := m.loadArticle(ctx, articleID)
	if err != nil {
		return nil, fmt.Errorf("load article: %w", err)
	}

	var seed int64
	if m.deterministic {
		h := 0
		for _, c := range articleID + article.Title {
			h = h*31 + int(c)
		}
		seed = int64(h)
	} else {
		seed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(seed))

	industries := []string{"Healthcare", "Technology", "Finance", "Energy", "Agriculture", "Education", "Manufacturing", "Transportation", "Retail", "Cybersecurity", "Government"}
	technologies := []string{"Artificial Intelligence", "Computer Vision", "NLP/LLM", "Cloud Computing", "Cybersecurity", "Blockchain", "IoT", "Robotics", "Fintech", "Biotechnology", "Renewable Energy", "Electric Vehicles", "Edge Computing", "Data & Analytics"}
	countries := []string{"United States", "Indonesia", "Global", "Europe", "Asia", "China", "Japan"}
	stakeholders := []string{"Enterprise", "Government", "Consumer", "Startup", "Hospital", "University", "Investor", "Media", "Regulator", "Manufacturer", "Financial Institution"}

	cls := &Classification{
		ArticleID:             articleID,
		IndustryName:          industries[rng.Intn(len(industries))],
		IndustryConfidence:    float32(70+rng.Intn(25)) / 100.0,
		TechnologyName:        technologies[rng.Intn(len(technologies))],
		TechnologyConfidence:  float32(70+rng.Intn(25)) / 100.0,
		RegionName:            countries[rng.Intn(len(countries))],
		RegionConfidence:      float32(70+rng.Intn(25)) / 100.0,
		StakeholderName:       stakeholders[rng.Intn(len(stakeholders))],
		StakeholderConfidence: float32(70+rng.Intn(25)) / 100.0,
		OverallConfidence:     float32(75+rng.Intn(20)) / 100.0,
		LLMProvider:           "mock",
		LLMModel:              "mock-v1",
		RawJSON:               `{"mock": true}`,
	}

	// Store in DB so subsequent calls are cached.
	if err := m.store(ctx, articleID, cls); err != nil {
		return nil, fmt.Errorf("store mock classification: %w", err)
	}

	return cls, nil
}

func (m *MockClassifier) getExisting(ctx context.Context, articleID string) (*Classification, error) {
	row := m.pool.QueryRow(ctx, `
		SELECT article_id,
		       COALESCE(industry_name, ''),
		       COALESCE(industry_confidence, 0),
		       COALESCE(technology_name, ''),
		       COALESCE(technology_confidence, 0),
		       COALESCE(region_name, ''),
		       COALESCE(region_confidence, 0),
		       COALESCE(stakeholder_name, ''),
		       COALESCE(stakeholder_confidence, 0),
		       COALESCE(overall_confidence, 0),
		       COALESCE(llm_provider, ''),
		       COALESCE(llm_model, ''),
		       COALESCE(raw_json, '{}')
		FROM article_classifications
		WHERE article_id = $1
	`, articleID)

	var cls Classification
	err := row.Scan(
		&cls.ArticleID,
		&cls.IndustryName, &cls.IndustryConfidence,
		&cls.TechnologyName, &cls.TechnologyConfidence,
		&cls.RegionName, &cls.RegionConfidence,
		&cls.StakeholderName, &cls.StakeholderConfidence,
		&cls.OverallConfidence,
		&cls.LLMProvider, &cls.LLMModel, &cls.RawJSON,
	)
	if err != nil {
		return nil, err
	}
	return &cls, nil
}

func (m *MockClassifier) loadArticle(ctx context.Context, articleID string) (*articleData, error) {
	var a articleData
	err := m.pool.QueryRow(ctx, `
		SELECT a.id, a.title, COALESCE(a.content, ''), s.name
		FROM articles a
		JOIN sources s ON s.id = a.source_id
		WHERE a.id = $1
	`, articleID).Scan(&a.ID, &a.Title, &a.Content, &a.Source)
	if err != nil {
		return nil, fmt.Errorf("query article %s: %w", articleID, err)
	}
	return &a, nil
}

func (m *MockClassifier) store(ctx context.Context, articleID string, cls *Classification) error {
	_, err := m.pool.Exec(ctx, `
		INSERT INTO article_classifications
			(article_id, industry_name, industry_confidence,
			 technology_name, technology_confidence,
			 region_name, region_confidence,
			 stakeholder_name, stakeholder_confidence,
			 overall_confidence, llm_provider, llm_model, raw_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (article_id) DO NOTHING
	`,
		articleID,
		cls.IndustryName, cls.IndustryConfidence,
		cls.TechnologyName, cls.TechnologyConfidence,
		cls.RegionName, cls.RegionConfidence,
		cls.StakeholderName, cls.StakeholderConfidence,
		cls.OverallConfidence,
		cls.LLMProvider, cls.LLMModel, cls.RawJSON,
	)
	return err
}