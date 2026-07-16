package config

import (
	"os"
	"strings"
)

type Config struct {
	Port            string
	RedpandaBrokers []string
	OpenSearchURL   string
	LLMGatewayURL   string
	DatabaseURL     string
}

func Load() Config {
	brokers := envOr("REDPANDA_BROKERS", "redpanda:9092")
	return Config{
		Port:            envOr("PORT", "8003"),
		RedpandaBrokers: strings.Split(brokers, ","),
		OpenSearchURL:   envOr("OPENSEARCH_URL", "http://opensearch:9200"),
		LLMGatewayURL:   envOr("LLM_GATEWAY_URL", "http://verso-llm-gateway:8011"),
		DatabaseURL:     envOr("DATABASE_URL", "postgres://verso:verso_dev@postgres:5432/verso?search_path=ai&sslmode=disable"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
