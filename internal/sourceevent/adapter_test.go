package sourceevent

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"testing"
	"time"
)

type checkpointRepo struct {
	value sourceaccess.BindingCheckpoint
}

func (r *checkpointRepo) EnsureBindingCheckpoint(_ context.Context, tenant, source, binding string, version int64, now time.Time) (sourceaccess.BindingCheckpoint, error) {
	if r.value.BindingID == "" {
		r.value = sourceaccess.BindingCheckpoint{TenantID: tenant, SourceID: source, BindingID: binding, BindingVersion: version, CreatedAt: now, UpdatedAt: now}
	}
	return r.value, nil
}
func (r *checkpointRepo) BindingCheckpoint(_ context.Context, tenant, binding string, version int64) (sourceaccess.BindingCheckpoint, error) {
	return r.value, nil
}
func (r *checkpointRepo) AdvanceBindingCheckpoint(_ context.Context, expected sourceaccess.BindingCheckpoint, position sourceaccess.CheckpointPosition, at time.Time) (sourceaccess.BindingCheckpoint, error) {
	if r.value.Generation != expected.Generation || r.value.Position != expected.Position {
		return sourceaccess.BindingCheckpoint{}, sourceaccess.ErrCheckpointConflict
	}
	r.value.Position = position
	r.value.Generation++
	r.value.UpdatedAt = at
	return r.value, nil
}

func TestWatermarkCaptureIsReplaySafeAndMonotonic(t *testing.T) {
	ctx := context.Background()
	runtimeRepo := runtime.NewMemoryRepository()
	checkpointsRepo := &checkpointRepo{}
	adapter := NewAdapter(runtimeRepo, sourceaccess.NewCheckpointService(checkpointsRepo, runtimeRepo))
	adapter.now = func() time.Time { return time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC) }
	s := mustSession(t, adapter)
	view, binding := eventContracts(t, s, sourceaccess.CheckpointWatermark)
	event := sourceaccess.ChangeEvent{EventID: "evt-1", Position: &sourceaccess.CheckpointPosition{Kind: sourceaccess.CheckpointWatermark, Value: "10"}, Payload: json.RawMessage(`{"account_id":"A1","risk_score":7.5}`)}
	first, err := s.CaptureChange(ctx, view, binding, event)
	if err != nil || !first.Accepted || first.Duplicate {
		t.Fatalf("first=%#v err=%v", first, err)
	}
	duplicate, err := s.CaptureChange(ctx, view, binding, event)
	if err != nil || !duplicate.Duplicate {
		t.Fatalf("duplicate=%#v err=%v", duplicate, err)
	}
	if checkpointsRepo.value.Position.Value != "10" || checkpointsRepo.value.Generation != 1 {
		t.Fatalf("checkpoint=%#v", checkpointsRepo.value)
	}
	stale := sourceaccess.ChangeEvent{EventID: "evt-2", Position: &sourceaccess.CheckpointPosition{Kind: sourceaccess.CheckpointWatermark, Value: "9"}, Payload: event.Payload}
	if _, err := s.CaptureChange(ctx, view, binding, stale); !errors.Is(err, sourceaccess.ErrCheckpointConflict) {
		t.Fatalf("stale watermark err=%v", err)
	}
	claimed, err := runtimeRepo.ClaimOutbox(ctx, "test", time.Now().UTC(), time.Minute, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 1 {
		t.Fatalf("outbox events=%d want 1", len(claimed))
	}
}
func TestDuplicateEventIDDoesNotRegressLaterCheckpoint(t *testing.T) {
	ctx := context.Background()
	runtimeRepo := runtime.NewMemoryRepository()
	checkpointsRepo := &checkpointRepo{}
	s := mustSession(t, NewAdapter(runtimeRepo, sourceaccess.NewCheckpointService(checkpointsRepo, runtimeRepo)))
	view, binding := eventContracts(t, s, sourceaccess.CheckpointEventID)
	for _, eventID := range []string{"evt-a", "evt-b"} {
		if _, err := s.CaptureChange(ctx, view, binding, sourceaccess.ChangeEvent{EventID: eventID, Payload: json.RawMessage(`{"account_id":"A1","risk_score":1}`)}); err != nil {
			t.Fatal(err)
		}
	}
	if checkpointsRepo.value.Position.Value != "evt-b" {
		t.Fatalf("checkpoint=%#v", checkpointsRepo.value)
	}
	result, err := s.CaptureChange(ctx, view, binding, sourceaccess.ChangeEvent{EventID: "evt-a", Payload: json.RawMessage(`{"account_id":"A1","risk_score":1}`)})
	if err != nil || !result.Duplicate {
		t.Fatalf("replay=%#v err=%v", result, err)
	}
	if checkpointsRepo.value.Position.Value != "evt-b" {
		t.Fatalf("duplicate regressed checkpoint=%#v", checkpointsRepo.value)
	}
}
func TestWebhookRejectsSchemaDrift(t *testing.T) {
	runtimeRepo := runtime.NewMemoryRepository()
	checkpointsRepo := &checkpointRepo{}
	s := mustSession(t, NewAdapter(runtimeRepo, sourceaccess.NewCheckpointService(checkpointsRepo, runtimeRepo)))
	view, binding := eventContracts(t, s, sourceaccess.CheckpointEventID)
	_, err := s.CaptureChange(context.Background(), view, binding, sourceaccess.ChangeEvent{EventID: "evt-x", Payload: json.RawMessage(`{"account_id":"A1","risk_score":1,"unexpected":true}`)})
	if !errors.Is(err, sourceaccess.ErrSchemaDrift) {
		t.Fatalf("schema drift err=%v", err)
	}
}
func mustSession(t *testing.T, adapter *Adapter) *session {
	t.Helper()
	connection := sourceaccess.Connection{TenantID: "tenant-a", ID: "connection-a", SourceID: "source-a", Version: "1", AdapterKind: sourceaccess.AdapterWebhookEvent, AdapterVersion: sourceaccess.WebhookEventAdapterVersion, Definition: json.RawMessage(`{}`)}
	value, err := adapter.Open(context.Background(), connection, sourceaccess.EnvironmentSecretResolver{})
	if err != nil {
		t.Fatal(err)
	}
	return value.(*session)
}
func eventContracts(t *testing.T, s *session, mode sourceaccess.CheckpointPositionKind) (sourceaccess.View, sourceaccess.Binding) {
	t.Helper()
	view := sourceaccess.View{ID: "view-a", ConnectionID: "connection-a", Version: "1", OutputKind: sourceaccess.OutputRecords, Definition: json.RawMessage(`{"position_kind":"` + string(mode) + `","fields":[{"name":"account_id","native_type":"json:string","nullable":false},{"name":"risk_score","native_type":"json:number","nullable":false}]}`)}
	inspected, err := s.Inspect(context.Background(), view)
	if err != nil {
		t.Fatal(err)
	}
	view.NativeSchema, view.SchemaFingerprint = inspected.Fields, inspected.Receipt.SchemaFingerprint
	binding := sourceaccess.Binding{ID: "binding-a", ViewID: "view-a", Version: "1", Purpose: "event", Operations: []sourceaccess.Operation{sourceaccess.OperationChanges}, SelectedFields: []string{"account_id", "risk_score"}, Limits: sourceaccess.DefaultResourceLimits()}
	return view, binding
}

