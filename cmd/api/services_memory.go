//go:build !postgres

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/capture"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

func buildServices(ctx context.Context, _ config.Config, _ *slog.Logger) (serviceSet, error) {
	version, rules := authority.DemoPolicySet()
	auto := autonomy.NewService(autonomy.NewMemoryRepository())
	autonomy.SeedDemo(ctx, auto)
	return serviceSet{
		Mode:        "memory",
		Authority:   authority.NewResolver(version, rules),
		Governance:  governance.NewService(governance.NewMemoryRepository()),
		Capture:     capture.NewService(capture.DemoRequests()),
		Invitations: capture.NewInvitationService(time.Now),
		Today:       today.NewService(today.DemoItems()),
		Workflow:    workflow.NewService(workflow.NewMemoryRepository(workflow.DemoTasks())),
		Onboarding:  onboarding.NewService(onboarding.NewMemoryRepository()),
		Autonomy:    auto,
		Close:       func() {},
	}, nil
}
