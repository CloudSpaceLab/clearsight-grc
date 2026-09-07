package config

import (
	"testing"
	"time"
)

func TestArtifactScannerRejectsPermissiveConfiguration(t *testing.T) {
	t.Setenv("CLEARSIGHT_ARTIFACT_SCANNER", "always-clean")
	if _, err := Load(); err == nil {
		t.Fatal("unsupported scanner configuration accepted")
	}
}

func TestArtifactScannerConfigurationFailsClosed(t *testing.T) {
	t.Setenv("CLEARSIGHT_ARTIFACT_SCANNER", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ArtifactScanner != "unavailable" {
		t.Fatal("missing scanner enabled inspection")
	}
	t.Setenv("CLEARSIGHT_ARTIFACT_SCANNER", "clamav")
	t.Setenv("CLEARSIGHT_ARTIFACT_SCANNER_ADDRESS", "")
	if _, err := Load(); err == nil {
		t.Fatal("missing daemon address accepted")
	}
	t.Setenv("CLEARSIGHT_ARTIFACT_SCANNER_ADDRESS", "127.0.0.1:3310")
	cfg, err = Load()
	if err != nil || cfg.ArtifactScannerTimeout != 15*time.Second {
		t.Fatalf("config=%+v err=%v", cfg, err)
	}
	t.Setenv("CLEARSIGHT_ARTIFACT_SCANNER_TIMEOUT", "60s")
	if _, err := Load(); err == nil {
		t.Fatal("scanner timeout can exceed worker lease")
	}
}
