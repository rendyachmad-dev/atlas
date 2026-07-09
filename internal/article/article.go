package article

import (
	"context"
	"fmt"
	"time"

	"atlas/internal/industry"
	"atlas/internal/technology"
	"atlas/internal/topic"
	"atlas/internal/country"
	"atlas/internal/organization"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SourceArticleCount holds article count per source.
type SourceArticleCount struct {
	SourceID   string
	SourceName string
	Total      int
	New        int
	Analyzed   int
}

// FetchLogSummary holds recent fetch log data for display.
type FetchLogSummary struct {
	SourceName      string
	Status          string
	ArticlesFetched int
	ArticlesNew     int
	ErrorMessage    *string
	StartedAt       time.Time
	CompletedAt     *time.Time
}

// ReportItem is a full report entry joining article + analysis + opportunity + trend.
type ReportItem struct {
	Title          string
	SummaryShort   *string
	SummaryMedium  *string
	BusinessImpact *string
	Urgency        *int
	TimeHorizon    *string
	RiskScore      *int
	Confidence     *float32
	Opportunity    *string
	OpportunityDesc *string
	Trend          *string
	SourceName     string
	PublishedAt    *time.Time
	ArticleURL     string
}

// Article represents a row from the articles table.
type Article struct {
	ID          string     `json:"id"`
	SourceID    string     `json:"source_id"`
	ExternalID  *string    `json:"external_id,omitempty"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Author      *string    `json:"author,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Content     *string    `json:"content,omitempty"`
	ContentHash string     `json:"content_hash"`
	Language    string     `json:"language"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// FetchLog represents a row from the fetch_logs table.
type FetchLog struct {
	ID              string     `json:"id"`
	SourceID        string     `json:"source_id"`
	Status          string     `json:"status"`
	ArticlesFetched int        `json:"articles_fetched"`
	ArticlesNew     int        `json:"articles_new"`
	ErrorMessage    *string    `json:"error_message,omitempty"`
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// Insert inserts a new article. If the URL already exists, it's a no-op.
// Returns the article ID, or empty string if the article already existed.
func Insert(ctx context.Context, pool *pgxpool.Pool, a *Article) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO articles (source_id, external_id, title, url, author,
		                      published_at, content, content_hash, language, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (url) DO NOTHING
		RETURNING id
	`,
		a.SourceID, a.ExternalID, a.Title, a.URL, a.Author,
		a.PublishedAt, a.Content, a.ContentHash, a.Language, a.Status,
	).Scan(&id)

	if err != nil {
		// No rows means the URL already existed — not an error.
		return "", nil
	}
	return id, nil
}

// CreateFetchLog inserts a new fetch log with status 'running'.
func CreateFetchLog(ctx context.Context, pool *pgxpool.Pool, sourceID string) (*FetchLog, error) {
	fl := &FetchLog{}
	err := pool.QueryRow(ctx, `
		INSERT INTO fetch_logs (source_id, status, started_at)
		VALUES ($1, 'running', now())
		RETURNING id, source_id, status, articles_fetched, articles_new,
		          error_message, started_at, completed_at, created_at
	`, sourceID).Scan(
		&fl.ID, &fl.SourceID, &fl.Status, &fl.ArticlesFetched, &fl.ArticlesNew,
		&fl.ErrorMessage, &fl.StartedAt, &fl.CompletedAt, &fl.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create fetch log: %w", err)
	}
	return fl, nil
}

