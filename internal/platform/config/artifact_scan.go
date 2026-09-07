package config

import (
	"fmt"
	"net"
	"path/filepath"
	"time"
)

func loadArtifactScannerConfig(cfg *Config) error {
	cfg.ArtifactScanner = env("CLEARSIGHT_ARTIFACT_SCANNER", "unavailable")
	cfg.ArtifactScannerNetwork = env("CLEARSIGHT_ARTIFACT_SCANNER_NETWORK", "tcp")
	cfg.ArtifactScannerAddress = env("CLEARSIGHT_ARTIFACT_SCANNER_ADDRESS", "")
	var err error
	cfg.ArtifactScannerTimeout, err = duration("CLEARSIGHT_ARTIFACT_SCANNER_TIMEOUT", 15*time.Second)
	if err != nil {
		return err
	}
	if cfg.ArtifactScannerTimeout <= 0 || cfg.ArtifactScannerTimeout > 20*time.Second {
		return fmt.Errorf("CLEARSIGHT_ARTIFACT_SCANNER_TIMEOUT must be positive and at most 20s")
	}
	switch cfg.ArtifactScanner {
	case "unavailable":
		if cfg.ArtifactScannerAddress != "" {
			return fmt.Errorf("CLEARSIGHT_ARTIFACT_SCANNER_ADDRESS requires the clamav scanner")
		}
		return nil
	case "clamav":
		switch cfg.ArtifactScannerNetwork {
		case "tcp":
			host, port, err := net.SplitHostPort(cfg.ArtifactScannerAddress)
			if err != nil || host == "" || port == "" {
				return fmt.Errorf("CLEARSIGHT_ARTIFACT_SCANNER_ADDRESS must be a host:port for clamav")
			}
		case "unix":
			if !filepath.IsAbs(cfg.ArtifactScannerAddress) {
				return fmt.Errorf("CLEARSIGHT_ARTIFACT_SCANNER_ADDRESS must be an absolute socket path")
			}
		default:
			return fmt.Errorf("CLEARSIGHT_ARTIFACT_SCANNER_NETWORK must be tcp or unix")
		}
	default:
		return fmt.Errorf("CLEARSIGHT_ARTIFACT_SCANNER must be unavailable or clamav")
	}
	return nil
}
