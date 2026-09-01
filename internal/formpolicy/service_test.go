package formpolicy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type policyFormReader struct {
	form evidence.DistributionFormRevision
}

func (reader policyFormReader) GetDistributionFormRevision(context.Context, string, string, string, int64) (evidence.DistributionFormRevision, error) {
	return reader.form, nil
}

type policyResponseReader struct {
	items []evidence.CompletedResponseSummary
}

type policyActivationAuthority struct{ err error }

func (validator policyActivationAuthority) ValidatePolicyActivation(context.Context, Actor, Policy) error {
	return validator.err
}

func (reader *policyResponseReader) ListCompletedResponses(_ context.Context, query evidence.CompletedResponseQuery) (evidence.CompletedResponsePage, error) {
	items := make([]evidence.CompletedResponseSummary, 0, len(reader.items))
	for _, item := range reader.items {
		if item.TenantID == query.TenantID && item.LegalEntityID == query.LegalEntityID && item.FormTemplateID == query.FormTemplateID && item.FormTemplateVersion == query.FormTemplateVersion && (!query.CurrentOnly || item.Current) {
			items = append(items, item)
		}
	}
	return evidence.CompletedResponsePage{Items: items}, nil
}

func TestPolicyLifecycleRequiresFreshSimulationAndDistinctChecker(t *testing.T) {
	service, _, _ := newPolicyTestService(t)
	actor := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker"}
	draft := createPolicyFixture(t, service, actor, RolloutShadow)
	preview, err := service.Simulate(context.Background(), actor, draft.ID, draft.RecordVersion)
	if err != nil || preview.EligibleCount != 2 || preview.WouldCreateCount != 1 || preview.BlastSuppressedCount != 1 {
		t.Fatalf("preview = %#v, err = %v", preview, err)
	}
	pending, err := service.Submit(context.Background(), actor, draft.ID, draft.RecordVersion, preview.ID)
	if err != nil || pending.Status != PolicyPendingApproval {
		t.Fatalf("pending = %#v, err = %v", pending, err)
	}
	if _, err := service.Approve(context.Background(), actor, pending.ID, pending.RecordVersion, preview.ID); !errors.Is(err, ErrMakerChecker) {
		t.Fatalf("maker approval err = %v", err)
	}
	checker := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "checker"}
	approved, err := service.Approve(context.Background(), checker, pending.ID, pending.RecordVersion, preview.ID)
	if err != nil || approved.Status != PolicyApproved || approved.CheckerID != "checker" {
		t.Fatalf("approved = %#v, err = %v", approved, err)
	}
	if _, err := service.Activate(context.Background(), actor, approved.ID, approved.RecordVersion); !errors.Is(err, ErrMakerChecker) {
		t.Fatalf("maker activation err = %v", err)
	}
	active, err := service.Activate(context.Background(), checker, approved.ID, approved.RecordVersion)
	if err != nil || active.Status != PolicyActive || active.ActivatedAt == nil {
		t.Fatalf("active = %#v, err = %v", active, err)
	}
}

func TestPolicyActivationFailsClosedWithoutCurrentAuthorityValidation(t *testing.T) {
	service, _, _ := newPolicyTestService(t)
	service.activationAuthority = nil
	maker := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker"}
	checker := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "checker"}
	draft := createPolicyFixture(t, service, maker, RolloutShadow)
	preview, _ := service.Simulate(context.Background(), maker, draft.ID, draft.RecordVersion)
	pending, _ := service.Submit(context.Background(), maker, draft.ID, draft.RecordVersion, preview.ID)
	approved, _ := service.Approve(context.Background(), checker, pending.ID, pending.RecordVersion, preview.ID)
	if _, err := service.Activate(context.Background(), checker, approved.ID, approved.RecordVersion); !errors.Is(err, ErrAuthorityUnavailable) {
		t.Fatalf("activation without current authority err = %v", err)
	}
	service.ConfigureActivationAuthority(policyActivationAuthority{err: ErrActivationAuthority})
	if _, err := service.Activate(context.Background(), checker, approved.ID, approved.RecordVersion); !errors.Is(err, ErrActivationAuthority) {
		t.Fatalf("activation with invalid current route err = %v", err)
	}
}

func TestPolicyRejectsVersionConflictAndInactiveForm(t *testing.T) {
	service, _, forms := newPolicyTestService(t)
	actor := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker"}
	draft := createPolicyFixture(t, service, actor, RolloutShadow)
	if _, err := service.Simulate(context.Background(), actor, draft.ID, draft.RecordVersion+1); !errors.Is(err, ErrConflict) {
		t.Fatalf("version err = %v", err)
	}
	forms.form.Active = false
	if _, err := service.Create(context.Background(), actor, validPolicyInput("inactive", RolloutShadow)); !errors.Is(err, ErrFormInactive) {
		t.Fatalf("inactive form err = %v", err)
	}
}

