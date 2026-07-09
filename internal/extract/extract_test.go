package extract

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testPool is the shared database pool for integration tests.
// Set via DATABASE_URL or DATABASE_TEST_URL env var.
// Tests are skipped if no test DB is available.
var testPool *pgxpool.Pool
var testSourceID = "e0000000-0000-4000-8000-000000000001"
var testArticleID = "f0000000-0000-4000-8000-000000000001"

func TestMain(m *testing.M) {
	url := os.Getenv("DATABASE_TEST_URL")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}

	var err error
	testPool, err = pgxpool.New(context.Background(), url)
	if err != nil {
		log.Printf("WARNING: no test database available (%v) — skipping DB-backed tests", err)
		testPool = nil
		os.Exit(m.Run())
	}

	if err := testPool.Ping(context.Background()); err != nil {
		log.Printf("WARNING: cannot ping test database (%v) — skipping DB-backed tests", err)
		testPool.Close()
		testPool = nil
		os.Exit(m.Run())
	}

	code := m.Run()
	testPool.Close()
	os.Exit(code)
}

// requireDB skips the test if no test database is available.
func requireDB(t *testing.T) {
	if testPool == nil {
		t.Skip("skipping: no test database available (set DATABASE_URL or DATABASE_TEST_URL)")
	}
}

// setupTestData creates a test source and article in the database.
func setupTestData(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	_, err := testPool.Exec(ctx, `
		INSERT INTO sources (id, name, type, url, is_active, fetch_interval)
		VALUES ($1, 'Test Source', 'RSS', 'https://example.com/rss', true, 3600)
		ON CONFLICT (id) DO NOTHING
	`, testSourceID)
	if err != nil {
		t.Fatalf("create test source: %v", err)
	}

	_, err = testPool.Exec(ctx, `
		INSERT INTO articles (id, source_id, title, url, content, content_hash, language, status)
		VALUES ($1, $2, 'Test Article', 'https://example.com/article', 'Test content for extraction.', 'abc123', 'en', 'new')
		ON CONFLICT (id) DO NOTHING
	`, testArticleID, testSourceID)
	if err != nil {
		t.Fatalf("create test article: %v", err)
	}
}

// cleanupTestData removes test data from the database.
func cleanupTestData(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	_, _ = testPool.Exec(ctx, `DELETE FROM extraction_results WHERE article_id = $1`, testArticleID)
	_, _ = testPool.Exec(ctx, `DELETE FROM extraction_jobs WHERE article_id = $1`, testArticleID)
	_, _ = testPool.Exec(ctx, `DELETE FROM article_topics WHERE article_id = $1`, testArticleID)
	_, _ = testPool.Exec(ctx, `DELETE FROM article_industries WHERE article_id = $1`, testArticleID)
	_, _ = testPool.Exec(ctx, `DELETE FROM article_countries WHERE article_id = $1`, testArticleID)
	_, _ = testPool.Exec(ctx, `DELETE FROM article_technologies WHERE article_id = $1`, testArticleID)
	_, _ = testPool.Exec(ctx, `DELETE FROM article_organizations WHERE article_id = $1`, testArticleID)
	_, _ = testPool.Exec(ctx, `DELETE FROM articles WHERE id = $1`, testArticleID)
	_, _ = testPool.Exec(ctx, `DELETE FROM sources WHERE id = $1`, testSourceID)
}

// ============================================================
// ResolveAndInsert tests
// ============================================================

