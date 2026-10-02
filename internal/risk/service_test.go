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

func TestExpiredNewerAppetiteDoesNotReactivateOlderStatement(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 10, 2, 9, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	created := createTestRisk(t, service, ctx, "bank", "entity-a", "RISK-EXPIRE")

	firstRisk, first, err := service.ActivateAppetite(ctx, AppetiteInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedRiskVersion: created.Version,
		Statement: "Keep disruption below 30 minutes.", Rule: json.RawMessage(`{"max_minutes":30}`),
		ActorID: "cro-1", EffectiveFrom: now.Add(-2 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	until := now.Add(time.Minute)
	secondRisk, second, err := service.ActivateAppetite(ctx, AppetiteInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedRiskVersion: firstRisk.Version,
		Statement: "Temporary 15 minute tolerance.", Rule: json.RawMessage(`{"max_minutes":15}`),
		ActorID: "cro-1", EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: &until,
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err := repo.CurrentAppetite(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, created.ID, now)
	if err != nil || current == nil || current.ID != second.ID {
		t.Fatalf("temporary appetite is not current: current=%#v err=%v", current, err)
	}

	now = until.Add(time.Minute)
	current, err = repo.CurrentAppetite(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, created.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if current != nil {
		t.Fatalf("older appetite %s reactivated after newer statement expired: %#v", first.ID, current)
	}

	_, _, err = service.AddAssessment(ctx, AssessmentInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedRiskVersion: secondRisk.Version,
		Kind: AssessmentCurrent, MethodCode: "QUAL-5X5", MethodVersion: "v1",
		Dimensions: json.RawMessage(`{"likelihood":3,"impact":4}`), AppetiteStatementID: first.ID,
		AppetitePosition: AppetiteBreached, ActorID: "reviewer-1",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expired appetite window allowed older statement: %v", err)
	}
}

func TestExpiredAppetiteMakesRecordedPositionCurrentlyUnknown(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	now := time.Date(2026, 10, 2, 9, 30, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	created := createTestRisk(t, service, ctx, "bank", "entity-a", "RISK-TIMED")
	until := now.Add(2 * time.Minute)
	withAppetite, appetite, err := service.ActivateAppetite(ctx, AppetiteInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedRiskVersion: created.Version,
		Statement: "Keep disruption below 30 minutes.", Rule: json.RawMessage(`{"max_minutes":30}`),
		ActorID: "cro-1", EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: &until,
	})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	assessed, _, err := service.AddAssessment(ctx, AssessmentInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedRiskVersion: withAppetite.Version,
		Kind: AssessmentResidual, MethodCode: "QUAL-5X5", MethodVersion: "v1",
		Dimensions: json.RawMessage(`{"likelihood":4,"impact":5}`), AppetiteStatementID: appetite.ID,
		AppetitePosition: AppetiteBreached, ActorID: "reviewer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	breached, err := service.List(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, ListFilter{
		AppetitePosition: AppetiteBreached, Limit: 10,
	})
	if err != nil || len(breached.Items) != 1 {
		t.Fatalf("current breached posture missing: page=%#v err=%v", breached, err)
	}

	now = until.Add(time.Minute)
	unknown, err := service.List(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, ListFilter{
		AppetitePosition: AppetiteUnknown, Limit: 10,
	})
	if err != nil || len(unknown.Items) != 1 || unknown.Items[0].Risk.ID != assessed.ID {
		t.Fatalf("expired appetite was not UNKNOWN: page=%#v err=%v", unknown, err)
	}
	breached, err = service.List(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, ListFilter{
		AppetitePosition: AppetiteBreached, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(breached.Items) != 0 {
		t.Fatalf("expired appetite retained breached posture: %#v", breached.Items)
	}
	aggregate, err := service.Get(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, assessed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if aggregate.ActiveAppetite != nil {
		t.Fatalf("expired appetite returned as active: %#v", aggregate.ActiveAppetite)
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
		Dimensions:       json.RawMessage(`{"likelihood":3,"impact":4}`),
		AppetitePosition: AppetiteWithin, ActorID: "reviewer-1",
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("assessment claimed appetite without statement: %v", err)
	}

	updated, assessment, err := service.AddAssessment(ctx, AssessmentInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedRiskVersion: created.Version,
		Kind: AssessmentInherent, MethodCode: "QUAL-5X5", MethodVersion: "v1",
		Dimensions:       json.RawMessage(`{"likelihood":3,"impact":4}`),
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
	if len(filtered.Items) != 3 {
		t.Fatalf("UNKNOWN must include assessed-unknown and unassessed risks: %#v", filtered)
	}
	if filtered.Items[0].Risk.ID != first.ID || filtered.Items[0].LatestAssessment == nil || filtered.Items[0].LatestAssessment.MethodVersion != "v2" {
		t.Fatalf("latest assessment projection failed: %#v", filtered.Items[0])
	}
}

func TestRiskListDoesNotTreatStaleAssessmentAsCurrentAppetitePosition(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	now := time.Date(2026, 10, 2, 12, 30, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	created := createTestRisk(t, service, ctx, "bank", "entity-a", "RISK-STALE")

	assessedRisk, _, err := service.AddAssessment(ctx, AssessmentInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedRiskVersion: created.Version,
		Kind: AssessmentCurrent, MethodCode: "QUAL-5X5", MethodVersion: "v1",
		Dimensions: json.RawMessage(`{"likelihood":5,"impact":5}`), AppetitePosition: AppetiteUnknown, ActorID: "reviewer-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	now = now.Add(time.Minute)
	updated, err := service.Update(ctx, UpdateInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedVersion: assessedRisk.Version,
		Name: created.Name, Category: created.Category, Statement: created.Statement,
		Cause: created.Cause, Event: created.Event, Impact: created.Impact, Scope: created.Scope,
		OwnerPrincipalID: created.OwnerPrincipalID, Status: StatusActive, ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version == assessedRisk.Version {
		t.Fatal("risk version did not advance")
	}

	page, err := service.List(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, ListFilter{
		AppetitePosition: AppetiteUnknown, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Risk.ID != created.ID {
		t.Fatalf("stale assessment was not treated as current UNKNOWN: %#v", page.Items)
	}
	breached, err := service.List(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, ListFilter{
		AppetitePosition: AppetiteBreached, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(breached.Items) != 0 {
		t.Fatalf("stale assessment drove BREACHED filtering: %#v", breached.Items)
	}

	unfiltered, err := service.List(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, ListFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(unfiltered.Items) != 1 || unfiltered.Items[0].LatestAssessment == nil ||
		unfiltered.Items[0].LatestAssessment.RiskVersion != assessedRisk.Version {
		t.Fatalf("assessment history was not retained: %#v", unfiltered.Items)
	}
}

func TestRiskControlLinkIsVersionedAndDuplicateSafe(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryRepository())
	service.ConfigureControlLinkValidator(func(_ context.Context, scope Scope, catalogLinkID string) error {
		if scope.TenantID == "bank" && scope.LegalEntityID == "entity-a" && catalogLinkID == "catalog-link-1" {
			return nil
		}
		return ErrInvalid
	})
	now := time.Date(2026, 10, 2, 14, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	created := createTestRisk(t, service, ctx, "bank", "entity-a", "RISK-CONTROL")

	updated, linked, err := service.LinkControl(ctx, LinkControlInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID,
		ExpectedRiskVersion: created.Version, CatalogLinkID: "catalog-link-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 || linked.RiskVersion != 2 || linked.CatalogLinkID != "catalog-link-1" {
		t.Fatalf("updated=%#v linked=%#v", updated, linked)
	}
	aggregate, err := service.Get(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(aggregate.Controls) != 1 || aggregate.Controls[0].ID != linked.ID {
		t.Fatalf("controls=%#v", aggregate.Controls)
	}

	now = now.Add(time.Minute)
	_, _, err = service.LinkControl(ctx, LinkControlInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID,
		ExpectedRiskVersion: updated.Version, CatalogLinkID: "catalog-link-1", ActorID: "owner-1",
	})
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate control link error=%v", err)
	}
	after, err := service.Get(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Risk.Version != 2 || len(after.Controls) != 1 {
		t.Fatalf("duplicate link changed Risk: %#v", after)
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
		Scope:  json.RawMessage(`{"service":"critical-network"}`), OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	}
}
