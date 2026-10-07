package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/database"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/itgovernance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

const acceptanceCommandTimeout = 90 * time.Second

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "indicator source acceptance failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		tenantID       string
		legalEntity    string
		bindingID      string
		branchRef      string
		headOfficeRef  string
		monitoringCheck string
		limit          int
	)
	flag.StringVar(&tenantID, "tenant", "", "Tenant UUID or slug.")
	flag.StringVar(&legalEntity, "legal-entity", "", "Active legal-entity UUID or code.")
	flag.StringVar(&bindingID, "binding", "", "Optional exact active channel-performance Binding UUID.")
	flag.StringVar(&branchRef, "branch-ref", "", "Optional exact branch organization reference that must be observed. The value is never emitted.")
	flag.StringVar(&headOfficeRef, "head-office-ref", "", "Optional exact head-office organization reference that must be observed. The value is never emitted.")
	flag.StringVar(&monitoringCheck, "monitoring-check", "", "Optional Monitoring Check UUID whose latest persisted native result must match the accepted source revision.")
	flag.IntVar(&limit, "limit", sourceaccess.HardMaxPreviewRows, "Maximum live source rows to inspect (1-50).")
	flag.Parse()

	tenantID = strings.TrimSpace(tenantID)
	legalEntity = strings.TrimSpace(legalEntity)
	bindingID = strings.TrimSpace(bindingID)
	monitoringCheck = strings.TrimSpace(monitoringCheck)
	if tenantID == "" || legalEntity == "" {
		return fmt.Errorf("-tenant and -legal-entity are required")
	}
	if limit < 1 || limit > sourceaccess.HardMaxPreviewRows {
		return fmt.Errorf("-limit must be between 1 and %d", sourceaccess.HardMaxPreviewRows)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), acceptanceCommandTimeout)
	defer cancel()

	pool, err := database.Open(ctx, cfg)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	continuityService := continuity.NewService(continuity.NewPostgresRepository(pool))
	canonicalEntity, err := continuityService.ResolveLegalEntity(ctx, tenantID, legalEntity)
	if err != nil {
		return fmt.Errorf("resolve legal entity: %w", err)
	}

	store, err := evidence.NewLocalObjectStore(cfg.ArtifactRoot)
	if err != nil {
		return fmt.Errorf("open artifact store: %w", err)
	}
	evidenceService := evidence.NewService(evidence.NewPostgresRepository(pool), store)
	evidenceService.ConfigureLegalEntityResolver(continuityService)

	documentService := documentimport.NewService(documentimport.NewPostgresRepository(pool), store)
	documentService.Configure(cfg.MaxArtifactBytes, cfg.DocumentImportAllowUnscannedAnalysis)

	catalogRepo := sourceaccess.NewPostgresCatalogRepository(pool)
	adapters := sourceaccess.DefaultCatalogAdapters()
	adapters[sourceaccess.AdapterTabularArtifact] = documentService.SourceAccessAdapter()
	catalogService := sourceaccess.NewCatalogService(catalogRepo, sourceaccess.EnvironmentSecretResolver{}, adapters)

	candidate, err := discoverAcceptanceBinding(ctx, tenantID, canonicalEntity, bindingID, evidenceService, catalogRepo)
	if err != nil {
		return err
	}
	page, err := catalogService.PreviewBinding(ctx, tenantID, candidate.Binding.BindingID, candidate.Binding.Version, sourceaccess.PageRequest{Limit: limit})
	if err != nil {
		return fmt.Errorf("execute bounded source read: %w", err)
	}

	var check *monitoring.MonitoringCheck
	var result *monitoring.MonitoringResult
	if monitoringCheck != "" {
		monitoringRepo := monitoring.NewPostgresRepository(pool)
		loadedCheck, err := monitoringRepo.LatestCheckRevision(ctx, tenantID, monitoringCheck)
		if err != nil {
			return fmt.Errorf("load Monitoring Check: %w", err)
		}
		program, err := continuityService.GetProgram(ctx, tenantID, loadedCheck.ProgramID)
		if err != nil {
			return fmt.Errorf("load Monitoring Check Program: %w", err)
		}
		if program.Program.LegalEntityID != canonicalEntity {
			return fmt.Errorf("Monitoring Check is outside the selected legal entity")
		}
		loadedResult, err := monitoringRepo.LatestResultRevision(ctx, tenantID, loadedCheck.ID, loadedCheck.Version)
		if err != nil {
			return fmt.Errorf("load persisted Monitoring Result: %w", err)
		}
		check, result = &loadedCheck, &loadedResult
	}

	receipt, err := itgovernance.BuildIndicatorSourceAcceptance(itgovernance.IndicatorSourceAcceptanceInput{
		Binding: candidate.Binding,
		View: candidate.View,
		Page: page,
		BranchRef: branchRef,
		HeadOfficeRef: headOfficeRef,
		Check: check,
		Result: result,
		GeneratedAt: time.Now().UTC(),
	})
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(receipt); err != nil {
		return fmt.Errorf("encode acceptance receipt: %w", err)
	}
	return nil
}
