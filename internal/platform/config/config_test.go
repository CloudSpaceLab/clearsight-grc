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