func TestResolveAndInsert_ValidEntities(t *testing.T) {
	requireDB(t)
	defer cleanupTestData(t)
	setupTestData(t)

	ctx := context.Background()

	k := &Knowledge{
		Topics: []Entity{
			{Code: "machine-learning", Confidence: 0.95},
			{Code: "cloud-computing", Confidence: 0.85},
		},
		Industries: []Entity{
			{Code: "tech", Confidence: 0.9},
		},
		Countries: []Entity{
			{Code: "US", Confidence: 0.8},
		},
		Technologies: []Entity{
			{Code: "ai", Confidence: 0.92},
		},
		Organizations: []Entity{
			{Code: "openai", Confidence: 0.88},
		},
	}

	err := ResolveAndInsert(ctx, testPool, testArticleID, k)
	if err != nil {
		t.Fatalf("ResolveAndInsert: %v", err)
	}

	var topicCount int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM article_topics WHERE article_id = $1`, testArticleID).Scan(&topicCount)
	if err != nil {
		t.Fatalf("count topics: %v", err)
	}
	if topicCount != 2 {
		t.Errorf("expected 2 topics, got %d", topicCount)
	}

	var indCount int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM article_industries WHERE article_id = $1`, testArticleID).Scan(&indCount)
	if err != nil {
		t.Fatalf("count industries: %v", err)
	}
	if indCount != 1 {
		t.Errorf("expected 1 industry, got %d", indCount)
	}

	var countryCount int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM article_countries WHERE article_id = $1`, testArticleID).Scan(&countryCount)
	if err != nil {
		t.Fatalf("count countries: %v", err)
	}
	if countryCount != 1 {
		t.Errorf("expected 1 country, got %d", countryCount)
	}

	var techCount int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM article_technologies WHERE article_id = $1`, testArticleID).Scan(&techCount)
	if err != nil {
		t.Fatalf("count technologies: %v", err)
	}
	if techCount != 1 {
		t.Errorf("expected 1 technology, got %d", techCount)
	}

	var orgCount int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM article_organizations WHERE article_id = $1`, testArticleID).Scan(&orgCount)
	if err != nil {
		t.Fatalf("count organizations: %v", err)
	}
	if orgCount != 1 {
		t.Errorf("expected 1 organization, got %d", orgCount)
	}
}

func TestResolveAndInsert_DuplicateEntities(t *testing.T) {
	requireDB(t)
	defer cleanupTestData(t)
	setupTestData(t)

	ctx := context.Background()

	k := &Knowledge{
		Topics: []Entity{
			{Code: "machine-learning", Confidence: 0.95},
		},
		Industries: []Entity{
			{Code: "tech", Confidence: 0.9},
		},
		Countries: []Entity{
			{Code: "US", Confidence: 0.8},
		},
		Technologies: []Entity{
			{Code: "ai", Confidence: 0.92},
		},
		Organizations: []Entity{
			{Code: "openai", Confidence: 0.88},
		},
	}

	err := ResolveAndInsert(ctx, testPool, testArticleID, k)
	if err != nil {
		t.Fatalf("first ResolveAndInsert: %v", err)
	}

	err = ResolveAndInsert(ctx, testPool, testArticleID, k)
	if err != nil {
		t.Fatalf("second ResolveAndInsert: %v", err)
	}

	var topicCount int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM article_topics WHERE article_id = $1`, testArticleID).Scan(&topicCount)
	if err != nil {
		t.Fatalf("count topics: %v", err)
	}
	if topicCount != 1 {
		t.Errorf("expected 1 topic (dedup), got %d", topicCount)
	}

	var indCount int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM article_industries WHERE article_id = $1`, testArticleID).Scan(&indCount)
	if err != nil {
		t.Fatalf("count industries: %v", err)
	}
	if indCount != 1 {
		t.Errorf("expected 1 industry (dedup), got %d", indCount)
	}
}

func TestResolveAndInsert_UnknownCodes(t *testing.T) {
	requireDB(t)
	defer cleanupTestData(t)
	setupTestData(t)

	ctx := context.Background()

	k := &Knowledge{
		Topics: []Entity{
			{Code: "nonexistent-topic-code", Confidence: 0.5},
		},
		Industries: []Entity{
			{Code: "nonexistent-industry", Confidence: 0.5},
		},
		Countries: []Entity{
			{Code: "XX", Confidence: 0.5},
		},
		Technologies: []Entity{
			{Code: "nonexistent-tech", Confidence: 0.5},
		},
		Organizations: []Entity{
			{Code: "nonexistent-org", Confidence: 0.5},
		},
	}

	err := ResolveAndInsert(ctx, testPool, testArticleID, k)
	if err != nil {
		t.Fatalf("ResolveAndInsert with unknown codes: %v", err)
	}

	var total int
	err = testPool.QueryRow(ctx, `
		SELECT (
			(SELECT COUNT(*) FROM article_topics WHERE article_id = $1) +
			(SELECT COUNT(*) FROM article_industries WHERE article_id = $1) +
			(SELECT COUNT(*) FROM article_countries WHERE article_id = $1) +
			(SELECT COUNT(*) FROM article_technologies WHERE article_id = $1) +
			(SELECT COUNT(*) FROM article_organizations WHERE article_id = $1)
		)`, testArticleID).Scan(&total)
	if err != nil {
		t.Fatalf("count all: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 total entities for unknown codes, got %d", total)
	}
}

// ============================================================
// Worker tests (retry, fail)
// ============================================================

// failThenSucceedProvider returns invalid entities on first call, valid on second.
type failThenSucceedProvider struct {
	firstCall bool
}

func (p *failThenSucceedProvider) Analyze(article Article) (*Knowledge, error) {
	if !p.firstCall {
		p.firstCall = true
		return &Knowledge{
			Topics: []Entity{
				{Code: "", Confidence: 0.5},
			},
			Industries:    []Entity{},
			Countries:     []Entity{},
			Technologies:  []Entity{},
			Organizations: []Entity{},
		}, nil
	}
	return &Knowledge{
		Topics: []Entity{
			{Code: "machine-learning", Confidence: 0.95},
		},
		Industries: []Entity{
			{Code: "tech", Confidence: 0.9},
		},
		Countries: []Entity{
			{Code: "US", Confidence: 0.8},
		},
		Technologies: []Entity{
			{Code: "ai", Confidence: 0.92},
		},
		Organizations: []Entity{
			{Code: "openai", Confidence: 0.88},
		},
	}, nil
}

// alwaysFailProvider returns entities that never pass validation.
type alwaysFailProvider struct{}

func (p *alwaysFailProvider) Analyze(article Article) (*Knowledge, error) {
	return &Knowledge{
		Topics: []Entity{
			{Code: "", Confidence: 0.5},
		},
		Industries:    []Entity{},
		Countries:     []Entity{},
		Technologies:  []Entity{},
		Organizations: []Entity{},
	}, nil
}

func TestWorker_RetryOnce(t *testing.T) {
	requireDB(t)
	defer cleanupTestData(t)
	setupTestData(t)

	ctx := context.Background()

	_, err := testPool.Exec(ctx, `
		INSERT INTO extraction_jobs (article_id, status, priority)
		VALUES ($1, 'queued', 5)
		ON CONFLICT (article_id) DO NOTHING
	`, testArticleID)
	if err != nil {
		t.Fatalf("create extraction job: %v", err)
	}
	defer func() {
		testPool.Exec(ctx, `DELETE FROM extraction_jobs WHERE article_id = $1`, testArticleID)
	}()

	store := NewJobStore(testPool)
	provider := &failThenSucceedProvider{}
	worker := NewWorker(store, provider, testPool, "", "")

	ok, err := worker.RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if !ok {
		t.Fatal("expected job to be processed (ok=true)")
	}

	var status string
	err = testPool.QueryRow(ctx, `SELECT status FROM extraction_jobs WHERE article_id = $1`, testArticleID).Scan(&status)
	if err != nil {
		t.Fatalf("get job status: %v", err)
	}
	if status != "completed" {
		t.Errorf("expected job status 'completed', got '%s'", status)
	}

	var topicCount int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM article_topics WHERE article_id = $1`, testArticleID).Scan(&topicCount)
	if err != nil {
		t.Fatalf("count topics: %v", err)
	}
	if topicCount != 1 {
		t.Errorf("expected 1 topic from retry, got %d", topicCount)
	}

	var resultCount int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM extraction_results WHERE article_id = $1`, testArticleID).Scan(&resultCount)
	if err != nil {
		t.Fatalf("count results: %v", err)
	}
	if resultCount != 1 {
		t.Errorf("expected 1 extraction result, got %d", resultCount)
	}
}

func TestWorker_FailedJob(t *testing.T) {
	requireDB(t)
	defer cleanupTestData(t)
	setupTestData(t)

	ctx := context.Background()

	_, err := testPool.Exec(ctx, `
		INSERT INTO extraction_jobs (article_id, status, priority)
		VALUES ($1, 'queued', 5)
		ON CONFLICT (article_id) DO NOTHING
	`, testArticleID)
	if err != nil {
		t.Fatalf("create extraction job: %v", err)
	}
	defer func() {
		testPool.Exec(ctx, `DELETE FROM extraction_jobs WHERE article_id = $1`, testArticleID)
	}()

	store := NewJobStore(testPool)
	provider := &alwaysFailProvider{}
	worker := NewWorker(store, provider, testPool, "", "")

	ok, err := worker.RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if !ok {
		t.Fatal("expected job to be processed (ok=true)")
	}

	var status string
	var errMsg *string
	err = testPool.QueryRow(ctx, `SELECT status, error_message FROM extraction_jobs WHERE article_id = $1`, testArticleID).Scan(&status, &errMsg)
	if err != nil {
		t.Fatalf("get job status: %v", err)
	}
	if status != "failed" {
		t.Errorf("expected job status 'failed', got '%s'", status)
	}
	if errMsg == nil || *errMsg == "" {
		t.Error("expected non-empty error message on failed job")
	} else {
		t.Logf("job error message: %s", *errMsg)
	}

	var total int
	err = testPool.QueryRow(ctx, `
		SELECT (
			(SELECT COUNT(*) FROM article_topics WHERE article_id = $1) +
			(SELECT COUNT(*) FROM article_industries WHERE article_id = $1) +
			(SELECT COUNT(*) FROM article_countries WHERE article_id = $1) +
			(SELECT COUNT(*) FROM article_technologies WHERE article_id = $1) +
			(SELECT COUNT(*) FROM article_organizations WHERE article_id = $1)
		)`, testArticleID).Scan(&total)
	if err != nil {
		t.Fatalf("count all: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 entities on failed job, got %d", total)
	}
}

func TestWorker_NoJobs(t *testing.T) {
	requireDB(t)

	store := NewJobStore(testPool)
	provider := NewMockProvider(true)
	worker := NewWorker(store, provider, testPool, "", "")

	ctx := context.Background()
	ok, err := worker.RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if ok {
		t.Fatal("expected false when no jobs available")
	}
}

func TestWorker_RunAll(t *testing.T) {
	requireDB(t)
	defer cleanupTestData(t)
	setupTestData(t)

	ctx := context.Background()

	_, err := testPool.Exec(ctx, `
		INSERT INTO extraction_jobs (article_id, status, priority)
		VALUES ($1, 'queued', 5)
		ON CONFLICT (article_id) DO NOTHING
	`, testArticleID)
	if err != nil {
		t.Fatalf("create extraction job: %v", err)
	}
	defer func() {
		testPool.Exec(ctx, `DELETE FROM extraction_jobs WHERE article_id = $1`, testArticleID)
	}()

	store := NewJobStore(testPool)
	provider := NewMockProvider(true)
	worker := NewWorker(store, provider, testPool, "", "")

	processed, err := worker.Run(ctx, 0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if processed != 1 {
		t.Errorf("expected 1 processed job, got %d", processed)
	}

	var resultCount int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM extraction_results WHERE article_id = $1`, testArticleID).Scan(&resultCount)
	if err != nil {
		t.Fatalf("count results: %v", err)
	}
	if resultCount != 1 {
		t.Errorf("expected 1 extraction result, got %d", resultCount)
	}
}

