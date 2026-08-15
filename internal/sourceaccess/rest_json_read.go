package sourceaccess

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type restJSONResponse struct {
	root        any
	bodyBytes   int64
	etag        string
	notModified bool
}

func (s *RESTJSONSession) Inspect(ctx context.Context, view View) (SchemaResult, error) {
	if err := s.ready(view); err != nil {
		return SchemaResult{}, err
	}
	definition, err := decodeRESTJSONView(view)
	if err != nil {
		return SchemaResult{}, err
	}
	operationCtx, cancel := context.WithTimeout(ctx, DefaultOperationTimeout)
	defer cancel()
	query := cloneRESTQuery(definition.FixedQuery)
	if definition.Pagination.PageSizeQueryParam != "" {
		query.Set(definition.Pagination.PageSizeQueryParam, strconv.Itoa(DefaultPreviewRows))
	}
	response, err := s.getJSON(operationCtx, definition.Path, query, "", DefaultResponseBytes)
	if err != nil {
		return SchemaResult{}, err
	}
	if response.notModified {
		return SchemaResult{}, ErrExecution
	}
	records, err := extractRESTRecords(response.root, definition.RecordsPointer, DefaultPreviewRows)
	if err != nil {
		return SchemaResult{}, err
	}
	if len(records) == 0 {
		return SchemaResult{}, fmt.Errorf("%w: REST schema inspection requires at least one bounded record", ErrExecution)
	}
	fields, schemaFingerprint, err := inferRESTSchema(records)
	if err != nil {
		return SchemaResult{}, err
	}
	fingerprint, err := ViewFingerprint(view)
	if err != nil {
		return SchemaResult{}, err
	}
	return SchemaResult{
		Fields: fields,
		Receipt: s.receipt(view, Binding{}, OperationInspect, int64(len(fields)), response.bodyBytes,
			CompletenessComplete, fingerprint, schemaFingerprint, nil, restRetryIdentity(view, Binding{}, OperationInspect, nil, nil)),
	}, nil
}

func (s *RESTJSONSession) ReadPage(ctx context.Context, view View, binding Binding, request PageRequest) (RecordPage, error) {
	definition, limits, err := s.validateBoundOperation(view, binding, OperationPage)
	if err != nil {
		return RecordPage{}, err
	}
	if len(binding.KeyFields) != 1 {
		return RecordPage{}, fmt.Errorf("%w: REST/JSON page requires one stable key", ErrCapabilityUnavailable)
	}
	limit := request.Limit
	if limit < 0 {
		return RecordPage{}, ErrLimitExceeded
	}
	if limit == 0 {
		limit = limits.PageRows
	}
	if limit > limits.PageRows {
		return RecordPage{}, ErrLimitExceeded
	}
	if request.After != nil {
		if err := request.After.ValidateInput(); err != nil {
			return RecordPage{}, err
		}
	}

	query := cloneRESTQuery(definition.FixedQuery)
	if definition.Pagination.PageSizeQueryParam != "" {
		query.Set(definition.Pagination.PageSizeQueryParam, strconv.Itoa(limit))
	}
	ifNoneMatch := ""
	switch definition.Pagination.Mode {
	case RESTJSONPaginationNone:
		if request.After != nil {
			return RecordPage{}, fmt.Errorf("%w: non-paginated REST views do not accept a cursor", ErrDefinitionInvalid)
		}
	case RESTJSONPaginationCursor:
		if request.After != nil {
			query.Set(definition.Pagination.CursorQueryParam, request.After.Text)
		}
	case RESTJSONPaginationETag:
		if request.After != nil {
			if request.After.Kind != ScalarString {
				return RecordPage{}, fmt.Errorf("%w: REST ETag checkpoint must be a string", ErrDefinitionInvalid)
			}
			ifNoneMatch = request.After.Text
		}
	default:
		return RecordPage{}, ErrDefinitionInvalid
	}

	operationCtx, cancel := context.WithTimeout(ctx, limits.Timeout)
	defer cancel()
	response, err := s.getJSON(operationCtx, definition.Path, query, ifNoneMatch, limits.ResponseBytes)
	if err != nil {
		return RecordPage{}, err
	}
	if response.notModified {
		fingerprint, err := BindingFingerprint(view, binding)
		if err != nil {
			return RecordPage{}, err
		}
		return RecordPage{
			Records: []Record{},
			Receipt: s.receipt(view, binding, OperationPage, 0, 0, CompletenessComplete, fingerprint,
				view.SchemaFingerprint, nil, restRetryIdentity(view, binding, OperationPage, request.After, nil)),
		}, nil
	}
	fullRecords, err := extractRESTRecords(response.root, definition.RecordsPointer, limit)
	if err != nil {
		return RecordPage{}, err
	}
	schemaFingerprint := view.SchemaFingerprint
	if len(fullRecords) > 0 {
		_, observedFingerprint, err := inferRESTSchema(fullRecords)
		if err != nil {
			return RecordPage{}, err
		}
		if err := validateExpectedSchema(view, observedFingerprint); err != nil {
			return RecordPage{}, err
		}
		schemaFingerprint = observedFingerprint
	}
	records, err := projectRESTRecords(fullRecords, binding.SelectedFields)
	if err != nil {
		return RecordPage{}, err
	}
	if err := validateRESTStableRecords(records, binding.KeyFields[0]); err != nil {
		return RecordPage{}, err
	}

	next, position, completeness, err := restNextPosition(definition, response, request.After)
	if err != nil {
		return RecordPage{}, err
	}
	fingerprint, err := BindingFingerprint(view, binding)
	if err != nil {
		return RecordPage{}, err
	}
	return RecordPage{
		Records:    records,
		NextCursor: next,
		Receipt: s.receipt(view, binding, OperationPage, int64(len(records)), response.bodyBytes, completeness,
			fingerprint, schemaFingerprint, position, restRetryIdentity(view, binding, OperationPage, request.After, nil)),
	}, nil
}

