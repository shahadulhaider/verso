// Package config loads service configuration from environment variables.
package config

import (
	"os"
	"strings"
)

// Config holds all service configuration.
type Config struct {
	Port            string
	DatabaseURL     string
	JWKSURL         string
	RedpandaBrokers []string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	brokers := envOr("REDPANDA_BROKERS", "redpanda:9092")
	return Config{
		Port:            envOr("SERVICE_PORT", "8004"),
		DatabaseURL:     envOr("DATABASE_URL", "postgres://verso:verso_dev@localhost:5432/verso?search_path=profile&sslmode=disable"),
		JWKSURL:         envOr("JWKS_URL", "http://localhost:8001/.well-known/jwks.json"),
		RedpandaBrokers: strings.Split(brokers, ","),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
