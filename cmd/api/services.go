package main

import (
	"context"
	"log/slog"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/aigovernance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentcoverage"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formpolicy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
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
	ThirdPartyAssessmentSetup      *thirdparty.AssessmentProvisioner
	SourceCatalog                  *sourceaccess.CatalogService
	DocumentImports                *documentimport.Service
	Coverage                       *documentcoverage.Service
	Continuity                     *continuity.Service
	Today                          *today.Service
	Oversight                      *oversight.Service
	Workflow                       *workflow.Service
	Onboarding                     *onboarding.Service
	Autonomy                       *autonomy.Service
	AIGovernance                   *aigovernance.Service
	BankVerticals                  *bankverticals.Service
	BackgroundJobs                 *operations.Service
	Access                         access.Resolver
	RuntimeContext                 runtimecontext.Resolver
	AccessAdmin                    access.Administrator
	SessionStore                   scs.Store
	SCIM                           *scimapi.Service
	Close                          func()
}

type monitoringFormRepository interface {
	monitoring.Repository
	monitoring.ReusableFormRepository
}

type serviceBuilder func(context.Context, config.Config, *slog.Logger) (serviceSet, error)
