package documentimport

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

	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type tabularSourceViewDefinition struct {
	DocumentID string `json:"document_id"`
	Resource   string `json:"resource,omitempty"`
}

type tabularArtifactAdapter struct{ service *Service }

type tabularArtifactSession struct {
	service    *Service
	connection sourceaccess.Connection
}

// SourceAccessAdapter exposes exact governed tabular imports through the shared
// source-access capability contract. The original artifact is the only row
// store: every operation reopens, verifies and reparses that immutable object.
func (s *Service) SourceAccessAdapter() sourceaccess.Adapter {
	return &tabularArtifactAdapter{service: s}
}

func (a *tabularArtifactAdapter) Open(ctx context.Context, connection sourceaccess.Connection, _ sourceaccess.SecretResolver) (sourceaccess.Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if a == nil || a.service == nil {
		return nil, sourceaccess.ErrConnection
	}
	if err := connection.Validate(); err != nil {
		return nil, err
	}
	if connection.AdapterKind != sourceaccess.AdapterTabularArtifact || connection.AdapterVersion != sourceaccess.TabularArtifactAdapterVersion {
		return nil, sourceaccess.ErrDefinitionInvalid
	}
	if strings.TrimSpace(connection.TenantID) == "" || connection.TenantID != strings.TrimSpace(connection.TenantID) || connection.SecretRef != "" || len(connection.Definition) != 0 {
		return nil, sourceaccess.ErrDefinitionInvalid
	}
	return &tabularArtifactSession{service: a.service, connection: connection}, nil
}

func (s *tabularArtifactSession) Connection() sourceaccess.Connection {
	if s == nil {
		return sourceaccess.Connection{}
	}
	return s.connection
}

func (s *tabularArtifactSession) Capabilities() sourceaccess.CapabilitySet {
	return sourceaccess.NewCapabilitySet(sourceaccess.CapabilityInspect, sourceaccess.CapabilityPage, sourceaccess.CapabilityLookup)
}

func (s *tabularArtifactSession) Close() error { return nil }

func (s *tabularArtifactSession) Inspect(ctx context.Context, view sourceaccess.View) (sourceaccess.SchemaResult, error) {
	document, resource, _, err := s.resolve(ctx, view)
	if err != nil {
		return sourceaccess.SchemaResult{}, err
	}
	fields := make([]sourceaccess.NativeField, 0, len(resource.Fields))
	for _, field := range resource.Fields {
		fields = append(fields, sourceaccess.NativeField{Name: field.Name, NativeType: field.NativeType, Nullable: field.Nullable})
	}
	if len(fields) == 0 || resource.SchemaFingerprint == "" {
		return sourceaccess.SchemaResult{}, sourceaccess.ErrExecution
	}
	fingerprint, err := sourceaccess.ViewFingerprint(view)
	if err != nil {
		return sourceaccess.SchemaResult{}, err
	}
	return sourceaccess.SchemaResult{
		Fields: fields,
		Receipt: s.receipt(document, resource, view, sourceaccess.Binding{}, sourceaccess.OperationInspect, int64(len(fields)), 0,
			sourceaccess.CompletenessComplete, fingerprint, nil, tabularRetryIdentity(document, view, sourceaccess.Binding{}, sourceaccess.OperationInspect, nil, nil)),
	}, nil
}

