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
	if checkpoint.Position.Kind != "" || checkpoint.Attempts != 0 {
		t.Fatalf("unexpected initial checkpoint: %#v", checkpoint)
	}
	claims, err := service.Claim(ctx, "worker-a", now, time.Minute, 10)
	if err != nil || len(claims) != 1 {
		t.Fatalf("claims=%#v err=%v", claims, err)
	}
	claim := claims[0]
	position := CheckpointPosition{Kind: CheckpointCursor, Value: "cursor-100"}
	expectedEventID, err := CheckpointInboxEventID(claim, "source-consumer", position)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AdvanceAfterInbox(ctx, claim, "source-consumer", position, now.Add(10*time.Second), now.Add(time.Minute)); !errors.Is(err, ErrCheckpointProcessingProof) {
		t.Fatalf("checkpoint advanced without durable processing proof: %v", err)
	}

	wrongEventID, err := CheckpointInboxEventID(claim, "source-consumer", CheckpointPosition{Kind: CheckpointCursor, Value: "cursor-999"})
	if err != nil {
		t.Fatal(err)
	}
	proof.processedEventID = wrongEventID
	if _, err := service.AdvanceAfterInbox(ctx, claim, "source-consumer", position, now.Add(10*time.Second), now.Add(time.Minute)); !errors.Is(err, ErrCheckpointProcessingProof) {
		t.Fatalf("unrelated durable receipt authorized checkpoint advancement: %v", err)
	}

	proof.processedEventID = expectedEventID
	advanced, err := service.AdvanceAfterInbox(ctx, claim, "source-consumer", position, now.Add(10*time.Second), now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if advanced.Position != position || advanced.Attempts != 0 || advanced.LockedBy != "" || advanced.LeaseUntil != nil {
		t.Fatalf("checkpoint did not advance cleanly: %#v", advanced)
	}
	if _, err := repository.AdvanceBindingCheckpoint(ctx, claim, CheckpointPosition{Kind: CheckpointCursor, Value: "cursor-101"}, now.Add(20*time.Second), now.Add(time.Minute)); !errors.Is(err, ErrCheckpointClaimLost) {
		t.Fatalf("stale worker retained checkpoint authority: %v", err)
	}
}

func TestCheckpointLeaseExpiryReplaysSamePositionAndFailureBacksOff(t *testing.T) {
	ctx := context.Background()
	catalog, binding := memoryCheckpointCatalog(t)
	repository := NewMemoryCheckpointRepository(catalog)
	service := NewCheckpointService(repository, &checkpointInboxProof{})
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	if _, err := service.Ensure(ctx, binding.TenantID, binding.SourceID, binding.BindingID, binding.Version, now); err != nil {
		t.Fatal(err)
	}

	first, err := service.Claim(ctx, "worker-a", now, time.Minute, 10)
	if err != nil || len(first) != 1 {
		t.Fatalf("first claim=%#v err=%v", first, err)
	}
	if first[0].Position.Kind != "" {
		t.Fatalf("unexpected starting position: %#v", first[0].Position)
	}
	replayed, err := service.Claim(ctx, "worker-b", now.Add(2*time.Minute), time.Minute, 10)
	if err != nil || len(replayed) != 1 {
		t.Fatalf("expired lease was not replayed: claims=%#v err=%v", replayed, err)
	}
	if replayed[0].Position != first[0].Position || replayed[0].Attempts != 2 {
		t.Fatalf("replay skipped or reset state: first=%#v replay=%#v", first[0], replayed[0])
	}

	terminal, err := service.Fail(ctx, replayed[0], 3, "SOURCE_UNAVAILABLE", now.Add(2*time.Minute+10*time.Second), now.Add(5*time.Minute))
	if err != nil || terminal {
		t.Fatalf("nonterminal failure: terminal=%v err=%v", terminal, err)
	}
	if claims, err := service.Claim(ctx, "worker-c", now.Add(4*time.Minute), time.Minute, 10); err != nil || len(claims) != 0 {
		t.Fatalf("backoff was ignored: claims=%#v err=%v", claims, err)
	}
	claims, err := service.Claim(ctx, "worker-c", now.Add(6*time.Minute), time.Minute, 10)
	if err != nil || len(claims) != 1 || claims[0].Attempts != 3 {
		t.Fatalf("retry after backoff failed: claims=%#v err=%v", claims, err)
	}
	terminal, err = service.Fail(ctx, claims[0], 3, "SOURCE_UNAVAILABLE", now.Add(6*time.Minute+10*time.Second), now.Add(9*time.Minute))
	if err != nil || !terminal {
		t.Fatalf("checkpoint did not enter terminal failure: terminal=%v err=%v", terminal, err)
	}
	persisted, err := repository.BindingCheckpoint(ctx, binding.TenantID, binding.BindingID, binding.Version)
	if err != nil || persisted.FailedAt == nil || persisted.LastErrorCode != "SOURCE_UNAVAILABLE" {
		t.Fatalf("terminal failure not retained: checkpoint=%#v err=%v", persisted, err)
	}
}

func TestCheckpointLeaseGenerationFencesStaleClaimFromSameWorker(t *testing.T) {
	ctx := context.Background()
	catalog, binding := memoryCheckpointCatalog(t)
	repository := NewMemoryCheckpointRepository(catalog)
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	if _, err := repository.EnsureBindingCheckpoint(ctx, binding.TenantID, binding.SourceID, binding.BindingID, binding.Version, now); err != nil {
		t.Fatal(err)
	}
	first, err := repository.ClaimBindingCheckpoints(ctx, "worker-a", now, time.Minute, 1)
	if err != nil || len(first) != 1 {
		t.Fatalf("first claim=%#v err=%v", first, err)
	}
	second, err := repository.ClaimBindingCheckpoints(ctx, "worker-a", now.Add(2*time.Minute), time.Minute, 1)
	if err != nil || len(second) != 1 {
		t.Fatalf("same worker did not reclaim expired lease: claims=%#v err=%v", second, err)
	}
	if first[0].LeaseUntil == nil || second[0].LeaseUntil == nil || first[0].LeaseUntil.Equal(*second[0].LeaseUntil) {
		t.Fatalf("lease generation did not change: first=%#v second=%#v", first[0], second[0])
	}
	at := now.Add(2*time.Minute + 10*time.Second)
	if _, err := repository.AdvanceBindingCheckpoint(ctx, first[0], CheckpointPosition{Kind: CheckpointCursor, Value: "stale"}, at, now.Add(4*time.Minute)); !errors.Is(err, ErrCheckpointClaimLost) {
		t.Fatalf("stale same-worker claim advanced the newer lease: %v", err)
	}
	if _, err := repository.FailBindingCheckpoint(ctx, first[0], 3, "STALE_FAILURE", at, now.Add(4*time.Minute)); !errors.Is(err, ErrCheckpointClaimLost) {
		t.Fatalf("stale same-worker claim failed the newer lease: %v", err)
	}
	advanced, err := repository.AdvanceBindingCheckpoint(ctx, second[0], CheckpointPosition{Kind: CheckpointCursor, Value: "current"}, at, now.Add(4*time.Minute))
	if err != nil || advanced.Position.Value != "current" {
		t.Fatalf("current lease could not advance: checkpoint=%#v err=%v", advanced, err)
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
