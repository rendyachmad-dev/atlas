-- +goose Up
-- ============================================================
-- Taxonomy Layer: topics, countries, organizations + M2M bridges
-- ============================================================

-- ============================================================
-- TABLE: topics
-- ============================================================

CREATE TABLE IF NOT EXISTS topics (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code         VARCHAR(50)  NOT NULL,
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    parent_id    UUID REFERENCES topics(id) ON DELETE SET NULL,
    is_active    BOOLEAN      DEFAULT true,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_topics_code UNIQUE (code)
);

CREATE INDEX IF NOT EXISTS idx_topics_parent_id  ON topics (parent_id);
CREATE INDEX IF NOT EXISTS idx_topics_is_active  ON topics (is_active);

CREATE TRIGGER trg_topics_updated_at
    BEFORE UPDATE ON topics
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- ============================================================
-- TABLE: countries (ISO-3166)
-- ============================================================

CREATE TABLE IF NOT EXISTS countries (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code          CHAR(2)     NOT NULL,
    alpha3        CHAR(3)     NOT NULL,
    numeric_code  CHAR(3),
    name          VARCHAR(255) NOT NULL,
    official_name VARCHAR(255),
    region        VARCHAR(100),
    subregion     VARCHAR(100),
    is_active     BOOLEAN      DEFAULT true,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_countries_code   UNIQUE (code),
    CONSTRAINT uq_countries_alpha3 UNIQUE (alpha3)
);

CREATE INDEX IF NOT EXISTS idx_countries_region ON countries (region);
CREATE INDEX IF NOT EXISTS idx_countries_active ON countries (is_active);

-- ============================================================
-- TABLE: organizations
-- ============================================================

CREATE TABLE IF NOT EXISTS organizations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code          VARCHAR(50)  NOT NULL,
    name          VARCHAR(255) NOT NULL,
    type          VARCHAR(50),     -- 'company', 'government', 'ngo', 'academic', 'media', 'multilateral'
    description   TEXT,
    country_code  CHAR(2) REFERENCES countries(code) ON DELETE SET NULL,
    is_active     BOOLEAN      DEFAULT true,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_organizations_code UNIQUE (code)
);

CREATE INDEX IF NOT EXISTS idx_organizations_type ON organizations (type);
CREATE INDEX IF NOT EXISTS idx_organizations_active ON organizations (is_active);
CREATE INDEX IF NOT EXISTS idx_organizations_country ON organizations (country_code);

CREATE TRIGGER trg_organizations_updated_at
    BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- ============================================================
-- M2M Junction Tables: article <-> taxonomy entities
-- ============================================================

-- Article <-> Industries
CREATE TABLE IF NOT EXISTS article_industries (
    article_id  UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    industry_id UUID NOT NULL REFERENCES industries(id) ON DELETE CASCADE,

    CONSTRAINT pk_article_industries PRIMARY KEY (article_id, industry_id)
);
CREATE INDEX IF NOT EXISTS idx_article_industries_article  ON article_industries (article_id);
CREATE INDEX IF NOT EXISTS idx_article_industries_industry ON article_industries (industry_id);

-- Article <-> Topics
CREATE TABLE IF NOT EXISTS article_topics (
    article_id UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    topic_id   UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,

    CONSTRAINT pk_article_topics PRIMARY KEY (article_id, topic_id)
);
CREATE INDEX IF NOT EXISTS idx_article_topics_article ON article_topics (article_id);
CREATE INDEX IF NOT EXISTS idx_article_topics_topic   ON article_topics (topic_id);

-- Article <-> Countries
CREATE TABLE IF NOT EXISTS article_countries (
    article_id  UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    country_id  UUID NOT NULL REFERENCES countries(id) ON DELETE CASCADE,

    CONSTRAINT pk_article_countries PRIMARY KEY (article_id, country_id)
);
CREATE INDEX IF NOT EXISTS idx_article_countries_article ON article_countries (article_id);
CREATE INDEX IF NOT EXISTS idx_article_countries_country ON article_countries (country_id);

-- Article <-> Technologies
CREATE TABLE IF NOT EXISTS article_technologies (
    article_id     UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    technology_id  UUID NOT NULL REFERENCES technologies(id) ON DELETE CASCADE,

    CONSTRAINT pk_article_technologies PRIMARY KEY (article_id, technology_id)
);
CREATE INDEX IF NOT EXISTS idx_article_technologies_article     ON article_technologies (article_id);
CREATE INDEX IF NOT EXISTS idx_article_technologies_technology  ON article_technologies (technology_id);

-- Article <-> Organizations
CREATE TABLE IF NOT EXISTS article_organizations (
    article_id       UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    organization_id  UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    CONSTRAINT pk_article_organizations PRIMARY KEY (article_id, organization_id)
);
CREATE INDEX IF NOT EXISTS idx_article_organizations_article  ON article_organizations (article_id);
CREATE INDEX IF NOT EXISTS idx_article_organizations_org      ON article_organizations (organization_id);

-- ============================================================
-- SEED: Topics (120+ entries across 12 categories)
-- ============================================================

-- Categories (parent topics, no parent_id)
INSERT INTO topics (id, code, name, description, parent_id) VALUES
  -- AI & Technology
  ('d0000000-0000-4000-8000-000000000001', 'ai-ml', 'Artificial Intelligence & ML', 'AI, machine learning, and data science topics', NULL),
  ('d0000000-0000-4000-8000-000000000002', 'software-dev', 'Software Development', 'Programming, engineering, and DevOps topics', NULL),
  ('d0000000-0000-4000-8000-000000000003', 'hardware', 'Hardware & Semiconductors', 'Chips, processors, and physical computing', NULL),
  -- Healthcare
  ('d0000000-0000-4000-8000-000000000004', 'healthcare', 'Healthcare & Biotech', 'Medical, pharmaceutical, and biotechnology topics', NULL),
  -- Agriculture
  ('d0000000-0000-4000-8000-000000000005', 'agriculture', 'Agriculture & Food', 'Farming, food production, and agritech', NULL),
  -- Cybersecurity
  ('d0000000-0000-4000-8000-000000000006', 'cybersecurity', 'Cybersecurity & Privacy', 'Security, privacy, and threat protection', NULL),
  -- Finance
  ('d0000000-0000-4000-8000-000000000007', 'finance', 'Finance & Economics', 'Banking, finance, investment, and economic policy', NULL),
  -- Regulation
  ('d0000000-0000-4000-8000-000000000008', 'regulation', 'Regulation & Policy', 'Laws, regulations, and government policy', NULL),
  -- Manufacturing
  ('d0000000-0000-4000-8000-000000000009', 'manufacturing', 'Manufacturing & Industry', 'Industrial production and supply chain', NULL),
  -- Climate & Energy
  ('d0000000-0000-4000-8000-00000000000a', 'climate-energy', 'Climate & Energy', 'Climate change, energy production, and sustainability', NULL),
  -- Logistics
  ('d0000000-0000-4000-8000-00000000000b', 'logistics', 'Logistics & Transport', 'Transportation, shipping, and mobility', NULL),
  -- Education
  ('d0000000-0000-4000-8000-00000000000c', 'education', 'Education & Workforce', 'Learning, training, and labor markets', NULL)
ON CONFLICT (code) DO NOTHING;

