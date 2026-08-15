package sourceevent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

const (
	providerConsumer   = "source-webhook-provider-v1"
	checkpointConsumer = "source-webhook-checkpoint-v1"
)

type RuntimeStore interface {
	sourceaccess.InboxReceiptReader
	RecordInbox(context.Context, string, string, string, time.Time) (bool, error)
	RecordInboxWithOutbox(context.Context, []runtime.InboxReceipt, runtime.OutboxEvent, time.Time) (bool, error)
}

type Adapter struct {
	store       RuntimeStore
	checkpoints *sourceaccess.CheckpointService
	now         func() time.Time
	newID       func() (string, error)
}
type session struct {
	adapter    *Adapter
	connection sourceaccess.Connection
}

func NewAdapter(store RuntimeStore, checkpoints *sourceaccess.CheckpointService) *Adapter {
	return &Adapter{store: store, checkpoints: checkpoints, now: time.Now, newID: id.NewUUIDv7}
}
func (a *Adapter) Open(ctx context.Context, connection sourceaccess.Connection, _ sourceaccess.SecretResolver) (sourceaccess.Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if a == nil || a.store == nil || a.checkpoints == nil {
		return nil, sourceaccess.ErrConnection
	}
	if err := connection.Validate(); err != nil {
		return nil, err
	}
	if connection.AdapterKind != sourceaccess.AdapterWebhookEvent || connection.AdapterVersion != sourceaccess.WebhookEventAdapterVersion || strings.TrimSpace(connection.TenantID) == "" || connection.SecretRef != "" {
		return nil, sourceaccess.ErrDefinitionInvalid
	}
	definition := bytes.TrimSpace(connection.Definition)
	if len(definition) > 0 && string(definition) != "{}" {
		return nil, sourceaccess.ErrDefinitionInvalid
	}
	return &session{adapter: a, connection: connection}, nil
}
func (s *session) Connection() sourceaccess.Connection { return s.connection }
func (s *session) Capabilities() sourceaccess.CapabilitySet {
	return sourceaccess.NewCapabilitySet(sourceaccess.CapabilityInspect, sourceaccess.CapabilityChanges)
}
func (s *session) Close() error { return nil }

func (s *session) Inspect(ctx context.Context, view sourceaccess.View) (sourceaccess.SchemaResult, error) {
	if err := ctx.Err(); err != nil {
		return sourceaccess.SchemaResult{}, err
	}
	if err := view.Validate(s.connection); err != nil {
		return sourceaccess.SchemaResult{}, err
	}
	definition, err := sourceaccess.DecodeWebhookEventViewDefinition(view.Definition)
	if err != nil {
		return sourceaccess.SchemaResult{}, err
	}
	definitionFingerprint, err := sourceaccess.ViewFingerprint(view)
	if err != nil {
		return sourceaccess.SchemaResult{}, err
	}
	schemaFingerprint, err := nativeSchemaFingerprint(definition.Fields)
	if err != nil {
		return sourceaccess.SchemaResult{}, err
	}
	fields := append([]sourceaccess.NativeField(nil), definition.Fields...)
	return sourceaccess.SchemaResult{Fields: fields, Receipt: sourceaccess.OperationReceipt{SourceID: s.connection.SourceID, ConnectionID: s.connection.ID, ConnectionVersion: s.connection.Version, AdapterKind: s.connection.AdapterKind, AdapterVersion: s.connection.AdapterVersion, ViewID: view.ID, ViewVersion: view.Version, DefinitionFingerprint: definitionFingerprint, SchemaFingerprint: schemaFingerprint, Operation: sourceaccess.OperationInspect, ObservedAt: s.adapter.now().UTC(), Count: int64(len(fields)), Completeness: sourceaccess.CompletenessComplete, RetryIdentity: webhookRetryIdentity(view, sourceaccess.Binding{}, "inspect", "", nil)}}, nil
}

