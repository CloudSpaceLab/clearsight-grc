package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr                              string
	Environment                           string
	ReleaseSHA                            string
	AllowedOrigin                         string
	DatabaseURL                           string
	DatabaseMinConns                      int32
	DatabaseMaxConns                      int32
	QueryTimeout                          time.Duration
	ReadTimeout                           time.Duration
	WriteTimeout                          time.Duration
	IdleTimeout                           time.Duration
	WorkerID                              string
	WorkerPoll                            time.Duration
	VendorRefreshBatchSize                int
	VendorRefreshCadence                  time.Duration
	VendorRefreshLease                    time.Duration
	VendorRefreshDocumentLead             time.Duration
	VendorRefreshFactConfirmationInterval time.Duration
	ArtifactRoot                          string
	MaxArtifactBytes                      int64
	CaptureSessionTTL                     time.Duration
	CapturePublicBaseURL                  string
	RecipientSecurity                     RecipientSecurityConfig
	IdentityMode                          string
	IdentityHMACSecret                    string
	IdentityMaxSkew                       time.Duration
	OIDCIssuer                            string
	OIDCClientID                          string
	OIDCClientSecret                      string
	OIDCRedirectURL                       string
	OIDCSessionLifetime                   time.Duration
	OIDCSessionIdleTimeout                time.Duration
	OIDCSecureCookies                     bool
	CommandAuthorizationMode              string
	DemoMode                              bool
	DocumentImportAllowUnscannedAnalysis  bool
	VendorBrandDiscoveryEnabled           bool
	DemoTenantID                          string
	DemoPrincipalID                       string
	DemoLegalEntityID                     string
	DemoRoleCodes                         []string
	LogLevel                              slog.Level
}

var releaseSHAPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

