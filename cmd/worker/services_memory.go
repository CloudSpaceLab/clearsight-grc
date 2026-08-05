//go:build !postgres

package main

import (
	"context"
	"log/slog"

	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

func buildWorker(_ context.Context, cfg config.Config, logger *slog.Logger) (workerSet, error) {
	repository := workflowruntime.NewMemoryRepository()
	lifecycle := governance.NewMemoryRepository()
	service := workflowruntime.NewService(repository, lifecycle, workflowruntime.LogPublisher{Logger: logger}, cfg.WorkerID)
	return workerSet{Runtime: service, Close: func() {}}, nil
}