func TestActivationInvalidatesChangedOrExpiredSimulation(t *testing.T) {
	service, responses, _ := newPolicyTestService(t)
	actor := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker"}
	checker := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "checker"}
	draft := createPolicyFixture(t, service, actor, RolloutShadow)
	preview, _ := service.Simulate(context.Background(), actor, draft.ID, draft.RecordVersion)
	pending, _ := service.Submit(context.Background(), actor, draft.ID, draft.RecordVersion, preview.ID)
	approved, _ := service.Approve(context.Background(), checker, pending.ID, pending.RecordVersion, preview.ID)
	responses.items = append(responses.items, scoredResponse("response-new", 10, 90, formcontract.ConcernCritical, time.Date(2026, 9, 1, 11, 0, 0, 0, time.UTC)))
	if _, err := service.Activate(context.Background(), checker, approved.ID, approved.RecordVersion); !errors.Is(err, ErrPreviewStale) {
		t.Fatalf("changed population err = %v", err)
	}

	service, _, _ = newPolicyTestService(t)
	draft = createPolicyFixture(t, service, actor, RolloutShadow)
	preview, _ = service.Simulate(context.Background(), actor, draft.ID, draft.RecordVersion)
	pending, _ = service.Submit(context.Background(), actor, draft.ID, draft.RecordVersion, preview.ID)
	approved, _ = service.Approve(context.Background(), checker, pending.ID, pending.RecordVersion, preview.ID)
	service.now = func() time.Time { return time.Date(2026, 9, 2, 11, 0, 1, 0, time.UTC) }
	if _, err := service.Activate(context.Background(), checker, approved.ID, approved.RecordVersion); !errors.Is(err, ErrPreviewStale) {
		t.Fatalf("expired preview err = %v", err)
	}
}

func TestSubmitRejectsSimulationWithChangedPolicyChecksum(t *testing.T) {
	service, _, _ := newPolicyTestService(t)
	repo := service.repo.(*MemoryRepository)
	actor := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker"}
	draft := createPolicyFixture(t, service, actor, RolloutShadow)
	preview, _ := service.Simulate(context.Background(), actor, draft.ID, draft.RecordVersion)
	key := policyKey(preview.TenantID, preview.LegalEntityID, preview.ID)
	repo.mu.Lock()
	changed := repo.simulations[key]
	changed.PolicyChecksum = "changed-after-simulation"
	repo.simulations[key] = changed
	repo.mu.Unlock()
	if _, err := service.Submit(context.Background(), actor, draft.ID, draft.RecordVersion, preview.ID); !errors.Is(err, ErrPreviewStale) {
		t.Fatalf("checksum err = %v", err)
	}
}

func TestSimulationAppliesScoreDirectionPredicates(t *testing.T) {
	service, _, _ := newPolicyTestService(t)
	actor := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker"}
	input := validPolicyInput("low-compliance", RolloutShadow)
	below := 30.0
	adverse := 70.0
	input.Eligibility.RawBelow = &below
	input.Eligibility.AdverseAtLeast = &adverse
	draft, err := service.Create(context.Background(), actor, input)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := service.Simulate(context.Background(), actor, draft.ID, draft.RecordVersion)
	if err != nil || preview.PopulationCount != 3 || preview.EligibleCount != 1 {
		t.Fatalf("preview = %#v, err = %v", preview, err)
	}
}

func TestEnforceActivationRequiresPriorShadowHistory(t *testing.T) {
	service, _, _ := newPolicyTestService(t)
	maker := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker"}
	checker := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "checker"}
	draft := createPolicyFixture(t, service, maker, RolloutEnforce)
	preview, _ := service.Simulate(context.Background(), maker, draft.ID, draft.RecordVersion)
	pending, _ := service.Submit(context.Background(), maker, draft.ID, draft.RecordVersion, preview.ID)
	approved, _ := service.Approve(context.Background(), checker, pending.ID, pending.RecordVersion, preview.ID)
	if _, err := service.Activate(context.Background(), checker, approved.ID, approved.RecordVersion); !errors.Is(err, ErrShadowRequired) {
		t.Fatalf("enforce activation err = %v", err)
	}
}