-- Sub-topics (reference parent by UUID from the INSERT above)
INSERT INTO topics (id, code, name, description, parent_id) VALUES
  -- AI & ML sub-topics
  ('d0000000-0000-4000-8000-000000000101', 'machine-learning', 'Machine Learning', 'Statistical learning models and algorithms', 'd0000000-0000-4000-8000-000000000001'),
  ('d0000000-0000-4000-8000-000000000102', 'deep-learning', 'Deep Learning', 'Neural networks and deep learning architectures', 'd0000000-0000-4000-8000-000000000001'),
  ('d0000000-0000-4000-8000-000000000103', 'nlp', 'Natural Language Processing', 'LLMs, text analysis, and language models', 'd0000000-0000-4000-8000-000000000001'),
  ('d0000000-0000-4000-8000-000000000104', 'computer-vision', 'Computer Vision', 'Image and video understanding', 'd0000000-0000-4000-8000-000000000001'),
  ('d0000000-0000-4000-8000-000000000105', 'generative-ai', 'Generative AI', 'Content generation via AI models', 'd0000000-0000-4000-8000-000000000001'),
  ('d0000000-0000-4000-8000-000000000106', 'reinforcement-learning', 'Reinforcement Learning', 'RL, game theory, and autonomous decision-making', 'd0000000-0000-4000-8000-000000000001'),
  ('d0000000-0000-4000-8000-000000000107', 'ai-agents', 'AI Agents', 'Autonomous agent systems and multi-agent coordination', 'd0000000-0000-4000-8000-000000000001'),
  ('d0000000-0000-4000-8000-000000000108', 'ai-safety', 'AI Safety', 'Alignment, robustness, and responsible AI', 'd0000000-0000-4000-8000-000000000001'),
  ('d0000000-0000-4000-8000-000000000109', 'data-science', 'Data Science', 'Data analysis, statistics, and visualization', 'd0000000-0000-4000-8000-000000000001'),
  ('d0000000-0000-4000-8000-00000000010a', 'robotics', 'Robotics', 'Robotic systems and automation', 'd0000000-0000-4000-8000-000000000001'),
  ('d0000000-0000-4000-8000-00000000010b', 'edge-ai', 'Edge AI', 'AI inference on edge devices', 'd0000000-0000-4000-8000-000000000001'),
  -- Software Dev sub-topics
  ('d0000000-0000-4000-8000-000000000201', 'cloud-computing', 'Cloud Computing', 'IaaS, PaaS, SaaS, and cloud infrastructure', 'd0000000-0000-4000-8000-000000000002'),
  ('d0000000-0000-4000-8000-000000000202', 'devops', 'DevOps & SRE', 'CI/CD, monitoring, and site reliability', 'd0000000-0000-4000-8000-000000000002'),
  ('d0000000-0000-4000-8000-000000000203', 'open-source', 'Open Source', 'Open source software and communities', 'd0000000-0000-4000-8000-000000000002'),
  ('d0000000-0000-4000-8000-000000000204', 'api-development', 'API Development', 'REST, GraphQL, and API design patterns', 'd0000000-0000-4000-8000-000000000002'),
  ('d0000000-0000-4000-8000-000000000205', 'mobile-dev', 'Mobile Development', 'iOS, Android, and cross-platform development', 'd0000000-0000-4000-8000-000000000002'),
  ('d0000000-0000-4000-8000-000000000206', 'web-dev', 'Web Development', 'Frontend, backend, and full-stack web', 'd0000000-0000-4000-8000-000000000002'),
  ('d0000000-0000-4000-8000-000000000207', 'database', 'Databases', 'SQL, NoSQL, and data storage', 'd0000000-0000-4000-8000-000000000002'),
  ('d0000000-0000-4000-8000-000000000208', 'programming-languages', 'Programming Languages', 'Language design and ecosystem', 'd0000000-0000-4000-8000-000000000002'),
  -- Hardware sub-topics
  ('d0000000-0000-4000-8000-000000000301', 'semiconductors', 'Semiconductors', 'Chip design, fabs, and fabrication', 'd0000000-0000-4000-8000-000000000003'),
  ('d0000000-0000-4000-8000-000000000302', 'gpu-computing', 'GPU Computing', 'Graphics processors and parallel computing', 'd0000000-0000-4000-8000-000000000003'),
  ('d0000000-0000-4000-8000-000000000303', 'iot', 'Internet of Things', 'Connected devices and sensor networks', 'd0000000-0000-4000-8000-000000000003'),
  ('d0000000-0000-4000-8000-000000000304', 'quantum-computing', 'Quantum Computing', 'Quantum information processing', 'd0000000-0000-4000-8000-000000000003'),
  ('d0000000-0000-4000-8000-000000000305', 'embedded-systems', 'Embedded Systems', 'Firmware and embedded software', 'd0000000-0000-4000-8000-000000000003'),
  -- Healthcare sub-topics
  ('d0000000-0000-4000-8000-000000000401', 'biotechnology', 'Biotechnology', 'Bio-based innovation and products', 'd0000000-0000-4000-8000-000000000004'),
  ('d0000000-0000-4000-8000-000000000402', 'healthtech', 'HealthTech', 'Technology-enabled healthcare solutions', 'd0000000-0000-4000-8000-000000000004'),
  ('d0000000-0000-4000-8000-000000000403', 'pharmaceuticals', 'Pharmaceuticals', 'Drug development and clinical trials', 'd0000000-0000-4000-8000-000000000004'),
  ('d0000000-0000-4000-8000-000000000404', 'telemedicine', 'Telemedicine', 'Remote healthcare and virtual consultations', 'd0000000-0000-4000-8000-000000000004'),
  ('d0000000-0000-4000-8000-000000000405', 'genomics', 'Genomics', 'Genetic sequencing and personalized medicine', 'd0000000-0000-4000-8000-000000000004'),
  ('d0000000-0000-4000-8000-000000000406', 'medical-devices', 'Medical Devices', 'Diagnostic and therapeutic devices', 'd0000000-0000-4000-8000-000000000004'),
  ('d0000000-0000-4000-8000-000000000407', 'mental-health', 'Mental Health', 'Psychological wellness and therapy', 'd0000000-0000-4000-8000-000000000004'),
  ('d0000000-0000-4000-8000-000000000408', 'public-health', 'Public Health', 'Population health, epidemiology, and prevention', 'd0000000-0000-4000-8000-000000000004'),
  ('d0000000-0000-4000-8000-000000000409', 'aging-longevity', 'Aging & Longevity', 'Age-related research and longevity science', 'd0000000-0000-4000-8000-000000000004'),
  ('d0000000-0000-4000-8000-00000000040a', 'vaccines', 'Vaccines & Immunology', 'Vaccine development and immune system research', 'd0000000-0000-4000-8000-000000000004'),
  -- Agriculture sub-topics
  ('d0000000-0000-4000-8000-000000000501', 'agritech', 'AgriTech', 'Technology in agriculture and farming', 'd0000000-0000-4000-8000-000000000005'),
  ('d0000000-0000-4000-8000-000000000502', 'precision-farming', 'Precision Farming', 'Data-driven precision agriculture', 'd0000000-0000-4000-8000-000000000005'),
  ('d0000000-0000-4000-8000-000000000503', 'food-tech', 'Food Technology', 'Food processing, alternative proteins, and innovation', 'd0000000-0000-4000-8000-000000000005'),
  ('d0000000-0000-4000-8000-000000000504', 'supply-chain-food', 'Food Supply Chain', 'Farm-to-table logistics and food distribution', 'd0000000-0000-4000-8000-000000000005'),
  ('d0000000-0000-4000-8000-000000000505', 'vertical-farming', 'Vertical Farming', 'Indoor and urban farming technologies', 'd0000000-0000-4000-8000-000000000005'),
  ('d0000000-0000-4000-8000-000000000506', 'aquaculture', 'Aquaculture', 'Fish farming and marine agriculture', 'd0000000-0000-4000-8000-000000000005'),
  -- Cybersecurity sub-topics
  ('d0000000-0000-4000-8000-000000000601', 'threat-intelligence', 'Threat Intelligence', 'Cyber threat detection and analysis', 'd0000000-0000-4000-8000-000000000006'),
  ('d0000000-0000-4000-8000-000000000602', 'zero-trust', 'Zero Trust', 'Zero trust security architecture', 'd0000000-0000-4000-8000-000000000006'),
  ('d0000000-0000-4000-8000-000000000603', 'encryption', 'Encryption & Cryptography', 'Data protection and cryptographic systems', 'd0000000-0000-4000-8000-000000000006'),
  ('d0000000-0000-4000-8000-000000000604', 'network-security', 'Network Security', 'Perimeter defense and network protection', 'd0000000-0000-4000-8000-000000000006'),
  ('d0000000-0000-4000-8000-000000000605', 'cloud-security', 'Cloud Security', 'Cloud infrastructure and workload protection', 'd0000000-0000-4000-8000-000000000006'),
  ('d0000000-0000-4000-8000-000000000606', 'identity-security', 'Identity & Access', 'IAM, authentication, and authorization', 'd0000000-0000-4000-8000-000000000006'),
  ('d0000000-0000-4000-8000-000000000607', 'ransomware', 'Ransomware & Malware', 'Malware defense and incident response', 'd0000000-0000-4000-8000-000000000006'),
  ('d0000000-0000-4000-8000-000000000608', 'data-privacy', 'Data Privacy', 'Privacy regulations and data protection', 'd0000000-0000-4000-8000-000000000006'),
  ('d0000000-0000-4000-8000-000000000609', 'app-security', 'Application Security', 'Secure software development and DevSecOps', 'd0000000-0000-4000-8000-000000000006'),
  -- Finance sub-topics
  ('d0000000-0000-4000-8000-000000000701', 'fintech', 'FinTech', 'Technology-enabled financial services', 'd0000000-0000-4000-8000-000000000007'),
  ('d0000000-0000-4000-8000-000000000702', 'digital-banking', 'Digital Banking', 'Online banking and neobanks', 'd0000000-0000-4000-8000-000000000007'),
  ('d0000000-0000-4000-8000-000000000703', 'payments', 'Payments', 'Digital payments and remittances', 'd0000000-0000-4000-8000-000000000007'),
  ('d0000000-0000-4000-8000-000000000704', 'blockchain', 'Blockchain & DLT', 'Distributed ledger technology', 'd0000000-0000-4000-8000-000000000007'),
  ('d0000000-0000-4000-8000-000000000705', 'crypto', 'Cryptocurrency', 'Digital currencies and tokens', 'd0000000-0000-4000-8000-000000000007'),
  ('d0000000-0000-4000-8000-000000000706', 'defi', 'DeFi', 'Decentralized finance protocols', 'd0000000-0000-4000-8000-000000000007'),
  ('d0000000-0000-4000-8000-000000000707', 'investment', 'Investment & Trading', 'Stock markets, trading, and portfolio management', 'd0000000-0000-4000-8000-000000000007'),
  ('d0000000-0000-4000-8000-000000000708', 'insurtech', 'InsurTech', 'Technology in insurance', 'd0000000-0000-4000-8000-000000000007'),
  ('d0000000-0000-4000-8000-000000000709', 'central-banking', 'Central Banking', 'Monetary policy and central bank operations', 'd0000000-0000-4000-8000-000000000007'),
  ('d0000000-0000-4000-8000-00000000070a', 'microfinance', 'Microfinance', 'Small-scale financial services for underserved', 'd0000000-0000-4000-8000-000000000007'),
  -- Regulation sub-topics
  ('d0000000-0000-4000-8000-000000000801', 'ai-regulation', 'AI Regulation', 'AI governance and regulatory frameworks', 'd0000000-0000-4000-8000-000000000008'),
  ('d0000000-0000-4000-8000-000000000802', 'data-protection', 'Data Protection Laws', 'GDPR, CCPA, and data sovereignty', 'd0000000-0000-4000-8000-000000000008'),
  ('d0000000-0000-4000-8000-000000000803', 'antitrust', 'Antitrust & Competition', 'Competition law and market regulation', 'd0000000-0000-4000-8000-000000000008'),
  ('d0000000-0000-4000-8000-000000000804', 'trade-policy', 'Trade Policy', 'International trade agreements and tariffs', 'd0000000-0000-4000-8000-000000000008'),
  ('d0000000-0000-4000-8000-000000000805', 'environmental-regulation', 'Environmental Regulation', 'Climate and environmental policy', 'd0000000-0000-4000-8000-000000000008'),
  ('d0000000-0000-4000-8000-000000000806', 'financial-regulation', 'Financial Regulation', 'Banking and securities regulation', 'd0000000-0000-4000-8000-000000000008'),
  ('d0000000-0000-4000-8000-000000000807', 'health-regulation', 'Health Regulation', 'Healthcare compliance and FDA/EMA approvals', 'd0000000-0000-4000-8000-000000000008'),
  ('d0000000-0000-4000-8000-000000000808', 'digital-sovereignty', 'Digital Sovereignty', 'Data localization and digital independence', 'd0000000-0000-4000-8000-000000000008'),
  ('d0000000-0000-4000-8000-000000000809', 'consumer-protection', 'Consumer Protection', 'Consumer rights and product safety', 'd0000000-0000-4000-8000-000000000008'),
  -- Manufacturing sub-topics
  ('d0000000-0000-4000-8000-000000000901', 'industry-4.0', 'Industry 4.0', 'Smart manufacturing and digital factories', 'd0000000-0000-4000-8000-000000000009'),
  ('d0000000-0000-4000-8000-000000000902', 'additive-manufacturing', 'Additive Manufacturing', '3D printing and advanced fabrication', 'd0000000-0000-4000-8000-000000000009'),
  ('d0000000-0000-4000-8000-000000000903', 'automation', 'Industrial Automation', 'Automated production and process control', 'd0000000-0000-4000-8000-000000000009'),
  ('d0000000-0000-4000-8000-000000000904', 'supply-chain-mfg', 'Supply Chain Management', 'Production supply chain and sourcing', 'd0000000-0000-4000-8000-000000000009'),
  ('d0000000-0000-4000-8000-000000000905', 'quality-control', 'Quality Control', 'Inspection, testing, and quality assurance', 'd0000000-0000-4000-8000-000000000009'),
  ('d0000000-0000-4000-8000-000000000906', 'advanced-materials', 'Advanced Materials', 'New materials and composites', 'd0000000-0000-4000-8000-000000000009'),
  ('d0000000-0000-4000-8000-000000000907', 'ev-manufacturing', 'EV Manufacturing', 'Electric vehicle production and battery manufacturing', 'd0000000-0000-4000-8000-000000000009'),
  -- Climate & Energy sub-topics
  ('d0000000-0000-4000-8000-000000000a01', 'renewable-energy', 'Renewable Energy', 'Solar, wind, hydro, and geothermal power', 'd0000000-0000-4000-8000-00000000000a'),
  ('d0000000-0000-4000-8000-000000000a02', 'solar-power', 'Solar Power', 'Photovoltaic and solar thermal energy', 'd0000000-0000-4000-8000-00000000000a'),
  ('d0000000-0000-4000-8000-000000000a03', 'wind-power', 'Wind Power', 'Onshore and offshore wind energy', 'd0000000-0000-4000-8000-00000000000a'),
  ('d0000000-0000-4000-8000-000000000a04', 'carbon-capture', 'Carbon Capture', 'Carbon capture, utilization, and storage', 'd0000000-0000-4000-8000-00000000000a'),
  ('d0000000-0000-4000-8000-000000000a05', 'climate-tech', 'Climate Tech', 'Technology solutions for climate change', 'd0000000-0000-4000-8000-00000000000a'),
  ('d0000000-0000-4000-8000-000000000a06', 'battery-tech', 'Battery Technology', 'Energy storage and battery innovation', 'd0000000-0000-4000-8000-00000000000a'),
  ('d0000000-0000-4000-8000-000000000a07', 'nuclear-energy', 'Nuclear Energy', 'Nuclear power and advanced reactors', 'd0000000-0000-4000-8000-00000000000a'),
  ('d0000000-0000-4000-8000-000000000a08', 'hydrogen', 'Hydrogen Economy', 'Green hydrogen and fuel cells', 'd0000000-0000-4000-8000-00000000000a'),
  ('d0000000-0000-4000-8000-000000000a09', 'sustainability', 'Sustainability', 'ESG, circular economy, and sustainable practices', 'd0000000-0000-4000-8000-00000000000a'),
  ('d0000000-0000-4000-8000-000000000a0a', 'grid-modernization', 'Grid Modernization', 'Smart grid and energy distribution', 'd0000000-0000-4000-8000-00000000000a'),
  -- Logistics sub-topics
  ('d0000000-0000-4000-8000-000000000b01', 'mobility', 'Mobility & Transportation', 'Urban mobility and transport systems', 'd0000000-0000-4000-8000-00000000000b'),
  ('d0000000-0000-4000-8000-000000000b02', 'ev-transport', 'Electric Vehicles', 'EVs, charging infrastructure, and battery transport', 'd0000000-0000-4000-8000-00000000000b'),
  ('d0000000-0000-4000-8000-000000000b03', 'autonomous-vehicles', 'Autonomous Vehicles', 'Self-driving cars, trucks, and drones', 'd0000000-0000-4000-8000-00000000000b'),
  ('d0000000-0000-4000-8000-000000000b04', 'logistics-tech', 'Logistics Technology', 'Supply chain tech and optimization', 'd0000000-0000-4000-8000-00000000000b'),
  ('d0000000-0000-4000-8000-000000000b05', 'shipping-maritime', 'Shipping & Maritime', 'Ocean freight and port operations', 'd0000000-0000-4000-8000-00000000000b'),
  ('d0000000-0000-4000-8000-000000000b06', 'aviation', 'Aviation & Aerospace', 'Air travel, airlines, and aerospace', 'd0000000-0000-4000-8000-00000000000b'),
  ('d0000000-0000-4000-8000-000000000b07', 'last-mile-delivery', 'Last-Mile Delivery', 'Final leg of delivery and courier services', 'd0000000-0000-4000-8000-00000000000b'),
  ('d0000000-0000-4000-8000-000000000b08', 'ridesharing', 'Ride-sharing & Micro-mobility', 'Ride-hailing, scooters, and bike-sharing', 'd0000000-0000-4000-8000-00000000000b'),
  -- Education sub-topics
  ('d0000000-0000-4000-8000-000000000c01', 'edtech', 'EdTech', 'Technology in education and learning', 'd0000000-0000-4000-8000-00000000000c'),
  ('d0000000-0000-4000-8000-000000000c02', 'online-learning', 'Online Learning', 'MOOCs, remote learning, and digital classrooms', 'd0000000-0000-4000-8000-00000000000c'),
  ('d0000000-0000-4000-8000-000000000c03', 'workforce-training', 'Workforce Training', 'Skills development and vocational training', 'd0000000-0000-4000-8000-00000000000c'),
  ('d0000000-0000-4000-8000-000000000c04', 'remote-work', 'Remote Work', 'Distributed teams and hybrid work models', 'd0000000-0000-4000-8000-00000000000c'),
  ('d0000000-0000-4000-8000-000000000c05', 'talent-acquisition', 'Talent & Hiring', 'Recruitment, HR tech, and talent markets', 'd0000000-0000-4000-8000-00000000000c'),
  ('d0000000-0000-4000-8000-000000000c06', 'stem-education', 'STEM Education', 'Science, tech, engineering, and math education', 'd0000000-0000-4000-8000-00000000000c'),
  ('d0000000-0000-4000-8000-000000000c07', 'gig-economy', 'Gig Economy', 'Freelance, contract work, and platforms', 'd0000000-0000-4000-8000-00000000000c'),
  -- Retail & Consumer
  ('d0000000-0000-4000-8000-000000000d01', 'ecommerce', 'E-Commerce', 'Online retail and digital storefronts', 'd0000000-0000-4000-8000-000000000007'),
  ('d0000000-0000-4000-8000-000000000d02', 'direct-to-consumer', 'Direct-to-Consumer', 'DTC brands and subscription commerce', 'd0000000-0000-4000-8000-000000000007'),
  ('d0000000-0000-4000-8000-000000000d03', 'social-commerce', 'Social Commerce', 'Selling through social media platforms', 'd0000000-0000-4000-8000-000000000007'),
  ('d0000000-0000-4000-8000-000000000d04', 'consumer-electronics', 'Consumer Electronics', 'Smartphones, wearables, and gadgets', 'd0000000-0000-4000-8000-000000000003')
