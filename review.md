# PostgreSQL Performance Review — rss_atlas

## 1. Missing Indexes

### Critical

| Table | Missing Index | Query Pattern | Impact |
|-------|--------------|---------------|--------|
| `articles` | `(source_id, status)` | Collector: "get articles from source X that are still 'new'" | Seq scan on every fetch cycle |
| `articles` | `(published_at, status)` | Trend Engine: "find articles published in last 24h with status='analyzed'" | Seq scan 10k+ rows daily |
| `sources` | `(last_fetch) WHERE is_active` | Scheduler: "which sources are stale?" | Seq scan all sources every cycle |
| `ai_analysis` | `(sentiment, time_horizon)` | Dashboard: "filter by sentiment + time horizon" | Seq scan 3.6M+ rows |
| `ai_analysis` | `(generated_at DESC)` | Dashboard: "most recent analyses" | Sort + seq scan |
| `opportunities` | `(created_at DESC)` | Dashboard: "latest opportunities" | Sort + seq scan |

### Moderate

| Table | Missing Index | Why |
|-------|--------------|-----|
| `articles` | `GIN (to_tsvector('english', title \|\| ' ' \|\| content))` | Full-text search on 100k+ articles. No tsvector = sequential scan for every search query |
| `fetch_logs` | `(source_id, started_at DESC)` | "most recent fetch for source X" — current index on source_id alone leaves filter on started_at unoptimized |
| `trend_signals` | UNIQUE `(signal_date, topic_keywords)` | Same signal can be inserted twice. No dedup protection |

### Low

| Table | Missing Index | Why |
|-------|--------------|-----|
| `trend_article_links` | `(relevance_score DESC)` | If dashboard sorts by relevance |
| `opportunity_trends` | `(strength DESC)` | If dashboard sorts by strength |

---

## 2. Missing Constraints — Normalization Issues

### Enums vs VARCHAR

5 columns use VARCHAR where CHECK or enum should enforce domain:

| Table | Column | Current | Values | Fix |
|-------|--------|---------|--------|-----|
| `ai_analysis` | `sentiment` | VARCHAR(20) no CHECK | positive/negative/neutral/mixed | Add CHECK or enum |
| `ai_analysis` | `time_horizon` | VARCHAR(20) no CHECK | short-term/mid-term/long-term | Add CHECK or enum |
| `fetch_logs` | `status` | VARCHAR(20) no CHECK | success/error/partial | Add CHECK or enum |
| `opportunities` | `market_scope` | VARCHAR(50) no CHECK | local/regional/national/global | Add CHECK or enum |
| `opportunities` | `estimated_maturity` | VARCHAR(20) no CHECK | idea/prototype/early-stage/mature | Add CHECK or enum |

**Cardinality proof:** sentiment = 4 values, time_horizon = 3, market_scope = 3, estimated_maturity = 4. All low-cardinality, perfect for enum.

### Polymorphic Anti-Pattern in `user_bookmarks`

```sql
CREATE TABLE user_bookmarks (
    user_id         UUID NOT NULL,
    article_id      UUID NULLABLE,       -- anti-pattern
    opportunity_id  UUID NULLABLE,       -- anti-pattern
    ...
);
```

**Problem:** Two nullable FKs where exactly one must be non-null. Enforceable only via CHECK constraint (`(article_id IS NOT NULL) != (opportunity_id IS NOT NULL)`). Missing this CHECK = data integrity gap.

**Fix:** Either:
- Add CHECK constraint: `CHECK ((article_id IS NOT NULL) != (opportunity_id IS NOT NULL))`
- Or split into two tables: `user_article_bookmarks` + `user_opportunity_bookmarks`

### Missing CHECK on `content_hash`

`articles.content_hash` is VARCHAR(64) with no UNIQUE constraint. Two articles with identical content can exist. **Dedup is the entire reason this column exists.** Without a UNIQUE constraint (or at least a partial unique index WHERE status != 'archived'), the dedup strategy is enforcement-free.

### Missing CHECK on `sources.fetch_interval`

`fetch_interval` is INTEGER DEFAULT 30. No CHECK for `> 0`. A value of 0 or negative would crash the scheduler with infinite loops.

