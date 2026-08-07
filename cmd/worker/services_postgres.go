//go:build postgres

package main

import (
	"context"
	"log/slog"

	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/database"
	"github.com/CloudSpaceLab/clearsight-grc/internal/reconciliation"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

func buildWorker(ctx context.Context, cfg config.Config, logger *slog.Logger) (workerSet, error) {
	pool, err := database.Open(ctx, cfg)
	if err != nil {
		return workerSet{}, err
	}
	runtimeRepository := workflowruntime.NewPostgresRepository(pool)
	lifecycle := governance.NewPostgresRepository(pool)
	continuityRepository := continuity.NewPostgresRepository(pool)
	continuityService := continuity.NewService(continuityRepository)
	autonomyService := autonomy.NewService(autonomy.NewPostgresRepository(pool))
	sourceHealth := &reconciliation.SourceHealthConsumer{
		Inbox: runtimeRepository, Dependencies: continuityRepository,
		Signals: autonomyService, Programs: continuityService,
	}
	publisher := workflowruntime.NewCompositePublisher(sourceHealth, workflowruntime.LogPublisher{Logger: logger})
	service := workflowruntime.NewService(runtimeRepository, lifecycle, publisher, cfg.WorkerID)
	configureWorkerRuntime(service, cfg, logger)

	evidenceService := evidence.NewService(evidence.NewPostgresRepository(pool), evidence.NewMemoryObjectStore())
	service.AddMaintainerClass(workflowruntime.WorkClassEvidenceMaintenance, evidenceService)
	service.AddMaintainerClass(workflowruntime.WorkClassProgramProjection, &continuity.ProjectionMaintainer{Service: continuityService, Repo: continuityRepository, WorkerID: cfg.WorkerID})
	return workerSet{Runtime: service, Close: pool.Close}, nil
}