func TestPolicyValidationRejectsUnsafeDatesBlastRadiusAndTemplateVariables(t *testing.T) {
	service, _, _ := newPolicyTestService(t)
	actor := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker"}
	for name, mutate := range map[string]func(*CreateInput){
		"dates": func(input *CreateInput) {
			before := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
			after := before.Add(-time.Hour)
			input.EffectiveFrom, input.EffectiveUntil = &before, &after
		},
		"blast":    func(input *CreateInput) { input.BlastRadius = BlastRadius{PerRun: 0, PerDay: 2} },
		"variable": func(input *CreateInput) { input.Action.TitleTemplate = "Review {{recipient_email}}" },
		"subject":  func(input *CreateInput) { input.Eligibility.SubjectTypes = []string{"VENDOR", strings.Repeat("x", 81)} },
		"band": func(input *CreateInput) {
			input.Eligibility.Bands = []formcontract.ConcernBand{formcontract.ConcernHigh, "URGENT"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			input := validPolicyInput("invalid-"+name, RolloutShadow)
			mutate(&input)
			if _, err := service.Create(context.Background(), actor, input); !errors.Is(err, ErrInvalid) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestRollbackCreatesAuditableDraftRevision(t *testing.T) {
	service, _, _ := newPolicyTestService(t)
	maker := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker"}
	checker := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "checker"}
	first := createPolicyFixture(t, service, maker, RolloutShadow)
	preview, _ := service.Simulate(context.Background(), maker, first.ID, first.RecordVersion)
	pending, _ := service.Submit(context.Background(), maker, first.ID, first.RecordVersion, preview.ID)
	active, _ := service.Approve(context.Background(), checker, pending.ID, pending.RecordVersion, preview.ID)
	active, _ = service.Activate(context.Background(), checker, active.ID, active.RecordVersion)
	rolled, err := service.Rollback(context.Background(), checker, active.ID, active.RecordVersion, first.ID)
	if err != nil || rolled.Status != PolicyDraft || rolled.Version != first.Version+1 || rolled.RollbackOfPolicyID != active.ID || rolled.SupersedesPolicyID != active.ID || rolled.MakerID != "checker" {
		t.Fatalf("rollback = %#v, err = %v", rolled, err)
	}
}

func newPolicyTestService(t *testing.T) (*Service, *policyResponseReader, *policyFormReader) {
	t.Helper()
	responses := &policyResponseReader{items: []evidence.CompletedResponseSummary{
		scoredResponse("response-high", 20, 80, formcontract.ConcernHigh, time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)),
		scoredResponse("response-critical", 40, 75, formcontract.ConcernCritical, time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)),
		scoredResponse("response-low", 90, 10, formcontract.ConcernLow, time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)),
	}}
	forms := &policyFormReader{form: evidence.DistributionFormRevision{ID: "form-a", TenantID: "bank", LegalEntityID: "entity", Version: 3, Active: true, ScoringMode: formcontract.ScoringCompliance}}
	service := NewService(NewMemoryRepository(), forms, responses)
	service.ConfigureActivationAuthority(policyActivationAuthority{})
	service.now = func() time.Time { return time.Date(2026, 9, 1, 11, 0, 0, 0, time.UTC) }
	next := 0
	service.newID = func() (string, error) { next++; return fmt.Sprintf("id-%d", next), nil }
	return service, responses, forms
}

func createPolicyFixture(t *testing.T, service *Service, actor Actor, rollout RolloutMode) Policy {
	t.Helper()
	policy, err := service.Create(context.Background(), actor, validPolicyInput("poor-vendor-response", rollout))
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func validPolicyInput(code string, rollout RolloutMode) CreateInput {
	return CreateInput{
		Code: code, Name: "Poor vendor response", Purpose: "Open an issue when a completed vendor response needs follow-up.",
		AutomationPolicyID: "automation-a", AutomationPolicyVersion: 2,
		Eligibility: Eligibility{FormTemplateID: "form-a", FormTemplateVersion: 3, SubjectTypes: []string{"VENDOR"}, CurrentOnly: true, MinimumCoverage: 0.8, Bands: []formcontract.ConcernBand{formcontract.ConcernHigh, formcontract.ConcernCritical}},
		Action:      MatterAction{Type: "VENDOR_DEFICIENCY", Priority: 4, TitleTemplate: "Review {{form_title}} for {{subject_id}}", SummaryTemplate: "The completed response is {{concern}} concern.", RequestedHandling: "Review the response, assign an owner and confirm corrective evidence."},
		BlastRadius: BlastRadius{PerRun: 1, PerDay: 10},
		Outcome:     OutcomeContract{ExpectedOutcome: "The response concern is resolved and supporting evidence is independently checked.", CheckAfterMinutes: 1440, FailureResponse: "ESCALATE"},
		Rollout:     rollout,
	}
}

func scoredResponse(id string, raw, adverse float64, band formcontract.ConcernBand, at time.Time) evidence.CompletedResponseSummary {
	return evidence.CompletedResponseSummary{ID: id, TenantID: "bank", LegalEntityID: "entity", FormTemplateID: "form-a", FormTemplateVersion: 3, Title: "Vendor certification refresh", SubjectType: "VENDOR", SubjectID: "vendor-a", Current: true, CompletedAt: at, Score: &evidence.ResponseScoreResult{Mode: formcontract.ScoringCompliance, Direction: formcontract.DirectionLowIsPoor, RawScore: &raw, AdverseScore: &adverse, Band: band, Coverage: 1, Final: true, State: evidence.ResponseScoreFinal}}
}
