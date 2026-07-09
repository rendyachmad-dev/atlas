package extract

import "time"

// ExtractionJob represents a row from the extraction_jobs table.
type ExtractionJob struct {
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

// ExtractionResult represents a row from the extraction_results table.
type ExtractionResult struct {
	ID          string    `json:"id"`
	ArticleID   string    `json:"article_id"`
	RawJSON     *string   `json:"raw_json,omitempty"`
	LLMProvider *string   `json:"llm_provider,omitempty"`
	LLMModel    *string   `json:"llm_model,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Entity holds a single entity code + confidence from the LLM.
type Entity struct {
	Code       string  `json:"code"`
	Confidence float64 `json:"confidence"`
}

// Knowledge is the top-level JSON structure returned by the LLM.
type Knowledge struct {
	Topics        []Entity `json:"topics"`
	Industries    []Entity `json:"industries"`
	Countries     []Entity `json:"countries"`
	Technologies  []Entity `json:"technologies"`
	Organizations []Entity `json:"organizations"`
}

// Article is the minimal article data needed for extraction.
type Article struct {
	ID      string
	Title   string
	Content string
	URL     string
	Source  string
	Author  *string
}