package monitoring

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestResultUsesVerifiedTenantAndAdverseLinkedIssueEligibility(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo, nil)
	now := time.Date(2026, 8, 26, 15, 0, 0, 0, time.UTC)
	result := MonitoringResult{
		ID: "result-1", TenantID: "bank-a", ProgramID: "program-1", MonitoringCheckID: "check-1", MonitoringCheckVersion: 2,
		InputKind: InputSource, InputReferenceID: "receipt-1", InputReferenceVersion: 1,
		Evaluation:  Evaluation{Band: RiskHigh, Coverage: 1, RuleResults: []RuleResult{{FieldID: "status", Outcome: RuleFailed, Points: 60, Reason: "The expected status was not returned."}}},
		EvaluatedAt: now, EvaluatorVersion: "risk-v1", CreatedAt: now,
	}
	if _, err := repo.AppendResult(t.Context(), result); err != nil {
		t.Fatal(err)
	}
	loaded, err := service.Result(t.Context(), Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "reviewer-1"}, result.ID)
	if err != nil || loaded.ID != result.ID {
		t.Fatalf("exact result = %#v, err=%v", loaded, err)
	}
	if _, err = service.Result(t.Context(), Actor{TenantID: "bank-b", LegalEntityID: "entity-b", PrincipalID: "reviewer-1"}, result.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant result error = %v, want not found", err)
	}
	check := MonitoringCheck{
		ID: "check-1", TenantID: "bank-a", ProgramID: "program-1", FailureAction: FailureRecommendMatter,
		MinimumCoverage: 1,
		Lifecycle:       Lifecycle{Status: LifecycleActive, IsCurrent: true, Version: 2},
	}
	if !EligibleForLinkedIssue(check, result) {
		t.Fatal("current adverse result configured to recommend an issue was not eligible")
	}
	coverageFailure := result
	coverageFailure.Evaluation.Band = RiskNotAssessed
	coverageFailure.Evaluation.Coverage = 0.5
	coverageFailure.Evaluation.RuleResults = nil
	if !EligibleForLinkedIssue(check, coverageFailure) {
		t.Fatal("result below the check minimum coverage was not eligible")
	}
	criticalFailure := result
	criticalFailure.Evaluation.Band = RiskModerate
	criticalFailure.Evaluation.RuleResults = []RuleResult{{FieldID: "status", Outcome: RuleFailed, Critical: true}}
	if !EligibleForLinkedIssue(check, criticalFailure) {
		t.Fatal("critical rule failure was not eligible")
	}
	for name, mutate := range map[string]func(*MonitoringCheck, *MonitoringResult){
		"review only": func(c *MonitoringCheck, _ *MonitoringResult) { c.FailureAction = FailureReview },
		"low result":  func(_ *MonitoringCheck, r *MonitoringResult) { r.Evaluation.Band = RiskLow },
		"moderate without critical failure": func(_ *MonitoringCheck, r *MonitoringResult) {
			r.Evaluation.Band = RiskModerate
			r.Evaluation.RuleResults[0].Critical = false
		},
		"not assessed with full coverage": func(_ *MonitoringCheck, r *MonitoringResult) {
			r.Evaluation.Band = RiskNotAssessed
			r.Evaluation.RuleResults[0].Critical = false
		},
		"old check version": func(_ *MonitoringCheck, r *MonitoringResult) { r.MonitoringCheckVersion = 1 },
		"wrong Program":     func(_ *MonitoringCheck, r *MonitoringResult) { r.ProgramID = "program-2" },
		"retired check":     func(c *MonitoringCheck, _ *MonitoringResult) { c.Status = LifecycleRetired },
		"superseded check":  func(c *MonitoringCheck, _ *MonitoringResult) { c.IsCurrent = false },
	} {
		t.Run(name, func(t *testing.T) {
			candidateCheck := check
			candidateResult := result
			candidateResult.Evaluation.RuleResults = append([]RuleResult(nil), result.Evaluation.RuleResults...)
			mutate(&candidateCheck, &candidateResult)
			if EligibleForLinkedIssue(candidateCheck, candidateResult) {
				t.Fatalf("ineligible result was accepted: check=%#v result=%s", candidateCheck, mustJSON(t, candidateResult))
			}
		})
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
