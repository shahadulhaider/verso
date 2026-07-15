// Package config loads service configuration from environment variables.
package config

import (
	"os"
	"time"
)

// Config holds all service configuration.
type Config struct {
	Port        string
	DatabaseURL string
	JWTKeyPath  string
	JWTExpiry   time.Duration
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	cfg := Config{
		Port:        envOr("PORT", "8001"),
		DatabaseURL: envOr("DATABASE_URL", "postgres://verso:verso_dev@localhost:5432/verso?search_path=identity&sslmode=disable"),
		JWTKeyPath:  envOr("JWT_KEY_PATH", "/tmp/verso-jwt-key.pem"),
	}

	expiryStr := envOr("JWT_EXPIRY", "24h")
	d, err := time.ParseDuration(expiryStr)
	if err != nil {
		d = 24 * time.Hour
	}
	cfg.JWTExpiry = d

	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
