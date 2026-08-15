from pathlib import Path
import json


def replace_once(path, old, new):
    p = Path(path)
    text = p.read_text()
    if old not in text:
        raise SystemExit(f"anchor missing in {path}: {old[:100]!r}")
    p.write_text(text.replace(old, new, 1))


def write(path, content):
    p = Path(path)
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(content)

# Shared sourceaccess contracts.
replace_once("internal/sourceaccess/contracts_types.go",
'''const (\n\tAdapterPostgres        AdapterKind = "POSTGRES"\n\tAdapterRESTJSON        AdapterKind = "REST_JSON"\n\tAdapterTabularArtifact AdapterKind = "TABULAR_ARTIFACT"\n)\n\nconst TabularArtifactAdapterVersion = "tabular-artifact-v1"\n''',
'''const (\n\tAdapterPostgres        AdapterKind = "POSTGRES"\n\tAdapterRESTJSON        AdapterKind = "REST_JSON"\n\tAdapterTabularArtifact AdapterKind = "TABULAR_ARTIFACT"\n\tAdapterWebhookEvent    AdapterKind = "WEBHOOK_EVENT"\n)\n\nconst (\n\tTabularArtifactAdapterVersion = "tabular-artifact-v1"\n\tWebhookEventAdapterVersion    = "webhook-event-v1"\n)\n''')
replace_once("internal/sourceaccess/contracts_types.go",
'''type LookupResult struct {\n\tRecords []Record         `json:"records"`\n\tReceipt OperationReceipt `json:"receipt"`\n}\n\n''',
'''type LookupResult struct {\n\tRecords []Record         `json:"records"`\n\tReceipt OperationReceipt `json:"receipt"`\n}\n\ntype ChangeEvent struct {\n\tEventID  string              `json:"event_id"`\n\tPosition *CheckpointPosition `json:"position,omitempty"`\n\tPayload  json.RawMessage     `json:"payload"`\n}\n\ntype ChangeCaptureResult struct {\n\tAccepted  bool             `json:"accepted"`\n\tDuplicate bool             `json:"duplicate"`\n\tReceipt   OperationReceipt `json:"receipt"`\n}\n\n''')
replace_once("internal/sourceaccess/contracts_types.go",
'''type LookupReader interface {\n\tLookup(context.Context, View, Binding, LookupRequest) (LookupResult, error)\n}\n''',
'''type LookupReader interface {\n\tLookup(context.Context, View, Binding, LookupRequest) (LookupResult, error)\n}\n\ntype ChangeReceiver interface {\n\tCaptureChange(context.Context, View, Binding, ChangeEvent) (ChangeCaptureResult, error)\n}\n''')
replace_once("internal/sourceaccess/contracts_validation.go",
'''func (s Scalar) ValidateInput() error {\n''',
'''func (e ChangeEvent) ValidateInput() error {\n\tif err := validateOpaqueID(e.EventID, "provider event id"); err != nil {\n\t\treturn err\n\t}\n\tif e.Position != nil {\n\t\tif err := validateCheckpointPosition(*e.Position); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\tif len(e.Payload) == 0 || len(e.Payload) > HardMaxResponseBytes || !json.Valid(e.Payload) {\n\t\treturn fmt.Errorf("%w: bounded JSON event payload is required", ErrDefinitionInvalid)\n\t}\n\treturn nil\n}\n\nfunc (s Scalar) ValidateInput() error {\n''')

write("internal/sourceaccess/webhook_event_definition.go", r'''package sourceaccess

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "sort"
    "strings"
)

type WebhookEventViewDefinition struct {
    PositionKind CheckpointPositionKind `json:"position_kind"`
    Fields       []NativeField          `json:"fields"`
}

func NormalizeWebhookEventViewDefinition(raw json.RawMessage) (json.RawMessage, error) {
    definition, err := DecodeWebhookEventViewDefinition(raw)
    if err != nil { return nil, err }
    return json.Marshal(definition)
}

func DecodeWebhookEventViewDefinition(raw json.RawMessage) (WebhookEventViewDefinition, error) {
    if len(raw) == 0 || len(raw) > HardMaxDefinitionBytes { return WebhookEventViewDefinition{}, ErrLimitExceeded }
    decoder := json.NewDecoder(bytes.NewReader(raw)); decoder.DisallowUnknownFields()
    var definition WebhookEventViewDefinition
    if err := decoder.Decode(&definition); err != nil { return WebhookEventViewDefinition{}, fmt.Errorf("%w: webhook event view definition is invalid", ErrDefinitionInvalid) }
    var trailing any
    if err := decoder.Decode(&trailing); err != io.EOF { return WebhookEventViewDefinition{}, fmt.Errorf("%w: webhook event view definition has trailing data", ErrDefinitionInvalid) }
    switch definition.PositionKind { case CheckpointEventID, CheckpointWatermark: default: return WebhookEventViewDefinition{}, fmt.Errorf("%w: webhook position_kind must be EVENT_ID or WATERMARK", ErrDefinitionInvalid) }
    if len(definition.Fields) == 0 || len(definition.Fields) > HardMaxSchemaFields { return WebhookEventViewDefinition{}, fmt.Errorf("%w: webhook fields are outside the supported range", ErrDefinitionInvalid) }
    seen := make(map[string]struct{}, len(definition.Fields))
    for index := range definition.Fields {
        field := &definition.Fields[index]; field.NativeType = strings.TrimSpace(field.NativeType)
        if !ValidFieldName(field.Name) { return WebhookEventViewDefinition{}, fmt.Errorf("%w: webhook field name is invalid", ErrDefinitionInvalid) }
        if _, exists := seen[field.Name]; exists { return WebhookEventViewDefinition{}, fmt.Errorf("%w: webhook fields contain duplicates", ErrDefinitionInvalid) }
        seen[field.Name] = struct{}{}
        switch field.NativeType { case "json:string", "json:number", "json:boolean", "json:time": default: return WebhookEventViewDefinition{}, fmt.Errorf("%w: webhook fields must use bounded scalar JSON types", ErrDefinitionInvalid) }
    }
    sort.Slice(definition.Fields, func(i, j int) bool { return definition.Fields[i].Name < definition.Fields[j].Name })
    return definition, nil
}
''')

