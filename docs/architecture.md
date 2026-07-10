# Decision Intelligence Architecture

## Sprint 3.0 — Architecture & Design

---

## Table of Contents

1. [Why a Decision Layer](#1-why-a-decision-layer)
2. [Architecture Overview](#2-architecture-overview)
3. [Layer 1: Knowledge → Signals](#3-layer-1-knowledge--signals)
4. [Layer 2: Signals → Insights](#4-layer-2-signals--insights)
5. [Layer 3: Insights → Decisions](#5-layer-3-insights--decisions)
6. [Layer 4: Decisions → Actions](#6-layer-4-decisions--actions)
7. [End-to-End Flow](#7-end-to-end-flow)
8. [Real-World Examples](#8-real-world-examples)
9. [Extensibility Guide](#9-extensibility-guide)

---

## 1. Why a Decision Layer

Atlas converts raw news articles into structured knowledge — topics, industries,
countries, technologies, and organizations mentioned in each article. The
architecture gradually transforms this raw material into increasingly meaningful
forms.

The original architecture defined three layers:

| Layer | Answers | Example |
|-------|---------|---------|
| **Signal** | "What happened?" | "OpenAI received investment funding" |
| **Insight** | "What does it mean?" | "AI investment cluster forming — 3 funding rounds in 7 days" |
| **Action** | "What should I do?" | "Monitor OpenAI's valuation trajectory for IPO signals" |

After review, a gap was identified between **Insight** and **Action**.

### The missing step

An **Insight** describes what is happening. It is an observation:

- "Healthcare AI investment is increasing."
- "AI regulation is accelerating globally."
- "Supply chain disruption is affecting manufacturing."

An **Action** describes what a user can do. It is a recommendation:

- "Learn Healthcare AI."
- "Build a healthcare SaaS."
- "Monitor AI regulation."

But there is a missing reasoning step. Before recommending an action, Atlas
must first determine **what decision should be made** — what conclusion should
be drawn from the available evidence.

Consider:

```
Insight: "Healthcare AI investment is increasing."
         ───────────────────────────────────────
         This is an observation. It tells us something
         is happening in the world.

Decision: "Healthcare AI is a strategic opportunity
           worth pursuing."
         ───────────────────────────────────────
         This is a conclusion. It tells us what to
         think about the observation.

Action: "Learn Healthcare AI. Build a healthcare SaaS."
       ────────────────────────────────────────────────
       This is a recommendation. It tells us what to
       do about the conclusion.
```

The Insight says "X is happening." The Decision says "X matters because Y, and
we should conclude Z." The Action says "here is what to do about Z."

### The four-layer architecture

| Layer | Answers | Example |
|-------|---------|---------|
| **Signal** | "What happened?" | "OpenAI received investment funding" |
| **Insight** | "What does it mean?" | "AI investment cluster forming — 3 funding rounds in 7 days" |
| **Decision** | "What should we conclude?" | "Healthcare AI is a strategic opportunity worth pursuing" |
| **Action** | "What should we do?" | "Learn Healthcare AI, build a healthcare SaaS" |

### Why the Decision layer is distinct

| Question | Insight | Decision | Action |
|----------|---------|----------|--------|
| Nature | Observation | Conclusion | Recommendation |
| Tense | Present | Evaluative | Imperative |
| Certainty | "X is happening" | "X matters because Y" | "Do Z" |
| Risk | No | Yes — weighs pros/cons | Yes — execution risk |
| Trade-offs | No | Yes — "invest here, not there" | Yes — resource allocation |
| Time horizon | Past → present | Present → future | Future |
| Evidence | Summarizes signals | Weighs conflicting insights | Justifies one path |
| Reversibility | N/A | Reversible (re-evaluate) | Costly to reverse |

### What problem does the Decision layer solve?

**Without a Decision layer**, the system jumps directly from observation to
recommendation. This causes several problems:

1. **Conflicting insights produce no guidance.** If Insight A says "AI
   investment is growing" and Insight B says "AI regulation is tightening,"
   the system cannot determine which matters more. It either produces both
   actions (diluting focus) or arbitrarily picks one.

2. **No risk assessment.** An action like "invest in AI" is recommended
   without evaluating the risks (regulatory headwinds, market saturation).
   The user must do this mentally.

3. **No traceability.** When the system says "Monitor OpenAI," the user
   cannot ask "why?" and get a structured answer. The reasoning is implicit
   in the code that maps insights to actions.

4. **No conflict resolution.** Two insights may point in opposite directions:
   "market opportunity" vs "regulatory risk." The system needs a layer that
   weighs both and produces a balanced conclusion.

**With a Decision layer**, the system produces explicit, structured conclusions
that are:

- **Traceable** — every decision links back to the insights and signals that
  support it, and the original articles that produced them
- **Explainable** — "Why did you recommend this?" → "Because Decision X
  concluded Y, supported by Insights A, B, C"
- **Weighable** — conflicting evidence is surfaced, not hidden
- **Revisable** — when new signals arrive, the decision can be re-evaluated
  without changing the action logic

---

## 2. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                      DATA INGESTION LAYER                           │
│                                                                     │
│  RSS Feed ──▶ Fetch ──▶ Article ──▶ LLM Extract ──▶ Knowledge      │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    DECISION INTELLIGENCE LAYER                       │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ LAYER 1: SIGNAL DETECTION          (Sprint 3.1)             │   │
│  │                                                             │   │
│  │  Knowledge ──▶ Rule Engine ──▶ Signals                       │   │
│  │  "What happened?"                                           │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                    │                                │
│                                    ▼                                │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ LAYER 2: INSIGHT SYNTHESIS        (Sprint 3.2)             │   │
│  │                                                             │   │
│  │  Signals ──▶ Pattern Engine ──▶ Insights                     │   │
│  │  "What does it mean?"                                       │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                    │                                │
│                                    ▼                                │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ LAYER 3: DECISION FORMATION      (Sprint 3.3)             │   │
│  │                                                             │   │
│  │  Insights ──▶ Evaluation Engine ──▶ Decisions                │   │
│  │  "What should we conclude?"                                  │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                    │                                │
│                                    ▼                                │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ LAYER 4: ACTION RECOMMENDATION    (Sprint 3.4)             │   │
│  │                                                             │   │
│  │  Decisions ──▶ Scoring Engine ──▶ Actions                    │   │
│  │  "What should we do?"                                       │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                    │                                │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      PRESENTATION LAYER     (Sprint 3.5)            │
│                                                                     │
│  Dashboard ──▶ Signals ──▶ Insights ──▶ Decisions ──▶ Actions       │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Component diagram

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│              │    │              │    │              │    │              │
│  KNOWLEDGE   │───▶│   SIGNALS    │───▶│   INSIGHTS   │───▶│  DECISIONS   │───▶ ACTIONS
│  (entities)  │    │  (events)    │    │  (patterns)  │    │ (conclusions)│
│              │    │              │    │              │    │              │
└──────────────┘    └──────────────┘    └──────────────┘    └──────────────┘
       │                   │                   │                   │
       │                   │                   │                   │
       ▼                   ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  LLM Extract │    │ Rule-Based   │    │  Pattern     │    │  Evaluation  │
│  + Validate  │    │ Mappings     │    │  Engine      │    │  Engine      │
│              │    │              │    │              │    │              │
│  Output:     │    │  Output:     │    │  Output:     │    │  Output:     │
│  topics      │    │  signal_type │    │  title       │    │  statement   │
│  industries  │    │  entity      │    │  description │    │  confidence  │
│  countries   │    │  confidence  │    │  confidence  │    │  impact      │
│  tech        │    │  detected_at │    │  signals     │    │  urgency     │
│  orgs        │    │              │    │  window      │    │  evidence    │
└──────────────┘    └──────────────┘    └──────────────┘    └──────────────┘
```

### Data flow

```
                    ┌─────────────────┐
                    │     Article     │
                    │   (raw text)    │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │   Knowledge     │
                    │   (entities)    │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │    Signals      │  ← 28 signal types from 5 entity categories
                    │   (events)      │     confidence-gated, idempotent
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │    Insights     │  ← 10 deterministic rules
                    │   (patterns)    │     time-windowed, composable
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │   Decisions     │  ← NEW LAYER
                    │  (conclusions)  │     weighted, evidence-traceable
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │    Actions      │  ← 5 action types
                    │ (recommendations)│    priority-scored, lifecycle-managed
                    └─────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │    Dashboard    │
                    │   (user view)   │
                    └─────────────────┘
```

### Design principles

1. **Each layer is independently testable.** The Signal layer can be verified
   against a golden dataset without the Insight layer existing. The Decision
   layer can be tested with mock insights.

2. **Each layer has a single responsibility.** Signals detect events. Insights
   find patterns. Decisions form conclusions. Actions recommend tasks.

3. **Layers communicate through well-defined interfaces.** The output of one
   layer is the input of the next. This allows swapping implementations.

4. **All transformation logic is deterministic.** No ML, no LLM calls, no
   probabilistic models in the decision layer. The same input always produces
   the same output. Critical for auditability and debugging.

5. **Every Decision is traceable.** The system must always answer "why did you
   make this recommendation?" by following the chain back through Decisions →
   Insights → Signals → Knowledge → Articles.

6. **The system is extensible by configuration.** Adding a new signal type
   requires a new row in the signal_types table, not new code. Adding a new
   decision rule uses an existing evaluation engine.

---

## 3. Layer 1: Knowledge → Signals

### Why this layer exists

The LLM extraction produces a bag of entities. This is the raw material, but
it lacks business semantics.

```
Knowledge Topics: [machine-learning, fintech, cybersecurity]
```

A human knows these represent *events* — an ML breakthrough, a fintech
product launch, a cybersecurity threat. But the system sees only strings.

The Signal layer adds the missing semantic layer: it maps each entity to a
named business event type, applies confidence thresholds, and normalizes
the output into a consistent format.

### Purpose

Transform raw taxonomy entities into named, normalized business events.

### Responsibilities

- Map each Knowledge entity to zero or more Signal types
- Apply minimum confidence thresholds to filter noise
- Deduplicate signals within a single article
- Store signals idempotently (running twice produces the same result)
- Preserve the original entity code and confidence for traceability

### Input

```
Knowledge {
  topics:        [{code: "machine-learning", confidence: 0.95}]
  industries:    [{code: "tech", confidence: 0.90}]
  countries:     [{code: "US", confidence: 0.95}]
  technologies:  [{code: "ai", confidence: 0.92}]
  organizations: [{code: "openai", confidence: 0.95}]
}
```

### Output

```
Signal { type: "machine_learning",     entity: "machine-learning", confidence: 0.95 }
Signal { type: "tech_growth",          entity: "tech",             confidence: 0.90 }
Signal { type: "geopolitical_event",   entity: "US",              confidence: 0.95 }
Signal { type: "technology_adoption",  entity: "ai",              confidence: 0.92 }
Signal { type: "investment",           entity: "openai",          confidence: 0.95 }
Signal { type: "partnership",          entity: "openai",          confidence: 0.95 }
```

### Processing flow

```
┌──────────────┐
│  Knowledge   │
│  from LLM    │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Lookup      │  For each entity, find matching signal type mappings
│  mappings    │  by entity_category + entity_code
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Confidence  │  Skip if entity.confidence < mapping.min_confidence
│  gate        │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Dedup       │  One signal type per entity per article
│  within      │  (e.g., don't emit "investment" twice for same org)
│  article     │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Store       │  Idempotent insert
│  idempotent  │  (article_id + signal_type + entity_code)
└──────────────┘
```

### Mapping rules

Each mapping is a triple: `(entity_category, entity_code, signal_type_code)`.

**Exact-match mappings** — fire when a specific entity code is detected:

| Entity Category | Entity Code | Signal Type |
|----------------|-------------|-------------|
| topics | machine-learning | machine_learning |
| topics | fintech | fintech_innovation |
| topics | cybersecurity | cybersecurity_event |
| topics | renewable-energy | renewable_energy |
| topics | biotechnology | biotech_breakthrough |
| topics | cloud-computing | cloud_adoption |
| topics | ai-regulation | ai_regulation |
| topics | supply-chain-mfg | supply_chain_event |
| topics | generative-ai | gen_ai_innovation |
| topics | blockchain | blockchain_adoption |
| topics | healthtech | healthtech_innovation |
| topics | edtech | edtech_growth |
| industries | tech | tech_growth |
| industries | finance | financial_activity |
| industries | healthcare | healthcare_development |
| industries | energy | energy_development |
| industries | manufacturing | manufacturing_activity |
| industries | agriculture | agriculture_development |

**Catch-all mappings** — fire for any entity in the category:

| Entity Category | Signal Type | Min Confidence |
|----------------|-------------|----------------|
| countries | geopolitical_event | 0.3 |
| countries | market_entry | 0.4 |
| technologies | technology_adoption | 0.4 |
| organizations | partnership | 0.3 |
| organizations | investment | 0.3 |
| organizations | acquisition | 0.3 |
| organizations | regulatory_action | 0.3 |
| organizations | research_publication | 0.3 |
| organizations | patent_filing | 0.3 |

### Why catch-all mappings exist

For organizations, the entity code alone cannot distinguish a partnership from
an investment from an acquisition. The catch-all approach produces all possible
signal types. The Insight layer (Layer 2) later disambiguates by looking at
co-occurring signals. For example, "investment" + "partnership" signals on the
same organization within the same period suggest a funding round with strategic
partners.

### Signal type categories

| Category | Signal types |
|----------|--------------|
| technology | machine_learning, biotech_breakthrough, cloud_adoption, gen_ai_innovation, blockchain_adoption, healthtech_innovation, technology_adoption, research_publication, patent_filing |
| business | fintech_innovation, supply_chain_event, edtech_growth, tech_growth, financial_activity, manufacturing_activity, market_entry, infrastructure_development, partnership, investment, acquisition |
| regulatory | cybersecurity_event, ai_regulation, geopolitical_event, regulatory_action |
| environmental | renewable_energy, agriculture_development |
| energy | energy_development |
| healthcare | healthcare_development |

### Database model

```
signal_types
├── id (UUID, PK)
├── code (VARCHAR, UNIQUE)          -- e.g. "machine_learning", "investment"
├── name (VARCHAR)                  -- e.g. "Machine Learning Advancement"
├── description (TEXT)
├── category (VARCHAR)              -- technology / business / regulatory
└── is_active (BOOLEAN)

signals
├── id (UUID, PK)
├── article_id (UUID, FK → articles)
├── signal_type_id (UUID, FK → signal_types)
├── entity_code (VARCHAR)           -- the original taxonomy code
├── entity_category (VARCHAR)       -- topics / industries / countries / tech / orgs
├── confidence (REAL)
├── detected_at (TIMESTAMPTZ)
└── created_at (TIMESTAMPTZ)

UNIQUE(article_id, signal_type_id, entity_code)  ← idempotency
```

---

## 4. Layer 2: Signals → Insights

### Why this layer exists

A single signal is still just a fact. "OpenAI received investment" is useful
but not actionable on its own. The Insight layer adds the intelligence that
makes signals useful:

- **Temporal context**: Was this investment part of a trend or an isolated event?
- **Cross-entity relationships**: Is the same topic appearing across multiple industries?
- **Pattern recognition**: Are multiple signals forming a coherent story?

Without this layer, users would need to manually connect the dots across
hundreds of signals. The Insight layer does this automatically.

### Purpose

Synthesize multiple signals into coherent, named patterns that represent
observable business phenomena.

### Responsibilities

- Match signals against deterministic pattern rules
- Group related signals into insights
- Compute insight confidence from constituent signal confidences
- Track insight time windows (first detected, last detected)
- Merge duplicate insights (same rule, same entities, overlapping time)
- Age out stale insights (no new signals in the lookback window)

### Input

```
Signal { type: "investment",       entity: "openai",    confidence: 0.95 }
Signal { type: "partnership",      entity: "microsoft", confidence: 0.90 }
Signal { type: "machine_learning", entity: "machine-learning", conf: 0.95 }
Signal { type: "gen_ai_innovation", entity: "generative-ai", conf: 0.95 }
```

### Output

```
Insight {
  type: "investment_signal_cluster"
  title: "OpenAI + Microsoft investment and partnership activity"
  confidence: 0.87
  category: "opportunity"
  signals: [investment, partnership]
  entities: [openai, microsoft]
  first_detected: "2026-07-08"
  last_detected: "2026-07-08"
}
```

### Processing flow

```
┌──────────────┐
│  New Signal  │
│  ingested    │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Query       │  Fetch active signals from lookback window
│  recent      │  (default: 7 days, same entity/same type)
│  signals     │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Match       │  Evaluate each rule against signal set
│  rules       │  Each rule: condition → insight template
│  against     │  Rules are deterministic functions
│  signals     │
└──────┬───────┘
       │
       ├── No match → exit (signal stored, no insight created)
       │
       ▼
┌──────────────┐
│  Check       │  Does this insight already exist?
│  existing    │  (same rule + same entities + overlapping window)
│  insights    │
└──────┬───────┘
       │
       ├── Exists → update: extend window, adjust confidence, link new signal
       │
       ▼
┌──────────────┐
│  Create      │  New insight record
│  new insight │  Link all constituent signals
│  + link      │
└──────────────┘
```

### Insight rules

Each rule is defined by:

1. **Condition**: a boolean expression over the signal set
2. **Template**: how to construct the insight title and description
3. **Parameters**: lookback window, minimum signal count, confidence formula

| Rule | Category | Condition | Min | Lookback |
|------|----------|-----------|-----|----------|
| **Signal concentration** | trend | ≥N signals of same type | 3 | 7 days |
| **Cross-sector overlap** | opportunity | signals from ≥2 industries share a topic | 2 | 3 days |
| **Geographic hotspot** | trend | ≥N signals from same country | 2 | 3 days |
| **Investment cluster** | opportunity | investment + partnership on same org | 2 | 3 days |
| **Regulatory cluster** | risk | ai_regulation + cybersecurity + regulatory_action | 2 | 7 days |
| **Technology convergence** | opportunity | ≥2 technology signals from same article | 2 | 1 day |
| **Supply chain risk** | risk | supply_chain + manufacturing | 1 | 3 days |
| **Emerging tech cycle** | trend | patent + research + gen_ai_innovation | 3 | 14 days |
| **Market momentum** | opportunity | investment + acquisition + partnership | 3 | 14 days |
| **Sector decay** | risk | ≥3 same-type signals with low confidence (<0.5) | 3 | 7 days |

### Rule examples

**Signal Concentration**
```
Condition: count(signals WHERE type = X) ≥ 3 in 7 days
Output:    "Machine learning cluster detected — 5 occurrences in 3 days"
```

**Cross-Sector Overlap**
```
Condition: count(DISTINCT industry) ≥ 2 AND topic appears in all
Output:    "AI impacting both technology and healthcare sectors"
```

**Geographic Hotspot**
```
Condition: count(signals WHERE country = X) ≥ 2 in 3 days
Output:    "Increased activity in Indonesia — 4 signals in 2 days"
```

**Investment Cluster**
```
Condition: investment + partnership signals on same org in 3 days
Output:    "OpenAI investment and partnership activity detected"
```

**Regulatory Cluster**
```
Condition: ai_regulation + cybersecurity + regulatory_action in 7 days
Output:    "Regulatory risk cluster: AI regulation + cybersecurity"
```

### Confidence composition

```
insight_confidence = avg_signal_confidence × coverage_factor × time_factor

coverage_factor = min(1.0, signal_count / min_signals_required)
time_factor     = max(0.5, 1.0 - days_since_first / decay_period)
```

- More signals → higher confidence (up to min required)
- Recent signals → higher confidence
- Stale insights decay over time

### Database model

```
insight_types
├── id (UUID, PK)
├── code (VARCHAR, UNIQUE)          -- e.g. "signal_concentration"
├── name (VARCHAR)
├── description (TEXT)
├── category (VARCHAR)              -- trend / opportunity / risk
├── min_signals (INTEGER)
└── is_active (BOOLEAN)

insights
├── id (UUID, PK)
├── insight_type_id (UUID, FK)
├── title (VARCHAR)
├── description (TEXT)
├── confidence (REAL)
├── signal_count (INTEGER)
├── first_detected_at (TIMESTAMPTZ)
├── last_detected_at (TIMESTAMPTZ)
├── is_active (BOOLEAN)
└── created_at / updated_at (TIMESTAMPTZ)

insight_signals (M2M bridge)
├── insight_id (UUID, FK)
├── signal_id (UUID, FK)
└── PK(insight_id, signal_id)
```

---

## 5. Layer 3: Insights → Decisions

### Why this layer exists

An Insight describes what is happening. But knowing what is happening is not
the same as knowing what to conclude from it.

Consider two insights arriving simultaneously:

```
Insight A: "Healthcare AI investment is increasing — 5 funding rounds in 7 days"
Insight B: "AI regulation is accelerating globally — 3 regulatory signals in 7 days"
```

These insights point in opposite directions. Insight A suggests opportunity.
Insight B suggests risk. Without a Decision layer, the system cannot resolve
this tension. It either:

- Produces both actions (diluting focus): "Invest in healthcare AI" AND
  "Prepare for AI regulation"
- Arbitrarily prioritizes one over the other
- Forces the user to resolve the conflict mentally

The Decision layer adds the missing reasoning step. It evaluates insights,
weighs conflicting evidence, considers risks and trade-offs, and produces a
structured conclusion.

### What is a Decision?

A Decision is a **conclusion supported by evidence**. It answers the question
"what should we think about this situation?" before the system asks "what
should we do about it?"

Each Decision includes:

- **Statement** — a clear conclusion: "Healthcare AI is a strategic opportunity."
- **Confidence** — how certain we are (0.0 – 1.0)
- **Impact** — the potential significance (low / medium / high)
- **Urgency** — how time-sensitive it is (low / medium / high)
- **Supporting Insights** — which insights led to this conclusion
- **Supporting Signals** — the underlying signals
- **Evidence summary** — a structured explanation of the reasoning
- **Risks** — factors that could make the decision wrong
- **Time horizon** — short-term / mid-term / long-term

### How is a Decision different from an Insight?

| | Insight | Decision |
|---|---|---|
| **Nature** | An observation | A conclusion |
| **Question** | "What is happening?" | "What should we conclude?" |
| **Tense** | "X is increasing" | "X is worth pursuing" |
| **Confidence** | Derived from signal math | Derived from insight + risk weighting |
| **Conflict** | Multiple insights can conflict | Decision resolves conflicts |
| **Risk** | No risk assessment | Explicit risk evaluation |
| **Trade-offs** | None | "Opportunity A outweighs Risk B" |
| **Time horizon** | Past → present | Present → future |

### How is a Decision different from an Action?

| | Decision | Action |
|---|---|---|
| **Nature** | A conclusion | A recommendation |
| **Question** | "What should we conclude?" | "What should we do?" |
| **Output** | "Healthcare AI is an opportunity" | "Learn Healthcare AI, build a SaaS" |
| **Ownership** | Strategic (what to think) | Operational (what to do) |
| **Reversibility** | Reversible (re-evaluate) | Costly to reverse |
| **Lifecycle** | Active / superseded | Pending / active / completed / dismissed |

### Purpose

Form structured conclusions from patterns, weighing evidence, risks, and
trade-offs.

### Responsibilities

- Evaluate insights against decision rules
- Weigh conflicting insights (opportunity vs risk)
- Calculate decision confidence from insight confidences + risk factors
- Assign impact and urgency
- Document evidence chains for traceability
- Detect when a decision should be superseded by new evidence

### Input

```
Insight {
  type: "investment_cluster"
  title: "Healthcare AI investment cluster — 5 funding rounds in 7 days"
  confidence: 0.91
  category: "opportunity"
  entities: [healthcare, ai, openai]
}

Insight {
  type: "regulatory_cluster"
  title: "AI regulation accelerating — 3 regulatory signals in 7 days"
  confidence: 0.85
  category: "risk"
  entities: [ai, GDPR, EU]
}
```

### Output

```
Decision {
  statement: "Healthcare AI is a high-confidence strategic opportunity
              worth pursuing, despite moderate regulatory headwinds."
  confidence: 0.78
  impact: "high"
  urgency: "medium"
  supporting_insights: [investment_cluster, regulatory_cluster]
  evidence_summary: "5 funding rounds indicate strong market momentum.
                     Regulatory signals exist but have not yet materialized
                     into binding legislation."
  risks: ["Regulatory changes could increase compliance costs",
          "Market saturation in AI healthcare"]
  time_horizon: "mid-term"
}
```

### Processing flow

```
┌──────────────┐
│  New Insight │
│  created     │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Gather      │  Fetch all active insights within scope
│  related     │  (same entities, overlapping time windows,
│  insights    │   related categories)
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Evaluate    │  For each decision rule:
│  decision    │    - Does the insight set match the condition?
│  rules       │    - What is the net confidence?
│              │    - What are the risks and trade-offs?
└──────┬───────┘
       │
       ├── No match → exit (insight stored, no decision created)
       │
       ▼
┌──────────────┐
│  Determine   │  Based on evidence strength:
│  confidence  │    - Base confidence from insight scores
│  + impact    │    - Risk discount for conflicting signals
│  + urgency   │    - Impact from scope (entity count, sectors)
│              │    - Urgency from time sensitivity
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Check       │  Does this decision already exist?
│  existing    │  (same statement + same entities)
│  decisions   │
└──────┬───────┘
       │
       ├── Exists → compare confidence:
       │    - New confidence higher → update decision
       │    - New confidence lower → no change (stable)
       │    - Contradictory evidence → flag for review
       │
       ▼
┌──────────────┐
│  Create      │  New decision with full evidence chain
│  decision    │  Link all supporting insights + signals
│  + link      │
└──────────────┘
```

### Decision rules

Each rule defines a condition over the insight set and produces a structured
decision.

| Rule | Condition | Produces | Confidence basis |
|------|-----------|----------|-----------------|
| **Strategic opportunity** | Opportunity insight ≥ 0.8 confidence, no conflicting risk insight | "X is a strategic opportunity" | avg(insight.confidence) × 0.9 |
| **Cautious opportunity** | Opportunity insight ≥ 0.6 AND conflicting risk insight exists | "X is an opportunity, but monitor Y" | avg(opp.confidence, 1-risk.confidence) |
| **Active risk** | Risk insight ≥ 0.8, no mitigating opportunity | "X is an active risk requiring preparation" | risk.confidence × 1.0 |
| **Balanced watch** | Risk + opportunity insight both present, similar confidence | "X requires balanced watch — opportunity and risk coexist" | avg(opp.confidence, risk.confidence) × 0.7 |
| **Emerging trend** | Trend insight, no clear opportunity or risk category | "X is an emerging trend worth monitoring" | trend.confidence × 0.8 |
| **Declining sector** | Sector decay insight active | "X sector is showing signs of decline" | decay.confidence × 0.9 |

### Confidence and impact determination

Decision confidence is not simply averaging insight confidences. It must
account for conflicting evidence:

```
decision_confidence = base_confidence × conflict_penalty × confidence_boost

base_confidence = avg(supporting_insight.confidence)
conflict_penalty = 1.0 - (conflicting_insight.confidence × 0.3)
confidence_boost = min(0.2, insight_count × 0.05)
```

**Impact** is determined by scope:

| Factor | Low | Medium | High |
|--------|-----|--------|------|
| Entities involved | 1 entity | 2–3 entities | 4+ entities |
| Sectors affected | 1 sector | 2 sectors | 3+ sectors |
| Signal volume | 1–2 signals | 3–5 signals | 6+ signals |
| Geographic scope | Local | Regional | Global |

**Urgency** is determined by time sensitivity:

| Factor | Low | Medium | High |
|--------|-----|--------|------|
| Signal density | Sparse (1/week) | Moderate (2–3/week) | Dense (daily) |
| First signal age | 30+ days ago | 7–30 days ago | < 7 days ago |
| Insight category | Trend | Opportunity | Risk |

### Evidence chain

Every Decision must be traceable back to the original articles:

```
Decision: "Healthcare AI is a strategic opportunity"
    │
    ├── Insight: "Healthcare AI investment cluster — 5 rounds in 7 days"
    │       │
    │       ├── Signal: investment (openai, 0.95) → Article: "OpenAI raises $6.5B"
    │       ├── Signal: partnership (microsoft, 0.90) → Article: "Microsoft partners with OpenAI"
    │       ├── Signal: investment (anthropic, 0.85) → Article: "Anthropic closes $2B round"
    │       └── Signal: investment (healthtech, 0.80) → Article: "Healthtech AI startup raises $50M"
    │
    └── Insight: "AI regulation accelerating globally"
            │
            ├── Signal: ai_regulation (0.95) → Article: "EU proposes AI Act amendments"
            └── Signal: regulatory_action (0.85) → Article: "FTC investigates AI companies"
```

### Decision lifecycle

```
                    ┌──────────┐
                    │  ACTIVE  │  Decision formed, supported by active insights
                    └────┬─────┘
                         │
                    ┌────▼─────┐
                    │ STRENGTHENED │  New supporting evidence increases confidence
                    └────┬─────┘
                         │
              ┌──────────┼──────────┐
              │          │          │
         ┌────▼───┐ ┌────▼───┐ ┌────▼───┐
         │SUPERSEDED│ │ UNCHANGED │ │WEAKENED│
         │(new evidence│ │(stable)  │ │(conflicting│
         │ contradicts)│ │          │ │ evidence) │
         └──────────┘ └──────────┘ └──────────┘
```

### Database model

```
decision_types
├── id (UUID, PK)
├── code (VARCHAR, UNIQUE)          -- e.g. "strategic_opportunity"
├── name (VARCHAR)
├── description (TEXT)
└── is_active (BOOLEAN)

decisions
├── id (UUID, PK)
├── decision_type_id (UUID, FK)
├── statement (TEXT)                 -- "Healthcare AI is a strategic opportunity"
├── confidence (REAL)
├── impact (VARCHAR)                 -- low / medium / high
├── urgency (VARCHAR)                -- low / medium / high
├── evidence_summary (TEXT)          -- structured reasoning
├── risks (JSONB)                    -- list of risk statements
├── time_horizon (VARCHAR)           -- short-term / mid-term / long-term
├── status (VARCHAR)                 -- active / strengthened / weakened / superseded
├── first_detected_at (TIMESTAMPTZ)
├── last_updated_at (TIMESTAMPTZ)
└── created_at (TIMESTAMPTZ)

decision_insights (M2M bridge)
├── decision_id (UUID, FK)
├── insight_id (UUID, FK)
└── PK(decision_id, insight_id)
```

---

## 6. Layer 4: Decisions → Actions

### Why this layer exists

A Decision tells us what to conclude. But the user still needs to know what to
do. "Healthcare AI is a strategic opportunity" is a conclusion, not a task.
The Action layer translates conclusions into actionable recommendations.

Without this layer:
- Users must manually translate decisions into tasks
- Critical decisions can be delayed ("I know this is important, but what do I do?")
- The system provides analysis but not execution guidance

### Purpose

Transform decisions into prioritized, actionable recommendations.

### Responsibilities

- Score decisions by priority, urgency, and confidence
- Map decisions to action types based on threshold rules
- Generate human-readable action descriptions
- Track action lifecycle (pending → active → completed / dismissed)
- Group related actions and deduplicate

### Input

```
Decision {
  statement: "Healthcare AI is a high-confidence strategic opportunity
              worth pursuing, despite moderate regulatory headwinds."
  confidence: 0.78
  impact: "high"
  urgency: "medium"
  risks: ["Regulatory changes could increase compliance costs"]
  time_horizon: "mid-term"
}
```

### Output

```
Action {
  type: "engage"
  title: "Engage with healthcare AI opportunity"
  priority: 7
  confidence: 0.78
  target: "healthcare AI sector"
  suggested_action: "Research top healthcare AI startups,
                     evaluate partnership opportunities,
                     monitor regulatory developments"
  status: "pending"
}
```

### Processing flow

```
┌──────────────┐
│  New Decision│
│  created     │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Score       │
│  priority    │  base_priority × confidence × urgency_multiplier
│  + urgency   │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Match       │
│  action      │  Map decision to action type
│  type        │  based on impact + urgency + confidence
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Generate    │
│  description │  Template-based: title, description, suggested action
│              │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Dedup       │  Same action for same target already active?
│  against     │  → Update confidence, extend window
│  existing    │  → Otherwise, create new
└──────────────┘
```

### Action types

| Code | Priority | When to trigger | Description |
|------|----------|-----------------|-------------|
| monitor | 3 | Decision: "emerging trend" or "balanced watch", impact low/medium | Watch this space — no immediate action needed |
| investigate | 5 | Decision: "cautious opportunity" or "balanced watch", impact medium | Look deeper — needs analysis before action |
| engage | 7 | Decision: "strategic opportunity", impact high, urgency medium/high | Take action — clear opportunity with next steps |
| prepare | 8 | Decision: "active risk", impact high, urgency medium/high | Get ready — risk identified, preemptive preparation needed |
| alert | 10 | Decision: "active risk", impact high, urgency high | Act now — imminent impact, requires immediate attention |

### Priority scoring

```
final_priority = base_priority × confidence_factor × urgency_factor

confidence_factor = 0.5 + (decision.confidence × 0.5)
urgency_factor    = 1.0 + urgency_multiplier(decision.urgency)
```

### Action lifecycle

```
                    ┌──────────┐
                    │ PENDING  │  Created by the system
                    └────┬─────┘
                         │
                    ┌────▼─────┐
                    │  ACTIVE  │  Reviewed and acknowledged by user
                    └────┬─────┘
                         │
              ┌──────────┼──────────┐
              │          │          │
         ┌────▼───┐ ┌────▼───┐ ┌────▼───┐
         │COMPLETED│ │DISMISSED│ │ESCALATED│
         │(resolved)│ │(rejected)│ │(priority↑)│
         └────────┘ └────────┘ └────────┘
```

### Decision-to-action mapping

| Decision characteristic | Action type | Example |
|------------------------|-------------|---------|
| emerging trend, low impact | monitor | "Monitor ML advancement in fintech" |
| cautious opportunity, medium impact | investigate | "Investigate AI regulation impact on healthcare" |
| strategic opportunity, high impact | engage | "Engage with healthcare AI opportunity" |
| active risk, high impact | prepare | "Prepare for EU AI compliance changes" |
| active risk, high urgency | alert | "ALERT: Supply chain disruption in manufacturing" |

### Database model

```
action_types
├── id (UUID, PK)
├── code (VARCHAR, UNIQUE)          -- e.g. "monitor", "engage", "alert"
├── name (VARCHAR)
├── description (TEXT)
├── priority (INTEGER)
└── is_active (BOOLEAN)

actions
├── id (UUID, PK)
├── action_type_id (UUID, FK)
├── title (VARCHAR)
├── description (TEXT)
├── priority (INTEGER)
├── confidence (REAL)
├── status (VARCHAR)                -- pending / active / completed / dismissed
├── target_entity (VARCHAR)         -- e.g. "healthcare AI", "EU market"
├── target_category (VARCHAR)       -- e.g. "sector", "region"
├── suggested_action (VARCHAR)      -- e.g. "Monitor", "Investigate"
├── generated_at (TIMESTAMPTZ)
└── created_at / updated_at (TIMESTAMPTZ)

action_decisions (M2M bridge)
├── action_id (UUID, FK)
├── decision_id (UUID, FK)
└── PK(action_id, decision_id)
```

---

## 7. End-to-End Flow

### Complete pipeline

```
                    DECISION PIPELINE
                    ═══════════════

    ARTICLE
    ────────
    "OpenAI raises $6.5B at $150B valuation.
     Microsoft leads the round with strategic
     cloud partnership. OpenAI plans to expand
     AI research and data centers in Southeast Asia."

        │
        ▼
    ─────────────────────────────────────────────────────────
    LAYER 0: EXTRACTION            (LLM — Sprint 2)
    ─────────────────────────────────────────────────────────

    Knowledge:
      Topics:        [generative-ai (0.95), machine-learning (0.90), cloud-computing (0.85)]
      Industries:    [tech (0.95), finance (0.70)]
      Countries:     [US (0.95), SG (0.85), ID (0.85)]
      Technologies:  [ai (0.95), cloud (0.90)]
      Organizations: [openai (0.95), microsoft (0.90)]

        │
        ▼
    ─────────────────────────────────────────────────────────
    LAYER 1: SIGNAL DETECTION       (Rule engine — Sprint 3.1)
    ─────────────────────────────────────────────────────────

    Signals (14):
      gen_ai_innovation       (generative-ai, 0.95)
      machine_learning        (machine-learning, 0.90)
      cloud_adoption          (cloud-computing, 0.85)
      tech_growth             (tech, 0.95)
      financial_activity      (finance, 0.70)
      geopolitical_event      (US, 0.95)  (SG, 0.85)  (ID, 0.85)
      market_entry            (US, 0.95)  (SG, 0.85)  (ID, 0.85)
      technology_adoption     (ai, 0.95)  (cloud, 0.90)
      partnership             (openai, 0.95)  (microsoft, 0.90)
      investment              (openai, 0.95)  (microsoft, 0.90)
      acquisition             (openai, 0.95)  (microsoft, 0.90)
      regulatory_action       (openai, 0.95)  (microsoft, 0.90)
      research_publication    (openai, 0.95)  (microsoft, 0.90)
      patent_filing           (openai, 0.95)  (microsoft, 0.90)

        │
        ▼
    ─────────────────────────────────────────────────────────
    LAYER 2: INSIGHT SYNTHESIS     (Pattern engine — Sprint 3.2)
    ─────────────────────────────────────────────────────────

    Existing signals in lookback window (7 days):
      [3 more investment signals for AI companies]

    Insights produced:
      ┌─────────────────────────────────────────────────────┐
      │ Investment Cluster (investment + partnership)       │
      │ "OpenAI + Microsoft investment round with          │
      │  strategic partnership"                             │
      │ Confidence: 0.87  |  Category: opportunity          │
      └─────────────────────────────────────────────────────┘
      ┌─────────────────────────────────────────────────────┐
      │ Signal Concentration (machine_learning × 4 in 7d)  │
      │ "ML advancement cluster: 4 occurrences in 7 days"  │
      │ Confidence: 0.92  |  Category: trend                │
      └─────────────────────────────────────────────────────┘
      ┌─────────────────────────────────────────────────────┐
      │ Geographic Hotspot (SG + ID)                        │
      │ "Increased activity in Southeast Asia — 3 signals  │
      │  in 2 days"                                         │
      │ Confidence: 0.85  |  Category: trend                │
      └─────────────────────────────────────────────────────┘
      ┌─────────────────────────────────────────────────────┐
      │ Technology Convergence (AI + cloud)                 │
      │ "Technology convergence: AI + cloud computing"      │
      │ Confidence: 0.90  |  Category: opportunity          │
      └─────────────────────────────────────────────────────┘

        │
        ▼
    ─────────────────────────────────────────────────────────
    LAYER 3: DECISION FORMATION     (Evaluation engine — Sprint 3.3)
    ─────────────────────────────────────────────────────────

    Decision rules evaluated:
      - Strategic opportunity?  Yes (investment_cluster + tech_convergence)
      - Active risk?            No (no conflicting risk insights)
      - Emerging trend?         Yes (ML concentration + geographic hotspot)

    Decisions produced:
      ┌─────────────────────────────────────────────────────┐
      │ DECISION: Strategic Opportunity                      │
      │                                                     │
      │ "OpenAI's Southeast Asia expansion is a high-       │
      │  confidence strategic opportunity in AI + cloud      │
      │  convergence."                                       │
      │                                                     │
      │ Confidence: 0.84  |  Impact: high  |  Urgency: mid  │
      │ Evidence: investment_cluster + tech_convergence +    │
      │           geographic_hotspot                         │
      │ Risks: ["Regulatory headwinds in Southeast Asia"]    │
      │ Time horizon: mid-term                               │
      └─────────────────────────────────────────────────────┘
      ┌─────────────────────────────────────────────────────┐
      │ DECISION: Emerging Trend                             │
      │                                                     │
      │ "ML advancement is accelerating across technology   │
      │  sector, with 4 occurrences in 7 days."              │
      │                                                     │
      │ Confidence: 0.74  |  Impact: medium  |  Urgency: low│
      │ Evidence: signal_concentration (ML)                  │
      │ Time horizon: long-term                              │
      └─────────────────────────────────────────────────────┘

        │
        ▼
    ─────────────────────────────────────────────────────────
    LAYER 4: ACTION RECOMMENDATION  (Scoring engine — Sprint 3.4)
    ─────────────────────────────────────────────────────────

    Actions produced:
      ┌─────────────────────────────────────────────────────┐
      │ ENGAGE  (Priority 7)                                 │
      │ "Engage with OpenAI's Southeast Asia expansion"     │
      │ Target: OpenAI + Southeast Asia market              │
      │ Suggested: "Research partnership opportunities,     │
      │  evaluate competitive landscape, monitor regulatory │
      │  developments in SG and ID"                         │
      │ Based on decision: strategic_opportunity             │
      └─────────────────────────────────────────────────────┘
      ┌─────────────────────────────────────────────────────┐
      │ MONITOR  (Priority 4)                                │
      │ "Monitor ML advancement in technology sector"       │
      │ Target: technology sector                           │
      │ Suggested: "Track ML research papers, product       │
      │  launches, and funding rounds"                      │
      │ Based on decision: emerging_trend                    │
      └─────────────────────────────────────────────────────┘
```

### Pipeline summary

| Layer | Input | Process | Output | Answers |
|-------|-------|---------|--------|---------|
| 0: Extract | Article text | LLM call | Knowledge | — |
| 1: Detect | Knowledge | Rule mapping | Signals | "What happened?" |
| 2: Synthesize | Signals | Pattern matching | Insights | "What does it mean?" |
| 3: Decide | Insights | Evaluation engine | Decisions | "What should we conclude?" |
| 4: Recommend | Decisions | Priority scoring | Actions | "What should we do?" |

---

## 8. Real-World Examples

### Example 1: Fintech Regulation (expanded)

This example shows the full pipeline with the Decision layer. The Decision
layer is critical here because Opportunity and Risk insights conflict.

**Article:** "EU Proposes New AI Guidelines for Financial Services"

```
Knowledge:
  Topics:        [ai-regulation (0.95), fintech (0.90)]
  Industries:    [finance (0.95), tech (0.80)]
  Countries:     [DE (0.90), GB (0.85)]
  Organizations: []

        │
        ▼

Signals (8):
  ai_regulation (0.95), fintech_innovation (0.90),
  financial_activity (0.95), tech_growth (0.80),
  geopolitical_event (0.90, 0.85), market_entry (0.90, 0.85)

        │
        ▼

Insights:
  ┌─────────────────────────────────────────────────────────┐
  │ INSIGHT A: Regulatory Cluster (risk)                    │
  │ "AI regulation + fintech cluster in EU"                 │
  │ Confidence: 0.88  |  Signals: 3  |  Category: risk      │
  └─────────────────────────────────────────────────────────┘
  ┌─────────────────────────────────────────────────────────┐
  │ INSIGHT B: Cross-Sector Overlap (opportunity)           │
  │ "AI regulation impacting both tech and finance"         │
  │ Confidence: 0.85  |  Signals: 2  |  Category: opp      │
  └─────────────────────────────────────────────────────────┘

        │
        ▼
    ──── WHY THE DECISION LAYER IS NEEDED ────

    Insight A says "risk" — regulatory action is coming.
    Insight B says "opportunity" — AI regulation creates
    market demand for compliance solutions.

    Without the Decision layer, the system would produce
    conflicting actions: "prepare for regulation" AND
    "invest in compliance tech" — with no guidance on
    which is more important.

    The Decision layer resolves this by weighing evidence,
    assessing trade-offs, and producing a single conclusion.

        │
        ▼

Decision:
  ┌─────────────────────────────────────────────────────────┐
  │ DECISION: Cautious Opportunity                          │
  │                                                         │
  │ "EU AI regulation in financial services is a high-      │
  │  impact cautious opportunity. The regulatory shift      │
  │  creates compliance demand, but implementation risk     │
  │  is significant. Pursue opportunity with active risk    │
  │  monitoring."                                           │
  │                                                         │
  │ Confidence: 0.72  |  Impact: high  |  Urgency: medium   │
  │                                                         │
  │ Evidence:                                                │
  │   - Insight A (regulatory_cluster, 0.88): confirms      │
  │     regulatory momentum                                  │
  │   - Insight B (cross_sector_overlap, 0.85): confirms    │
  │     market opportunity                                   │
  │                                                         │
  │ Risks:                                                   │
  │   - Regulatory scope may expand beyond current proposals│
  │   - Compliance costs may reduce profit margins          │
  │   - Timeline uncertainty (12-24 months to enactment)    │
  │                                                         │
  │ Time horizon: mid-term                                  │
  └─────────────────────────────────────────────────────────┘

        │
        ▼

Actions:
  ┌─────────────────────────────────────────────────────────┐
  │ INVESTIGATE (Priority 6)                                 │
  │ "Investigate EU AI regulation impact on fintech"        │
  │ Target: EU financial sector                             │
  │ Suggested: "Analyze current AI tooling for compliance,  │
  │  identify gaps, estimate compliance costs"              │
  │ Based on decision: cautious_opportunity                 │
  └─────────────────────────────────────────────────────────┘
  ┌─────────────────────────────────────────────────────────┐
  │ PREPARE (Priority 5)                                     │
  │ "Prepare for EU AI compliance in financial services"    │
  │ Target: EU financial sector                             │
  │ Suggested: "Review AI governance framework, audit       │
  │  data practices, engage compliance consultants"         │
  │ Based on decision: cautious_opportunity (risk factor)   │
  └─────────────────────────────────────────────────────────┘
```

**Why the Decision exists here:**

The two insights conflict: Insight A says "risk" (regulatory cluster), Insight B
says "opportunity" (cross-sector overlap). The system cannot simply produce both
actions without explaining which insight should guide the user's thinking.

The Decision layer resolves this conflict by:
1. **Weighing evidence** — both insights are high-confidence, so neither is dismissed
2. **Assessing trade-offs** — the opportunity exists but carries implementation risk
3. **Producing a balanced conclusion** — "cautious opportunity" rather than
   "pure opportunity" or "pure risk"
4. **Providing traceability** — the user can see exactly which insights and
   signals contributed to the conclusion

Without the Decision layer, the system would jump from conflicting insights
directly to actions, and the user would have no way to understand why the
system recommended both "investigate" and "prepare" — or whether one was
more important than the other.

---

### Example 2: Supply Chain Disruption

**Article:** "Semiconductor Shortage Forces Auto Manufacturers to Halt Production"

```
Knowledge:
  Topics:        [supply-chain-mfg (0.95)]
  Industries:    [manufacturing (0.95), tech (0.85)]
  Countries:     [JP (0.90), DE (0.85), US (0.80)]
  Technologies:  [iot (0.60)]
  Organizations: []

Signals (6):
  supply_chain_event (0.95), manufacturing_activity (0.95),
  tech_growth (0.85), geopolitical_event (0.90, 0.85, 0.80),
  market_entry (0.90, 0.85, 0.80), technology_adoption (0.60)

Insights:
  ┌─────────────────────────────────────────────────────────┐
  │ Supply Chain Risk (supply_chain + manufacturing)        │
  │ "Supply chain risk in manufacturing sector"             │
  │ Confidence: 0.92  |  Category: risk                     │
  └─────────────────────────────────────────────────────────┘
  ┌─────────────────────────────────────────────────────────┐
  │ Geographic Hotspot (JP + DE + US)                       │
  │ "Global semiconductor disruption: 3 countries affected" │
  │ Confidence: 0.90  |  Category: trend                    │
  └─────────────────────────────────────────────────────────┘

Decision:
  ┌─────────────────────────────────────────────────────────┐
  │ DECISION: Active Risk                                   │
  │                                                         │
  │ "Global semiconductor supply chain disruption is an     │
  │  active risk requiring immediate preparation. Three     │
  │  major manufacturing countries affected. No mitigating  │
  │  opportunity signals detected."                         │
  │                                                         │
  │ Confidence: 0.92  |  Impact: high  |  Urgency: high     │
  │ Evidence: supply_chain_risk (0.92) + geographic_hotspot │
  │ Risks: ["Production halt may extend to 6+ months",      │
  │         "Price increases across affected industries",   │
  │         "Potential job losses in manufacturing"]        │
  │ Time horizon: short-term                                │
  └─────────────────────────────────────────────────────────┘

Actions:
  ┌─────────────────────────────────────────────────────────┐
  │ ALERT (Priority 10)                                     │
  │ "ALERT: Global semiconductor supply chain disruption"   │
  │ Target: manufacturing sector                            │
  │ Suggested: "Identify alternative suppliers, review      │
  │  inventory buffers, assess production impact"           │
  │ Based on decision: active_risk                           │
  └─────────────────────────────────────────────────────────┘
  ┌─────────────────────────────────────────────────────────┐
  │ MONITOR (Priority 4)                                    │
  │ "Monitor semiconductor shortage in Japan, Germany, US"  │
  │ Target: JP/DE/US markets                                │
  │ Suggested: "Track semiconductor supply chain news"      │
  │ Based on decision: active_risk (geographic scope)       │
  └─────────────────────────────────────────────────────────┘
```

---

### Example 3: Emerging Tech Cycle

**Article:** "New Quantum Computing Patent Filed by MIT Researchers"

```
Knowledge:
  Topics:        [biotechnology (0.70), machine-learning (0.60)]
  Industries:    [tech (0.80), healthcare (0.50)]
  Countries:     [US (0.95)]
  Technologies:  [ai (0.75)]
  Organizations: [microsoft (0.60)]

Signals (multiple):
  biotech_breakthrough, machine_learning, tech_growth,
  geopolitical_event, market_entry, technology_adoption,
  partnership, investment, research_publication, patent_filing

Insights (with existing signals in 14-day window):
  ┌─────────────────────────────────────────────────────────┐
  │ Emerging Tech Cycle (patent + research + gen_ai)        │
  │ "Emerging tech cycle: quantum computing — 2 patent      │
  │  filings, 3 research papers, 1 product launch"          │
  │ Confidence: 0.78  |  Category: trend                    │
  └─────────────────────────────────────────────────────────┘

Decision:
  ┌─────────────────────────────────────────────────────────┐
  │ DECISION: Emerging Trend                                │
  │                                                         │
  │ "Quantum computing is an emerging technology cycle      │
  │  worth monitoring. Patent filings and research papers   │
  │  indicate early-stage development. No clear commercial  │
  │  opportunity signals yet."                              │
  │                                                         │
  │ Confidence: 0.62  |  Impact: low  |  Urgency: low       │
  │ Evidence: emerging_tech_cycle (0.78)                    │
  │ Risks: ["Technology readiness still low (5-10 years)",  │
  │         "Competing technologies may emerge"]            │
  │ Time horizon: long-term                                 │
  └─────────────────────────────────────────────────────────┘

Actions:
  ┌─────────────────────────────────────────────────────────┐
  │ MONITOR (Priority 3)                                    │
  │ "Monitor quantum computing patent activity"             │
  │ Target: quantum computing                               │
  │ Suggested: "Track patent filings, research output,      │
  │  and talent movement in quantum computing"              │
  │ Based on decision: emerging_trend                        │
  └─────────────────────────────────────────────────────────┘
```

---

## 9. Extensibility Guide

### Adding a new signal type

1. Add a row to the `signal_types` table (name, code, description, category)
2. Add a mapping entry: `(entity_category, entity_code → signal_type_code)`
3. No code changes beyond the mapping table

### Adding a new insight rule

1. Add a row to the `insight_types` table (code, name, category, min_signals)
2. Implement the rule condition function
3. Register the rule in the insight engine's rule registry

### Adding a new decision rule

1. Add a row to the `decision_types` table (code, name, description)
2. Implement the evaluation function (weighs insights, computes confidence)
3. Register the rule in the decision engine's rule registry

### Adding a new action type

1. Add a row to the `action_types` table (code, name, priority)
2. Add a threshold mapping in the action recommender configuration

### Provider-agnostic extraction

The `Provider` interface allows swapping LLM backends (Anthropic, Gemini, Groq,
Ollama) without changing the decision pipeline. Signal detection, insight
synthesis, decision formation, and action recommendation operate on normalized
data regardless of which LLM produced it.

### Traceability guarantee

Every decision must be traceable back through the pipeline:

```
Action ──▶ Decision ──▶ Insights ──▶ Signals ──▶ Knowledge ──▶ Article
  │            │            │            │            │            │
  │            │            │            │            │            │
  └────────────┴────────────┴────────────┴────────────┴────────────┘
                           "Why did you recommend this?"
```

The system must always answer this question without relying on opaque LLM
reasoning. Each layer stores explicit links to its inputs, forming a complete
evidence chain.

---

*Document version: 2.0 — Sprint 3.0 Decision Architecture (revised with Decision layer)*