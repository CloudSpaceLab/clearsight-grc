package risk

import (
	"context"
	"testing"
	"time"
)

func TestIndicatorPopulationCountsSharedCheckOnce(t *testing.T) {
	service := NewService(NewMemoryRepository())
	service.Now = func() time.Time { return time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC) }
	service.ConfigureIndicatorLinkValidator(func(_ context.Context, _ Scope, checkID string, version int64) (string, error) {
		if checkID == "check-shared" && version == 3 {
			return "program-1", nil
		}
		return "", ErrInvalid
	})
	ctx := context.Background()
	first, err := service.Create(ctx, CreateInput{
		TenantID: "bank", LegalEntityID: "entity", OrganizationScopeID: "branch-a",
		Code: "R-1", Name: "First", Statement: "First risk", Impact: "Impact", ActorID: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(ctx, CreateInput{
		TenantID: "bank", LegalEntityID: "entity", OrganizationScopeID: "branch-b",
		Code: "R-2", Name: "Second", Statement: "Second risk", Impact: "Impact", ActorID: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []Risk{first, second} {
		if _, _, err = service.LinkIndicator(ctx, LinkIndicatorInput{
			TenantID: "bank", LegalEntityID: "entity", RiskID: value.ID, ExpectedRiskVersion: value.Version,
			MonitoringCheckID: "check-shared", MonitoringCheckVersion: 3, Kind: IndicatorKRI, ActorID: "owner",
		}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := service.ListIndicatorPopulation(ctx, Scope{TenantID: "bank", LegalEntityID: "entity"}, IndicatorPopulationFilter{Kind: IndicatorKRI, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].RiskCount != 2 || len(page.Items[0].Risks) != 2 {
		t.Fatalf("population=%#v", page)
	}
	if page.Items[0].Link.MonitoringCheckID != "check-shared" || page.Items[0].KindConflict {
		t.Fatalf("indicator identity=%#v", page.Items[0])
	}
}

func TestIndicatorPopulationHonorsOrganizationDescendantSelection(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewService(repository)
	service.Now = func() time.Time { return time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC) }
	service.ConfigureIndicatorLinkValidator(func(_ context.Context, _ Scope, checkID string, version int64) (string, error) {
		return "program-1", nil
	})
	ctx := context.Background()
	for index, scopeID := range []string{"head-office", "branch-a"} {
		record, err := service.Create(ctx, CreateInput{
			TenantID: "bank", LegalEntityID: "entity", OrganizationScopeID: scopeID,
			Code: "R-" + string(rune('1'+index)), Name: scopeID, Statement: scopeID + " risk", Impact: "Impact", ActorID: "owner",
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err = service.LinkIndicator(ctx, LinkIndicatorInput{
			TenantID: "bank", LegalEntityID: "entity", RiskID: record.ID, ExpectedRiskVersion: record.Version,
			MonitoringCheckID: "check-" + scopeID, MonitoringCheckVersion: 1, Kind: IndicatorKRI, ActorID: "owner",
		}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := service.ListIndicatorPopulation(ctx, Scope{TenantID: "bank", LegalEntityID: "entity"}, IndicatorPopulationFilter{
		Kind: IndicatorKRI, OrganizationScopeID: "head-office", OrganizationScopeIDs: []string{"head-office", "branch-a"}, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.OrganizationScopeID != "head-office" {
		t.Fatalf("scoped population=%#v", page)
	}
}

func TestIndicatorPopulationCursorIsStableAcrossPages(t *testing.T) {
	service := NewService(NewMemoryRepository())
	service.Now = func() time.Time { return time.Date(2026, 10, 6, 8, 0, 0, 0, time.UTC) }
	service.ConfigureIndicatorLinkValidator(func(_ context.Context, _ Scope, _ string, _ int64) (string, error) {
		return "program-1", nil
	})
	ctx := context.Background()
	for index, checkID := range []string{"check-1", "check-2", "check-3"} {
		record, err := service.Create(ctx, CreateInput{
			TenantID: "bank", LegalEntityID: "entity",
			Code: "CURSOR-" + string(rune('1'+index)), Name: checkID,
			Statement: checkID + " risk", Impact: "Impact", ActorID: "owner",
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err = service.LinkIndicator(ctx, LinkIndicatorInput{
			TenantID: "bank", LegalEntityID: "entity", RiskID: record.ID, ExpectedRiskVersion: record.Version,
			MonitoringCheckID: checkID, MonitoringCheckVersion: 1, Kind: IndicatorKRI, ActorID: "owner",
		}); err != nil {
			t.Fatal(err)
		}
	}

	first, err := service.ListIndicatorPopulation(ctx, Scope{TenantID: "bank", LegalEntityID: "entity"}, IndicatorPopulationFilter{Kind: IndicatorKRI, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || first.Items[0].Link.MonitoringCheckID != "check-3" || first.NextCursor == "" || !first.Truncated {
		t.Fatalf("first page=%#v", first)
	}
	second, err := service.ListIndicatorPopulation(ctx, Scope{TenantID: "bank", LegalEntityID: "entity"}, IndicatorPopulationFilter{Kind: IndicatorKRI, Limit: 1, Cursor: first.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].Link.MonitoringCheckID != "check-2" || second.NextCursor == "" || !second.Truncated {
		t.Fatalf("second page=%#v", second)
	}
	third, err := service.ListIndicatorPopulation(ctx, Scope{TenantID: "bank", LegalEntityID: "entity"}, IndicatorPopulationFilter{Kind: IndicatorKRI, Limit: 1, Cursor: second.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(third.Items) != 1 || third.Items[0].Link.MonitoringCheckID != "check-1" || third.NextCursor != "" || third.Truncated {
		t.Fatalf("third page=%#v", third)
	}
	if _, err := service.ListIndicatorPopulation(ctx, Scope{TenantID: "bank", LegalEntityID: "entity"}, IndicatorPopulationFilter{Cursor: "not-a-cursor"}); err != ErrInvalid {
		t.Fatalf("invalid cursor error=%v", err)
	}
}
