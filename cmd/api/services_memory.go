//go:build !postgres

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
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
	evidenceService := evidence.NewService(evidence.NewMemoryRepository(evidence.DemoSources(), evidence.DemoRequests()), evidence.NewMemoryObjectStore())
	evidenceService.Configure(cfg.CaptureSessionTTL, cfg.MaxArtifactBytes)
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	if err := continuity.SeedDemo(ctx, continuityService); err != nil {
		return serviceSet{}, err
	}
	return serviceSet{Mode: "memory", Authority: authority.NewResolver(version, rules), Governance: governance.NewService(governance.NewMemoryRepository()), Capture: capture.NewService(capture.DemoRequests()), Invitations: capture.NewInvitationService(time.Now), Evidence: evidenceService, Continuity: continuityService, Today: today.NewService(today.DemoItems()), Workflow: workflow.NewService(workflow.NewMemoryRepository(workflow.DemoTasks())), Onboarding: onboarding.NewService(onboarding.NewMemoryRepository()), Autonomy: auto, Close: func() {}}, nil
}
