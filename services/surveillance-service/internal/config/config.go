package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                         string
	DatabaseURL                  string
	KafkaBrokers                 []string
	KafkaTopics                  []string
	JWTSecret                    string
	LargeOrderThreshold          float64
	RapidOrderLimit              int
	RapidOrderWindowSeconds      int
	HighCancelLimit              int
	HighCancelWindowSeconds      int
	RiskScoreThreshold           float64
	AbnormalPriceMovementPercent float64
	EventProcessingMaxRetries    int
	EventProcessingBackoffMS     int
	EventProcessingMultiplier    float64
}

func Load() (Config, error) {
	cfg := Config{
		Port:                         getenv("SURVEILLANCE_SERVICE_PORT", "8090"),
		DatabaseURL:                  os.Getenv("SURVEILLANCE_DATABASE_URL"),
		KafkaBrokers:                 splitCSV(getenv("SURVEILLANCE_KAFKA_BROKERS", "redpanda:29092")),
		KafkaTopics:                  splitCSV(getenv("SURVEILLANCE_KAFKA_TOPICS", "order.created,order.filled,order.cancelled,portfolio.updated,risk.score.updated,market.ticks,strategy.signal.generated")),
		JWTSecret:                    os.Getenv("SURVEILLANCE_JWT_SECRET"),
		LargeOrderThreshold:          getenvFloat("SURVEILLANCE_LARGE_ORDER_THRESHOLD", 100000),
		RapidOrderLimit:              getenvInt("SURVEILLANCE_RAPID_ORDER_LIMIT", 5),
		RapidOrderWindowSeconds:      getenvInt("SURVEILLANCE_RAPID_ORDER_WINDOW_SECONDS", 60),
		HighCancelLimit:              getenvInt("SURVEILLANCE_HIGH_CANCEL_LIMIT", 3),
		HighCancelWindowSeconds:      getenvInt("SURVEILLANCE_HIGH_CANCEL_WINDOW_SECONDS", 300),
		RiskScoreThreshold:           getenvFloat("SURVEILLANCE_RISK_SCORE_THRESHOLD", 80),
		AbnormalPriceMovementPercent: getenvFloat("SURVEILLANCE_ABNORMAL_PRICE_MOVEMENT_PERCENT", 10),
		EventProcessingMaxRetries:    getenvInt("EVENT_PROCESSING_MAX_RETRIES", 3),
		EventProcessingBackoffMS:     getenvInt("EVENT_PROCESSING_RETRY_BACKOFF_MS", 500),
		EventProcessingMultiplier:    getenvFloat("EVENT_PROCESSING_RETRY_BACKOFF_MULTIPLIER", 2),
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("SURVEILLANCE_DATABASE_URL is required")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return cfg, errors.New("SURVEILLANCE_KAFKA_BROKERS is required")
	}
	if len(cfg.KafkaTopics) == 0 {
		return cfg, errors.New("SURVEILLANCE_KAFKA_TOPICS is required")
	}
	if cfg.JWTSecret == "" {
		return cfg, errors.New("SURVEILLANCE_JWT_SECRET is required")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvFloat(key string, fallback float64) float64 {
	value, err := strconv.ParseFloat(getenv(key, ""), 64)
	if err != nil {
		return fallback
	}
	return value
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