---

## 3. Unnecessary or Redundant

### `trend_signals` as Physical Table

`trend_signals` is a materialized aggregation of `ai_analysis`. Should be a PostgreSQL **MATERIALIZED VIEW** with `REFRESH MATERIALIZED VIEW CONCURRENTLY`, not a manually managed table.

**Savings:** Remove manual INSERT logic from application code. PostgreSQL handles staleness tracking. CONCURRENTLY refresh doesn't block reads.

### Duplicate Indexes

| Index | Redundant because | Action |
|-------|------------------|--------|
| `idx_analysis_queue_status` | High-cardinality guess: most rows are 'completed'. Partial index `WHERE status = 'queued'` already covers the hot path | Consider dropping if queued rows are < 1% of table |
| `idx_articles_language` | Only 2 values (en/id). 50% selectivity. B-tree useless. | Drop — only useful as last column of composite index |
| `idx_opportunities_maturity` | Only 4 values. 25% selectivity. | Drop — only useful as last column of composite index |

### `articles` Duplicate `external_id` Column

`external_id` is VARCHAR(512) with no UNIQUE constraint. It's documented as "source-side ID for idempotent re-fetch" but has no index or constraint. Either:
- Add UNIQUE (source_id, external_id) — actually enforce idempotency
- Or drop the column entirely

---

## 4. Scalability Issues

### 4.1 UUID v4 B-Tree Index Fragmentation

All 18 tables use `gen_random_uuid()` (UUID v4) as PK. UUID v4 is random, causing B-tree index page splits and bloat at scale.

**At 3.6M articles/year:** Index on `articles_pkey` will be ~40% larger than necessary. INSERT throughput degrades as index grows.

**Fix:** Migrate to **UUID v7** (time-ordered) or use `gen_random_uuid()` with `pg_collkey` or simple BIGSERIAL for PKs with UUID as business key only.

**Priority:** Medium. Not urgent until >1M rows.

### 4.2 No Table Partitioning

At projected scale:
- `articles`: 3.6M rows/year × 200KB avg = 720GB/year
- `ai_analysis`: 3.6M rows/year × 2KB avg = 7.2GB/year
- `fetch_logs`: 17M rows/year × 200 bytes = 3.4GB/year

**Without partitioning:**
- `DELETE FROM articles WHERE published_at < now() - interval '6 months'` causes massive vacuum + index bloat
- Full table scans on 3.6M+ rows for routine queries
- No partition pruning

**Fix:** Partition by month on `published_at` (articles) and `created_at` (fetch_logs, analysis_queue). Use declarative partitioning:

```sql
CREATE TABLE articles (...) PARTITION BY RANGE (published_at);
CREATE TABLE articles_2026_07 PARTITION OF articles
  FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
```

### 4.3 No Full-Text Search Capability

Dashboard will need to search articles by title + content. Without a `tsvector` column + GIN index, every search is a sequential scan scanning 100% of rows.

**Fix:**
```sql
ALTER TABLE articles ADD COLUMN search_vector tsvector
  GENERATED ALWAYS AS (to_tsvector('simple', coalesce(title,'') || ' ' || coalesce(content,''))) STORED;
CREATE INDEX idx_articles_search ON articles USING GIN (search_vector);
```

### 4.4 No Data Lifecycle Management

| Table | Projected Annual Growth | Has Cleanup? |
|-------|------------------------|--------------|
| `fetch_logs` | 17M rows | No |
| `analysis_queue` | 3.6M rows (completed) | No |
| `trend_signals` | 10k rows | No |

**Fix:** Add retention policies:
- `fetch_logs`: auto-delete after 30 days (or aggregate to daily summary, then delete raw)
- `analysis_queue`: auto-delete completed rows after 7 days
- Use pg_cron or a background worker

### 4.5 `content_hash` Not Unique

200 articles with same content → 200 rows with same hash. The dedup column is unenforced. At 10k articles/day, even 1% duplicate rate = 100 wasted analysis jobs/day.

