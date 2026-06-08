package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

const defaultTopics = "user.registered,user.login,user.logout,order.created,order.cancelled,order.filled,portfolio.updated,risk.score.updated,risk.breached,surveillance.alert.created,surveillance.alert.acknowledged,surveillance.alert.resolved,surveillance.alert.dismissed,notification.read,notification.failed,notification.sent,notification.retry_requested"

type Config struct {
	ServiceName               string
	Port                      string
	DatabaseURL               string
	JWTSecret                 string
	KafkaBrokers              []string
	KafkaTopics               []string
	KafkaGroupID              string
	EventProcessingMaxRetries int
	EventProcessingBackoffMS  int
	EventProcessingMultiplier float64
}

func Load() (Config, error) {
	cfg := Config{
		ServiceName:               getenv("SERVICE_NAME", "audit-service"),
		Port:                      getenv("PORT", getenv("AUDIT_SERVICE_PORT", "8092")),
		DatabaseURL:               getenv("DATABASE_URL", os.Getenv("AUDIT_DATABASE_URL")),
		JWTSecret:                 getenv("JWT_SECRET", os.Getenv("AUDIT_JWT_SECRET")),
		KafkaBrokers:              splitCSV(getenv("KAFKA_BROKERS", getenv("AUDIT_KAFKA_BROKERS", "redpanda:29092"))),
		KafkaTopics:               splitCSV(getenv("AUDIT_KAFKA_TOPICS", defaultTopics)),
		KafkaGroupID:              getenv("KAFKA_GROUP_ID", "audit-service"),
		EventProcessingMaxRetries: getenvInt("EVENT_PROCESSING_MAX_RETRIES", 3),
		EventProcessingBackoffMS:  getenvInt("EVENT_PROCESSING_RETRY_BACKOFF_MS", 500),
		EventProcessingMultiplier: getenvFloat("EVENT_PROCESSING_RETRY_BACKOFF_MULTIPLIER", 2),
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL or AUDIT_DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return cfg, errors.New("JWT_SECRET or AUDIT_JWT_SECRET is required")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return cfg, errors.New("KAFKA_BROKERS is required")
	}
	if len(cfg.KafkaTopics) == 0 {
		return cfg, errors.New("AUDIT_KAFKA_TOPICS is required")
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

func getenvFloat(key string, fallback float64) float64 {
	value, err := strconv.ParseFloat(getenv(key, ""), 64)
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
