package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	MigrateOnStart  bool
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
	SessionTTL      time.Duration
	// CookieSecure should only be disabled for local development over http.
	CookieSecure bool
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    getEnv("API_HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	var err error
	if cfg.LogLevel, err = parseLevel(getEnv("API_LOG_LEVEL", "info")); err != nil {
		return Config{}, err
	}
	if cfg.MigrateOnStart, err = getBool("API_MIGRATE_ON_START", false); err != nil {
		return Config{}, err
	}
	if cfg.CookieSecure, err = getBool("API_COOKIE_SECURE", true); err != nil {
		return Config{}, err
	}
	if cfg.ShutdownTimeout, err = getDuration("API_SHUTDOWN_TIMEOUT", 10*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.SessionTTL, err = getDuration("API_SESSION_TTL", 90*24*time.Hour); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getBool(key string, fallback bool) (bool, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return d, nil
}

func parseLevel(s string) (slog.Level, error) {
	var l slog.Level
	if err := l.UnmarshalText([]byte(strings.ToUpper(s))); err != nil {
		return 0, fmt.Errorf("API_LOG_LEVEL: %w", err)
	}
	return l, nil
}
