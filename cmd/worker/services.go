package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

const formCommunicationReminderClass = "form-communication-reminders"

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
		thirdparty.VendorRefreshMaintenanceWorkClass,
		workflowruntime.WorkClassThirdPartyVendorBrand,
		workflowruntime.WorkClassThirdPartyVendorBrandCleanup,
		formCommunicationReminderClass,
	} {
		service.ConfigureClass(name, options)
	}
	service.ConfigureClass(formCommunicationReminderClass, workflowruntime.WorkClassOptions{Poll: time.Minute, Batch: 100, Timeout: 20 * time.Second})
	service.ConfigureClass(workflowruntime.WorkClassThirdPartyVendorBrand, vendorBrandWorkClassOptions(cfg.WorkerPoll))
	service.ConfigureClass(workflowruntime.WorkClassThirdPartyVendorBrandCleanup, workflowruntime.WorkClassOptions{Poll: cfg.WorkerPoll, Timeout: 20 * time.Second, Lease: time.Minute, Batch: 25, MaxAttempts: 5, MaxBackoff: 5 * time.Minute})
	policy, refreshOptions := vendorRefreshWorkerSettings(cfg)
	_ = policy
	service.ConfigureClass(thirdparty.VendorRefreshMaintenanceWorkClass, refreshOptions)
	service.ConfigureClass(monitoring.CollectionRenewalWorkClass, workflowruntime.WorkClassOptions{Poll: 5 * time.Second, Batch: 50})
}

func vendorRefreshWorkerSettings(cfg config.Config) (thirdparty.RefreshMaintenancePolicy, workflowruntime.WorkClassOptions) {
	batch, cadence, lease := cfg.VendorRefreshBatchSize, cfg.VendorRefreshCadence, cfg.VendorRefreshLease
	documentLead, factInterval := cfg.VendorRefreshDocumentLead, cfg.VendorRefreshFactConfirmationInterval
	legacyDefaults := batch == 0 && cadence == 0 && lease == 0 && documentLead == 0 && factInterval == 0
	if batch == 0 {
		batch = 100
	}
	if cadence == 0 {
		cadence = 15 * time.Minute
	}
	if lease == 0 {
		lease = time.Minute
	}
	if legacyDefaults {
		documentLead = 30 * 24 * time.Hour
	}
	if factInterval == 0 {
		factInterval = 365 * 24 * time.Hour
	}
	policy := thirdparty.RefreshMaintenancePolicy{BatchSize: batch, Lease: lease, DocumentLead: documentLead, FactConfirmationInterval: factInterval}
	options := workflowruntime.WorkClassOptions{Poll: cadence, Timeout: 20 * time.Second, Lease: lease, Batch: batch, MaxAttempts: 5, MaxBackoff: 5 * time.Minute}
	return policy, options
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
