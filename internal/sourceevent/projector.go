package sourceevent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type checkpointProjectionPayload struct {
	SourceID       string                          `json:"source_id"`
	BindingID      string                          `json:"binding_id"`
	BindingVersion string                          `json:"binding_version"`
	Position       sourceaccess.CheckpointPosition `json:"position"`
}

// CheckpointProjector makes webhook checkpoint convergence part of the existing
// retrying outbox-delivery path. It owns no queue and stores no source rows.
type CheckpointProjector struct {
	adapter *Adapter
	now     func() time.Time
}

func NewCheckpointProjector(store RuntimeStore, checkpoints *sourceaccess.CheckpointService) *CheckpointProjector {
	return &CheckpointProjector{adapter: NewAdapter(store, checkpoints), now: time.Now}
}

func (p *CheckpointProjector) Publish(ctx context.Context, event runtime.OutboxEvent) error {
	if event.EventType != "SourceBindingChanged" {
		return nil
	}
	if event.AggregateType != "SOURCE_BINDING" || p == nil || p.adapter == nil || p.adapter.store == nil || p.adapter.checkpoints == nil {
		return fmt.Errorf("source binding change event cannot be projected")
	}
	var payload checkpointProjectionPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("decode source binding change: %w", err)
	}
	if payload.BindingID == "" || payload.BindingID != event.AggregateID || payload.SourceID == "" {
		return fmt.Errorf("source binding change identity is invalid")
	}
	bindingVersion, err := strconv.ParseInt(payload.BindingVersion, 10, 64)
	if err != nil || bindingVersion < 1 {
		return fmt.Errorf("source binding change version is invalid")
	}
	checkpoint, err := p.adapter.checkpoints.Get(ctx, event.TenantID, payload.BindingID, bindingVersion)
	if err != nil {
		return err
	}
	if checkpoint.SourceID != payload.SourceID {
		return fmt.Errorf("source binding change source does not match checkpoint")
	}
	now := p.now().UTC()
	switch payload.Position.Kind {
	case sourceaccess.CheckpointEventID:
		// EVENT_ID is an idempotency position rather than an ordering primitive.
		// Never move an already-observed event checkpoint backwards or sideways.
		if checkpoint.Position.Kind != "" {
			return nil
		}
		transitionID, err := sourceaccess.CheckpointInboxEventID(checkpoint, checkpointConsumer, payload.Position)
		if err != nil {
			return err
		}
		if _, err := p.adapter.store.RecordInbox(ctx, checkpoint.TenantID, checkpointConsumer, transitionID, now); err != nil {
			return err
		}
		_, err = p.adapter.checkpoints.AdvanceAfterInbox(ctx, checkpoint, checkpointConsumer, payload.Position, now)
		if err == sourceaccess.ErrCheckpointConflict {
			return nil
		}
		return err
	case sourceaccess.CheckpointWatermark:
		return (&session{adapter: p.adapter}).advanceWatermarkFromCurrent(ctx, checkpoint, payload.Position, now)
	default:
		return fmt.Errorf("source binding change position is unsupported")
	}
}

var _ runtime.Publisher = (*CheckpointProjector)(nil)