write("internal/sourceaccess/catalog_changes.go", r'''package sourceaccess

import "context"

func (s *CatalogService) CaptureBindingChange(ctx context.Context, tenantID, bindingID string, version int64, event ChangeEvent) (ChangeCaptureResult, error) {
    if err := event.ValidateInput(); err != nil { return ChangeCaptureResult{}, err }
    bindingRevision, err := s.binding(ctx, tenantID, bindingID, version)
    if err != nil { return ChangeCaptureResult{}, err }
    if !statefulBindingRevision(bindingRevision) || !bindingRevisionAllows(bindingRevision, OperationChanges) { return ChangeCaptureResult{}, ErrCatalogInvalid }
    viewRevision, err := s.repoOrError().ViewRevision(ctx, tenantID, bindingRevision.ViewID, bindingRevision.ViewVersion)
    if err != nil { return ChangeCaptureResult{}, err }
    connectionRevision, err := s.repoOrError().ConnectionRevision(ctx, tenantID, viewRevision.ConnectionID, viewRevision.ConnectionVersion)
    if err != nil { return ChangeCaptureResult{}, err }
    connection, view, adapter, err := s.executionContracts(connectionRevision, viewRevision)
    if err != nil { return ChangeCaptureResult{}, err }
    binding, err := bindingRevision.Contract(viewRevision)
    if err != nil { return ChangeCaptureResult{}, err }
    limits, err := binding.NormalizedLimits()
    if err != nil { return ChangeCaptureResult{}, err }
    operationCtx, cancel := context.WithTimeout(ctx, limits.Timeout); defer cancel()
    session, err := adapter.Open(operationCtx, connection, s.secrets)
    if err != nil { return ChangeCaptureResult{}, err }
    defer session.Close()
    if !session.Capabilities().Has(CapabilityChanges) { return ChangeCaptureResult{}, ErrCapabilityUnavailable }
    receiver, ok := session.(ChangeReceiver)
    if !ok { return ChangeCaptureResult{}, ErrCapabilityUnavailable }
    result, err := receiver.CaptureChange(operationCtx, view, binding, event)
    if err != nil { return ChangeCaptureResult{}, err }
    if err := validateCatalogReceipt(result.Receipt, connection, view, binding, OperationChanges, 1); err != nil { return ChangeCaptureResult{}, err }
    if err := validateExpectedSchema(view, result.Receipt.SchemaFingerprint); err != nil { return ChangeCaptureResult{}, err }
    if result.Receipt.Position == nil || result.Receipt.Bytes > limits.ResponseBytes { return ChangeCaptureResult{}, ErrLimitExceeded }
    if !result.Accepted { return ChangeCaptureResult{}, ErrExecution }
    return result, nil
}

func bindingRevisionAllows(binding BindingRevision, operation Operation) bool {
    for _, candidate := range binding.Operations { if candidate == operation { return true } }
    return false
}
''')

replace_once("internal/sourceaccess/catalog_service.go",
'''\tcase OperationLookup:\n\t\tif receipt.BindingID != binding.ID || receipt.BindingVersion != binding.Version || (receipt.Completeness != CompletenessComplete && receipt.Completeness != CompletenessPartial) {\n\t\t\treturn ErrExecution\n\t\t}\n\tdefault:\n''',
'''\tcase OperationLookup:\n\t\tif receipt.BindingID != binding.ID || receipt.BindingVersion != binding.Version || (receipt.Completeness != CompletenessComplete && receipt.Completeness != CompletenessPartial) {\n\t\t\treturn ErrExecution\n\t\t}\n\tcase OperationChanges:\n\t\tif receipt.BindingID != binding.ID || receipt.BindingVersion != binding.Version || receipt.Completeness != CompletenessComplete || receipt.Position == nil {\n\t\t\treturn ErrExecution\n\t\t}\n\t\tif err := validateCheckpointPosition(*receipt.Position); err != nil { return ErrExecution }\n\tdefault:\n''')
replace_once("internal/sourceaccess/catalog_service.go",
'''\tcase AdapterTabularArtifact:\n\t\tif input.SecretRef != "" {\n\t\t\treturn fmt.Errorf("%w: tabular artifact connections do not carry credentials", ErrCatalogInvalid)\n\t\t}\n\t\tdefinition, err := normalizeJSONObject(defaultJSONObject(input.Definition), HardMaxDefinitionBytes, "connection definition")\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tif string(definition) != "{}" {\n\t\t\treturn fmt.Errorf("%w: tabular artifact connection definitions are not supported", ErrCatalogInvalid)\n\t\t}\n\t}\n''',
'''\tcase AdapterTabularArtifact:\n\t\tif input.SecretRef != "" {\n\t\t\treturn fmt.Errorf("%w: tabular artifact connections do not carry credentials", ErrCatalogInvalid)\n\t\t}\n\t\tdefinition, err := normalizeJSONObject(defaultJSONObject(input.Definition), HardMaxDefinitionBytes, "connection definition")\n\t\tif err != nil { return err }\n\t\tif string(definition) != "{}" { return fmt.Errorf("%w: tabular artifact connection definitions are not supported", ErrCatalogInvalid) }\n\tcase AdapterWebhookEvent:\n\t\tif input.SecretRef != "" { return fmt.Errorf("%w: webhook event connections use verified service identity rather than connector credentials", ErrCatalogInvalid) }\n\t\tdefinition, err := normalizeJSONObject(defaultJSONObject(input.Definition), HardMaxDefinitionBytes, "connection definition")\n\t\tif err != nil { return err }\n\t\tif string(definition) != "{}" { return fmt.Errorf("%w: webhook event connection definitions are not supported", ErrCatalogInvalid) }\n\t}\n''')
replace_once("internal/sourceaccess/catalog_service.go",
'''\tcase AdapterTabularArtifact:\n\t\tif version != TabularArtifactAdapterVersion {\n\t\t\treturn nil, fmt.Errorf("%w: unsupported tabular artifact adapter version", ErrCatalogInvalid)\n\t\t}\n\t}\n''',
'''\tcase AdapterTabularArtifact:\n\t\tif version != TabularArtifactAdapterVersion { return nil, fmt.Errorf("%w: unsupported tabular artifact adapter version", ErrCatalogInvalid) }\n\tcase AdapterWebhookEvent:\n\t\tif version != WebhookEventAdapterVersion { return nil, fmt.Errorf("%w: unsupported webhook event adapter version", ErrCatalogInvalid) }\n\t}\n''')
replace_once("internal/sourceaccess/catalog_service.go",
'''\tif connection.AdapterKind == AdapterTabularArtifact {\n\t\tdefinition, err := NormalizeTabularArtifactViewDefinition(raw)\n\t\tif err != nil {\n\t\t\treturn nil, errors.Join(ErrCatalogInvalid, err)\n\t\t}\n\t\treturn definition, nil\n\t}\n\treturn normalizeJSONObject(raw, HardMaxDefinitionBytes, "view definition")\n''',
'''\tif connection.AdapterKind == AdapterTabularArtifact {\n\t\tdefinition, err := NormalizeTabularArtifactViewDefinition(raw)\n\t\tif err != nil { return nil, errors.Join(ErrCatalogInvalid, err) }\n\t\treturn definition, nil\n\t}\n\tif connection.AdapterKind == AdapterWebhookEvent {\n\t\tdefinition, err := NormalizeWebhookEventViewDefinition(raw)\n\t\tif err != nil { return nil, errors.Join(ErrCatalogInvalid, err) }\n\t\treturn definition, nil\n\t}\n\treturn normalizeJSONObject(raw, HardMaxDefinitionBytes, "view definition")\n''')