func (s *tabularArtifactSession) ReadPage(ctx context.Context, view sourceaccess.View, binding sourceaccess.Binding, request sourceaccess.PageRequest) (sourceaccess.RecordPage, error) {
	document, resource, definition, limits, err := s.validateBound(ctx, view, binding, sourceaccess.OperationPage)
	if err != nil {
		return sourceaccess.RecordPage{}, err
	}
	if len(binding.KeyFields) != 1 {
		return sourceaccess.RecordPage{}, sourceaccess.ErrCapabilityUnavailable
	}
	limit := request.Limit
	if limit < 0 {
		return sourceaccess.RecordPage{}, sourceaccess.ErrLimitExceeded
	}
	if limit == 0 {
		limit = limits.PageRows
	}
	if limit > limits.PageRows {
		return sourceaccess.RecordPage{}, sourceaccess.ErrLimitExceeded
	}
	afterRow := 0
	if request.After != nil {
		if err := request.After.ValidateInput(); err != nil || request.After.Kind != sourceaccess.ScalarNumber {
			return sourceaccess.RecordPage{}, sourceaccess.ErrDefinitionInvalid
		}
		parsed, parseErr := strconv.Atoi(request.After.Text)
		if parseErr != nil || parsed < 1 {
			return sourceaccess.RecordPage{}, sourceaccess.ErrDefinitionInvalid
		}
		afterRow = parsed
	}
	data, err := s.readArtifact(ctx, document)
	if err != nil {
		return sourceaccess.RecordPage{}, err
	}
	records := make([]sourceaccess.Record, 0, limit)
	seenKeys := make(map[string]struct{}, limit)
	byteCount := int64(0)
	lastRow := 0
	more := false
	_, scanErr := scanTabularArtifact(ctx, document.Tabular.Format, data, s.service.extractionPolicy, definition.Resource, func(row tabularRow) error {
		if row.Number <= afterRow {
			return nil
		}
		if len(records) >= limit {
			more = true
			return errTabularStop
		}
		record, err := sourceRecord(row, binding.SelectedFields)
		if err != nil {
			return err
		}
		key := record[binding.KeyFields[0]]
		if key.Kind == sourceaccess.ScalarNull {
			return sourceaccess.ErrExecution
		}
		identity := sourceScalarIdentity(key)
		if _, exists := seenKeys[identity]; exists {
			return sourceaccess.ErrExecution
		}
		seenKeys[identity] = struct{}{}
		encoded, err := json.Marshal(record)
		if err != nil {
			return sourceaccess.ErrExecution
		}
		byteCount += int64(len(encoded))
		if byteCount > limits.ResponseBytes {
			return sourceaccess.ErrLimitExceeded
		}
		records = append(records, record)
		lastRow = row.Number
		return nil
	})
	if scanErr != nil && !errors.Is(scanErr, errTabularStop) {
		return sourceaccess.RecordPage{}, mapTabularSourceError(scanErr)
	}
	fingerprint, err := sourceaccess.BindingFingerprint(view, binding)
	if err != nil {
		return sourceaccess.RecordPage{}, err
	}
	completeness := sourceaccess.CompletenessComplete
	if more || resource.RowsRejected > 0 {
		completeness = sourceaccess.CompletenessPartial
	}
	var next *sourceaccess.Scalar
	var position *sourceaccess.CheckpointPosition
	if more {
		if lastRow < 1 {
			return sourceaccess.RecordPage{}, sourceaccess.ErrExecution
		}
		cursor := sourceaccess.Scalar{Kind: sourceaccess.ScalarNumber, Text: strconv.Itoa(lastRow)}
		next = &cursor
		position = &sourceaccess.CheckpointPosition{Kind: sourceaccess.CheckpointCursor, Value: cursor.Text}
	}
	return sourceaccess.RecordPage{
		Records: records,
		NextCursor: next,
		Receipt: s.receipt(document, resource, view, binding, sourceaccess.OperationPage, int64(len(records)), byteCount,
			completeness, fingerprint, position, tabularRetryIdentity(document, view, binding, sourceaccess.OperationPage, request.After, nil)),
	}, nil
}

