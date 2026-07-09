package evaluate

import (
	"time"

	"atlas/internal/extract"
)

// GoldenRecord is a labeled article with expected extraction entities.
type GoldenRecord struct {
	ID       string           `json:"id"`
	Title    string           `json:"title"`
	Content  string           `json:"content"`
	URL      string           `json:"url"`
	Source   string           `json:"source"`
	Expected extract.Knowledge `json:"expected"`
}

// GoldenDataset is a collection of labeled records.
type GoldenDataset struct {
	Name    string         `json:"name"`
	Records []GoldenRecord `json:"records"`
}

// CategoryMetrics holds precision/recall for one entity category.
type CategoryMetrics struct {
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	F1        float64 `json:"f1"`
	TP        int     `json:"true_positives"`
	FP        int     `json:"false_positives"`
	FN        int     `json:"false_negatives"`
}

// EvalResult is the full evaluation output.
type EvalResult struct {
	DatasetName          string                    `json:"dataset_name"`
	TotalRecords         int                       `json:"total_records"`
	Provider             string                    `json:"provider"`
	Model                string                    `json:"model"`
	JSONValidationRate   float64                   `json:"json_validation_rate"`
	AvgProcessingTime    time.Duration             `json:"avg_processing_time"`
	TotalProcessingTime  time.Duration             `json:"total_processing_time"`
	IndustryAccuracy     float64                   `json:"industry_accuracy"`
	TopicAccuracy        float64                   `json:"topic_accuracy"`
	CountryAccuracy      float64                   `json:"country_accuracy"`
	TechnologyAccuracy   float64                   `json:"technology_accuracy"`
	OrganizationAccuracy float64                   `json:"organization_accuracy"`
	OverallAccuracy      float64                   `json:"overall_accuracy"`
	Categories           map[string]CategoryMetrics `json:"categories"`
	Records              []RecordResult            `json:"records,omitempty"`
}

// RecordResult is per-record evaluation data.
type RecordResult struct {
	RecordID        string             `json:"record_id"`
	Valid           bool               `json:"valid"`
	ValidationError string             `json:"validation_error,omitempty"`
	ProcessingTime  time.Duration      `json:"processing_time"`
	Extracted       *extract.Knowledge `json:"extracted,omitempty"`
	MatchDetails    map[string]MatchDetail `json:"match_details,omitempty"`
}

// MatchDetail shows per-category match info for a single record.
type MatchDetail struct {
	Expected []string `json:"expected"`
	Got      []string `json:"got"`
	Correct  []string `json:"correct"`
	Missing  []string `json:"missing"`
	Extra    []string `json:"extra"`
}