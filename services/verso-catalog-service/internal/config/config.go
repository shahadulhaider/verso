package config

import "os"

type Config struct {
	Port        string
	DatabaseURL string
	JWKSURL     string
}

func Load() Config {
	return Config{
		Port:        envOr("PORT", "8002"),
		DatabaseURL: envOr("DATABASE_URL", "postgres://verso:verso_dev@localhost:5432/verso?search_path=catalog&sslmode=disable"),
		JWKSURL:     envOr("JWKS_URL", "http://localhost:8001/.well-known/jwks.json"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
