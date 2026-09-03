package monitoring

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestMemoryRepositoryStoresExactFormVersionsWithinTenant(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 17, 10, 0, 0, 0, time.UTC)
	form := FormTemplate{
		ID: "form-1", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: "program-1", Code: "PASSWORD-RESET", Name: "Password reset review",
		Purpose:      "Collect the weekly password reset control review.",
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard, AllowModeSwitch: true},
		Sections:     []formcontract.Section{{ID: "identity-checks", Title: "Identity checks"}},
		Fields:       []TemplateField{{ID: "identity", SectionID: "identity-checks", Label: "Identity verification completed", Type: "single_select", Required: true, Options: []string{"Yes", "No"}}},
		Lifecycle:    Lifecycle{Status: LifecycleDraft, Version: 1, CreatedBy: "maker", CreatedAt: now, UpdatedAt: now},
	}
	created, err := repo.CreateFormRevision(context.Background(), form)
	if err != nil {
		t.Fatalf("create form: %v", err)
	}
	created.Name = "mutated outside repository"

	stored, err := repo.FormRevision(context.Background(), "bank-a", "entity-a", "program-1", "form-1", 1)
	if err != nil {
		t.Fatalf("get form: %v", err)
	}
	if stored.Name != "Password reset review" || stored.Presentation.DefaultMode != formcontract.PresentationWizard || !stored.Presentation.AllowModeSwitch || len(stored.Sections) != 1 || stored.Sections[0].Title != "Identity checks" || stored.Fields[0].Options[1] != "No" {
		t.Fatalf("stored form mutated: %#v", stored)
	}
	if _, err := repo.FormRevision(context.Background(), "bank-b", "entity-a", "program-1", "form-1", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant read error = %v, want not found", err)
	}
	if _, err := repo.FormRevision(context.Background(), "bank-a", "entity-b", "program-1", "form-1", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity read error = %v, want not found", err)
	}
	if _, err := repo.FormRevision(context.Background(), "bank-a", "entity-a", "program-2", "form-1", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-Program read error = %v, want not found", err)
	}
}

func TestMemoryRepositoryRejectsDuplicateFormVersion(t *testing.T) {
	repo := NewMemoryRepository()
	form := FormTemplate{ID: "form-1", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: "program-1", Code: "FORM", Name: "Form", Purpose: "Purpose", Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1}}
	if _, err := repo.CreateFormRevision(context.Background(), form); err != nil {
		t.Fatalf("create form: %v", err)
	}
	if _, err := repo.CreateFormRevision(context.Background(), form); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate error = %v, want conflict", err)
	}
}

func TestMemoryRepositoryListsOnlyCurrentActiveReusableFormsForLegalEntity(t *testing.T) {
	repo := NewMemoryRepository()
	forms := []FormTemplate{
		{ID: "form-a", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: "program-1", Code: "ACCESS", Name: "Access review", Purpose: "Confirm access.", Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, Version: 2}},
		{ID: "form-b", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: "program-2", Code: "RESILIENCE", Name: "Resilience review", Purpose: "Confirm resilience.", Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, Version: 1}},
		{ID: "form-c", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: "program-1", Code: "DRAFT", Name: "Draft review", Purpose: "Not approved.", Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1}},
		{ID: "form-d", TenantID: "bank-a", LegalEntityID: "entity-b", ProgramID: "program-3", Code: "OTHER", Name: "Other entity review", Purpose: "Other entity.", Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, Version: 1}},
	}
	for _, form := range forms {
		if _, err := repo.CreateFormRevision(context.Background(), form); err != nil {
			t.Fatalf("create form %s: %v", form.ID, err)
		}
	}

	values, err := repo.ListReusableFormRevisions(context.Background(), "bank-a", "entity-a", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0].ID != "form-a" || values[1].ID != "form-b" {
		t.Fatalf("reusable forms = %#v", values)
	}
	if _, err := repo.ReusableFormRevision(context.Background(), "bank-a", "entity-b", "form-a", 2); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity exact read error = %v, want not found", err)
	}
}