func (s *RESTJSONSession) Lookup(ctx context.Context, view View, binding Binding, request LookupRequest) (LookupResult, error) {
	definition, limits, err := s.validateBoundOperation(view, binding, OperationLookup)
	if err != nil {
		return LookupResult{}, err
	}
	if definition.Lookup == nil {
		return LookupResult{}, ErrCapabilityUnavailable
	}
	if len(binding.KeyFields) != 1 {
		return LookupResult{}, fmt.Errorf("%w: REST/JSON lookup requires one stable key", ErrCapabilityUnavailable)
	}
	if len(request.Values) == 0 || len(request.Values) > limits.LookupValues {
		return LookupResult{}, ErrLimitExceeded
	}
	seen := make(map[string]struct{}, len(request.Values))
	var kind ScalarKind
	for index, value := range request.Values {
		if err := value.ValidateInput(); err != nil {
			return LookupResult{}, err
		}
		if index == 0 {
			kind = value.Kind
		} else if value.Kind != kind {
			return LookupResult{}, fmt.Errorf("%w: lookup values must use one scalar type", ErrDefinitionInvalid)
		}
		identity := scalarIdentity(value)
		if _, exists := seen[identity]; exists {
			return LookupResult{}, fmt.Errorf("%w: duplicate lookup value", ErrDefinitionInvalid)
		}
		seen[identity] = struct{}{}
	}
	if expected, ok := restExpectedFieldKind(view.NativeSchema, binding.KeyFields[0]); ok && expected != kind {
		return LookupResult{}, fmt.Errorf("%w: lookup value type does not match the inspected key", ErrDefinitionInvalid)
	}

	query := cloneRESTQuery(definition.FixedQuery)
	for _, value := range request.Values {
		query.Add(definition.Lookup.QueryParam, value.Text)
	}
	operationCtx, cancel := context.WithTimeout(ctx, limits.Timeout)
	defer cancel()
	response, err := s.getJSON(operationCtx, definition.Lookup.Path, query, "", limits.ResponseBytes)
	if err != nil {
		return LookupResult{}, err
	}
	fullRecords, err := extractRESTRecords(response.root, definition.RecordsPointer, len(request.Values))
	if err != nil {
		return LookupResult{}, err
	}
	schemaFingerprint := view.SchemaFingerprint
	if len(fullRecords) > 0 {
		_, observedFingerprint, err := inferRESTSchema(fullRecords)
		if err != nil {
			return LookupResult{}, err
		}
		if err := validateExpectedSchema(view, observedFingerprint); err != nil {
			return LookupResult{}, err
		}
		schemaFingerprint = observedFingerprint
	}
	records, err := projectRESTRecords(fullRecords, binding.SelectedFields)
	if err != nil {
		return LookupResult{}, err
	}
	if err := validateRESTLookupRecords(records, binding.KeyFields[0], seen); err != nil {
		return LookupResult{}, err
	}
	fingerprint, err := BindingFingerprint(view, binding)
	if err != nil {
		return LookupResult{}, err
	}
	return LookupResult{
		Records: records,
		Receipt: s.receipt(view, binding, OperationLookup, int64(len(records)), response.bodyBytes, CompletenessComplete,
			fingerprint, schemaFingerprint, nil, restRetryIdentity(view, binding, OperationLookup, nil, request.Values)),
	}, nil
}

