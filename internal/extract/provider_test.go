package extract

import (
	"testing"
)

func TestMockProvider_Deterministic(t *testing.T) {
	p := NewMockProvider(true)

	article1 := Article{
		ID:    "article-1",
		Title: "Test Article One",
	}
	article2 := Article{
		ID:    "article-2",
		Title: "Test Article Two",
	}

	// Same article should return same result.
	r1a, err := p.Analyze(article1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r1b, err := p.Analyze(article1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r1a.Topics) != len(r1b.Topics) {
		t.Errorf("deterministic: topic count mismatch: %d vs %d", len(r1a.Topics), len(r1b.Topics))
	}
	if len(r1a.Topics) > 0 && len(r1b.Topics) > 0 {
		if r1a.Topics[0].Code != r1b.Topics[0].Code {
			t.Errorf("deterministic: first topic code mismatch: %s vs %s", r1a.Topics[0].Code, r1b.Topics[0].Code)
		}
	}

	// Different articles should return different results.
	r2, err := p.Analyze(article2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sameTopic := len(r1a.Topics) > 0 && len(r2.Topics) > 0 && r1a.Topics[0].Code == r2.Topics[0].Code
	sameIndustry := len(r1a.Industries) > 0 && len(r2.Industries) > 0 && r1a.Industries[0].Code == r2.Industries[0].Code
	if sameTopic && sameIndustry {
		t.Log("warning: different articles produced same first entity — unlikely but possible")
	}
}

func TestMockProvider_NonDeterministic(t *testing.T) {
	p := NewMockProvider(false)

	article := Article{
		ID:    "article-1",
		Title: "Test Article",
	}

	r1, err := p.Analyze(article)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	r2, err := p.Analyze(article)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	same := len(r1.Topics) == len(r2.Topics)
	if same {
		for i := range r1.Topics {
			if r1.Topics[i].Code != r2.Topics[i].Code {
				same = false
				break
			}
		}
	}
	if same {
		t.Log("warning: non-deterministic mock produced same results — collision")
	}
}

func TestMockProvider_AllCategoriesPopulated(t *testing.T) {
	p := NewMockProvider(true)

	article := Article{
		ID:    "article-1",
		Title: "Test Article",
	}

	result, err := p.Analyze(article)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Topics) == 0 {
		t.Error("expected at least 1 topic, got 0")
	}
	if len(result.Industries) == 0 {
		t.Error("expected at least 1 industry, got 0")
	}
	if len(result.Countries) == 0 {
		t.Error("expected at least 1 country, got 0")
	}
	if len(result.Technologies) == 0 {
		t.Error("expected at least 1 technology, got 0")
	}
	if len(result.Organizations) == 0 {
		t.Error("expected at least 1 organization, got 0")
	}

	for _, e := range result.Topics {
		if e.Code == "" {
			t.Error("found empty topic code")
		}
		if e.Confidence < 0 || e.Confidence > 1 {
			t.Errorf("topic %s: confidence out of range: %f", e.Code, e.Confidence)
		}
	}
	for _, e := range result.Industries {
		if e.Code == "" {
			t.Error("found empty industry code")
		}
		if e.Confidence < 0 || e.Confidence > 1 {
			t.Errorf("industry %s: confidence out of range: %f", e.Code, e.Confidence)
		}
	}
	for _, e := range result.Countries {
		if e.Code == "" {
			t.Error("found empty country code")
		}
		if e.Confidence < 0 || e.Confidence > 1 {
			t.Errorf("country %s: confidence out of range: %f", e.Code, e.Confidence)
		}
	}
	for _, e := range result.Technologies {
		if e.Code == "" {
			t.Error("found empty technology code")
		}
		if e.Confidence < 0 || e.Confidence > 1 {
			t.Errorf("technology %s: confidence out of range: %f", e.Code, e.Confidence)
		}
	}
	for _, e := range result.Organizations {
		if e.Code == "" {
			t.Error("found empty organization code")
		}
		if e.Confidence < 0 || e.Confidence > 1 {
			t.Errorf("organization %s: confidence out of range: %f", e.Code, e.Confidence)
		}
	}
}

func TestMockProvider_KnownCodes(t *testing.T) {
	p := NewMockProvider(true)

	article := Article{
		ID:    "article-1",
		Title: "Test Article",
	}

	result, err := p.Analyze(article)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	knownTopics := make(map[string]bool)
	for _, c := range mockTopicCodes {
		knownTopics[c] = true
	}
	for _, e := range result.Topics {
		if !knownTopics[e.Code] {
			t.Errorf("unknown topic code: %s", e.Code)
		}
	}

	knownIndustries := make(map[string]bool)
	for _, c := range mockIndustryCodes {
		knownIndustries[c] = true
	}
	for _, e := range result.Industries {
		if !knownIndustries[e.Code] {
			t.Errorf("unknown industry code: %s", e.Code)
		}
	}

	knownCountries := make(map[string]bool)
	for _, c := range mockCountryCodes {
		knownCountries[c] = true
	}
	for _, e := range result.Countries {
		if !knownCountries[e.Code] {
			t.Errorf("unknown country code: %s", e.Code)
		}
	}

	knownTechs := make(map[string]bool)
	for _, c := range mockTechnologyCodes {
		knownTechs[c] = true
	}
	for _, e := range result.Technologies {
		if !knownTechs[e.Code] {
			t.Errorf("unknown technology code: %s", e.Code)
		}
	}

	knownOrgs := make(map[string]bool)
	for _, c := range mockOrgCodes {
		knownOrgs[c] = true
	}
	for _, e := range result.Organizations {
		if !knownOrgs[e.Code] {
			t.Errorf("unknown organization code: %s", e.Code)
		}
	}
}

func TestMockResultJSON(t *testing.T) {
	k := &Knowledge{
		Topics: []Entity{
			{Code: "ml", Confidence: 0.9},
		},
		Industries:    []Entity{},
		Countries:     []Entity{},
		Technologies:  []Entity{},
		Organizations: []Entity{},
	}

	json := MockResultJSON(k)
	if json == "" {
		t.Error("expected non-empty JSON string")
	}
}