func (s *session) CaptureChange(ctx context.Context, view sourceaccess.View, binding sourceaccess.Binding, event sourceaccess.ChangeEvent) (sourceaccess.ChangeCaptureResult, error) {
	if err := event.ValidateInput(); err != nil {
		return sourceaccess.ChangeCaptureResult{}, err
	}
	if err := view.Validate(s.connection); err != nil {
		return sourceaccess.ChangeCaptureResult{}, err
	}
	if err := binding.Validate(view); err != nil {
		return sourceaccess.ChangeCaptureResult{}, err
	}
	if !binding.Allows(sourceaccess.OperationChanges) {
		return sourceaccess.ChangeCaptureResult{}, sourceaccess.ErrCapabilityUnavailable
	}
	limits, err := binding.NormalizedLimits()
	if err != nil {
		return sourceaccess.ChangeCaptureResult{}, err
	}
	if int64(len(event.Payload)) > limits.ResponseBytes {
		return sourceaccess.ChangeCaptureResult{}, sourceaccess.ErrLimitExceeded
	}
	definition, err := sourceaccess.DecodeWebhookEventViewDefinition(view.Definition)
	if err != nil {
		return sourceaccess.ChangeCaptureResult{}, err
	}
	position, err := normalizePosition(definition.PositionKind, event)
	if err != nil {
		return sourceaccess.ChangeCaptureResult{}, err
	}
	record, err := decodeRecord(event.Payload, definition.Fields, binding.SelectedFields)
	if err != nil {
		return sourceaccess.ChangeCaptureResult{}, err
	}
	schemaFingerprint, err := nativeSchemaFingerprint(definition.Fields)
	if err != nil {
		return sourceaccess.ChangeCaptureResult{}, err
	}
	if view.SchemaFingerprint != "" && view.SchemaFingerprint != schemaFingerprint {
		return sourceaccess.ChangeCaptureResult{}, sourceaccess.ErrSchemaDrift
	}
	bindingVersion, err := strconv.ParseInt(binding.Version, 10, 64)
	if err != nil || bindingVersion < 1 {
		return sourceaccess.ChangeCaptureResult{}, sourceaccess.ErrDefinitionInvalid
	}
	now := s.adapter.now().UTC()
	checkpoint, err := s.adapter.checkpoints.Ensure(ctx, s.connection.TenantID, s.connection.SourceID, binding.ID, bindingVersion, now)
	if err != nil {
		return sourceaccess.ChangeCaptureResult{}, err
	}
	providerID := providerInboxEventID(s.connection, binding, event.EventID)
	duplicate, err := s.adapter.store.InboxProcessed(ctx, s.connection.TenantID, providerConsumer, providerID)
	if err != nil {
		return sourceaccess.ChangeCaptureResult{}, err
	}
	if !duplicate && definition.PositionKind == sourceaccess.CheckpointWatermark {
		if err := requireForwardWatermark(checkpoint.Position, position); err != nil {
			return sourceaccess.ChangeCaptureResult{}, err
		}
	}
	definitionFingerprint, err := sourceaccess.BindingFingerprint(view, binding)
	if err != nil {
		return sourceaccess.ChangeCaptureResult{}, err
	}
	receipt := sourceaccess.OperationReceipt{SourceID: s.connection.SourceID, ConnectionID: s.connection.ID, ConnectionVersion: s.connection.Version, AdapterKind: s.connection.AdapterKind, AdapterVersion: s.connection.AdapterVersion, ViewID: view.ID, ViewVersion: view.Version, BindingID: binding.ID, BindingVersion: binding.Version, DefinitionFingerprint: definitionFingerprint, SchemaFingerprint: schemaFingerprint, Operation: sourceaccess.OperationChanges, ObservedAt: now, Count: 1, Bytes: int64(len(event.Payload)), Completeness: sourceaccess.CompletenessComplete, Position: &position, RetryIdentity: webhookRetryIdentity(view, binding, event.EventID, position.Value, event.Payload)}
	if duplicate {
		if err := s.reconcileDuplicate(ctx, checkpoint, definition.PositionKind, position, now); err != nil {
			return sourceaccess.ChangeCaptureResult{}, err
		}
		return sourceaccess.ChangeCaptureResult{Accepted: true, Duplicate: true, Receipt: receipt}, nil
	}
	transitionID, err := sourceaccess.CheckpointInboxEventID(checkpoint, checkpointConsumer, position)
	if err != nil {
		return sourceaccess.ChangeCaptureResult{}, err
	}
	outbound, err := json.Marshal(map[string]any{"source_id": s.connection.SourceID, "connection_id": s.connection.ID, "connection_version": s.connection.Version, "view_id": view.ID, "view_version": view.Version, "binding_id": binding.ID, "binding_version": binding.Version, "provider_event_id": event.EventID, "position": position, "schema_fingerprint": schemaFingerprint, "observed_at": now, "record": record})
	if err != nil || int64(len(outbound)) > sourceaccess.HardMaxResponseBytes {
		return sourceaccess.ChangeCaptureResult{}, sourceaccess.ErrLimitExceeded
	}
	outboxID, err := s.adapter.newID()
	if err != nil {
		return sourceaccess.ChangeCaptureResult{}, err
	}
	created, err := s.adapter.store.RecordInboxWithOutbox(ctx, []runtime.InboxReceipt{{TenantID: s.connection.TenantID, Consumer: providerConsumer, EventID: providerID}, {TenantID: s.connection.TenantID, Consumer: checkpointConsumer, EventID: transitionID}}, runtime.OutboxEvent{ID: outboxID, TenantID: s.connection.TenantID, AggregateType: "SOURCE_BINDING", AggregateID: binding.ID, EventType: "SourceBindingChanged", Payload: outbound, OccurredAt: now}, now)
	if err != nil {
		return sourceaccess.ChangeCaptureResult{}, err
	}
	if !created {
		checkpoint, getErr := s.adapter.checkpoints.Get(ctx, s.connection.TenantID, binding.ID, bindingVersion)
		if getErr != nil {
			return sourceaccess.ChangeCaptureResult{}, getErr
		}
		if err := s.reconcileDuplicate(ctx, checkpoint, definition.PositionKind, position, now); err != nil {
			return sourceaccess.ChangeCaptureResult{}, err
		}
		return sourceaccess.ChangeCaptureResult{Accepted: true, Duplicate: true, Receipt: receipt}, nil
	}
	if err := s.advanceAccepted(ctx, checkpoint, definition.PositionKind, position, now); err != nil {
		return sourceaccess.ChangeCaptureResult{}, err
	}
	return sourceaccess.ChangeCaptureResult{Accepted: true, Receipt: receipt}, nil
}