// ============================================================
// JobStore tests
// ============================================================

func TestJobStore_EnqueueNewArticles(t *testing.T) {
	requireDB(t)
	defer cleanupTestData(t)
	setupTestData(t)

	ctx := context.Background()

	store := NewJobStore(testPool)

	count, err := store.EnqueueNewArticles(ctx, 5)
	if err != nil {
		t.Fatalf("EnqueueNewArticles: %v", err)
	}
	if count == 0 {
		var status string
		err := testPool.QueryRow(ctx, `SELECT status FROM articles WHERE id = $1`, testArticleID).Scan(&status)
		if err != nil {
			t.Fatalf("get article status: %v", err)
		}
		if status == "new" {
			t.Errorf("expected 1 enqueued job for 'new' article, got 0")
		} else {
			t.Logf("article status is '%s', skipping enqueue count assertion", status)
		}
	}
	defer func() {
		testPool.Exec(ctx, `DELETE FROM extraction_jobs WHERE article_id = $1`, testArticleID)
	}()
}

func TestJobStore_ClaimNextJob(t *testing.T) {
	requireDB(t)
	defer cleanupTestData(t)
	setupTestData(t)

	ctx := context.Background()

	_, err := testPool.Exec(ctx, `
		INSERT INTO extraction_jobs (article_id, status, priority)
		VALUES ($1, 'queued', 5)
		ON CONFLICT (article_id) DO NOTHING
	`, testArticleID)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	defer func() {
		testPool.Exec(ctx, `DELETE FROM extraction_jobs WHERE article_id = $1`, testArticleID)
	}()

	store := NewJobStore(testPool)
	job, err := store.ClaimNextJob(ctx, "test-worker")
	if err != nil {
		t.Fatalf("ClaimNextJob: %v", err)
	}
	if job == nil {
		t.Fatal("expected a job, got nil")
	}
	if job.ArticleID != testArticleID {
		t.Errorf("expected article %s, got %s", testArticleID, job.ArticleID)
	}
	if job.Status != "processing" {
		t.Errorf("expected status 'processing', got '%s'", job.Status)
	}

	_, err = store.ClaimNextJob(ctx, "test-worker-2")
	if err == nil {
		t.Error("expected error on second claim (no more jobs)")
	}
}