// CompleteFetchLog updates a fetch log with final status and counts.
// CountArticlesBySource returns article counts grouped by source.
func CountArticlesBySource(ctx context.Context, pool *pgxpool.Pool) ([]SourceArticleCount, error) {
	rows, err := pool.Query(ctx, `
		SELECT s.id, s.name,
		       COUNT(a.id) AS total,
		       COUNT(*) FILTER (WHERE a.status = 'new') AS new,
		       COUNT(*) FILTER (WHERE a.status = 'analyzed') AS analyzed
		FROM sources s
		LEFT JOIN articles a ON a.source_id = s.id
		GROUP BY s.id, s.name
		ORDER BY s.name
	`)
	if err != nil {
		return nil, fmt.Errorf("count articles: %w", err)
	}
	defer rows.Close()

	var results []SourceArticleCount
	for rows.Next() {
		var r SourceArticleCount
		if err := rows.Scan(&r.SourceID, &r.SourceName, &r.Total, &r.New, &r.Analyzed); err != nil {
			return nil, fmt.Errorf("scan count: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// ListRecentFetchLogs returns the most recent fetch logs, ordered by time DESC.
func ListRecentFetchLogs(ctx context.Context, pool *pgxpool.Pool, limit int) ([]FetchLogSummary, error) {
	rows, err := pool.Query(ctx, `
		SELECT s.name, fl.status, fl.articles_fetched, fl.articles_new,
		       fl.error_message, fl.started_at, fl.completed_at
		FROM fetch_logs fl
		JOIN sources s ON s.id = fl.source_id
		ORDER BY fl.started_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list fetch logs: %w", err)
	}
	defer rows.Close()

	var logs []FetchLogSummary
	for rows.Next() {
		var l FetchLogSummary
		if err := rows.Scan(&l.SourceName, &l.Status, &l.ArticlesFetched, &l.ArticlesNew,
			&l.ErrorMessage, &l.StartedAt, &l.CompletedAt); err != nil {
			return nil, fmt.Errorf("scan fetch log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// ListReports returns analyzed articles with full analysis, opportunities, and trends.
func ListReports(ctx context.Context, pool *pgxpool.Pool, limit int) ([]ReportItem, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			a.title,
			ar.summary,
			NULL,
			ar.business_impact,
			ar.urgency,
			NULL,
			NULL,
			ar.confidence,
			o.title AS opp_title,
			o.description AS opp_desc,
			t.title AS trend_title,
			s.name AS source_name,
			a.published_at,
			a.url
		FROM articles a
		JOIN sources s ON s.id = a.source_id
		LEFT JOIN analysis_results ar ON ar.article_id = a.id
		LEFT JOIN opportunities o ON o.article_id = a.id
		LEFT JOIN trend_article_links tal ON tal.article_id = a.id
		LEFT JOIN trends t ON t.id = tal.trend_id
		WHERE a.status = 'analyzed'
		ORDER BY COALESCE(ar.urgency, 0) DESC, a.published_at DESC NULLS LAST
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	var items []ReportItem
	for rows.Next() {
		var r ReportItem
		if err := rows.Scan(
			&r.Title, &r.SummaryShort, &r.SummaryMedium,
			&r.BusinessImpact, &r.Urgency, &r.TimeHorizon,
			&r.RiskScore, &r.Confidence,
			&r.Opportunity, &r.OpportunityDesc,
			&r.Trend, &r.SourceName, &r.PublishedAt, &r.ArticleURL,
		); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

// ListRecentReports returns the most recent analyzed articles (no join on opp/trend),
// for lightweight display.
func ListRecentReports(ctx context.Context, pool *pgxpool.Pool, limit int) ([]ReportItem, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			a.title,
			ar.summary,
			NULL,
			ar.business_impact,
			ar.urgency,
			NULL,
			NULL,
			ar.confidence,
			NULL AS opp_title,
			NULL AS opp_desc,
			NULL AS trend_title,
			s.name AS source_name,
			a.published_at,
			a.url
		FROM articles a
		JOIN sources s ON s.id = a.source_id
		LEFT JOIN analysis_results ar ON ar.article_id = a.id
		WHERE a.status = 'analyzed'
		ORDER BY a.published_at DESC NULLS LAST
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent reports: %w", err)
	}
	defer rows.Close()

	var items []ReportItem
	for rows.Next() {
		var r ReportItem
		if err := rows.Scan(
			&r.Title, &r.SummaryShort, &r.SummaryMedium,
			&r.BusinessImpact, &r.Urgency, &r.TimeHorizon,
			&r.RiskScore, &r.Confidence,
			&r.Opportunity, &r.OpportunityDesc,
			&r.Trend, &r.SourceName, &r.PublishedAt, &r.ArticleURL,
		); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		items = append(items, r)
	}
	return items, rows.Err()
}

func CompleteFetchLog(ctx context.Context, pool *pgxpool.Pool, id, status string, fetched, new int, errMsg *string) error {
	_, err := pool.Exec(ctx, `
		UPDATE fetch_logs
		SET status = $1, articles_fetched = $2, articles_new = $3,
		    error_message = $4, completed_at = now()
		WHERE id = $5
	`, status, fetched, new, errMsg, id)
	if err != nil {
		return fmt.Errorf("complete fetch log: %w", err)
	}
	return nil
}

// WelcomeStats holds stats displayed on the welcome banner.
type WelcomeStats struct {
	AnalyzedYesterday int
	EmergingTrends    int
	Opportunities     int
}

// HighestConfidence holds the top-confidence analysis result.
type HighestConfidence struct {
	Title      string
	Confidence float32
}

// RecommendedAction holds the top-urgency article recommendation.
type RecommendedAction struct {
	Title   string
	Summary string
	Urgency int
}

// CountAnalyzedYesterday returns articles analyzed in the previous calendar day (WIB).
func CountAnalyzedYesterday(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM analysis_results
		WHERE created_at >= date_trunc('day', now() AT TIME ZONE 'Asia/Jakarta' - interval '1 day') AT TIME ZONE 'Asia/Jakarta'
		  AND created_at <  date_trunc('day', now() AT TIME ZONE 'Asia/Jakarta') AT TIME ZONE 'Asia/Jakarta'
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count analyzed yesterday: %w", err)
	}
	return count, nil
}

// CountEmergingTrends returns the number of trends with status 'emerging'.
func CountEmergingTrends(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var count int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM trends WHERE status = 'emerging'`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count emerging trends: %w", err)
	}
	return count, nil
}

// CountOpportunities returns the total number of opportunities.
func CountOpportunities(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var count int
	err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM opportunities`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count opportunities: %w", err)
	}
	return count, nil
}

// GetWelcomeStats returns all welcome banner stats.
func GetWelcomeStats(ctx context.Context, pool *pgxpool.Pool) (*WelcomeStats, error) {
	analyzed, err := CountAnalyzedYesterday(ctx, pool)
	if err != nil {
		return nil, err
	}
	trends, err := CountEmergingTrends(ctx, pool)
	if err != nil {
		return nil, err
	}
	opps, err := CountOpportunities(ctx, pool)
	if err != nil {
		return nil, err
	}
	return &WelcomeStats{
		AnalyzedYesterday: analyzed,
		EmergingTrends:    trends,
		Opportunities:     opps,
	}, nil
}

// GetHighestConfidence returns the article with the highest confidence analysis.
func GetHighestConfidence(ctx context.Context, pool *pgxpool.Pool) (*HighestConfidence, error) {
	var h HighestConfidence
	err := pool.QueryRow(ctx, `
		SELECT a.title, ar.confidence
		FROM analysis_results ar
		JOIN articles a ON a.id = ar.article_id
		ORDER BY ar.confidence DESC
		LIMIT 1
	`).Scan(&h.Title, &h.Confidence)
	if err != nil {
		return nil, fmt.Errorf("highest confidence: %w", err)
	}
	return &h, nil
}

// GetRecommendedAction returns the highest-urgency analyzed article for recommendations.
func GetRecommendedAction(ctx context.Context, pool *pgxpool.Pool) (*RecommendedAction, error) {
	var r RecommendedAction
	err := pool.QueryRow(ctx, `
		SELECT a.title, COALESCE(ar.summary, ''), ar.urgency
		FROM analysis_results ar
		JOIN articles a ON a.id = ar.article_id
		ORDER BY ar.urgency DESC
		LIMIT 1
	`).Scan(&r.Title, &r.Summary, &r.Urgency)
	if err != nil {
		return nil, fmt.Errorf("recommended action: %w", err)
	}
	return &r, nil
}
// ============================================================
// Article-Taxonomy query helpers (M2M junction lookups)
// ============================================================

// GetIndustries returns all industries linked to an article.
func GetIndustries(ctx context.Context, pool *pgxpool.Pool, articleID string) ([]industry.Industry, error) {
	rows, err := pool.Query(ctx, `
		SELECT ind.id, ind.code, ind.name, ind.description, ind.parent_id,
		       ind.is_active, ind.created_at, ind.updated_at
		FROM article_industries ai
		JOIN industries ind ON ind.id = ai.industry_id
		WHERE ai.article_id = $1
		ORDER BY ind.name
	`, articleID)
	if err != nil {
		return nil, fmt.Errorf("get article industries: %w", err)
	}
	defer rows.Close()

	var items []industry.Industry
	for rows.Next() {
		var i industry.Industry
		if err := rows.Scan(
			&i.ID, &i.Code, &i.Name, &i.Description,
			&i.ParentID, &i.IsActive, &i.CreatedAt, &i.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan industry: %w", err)
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// GetTopics returns all topics linked to an article.
func GetTopics(ctx context.Context, pool *pgxpool.Pool, articleID string) ([]topic.Topic, error) {
	rows, err := pool.Query(ctx, `
		SELECT t.id, t.code, t.name, t.description, t.parent_id,
		       t.is_active, t.created_at, t.updated_at
		FROM article_topics at
		JOIN topics t ON t.id = at.topic_id
		WHERE at.article_id = $1
		ORDER BY t.name
	`, articleID)
	if err != nil {
		return nil, fmt.Errorf("get article topics: %w", err)
	}
	defer rows.Close()

	var items []topic.Topic
	for rows.Next() {
		var t topic.Topic
		if err := rows.Scan(
			&t.ID, &t.Code, &t.Name, &t.Description,
			&t.ParentID, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan topic: %w", err)
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

// GetCountries returns all countries linked to an article.
func GetCountries(ctx context.Context, pool *pgxpool.Pool, articleID string) ([]country.Country, error) {
	rows, err := pool.Query(ctx, `
		SELECT c.id, c.code, c.alpha3, c.numeric_code, c.name, c.official_name,
		       c.region, c.subregion, c.is_active, c.created_at
		FROM article_countries ac
		JOIN countries c ON c.id = ac.country_id
		WHERE ac.article_id = $1
		ORDER BY c.name
	`, articleID)
	if err != nil {
		return nil, fmt.Errorf("get article countries: %w", err)
	}
	defer rows.Close()

	var items []country.Country
	for rows.Next() {
		var c country.Country
		if err := rows.Scan(
			&c.ID, &c.Code, &c.Alpha3, &c.NumericCode,
			&c.Name, &c.OfficialName, &c.Region, &c.Subregion,
			&c.IsActive, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan country: %w", err)
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

// GetTechnologies returns all technologies linked to an article.
func GetTechnologies(ctx context.Context, pool *pgxpool.Pool, articleID string) ([]technology.Technology, error) {
	rows, err := pool.Query(ctx, `
		SELECT t.id, t.code, t.name, t.description, t.parent_id,
		       t.is_active, t.created_at, t.updated_at
		FROM article_technologies at
		JOIN technologies t ON t.id = at.technology_id
		WHERE at.article_id = $1
		ORDER BY t.name
	`, articleID)
	if err != nil {
		return nil, fmt.Errorf("get article technologies: %w", err)
	}
	defer rows.Close()

	var items []technology.Technology
	for rows.Next() {
		var t technology.Technology
		if err := rows.Scan(
			&t.ID, &t.Code, &t.Name, &t.Description,
			&t.ParentID, &t.IsActive, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan technology: %w", err)
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

// GetOrganizations returns all organizations linked to an article.
func GetOrganizations(ctx context.Context, pool *pgxpool.Pool, articleID string) ([]organization.Organization, error) {
	rows, err := pool.Query(ctx, `
		SELECT o.id, o.code, o.name, o.type, o.description, o.country_code,
		       o.is_active, o.created_at, o.updated_at
		FROM article_organizations ao
		JOIN organizations o ON o.id = ao.organization_id
		WHERE ao.article_id = $1
		ORDER BY o.name
	`, articleID)
	if err != nil {
		return nil, fmt.Errorf("get article organizations: %w", err)
	}
	defer rows.Close()

	var items []organization.Organization
	for rows.Next() {
		var o organization.Organization
		if err := rows.Scan(
			&o.ID, &o.Code, &o.Name, &o.Type, &o.Description,
			&o.CountryCode, &o.IsActive, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan organization: %w", err)
		}
		items = append(items, o)
	}
	return items, rows.Err()
}
