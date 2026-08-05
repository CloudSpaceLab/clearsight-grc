package main

import (
	"context"
	"log/slog"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/capture"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

type serviceSet struct {
	Mode        string
	Authority   authority.Service
	Governance  *governance.Service
	Capture     *capture.Service
	Invitations *capture.InvitationService
	Evidence    *evidence.Service
	Today       *today.Service
	Workflow    *workflow.Service
	Onboarding  *onboarding.Service
	Autonomy    *autonomy.Service
	Close       func()
}

type serviceBuilder func(context.Context, config.Config, *slog.Logger) (serviceSet, error)
