package formpolicy

import (
	"context"
	"encoding/json"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"testing"
	"time"
)

func TestResultBasisRejectsUnsupportedValue(t *testing.T) {
	input := validPolicyInput("basis-test", RolloutShadow)
	payload, _ := json.Marshal(input)
	var raw map[string]any
	_ = json.Unmarshal(payload, &raw)
	raw["eligibility"].(map[string]any)["result_basis"] = "DRAFT_REVIEW"
	payload, _ = json.Marshal(raw)
	_ = json.Unmarshal(payload, &input)
	if err := normalizeCreateInput(&input, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("unsupported result basis must fail closed")
	}
}

func TestBankAssessedPolicyDoesNotUseAutomaticScore(t *testing.T) {
	input := validPolicyInput("basis-test", RolloutShadow)
	payload, _ := json.Marshal(input.Eligibility)
	var raw map[string]any
	_ = json.Unmarshal(payload, &raw)
	raw["result_basis"] = "BANK_ASSESSED"
	payload, _ = json.Marshal(raw)
	_ = json.Unmarshal(payload, &input.Eligibility)
	response := completedResponse("response-1", "subject-a", 90)
	input.Eligibility.SubjectTypes = []string{response.SubjectType}
	policy := Policy{TenantID: response.TenantID, LegalEntityID: response.LegalEntityID, Eligibility: input.Eligibility}
	if policyMatches(policy, response) {
		t.Fatal("bank assessed policy must not act on automatic submission score")
	}
}

func TestBankPolicyRequiresFinalCompleteMinimumCoverageAssessment(t *testing.T) {
	response := completedResponse("response", "subject", 90)
	policy := Policy{TenantID: response.TenantID, LegalEntityID: response.LegalEntityID, Eligibility: Eligibility{FormTemplateID: response.FormTemplateID, FormTemplateVersion: response.FormTemplateVersion, SubjectTypes: []string{response.SubjectType}, ResultBasis: ResultBankAssessed, MinimumCoverage: 0.8}}
	score := *response.Score
	response.BankAssessment = &evidence.ResponseAssessmentSummary{Version: 1, State: "ASSESSED", RequiredCount: 2, ReviewedRequiredCount: 2, Score: &score}
	if !policyMatches(policy, response) {
		t.Fatal("final assessed result must match")
	}
	response.BankAssessment.State = "IN_REVIEW"
	if policyMatches(policy, response) {
		t.Fatal("provisional review matched")
	}
	response.BankAssessment.State = "ASSESSED"
	response.BankAssessment.ReviewedRequiredCount = 1
	if policyMatches(policy, response) {
		t.Fatal("incomplete assessment matched")
	}
	response.BankAssessment.ReviewedRequiredCount = 2
	score.Coverage = 0.7
	if policyMatches(policy, response) {
		t.Fatal("low coverage matched")
	}
	score.Coverage = 1
	score.Final = false
	if policyMatches(policy, response) {
		t.Fatal("provisional score matched")
	}
}

func (reader executionResponseReaderStub) GetAssessedResponseForExecution(ctx context.Context, tenant, id string) (evidence.CompletedResponseSummary, error) {
	return reader.GetCompletedResponseForExecution(ctx, tenant, id)
}

func TestAutomaticAndBankResultsReuseOneAdverseResponseEpisode(t *testing.T) {
	executor := newExecutorFixture(t, RolloutEnforce)
	repo := executor.store.(*MemoryRepository)
	automatic, err := executor.Handle(t.Context(), scoredEvent("automatic", "response-1"))
	if err != nil || len(automatic) != 1 || !automatic[0].CreatedMatter {
		t.Fatalf("automatic=%#v err=%v", automatic, err)
	}
	later, err := executor.Handle(t.Context(), scoredEvent("automatic-later", "response-2"))
	if err != nil || len(later) != 1 || later[0].MatterID != automatic[0].MatterID {
		t.Fatalf("later=%#v err=%v", later, err)
	}
	policy := repo.policies[policyKey("bank", "entity", "policy-a")]
	policy.ID = "policy-b"
	policy.Code = "bank-reviewed-concern"
	policy.Eligibility.ResultBasis = ResultBankAssessed
	if _, err := repo.CreatePolicy(t.Context(), policy); err != nil {
		t.Fatal(err)
	}
	reader := executor.responses.(executionResponseReaderStub)
	response := reader.responses["response-1"]
	response.BankAssessment = &evidence.ResponseAssessmentSummary{Version: 1, State: "ASSESSED", Score: response.Score, RequiredCount: 1, ReviewedRequiredCount: 1}
	reader.responses["response-1"] = response
	event := scoredEvent("assessed", "response-1")
	event.ResultBasis = ResultBankAssessed
	event.AssessmentVersion = 1
	assessed, err := executor.Handle(t.Context(), event)
	if err != nil || len(assessed) != 1 || assessed[0].CreatedMatter || assessed[0].MatterID != automatic[0].MatterID {
		t.Fatalf("same adverse response duplicated: assessed=%#v err=%v", assessed, err)
	}
	response.BankAssessment.Version = 2
	reader.responses["response-1"] = response
	event.ID = "assessment-correction"
	event.AssessmentVersion = 2
	corrected, err := executor.Handle(t.Context(), event)
	if err != nil || len(corrected) != 1 || corrected[0].CreatedMatter || corrected[0].MatterID != automatic[0].MatterID || corrected[0].AssessmentVersion != 2 {
		t.Fatalf("bank correction duplicated episode: %#v err=%v", corrected, err)
	}

}

func TestManualOnlyFormAllowsOnlyBankAssessedPolicy(t *testing.T) {
	service, _, forms := newPolicyTestService(t)
	forms.form.ScoringMode = formcontract.ScoringNone
	forms.form.Fields = []formcontract.Field{{ID: "review", Assessment: &formcontract.FieldAssessment{Mode: formcontract.AssessmentManual, Weight: 100, Rubric: []formcontract.AssessmentOutcome{{ID: "poor", Points: 90}}}}}
	eligibility := validPolicyInput("manual", RolloutShadow).Eligibility
	actor := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker"}
	if err := service.requireActiveForm(t.Context(), actor, eligibility); err == nil {
		t.Fatal("automatic policy accepted unscored form")
	}
	eligibility.ResultBasis = ResultBankAssessed
	if err := service.requireActiveForm(t.Context(), actor, eligibility); err != nil {
		t.Fatalf("manual-only bank policy rejected: %v", err)
	}
}

func (reader executionResponseReaderStub) WithAssessedResponseForExecution(ctx context.Context, tenant, responseID string, version int64, apply func() error) error {
	response, err := reader.GetCompletedResponseForExecution(ctx, tenant, responseID)
	if err != nil {
		return err
	}
	if !response.Current || response.BankAssessment == nil || response.BankAssessment.Version != version || response.BankAssessment.State != "ASSESSED" {
		return evidence.ErrAssessmentConflict
	}
	return apply()
}
