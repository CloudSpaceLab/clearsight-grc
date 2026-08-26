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

func newAssessmentSubmissionConsumer(inbox thirdparty.AssessmentSubmissionInbox, requests thirdparty.AssessmentSubmissionRequestReader, assessments thirdparty.AssessmentRepository) *thirdparty.AssessmentConsumer {
	return &thirdparty.AssessmentConsumer{
		Inbox: inbox, Requests: requests, Resolver: assessments,
		Reactions: thirdparty.NewAssessmentService(assessments, nil),
	}
}

func newVendorWorkSubmissionConsumer(inbox thirdparty.AssessmentSubmissionInbox, requests thirdparty.AssessmentSubmissionRequestReader, work thirdparty.VendorWorkRepository) *thirdparty.VendorWorkConsumer {
	return &thirdparty.VendorWorkConsumer{Inbox: inbox, Requests: requests, Resolver: work, Reactions: thirdparty.NewVendorWorkSubmissionRecorder(work)}
}
