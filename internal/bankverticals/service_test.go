package bankverticals

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestSampleJourneysConnectProgramEvidenceDecisionsResponsesAndOutcomeChecks(t *testing.T) {
	ctx := continuity.WithTrustedSystemScope(context.Background())
	now := time.Date(2026, 8, 5, 18, 0, 0, 0, time.UTC)
	continuityRepo := continuity.NewMemoryRepository()
	continuityService := continuity.NewServiceWithClock(continuityRepo, func() time.Time { return now })
	evidenceService := newReferenceEvidenceService(now, "bank-ng")
	service := NewService(continuityService, evidenceService)
	config := DemoSeedConfig()
	config.Now = now

	journeys, err := service.SeedSample(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	maintainer := &continuity.ProjectionMaintainer{Service: continuityService, Repo: continuityRepo, WorkerID: "journey-test", Now: func() time.Time { return now.Add(time.Hour) }}
	for {
		completed, maintainErr := maintainer.Maintain(ctx, now.Add(time.Hour), 20)
		if maintainErr != nil {
			t.Fatal(maintainErr)
		}
		if completed == 0 {
			break
		}
	}
	journeys, err = service.List(ctx, config.TenantID)
	if err != nil {
		t.Fatal(err)
	}
	if len(journeys) != 4 {
		t.Fatalf("expected four reference journeys, got %d", len(journeys))
	}

	ndpa := journeyByCode(t, journeys, JourneyNDPAContinuous)
	if ndpa.ProgramID == "" || ndpa.EvidenceRequestID == "" || ndpa.CompletedSteps != ndpa.TotalSteps || ndpa.Status != string(continuity.ProgramActive) {
		t.Fatalf("NDPA journey is incomplete: %#v", ndpa)
	}
	regulatory := journeyByCode(t, journeys, JourneyRegulatoryChange)
	if regulatory.MatterID == "" || regulatory.CompletedSteps != 4 || regulatory.TotalSteps != 5 || regulatory.Status != string(continuity.MatterActionsInProgress) {
		t.Fatalf("regulatory-change journey should be waiting for implementation and outcome evidence: %#v", regulatory)
	}
	authority := journeyByCode(t, journeys, JourneyAuthorityRequest)
	if !authority.Sensitive || authority.Status != string(continuity.MatterClosed) || authority.CompletedSteps != authority.TotalSteps || len(authority.AllowedPrincipalIDs) < 4 {
		t.Fatalf("protected authority journey is incomplete: %#v", authority)
	}
	finding := journeyByCode(t, journeys, JourneyFindingRemediation)
	if finding.Status != string(continuity.MatterClosed) || finding.CompletedSteps != finding.TotalSteps {
		t.Fatalf("finding did not close through a passed outcome check: %#v", finding)
	}

	again, err := service.SeedSample(ctx, config)
	if err != nil || len(again) != 4 {
		t.Fatalf("sample seeding is not idempotent: len=%d err=%v", len(again), err)
	}
	programs, err := continuityService.ListPrograms(ctx, config.TenantID, 20)
	if err != nil || len(programs) != 1 {
		t.Fatalf("expected one NDPA Program after repeat seed, len=%d err=%v", len(programs), err)
	}
	matters, err := continuityService.ListMatters(ctx, config.TenantID, "", 20)
	if err != nil || len(matters) != 3 {
		t.Fatalf("expected three journey issues after repeat seed, len=%d err=%v", len(matters), err)
	}
}

func TestReferenceProgramEvidenceStartsSupportedWithoutOpeningDuplicateIssues(t *testing.T) {
	ctx := continuity.WithTrustedSystemScope(context.Background())
	now := time.Date(2026, 8, 5, 18, 0, 0, 0, time.UTC)
	continuityService := continuity.NewServiceWithClock(continuity.NewMemoryRepository(), func() time.Time { return now })
	evidenceService := newReferenceEvidenceService(now, "bank-ng")
	service := NewService(continuityService, evidenceService)
	config := normalizeSeedConfig(DemoSeedConfig())
	config.Now = now

	sourceIDs, err := service.seedSources(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	program, err := service.seedNDPAProgram(ctx, config, sourceIDs)
	if err != nil {
		t.Fatal(err)
	}
	contracts := make(map[string]continuity.EvidenceContract, len(program.EvidenceContracts))
	for _, contract := range program.EvidenceContracts {
		contracts[contract.ID] = contract
	}
	if len(program.EvidenceAssessments) != len(contracts) {
		t.Fatalf("reference Program has %d evidence checks but %d current assessments", len(contracts), len(program.EvidenceAssessments))
	}
	for _, assessment := range program.EvidenceAssessments {
		contract := contracts[assessment.ContractID]
		if assessment.Conclusion != continuity.EvidenceSupported || assessment.Coverage < contract.MinimumCoverage {
			t.Fatalf("reference assessment opened with an unsupported result: contract=%s conclusion=%s coverage=%v minimum=%v", contract.Code, assessment.Conclusion, assessment.Coverage, contract.MinimumCoverage)
		}
	}
	matters, err := continuityService.ListMatters(ctx, config.TenantID, "", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(matters) != 0 {
		t.Fatalf("reference Program evidence created %d duplicate issues before the curated journeys were installed", len(matters))
	}
}

func TestTodayItemsExcludeCompletedJourneys(t *testing.T) {
	now := time.Date(2026, 8, 5, 18, 0, 0, 0, time.UTC)
	items := TodayItems([]Journey{
		{Code: JourneyNDPAContinuous, Title: "Nigeria data protection", Summary: "Current evidence needs review.", Status: "ACTIVE", StatusLabel: "Evidence incomplete", NextAction: "Provide three privacy review records", Owner: "Data Protection Office", DueAt: timePointer(now.Add(24 * time.Hour)), SourceNames: []string{"Nigeria Data Protection Act 2023"}},
		{Code: JourneyAuthorityRequest, Title: "Regulator request", Status: "CLOSED", StatusLabel: "Closed", NextAction: "Review record"},
	}, now)
	if len(items) != 1 || items[0].Title != "Provide three privacy review records" {
		t.Fatalf("unexpected Today projection: %#v", items)
	}
}

func journeyByCode(t *testing.T, journeys []Journey, code Code) Journey {
	t.Helper()
	for _, journey := range journeys {
		if journey.Code == code {
			return journey
		}
	}
	t.Fatalf("journey %s not found", code)
	return Journey{}
}