func (s *session) advanceAccepted(ctx context.Context, checkpoint sourceaccess.BindingCheckpoint, mode sourceaccess.CheckpointPositionKind, position sourceaccess.CheckpointPosition, now time.Time) error {
	_, err := s.adapter.checkpoints.AdvanceAfterInbox(ctx, checkpoint, checkpointConsumer, position, now)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sourceaccess.ErrCheckpointConflict) {
		return err
	}
	current, getErr := s.adapter.checkpoints.Get(ctx, checkpoint.TenantID, checkpoint.BindingID, checkpoint.BindingVersion)
	if getErr != nil {
		return getErr
	}
	if mode == sourceaccess.CheckpointEventID {
		return nil
	}
	return s.advanceWatermarkFromCurrent(ctx, current, position, now)
}
func (s *session) reconcileDuplicate(ctx context.Context, checkpoint sourceaccess.BindingCheckpoint, mode sourceaccess.CheckpointPositionKind, position sourceaccess.CheckpointPosition, now time.Time) error {
	if checkpoint.Position.Kind == position.Kind && checkpoint.Position.Value == position.Value {
		return nil
	}
	transitionID, err := sourceaccess.CheckpointInboxEventID(checkpoint, checkpointConsumer, position)
	if err != nil {
		return err
	}
	processed, err := s.adapter.store.InboxProcessed(ctx, checkpoint.TenantID, checkpointConsumer, transitionID)
	if err != nil {
		return err
	}
	if processed {
		_, err = s.adapter.checkpoints.AdvanceAfterInbox(ctx, checkpoint, checkpointConsumer, position, now)
		if err == nil || errors.Is(err, sourceaccess.ErrCheckpointConflict) {
			return nil
		}
		return err
	}
	if mode == sourceaccess.CheckpointWatermark {
		return s.advanceWatermarkFromCurrent(ctx, checkpoint, position, now)
	}
	return nil
}
func (s *session) advanceWatermarkFromCurrent(ctx context.Context, checkpoint sourceaccess.BindingCheckpoint, position sourceaccess.CheckpointPosition, now time.Time) error {
	for attempts := 0; attempts < 4; attempts++ {
		if checkpoint.Position.Kind != "" {
			current, err := watermarkValue(checkpoint.Position)
			if err != nil {
				return err
			}
			target, err := watermarkValue(position)
			if err != nil {
				return err
			}
			if current >= target {
				return nil
			}
		}
		transitionID, err := sourceaccess.CheckpointInboxEventID(checkpoint, checkpointConsumer, position)
		if err != nil {
			return err
		}
		if _, err := s.adapter.store.RecordInbox(ctx, checkpoint.TenantID, checkpointConsumer, transitionID, now); err != nil {
			return err
		}
		_, err = s.adapter.checkpoints.AdvanceAfterInbox(ctx, checkpoint, checkpointConsumer, position, now)
		if err == nil {
			return nil
		}
		if !errors.Is(err, sourceaccess.ErrCheckpointConflict) {
			return err
		}
		checkpoint, err = s.adapter.checkpoints.Get(ctx, checkpoint.TenantID, checkpoint.BindingID, checkpoint.BindingVersion)
		if err != nil {
			return err
		}
	}
	return sourceaccess.ErrCheckpointConflict
}
func normalizePosition(mode sourceaccess.CheckpointPositionKind, event sourceaccess.ChangeEvent) (sourceaccess.CheckpointPosition, error) {
	if mode == sourceaccess.CheckpointEventID {
		if event.Position == nil {
			return sourceaccess.CheckpointPosition{Kind: sourceaccess.CheckpointEventID, Value: event.EventID}, nil
		}
		if event.Position.Kind != sourceaccess.CheckpointEventID || event.Position.Value != event.EventID {
			return sourceaccess.CheckpointPosition{}, sourceaccess.ErrDefinitionInvalid
		}
		return *event.Position, nil
	}
	if event.Position == nil || event.Position.Kind != sourceaccess.CheckpointWatermark {
		return sourceaccess.CheckpointPosition{}, sourceaccess.ErrDefinitionInvalid
	}
	if _, err := watermarkValue(*event.Position); err != nil {
		return sourceaccess.CheckpointPosition{}, err
	}
	return *event.Position, nil
}
func requireForwardWatermark(current, target sourceaccess.CheckpointPosition) error {
	if current.Kind == "" {
		return nil
	}
	if current.Kind != sourceaccess.CheckpointWatermark {
		return sourceaccess.ErrCheckpointConflict
	}
	left, err := watermarkValue(current)
	if err != nil {
		return err
	}
	right, err := watermarkValue(target)
	if err != nil {
		return err
	}
	if right <= left {
		return sourceaccess.ErrCheckpointConflict
	}
	return nil
}
func watermarkValue(position sourceaccess.CheckpointPosition) (uint64, error) {
	if position.Kind != sourceaccess.CheckpointWatermark || position.Value == "" || strings.HasPrefix(position.Value, "+") {
		return 0, sourceaccess.ErrDefinitionInvalid
	}
	value, err := strconv.ParseUint(position.Value, 10, 64)
	if err != nil || strconv.FormatUint(value, 10) != position.Value {
		return 0, sourceaccess.ErrDefinitionInvalid
	}
	return value, nil
}

