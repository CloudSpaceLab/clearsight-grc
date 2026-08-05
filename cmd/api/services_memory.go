//go:build !postgres

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/capture"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

func buildServices(ctx context.Context, cfg config.Config, _ *slog.Logger) (serviceSet, error) {
	version, rules := authority.DemoPolicySet()
	auto := autonomy.NewService(autonomy.NewMemoryRepository())
	autonomy.SeedDemo(ctx, auto)
	evidenceService := evidence.NewService(evidence.NewMemoryRepository(nil, nil), evidence.NewMemoryObjectStore())
	evidenceService.Configure(cfg.CaptureSessionTTL, cfg.MaxArtifactBytes)
	continuityRepo := continuity.NewMemoryRepository()
	continuityService := continuity.NewService(continuityRepo)
	verticals := bankverticals.NewService(continuityService, evidenceService)
	journeys, err := verticals.SeedSample(ctx, bankverticals.DemoSeedConfig())
	if err != nil {
		return serviceSet{}, err
	}
	maintainer := &continuity.ProjectionMaintainer{Service: continuityService, Repo: continuityRepo, WorkerID: "memory-bank-journeys"}
	for {
		completed, maintainErr := maintainer.Maintain(ctx, time.Now().UTC().Add(time.Hour), 50)
		if maintainErr != nil {
			return serviceSet{}, maintainErr
		}
		if completed == 0 {
			break
		}
	}
	journeys, err = verticals.List(ctx, bankverticals.DemoSeedConfig().TenantID)
	if err != nil {
		return serviceSet{}, err
	}
	return serviceSet{Mode: "memory", Authority: authority.NewResolver(version, rules), Governance: governance.NewService(governance.NewMemoryRepository()), Capture: capture.NewService(capture.DemoRequests()), Invitations: capture.NewInvitationService(time.Now), Evidence: evidenceService, Continuity: continuityService, Today: today.NewService(bankverticals.TodayItems(journeys, time.Now().UTC())), Workflow: workflow.NewService(workflow.NewMemoryRepository(workflow.DemoTasks())), Onboarding: onboarding.NewService(onboarding.NewMemoryRepository()), Autonomy: auto, BankVerticals: verticals, Close: func() {}}, nil
}
