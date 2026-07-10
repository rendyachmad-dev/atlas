package configs

import "os"

// Config holds all application configuration.
type Config struct {
	DatabaseURL    string
	Port           string
	LogLevel       string
	LLMProvider    string
	LLMAPIKey      string
	LLMModel       string
	// Deprecated: use LLMAPIKey / LLMModel instead
	AnthropicAPIKey string
	AnthropicModel  string
}

// Load reads configuration from environment variables.
// Missing optional vars fall back to defaults.
func Load() *Config {
	// LLM_API_KEY is the universal key. Fall back to ANTHROPIC_API_KEY for backward compat.
	apiKey := envOrDefault("LLM_API_KEY", "")
	if apiKey == "" {
		apiKey = envOrDefault("ANTHROPIC_API_KEY", "")
	}
	model := envOrDefault("LLM_MODEL", "")
	if model == "" {
		model = envOrDefault("ANTHROPIC_MODEL", "claude-sonnet-5")
	}
	return &Config{
		DatabaseURL:     envOrDefault("DATABASE_URL", "postgres://localhost:5432/rss_atlas?sslmode=disable"),
		Port:            envOrDefault("PORT", "8080"),
		LogLevel:        envOrDefault("LOG_LEVEL", "info"),
		LLMProvider:     envOrDefault("LLM_PROVIDER", "anthropic"),
		LLMAPIKey:       apiKey,
		LLMModel:        model,
		AnthropicAPIKey: apiKey,
		AnthropicModel:  model,
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}