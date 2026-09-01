package formpolicy

import (
	"fmt"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestReferenceVendorCertificationJourneyCreatesOneMatterPerAdverseEpisode(t *testing.T) {
	profile := formcontract.ScoreProfile{
		Version: "reference-vendor-certification-v1", Mode: formcontract.ScoringCompliance, Direction: formcontract.DirectionLowIsPoor,
		Contributions: []formcontract.ScoreContribution{{
			ID: "certification-current", Label: "Required certification is current", Weight: 100, Required: true,
			Predicate:   formcontract.Predicate{FieldID: "certification_current", Operator: formcontract.PredicateEquals, Values: []string{"Yes"}},
			MatchPoints: 100, NonMatchPoints: 0, Missing: formcontract.MissingIndeterminate,
		}},
		Rules: []formcontract.ScoreRule{{
			ID: "required-certification-expired", Label: "Required certification is not current",
			Predicate: formcontract.Predicate{FieldID: "certification_current", Operator: formcontract.PredicateEquals, Values: []string{"No"}},
			Effect:    formcontract.RuleEffect{Kind: formcontract.EffectDisqualify},
		}},
		Bands: formcontract.DefaultConcernBands(),
	}
	contract := formcontract.Contract{
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard},
		ScoringMode:  formcontract.ScoringCompliance, ScoreProfile: &profile,
		Sections: []formcontract.Section{{ID: "certifications", Title: "Current certifications"}},
		Fields:   []formcontract.Field{{ID: "certification_current", SectionID: "certifications", Label: "Is the required certification current?", Type: formcontract.TypeYesNo, Required: true}},
	}
	result, err := formcontract.EvaluateScoreProfile(profile, contract, formcontract.TextAnswers(map[string]string{"certification_current": "No"}))
	if err != nil || !result.Final || !result.Disqualified || result.Band != formcontract.ConcernCritical || result.RawScore == nil || result.AdverseScore == nil {
		t.Fatalf("reference score=%#v err=%v", result, err)
	}

	now := time.Date(2026, 9, 1, 14, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	activatedAt := now.Add(-6 * time.Hour)
	policy := Policy{
		ID: "sample-policy", TenantID: "bank", LegalEntityID: "entity", Code: "sample-poor-vendor-certification", Name: "Sample poor vendor certification response", Purpose: "Reference data: create one issue when the current vendor certification response is Critical.",
		ActionClass: ActionClassCreateMatter, AutomationPolicyID: "sample-automation", AutomationPolicyVersion: 1,
		Eligibility: Eligibility{FormTemplateID: "sample-certification-form", FormTemplateVersion: 1, SubjectTypes: []string{"VENDOR_RELATIONSHIP"}, CurrentOnly: true, MinimumCoverage: 1, Bands: []formcontract.ConcernBand{formcontract.ConcernCritical}},
		Action:      MatterAction{Type: "VENDOR_DEFICIENCY", Priority: 4, TitleTemplate: "Review {{form_title}}", SummaryTemplate: "The sample response is {{concern}} concern.", RequestedHandling: "Obtain current certification evidence and record the independent outcome check."},
		BlastRadius: BlastRadius{PerRun: 10, PerDay: 25}, Outcome: OutcomeContract{ExpectedOutcome: "The required vendor certification is current and independently checked.", CheckAfterMinutes: 60, FailureResponse: "REVIEW"},
		Rollout: RolloutEnforce, Status: PolicyActive, MakerID: "sample-maker", CheckerID: "sample-checker", Checksum: "sample-policy-checksum", ActivatedAt: &activatedAt, Version: 1, RecordVersion: 4, CreatedAt: activatedAt.Add(-time.Hour), UpdatedAt: activatedAt,
	}
	if _, err := repo.CreatePolicy(t.Context(), policy); err != nil {
		t.Fatal(err)
	}

	responses := map[string]evidence.CompletedResponseSummary{}
	for index, completedAt := range []time.Time{now.Add(-3 * time.Hour), now.Add(-2 * time.Hour), now.Add(time.Hour)} {
		id := fmt.Sprintf("sample-response-%d", index+1)
		responses[id] = evidence.CompletedResponseSummary{
			ID: id, TenantID: "bank", LegalEntityID: "entity", FormTemplateID: "sample-certification-form", FormTemplateVersion: 1,
			Title: "Sample vendor certification refresh", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "sample-vendor", Revision: int64(index + 1), Current: true, State: evidence.ResponseRevisionFinal,
			Score: &evidence.ResponseScoreResult{Mode: formcontract.ScoringCompliance, Direction: formcontract.DirectionLowIsPoor, RawScore: result.RawScore, AdverseScore: result.AdverseScore, Band: result.Band, Coverage: result.Coverage, Final: result.Final, State: evidence.ResponseScoreFinal}, CompletedAt: completedAt,
		}
	}
	executor := NewExecutor(repo, executionResponseReaderStub{responses: responses}, executionAuthorityStub{route: ExecutionRoute{ServicePrincipalID: "sample-automation-service", OwnerPrincipalID: "sample-owner", ReviewerPrincipalID: "sample-reviewer"}})
	executor.now = func() time.Time { return now }
	next := 0
	executor.newID = func() (string, error) { next++; return fmt.Sprintf("sample-generated-%d", next), nil }

	first, err := executor.Handle(t.Context(), ScoredResponseEvent{ID: "sample-event-1", TenantID: "bank", ResponseRevisionID: "sample-response-1", OccurredAt: responses["sample-response-1"].CompletedAt})
	if err != nil || len(first) != 1 || first[0].State != ExecutionApplied || !first[0].CreatedMatter {
		t.Fatalf("first execution=%#v err=%v", first, err)
	}
	replayed, err := executor.Handle(t.Context(), ScoredResponseEvent{ID: "sample-event-replay", TenantID: "bank", ResponseRevisionID: "sample-response-1", OccurredAt: responses["sample-response-1"].CompletedAt})
	if err != nil || len(replayed) != 1 || replayed[0].ID != first[0].ID || replayed[0].MatterID != first[0].MatterID {
		t.Fatalf("replay=%#v err=%v", replayed, err)
	}
	second, err := executor.Handle(t.Context(), ScoredResponseEvent{ID: "sample-event-2", TenantID: "bank", ResponseRevisionID: "sample-response-2", OccurredAt: responses["sample-response-2"].CompletedAt})
	if err != nil || len(second) != 1 || second[0].State != ExecutionReused || second[0].MatterID != first[0].MatterID || second[0].CreatedMatter {
		t.Fatalf("same episode=%#v err=%v", second, err)
	}

	closedAt := now.Add(30 * time.Minute)
	repo.mu.Lock()
	matter := repo.matters[first[0].MatterID]
	matter.Status, matter.ClosedAt = continuity.MatterClosed, &closedAt
	repo.matters[matter.ID] = matter
	repo.mu.Unlock()
	if processed, err := repo.MaintainOutcomeChecks(t.Context(), "sample-worker", closedAt.Add(2*time.Hour), time.Minute, 10); err != nil || processed != 1 {
		t.Fatalf("outcome maintenance processed=%d err=%v", processed, err)
	}
	executor.now = func() time.Time { return now.Add(2 * time.Hour) }
	third, err := executor.Handle(t.Context(), ScoredResponseEvent{ID: "sample-event-3", TenantID: "bank", ResponseRevisionID: "sample-response-3", OccurredAt: responses["sample-response-3"].CompletedAt})
	if err != nil || len(third) != 1 || !third[0].CreatedMatter || third[0].MatterID == first[0].MatterID {
		t.Fatalf("new episode=%#v err=%v", third, err)
	}
}
