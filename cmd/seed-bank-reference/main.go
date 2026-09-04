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
	flag.StringVar(&seed.TenantID, "tenant", "", "existing tenant UUID or slug")
	flag.StringVar(&seed.LegalEntityID, "legal-entity", "", "existing legal-entity UUID or code")
	flag.StringVar(&seed.BankName, "bank-name", "Reference Bank Nigeria", "display name used only inside reference records")
	flag.StringVar(&seed.ActorID, "actor", "", "principal UUID installing the reference data")
	flag.StringVar(&seed.OwnerPrincipalID, "owner", "", "principal UUID owning the reference work")
	flag.StringVar(&seed.ContributorPrincipalID, "contributor", "", "principal UUID performing reference evidence work")
	flag.StringVar(&seed.ReviewerPrincipalID, "reviewer", "", "independent reviewer principal UUID")
	flag.StringVar(&seed.SignatoryPrincipalID, "signatory", "", "authorized signatory principal UUID")
	flag.Parse()

	cfg, err := config.Load()
	fatalIf(err)
	if strings.EqualFold(cfg.Environment, "production") {
		fatalIf(fmt.Errorf("reference data cannot be installed while CLEARSIGHT_ENV=production"))
	}
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		fatalIf(fmt.Errorf("DATABASE_URL is required"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := database.Open(ctx, cfg)
	fatalIf(err)
	defer pool.Close()

	continuityRepo := continuity.NewPostgresRepository(pool)
	seed.Now = time.Now().UTC()
	continuityService := continuity.NewServiceWithClock(continuityRepo, func() time.Time { return seed.Now })
	evidenceService := evidence.NewService(evidence.NewPostgresRepository(pool), evidence.NewMemoryObjectStore())
	monitoringService := monitoring.NewService(monitoring.NewPostgresRepository(pool), evidenceService)
	thirdPartyService := thirdparty.NewService(thirdparty.NewPostgresRepository(pool))
	installer := bankverticals.NewService(continuityService, evidenceService)
	installer.ConfigureMonitoring(monitoringService)
	installer.ConfigureReferenceTimeline(func(at time.Time) *continuity.Service {
		service := continuity.NewServiceWithClock(continuityRepo, func() time.Time { return at.UTC() })
		service.ConfigureEvidenceSourceValidator(evidenceService)
		return service
	})
	journeys, err := installer.InstallSample(ctx, seed)
	fatalIf(err)
	referenceVendor, err := installer.EnsureReferenceVendor(ctx, seed, thirdPartyService)
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
	journeys, err = installer.List(ctx, seed.TenantID)
	fatalIf(err)
	fatalIf(json.NewEncoder(os.Stdout).Encode(map[string]any{
		"installed_at":                     seed.Now,
		"tenant_id":                        seed.TenantID,
		"journeys":                         journeys,
		"reference_vendor_id":              referenceVendor.Vendor.ID,
		"reference_vendor_relationship_id": referenceVendor.Relationship.ID,
		"oversight_projection":             oversightSnapshot.ProjectionVersion,
		"oversight_generated_at":           oversightSnapshot.GeneratedAt,
		"oversight_population":             oversightSnapshot.Coverage.Population,
		"oversight_resolution_ranges":      len(oversightSnapshot.Estimates),
		"oversight_source_high_water":      oversightSnapshot.SourceHighWater,
	}))
}

func fatalIf(err error) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
