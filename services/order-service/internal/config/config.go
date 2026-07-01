package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port            string
	DatabaseURL     string
	KafkaBrokers    []string
	JWTSecret       string
	Outbox          OutboxConfig
	Expiry          ExpiryConfig
	ShutdownTimeout time.Duration
}

type OutboxConfig struct {
	Enabled            bool
	PollInterval       time.Duration
	BatchSize          int
	PublishConcurrency int
	PublishTimeout     time.Duration
	ShutdownTimeout    time.Duration
	LeaseDuration      time.Duration
	BaseBackoff        time.Duration
	MaxBackoff         time.Duration
	MaxAttempts        int
	ErrorMaxLength     int
}

type ExpiryConfig struct {
	Enabled           bool
	PollInterval      time.Duration
	BatchSize         int
	MaxBatchesPerPoll int
	ProcessingTimeout time.Duration
	ShutdownTimeout   time.Duration
	DayTimezone       string
	DayCloseTime      string
}

func Load() (Config, error) {
	cfg := Config{
		Port:            getenv("ORDER_SERVICE_PORT", "8080"),
		DatabaseURL:     os.Getenv("ORDER_DATABASE_URL"),
		KafkaBrokers:    splitCSV(getenv("ORDER_KAFKA_BROKERS", "redpanda:29092")),
		JWTSecret:       os.Getenv("ORDER_JWT_SECRET"),
		ShutdownTimeout: durationEnv("ORDER_SHUTDOWN_TIMEOUT", 10*time.Second),
		Outbox: OutboxConfig{
			Enabled:            boolEnv("OUTBOX_ENABLED", true),
			PollInterval:       durationEnv("OUTBOX_POLL_INTERVAL", time.Second),
			BatchSize:          intEnv("OUTBOX_BATCH_SIZE", 50),
			PublishConcurrency: intEnv("OUTBOX_PUBLISH_CONCURRENCY", 5),
			PublishTimeout:     durationEnv("OUTBOX_PUBLISH_TIMEOUT", 5*time.Second),
			ShutdownTimeout:    durationEnv("OUTBOX_SHUTDOWN_TIMEOUT", 15*time.Second),
			LeaseDuration:      durationEnv("OUTBOX_LEASE_DURATION", 30*time.Second),
			BaseBackoff:        durationEnv("OUTBOX_BASE_BACKOFF", time.Second),
			MaxBackoff:         durationEnv("OUTBOX_MAX_BACKOFF", 5*time.Minute),
			MaxAttempts:        intEnv("OUTBOX_MAX_ATTEMPTS", 10),
			ErrorMaxLength:     intEnv("OUTBOX_ERROR_MAX_LENGTH", 512),
		},
		Expiry: ExpiryConfig{
			Enabled:           boolEnv("ORDER_EXPIRY_ENABLED", true),
			PollInterval:      durationEnv("ORDER_EXPIRY_POLL_INTERVAL", 5*time.Second),
			BatchSize:         intEnv("ORDER_EXPIRY_BATCH_SIZE", 100),
			MaxBatchesPerPoll: intEnv("ORDER_EXPIRY_MAX_BATCHES_PER_POLL", 10),
			ProcessingTimeout: durationEnv("ORDER_EXPIRY_PROCESSING_TIMEOUT", 10*time.Second),
			ShutdownTimeout:   durationEnv("ORDER_EXPIRY_SHUTDOWN_TIMEOUT", 15*time.Second),
			DayTimezone:       getenv("ORDER_DAY_TIMEZONE", "America/New_York"),
			DayCloseTime:      getenv("ORDER_DAY_CLOSE_TIME", "16:00"),
		},
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
	if err := validateOutbox(cfg.Outbox); err != nil {
		return cfg, err
	}
	if err := validateExpiry(cfg.Expiry); err != nil {
		return cfg, err
	}
	if cfg.ShutdownTimeout <= 0 {
		return cfg, errors.New("ORDER_SHUTDOWN_TIMEOUT must be positive")
	}
	return cfg, nil
}

func validateExpiry(cfg ExpiryConfig) error {
	if cfg.PollInterval <= 0 {
		return errors.New("ORDER_EXPIRY_POLL_INTERVAL must be positive")
	}
	if cfg.BatchSize <= 0 {
		return errors.New("ORDER_EXPIRY_BATCH_SIZE must be greater than zero")
	}
	if cfg.MaxBatchesPerPoll <= 0 {
		return errors.New("ORDER_EXPIRY_MAX_BATCHES_PER_POLL must be greater than zero")
	}
	if cfg.ProcessingTimeout <= 0 {
		return errors.New("ORDER_EXPIRY_PROCESSING_TIMEOUT must be positive")
	}
	if cfg.ShutdownTimeout <= 0 {
		return errors.New("ORDER_EXPIRY_SHUTDOWN_TIMEOUT must be positive")
	}
	if _, err := time.LoadLocation(cfg.DayTimezone); err != nil {
		return fmt.Errorf("ORDER_DAY_TIMEZONE is invalid: %w", err)
	}
	if _, err := time.Parse("15:04", cfg.DayCloseTime); err != nil {
		return fmt.Errorf("ORDER_DAY_CLOSE_TIME must use HH:MM: %w", err)
	}
	return nil
}

func validateOutbox(cfg OutboxConfig) error {
	if cfg.PollInterval <= 0 {
		return errors.New("OUTBOX_POLL_INTERVAL must be positive")
	}
	if cfg.BatchSize <= 0 {
		return errors.New("OUTBOX_BATCH_SIZE must be greater than zero")
	}
	if cfg.PublishConcurrency <= 0 {
		return errors.New("OUTBOX_PUBLISH_CONCURRENCY must be greater than zero")
	}
	if cfg.PublishConcurrency > cfg.BatchSize {
		return errors.New("OUTBOX_PUBLISH_CONCURRENCY must not exceed OUTBOX_BATCH_SIZE")
	}
	if cfg.PublishTimeout <= 0 {
		return errors.New("OUTBOX_PUBLISH_TIMEOUT must be positive")
	}
	if cfg.LeaseDuration <= cfg.PublishTimeout {
		return errors.New("OUTBOX_LEASE_DURATION must be greater than OUTBOX_PUBLISH_TIMEOUT")
	}
	if cfg.BaseBackoff <= 0 {
		return errors.New("OUTBOX_BASE_BACKOFF must be positive")
	}
	if cfg.MaxBackoff < cfg.BaseBackoff {
		return errors.New("OUTBOX_MAX_BACKOFF must be greater than or equal to OUTBOX_BASE_BACKOFF")
	}
	if cfg.MaxAttempts <= 0 {
		return errors.New("OUTBOX_MAX_ATTEMPTS must be greater than zero")
	}
	if cfg.ShutdownTimeout <= 0 {
		return errors.New("OUTBOX_SHUTDOWN_TIMEOUT must be positive")
	}
	if cfg.ErrorMaxLength <= 0 {
		return errors.New("OUTBOX_ERROR_MAX_LENGTH must be greater than zero")
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

func intEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
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
		if millis, convErr := strconv.Atoi(value); convErr == nil {
			return time.Duration(millis) * time.Millisecond
		}
		return fallback
	}
	if parsed <= 0 {
		return parsed
	}
	return parsed
}

func (c OutboxConfig) String() string {
	return fmt.Sprintf("enabled=%t poll=%s batch=%d concurrency=%d timeout=%s lease=%s maxAttempts=%d", c.Enabled, c.PollInterval, c.BatchSize, c.PublishConcurrency, c.PublishTimeout, c.LeaseDuration, c.MaxAttempts)
}
