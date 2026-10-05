package main

import (
	"context"
	"encoding/json"
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
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oploss"
	"github.com/CloudSpaceLab/clearsight-grc/internal/organization"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/CloudSpaceLab/clearsight-grc/internal/people"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/notificationprefs"
	"github.com/CloudSpaceLab/clearsight-grc/internal/presentationprefs"
	"github.com/CloudSpaceLab/clearsight-grc/internal/rcsa"
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
	RCSA                           *rcsa.Service
	OperationalLoss                *oploss.Service
	ControlCatalog                 *controlcatalog.Service
	MatterFormRemediationRepo      continuity.MatterFormRemediationRepository
	Today                          *today.Service
	Oversight                      *oversight.Service
	GroupOversight                 *oversight.GroupService
	MetricMembership               metricview.MembershipReader
	MetricTrends                   metricview.TrendReader
	MetricMatrices                 metricview.MatrixReader
	DomainMetrics                  metricview.DomainReader
	PresentationPreferences        *presentationprefs.Service
	NotificationPreferences          *notificationprefs.Service
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

type organizationScopeGetter interface {
	Get(context.Context, string, string, string) (organization.Scope, error)
}

func configureOperationalLosses(
	losses *oploss.Service,
	risks *risk.Service,
	matters *continuity.Service,
	organizationScopes organizationScopeGetter,
) {
	if losses == nil {
		return
	}
	losses.ConfigureReferenceValidator(func(ctx context.Context, scope oploss.Scope, value oploss.Loss) error {
		if value.OrganizationScopeID != "" {
			if organizationScopes == nil {
				return oploss.ErrInvalid
			}
			if _, err := organizationScopes.Get(ctx, scope.TenantID, scope.LegalEntityID, value.OrganizationScopeID); err != nil {
				return oploss.ErrInvalid
			}
		}
		if value.RiskID != "" {
			if risks == nil {
				return oploss.ErrInvalid
			}
			if _, err := risks.Get(ctx, risk.Scope{TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID}, value.RiskID); err != nil {
				return oploss.ErrInvalid
			}
		}
		if value.MatterID != "" {
			if matters == nil {
				return oploss.ErrInvalid
			}
			aggregate, err := matters.GetMatter(
				continuity.WithTrustedSystemEntityScope(ctx, scope.TenantID, scope.LegalEntityID),
				scope.TenantID,
				value.MatterID,
			)
			if err != nil || aggregate.Matter.LegalEntityID != scope.LegalEntityID ||
				aggregate.Matter.Type != continuity.MatterOperationalLoss ||
				aggregate.Matter.OrganizationScopeID != value.OrganizationScopeID ||
				aggregate.Matter.SourceType != "OPERATIONAL_LOSS" ||
				aggregate.Matter.SourceID != value.ID ||
				aggregate.Matter.TriggerType != "MATERIAL_OPERATIONAL_LOSS" ||
				aggregate.Matter.TriggerID != value.ID ||
				aggregate.Matter.TriggerKey != "operational-loss:"+value.ID {
				return oploss.ErrInvalid
			}
		}
		return nil
	})
}

const rcsaChallengeDecisionType = "RCSA_CHALLENGE"

var rcsaChallengeOptions = json.RawMessage(`["ACCEPT_FIRST_LINE","REQUIRE_CHANGES","DEFICIENCY_CONFIRMED"]`)

func configureRCSAChallenge(cycles *rcsa.Service, matters *continuity.Service) {
	if cycles == nil {
		return
	}
	cycles.ConfigureChallenge(
		func(ctx context.Context, scope rcsa.Scope, cycle rcsa.Cycle, actorID string) (string, error) {
			if matters == nil {
				return "", rcsa.ErrInvalid
			}
			trusted := continuity.WithTrustedSystemEntityScope(ctx, scope.TenantID, scope.LegalEntityID)
			triggerKey := "rcsa-challenge:" + cycle.ID
			aggregate, err := matters.MatterByTriggerKey(trusted, scope.TenantID, triggerKey)
			if errors.Is(err, continuity.ErrNotFound) {
				scopeJSON, marshalErr := json.Marshal(map[string]any{
					"rcsa_cycle_id":                   cycle.ID,
					"population_checksum":             cycle.PopulationChecksum,
					"first_line_response_revision_id": cycle.FirstLineResponseRevisionID,
				})
				if marshalErr != nil {
					return "", marshalErr
				}
				knownFacts, marshalErr := json.Marshal(map[string]any{
					"rcsa_cycle_id":                   cycle.ID,
					"population_checksum":             cycle.PopulationChecksum,
					"first_line_response_revision_id": cycle.FirstLineResponseRevisionID,
				})
				if marshalErr != nil {
					return "", marshalErr
				}
				aggregate, err = matters.CreateMatter(trusted, continuity.CreateMatterInput{
					TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
					Type: continuity.MatterRiskSituation, Priority: 3,
					Title:   "Challenge " + cycle.Name,
					Summary: "Independent review of the completed first-line RCSA assessment.",
					Scope:   scopeJSON, SourceType: "RCSA_CYCLE", SourceID: cycle.ID,
					TriggerType: "RCSA_CHALLENGE_REQUIRED", TriggerID: cycle.ID, TriggerKey: triggerKey,
					KnownFacts: knownFacts, MissingFacts: json.RawMessage(`[]`), Contradictions: json.RawMessage(`[]`),
					OwnerPrincipalID: actorID, RequiredAuthority: "AUTHORIZER", ActorID: actorID,
				})
				if errors.Is(err, continuity.ErrDuplicate) {
					aggregate, err = matters.MatterByTriggerKey(trusted, scope.TenantID, triggerKey)
				}
			}
			if err != nil || !validRCSAChallengeMatter(aggregate, scope, cycle) {
				return "", rcsa.ErrInvalid
			}
			if continuity.CurrentDecisionForType(aggregate.Decisions, rcsaChallengeDecisionType) == nil {
				aggregate, err = matters.RecordDecisionLifecycle(trusted, continuity.AddDecisionInput{
					TenantID: scope.TenantID, MatterID: aggregate.Matter.ID, ExpectedVersion: aggregate.Matter.Version,
					Type: rcsaChallengeDecisionType, Status: continuity.DecisionProposed,
					Options:    rcsaChallengeOptions,
					Rationale:  "Independent challenge of the submitted first-line RCSA assessment.",
					Conditions: json.RawMessage(`[]`), AuthorityPrincipalID: actorID,
				})
				if err != nil {
					return "", err
				}
			}
			return aggregate.Matter.ID, nil
		},
		func(ctx context.Context, scope rcsa.Scope, cycle rcsa.Cycle) error {
			if matters == nil || cycle.ChallengeMatterID == "" {
				return rcsa.ErrInvalid
			}
			trusted := continuity.WithTrustedSystemEntityScope(ctx, scope.TenantID, scope.LegalEntityID)
			aggregate, err := matters.GetMatter(trusted, scope.TenantID, cycle.ChallengeMatterID)
			if err != nil || !validRCSAChallengeMatter(aggregate, scope, cycle) {
				return rcsa.ErrInvalid
			}
			decision := continuity.CurrentDecisionForType(aggregate.Decisions, rcsaChallengeDecisionType)
			if decision == nil || decision.AuthorityPrincipalID == "" || decision.AuthorityPrincipalID == cycle.FirstLineOwnerID {
				return rcsa.ErrInvalid
			}
			switch decision.Status {
			case continuity.DecisionApproved, continuity.DecisionConditionallyApproved, continuity.DecisionRejected:
			default:
				return rcsa.ErrInvalid
			}
			switch decision.SelectedOption {
			case "ACCEPT_FIRST_LINE", "REQUIRE_CHANGES", "DEFICIENCY_CONFIRMED":
				return nil
			default:
				return rcsa.ErrInvalid
			}
		},
	)
}

func validRCSAChallengeMatter(aggregate continuity.MatterAggregate, scope rcsa.Scope, cycle rcsa.Cycle) bool {
	matter := aggregate.Matter
	if matter.ID == "" || matter.TenantID != cycle.TenantID || matter.LegalEntityID != cycle.LegalEntityID ||
		matter.LegalEntityID != scope.LegalEntityID || matter.Type != continuity.MatterRiskSituation ||
		matter.SourceType != "RCSA_CYCLE" || matter.SourceID != cycle.ID ||
		matter.TriggerType != "RCSA_CHALLENGE_REQUIRED" || matter.TriggerKey != "rcsa-challenge:"+cycle.ID {
		return false
	}
	var metadata struct {
		CycleID                   string `json:"rcsa_cycle_id"`
		PopulationChecksum        string `json:"population_checksum"`
		FirstLineResponseRevision string `json:"first_line_response_revision_id"`
	}
	if err := json.Unmarshal(matter.Scope, &metadata); err != nil {
		return false
	}
	return metadata.CycleID == cycle.ID &&
		metadata.PopulationChecksum == cycle.PopulationChecksum &&
		metadata.FirstLineResponseRevision == cycle.FirstLineResponseRevisionID
}

func configureRCSAFirstLine(cycles *rcsa.Service, distributions *evidence.DistributionService) {
	if cycles == nil {
		return
	}
	cycles.ConfigureFirstLine(
		func(ctx context.Context, scope rcsa.Scope, cycle rcsa.Cycle, distributionID string) error {
			if distributions == nil {
				return rcsa.ErrInvalid
			}
			bundle, err := distributions.Get(ctx, scope.TenantID, scope.LegalEntityID, distributionID)
			if err != nil {
				return rcsa.ErrInvalid
			}
			distribution := bundle.Distribution
			if distribution.ID != distributionID || distribution.SubjectType != "RCSA_CYCLE" ||
				distribution.SubjectID != cycle.ID || distribution.LegalEntityID != scope.LegalEntityID ||
				(distribution.Status != evidence.DistributionOpen && distribution.Status != evidence.DistributionCompleted) {
				return rcsa.ErrInvalid
			}
			for _, recipient := range bundle.Recipients {
				if recipient.Role == evidence.RecipientTo &&
					recipient.Type == evidence.RecipientInternalPrincipal &&
					recipient.PrincipalID == cycle.FirstLineOwnerID &&
					recipient.State != evidence.DistributionRecipientRevoked {
					return nil
				}
			}
			return rcsa.ErrInvalid
		},
		func(ctx context.Context, scope rcsa.Scope, cycle rcsa.Cycle) (string, error) {
			if distributions == nil || cycle.FirstLineDistributionID == "" {
				return "", rcsa.ErrInvalid
			}
			bundle, err := distributions.Get(ctx, scope.TenantID, scope.LegalEntityID, cycle.FirstLineDistributionID)
			if err != nil {
				return "", rcsa.ErrInvalid
			}
			distribution := bundle.Distribution
			if distribution.SubjectType != "RCSA_CYCLE" || distribution.SubjectID != cycle.ID ||
				distribution.Status != evidence.DistributionCompleted {
				return "", rcsa.ErrInvalid
			}
			revisions, err := distributions.ListResponseRevisions(ctx, scope.TenantID, scope.LegalEntityID, distribution.ID, 100)
			if err != nil {
				return "", rcsa.ErrInvalid
			}
			for _, revision := range revisions {
				if revision.Current && revision.State == evidence.ResponseRevisionFinal && revision.ID != "" {
					return revision.ID, nil
				}
			}
			return "", rcsa.ErrInvalid
		},
	)
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
		if err != nil || program.Program.LegalEntityID != scope.LegalEntityID || program.Program.Status != continuity.ProgramActive {
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
