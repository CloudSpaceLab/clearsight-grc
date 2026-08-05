//go:build !postgres

package main

import (
	"context"
	"log/slog"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

func buildWorker(_ context.Context, cfg config.Config, logger *slog.Logger) (workerSet, error) {
	repository := workflowruntime.NewMemoryRepository()
	lifecycle := governance.NewMemoryRepository()
	service := workflowruntime.NewService(repository, lifecycle, workflowruntime.LogPublisher{Logger: logger}, cfg.WorkerID)
	evidenceService := evidence.NewService(evidence.NewMemoryRepository(evidence.DemoSources(), evidence.DemoRequests()), evidence.NewMemoryObjectStore())
	service.AddMaintainer(evidenceService)
	continuityRepository := continuity.NewMemoryRepository()
	continuityService := continuity.NewService(continuityRepository)
	service.AddMaintainer(&continuity.ProjectionMaintainer{Service: continuityService, Repo: continuityRepository, WorkerID: cfg.WorkerID})
	return workerSet{Runtime: service, Close: func() {}}, nil
}
