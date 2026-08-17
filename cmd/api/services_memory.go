//go:build !postgres

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentcoverage"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceevent"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

const todayItemLimit = 50

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
	sourceScopes := []sourceaccess.SourceScope{}
	if cfg.DemoMode {
		for _, source := range evidence.DemoSources() {
			sourceScopes = append(sourceScopes, sourceaccess.SourceScope{TenantID: source.TenantID, SourceID: source.ID})
		}
	}
	documentService := documentimport.NewService(documentimport.NewMemoryRepository(), store)
	documentService.Configure(cfg.MaxArtifactBytes, cfg.DocumentImportAllowUnscannedAnalysis)
	catalogRepo := sourceaccess.NewMemoryCatalogRepository(sourceScopes)
	runtimeRepo := runtime.NewMemoryRepository()
	checkpoints := sourceaccess.NewCheckpointService(sourceaccess.NewMemoryCheckpointRepository(catalogRepo), runtimeRepo)
	adapters := sourceaccess.DefaultCatalogAdapters()
	adapters[sourceaccess.AdapterTabularArtifact] = documentService.SourceAccessAdapter()
	adapters[sourceaccess.AdapterWebhookEvent] = sourceevent.NewAdapter(runtimeRepo, checkpoints)
	sourceCatalog := sourceaccess.NewCatalogService(catalogRepo, sourceaccess.EnvironmentSecretResolver{}, adapters)
	evidenceService.ConfigureSourceBindings(sourceCatalog)
	monitoringService := monitoring.NewService(monitoring.NewMemoryRepository(), evidenceService)
	continuityRepo := continuity.NewMemoryRepository()
	continuityService := continuity.NewService(continuityRepo)
	coverageService := documentcoverage.NewService(documentcoverage.NewMemoryRepository(), documentService, continuityService)
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

	tasks := workflow.DemoTasks()
	if !cfg.DemoMode {
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

		assigned, err := workflowService.List(loadCtx, workflow.ListFilter{
			TenantID: actor.TenantID, PrincipalID: actor.PrincipalID,
			ActiveOnly: true, VisibleMatterWorkOnly: true, Limit: todayItemLimit,
		})
		if err != nil {
			return nil, err
		}
		return today.FromWorkflowTasksForActor(assigned, actor.PrincipalID), nil
	})

	return serviceSet{
		Mode: "memory", Authority: authority.NewResolver(version, rules), Governance: governance.NewService(governance.NewMemoryRepository()),
		Evidence: evidenceService, Monitoring: monitoringService, SourceCatalog: sourceCatalog, DocumentImports: documentService, Coverage: coverageService, Continuity: continuityService, Today: todayService,
		Workflow: workflowService, Onboarding: onboarding.NewService(onboarding.NewMemoryRepository()),
		Autonomy: auto, BankVerticals: verticals, BackgroundJobs: operations.NewService(continuityRepo, runtimeRepo), Close: func() {},
	}, nil
}