func TestDuplicateProviderEventCannotMoveWatermarkOrRepublish(t *testing.T) {
	ctx := context.Background()
	runtimeRepo := runtime.NewMemoryRepository()
	checkpointRepo := &checkpointRepo{}
	s := mustSession(t, NewAdapter(runtimeRepo, sourceaccess.NewCheckpointService(checkpointRepo, runtimeRepo)))
	view, binding := eventContracts(t, s, sourceaccess.CheckpointWatermark)
	first := sourceaccess.ChangeEvent{EventID: "evt-fixed", Position: &sourceaccess.CheckpointPosition{Kind: sourceaccess.CheckpointWatermark, Value: "10"}, Payload: json.RawMessage(`{"account_id":"A1","risk_score":1}`)}
	if _, err := s.CaptureChange(ctx, view, binding, first); err != nil {
		t.Fatal(err)
	}
	replay := sourceaccess.ChangeEvent{EventID: "evt-fixed", Position: &sourceaccess.CheckpointPosition{Kind: sourceaccess.CheckpointWatermark, Value: "99"}, Payload: json.RawMessage(`{"account_id":"ALTERED","risk_score":999}`)}
	result, err := s.CaptureChange(ctx, view, binding, replay)
	if err != nil || !result.Accepted || !result.Duplicate {
		t.Fatalf("replay=%#v err=%v", result, err)
	}
	if checkpointRepo.value.Position.Value != "10" || checkpointRepo.value.Generation != 1 {
		t.Fatalf("duplicate provider ID moved checkpoint: %#v", checkpointRepo.value)
	}
	claimed, err := runtimeRepo.ClaimOutbox(ctx, "audit", time.Now().UTC(), time.Minute, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 1 {
		t.Fatalf("duplicate provider ID published %d outbox events; want 1", len(claimed))
	}
}
