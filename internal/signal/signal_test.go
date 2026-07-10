package signal

import (
	"context"
	"testing"

	"atlas/internal/extract"
)

func TestDetect_TopicMappings(t *testing.T) {
	d := NewRuleBasedDetector()
	ctx := context.Background()

	tests := []struct {
		name      string
		entity    extract.Entity
		category  string
		expect    SignalTypeCode
		expectLen int
	}{
		{"machine learning → ML advancement", extract.Entity{Code: "machine-learning", Confidence: 0.9}, CatTopics, SignalMLAdvancement, 1},
		{"fintech → fintech innovation", extract.Entity{Code: "fintech", Confidence: 0.8}, CatTopics, SignalFintechInnovation, 1},
		{"cybersecurity → cybersecurity event", extract.Entity{Code: "cybersecurity", Confidence: 0.85}, CatTopics, SignalCybersecurityEvent, 1},
		{"renewable-energy → renewable energy", extract.Entity{Code: "renewable-energy", Confidence: 0.9}, CatTopics, SignalRenewableEnergy, 1},
		{"biotechnology → biotech breakthrough", extract.Entity{Code: "biotechnology", Confidence: 0.85}, CatTopics, SignalBiotechBreakthrough, 1},
		{"cloud-computing → cloud adoption", extract.Entity{Code: "cloud-computing", Confidence: 0.8}, CatTopics, SignalCloudAdoption, 1},
		{"ai-regulation → AI regulation", extract.Entity{Code: "ai-regulation", Confidence: 0.9}, CatTopics, SignalAIRegulation, 1},
		{"supply-chain-mfg → supply chain event", extract.Entity{Code: "supply-chain-mfg", Confidence: 0.85}, CatTopics, SignalSupplyChainEvent, 1},
		{"generative-ai → gen AI innovation", extract.Entity{Code: "generative-ai", Confidence: 0.9}, CatTopics, SignalGenAIInnovation, 1},
		{"blockchain → blockchain adoption", extract.Entity{Code: "blockchain", Confidence: 0.8}, CatTopics, SignalBlockchainAdoption, 1},
		{"healthtech → healthtech innovation", extract.Entity{Code: "healthtech", Confidence: 0.85}, CatTopics, SignalHealthtechInnovation, 1},
		{"edtech → edtech growth", extract.Entity{Code: "edtech", Confidence: 0.8}, CatTopics, SignalEdtechGrowth, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			knowledge := &extract.Knowledge{}
			switch tt.category {
			case CatTopics:
				knowledge.Topics = []extract.Entity{tt.entity}
			case CatIndustries:
				knowledge.Industries = []extract.Entity{tt.entity}
			}

			signals, err := d.Detect(ctx, "article-1", knowledge)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(signals) != tt.expectLen {
				t.Errorf("expected %d signals, got %d", tt.expectLen, len(signals))
			}
			if len(signals) > 0 && signals[0].SignalTypeCode != tt.expect {
				t.Errorf("expected signal type %s, got %s", tt.expect, signals[0].SignalTypeCode)
			}
		})
	}
}

func TestDetect_IndustryMappings(t *testing.T) {
	d := NewRuleBasedDetector()
	ctx := context.Background()

	tests := []struct {
		code     string
		expected SignalTypeCode
	}{
		{"tech", SignalTechGrowth},
		{"finance", SignalFinancialActivity},
		{"healthcare", SignalHealthcareDev},
		{"energy", SignalEnergyDev},
		{"manufacturing", SignalManufacturingActivity},
		{"agriculture", SignalAgricultureDev},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			knowledge := &extract.Knowledge{
				Industries: []extract.Entity{{Code: tt.code, Confidence: 0.9}},
			}
			signals, err := d.Detect(ctx, "article-1", knowledge)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(signals) == 0 {
				t.Fatalf("expected at least 1 signal for %s", tt.code)
			}
			if signals[0].SignalTypeCode != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, signals[0].SignalTypeCode)
			}
		})
	}
}

