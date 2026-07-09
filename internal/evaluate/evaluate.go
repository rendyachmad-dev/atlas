package evaluate

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"atlas/internal/extract"
)

// LoadDataset reads a golden dataset from a JSON file.
func LoadDataset(path string) (*GoldenDataset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dataset: %w", err)
	}

	var ds GoldenDataset
	if err := json.Unmarshal(data, &ds); err != nil {
		return nil, fmt.Errorf("parse dataset: %w", err)
	}

	if len(ds.Records) == 0 {
		return nil, fmt.Errorf("dataset %q has no records", path)
	}

	return &ds, nil
}

// Evaluate runs the provider against a golden dataset and returns metrics.
func Evaluate(provider extract.Provider, ds *GoldenDataset, model string) (*EvalResult, error) {
	providerName := providerNameOf(provider)
	result := &EvalResult{
		DatasetName:  ds.Name,
		TotalRecords: len(ds.Records),
		Provider:     providerName,
		Model:        model,
		Categories:   make(map[string]CategoryMetrics),
	}

	var totalProcessTime time.Duration
	validCount := 0

	// Per-category accumulators.
	cats := []string{"topics", "industries", "countries", "technologies", "organizations"}
	acc := newAccumMap(cats)

	for _, record := range ds.Records {
		rr := RecordResult{
			RecordID:     record.ID,
			MatchDetails: make(map[string]MatchDetail),
		}

		start := time.Now()
		k, err := provider.Analyze(extract.Article{
			ID:      record.ID,
			Title:   record.Title,
			Content: record.Content,
			URL:     record.URL,
			Source:  record.Source,
		})
		elapsed := time.Since(start)
		rr.ProcessingTime = elapsed
		totalProcessTime += elapsed

		if err != nil {
			rr.Valid = false
			rr.ValidationError = fmt.Sprintf("provider error: %v", err)
			result.Records = append(result.Records, rr)
			continue
		}

		// Validate JSON schema.
		if err := extract.ValidateKnowledge(k); err != nil {
			rr.Valid = false
			rr.ValidationError = fmt.Sprintf("validation error: %v", err)
			rr.Extracted = k
			result.Records = append(result.Records, rr)
			continue
		}

		rr.Valid = true
		validCount++
		rr.Extracted = k

		// Compare each category.
		compareCategory(&rr, acc, "topics", codes(record.Expected.Topics), codes(k.Topics))
		compareCategory(&rr, acc, "industries", codes(record.Expected.Industries), codes(k.Industries))
		compareCategory(&rr, acc, "countries", codes(record.Expected.Countries), codes(k.Countries))
		compareCategory(&rr, acc, "technologies", codes(record.Expected.Technologies), codes(k.Technologies))
		compareCategory(&rr, acc, "organizations", codes(record.Expected.Organizations), codes(k.Organizations))

		result.Records = append(result.Records, rr)
	}

	// Compute aggregate metrics.
	result.JSONValidationRate = float64(validCount) / float64(len(ds.Records))
	result.AvgProcessingTime = totalProcessTime / time.Duration(len(ds.Records))
	result.TotalProcessingTime = totalProcessTime

	// Per-category metrics.
	totalCorrect := 0
	totalFields := 0
	for _, cat := range cats {
		a := acc[cat]
		prec := safeDiv(float64(a.TP), float64(a.TP+a.FP))
		rec := safeDiv(float64(a.TP), float64(a.TP+a.FN))
		f1 := safeDiv(2*prec*rec, prec+rec)
		result.Categories[cat] = CategoryMetrics{
			Precision: prec,
			Recall:    rec,
			F1:        f1,
			TP:        a.TP,
			FP:        a.FP,
			FN:        a.FN,
		}
		totalCorrect += a.correct
		totalFields += a.total
	}

	result.IndustryAccuracy = acc["industries"].correctRate()
	result.TopicAccuracy = acc["topics"].correctRate()
	result.CountryAccuracy = acc["countries"].correctRate()
	result.TechnologyAccuracy = acc["technologies"].correctRate()
	result.OrganizationAccuracy = acc["organizations"].correctRate()
	result.OverallAccuracy = safeDiv(float64(totalCorrect), float64(totalFields))

	return result, nil
}

// codes extracts entity codes from a slice of entities.
func codes(ents []extract.Entity) []string {
	out := make([]string, len(ents))
	for i, e := range ents {
		out[i] = e.Code
	}
	sort.Strings(out)
	return out
}