# Runtime atomic provider dedupe + outbox write.
replace_once("internal/runtime/model.go", '''type OutboxEvent struct {\n''', '''type InboxReceipt struct {\n\tTenantID string `json:"tenant_id"`\n\tConsumer string `json:"consumer"`\n\tEventID  string `json:"event_id"`\n}\n\ntype OutboxEvent struct {\n''')
replace_once("internal/runtime/repository.go",
'''\tInboxProcessed(context.Context, string, string, string) (bool, error)\n\tRecordInbox(context.Context, string, string, string, time.Time) (bool, error)\n''',
'''\tInboxProcessed(context.Context, string, string, string) (bool, error)\n\tRecordInbox(context.Context, string, string, string, time.Time) (bool, error)\n\tRecordInboxWithOutbox(context.Context, []InboxReceipt, OutboxEvent, time.Time) (bool, error)\n''')
replace_once("internal/runtime/memory.go",
'''func (r *MemoryRepository) RecordInbox(_ context.Context, tenant, consumer, eventID string, at time.Time) (bool, error) {\n\tr.mu.Lock()\n\tdefer r.mu.Unlock()\n\tk := tenant + ":" + consumer + ":" + eventID\n\tif _, ok := r.inbox[k]; ok {\n\t\treturn false, nil\n\t}\n\tr.inbox[k] = struct{}{}\n\treturn true, nil\n}\n''',
'''func (r *MemoryRepository) RecordInbox(_ context.Context, tenant, consumer, eventID string, at time.Time) (bool, error) {\n\tr.mu.Lock()\n\tdefer r.mu.Unlock()\n\tk := tenant + ":" + consumer + ":" + eventID\n\tif _, ok := r.inbox[k]; ok { return false, nil }\n\tr.inbox[k] = struct{}{}\n\treturn true, nil\n}\nfunc (r *MemoryRepository) RecordInboxWithOutbox(_ context.Context, receipts []InboxReceipt, event OutboxEvent, _ time.Time) (bool, error) {\n\tif len(receipts) == 0 { return false, errors.New("at least one inbox receipt is required") }\n\tr.mu.Lock(); defer r.mu.Unlock()\n\tfirst := receipts[0].TenantID + ":" + receipts[0].Consumer + ":" + receipts[0].EventID\n\tif _, exists := r.inbox[first]; exists { return false, nil }\n\tkeys := make([]string, len(receipts))\n\tfor index, receipt := range receipts {\n\t\tif receipt.TenantID != event.TenantID || receipt.TenantID == "" || receipt.Consumer == "" || receipt.EventID == "" { return false, errors.New("inbox receipt does not match outbox tenant") }\n\t\tkeys[index] = receipt.TenantID + ":" + receipt.Consumer + ":" + receipt.EventID\n\t\tif index > 0 { if _, exists := r.inbox[keys[index]]; exists { return false, errors.New("secondary inbox receipt already exists") } }\n\t}\n\tif _, exists := r.outbox[event.ID]; exists { return false, errors.New("outbox event already exists") }\n\tfor _, key := range keys { r.inbox[key] = struct{}{} }\n\tr.outbox[event.ID] = event\n\treturn true, nil\n}\n''')
replace_once("internal/runtime/postgres.go",
'''func (r *PostgresRepository) RecordInbox(ctx context.Context, tenant, consumer, eventID string, at time.Time) (bool, error) {\n\ttag, err := r.pool.Exec(ctx, `INSERT INTO inbox_receipts(tenant_id,consumer,event_id,processed_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3,$4) ON CONFLICT DO NOTHING`, tenant, consumer, eventID, at)\n\tif err != nil {\n\t\treturn false, err\n\t}\n\treturn tag.RowsAffected() == 1, nil\n}\n''',
'''func (r *PostgresRepository) RecordInbox(ctx context.Context, tenant, consumer, eventID string, at time.Time) (bool, error) {\n\ttag, err := r.pool.Exec(ctx, `INSERT INTO inbox_receipts(tenant_id,consumer,event_id,processed_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3,$4) ON CONFLICT DO NOTHING`, tenant, consumer, eventID, at)\n\tif err != nil { return false, err }\n\treturn tag.RowsAffected() == 1, nil\n}\nfunc (r *PostgresRepository) RecordInboxWithOutbox(ctx context.Context, receipts []InboxReceipt, event OutboxEvent, at time.Time) (bool, error) {\n\tif len(receipts) == 0 { return false, errors.New("at least one inbox receipt is required") }\n\ttx, err := r.pool.Begin(ctx); if err != nil { return false, err }; defer tx.Rollback(ctx)\n\tfirst := receipts[0]\n\tif first.TenantID != event.TenantID { return false, errors.New("inbox receipt does not match outbox tenant") }\n\ttag, err := tx.Exec(ctx, `INSERT INTO inbox_receipts(tenant_id,consumer,event_id,processed_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3,$4) ON CONFLICT DO NOTHING`, first.TenantID, first.Consumer, first.EventID, at)\n\tif err != nil { return false, err }\n\tif tag.RowsAffected() == 0 { return false, nil }\n\tfor _, receipt := range receipts[1:] {\n\t\tif receipt.TenantID != event.TenantID { return false, errors.New("inbox receipt does not match outbox tenant") }\n\t\tif _, err := tx.Exec(ctx, `INSERT INTO inbox_receipts(tenant_id,consumer,event_id,processed_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3,$4)`, receipt.TenantID, receipt.Consumer, receipt.EventID, at); err != nil { return false, err }\n\t}\n\tif _, err := tx.Exec(ctx, `INSERT INTO outbox_events(id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4::uuid,$5,$6,$7,$7,$7)`, event.ID, event.TenantID, event.AggregateType, event.AggregateID, event.EventType, event.Payload, event.OccurredAt); err != nil { return false, err }\n\tif err := tx.Commit(ctx); err != nil { return false, err }\n\treturn true, nil\n}\n''')

