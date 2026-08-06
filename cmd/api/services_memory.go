//go:build !postgres

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
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

func buildServices(ctx context.Context, cfg config.Config, _ *slog.Logger) (serviceSet, error) {
	version, rules := authority.DemoPolicySet()
	if !cfg.DemoMode {
		version = "no-demo-policy"
		rules = nil
	}
	auto := autonomy.NewService(autonomy.NewMemoryRepository())
	if cfg.DemoMode {
		autonomy.SeedDemo(ctx, auto)
	}
	store := evidence.NewMemoryObjectStore()
	evidenceService := evidence.NewService(evidence.NewMemoryRepository(nil, nil), store)
	evidenceService.Configure(cfg.CaptureSessionTTL, cfg.MaxArtifactBytes)
	documentService := documentimport.NewService(documentimport.NewMemoryRepository(), store)
	documentService.Configure(cfg.MaxArtifactBytes, cfg.DocumentImportAllowUnscannedAnalysis)
	continuityRepo := continuity.NewMemoryRepository()
	continuityService := continuity.NewService(continuityRepo)
	verticals := bankverticals.NewService(continuityService, evidenceService)
	if cfg.DemoMode {
		if _, err := verticals.InstallSample(ctx, bankverticals.DemoSeedConfig()); err != nil {
			return serviceSet{}, err
		}
		maintainer := &continuity.ProjectionMaintainer{Service: continuityService, Repo: continuityRepo, WorkerID: "memory-bank-journeys"}
		for {
			completed, maintainErr := maintainer.Maintain(ctx, time.Now().UTC().Add(time.Hour), 50)
			if maintainErr != nil {
				return serviceSet{}, maintainErr
			}
			if completed == 0 {
				break
			}
		}
	}

	requests := capture.DemoRequests()
	tasks := workflow.DemoTasks()
	if !cfg.DemoMode {
		requests = nil
		tasks = nil
	}
	workflowService := workflow.NewService(workflow.NewMemoryRepository(tasks))
	todayService := today.NewDynamicService(func(loadCtx context.Context, actor identity.Actor) ([]today.AttentionItem, error) {
		if cfg.DemoMode {
			journeys, err := verticals.List(loadCtx, actor.TenantID)
			if err != nil {
				return nil, err
			}
			visible := make([]bankverticals.Journey, 0, len(journeys))
			for _, journey := range journeys {
				if journey.VisibleTo(actor.PrincipalID) {
					visible = append(visible, journey)
				}
			}
			return bankverticals.TodayItems(visible, time.Now().UTC()), nil
		}

		assigned, err := workflowService.List(loadCtx, workflow.ListFilter{TenantID: actor.TenantID, PrincipalID: actor.PrincipalID, Limit: 50})
		if err != nil {
			return nil, err
		}
		return today.FromWorkflowTasks(assigned), nil
	})

	return serviceSet{
		Mode: "memory", Authority: authority.NewResolver(version, rules), Governance: governance.NewService(governance.NewMemoryRepository()),
		Capture: capture.NewService(requests), Invitations: capture.NewInvitationService(time.Now), Evidence: evidenceService,
		DocumentImports: documentService, Continuity: continuityService, Today: todayService,
		Workflow: workflowService, Onboarding: onboarding.NewService(onboarding.NewMemoryRepository()),
		Autonomy: auto, BankVerticals: verticals, Close: func() {},
	}, nil
}