ON CONFLICT (code) DO NOTHING;

-- ============================================================
-- SEED: Countries (ISO-3166 — all UN members + observers)
-- ============================================================

INSERT INTO countries (code, alpha3, numeric_code, name, official_name, region, subregion) VALUES
  ('AF', 'AFG', '004', 'Afghanistan', 'Islamic Republic of Afghanistan', 'Asia', 'Southern Asia'),
  ('AL', 'ALB', '008', 'Albania', 'Republic of Albania', 'Europe', 'Southern Europe'),
  ('DZ', 'DZA', '012', 'Algeria', 'People''s Democratic Republic of Algeria', 'Africa', 'Northern Africa'),
  ('AD', 'AND', '020', 'Andorra', 'Principality of Andorra', 'Europe', 'Southern Europe'),
  ('AO', 'AGO', '024', 'Angola', 'Republic of Angola', 'Africa', 'Middle Africa'),
  ('AG', 'ATG', '028', 'Antigua and Barbuda', 'Antigua and Barbuda', 'Americas', 'Caribbean'),
  ('AR', 'ARG', '032', 'Argentina', 'Argentine Republic', 'Americas', 'South America'),
  ('AM', 'ARM', '051', 'Armenia', 'Republic of Armenia', 'Asia', 'Western Asia'),
  ('AU', 'AUS', '036', 'Australia', 'Commonwealth of Australia', 'Oceania', 'Australia and New Zealand'),
  ('AT', 'AUT', '040', 'Austria', 'Republic of Austria', 'Europe', 'Western Europe'),
  ('AZ', 'AZE', '031', 'Azerbaijan', 'Republic of Azerbaijan', 'Asia', 'Western Asia'),
  ('BS', 'BHS', '044', 'Bahamas', 'Commonwealth of the Bahamas', 'Americas', 'Caribbean'),
  ('BH', 'BHR', '048', 'Bahrain', 'Kingdom of Bahrain', 'Asia', 'Western Asia'),
  ('BD', 'BGD', '050', 'Bangladesh', 'People''s Republic of Bangladesh', 'Asia', 'Southern Asia'),
  ('BB', 'BRB', '052', 'Barbados', 'Barbados', 'Americas', 'Caribbean'),
  ('BY', 'BLR', '112', 'Belarus', 'Republic of Belarus', 'Europe', 'Eastern Europe'),
  ('BE', 'BEL', '056', 'Belgium', 'Kingdom of Belgium', 'Europe', 'Western Europe'),
  ('BZ', 'BLZ', '084', 'Belize', 'Belize', 'Americas', 'Central America'),
  ('BJ', 'BEN', '204', 'Benin', 'Republic of Benin', 'Africa', 'Western Africa'),
  ('BT', 'BTN', '064', 'Bhutan', 'Kingdom of Bhutan', 'Asia', 'Southern Asia'),
  ('BO', 'BOL', '068', 'Bolivia', 'Plurinational State of Bolivia', 'Americas', 'South America'),
  ('BA', 'BIH', '070', 'Bosnia and Herzegovina', 'Bosnia and Herzegovina', 'Europe', 'Southern Europe'),
  ('BW', 'BWA', '072', 'Botswana', 'Republic of Botswana', 'Africa', 'Southern Africa'),
  ('BR', 'BRA', '076', 'Brazil', 'Federative Republic of Brazil', 'Americas', 'South America'),
  ('BN', 'BRN', '096', 'Brunei', 'Brunei Darussalam', 'Asia', 'South-Eastern Asia'),
  ('BG', 'BGR', '100', 'Bulgaria', 'Republic of Bulgaria', 'Europe', 'Eastern Europe'),
  ('BF', 'BFA', '854', 'Burkina Faso', 'Burkina Faso', 'Africa', 'Western Africa'),
  ('BI', 'BDI', '108', 'Burundi', 'Republic of Burundi', 'Africa', 'Eastern Africa'),
  ('CV', 'CPV', '132', 'Cabo Verde', 'Republic of Cabo Verde', 'Africa', 'Western Africa'),
  ('KH', 'KHM', '116', 'Cambodia', 'Kingdom of Cambodia', 'Asia', 'South-Eastern Asia'),
  ('CM', 'CMR', '120', 'Cameroon', 'Republic of Cameroon', 'Africa', 'Middle Africa'),
  ('CA', 'CAN', '124', 'Canada', 'Canada', 'Americas', 'Northern America'),
  ('CF', 'CAF', '140', 'Central African Republic', 'Central African Republic', 'Africa', 'Middle Africa'),
  ('TD', 'TCD', '148', 'Chad', 'Republic of Chad', 'Africa', 'Middle Africa'),
  ('CL', 'CHL', '152', 'Chile', 'Republic of Chile', 'Americas', 'South America'),
  ('CN', 'CHN', '156', 'China', 'People''s Republic of China', 'Asia', 'Eastern Asia'),
  ('CO', 'COL', '170', 'Colombia', 'Republic of Colombia', 'Americas', 'South America'),
  ('KM', 'COM', '174', 'Comoros', 'Union of the Comoros', 'Africa', 'Eastern Africa'),
  ('CG', 'COG', '178', 'Congo', 'Republic of the Congo', 'Africa', 'Middle Africa'),
  ('CD', 'COD', '180', 'DRC', 'Democratic Republic of the Congo', 'Africa', 'Middle Africa'),
  ('CR', 'CRI', '188', 'Costa Rica', 'Republic of Costa Rica', 'Americas', 'Central America'),
  ('CI', 'CIV', '384', 'Cote d''Ivoire', 'Republic of Cote d''Ivoire', 'Africa', 'Western Africa'),
  ('HR', 'HRV', '191', 'Croatia', 'Republic of Croatia', 'Europe', 'Southern Europe'),
  ('CU', 'CUB', '192', 'Cuba', 'Republic of Cuba', 'Americas', 'Caribbean'),
  ('CY', 'CYP', '196', 'Cyprus', 'Republic of Cyprus', 'Europe', 'Southern Europe'),
  ('CZ', 'CZE', '203', 'Czech Republic', 'Czech Republic', 'Europe', 'Eastern Europe'),
  ('DK', 'DNK', '208', 'Denmark', 'Kingdom of Denmark', 'Europe', 'Northern Europe'),
  ('DJ', 'DJI', '262', 'Djibouti', 'Republic of Djibouti', 'Africa', 'Eastern Africa'),
  ('DM', 'DMA', '212', 'Dominica', 'Commonwealth of Dominica', 'Americas', 'Caribbean'),
  ('DO', 'DOM', '214', 'Dominican Republic', 'Dominican Republic', 'Americas', 'Caribbean'),
  ('EC', 'ECU', '218', 'Ecuador', 'Republic of Ecuador', 'Americas', 'South America'),
  ('EG', 'EGY', '818', 'Egypt', 'Arab Republic of Egypt', 'Africa', 'Northern Africa'),
  ('SV', 'SLV', '222', 'El Salvador', 'Republic of El Salvador', 'Americas', 'Central America'),
  ('GQ', 'GNQ', '226', 'Equatorial Guinea', 'Republic of Equatorial Guinea', 'Africa', 'Middle Africa'),
  ('ER', 'ERI', '232', 'Eritrea', 'State of Eritrea', 'Africa', 'Eastern Africa'),
  ('EE', 'EST', '233', 'Estonia', 'Republic of Estonia', 'Europe', 'Northern Europe'),
  ('SZ', 'SWZ', '748', 'Eswatini', 'Kingdom of Eswatini', 'Africa', 'Southern Africa'),
  ('ET', 'ETH', '231', 'Ethiopia', 'Federal Democratic Republic of Ethiopia', 'Africa', 'Eastern Africa'),
  ('FJ', 'FJI', '242', 'Fiji', 'Republic of Fiji', 'Oceania', 'Melanesia'),
  ('FI', 'FIN', '246', 'Finland', 'Republic of Finland', 'Europe', 'Northern Europe'),
  ('FR', 'FRA', '250', 'France', 'French Republic', 'Europe', 'Western Europe'),
  ('GA', 'GAB', '266', 'Gabon', 'Gabonese Republic', 'Africa', 'Middle Africa'),
  ('GM', 'GMB', '270', 'Gambia', 'Republic of the Gambia', 'Africa', 'Western Africa'),
  ('GE', 'GEO', '268', 'Georgia', 'Georgia', 'Asia', 'Western Asia'),
  ('DE', 'DEU', '276', 'Germany', 'Federal Republic of Germany', 'Europe', 'Western Europe'),
  ('GH', 'GHA', '288', 'Ghana', 'Republic of Ghana', 'Africa', 'Western Africa'),
  ('GR', 'GRC', '300', 'Greece', 'Hellenic Republic', 'Europe', 'Southern Europe'),
  ('GD', 'GRD', '308', 'Grenada', 'Grenada', 'Americas', 'Caribbean'),
  ('GT', 'GTM', '320', 'Guatemala', 'Republic of Guatemala', 'Americas', 'Central America'),
  ('GN', 'GIN', '324', 'Guinea', 'Republic of Guinea', 'Africa', 'Western Africa'),
  ('GW', 'GNB', '624', 'Guinea-Bissau', 'Republic of Guinea-Bissau', 'Africa', 'Western Africa'),
  ('GY', 'GUY', '328', 'Guyana', 'Co-operative Republic of Guyana', 'Americas', 'South America'),
  ('HT', 'HTI', '332', 'Haiti', 'Republic of Haiti', 'Americas', 'Caribbean'),
  ('HN', 'HND', '340', 'Honduras', 'Republic of Honduras', 'Americas', 'Central America'),
  ('HU', 'HUN', '348', 'Hungary', 'Hungary', 'Europe', 'Eastern Europe'),
  ('IS', 'ISL', '352', 'Iceland', 'Iceland', 'Europe', 'Northern Europe'),
  ('IN', 'IND', '356', 'India', 'Republic of India', 'Asia', 'Southern Asia'),
  ('ID', 'IDN', '360', 'Indonesia', 'Republic of Indonesia', 'Asia', 'South-Eastern Asia'),
  ('IR', 'IRN', '364', 'Iran', 'Islamic Republic of Iran', 'Asia', 'Southern Asia'),
  ('IQ', 'IRQ', '368', 'Iraq', 'Republic of Iraq', 'Asia', 'Western Asia'),
  ('IE', 'IRL', '372', 'Ireland', 'Ireland', 'Europe', 'Northern Europe'),
  ('IL', 'ISR', '376', 'Israel', 'State of Israel', 'Asia', 'Western Asia'),
  ('IT', 'ITA', '380', 'Italy', 'Italian Republic', 'Europe', 'Southern Europe'),
  ('JM', 'JAM', '388', 'Jamaica', 'Jamaica', 'Americas', 'Caribbean'),
  ('JP', 'JPN', '392', 'Japan', 'Japan', 'Asia', 'Eastern Asia'),
  ('JO', 'JOR', '400', 'Jordan', 'Hashemite Kingdom of Jordan', 'Asia', 'Western Asia'),
  ('KZ', 'KAZ', '398', 'Kazakhstan', 'Republic of Kazakhstan', 'Asia', 'Central Asia'),
  ('KE', 'KEN', '404', 'Kenya', 'Republic of Kenya', 'Africa', 'Eastern Africa'),
  ('KI', 'KIR', '296', 'Kiribati', 'Republic of Kiribati', 'Oceania', 'Micronesia'),
  ('KP', 'PRK', '408', 'North Korea', 'Democratic People''s Republic of Korea', 'Asia', 'Eastern Asia'),
  ('KR', 'KOR', '410', 'South Korea', 'Republic of Korea', 'Asia', 'Eastern Asia'),
  ('KW', 'KWT', '414', 'Kuwait', 'State of Kuwait', 'Asia', 'Western Asia'),
  ('KG', 'KGZ', '417', 'Kyrgyzstan', 'Kyrgyz Republic', 'Asia', 'Central Asia'),
  ('LA', 'LAO', '418', 'Laos', 'Lao People''s Democratic Republic', 'Asia', 'South-Eastern Asia'),
  ('LV', 'LVA', '428', 'Latvia', 'Republic of Latvia', 'Europe', 'Northern Europe'),
  ('LB', 'LBN', '422', 'Lebanon', 'Lebanese Republic', 'Asia', 'Western Asia'),
  ('LS', 'LSO', '426', 'Lesotho', 'Kingdom of Lesotho', 'Africa', 'Southern Africa'),
  ('LR', 'LBR', '430', 'Liberia', 'Republic of Liberia', 'Africa', 'Western Africa'),
  ('LY', 'LBY', '434', 'Libya', 'State of Libya', 'Africa', 'Northern Africa'),
  ('LI', 'LIE', '438', 'Liechtenstein', 'Principality of Liechtenstein', 'Europe', 'Western Europe'),
  ('LT', 'LTU', '440', 'Lithuania', 'Republic of Lithuania', 'Europe', 'Northern Europe'),
  ('LU', 'LUX', '442', 'Luxembourg', 'Grand Duchy of Luxembourg', 'Europe', 'Western Europe'),
  ('MG', 'MDG', '450', 'Madagascar', 'Republic of Madagascar', 'Africa', 'Eastern Africa'),
  ('MW', 'MWI', '454', 'Malawi', 'Republic of Malawi', 'Africa', 'Eastern Africa'),
  ('MY', 'MYS', '458', 'Malaysia', 'Malaysia', 'Asia', 'South-Eastern Asia'),
  ('MV', 'MDV', '462', 'Maldives', 'Republic of Maldives', 'Asia', 'Southern Asia'),
  ('ML', 'MLI', '466', 'Mali', 'Republic of Mali', 'Africa', 'Western Africa'),
  ('MT', 'MLT', '470', 'Malta', 'Republic of Malta', 'Europe', 'Southern Europe'),
  ('MH', 'MHL', '584', 'Marshall Islands', 'Republic of the Marshall Islands', 'Oceania', 'Micronesia'),
  ('MR', 'MRT', '478', 'Mauritania', 'Islamic Republic of Mauritania', 'Africa', 'Western Africa'),
  ('MU', 'MUS', '480', 'Mauritius', 'Republic of Mauritius', 'Africa', 'Eastern Africa'),
  ('MX', 'MEX', '484', 'Mexico', 'United Mexican States', 'Americas', 'Central America'),
  ('FM', 'FSM', '583', 'Micronesia', 'Federated States of Micronesia', 'Oceania', 'Micronesia'),
  ('MD', 'MDA', '498', 'Moldova', 'Republic of Moldova', 'Europe', 'Eastern Europe'),
  ('MC', 'MCO', '492', 'Monaco', 'Principality of Monaco', 'Europe', 'Western Europe'),
  ('MN', 'MNG', '496', 'Mongolia', 'Mongolia', 'Asia', 'Eastern Asia'),
  ('ME', 'MNE', '499', 'Montenegro', 'Montenegro', 'Europe', 'Southern Europe'),
  ('MA', 'MAR', '504', 'Morocco', 'Kingdom of Morocco', 'Africa', 'Northern Africa'),
  ('MZ', 'MOZ', '508', 'Mozambique', 'Republic of Mozambique', 'Africa', 'Eastern Africa'),
  ('MM', 'MMR', '104', 'Myanmar', 'Republic of the Union of Myanmar', 'Asia', 'South-Eastern Asia'),
  ('NA', 'NAM', '516', 'Namibia', 'Republic of Namibia', 'Africa', 'Southern Africa'),
  ('NR', 'NRU', '520', 'Nauru', 'Republic of Nauru', 'Oceania', 'Micronesia'),
  ('NP', 'NPL', '524', 'Nepal', 'Federal Democratic Republic of Nepal', 'Asia', 'Southern Asia'),
  ('NL', 'NLD', '528', 'Netherlands', 'Kingdom of the Netherlands', 'Europe', 'Western Europe'),
  ('NZ', 'NZL', '554', 'New Zealand', 'New Zealand', 'Oceania', 'Australia and New Zealand'),
  ('NI', 'NIC', '558', 'Nicaragua', 'Republic of Nicaragua', 'Americas', 'Central America'),
  ('NE', 'NER', '562', 'Niger', 'Republic of Niger', 'Africa', 'Western Africa'),
  ('NG', 'NGA', '566', 'Nigeria', 'Federal Republic of Nigeria', 'Africa', 'Western Africa'),
  ('MK', 'MKD', '807', 'North Macedonia', 'Republic of North Macedonia', 'Europe', 'Southern Europe'),
  ('NO', 'NOR', '578', 'Norway', 'Kingdom of Norway', 'Europe', 'Northern Europe'),
  ('OM', 'OMN', '512', 'Oman', 'Sultanate of Oman', 'Asia', 'Western Asia'),
  ('PK', 'PAK', '586', 'Pakistan', 'Islamic Republic of Pakistan', 'Asia', 'Southern Asia'),
  ('PW', 'PLW', '585', 'Palau', 'Republic of Palau', 'Oceania', 'Micronesia'),
  ('PS', 'PSE', '275', 'Palestine', 'State of Palestine', 'Asia', 'Western Asia'),
  ('PA', 'PAN', '591', 'Panama', 'Republic of Panama', 'Americas', 'Central America'),
  ('PG', 'PNG', '598', 'Papua New Guinea', 'Independent State of Papua New Guinea', 'Oceania', 'Melanesia'),
  ('PY', 'PRY', '600', 'Paraguay', 'Republic of Paraguay', 'Americas', 'South America'),
  ('PE', 'PER', '604', 'Peru', 'Republic of Peru', 'Americas', 'South America'),
  ('PH', 'PHL', '608', 'Philippines', 'Republic of the Philippines', 'Asia', 'South-Eastern Asia'),
  ('PL', 'POL', '616', 'Poland', 'Republic of Poland', 'Europe', 'Eastern Europe'),
  ('PT', 'PRT', '620', 'Portugal', 'Portuguese Republic', 'Europe', 'Southern Europe'),
  ('QA', 'QAT', '634', 'Qatar', 'State of Qatar', 'Asia', 'Western Asia'),
  ('RO', 'ROU', '642', 'Romania', 'Romania', 'Europe', 'Eastern Europe'),
  ('RU', 'RUS', '643', 'Russia', 'Russian Federation', 'Europe', 'Eastern Europe'),
  ('RW', 'RWA', '646', 'Rwanda', 'Republic of Rwanda', 'Africa', 'Eastern Africa'),
  ('KN', 'KNA', '659', 'Saint Kitts and Nevis', 'Saint Kitts and Nevis', 'Americas', 'Caribbean'),
  ('LC', 'LCA', '662', 'Saint Lucia', 'Saint Lucia', 'Americas', 'Caribbean'),
  ('VC', 'VCT', '670', 'Saint Vincent and the Grenadines', 'Saint Vincent and the Grenadines', 'Americas', 'Caribbean'),
  ('WS', 'WSM', '882', 'Samoa', 'Independent State of Samoa', 'Oceania', 'Polynesia'),
  ('SM', 'SMR', '674', 'San Marino', 'Republic of San Marino', 'Europe', 'Southern Europe'),
  ('ST', 'STP', '678', 'Sao Tome and Principe', 'Democratic Republic of Sao Tome and Principe', 'Africa', 'Middle Africa'),
  ('SA', 'SAU', '682', 'Saudi Arabia', 'Kingdom of Saudi Arabia', 'Asia', 'Western Asia'),
  ('SN', 'SEN', '686', 'Senegal', 'Republic of Senegal', 'Africa', 'Western Africa'),
  ('RS', 'SRB', '688', 'Serbia', 'Republic of Serbia', 'Europe', 'Southern Europe'),
  ('SC', 'SYC', '690', 'Seychelles', 'Republic of Seychelles', 'Africa', 'Eastern Africa'),
  ('SL', 'SLE', '694', 'Sierra Leone', 'Republic of Sierra Leone', 'Africa', 'Western Africa'),
  ('SG', 'SGP', '702', 'Singapore', 'Republic of Singapore', 'Asia', 'South-Eastern Asia'),
  ('SK', 'SVK', '703', 'Slovakia', 'Slovak Republic', 'Europe', 'Eastern Europe'),
  ('SI', 'SVN', '705', 'Slovenia', 'Republic of Slovenia', 'Europe', 'Southern Europe'),
  ('SB', 'SLB', '090', 'Solomon Islands', 'Solomon Islands', 'Oceania', 'Melanesia'),
  ('SO', 'SOM', '706', 'Somalia', 'Federal Republic of Somalia', 'Africa', 'Eastern Africa'),
  ('ZA', 'ZAF', '710', 'South Africa', 'Republic of South Africa', 'Africa', 'Southern Africa'),
  ('SS', 'SSD', '728', 'South Sudan', 'Republic of South Sudan', 'Africa', 'Eastern Africa'),
  ('ES', 'ESP', '724', 'Spain', 'Kingdom of Spain', 'Europe', 'Southern Europe'),
  ('LK', 'LKA', '144', 'Sri Lanka', 'Democratic Socialist Republic of Sri Lanka', 'Asia', 'Southern Asia'),
  ('SD', 'SDN', '729', 'Sudan', 'Republic of the Sudan', 'Africa', 'Northern Africa'),
  ('SR', 'SUR', '740', 'Suriname', 'Republic of Suriname', 'Americas', 'South America'),
  ('SE', 'SWE', '752', 'Sweden', 'Kingdom of Sweden', 'Europe', 'Northern Europe'),
  ('CH', 'CHE', '756', 'Switzerland', 'Swiss Confederation', 'Europe', 'Western Europe'),
  ('SY', 'SYR', '760', 'Syria', 'Syrian Arab Republic', 'Asia', 'Western Asia'),
  ('TW', 'TWN', '158', 'Taiwan', 'Taiwan', 'Asia', 'Eastern Asia'),
  ('TJ', 'TJK', '762', 'Tajikistan', 'Republic of Tajikistan', 'Asia', 'Central Asia'),
  ('TZ', 'TZA', '834', 'Tanzania', 'United Republic of Tanzania', 'Africa', 'Eastern Africa'),
  ('TH', 'THA', '764', 'Thailand', 'Kingdom of Thailand', 'Asia', 'South-Eastern Asia'),
  ('TL', 'TLS', '626', 'Timor-Leste', 'Democratic Republic of Timor-Leste', 'Asia', 'South-Eastern Asia'),
  ('TG', 'TGO', '768', 'Togo', 'Togolese Republic', 'Africa', 'Western Africa'),
  ('TO', 'TON', '776', 'Tonga', 'Kingdom of Tonga', 'Oceania', 'Polynesia'),
  ('TT', 'TTO', '780', 'Trinidad and Tobago', 'Republic of Trinidad and Tobago', 'Americas', 'Caribbean'),
  ('TN', 'TUN', '788', 'Tunisia', 'Republic of Tunisia', 'Africa', 'Northern Africa'),
  ('TR', 'TUR', '792', 'Turkey', 'Republic of Turkey', 'Asia', 'Western Asia'),
  ('TM', 'TKM', '795', 'Turkmenistan', 'Turkmenistan', 'Asia', 'Central Asia'),
  ('TV', 'TUV', '798', 'Tuvalu', 'Tuvalu', 'Oceania', 'Polynesia'),
  ('UG', 'UGA', '800', 'Uganda', 'Republic of Uganda', 'Africa', 'Eastern Africa'),
  ('UA', 'UKR', '804', 'Ukraine', 'Ukraine', 'Europe', 'Eastern Europe'),
  ('AE', 'ARE', '784', 'United Arab Emirates', 'United Arab Emirates', 'Asia', 'Western Asia'),
  ('GB', 'GBR', '826', 'United Kingdom', 'United Kingdom of Great Britain and Northern Ireland', 'Europe', 'Northern Europe'),
  ('US', 'USA', '840', 'United States', 'United States of America', 'Americas', 'Northern America'),
  ('UY', 'URY', '858', 'Uruguay', 'Oriental Republic of Uruguay', 'Americas', 'South America'),
  ('UZ', 'UZB', '860', 'Uzbekistan', 'Republic of Uzbekistan', 'Asia', 'Central Asia'),
  ('VU', 'VUT', '548', 'Vanuatu', 'Republic of Vanuatu', 'Oceania', 'Melanesia'),
  ('VA', 'VAT', '336', 'Vatican City', 'Vatican City State', 'Europe', 'Southern Europe'),
  ('VE', 'VEN', '862', 'Venezuela', 'Bolivarian Republic of Venezuela', 'Americas', 'South America'),
  ('VN', 'VNM', '704', 'Vietnam', 'Socialist Republic of Vietnam', 'Asia', 'South-Eastern Asia'),
  ('YE', 'YEM', '887', 'Yemen', 'Republic of Yemen', 'Asia', 'Western Asia'),
  ('ZM', 'ZMB', '894', 'Zambia', 'Republic of Zambia', 'Africa', 'Eastern Africa'),
  ('ZW', 'ZWE', '716', 'Zimbabwe', 'Republic of Zimbabwe', 'Africa', 'Eastern Africa')