write("internal/sourceevent/adapter.go", r'''package sourceevent

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
    providerConsumer = "source-webhook-provider-v1"
    checkpointConsumer = "source-webhook-checkpoint-v1"
)

type RuntimeStore interface {
    sourceaccess.InboxReceiptReader
    RecordInbox(context.Context, string, string, string, time.Time) (bool, error)
    RecordInboxWithOutbox(context.Context, []runtime.InboxReceipt, runtime.OutboxEvent, time.Time) (bool, error)
}

type Adapter struct { store RuntimeStore; checkpoints *sourceaccess.CheckpointService; now func() time.Time; newID func() (string,error) }
type session struct { adapter *Adapter; connection sourceaccess.Connection }

func NewAdapter(store RuntimeStore, checkpoints *sourceaccess.CheckpointService) *Adapter { return &Adapter{store:store, checkpoints:checkpoints, now:time.Now, newID:id.NewUUIDv7} }
func (a *Adapter) Open(ctx context.Context, connection sourceaccess.Connection, _ sourceaccess.SecretResolver) (sourceaccess.Session,error) {
    if err := ctx.Err(); err != nil { return nil, err }
    if a == nil || a.store == nil || a.checkpoints == nil { return nil, sourceaccess.ErrConnection }
    if err := connection.Validate(); err != nil { return nil, err }
    if connection.AdapterKind != sourceaccess.AdapterWebhookEvent || connection.AdapterVersion != sourceaccess.WebhookEventAdapterVersion || strings.TrimSpace(connection.TenantID)=="" || connection.SecretRef!="" { return nil, sourceaccess.ErrDefinitionInvalid }
    definition := bytes.TrimSpace(connection.Definition); if len(definition)>0 && string(definition)!="{}" { return nil, sourceaccess.ErrDefinitionInvalid }
    return &session{adapter:a, connection:connection}, nil
}
func (s *session) Connection() sourceaccess.Connection { return s.connection }
func (s *session) Capabilities() sourceaccess.CapabilitySet { return sourceaccess.NewCapabilitySet(sourceaccess.CapabilityInspect, sourceaccess.CapabilityChanges) }
func (s *session) Close() error { return nil }

func (s *session) Inspect(ctx context.Context, view sourceaccess.View) (sourceaccess.SchemaResult,error) {
    if err:=ctx.Err(); err!=nil { return sourceaccess.SchemaResult{},err }
    if err:=view.Validate(s.connection); err!=nil { return sourceaccess.SchemaResult{},err }
    definition,err:=sourceaccess.DecodeWebhookEventViewDefinition(view.Definition); if err!=nil { return sourceaccess.SchemaResult{},err }
    definitionFingerprint,err:=sourceaccess.ViewFingerprint(view); if err!=nil { return sourceaccess.SchemaResult{},err }
    schemaFingerprint,err:=nativeSchemaFingerprint(definition.Fields); if err!=nil { return sourceaccess.SchemaResult{},err }
    fields:=append([]sourceaccess.NativeField(nil),definition.Fields...)
    return sourceaccess.SchemaResult{Fields:fields,Receipt:sourceaccess.OperationReceipt{SourceID:s.connection.SourceID,ConnectionID:s.connection.ID,ConnectionVersion:s.connection.Version,AdapterKind:s.connection.AdapterKind,AdapterVersion:s.connection.AdapterVersion,ViewID:view.ID,ViewVersion:view.Version,DefinitionFingerprint:definitionFingerprint,SchemaFingerprint:schemaFingerprint,Operation:sourceaccess.OperationInspect,ObservedAt:s.adapter.now().UTC(),Count:int64(len(fields)),Completeness:sourceaccess.CompletenessComplete,RetryIdentity:webhookRetryIdentity(view,sourceaccess.Binding{},"inspect","",nil)}},nil
}

func (s *session) CaptureChange(ctx context.Context, view sourceaccess.View, binding sourceaccess.Binding, event sourceaccess.ChangeEvent) (sourceaccess.ChangeCaptureResult,error) {
    if err:=event.ValidateInput(); err!=nil { return sourceaccess.ChangeCaptureResult{},err }
    if err:=view.Validate(s.connection); err!=nil { return sourceaccess.ChangeCaptureResult{},err }
    if err:=binding.Validate(view); err!=nil { return sourceaccess.ChangeCaptureResult{},err }
    if !binding.Allows(sourceaccess.OperationChanges) { return sourceaccess.ChangeCaptureResult{},sourceaccess.ErrCapabilityUnavailable }
    limits,err:=binding.NormalizedLimits(); if err!=nil { return sourceaccess.ChangeCaptureResult{},err }
    if int64(len(event.Payload))>limits.ResponseBytes { return sourceaccess.ChangeCaptureResult{},sourceaccess.ErrLimitExceeded }
    definition,err:=sourceaccess.DecodeWebhookEventViewDefinition(view.Definition); if err!=nil { return sourceaccess.ChangeCaptureResult{},err }
    position,err:=normalizePosition(definition.PositionKind,event); if err!=nil { return sourceaccess.ChangeCaptureResult{},err }
    record,err:=decodeRecord(event.Payload,definition.Fields,binding.SelectedFields); if err!=nil { return sourceaccess.ChangeCaptureResult{},err }
    schemaFingerprint,err:=nativeSchemaFingerprint(definition.Fields); if err!=nil { return sourceaccess.ChangeCaptureResult{},err }
    if view.SchemaFingerprint!="" && view.SchemaFingerprint!=schemaFingerprint { return sourceaccess.ChangeCaptureResult{},sourceaccess.ErrSchemaDrift }
    bindingVersion,err:=strconv.ParseInt(binding.Version,10,64); if err!=nil || bindingVersion<1 { return sourceaccess.ChangeCaptureResult{},sourceaccess.ErrDefinitionInvalid }
    now:=s.adapter.now().UTC(); checkpoint,err:=s.adapter.checkpoints.Ensure(ctx,s.connection.TenantID,s.connection.SourceID,binding.ID,bindingVersion,now); if err!=nil { return sourceaccess.ChangeCaptureResult{},err }
    providerID:=providerInboxEventID(s.connection,binding,event.EventID); duplicate,err:=s.adapter.store.InboxProcessed(ctx,s.connection.TenantID,providerConsumer,providerID); if err!=nil { return sourceaccess.ChangeCaptureResult{},err }
    if !duplicate && definition.PositionKind==sourceaccess.CheckpointWatermark { if err:=requireForwardWatermark(checkpoint.Position,position); err!=nil { return sourceaccess.ChangeCaptureResult{},err } }
    definitionFingerprint,err:=sourceaccess.BindingFingerprint(view,binding); if err!=nil { return sourceaccess.ChangeCaptureResult{},err }
    receipt:=sourceaccess.OperationReceipt{SourceID:s.connection.SourceID,ConnectionID:s.connection.ID,ConnectionVersion:s.connection.Version,AdapterKind:s.connection.AdapterKind,AdapterVersion:s.connection.AdapterVersion,ViewID:view.ID,ViewVersion:view.Version,BindingID:binding.ID,BindingVersion:binding.Version,DefinitionFingerprint:definitionFingerprint,SchemaFingerprint:schemaFingerprint,Operation:sourceaccess.OperationChanges,ObservedAt:now,Count:1,Bytes:int64(len(event.Payload)),Completeness:sourceaccess.CompletenessComplete,Position:&position,RetryIdentity:webhookRetryIdentity(view,binding,event.EventID,position.Value,event.Payload)}
    if duplicate { if err:=s.reconcileDuplicate(ctx,checkpoint,definition.PositionKind,position,now); err!=nil { return sourceaccess.ChangeCaptureResult{},err }; return sourceaccess.ChangeCaptureResult{Accepted:true,Duplicate:true,Receipt:receipt},nil }
    transitionID,err:=sourceaccess.CheckpointInboxEventID(checkpoint,checkpointConsumer,position); if err!=nil { return sourceaccess.ChangeCaptureResult{},err }
    outbound,err:=json.Marshal(map[string]any{"source_id":s.connection.SourceID,"connection_id":s.connection.ID,"connection_version":s.connection.Version,"view_id":view.ID,"view_version":view.Version,"binding_id":binding.ID,"binding_version":binding.Version,"provider_event_id":event.EventID,"position":position,"schema_fingerprint":schemaFingerprint,"observed_at":now,"record":record}); if err!=nil || int64(len(outbound))>sourceaccess.HardMaxResponseBytes { return sourceaccess.ChangeCaptureResult{},sourceaccess.ErrLimitExceeded }
    outboxID,err:=s.adapter.newID(); if err!=nil { return sourceaccess.ChangeCaptureResult{},err }
    created,err:=s.adapter.store.RecordInboxWithOutbox(ctx,[]runtime.InboxReceipt{{TenantID:s.connection.TenantID,Consumer:providerConsumer,EventID:providerID},{TenantID:s.connection.TenantID,Consumer:checkpointConsumer,EventID:transitionID}},runtime.OutboxEvent{ID:outboxID,TenantID:s.connection.TenantID,AggregateType:"SOURCE_BINDING",AggregateID:binding.ID,EventType:"SourceBindingChanged",Payload:outbound,OccurredAt:now},now); if err!=nil { return sourceaccess.ChangeCaptureResult{},err }
    if !created { checkpoint,getErr:=s.adapter.checkpoints.Get(ctx,s.connection.TenantID,binding.ID,bindingVersion); if getErr!=nil { return sourceaccess.ChangeCaptureResult{},getErr }; if err:=s.reconcileDuplicate(ctx,checkpoint,definition.PositionKind,position,now); err!=nil { return sourceaccess.ChangeCaptureResult{},err }; return sourceaccess.ChangeCaptureResult{Accepted:true,Duplicate:true,Receipt:receipt},nil }
    if err:=s.advanceAccepted(ctx,checkpoint,definition.PositionKind,position,now); err!=nil { return sourceaccess.ChangeCaptureResult{},err }
    return sourceaccess.ChangeCaptureResult{Accepted:true,Receipt:receipt},nil
}

func (s *session) advanceAccepted(ctx context.Context, checkpoint sourceaccess.BindingCheckpoint, mode sourceaccess.CheckpointPositionKind, position sourceaccess.CheckpointPosition, now time.Time) error {
    _,err:=s.adapter.checkpoints.AdvanceAfterInbox(ctx,checkpoint,checkpointConsumer,position,now); if err==nil { return nil }; if !errors.Is(err,sourceaccess.ErrCheckpointConflict) { return err }
    current,getErr:=s.adapter.checkpoints.Get(ctx,checkpoint.TenantID,checkpoint.BindingID,checkpoint.BindingVersion); if getErr!=nil { return getErr }
    if mode==sourceaccess.CheckpointEventID { return nil }
    return s.advanceWatermarkFromCurrent(ctx,current,position,now)
}
func (s *session) reconcileDuplicate(ctx context.Context, checkpoint sourceaccess.BindingCheckpoint, mode sourceaccess.CheckpointPositionKind, position sourceaccess.CheckpointPosition, now time.Time) error {
    if checkpoint.Position.Kind==position.Kind && checkpoint.Position.Value==position.Value { return nil }
    transitionID,err:=sourceaccess.CheckpointInboxEventID(checkpoint,checkpointConsumer,position); if err!=nil { return err }
    processed,err:=s.adapter.store.InboxProcessed(ctx,checkpoint.TenantID,checkpointConsumer,transitionID); if err!=nil { return err }
    if processed { _,err=s.adapter.checkpoints.AdvanceAfterInbox(ctx,checkpoint,checkpointConsumer,position,now); if err==nil || errors.Is(err,sourceaccess.ErrCheckpointConflict) { return nil }; return err }
    if mode==sourceaccess.CheckpointWatermark { return s.advanceWatermarkFromCurrent(ctx,checkpoint,position,now) }
    return nil
}
func (s *session) advanceWatermarkFromCurrent(ctx context.Context, checkpoint sourceaccess.BindingCheckpoint, position sourceaccess.CheckpointPosition, now time.Time) error {
    for attempts:=0;attempts<4;attempts++ {
        if checkpoint.Position.Kind!="" { current,err:=watermarkValue(checkpoint.Position); if err!=nil { return err }; target,err:=watermarkValue(position); if err!=nil { return err }; if current>=target { return nil } }
        transitionID,err:=sourceaccess.CheckpointInboxEventID(checkpoint,checkpointConsumer,position); if err!=nil { return err }
        if _,err:=s.adapter.store.RecordInbox(ctx,checkpoint.TenantID,checkpointConsumer,transitionID,now); err!=nil { return err }
        _,err=s.adapter.checkpoints.AdvanceAfterInbox(ctx,checkpoint,checkpointConsumer,position,now); if err==nil { return nil }; if !errors.Is(err,sourceaccess.ErrCheckpointConflict) { return err }
        checkpoint,err=s.adapter.checkpoints.Get(ctx,checkpoint.TenantID,checkpoint.BindingID,checkpoint.BindingVersion); if err!=nil { return err }
    }
    return sourceaccess.ErrCheckpointConflict
}
func normalizePosition(mode sourceaccess.CheckpointPositionKind,event sourceaccess.ChangeEvent)(sourceaccess.CheckpointPosition,error){
    if mode==sourceaccess.CheckpointEventID { if event.Position==nil { return sourceaccess.CheckpointPosition{Kind:sourceaccess.CheckpointEventID,Value:event.EventID},nil }; if event.Position.Kind!=sourceaccess.CheckpointEventID || event.Position.Value!=event.EventID { return sourceaccess.CheckpointPosition{},sourceaccess.ErrDefinitionInvalid }; return *event.Position,nil }
    if event.Position==nil || event.Position.Kind!=sourceaccess.CheckpointWatermark { return sourceaccess.CheckpointPosition{},sourceaccess.ErrDefinitionInvalid }; if _,err:=watermarkValue(*event.Position);err!=nil{return sourceaccess.CheckpointPosition{},err};return *event.Position,nil
}
func requireForwardWatermark(current,target sourceaccess.CheckpointPosition)error{if current.Kind==""{return nil};if current.Kind!=sourceaccess.CheckpointWatermark{return sourceaccess.ErrCheckpointConflict};left,err:=watermarkValue(current);if err!=nil{return err};right,err:=watermarkValue(target);if err!=nil{return err};if right<=left{return sourceaccess.ErrCheckpointConflict};return nil}
func watermarkValue(position sourceaccess.CheckpointPosition)(uint64,error){if position.Kind!=sourceaccess.CheckpointWatermark||position.Value==""||strings.HasPrefix(position.Value,"+"){return 0,sourceaccess.ErrDefinitionInvalid};value,err:=strconv.ParseUint(position.Value,10,64);if err!=nil||strconv.FormatUint(value,10)!=position.Value{return 0,sourceaccess.ErrDefinitionInvalid};return value,nil}

func decodeRecord(payload json.RawMessage,fields []sourceaccess.NativeField,selected []string)(sourceaccess.Record,error){
    decoder:=json.NewDecoder(bytes.NewReader(payload));decoder.UseNumber();var object map[string]any;if err:=decoder.Decode(&object);err!=nil{return nil,sourceaccess.ErrDefinitionInvalid};var trailing any;if err:=decoder.Decode(&trailing);err!=io.EOF{return nil,sourceaccess.ErrDefinitionInvalid}
    schema:=make(map[string]sourceaccess.NativeField,len(fields));for _,field:=range fields{schema[field.Name]=field};for name:=range object{if _,exists:=schema[name];!exists{return nil,sourceaccess.ErrSchemaDrift}}
    for name,field:=range schema{value,exists:=object[name];if !exists{if !field.Nullable{return nil,sourceaccess.ErrSchemaDrift};continue};if value==nil&&!field.Nullable{return nil,sourceaccess.ErrSchemaDrift};if _,err:=scalarFromJSON(value,field);err!=nil{return nil,err}}
    record:=make(sourceaccess.Record,len(selected));for _,name:=range selected{field,exists:=schema[name];if !exists{return nil,sourceaccess.ErrSchemaDrift};value,exists:=object[name];if !exists||value==nil{record[name]=sourceaccess.Scalar{Kind:sourceaccess.ScalarNull};continue};scalar,err:=scalarFromJSON(value,field);if err!=nil{return nil,err};record[name]=scalar};return record,nil
}
func scalarFromJSON(value any,field sourceaccess.NativeField)(sourceaccess.Scalar,error){if value==nil{return sourceaccess.Scalar{Kind:sourceaccess.ScalarNull},nil};switch field.NativeType{case"json:string":typed,ok:=value.(string);if !ok||len(typed)>64<<10{return sourceaccess.Scalar{},sourceaccess.ErrSchemaDrift};return sourceaccess.Scalar{Kind:sourceaccess.ScalarString,Text:typed},nil;case"json:number":typed,ok:=value.(json.Number);if !ok{return sourceaccess.Scalar{},sourceaccess.ErrSchemaDrift};return sourceaccess.Scalar{Kind:sourceaccess.ScalarNumber,Text:typed.String()},nil;case"json:boolean":typed,ok:=value.(bool);if !ok{return sourceaccess.Scalar{},sourceaccess.ErrSchemaDrift};return sourceaccess.Scalar{Kind:sourceaccess.ScalarBool,Text:strconv.FormatBool(typed)},nil;case"json:time":typed,ok:=value.(string);if !ok{return sourceaccess.Scalar{},sourceaccess.ErrSchemaDrift};parsed,err:=time.Parse(time.RFC3339Nano,typed);if err!=nil{return sourceaccess.Scalar{},sourceaccess.ErrSchemaDrift};return sourceaccess.Scalar{Kind:sourceaccess.ScalarTime,Text:parsed.UTC().Format(time.RFC3339Nano)},nil;default:return sourceaccess.Scalar{},sourceaccess.ErrUnsupportedValue}}
func nativeSchemaFingerprint(fields []sourceaccess.NativeField)(string,error){canonical:=append([]sourceaccess.NativeField(nil),fields...);sort.Slice(canonical,func(i,j int)bool{return canonical[i].Name<canonical[j].Name});encoded,err:=json.Marshal(canonical);if err!=nil{return"",err};hash:=sha256.Sum256(encoded);return hex.EncodeToString(hash[:]),nil}
func providerInboxEventID(connection sourceaccess.Connection,binding sourceaccess.Binding,eventID string)string{hash:=sha256.New();_,_=fmt.Fprintf(hash,"%s\x1f%s\x1f%s\x1f%s\x1f%s",connection.TenantID,connection.SourceID,binding.ID,binding.Version,eventID);return"source-webhook:"+hex.EncodeToString(hash.Sum(nil))}
func webhookRetryIdentity(view sourceaccess.View,binding sourceaccess.Binding,eventID,position string,payload []byte)string{payloadHash:=sha256.Sum256(payload);hash:=sha256.New();_,_=fmt.Fprintf(hash,"%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s",view.ConnectionID,view.ID,view.Version,binding.ID,binding.Version,eventID,position);_,_=hash.Write(payloadHash[:]);return"webhook-event:"+hex.EncodeToString(hash.Sum(nil))}
var _ sourceaccess.Adapter=(*Adapter)(nil);var _ sourceaccess.Session=(*session)(nil);var _ sourceaccess.SchemaReader=(*session)(nil);var _ sourceaccess.ChangeReceiver=(*session)(nil)
''')

