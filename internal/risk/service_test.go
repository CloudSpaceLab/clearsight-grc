package risk

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestRiskAssessmentRequiresCurrentApplicableAppetite(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }

	created := createTestRisk(t, service, ctx, "bank", "entity-a", "RISK-001")
	firstRisk, first, err := service.ActivateAppetite(ctx, AppetiteInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedRiskVersion: created.Version,
		Statement: "Keep service interruption below approved tolerance.", Rule: json.RawMessage(`{"max_minutes":30}`),
		Rationale: "Protect critical customer services.", OwnerPrincipalID: "owner-1", ActorID: "cro-1", EffectiveFrom: now.Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	secondRisk, second, err := service.ActivateAppetite(ctx, AppetiteInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedRiskVersion: firstRisk.Version,
		Statement: "Keep service interruption below revised tolerance.", Rule: json.RawMessage(`{"max_minutes":20}`),
		Rationale: "Revised board appetite.", OwnerPrincipalID: "owner-1", ActorID: "cro-1", EffectiveFrom: now.Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = service.AddAssessment(ctx, AssessmentInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedRiskVersion: secondRisk.Version,
		Kind: AssessmentResidual, MethodCode: "QUAL-5X5", MethodVersion: "v1",
		Dimensions: json.RawMessage(`{"likelihood":4,"impact":5}`), AppetiteStatementID: first.ID,
		AppetitePosition: AppetiteBreached, AppetiteRationale: "Outside the old tolerance.", ActorID: "reviewer-1",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("stale appetite accepted: %v", err)
	}

	updated, assessment, err := service.AddAssessment(ctx, AssessmentInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedRiskVersion: secondRisk.Version,
		Kind: AssessmentResidual, MethodCode: "QUAL-5X5", MethodVersion: "v1",
		Dimensions: json.RawMessage(`{"likelihood":4,"impact":5}`), AppetiteStatementID: second.ID,
		AppetitePosition: AppetiteBreached, AppetiteRationale: "Recovery exceeds the current tolerance.", ActorID: "reviewer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != secondRisk.Version+1 || assessment.RiskVersion != updated.Version {
		t.Fatalf("version mismatch risk=%d assessment=%d", updated.Version, assessment.RiskVersion)
	}
	if !assessment.AssessedAt.Equal(now) || !updated.UpdatedAt.Equal(now) {
		t.Fatalf("timestamps are not command-consistent: risk=%s assessment=%s now=%s", updated.UpdatedAt, assessment.AssessedAt, now)
	}
}

func TestAssessmentWithoutAppetiteIsExplicitlyUnknown(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	now := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	created := createTestRisk(t, service, ctx, "bank", "entity-a", "RISK-002")

	_, _, err := service.AddAssessment(ctx, AssessmentInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedRiskVersion: created.Version,
		Kind: AssessmentInherent, MethodCode: "QUAL-5X5", MethodVersion: "v1",
		Dimensions: json.RawMessage(`{"likelihood":3,"impact":4}`),
		AppetitePosition: AppetiteWithin, ActorID: "reviewer-1",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("assessment claimed appetite without statement: %v", err)
	}

	updated, assessment, err := service.AddAssessment(ctx, AssessmentInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedRiskVersion: created.Version,
		Kind: AssessmentInherent, MethodCode: "QUAL-5X5", MethodVersion: "v1",
		Dimensions: json.RawMessage(`{"likelihood":3,"impact":4}`),
		AppetitePosition: AppetiteUnknown, ActorID: "reviewer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 || assessment.AppetitePosition != AppetiteUnknown {
		t.Fatalf("unexpected result: risk=%#v assessment=%#v", updated, assessment)
	}
}

func TestRiskMemoryRepositoryEnforcesEntityIsolationAndVersioning(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	now := time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	created := createTestRisk(t, service, ctx, "bank", "entity-a", "RISK-003")

	if _, err := service.Get(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-b"}, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity risk disclosed: %v", err)
	}
	_, err := service.Update(ctx, UpdateInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedVersion: created.Version + 1,
		Name: created.Name, Category: created.Category, Statement: created.Statement, Impact: created.Impact,
		Status: StatusActive, ActorID: "owner-1",
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale version error = %v", err)
	}
}

func TestRiskListUsesStableKeysetAndLatestAssessment(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }

	first := createTestRisk(t, service, ctx, "bank", "entity-a", "RISK-A")
	now = now.Add(time.Minute)
	second := createTestRisk(t, service, ctx, "bank", "entity-a", "RISK-B")
	now = now.Add(time.Minute)
	third := createTestRisk(t, service, ctx, "bank", "entity-a", "RISK-C")

	page1, err := service.List(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, ListFilter{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1.Items) != 2 || page1.Items[0].Risk.ID != third.ID || page1.Items[1].Risk.ID != second.ID || page1.NextCursor == "" {
		t.Fatalf("unexpected first page: %#v", page1)
	}
	page2, err := service.List(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, ListFilter{Limit: 2, Cursor: page1.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Items) != 1 || page2.Items[0].Risk.ID != first.ID || page2.NextCursor != "" {
		t.Fatalf("unexpected second page: %#v", page2)
	}

	now = now.Add(time.Minute)
	riskWithAssessment, _, err := service.AddAssessment(ctx, AssessmentInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: first.ID, ExpectedRiskVersion: first.Version,
		Kind: AssessmentCurrent, MethodCode: "QUAL-5X5", MethodVersion: "v1",
		Dimensions: json.RawMessage(`{"likelihood":2,"impact":2}`), AppetitePosition: AppetiteUnknown, ActorID: "reviewer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	_, _, err = service.AddAssessment(ctx, AssessmentInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: first.ID, ExpectedRiskVersion: riskWithAssessment.Version,
		Kind: AssessmentCurrent, MethodCode: "QUAL-5X5", MethodVersion: "v2",
		Dimensions: json.RawMessage(`{"likelihood":5,"impact":5}`), AppetitePosition: AppetiteUnknown, ActorID: "reviewer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	filtered, err := service.List(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, ListFilter{AppetitePosition: AppetiteUnknown, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Items) != 1 || filtered.Items[0].Risk.ID != first.ID || filtered.Items[0].LatestAssessment == nil || filtered.Items[0].LatestAssessment.MethodVersion != "v2" {
		t.Fatalf("latest assessment filter failed: %#v", filtered)
	}
}

func TestDuplicateRiskCodeIsScopedPerLegalEntity(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	service.Now = func() time.Time { return time.Date(2026, 10, 2, 13, 0, 0, 0, time.UTC) }
	createTestRisk(t, service, ctx, "bank", "entity-a", "RISK-004")

	_, err := service.Create(ctx, validCreate("bank", "entity-a", "RISK-004"))
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate risk code error = %v", err)
	}
	if _, err := service.Create(ctx, validCreate("bank", "entity-b", "RISK-004")); err != nil {
		t.Fatalf("same code in another entity rejected: %v", err)
	}
}

func createTestRisk(t *testing.T, service *Service, ctx context.Context, tenant, entity, code string) Risk {
	t.Helper()
	value, err := service.Create(ctx, validCreate(tenant, entity, code))
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func validCreate(tenant, entity, code string) CreateInput {
	return CreateInput{
		TenantID: tenant, LegalEntityID: entity, Code: code, Name: "Network service resilience",
		Category: "Operational resilience", Statement: "Critical network service may exceed approved recovery tolerance.",
		Cause: "Primary and recovery paths become unavailable.", Event: "Network service interruption",
		Impact: "Customers cannot access critical services within the approved tolerance.",
		Scope: json.RawMessage(`{"service":"critical-network"}`), OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	}
}
