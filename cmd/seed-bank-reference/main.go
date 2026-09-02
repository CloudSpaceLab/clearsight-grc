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
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/database"
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
	continuityService := continuity.NewService(continuityRepo)
	evidenceRepo := evidence.NewPostgresRepository(pool)
	evidenceService := evidence.NewService(evidenceRepo, evidence.NewMemoryObjectStore())
	monitoringRepo := monitoring.NewPostgresRepository(pool)
	monitoringService := monitoring.NewService(monitoringRepo, evidenceService)
	installer := bankverticals.NewService(continuityService, evidenceService)
	installer.ConfigureMonitoring(monitoringService)
	journeys, err := installer.InstallSample(ctx, seed)
	fatalIf(err)

	maintainer := &continuity.ProjectionMaintainer{Service: continuityService, Repo: continuityRepo, WorkerID: "reference-installer"}
	for {
		completed, maintainErr := maintainer.Maintain(ctx, time.Now().UTC().Add(time.Hour), 100)
		fatalIf(maintainErr)
		if completed == 0 {
			break
		}
	}
	journeys, err = installer.List(ctx, seed.TenantID)
	fatalIf(err)
	scoring, err := seedScoringAcceptanceResponses(ctx, cfg, pool, seed, journeys, monitoringRepo, evidenceRepo)
	fatalIf(err)
	policy, err := seedFormPolicyAcceptance(ctx, cfg, pool, seed, monitoringRepo, evidenceRepo, scoring)
	fatalIf(err)
	fatalIf(json.NewEncoder(os.Stdout).Encode(map[string]any{
		"installed_at": time.Now().UTC(),
		"tenant_id":    seed.TenantID,
		"journeys":     journeys,
		"scoring_acceptance": scoring,
		"form_policy_acceptance": policy,
	}))
}

func fatalIf(err error) {
	if err == nil {
		return
	}
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}