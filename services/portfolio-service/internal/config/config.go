package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                      string
	DatabaseURL               string
	KafkaBrokers              []string
	OrderFilledTopic          string
	TradeTopic                string
	TradeConsumerGroup        string
	TradeDLQTopic             string
	PortfolioTopic            string
	SnapshotTopic             string
	JWTSecret                 string
	InitialCash               float64
	EventProcessingMaxRetries int
	EventProcessingBackoffMS  int
	EventProcessingMultiplier float64
	Outbox                    OutboxConfig
	ConsumerObs               ConsumerObsConfig
}

type OutboxConfig struct {
	Enabled      bool
	PollInterval time.Duration
	BatchSize    int
	MaxAttempts  int
	BaseBackoff  time.Duration
	MaxBackoff   time.Duration
}

type ConsumerObsConfig struct {
	Enabled              bool
	PollInterval         time.Duration
	LagWarnThreshold     int64
	LagCriticalThreshold int64
	StalledAfter         time.Duration
	MaxTopicScan         int
	DLQEnabled           bool
	DLQTopicSuffix       string
	DLQOldestAgeWarn     time.Duration
	DLQOldestAgeCritical time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Port:                      getenv("PORTFOLIO_SERVICE_PORT", "8080"),
		DatabaseURL:               os.Getenv("PORTFOLIO_DATABASE_URL"),
		KafkaBrokers:              splitCSV(getenv("PORTFOLIO_KAFKA_BROKERS", "redpanda:29092")),
		OrderFilledTopic:          getenv("PORTFOLIO_ORDER_FILLED_TOPIC", "order.filled"),
		TradeTopic:                getenv("PORTFOLIO_TRADE_TOPIC", "trade.executed"),
		TradeConsumerGroup:        getenv("PORTFOLIO_TRADE_CONSUMER_GROUP", "portfolio-service-trade-v1"),
		TradeDLQTopic:             getenv("PORTFOLIO_TRADE_DLQ_TOPIC", "portfolio.trade.dlq"),
		PortfolioTopic:            getenv("PORTFOLIO_UPDATED_TOPIC", "portfolio.updated"),
		SnapshotTopic:             getenv("PORTFOLIO_SNAPSHOT_TOPIC", "portfolio.snapshot.created"),
		JWTSecret:                 os.Getenv("PORTFOLIO_JWT_SECRET"),
		InitialCash:               floatEnv("PORTFOLIO_INITIAL_CASH", 100000),
		EventProcessingMaxRetries: intEnv("EVENT_PROCESSING_MAX_RETRIES", 3),
		EventProcessingBackoffMS:  intEnv("EVENT_PROCESSING_RETRY_BACKOFF_MS", 500),
		EventProcessingMultiplier: floatEnv("EVENT_PROCESSING_RETRY_BACKOFF_MULTIPLIER", 2),
		Outbox: OutboxConfig{
			Enabled:      boolEnv("PORTFOLIO_OUTBOX_ENABLED", true),
			PollInterval: durationEnv("PORTFOLIO_OUTBOX_POLL_INTERVAL", 2*time.Second),
			BatchSize:    intEnv("PORTFOLIO_OUTBOX_BATCH_SIZE", 100),
			MaxAttempts:  intEnv("PORTFOLIO_OUTBOX_MAX_ATTEMPTS", 10),
			BaseBackoff:  durationEnv("PORTFOLIO_OUTBOX_BASE_BACKOFF", time.Second),
			MaxBackoff:   durationEnv("PORTFOLIO_OUTBOX_MAX_BACKOFF", time.Minute),
		},
		ConsumerObs: ConsumerObsConfig{
			Enabled:              boolEnv("CONSUMER_OBS_ENABLED", true),
			PollInterval:         durationEnv("CONSUMER_OBS_POLL_INTERVAL", 10*time.Second),
			LagWarnThreshold:     int64Env("CONSUMER_OBS_LAG_WARN_THRESHOLD", 1000),
			LagCriticalThreshold: int64Env("CONSUMER_OBS_LAG_CRITICAL_THRESHOLD", 10000),
			StalledAfter:         durationEnv("CONSUMER_OBS_STALLED_AFTER", 2*time.Minute),
			MaxTopicScan:         intEnv("CONSUMER_OBS_MAX_TOPIC_SCAN", 100),
			DLQEnabled:           boolEnv("DLQ_OBS_ENABLED", true),
			DLQTopicSuffix:       getenv("DLQ_TOPIC_SUFFIX", ".dlq"),
			DLQOldestAgeWarn:     durationEnv("DLQ_OLDEST_AGE_WARN", 5*time.Minute),
			DLQOldestAgeCritical: durationEnv("DLQ_OLDEST_AGE_CRITICAL", 30*time.Minute),
		},
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("PORTFOLIO_DATABASE_URL is required")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return cfg, errors.New("PORTFOLIO_KAFKA_BROKERS is required")
	}
	if cfg.TradeTopic == "" || cfg.TradeConsumerGroup == "" || cfg.TradeDLQTopic == "" {
		return cfg, errors.New("portfolio trade topic, consumer group, and DLQ topic are required")
	}
	if cfg.JWTSecret == "" {
		return cfg, errors.New("PORTFOLIO_JWT_SECRET is required")
	}
	if err := validateOutbox(cfg.Outbox); err != nil {
		return cfg, err
	}
	if err := validateConsumerObs(cfg.ConsumerObs); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func validateConsumerObs(cfg ConsumerObsConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if cfg.PollInterval <= 0 {
		return errors.New("CONSUMER_OBS_POLL_INTERVAL must be positive")
	}
	if cfg.LagWarnThreshold < 0 || cfg.LagCriticalThreshold < 0 {
		return errors.New("consumer lag thresholds must be non-negative")
	}
	if cfg.LagCriticalThreshold > 0 && cfg.LagWarnThreshold > cfg.LagCriticalThreshold {
		return errors.New("CONSUMER_OBS_LAG_WARN_THRESHOLD must be less than or equal to CONSUMER_OBS_LAG_CRITICAL_THRESHOLD")
	}
	if cfg.StalledAfter <= 0 {
		return errors.New("CONSUMER_OBS_STALLED_AFTER must be positive")
	}
	return nil
}

func validateOutbox(cfg OutboxConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if cfg.PollInterval <= 0 {
		return errors.New("PORTFOLIO_OUTBOX_POLL_INTERVAL must be positive")
	}
	if cfg.BatchSize <= 0 {
		return errors.New("PORTFOLIO_OUTBOX_BATCH_SIZE must be greater than zero")
	}
	if cfg.MaxAttempts <= 0 {
		return errors.New("PORTFOLIO_OUTBOX_MAX_ATTEMPTS must be greater than zero")
	}
	if cfg.BaseBackoff <= 0 {
		return errors.New("PORTFOLIO_OUTBOX_BASE_BACKOFF must be positive")
	}
	if cfg.MaxBackoff < cfg.BaseBackoff {
		return errors.New("PORTFOLIO_OUTBOX_MAX_BACKOFF must be greater than or equal to PORTFOLIO_OUTBOX_BASE_BACKOFF")
	}
	return nil
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

func intEnv(key string, fallback int) int {
	value, err := strconv.Atoi(getenv(key, ""))
	if err != nil {
		return fallback
	}
	return value
}

func int64Env(key string, fallback int64) int64 {
	value, err := strconv.ParseInt(getenv(key, ""), 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

func boolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