func (s *RESTJSONSession) validateBoundOperation(view View, binding Binding, operation Operation) (RESTJSONViewDefinition, ResourceLimits, error) {
	if err := s.ready(view); err != nil {
		return RESTJSONViewDefinition{}, ResourceLimits{}, err
	}
	if err := binding.Validate(view); err != nil {
		return RESTJSONViewDefinition{}, ResourceLimits{}, err
	}
	if !binding.Allows(operation) {
		return RESTJSONViewDefinition{}, ResourceLimits{}, ErrCapabilityUnavailable
	}
	definition, err := decodeRESTJSONView(view)
	if err != nil {
		return RESTJSONViewDefinition{}, ResourceLimits{}, err
	}
	limits, err := binding.NormalizedLimits()
	if err != nil {
		return RESTJSONViewDefinition{}, ResourceLimits{}, err
	}
	return definition, limits, nil
}

func (s *RESTJSONSession) getJSON(ctx context.Context, path string, query url.Values, ifNoneMatch string, maximum int64) (restJSONResponse, error) {
	if err := ctx.Err(); err != nil {
		return restJSONResponse{}, err
	}
	if maximum <= 0 || maximum > HardMaxResponseBytes {
		return restJSONResponse{}, ErrLimitExceeded
	}
	endpoint := *s.baseURL
	endpoint.Path = strings.TrimRight(s.baseURL.Path, "/") + path
	endpoint.RawPath = ""
	endpoint.RawQuery = query.Encode()
	endpoint.Fragment = ""
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return restJSONResponse{}, ErrDefinitionInvalid
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Cache-Control", "no-store")
	if ifNoneMatch != "" {
		if len(ifNoneMatch) > hardMaxLookupScalarBytes || containsControl(ifNoneMatch) {
			return restJSONResponse{}, ErrDefinitionInvalid
		}
		request.Header.Set("If-None-Match", ifNoneMatch)
	}
	if err := s.authenticate(request); err != nil {
		return restJSONResponse{}, err
	}
	response, err := s.client.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return restJSONResponse{}, ctxErr
		}
		return restJSONResponse{}, ErrConnection
	}
	defer response.Body.Close()
	if response.Request == nil || !restSameOrigin(s.baseURL, response.Request.URL) {
		return restJSONResponse{}, ErrConnection
	}
	if response.StatusCode == http.StatusNotModified {
		if ifNoneMatch == "" {
			return restJSONResponse{}, ErrExecution
		}
		return restJSONResponse{notModified: true, etag: ifNoneMatch}, nil
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return restJSONResponse{}, restHTTPStatusError(response.StatusCode)
	}
	etag := strings.TrimSpace(response.Header.Get("ETag"))
	if len(etag) > hardMaxLookupScalarBytes || containsControl(etag) {
		return restJSONResponse{}, ErrExecution
	}
	if response.StatusCode == http.StatusNoContent {
		return restJSONResponse{root: []any{}, etag: etag}, nil
	}
	if !restContentTypeJSON(response) {
		return restJSONResponse{}, ErrExecution
	}
	if response.ContentLength > maximum {
		return restJSONResponse{}, ErrLimitExceeded
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, maximum+1))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return restJSONResponse{}, ctxErr
		}
		return restJSONResponse{}, ErrConnection
	}
	if int64(len(payload)) > maximum {
		return restJSONResponse{}, ErrLimitExceeded
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return restJSONResponse{}, ErrExecution
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return restJSONResponse{}, ErrExecution
	}
	return restJSONResponse{root: root, bodyBytes: int64(len(payload)), etag: etag}, nil
}

func restHTTPStatusError(status int) error {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ErrCredentials
	case status == http.StatusRequestTimeout || status == http.StatusTooEarly || status == http.StatusTooManyRequests || status >= 500:
		return ErrConnection
	default:
		return ErrExecution
	}
}