write("internal/sourceevent/adapter_test.go", r'''package sourceevent

import (
    "context"
    "encoding/json"
    "errors"
    "testing"
    "time"
    "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
    "github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type checkpointRepo struct{ value sourceaccess.BindingCheckpoint }
func(r *checkpointRepo)EnsureBindingCheckpoint(_ context.Context,tenant,source,binding string,version int64,now time.Time)(sourceaccess.BindingCheckpoint,error){if r.value.BindingID==""{r.value=sourceaccess.BindingCheckpoint{TenantID:tenant,SourceID:source,BindingID:binding,BindingVersion:version,CreatedAt:now,UpdatedAt:now}};return r.value,nil}
func(r *checkpointRepo)BindingCheckpoint(_ context.Context,tenant,binding string,version int64)(sourceaccess.BindingCheckpoint,error){return r.value,nil}
func(r *checkpointRepo)AdvanceBindingCheckpoint(_ context.Context,expected sourceaccess.BindingCheckpoint,position sourceaccess.CheckpointPosition,at time.Time)(sourceaccess.BindingCheckpoint,error){if r.value.Generation!=expected.Generation||r.value.Position!=expected.Position{return sourceaccess.BindingCheckpoint{},sourceaccess.ErrCheckpointConflict};r.value.Position=position;r.value.Generation++;r.value.UpdatedAt=at;return r.value,nil}

func TestWatermarkCaptureIsReplaySafeAndMonotonic(t *testing.T){ctx:=context.Background();runtimeRepo:=runtime.NewMemoryRepository();checkpointsRepo:=&checkpointRepo{};adapter:=NewAdapter(runtimeRepo,sourceaccess.NewCheckpointService(checkpointsRepo,runtimeRepo));adapter.now=func()time.Time{return time.Date(2026,8,15,12,0,0,0,time.UTC)};s:=mustSession(t,adapter);view,binding:=eventContracts(t,s,sourceaccess.CheckpointWatermark);event:=sourceaccess.ChangeEvent{EventID:"evt-1",Position:&sourceaccess.CheckpointPosition{Kind:sourceaccess.CheckpointWatermark,Value:"10"},Payload:json.RawMessage(`{"account_id":"A1","risk_score":7.5}`)};first,err:=s.CaptureChange(ctx,view,binding,event);if err!=nil||!first.Accepted||first.Duplicate{t.Fatalf("first=%#v err=%v",first,err)};duplicate,err:=s.CaptureChange(ctx,view,binding,event);if err!=nil||!duplicate.Duplicate{t.Fatalf("duplicate=%#v err=%v",duplicate,err)};if checkpointsRepo.value.Position.Value!="10"||checkpointsRepo.value.Generation!=1{t.Fatalf("checkpoint=%#v",checkpointsRepo.value)};stale:=sourceaccess.ChangeEvent{EventID:"evt-2",Position:&sourceaccess.CheckpointPosition{Kind:sourceaccess.CheckpointWatermark,Value:"9"},Payload:event.Payload};if _,err:=s.CaptureChange(ctx,view,binding,stale);!errors.Is(err,sourceaccess.ErrCheckpointConflict){t.Fatalf("stale watermark err=%v",err)};claimed,err:=runtimeRepo.ClaimOutbox(ctx,"test",time.Now().UTC(),time.Minute,10);if err!=nil{t.Fatal(err)};if len(claimed)!=1{t.Fatalf("outbox events=%d want 1",len(claimed))}}
func TestDuplicateEventIDDoesNotRegressLaterCheckpoint(t *testing.T){ctx:=context.Background();runtimeRepo:=runtime.NewMemoryRepository();checkpointsRepo:=&checkpointRepo{};s:=mustSession(t,NewAdapter(runtimeRepo,sourceaccess.NewCheckpointService(checkpointsRepo,runtimeRepo)));view,binding:=eventContracts(t,s,sourceaccess.CheckpointEventID);for _,eventID:=range[]string{"evt-a","evt-b"}{if _,err:=s.CaptureChange(ctx,view,binding,sourceaccess.ChangeEvent{EventID:eventID,Payload:json.RawMessage(`{"account_id":"A1","risk_score":1}`)});err!=nil{t.Fatal(err)}};if checkpointsRepo.value.Position.Value!="evt-b"{t.Fatalf("checkpoint=%#v",checkpointsRepo.value)};result,err:=s.CaptureChange(ctx,view,binding,sourceaccess.ChangeEvent{EventID:"evt-a",Payload:json.RawMessage(`{"account_id":"A1","risk_score":1}`)});if err!=nil||!result.Duplicate{t.Fatalf("replay=%#v err=%v",result,err)};if checkpointsRepo.value.Position.Value!="evt-b"{t.Fatalf("duplicate regressed checkpoint=%#v",checkpointsRepo.value)}}
func TestWebhookRejectsSchemaDrift(t *testing.T){runtimeRepo:=runtime.NewMemoryRepository();checkpointsRepo:=&checkpointRepo{};s:=mustSession(t,NewAdapter(runtimeRepo,sourceaccess.NewCheckpointService(checkpointsRepo,runtimeRepo)));view,binding:=eventContracts(t,s,sourceaccess.CheckpointEventID);_,err:=s.CaptureChange(context.Background(),view,binding,sourceaccess.ChangeEvent{EventID:"evt-x",Payload:json.RawMessage(`{"account_id":"A1","risk_score":1,"unexpected":true}`)});if !errors.Is(err,sourceaccess.ErrSchemaDrift){t.Fatalf("schema drift err=%v",err)}}
func mustSession(t *testing.T,adapter *Adapter)*session{t.Helper();connection:=sourceaccess.Connection{TenantID:"tenant-a",ID:"connection-a",SourceID:"source-a",Version:"1",AdapterKind:sourceaccess.AdapterWebhookEvent,AdapterVersion:sourceaccess.WebhookEventAdapterVersion,Definition:json.RawMessage(`{}`)};value,err:=adapter.Open(context.Background(),connection,sourceaccess.EnvironmentSecretResolver{});if err!=nil{t.Fatal(err)};return value.(*session)}
func eventContracts(t *testing.T,s *session,mode sourceaccess.CheckpointPositionKind)(sourceaccess.View,sourceaccess.Binding){t.Helper();view:=sourceaccess.View{ID:"view-a",ConnectionID:"connection-a",Version:"1",OutputKind:sourceaccess.OutputRecords,Definition:json.RawMessage(`{"position_kind":"`+string(mode)+`","fields":[{"name":"account_id","native_type":"json:string","nullable":false},{"name":"risk_score","native_type":"json:number","nullable":false}]}`)};inspected,err:=s.Inspect(context.Background(),view);if err!=nil{t.Fatal(err)};view.NativeSchema,view.SchemaFingerprint=inspected.Fields,inspected.Receipt.SchemaFingerprint;binding:=sourceaccess.Binding{ID:"binding-a",ViewID:"view-a",Version:"1",Purpose:"event",Operations:[]sourceaccess.Operation{sourceaccess.OperationChanges},SelectedFields:[]string{"account_id","risk_score"},Limits:sourceaccess.DefaultResourceLimits()};return view,binding}
''')

