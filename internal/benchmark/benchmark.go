package benchmark

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"atlas/internal/evaluate"
)

// Runner orchestrates benchmarking multiple providers.
type Runner struct {
	dataset     *evaluate.GoldenDataset
	datasetPath string
}

// NewRunner creates a benchmark runner with a loaded dataset.
func NewRunner(ds *evaluate.GoldenDataset, path string) *Runner {
	return &Runner{
		dataset:     ds,
		datasetPath: path,
	}
}

// Run benchmarks all configured providers and returns a report.
func (r *Runner) Run(ctx context.Context, runs []ProviderRun) (*Report, error) {
	if len(runs) == 0 {
		return nil, fmt.Errorf("no providers configured for benchmark")
	}

	report := &Report{
		DatasetName:  r.dataset.Name,
		DatasetPath:  r.datasetPath,
		TotalRecords: len(r.dataset.Records),
	}

	for _, run := range runs {
		result := r.benchmarkOne(ctx, run)
		report.Runs = append(report.Runs, result)
	}

	report.CompareRows = buildCompareRows(report.Runs)
	report.RankedByAccuracy = rankByAccuracy(report.CompareRows)
	report.RankedBySpeed = rankBySpeed(report.CompareRows)
	report.Recommendations = buildRecommendations(report.CompareRows, report.RankedByAccuracy, report.RankedBySpeed)

	return report, nil
}

// benchmarkOne runs a single provider against the dataset.
func (r *Runner) benchmarkOne(ctx context.Context, run ProviderRun) Result {
	start := time.Now()

	provider, err := CreateProvider(run)
	if err != nil {
		return Result{
			Label:   run.Label,
			Config:  run,
			Error:   fmt.Sprintf("create provider: %v", err),
			Elapsed: time.Since(start),
		}
	}

	model := run.Model
	if run.Mock {
		model = "mock-v1"
	}
	evalResult, err := evaluate.Evaluate(provider, r.dataset, model)
	elapsed := time.Since(start)

	if err != nil {
		return Result{
			Label:   run.Label,
			Config:  run,
			Error:   fmt.Sprintf("evaluate: %v", err),
			Elapsed: elapsed,
		}
	}

	conv := make(map[string]CategoryMetrics)
	for k, v := range evalResult.Categories {
		conv[k] = CategoryMetrics{
			Precision: v.Precision,
			Recall:    v.Recall,
			F1:        v.F1,
			TP:        v.TP,
			FP:        v.FP,
			FN:        v.FN,
		}
	}

	result := Result{
		Label:   run.Label,
		Config:  run,
		Elapsed: elapsed,
		Result: &evaluateResult{
			JSONValidationRate:    evalResult.JSONValidationRate,
			AvgProcessingTime:    evalResult.AvgProcessingTime,
			TotalProcessingTime:  evalResult.TotalProcessingTime,
			IndustryAccuracy:     evalResult.IndustryAccuracy,
			TopicAccuracy:        evalResult.TopicAccuracy,
			CountryAccuracy:      evalResult.CountryAccuracy,
			TechnologyAccuracy:   evalResult.TechnologyAccuracy,
			OrganizationAccuracy: evalResult.OrganizationAccuracy,
			OverallAccuracy:      evalResult.OverallAccuracy,
			Categories:           conv,
		},
	}

	return result
}

func buildCompareRows(results []Result) []CompareRow {
	var rows []CompareRow
	for _, r := range results {
		if r.Error != "" || r.Result == nil {
			rows = append(rows, CompareRow{
				Label:    r.Label,
				Provider: r.Config.Provider,
				Model:    r.Config.Model,
			})
			continue
		}
		res := r.Result
		rows = append(rows, CompareRow{
			Label:                r.Label,
			Provider:             r.Config.Provider,
			Model:                r.Config.Model,
			OverallAccuracy:      res.OverallAccuracy,
			JSONValidRate:        res.JSONValidationRate,
			AvgProcessingTime:    res.AvgProcessingTime,
			TotalTime:            res.TotalProcessingTime,
			TopicAccuracy:        res.TopicAccuracy,
			IndustryAccuracy:     res.IndustryAccuracy,
			CountryAccuracy:      res.CountryAccuracy,
			TechnologyAccuracy:   res.TechnologyAccuracy,
			OrganizationAccuracy: res.OrganizationAccuracy,
			Categories:           res.Categories,
		})
	}
	return rows
}

func rankByAccuracy(rows []CompareRow) []CompareRow {
	sorted := make([]CompareRow, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].OverallAccuracy > sorted[j].OverallAccuracy
	})
	return sorted
}

func rankBySpeed(rows []CompareRow) []CompareRow {
	sorted := make([]CompareRow, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].AvgProcessingTime < sorted[j].AvgProcessingTime
	})
	return sorted
}

func buildRecommendations(rows, byAccuracy, bySpeed []CompareRow) Recommendations {
	rec := Recommendations{}
	validRows := filterValid(rows)
	if len(validRows) == 0 {
		return rec
	}
	if len(byAccuracy) > 0 {
		rec.BestAccuracy = fmt.Sprintf("%s (%.1f%%)", byAccuracy[0].Label, byAccuracy[0].OverallAccuracy*100)
	}
	realSpeed := filterReal(bySpeed)
	if len(realSpeed) > 0 {
		rec.BestSpeed = fmt.Sprintf("%s (%s/record)", realSpeed[0].Label, realSpeed[0].AvgProcessingTime.Round(time.Millisecond))
	}
	rec.BestCost = bestCost(validRows)
	realAccuracy := filterReal(byAccuracy)
	if len(realAccuracy) > 0 {
		rec.BestOverall = fmt.Sprintf("%s (%.1f%%)", realAccuracy[0].Label, realAccuracy[0].OverallAccuracy*100)
	}
	return rec
}

