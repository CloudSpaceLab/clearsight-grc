package evidence

import (
	"context"
	"testing"
	"time"
)

func TestScopedSourceHealthWorstCurrentScopeWinsAndRecovers(t *testing.T) {
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	source := Source{
		ID: "source-1", TenantID: "bank", Code: "CORE", Name: "Core", Type: SourceSystem,
		AuthorityClass: "SYSTEM_OF_RECORD", ExpectedFreshnessMinutes: 30,
		Health: HealthUnknown, Status: SourceActive, Version: 1, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	}
	repo := NewMemoryRepository([]Source{source}, nil)
	service := NewService(repo, NewMemoryObjectStore())
	service.now = func() time.Time { return now }

	connection := SourceObservation{
		TenantID: "bank", SourceID: source.ID, Scope: ObservationScopeConnection,
		ConnectionID: "connection-1", ConnectionVersion: 3, ObservedAt: now, Success: true,
	}
	if updated, err := service.RecordSourceObservation(context.Background(), connection); err != nil || updated.Health != HealthCurrent {
		t.Fatalf("healthy Connection: source=%#v err=%v", updated, err)
	}

	bindingFailure := SourceObservation{
		TenantID: "bank", SourceID: source.ID, Scope: ObservationScopeBinding,
		ConnectionID: "connection-1", ConnectionVersion: 3,
		ViewID: "view-1", ViewVersion: 5, BindingID: "binding-1", BindingVersion: 2,
		ObservedAt: now.Add(time.Minute), Unavailable: true,
	}
	now = now.Add(time.Minute)
	updated, err := service.RecordSourceObservation(context.Background(), bindingFailure)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Health != HealthUnavailable {
		t.Fatalf("Binding outage was hidden by healthy Connection: %#v", updated)
	}

	now = now.Add(time.Minute)
	recovery := bindingFailure
	recovery.ObservedAt = now
	recovery.Unavailable = false
	recovery.Success = true
	updated, err = service.RecordSourceObservation(context.Background(), recovery)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Health != HealthCurrent {
		t.Fatalf("source did not recover after all scopes recovered: %#v", updated)
	}

	scopes, err := service.ListSourceScopeHealth(context.Background(), "bank", source.ID, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(scopes) != 2 {
		t.Fatalf("scope health count=%d want=2: %#v", len(scopes), scopes)
	}
}

func TestScopedSourceHealthIgnoresOutOfOrderOlderObservation(t *testing.T) {
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	source := Source{ID: "source-1", TenantID: "bank", Code: "CORE", Name: "Core", Type: SourceSystem, AuthorityClass: "SYSTEM_OF_RECORD", ExpectedFreshnessMinutes: 60, Health: HealthUnknown, Status: SourceActive, Version: 1, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)}
	repo := NewMemoryRepository([]Source{source}, nil)
	service := NewService(repo, NewMemoryObjectStore())
	service.now = func() time.Time { return now }

	newer := SourceObservation{TenantID: "bank", SourceID: source.ID, Scope: ObservationScopeConnection, ConnectionID: "connection-1", ConnectionVersion: 1, ObservedAt: now, Success: true}
	if _, err := service.RecordSourceObservation(context.Background(), newer); err != nil {
		t.Fatal(err)
	}
	older := newer
	older.ObservedAt = now.Add(-10 * time.Minute)
	older.Success = false
	now = now.Add(time.Minute)
	updated, err := service.RecordSourceObservation(context.Background(), older)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Health != HealthCurrent {
		t.Fatalf("older failure rolled current health backward: %#v", updated)
	}
	if updated.LastObservedAt == nil || !updated.LastObservedAt.Equal(newer.ObservedAt) {
		t.Fatalf("last observed time moved backward: %#v", updated.LastObservedAt)
	}
}

func TestScopedSourceHealthMaintenanceMarksSuccessfulScopesStale(t *testing.T) {
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	source := Source{ID: "source-1", TenantID: "bank", Code: "CORE", Name: "Core", Type: SourceSystem, AuthorityClass: "SYSTEM_OF_RECORD", ExpectedFreshnessMinutes: 5, Health: HealthUnknown, Status: SourceActive, Version: 1, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)}
	repo := NewMemoryRepository([]Source{source}, nil)
	service := NewService(repo, NewMemoryObjectStore())
	service.now = func() time.Time { return now }
	observation := SourceObservation{TenantID: "bank", SourceID: source.ID, Scope: ObservationScopeSource, ObservedAt: now, Success: true}
	if updated, err := service.RecordSourceObservation(context.Background(), observation); err != nil || updated.Health != HealthCurrent {
		t.Fatalf("initial observation: source=%#v err=%v", updated, err)
	}

	now = now.Add(6 * time.Minute)
	count, err := service.Maintain(context.Background(), now, 20)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("maintenance changed=%d want=1", count)
	}
	values, err := service.ListSources(context.Background(), "bank", 20)
	if err != nil || len(values) != 1 || values[0].Health != HealthStale {
		t.Fatalf("stale source not aggregated: values=%#v err=%v", values, err)
	}
}

func TestScopedObservationShapeIsExplicit(t *testing.T) {
	_, err := normalizeSourceObservationScope(SourceObservation{
		TenantID: "bank", SourceID: "source", Scope: ObservationScopeBinding,
		BindingID: "binding", BindingVersion: 1,
	})
	if err == nil {
		t.Fatal("Binding observation without exact Connection/View lineage was accepted")
	}
}