# Service-only operational ingress.
write("internal/httpapi/source_event_handlers.go", r'''package httpapi

import (
    "encoding/json"
    "errors"
    "net/http"
    "github.com/CloudSpaceLab/clearsight-grc/internal/identity"
    "github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
    "github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type sourceEventRequest struct { TenantID string `json:"tenant_id"`; EventID string `json:"event_id"`; Position *sourceaccess.CheckpointPosition `json:"position,omitempty"`; Payload json.RawMessage `json:"payload"` }
func(a *API)ingestSourceBindingEvent(w http.ResponseWriter,r *http.Request){service,ok:=a.sourceCatalog(w);if !ok{return};actor,err:=identity.Require(r.Context());if err!=nil{httpx.WriteError(w,http.StatusUnauthorized,"identity_required","A verified service identity is required.");return};if actor.Kind!="SERVICE"{httpx.WriteError(w,http.StatusForbidden,"service_identity_required","Source events must be delivered by a verified service principal.");return};var input sourceEventRequest;if err:=httpx.DecodeJSON(w,r,&input);err!=nil{httpx.WriteError(w,http.StatusBadRequest,"invalid_request",err.Error());return};result,err:=service.CaptureBindingChange(r.Context(),actor.TenantID,r.PathValue("id"),0,sourceaccess.ChangeEvent{EventID:input.EventID,Position:input.Position,Payload:input.Payload});if err!=nil{writeSourceEventError(w,err);return};httpx.WriteJSON(w,http.StatusAccepted,result)}
func writeSourceEventError(w http.ResponseWriter,err error){switch{case errors.Is(err,sourceaccess.ErrCheckpointConflict):httpx.WriteError(w,http.StatusConflict,"source_event_out_of_order","The event position is not newer than the current governed checkpoint.");case errors.Is(err,sourceaccess.ErrSchemaDrift):httpx.WriteError(w,http.StatusUnprocessableEntity,"source_schema_drift","The event payload does not match the governed source schema.");default:writeSourceCatalogError(w,err)}}
''')
replace_once("internal/httpapi/route_registry.go",
'''\t\twithPermission(read("/api/v1/config/source-bindings/{binding_id}/where-used", a.sourceCatalogWhereUsed(sourceaccess.UsageBinding, "binding_id")), identity.PermissionConfigRead),\n\n\t\tread("/api/v1/evidence/sources", a.listEvidenceSources),\n''',
'''\t\twithPermission(read("/api/v1/config/source-bindings/{binding_id}/where-used", a.sourceCatalogWhereUsed(sourceaccess.UsageBinding, "binding_id")), identity.PermissionConfigRead),\n\t\tmaterialService("/api/v1/source-bindings/{id}/events", "source.binding.event.ingest", a.ingestSourceBindingEvent, commandPolicy{ObjectType: "SOURCE_BINDING", Responsibility: authority.ResponsibilityPerformer, Materiality: 2, ActorField: noActorField}),\n\n\t\tread("/api/v1/evidence/sources", a.listEvidenceSources),\n''')
replace_once("api/runtime.openapi.json",
'''    "/api/v1/config/source-bindings/{binding_id}/where-used": { "get": { "operationId": "sourceBindingWhereUsed", "x-clearsight-route-class": "AUTHENTICATED_READ", "x-clearsight-permission": "CONFIG_READ" } },\n''',
'''    "/api/v1/config/source-bindings/{binding_id}/where-used": { "get": { "operationId": "sourceBindingWhereUsed", "x-clearsight-route-class": "AUTHENTICATED_READ", "x-clearsight-permission": "CONFIG_READ" } },\n    "/api/v1/source-bindings/{id}/events": { "post": { "operationId": "ingestSourceBindingEvent", "x-clearsight-route-class": "MATERIAL_COMMAND" } },\n''')

