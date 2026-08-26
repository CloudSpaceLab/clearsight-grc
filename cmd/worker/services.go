package main

import (
	"context"
	"log/slog"
	"time"

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
		workflowruntime.WorkClassThirdPartyVendorBrand,
	} {
		service.ConfigureClass(name, options)
	}
	service.ConfigureClass(workflowruntime.WorkClassThirdPartyVendorBrand, vendorBrandWorkClassOptions(cfg.WorkerPoll))
}

func vendorBrandWorkClassOptions(poll time.Duration) workflowruntime.WorkClassOptions {
	return workflowruntime.WorkClassOptions{Poll: poll, Timeout: 20 * time.Second, Lease: time.Minute, MaxBackoff: 5 * time.Minute, Batch: 5, MaxAttempts: 5}
}

func configureVendorBrandWorker(worker *thirdparty.VendorBrandWorker, poll time.Duration) {
	options := vendorBrandWorkClassOptions(poll)
	worker.Configure(options.Lease, options.MaxAttempts, options.MaxBackoff)
}

func newAssessmentSubmissionConsumer(inbox thirdparty.AssessmentSubmissionInbox, requests thirdparty.AssessmentSubmissionRequestReader, assessments thirdparty.AssessmentRepository) *thirdparty.AssessmentConsumer {
	return &thirdparty.AssessmentConsumer{
		Inbox: inbox, Requests: requests, Resolver: assessments,
		Reactions: thirdparty.NewAssessmentService(assessments, nil),
	}
}

func newAssessmentCancellationConsumer(revoker thirdparty.AssessmentCancellationRevoker) *thirdparty.AssessmentCancellationConsumer {
	return thirdparty.NewAssessmentCancellationConsumer(revoker)
}

func newVendorWorkSubmissionConsumer(inbox thirdparty.AssessmentSubmissionInbox, requests thirdparty.AssessmentSubmissionRequestReader, work thirdparty.VendorWorkRepository) *thirdparty.VendorWorkConsumer {
	return &thirdparty.VendorWorkConsumer{Inbox: inbox, Requests: requests, Resolver: work, Reactions: thirdparty.NewVendorWorkSubmissionRecorder(work)}
}
