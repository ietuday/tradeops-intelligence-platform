package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	Port         string
	DatabaseURL  string
	KafkaBrokers []string
	JWTSecret    string
}

func Load() (Config, error) {
	cfg := Config{
		Port:         getenv("ORDER_SERVICE_PORT", "8080"),
		DatabaseURL:  os.Getenv("ORDER_DATABASE_URL"),
		KafkaBrokers: splitCSV(getenv("ORDER_KAFKA_BROKERS", "redpanda:29092")),
		JWTSecret:    os.Getenv("ORDER_JWT_SECRET"),
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("ORDER_DATABASE_URL is required")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return cfg, errors.New("ORDER_KAFKA_BROKERS is required")
	}
	if cfg.JWTSecret == "" {
		return cfg, errors.New("ORDER_JWT_SECRET is required")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	var values []string
	for _, part := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
