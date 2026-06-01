package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port              string
	DatabaseURL       string
	MQTTBroker        string
	MQTTTopic         string
	KafkaBrokers      []string
	KafkaTopic        string
	SimulatorEnabled  bool
	SimulatorInterval time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Port:              getenv("MARKET_DATA_SERVICE_PORT", "8080"),
		DatabaseURL:       os.Getenv("MARKET_DATA_DATABASE_URL"),
		MQTTBroker:        getenv("MARKET_DATA_MQTT_BROKER", "tcp://mosquitto:1883"),
		MQTTTopic:         getenv("MARKET_DATA_MQTT_TOPIC", "market/+/tick"),
		KafkaBrokers:      splitCSV(getenv("MARKET_DATA_KAFKA_BROKERS", "redpanda:29092")),
		KafkaTopic:        getenv("MARKET_DATA_KAFKA_TOPIC", "market.ticks"),
		SimulatorEnabled:  strings.EqualFold(getenv("MARKET_SIMULATOR_ENABLED", "true"), "true"),
		SimulatorInterval: milliseconds("MARKET_SIMULATOR_INTERVAL_MS", 1000),
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("MARKET_DATA_DATABASE_URL is required")
	}
	if len(cfg.KafkaBrokers) == 0 {
		return cfg, errors.New("MARKET_DATA_KAFKA_BROKERS is required")
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

func milliseconds(key string, fallback int) time.Duration {
	value, err := strconv.Atoi(getenv(key, strconv.Itoa(fallback)))
	if err != nil || value <= 0 {
		value = fallback
	}
	return time.Duration(value) * time.Millisecond
}
