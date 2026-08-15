package sourceevent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func TestCheckpointProjectorRecoversWatermarkFromAcceptedOutbox(t *testing.T) {
	ctx := context.Background()
	runtimeRepo := runtime.NewMemoryRepository()
	checkpointRepo := &checkpointRepo{}
	checkpoints := sourceaccess.NewCheckpointService(checkpointRepo, runtimeRepo)
	now := time.Date(2026, 8, 15, 13, 0, 0, 0, time.UTC)
	checkpoint, err := checkpoints.Ensure(ctx, "tenant-a", "source-a", "binding-a", 1, now)
	if err != nil {
		t.Fatal(err)
	}
	position := sourceaccess.CheckpointPosition{Kind: sourceaccess.CheckpointWatermark, Value: "42"}
	providerID := "provider-event-proof"
	transitionID, err := sourceaccess.CheckpointInboxEventID(checkpoint, checkpointConsumer, position)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(checkpointProjectionPayload{SourceID: "source-a", BindingID: "binding-a", BindingVersion: "1", Position: position})
	outbox := runtime.OutboxEvent{ID: "event-a", TenantID: "tenant-a", AggregateType: "SOURCE_BINDING", AggregateID: "binding-a", EventType: "SourceBindingChanged", Payload: payload, OccurredAt: now}
	created, err := runtimeRepo.RecordInboxWithOutbox(ctx, []runtime.InboxReceipt{
		{TenantID: "tenant-a", Consumer: providerConsumer, EventID: providerID},
		{TenantID: "tenant-a", Consumer: checkpointConsumer, EventID: transitionID},
	}, outbox, now)
	if err != nil || !created {
		t.Fatalf("accepted event was not staged: created=%v err=%v", created, err)
	}
	if checkpointRepo.value.Position.Kind != "" {
		t.Fatalf("fixture unexpectedly advanced checkpoint: %#v", checkpointRepo.value)
	}
	projector := NewCheckpointProjector(runtimeRepo, checkpoints)
	projector.now = func() time.Time { return now.Add(time.Second) }
	if err := projector.Publish(ctx, outbox); err != nil {
		t.Fatal(err)
	}
	if checkpointRepo.value.Position != position || checkpointRepo.value.Generation != 1 {
		t.Fatalf("outbox did not recover checkpoint: %#v", checkpointRepo.value)
	}
}

func TestCheckpointProjectorDoesNotRegressEventID(t *testing.T) {
	ctx := context.Background()
	runtimeRepo := runtime.NewMemoryRepository()
	checkpointRepo := &checkpointRepo{value: sourceaccess.BindingCheckpoint{
		TenantID: "tenant-a", SourceID: "source-a", BindingID: "binding-a", BindingVersion: 1,
		Position: sourceaccess.CheckpointPosition{Kind: sourceaccess.CheckpointEventID, Value: "evt-new"}, Generation: 2,
	}}
	projector := NewCheckpointProjector(runtimeRepo, sourceaccess.NewCheckpointService(checkpointRepo, runtimeRepo))
	payload, _ := json.Marshal(checkpointProjectionPayload{SourceID: "source-a", BindingID: "binding-a", BindingVersion: "1", Position: sourceaccess.CheckpointPosition{Kind: sourceaccess.CheckpointEventID, Value: "evt-old"}})
	event := runtime.OutboxEvent{TenantID: "tenant-a", AggregateType: "SOURCE_BINDING", AggregateID: "binding-a", EventType: "SourceBindingChanged", Payload: payload}
	if err := projector.Publish(ctx, event); err != nil {
		t.Fatal(err)
	}
	if checkpointRepo.value.Position.Value != "evt-new" || checkpointRepo.value.Generation != 2 {
		t.Fatalf("old event projection regressed checkpoint: %#v", checkpointRepo.value)
	}
}

func TestCheckpointProjectorIgnoresUnrelatedOutboxEvents(t *testing.T) {
	projector := NewCheckpointProjector(runtime.NewMemoryRepository(), sourceaccess.NewCheckpointService(&checkpointRepo{}, runtime.NewMemoryRepository()))
	if err := projector.Publish(context.Background(), runtime.OutboxEvent{EventType: "MatterChanged"}); err != nil {
		t.Fatal(err)
	}
}
