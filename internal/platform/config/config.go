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
	HTTPAddr                 string
	Environment              string
	AllowedOrigin            string
	DatabaseURL              string
	DatabaseMinConns         int32
	DatabaseMaxConns         int32
	QueryTimeout             time.Duration
	ReadTimeout              time.Duration
	WriteTimeout             time.Duration
	IdleTimeout              time.Duration
	WorkerID                 string
	WorkerPoll               time.Duration
	ArtifactRoot             string
	MaxArtifactBytes         int64
	CaptureSessionTTL        time.Duration
	IdentityMode             string
	IdentityHMACSecret       string
	IdentityMaxSkew          time.Duration
	CommandAuthorizationMode string
	DemoTenantID             string
	DemoPrincipalID          string
	DemoLegalEntityID        string
	LogLevel                 slog.Level
}

func Load() (Config, error) {
	environment := env("CLEARSIGHT_ENV", "development")
	defaultIdentityMode := "development"
	defaultCommandMode := "audit"
	if strings.EqualFold(environment, "production") {
		defaultIdentityMode = "signed"
		defaultCommandMode = "enforce"
	}
	cfg := Config{
		HTTPAddr:                 env("CLEARSIGHT_HTTP_ADDR", ":8080"),
		Environment:              environment,
		AllowedOrigin:            env("CLEARSIGHT_ALLOWED_ORIGIN", "http://localhost:5173"),
		DatabaseURL:              env("DATABASE_URL", ""),
		DatabaseMinConns:         2,
		DatabaseMaxConns:         20,
		QueryTimeout:             3 * time.Second,
		ReadTimeout:              5 * time.Second,
		WriteTimeout:             10 * time.Second,
		IdleTimeout:              60 * time.Second,
		WorkerID:                 env("CLEARSIGHT_WORKER_ID", "worker-local"),
		WorkerPoll:               time.Second,
		ArtifactRoot:             env("CLEARSIGHT_ARTIFACT_ROOT", "./var/artifacts"),
		MaxArtifactBytes:         20 << 20,
		CaptureSessionTTL:        20 * time.Minute,
		IdentityMode:             strings.ToLower(env("CLEARSIGHT_IDENTITY_MODE", defaultIdentityMode)),
		IdentityHMACSecret:       env("CLEARSIGHT_IDENTITY_HMAC_SECRET", ""),
		IdentityMaxSkew:          2 * time.Minute,
		CommandAuthorizationMode: strings.ToLower(env("CLEARSIGHT_COMMAND_AUTHORIZATION", defaultCommandMode)),
		DemoTenantID:             env("CLEARSIGHT_DEMO_TENANT_ID", "bank-demo"),
		DemoPrincipalID:          env("CLEARSIGHT_DEMO_PRINCIPAL_ID", "role-cro"),
		DemoLegalEntityID:        env("CLEARSIGHT_DEMO_LEGAL_ENTITY_ID", "bank-ng"),
		LogLevel:                 slog.LevelInfo,
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
	if cfg.QueryTimeout, err = duration("CLEARSIGHT_QUERY_TIMEOUT", cfg.QueryTimeout); err != nil {
		return Config{}, err
	}
	if cfg.WorkerPoll, err = duration("CLEARSIGHT_WORKER_POLL", cfg.WorkerPoll); err != nil {
		return Config{}, err
	}
	if cfg.CaptureSessionTTL, err = duration("CLEARSIGHT_CAPTURE_SESSION_TTL", cfg.CaptureSessionTTL); err != nil {
		return Config{}, err
	}
	if cfg.IdentityMaxSkew, err = duration("CLEARSIGHT_IDENTITY_MAX_SKEW", cfg.IdentityMaxSkew); err != nil {
		return Config{}, err
	}
	if cfg.WorkerPoll <= 0 || cfg.CaptureSessionTTL < time.Minute || cfg.CaptureSessionTTL > time.Hour {
		return Config{}, fmt.Errorf("worker poll must be positive and capture session ttl must be 1-60 minutes")
	}
	if cfg.IdentityMaxSkew <= 0 || cfg.IdentityMaxSkew > 10*time.Minute {
		return Config{}, fmt.Errorf("identity maximum clock skew must be between 1 second and 10 minutes")
	}
	if cfg.DatabaseMinConns, err = int32Value("CLEARSIGHT_DB_MIN_CONNS", cfg.DatabaseMinConns); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMaxConns, err = int32Value("CLEARSIGHT_DB_MAX_CONNS", cfg.DatabaseMaxConns); err != nil {
		return Config{}, err
	}
	if cfg.MaxArtifactBytes, err = int64Value("CLEARSIGHT_MAX_ARTIFACT_BYTES", cfg.MaxArtifactBytes); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMinConns < 0 || cfg.DatabaseMaxConns < 1 || cfg.DatabaseMinConns > cfg.DatabaseMaxConns {
		return Config{}, fmt.Errorf("invalid database pool bounds")
	}
	if cfg.MaxArtifactBytes < 1024 || cfg.MaxArtifactBytes > 100<<20 {
		return Config{}, fmt.Errorf("CLEARSIGHT_MAX_ARTIFACT_BYTES must be between 1 KiB and 100 MiB")
	}
	switch cfg.IdentityMode {
	case "development":
	case "signed":
		if len(cfg.IdentityHMACSecret) < 32 {
			return Config{}, fmt.Errorf("CLEARSIGHT_IDENTITY_HMAC_SECRET must contain at least 32 characters in signed identity mode")
		}
	default:
		return Config{}, fmt.Errorf("CLEARSIGHT_IDENTITY_MODE must be development or signed")
	}
	switch cfg.CommandAuthorizationMode {
	case "off", "audit", "enforce":
	default:
		return Config{}, fmt.Errorf("CLEARSIGHT_COMMAND_AUTHORIZATION must be off, audit or enforce")
	}
	if strings.EqualFold(cfg.Environment, "production") {
		if cfg.IdentityMode != "signed" || cfg.CommandAuthorizationMode != "enforce" {
			return Config{}, fmt.Errorf("production requires signed identity and enforced command authorization")
		}
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
func int64Value(name string, fallback int64) (int64, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}
