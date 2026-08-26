package main

import (
	"context"
	"log/slog"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
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
		thirdparty.AssessmentSetupWorkClass,
	} {
		service.ConfigureClass(name, options)
	}
}