func extractRESTRecords(root any, pointer string, maximum int) ([]map[string]any, error) {
	if maximum < 0 || maximum > HardMaxPageRows {
		return nil, ErrLimitExceeded
	}
	if root == nil {
		return []map[string]any{}, nil
	}
	target, found, err := resolveJSONPointer(root, pointer)
	if err != nil || !found {
		return nil, ErrExecution
	}
	values, ok := target.([]any)
	if !ok {
		return nil, fmt.Errorf("%w: REST records pointer must resolve to an array", ErrExecution)
	}
	if len(values) > maximum {
		return nil, ErrLimitExceeded
	}
	records := make([]map[string]any, 0, len(values))
	for _, value := range values {
		record, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%w: REST record must be a JSON object", ErrExecution)
		}
		records = append(records, record)
	}
	return records, nil
}

func resolveJSONPointer(root any, pointer string) (any, bool, error) {
	if err := validateJSONPointer(pointer); err != nil {
		return nil, false, err
	}
	if pointer == "" {
		return root, true, nil
	}
	current := root
	for _, raw := range strings.Split(pointer[1:], "/") {
		token := strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
		switch typed := current.(type) {
		case map[string]any:
			value, exists := typed[token]
			if !exists {
				return nil, false, nil
			}
			current = value
		case []any:
			index, err := strconv.Atoi(token)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, false, nil
			}
			current = typed[index]
		default:
			return nil, false, nil
		}
	}
	return current, true, nil
}

