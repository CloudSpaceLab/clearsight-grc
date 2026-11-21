package reporting

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMemoryReportRunIdempotency(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	scope := testScope()
	definition := activeTestDefinition(DefinitionActive)
	repo.definitions[definition.ID] = definition
	run := ReportRun{ID: "00000000-0000-7000-8000-000000000401", TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, DefinitionID: definition.ID, DefinitionVersion: definition.CurrentVersion, DefinitionCode: definition.Code, DefinitionChecksum: definition.StoredChecksum, ScopeKind: definition.ScopeKind, ScopeRef: definition.ScopeRef, RequestedByRef: testPerformerID, AsOf: serviceTestNow, Filter: cloneReportFilter(definition.Filter), Dataset: definition.Dataset, Format: definition.Format, Status: RunQueued, CreatedAt: serviceTestNow, ExpiresAt: serviceTestNow.Add(ReportRunRetention), SourceBoundary: testSourceBoundary(0)}
	key, fingerprint := "550e8400-e29b-41d4-a716-446655440000", strings.Repeat("a", 64)
	first, err := repo.CreateRunOnce(ctx, scope, run, key, fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	run.ID = "00000000-0000-7000-8000-000000000402"
	run.CreatedAt = run.CreatedAt.Add(time.Second)
	second, err := repo.CreateRunOnce(ctx, scope, run, key, fingerprint)
	if err != nil || second.ID != first.ID || len(repo.runs) != 1 {
		t.Fatalf("replay: %+v %v", second, err)
	}
	if _, err := repo.CreateRunOnce(ctx, scope, run, key, strings.Repeat("b", 64)); !errors.Is(err, ErrConflict) {
		t.Fatalf("different request: %v", err)
	}
	if _, err := repo.FindRunReceipt(ctx, scope, "other-person", key, fingerprint); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other actor: %v", err)
	}
}

func TestReportRunReceiptRecoversAfterDefinitionChanges(t *testing.T) {
	repo := NewMemoryRepository()
	definition := activeTestDefinition(DefinitionActive)
	repo.definitions[definition.ID] = definition
	service := NewService(repo, newServiceTestObjectStore(), nil)
	service.Now = func() time.Time { return serviceTestNow }
	input := CreateRunInput{Scope: testScope(), DefinitionID: definition.ID, ExpectedDefinitionVersion: definition.CurrentVersion, IdempotencyKey: "550e8400-e29b-41d4-a716-446655440001"}
	first, err := service.CreateRun(reportActorContext(testPerformerID), input)
	if err != nil {
		t.Fatal(err)
	}
	definition.Status = DefinitionRetired
	repo.definitions[definition.ID] = definition
	again, err := service.CreateRun(reportActorContext(testPerformerID), input)
	if err != nil || again.ID != first.ID {
		t.Fatalf("receipt: %+v %v", again, err)
	}
	input.Parameters.StartDate = "2026-01-01"
	input.Parameters.EndDate = "2026-02-01"
	if _, err := service.CreateRun(reportActorContext(testPerformerID), input); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed parameters: %v", err)
	}
}

func TestMemoryReportRunIdempotencyConcurrent(t *testing.T) {
	repo := NewMemoryRepository()
	definition := activeTestDefinition(DefinitionActive)
	repo.definitions[definition.ID] = definition
	service := NewService(repo, newServiceTestObjectStore(), nil)
	service.Now = func() time.Time { return serviceTestNow }
	input := CreateRunInput{Scope: testScope(), DefinitionID: definition.ID, ExpectedDefinitionVersion: definition.CurrentVersion, IdempotencyKey: "550e8400-e29b-41d4-a716-446655440002"}
	type result struct {
		value ReportRun
		err   error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			value, err := service.CreateRun(reportActorContext(testPerformerID), input)
			results <- result{value, err}
		}()
	}
	a, b := <-results, <-results
	if a.err != nil || b.err != nil || a.value.ID != b.value.ID || len(repo.runs) != 1 {
		t.Fatalf("concurrent receipts: %+v %+v", a, b)
	}
}
