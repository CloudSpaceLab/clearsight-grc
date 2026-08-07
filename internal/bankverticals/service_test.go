package bankverticals

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func TestSampleJourneysConnectProgramEvidenceDecisionsResponsesAndOutcomeChecks(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 5, 18, 0, 0, 0, time.UTC)
	continuityRepo := continuity.NewMemoryRepository()
	continuityService := continuity.NewServiceWithClock(continuityRepo, func() time.Time { return now })
	evidenceService := evidence.NewServiceWithClock(evidence.NewMemoryRepository(nil, nil), evidence.NewMemoryObjectStore(), func() time.Time { return now })
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

func TestTodayItemsExcludeCompletedJourneys(t *testing.T) {
	now := time.Date(2026, 8, 5, 18, 0, 0, 0, time.UTC)
	items := TodayItems([]Journey{
		{Code: JourneyNDPAContinuous, Title: "Nigeria data protection", Summary: "Current evidence needs review.", Status: "ACTIVE", StatusLabel: "Evidence incomplete", NextAction: "Provide three privacy review records", Owner: "Data Protection Office", DueAt: timePointer(now.Add(24 * time.Hour)), SourceNames: []string{"Nigeria Data Protection Act 2023"}},
		{Code: JourneyAuthorityRequest, Title: "Regulator request", Status: "CLOSED", StatusLabel: "Closed", NextAction: "Review record"},
	}, now)
	if len(items) != 1 {
		t.Fatalf("expected one active item, got %d", len(items))
	}
	if items[0].ActionTargetType != ActionTargetProgram || items[0].ActionTargetID == "" {
		t.Fatalf("expected navigable program action, got %#v", items[0])
	}
}

func TestRedactJourneyForRestrictedActor(t *testing.T) {
	journey := Journey{Code: JourneyAuthorityRequest, Title: "Regulator request", Summary: "Restricted content", Sensitive: true, AllowedPrincipalIDs: []string{"allowed"}, MatterID: "matter-1", OwnerPrincipalID: "owner", SourceNames: []string{"Restricted letter"}, Steps: []Step{{Code: "received", Label: "Restricted", Complete: true}}}
	redacted := RedactJourney(journey, "denied")
	if redacted.Summary != "Restricted regulator-response record." || redacted.MatterID != "" || redacted.OwnerPrincipalID != "" || len(redacted.SourceNames) != 0 || len(redacted.Steps) != 0 {
		t.Fatalf("restricted journey leaked protected details: %#v", redacted)
	}
	visible := RedactJourney(journey, "allowed")
	if visible.MatterID != "matter-1" || visible.Summary != "Restricted content" {
		t.Fatalf("allowed actor was over-redacted: %#v", visible)
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
