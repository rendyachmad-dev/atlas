package discover

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"atlas/internal/rss"

	"github.com/mmcdole/gofeed"
)

// DiscoverResult holds metadata about a discovered source.
type DiscoverResult struct {
	Domain           string     `json:"domain"`
	SourceURL        string     `json:"source_url"`
	RSSURL           string     `json:"rss_url,omitempty"`
	RSSAvailable     bool       `json:"rss_available"`
	Title            string     `json:"title,omitempty"`
	Description      string     `json:"description,omitempty"`
	Categories       []string   `json:"categories,omitempty"`
	LastUpdated      *time.Time `json:"last_updated,omitempty"`
	ArticleCount     int        `json:"article_count"`
	AvgArticlesPerDay float64   `json:"avg_articles_per_day"`
	Score            string     `json:"score"`        // "active", "inactive", "no_rss"
	Recommendation   string     `json:"recommendation"` // "layak", "tidak_layak"
	HTTPStatus       int        `json:"http_status"`
	Error            string     `json:"error,omitempty"`
}

// commonRSSPaths is the ordered list of paths to probe for RSS feeds.
var commonRSSPaths = []string{
	"/feed",
	"/rss",
	"/rss.xml",
	"/feed.xml",
	"/atom.xml",
	"/feeds/posts/default",
	"/getrss/all",
}

// subdomainRSSPrefixes is checked when path-based discovery fails.
var subdomainRSSPrefixes = []struct {
	prefix string
	suffix string
}{
	{"https://rss.", ""},
	{"https://feeds.", ""},
	{"https://feed.", ""},
	{"http://rss.", ""},
}

// autoDiscoveryPattern matches <link> tags for RSS/Atom auto-discovery.
var autoDiscoveryPattern = regexp.MustCompile(
	`<link[^>]*?(?:type\s*=\s*["']application/(?:rss|atom)\+xml["'][^>]*?href\s*=\s*["']([^"']+)["']|href\s*=\s*["']([^"']+)["'][^>]*?type\s*=\s*["']application/(?:rss|atom)\+xml["'])`,
)

// Discover probes a domain for RSS feeds and returns metadata.
func Discover(ctx context.Context, rawURL string) (*DiscoverResult, error) {
	// Normalize URL.
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	result := &DiscoverResult{
		Domain:    parsed.Host,
		SourceURL: fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host),
	}

	// Step 1: Try common RSS paths.
	for _, path := range commonRSSPaths {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		feedURL := fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, path)
		feed, httpStatus, err := tryFetchFeed(ctx, feedURL)
		if err == nil && feed != nil {
			result.RSSURL = feedURL
			result.RSSAvailable = true
			result.HTTPStatus = httpStatus
			fillFromFeed(result, feed)
			scoreResult(result)
			return result, nil
		}

		if httpStatus > 0 {
			result.HTTPStatus = httpStatus
		}
	}

	// Step 1.5: Try subdomain-based RSS (rss.domain.com, feeds.domain.com).
	if !result.RSSAvailable {
		probeSubdomains(ctx, parsed, result)
		if result.RSSAvailable {
			scoreResult(result)
			return result, nil
		}
	}

	// Step 2: Try auto-discovery from homepage HTML.
	if !result.RSSAvailable {
		feedURL, httpStatus := tryAutoDiscovery(ctx, parsed)
		if feedURL != "" {
			result.HTTPStatus = httpStatus
			feed, _, err := tryFetchFeed(ctx, feedURL)
			if err == nil && feed != nil {
				result.RSSURL = feedURL
				result.RSSAvailable = true
				fillFromFeed(result, feed)
				scoreResult(result)
				return result, nil
			}
		} else if httpStatus > 0 {
			result.HTTPStatus = httpStatus
		}

		// Step 3: Try fetching homepage to at least get title.
		if result.HTTPStatus == 0 {
			status := tryFetchHomepage(ctx, parsed, result)
			result.HTTPStatus = status
		}
	}

	scoreResult(result)
	return result, nil
}

func tryFetchFeed(ctx context.Context, feedURL string) (*gofeed.Feed, int, error) {
	// Use a HEAD request first to check availability.
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, feedURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "AtlasRSS/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	resp.Body.Close()

	// Accept 200 + redirect statuses — the GET might follow redirect to real feed.
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusMovedPermanently &&
		resp.StatusCode != http.StatusFound &&
		resp.StatusCode != http.StatusTemporaryRedirect &&
		resp.StatusCode != http.StatusPermanentRedirect {
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// HEAD OK — fetch full feed.
	feed, err := rss.FetchFeed(ctx, feedURL)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return feed, resp.StatusCode, nil
}

func tryAutoDiscovery(ctx context.Context, parsed *url.URL) (string, int) {
	homepage := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, homepage, nil)
	if err != nil {
		return "", 0
	}
	req.Header.Set("User-Agent", "AtlasRSS/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", resp.StatusCode
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", resp.StatusCode
	}

	html := string(body)
	matches := autoDiscoveryPattern.FindStringSubmatch(html)
	if len(matches) > 0 {
		feedURL := matches[1]
		if feedURL == "" {
			feedURL = matches[2]
		}
		// Resolve relative URLs.
		if strings.HasPrefix(feedURL, "/") {
			feedURL = fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, feedURL)
		} else if !strings.HasPrefix(feedURL, "http") {
			feedURL = fmt.Sprintf("%s://%s/%s", parsed.Scheme, parsed.Host, strings.TrimPrefix(feedURL, "/"))
		}
		return feedURL, resp.StatusCode
	}

	return "", resp.StatusCode
}

