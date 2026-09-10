//go:build postgres

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/database"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func main() {
	var seed bankverticals.SeedConfig
	var documentSamplesOnly bool
	var sourceEmployeesOnly bool
	var cloudspaceRelationshipID string
	var sourceRecordsOnly bool
	var sourceManifestDir string
	flag.StringVar(&sourceManifestDir, "source-manifest-dir", "", "private directory containing the source-record manifests")
	flag.BoolVar(&sourceRecordsOnly, "source-records-only", false, "install the supplied IT, vendor and operational risk captures and linked issues only")
	flag.BoolVar(&sourceEmployeesOnly, "source-employees-only", false, "install named Fidelity and Ops Risk demo employees with scoped performer assignments")
	flag.StringVar(&cloudspaceRelationshipID, "cloudspace-relationship", "", "install only the Cloudspace sample response for this exact existing demo relationship UUID")
	flag.BoolVar(&documentSamplesOnly, "document-samples-only", false, "install only fictional submitted document samples after the normal worker is ready")
	flag.StringVar(&seed.TenantID, "tenant", "", "existing tenant UUID or slug")
	flag.StringVar(&seed.LegalEntityID, "legal-entity", "", "existing legal-entity UUID or code")
	flag.StringVar(&seed.BankName, "bank-name", "Reference Bank Nigeria", "display name used only inside reference records")
	flag.StringVar(&seed.ActorID, "actor", "", "principal UUID installing the reference data")
	flag.StringVar(&seed.OwnerPrincipalID, "owner", "", "principal UUID owning the reference work")
	flag.StringVar(&seed.ContributorPrincipalID, "contributor", "", "principal UUID performing reference evidence work")
	flag.StringVar(&seed.ReviewerPrincipalID, "reviewer", "", "independent reviewer principal UUID")
	flag.StringVar(&seed.SignatoryPrincipalID, "signatory", "", "authorized signatory principal UUID")
	flag.Parse()
	if (sourceRecordsOnly && (sourceEmployeesOnly || documentSamplesOnly || cloudspaceRelationshipID != "")) || (sourceEmployeesOnly && (documentSamplesOnly || cloudspaceRelationshipID != "")) || (documentSamplesOnly && cloudspaceRelationshipID != "") {
		fatalIf(fmt.Errorf("choose one scoped sample operation"))
	}

	cfg, err := config.Load()
	fatalIf(err)
	if documentSamplesOnly && strings.TrimSpace(os.Getenv("CLEARSIGHT_ARTIFACT_ROOT")) == "" {
		fatalIf(fmt.Errorf("document samples require an explicitly configured CLEARSIGHT_ARTIFACT_ROOT shared with the API and worker"))
	}
	if strings.EqualFold(cfg.Environment, "production") {
		fatalIf(fmt.Errorf("reference data cannot be installed while CLEARSIGHT_ENV=production"))
	}
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		fatalIf(fmt.Errorf("DATABASE_URL is required"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	pool, err := database.Open(ctx, cfg)
	fatalIf(err)
	defer pool.Close()
	if sourceRecordsOnly {
		if strings.TrimSpace(sourceManifestDir) == "" {
			fatalIf(fmt.Errorf("source-records-only requires -source-manifest-dir"))
		}
		sourceRecordFiles = os.DirFS(sourceManifestDir)
		receipt, installErr := installSourceRecords(ctx, cfg, pool, seed)
		fatalIf(installErr)
		fatalIf(json.NewEncoder(os.Stdout).Encode(receipt))
		return
	}
	if sourceEmployeesOnly {
		receipt, installErr := seedSourceEmployees(ctx, pool, seed)
		fatalIf(installErr)
		fatalIf(json.NewEncoder(os.Stdout).Encode(receipt))
		return
	}
	if cloudspaceRelationshipID != "" {
		receipt, installErr := installCloudspaceSample(ctx, cfg, pool, seed, cloudspaceRelationshipID)
		fatalIf(installErr)
		fatalIf(json.NewEncoder(os.Stdout).Encode(receipt))
		return
	}
	if documentSamplesOnly {
		receipt, installErr := installDocumentSamples(ctx, cfg, pool, seed)
		fatalIf(installErr)
		fatalIf(json.NewEncoder(os.Stdout).Encode(receipt))
		return
	}

	continuityRepo := continuity.NewPostgresRepository(pool)
	seed.Now = time.Now().UTC()
	continuityService := continuity.NewServiceWithClock(continuityRepo, func() time.Time { return seed.Now })
	evidenceRepo := evidence.NewPostgresRepository(pool)
	evidenceService := evidence.NewService(evidenceRepo, evidence.NewMemoryObjectStore())
	monitoringRepo := monitoring.NewPostgresRepository(pool)
	monitoringService := monitoring.NewService(monitoringRepo, evidenceService)
	thirdPartyService := thirdparty.NewService(thirdparty.NewPostgresRepository(pool))
	installer := bankverticals.NewService(continuityService, evidenceService)
	installer.ConfigureMonitoring(monitoringService)
	installer.ConfigureReferenceTimeline(func(at time.Time) *continuity.Service {
		service := continuity.NewServiceWithClock(continuityRepo, func() time.Time { return at.UTC() })
		service.ConfigureEvidenceSourceValidator(evidenceService)
		return service
	})
	installedJourneys, err := installer.InstallSample(ctx, seed)
	fatalIf(err)
	operatingDemo, err := installer.EnsureOperatingDemo(ctx, seed)
	fatalIf(err)
	programID := referenceProgramID(installedJourneys)
	if programID == "" {
		fatalIf(fmt.Errorf("response-policy acceptance requires an installed Program subject"))
	}
	upgradedVendorForm, upgradeErr := upgradeReferenceVendorForm(ctx, pool, seed, programID)
	fatalIf(upgradeErr)
	if upgradedVendorForm {
		fmt.Fprintln(os.Stderr, "Reference vendor form upgraded through owner submission and independent review; prior requests retain their form revision.")
	}
	operatingVendors, err := installer.EnsureOperatingVendors(ctx, seed, thirdPartyService)
	fatalIf(err)
	referenceVendor := operatingVendors[0]
	formSamples, err := seedOperatingFormSamples(ctx, cfg, pool, seed, operatingDemo, operatingVendors, monitoringRepo, evidenceRepo)
	fatalIf(err)

	maintainer := &continuity.ProjectionMaintainer{Service: continuityService, Repo: continuityRepo, WorkerID: "reference-installer"}
	for {
		completed, maintainErr := maintainer.Maintain(ctx, seed.Now.Add(time.Hour), 100)
		fatalIf(maintainErr)
		if completed == 0 {
			break
		}
	}
	oversightRepository := oversight.NewPostgresRepository(pool)
	oversightMaintainer := &oversight.Maintainer{Repository: oversightRepository}
	_, err = oversightMaintainer.Maintain(ctx, seed.Now.Add(5*time.Minute), 100)
	fatalIf(err)
	oversightSnapshot, err := oversight.NewService(oversightRepository).Get(ctx, oversight.Scope{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID})
	fatalIf(err)
	if len(oversightSnapshot.Estimates) == 0 || oversightSnapshot.Estimates[0].SampleSize < 5 {
		fatalIf(fmt.Errorf("oversight reference cohort did not produce a minimum-five resolution range"))
	}
	if oversightSnapshot.SourceHighWater["matters"].IsZero() || oversightSnapshot.SourceHighWater["continuity_events"].IsZero() {
		fatalIf(fmt.Errorf("oversight snapshot is missing authoritative Matter or event high-water marks"))
	}
	journeys, err := installer.List(ctx, seed.TenantID)
	fatalIf(err)

	fatalIf(installer.EnsureResponsePolicyAcceptanceForm(ctx, seed, programID))
	scoring, err := seedScoringAcceptanceResponses(ctx, cfg, pool, seed, installedJourneys, monitoringRepo, evidenceRepo)
	fatalIf(err)
	fatalIf(ensureAcceptanceExecutionAuthority(ctx, pool, seed))
	policySeed := seed
	policySeed.ActorID = seed.ContributorPrincipalID
	policySeed.SignatoryPrincipalID = seed.ActorID
	policy, err := seedFormPolicyAcceptance(ctx, cfg, pool, policySeed, monitoringRepo, evidenceRepo, scoring)
	fatalIf(err)
	if !policy.ExactlyOnceMatter {
		fatalIf(fmt.Errorf("response-policy acceptance did not prove exactly-once Matter execution"))
	}

	fatalIf(json.NewEncoder(os.Stdout).Encode(map[string]any{
		"installed_at":                     seed.Now,
		"tenant_id":                        seed.TenantID,
		"journeys":                         journeys,
		"reference_vendor_id":              referenceVendor.Vendor.ID,
		"reference_vendor_relationship_id": referenceVendor.Relationship.ID,
		"operating_demo":                   operatingDemo,
		"operating_vendor_count":           len(operatingVendors),
		"operating_form_samples":           formSamples,
		"oversight_projection":             oversightSnapshot.ProjectionVersion,
		"oversight_generated_at":           oversightSnapshot.GeneratedAt,
		"oversight_population":             oversightSnapshot.Coverage.Population,
		"oversight_resolution_ranges":      len(oversightSnapshot.Estimates),
		"oversight_source_high_water":      oversightSnapshot.SourceHighWater,
		"scoring_acceptance":               scoring,
		"form_policy_acceptance":           policy,
	}))
}

func fatalIf(err error) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
