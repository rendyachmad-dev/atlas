package signal

import "time"

// SignalTypeCode enumerates all known signal type codes.
type SignalTypeCode string

const (
	// Topic-based signals
	SignalMLAdvancement      SignalTypeCode = "machine_learning"
	SignalFintechInnovation  SignalTypeCode = "fintech_innovation"
	SignalCybersecurityEvent SignalTypeCode = "cybersecurity_event"
	SignalRenewableEnergy    SignalTypeCode = "renewable_energy"
	SignalBiotechBreakthrough SignalTypeCode = "biotech_breakthrough"
	SignalCloudAdoption      SignalTypeCode = "cloud_adoption"
	SignalAIRegulation       SignalTypeCode = "ai_regulation"
	SignalSupplyChainEvent   SignalTypeCode = "supply_chain_event"
	SignalGenAIInnovation    SignalTypeCode = "gen_ai_innovation"
	SignalBlockchainAdoption SignalTypeCode = "blockchain_adoption"
	SignalHealthtechInnovation SignalTypeCode = "healthtech_innovation"
	SignalEdtechGrowth       SignalTypeCode = "edtech_growth"

	// Industry-based signals
	SignalTechGrowth         SignalTypeCode = "tech_growth"
	SignalFinancialActivity  SignalTypeCode = "financial_activity"
	SignalHealthcareDev      SignalTypeCode = "healthcare_development"
	SignalEnergyDev          SignalTypeCode = "energy_development"
	SignalManufacturingActivity SignalTypeCode = "manufacturing_activity"
	SignalAgricultureDev     SignalTypeCode = "agriculture_development"

	// Country-based signals
	SignalGeopoliticalEvent  SignalTypeCode = "geopolitical_event"
	SignalMarketEntry        SignalTypeCode = "market_entry"

	// Technology-based signals
	SignalTechAdoption       SignalTypeCode = "technology_adoption"
	SignalInfrastructureDev  SignalTypeCode = "infrastructure_development"

	// Organization-based signals
	SignalPartnership        SignalTypeCode = "partnership"
	SignalInvestment         SignalTypeCode = "investment"
	SignalAcquisition        SignalTypeCode = "acquisition"
	SignalRegulatoryAction   SignalTypeCode = "regulatory_action"
	SignalResearchPublication SignalTypeCode = "research_publication"
	SignalPatentFiling       SignalTypeCode = "patent_filing"
)

// SignalType represents a row from signal_types table.
type SignalType struct {
	ID          string         `json:"id"`
	Code        SignalTypeCode `json:"code"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Category    string         `json:"category"`
	IsActive    bool           `json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
}

// Signal represents one detected business signal.
type Signal struct {
	ID             string         `json:"id"`
	ArticleID      string         `json:"article_id"`
	SignalTypeID   string         `json:"signal_type_id"`
	SignalTypeCode SignalTypeCode `json:"signal_type_code"`
	EntityCode     string         `json:"entity_code"`
	EntityCategory string         `json:"entity_category"`
	Confidence     float64        `json:"confidence"`
	DetectedAt     time.Time      `json:"detected_at"`
	CreatedAt      time.Time      `json:"created_at"`
}

// EntityCategory constants used in signals.
const (
	CatTopics        = "topics"
	CatIndustries    = "industries"
	CatCountries     = "countries"
	CatTechnologies  = "technologies"
	CatOrganizations = "organizations"
)

// SignalTypeMapping maps taxonomy codes to signal type codes.
// Each knowledge entity can produce one or more signals.
type SignalTypeMapping struct {
	EntityCategory string
	EntityCode     string
	SignalTypeCode SignalTypeCode
	MinConfidence  float64
}

// DefaultMappings returns the standard taxonomy-to-signal mappings.
func DefaultMappings() []SignalTypeMapping {
	return []SignalTypeMapping{
		// Topic → signal mappings
		{CatTopics, "machine-learning", SignalMLAdvancement, 0.5},
		{CatTopics, "fintech", SignalFintechInnovation, 0.5},
		{CatTopics, "cybersecurity", SignalCybersecurityEvent, 0.5},
		{CatTopics, "renewable-energy", SignalRenewableEnergy, 0.5},
		{CatTopics, "biotechnology", SignalBiotechBreakthrough, 0.5},
		{CatTopics, "cloud-computing", SignalCloudAdoption, 0.5},
		{CatTopics, "ai-regulation", SignalAIRegulation, 0.5},
		{CatTopics, "supply-chain-mfg", SignalSupplyChainEvent, 0.5},
		{CatTopics, "generative-ai", SignalGenAIInnovation, 0.5},
		{CatTopics, "blockchain", SignalBlockchainAdoption, 0.5},
		{CatTopics, "healthtech", SignalHealthtechInnovation, 0.5},
		{CatTopics, "edtech", SignalEdtechGrowth, 0.5},

		// Industry → signal mappings
		{CatIndustries, "tech", SignalTechGrowth, 0.5},
		{CatIndustries, "finance", SignalFinancialActivity, 0.5},
		{CatIndustries, "healthcare", SignalHealthcareDev, 0.5},
		{CatIndustries, "energy", SignalEnergyDev, 0.5},
		{CatIndustries, "manufacturing", SignalManufacturingActivity, 0.5},
		{CatIndustries, "agriculture", SignalAgricultureDev, 0.5},

		// Country → signal mappings (always produce geopolitical_event)
		{CatCountries, "", SignalGeopoliticalEvent, 0.3},
		{CatCountries, "", SignalMarketEntry, 0.4},

		// Technology → signal mappings (catch-all)
		{CatTechnologies, "", SignalTechAdoption, 0.4},

		// Organization → signal mappings (catch-all — context needed for specific type)
		{CatOrganizations, "", SignalPartnership, 0.3},
		{CatOrganizations, "", SignalInvestment, 0.3},
		{CatOrganizations, "", SignalAcquisition, 0.3},
		{CatOrganizations, "", SignalRegulatoryAction, 0.3},
		{CatOrganizations, "", SignalResearchPublication, 0.3},
		{CatOrganizations, "", SignalPatentFiling, 0.3},
	}
}