// compareCategory computes match stats for one category and updates accumulators.
func compareCategory(rr *RecordResult, acc map[string]*catAccum, cat string, expected, got []string) {
	correct := intersectStrings(expected, got)
	missing := differenceStrings(expected, got)
	extra := differenceStrings(got, expected)

	md := MatchDetail{
		Expected: expected,
		Got:      got,
		Correct:  correct,
		Missing:  missing,
		Extra:    extra,
	}
	rr.MatchDetails[cat] = md

	a := acc[cat]
	a.correct += len(correct)
	a.total += len(expected)

	// TP = correct, FP = extra, FN = missing.
	a.TP += len(correct)
	a.FP += len(extra)
	a.FN += len(missing)
}

// Report formats the evaluation result as a human-readable string.
func Report(r *EvalResult) string {
	out := fmt.Sprintf("━━━ Evaluation Report ━━━\n")
	out += fmt.Sprintf("Dataset: %s  (%d records)\n", r.DatasetName, r.TotalRecords)
	out += fmt.Sprintf("Provider: %s  Model: %s\n", r.Provider, r.Model)
	out += fmt.Sprintf("\n")

	out += fmt.Sprintf("JSON Validation Rate:  %.1f%%\n", r.JSONValidationRate*100)
	out += fmt.Sprintf("Avg Processing Time:   %s\n", r.AvgProcessingTime.Round(time.Millisecond))
	out += fmt.Sprintf("Total Processing Time: %s\n", r.TotalProcessingTime.Round(time.Millisecond))
	out += fmt.Sprintf("\n")

	out += fmt.Sprintf("─── Entity Accuracy ───\n")
	out += fmt.Sprintf("  Industry:      %.1f%%\n", r.IndustryAccuracy*100)
	out += fmt.Sprintf("  Topic:         %.1f%%\n", r.TopicAccuracy*100)
	out += fmt.Sprintf("  Country:       %.1f%%\n", r.CountryAccuracy*100)
	out += fmt.Sprintf("  Technology:    %.1f%%\n", r.TechnologyAccuracy*100)
	out += fmt.Sprintf("  Organization:  %.1f%%\n", r.OrganizationAccuracy*100)
	out += fmt.Sprintf("  Overall:       %.1f%%\n", r.OverallAccuracy*100)
	out += fmt.Sprintf("\n")

	out += fmt.Sprintf("─── Precision / Recall / F1 ───\n")
	for _, cat := range []string{"topics", "industries", "countries", "technologies", "organizations"} {
		c := r.Categories[cat]
		out += fmt.Sprintf("  %-14s P: %.1f%%  R: %.1f%%  F1: %.1f%%  (TP:%d FP:%d FN:%d)\n",
			cat+":", c.Precision*100, c.Recall*100, c.F1*100, c.TP, c.FP, c.FN)
	}

	// Summary of per-record results.
	failures := 0
	for _, rr := range r.Records {
		if !rr.Valid {
			failures++
		}
	}
	if failures > 0 {
		out += fmt.Sprintf("\n")
		out += fmt.Sprintf("─── Failures (%d) ───\n", failures)
		for _, rr := range r.Records {
			if !rr.Valid {
				out += fmt.Sprintf("  [%s] %s\n", shortID(rr.RecordID), rr.ValidationError)
			}
		}
	}

	return out
}

// JSONReport returns the evaluation result as formatted JSON.
func JSONReport(r *EvalResult) (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal eval result: %w", err)
	}
	return string(b), nil
}

// providerNameOf returns a human-readable name for the provider.
func providerNameOf(p extract.Provider) string {
	switch p.(type) {
	case *extract.MockProvider:
		return "mock"
	case *extract.AnthropicExtractor:
		return "anthropic"
	default:
		return fmt.Sprintf("%T", p)
	}
}

// ── helpers ──

type catAccum struct {
	correct int
	total   int
	TP      int
	FP      int
	FN      int
}

func newAccumMap(cats []string) map[string]*catAccum {
	m := make(map[string]*catAccum, len(cats))
	for _, c := range cats {
		m[c] = &catAccum{}
	}
	return m
}

func (a *catAccum) correctRate() float64 {
	if a.total == 0 {
		return 1.0
	}
	return float64(a.correct) / float64(a.total)
}

func sliceToSet(s []string) map[string]bool {
	m := make(map[string]bool, len(s))
	for _, v := range s {
		m[v] = true
	}
	return m
}

func intersectStrings(a, b []string) []string {
	setB := sliceToSet(b)
	var out []string
	for _, v := range a {
		if setB[v] {
			out = append(out, v)
		}
	}
	return out
}

func differenceStrings(a, b []string) []string {
	setB := sliceToSet(b)
	var out []string
	for _, v := range a {
		if !setB[v] {
			out = append(out, v)
		}
	}
	return out
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func shortID(id string) string {
	if len(id) < 8 {
		return id
	}
	return id[:8]
}

// roundFloat rounds to the given number of decimal places.
func roundFloat(val float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(val*pow) / pow
}