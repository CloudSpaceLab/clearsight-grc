package main

import (
	"context"
	"log/slog"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/scimapi"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
	"github.com/alexedwards/scs/v2"
)

type serviceSet struct {
	Mode            string
	Authority       authority.Service
	Governance      *governance.Service
	Evidence        *evidence.Service
	DocumentImports *documentimport.Service
	Continuity      *continuity.Service
	Today           *today.Service
	Workflow        *workflow.Service
	Onboarding      *onboarding.Service
	Autonomy        *autonomy.Service
	BankVerticals   *bankverticals.Service
	BackgroundJobs  *operations.Service
	Access          access.Resolver
	SessionStore    scs.Store
	SCIM            *scimapi.Service
	Close           func()
}

type serviceBuilder func(context.Context, config.Config, *slog.Logger) (serviceSet, error)
