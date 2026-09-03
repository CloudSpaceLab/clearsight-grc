package monitoring

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNormalizeCollectionPolicyDefaults(t *testing.T) {
	got, err := normalizeCollectionPolicy(&CollectionPolicy{ValidityMonths: 12})
	if err != nil {
		t.Fatal(err)
	}
	want := CollectionPolicy{ValidityMonths: 12, RenewalWindowDays: 30, ReminderCount: 3}
	if got != want {
		t.Fatalf("policy = %#v, want %#v", got, want)
	}
}

func TestNormalizeCollectionPolicyRejectsWindowBeyondOneMonth(t *testing.T) {
	_, err := normalizeCollectionPolicy(&CollectionPolicy{ValidityMonths: 1, RenewalWindowDays: 30, ReminderCount: 3})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid policy, got %v", err)
	}
}

func TestNormalizeCollectionPolicyAcceptsOneToFiveReminders(t *testing.T) {
	for reminders := 1; reminders <= 5; reminders++ {
		got, err := normalizeCollectionPolicy(&CollectionPolicy{ValidityMonths: 12, RenewalWindowDays: 30, ReminderCount: reminders})
		if err != nil || got.ReminderCount != reminders {
			t.Fatalf("reminders %d: policy = %#v, err = %v", reminders, got, err)
		}
	}
	if _, err := normalizeCollectionPolicy(&CollectionPolicy{ValidityMonths: 12, RenewalWindowDays: 30, ReminderCount: 6}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("six reminders error = %v, want invalid", err)
	}
}

func TestServiceRequiresCollectionPolicyForNewFormCheck(t *testing.T) {
	repo, service, actor := collectionPolicyService(t)
	_, err := service.CreateCheck(context.Background(), actor, formCheckInput(nil))
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing policy error = %v, want invalid", err)
	}

	created, err := service.CreateCheck(context.Background(), actor, formCheckInput(&CollectionPolicy{ValidityMonths: 12}))
	if err != nil {
		t.Fatal(err)
	}
	want := &CollectionPolicy{ValidityMonths: 12, RenewalWindowDays: 30, ReminderCount: 3}
	if created.CollectionPolicy == nil || *created.CollectionPolicy != *want {
		t.Fatalf("collection policy = %#v, want %#v", created.CollectionPolicy, want)
	}
	if _, err := repo.CheckRevision(context.Background(), actor.TenantID, created.ID, created.Version); err != nil {
		t.Fatal(err)
	}
}

func TestServiceRejectsCollectionPolicyForSourceCheck(t *testing.T) {
	_, service, actor := collectionPolicyService(t)
	_, err := service.CreateCheck(context.Background(), actor, CreateCheckInput{
		ProgramID: "program-1", Code: "SOURCE", Name: "Connected source", Claim: "The source remains current.", InputKind: InputSource,
		BindingID: "binding-1", BindingVersion: 1, SourceRules: []SourceRule{{ID: "present", Field: "status", Operator: OperatorPresent}},
		CollectionPolicy: &CollectionPolicy{ValidityMonths: 12}, FreshnessMinutes: 60, MinimumCoverage: 1,
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("source policy error = %v, want invalid", err)
	}
}

func TestServiceReadsLegacyActiveFormCheckWithoutPolicy(t *testing.T) {
	repo := NewMemoryRepository()
	activeAt := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	legacy := MonitoringCheck{
		ID: "check-legacy", TenantID: "bank-a", ProgramID: "program-1", Code: "LEGACY", Name: "Legacy collection", Claim: "A response exists.",
		InputKind: InputForm, FormTemplateID: "form-1", FormTemplateVersion: 1, Thresholds: DefaultThresholds(), FreshnessMinutes: 1440, MinimumCoverage: 1, FailureAction: FailureReview,
		Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 4},
	}
	if _, err := repo.CreateCheckRevision(context.Background(), legacy); err != nil {
		t.Fatal(err)
	}
	service := NewService(repo, nil)
	checks, err := service.ListChecks(context.Background(), Actor{TenantID: "bank-a", PrincipalID: "owner"}, "program-1", 10)
	if err != nil || len(checks) != 1 || checks[0].CollectionPolicy != nil {
		t.Fatalf("legacy checks = %#v, err = %v", checks, err)
	}
}

