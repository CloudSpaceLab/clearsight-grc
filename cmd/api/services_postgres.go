//go:build postgres

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/database"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

const todayItemLimit = 50

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
	continuityRepo := continuity.NewReliablePostgresRepository(pool)
	continuityService := continuity.NewService(continuityRepo)
	runtimeRepo := runtime.NewPostgresRepository(pool)
	verticals := bankverticals.NewService(continuityService, evidenceService)
	workflowService := workflow.NewService(workflow.NewPostgresRepository(pool))
	todayService := today.NewDynamicService(func(loadCtx context.Context, actor identity.Actor) ([]today.AttentionItem, error) {
		if cfg.DemoMode {
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
		}

		assigned, listErr := workflowService.List(loadCtx, workflow.ListFilter{
			TenantID: actor.TenantID, PrincipalID: actor.PrincipalID,
			ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: todayItemLimit,
		})
		if listErr != nil {
			return nil, listErr
		}
		return today.FromWorkflowTasksForActor(assigned, actor.PrincipalID), nil
	})
	logger.Info("postgres repositories enabled", "max_connections", cfg.DatabaseMaxConns, "artifact_root", cfg.ArtifactRoot, "demo_mode", cfg.DemoMode)
	return serviceSet{
		Mode: "postgres", Authority: authority.NewEffectivePostgresService(pool), Governance: governance.NewService(governance.NewPostgresRepository(pool)),
		Evidence: evidenceService, DocumentImports: documentService, Continuity: continuityService, Today: todayService,
		Workflow: workflowService, Onboarding: onboarding.NewService(onboarding.NewPostgresRepository(pool)),
		Autonomy: auto, BankVerticals: verticals, BackgroundJobs: operations.NewService(continuityRepo, runtimeRepo), Close: pool.Close,
	}, nil
}
