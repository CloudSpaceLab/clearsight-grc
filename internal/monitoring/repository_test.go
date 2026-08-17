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
		ID: "form-1", TenantID: "bank-a", Code: "PASSWORD-RESET", Name: "Password reset review",
		Purpose: "Collect the weekly password reset control review.", Fields: []TemplateField{{ID: "identity", Label: "Identity verification completed", Type: "single_select", Required: true, Options: []string{"Yes", "No"}}},
		Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1, CreatedBy: "maker", CreatedAt: now, UpdatedAt: now},
	}
	created, err := repo.CreateFormRevision(context.Background(), form)
	if err != nil {
		t.Fatalf("create form: %v", err)
	}
	created.Name = "mutated outside repository"

	stored, err := repo.FormRevision(context.Background(), "bank-a", "form-1", 1)
	if err != nil {
		t.Fatalf("get form: %v", err)
	}
	if stored.Name != "Password reset review" || stored.Fields[0].Options[1] != "No" {
		t.Fatalf("stored form mutated: %#v", stored)
	}
	if _, err := repo.FormRevision(context.Background(), "bank-b", "form-1", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant read error = %v, want not found", err)
	}
}

func TestMemoryRepositoryRejectsDuplicateFormVersion(t *testing.T) {
	repo := NewMemoryRepository()
	form := FormTemplate{ID: "form-1", TenantID: "bank-a", Code: "FORM", Name: "Form", Purpose: "Purpose", Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1}}
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
	items, err := repo.ListResults(context.Background(), "bank-a", "check-1", 10)
	if err != nil || len(items) != 1 || items[0].Evaluation.Score == nil || *items[0].Evaluation.Score != 80 {
		t.Fatalf("results = %#v, err = %v", items, err)
	}
}
