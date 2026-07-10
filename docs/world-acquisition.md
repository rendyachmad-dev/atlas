# World Acquisition Platform

## Architecture & Design

---

## Table of Contents

1. [Why World Acquisition](#1-why-world-acquisition)
2. [Design Principles](#2-design-principles)
3. [Architecture Overview](#3-architecture-overview)
4. [The Observation Model](#4-the-observation-model)
5. [Source Adapter Framework](#5-source-adapter-framework)
6. [Source Registry](#6-source-registry)
7. [Ingestion Pipeline](#7-ingestion-pipeline)
8. [Source Trust Scoring](#8-source-trust-scoring)
9. [Refresh Strategies](#9-refresh-strategies)
10. [Resilience & Monitoring](#10-resilience--monitoring)
11. [Extension Points](#11-extension-points)
12. [Comparison: Current vs World Acquisition](#12-comparison-current-vs-world-acquisition)

---

## 1. Why World Acquisition

Atlas is a Personal Decision Intelligence platform. Its purpose is NOT to
collect as many news articles as possible. Its purpose is to continuously
observe trusted external sources and transform them into normalized
observations for downstream intelligence engines.

The current system has a single source type: RSS feeds. The scheduler fetches
them, parses entries, and inserts articles directly into the database. This
model served the prototype, but it has fundamental limitations:

### Current limitations

| Problem | Impact |
|---------|--------|
| **RSS-only** | Cannot observe APIs, government portals, research repositories, or HTML pages |
| **Tight coupling** | RSS parsing logic is mixed with scheduling, deduplication, and storage |
| **No observation model** | Articles are stored as raw text, not normalized observations |
| **No source registry** | Sources are hardcoded; adding a new source requires code changes |
| **No trust scoring** | All sources are treated equally regardless of reliability |
| **No adapter pattern** | Every new source type requires rewriting the entire fetch pipeline |
| **No retry strategy** | Failures are logged and ignored; no exponential backoff or retry queues |
| **No rate limiting** | Sources are fetched as fast as the loop allows, risking rate limiting |

### The World Acquisition vision

The World Acquisition Platform replaces the monolithic RSS fetcher with a
pluggable, source-agnostic acquisition pipeline. Every source — whether RSS,
REST API, HTML page, JSON Feed, government portal, or research repository —
produces the same canonical output: an **Observation**.

```
                  ┌─────────────────────────────────────┐
                  │        WORLD ACQUISITION            │
                  │                                     │
                  │  RSS    ──▶  Adapter  ──▶           │
                  │  REST   ──▶  Adapter  ──▶           │
                  │  HTML   ──▶  Adapter  ──▶  Observation │
                  │  JSON   ──▶  Adapter  ──▶           │
                  │  Gov    ──▶  Adapter  ──▶           │
                  │  Research─▶  Adapter  ──▶           │
                  └─────────────────────────────────────┘
```

---

## 2. Design Principles

### 2.1 Source-agnostic

The acquisition pipeline treats all sources uniformly. Only the Adapter knows
the specifics of each source type. The pipeline, the Observation model, the
validation logic, and the storage layer are all source-type agnostic.

### 2.2 Adapter per source type

Each source type (RSS, REST API, HTML, JSON Feed, Government Portal, Research
Repository) has exactly one Adapter implementation. The Adapter knows how to
connect to that source type, parse its response, and produce Observations.

### 2.3 Configuration-driven sources

Sources are defined in a Source Registry. Adding a new source is a
configuration change — not a code change. The registry stores the source URL,
type, adapter parameters, refresh schedule, and trust metadata.

### 2.4 Observation is the canonical currency

Every piece of external data, regardless of origin, is normalized into an
Observation before entering the Atlas pipeline. The Observation is the single
contract between the acquisition layer and the intelligence layer.

### 2.5 Trust before processing

Every source has a trust score. Observations from low-trust sources are
tagged with lower confidence, allowing downstream layers to weight them
appropriately. Trust is built over time through consistent delivery of
valid, non-duplicate content.

### 2.6 Fail gracefully

The acquisition pipeline must handle partial failures. One broken source
should not block the entire cycle. Rate limits, network errors, and
malformed responses are handled per-source, not globally.

---

## 3. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                       SOURCE REGISTRY                               │
│                                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐           │
│  │  RSS     │  │  REST    │  │  HTML    │  │  JSON    │  ...       │
│  │  Source  │  │  Source  │  │  Source  │  │  Source  │           │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘           │
│       │              │              │              │                │
└───────┼──────────────┼──────────────┼──────────────┼────────────────┘
        │              │              │              │
        ▼              ▼              ▼              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       SCHEDULER                                     │
│                                                                     │
│  For each source due: look up adapter by source.type → acquire      │
│                                                                     │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐   │
│  │RSS Adapter │  │REST Adapter│  │HTML Adapter│  │JSON Adapter│   │
│  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘   │
│        │              │              │              │              │
└────────┼──────────────┼──────────────┼──────────────┼──────────────┘
         │              │              │              │
         ▼              ▼              ▼              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       INGESTION PIPELINE                            │
│                                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐           │
│  │  Raw     │──▶│ Validate │──▶│ Dedup    │──▶│Observation│         │
│  │ Response │   │  Schema  │   │  (hash)  │   │  Store   │         │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘           │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      OBSERVATION STORE                              │
│                                                                     │
│  observations: [id, source_id, source_type, external_id, title,     │
│                  content, url, author, published_at, observed_at,    │
│                  content_hash, trust_score, confidence, metadata]    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    DOWNSTREAM INTELLIGENCE                          │
│                                                                     │
│  Knowledge Extraction ──▶ Signal Detection ──▶ Insights ──▶ ...    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Component diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    WORLD ACQUISITION PLATFORM                    │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                  SOURCE REGISTRY                         │   │
│  │  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐     │   │
│  │  │RSS   │  │REST  │  │HTML  │  │JSON  │  │Gov   │  ... │   │
│  │  │src   │  │src   │  │src   │  │src   │  │src   │     │   │
│  │  └──┬───┘  └──┬───┘  └──┬───┘  └──┬───┘  └──┬───┘     │   │
│  └─────┼─────────┼─────────┼─────────┼─────────┼──────────┘   │
│        │         │         │         │         │              │
│  ┌─────▼─────────▼─────────▼─────────▼─────────▼──────────┐   │
│  │                    SCHEDULER                            │   │
│  │  ┌─────────────┐                                       │   │
│  │  │ Tick loop   │──▶ Due sources ──▶ Dispatch per src   │   │
│  │  └─────────────┘                                       │   │
│  └────────────────────────────────────────────────────────┘   │
│        │         │         │         │         │              │
│  ┌─────▼─────────▼─────────▼─────────▼─────────▼──────────┐   │
│  │                 ADAPTERS                                │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │   │
│  │  │RSSAdapter│ │RESTAdapt.│ │HTMLAdapt.│ │JSONAdapt.│  │   │
│  │  │gofeed    │ │http call │ │goquery   │ │json parse│  │   │
│  │  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘  │   │
│  │       │ raw items  │ raw json   │ raw html   │ raw json│   │
│  └───────┼────────────┼────────────┼────────────┼─────────┘   │
│          │            │            │            │             │
│  ┌───────▼────────────▼────────────▼────────────▼─────────┐   │
│  │              INGESTION PIPELINE                         │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐   │   │
│  │  │Validate │──▶│Enrich   │──▶│Dedup    │──▶│Store    │   │   │
│  │  │ schema  │  │ trust   │  │ (hash)  │  │ Observ. │   │   │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘   │   │
│  └────────────────────────────────────────────────────────┘   │
│                              │                                │
│  ┌───────────────────────────▼────────────────────────────┐   │
│  │                   OBSERVATION STORE                     │   │
│  │  observations: normalized, deduplicated, timestamped    │   │
│  └────────────────────────────────────────────────────────┘   │
│                              │                                │
└──────────────────────────────┼────────────────────────────────┘
                               │
                               ▼
                      ┌─────────────────┐
                      │  Intelligence   │
                      │  Pipeline       │
                      │  (Sprint 2+ )   │
                      └─────────────────┘
```

---

## 4. The Observation Model

The Observation is the canonical output of every World Acquisition source.
It is the single contract between the acquisition layer and the intelligence
layer.

### Why Observations instead of Articles

The current system stores "Articles" — raw text with metadata. This works for
RSS feeds, but it is not a general model. A government regulation PDF, a
research paper abstract, a REST API status update, and a social media post
are all fundamentally different, but they share a common core: they are
**observations about the world at a point in time**.

The Observation model captures this common core. Everything else is metadata.

### Observation fields

```
observation
├── id                    UUID         — unique identifier
├── source_id             UUID         — FK → sources
├── source_type           VARCHAR      — "rss", "rest", "html", "jsonfeed", "gov", "research"
├── external_id           VARCHAR      — original ID from the source (e.g., RSS GUID)
├──
├── title                 TEXT         — headline or title (nullable)
├── content               TEXT         — body text (nullable)
├── summary               TEXT         — short description or abstract (nullable)
├── url                   TEXT         — permalink (nullable)
├──
├── author                VARCHAR      — author name (nullable)
├── published_at          TIMESTAMPTZ  — when the source says it was published
├── observed_at           TIMESTAMPTZ  — when Atlas observed it
├──
├── content_hash          VARCHAR(64)  — SHA-256 of (url + title + content)
├── language              VARCHAR(10)  — ISO 639-1 language code
├──
├── trust_score           REAL         — 0.0–1.0, inherited from source
├── confidence            REAL         — 0.0–1.0, observation-specific confidence
├──
├── raw                   JSONB        — original raw response (for debugging/replay)
├── metadata              JSONB        — source-type-specific metadata
│
│   ── metadata examples by source type ──
│   RSS:        {feed_title, feed_url, categories, guid}
│   REST API:   {endpoint, status_code, rate_limit_remaining}
│   HTML:       {selector, content_type, http_status}
│   JSON Feed:  {feed_url, version, icon}
│   Government: {agency, docket_id, document_type, regulation_id}
│   Research:   {doi, journal, authors[], citations, fields_of_study}
│
└──
```

### Observation states

```
                    ┌──────────────┐
                    │    NEW       │  Freshly observed, not yet processed
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │  ENRICHED    │  Trust score + confidence assigned
                    └──────┬───────┘
                           │
                           ▼
                    ┌──────────────┐
                    │  QUEUED      │  Ready for intelligence pipeline
                    └──────┬───────┘
                           │
                    ┌──────┴──────┐
                    │             │
                    ▼             ▼
            ┌────────────┐ ┌────────────┐
            │ PROCESSED  │ │  SKIPPED  │
            │ (extracted) │ │ (duplicate)│
            └────────────┘ └────────────┘
```

### Why this model

| Requirement | How Observation satisfies it |
|-------------|------------------------------|
| **Source-agnostic** | All sources produce the same core fields; differences go in `metadata` JSONB |
| **Extensible** | New source types add new metadata shapes without schema changes |
| **Traceable** | `observed_at` + `raw` JSONB allow full audit trail |
| **Deduplicable** | `content_hash` enables content-based dedup across source types |
| **Confidence-aware** | `trust_score` (source-level) + `confidence` (observation-level) |
| **Downstream-friendly** | Intelligence pipeline only depends on `content` + `title` + `metadata` |

---

## 5. Source Adapter Framework

### Adapter interface

Every adapter implements the same interface:

```
Adapter
├── Type() string
│     Returns the source type identifier, e.g. "rss", "rest", "html"
│
├── Acquire(ctx, source) → []RawItem, error
│     Fetches raw data from the source and returns a list of raw items.
│     Each RawItem is source-type-specific.
│     Handles: HTTP request, auth, pagination, timeout
│
├── Normalize(rawItem) → Observation
│     Converts a source-type-specific raw item into a canonical Observation.
│     Maps source-specific fields to Observation fields.
│     Fills metadata JSONB with source-specific extras.
│
└── Validate(rawResponse) → error
        Validates the raw response before parsing.
        Checks: HTTP status, content-type, size limits, schema validity
```

### Adapter catalog

#### RSS Adapter

| Property | Description |
|----------|-------------|
| Type | `rss` |
| Library | `gofeed` (existing) |
| Acquire | HTTP GET → parse RSS/Atom XML → extract items |
| Normalize | Map `Item.Title` → `observation.title`, `Item.Link` → `observation.url`, `Item.Content` → `observation.content`, `Item.GUID` → `external_id`, `Item.PublishedParsed` → `published_at` |
| Metadata | `{feed_title, feed_url, categories[], guid}` |
| Rate limit | Configurable per-source, default 60s between fetches |
| Error handling | Non-200 → skip, mark source degraded; parse error → log raw response |

#### REST API Adapter

| Property | Description |
|----------|-------------|
| Type | `rest` |
| Acquire | HTTP GET with configurable headers, auth, query params → JSON response |
| Normalize | JSONPath expressions map response fields to observation fields. Configurable per-source: `{title_path: "$.data.title", content_path: "$.data.body", url_path: "$.data.url"}` |
| Metadata | `{endpoint, status_code, rate_limit_remaining, rate_limit_reset}` |
| Pagination | Configurable: `{type: "cursor", cursor_path: "$.next", param: "after"}` or `{type: "page", param: "page", max: 10}` |
| Error handling | 429 → backoff + retry; 4xx → skip; 5xx → retry with backoff |

#### HTML Adapter

| Property | Description |
|----------|-------------|
| Type | `html` |
| Library | `goquery` or `colly` |
| Acquire | HTTP GET → parse HTML → extract items via CSS selectors |
| Normalize | CSS selector config per-source: `{title_selector: "h1.article-title", content_selector: "div.article-body", url_selector: "link[rel=canonical]"} |
| Metadata | `{selector, content_type, http_status}` |
| Use case | Government announcement pages, research lab news, corporate press release pages |
| Error handling | Selector not found → log warning, produce empty result; HTML parse error → retry once |

#### JSON Feed Adapter

| Property | Description |
|----------|-------------|
| Type | `jsonfeed` |
| Standard | JSON Feed v1.1 (jsonfeed.org) |
| Acquire | HTTP GET → parse JSON Feed → extract items |
| Normalize | Map `Item.title` → `observation.title`, `Item.url` → `observation.url`, `Item.content_text` → `observation.content`, `Item.id` → `external_id`, `Item.date_published` → `published_at` |
| Metadata | `{feed_url, version, icon, authors[]}` |
| Rate limit | Same as RSS |

#### Government Portal Adapter

| Property | Description |
|----------|-------------|
| Type | `gov` |
| Acquire | Multi-step: fetch listing page → parse document links → fetch each document. Supports authentication for restricted portals. |
| Normalize | Document-specific: `{title, content, agency, docket_id, document_type, publication_date, regulation_id}` |
| Metadata | `{agency, docket_id, document_type, regulation_id, register_url}` |
| Use case | Federal Register, SEC filings, patent offices, legislative trackers |
| Error handling | Auth failure → alert; document not found → skip; partial success → report partial |

#### Research Repository Adapter

| Property | Description |
|----------|-------------|
| Type | `research` |
| Acquire | REST API call to repository API (arXiv, Crossref, PubMed, Semantic Scholar) |
| Normalize | Map `{title, abstract, authors, doi, journal, published_date}` → observation fields |
| Metadata | `{doi, journal, authors[], citations, fields_of_study, repository}` |
| Pagination | Cursor-based via API-specific parameters |
| Rate limit | API-specific (e.g., arXiv: 1 req/3s, Semantic Scholar: 100 req/min) |

### Adapter registration

Adapters are registered by type in the Source Registry. When a source is
created with `type = "rest"`, the scheduler looks up the REST adapter.

```
adapter_registry = {
  "rss":       RSSAdapter,
  "rest":      RESTAdapter,
  "html":      HTMLAdapter,
  "jsonfeed":  JSONFeedAdapter,
  "gov":       GovPortalAdapter,
  "research":  ResearchAdapter,
}
```

---

## 6. Source Registry

The Source Registry is the single source of truth for all observed sources.
It replaces the current flat `sources` table with a richer model that
includes adapter configuration, trust metadata, and refresh strategy.

### Source model

```
source
├── id                  UUID         — unique identifier
├── name                VARCHAR      — human-readable name
├── type                VARCHAR      — adapter type: "rss", "rest", "html", "jsonfeed", "gov", "research"
├── url                 TEXT         — base URL
├──
├── adapter_config      JSONB        — adapter-specific configuration
│                                     e.g. REST: {headers, auth, paths, pagination}
│                                     e.g. HTML: {selectors, encoding}
│                                     e.g. Gov:  {agency, auth_type, document_types}
│
├── is_active           BOOLEAN      — whether to fetch this source
├──
├── trust_score         REAL         — 0.0–1.0 (see §8 for scoring)
├── trust_tier          VARCHAR      — "curated", "verified", "community", "experimental"
├── category            VARCHAR      — "news", "government", "research", "industry", "social"
├── language            VARCHAR(10)  — ISO 639-1
├── country             VARCHAR(5)   — ISO 3166-1 alpha-2
├──
├── refresh_strategy    JSONB        — {type, interval_min, interval_max, backoff, jitter}
├── rate_limit          JSONB        — {requests_per_minute, concurrent: false}
├──
├── health              JSONB        — {status, consecutive_failures, last_success, last_error}
├──
├── tags                TEXT[]       — free-form labels for filtering
├── notes               TEXT         — curation notes
├──
└── created_at / updated_at
```

### Source registry catalog

**RSS sources:**

| Name | URL | Trust Tier | Refresh | Category |
|------|-----|-----------|---------|----------|
| TechCrunch | https://techcrunch.com/feed/ | curated | 30 min | news |
| Reuters | https://www.reuters.com/tools/rss | curated | 30 min | news |
| Antara News | https://www.antaranews.com/rss | verified | 60 min | news |
| CNBC | https://www.cnbc.com/id/100003114/device/rss | curated | 30 min | news |
| arXiv AI | http://export.arxiv.org/rss/cs.AI | curated | 60 min | research |

**REST API sources:**

| Name | URL | Trust Tier | Refresh | Category |
|------|-----|-----------|---------|----------|
| GitHub Trending | https://api.github.com/search/repositories?q=... | verified | 60 min | industry |
| SEC EDGAR | https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany | curated | 24 h | government |

**Government sources:**

| Name | URL | Trust Tier | Refresh | Category |
|------|-----|-----------|---------|----------|
| Federal Register | https://www.federalregister.gov/api/v1/documents | curated | 24 h | government |
| EU AI Act Tracker | https://artificialintelligenceact.eu/ | curated | 24 h | government |
| BPS Indonesia | https://www.bps.go.id/rss | verified | 24 h | government |

**Research sources:**

| Name | URL | Trust Tier | Refresh | Category |
|------|-----|-----------|---------|----------|
| arXiv (AI/ML) | https://export.arxiv.org/api/query?search_query=cat:cs.AI | curated | 60 min | research |
| PubMed | https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi | verified | 60 min | research |
| Semantic Scholar | https://api.semanticscholar.org/graph/v1/paper/search | curated | 60 min | research |

### Source lifecycle

```
                    ┌──────────────┐
                    │   DRAFT      │  Created but not yet active
                    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │   ACTIVE     │  Being fetched on schedule
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
         ┌────▼────┐  ┌────▼────┐  ┌────▼────┐
         │ DEGRADED│  │  HEALTHY │  │  RATE    │
         │(errors) │  │  (ok)    │  │ LIMITED  │
         └────┬────┘  └──────────┘  └────┬────┘
              │                          │
         ┌────▼────┐                ┌────▼────┐
         │  DOWN   │                │ BACKING  │
         │(stale)  │                │  OFF     │
         └─────────┘                └─────────┘
```

---

## 7. Ingestion Pipeline

Every observation passes through the same pipeline, regardless of source type.

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  RAW     │───▶│ VALIDATE │───▶│ ENRICH   │───▶│ DEDUP    │───▶│  STORE   │
│  ITEM    │    │  SCHEMA  │    │  TRUST   │    │  (HASH)  │    │ OBSERV.  │
└──────────┘    └──────────┘    └──────────┘    └──────────┘    └──────────┘
```

### Stage 1: Raw Item

The Adapter produces a list of raw items. Each raw item is a source-type-
specific data structure. At this stage, it is still opaque — the pipeline
has not yet validated it.

```
Input:  RawItem{source_type: "rss", raw: <gofeed.Item>}
Output: RawItem (same, passed to next stage)
```

### Stage 2: Validate Schema

The raw item is validated for structural completeness. Required fields are
checked before any normalization occurs.

```
Checks performed:
┌──────────────────────────────────────────────────────────────┐
│ 1. title is non-empty (or url is non-empty)                  │
│ 2. content is non-empty (or summary is non-empty)            │
│ 3. url is valid URL (if present)                             │
│ 4. published_at is parseable (if present)                    │
│ 5. content_hash is computable                                │
│ 6. size limits: content ≤ 1MB, title ≤ 1000 chars           │
│ 7. language is valid ISO code (if present)                   │
└──────────────────────────────────────────────────────────────┘

Outcomes:
  ✓ Valid     → continue to enrichment
  ✗ Invalid   → log warning, skip item (mark source for review if >50% fail)
```

### Stage 3: Enrich Trust

The observation is enriched with trust metadata before storage.

```
Enrichment:
┌──────────────────────────────────────────────────────────────┐
│ 1. Inherit trust_score from source                           │
│ 2. Compute observation confidence:                           │
│      confidence = source.trust_score × content_quality       │
│      content_quality = min(1.0, content_length / 500)        │
│ 3. Assign language if not already set (detect from content)  │
│ 4. Normalize author field (trim, lowercase email)            │
│ 5. Compute content_hash = SHA-256(url + title + content)     │
└──────────────────────────────────────────────────────────────┘
```

### Stage 4: Dedup

Deduplication ensures the same observation is not stored twice.

```
Dedup strategy:
┌──────────────────────────────────────────────────────────────┐
│ Primary key:    content_hash (SHA-256 of url + title + content)│
│ Secondary key:  external_id + source_id (if available)        │
│ Tertiary key:   url (if available)                            │
│                                                               │
│ Lookup:         Check all three keys against existing store   │
│ Outcome:        Any match → skip insert (mark as duplicate)   │
│                 No match  → insert as new observation         │
└──────────────────────────────────────────────────────────────┘
```

### Stage 5: Store

The normalized, validated, enriched, deduplicated observation is stored.

```
Stored fields:
  id, source_id, source_type, external_id,
  title, content, summary, url,
  author, published_at, observed_at,
  content_hash, language,
  trust_score, confidence,
  raw, metadata,
  status: "new"
```

### Pipeline error handling

| Failure mode | Action |
|-------------|--------|
| Validation failure (individual item) | Skip item, log warning, continue pipeline |
| Validation failure (entire batch) | Mark source degraded, alert operator |
| Dedup match | Increment duplicate counter, skip insert |
| Storage failure | Retry once, then fail batch, mark source for retry |
| Adapter timeout | Retry once with backoff, then skip cycle |

---

## 8. Source Trust Scoring

Trust scoring determines how much weight to give observations from a source.
It is used by downstream intelligence layers to prioritize and filter.

### Trust tiers

| Tier | Label | Score Range | Requirements | Examples |
|------|-------|-------------|--------------|----------|
| 1 | Curated | 0.90–1.00 | Manually reviewed, verified domain, consistent content | Reuters, TechCrunch, arXiv, Federal Register |
| 2 | Verified | 0.70–0.89 | Automated verification, known domain, good history | Antara News, CNBC, PubMed |
| 3 | Community | 0.50–0.69 | Community-sourced, moderate history | Medium blogs, Substack |
| 4 | Experimental | 0.00–0.49 | New source, unknown quality, limited history | User-submitted URLs |

### Dynamic trust adjustment

Trust is not static. It changes based on source behavior:

```
trust_score = base_tier_score × freshness × consistency × validity

freshness    = 1.0 - (hours_since_last_update / expected_interval)
consistency  = successes / (successes + failures) over last 30 days
validity     = valid_observations / total_observations over last 30 days
```

| Event | Impact on trust |
|-------|----------------|
| Successful fetch on schedule | +0.01 (capped at tier max) |
| Failed fetch (network error) | -0.02 |
| Failed fetch (auth error) | -0.05 |
| >50% observations invalid | -0.10 |
| Consistent valid observations | +0.005 per cycle |
| Source back after downtime | Gradual recovery (+0.02 per success) |
| Rate limited by source | No change (system error, not source error) |

### Trust thresholds for downstream use

| Trust level | Included in intelligence pipeline? |
|-------------|-----------------------------------|
| ≥ 0.80 | Always included at full confidence |
| 0.50–0.79 | Included with confidence penalty (-0.2) |
| 0.20–0.49 | Included only if explicitly requested |
| < 0.20 | Excluded from intelligence pipeline |

---

## 9. Refresh Strategies

Different source categories need different refresh schedules. A news site
publishes every few minutes; a government portal updates daily; a research
repository may have weekly releases.

### Strategy catalog

| Strategy | Description | Interval | Jitter | Use case |
|----------|-------------|----------|--------|----------|
| **high-frequency** | News, blogs, social feeds | 15–30 min | ±5 min | TechCrunch, Reuters, Twitter |
| **medium-frequency** | Industry analysis, newsletters | 60 min | ±10 min | arXiv, Medium, Substack |
| **low-frequency** | Government, regulatory | 6–24 h | ±30 min | Federal Register, SEC, BPS |
| **batch** | Research repositories | 24 h – 7 d | ±1 h | PubMed, Semantic Scholar, patent offices |
| **on-demand** | User-triggered only | Manual | none | One-off research, custom queries |

### Refresh strategy model

```
refresh_strategy = {
  type:         "high-frequency" | "medium-frequency" | "low-frequency" | "batch" | "on-demand",
  interval_min:  15,     -- minimum interval in minutes
  interval_max:  30,     -- maximum interval in minutes (for jitter)
  backoff:       true,    -- exponential backoff on failure
  backoff_max:   1440,    -- max backoff in minutes (24 hours)
  jitter:        true,    -- add random jitter to prevent thundering herd
  jitter_pct:    0.2,     -- jitter = interval × jitter_pct
}
```

### Jitter calculation

```
actual_interval = base_interval + random(-jitter, +jitter)
```

Jitter prevents all sources of the same type from hitting their targets at
the same instant. Without jitter, 100 RSS feeds with 30-minute intervals
would all fire at :00 and :30, creating traffic spikes.

### Backoff progression

```
attempt 1:  wait 1 min
attempt 2:  wait 2 min
attempt 3:  wait 4 min
attempt 4:  wait 8 min
...
attempt N:  wait min(2^(N-1), backoff_max) min
```

After a successful fetch, backoff resets to zero.

---

## 10. Resilience & Monitoring

### Rate limiting

Rate limiting is per-source, not per-adapter. Each source tracks its own
rate limit state.

```
rate_limit = {
  requests_per_minute:  10,    -- max requests per minute
  concurrent:           false, -- allow concurrent requests?
  last_request:         "2026-07-10T10:00:00Z",
  remaining:            8,      -- remaining requests this minute
  reset_at:             "2026-07-10T10:01:00Z",
}
```

Rate limit enforcement:

1. Before requesting, check `remaining > 0`
2. If `remaining == 0`, wait until `reset_at`
3. After response, parse rate limit headers (if REST API) and update state
4. If 429 received, set `remaining = 0` and use `Retry-After` header

### Retry strategy

| Failure type | Retry count | Backoff | Max backoff |
|-------------|-------------|---------|-------------|
| Network timeout | 2 | exponential | 5 min |
| HTTP 5xx | 2 | exponential | 15 min |
| HTTP 429 (rate limit) | 3 | linear (Retry-After) | 1 hour |
| HTTP 4xx (auth) | 0 | none | — |
| Parse error | 1 | none | — |
| DNS failure | 2 | exponential | 5 min |

### Monitoring metrics

Each source and adapter exports metrics for observability:

```
── Source-level metrics ──
source_fetches_total{source, type, status}         — fetch count
source_fetch_duration_seconds{source, type}        — histogram
source_observations_total{source, type, outcome}   — observation count
source_bytes_total{source, type}                   — bandwidth
source_errors_total{source, type, error_type}      — error count
source_trust_score{source}                         — current trust score
source_last_success_timestamp{source}              — last good fetch
source_consecutive_failures{source}                — failure streak

── Adapter-level metrics ──
adapter_fetches_total{type, status}                — fetch count by adapter
adapter_duration_seconds{type}                     — histogram by adapter
adapter_errors_total{type, error_type}             — error count by adapter

── Pipeline metrics ──
pipeline_observations_total{stage, outcome}        — per-stage counts
pipeline_duration_seconds{stage}                   — per-stage latency
pipeline_dedup_rate{source}                        — % duplicates
```

### Health checks

| Signal | Threshold | Action |
|--------|-----------|--------|
| Consecutive failures | ≥ 3 | Mark source degraded, continue fetching |
| Consecutive failures | ≥ 10 | Mark source down, stop fetching, alert |
| Trust score drop | > 0.2 in 24h | Alert operator |
| Zero observations for 7 days | — | Flag source as stale, suggest review |
| Fetch duration > 30s | — | Log slow source, reduce priority |
| Pipeline error rate | > 10% in 1h | Alert operator |

### Alerting

Alerts are generated for conditions that require human intervention:

| Alert | Severity | Trigger |
|-------|----------|---------|
| Source down | high | 10 consecutive failures |
| Source degraded | medium | 3 consecutive failures |
| Auth failure | high | 401/403 on previously working source |
| Rate limit exceeded | low | Source consistently hitting rate limits |
| Zero observations | medium | Source returning empty results for 7 days |
| Adapter crash | critical | Adapter panics or returns unexpected errors |
| Pipeline stall | critical | No observations processed in 15 minutes |

---

## 11. Extension Points

The World Acquisition Platform is designed to be extended without modifying
core pipeline logic.

### Extension point 1: New adapter

Adding a new source type requires only implementing the Adapter interface.

```
1. Create Adapter implementation
   ├── Type() string          → return unique type identifier
   ├── Acquire(ctx, source)   → fetch raw data
   └── Normalize(rawItem)     → convert to Observation

2. Register in adapter registry
   adapter_registry["newtype"] = NewAdapter()
```

No changes to: scheduler, pipeline, observation store, source registry, trust
scoring, monitoring.

### Extension point 2: New source in registry

Adding a new source to an existing type requires only a registry entry.

```
1. Insert row in source registry
   ├── type: "rss"
   ├── url: "https://example.com/feed.xml"
   ├── trust_tier: "verified"
   └── refresh_strategy: {type: "medium-frequency"}
```

No code changes of any kind.

### Extension point 3: Custom validation rules

The validation stage supports pluggable validators.

```
1. Implement Validator interface
   Validate(rawItem) → error

2. Register by source type
   validators["rss"] = append(validators["rss"], customValidator)
```

Validators can be added per-source-type, per-source, or globally.

### Extension point 4: Custom normalization

The enrichment stage supports pluggable enrichments.

```
1. Implement Enricher interface
   Enrich(observation) → observation

2. Register by source type or globally
   enrichers["gov"] = append(enrichers["gov"], GovDocumentEnricher)
```

### Extension point 5: Custom rate limiters

The rate limiter supports pluggable strategies.

```
1. Implement RateLimiter interface
   Acquire(ctx, source) → error  (blocks until allowed)

2. Register by source type
   rate_limiters["rest"] = TokenBucketLimiter{rate: 10, burst: 5}
```

### Extension point 6: Custom refresh strategies

New refresh strategies can be added without modifying the scheduler.

```
1. Implement RefreshStrategy interface
   NextFetch(source) → time.Time

2. Register by strategy name
   refresh_strategies["custom"] = CustomStrategy{...}
```

---

## 12. Comparison: Current vs World Acquisition

| Aspect | Current | World Acquisition |
|--------|---------|-------------------|
| **Source types** | RSS only | RSS, REST, HTML, JSON Feed, Gov, Research |
| **Output model** | Article (raw text) | Observation (normalized) |
| **Adapter pattern** | None (hardcoded RSS) | Pluggable adapter per type |
| **Source config** | Flat DB columns | Rich JSONB config per source |
| **Trust scoring** | None | 4-tier scoring with dynamic adjustment |
| **Refresh strategy** | Fixed interval per source | 5 strategies with jitter + backoff |
| **Rate limiting** | None | Per-source with token bucket |
| **Retry** | None | Exponential backoff, per-failure-type |
| **Dedup** | Content hash only | 3-key dedup (hash, external_id, URL) |
| **Monitoring** | Basic logging | Per-source + per-adapter metrics |
| **Extensibility** | Code changes required | Configuration-driven, pluggable adapters |
| **Failure isolation** | One failure blocks all | Per-source isolation, partial success |
| **Pipeline** | Fetch → Insert | Fetch → Validate → Enrich → Dedup → Store |

---

*Document version: 1.0 — World Acquisition Platform Architecture*