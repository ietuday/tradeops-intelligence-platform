package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                  string
	DatabaseURL           string
	JWTSecret             string
	KafkaBrokers          []string
	KafkaTopics           []string
	WebhookTimeoutSeconds int
	WebhookMaxRetries     int
}

func Load() (Config, error) {
	cfg := Config{
		Port:                  getenv("NOTIFICATION_SERVICE_PORT", "8091"),
		DatabaseURL:           os.Getenv("NOTIFICATION_DATABASE_URL"),
		JWTSecret:             os.Getenv("NOTIFICATION_JWT_SECRET"),
		KafkaBrokers:          splitCSV(getenv("NOTIFICATION_KAFKA_BROKERS", "redpanda:29092")),
		KafkaTopics:           splitCSV(getenv("NOTIFICATION_KAFKA_TOPICS", "surveillance.alert.created,surveillance.alert.acknowledged,surveillance.alert.resolved,surveillance.alert.dismissed")),
		WebhookTimeoutSeconds: getenvInt("NOTIFICATION_WEBHOOK_TIMEOUT_SECONDS", 3),
		WebhookMaxRetries:     getenvInt("NOTIFICATION_WEBHOOK_MAX_RETRIES", 3),
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("NOTIFICATION_DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return cfg, errors.New("NOTIFICATION_JWT_SECRET is required")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return cfg, errors.New("NOTIFICATION_KAFKA_BROKERS is required")
	}
	if len(cfg.KafkaTopics) == 0 {
		return cfg, errors.New("NOTIFICATION_KAFKA_TOPICS is required")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	value, err := strconv.Atoi(getenv(key, ""))
	if err != nil {
		return fallback
	}
	return value
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
