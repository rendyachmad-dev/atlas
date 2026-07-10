package extract

import (
	"testing"
)

func TestValidateKnowledge_Valid(t *testing.T) {
	k := &Knowledge{
		Topics: []Entity{
			{Code: "machine-learning", Confidence: 0.95},
			{Code: "cloud-computing", Confidence: 0.8},
		},
		Industries: []Entity{
			{Code: "tech", Confidence: 0.9},
		},
		Countries: []Entity{
			{Code: "US", Confidence: 0.85},
		},
		Technologies: []Entity{
			{Code: "ai", Confidence: 0.92},
		},
		Organizations: []Entity{
			{Code: "openai", Confidence: 0.88},
		},
	}

	err := ValidateKnowledge(k)
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
}

func TestValidateKnowledge_EmptySlices(t *testing.T) {
	k := &Knowledge{
		Topics:        []Entity{},
		Industries:    []Entity{},
		Countries:     []Entity{},
		Technologies:  []Entity{},
		Organizations: []Entity{},
	}

	err := ValidateKnowledge(k)
	if err != nil {
		t.Errorf("expected nil error for empty slices, got: %v", err)
	}
}

func TestValidateKnowledge_EmptyCode(t *testing.T) {
	k := &Knowledge{
		Topics: []Entity{
			{Code: "", Confidence: 0.5},
		},
		Industries:    []Entity{},
		Countries:     []Entity{},
		Technologies:  []Entity{},
		Organizations: []Entity{},
	}

	err := ValidateKnowledge(k)
	if err == nil {
		t.Fatal("expected error for empty code, got nil")
	}
	if err.Error() != "validation errors: topics[0].code is empty string" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestValidateKnowledge_ConfidenceTooHigh(t *testing.T) {
	k := &Knowledge{
		Topics: []Entity{
			{Code: "ml", Confidence: 1.5},
		},
		Industries:    []Entity{},
		Countries:     []Entity{},
		Technologies:  []Entity{},
		Organizations: []Entity{},
	}

	err := ValidateKnowledge(k)
	if err == nil {
		t.Fatal("expected error for confidence > 1, got nil")
	}
}

func TestValidateKnowledge_ConfidenceTooLow(t *testing.T) {
	k := &Knowledge{
		Topics: []Entity{
			{Code: "ml", Confidence: -0.1},
		},
		Industries:    []Entity{},
		Countries:     []Entity{},
		Technologies:  []Entity{},
		Organizations: []Entity{},
	}

	err := ValidateKnowledge(k)
	if err == nil {
		t.Fatal("expected error for negative confidence, got nil")
	}
}

func TestValidateKnowledge_ConfidenceBoundary(t *testing.T) {
	k := &Knowledge{
		Topics: []Entity{
			{Code: "ml", Confidence: 0.0},
			{Code: "ai", Confidence: 1.0},
		},
		Industries:    []Entity{},
		Countries:     []Entity{},
		Technologies:  []Entity{},
		Organizations: []Entity{},
	}

	err := ValidateKnowledge(k)
	if err != nil {
		t.Errorf("expected nil error for boundary confidence, got: %v", err)
	}
}

func TestValidateKnowledge_MultipleErrors(t *testing.T) {
	k := &Knowledge{
		Topics: []Entity{
			{Code: "", Confidence: 0.5},
			{Code: "ml", Confidence: 1.5},
		},
		Industries: []Entity{
			{Code: "", Confidence: -0.5},
		},
		Countries:     []Entity{},
		Technologies:  []Entity{},
		Organizations: []Entity{},
	}

	err := ValidateKnowledge(k)
	if err == nil {
		t.Fatal("expected error for multiple validation failures, got nil")
	}

	msg := err.Error()
	if msg == "" {
		t.Fatal("error message should not be empty")
	}
}

func TestValidateKnowledge_NilFields(t *testing.T) {
	k := &Knowledge{
		Topics:        nil,
		Industries:    nil,
		Countries:     nil,
		Technologies:  nil,
		Organizations: nil,
	}

	NormalizeKnowledge(k)
	err := ValidateKnowledge(k)
	if err != nil {
		t.Errorf("expected nil error after normalization, got: %v", err)
	}
}