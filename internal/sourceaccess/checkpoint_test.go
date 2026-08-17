package sourceaccess

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCheckpointAdvancesOnlyAfterMatchingDurableProcessingReceipt(t *testing.T) {
	ctx := context.Background()
	catalog, binding := memoryCheckpointCatalog(t)
	repository := NewMemoryCheckpointRepository(catalog)
	proof := &checkpointInboxProof{}
	service := NewCheckpointService(repository, proof)
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)

	checkpoint, err := service.Ensure(ctx, binding.TenantID, binding.SourceID, binding.BindingID, binding.Version, now)
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.Position.Kind != "" || checkpoint.Generation != 0 {
		t.Fatalf("unexpected initial checkpoint: %#v", checkpoint)
	}
	position := CheckpointPosition{Kind: CheckpointCursor, Value: "cursor-100"}
	expectedEventID, err := CheckpointInboxEventID(checkpoint, "source-consumer", position)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AdvanceAfterInbox(ctx, checkpoint, "source-consumer", position, now.Add(10*time.Second)); !errors.Is(err, ErrCheckpointProcessingProof) {
		t.Fatalf("checkpoint advanced without durable processing proof: %v", err)
	}

	wrongEventID, err := CheckpointInboxEventID(checkpoint, "source-consumer", CheckpointPosition{Kind: CheckpointCursor, Value: "cursor-999"})
	if err != nil {
		t.Fatal(err)
	}
	proof.processedEventID = wrongEventID
	if _, err := service.AdvanceAfterInbox(ctx, checkpoint, "source-consumer", position, now.Add(10*time.Second)); !errors.Is(err, ErrCheckpointProcessingProof) {
		t.Fatalf("unrelated durable receipt authorized checkpoint advancement: %v", err)
	}

	proof.processedEventID = expectedEventID
	advanced, err := service.AdvanceAfterInbox(ctx, checkpoint, "source-consumer", position, now.Add(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if advanced.Position != position || advanced.Generation != 1 {
		t.Fatalf("checkpoint did not advance cleanly: %#v", advanced)
	}
	if _, err := repository.AdvanceBindingCheckpoint(ctx, checkpoint, CheckpointPosition{Kind: CheckpointCursor, Value: "cursor-101"}, now.Add(20*time.Second)); !errors.Is(err, ErrCheckpointConflict) {
		t.Fatalf("stale checkpoint snapshot retained write authority: %v", err)
	}
}

func TestCheckpointEventIdentityIsStableForReplayAndChangesAfterAdvance(t *testing.T) {
	ctx := context.Background()
	catalog, binding := memoryCheckpointCatalog(t)
	repository := NewMemoryCheckpointRepository(catalog)
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	checkpoint, err := repository.EnsureBindingCheckpoint(ctx, binding.TenantID, binding.SourceID, binding.BindingID, binding.Version, now)
	if err != nil {
		t.Fatal(err)
	}
	position := CheckpointPosition{Kind: CheckpointWatermark, Value: "2026-08-15T06:00:00Z"}
	first, err := CheckpointInboxEventID(checkpoint, "source-consumer", position)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CheckpointInboxEventID(checkpoint, "source-consumer", position)
	if err != nil || second != first {
		t.Fatalf("same source batch changed replay identity: first=%q second=%q err=%v", first, second, err)
	}
	advanced, err := repository.AdvanceBindingCheckpoint(ctx, checkpoint, position, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	nextPosition := CheckpointPosition{Kind: CheckpointWatermark, Value: "2026-08-15T06:05:00Z"}
	next, err := CheckpointInboxEventID(advanced, "source-consumer", nextPosition)
	if err != nil {
		t.Fatal(err)
	}
	if next == first {
		t.Fatal("different checkpoint generation reused an earlier event identity")
	}
}

func TestCheckpointRequiresCurrentActiveStatefulBinding(t *testing.T) {
	ctx := context.Background()
	catalog, binding := memoryCheckpointCatalog(t)
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	stateless := binding
	stateless.RevisionID = "binding-stateless-revision"
	stateless.BindingID = "binding-stateless"
	stateless.Code = "STATELESS"
	stateless.Operations = []Operation{OperationAggregate}
	stateless.KeyFields = nil
	if _, err := catalog.CreateBindingRevision(ctx, stateless); err != nil {
		t.Fatal(err)
	}
	repository := NewMemoryCheckpointRepository(catalog)
	if _, err := repository.EnsureBindingCheckpoint(ctx, stateless.TenantID, stateless.SourceID, stateless.BindingID, stateless.Version, now); !errors.Is(err, ErrCatalogInvalid) {
		t.Fatalf("stateless Binding received a checkpoint: %v", err)
	}
}

func memoryCheckpointCatalog(t *testing.T) (*MemoryCatalogRepository, BindingRevision) {
	t.Helper()
	now := time.Date(2026, 8, 15, 5, 0, 0, 0, time.UTC)
	effective := now.Add(-time.Hour)
	catalog := NewMemoryCatalogRepository([]SourceScope{{TenantID: "bank", SourceID: "source-1"}})
	connection := ConnectionRevision{
		RevisionID: "connection-revision", ConnectionID: "connection-1", TenantID: "bank", SourceID: "source-1",
		Code: "CORE", Name: "Core", AdapterKind: AdapterPostgres, AdapterVersion: PostgresAdapterVersion,
		SecretRef: "vault://bank/core", DeclaredCapabilities: []Capability{CapabilityPage, CapabilityAggregate}, VerifiedCapabilities: []Capability{CapabilityPage, CapabilityAggregate},
		RevisionLifecycle: RevisionLifecycle{Status: RevisionActive, IsCurrent: true, EffectiveFrom: &effective, Version: 1, CreatedAt: effective, UpdatedAt: effective},
	}
	if _, err := catalog.CreateConnectionRevision(context.Background(), connection); err != nil {
		t.Fatal(err)
	}
	view := ViewRevision{
		RevisionID: "view-revision", ViewID: "view-1", TenantID: "bank", SourceID: "source-1", ConnectionID: connection.ConnectionID, ConnectionVersion: 1,
		Code: "ACCOUNTS", Name: "Accounts", Definition: []byte(`{"query":"SELECT account_id FROM accounts"}`), OutputKind: OutputRecords,
		StableKeys: []string{"account_id"}, NativeSchema: []NativeField{{Name: "account_id", NativeType: "uuid"}}, SchemaFingerprint: strings.Repeat("a", 64),
		RevisionLifecycle: RevisionLifecycle{Status: RevisionActive, IsCurrent: true, EffectiveFrom: &effective, Version: 1, CreatedAt: effective, UpdatedAt: effective},
	}
	if _, err := catalog.CreateViewRevision(context.Background(), view); err != nil {
		t.Fatal(err)
	}
	binding := BindingRevision{
		RevisionID: "binding-revision", BindingID: "binding-1", TenantID: "bank", SourceID: "source-1", ViewID: view.ViewID, ViewVersion: 1,
		Code: "ACCOUNT_PAGE", Name: "Account page", Purpose: "assurance", Operations: []Operation{OperationPage},
		SelectedFields: []string{"account_id"}, KeyFields: []string{"account_id"}, Limits: DefaultResourceLimits(), Completeness: CompletenessRequireFull,
		RevisionLifecycle: RevisionLifecycle{Status: RevisionActive, IsCurrent: true, EffectiveFrom: &effective, Version: 1, CreatedAt: effective, UpdatedAt: effective},
	}
	if _, err := catalog.CreateBindingRevision(context.Background(), binding); err != nil {
		t.Fatal(err)
	}
	return catalog, binding
}

type checkpointInboxProof struct{ processedEventID string }

func (p *checkpointInboxProof) InboxProcessed(_ context.Context, _, _, eventID string) (bool, error) {
	return eventID != "" && eventID == p.processedEventID, nil
}