func TestDetect_ConfidenceThreshold(t *testing.T) {
	d := NewRuleBasedDetector()
	ctx := context.Background()

	// Below threshold → no signal.
	knowledge := &extract.Knowledge{
		Topics: []extract.Entity{{Code: "machine-learning", Confidence: 0.1}},
	}
	signals, err := d.Detect(ctx, "article-1", knowledge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(signals) != 0 {
		t.Errorf("expected 0 signals below threshold, got %d", len(signals))
	}
}

func TestDetect_MultipleEntities(t *testing.T) {
	d := NewRuleBasedDetector()
	ctx := context.Background()

	knowledge := &extract.Knowledge{
		Topics: []extract.Entity{
			{Code: "machine-learning", Confidence: 0.9},
			{Code: "fintech", Confidence: 0.8},
			{Code: "cybersecurity", Confidence: 0.85},
		},
	}
	signals, err := d.Detect(ctx, "article-1", knowledge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Each topic should produce exactly 1 signal.
	if len(signals) != 3 {
		t.Errorf("expected 3 signals for 3 topics, got %d", len(signals))
	}

	// Verify all three types present.
	types := make(map[SignalTypeCode]bool)
	for _, s := range signals {
		types[s.SignalTypeCode] = true
	}
	if !types[SignalMLAdvancement] {
		t.Error("missing machine_learning signal")
	}
	if !types[SignalFintechInnovation] {
		t.Error("missing fintech_innovation signal")
	}
	if !types[SignalCybersecurityEvent] {
		t.Error("missing cybersecurity_event signal")
	}
}

func TestDetect_FullKnowledgeRecord(t *testing.T) {
	d := NewRuleBasedDetector()
	ctx := context.Background()

	knowledge := &extract.Knowledge{
		Topics:        []extract.Entity{{Code: "machine-learning", Confidence: 0.9}, {Code: "cloud-computing", Confidence: 0.8}},
		Industries:    []extract.Entity{{Code: "tech", Confidence: 0.95}},
		Countries:     []extract.Entity{{Code: "US", Confidence: 0.9}, {Code: "ID", Confidence: 0.8}},
		Technologies:  []extract.Entity{{Code: "ai", Confidence: 0.92}},
		Organizations: []extract.Entity{{Code: "microsoft", Confidence: 0.88}},
	}

	signals, err := d.Detect(ctx, "article-full", knowledge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(signals) == 0 {
		t.Fatal("expected at least 1 signal")
	}

	// Verify entity categories are correctly set.
	catCounts := make(map[string]int)
	for _, s := range signals {
		catCounts[s.EntityCategory]++
		catCounts["_total"]++
	}

	t.Logf("signals: %d total", len(signals))
	for _, s := range signals {
		t.Logf("  %s → %s (%.0f%%)", s.EntityCategory, s.SignalTypeCode, s.Confidence*100)
	}

	// Topics should produce 2 signals (ML + cloud).
	if catCounts[CatTopics] < 1 {
		t.Errorf("expected at least 1 topic signal, got %d", catCounts[CatTopics])
	}
	// Industries should produce at least 1 signal.
	if catCounts[CatIndustries] < 1 {
		t.Errorf("expected at least 1 industry signal, got %d", catCounts[CatIndustries])
	}
}

func TestDetect_UnknownEntityFallback(t *testing.T) {
	d := NewRuleBasedDetector()
	ctx := context.Background()

	// Unknown topic code — should not produce signals (no mapping for "unknown-tech").
	knowledge := &extract.Knowledge{
		Topics: []extract.Entity{{Code: "unknown-topic-code", Confidence: 0.9}},
	}
	signals, err := d.Detect(ctx, "article-1", knowledge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(signals) != 0 {
		t.Errorf("expected 0 signals for unknown topic, got %d", len(signals))
	}
}

func TestDetect_UnknownIndustryFallback(t *testing.T) {
	d := NewRuleBasedDetector()
	ctx := context.Background()

	// Unknown industry — has topic-like mappings but no industry mapping for "unknown".
	knowledge := &extract.Knowledge{
		Industries: []extract.Entity{{Code: "unknown-industry", Confidence: 0.9}},
	}
	signals, err := d.Detect(ctx, "article-1", knowledge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(signals) != 0 {
		t.Errorf("expected 0 signals for unknown industry, got %d", len(signals))
	}
}

func TestMemoryStore_InsertSignals(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	signals := []Signal{
		{ArticleID: "a1", SignalTypeCode: SignalMLAdvancement, EntityCode: "machine-learning", EntityCategory: CatTopics, Confidence: 0.9},
		{ArticleID: "a1", SignalTypeCode: SignalFintechInnovation, EntityCode: "fintech", EntityCategory: CatTopics, Confidence: 0.8},
	}

	count, err := store.InsertSignals(ctx, signals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 inserted, got %d", count)
	}
}

func TestMemoryStore_Idempotent(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	s := Signal{ArticleID: "a1", SignalTypeCode: SignalMLAdvancement, EntityCode: "machine-learning", EntityCategory: CatTopics, Confidence: 0.9}

	// Insert same signal twice.
	count1, err := store.InsertSignals(ctx, []Signal{s})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count1 != 1 {
		t.Errorf("expected 1 on first insert, got %d", count1)
	}

	count2, err := store.InsertSignals(ctx, []Signal{s})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count2 != 0 {
		t.Errorf("expected 0 on second insert (idempotent), got %d", count2)
	}

	// Still only 1 in store.
	all, err := store.GetSignalsByArticle(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 signal total, got %d", len(all))
	}
}

func TestMemoryStore_GetSignalsByArticle(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	store.InsertSignals(ctx, []Signal{
		{ArticleID: "a1", SignalTypeCode: SignalMLAdvancement, EntityCode: "ml", EntityCategory: CatTopics, Confidence: 0.9},
		{ArticleID: "a1", SignalTypeCode: SignalFintechInnovation, EntityCode: "fintech", EntityCategory: CatTopics, Confidence: 0.8},
		{ArticleID: "a2", SignalTypeCode: SignalCybersecurityEvent, EntityCode: "cybersec", EntityCategory: CatTopics, Confidence: 0.85},
	})

	signals, err := store.GetSignalsByArticle(ctx, "a1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(signals) != 2 {
		t.Errorf("expected 2 signals for a1, got %d", len(signals))
	}

	signals2, err := store.GetSignalsByArticle(ctx, "a2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(signals2) != 1 {
		t.Errorf("expected 1 signal for a2, got %d", len(signals2))
	}
}

func TestPipeline_ProcessOne(t *testing.T) {
	detector := NewRuleBasedDetector()
	store := NewMemoryStore()
	pipeline := NewPipeline(detector, store)
	ctx := context.Background()

	knowledge := &extract.Knowledge{
		Topics: []extract.Entity{
			{Code: "machine-learning", Confidence: 0.9},
			{Code: "fintech", Confidence: 0.8},
		},
	}

	count, err := pipeline.ProcessOne(ctx, "article-p1", knowledge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 new signals, got %d", count)
	}

	// Second call should be idempotent.
	count2, err := pipeline.ProcessOne(ctx, "article-p1", knowledge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count2 != 0 {
		t.Errorf("expected 0 new signals on idempotent call, got %d", count2)
	}
}

func TestPipeline_ProcessOne_EmptyKnowledge(t *testing.T) {
	detector := NewRuleBasedDetector()
	store := NewMemoryStore()
	pipeline := NewPipeline(detector, store)
	ctx := context.Background()

	knowledge := &extract.Knowledge{
		Topics:        []extract.Entity{},
		Industries:    []extract.Entity{},
		Countries:     []extract.Entity{},
		Technologies:  []extract.Entity{},
		Organizations: []extract.Entity{},
	}

	count, err := pipeline.ProcessOne(ctx, "article-empty", knowledge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 signals for empty knowledge, got %d", count)
	}
}

func TestDefaultMappings_NotEmpty(t *testing.T) {
	mappings := DefaultMappings()
	if len(mappings) == 0 {
		t.Fatal("expected non-empty default mappings")
	}
}

func TestSignalTypeCode_String(t *testing.T) {
	if string(SignalMLAdvancement) != "machine_learning" {
		t.Errorf("unexpected string: %s", SignalMLAdvancement)
	}
}