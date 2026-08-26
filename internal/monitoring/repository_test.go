package monitoring

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryRepositoryStoresExactFormVersionsWithinTenant(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 17, 10, 0, 0, 0, time.UTC)
	form := FormTemplate{
		ID: "form-1", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: "program-1", Code: "PASSWORD-RESET", Name: "Password reset review",
		Purpose: "Collect the weekly password reset control review.", Fields: []TemplateField{{ID: "identity", Label: "Identity verification completed", Type: "single_select", Required: true, Options: []string{"Yes", "No"}}},
		Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1, CreatedBy: "maker", CreatedAt: now, UpdatedAt: now},
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
	if stored.Name != "Password reset review" || stored.Fields[0].Options[1] != "No" {
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
