package config

import (
	"errors"
	"os"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
}

func Load() (Config, error) {
	cfg := Config{
		Port:        getenv("NOTIFICATION_SERVICE_PORT", "8091"),
		DatabaseURL: os.Getenv("NOTIFICATION_DATABASE_URL"),
		JWTSecret:   os.Getenv("NOTIFICATION_JWT_SECRET"),
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("NOTIFICATION_DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return cfg, errors.New("NOTIFICATION_JWT_SECRET is required")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
