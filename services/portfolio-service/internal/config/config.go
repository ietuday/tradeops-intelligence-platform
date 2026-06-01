package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port             string
	DatabaseURL      string
	KafkaBrokers     []string
	OrderFilledTopic string
	PortfolioTopic   string
	SnapshotTopic    string
	JWTSecret        string
	InitialCash      float64
}

func Load() (Config, error) {
	cfg := Config{
		Port:             getenv("PORTFOLIO_SERVICE_PORT", "8080"),
		DatabaseURL:      os.Getenv("PORTFOLIO_DATABASE_URL"),
		KafkaBrokers:     splitCSV(getenv("PORTFOLIO_KAFKA_BROKERS", "redpanda:29092")),
		OrderFilledTopic: getenv("PORTFOLIO_ORDER_FILLED_TOPIC", "order.filled"),
		PortfolioTopic:   getenv("PORTFOLIO_UPDATED_TOPIC", "portfolio.updated"),
		SnapshotTopic:    getenv("PORTFOLIO_SNAPSHOT_TOPIC", "portfolio.snapshot.created"),
		JWTSecret:        os.Getenv("PORTFOLIO_JWT_SECRET"),
		InitialCash:      floatEnv("PORTFOLIO_INITIAL_CASH", 100000),
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("PORTFOLIO_DATABASE_URL is required")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return cfg, errors.New("PORTFOLIO_KAFKA_BROKERS is required")
	}
	if cfg.JWTSecret == "" {
		return cfg, errors.New("PORTFOLIO_JWT_SECRET is required")
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

func floatEnv(key string, fallback float64) float64 {
	value, err := strconv.ParseFloat(getenv(key, ""), 64)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}
