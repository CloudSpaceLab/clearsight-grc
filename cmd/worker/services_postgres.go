//go:build postgres

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentcoverage"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/database"
	"github.com/CloudSpaceLab/clearsight-grc/internal/reconciliation"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceevent"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

const (
	matterWorkProjectionClass   = "matter-work-projection"
	evidenceWorkProjectionClass = "evidence-request-work-projection"
)

func buildWorker(ctx context.Context, cfg config.Config, logger *slog.Logger) (workerSet, error) {
	pool, err := database.Open(ctx, cfg)
	if err != nil {
		return workerSet{}, err
	}
	store, err := evidence.NewLocalObjectStore(cfg.ArtifactRoot)
	if err != nil {
		pool.Close()
		return workerSet{}, err
	}
	runtimeRepository := workflowruntime.NewPostgresRepository(pool)
	sourceCheckpoints := sourceaccess.NewCheckpointService(sourceaccess.NewPostgresCheckpointRepository(pool), runtimeRepository)
	sourceEventCheckpoint := sourceevent.NewCheckpointProjector(runtimeRepository, sourceCheckpoints)
	lifecycle := governance.NewPostgresRepository(pool)
	governanceService := governance.NewService(lifecycle)
	continuityRepository := continuity.NewCurrentPostgresRepository(pool)
	continuityService := continuity.NewService(continuityRepository)
	authorityService := authority.NewEffectivePostgresService(pool)
	autonomyService := autonomy.NewService(autonomy.NewPostgresRepository(pool))
	sourceHealth := &reconciliation.SourceHealthConsumer{
		Inbox: runtimeRepository, Dependencies: continuityRepository,
		Signals: autonomyService, Programs: continuityService,
	}
	workflowRepository := workflow.NewPostgresRepository(pool)
	actionWork := &workflow.MatterActionProjector{Repo: workflowRepository}
	lifecycleWork := &workflow.MatterLifecycleProjector{
		Repo: workflowRepository, Continuity: continuityService, Authority: authorityService, Sequence: governanceService,
	}
	escalationWork := &workflow.MatterEscalationCoordinator{
		Repo: workflowRepository, Runtime: runtimeRepository, Authority: authorityService, Continuity: continuityService,
	}
	evidenceWork := &workflow.EvidenceRequestProjector{Repo: workflowRepository}
	documentService := documentimport.NewService(documentimport.NewPostgresRepository(pool), store)
	documentService.Configure(cfg.MaxArtifactBytes, cfg.DocumentImportAllowUnscannedAnalysis)
	coverageService := documentcoverage.NewService(documentcoverage.NewPostgresRepository(pool), documentService, continuityService)
	evidenceRepository := evidence.NewPostgresRepository(pool)
	evidenceService := evidence.NewService(evidenceRepository, store)
	assessmentRepository := thirdparty.NewPostgresRepository(pool)
	assessmentSubmission := newAssessmentSubmissionConsumer(runtimeRepository, evidenceService, assessmentRepository)
	publisher := workflowruntime.NewCompositePublisher(sourceEventCheckpoint, sourceHealth, actionWork, lifecycleWork, escalationWork, documentService, coverageService, assessmentSubmission, workflowruntime.LogPublisher{Logger: logger})
	service := workflowruntime.NewService(runtimeRepository, lifecycle, publisher, cfg.WorkerID)
	configureWorkerRuntime(service, cfg, logger)
	// Matter events update immediately through the outbox publisher. This slower
	// reconciliation pass exists for restart/backfill and authority/delegation/
	// routing-policy convergence rather than continuously scanning all Matters.
	service.ConfigureClass(matterWorkProjectionClass, workflowruntime.WorkClassOptions{Poll: 30 * time.Second, Batch: 100})
	// Escalation scheduling is another bounded maintainer on the existing runtime.
	// Timer firing and retries continue to use the shared workflow-timer/outbox path.
	service.ConfigureClass(workflow.MatterEscalationWorkClass, workflowruntime.WorkClassOptions{Poll: 5 * time.Second, Batch: 100})
	// Evidence Request assignment is canonical request state rather than an
	// authority route. A short bounded reconciliation pass gives create/
	// reassignment/wrong-recipient/principal-status changes one rebuildable Today
	// projection without adding another event or worker stack.
	service.ConfigureClass(evidenceWorkProjectionClass, workflowruntime.WorkClassOptions{Poll: 5 * time.Second, Batch: 100})

	assessmentProvisioner := thirdparty.NewAssessmentProvisioner(assessmentRepository, continuityService, cfg.WorkerID)
	assessmentProvisioner.ConfigureAuthority(authorityService)
	service.AddMaintainerClass(workflowruntime.WorkClassEvidenceMaintenance, evidenceService)
	service.AddMaintainerClass(workflowruntime.WorkClassProgramProjection, &continuity.ProjectionMaintainer{Service: continuityService, Repo: continuityRepository, WorkerID: cfg.WorkerID})
	service.AddMaintainerClass(thirdparty.AssessmentSetupWorkClass, assessmentProvisioner)
	service.AddMaintainerClass(matterWorkProjectionClass, lifecycleWork)
	service.AddMaintainerClass(workflow.MatterEscalationWorkClass, escalationWork)
	service.AddMaintainerClass(evidenceWorkProjectionClass, evidenceWork)
	return workerSet{Runtime: service, Close: pool.Close}, nil
}
