package config

import (
	"os"
	"strings"
)

type Config struct {
	Port            string
	RedpandaBrokers []string
	OpenSearchURL   string
}

func Load() Config {
	brokers := envOr("REDPANDA_BROKERS", "redpanda:9092")
	return Config{
		Port:            envOr("PORT", "8003"),
		RedpandaBrokers: strings.Split(brokers, ","),
		OpenSearchURL:   envOr("OPENSEARCH_URL", "http://opensearch:9200"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
