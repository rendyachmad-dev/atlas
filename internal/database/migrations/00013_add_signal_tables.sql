-- +goose Up
-- ============================================================
-- Signal Detection Engine: signal_types + signals
-- ============================================================

-- ============================================================
-- TABLE: signal_types
-- ============================================================

CREATE TABLE IF NOT EXISTS signal_types (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(50)  NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    category    VARCHAR(50)  NOT NULL DEFAULT 'general',
    is_active   BOOLEAN      DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_signal_types_code UNIQUE (code)
);

-- ============================================================
-- TABLE: signals
-- ============================================================

CREATE TABLE IF NOT EXISTS signals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id      UUID         NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    signal_type_id  UUID         NOT NULL REFERENCES signal_types(id) ON DELETE CASCADE,
    entity_code     VARCHAR(100) NOT NULL,
    entity_category VARCHAR(50)  NOT NULL,
    confidence      REAL         NOT NULL DEFAULT 0.0,
    detected_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    -- Idempotency: one signal per (article, signal_type, entity).
    CONSTRAINT uq_signals_article_type_entity UNIQUE (article_id, signal_type_id, entity_code)
);

CREATE INDEX IF NOT EXISTS idx_signals_article      ON signals (article_id);
CREATE INDEX IF NOT EXISTS idx_signals_type         ON signals (signal_type_id);
CREATE INDEX IF NOT EXISTS idx_signals_detected_at  ON signals (detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_signals_entity_code  ON signals (entity_code);
CREATE INDEX IF NOT EXISTS idx_signals_confidence   ON signals (confidence DESC);

-- ============================================================
-- Seed signal types from taxonomy mapping
-- ============================================================

-- Topic-based signals
INSERT INTO signal_types (code, name, description, category) VALUES
    ('machine_learning', 'Machine Learning Advancement',    'Breakthroughs or adoption of ML technology', 'technology'),
    ('fintech_innovation', 'Fintech Innovation',            'New financial technology products or services', 'business'),
    ('cybersecurity_event', 'Cybersecurity Event',          'Security breach, threat, or new defense measure', 'regulatory'),
    ('renewable_energy', 'Renewable Energy Development',    'Adoption or innovation in renewable energy', 'environmental'),
    ('biotech_breakthrough', 'Biotech Breakthrough',        'Scientific breakthrough in biotechnology', 'technology'),
    ('cloud_adoption', 'Cloud Computing Adoption',          'Cloud infrastructure migration or expansion', 'technology'),
    ('ai_regulation', 'AI Regulation',                      'New or proposed AI-related regulation', 'regulatory'),
    ('supply_chain_event', 'Supply Chain Event',            'Disruption, optimization, or shift in supply chain', 'business'),
    ('gen_ai_innovation', 'Generative AI Innovation',       'New generative AI model, product, or capability', 'technology'),
    ('blockchain_adoption', 'Blockchain Adoption',          'Blockchain implementation or expansion', 'technology'),
    ('healthtech_innovation', 'Healthtech Innovation',      'New health technology product or service', 'technology'),
    ('edtech_growth', 'Edtech Growth',                      'Education technology expansion or investment', 'business')
ON CONFLICT (code) DO NOTHING;

-- Industry-based signals
INSERT INTO signal_types (code, name, description, category) VALUES
    ('tech_growth', 'Technology Sector Growth',             'Expansion, investment, or innovation in tech sector', 'business'),
    ('financial_activity', 'Financial Activity',            'Banking, investment, or financial market activity', 'business'),
    ('healthcare_development', 'Healthcare Development',    'Healthcare sector development or reform', 'healthcare'),
    ('energy_development', 'Energy Sector Development',     'Energy production, policy, or market shift', 'energy'),
    ('manufacturing_activity', 'Manufacturing Activity',    'Manufacturing output, expansion, or disruption', 'business'),
    ('agriculture_development', 'Agriculture Development',  'Agricultural technology or policy change', 'environmental')
ON CONFLICT (code) DO NOTHING;

-- Country-based signals
INSERT INTO signal_types (code, name, description, category) VALUES
    ('geopolitical_event', 'Geopolitical Event',            'Cross-border political or economic event', 'regulatory'),
    ('market_entry', 'Market Entry or Expansion',           'Company entering or expanding in a new market', 'business')
ON CONFLICT (code) DO NOTHING;

-- Technology-based signals
INSERT INTO signal_types (code, name, description, category) VALUES
    ('technology_adoption', 'Technology Adoption',          'Adoption of a specific technology', 'technology'),
    ('infrastructure_development', 'Infrastructure Development', 'Infrastructure project or investment', 'business')
ON CONFLICT (code) DO NOTHING;

-- Organization-based signals
INSERT INTO signal_types (code, name, description, category) VALUES
    ('partnership', 'Partnership or Collaboration',         'Strategic partnership between organizations', 'business'),
    ('investment', 'Investment or Funding',                 'Investment, funding round, or capital raise', 'business'),
    ('acquisition', 'Acquisition or Merger',                'Company acquisition or merger activity', 'business'),
    ('regulatory_action', 'Regulatory Action',              'Regulatory action targeting an organization', 'regulatory'),
    ('research_publication', 'Research Publication',        'Research paper or study publication', 'technology'),
    ('patent_filing', 'Patent Filing',                      'New patent application or grant', 'technology')
ON CONFLICT (code) DO NOTHING;

-- +goose Down

DROP TABLE IF EXISTS signals CASCADE;
DROP TABLE IF EXISTS signal_types CASCADE;