func TestMemoryRepositoryStoresChecksAndAppendOnlyResults(t *testing.T) {
	repo := NewMemoryRepository()
	check := MonitoringCheck{
		ID: "check-1", TenantID: "bank-a", ProgramID: "program-1", Code: "RESET-CHECK", Name: "Password reset control",
		Claim: "Password reset safeguards are operating.", InputKind: InputForm, FormTemplateID: "form-1", FormTemplateVersion: 3,
		Thresholds: DefaultThresholds(), Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, Version: 1},
	}
	if _, err := repo.CreateCheckRevision(context.Background(), check); err != nil {
		t.Fatalf("create check: %v", err)
	}
	stored, err := repo.CheckRevision(context.Background(), "bank-a", "check-1", 1)
	if err != nil || stored.FormTemplateVersion != 3 {
		t.Fatalf("stored check = %#v, err = %v", stored, err)
	}

	score := 80.0
	result := MonitoringResult{ID: "result-1", TenantID: "bank-a", ProgramID: "program-1", MonitoringCheckID: "check-1", MonitoringCheckVersion: 1, InputKind: InputForm, InputReferenceID: "submission-1", InputReferenceVersion: 1, EvaluatorVersion: "risk-v1", Evaluation: Evaluation{Score: &score, Band: RiskCritical, Coverage: 1}}
	if _, err := repo.AppendResult(context.Background(), result); err != nil {
		t.Fatalf("append result: %v", err)
	}
	if _, err := repo.AppendResult(context.Background(), result); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate result error = %v, want conflict", err)
	}
	events := repo.MonitoringEvents("bank-a", "MONITORING_RESULT", "result-1")
	outbox := repo.MonitoringOutbox("bank-a", "MONITORING_RESULT", "result-1")
	if len(events) != 1 || events[0].Type != "MONITORING_RESULT_RECORDED" || events[0].AggregateVersion != 1 {
		t.Fatalf("result events = %#v, want one immutable recorded event", events)
	}
	if len(outbox) != 1 || outbox[0].Type != "MONITORING_RESULT_RECORDED" {
		t.Fatalf("result outbox = %#v, want one delivery event", outbox)
	}
	items, err := repo.ListResults(context.Background(), "bank-a", "check-1", 10)
	if err != nil || len(items) != 1 || items[0].Evaluation.Score == nil || *items[0].Evaluation.Score != 80 {
		t.Fatalf("results = %#v, err = %v", items, err)
	}
}

func TestMemoryRepositoryLoadsResultByExactTenantAndID(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	result := MonitoringResult{
		ID: "result-exact", TenantID: "bank-a", ProgramID: "program-a", MonitoringCheckID: "check-a", MonitoringCheckVersion: 2,
		InputKind: InputSource, InputReferenceID: "receipt-a", InputReferenceVersion: 3,
		Evaluation: Evaluation{Band: RiskHigh, Coverage: 1}, EvaluatedAt: now, EvaluatorVersion: "risk-v2", CreatedAt: now,
	}
	if _, err := repo.AppendResult(t.Context(), result); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.Result(t.Context(), "bank-a", result.ID)
	if err != nil {
		t.Fatalf("load exact result: %v", err)
	}
	if stored.ID != result.ID || stored.TenantID != result.TenantID || stored.MonitoringCheckID != result.MonitoringCheckID {
		t.Fatalf("stored result = %#v, want exact immutable result", stored)
	}
	if _, err := repo.Result(t.Context(), "bank-b", result.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant result read error = %v, want not found", err)
	}
	if _, err := repo.Result(t.Context(), "bank-a", "missing-result"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing result read error = %v, want not found", err)
	}
}