func TestJobStore_CompleteAndFail(t *testing.T) {
	requireDB(t)
	defer cleanupTestData(t)
	setupTestData(t)

	ctx := context.Background()

	_, err := testPool.Exec(ctx, `
		INSERT INTO extraction_jobs (id, article_id, status, priority)
		VALUES ('e0000000-0000-4000-8000-000000000001', $1, 'queued', 5)
		ON CONFLICT (id) DO NOTHING
	`, testArticleID)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	defer func() {
		testPool.Exec(ctx, `DELETE FROM extraction_jobs WHERE id = 'e0000000-0000-4000-8000-000000000001'`)
	}()

	store := NewJobStore(testPool)

	err = store.CompleteJob(ctx, "e0000000-0000-4000-8000-000000000001")
	if err != nil {
		t.Fatalf("CompleteJob: %v", err)
	}

	var status string
	err = testPool.QueryRow(ctx, `SELECT status FROM extraction_jobs WHERE id = 'e0000000-0000-4000-8000-000000000001'`).Scan(&status)
	if err != nil {
		t.Fatalf("get job status: %v", err)
	}
	if status != "completed" {
		t.Errorf("expected 'completed', got '%s'", status)
	}

	_, err = testPool.Exec(ctx, `UPDATE extraction_jobs SET status = 'queued' WHERE id = 'e0000000-0000-4000-8000-000000000001'`)
	if err != nil {
		t.Fatalf("reset job: %v", err)
	}

	err = store.FailJob(ctx, "e0000000-0000-4000-8000-000000000001", "test error message")
	if err != nil {
		t.Fatalf("FailJob: %v", err)
	}

	err = testPool.QueryRow(ctx, `SELECT status FROM extraction_jobs WHERE id = 'e0000000-0000-4000-8000-000000000001'`).Scan(&status)
	if err != nil {
		t.Fatalf("get job status: %v", err)
	}
	if status != "failed" {
		t.Errorf("expected 'failed', got '%s'", status)
	}
}

