//go:build !postgres

package main

import (
	"context"
	"log/slog"
	"time"

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
	evidenceRepo := evidence.NewMemoryRepository(nil, nil)
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
		distributionStore = evidence.NewMemoryDistributionStore(evidenceRepo, formDistributionReader{repo: monitoringRepo}, keyring)
	} else {
		distributionStore = evidence.NewMemoryDistributionStore(evidenceRepo, formDistributionReader{repo: monitoringRepo}, nil)
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
	continuityService.ConfigureEvidenceSourceValidator(evidenceService)
	assessmentSetup := thirdparty.NewAssessmentProvisioner(thirdPartyRepo, continuityService, "memory-api")
	aiGovernanceRepo := aigovernance.NewMemoryRepository()
	aiGovernanceService := aigovernance.NewService(aiGovernanceRepo, auto, sourceCatalog, continuityService)
	coverageService := documentcoverage.NewService(documentcoverage.NewMemoryRepository(), documentService, continuityService)
	verticals := bankverticals.NewService(continuityService, evidenceService)
	configureReferenceVerticals(verticals, monitoringService)
	if cfg.DemoMode {
		if _, err := verticals.InstallSample(ctx, bankverticals.DemoSeedConfig()); err != nil {
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

	tasks := workflow.DemoTasks()
	if !cfg.DemoMode {
		tasks = nil
	}
	workflowService := workflow.NewService(workflow.NewMemoryRepository(tasks))
	backgroundJobs := operations.NewService(continuityRepo, runtimeRepo)
	todayService := actorTodayService(workflowService, nil, backgroundJobs)
	oversightRepo := oversight.NewMemoryRepository(nil)
	if cfg.DemoMode {
		matters, listErr := continuityService.ListMatters(continuity.WithTrustedSystemScope(ctx), cfg.DemoTenantID, "", 200)
		if listErr != nil {
			return serviceSet{}, listErr
		}
		oversightRepo.Put(oversight.FromMatterAggregates(cfg.DemoTenantID, cfg.DemoLegalEntityID, matters, time.Now().UTC()))
	}
	oversightService := oversight.NewService(oversightRepo)

	return serviceSet{
		Mode: "memory", RuntimeContext: configuredRuntimeContext(cfg), Authority: authorityService, Governance: governance.NewService(governance.NewMemoryRepository()),
		Evidence: evidenceService, FormDistributions: distributionService, FormDistributionAccess: distributionAccess,
		FormCommunications: communicationService, FormCommunicationBrands: communicationBrands, FormCommunicationTestDelivery: communicationDelivery,
		FormPolicies: formPolicies,
		ObjectStore:  store, Monitoring: monitoringService, FormProposals: proposalService, ThirdParty: thirdPartyService, ThirdPartyBrandRepo: thirdPartyRepo, ThirdPartyRelationshipLinks: thirdPartyRelationshipLinks, ThirdPartyRelationshipLinkRepo: thirdPartyRelationshipLinkRepo, ThirdPartyWorkRepo: thirdPartyWorkRepo, MonitoringRepo: monitoringRepo, ThirdPartyAssessmentRepo: thirdPartyRepo, ThirdPartyAssessmentSetup: assessmentSetup, SourceCatalog: sourceCatalog, DocumentImports: documentService, Coverage: coverageService, Continuity: continuityService, Today: todayService, Oversight: oversightService,
		Workflow: workflowService, Onboarding: onboarding.NewService(onboarding.NewMemoryRepository()),
		Autonomy: auto, AIGovernance: aiGovernanceService, BankVerticals: verticals, BackgroundJobs: backgroundJobs, Close: func() {},
	}, nil
}

func configureReferenceVerticals(verticals *bankverticals.Service, monitoringService *monitoring.Service) {
	verticals.ConfigureMonitoring(monitoringService)
}
