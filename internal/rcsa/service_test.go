package rcsa

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

func TestCreateCycleFreezesRiskControlAndOwnerPopulation(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)

	riskService := risk.NewService(risk.NewMemoryRepository())
	riskService.Now = func() time.Time { return now }
	riskService.ConfigureControlLinkValidator(func(_ context.Context, scope risk.Scope, catalogLinkID string) error {
		if scope.TenantID == "bank" && scope.LegalEntityID == "entity-a" && catalogLinkID == "catalog-link-1" {
			return nil
		}
		return risk.ErrInvalid
	})
	created, err := riskService.Create(ctx, risk.CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "RISK-001", Name: "Payment resilience",
		Category: "Operational resilience", Statement: "Payment service may breach tolerance.",
		Impact: "Customers may be unable to transact.", Scope: json.RawMessage(`{"service":"payments"}`),
		OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	current, control, err := riskService.LinkControl(ctx, risk.LinkControlInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedRiskVersion: created.Version,
		CatalogLinkID: "catalog-link-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	forms := monitoring.NewMemoryRepository()
	_, err = forms.CreateFormRevision(ctx, monitoring.FormTemplate{
		ID: "form-1", TenantID: "bank", LegalEntityID: "entity-a", Code: "RCSA", Name: "RCSA assessment",
		Purpose: "Assess current risk and control conditions.",
		Lifecycle: monitoring.Lifecycle{
			Status: monitoring.LifecycleActive, IsCurrent: true, Version: 3,
			CreatedBy: "risk-team", CreatedAt: now, UpdatedAt: now,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	service := NewService(NewMemoryRepository(), riskService, forms)
	service.Now = func() time.Time { return now }
	aggregate, err := service.Create(ctx, CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "RCSA-2026-Q4", Name: "Q4 RCSA",
		FormTemplateID: "form-1", FormTemplateVersion: 3,
		PeriodStart: now.Add(-30 * 24 * time.Hour), PeriodEnd: now,
		DueAt: now.Add(14 * 24 * time.Hour), ChallengeDueAt: now.Add(21 * 24 * time.Hour),
		RiskIDs: []string{created.ID}, ActorID: "risk-team",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(aggregate.Items) != 1 {
		t.Fatalf("items=%#v", aggregate.Items)
	}
	item := aggregate.Items[0]
	if item.RiskID != created.ID || item.RiskVersion != current.Version || item.RiskCode != current.Code ||
		item.RiskName != current.Name || item.RespondentPrincipalID != "owner-1" {
		t.Fatalf("item snapshot=%#v current=%#v", item, current)
	}
	if got := aggregate.Controls[item.ID]; len(got) != 1 || got[0].ControlLinkID != control.ID || got[0].RiskID != created.ID {
		t.Fatalf("control snapshot=%#v", got)
	}
	if aggregate.Cycle.FormTemplateID != "form-1" || aggregate.Cycle.FormTemplateVersion != 3 {
		t.Fatalf("cycle form=%#v", aggregate.Cycle)
	}

	now = now.Add(time.Hour)
	_, err = riskService.Update(ctx, risk.UpdateInput{
		TenantID: "bank", LegalEntityID: "entity-a", RiskID: created.ID, ExpectedVersion: current.Version,
		Name: "Renamed current risk", Category: current.Category, Statement: current.Statement,
		Cause: current.Cause, Event: current.Event, Impact: current.Impact, Scope: current.Scope,
		OwnerPrincipalID: current.OwnerPrincipalID, Status: risk.StatusActive, ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	stored, err := service.Get(ctx, Scope{TenantID: "bank", LegalEntityID: "entity-a"}, aggregate.Cycle.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Items[0].RiskVersion != current.Version || stored.Items[0].RiskName != "Payment resilience" {
		t.Fatalf("cycle snapshot drifted after Risk update: %#v", stored.Items[0])
	}
}

func TestCreateCycleRejectsDuplicateRiskAndInactiveForm(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	riskService := risk.NewService(risk.NewMemoryRepository())
	riskService.Now = func() time.Time { return now }
	created, err := riskService.Create(ctx, risk.CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "RISK-002", Name: "Treasury risk",
		Statement: "Treasury exposure may exceed tolerance.", Impact: "Material loss.",
		Scope: json.RawMessage(`{}`), OwnerPrincipalID: "owner-2", ActorID: "owner-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	forms := monitoring.NewMemoryRepository()
	_, err = forms.CreateFormRevision(ctx, monitoring.FormTemplate{
		ID: "form-paused", TenantID: "bank", LegalEntityID: "entity-a", Code: "RCSA-P", Name: "Paused RCSA",
		Purpose: "Inactive form.", Lifecycle: monitoring.Lifecycle{
			Status: monitoring.LifecyclePaused, IsCurrent: true, Version: 1, CreatedBy: "risk-team", CreatedAt: now, UpdatedAt: now,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(NewMemoryRepository(), riskService, forms)
	service.Now = func() time.Time { return now }
	base := CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "RCSA-X", Name: "RCSA",
		FormTemplateID: "form-paused", FormTemplateVersion: 1,
		PeriodStart: now.Add(-time.Hour), PeriodEnd: now, DueAt: now.Add(time.Hour),
		ChallengeDueAt: now.Add(2 * time.Hour), RiskIDs: []string{created.ID}, ActorID: "risk-team",
	}
	if _, err := service.Create(ctx, base); !errors.Is(err, ErrInvalid) {
		t.Fatalf("inactive form accepted: %v", err)
	}

	_, err = forms.CreateFormRevision(ctx, monitoring.FormTemplate{
		ID: "form-active", TenantID: "bank", LegalEntityID: "entity-a", Code: "RCSA-A", Name: "Active RCSA",
		Purpose: "Active form.", Lifecycle: monitoring.Lifecycle{
			Status: monitoring.LifecycleActive, IsCurrent: true, Version: 1, CreatedBy: "risk-team", CreatedAt: now, UpdatedAt: now,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	base.FormTemplateID = "form-active"
	base.RiskIDs = []string{created.ID, created.ID}
	if _, err := service.Create(ctx, base); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate Risk accepted: %v", err)
	}
}

func TestAttachDistributionIsIdempotentAndSingleAssignment(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	cycle := Cycle{ID: "cycle-1", TenantID: "bank", LegalEntityID: "entity-a", Code: "RCSA", Name: "RCSA", CreatedAt: now}
	item := Item{ID: "item-1", CycleID: cycle.ID, TenantID: "bank", LegalEntityID: "entity-a", RiskID: "risk-1", RiskVersion: 1, RespondentPrincipalID: "owner-1", CreatedAt: now}
	if _, err := repo.CreateCycle(context.Background(), cycle, []Item{item}, nil); err != nil {
		t.Fatal(err)
	}
	first := DistributionLink{ItemID: item.ID, DistributionID: "distribution-1", IssuedAt: now}
	stored, err := repo.AttachDistribution(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity-a"}, first)
	if err != nil || stored.DistributionID != first.DistributionID {
		t.Fatalf("first attach=%#v err=%v", stored, err)
	}
	replayed, err := repo.AttachDistribution(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity-a"}, first)
	if err != nil || replayed.DistributionID != first.DistributionID {
		t.Fatalf("idempotent replay=%#v err=%v", replayed, err)
	}
	_, err = repo.AttachDistribution(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity-a"}, DistributionLink{ItemID: item.ID, DistributionID: "distribution-2", IssuedAt: now})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("second distribution replaced first: %v", err)
	}
}