func (s *tabularArtifactSession) Lookup(ctx context.Context, view sourceaccess.View, binding sourceaccess.Binding, request sourceaccess.LookupRequest) (sourceaccess.LookupResult, error) {
	document, resource, definition, limits, err := s.validateBound(ctx, view, binding, sourceaccess.OperationLookup)
	if err != nil {
		return sourceaccess.LookupResult{}, err
	}
	if len(binding.KeyFields) != 1 {
		return sourceaccess.LookupResult{}, sourceaccess.ErrCapabilityUnavailable
	}
	if len(request.Values) == 0 || len(request.Values) > limits.LookupValues {
		return sourceaccess.LookupResult{}, sourceaccess.ErrLimitExceeded
	}
	requested := make(map[string]struct{}, len(request.Values))
	var kind sourceaccess.ScalarKind
	for index, value := range request.Values {
		if err := value.ValidateInput(); err != nil {
			return sourceaccess.LookupResult{}, err
		}
		if index == 0 {
			kind = value.Kind
		} else if value.Kind != kind {
			return sourceaccess.LookupResult{}, sourceaccess.ErrDefinitionInvalid
		}
		identity := sourceScalarIdentity(value)
		if _, exists := requested[identity]; exists {
			return sourceaccess.LookupResult{}, sourceaccess.ErrDefinitionInvalid
		}
		requested[identity] = struct{}{}
	}
	if expected, ok := tabularExpectedKind(resource, binding.KeyFields[0]); ok && expected != kind {
		return sourceaccess.LookupResult{}, sourceaccess.ErrDefinitionInvalid
	}
	data, err := s.readArtifact(ctx, document)
	if err != nil {
		return sourceaccess.LookupResult{}, err
	}
	records := make([]sourceaccess.Record, 0, len(request.Values))
	seenResults := make(map[string]struct{}, len(request.Values))
	byteCount := int64(0)
	_, scanErr := scanTabularArtifact(ctx, document.Tabular.Format, data, s.service.extractionPolicy, definition.Resource, func(row tabularRow) error {
		cell, exists := row.Values[binding.KeyFields[0]]
		if !exists || cell.kind == "null" {
			return sourceaccess.ErrExecution
		}
		key, err := sourceScalar(cell)
		if err != nil {
			return err
		}
		identity := sourceScalarIdentity(key)
		if _, wanted := requested[identity]; !wanted {
			return nil
		}
		if _, duplicate := seenResults[identity]; duplicate {
			return sourceaccess.ErrExecution
		}
		seenResults[identity] = struct{}{}
		record, err := sourceRecord(row, binding.SelectedFields)
		if err != nil {
			return err
		}
		encoded, err := json.Marshal(record)
		if err != nil {
			return sourceaccess.ErrExecution
		}
		byteCount += int64(len(encoded))
		if byteCount > limits.ResponseBytes {
			return sourceaccess.ErrLimitExceeded
		}
		records = append(records, record)
		return nil
	})
	if scanErr != nil {
		return sourceaccess.LookupResult{}, mapTabularSourceError(scanErr)
	}
	fingerprint, err := sourceaccess.BindingFingerprint(view, binding)
	if err != nil {
		return sourceaccess.LookupResult{}, err
	}
	completeness := sourceaccess.CompletenessComplete
	if resource.RowsRejected > 0 {
		completeness = sourceaccess.CompletenessPartial
	}
	return sourceaccess.LookupResult{
		Records: records,
		Receipt: s.receipt(document, resource, view, binding, sourceaccess.OperationLookup, int64(len(records)), byteCount,
			completeness, fingerprint, nil, tabularRetryIdentity(document, view, binding, sourceaccess.OperationLookup, nil, request.Values)),
	}, nil
}

func (s *tabularArtifactSession) validateBound(ctx context.Context, view sourceaccess.View, binding sourceaccess.Binding, operation sourceaccess.Operation) (Document, TabularResource, tabularSourceViewDefinition, sourceaccess.ResourceLimits, error) {
	document, resource, definition, err := s.resolve(ctx, view)
	if err != nil {
		return Document{}, TabularResource{}, tabularSourceViewDefinition{}, sourceaccess.ResourceLimits{}, err
	}
	if err := binding.Validate(view); err != nil {
		return Document{}, TabularResource{}, tabularSourceViewDefinition{}, sourceaccess.ResourceLimits{}, err
	}
	if !binding.Allows(operation) {
		return Document{}, TabularResource{}, tabularSourceViewDefinition{}, sourceaccess.ResourceLimits{}, sourceaccess.ErrCapabilityUnavailable
	}
	limits, err := binding.NormalizedLimits()
	if err != nil {
		return Document{}, TabularResource{}, tabularSourceViewDefinition{}, sourceaccess.ResourceLimits{}, err
	}
	return document, resource, definition, limits, nil
}

