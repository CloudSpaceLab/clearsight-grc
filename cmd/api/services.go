package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/activity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/aigovernance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/controlcatalog"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentcoverage"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formpolicy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/CloudSpaceLab/clearsight-grc/internal/people"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/registermigration"
	"github.com/CloudSpaceLab/clearsight-grc/internal/reporting"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
	"github.com/CloudSpaceLab/clearsight-grc/internal/scimapi"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
	"github.com/alexedwards/scs/v2"
)

type serviceSet struct {
	Mode                           string
	Authority                      authority.Service
	Governance                     *governance.Service
	Evidence                       *evidence.Service
	FormDistributions              *evidence.DistributionService
	FormDistributionAccess         *evidence.DistributionAccessService
	FormCommunications             *evidence.CommunicationService
	FormCommunicationBrands        *evidence.CommunicationBrandService
	FormCommunicationTestDelivery  *evidence.InvitationDeliveryService
	FormPolicies                   *formpolicy.Service
	Monitoring                     *monitoring.Service
	FormProposals                  *monitoring.FormProposalService
	ThirdParty                     *thirdparty.Service
	ThirdPartyBrandRepo            thirdparty.VendorBrandMutationRepository
	ObjectStore                    evidence.ObjectStore
	ThirdPartyRelationshipLinks    *thirdparty.RelationshipLinkService
	ThirdPartyRelationshipLinkRepo thirdparty.RelationshipLinkRepository
	ThirdPartyWorkRepo             thirdparty.VendorWorkRepository
	MonitoringRepo                 monitoringFormRepository
	ThirdPartyAssessmentRepo       thirdparty.AssessmentRepository
	ThirdPartyActivationRepo       thirdparty.ActivationRepository
	ThirdPartyAssessmentSetup      *thirdparty.AssessmentProvisioner
	SourceCatalog                  *sourceaccess.CatalogService
	DocumentImports                *documentimport.Service
	RegisterMigrations             registermigration.Repository
	Coverage                       *documentcoverage.Service
	Continuity                     *continuity.Service
	Ropa                           *ropa.Service
	RopaEventsReader               ropa.Repository
	Reporting                      *reporting.Service
	Risk                           *risk.Service
	ControlCatalog                 *controlcatalog.Service
	MatterFormRemediationRepo      continuity.MatterFormRemediationRepository
	Today                          *today.Service
	Oversight                      *oversight.Service
	Workflow                       *workflow.Service
	Onboarding                     *onboarding.Service
	Autonomy                       *autonomy.Service
	AIGovernance                   *aigovernance.Service
	BankVerticals                  *bankverticals.Service
	BackgroundJobs                 *operations.Service
	Activity                       *activity.Service
	People                         *people.Service
	AuditExports                   *activity.ExportService
	Access                         access.Resolver
	RuntimeContext                 runtimecontext.Resolver
	AccessAdmin                    access.Administrator
	SessionStore                   scs.Store
	SCIM                           *scimapi.Service
	Close                          func()
}

func configureRiskIndicators(risks *risk.Service, checks *monitoring.Service, programs *continuity.Service) {
	if risks == nil {
		return
	}
	risks.ConfigureIndicatorLinkValidator(func(ctx context.Context, scope risk.Scope, checkID string, version int64) (string, error) {
		if checks == nil || programs == nil {
			return "", risk.ErrInvalid
		}
		check, err := checks.Check(ctx, monitoring.Actor{
			TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, PrincipalID: "risk-indicator-validator",
		}, checkID, version)
		if err != nil || check.Status != monitoring.LifecycleActive || !check.IsCurrent {
			return "", risk.ErrInvalid
		}
		program, err := programs.GetProgram(continuity.WithTrustedSystemScope(ctx), scope.TenantID, check.ProgramID)
		if err != nil || program.Program.LegalEntityID != scope.LegalEntityID {
			return "", risk.ErrInvalid
		}
		return check.ProgramID, nil
	})
}

func configureRiskControlCatalog(risks *risk.Service, catalog *controlcatalog.Service) {
	if risks == nil {
		return
	}
	risks.ConfigureControlLinkValidator(func(ctx context.Context, scope risk.Scope, catalogLinkID string) error {
		if catalog == nil {
			return risk.ErrInvalid
		}
		_, err := catalog.GetImplementationLink(ctx, scope.TenantID, scope.LegalEntityID, catalogLinkID)
		switch {
		case err == nil:
			return nil
		case errors.Is(err, controlcatalog.ErrNotFound), errors.Is(err, controlcatalog.ErrInvalid):
			return risk.ErrInvalid
		default:
			return err
		}
	})
}

type monitoringFormRepository interface {
	monitoring.Repository
	monitoring.ReusableFormRepository
}

type serviceBuilder func(context.Context, config.Config, *slog.Logger) (serviceSet, error)
