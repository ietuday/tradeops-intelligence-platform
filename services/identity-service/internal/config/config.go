package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port               string
	DatabaseURL        string
	RedisAddr          string
	JWTSecret          string
	IssuerURL          string
	Audience           string
	JWTPrivateKeyPath  string
	JWTKeyID           string
	OIDCEnabled        bool
	RefreshTokenSecret string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Port:               getenv("IDENTITY_SERVICE_PORT", "8080"),
		DatabaseURL:        os.Getenv("IDENTITY_DATABASE_URL"),
		RedisAddr:          getenv("IDENTITY_REDIS_ADDR", "redis:6379"),
		JWTSecret:          os.Getenv("IDENTITY_JWT_SECRET"),
		IssuerURL:          getenv("IDENTITY_ISSUER_URL", "http://identity-service:8080"),
		Audience:           getenv("IDENTITY_OIDC_AUDIENCE", "tradeops-api"),
		JWTPrivateKeyPath:  os.Getenv("IDENTITY_JWT_PRIVATE_KEY_PATH"),
		JWTKeyID:           getenv("IDENTITY_JWT_KID", "local-dev-key-1"),
		OIDCEnabled:        getenv("IDENTITY_OIDC_ENABLED", "true") != "false",
		RefreshTokenSecret: os.Getenv("IDENTITY_REFRESH_TOKEN_SECRET"),
		AccessTokenTTL:     minutes("IDENTITY_ACCESS_TOKEN_TTL_MINUTES", 15),
		RefreshTokenTTL:    hours("IDENTITY_REFRESH_TOKEN_TTL_HOURS", 24),
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("IDENTITY_DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" && !cfg.OIDCEnabled {
		return cfg, errors.New("IDENTITY_JWT_SECRET is required")
	}
	if cfg.RefreshTokenSecret == "" {
		return cfg, errors.New("IDENTITY_REFRESH_TOKEN_SECRET is required")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func minutes(key string, fallback int) time.Duration {
	value, err := strconv.Atoi(getenv(key, strconv.Itoa(fallback)))
	if err != nil || value <= 0 {
		value = fallback
	}
	return time.Duration(value) * time.Minute
}

func hours(key string, fallback int) time.Duration {
	value, err := strconv.Atoi(getenv(key, strconv.Itoa(fallback)))
	if err != nil || value <= 0 {
		value = fallback
	}
	return time.Duration(value) * time.Hour
}
