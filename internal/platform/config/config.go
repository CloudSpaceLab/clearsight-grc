package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr         string
	Environment      string
	AllowedOrigin    string
	DatabaseURL      string
	DatabaseMinConns int32
	DatabaseMaxConns int32
	QueryTimeout     time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	IdleTimeout      time.Duration
	LogLevel         slog.Level
}

func Load() (Config, error) {
	cfg := Config{HTTPAddr: env("CLEARSIGHT_HTTP_ADDR", ":8080"), Environment: env("CLEARSIGHT_ENV", "development"), AllowedOrigin: env("CLEARSIGHT_ALLOWED_ORIGIN", "http://localhost:5173"), DatabaseURL: env("DATABASE_URL", ""), DatabaseMinConns: 2, DatabaseMaxConns: 20, QueryTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, LogLevel: slog.LevelInfo}
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
	if cfg.QueryTimeout, err = duration("CLEARSIGHT_QUERY_TIMEOUT", cfg.QueryTimeout); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMinConns, err = int32Value("CLEARSIGHT_DB_MIN_CONNS", cfg.DatabaseMinConns); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMaxConns, err = int32Value("CLEARSIGHT_DB_MAX_CONNS", cfg.DatabaseMaxConns); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMinConns < 0 || cfg.DatabaseMaxConns < 1 || cfg.DatabaseMinConns > cfg.DatabaseMaxConns {
		return Config{}, fmt.Errorf("invalid database pool bounds")
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
func int32Value(name string, fallback int32) (int32, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return int32(parsed), nil
}