func filterValid(rows []CompareRow) []CompareRow {
	var out []CompareRow
	for _, r := range rows {
		if r.Provider != "" {
			out = append(out, r)
		}
	}
	return out
}

func filterReal(rows []CompareRow) []CompareRow {
	var out []CompareRow
	for _, r := range rows {
		if r.Provider != "" && r.Provider != "mock" {
			out = append(out, r)
		}
	}
	return out
}

func bestCost(rows []CompareRow) string {
	for _, r := range rows {
		switch r.Provider {
		case "groq":
			return "Groq (free tier, $0.59/M tokens)"
		case "anthropic":
			return "Anthropic ($3/M input tokens)"
		case "mock":
			return "Mock (free, no API cost)"
		}
	}
	return "N/A"
}

// ── Report formatting ────────────────────────────────────────

// ReportText formats the benchmark report as a human-readable string.
func ReportText(report *Report) string {
	var b strings.Builder

	b.WriteString("══════════════════════════════════════════════════════════\n")
	b.WriteString("           ATLAS BENCHMARK SUITE\n")
	b.WriteString("══════════════════════════════════════════════════════════\n\n")

	b.WriteString(fmt.Sprintf("Dataset: %s  (%d records)\n", report.DatasetName, report.TotalRecords))
	b.WriteString(fmt.Sprintf("Path:    %s\n", report.DatasetPath))
	b.WriteString("\n")

	// Comparison table.
	b.WriteString("─── Provider Comparison ───\n")
	b.WriteString(fmt.Sprintf("%-24s %7s %7s %7s %7s %7s %7s %8s\n",
		"Provider", "Overall", "Topic", "Indust", "Country", "Tech", "Org", "Avg Time"))
	b.WriteString(strings.Repeat("─", 88) + "\n")

	for _, row := range report.CompareRows {
		if row.Provider == "" {
			b.WriteString(fmt.Sprintf("%-24s %s\n", row.Label, "FAILED"))
			continue
		}
		b.WriteString(fmt.Sprintf("%-24s %6.1f%% %6.1f%% %6.1f%% %6.1f%% %6.1f%% %6.1f%% %8s\n",
			truncateLabel(row.Label, 24),
			row.OverallAccuracy*100,
			row.TopicAccuracy*100,
			row.IndustryAccuracy*100,
			row.CountryAccuracy*100,
			row.TechnologyAccuracy*100,
			row.OrganizationAccuracy*100,
			row.AvgProcessingTime.Round(time.Millisecond).String()))
	}
	b.WriteString("\n")

	// Precision / Recall / F1 per provider.
	b.WriteString("─── Per-Category Precision / Recall / F1 ───\n")
	for _, row := range report.CompareRows {
		if row.Provider == "" || row.Categories == nil {
			continue
		}
		b.WriteString(fmt.Sprintf("  %s:\n", row.Label))
		for _, cat := range []string{"topics", "industries", "countries", "technologies", "organizations"} {
			c := row.Categories[cat]
			b.WriteString(fmt.Sprintf("    %-14s P: %.1f%%  R: %.1f%%  F1: %.1f%%  (TP:%d FP:%d FN:%d)\n",
				cat+":", c.Precision*100, c.Recall*100, c.F1*100, c.TP, c.FP, c.FN))
		}
		b.WriteString(fmt.Sprintf("    JSON Valid:    %.1f%%  |  Avg: %s  |  Total: %s\n",
			row.JSONValidRate*100,
			row.AvgProcessingTime.Round(time.Millisecond).String(),
			row.TotalTime.Round(time.Millisecond).String()))
	}
	b.WriteString("\n")

	// Rankings.
	b.WriteString("─── Rankings ───\n\n")

	b.WriteString("By Accuracy:\n")
	for i, row := range report.RankedByAccuracy {
		suffix := ""
		if row.Provider == "" {
			suffix = " (FAILED)"
		}
		b.WriteString(fmt.Sprintf("  %d. %-24s %6.1f%%%s\n", i+1, row.Label, row.OverallAccuracy*100, suffix))
	}
	b.WriteString("\n")

	b.WriteString("By Speed (fastest → slowest):\n")
	for i, row := range report.RankedBySpeed {
		suffix := ""
		if row.Provider == "" {
			suffix = " (FAILED)"
		}
		b.WriteString(fmt.Sprintf("  %d. %-24s %8s%s\n", i+1, row.Label, row.AvgProcessingTime.Round(time.Millisecond).String(), suffix))
	}
	b.WriteString("\n")

	// Recommendations.
	b.WriteString("─── Recommendations ───\n\n")
	rec := report.Recommendations
	if rec.BestAccuracy != "" {
		b.WriteString(fmt.Sprintf("  Best Accuracy:  %s\n", rec.BestAccuracy))
	}
	if rec.BestSpeed != "" {
		b.WriteString(fmt.Sprintf("  Best Speed:     %s\n", rec.BestSpeed))
	}
	if rec.BestCost != "" {
		b.WriteString(fmt.Sprintf("  Best Cost:      %s\n", rec.BestCost))
	}
	if rec.BestOverall != "" {
		b.WriteString(fmt.Sprintf("  Best Overall:   %s\n", rec.BestOverall))
	}
	b.WriteString("\n")
	b.WriteString("══════════════════════════════════════════════════════════\n")

	return b.String()
}

func truncateLabel(label string, maxLen int) string {
	if len(label) <= maxLen {
		return label
	}
	return label[:maxLen-3] + "..."
}