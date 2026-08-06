//go:build postgres

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/capture"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
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
	documentService := documentimport.NewService(documentimport.NewPostgresRepository(pool), store)
	documentService.Configure(cfg.MaxArtifactBytes, cfg.DocumentImportAllowUnscannedAnalysis)
	continuityService := continuity.NewService(continuity.NewPostgresRepository(pool))
	verticals := bankverticals.NewService(continuityService, evidenceService)
	todayService := today.NewDynamicService(func(loadCtx context.Context, actor identity.Actor) ([]today.AttentionItem, error) {
		if !cfg.DemoMode {
			return []today.AttentionItem{}, nil
		}
		journeys, listErr := verticals.List(loadCtx, actor.TenantID)
		if listErr != nil {
			return nil, listErr
		}
		visible := make([]bankverticals.Journey, 0, len(journeys))
		for _, journey := range journeys {
			if journey.VisibleTo(actor.PrincipalID) {
				visible = append(visible, journey)
			}
		}
		return bankverticals.TodayItems(visible, time.Now().UTC()), nil
	})
	requests := capture.DemoRequests()
	if !cfg.DemoMode {
		requests = nil
	}
	logger.Info("postgres repositories enabled", "max_connections", cfg.DatabaseMaxConns, "artifact_root", cfg.ArtifactRoot, "demo_mode", cfg.DemoMode)
	return serviceSet{
		Mode: "postgres", Authority: authority.NewPostgresService(pool), Governance: governance.NewService(governance.NewPostgresRepository(pool)),
		Capture: capture.NewService(requests), Invitations: capture.NewInvitationService(time.Now), Evidence: evidenceService,
		DocumentImports: documentService, Continuity: continuityService, Today: todayService,
		Workflow: workflow.NewService(workflow.NewPostgresRepository(pool)), Onboarding: onboarding.NewService(onboarding.NewPostgresRepository(pool)),
		Autonomy: auto, BankVerticals: verticals, Close: pool.Close,
	}, nil
}
