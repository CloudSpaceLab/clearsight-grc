package main

import (
	"log/slog"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

func configureArtifactScanWorker(runtime *workflowruntime.Service, cfg config.Config, logger *slog.Logger, repo evidence.ArtifactScanRepository, store evidence.ObjectStore) error {
	var scanner evidence.ArtifactScanner
	switch cfg.ArtifactScanner {
	case "", "unavailable":
		logger.Warn("capture artifact scanner is unavailable; uploads remain unavailable for download and acceptance")
	case "clamav":
		adapter, err := evidence.NewClamAVScanner(cfg.ArtifactScannerNetwork, cfg.ArtifactScannerAddress, cfg.ArtifactScannerTimeout)
		if err != nil {
			return err
		}
		scanner = adapter
	default:
		return evidence.ErrArtifactScannerUnavailable
	}
	runtime.ConfigureClass(evidence.ArtifactScanWorkClass, workflowruntime.WorkClassOptions{Poll: 5 * time.Second, Timeout: 30 * time.Second, Lease: time.Minute, Batch: 1})
	runtime.AddMaintainerClass(evidence.ArtifactScanWorkClass, evidence.NewArtifactScanWorker(repo, store, scanner, cfg.WorkerID, cfg.MaxArtifactBytes))
	return nil
}