func tryFetchHomepage(ctx context.Context, parsed *url.URL, result *DiscoverResult) int {
	homepage := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, homepage, nil)
	if err != nil {
		return 0
	}
	req.Header.Set("User-Agent", "AtlasRSS/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return resp.StatusCode
	}

	// Try to extract <title> tag.
	html := string(body)
	titleMatch := regexp.MustCompile(`<title[^>]*>([^<]+)</title>`).FindStringSubmatch(html)
	if len(titleMatch) > 1 {
		result.Title = strings.TrimSpace(titleMatch[1])
	}

	return resp.StatusCode
}

func probeSubdomains(ctx context.Context, parsed *url.URL, result *DiscoverResult) {
	host := strings.TrimPrefix(parsed.Host, "www.")
	for _, s := range subdomainRSSPrefixes {
		select {
		case <-ctx.Done():
			return
		default:
		}

		feedURL := s.prefix + host + s.suffix
		feed, httpStatus, err := tryFetchFeed(ctx, feedURL)
		if err == nil && feed != nil {
			result.RSSURL = feedURL
			result.RSSAvailable = true
			result.HTTPStatus = httpStatus
			fillFromFeed(result, feed)
			return
		}
		if httpStatus > 0 && result.HTTPStatus == 0 {
			result.HTTPStatus = httpStatus
		}
	}
}

func fillFromFeed(result *DiscoverResult, feed *gofeed.Feed) {
	if feed.Title != "" {
		result.Title = feed.Title
	}
	if feed.Description != "" {
		result.Description = feed.Description
	}

	// Extract categories.
	catSet := make(map[string]bool)
	for _, cat := range feed.Categories {
		if cat != "" && !catSet[cat] {
			catSet[cat] = true
			result.Categories = append(result.Categories, cat)
		}
	}
	if len(result.Categories) == 0 {
		for _, item := range feed.Items {
			for _, cat := range item.Categories {
				if cat != "" && !catSet[cat] {
					catSet[cat] = true
					result.Categories = append(result.Categories, cat)
				}
			}
			if len(result.Categories) >= 10 {
				break
			}
		}
	}
	if len(result.Categories) > 10 {
		result.Categories = result.Categories[:10]
	}

	// Article count and timing.
	result.ArticleCount = len(feed.Items)
	if result.ArticleCount > 0 {
		var newest, oldest *time.Time
		for _, item := range feed.Items {
			if item.PublishedParsed != nil {
				if newest == nil || item.PublishedParsed.After(*newest) {
					newest = item.PublishedParsed
				}
				if oldest == nil || item.PublishedParsed.Before(*oldest) {
					oldest = item.PublishedParsed
				}
			}
			if item.UpdatedParsed != nil {
				if newest == nil || item.UpdatedParsed.After(*newest) {
					newest = item.UpdatedParsed
				}
				if oldest == nil || item.UpdatedParsed.Before(*oldest) {
					oldest = item.UpdatedParsed
				}
			}
		}

		if newest != nil {
			result.LastUpdated = newest
		}

		if oldest != nil && newest != nil && !newest.Equal(*oldest) {
			days := newest.Sub(*oldest).Hours() / 24
			if days > 0 {
				result.AvgArticlesPerDay = math.Round(float64(result.ArticleCount)/days*10) / 10
			} else {
				result.AvgArticlesPerDay = float64(result.ArticleCount)
			}
		} else if result.ArticleCount > 0 {
			result.AvgArticlesPerDay = 1.0
		}
	}
}

func scoreResult(result *DiscoverResult) {
	if !result.RSSAvailable {
		result.Score = "no_rss"
		result.Recommendation = "tidak_layak"
		return
	}

	if result.LastUpdated != nil {
		daysSince := time.Since(*result.LastUpdated).Hours() / 24
		if daysSince <= 7 {
			result.Score = "active"
		} else if daysSince <= 30 {
			result.Score = "moderate"
		} else {
			result.Score = "inactive"
		}
	} else {
		result.Score = "unknown"
	}

	switch {
	case result.Score == "active" && result.AvgArticlesPerDay >= 0.3:
		result.Recommendation = "layak"
	case result.Score == "active":
		result.Recommendation = "layak"
	case result.Score == "moderate" && result.AvgArticlesPerDay >= 1:
		result.Recommendation = "layak"
	default:
		result.Recommendation = "tidak_layak"
	}
}