func TestServiceRevisesActiveCollectionPolicyThroughMakerChecker(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	activeAt := now.Add(-24 * time.Hour)
	form := FormTemplate{
		ID: "form-1", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: "program-1", Code: "VENDOR-FORM", Name: "Vendor review", Purpose: "Collect a current vendor response.",
		Fields:    []TemplateField{{ID: "answer", Label: "Answer", Type: "text", Required: true}},
		Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 2},
	}
	if _, err := repo.CreateFormRevision(context.Background(), form); err != nil {
		t.Fatal(err)
	}
	active := MonitoringCheck{
		ID: "check-1", TenantID: "bank-a", ProgramID: "program-1", Code: "VENDOR", Name: "Vendor review", Claim: "The vendor response remains current.",
		InputKind: InputForm, FormTemplateID: "form-1", FormTemplateVersion: 2, CollectionPolicy: &CollectionPolicy{ValidityMonths: 12, RenewalWindowDays: 30, ReminderCount: 3},
		Thresholds: DefaultThresholds(), FreshnessMinutes: 10080, MinimumCoverage: 1, FailureAction: FailureReview,
		Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 2, ApprovedBy: "reviewer", CreatedAt: activeAt, UpdatedAt: activeAt},
	}
	if _, err := repo.CreateCheckRevision(context.Background(), active); err != nil {
		t.Fatal(err)
	}
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }
	maker := Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "maker"}
	draft, err := service.UpdateCollectionPolicy(context.Background(), maker, UpdateCollectionPolicyInput{
		ID: active.ID, ExpectedVersion: active.Version, Policy: CollectionPolicy{ValidityMonths: 24, RenewalWindowDays: 45, ReminderCount: 5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if draft.Status != LifecycleDraft || draft.IsCurrent || draft.Version != 3 || draft.CollectionPolicy == nil || draft.CollectionPolicy.ValidityMonths != 24 {
		t.Fatalf("draft revision = %#v", draft)
	}
	stillActive, err := repo.CheckRevision(context.Background(), "bank-a", active.ID, active.Version)
	if err != nil || !stillActive.IsCurrent || stillActive.Status != LifecycleActive {
		t.Fatalf("active revision = %#v, err = %v", stillActive, err)
	}

	pending, err := service.TransitionCheck(context.Background(), maker, TransitionInput{ID: draft.ID, ExpectedVersion: draft.Version, To: LifecyclePendingApproval})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionCheck(context.Background(), maker, TransitionInput{ID: pending.ID, ExpectedVersion: pending.Version, To: LifecycleActive}); !errors.Is(err, ErrMakerChecker) {
		t.Fatalf("same-maker approval error = %v", err)
	}
	approved, err := service.TransitionCheck(context.Background(), Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "approver"}, TransitionInput{ID: pending.ID, ExpectedVersion: pending.Version, To: LifecycleActive})
	if err != nil || !approved.IsCurrent || approved.Status != LifecycleActive || approved.CollectionPolicy == nil || approved.CollectionPolicy.ValidityMonths != 24 {
		t.Fatalf("approved revision = %#v, err = %v", approved, err)
	}
}

func collectionPolicyService(t *testing.T) (*MemoryRepository, *Service, Actor) {
	t.Helper()
	repo := NewMemoryRepository()
	activeAt := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	form := FormTemplate{
		ID: "form-1", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: "program-1", Code: "FORM", Name: "Vendor review", Purpose: "Collect a current vendor response.",
		Fields:    []TemplateField{{ID: "answer", Label: "Answer", Type: "text", Required: true}},
		Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 2},
	}
	if _, err := repo.CreateFormRevision(context.Background(), form); err != nil {
		t.Fatal(err)
	}
	service := NewService(repo, nil)
	service.newID = func() (string, error) { return "check-1", nil }
	service.now = func() time.Time { return time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC) }
	return repo, service, Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "maker"}
}

func formCheckInput(policy *CollectionPolicy) CreateCheckInput {
	return CreateCheckInput{
		ProgramID: "program-1", Code: "VENDOR", Name: "Vendor review", Claim: "The vendor response remains current.", InputKind: InputForm,
		FormTemplateID: "form-1", FormTemplateVersion: 2, CollectionPolicy: policy,
		FreshnessMinutes: 10080, MinimumCoverage: 1, FailureAction: FailureReview,
	}
}