func (s *tabularArtifactSession) resolve(ctx context.Context, view sourceaccess.View) (Document, TabularResource, tabularSourceViewDefinition, error) {
	if s == nil || s.service == nil {
		return Document{}, TabularResource{}, tabularSourceViewDefinition{}, sourceaccess.ErrConnection
	}
	if err := view.Validate(s.connection); err != nil {
		return Document{}, TabularResource{}, tabularSourceViewDefinition{}, err
	}
	definition, err := decodeTabularSourceView(view.Definition)
	if err != nil {
		return Document{}, TabularResource{}, tabularSourceViewDefinition{}, err
	}
	document, err := s.service.Get(ctx, s.connection.TenantID, definition.DocumentID)
	if err != nil {
		return Document{}, TabularResource{}, tabularSourceViewDefinition{}, sourceaccess.ErrExecution
	}
	if document.Tabular == nil || document.Tabular.ParserVersion != TabularParserVersion || document.Tabular.FatalError != "" {
		return Document{}, TabularResource{}, tabularSourceViewDefinition{}, sourceaccess.ErrExecution
	}
	if definition.Resource == "" {
		if len(document.Tabular.Resources) != 1 {
			return Document{}, TabularResource{}, tabularSourceViewDefinition{}, sourceaccess.ErrDefinitionInvalid
		}
		definition.Resource = document.Tabular.Resources[0].Name
	}
	var resource *TabularResource
	for index := range document.Tabular.Resources {
		if document.Tabular.Resources[index].Name == definition.Resource {
			resource = &document.Tabular.Resources[index]
			break
		}
	}
	if resource == nil || len(resource.Fields) == 0 || resource.SchemaFingerprint == "" {
		return Document{}, TabularResource{}, tabularSourceViewDefinition{}, sourceaccess.ErrExecution
	}
	if view.SchemaFingerprint != "" && view.SchemaFingerprint != resource.SchemaFingerprint {
		return Document{}, TabularResource{}, tabularSourceViewDefinition{}, sourceaccess.ErrSchemaDrift
	}
	return document, *resource, definition, nil
}

func decodeTabularSourceView(raw json.RawMessage) (tabularSourceViewDefinition, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var definition tabularSourceViewDefinition
	if err := decoder.Decode(&definition); err != nil {
		return tabularSourceViewDefinition{}, sourceaccess.ErrDefinitionInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return tabularSourceViewDefinition{}, sourceaccess.ErrDefinitionInvalid
	}
	definition.DocumentID = strings.TrimSpace(definition.DocumentID)
	definition.Resource = strings.TrimSpace(definition.Resource)
	if definition.DocumentID == "" || len(definition.DocumentID) > sourceaccess.HardMaxIdentifierBytes || len(definition.Resource) > sourceaccess.HardMaxIdentifierBytes {
		return tabularSourceViewDefinition{}, sourceaccess.ErrDefinitionInvalid
	}
	return definition, nil
}

func (s *tabularArtifactSession) readArtifact(ctx context.Context, document Document) ([]byte, error) {
	stream, err := s.service.store.Open(ctx, document.StorageKey)
	if err != nil {
		return nil, sourceaccess.ErrConnection
	}
	defer stream.Close()
	data, err := readBounded(ctx, stream, s.service.maximumBytes())
	if err != nil {
		return nil, mapTabularSourceError(err)
	}
	digest := sha256.Sum256(data)
	if int64(len(data)) != document.SizeBytes || hex.EncodeToString(digest[:]) != document.SHA256 {
		return nil, sourceaccess.ErrExecution
	}
	return data, nil
}

func sourceRecord(row tabularRow, selected []string) (sourceaccess.Record, error) {
	record := make(sourceaccess.Record, len(selected))
	for _, field := range selected {
		cell, exists := row.Values[field]
		if !exists {
			record[field] = sourceaccess.Scalar{Kind: sourceaccess.ScalarNull}
			continue
		}
		value, err := sourceScalar(cell)
		if err != nil {
			return nil, err
		}
		record[field] = value
	}
	return record, nil
}