func TestJobStore_LoadArticleData(t *testing.T) {
	requireDB(t)
	defer cleanupTestData(t)
	setupTestData(t)

	ctx := context.Background()

	store := NewJobStore(testPool)
	article, err := store.LoadArticleData(ctx, testArticleID)
	if err != nil {
		t.Fatalf("LoadArticleData: %v", err)
	}
	if article == nil {
		t.Fatal("expected article data, got nil")
	}
	if article.Title != "Test Article" {
		t.Errorf("expected 'Test Article', got '%s'", article.Title)
	}
	if article.Source != "Test Source" {
		t.Errorf("expected 'Test Source', got '%s'", article.Source)
	}
}

// ============================================================
// Service tests
// ============================================================

func TestService_RunAll(t *testing.T) {
	requireDB(t)
	defer cleanupTestData(t)
	setupTestData(t)

	ctx := context.Background()

	_, err := testPool.Exec(ctx, `
		INSERT INTO extraction_jobs (article_id, status, priority)
		VALUES ($1, 'queued', 5)
		ON CONFLICT (article_id) DO NOTHING
	`, testArticleID)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	defer func() {
		testPool.Exec(ctx, `DELETE FROM extraction_jobs WHERE article_id = $1`, testArticleID)
	}()

	provider := NewMockProvider(true)
	svc := NewService(testPool, provider, "", "")

	processed, err := svc.RunAll(ctx, 0)
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if processed != 1 {
		t.Errorf("expected 1 processed job, got %d", processed)
	}
}

// ============================================================
// SaveResult tests
// ============================================================

func TestSaveResult(t *testing.T) {
	requireDB(t)
	defer cleanupTestData(t)
	setupTestData(t)

	ctx := context.Background()

	rawJSON := `{"topics":[{"code":"ml","confidence":0.9}]}`
	provider := "anthropic"
	model := "claude-sonnet-5"

	store := NewJobStore(testPool)
	result := &ExtractionResult{
		ArticleID:   testArticleID,
		RawJSON:     &rawJSON,
		LLMProvider: &provider,
		LLMModel:    &model,
	}

	err := store.SaveResult(ctx, result)
	if err != nil {
		t.Fatalf("SaveResult: %v", err)
	}

	var savedRaw *string
	var savedProvider *string
	err = testPool.QueryRow(ctx, `SELECT raw_json, llm_provider FROM extraction_results WHERE article_id = $1`, testArticleID).Scan(&savedRaw, &savedProvider)
	if err != nil {
		t.Fatalf("get result: %v", err)
	}
	if savedRaw == nil || *savedRaw != rawJSON {
		t.Errorf("raw_json mismatch: expected %s, got %v", rawJSON, *savedRaw)
	}
	if savedProvider == nil || *savedProvider != provider {
		t.Errorf("provider mismatch: expected %s, got %v", provider, *savedProvider)
	}
}