ON CONFLICT (code) DO NOTHING;

-- ============================================================
-- SEED: Organizations (~60 major global organizations)
-- ============================================================

WITH c AS (SELECT code, id FROM countries) INSERT INTO organizations (code, name, type, description, country_code)
SELECT * FROM (VALUES
  -- Tech Giants
  ('microsoft',       'Microsoft',            'company',      'Software, cloud, and AI platforms', 'US'),
  ('google',          'Google / Alphabet',    'company',      'Search, cloud, AI, and advertising', 'US'),
  ('amazon',          'Amazon',               'company',      'E-commerce, cloud (AWS), and AI', 'US'),
  ('meta',            'Meta',                 'company',      'Social media, VR, and AI research', 'US'),
  ('apple',           'Apple',                'company',      'Consumer hardware, services, and AI', 'US'),
  ('nvidia',          'NVIDIA',               'company',      'GPU, AI chips, and accelerated computing', 'US'),
  ('tsmc',            'TSMC',                 'company',      'World''s largest semiconductor foundry', 'TW'),
  ('samsung',         'Samsung',              'company',      'Electronics, semiconductors, appliances', 'KR'),
  ('openai',          'OpenAI',               'company',      'AI research and LLM products', 'US'),
  ('anthropic',       'Anthropic',            'company',      'AI safety research and Claude models', 'US'),
  ('tesla',           'Tesla',                'company',      'EV, energy, and autonomous driving', 'US'),
  ('spacex',          'SpaceX',               'company',      'Space launch and satellite communications', 'US'),
  -- Enterprise & Cloud
  ('oracle',          'Oracle',               'company',      'Enterprise database and cloud solutions', 'US'),
  ('salesforce',      'Salesforce',           'company',      'CRM and enterprise SaaS', 'US'),
  ('ibm',             'IBM',                  'company',      'Enterprise IT, AI, and consulting', 'US'),
  ('sap',             'SAP',                  'company',      'Enterprise resource planning software', 'DE'),
  ('adobe',           'Adobe',                'company',      'Creative software and digital marketing', 'US'),
  ('intel',           'Intel',                'company',      'Semiconductor and processor manufacturing', 'US'),
  ('amd',             'AMD',                  'company',      'CPU and GPU manufacturing', 'US'),
  ('arm',             'Arm',                  'company',      'Chip architecture and licensing', 'GB'),
  -- Financial
  ('jpmorgan',        'JPMorgan Chase',       'company',      'Largest US bank by assets', 'US'),
  ('goldman-sachs',   'Goldman Sachs',        'company',      'Global investment bank', 'US'),
  ('blackrock',       'BlackRock',            'company',      'World''s largest asset manager', 'US'),
  ('stripe',          'Stripe',               'company',      'Global payment processing platform', 'US'),
  ('visa',            'Visa',                 'company',      'Digital payment network', 'US'),
  ('mastercard',      'Mastercard',           'company',      'Payment technology and services', 'US'),
  -- Indonesian
  ('goto',            'GoTo Group',           'company',      'Indonesia digital ecosystem (Gojek+Tokopedia)', 'ID'),
  ('traveloka',       'Traveloka',            'company',      'SE Asia travel and lifestyle platform', 'ID'),
  ('bukalapak',       'Bukalapak',            'company',      'Indonesia e-commerce platform', 'ID'),
  ('bank-mandiri',    'Bank Mandiri',         'company',      'Largest Indonesian state-owned bank', 'ID'),
  ('bri',             'BRI',                  'company',      'Bank Rakyat Indonesia', 'ID'),
  ('telkom-id',       'Telkom Indonesia',     'company',      'Indonesia telecom and digital services', 'ID'),
  -- Social & Media
  ('tiktok',          'TikTok / ByteDance',   'company',      'Short video platform and AI', 'CN'),
  ('twitter',         'X / Twitter',          'company',      'Social media platform', 'US'),
  ('netflix',         'Netflix',              'company',      'Streaming entertainment', 'US'),
  ('spotify',         'Spotify',              'company',      'Music streaming and audio', 'SE'),
  -- Multilateral / Government
  ('united-nations',  'United Nations',       'multilateral', 'International peace and cooperation', 'US'),
  ('world-bank',      'World Bank',           'multilateral', 'International financial institution for development', 'US'),
  ('imf',             'IMF',                  'multilateral', 'International Monetary Fund', 'US'),
  ('who',             'World Health Org',     'multilateral', 'Global health governance and response', 'CH'),
  ('wto',             'WTO',                  'multilateral', 'World Trade Organization', 'CH'),
  ('oecd',            'OECD',                 'multilateral', 'Economic cooperation and development', 'FR'),
  ('world-economic-forum', 'World Economic Forum', 'ngo',    'Global economic policy forum', 'CH'),
  ('eu',              'European Union',       'government',   'European political and economic union', 'BE'),
  ('asean',           'ASEAN',                'multilateral', 'Association of Southeast Asian Nations', 'ID'),
  ('world-bank-group','World Bank Group',     'multilateral', 'Development finance and poverty reduction', 'US'),
  -- NGOs
  ('greenpeace',      'Greenpeace',           'ngo',          'Environmental activism and advocacy', 'NL'),
  ('world-wildlife',  'WWF',                  'ngo',          'Wildlife conservation and sustainability', 'CH'),
  ('red-cross',       'Red Cross',            'ngo',          'Humanitarian aid and disaster response', 'CH'),
  ('doctors-wo-borders','Doctors Without Borders', 'ngo',     'Emergency medical humanitarian aid', 'CH'),
  -- Academic / Research
  ('mit',             'MIT',                  'academic',     'Massachusetts Institute of Technology', 'US'),
  ('stanford',        'Stanford University',  'academic',     'Stanford University', 'US'),
  ('oxford',          'Oxford University',    'academic',     'University of Oxford', 'GB'),
  ('harvard',         'Harvard University',   'academic',     'Harvard University', 'US'),
  ('cambridge',       'Cambridge University', 'academic',     'University of Cambridge', 'GB')
) AS v(code, name, type, description, country_code)
WHERE NOT EXISTS (SELECT 1 FROM organizations o WHERE o.code = v.code);

-- ============================================================
-- Down
-- ============================================================
-- +goose Down

DROP TABLE IF EXISTS article_organizations CASCADE;
DROP TABLE IF EXISTS article_technologies CASCADE;
DROP TABLE IF EXISTS article_countries CASCADE;
DROP TABLE IF EXISTS article_topics CASCADE;
DROP TABLE IF EXISTS article_industries CASCADE;

DROP TRIGGER IF EXISTS trg_organizations_updated_at ON organizations;
DROP TRIGGER IF EXISTS trg_topics_updated_at ON topics;

DROP TABLE IF EXISTS organizations CASCADE;
DROP TABLE IF EXISTS countries CASCADE;
DROP TABLE IF EXISTS topics CASCADE;