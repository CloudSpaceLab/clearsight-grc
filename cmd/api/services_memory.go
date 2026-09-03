//go:build !postgres

package main

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/activity"
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
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceevent"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

func buildServices(ctx context.Context, cfg config.Config, _ *slog.Logger) (serviceSet, error) {
	version, rules := authority.DemoPolicySet()
	if !cfg.DemoMode {
		version = "no-demo-policy"
		rules = nil
	}
	authorityService := authority.NewResolver(version, rules)
	autonomyRepo := autonomy.NewMemoryRepository()
	auto := autonomy.NewService(autonomyRepo)
	if cfg.DemoMode {
		autonomy.SeedDemo(ctx, auto)
	}
	store := evidence.NewMemoryObjectStore()
	evidenceRepo := &memoryEvidenceRepository{MemoryRepository: evidence.NewMemoryRepository(nil, nil)}
	evidenceService := evidence.NewService(evidenceRepo, store)
	evidenceService.Configure(cfg.CaptureSessionTTL, cfg.MaxArtifactBytes)
	sourceScopes := []sourceaccess.SourceScope{}
	if cfg.DemoMode {
		for _, source := range evidence.DemoSources() {
			sourceScopes = append(sourceScopes, sourceaccess.SourceScope{TenantID: source.TenantID, SourceID: source.ID})
		}
	}
	documentService := documentimport.NewService(documentimport.NewMemoryRepository(), store)
	documentService.Configure(cfg.MaxArtifactBytes, cfg.DocumentImportAllowUnscannedAnalysis)
	catalogRepo := sourceaccess.NewMemoryCatalogRepository(sourceScopes)
	runtimeRepo := runtime.NewMemoryRepository()
	checkpoints := sourceaccess.NewCheckpointService(sourceaccess.NewMemoryCheckpointRepository(catalogRepo), runtimeRepo)
	adapters := sourceaccess.DefaultCatalogAdapters()
	adapters[sourceaccess.AdapterTabularArtifact] = documentService.SourceAccessAdapter()
	adapters[sourceaccess.AdapterWebhookEvent] = sourceevent.NewAdapter(runtimeRepo, checkpoints)
	sourceCatalog := sourceaccess.NewCatalogService(catalogRepo, sourceaccess.EnvironmentSecretResolver{}, adapters)
	evidenceService.ConfigureSourceBindings(sourceCatalog)
	monitoringRepo := monitoring.NewMemoryRepository()
	monitoringService := monitoring.NewService(monitoringRepo, evidenceService)
	monitoringService.ConfigureSourceReader(sourceCatalog)
	monitoringService.ConfigureSourceValidator(evidenceService)
	proposalService := monitoring.NewFormProposalService(monitoring.NewMemoryFormProposalStore(), documentService, monitoringService)

	keyring, hasKeyring, err := configuredRecipientKeyring(cfg)
	if err != nil {
		return serviceSet{}, err
	}
	var distributionStore *evidence.MemoryDistributionStore
	if hasKeyring {
		distributionStore = evidence.NewMemoryDistributionStore(evidenceRepo.MemoryRepository, formDistributionReader{repo: monitoringRepo}, keyring)
	} else {
		distributionStore = evidence.NewMemoryDistributionStore(evidenceRepo.MemoryRepository, formDistributionReader{repo: monitoringRepo}, nil)
	}
	distributionAccessStore := evidence.NewMemoryDistributionAccessStore(distributionStore)
	distributionAccess, err := configuredDistributionAccessService(distributionAccessStore, keyring, hasKeyring, cfg)
	if err != nil {
		return serviceSet{}, err
	}
	distributionService := evidence.NewDistributionService(distributionStore)
	formPolicies := formpolicy.NewService(formpolicy.NewMemoryRepository(), formDistributionReader{repo: monitoringRepo}, distributionService)
	formPolicies.ConfigureActivationAuthority(formPolicyActivationAuthority{Automation: auto, Authority: authorityService, Subjects: evidence.CanonicalSubjectTypeRegistry{}})
	communicationService := evidence.NewCommunicationService(evidence.NewMemoryCommunicationStore())
	communicationBrands := evidence.NewCommunicationBrandService(evidence.NewMemoryCommunicationBrandStore(), store)
	communicationDelivery, err := configuredCommunicationDelivery(cfg)
	if err != nil {
		return serviceSet{}, err
	}

	thirdPartyRepo := thirdparty.NewMemoryAssessmentRepository()
	thirdPartyService := thirdparty.NewService(thirdPartyRepo)
	thirdPartyRelationshipLinkRepo := thirdparty.RelationshipLinkRepository(thirdPartyRepo)
	thirdPartyRelationshipLinks := thirdparty.NewRelationshipLinkService(thirdPartyRelationshipLinkRepo)
	thirdPartyWorkRepo := thirdparty.NewMemoryVendorWorkRepository()
	continuityRepo := continuity.NewMemoryRepository()
	continuityService := continuity.NewService(continuityRepo)
	evidenceRepo.continuity = continuityService
	continuityService.ConfigureEvidenceSourceValidator(evidenceService)
	assessmentSetup := thirdparty.NewAssessmentProvisioner(thirdPartyRepo, continuityService, "memory-api")
	aiGovernanceRepo := aigovernance.NewMemoryRepository()
	aiGovernanceService := aigovernance.NewService(aiGovernanceRepo, auto, sourceCatalog, continuityService)
	coverageService := documentcoverage.NewService(documentcoverage.NewMemoryRepository(), documentService, continuityService)
	verticals := bankverticals.NewService(continuityService, evidenceService)
	configureReferenceVerticals(verticals, monitoringService)
	if cfg.DemoMode {
		referenceNow := time.Now().UTC()
		verticals.ConfigureReferenceTimeline(func(at time.Time) *continuity.Service {
			service := continuity.NewServiceWithClock(continuityRepo, func() time.Time { return at.UTC() })
			service.ConfigureEvidenceSourceValidator(evidenceService)
			return service
		})
		seed := bankverticals.DemoSeedConfig()
		seed.Now = referenceNow
		if _, err := verticals.InstallSample(ctx, seed); err != nil {
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
	}

	var seededMatters []continuity.MatterAggregate
	if cfg.DemoMode {
		var listErr error
		seededMatters, listErr = continuityService.ListMatters(continuity.WithTrustedSystemScope(ctx), cfg.DemoTenantID, "", 200)
		if listErr != nil {
			return serviceSet{}, listErr
		}
	}
	tasks := workflow.TasksFromMatterAggregates(seededMatters)
	workflowService := workflow.NewService(workflow.NewMemoryRepository(tasks))
	backgroundJobs := operations.NewService(continuityRepo, runtimeRepo)
	activityService := activity.NewService(activity.NewMemoryRepository())
	todayService := actorTodayService(workflowService, continuityService, authorityService, nil, backgroundJobs)
	oversightRepo := oversight.NewMemoryRepository(nil)
	if cfg.DemoMode {
		oversightRepo.Put(oversight.FromMatterAggregates(cfg.DemoTenantID, cfg.DemoLegalEntityID, seededMatters, time.Now().UTC()))
	}
	oversightService := oversight.NewService(oversightRepo)

	return serviceSet{
		Mode: "memory", Authority: authorityService, Governance: governance.NewService(governance.NewMemoryRepository()),
		Evidence: evidenceService, FormDistributions: distributionService, FormDistributionAccess: distributionAccess,
		FormCommunications: communicationService, FormCommunicationBrands: communicationBrands, FormCommunicationTestDelivery: communicationDelivery,
		FormPolicies: formPolicies,
		ObjectStore: store, Monitoring: monitoringService, FormProposals: proposalService, ThirdParty: thirdPartyService, ThirdPartyBrandRepo: thirdPartyRepo, ThirdPartyRelationshipLinks: thirdPartyRelationshipLinks, ThirdPartyRelationshipLinkRepo: thirdPartyRelationshipLinkRepo, ThirdPartyWorkRepo: thirdPartyWorkRepo, MonitoringRepo: monitoringRepo, ThirdPartyAssessmentRepo: thirdPartyRepo, ThirdPartyActivationRepo: thirdparty.NewMemoryActivationRepository(thirdPartyRepo), ThirdPartyAssessmentSetup: assessmentSetup, SourceCatalog: sourceCatalog, DocumentImports: documentService, Coverage: coverageService, Continuity: continuityService, MatterFormRemediationRepo: continuityRepo, Today: todayService, Oversight: oversightService,
		Workflow: workflowService, Onboarding: onboarding.NewService(onboarding.NewMemoryRepository()),
		Autonomy: auto, AIGovernance: aiGovernanceService, BankVerticals: verticals, BackgroundJobs: backgroundJobs, Activity: activityService, Close: func() {},
		RuntimeContext: runtimecontext.IdentifierResolver{},
	}, nil
}

func configureReferenceVerticals(verticals *bankverticals.Service, monitoringService *monitoring.Service) {
	verticals.ConfigureMonitoring(monitoringService)
}

type memoryEvidenceRepository struct {
	*evidence.MemoryRepository
	continuity *continuity.Service
}

func (r *memoryEvidenceRepository) ResolveSubjectScope(ctx context.Context, tenant, subjectType, subjectID string) (evidence.SubjectScope, error) {
	if r == nil || r.continuity == nil {
		return evidence.SubjectScope{}, evidence.ErrSubjectUnsupported
	}
	subjectType = strings.ToUpper(strings.TrimSpace(subjectType))
	system := continuity.WithTrustedSystemScope(ctx)
	switch subjectType {
	case "PROGRAM":
		value, err := r.continuity.GetProgram(system, tenant, subjectID)
		if err != nil {
			return evidence.SubjectScope{}, evidence.ErrSubjectUnsupported
		}
		return evidence.SubjectScope{TenantID: tenant, LegalEntityID: value.Program.LegalEntityID, SubjectType: subjectType, SubjectID: subjectID}, nil
	case "MATTER":
		value, err := r.continuity.GetMatter(system, tenant, subjectID)
		if err != nil {
			return evidence.SubjectScope{}, evidence.ErrSubjectUnsupported
		}
		return evidence.SubjectScope{TenantID: tenant, LegalEntityID: value.Matter.LegalEntityID, SubjectType: subjectType, SubjectID: subjectID}, nil
	default:
		return evidence.SubjectScope{}, evidence.ErrSubjectUnsupported
	}
}

func (r *memoryEvidenceRepository) CanReadSubject(ctx context.Context, tenant, principalID, subjectType, subjectID string) (bool, error) {
	if strings.TrimSpace(principalID) == "" {
		return false, nil
	}
	scope, err := r.ResolveSubjectScope(ctx, tenant, subjectType, subjectID)
	if err != nil {
		return false, err
	}
	if scope.SubjectType == "MATTER" {
		value, readErr := r.continuity.GetMatter(continuity.WithTrustedSystemScope(ctx), tenant, subjectID)
		return readErr == nil && continuity.MatterAggregateVisibleTo(value, principalID), readErr
	}
	return true, nil
}
