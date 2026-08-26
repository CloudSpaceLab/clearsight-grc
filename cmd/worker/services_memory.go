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
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func buildWorker(_ context.Context, cfg config.Config, logger *slog.Logger) (workerSet, error) {
	repository := workflowruntime.NewMemoryRepository()
	lifecycle := governance.NewMemoryRepository()
	continuityRepository := continuity.NewMemoryRepository()
	continuityService := continuity.NewService(continuityRepository)
	evidenceRepository := evidence.NewMemoryRepository(evidence.DemoSources(), evidence.DemoRequests())
	objectStore := evidence.NewMemoryObjectStore()
	evidenceService := evidence.NewService(evidenceRepository, objectStore)
	assessmentRepository := thirdparty.NewMemoryAssessmentRepository()
	assessmentSubmission := newAssessmentSubmissionConsumer(repository, evidenceService, assessmentRepository)
	assessmentCancellation := newAssessmentCancellationConsumer(evidenceService)
	vendorWorkRepository := thirdparty.NewMemoryVendorWorkRepository()
	vendorWorkSubmission := newVendorWorkSubmissionConsumer(repository, evidenceService, vendorWorkRepository)
	publisher := workflowruntime.NewCompositePublisher(assessmentSubmission, assessmentCancellation, vendorWorkSubmission, workflowruntime.LogPublisher{Logger: logger})
	service := workflowruntime.NewService(repository, lifecycle, publisher, cfg.WorkerID)
	configureWorkerRuntime(service, cfg, logger)

	service.AddMaintainerClass(workflowruntime.WorkClassEvidenceMaintenance, evidenceService)
	service.AddMaintainerClass(workflowruntime.WorkClassProgramProjection, &continuity.ProjectionMaintainer{Service: continuityService, Repo: continuityRepository, WorkerID: cfg.WorkerID})
	assessmentProvisioner := thirdparty.NewAssessmentProvisioner(assessmentRepository, continuityService, cfg.WorkerID)
	service.AddMaintainerClass(thirdparty.AssessmentSetupWorkClass, assessmentProvisioner)
	if cfg.VendorBrandDiscoveryEnabled {
		vendorBrandRepository := thirdparty.NewMemoryRepository()
		vendorBrandWorker := thirdparty.NewVendorBrandWorker(vendorBrandRepository, objectStore, thirdparty.NewDefaultVendorBrandDiscoverer(), cfg.WorkerID)
		configureVendorBrandWorker(vendorBrandWorker, cfg.WorkerPoll)
		service.AddMaintainerClass(workflowruntime.WorkClassThirdPartyVendorBrand, vendorBrandWorker)
	}
	return workerSet{Runtime: service, Close: func() {}}, nil
}
