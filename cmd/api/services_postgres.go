//go:build postgres

package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
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
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/database"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
	"github.com/CloudSpaceLab/clearsight-grc/internal/scimapi"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceevent"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
	"github.com/alexedwards/scs/pgxstore"
)

func buildServices(ctx context.Context, cfg config.Config, logger *slog.Logger) (serviceSet, error) {
	pool, err := database.Open(ctx, cfg)
	if err != nil {
		return serviceSet{}, err
	}
	authorityService := authority.NewEffectivePostgresService(pool)
	store, err := evidence.NewLocalObjectStore(cfg.ArtifactRoot)
	if err != nil {
		pool.Close()
		return serviceSet{}, err
	}
	autonomyRepo := autonomy.NewPostgresRepository(pool)
	auto := autonomy.NewService(autonomyRepo)
	evidenceRepo := evidence.NewPostgresRepository(pool)
	evidenceService := evidence.NewService(evidenceRepo, store)
	evidenceService.Configure(cfg.CaptureSessionTTL, cfg.MaxArtifactBytes)
	documentService := documentimport.NewService(documentimport.NewPostgresRepository(pool), store)
	documentService.Configure(cfg.MaxArtifactBytes, cfg.DocumentImportAllowUnscannedAnalysis)
	catalogRepo := sourceaccess.NewPostgresCatalogRepository(pool)
	runtimeRepo := runtime.NewPostgresRepository(pool)
	checkpoints := sourceaccess.NewCheckpointService(sourceaccess.NewPostgresCheckpointRepository(pool), runtimeRepo)
	adapters := sourceaccess.DefaultCatalogAdapters()
	adapters[sourceaccess.AdapterTabularArtifact] = documentService.SourceAccessAdapter()
	adapters[sourceaccess.AdapterWebhookEvent] = sourceevent.NewAdapter(runtimeRepo, checkpoints)
	sourceCatalog := sourceaccess.NewCatalogService(catalogRepo, sourceaccess.EnvironmentSecretResolver{}, adapters)
	evidenceService.ConfigureSourceBindings(sourceCatalog)
	monitoringRepo := monitoring.NewPostgresRepository(pool)
	monitoringService := monitoring.NewService(monitoringRepo, evidenceService)
	monitoringService.ConfigureSourceReader(sourceCatalog)
	monitoringService.ConfigureSourceValidator(evidenceService)
	proposalService := monitoring.NewFormProposalService(monitoring.NewPostgresFormProposalStore(pool), documentService, monitoringService)

	keyring, hasKeyring, err := configuredRecipientKeyring(cfg)
	if err != nil {
		pool.Close()
		return serviceSet{}, err
	}
	var distributionStore *evidence.PostgresDistributionStore
	if hasKeyring {
		distributionStore = evidence.NewPostgresDistributionStore(evidenceRepo, keyring)
	} else {
		distributionStore = evidence.NewPostgresDistributionStore(evidenceRepo, nil)
	}
	distributionAccess, err := configuredDistributionAccessService(distributionStore, keyring, hasKeyring, cfg)
	if err != nil {
		pool.Close()
		return serviceSet{}, err
	}
	distributionService := evidence.NewDistributionService(distributionStore)
	formPolicies := formpolicy.NewService(formpolicy.NewPostgresRepository(pool), formDistributionReader{repo: monitoringRepo}, distributionService)
	formPolicies.ConfigureActivationAuthority(formPolicyActivationAuthority{Automation: auto, Authority: authorityService, Subjects: evidence.CanonicalSubjectTypeRegistry{}})
	communicationStore := evidence.NewPostgresCommunicationStore(evidenceRepo)
	communicationService := evidence.NewCommunicationService(communicationStore)
	communicationBrands := evidence.NewCommunicationBrandService(evidence.NewPostgresCommunicationBrandStore(evidenceRepo), store)
	communicationDelivery, err := configuredCommunicationDelivery(cfg)
	if err != nil {
		pool.Close()
		return serviceSet{}, err
	}

	thirdPartyRepo := thirdparty.NewPostgresRepository(pool)
	thirdPartyService := thirdparty.NewService(thirdPartyRepo)
	thirdPartyRelationshipLinks := thirdparty.NewRelationshipLinkService(thirdPartyRepo)
	continuityRepo := continuity.NewReliablePostgresRepository(pool)
	continuityService := continuity.NewService(continuityRepo)
	continuityService.ConfigureEvidenceSourceValidator(evidenceService)
	assessmentSetup := thirdparty.NewAssessmentProvisioner(thirdPartyRepo, continuityService, "postgres-api")
	aiGovernanceRepo := aigovernance.NewPostgresRepository(pool)
	aiGovernanceService := aigovernance.NewService(aiGovernanceRepo, auto, sourceCatalog, continuityService)
	coverageService := documentcoverage.NewService(documentcoverage.NewPostgresRepository(pool), documentService, continuityService)
	verticals := bankverticals.NewService(continuityService, evidenceService)
	verticals.ConfigureMonitoring(monitoringService)
	workflowService := workflow.NewService(workflow.NewPostgresRepository(pool))
	accessAdmin := access.NewPostgresAdministrator(pool)
	backgroundJobs := operations.NewService(continuityRepo, runtimeRepo)
	activityService := activity.NewService(activity.NewPostgresRepository(pool))
	auditExports := activity.NewExportService(activityService, activity.NewPostgresExportRepository(pool), store)
	todayService := actorTodayService(workflowService, continuityService, authorityService, accessAdmin, backgroundJobs)
	oversightService := oversight.NewService(oversight.NewPostgresRepository(pool))
	sessionStore := pgxstore.NewWithConfig(pool, pgxstore.Config{CleanUpInterval: 5 * time.Minute, TableName: "web_sessions"})
	scimService, err := scimapi.New(scimapi.NewPostgresRepository(pool), logger)
	if err != nil {
		sessionStore.StopCleanup()
		pool.Close()
		return serviceSet{}, err
	}
	closeServices := func() {
		sessionStore.StopCleanup()
		pool.Close()
	}
	logger.Info("postgres repositories enabled", "max_connections", cfg.DatabaseMaxConns, "artifact_root", cfg.ArtifactRoot, "demo_mode", cfg.DemoMode)
	return serviceSet{
		Mode: "postgres", Authority: authorityService, Governance: governance.NewService(governance.NewPostgresRepository(pool)),
		Evidence: evidenceService, FormDistributions: distributionService, FormDistributionAccess: distributionAccess,
		FormCommunications: communicationService, FormCommunicationBrands: communicationBrands, FormCommunicationTestDelivery: communicationDelivery,
		FormPolicies: formPolicies,
		ObjectStore: store, Monitoring: monitoringService, FormProposals: proposalService, ThirdParty: thirdPartyService, ThirdPartyBrandRepo: thirdPartyRepo, ThirdPartyRelationshipLinks: thirdPartyRelationshipLinks, ThirdPartyRelationshipLinkRepo: thirdPartyRepo, ThirdPartyWorkRepo: thirdPartyRepo, MonitoringRepo: monitoringRepo, ThirdPartyAssessmentRepo: thirdPartyRepo, ThirdPartyActivationRepo: thirdPartyRepo, ThirdPartyAssessmentSetup: assessmentSetup, SourceCatalog: sourceCatalog, DocumentImports: documentService, Coverage: coverageService, Continuity: continuityService, MatterFormRemediationRepo: continuityRepo, Today: todayService, Oversight: oversightService,
		Workflow: workflowService, Onboarding: onboarding.NewService(onboarding.NewPostgresRepository(pool)),
		Autonomy: auto, AIGovernance: aiGovernanceService, BankVerticals: verticals, BackgroundJobs: backgroundJobs, Activity: activityService, AuditExports: auditExports,
		Access: access.NewPostgresResolver(pool), AccessAdmin: accessAdmin, SessionStore: sessionStore, SCIM: scimService, Close: closeServices,
		RuntimeContext: runtimecontext.NewPostgresResolver(pool),
	}, nil
}
