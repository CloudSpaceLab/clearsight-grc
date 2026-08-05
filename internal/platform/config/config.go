package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr      string
	Environment   string
	AllowedOrigin string
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	IdleTimeout   time.Duration
	LogLevel      slog.Level
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:      env("CLEARSIGHT_HTTP_ADDR", ":8080"),
		Environment:   env("CLEARSIGHT_ENV", "development"),
		AllowedOrigin: env("CLEARSIGHT_ALLOWED_ORIGIN", "http://localhost:5173"),
		ReadTimeout:   5 * time.Second,
		WriteTimeout:  10 * time.Second,
		IdleTimeout:   60 * time.Second,
		LogLevel:      slog.LevelInfo,
	}

	var err error
	if cfg.ReadTimeout, err = duration("CLEARSIGHT_READ_TIMEOUT", cfg.ReadTimeout); err != nil {
		return Config{}, err
	}
	if cfg.WriteTimeout, err = duration("CLEARSIGHT_WRITE_TIMEOUT", cfg.WriteTimeout); err != nil {
		return Config{}, err
	}
	if cfg.IdleTimeout, err = duration("CLEARSIGHT_IDLE_TIMEOUT", cfg.IdleTimeout); err != nil {
		return Config{}, err
	}
	if strings.EqualFold(env("CLEARSIGHT_LOG_LEVEL", "info"), "debug") {
		cfg.LogLevel = slog.LevelDebug
	}
	return cfg, nil
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func duration(name string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}