**Fix:** `CREATE UNIQUE INDEX uq_articles_content_hash ON articles (content_hash) WHERE content_hash IS NOT NULL;`

---

## 5. Performance Bottlenecks

### 5.1 Articles — TOAST Bloat from `content`

`articles.content` is TEXT, stored inline by default. 10k articles/day × 2000 chars avg = 20MB/day of content. PG's TOAST mechanism kicks in at ~2KB, but each TOAST value still requires I/O on read.

**At 3.6M rows:** Reading 100 articles for trend analysis = 100 TOAST fetches.

**Mitigation:** 
- Query only `title` and `summary` where possible (already in ai_analysis)
- Use `pg_column_size()` to monitor TOAST usage
- Consider external storage for content > 10KB (S3 with pointer in DB)

### 5.2 Analysis Queue Worker Query

The dequeue pattern:
```sql
SELECT * FROM analysis_queue
WHERE status = 'queued'
ORDER BY priority DESC, created_at ASC
LIMIT 10
FOR UPDATE SKIP LOCKED;
```

Current index `idx_analysis_queue_priority (priority DESC, created_at ASC)` does NOT include `status`. The partial index `idx_analysis_queue_scheduled` covers scheduled_at but not priority ordering.

**Fix:** Replace with composite partial index:
```sql
CREATE INDEX idx_analysis_queue_dequeue ON analysis_queue (priority DESC, created_at ASC)
  WHERE status = 'queued';
```
Drop `idx_analysis_queue_priority` and `idx_analysis_queue_status` — this single index covers the hot path.

### 5.3 Dashboard Aggregation Queries

Dashboard queries like "count opportunities by maturity" or "avg urgency by time_horizon" will scan the full table.

**Mitigation:** Create **materialized views** for common dashboard queries:
- `mv_opportunity_funnel` — COUNT, AVG scores grouped by maturity + market_scope
- `mv_daily_analysis_volume` — COUNT, AVG scores by day + sentiment
- `mv_source_health` — fetch success rate, latency p50/p95 per source

Refresh via `REFRESH MATERIALIZED VIEW CONCURRENTLY` on a schedule (5-15 min depending on freshness requirement).

### 5.4 `ai_analysis` — 1:1 with `articles` but Queried Separately

Every dashboard view that shows articles with analysis performs a JOIN. With 3.6M rows on both sides, this becomes expensive.

**Fix:** Consider adding commonly-queried analysis fields directly to `articles`:
- `articles.urgency` (INT 1-10) — avoids JOIN for sorting
- `articles.sentiment` (VARCHAR) — avoids JOIN for filtering
- `articles.analysis_summary` (TEXT) — avoids JOIN for list view

This is denormalization, justified by the 1:1 relationship and read-heavy dashboard pattern.

---

## 6. Summary — Priority Order

| Priority | Issue | Impact | Effort |
|----------|-------|--------|--------|
| **P0** | Missing `content_hash` UNIQUE | Dedup broken, wasted AI cost | 5 min |
| **P0** | Missing `(source_id, status)` index on articles | Seq scan every fetch cycle | 5 min |
| **P0** | Missing `last_fetch` index on sources | Scheduler scans all sources | 5 min |
| **P1** | Missing full-text search index | Dashboard search = seq scan | 15 min |
| **P1** | Polymorphic `user_bookmarks` without CHECK | Data integrity gap | 5 min |
| **P1** | VARCHAR columns without CHECK/enum | Domain drift, data quality | 15 min |
| **P1** | `analysis_queue` dequeue index suboptimal | Queue worker slowdown | 5 min |
| **P2** | No partitioning | Vacuum bloat at scale | 1 hour |
| **P2** | UUID v4 fragmentation | Index bloat at >1M rows | 2 hours (migration) |
| **P2** | `trend_signals` as physical table | Manual sync logic | 30 min |
| **P3** | `external_id` unenforced | Dead column | 5 min |
| **P3** | No data lifecycle management | Storage growth unbounded | 1 hour |
| **P3** | Dashboard materialized views | Slow aggregation queries | 2 hours |
| **P3** | Denormalize urgency/sentiment into articles | Reduces JOINs on dashboard | 30 min |