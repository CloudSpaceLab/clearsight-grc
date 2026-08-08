//go:build postgres

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/database"
	"github.com/CloudSpaceLab/clearsight-grc/internal/reconciliation"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

const matterWorkProjectionClass = "matter-work-projection"

func buildWorker(ctx context.Context, cfg config.Config, logger *slog.Logger) (workerSet, error) {
	pool, err := database.Open(ctx, cfg)
	if err != nil {
		return workerSet{}, err
	}
	store, err := evidence.NewLocalObjectStore(cfg.ArtifactRoot)
	if err != nil {
		pool.Close()
		return workerSet{}, err
	}
	runtimeRepository := workflowruntime.NewPostgresRepository(pool)
	lifecycle := governance.NewPostgresRepository(pool)
	governanceService := governance.NewService(lifecycle)
	continuityRepository := continuity.NewCurrentPostgresRepository(pool)
	continuityService := continuity.NewService(continuityRepository)
	autonomyService := autonomy.NewService(autonomy.NewPostgresRepository(pool))
	sourceHealth := &reconciliation.SourceHealthConsumer{
		Inbox: runtimeRepository, Dependencies: continuityRepository,
		Signals: autonomyService, Programs: continuityService,
	}
	workflowRepository := workflow.NewPostgresRepository(pool)
	actionWork := &workflow.MatterActionProjector{Repo: workflowRepository}
	lifecycleWork := &workflow.MatterLifecycleProjector{
		Repo: workflowRepository, Continuity: continuityService, Authority: authority.NewEffectivePostgresService(pool), Sequence: governanceService,
	}
	documentService := documentimport.NewService(documentimport.NewPostgresRepository(pool), store)
	documentService.Configure(cfg.MaxArtifactBytes, cfg.DocumentImportAllowUnscannedAnalysis)
	publisher := workflowruntime.NewCompositePublisher(sourceHealth, actionWork, lifecycleWork, documentService, workflowruntime.LogPublisher{Logger: logger})
	service := workflowruntime.NewService(runtimeRepository, lifecycle, publisher, cfg.WorkerID)
	configureWorkerRuntime(service, cfg, logger)
	// Matter events update immediately through the outbox publisher. This slower
	// reconciliation pass exists for restart/backfill and authority/delegation/
	// routing-policy convergence rather than continuously scanning all Matters.
	service.ConfigureClass(matterWorkProjectionClass, workflowruntime.WorkClassOptions{Poll: 30 * time.Second, Batch: 100})

	evidenceService := evidence.NewService(evidence.NewPostgresRepository(pool), store)
	service.AddMaintainerClass(workflowruntime.WorkClassEvidenceMaintenance, evidenceService)
	service.AddMaintainerClass(workflowruntime.WorkClassProgramProjection, &continuity.ProjectionMaintainer{Service: continuityService, Repo: continuityRepository, WorkerID: cfg.WorkerID})
	service.AddMaintainerClass(matterWorkProjectionClass, lifecycleWork)
	return workerSet{Runtime: service, Close: pool.Close}, nil
}