func Load() (Config, error) {
	environment := env("CLEARSIGHT_ENV", "development")
	production := strings.EqualFold(environment, "production")
	defaultIdentityMode := "development"
	defaultCommandMode := "audit"
	if production {
		defaultIdentityMode = "signed"
		defaultCommandMode = "enforce"
	}
	cfg := Config{
		HTTPAddr:                              env("CLEARSIGHT_HTTP_ADDR", ":8080"),
		Environment:                           environment,
		ReleaseSHA:                            strings.ToLower(env("CLEARSIGHT_RELEASE_SHA", "")),
		AllowedOrigin:                         env("CLEARSIGHT_ALLOWED_ORIGIN", "http://localhost:5173"),
		DatabaseURL:                           env("DATABASE_URL", ""),
		DatabaseMinConns:                      2,
		DatabaseMaxConns:                      20,
		QueryTimeout:                          3 * time.Second,
		ReadTimeout:                           5 * time.Second,
		WriteTimeout:                          10 * time.Second,
		IdleTimeout:                           60 * time.Second,
		WorkerID:                              env("CLEARSIGHT_WORKER_ID", "worker-local"),
		WorkerPoll:                            time.Second,
		VendorRefreshBatchSize:                100,
		VendorRefreshCadence:                  15 * time.Minute,
		VendorRefreshLease:                    time.Minute,
		VendorRefreshDocumentLead:             30 * 24 * time.Hour,
		VendorRefreshFactConfirmationInterval: 365 * 24 * time.Hour,
		ArtifactRoot:                          env("CLEARSIGHT_ARTIFACT_ROOT", "./var/artifacts"),
		MaxArtifactBytes:                      20 << 20,
		CaptureSessionTTL:                     20 * time.Minute,
		CapturePublicBaseURL:                  env("CLEARSIGHT_CAPTURE_PUBLIC_BASE_URL", ""),
		IdentityMode:                          strings.ToLower(env("CLEARSIGHT_IDENTITY_MODE", defaultIdentityMode)),
		IdentityHMACSecret:                    env("CLEARSIGHT_IDENTITY_HMAC_SECRET", ""),
		IdentityMaxSkew:                       2 * time.Minute,
		OIDCIssuer:                            env("CLEARSIGHT_OIDC_ISSUER", ""),
		OIDCClientID:                          env("CLEARSIGHT_OIDC_CLIENT_ID", ""),
		OIDCClientSecret:                      env("CLEARSIGHT_OIDC_CLIENT_SECRET", ""),
		OIDCRedirectURL:                       env("CLEARSIGHT_OIDC_REDIRECT_URL", ""),
		OIDCSessionLifetime:                   8 * time.Hour,
		OIDCSessionIdleTimeout:                30 * time.Minute,
		OIDCSecureCookies:                     production,
		CommandAuthorizationMode:              strings.ToLower(env("CLEARSIGHT_COMMAND_AUTHORIZATION", defaultCommandMode)),
		DemoMode:                              !production,
		DocumentImportAllowUnscannedAnalysis:  !production,
		VendorBrandDiscoveryEnabled:           !production,
		DemoTenantID:                          env("CLEARSIGHT_DEMO_TENANT_ID", "bank-demo"),
		DemoPrincipalID:                       env("CLEARSIGHT_DEMO_PRINCIPAL_ID", "role-cro"),
		DemoLegalEntityID:                     env("CLEARSIGHT_DEMO_LEGAL_ENTITY_ID", "bank-ng"),
		DemoRoleCodes:                         stringList("CLEARSIGHT_DEMO_ROLE_CODES", []string{"CRO", "EXECUTIVE"}),
		LogLevel:                              slog.LevelInfo,
	}
	if cfg.ReleaseSHA != "" && !releaseSHAPattern.MatchString(cfg.ReleaseSHA) {
		return Config{}, fmt.Errorf("CLEARSIGHT_RELEASE_SHA must be a 40-character Git commit SHA")
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
	if cfg.VendorRefreshBatchSize, err = intValue("CLEARSIGHT_VENDOR_REFRESH_BATCH_SIZE", cfg.VendorRefreshBatchSize); err != nil {
		return Config{}, err
	}
	if cfg.VendorRefreshCadence, err = duration("CLEARSIGHT_VENDOR_REFRESH_CADENCE", cfg.VendorRefreshCadence); err != nil {
		return Config{}, err
	}
	if cfg.VendorRefreshLease, err = duration("CLEARSIGHT_VENDOR_REFRESH_LEASE", cfg.VendorRefreshLease); err != nil {
		return Config{}, err
	}
	if cfg.VendorRefreshDocumentLead, err = duration("CLEARSIGHT_VENDOR_REFRESH_DOCUMENT_LEAD", cfg.VendorRefreshDocumentLead); err != nil {
		return Config{}, err
	}
	if cfg.VendorRefreshFactConfirmationInterval, err = duration("CLEARSIGHT_VENDOR_REFRESH_FACT_CONFIRMATION_INTERVAL", cfg.VendorRefreshFactConfirmationInterval); err != nil {
		return Config{}, err
	}
	if cfg.CaptureSessionTTL, err = duration("CLEARSIGHT_CAPTURE_SESSION_TTL", cfg.CaptureSessionTTL); err != nil {
		return Config{}, err
	}
	if cfg.IdentityMaxSkew, err = duration("CLEARSIGHT_IDENTITY_MAX_SKEW", cfg.IdentityMaxSkew); err != nil {
		return Config{}, err
	}
	if cfg.OIDCSessionLifetime, err = duration("CLEARSIGHT_OIDC_SESSION_LIFETIME", cfg.OIDCSessionLifetime); err != nil {
		return Config{}, err
	}
	if cfg.OIDCSessionIdleTimeout, err = duration("CLEARSIGHT_OIDC_SESSION_IDLE_TIMEOUT", cfg.OIDCSessionIdleTimeout); err != nil {
		return Config{}, err
	}
	if cfg.OIDCSecureCookies, err = boolValue("CLEARSIGHT_OIDC_SECURE_COOKIES", cfg.OIDCSecureCookies); err != nil {
		return Config{}, err
	}
	if cfg.DemoMode, err = boolValue("CLEARSIGHT_DEMO_MODE", cfg.DemoMode); err != nil {
		return Config{}, err
	}
	if cfg.DocumentImportAllowUnscannedAnalysis, err = boolValue("CLEARSIGHT_DOCUMENT_IMPORT_ALLOW_UNSCANNED_ANALYSIS", cfg.DocumentImportAllowUnscannedAnalysis); err != nil {
		return Config{}, err
	}
	if cfg.VendorBrandDiscoveryEnabled, err = boolValue("CLEARSIGHT_VENDOR_BRAND_DISCOVERY_ENABLED", cfg.VendorBrandDiscoveryEnabled); err != nil {
		return Config{}, err
	}
	if cfg.RecipientSecurity, err = loadRecipientSecurityConfig(); err != nil {
		return Config{}, err
	}
	if cfg.WorkerPoll <= 0 || cfg.CaptureSessionTTL < time.Minute || cfg.CaptureSessionTTL > time.Hour {
		return Config{}, fmt.Errorf("worker poll must be positive and capture session ttl must be 1-60 minutes")
	}
	if cfg.VendorRefreshBatchSize < 1 || cfg.VendorRefreshBatchSize > 500 || cfg.VendorRefreshCadence < time.Minute || cfg.VendorRefreshCadence > 24*time.Hour || cfg.VendorRefreshLease < time.Second || cfg.VendorRefreshLease > time.Hour || cfg.VendorRefreshDocumentLead < 0 || cfg.VendorRefreshDocumentLead > 365*24*time.Hour || cfg.VendorRefreshFactConfirmationInterval < 24*time.Hour || cfg.VendorRefreshFactConfirmationInterval > 10*365*24*time.Hour {
		return Config{}, fmt.Errorf("vendor refresh maintenance requires batch 1-500, cadence 1 minute-24 hours, lease 1 second-1 hour, document lead 0-365 days and fact confirmation 1 day-10 years")
	}
	if err := validateCapturePublicBaseURL(cfg.CapturePublicBaseURL, environment); err != nil {
		return Config{}, err
	}
	if cfg.RecipientSecurity.ExternalDeliveryEnabled && strings.TrimSpace(cfg.CapturePublicBaseURL) == "" {
		return Config{}, fmt.Errorf("external distribution delivery requires CLEARSIGHT_CAPTURE_PUBLIC_BASE_URL")
	}
	if cfg.IdentityMaxSkew <= 0 || cfg.IdentityMaxSkew > 10*time.Minute {
		return Config{}, fmt.Errorf("identity maximum clock skew must be between 1 second and 10 minutes")
	}
	if cfg.OIDCSessionLifetime < 5*time.Minute || cfg.OIDCSessionLifetime > 24*time.Hour || cfg.OIDCSessionIdleTimeout < time.Minute || cfg.OIDCSessionIdleTimeout > cfg.OIDCSessionLifetime {
		return Config{}, fmt.Errorf("OIDC session lifetime must be 5 minutes-24 hours and idle timeout must be 1 minute-lifetime")
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
	if len(cfg.DemoRoleCodes) > 32 {
		return Config{}, fmt.Errorf("CLEARSIGHT_DEMO_ROLE_CODES supports at most 32 role codes")
	}
	switch cfg.IdentityMode {
	case "development":
	case "signed":
		if len(cfg.IdentityHMACSecret) < 32 {
			return Config{}, fmt.Errorf("CLEARSIGHT_IDENTITY_HMAC_SECRET must contain at least 32 characters in signed identity mode")
		}
	case "oidc":
		if cfg.DatabaseURL == "" {
			return Config{}, fmt.Errorf("OIDC identity mode requires DATABASE_URL")
		}
		if cfg.OIDCIssuer == "" || cfg.OIDCClientID == "" || cfg.OIDCClientSecret == "" || cfg.OIDCRedirectURL == "" {
			return Config{}, fmt.Errorf("OIDC identity mode requires issuer, client id, client secret and redirect URL")
		}
	default:
		return Config{}, fmt.Errorf("CLEARSIGHT_IDENTITY_MODE must be development, signed or oidc")
	}
	switch cfg.CommandAuthorizationMode {
	case "off", "audit", "enforce":
	default:
		return Config{}, fmt.Errorf("CLEARSIGHT_COMMAND_AUTHORIZATION must be off, audit or enforce")
	}
	if production {
		if (cfg.IdentityMode != "signed" && cfg.IdentityMode != "oidc") || cfg.CommandAuthorizationMode != "enforce" {
			return Config{}, fmt.Errorf("production requires signed or OIDC identity and enforced command authorization")
		}
		if cfg.IdentityMode == "oidc" && !cfg.OIDCSecureCookies {
			return Config{}, fmt.Errorf("production OIDC requires secure session cookies")
		}
		if cfg.DemoMode {
			return Config{}, fmt.Errorf("production does not permit CLEARSIGHT_DEMO_MODE=true")
		}
	}
	if strings.EqualFold(env("CLEARSIGHT_LOG_LEVEL", "info"), "debug") {
		cfg.LogLevel = slog.LevelDebug
	}
	return cfg, nil
}

func validateCapturePublicBaseURL(value, environment string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || !parsed.IsAbs() || parsed.User != nil || parsed.Hostname() == "" || parsed.Fragment != "" {
		return fmt.Errorf("CLEARSIGHT_CAPTURE_PUBLIC_BASE_URL must be an absolute secure URL")
	}
	local := strings.EqualFold(parsed.Hostname(), "localhost") || parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "::1"
	if parsed.Scheme != "https" && !(strings.EqualFold(environment, "development") && parsed.Scheme == "http" && local) {
		return fmt.Errorf("CLEARSIGHT_CAPTURE_PUBLIC_BASE_URL must use HTTPS outside local development")
	}
	return nil
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func stringList(name string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return append([]string(nil), fallback...)
	}
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' || r == '|' })
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.ToUpper(strings.TrimSpace(part))
		part = strings.NewReplacer("-", "_", " ", "_").Replace(part)
		if part == "" {
			continue
		}
		if _, exists := seen[part]; exists {
			continue
		}
		seen[part] = struct{}{}
		result = append(result, part)
	}
	return result
}

func duration(name string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := parseDurationValue(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}

func parseDurationValue(value string) (time.Duration, error) {
	if strings.HasSuffix(value, "d") && !strings.ContainsAny(strings.TrimSuffix(value, "d"), ".eE") {
		days, err := strconv.ParseInt(strings.TrimSuffix(value, "d"), 10, 32)
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(value)
}

func boolValue(name string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
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

func intValue(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return parsed, nil
}
