-- +goose Up
-- ============================================================
-- Master Taxonomy: stakeholders (add hierarchy), technologies, companies
-- ============================================================

-- Add parent_id to stakeholders for hierarchy support
ALTER TABLE stakeholders
  ADD COLUMN IF NOT EXISTS parent_id UUID REFERENCES stakeholders(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_stakeholders_parent_id ON stakeholders (parent_id);

-- Technologies taxonomy
CREATE TABLE IF NOT EXISTS technologies (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code         VARCHAR(50)  NOT NULL,
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    parent_id    UUID REFERENCES technologies(id) ON DELETE SET NULL,
    is_active    BOOLEAN      DEFAULT true,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_technologies_code UNIQUE (code)
);

CREATE INDEX IF NOT EXISTS idx_technologies_parent_id  ON technologies (parent_id);
CREATE INDEX IF NOT EXISTS idx_technologies_is_active  ON technologies (is_active);

CREATE TRIGGER trg_technologies_updated_at
    BEFORE UPDATE ON technologies
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- Companies taxonomy
CREATE TABLE IF NOT EXISTS companies (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code         VARCHAR(50)  NOT NULL,
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    parent_id    UUID REFERENCES companies(id) ON DELETE SET NULL,
    is_active    BOOLEAN      DEFAULT true,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_companies_code UNIQUE (code)
);

CREATE INDEX IF NOT EXISTS idx_companies_parent_id  ON companies (parent_id);
CREATE INDEX IF NOT EXISTS idx_companies_is_active  ON companies (is_active);

CREATE TRIGGER trg_companies_updated_at
    BEFORE UPDATE ON companies
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- Seed: Stakeholders hierarchy
-- Step 1: Insert new top-level parents (codes not yet in DB)
INSERT INTO stakeholders (id, code, name, description, parent_id, is_active) VALUES
  ('a0000000-0000-4000-8000-00000000000f', 'institution', 'Institution', 'Institutional organizations', NULL, true),
  ('a0000000-0000-4000-8000-000000000014', 'individual',  'Individual',  'Individual persons and professionals', NULL, true)
ON CONFLICT (code) DO NOTHING;

-- Step 2: Set parent_id on existing children via code lookup
UPDATE stakeholders s SET parent_id = (SELECT id FROM stakeholders WHERE code = 'government')
  WHERE s.code IN ('central-gov', 'local-gov', 'regulator') AND EXISTS (SELECT 1 FROM stakeholders WHERE code = 'government');
UPDATE stakeholders s SET parent_id = (SELECT id FROM stakeholders WHERE code = 'enterprise')
  WHERE s.code IN ('startup', 'sme', 'mid-market', 'large-enterprise') AND EXISTS (SELECT 1 FROM stakeholders WHERE code = 'enterprise');
UPDATE stakeholders s SET parent_id = (SELECT id FROM stakeholders WHERE code = 'investor')
  WHERE s.code IN ('angel-investor', 'venture-capital', 'private-equity', 'institutional-inv') AND EXISTS (SELECT 1 FROM stakeholders WHERE code = 'investor');
UPDATE stakeholders s SET parent_id = (SELECT id FROM stakeholders WHERE code = 'institution')
  WHERE s.code IN ('university', 'hospital', 'research-institute', 'media') AND EXISTS (SELECT 1 FROM stakeholders WHERE code = 'institution');
UPDATE stakeholders s SET parent_id = (SELECT id FROM stakeholders WHERE code = 'individual')
  WHERE s.code IN ('farmer', 'professional', 'consumer') AND EXISTS (SELECT 1 FROM stakeholders WHERE code = 'individual');

-- Step 3: Insert new child stakeholders (referencing parent by code)
INSERT INTO stakeholders (id, code, name, description, parent_id, is_active) VALUES
  ('a0000000-0000-4000-8000-000000000002', 'central-gov',      'Central Government',   'National/federal government', (SELECT id FROM stakeholders WHERE code = 'government'), true),
  ('a0000000-0000-4000-8000-000000000003', 'local-gov',        'Local Government',     'Provincial/state/municipal government', (SELECT id FROM stakeholders WHERE code = 'government'), true),
  ('a0000000-0000-4000-8000-000000000004', 'regulator',        'Regulator',            'Regulatory bodies and agencies', (SELECT id FROM stakeholders WHERE code = 'government'), true),
  ('a0000000-0000-4000-8000-000000000008', 'mid-market',       'Mid-Market',           'Mid-sized companies', (SELECT id FROM stakeholders WHERE code = 'enterprise'), true),
  ('a0000000-0000-4000-8000-000000000009', 'large-enterprise', 'Large Enterprise',     'Large multinational corporations', (SELECT id FROM stakeholders WHERE code = 'enterprise'), true),
  ('a0000000-0000-4000-8000-00000000000b', 'angel-investor',   'Angel Investor',       'Individual early-stage investors', (SELECT id FROM stakeholders WHERE code = 'investor'), true),
  ('a0000000-0000-4000-8000-00000000000c', 'venture-capital',  'Venture Capital',      'VC firms and funds', (SELECT id FROM stakeholders WHERE code = 'investor'), true),
  ('a0000000-0000-4000-8000-00000000000d', 'private-equity',   'Private Equity',       'PE firms and buyout funds', (SELECT id FROM stakeholders WHERE code = 'investor'), true),
  ('a0000000-0000-4000-8000-00000000000e', 'institutional-inv','Institutional Investor','Pension funds, endowments, sovereign wealth', (SELECT id FROM stakeholders WHERE code = 'investor'), true),
  ('a0000000-0000-4000-8000-000000000011', 'research-institute','Research Institute',   'R&D organizations', (SELECT id FROM stakeholders WHERE code = 'institution'), true),
  ('a0000000-0000-4000-8000-000000000013', 'media',            'Media',                'News and content organizations', (SELECT id FROM stakeholders WHERE code = 'institution'), true),
  ('a0000000-0000-4000-8000-000000000016', 'professional',     'Professional',         'White-collar professionals', (SELECT id FROM stakeholders WHERE code = 'individual'), true),
  ('a0000000-0000-4000-8000-000000000017', 'consumer',         'Consumer',             'End consumers and general public', (SELECT id FROM stakeholders WHERE code = 'individual'), true)
ON CONFLICT (code) DO NOTHING;

-- Seed: Technologies
INSERT INTO technologies (id, code, name, description, parent_id, is_active) VALUES
  ('b0000000-0000-4000-8000-000000000001', 'ai',                'Artificial Intelligence',  'Machine-based intelligence and learning systems', NULL, true),
  ('b0000000-0000-4000-8000-000000000002', 'machine-learning',  'Machine Learning',        'Statistical learning models', 'b0000000-0000-4000-8000-000000000001', true),
  ('b0000000-0000-4000-8000-000000000003', 'nlp-llm',           'NLP / LLM',              'Natural language processing and large language models', 'b0000000-0000-4000-8000-000000000001', true),
  ('b0000000-0000-4000-8000-000000000004', 'computer-vision',   'Computer Vision',         'Image and video understanding', 'b0000000-0000-4000-8000-000000000001', true),
  ('b0000000-0000-4000-8000-000000000005', 'gen-ai',            'Generative AI',           'Content generation models', 'b0000000-0000-4000-8000-000000000001', true),
  ('b0000000-0000-4000-8000-000000000006', 'ai-agents',         'AI Agents',               'Autonomous AI agent systems', 'b0000000-0000-4000-8000-000000000001', true),
  ('b0000000-0000-4000-8000-000000000007', 'cloud',            'Cloud Computing',          'On-demand computing resources', NULL, true),
  ('b0000000-0000-4000-8000-000000000008', 'iaas',             'IaaS',                    'Infrastructure as a Service', 'b0000000-0000-4000-8000-000000000007', true),
  ('b0000000-0000-4000-8000-000000000009', 'paas',             'PaaS',                    'Platform as a Service', 'b0000000-0000-4000-8000-000000000007', true),
  ('b0000000-0000-4000-8000-00000000000a', 'saas',             'SaaS',                    'Software as a Service', 'b0000000-0000-4000-8000-000000000007', true),
  ('b0000000-0000-4000-8000-00000000000b', 'data-analytics',   'Data & Analytics',        'Data processing and analysis', NULL, true),
  ('b0000000-0000-4000-8000-00000000000c', 'big-data',         'Big Data',                'Large-scale data processing', 'b0000000-0000-4000-8000-00000000000b', true),
  ('b0000000-0000-4000-8000-00000000000d', 'data-engineering', 'Data Engineering',        'Data pipeline and infrastructure', 'b0000000-0000-4000-8000-00000000000b', true),
  ('b0000000-0000-4000-8000-00000000000e', 'bi',               'Business Intelligence',   'Reporting and dashboards', 'b0000000-0000-4000-8000-00000000000b', true),
  ('b0000000-0000-4000-8000-00000000000f', 'cybersecurity',    'Cybersecurity',            'Security and threat protection', NULL, true),
  ('b0000000-0000-4000-8000-000000000010', 'network-security', 'Network Security',        'Network protection', 'b0000000-0000-4000-8000-00000000000f', true),
  ('b0000000-0000-4000-8000-000000000011', 'app-security',     'Application Security',    'Software security', 'b0000000-0000-4000-8000-00000000000f', true),
  ('b0000000-0000-4000-8000-000000000012', 'identity',         'Identity & Access',       'IAM and authentication', 'b0000000-0000-4000-8000-00000000000f', true),
  ('b0000000-0000-4000-8000-000000000013', 'infrastructure',    'Infrastructure',          'Network and compute infrastructure', NULL, true),
  ('b0000000-0000-4000-8000-000000000014', '5g',               '5G / Telecom',            'Next-gen mobile networks', 'b0000000-0000-4000-8000-000000000013', true),
  ('b0000000-0000-4000-8000-000000000015', 'iot',              'IoT',                     'Internet of Things', 'b0000000-0000-4000-8000-000000000013', true),
  ('b0000000-0000-4000-8000-000000000016', 'edge-computing',   'Edge Computing',          'Distributed computing at edge', 'b0000000-0000-4000-8000-000000000013', true),
  ('b0000000-0000-4000-8000-000000000017', 'emerging-tech',    'Emerging Tech',           'Frontier and emerging technologies', NULL, true),
  ('b0000000-0000-4000-8000-000000000018', 'blockchain',       'Blockchain',              'Distributed ledger technology', 'b0000000-0000-4000-8000-000000000017', true),
  ('b0000000-0000-4000-8000-000000000019', 'quantum',          'Quantum Computing',       'Quantum information processing', 'b0000000-0000-4000-8000-000000000017', true),
  ('b0000000-0000-4000-8000-00000000001a', 'robotics',         'Robotics',                'Automated mechanical systems', 'b0000000-0000-4000-8000-000000000017', true),
  ('b0000000-0000-4000-8000-00000000001b', 'biotech',          'Biotechnology',           'Biological technology and engineering', NULL, true),
  ('b0000000-0000-4000-8000-00000000001c', 'healthtech',       'HealthTech',              'Healthcare technology', 'b0000000-0000-4000-8000-00000000001b', true),
  ('b0000000-0000-4000-8000-00000000001d', 'genomics',         'Genomics',                'Genetic sequencing and analysis', 'b0000000-0000-4000-8000-00000000001b', true),
  ('b0000000-0000-4000-8000-00000000001e', 'energy-tech',      'Energy Technology',       'Energy production and management tech', NULL, true),
  ('b0000000-0000-4000-8000-00000000001f', 'renewable-energy', 'Renewable Energy',        'Solar, wind, hydro, geothermal', 'b0000000-0000-4000-8000-00000000001e', true),
  ('b0000000-0000-4000-8000-000000000020', 'cleantech',        'CleanTech',               'Clean technology and sustainability', 'b0000000-0000-4000-8000-00000000001e', true),
  ('b0000000-0000-4000-8000-000000000021', 'mobility',         'Mobility & Transportation','Transportation technology', NULL, true),
  ('b0000000-0000-4000-8000-000000000022', 'ev',               'Electric Vehicles',       'EV and battery technology', 'b0000000-0000-4000-8000-000000000021', true),
  ('b0000000-0000-4000-8000-000000000023', 'autonomous-vehicles','Autonomous Vehicles',    'Self-driving vehicle technology', 'b0000000-0000-4000-8000-000000000021', true),
  ('b0000000-0000-4000-8000-000000000024', 'fintech',          'Financial Technology',    'Technology-enabled financial services', NULL, true),
  ('b0000000-0000-4000-8000-000000000025', 'payments',         'Payments',                'Digital payment systems', 'b0000000-0000-4000-8000-000000000024', true),
  ('b0000000-0000-4000-8000-000000000026', 'defi',             'DeFi',                    'Decentralized finance', 'b0000000-0000-4000-8000-000000000024', true),
  ('b0000000-0000-4000-8000-000000000027', 'embedded-finance', 'Embedded Finance',        'Embedded financial services', 'b0000000-0000-4000-8000-000000000024', true)
ON CONFLICT (code) DO NOTHING;

-- Seed: Companies
INSERT INTO companies (id, code, name, description, parent_id, is_active) VALUES
  -- Tech Giants
  ('c0000000-0000-4000-8000-000000000001', 'microsoft',         'Microsoft',              'Software, cloud, AI platforms', NULL, true),
  ('c0000000-0000-4000-8000-000000000002', 'google',            'Google / Alphabet',      'Search, cloud, AI, advertising', NULL, true),
  ('c0000000-0000-4000-8000-000000000003', 'amazon',            'Amazon',                 'E-commerce, cloud (AWS), AI', NULL, true),
  ('c0000000-0000-4000-8000-000000000004', 'meta',              'Meta',                   'Social media, VR, AI research', NULL, true),
  ('c0000000-0000-4000-8000-000000000005', 'apple',             'Apple',                  'Consumer hardware, services, AI', NULL, true),
  ('c0000000-0000-4000-8000-000000000006', 'nvidia',            'NVIDIA',                 'GPU, AI chips, accelerated computing', NULL, true),
  ('c0000000-0000-4000-8000-000000000007', 'tesla',             'Tesla',                  'EV, energy, autonomous driving', NULL, true),
  -- AI Companies
  ('c0000000-0000-4000-8000-000000000008', 'openai',             'OpenAI',                 'Leading AI/LLM research and products', NULL, true),
  ('c0000000-0000-4000-8000-000000000009', 'anthropic',          'Anthropic',              'AI safety research and Claude models', NULL, true),
  -- Cloud & Enterprise
  ('c0000000-0000-4000-8000-00000000000a', 'aws',                'AWS',                    'Amazon Web Services cloud platform', 'c0000000-0000-4000-8000-000000000003', true),
  ('c0000000-0000-4000-8000-00000000000b', 'gcp',                'Google Cloud',           'Google Cloud Platform', 'c0000000-0000-4000-8000-000000000002', true),
  ('c0000000-0000-4000-8000-00000000000c', 'azure',              'Microsoft Azure',        'Microsoft cloud platform', 'c0000000-0000-4000-8000-000000000001', true),
  ('c0000000-0000-4000-8000-00000000000d', 'oracle',             'Oracle',                 'Enterprise database and cloud', NULL, true),
  ('c0000000-0000-4000-8000-00000000000e', 'salesforce',         'Salesforce',             'CRM and enterprise SaaS', NULL, true),
  -- Indonesian Companies
  ('c0000000-0000-4000-8000-00000000000f', 'goto',               'GoTo Group',             'Indonesia digital ecosystem (Gojek+Tokopedia)', NULL, true),
  ('c0000000-0000-4000-8000-000000000010', 'traveloka',           'Traveloka',              'SE Asia travel and lifestyle platform', NULL, true),
  ('c0000000-0000-4000-8000-000000000011', 'bukalapak',           'Bukalapak',              'Indonesia e-commerce platform', NULL, true),
  ('c0000000-0000-4000-8000-000000000012', 'bank-mandiri',        'Bank Mandiri',           'Largest Indonesian state-owned bank', NULL, true),
  ('c0000000-0000-4000-8000-000000000013', 'bri',                'Bank BRI',               'Indonesian People''s Bank', NULL, true),
  ('c0000000-0000-4000-8000-000000000014', 'telkom-indonesia',    'Telkom Indonesia',       'Indonesia telecom and digital services', NULL, true),
  -- Consulting
  ('c0000000-0000-4000-8000-000000000015', 'mckinsey',           'McKinsey & Co',          'Global strategy consulting', NULL, true),
  ('c0000000-0000-4000-8000-000000000016', 'bcg',                'BCG',                    'Boston Consulting Group', NULL, true),
  ('c0000000-0000-4000-8000-000000000017', 'bain',               'Bain & Co',              'Global management consulting', NULL, true),
  -- Financial
  ('c0000000-0000-4000-8000-000000000018', 'jpmorgan',           'JPMorgan Chase',         'Largest US bank by assets', NULL, true),
  ('c0000000-0000-4000-8000-000000000019', 'goldman-sachs',      'Goldman Sachs',          'Global investment bank', NULL, true),
  ('c0000000-0000-4000-8000-00000000001a', 'stripe',             'Stripe',                 'Global payment processing platform', NULL, true),
  -- Social & Media
  ('c0000000-0000-4000-8000-00000000001b', 'tiktok',             'TikTok / ByteDance',     'Short video platform and AI', NULL, true),
  ('c0000000-0000-4000-8000-00000000001c', 'twitter',            'X / Twitter',            'Social media platform', NULL, true),
  -- Semiconductor
  ('c0000000-0000-4000-8000-00000000001d', 'tsmc',               'TSMC',                   'World''s largest semiconductor foundry', NULL, true),
  ('c0000000-0000-4000-8000-00000000001e', 'intel',              'Intel',                  'Semiconductor and processor manufacturer', NULL, true),
  ('c0000000-0000-4000-8000-00000000001f', 'amd',                'AMD',                    'CPU and GPU manufacturer', NULL, true)
ON CONFLICT (code) DO NOTHING;

-- +goose Down

DELETE FROM companies WHERE code IN (
  'microsoft','google','amazon','meta','apple','nvidia','tesla',
  'openai','anthropic',
  'aws','gcp','azure','oracle','salesforce',
  'goto','traveloka','bukalapak','bank-mandiri','bri','telkom-indonesia',
  'mckinsey','bcg','bain',
  'jpmorgan','goldman-sachs','stripe',
  'tiktok','twitter',
  'tsmc','intel','amd'
);

DELETE FROM technologies WHERE code LIKE 'tec-%';

DELETE FROM stakeholders WHERE code IN (
  'government','central-gov','local-gov','regulator',
  'enterprise','startup','sme','mid-market','large-enterprise',
  'investor','angel-investor','venture-capital','private-equity','institutional-inv',
  'institution','university','research-institute','hospital','media',
  'individual','farmer','professional','consumer'
);

ALTER TABLE stakeholders DROP COLUMN IF EXISTS parent_id;
DROP TABLE IF EXISTS companies CASCADE;
DROP TABLE IF EXISTS technologies CASCADE;