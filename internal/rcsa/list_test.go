package rcsa

import (
	"context"
	"testing"
	"time"
)

func TestCycleListUsesFrozenPopulationCountsAndStablePagination(t *testing.T) {
	repository := NewMemoryRepository()
	now := time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)
	for index, id := range []string{
		"11111111-1111-7111-8111-111111111111",
		"22222222-2222-7222-8222-222222222222",
		"33333333-3333-7333-8333-333333333333",
	} {
		cycle := Cycle{
			ID: id, TenantID: "bank", LegalEntityID: "entity-a",
			Code: "RCSA-" + string(rune('1'+index)), Name: "Cycle " + string(rune('1'+index)),
			TriggerKind: TriggerScheduled, FirstLineOwnerID: "owner-1", Status: StatusDraft,
			PopulationChecksum: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Version: 1, CreatedAt: now, UpdatedAt: now.Add(time.Duration(index) * time.Minute),
		}
		risks := []RiskSnapshot{{CycleID: id, RiskID: "risk-" + id, RiskVersion: 1, Code: "R", Name: "Risk"}}
		controls := []ControlSnapshot{}
		if index == 2 {
			controls = append(controls, ControlSnapshot{CycleID: id, RiskID: risks[0].RiskID})
		}
		if _, err := repository.Create(context.Background(), cycle, risks, controls, Event{}); err != nil {
			t.Fatal(err)
		}
	}
	service := NewService(repository, nil)
	first, err := service.List(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity-a"}, CycleFilter{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 2 || first.Items[0].Cycle.ID != "33333333-3333-7333-8333-333333333333" || first.NextCursor == "" {
		t.Fatalf("first page=%#v", first)
	}
	if first.Items[0].RiskCount != 1 || first.Items[0].ControlCount != 1 {
		t.Fatalf("frozen counts=%#v", first.Items[0])
	}
	second, err := service.List(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity-a"}, CycleFilter{Limit: 2, Cursor: first.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Items[0].Cycle.ID != "11111111-1111-7111-8111-111111111111" || second.NextCursor != "" {
		t.Fatalf("second page=%#v", second)
	}
}

func TestCycleListFiltersStatusAndLegalEntity(t *testing.T) {
	repository := NewMemoryRepository()
	now := time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)
	for _, value := range []Cycle{
		{ID: "11111111-1111-7111-8111-111111111111", TenantID: "bank", LegalEntityID: "entity-a", Code: "A", Name: "A", TriggerKind: TriggerManual, FirstLineOwnerID: "owner-a", Status: StatusAssessmentOpen, PopulationChecksum: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Version: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "22222222-2222-7222-8222-222222222222", TenantID: "bank", LegalEntityID: "entity-a", Code: "B", Name: "B", TriggerKind: TriggerManual, FirstLineOwnerID: "owner-b", Status: StatusCompleted, PopulationChecksum: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Version: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "33333333-3333-7333-8333-333333333333", TenantID: "bank", LegalEntityID: "entity-b", Code: "C", Name: "C", TriggerKind: TriggerManual, FirstLineOwnerID: "owner-c", Status: StatusAssessmentOpen, PopulationChecksum: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Version: 1, CreatedAt: now, UpdatedAt: now},
	} {
		if _, err := repository.Create(context.Background(), value, nil, nil, Event{}); err != nil {
			t.Fatal(err)
		}
	}
	service := NewService(repository, nil)
	page, err := service.List(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity-a"}, CycleFilter{Status: StatusAssessmentOpen, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Cycle.ID != "11111111-1111-7111-8111-111111111111" {
		t.Fatalf("filtered page=%#v", page)
	}
	if _, err := service.List(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity-a"}, CycleFilter{Cursor: "invalid"}); err != ErrInvalid {
		t.Fatalf("invalid cursor error=%v", err)
	}
	if _, err := service.List(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity-a"}, CycleFilter{Status: Status("NOT_A_STATE")}); err != ErrInvalid {
		t.Fatalf("invalid status error=%v", err)
	}
}
