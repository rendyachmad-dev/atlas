# Atlas Knowledge Graph

## Architecture & Design

---

## Table of Contents

1. [Why a Knowledge Graph](#1-why-a-knowledge-graph)
2. [Entity Types](#2-entity-types)
3. [Entity Relationships](#3-entity-relationships)
4. [Parent-Child Hierarchy](#4-parent-child-hierarchy)
5. [Synonyms](#5-synonyms)
6. [Aliases](#6-aliases)
7. [Related Entities](#7-related-entities)
8. [Entity Importance](#8-entity-importance)
9. [Graph Traversal](#9-graph-traversal)
10. [Conceptual Graph](#10-conceptual-graph)
11. [Entity Relationship Diagram](#11-entity-relationship-diagram)
12. [Integration with Decision Pipeline](#12-integration-with-decision-pipeline)

---

## 1. Why a Knowledge Graph

Atlas currently stores taxonomy entities in isolated tables. Topics live in
`topics`, industries in `industries`, countries in `countries`, technologies
in `technologies`, organizations in `organizations`. Each entity has a code,
name, description, and parent pointer.

This siloed structure is sufficient for entity extraction — the LLM can tag
articles with entity codes. But it is not sufficient for the Decision
Intelligence pipeline.

### The limitation of flat taxonomies

Consider this article:

```
"Microsoft invested $1B in OpenAI's GPT-5 training cluster in Singapore."
```

The current system extracts:

```
Topics:        [generative-ai (0.95), cloud-computing (0.85)]
Industries:    [tech (0.95)]
Countries:     [SG (0.85)]
Technologies:  [ai (0.95)]
Organizations: [microsoft (0.95), openai (0.95)]
```

These entities are stored as independent tags. The system knows:

- "Microsoft" is an organization
- "OpenAI" is an organization
- "Singapore" is a country
- "AI" is a technology

But it does **not** know:

- Microsoft **invested_in** OpenAI
- OpenAI **operates_in** Singapore
- GPT-5 **uses** cloud-computing infrastructure
- Microsoft and OpenAI have a **partnership** relationship
- Singapore is a **country_in** Southeast Asia region
- "AI" is a **subtype_of** machine-learning category
- Microsoft is a **larger** entity than OpenAI by market cap

### What a Knowledge Graph adds

A Knowledge Graph makes these relationships explicit. Instead of isolated
entity rows, the graph stores:

```
[microsoft] ──invested_in──▶ [openai]
[openai]    ──operates_in──▶ [SG]
[SG]        ──part_of──▶ [southeast-asia]
[gpt-5]     ──uses──▶ [cloud-computing]
[ai]        ──parent──▶ [machine-learning]
[microsoft] ──competitor──▶ [google]
```

### Why this matters for the Decision Pipeline

| Decision requirement | Without graph | With graph |
|---------------------|---------------|------------|
| "Which companies are related to OpenAI?" | Entity list lookup | Traverse: `invested_in`, `partnership`, `competitor` edges |
| "Which regions are affected by this supply chain signal?" | Country code only | Traverse: `country → region → continent` hierarchy |
| "Is this entity important?" | Confidence only | Confidence + graph centrality + entity importance score |
| "What are the alternative names for this entity?" | None | Synonyms + aliases |
| "What is the broader context of this topic?" | Parent code | Parent, children, related entities, cross-domain links |

---

## 2. Entity Types

The Knowledge Graph defines six primary entity types. These correspond to the
existing taxonomy tables but are extended with graph properties.

### Entity type catalog

| Type | Code prefix | Description | Examples |
|------|-------------|-------------|----------|
| **Topic** | `topic:` | Subject area or theme | `machine-learning`, `fintech`, `cybersecurity` |
| **Industry** | `industry:` | Business sector | `tech`, `healthcare`, `manufacturing` |
| **Country** | `country:` | Geographic nation | `US`, `ID`, `SG`, `JP` |
| **Region** | `region:` | Geographic grouping | `southeast-asia`, `europe`, `north-america` |
| **Technology** | `tech:` | Specific technology | `ai`, `cloud`, `blockchain`, `gpt-5` |
| **Organization** | `org:` | Company or institution | `microsoft`, `openai`, `united-nations` |

### Proposed: Composite entity types

Entity types that bridge categories. These don't exist in the current taxonomy
but are needed for rich graph traversal:

| Type | Code prefix | Description | Examples |
|------|-------------|-------------|----------|
| **Product** | `product:` | Specific product or service | `gpt-5`, `azure`, `starlink` |
| **Person** | `person:` | Key individual | `sam-altman`, `elon-musk` |
| **Event** | `event:` | Named event or incident | `covid-19`, `ukraine-conflict` |
| **Regulation** | `regulation:` | Specific regulation or law | `gdpr`, `eu-ai-act`, `section-230` |

### Entity fingerprint

Every entity in the graph has:

- **Unique ID** — UUID
- **Type** — one of the entity types above
- **Code** — human-readable slug (unique within type)
- **Name** — display name
- **Description** — one-line definition
- **Importance** — 0.0–1.0 score (see §8)
- **Created at** — when added to the graph
- **Updated at** — last modification

---

## 3. Entity Relationships

Relationships are directed edges between entities. Each edge has a type,
direction, and optional weight.

### Relationship types

| Relationship | Symbol | Description | Example |
|-------------|--------|-------------|---------|
| **invested_in** | → | Organization invested in another | `microsoft → openai` |
| **acquired** | → | Organization acquired another | `google → deepmind` |
| **partners_with** | ↔ | Strategic partnership | `microsoft ↔ openai` |
| **competes_with** | ↔ | Competitive relationship | `openai ↔ anthropic` |
| **operates_in** | → | Organization operates in a country | `openai → US` |
| **headquartered_in** | → | HQ location | `gojek → ID` |
| **regulates** | → | Regulation applies to entity | `gdpr → microsoft` |
| **affected_by** | ↔ | Event or regulation affects entity | `manufacturing → supply-chain-crisis` |
| **employs** | → | Organization employs person | `openai → sam-altman` |
| **founded_by** | → | Person founded organization | `sam-altman → openai` |
| **produces** | → | Organization produces product | `openai → gpt-5` |
| **uses** | → | Entity uses technology | `gpt-5 → cloud-computing` |
| **subsidiary_of** | → | Sub-organization | `deepmind → google` |
| **supplier_of** | → | Supply chain relationship | `tsmc → apple` |
| **client_of** | → | Business client relationship | `openai → microsoft-azure` |

### Relationship categories

| Category | Types | Symmetry |
|----------|-------|----------|
| **Financial** | `invested_in`, `acquired` | Directed |
| **Strategic** | `partners_with`, `competes_with` | Bidirectional |
| **Geographic** | `operates_in`, `headquartered_in` | Directed |
| **Regulatory** | `regulates`, `affected_by` | Directed |
| **Organizational** | `employs`, `founded_by`, `subsidiary_of` | Directed |
| **Product** | `produces`, `uses` | Directed |
| **Supply chain** | `supplier_of`, `client_of` | Directed |

### Relationship properties

Each relationship edge stores:

- **Source entity ID** — the "from" entity
- **Target entity ID** — the "to" entity
- **Relationship type** — from the catalog above
- **Confidence** — 0.0–1.0 (how certain we are this relationship exists)
- **Source** — how the relationship was discovered:
  - `extraction` — from LLM article extraction
  - `signal` — from signal detection pipeline
  - `seed` — from curated seed data
  - `inference` — derived from existing relationships
- **First detected** — when this relationship was first observed
- **Last detected** — when this relationship was last confirmed
- **Strength** — optional weight (0.0–1.0), e.g. investment amount

### Relationship discovery

Relationships are discovered through three methods:

**1. Extraction-based** — LLM extracts entity co-occurrence within the same
article. If an article mentions both "Microsoft" and "OpenAI" with investment
context, an `invested_in` edge is created.

**2. Signal-based** — The signal detection pipeline identifies co-occurring
signal types. If `investment` + `partnership` signals fire for the same
article with the same entities, those entities are linked.

**3. Inference-based** — New relationships are inferred from existing ones:
- If `A → invested_in → B` and `B → invested_in → C`, then `A → indirectly_related → C`
- If `A → partners_with → B` and `A → partners_with → C`, then `B` and `C` are
  in the same partnership network

---

## 4. Parent-Child Hierarchy

The existing taxonomy already supports parent-child relationships via the
`parent_id` column in `topics`, `technologies`, `industries`, and
`stakeholders` tables.

### Topic hierarchy

```
ai-ml (category)
├── machine-learning
│   ├── deep-learning
│   ├── reinforcement-learning
│   └── edge-ai
├── nlp
├── computer-vision
├── generative-ai
├── ai-agents
├── ai-safety
├── data-science
├── robotics
└── ... (10 sub-topics)

healthcare (category)
├── biotechnology
├── healthtech
├── pharmaceuticals
├── telemedicine
├── genomics
├── medical-devices
├── mental-health
├── public-health
├── aging-longevity
└── vaccines
```

### Technology hierarchy

```
ai (parent)
├── machine-learning
│   ├── deep-learning
│   └── llm
├── computer-vision
├── nlp
└── robotics

cloud (parent)
├── iaas
├── paas
└── saas
```

### Geographic hierarchy

```
world
├── asia
│   ├── southeast-asia
│   │   ├── ID (Indonesia)
│   │   ├── SG (Singapore)
│   │   ├── MY (Malaysia)
│   │   ├── TH (Thailand)
│   │   ├── VN (Vietnam)
│   │   └── PH (Philippines)
│   ├── south-asia
│   │   ├── IN (India)
│   │   └── PK (Pakistan)
│   └── east-asia
│       ├── JP (Japan)
│       ├── CN (China)
│       └── KR (South Korea)
├── europe
│   ├── western-europe
│   │   ├── DE (Germany)
│   │   ├── FR (France)
│   │   ├── GB (United Kingdom)
│   │   └── NL (Netherlands)
│   └── northern-europe
│       ├── SE (Sweden)
│       ├── DK (Denmark)
│       └── NO (Norway)
├── north-america
│   ├── US (United States)
│   └── CA (Canada)
└── ... (continents and sub-regions)
```

### Organizational hierarchy

```
microsoft
├── microsoft-azure (subsidiary)
├── linkedin (acquired)
├── github (acquired)
├── activision-blizzard (acquired)
└── skype (acquired)

alphabet
├── google
├── deepmind (subsidiary)
├── waymo
├── verily
└── chronicle
```

### Industry hierarchy

```
tech
├── software
│   ├── enterprise-software
│   ├── consumer-software
│   └── developer-tools
├── hardware
│   ├── semiconductors
│   └── electronics
└── it-services
    ├── consulting
    └── managed-services
```

### Hierarchy traversal rules

| Query | Traversal | Use case |
|-------|-----------|----------|
| All children of X | `X → parent_of → [children]` | "What are all sub-topics of AI/ML?" |
| All descendants of X | Recursive children | "Everything under healthcare" |
| Parent of X | `X → child_of → [parent]` | "What category does fintech belong to?" |
| Root of X | Walk up to root | "What is the top-level category of ransomware?" |
| Siblings of X | Same parent, different entity | "What other topics are under AI/ML?" |
| Depth of X | Distance from root | "How specific is this topic?" |

---

## 5. Synonyms

Synonyms are alternative names for the same entity. They allow the system to
recognize entities regardless of how they are expressed in text.

### Synonym types

| Type | Description | Example |
|------|-------------|---------|
| **Abbreviation** | Shortened form | `ML` → `machine-learning` |
| **Full form** | Expanded form | `artificial intelligence` → `ai` |
| **Common name** | Popular alternative | `AI` → `machine-learning` (broad usage) |
| **Technical name** | Formal/scientific term | `Generative Pre-trained Transformer` → `gpt` |
| **Brand name** | Marketing name | `Copilot` → `github-copilot` |
| **Legacy name** | Former name | `Twitter` → `X` (post-rebrand) |
| **Acronym** | Initialism | `LLM` → `large-language-model` |
| **Short code** | Ticker or code | `MSFT` → `microsoft` |

### Synonym catalog (examples)

**Technology synonyms:**

| Canonical entity | Synonyms |
|-----------------|----------|
| ai | artificial intelligence, AI, machine intelligence |
| machine-learning | machine learning, ML, statistical learning |
| deep-learning | deep learning, DL, neural networks |
| nlp | natural language processing, NLP, language models |
| generative-ai | generative AI, GenAI, generative artificial intelligence |
| llm | large language model, LLM, large language models |
| cloud-computing | cloud computing, cloud, cloud infrastructure |
| blockchain | blockchain, distributed ledger, DLT |
| fintech | financial technology, fintech, digital finance |
| cybersecurity | cybersecurity, cyber security, infosec, information security |

**Organization synonyms:**

| Canonical entity | Synonyms |
|-----------------|----------|
| microsoft | Microsoft, MSFT, Microsoft Corporation, Redmond |
| google | Google, Alphabet, GOOG, GOOGL |
| openai | OpenAI, OpenAI Inc., OpenAI LP |
| anthropic | Anthropic, Anthropic PBC |
| united-nations | UN, United Nations, United Nations Organization |

**Topic synonyms:**

| Canonical entity | Synonyms |
|-----------------|----------|
| ai-regulation | AI regulation, AI governance, AI policy, AI law |
| supply-chain-mfg | supply chain, supply chain management, SCM, logistics |
| renewable-energy | renewable energy, clean energy, green energy, sustainable energy |

### Synonym resolution rules

1. **Exact match first** — check entity code directly
2. **Synonym match** — check all synonyms
3. **Fuzzy match** — if no direct or synonym match, try Levenshtein distance
   (threshold: 0.8 similarity)
4. **Compound match** — split compound terms and match components
   (e.g., "cloud-computing" → "cloud" + "computing")
5. **Abbreviation expansion** — known acronyms →
   full form (e.g., "LLM" → "large-language-model")

### Synonym graph

Synonyms form a connected graph where the canonical entity is the hub:

```
               ┌──────────┐
               │  "ML"    │
               └────┬─────┘
                    │
          ┌─────────▼──────────┐
          │  machine-learning  │  ← canonical
          └─────────┬──────────┘
                    │
          ┌─────────┴──────────┐
          │  "machine learning" │
          └────────────────────┘
```

```
               ┌──────────┐
               │ "GPT-5"  │
               └────┬─────┘
                    │
          ┌─────────▼──────────┐
          │     gpt-5          │  ← canonical
          └─────────┬──────────┘
                    │
     ┌──────────────┼──────────────┐
     │              │              │
┌────▼────┐  ┌─────▼─────┐  ┌─────▼─────┐
│"GPT-5"  │  │"Generative │  │  "GPT-5   │
│(product)│  │Pre-trained │  │  model"   │
│         │  │Transformer │  │           │
│         │  │   5"       │  │           │
└─────────┘  └───────────┘  └───────────┘
```

---

## 6. Aliases

Aliases are different from synonyms. A synonym is an alternative name for the
**same** entity. An alias is a **redirect** — a name that points to a different
canonical entity.

### Alias types

| Type | Description | Example |
|------|-------------|---------|
| **Ticker** | Stock ticker symbol | `MSFT` → `microsoft` |
| **Handle** | Social media handle | `@OpenAI` → `openai` |
| **Domain** | Website domain | `openai.com` → `openai` |
| **Nickname** | Informal name | `Zuck` → `mark-zuckerberg` |
| **Translation** | Name in another language | `阿里巴巴` → `alibaba` |
| **Former name** | Pre-rebrand name | `Facebook` → `meta` |

### Alias catalog (examples)

| Alias | Resolves to | Type |
|-------|-------------|------|
| MSFT | microsoft | Stock ticker |
| AAPL | apple | Stock ticker |
| GOOG | alphabet | Stock ticker |
| AMZN | amazon | Stock ticker |
| META | meta | Stock ticker |
| 0700.HK | tencent | Stock ticker |
| @OpenAI | openai | Social handle |
| @Microsoft | microsoft | Social handle |
| openai.com | openai | Domain |
| microsoft.com | microsoft | Domain |
| FB | meta | Former name (rebrand) |
| Twitter | X | Former name (rebrand) |
| Google Health | verily | Former name |
| Facebook | meta | Former name (rebrand) |
| 阿里巴巴 | alibaba | Chinese name |
| 腾讯 | tencent | Chinese name |
| 百度 | baidu | Chinese name |
| Sam Altman | sam-altman | Common name |
| Elon Musk | elon-musk | Common name |

### Alias resolution process

1. **Exact alias match** — check alias table
2. **Redirect** — replace alias with canonical entity code
3. **Confidence scoring** — alias confidence is lower than direct entity
   match (e.g., "MSFT" → microsoft: 0.95, but "Redmond" → microsoft: 0.7)
4. **Ambiguity resolution** — if an alias could resolve to multiple entities,
   use context (co-occurring entities) to disambiguate

### Alias vs Synonym

```
                   ┌─────────────────────────────────────┐
                   │         Aliases                      │
                   │  "MSFT" → microsoft (redirect)       │
                   │  "FB" → meta (redirect)              │
                   │  "openai.com" → openai (redirect)    │
                   └─────────────────────────────────────┘

                   ┌─────────────────────────────────────┐
                   │         Synonyms                     │
                   │  "artificial intelligence" ↔ ai      │
                   │  "ML" ↔ machine-learning             │
                   │  "LLM" ↔ large-language-model        │
                   └─────────────────────────────────────┘
```

Key difference: Synonyms are **equivalent** (bidirectional). Aliases are
**redirects** (unidirectional, often from a shorthand to the canonical form).

---

## 7. Related Entities

Related entities are entities that are frequently associated but do not have
a direct relationship edge. These are discovered through co-occurrence
analysis.

### Co-occurrence graph

When two entities appear together in the same article multiple times, they
are inferred to be related:

```
Article count threshold: ≥ 3 articles in 30 days
Confidence: co-occurrence_count / total_articles_for_entity
```

### Cross-domain relatedness

Entities from different taxonomies can be related:

| Entity A | Entity B | Why related |
|----------|----------|-------------|
| machine-learning | ai | ai is the parent technology of machine-learning |
| cloud-computing | tech | cloud is a sub-domain of tech industry |
| fintech | financial-activity | fintech drives financial activity signals |
| cybersecurity | data-privacy | cybersecurity and privacy are overlapping domains |
| generative-ai | gpt-5 | GPT-5 is a product in the generative AI space |
| Indonesia | Singapore | Neighboring countries in Southeast Asia |
| openai | microsoft | Investor-investee relationship |

### Relatedness scoring

Relatedness is scored 0.0–1.0:

```
relatedness_score = co-occurrence_frequency × recency_factor × diversity_factor

co-occurrence_frequency = articles_with_both / articles_with_either
recency_factor = 1.0 - (days_since_last_co_occurrence / 90)
diversity_factor = min(1.0, unique_sources / 3)
```

### Related entity types

| Type | Description | Example |
|------|-------------|---------|
| **co_occurs_with** | Appears in same articles | `machine-learning` ↔ `cloud-computing` |
| **similar_to** | Same category, similar profile | `openai` ↔ `anthropic` |
| **complementary_to** | Used together frequently | `ai` ↔ `cloud-computing` |
| **conflicts_with** | Competing or opposing | `openai` ↔ `deepmind` |
| **depends_on** | Technology dependency | `gpt-5` ↔ `gpu-computing` |
| **enables** | Enables another field | `deep-learning` → `computer-vision` |

### Related entity discovery pipeline

```
┌──────────────┐
│  Article     │
│  ingested    │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Extract     │  Get all entities from this article
│  entities    │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Co-occur    │  For each pair of entities in the article:
│  pairs       │    increment co-occurrence count
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Score       │  Calculate relatedness score
│  relatedness │  using frequency + recency + diversity
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Store       │  Upsert related_entities edge
│  edge        │  with updated score
└──────────────┘
```

---

## 8. Entity Importance

Entity importance is a 0.0–1.0 score that represents how significant an entity
is within the Atlas system. It is used for prioritization, filtering, and
decision-making.

### Importance factors

| Factor | Weight | Description | Source |
|--------|--------|-------------|--------|
| **Article frequency** | 0.25 | How often this entity appears in articles | `article_entity links` |
| **Signal frequency** | 0.20 | How many signals this entity generates | `signals table` |
| **Graph centrality** | 0.20 | How connected this entity is in the graph | Graph traversal |
| **Recency** | 0.15 | How recently this entity was mentioned | `last_detected_at` |
| **Relationship diversity** | 0.10 | How many relationship types involve this entity | Edge count |
| **User engagement** | 0.10 | How often users interact with this entity's content | User actions |

### Importance calculation

```
importance = Σ(factor_weight × factor_score) × decay_factor

decay_factor = max(0.3, 1.0 - days_since_last_mention / 365)
```

### Importance tiers

| Tier | Score | Label | Example entities |
|------|-------|-------|-----------------|
| **Tier 1** | 0.80–1.00 | Critical | `microsoft`, `google`, `ai`, `US`, `tech` |
| **Tier 2** | 0.60–0.79 | Major | `openai`, `machine-learning`, `ID`, `healthcare` |
| **Tier 3** | 0.40–0.59 | Notable | `anthropic`, `fintech`, `SG`, `manufacturing` |
| **Tier 4** | 0.20–0.39 | Minor | `edtech`, `vn`, `mental-health` |
| **Tier 5** | 0.00–0.19 | Emerging | Newly added entities, low frequency |

### Importance decay

Entity importance decays over time if the entity is not mentioned:

```
Days since last mention:  7   30   90   180   365
Importance multiplier:   0.98 0.92 0.75  0.50  0.30
```

When an entity is mentioned again, its importance recovers:

```
new_importance = max(current_importance, old_importance × 0.8 + 0.2)
```

### Importance use cases

| Use case | How importance is used |
|----------|----------------------|
| **Signal priority** | Signals from Tier 1 entities get higher priority |
| **Decision weighting** | Decisions involving high-importance entities get confidence boost |
| **Search ranking** | Important entities appear first in search results |
| **Graph pruning** | Low-importance entities can be archived or lazily loaded |
| **Trend detection** | Sudden importance changes indicate emerging trends |
| **Dashboard** | Top N entities by importance shown on dashboard |

---

## 9. Graph Traversal

The Knowledge Graph is traversed to answer questions that the flat taxonomy
cannot.

### Traversal primitives

| Primitive | Description | Example |
|-----------|-------------|---------|
| `neighbors(entity, relation)` | All entities connected by a relation | `neighbors(openai, invested_in)` → `[microsoft]` |
| `path(entity_a, entity_b)` | Shortest path between two entities | `path(openai, Singapore)` → `openai → operates_in → SG` |
| `subgraph(entity, depth)` | All entities within N hops | `subgraph(microsoft, 2)` → subsidiaries, partners, competitors |
| `ancestors(entity)` | Chain of parents to root | `ancestors(ransomware)` → `[cybersecurity, ai-ml?]` |
| `descendants(entity)` | All children recursively | `descendants(ai-ml)` → `[ML, DL, NLP, CV, ...]` |
| `related(entity, min_score)` | Related entities above threshold | `related(ai, 0.7)` → `[ML, cloud, data]` |

### Traversal examples

#### Example 1: "Which companies are in OpenAI's investment network?"

```
Traversal:
  start: openai
  edge: invested_in (incoming)
  result: [microsoft, sequoia, a16z, thrive-capital]
  depth: 1

  If we go deeper (depth 2):
  start: openai
  edge: invested_in (incoming)
  for each investor:
    edge: invested_in (outgoing)
  result: [openai, anthropic, ...] ← other companies the investors fund

  This tells us: "Microsoft invested in OpenAI, Microsoft also invested in..."
```

#### Example 2: "Which regions are affected by this supply chain disruption?"

```
Article: "Semiconductor shortage in Japan, Germany, US"

Knowledge graph traversal:
  entities: [JP, DE, US]
  for each:
    ancestors(entity) → continent chain

  JP → east-asia → asia
  DE → western-europe → europe
  US → north-america

  Result: "Asia, Europe, and North America are affected"
  → Cross-region supply chain disruption
```

#### Example 3: "What is the competitive landscape for OpenAI?"

```
Traversal:
  start: openai
  edge: competes_with (bidirectional)
  result: [anthropic, cohere, mistral, google]

  Then for each competitor:
    edge: invested_in (incoming)
    → identifies who is funding the competition

  Extended:
    edge: partners_with
    → identifies strategic alliances

  Full competitive map:
    OpenAI competitors: [anthropic, cohere, mistral, google]
    Funding sources: [microsoft → openai, google → deepmind, amazon → anthropic]
    Partnership network: [openai ↔ microsoft, anthropic ↔ amazon]
```

#### Example 4: "How does AI regulation impact the fintech sector?"

```
Traversal:
  start: ai-regulation (topic)
  edge: co_occurs_with (related topics)
  result: [fintech, data-privacy, cybersecurity]

  Then:
    start: fintech
    edge: affects (industry)
    result: [finance, tech]

  Full path:
    ai-regulation → co_occurs_with → fintech
    fintech → affects → finance
    fintech → affects → tech

  Decision insight:
    "AI regulation directly impacts fintech,
     which cascades to the finance and tech industries."
```

#### Example 5: "Find all entities related to a specific article for enrichment"

```
Article: "Microsoft invests $1B in Indonesian AI startup"

Extracted entities: [microsoft, investment, ID, ai, startup]

Knowledge graph enrichment:
  for each entity:
    neighbors(entity, all_relations)

  microsoft → subsidiary_of: [azure, github, linkedin]
  microsoft → competes_with: [google, amazon, apple]
  ID → region: [southeast-asia]
  ai → parent: [machine-learning]
  ai → related: [deep-learning, nlp, computer-vision, robotics]
  startup → parent: [enterprise]

Enriched entity set:
  [microsoft, azure, github, linkedin, google, amazon,
   ID, southeast-asia, ai, machine-learning, deep-learning,
   nlp, startup, enterprise]
```

### Traversal query patterns

| Pattern | Query | SQL/Graph equivalent |
|---------|-------|----------------------|
| Direct neighbors | `neighbors(X, R)` | `SELECT * FROM edges WHERE source = X AND type = R` |
| Bidirectional | `neighbors_any(X, R)` | `WHERE source = X OR target = X` |
| Path finding | `shortest_path(A, B)` | BFS/DFS with depth limit |
| Subgraph | `subgraph(X, N)` | Recursive CTE, depth N |
| Community | `community(X, min_score)` | Connected components with edge weight filter |
| Centrality | `centrality(X)` | Count of incoming/outgoing edges |
| Influence | `influence(X)` | Sum of importance scores of connected entities |

### Indexing requirements

For efficient traversal:

1. **Adjacency index** — `(source_entity, relationship, target_entity)` for
   O(1) neighbor lookups
2. **Reverse index** — `(target_entity, relationship, source_entity)` for
   incoming relationship queries
3. **Path index** — materialized paths for common traversals (e.g., all
   descendants of a topic)
4. **Full-text index** — on entity names, descriptions, synonyms, and aliases

---

## 10. Conceptual Graph

The Atlas Knowledge Graph is a **labeled property graph**. Entities are nodes,
relationships are directed edges with types, and both nodes and edges have
properties.

### Graph notation

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│   [Entity] ──relationship──▶ [Entity]                               │
│   {properties}              {properties}                            │
│                                                                     │
│   [Entity] ◀──relationship── [Entity]  (bidirectional)              │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Full conceptual graph

```
                                     ┌──────────────────────────────────────────────┐
                                     │              KNOWLEDGE GRAPH                 │
                                     │                                              │
                                     │  ┌────────────────────────────────────────┐  │
                                     │  │           HIERARCHY LAYER              │  │
                                     │  │                                        │  │
                                     │  │  ai-ml ──parent──▶ machine-learning    │  │
                                     │  │  machine-learning ──parent──▶ deep-learning│  │
                                     │  │  asia ──parent──▶ southeast-asia       │  │
                                     │  │  southeast-asia ──parent──▶ ID         │  │
                                     │  │  tech ──parent──▶ software             │  │
                                     │  └────────────────────────────────────────┘  │
                                     │                        │                    │
                                     │                        ▼                    │
                                     │  ┌────────────────────────────────────────┐  │
                                     │  │         RELATIONSHIP LAYER            │  │
                                     │  │                                        │  │
                                     │  │  microsoft ──invested_in──▶ openai     │  │
                                     │  │  openai ──competes_with──▶ anthropic   │  │
                                     │  │  openai ──operates_in──▶ US            │  │
                                     │  │  openai ──produces──▶ gpt-5            │  │
                                     │  │  gpt-5 ──uses──▶ cloud-computing       │  │
                                     │  │  gdpr ──regulates──▶ microsoft          │  │
                                     │  └────────────────────────────────────────┘  │
                                     │                        │                    │
                                     │                        ▼                    │
                                     │  ┌────────────────────────────────────────┐  │
                                     │  │         SYNONYM / ALIAS LAYER         │  │
                                     │  │                                        │  │
                                     │  │  ai ◀──synonym──▶ artificial-intelligence│  │
                                     │  │  MSFT ──alias──▶ microsoft             │  │
                                     │  │  openai.com ──alias──▶ openai          │  │
                                     │  │  LLM ◀──synonym──▶ large-language-model │  │
                                     │  │  FB ──alias──▶ meta                    │  │
                                     │  └────────────────────────────────────────┘  │
                                     │                        │                    │
                                     │                        ▼                    │
                                     │  ┌────────────────────────────────────────┐  │
                                     │  │         RELATED ENTITY LAYER          │  │
                                     │  │                                        │  │
                                     │  │  ai ◀──co_occurs──▶ cloud-computing    │  │
                                     │  │  machine-learning ◀──similar──▶ deep-learning│  │
                                     │  │  openai ◀──conflicts──▶ deepmind      │  │
                                     │  │  fintech ◀──enables──▶ blockchain     │  │
                                     │  └────────────────────────────────────────┘  │
                                     │                                              │
                                     └──────────────────────────────────────────────┘
```

### Cross-layer graph example

A single article processed through the full graph:

```
Article: "Microsoft invests $1B in OpenAI's GPT-5 training in Singapore"

     ┌────────────────────────────────────────────────────────────┐
     │                    ARTICLE NODE                            │
     │  id: uuid-1234                                            │
     │  title: "Microsoft invests $1B in OpenAI's GPT-5..."      │
     │  source: TechCrunch                                       │
     └────────────────────────┬───────────────────────────────────┘
                              │
            ┌─────────────────┼─────────────────┐
            │                 │                  │
            ▼                 ▼                  ▼
     ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
     │  microsoft   │  │   openai     │  │   gpt-5      │
     │  type: org   │  │  type: org   │  │  type: prod  │
     │  importance: │  │  importance: │  │  importance: │
     │  0.92        │  │  0.78        │  │  0.45        │
     └──────┬───────┘  └──────┬───────┘  └──────┬───────┘
            │                 │                  │
            │     invested_in │                  │
            ├─────────────────▶                  │
            │                 │                  │
            │     partners    │                  │
            │     _with       │                  │
            ├─────────────────▶                  │
            │                 │                  │
            │                 │     produces     │
            │                 ├──────────────────▶
            │                 │                  │
            │                 │                  │
            ▼                 ▼                  ▼
     ┌──────────────────────────────────────────────────┐
     │  invested_in (edge)                              │
     │  confidence: 0.95                                │
     │  source: extraction                              │
     │  strength: 1.0 ($1B)                             │
     └──────────────────────────────────────────────────┘
```

---

## 11. Entity Relationship Diagram

### Logical ERD (entity types only)

```
┌─────────────────────────────────────────────────────────────────┐
│                    ENTITY NODES                                 │
│                                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │  TOPIC   │  │ INDUSTRY │  │ COUNTRY  │  │  REGION  │       │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘       │
│       │              │             │             │             │
│       ▼              ▼             ▼             ▼             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │   TECH   │  │   ORG    │  │ PRODUCT  │  │  PERSON  │       │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘       │
│       │              │             │             │             │
│       ▼              ▼             ▼             ▼             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                     │
│  │  EVENT   │  │REGULATION│  │  ARTICLE │                     │
│  └──────────┘  └──────────┘  └──────────┘                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    RELATIONSHIP EDGES                           │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  invested_in     acquired     partners_with             │   │
│  │  competes_with   operates_in  headquartered_in          │   │
│  │  regulates       affected_by  employs     founded_by    │   │
│  │  produces        uses         subsidiary_of            │   │
│  │  supplier_of     client_of    parent_of                 │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    METADATA NODES                               │
│                                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                     │
│  │ SYNONYM  │  │  ALIAS   │  │ RELATED  │                     │
│  │          │  │          │  │ ENTITY   │                     │
│  └──────────┘  └──────────┘  └──────────┘                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Physical ERD (storage tables)

```
entity_types
├── id (UUID, PK)
├── code (VARCHAR, UNIQUE)           -- topic / industry / country / region / tech / org
├── name (VARCHAR)
└── description (TEXT)

entities
├── id (UUID, PK)
├── entity_type_id (UUID, FK)
├── code (VARCHAR, UNIQUE)           -- e.g. "machine-learning", "microsoft"
├── name (VARCHAR)                   -- display name
├── description (TEXT)
├── parent_id (UUID, FK → entities)  -- hierarchy parent (nullable)
├── importance (REAL)                -- 0.0–1.0
├── is_active (BOOLEAN)
├── created_at (TIMESTAMPTZ)
└── updated_at (TIMESTAMPTZ)

entity_relationships
├── id (UUID, PK)
├── source_entity_id (UUID, FK → entities)
├── target_entity_id (UUID, FK → entities)
├── relationship_type (VARCHAR)      -- invested_in / partners_with / operates_in / ...
├── confidence (REAL)                -- 0.0–1.0
├── strength (REAL)                  -- optional weight (0.0–1.0)
├── source (VARCHAR)                 -- extraction / signal / seed / inference
├── first_detected_at (TIMESTAMPTZ)
├── last_detected_at (TIMESTAMPTZ)
├── created_at (TIMESTAMPTZ)
└── UNIQUE(source_entity, relationship_type, target_entity)

entity_synonyms
├── id (UUID, PK)
├── entity_id (UUID, FK → entities)
├── synonym (VARCHAR)                -- alternative name
├── type (VARCHAR)                   -- abbreviation / full_name / common_name / acronym
├── confidence (REAL)                -- 0.0–1.0
├── created_at (TIMESTAMPTZ)
└── UNIQUE(entity_id, synonym)

entity_aliases
├── id (UUID, PK)
├── entity_id (UUID, FK → entities)
├── alias (VARCHAR)                  -- redirect source
├── type (VARCHAR)                   -- ticker / domain / handle / former_name
├── confidence (REAL)                -- 0.0–1.0
├── created_at (TIMESTAMPTZ)
└── UNIQUE(alias)

entity_related
├── id (UUID, PK)
├── entity_a_id (UUID, FK → entities)
├── entity_b_id (UUID, FK → entities)
├── relatedness_type (VARCHAR)       -- co_occurs / similar / complementary / conflicts
├── score (REAL)                     -- 0.0–1.0
├── co_occurrence_count (INTEGER)
├── last_co_occurrence (TIMESTAMPTZ)
├── created_at (TIMESTAMPTZ)
├── updated_at (TIMESTAMPTZ)
└── UNIQUE(entity_a_id, entity_b_id)
```

### Indexes

```
-- Entity lookup
idx_entities_code            ON entities (code)
idx_entities_type            ON entities (entity_type_id)
idx_entities_parent          ON entities (parent_id)
idx_entities_importance      ON entities (importance DESC)

-- Relationship traversal
idx_relationships_source     ON entity_relationships (source_entity_id, relationship_type)
idx_relationships_target     ON entity_relationships (target_entity_id, relationship_type)
idx_relationships_confidence ON entity_relationships (confidence DESC)

-- Synonym/alias resolution
idx_synonyms_entity          ON entity_synonyms (entity_id)
idx_synonyms_text            ON entity_synonyms (synonym)  -- for lookup
idx_aliases_text             ON entity_aliases (alias)     -- for redirect resolution

-- Related entity queries
idx_related_entity_a         ON entity_related (entity_a_id, score DESC)
idx_related_entity_b         ON entity_related (entity_b_id, score DESC)
idx_related_score            ON entity_related (score DESC)
```

---

## 12. Integration with Decision Pipeline

The Knowledge Graph is not a standalone system. It integrates into every layer
of the Decision Intelligence pipeline.

### Pipeline integration

```
                    DECISION PIPELINE WITH KNOWLEDGE GRAPH
                    ════════════════════════════════════

    ARTICLE
    ────────
    "Microsoft invests $1B in OpenAI's GPT-5 training cluster
     in Singapore."

        │
        ▼
    ┌───────────────────────────────────────────────────────────────┐
    │ EXTRACTION  (LLM)                                             │
    │                                                               │
    │  Entities extracted: [microsoft, openai, gpt-5, SG, ai]      │
    │                                                               │
    │  Graph enrichment:                                            │
    │    microsoft → alias("MSFT"), synonym("Microsoft Corp")       │
    │    SG → parent(southeast-asia → asia)                         │
    │    ai → parent(machine-learning → ai-ml)                      │
    │    gpt-5 → new product entity created                         │
    │    microsoft → openai: invested_in edge created               │
    └───────────────────────────────────────────────────────────────┘
        │
        ▼
    ┌───────────────────────────────────────────────────────────────┐
    │ SIGNAL DETECTION  (Rule engine)                               │
    │                                                               │
    │  Signals: investment(0.95), partnership(0.90),                │
    │           technology_adoption(0.90), market_entry(0.85)       │
    │                                                               │
    │  Graph enhancement:                                           │
    │    Signal + entity → relationship confidence boosted          │
    │    Entity importance updated (co-occurrence count)            │
    │    Related entity scores refreshed                            │
    └───────────────────────────────────────────────────────────────┘
        │
        ▼
    ┌───────────────────────────────────────────────────────────────┐
    │ INSIGHT SYNTHESIS  (Pattern engine)                           │
    │                                                               │
    │  Insights: investment_cluster, geographic_hotspot,            │
    │            technology_convergence                             │
    │                                                               │
    │  Graph traversal for enrichment:                              │
    │    "Which other companies are in Microsoft's investment       │
    │     network?" → traversed from graph                          │
    │    "What is the regional impact of SG expansion?"             │
    │     → SG → southeast-asia → asia (hierarchy traversal)       │
    └───────────────────────────────────────────────────────────────┘
        │
        ▼
    ┌───────────────────────────────────────────────────────────────┐
    │ DECISION FORMATION  (Evaluation engine)                       │
    │                                                               │
    │  Decision: "OpenAI's Southeast Asia expansion is a            │
    │            strategic opportunity"                             │
    │                                                               │
    │  Graph contribution:                                          │
    │    Entity importance: microsoft(0.92), openai(0.78)           │
    │      → high-importance entities boost decision confidence     │
    │    Relationship strength: invested_in(1.0), partners(0.9)     │
    │      → strong relationship confirms strategic alignment       │
    │    Graph centrality: Microsoft is a hub node                  │
    │      → decisions involving hubs get higher impact rating      │
    └───────────────────────────────────────────────────────────────┘
        │
        ▼
    ┌───────────────────────────────────────────────────────────────┐
    │ ACTION RECOMMENDATION  (Scoring engine)                       │
    │                                                               │
    │  Action: "Engage with OpenAI's Southeast Asia expansion"      │
    │                                                               │
    │  Graph contribution:                                          │
    │    Target entity: "OpenAI" → all known aliases and synonyms   │
    │      for search and monitoring                                │
    │    Related entities: [microsoft, SG, azure, gpt-5]            │
    │      → expanded context for the action description            │
    │    Competitor entities: [anthropic, google, cohere]           │
    │      → competitive landscape included in briefing             │
    └───────────────────────────────────────────────────────────────┘
```

### Query patterns by layer

| Layer | Graph query | Why |
|-------|-------------|-----|
| Extraction | `alias_resolve("MSFT")` → microsoft | Normalize entity names |
| Extraction | `synonym_expand("AI")` → [ai, artificial-intelligence] | Broaden entity matching |
| Extraction | `hierarchy_ancestors("ransomware")` → [cybersecurity] | Infer broader context |
| Signals | `importance("openai")` → 0.78 | Boost signal confidence for important entities |
| Signals | `neighbors("openai", "invested_in")` → [microsoft] | Discover related entities for signal enrichment |
| Insights | `subgraph("microsoft", 2)` | Find all entities in Microsoft's ecosystem |
| Insights | `related("ai", 0.7)` → [ML, cloud, data] | Discover related topics for pattern expansion |
| Decisions | `centrality("microsoft")` → high | Boost decision impact for hub entities |
| Decisions | `path("openai", "SG")` → openai→operates_in→SG | Confirm geographic connection |
| Actions | `aliases("openai")` → [@OpenAI, openai.com] | Generate search terms for monitoring |
| Actions | `competitors("openai")` → [anthropic, google] | Include competitive context |

### Graph as a decision multiplier

The Knowledge Graph transforms the Decision Pipeline from a linear processor
into a connected intelligence system:

```
Without graph:                    With graph:
                                    ┌──────────┐
  Article ──▶ Knowledge              │  GRAPH   │
  Knowledge ──▶ Signals              │          │
  Signals ──▶ Insights         ◀────│ enriches  │
  Insights ──▶ Decisions       ◀────│ all       │
  Decisions ──▶ Actions        ◀────│ layers    │
                                    └──────────┘
```

Every layer queries the graph for:
- **Entity resolution** — normalize names via aliases and synonyms
- **Context expansion** — find related entities, ancestors, neighbors
- **Importance weighting** — boost confidence for important entities
- **Relationship discovery** — find edges between extracted entities
- **Traversal enrichment** — go beyond direct mentions to discover implicit connections

---

*Document version: 1.0 — Knowledge Graph Architecture*