func inferRESTSchema(records []map[string]any) ([]NativeField, string, error) {
	if len(records) == 0 {
		return nil, "", ErrExecution
	}
	types := map[string]string{}
	for _, record := range records {
		for name, value := range record {
			if !ValidFieldName(name) {
				return nil, "", ErrDefinitionInvalid
			}
			native := restNativeType(value)
			if current, exists := types[name]; exists && current != native && native != "json:null" {
				if current == "json:null" {
					types[name] = native
				} else {
					types[name] = "json:mixed"
				}
			} else if !exists {
				types[name] = native
			}
		}
	}
	names := make([]string, 0, len(types))
	for name := range types {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 || len(names) > HardMaxSchemaFields {
		return nil, "", ErrLimitExceeded
	}
	fields := make([]NativeField, 0, len(names))
	for _, name := range names {
		fields = append(fields, NativeField{Name: name, NativeType: types[name], Nullable: true})
	}
	fingerprint, err := nativeSchemaFingerprint(fields)
	if err != nil {
		return nil, "", err
	}
	return fields, fingerprint, nil
}

func restNativeType(value any) string {
	switch value.(type) {
	case nil:
		return "json:null"
	case string:
		return "json:string"
	case json.Number:
		return "json:number"
	case bool:
		return "json:boolean"
	case []any:
		return "json:array"
	case map[string]any:
		return "json:object"
	default:
		return "json:unknown"
	}
}

func projectRESTRecords(records []map[string]any, selected []string) ([]Record, error) {
	result := make([]Record, 0, len(records))
	for _, source := range records {
		record := make(Record, len(selected))
		for _, field := range selected {
			value, exists := source[field]
			if !exists {
				record[field] = Scalar{Kind: ScalarNull}
				continue
			}
			scalar, err := restScalar(value)
			if err != nil {
				return nil, err
			}
			record[field] = scalar
		}
		result = append(result, record)
	}
	return result, nil
}

func restScalar(value any) (Scalar, error) {
	switch typed := value.(type) {
	case nil:
		return Scalar{Kind: ScalarNull}, nil
	case string:
		if len(typed) > hardMaxRecordScalarBytes || strings.IndexByte(typed, 0) >= 0 {
			return Scalar{}, ErrLimitExceeded
		}
		return Scalar{Kind: ScalarString, Text: typed}, nil
	case json.Number:
		text := typed.String()
		if len(text) > hardMaxRecordScalarBytes || !validNumberText(text) {
			return Scalar{}, ErrUnsupportedValue
		}
		return Scalar{Kind: ScalarNumber, Text: text}, nil
	case bool:
		if typed {
			return Scalar{Kind: ScalarBool, Text: "true"}, nil
		}
		return Scalar{Kind: ScalarBool, Text: "false"}, nil
	default:
		return Scalar{}, ErrUnsupportedValue
	}
}

func validateRESTStableRecords(records []Record, key string) error {
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		value, exists := record[key]
		if !exists || value.Kind == ScalarNull {
			return ErrExecution
		}
		identity := scalarIdentity(value)
		if _, exists := seen[identity]; exists {
			return ErrExecution
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func validateRESTLookupRecords(records []Record, key string, requested map[string]struct{}) error {
	if err := validateRESTStableRecords(records, key); err != nil {
		return err
	}
	for _, record := range records {
		if _, exists := requested[scalarIdentity(record[key])]; !exists {
			return ErrExecution
		}
	}
	return nil
}

func restExpectedFieldKind(schema []NativeField, name string) (ScalarKind, bool) {
	for _, field := range schema {
		if field.Name != name {
			continue
		}
		switch field.NativeType {
		case "json:string":
			return ScalarString, true
		case "json:number":
			return ScalarNumber, true
		case "json:boolean":
			return ScalarBool, true
		default:
			return "", false
		}
	}
	return "", false
}

func restNextPosition(definition RESTJSONViewDefinition, response restJSONResponse, after *Scalar) (*Scalar, *CheckpointPosition, Completeness, error) {
	switch definition.Pagination.Mode {
	case RESTJSONPaginationNone:
		return nil, nil, CompletenessComplete, nil
	case RESTJSONPaginationCursor:
		value, found, err := resolveJSONPointer(response.root, definition.Pagination.NextCursorPointer)
		if err != nil {
			return nil, nil, "", err
		}
		if !found || value == nil {
			return nil, nil, CompletenessComplete, nil
		}
		next, err := restScalar(value)
		if err != nil || next.Kind == ScalarNull {
			return nil, nil, "", ErrExecution
		}
		if err := next.ValidateInput(); err != nil {
			return nil, nil, "", err
		}
		if after != nil && scalarIdentity(*after) == scalarIdentity(next) {
			return nil, nil, "", ErrExecution
		}
		copy := next
		return &copy, &CheckpointPosition{Kind: CheckpointCursor, Value: next.Text}, CompletenessPartial, nil
	case RESTJSONPaginationETag:
		if response.etag == "" {
			return nil, nil, CompletenessComplete, nil
		}
		if after != nil && after.Text == response.etag {
			return nil, nil, CompletenessComplete, nil
		}
		next := Scalar{Kind: ScalarString, Text: response.etag}
		return &next, &CheckpointPosition{Kind: CheckpointETag, Value: response.etag}, CompletenessPartial, nil
	default:
		return nil, nil, "", ErrDefinitionInvalid
	}
}

func cloneRESTQuery(input map[string]string) url.Values {
	values := make(url.Values, len(input))
	for key, value := range input {
		values.Set(key, value)
	}
	return values
}

func validateExpectedSchema(view View, observed string) error {
	if view.SchemaFingerprint != "" && observed != "" && view.SchemaFingerprint != observed {
		return ErrSchemaDrift
	}
	return nil
}

func restRetryIdentity(view View, binding Binding, operation Operation, after *Scalar, lookup []Scalar) string {
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s", view.ConnectionID, view.ID, view.Version, binding.ID, binding.Version, operation)
	if after != nil {
		_, _ = fmt.Fprintf(hash, "\x1f%s", scalarIdentity(*after))
	}
	for _, value := range lookup {
		_, _ = fmt.Fprintf(hash, "\x1f%s", scalarIdentity(value))
	}
	return "rest-json:" + hex.EncodeToString(hash.Sum(nil))
}

func (s *RESTJSONSession) receipt(view View, binding Binding, operation Operation, count, byteCount int64, completeness Completeness, definitionFingerprint, schemaFingerprint string, position *CheckpointPosition, retryIdentity string) OperationReceipt {
	receipt := OperationReceipt{
		SourceID:              s.connection.SourceID,
		ConnectionID:          s.connection.ID,
		ConnectionVersion:     s.connection.Version,
		AdapterKind:           s.connection.AdapterKind,
		AdapterVersion:        s.connection.AdapterVersion,
		ViewID:                view.ID,
		ViewVersion:           view.Version,
		DefinitionFingerprint: definitionFingerprint,
		SchemaFingerprint:     schemaFingerprint,
		Operation:             operation,
		ObservedAt:            s.now().UTC(),
		Count:                 count,
		Bytes:                 byteCount,
		Completeness:          completeness,
		Position:              position,
		RetryIdentity:         retryIdentity,
	}
	if binding.ID != "" {
		receipt.BindingID = binding.ID
		receipt.BindingVersion = binding.Version
	}
	return receipt
}

var _ SchemaReader = (*RESTJSONSession)(nil)
var _ PageReader = (*RESTJSONSession)(nil)
var _ LookupReader = (*RESTJSONSession)(nil)

func restIsContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

var _ = time.UTC
