package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

type workerSet struct {
	Runtime *workflowruntime.Service
	Close   func()
}

type workerBuilder func(context.Context, config.Config, *slog.Logger) (workerSet, error)

func configureWorkerRuntime(service *workflowruntime.Service, cfg config.Config, logger *slog.Logger) {
	service.SetLogger(logger)
	options := workflowruntime.WorkClassOptions{Poll: cfg.WorkerPoll}
	for _, name := range []string{
		workflowruntime.WorkClassEvidenceMaintenance,
		workflowruntime.WorkClassProgramProjection,
		workflowruntime.WorkClassDelegationLifecycle,
		workflowruntime.WorkClassWorkflowTimers,
		workflowruntime.WorkClassOutboxDelivery,
	} {
		service.ConfigureClass(name, options)
	}
	service.ConfigureClass(monitoring.CollectionRenewalWorkClass, workflowruntime.WorkClassOptions{Poll: 5 * time.Second, Batch: 50})
}