func TestMemoryRepositoryRecordsCheckRevisionAndTransitionEventsExactlyOnce(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC)
	check := MonitoringCheck{
		ID: "check-events", TenantID: "bank-a", ProgramID: "program-a", Code: "EVENTS", Name: "Event test", Claim: "Every material revision is reconstructable.",
		InputKind: InputSource, BindingID: "binding-1", BindingVersion: 1, Thresholds: DefaultThresholds(), FreshnessMinutes: 60, MinimumCoverage: 1,
		FailureAction: FailureReview, Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1, CreatedBy: "owner", CreatedAt: now, UpdatedAt: now},
	}
	if _, err := repo.CreateCheckRevision(t.Context(), check); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.TransitionCheck(t.Context(), LifecycleTransition{TenantID: "bank-a", ID: check.ID, ExpectedVersion: 1, To: LifecyclePendingApproval, ActorID: "owner", At: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	events := repo.MonitoringEvents("bank-a", "MONITORING_CHECK", check.ID)
	outbox := repo.MonitoringOutbox("bank-a", "MONITORING_CHECK", check.ID)
	if len(events) != 2 || len(outbox) != 2 {
		t.Fatalf("check journal/outbox counts = %d/%d, want 2/2", len(events), len(outbox))
	}
	if events[0].Type != "MONITORING_CHECK_CREATED" || events[0].AggregateVersion != 1 || events[1].Type != "MONITORING_CHECK_STATE_CHANGED" || events[1].AggregateVersion != 2 {
		t.Fatalf("unexpected check event history: %#v", events)
	}
	if _, err := repo.TransitionCheck(t.Context(), LifecycleTransition{TenantID: "bank-a", ID: check.ID, ExpectedVersion: 1, To: LifecyclePendingApproval, ActorID: "owner", At: now.Add(2 * time.Minute)}); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate transition error = %v, want conflict", err)
	}
	if got := len(repo.MonitoringEvents("bank-a", "MONITORING_CHECK", check.ID)); got != 2 {
		t.Fatalf("duplicate transition appended event, count = %d", got)
	}
}

