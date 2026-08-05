//go:build postgres

package main

import (
	"context"
	"log/slog"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/database"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

func buildWorker(ctx context.Context, cfg config.Config, logger *slog.Logger) (workerSet, error) {
	pool, err := database.Open(ctx, cfg)
	if err != nil {
		return workerSet{}, err
	}
	repository := workflowruntime.NewPostgresRepository(pool)
	lifecycle := governance.NewPostgresRepository(pool)
	service := workflowruntime.NewService(repository, lifecycle, workflowruntime.LogPublisher{Logger: logger}, cfg.WorkerID)
	evidenceService := evidence.NewService(evidence.NewPostgresRepository(pool), evidence.NewMemoryObjectStore())
	service.AddMaintainer(evidenceService)
	return workerSet{Runtime: service, Close: pool.Close}, nil
}