func decodeRecord(payload json.RawMessage, fields []sourceaccess.NativeField, selected []string) (sourceaccess.Record, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var object map[string]any
	if err := decoder.Decode(&object); err != nil {
		return nil, sourceaccess.ErrDefinitionInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, sourceaccess.ErrDefinitionInvalid
	}
	schema := make(map[string]sourceaccess.NativeField, len(fields))
	for _, field := range fields {
		schema[field.Name] = field
	}
	for name := range object {
		if _, exists := schema[name]; !exists {
			return nil, sourceaccess.ErrSchemaDrift
		}
	}
	for name, field := range schema {
		value, exists := object[name]
		if !exists {
			if !field.Nullable {
				return nil, sourceaccess.ErrSchemaDrift
			}
			continue
		}
		if value == nil && !field.Nullable {
			return nil, sourceaccess.ErrSchemaDrift
		}
		if _, err := scalarFromJSON(value, field); err != nil {
			return nil, err
		}
	}
	record := make(sourceaccess.Record, len(selected))
	for _, name := range selected {
		field, exists := schema[name]
		if !exists {
			return nil, sourceaccess.ErrSchemaDrift
		}
		value, exists := object[name]
		if !exists || value == nil {
			record[name] = sourceaccess.Scalar{Kind: sourceaccess.ScalarNull}
			continue
		}
		scalar, err := scalarFromJSON(value, field)
		if err != nil {
			return nil, err
		}
		record[name] = scalar
	}
	return record, nil
}
func scalarFromJSON(value any, field sourceaccess.NativeField) (sourceaccess.Scalar, error) {
	if value == nil {
		return sourceaccess.Scalar{Kind: sourceaccess.ScalarNull}, nil
	}
	switch field.NativeType {
	case "json:string":
		typed, ok := value.(string)
		if !ok || len(typed) > 64<<10 {
			return sourceaccess.Scalar{}, sourceaccess.ErrSchemaDrift
		}
		return sourceaccess.Scalar{Kind: sourceaccess.ScalarString, Text: typed}, nil
	case "json:number":
		typed, ok := value.(json.Number)
		if !ok {
			return sourceaccess.Scalar{}, sourceaccess.ErrSchemaDrift
		}
		return sourceaccess.Scalar{Kind: sourceaccess.ScalarNumber, Text: typed.String()}, nil
	case "json:boolean":
		typed, ok := value.(bool)
		if !ok {
			return sourceaccess.Scalar{}, sourceaccess.ErrSchemaDrift
		}
		return sourceaccess.Scalar{Kind: sourceaccess.ScalarBool, Text: strconv.FormatBool(typed)}, nil
	case "json:time":
		typed, ok := value.(string)
		if !ok {
			return sourceaccess.Scalar{}, sourceaccess.ErrSchemaDrift
		}
		parsed, err := time.Parse(time.RFC3339Nano, typed)
		if err != nil {
			return sourceaccess.Scalar{}, sourceaccess.ErrSchemaDrift
		}
		return sourceaccess.Scalar{Kind: sourceaccess.ScalarTime, Text: parsed.UTC().Format(time.RFC3339Nano)}, nil
	default:
		return sourceaccess.Scalar{}, sourceaccess.ErrUnsupportedValue
	}
}
func nativeSchemaFingerprint(fields []sourceaccess.NativeField) (string, error) {
	canonical := append([]sourceaccess.NativeField(nil), fields...)
	sort.Slice(canonical, func(i, j int) bool { return canonical[i].Name < canonical[j].Name })
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(encoded)
	return hex.EncodeToString(hash[:]), nil
}
func providerInboxEventID(connection sourceaccess.Connection, binding sourceaccess.Binding, eventID string) string {
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "%s\x1f%s\x1f%s\x1f%s\x1f%s", connection.TenantID, connection.SourceID, binding.ID, binding.Version, eventID)
	return "source-webhook:" + hex.EncodeToString(hash.Sum(nil))
}
func webhookRetryIdentity(view sourceaccess.View, binding sourceaccess.Binding, eventID, position string, payload []byte) string {
	payloadHash := sha256.Sum256(payload)
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s", view.ConnectionID, view.ID, view.Version, binding.ID, binding.Version, eventID, position)
	_, _ = hash.Write(payloadHash[:])
	return "webhook-event:" + hex.EncodeToString(hash.Sum(nil))
}

var _ sourceaccess.Adapter = (*Adapter)(nil)
var _ sourceaccess.Session = (*session)(nil)
var _ sourceaccess.SchemaReader = (*session)(nil)
var _ sourceaccess.ChangeReceiver = (*session)(nil)