func TestMemoryRepositoryLoadsLatestMonitoringCheckRevisionByID(t *testing.T) {
	repo := NewMemoryRepository()
	createdAt := time.Now().UTC()
	check := MonitoringCheck{
		ID: "check-latest", TenantID: "bank-a", ProgramID: "program-a", Code: "LATEST", Name: "Latest check", Claim: "The latest revision is used.",
		InputKind: InputSource, BindingID: "binding-1", BindingVersion: 1, Thresholds: DefaultThresholds(), FreshnessMinutes: 60, MinimumCoverage: 1,
		FailureAction: FailureReview, Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1, CreatedBy: "owner", CreatedAt: createdAt, UpdatedAt: createdAt},
	}
	if _, err := repo.CreateCheckRevision(t.Context(), check); err != nil {
		t.Fatal(err)
	}
	second, err := repo.TransitionCheck(t.Context(), LifecycleTransition{TenantID: "bank-a", ID: check.ID, ExpectedVersion: 1, To: LifecyclePendingApproval, ActorID: "owner", At: createdAt.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	latest, err := repo.LatestCheckRevision(t.Context(), "bank-a", check.ID)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Version != second.Version || latest.ProgramID != "program-a" {
		t.Fatalf("latest check = %#v, want version %d for program-a", latest, second.Version)
	}
	if _, err := repo.LatestCheckRevision(t.Context(), "bank-b", check.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant latest check error = %v, want not found", err)
	}
}

func TestMemoryRepositoryActivatingReplacementCheckRetiresCurrentCheckWithSameProgramCode(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	current := MonitoringCheck{
		ID: "check-current", TenantID: "bank-a", ProgramID: "program-a", Code: "ENCRYPTION-CHECK", Name: "Encryption check", Claim: "Encryption remains enabled.",
		InputKind: InputForm, FormTemplateID: "form-a", FormTemplateVersion: 8, Thresholds: DefaultThresholds(), FreshnessMinutes: 60, MinimumCoverage: 1,
		FailureAction: FailureReview, Lifecycle: Lifecycle{Status: LifecyclePaused, IsCurrent: true, Version: 3, EffectiveFrom: &now, CreatedAt: now, UpdatedAt: now},
	}
	replacement := current
	replacement.ID = "check-replacement"
	replacement.FormTemplateVersion = 9
	replacement.Lifecycle = Lifecycle{Status: LifecyclePendingApproval, Version: 2, SubmittedBy: "owner", CreatedAt: now, UpdatedAt: now}
	for _, check := range []MonitoringCheck{current, replacement} {
		if _, err := repo.CreateCheckRevision(t.Context(), check); err != nil {
			t.Fatal(err)
		}
	}

	activated, err := repo.TransitionCheck(t.Context(), LifecycleTransition{TenantID: "bank-a", ID: replacement.ID, ExpectedVersion: 2, ExpectedCurrentID: current.ID, ExpectedCurrentVersion: current.Version, To: LifecycleActive, ActorID: "reviewer", At: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if !activated.IsCurrent || activated.Status != LifecycleActive {
		t.Fatalf("replacement = %#v, want current active check", activated)
	}
	retired, err := repo.CheckRevision(t.Context(), "bank-a", current.ID, current.Version+1)
	if err != nil {
		t.Fatal(err)
	}
	if retired.IsCurrent || retired.Status != LifecycleRetired || retired.EffectiveUntil == nil {
		t.Fatalf("superseded check = %#v, want ended non-current check", retired)
	}
	events := repo.MonitoringEvents("bank-a", AggregateMonitoringCheck, current.ID)
	outbox := repo.MonitoringOutbox("bank-a", AggregateMonitoringCheck, current.ID)
	if len(events) != 2 || len(outbox) != 2 || events[1].Type != EventMonitoringCheckStateChanged || events[1].AggregateVersion != current.Version+1 {
		t.Fatalf("retirement event/outbox = %#v/%#v, want created and state-changed revision %d", events, outbox, current.Version+1)
	}
}

func TestMemoryRepositoryRejectsReplacementWhenExpectedCurrentCheckChanged(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 9, 2, 13, 0, 0, 0, time.UTC)
	base := MonitoringCheck{ID: "check-current", TenantID: "bank-a", ProgramID: "program-a", Code: "ENCRYPTION-CHECK", Name: "Encryption check", Claim: "Encryption remains enabled.", InputKind: InputForm, FormTemplateID: "form-a", FormTemplateVersion: 8, Thresholds: DefaultThresholds(), FreshnessMinutes: 60, MinimumCoverage: 1, FailureAction: FailureReview, Lifecycle: Lifecycle{Status: LifecyclePaused, IsCurrent: true, Version: 3, EffectiveFrom: &now, CreatedAt: now, UpdatedAt: now}}
	first := base
	first.ID = "replacement-one"
	first.Lifecycle = Lifecycle{Status: LifecyclePendingApproval, Version: 2, SubmittedBy: "owner", CreatedAt: now, UpdatedAt: now}
	second := first
	second.ID = "replacement-two"
	for _, check := range []MonitoringCheck{base, first, second} {
		if _, err := repo.CreateCheckRevision(t.Context(), check); err != nil {
			t.Fatal(err)
		}
	}
	input := LifecycleTransition{TenantID: "bank-a", ExpectedVersion: 2, ExpectedCurrentID: base.ID, ExpectedCurrentVersion: base.Version, To: LifecycleActive, ActorID: "reviewer", At: now.Add(time.Minute)}
	input.ID = first.ID
	if _, err := repo.TransitionCheck(t.Context(), input); err != nil {
		t.Fatal(err)
	}
	input.ID = second.ID
	if _, err := repo.TransitionCheck(t.Context(), input); !errors.Is(err, ErrConflict) {
		t.Fatalf("second stale approval error = %v, want conflict", err)
	}
}