# API composition.
replace_once("cmd/api/services_memory.go",'''\t"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"\n''','''\t"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/runtime"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/sourceevent"\n''')
replace_once("cmd/api/services_memory.go",'''\tadapters := sourceaccess.DefaultCatalogAdapters()\n\tadapters[sourceaccess.AdapterTabularArtifact] = documentService.SourceAccessAdapter()\n\tsourceCatalog := sourceaccess.NewCatalogService(sourceaccess.NewMemoryCatalogRepository(sourceScopes), sourceaccess.EnvironmentSecretResolver{}, adapters)\n''','''\tcatalogRepo := sourceaccess.NewMemoryCatalogRepository(sourceScopes)\n\truntimeRepo := runtime.NewMemoryRepository()\n\tcheckpoints := sourceaccess.NewCheckpointService(sourceaccess.NewMemoryCheckpointRepository(catalogRepo), runtimeRepo)\n\tadapters := sourceaccess.DefaultCatalogAdapters()\n\tadapters[sourceaccess.AdapterTabularArtifact] = documentService.SourceAccessAdapter()\n\tadapters[sourceaccess.AdapterWebhookEvent] = sourceevent.NewAdapter(runtimeRepo, checkpoints)\n\tsourceCatalog := sourceaccess.NewCatalogService(catalogRepo, sourceaccess.EnvironmentSecretResolver{}, adapters)\n''')
replace_once("cmd/api/services_memory.go",'''\t\tAutonomy: auto, BankVerticals: verticals, BackgroundJobs: operations.NewService(continuityRepo), Close: func() {},\n''','''\t\tAutonomy: auto, BankVerticals: verticals, BackgroundJobs: operations.NewService(continuityRepo, runtimeRepo), Close: func() {},\n''')
replace_once("cmd/api/services_postgres.go",'''\t"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/today"\n''','''\t"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/sourceevent"\n\t"github.com/CloudSpaceLab/clearsight-grc/internal/today"\n''')
replace_once("cmd/api/services_postgres.go",'''\tadapters := sourceaccess.DefaultCatalogAdapters()\n\tadapters[sourceaccess.AdapterTabularArtifact] = documentService.SourceAccessAdapter()\n\tsourceCatalog := sourceaccess.NewCatalogService(sourceaccess.NewPostgresCatalogRepository(pool), sourceaccess.EnvironmentSecretResolver{}, adapters)\n\tcontinuityRepo := continuity.NewReliablePostgresRepository(pool)\n\tcontinuityService := continuity.NewService(continuityRepo)\n\tcoverageService := documentcoverage.NewService(documentcoverage.NewPostgresRepository(pool), documentService, continuityService)\n\truntimeRepo := runtime.NewPostgresRepository(pool)\n''','''\tcatalogRepo := sourceaccess.NewPostgresCatalogRepository(pool)\n\truntimeRepo := runtime.NewPostgresRepository(pool)\n\tcheckpoints := sourceaccess.NewCheckpointService(sourceaccess.NewPostgresCheckpointRepository(pool), runtimeRepo)\n\tadapters := sourceaccess.DefaultCatalogAdapters()\n\tadapters[sourceaccess.AdapterTabularArtifact] = documentService.SourceAccessAdapter()\n\tadapters[sourceaccess.AdapterWebhookEvent] = sourceevent.NewAdapter(runtimeRepo, checkpoints)\n\tsourceCatalog := sourceaccess.NewCatalogService(catalogRepo, sourceaccess.EnvironmentSecretResolver{}, adapters)\n\tcontinuityRepo := continuity.NewReliablePostgresRepository(pool)\n\tcontinuityService := continuity.NewService(continuityRepo)\n\tcoverageService := documentcoverage.NewService(documentcoverage.NewPostgresRepository(pool), documentService, continuityService)\n''')