func sourceScalar(cell tabularCell) (sourceaccess.Scalar, error) {
	switch cell.kind {
	case "null":
		return sourceaccess.Scalar{Kind: sourceaccess.ScalarNull}, nil
	case "string":
		return sourceaccess.Scalar{Kind: sourceaccess.ScalarString, Text: cell.text}, nil
	case "number":
		return sourceaccess.Scalar{Kind: sourceaccess.ScalarNumber, Text: cell.text}, nil
	case "bool":
		return sourceaccess.Scalar{Kind: sourceaccess.ScalarBool, Text: cell.text}, nil
	default:
		return sourceaccess.Scalar{}, sourceaccess.ErrUnsupportedValue
	}
}

func sourceScalarIdentity(value sourceaccess.Scalar) string {
	return string(value.Kind) + "\x1f" + value.Text
}

func tabularExpectedKind(resource TabularResource, fieldName string) (sourceaccess.ScalarKind, bool) {
	for _, field := range resource.Fields {
		if field.Name != fieldName {
			continue
		}
		switch field.NativeType {
		case "tabular:string":
			return sourceaccess.ScalarString, true
		case "tabular:number":
			return sourceaccess.ScalarNumber, true
		case "tabular:boolean":
			return sourceaccess.ScalarBool, true
		default:
			return "", false
		}
	}
	return "", false
}

func mapTabularSourceError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, ErrTooLarge) || errors.Is(err, ErrResourceLimit) || errors.Is(err, sourceaccess.ErrLimitExceeded) {
		return sourceaccess.ErrLimitExceeded
	}
	if errors.Is(err, sourceaccess.ErrDefinitionInvalid) || errors.Is(err, sourceaccess.ErrUnsupportedValue) || errors.Is(err, sourceaccess.ErrExecution) {
		return err
	}
	return sourceaccess.ErrExecution
}

func tabularRetryIdentity(document Document, view sourceaccess.View, binding sourceaccess.Binding, operation sourceaccess.Operation, after *sourceaccess.Scalar, lookup []sourceaccess.Scalar) string {
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s", document.SHA256, view.ConnectionID, view.ID, view.Version, binding.ID, binding.Version, operation)
	if after != nil {
		_, _ = fmt.Fprintf(hash, "\x1f%s", sourceScalarIdentity(*after))
	}
	identities := make([]string, 0, len(lookup))
	for _, value := range lookup {
		identities = append(identities, sourceScalarIdentity(value))
	}
	sort.Strings(identities)
	for _, identity := range identities {
		_, _ = fmt.Fprintf(hash, "\x1f%s", identity)
	}
	return "tabular-artifact:" + hex.EncodeToString(hash.Sum(nil))
}

func (s *tabularArtifactSession) receipt(document Document, resource TabularResource, view sourceaccess.View, binding sourceaccess.Binding, operation sourceaccess.Operation, count, byteCount int64, completeness sourceaccess.Completeness, definitionFingerprint string, position *sourceaccess.CheckpointPosition, retryIdentity string) sourceaccess.OperationReceipt {
	receipt := sourceaccess.OperationReceipt{
		SourceID: s.connection.SourceID, ConnectionID: s.connection.ID, ConnectionVersion: s.connection.Version,
		AdapterKind: s.connection.AdapterKind, AdapterVersion: s.connection.AdapterVersion,
		ViewID: view.ID, ViewVersion: view.Version, DefinitionFingerprint: definitionFingerprint,
		SchemaFingerprint: resource.SchemaFingerprint, Operation: operation, ObservedAt: s.service.now().UTC(),
		Count: count, Bytes: byteCount, Completeness: completeness, Position: position, RetryIdentity: retryIdentity,
		ArtifactID: document.ID, ArtifactSHA256: document.SHA256, ParserVersion: document.Tabular.ParserVersion,
	}
	if binding.ID != "" {
		receipt.BindingID = binding.ID
		receipt.BindingVersion = binding.Version
	}
	return receipt
}

var _ sourceaccess.Adapter = (*tabularArtifactAdapter)(nil)
var _ sourceaccess.Session = (*tabularArtifactSession)(nil)
var _ sourceaccess.SchemaReader = (*tabularArtifactSession)(nil)
var _ sourceaccess.PageReader = (*tabularArtifactSession)(nil)
var _ sourceaccess.LookupReader = (*tabularArtifactSession)(nil)
