package config

import (
	"strings"
	"testing"
)

func TestLoadAllowsDemoModeToBeDisabledInDevelopment(t *testing.T) {
	t.Setenv("CLEARSIGHT_ENV", "development")
	t.Setenv("CLEARSIGHT_DEMO_MODE", "false")
	t.Setenv("CLEARSIGHT_DOCUMENT_IMPORT_ALLOW_UNSCANNED_ANALYSIS", "false")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DemoMode || cfg.DocumentImportAllowUnscannedAnalysis {
		t.Fatalf("expected explicit development flags to be disabled: %#v", cfg)
	}
}

func TestLoadRejectsDemoModeInProduction(t *testing.T) {
	t.Setenv("CLEARSIGHT_ENV", "production")
	t.Setenv("CLEARSIGHT_IDENTITY_MODE", "signed")
	t.Setenv("CLEARSIGHT_IDENTITY_HMAC_SECRET", strings.Repeat("s", 32))
	t.Setenv("CLEARSIGHT_COMMAND_AUTHORIZATION", "enforce")
	t.Setenv("CLEARSIGHT_DEMO_MODE", "true")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "does not permit") {
		t.Fatalf("expected production demo-mode rejection, got %v", err)
	}
}

func TestProductionDefaultsDisableUnscannedDocumentAnalysis(t *testing.T) {
	t.Setenv("CLEARSIGHT_ENV", "production")
	t.Setenv("CLEARSIGHT_IDENTITY_MODE", "signed")
	t.Setenv("CLEARSIGHT_IDENTITY_HMAC_SECRET", strings.Repeat("s", 32))
	t.Setenv("CLEARSIGHT_COMMAND_AUTHORIZATION", "enforce")
	t.Setenv("CLEARSIGHT_DEMO_MODE", "false")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DocumentImportAllowUnscannedAnalysis {
		t.Fatal("production must default unscanned document analysis to false")
	}
}

func TestProductionDefaultsDisableOutboundVendorBrandDiscovery(t *testing.T) {
	t.Setenv("CLEARSIGHT_ENV", "production")
	t.Setenv("CLEARSIGHT_IDENTITY_MODE", "signed")
	t.Setenv("CLEARSIGHT_IDENTITY_HMAC_SECRET", "01234567890123456789012345678901")
	t.Setenv("CLEARSIGHT_COMMAND_AUTHORIZATION", "enforce")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.VendorBrandDiscoveryEnabled {
		t.Fatal("production enabled outbound vendor brand discovery without an explicit opt-in")
	}
}

func TestProductionAllowsExplicitOutboundVendorBrandDiscoveryOptIn(t *testing.T) {
	t.Setenv("CLEARSIGHT_ENV", "production")
	t.Setenv("CLEARSIGHT_IDENTITY_MODE", "signed")
	t.Setenv("CLEARSIGHT_IDENTITY_HMAC_SECRET", "01234567890123456789012345678901")
	t.Setenv("CLEARSIGHT_COMMAND_AUTHORIZATION", "enforce")
	t.Setenv("CLEARSIGHT_VENDOR_BRAND_DISCOVERY_ENABLED", "true")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.VendorBrandDiscoveryEnabled {
		t.Fatal("production ignored explicit outbound vendor brand discovery opt-in")
	}
}

func TestProductionAllowsOIDCWithSecureServerSessions(t *testing.T) {
	t.Setenv("CLEARSIGHT_ENV", "production")
	t.Setenv("CLEARSIGHT_IDENTITY_MODE", "oidc")
	t.Setenv("CLEARSIGHT_COMMAND_AUTHORIZATION", "enforce")
	t.Setenv("CLEARSIGHT_DEMO_MODE", "false")
	t.Setenv("DATABASE_URL", "postgres://clearsight:clearsight@localhost/clearsight")
	t.Setenv("CLEARSIGHT_ALLOWED_ORIGIN", "https://clearsight.example.test")
	t.Setenv("CLEARSIGHT_OIDC_ISSUER", "https://idp.example.test/tenant")
	t.Setenv("CLEARSIGHT_OIDC_CLIENT_ID", "clearsight")
	t.Setenv("CLEARSIGHT_OIDC_CLIENT_SECRET", "secret")
	t.Setenv("CLEARSIGHT_OIDC_REDIRECT_URL", "https://api.example.test/auth/oidc/callback")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IdentityMode != "oidc" || !cfg.OIDCSecureCookies {
		t.Fatalf("unexpected production OIDC config: %#v", cfg)
	}
}

func TestProductionRejectsInsecureOIDCCookies(t *testing.T) {
	t.Setenv("CLEARSIGHT_ENV", "production")
	t.Setenv("CLEARSIGHT_IDENTITY_MODE", "oidc")
	t.Setenv("CLEARSIGHT_COMMAND_AUTHORIZATION", "enforce")
	t.Setenv("CLEARSIGHT_DEMO_MODE", "false")
	t.Setenv("DATABASE_URL", "postgres://clearsight:clearsight@localhost/clearsight")
	t.Setenv("CLEARSIGHT_OIDC_ISSUER", "https://idp.example.test/tenant")
	t.Setenv("CLEARSIGHT_OIDC_CLIENT_ID", "clearsight")
	t.Setenv("CLEARSIGHT_OIDC_CLIENT_SECRET", "secret")
	t.Setenv("CLEARSIGHT_OIDC_REDIRECT_URL", "https://api.example.test/auth/oidc/callback")
	t.Setenv("CLEARSIGHT_OIDC_SECURE_COOKIES", "false")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "secure session cookies") {
		t.Fatalf("expected insecure OIDC cookie rejection, got %v", err)
	}
}

func TestLoadAcceptsSecureCapturePublicBaseURL(t *testing.T) {
	t.Setenv("CLEARSIGHT_CAPTURE_PUBLIC_BASE_URL", "https://capture.example.test/respond")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CapturePublicBaseURL != "https://capture.example.test/respond" {
		t.Fatalf("unexpected capture base URL %q", cfg.CapturePublicBaseURL)
	}
}

func TestLoadRejectsInsecureCapturePublicBaseURLOutsideLocalDevelopment(t *testing.T) {
	t.Setenv("CLEARSIGHT_ENV", "production")
	t.Setenv("CLEARSIGHT_IDENTITY_MODE", "signed")
	t.Setenv("CLEARSIGHT_IDENTITY_HMAC_SECRET", strings.Repeat("s", 32))
	t.Setenv("CLEARSIGHT_COMMAND_AUTHORIZATION", "enforce")
	t.Setenv("CLEARSIGHT_DEMO_MODE", "false")
	t.Setenv("CLEARSIGHT_CAPTURE_PUBLIC_BASE_URL", "http://capture.example.test/respond")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "CLEARSIGHT_CAPTURE_PUBLIC_BASE_URL") {
		t.Fatalf("expected insecure capture URL rejection, got %v", err)
	}
}
