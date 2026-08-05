//go:build postgres

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/capture"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/database"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

func buildServices(ctx context.Context, cfg config.Config, logger *slog.Logger) (serviceSet, error) {
	pool, err := database.Open(ctx, cfg)
	if err != nil {
		return serviceSet{}, err
	}
	store, err := evidence.NewLocalObjectStore(cfg.ArtifactRoot)
	if err != nil {
		pool.Close()
		return serviceSet{}, err
	}
	auto := autonomy.NewService(autonomy.NewPostgresRepository(pool))
	evidenceService := evidence.NewService(evidence.NewPostgresRepository(pool), store)
	evidenceService.Configure(cfg.CaptureSessionTTL, cfg.MaxArtifactBytes)
	logger.Info("postgres repositories enabled", "max_connections", cfg.DatabaseMaxConns, "artifact_root", cfg.ArtifactRoot)
	return serviceSet{Mode: "postgres", Authority: authority.NewPostgresService(pool), Governance: governance.NewService(governance.NewPostgresRepository(pool)), Capture: capture.NewService(capture.DemoRequests()), Invitations: capture.NewInvitationService(time.Now), Evidence: evidenceService, Today: today.NewService(today.DemoItems()), Workflow: workflow.NewService(workflow.NewPostgresRepository(pool)), Onboarding: onboarding.NewService(onboarding.NewPostgresRepository(pool)), Autonomy: auto, Close: pool.Close}, nil
}
