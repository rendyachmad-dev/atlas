package configs

import "os"

// Config holds all application configuration.
type Config struct {
	DatabaseURL    string
	Port           string
	LogLevel       string
	AnthropicAPIKey string
	AnthropicModel  string
}

// Load reads configuration from environment variables.
// Missing optional vars fall back to defaults.
func Load() *Config {
	return &Config{
		DatabaseURL:     envOrDefault("DATABASE_URL", "postgres://localhost:5432/rss_atlas?sslmode=disable"),
		Port:            envOrDefault("PORT", "8080"),
		LogLevel:        envOrDefault("LOG_LEVEL", "info"),
		AnthropicAPIKey: envOrDefault("ANTHROPIC_API_KEY", ""),
		AnthropicModel:  envOrDefault("ANTHROPIC_MODEL", "claude-sonnet-5"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}