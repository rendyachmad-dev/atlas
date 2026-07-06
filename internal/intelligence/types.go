package intelligence

import (
	"time"
)

// AnalysisJob represents a row from the analysis_jobs table.
type AnalysisJob struct {
	ID           string     `json:"id"`
	ArticleID    string     `json:"article_id"`
	Status       string     `json:"status"`
	Priority     int        `json:"priority"`
	RetryCount   int        `json:"retry_count"`
	WorkerID     *string    `json:"worker_id,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// AnalysisResult represents a row from the analysis_results table.
type AnalysisResult struct {
	ID            string     `json:"id"`
	ArticleID     string     `json:"article_id"`
	Summary       *string    `json:"summary,omitempty"`
	BusinessImpact *string   `json:"business_impact,omitempty"`
	Urgency       *int       `json:"urgency,omitempty"`
	Confidence    *float32   `json:"confidence,omitempty"`
	Reasoning     *string    `json:"reasoning,omitempty"`
	RawJSON       *string    `json:"raw_json,omitempty"`
	LLMProvider   *string    `json:"llm_provider,omitempty"`
	LLMModel      *string    `json:"llm_model,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ArticleData is the minimal article data needed for analysis.
type ArticleData struct {
	ID      string
	Title   string
	Content string
	URL     string
	Source  string
	Author  *string
}

// MockAnalysisData is the deterministic fake data returned by MockProvider.
type MockAnalysisData struct {
	Summary       string  `json:"summary"`
	BusinessImpact string `json:"business_impact"`
	Urgency       int     `json:"urgency"`
	Confidence    float32 `json:"confidence"`
	Reasoning     string  `json:"reasoning"`
	Sentiment     string  `json:"sentiment"`
	TimeHorizon   string  `json:"time_horizon"`
	RiskScore     int     `json:"risk_score"`
	InnovationScore int   `json:"innovation_score"`
	MarketSizeScore int  `json:"market_size_score"`
}