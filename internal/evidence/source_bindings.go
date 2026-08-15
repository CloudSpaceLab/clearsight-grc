package evidence

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

const (
	maxFieldBindingReferences   = 4
	maxRequestBindingReferences = 20
	maxSourceOptionCount        = 50
)

// BindingReader is the narrow connected-source surface required by capture.
// Consumer records own references, selected values and receipts; the source
// catalog remains authoritative for connection, view and Binding execution.
type BindingReader interface {
	Binding(context.Context, string, string, int64) (sourceaccess.BindingRevision, error)
	PreviewBinding(context.Context, string, string, int64, sourceaccess.PageRequest) (sourceaccess.RecordPage, error)
	LookupBinding(context.Context, string, string, int64, sourceaccess.LookupRequest) (sourceaccess.LookupResult, error)
}

func (s *Service) ConfigureSourceBindings(reader BindingReader) {
	s.bindings = reader
}

func (s *Service) prepareRequestBindings(ctx context.Context, input CreateRequestInput) ([]Field, []RequestBindingReference, error) {
	fields := cloneFields(input.Fields)
	requestBindings := cloneRequestBindings(input.SourceBindings)
	hasReferences := len(requestBindings) > 0
	for index := range fields {
		fields[index].SourceResolutions = nil
		hasReferences = hasReferences || len(fields[index].Bindings) > 0
	}
	for index := range requestBindings {
		requestBindings[index].Resolution = nil
	}
	if !hasReferences {
		return fields, requestBindings, nil
	}
	if s.bindings == nil {
		return nil, nil, fmt.Errorf("connected-source binding resolution is unavailable")
	}
	if len(requestBindings) > maxRequestBindingReferences {
		return nil, nil, fmt.Errorf("request may reference at most %d evidence bindings", maxRequestBindingReferences)
	}

	cache := make(map[string]sourceaccess.BindingRevision)
	loadBinding := func(bindingID string, bindingVersion int64) (sourceaccess.BindingRevision, error) {
		bindingID = strings.TrimSpace(bindingID)
		if bindingID == "" || bindingVersion < 1 {
			return sourceaccess.BindingRevision{}, fmt.Errorf("binding_id and positive binding_version are required")
		}
		key := bindingID + ":" + strconv.FormatInt(bindingVersion, 10)
		if value, ok := cache[key]; ok {
			return value, nil
		}
		value, err := s.bindings.Binding(ctx, input.TenantID, bindingID, bindingVersion)
		if err != nil {
			return sourceaccess.BindingRevision{}, fmt.Errorf("resolve binding %s version %d: %w", bindingID, bindingVersion, err)
		}
		if value.BindingID != bindingID || value.Version != bindingVersion || value.TenantID != input.TenantID {
			return sourceaccess.BindingRevision{}, fmt.Errorf("resolved binding identity does not match the requested tenant/version")
		}
		if value.Status != sourceaccess.RevisionActive {
			return sourceaccess.BindingRevision{}, fmt.Errorf("binding %s version %d is not active", bindingID, bindingVersion)
		}
		cache[key] = value
		return value, nil
	}

	for fieldIndex := range fields {
		field := &fields[fieldIndex]
		if len(field.Bindings) > maxFieldBindingReferences {
			return nil, nil, fmt.Errorf("%s may reference at most %d source bindings", field.Label, maxFieldBindingReferences)
		}
		seenModes := make(map[BindingUseMode]struct{}, len(field.Bindings))
		revisions := make([]sourceaccess.BindingRevision, len(field.Bindings))
		for bindingIndex, reference := range field.Bindings {
			if _, exists := seenModes[reference.Mode]; exists {
				return nil, nil, fmt.Errorf("%s contains more than one %s binding", field.Label, reference.Mode)
			}
			seenModes[reference.Mode] = struct{}{}
			revision, err := loadBinding(reference.BindingID, reference.BindingVersion)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %w", field.Label, err)
			}
			if err := validateFieldBindingReference(*field, reference, revision); err != nil {
				return nil, nil, err
			}
			revisions[bindingIndex] = revision
		}

		// Resolve bounded option lists first so a source prefill can be checked
		// against the exact choices presented to the respondent.
		for bindingIndex, reference := range field.Bindings {
			if reference.Mode != BindingUseOptions {
				continue
			}
			resolution, options := s.resolveOptions(ctx, input.TenantID, reference, revisions[bindingIndex])
			field.SourceResolutions = append(field.SourceResolutions, resolution)
			if resolution.State == SourceResolutionCurrent {
				field.Options = options
			}
		}
		for bindingIndex, reference := range field.Bindings {
			switch reference.Mode {
			case BindingUsePrefill, BindingUseEvidence:
				lookup, ok := lookupValue(input, reference.LookupValue)
				resolution := sourceResolutionBase(reference.Mode, revisions[bindingIndex])
				if !ok {
					resolution.State = SourceResolutionNotFound
					resolution.FailureCode = "LOOKUP_VALUE_MISSING"
				} else {
					resolution = s.resolveLookup(ctx, input.TenantID, reference, revisions[bindingIndex], lookup)
				}
				if reference.Mode == BindingUsePrefill && resolution.State == SourceResolutionCurrent {
					value, found := resolvedScalar(resolution)
					if !found || s.validateAnswers(ctx, Request{TenantID: input.TenantID, Fields: []Field{*field}}, map[string]string{field.ID: value.Text}) != nil {
						resolution.State = SourceResolutionInvalid
						resolution.FailureCode = "PREFILL_VALUE_INVALID"
						resolution.Value = nil
					}
				}
				field.SourceResolutions = append(field.SourceResolutions, resolution)
			}
		}
	}

	seenRequestBindings := make(map[string]struct{}, len(requestBindings))
	for index := range requestBindings {
		reference := &requestBindings[index]
		key := strings.TrimSpace(reference.BindingID) + ":" + strconv.FormatInt(reference.BindingVersion, 10)
		if _, exists := seenRequestBindings[key]; exists {
			return nil, nil, fmt.Errorf("request contains duplicate evidence binding %s", key)
		}
		seenRequestBindings[key] = struct{}{}
		revision, err := loadBinding(reference.BindingID, reference.BindingVersion)
		if err != nil {
			return nil, nil, err
		}
		if !bindingAllows(revision, sourceaccess.OperationLookup) {
			return nil, nil, fmt.Errorf("binding %s version %d does not permit LOOKUP", reference.BindingID, reference.BindingVersion)
		}
		if err := validateLookupValueReference(&reference.LookupValue); err != nil {
			return nil, nil, fmt.Errorf("binding %s: %w", reference.BindingID, err)
		}
		resolution := sourceResolutionBase(BindingUseEvidence, revision)
		lookup, ok := lookupValue(input, &reference.LookupValue)
		if !ok {
			resolution.State = SourceResolutionNotFound
			resolution.FailureCode = "LOOKUP_VALUE_MISSING"
		} else {
			fieldReference := FieldBindingReference{
				BindingID:      reference.BindingID,
				BindingVersion: reference.BindingVersion,
				Mode:           BindingUseEvidence,
				LookupValue:    &reference.LookupValue,
			}
			resolution = s.resolveLookup(ctx, input.TenantID, fieldReference, revision, lookup)
		}
		reference.Resolution = &resolution
	}
	return fields, requestBindings, nil
}

func validateFieldBindingReference(field Field, reference FieldBindingReference, revision sourceaccess.BindingRevision) error {
	fieldType := normalizedFieldTypeName(field.Type)
	if fieldType == "photo" || fieldType == "file" || fieldType == "signature" {
		return fmt.Errorf("%s cannot attach a source binding to a file response", field.Label)
	}
	switch reference.Mode {
	case BindingUsePrefill:
		if !bindingAllows(revision, sourceaccess.OperationLookup) {
			return fmt.Errorf("%s PREFILL binding does not permit LOOKUP", field.Label)
		}
		if err := validateSelectedValueField(reference.ValueField, revision); err != nil {
			return fmt.Errorf("%s PREFILL binding: %w", field.Label, err)
		}
		if err := validateLookupValueReference(reference.LookupValue); err != nil {
			return fmt.Errorf("%s PREFILL binding: %w", field.Label, err)
		}
	case BindingUseOptions:
		if fieldType != "single_select" {
			return fmt.Errorf("%s OPTIONS binding requires a single_select field", field.Label)
		}
		if !bindingAllows(revision, sourceaccess.OperationPage) {
			return fmt.Errorf("%s OPTIONS binding does not permit PAGE", field.Label)
		}
		if reference.LookupValue != nil {
			return fmt.Errorf("%s OPTIONS binding must not define lookup_value", field.Label)
		}
		if err := validateSelectedValueField(reference.ValueField, revision); err != nil {
			return fmt.Errorf("%s OPTIONS binding: %w", field.Label, err)
		}
	case BindingUseValidate:
		if !bindingAllows(revision, sourceaccess.OperationLookup) {
			return fmt.Errorf("%s VALIDATE binding does not permit LOOKUP", field.Label)
		}
		if reference.LookupValue != nil {
			return fmt.Errorf("%s VALIDATE binding uses the respondent answer and must not define lookup_value", field.Label)
		}
		if err := validateSelectedValueField(reference.ValueField, revision); err != nil {
			return fmt.Errorf("%s VALIDATE binding: %w", field.Label, err)
		}
	case BindingUseEvidence:
		if !bindingAllows(revision, sourceaccess.OperationLookup) {
			return fmt.Errorf("%s EVIDENCE binding does not permit LOOKUP", field.Label)
		}
		if err := validateLookupValueReference(reference.LookupValue); err != nil {
			return fmt.Errorf("%s EVIDENCE binding: %w", field.Label, err)
		}
		if strings.TrimSpace(reference.ValueField) != "" {
			if err := validateSelectedValueField(reference.ValueField, revision); err != nil {
				return fmt.Errorf("%s EVIDENCE binding: %w", field.Label, err)
			}
		}
	default:
		return fmt.Errorf("%s contains unsupported binding mode %q", field.Label, reference.Mode)
	}
	return nil
}

func validateSelectedValueField(valueField string, revision sourceaccess.BindingRevision) error {
	valueField = strings.TrimSpace(valueField)
	if valueField == "" {
		return fmt.Errorf("value_field is required")
	}
	for _, field := range revision.SelectedFields {
		if field == valueField {
			return nil
		}
	}
	return fmt.Errorf("value_field %q is not selected by the binding", valueField)
}

func validateLookupValueReference(reference *LookupValueReference) error {
	if reference == nil {
		return fmt.Errorf("lookup_value is required")
	}
	switch reference.Source {
	case LookupValueSubjectID:
		if strings.TrimSpace(reference.Key) != "" {
			return fmt.Errorf("SUBJECT_ID lookup must not define key")
		}
	case LookupValueKnownFact:
		if strings.TrimSpace(reference.Key) == "" {
			return fmt.Errorf("KNOWN_FACT lookup requires key")
		}
	default:
		return fmt.Errorf("lookup value source is invalid")
	}
	return nil
}

func lookupValue(input CreateRequestInput, reference *LookupValueReference) (sourceaccess.Scalar, bool) {
	if reference == nil {
		return sourceaccess.Scalar{}, false
	}
	var value string
	switch reference.Source {
	case LookupValueSubjectID:
		value = input.SubjectID
	case LookupValueKnownFact:
		value = input.KnownFacts[strings.TrimSpace(reference.Key)]
	default:
		return sourceaccess.Scalar{}, false
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return sourceaccess.Scalar{}, false
	}
	return sourceaccess.StringValue(value), true
}

func (s *Service) resolveOptions(ctx context.Context, tenantID string, reference FieldBindingReference, revision sourceaccess.BindingRevision) (SourceResolution, []string) {
	resolution := sourceResolutionBase(reference.Mode, revision)
	page, err := s.bindings.PreviewBinding(ctx, tenantID, reference.BindingID, reference.BindingVersion, sourceaccess.PageRequest{Limit: maxSourceOptionCount})
	if err != nil {
		return sourceFailureResolution(resolution, err), nil
	}
	resolution.Receipt = cloneOperationReceipt(&page.Receipt)
	resolution.State = resolutionState(revision, page.Receipt, s.now().UTC())
	if page.NextCursor != nil && resolution.State == SourceResolutionCurrent {
		resolution.State = SourceResolutionPartial
		resolution.FailureCode = "OPTION_SET_EXCEEDS_BOUND"
	}
	if resolution.State != SourceResolutionCurrent {
		return resolution, nil
	}
	seen := make(map[string]struct{}, len(page.Records))
	options := make([]string, 0, len(page.Records))
	for _, record := range page.Records {
		value, ok := record[reference.ValueField]
		if !ok || value.Kind == sourceaccess.ScalarNull {
			continue
		}
		option := strings.TrimSpace(value.Text)
		if option == "" {
			continue
		}
		if _, exists := seen[option]; exists {
			continue
		}
		seen[option] = struct{}{}
		options = append(options, option)
	}
	if len(options) < 2 || len(options) > maxSourceOptionCount {
		resolution.State = SourceResolutionInvalid
		resolution.FailureCode = "INVALID_OPTION_SET"
		return resolution, nil
	}
	return resolution, options
}

func (s *Service) resolveLookup(ctx context.Context, tenantID string, reference FieldBindingReference, revision sourceaccess.BindingRevision, lookup sourceaccess.Scalar) SourceResolution {
	resolution := sourceResolutionBase(reference.Mode, revision)
	result, err := s.bindings.LookupBinding(ctx, tenantID, reference.BindingID, reference.BindingVersion, sourceaccess.LookupRequest{Values: []sourceaccess.Scalar{lookup}})
	if err != nil {
		return sourceFailureResolution(resolution, err)
	}
	resolution.Receipt = cloneOperationReceipt(&result.Receipt)
	resolution.State = resolutionState(revision, result.Receipt, s.now().UTC())
	if reference.Mode == BindingUseEvidence {
		resolution.Records = cloneRecords(result.Records)
	}
	if resolution.State != SourceResolutionCurrent {
		return resolution
	}
	if len(result.Records) == 0 {
		resolution.State = SourceResolutionNotFound
		return resolution
	}
	if reference.Mode != BindingUsePrefill {
		return resolution
	}
	if len(result.Records) != 1 {
		resolution.State = SourceResolutionAmbiguous
		resolution.FailureCode = "MULTIPLE_SOURCE_VALUES"
		return resolution
	}
	value, ok := selectedScalar(result.Records[0], reference.ValueField)
	if !ok {
		resolution.State = SourceResolutionInvalid
		resolution.FailureCode = "VALUE_FIELD_MISSING"
		return resolution
	}
	resolution.Value = &value
	return resolution
}

func sourceResolutionBase(mode BindingUseMode, revision sourceaccess.BindingRevision) SourceResolution {
	return SourceResolution{
		Mode:           mode,
		BindingID:      revision.BindingID,
		BindingVersion: revision.Version,
		BindingName:    revision.Name,
		SourceID:       revision.SourceID,
		State:          SourceResolutionUnavailable,
	}
}

func sourceFailureResolution(resolution SourceResolution, err error) SourceResolution {
	resolution.State = SourceResolutionUnavailable
	resolution.FailureCode = "EXECUTION_FAILED"
	switch {
	case errors.Is(err, sourceaccess.ErrSchemaDrift):
		resolution.State = SourceResolutionSchemaDrift
		resolution.FailureCode = "SCHEMA_DRIFT"
	case errors.Is(err, sourceaccess.ErrCapabilityUnavailable):
		resolution.FailureCode = "CAPABILITY_UNAVAILABLE"
	case errors.Is(err, sourceaccess.ErrLimitExceeded):
		resolution.FailureCode = "LIMIT_EXCEEDED"
	case errors.Is(err, sourceaccess.ErrCredentials):
		resolution.FailureCode = "CREDENTIALS_UNAVAILABLE"
	case errors.Is(err, sourceaccess.ErrConnection):
		resolution.FailureCode = "SOURCE_UNAVAILABLE"
	case errors.Is(err, sourceaccess.ErrCatalogNotFound), errors.Is(err, sourceaccess.ErrCatalogInvalid):
		resolution.FailureCode = "BINDING_UNAVAILABLE"
	case errors.Is(err, context.DeadlineExceeded):
		resolution.FailureCode = "TIMEOUT"
	case errors.Is(err, context.Canceled):
		resolution.FailureCode = "CANCELLED"
	}
	return resolution
}

func resolutionState(revision sourceaccess.BindingRevision, receipt sourceaccess.OperationReceipt, now time.Time) SourceResolutionState {
	if receipt.ObservedAt.IsZero() {
		return SourceResolutionUnavailable
	}
	switch receipt.Completeness {
	case sourceaccess.CompletenessPartial:
		return SourceResolutionPartial
	case sourceaccess.CompletenessComplete:
	default:
		return SourceResolutionUnavailable
	}
	if revision.RequiredFreshnessMinutes > 0 {
		freshUntil := receipt.ObservedAt.Add(time.Duration(revision.RequiredFreshnessMinutes) * time.Minute)
		if !now.Before(freshUntil) {
			return SourceResolutionStale
		}
	}
	return SourceResolutionCurrent
}

func bindingAllows(revision sourceaccess.BindingRevision, operation sourceaccess.Operation) bool {
	for _, allowed := range revision.Operations {
		if allowed == operation {
			return true
		}
	}
	return false
}

func selectedScalar(record sourceaccess.Record, field string) (sourceaccess.Scalar, bool) {
	value, ok := record[strings.TrimSpace(field)]
	return value, ok && value.Kind != sourceaccess.ScalarNull && strings.TrimSpace(value.Text) != ""
}

func resolvedScalar(resolution SourceResolution) (sourceaccess.Scalar, bool) {
	if resolution.State != SourceResolutionCurrent || resolution.Value == nil {
		return sourceaccess.Scalar{}, false
	}
	value := *resolution.Value
	return value, value.Kind != sourceaccess.ScalarNull && strings.TrimSpace(value.Text) != ""
}

func (s *Service) deriveAnswerProvenance(ctx context.Context, request Request, answers map[string]string) map[string]AnswerProvenance {
	result := make(map[string]AnswerProvenance, len(answers))
	for _, field := range request.Fields {
		answer, exists := answers[field.ID]
		if !exists || strings.TrimSpace(answer) == "" {
			continue
		}
		provenance := AnswerProvenance{Origin: AnswerRespondentEntered}
		for _, reference := range field.Bindings {
			if reference.Mode != BindingUsePrefill {
				continue
			}
			resolution, ok := matchingResolution(field.SourceResolutions, reference)
			if !ok {
				continue
			}
			sourceValue, ok := resolvedScalar(resolution)
			if !ok {
				continue
			}
			provenance.BindingID = reference.BindingID
			provenance.BindingVersion = reference.BindingVersion
			valueCopy := sourceValue
			provenance.SourceValue = &valueCopy
			provenance.SourceReceipt = cloneOperationReceipt(resolution.Receipt)
			if strings.TrimSpace(answer) == strings.TrimSpace(sourceValue.Text) {
				provenance.Origin = AnswerSourcePrefilled
			} else {
				provenance.Origin = AnswerRespondentCorrected
			}
			break
		}
		for _, reference := range field.Bindings {
			if reference.Mode != BindingUseValidate {
				continue
			}
			provenance.Validations = append(provenance.Validations, s.validateSourceAnswer(ctx, request.TenantID, reference, answer))
		}
		result[field.ID] = provenance
	}
	return result
}

func (s *Service) validateSourceAnswer(ctx context.Context, tenantID string, reference FieldBindingReference, answer string) SourceResolution {
	fallback := SourceResolution{
		Mode:           BindingUseValidate,
		BindingID:      strings.TrimSpace(reference.BindingID),
		BindingVersion: reference.BindingVersion,
		State:          SourceResolutionUnavailable,
		FailureCode:    "BINDING_UNAVAILABLE",
	}
	if s.bindings == nil {
		return fallback
	}
	revision, err := s.bindings.Binding(ctx, tenantID, reference.BindingID, reference.BindingVersion)
	if err != nil {
		return sourceFailureResolution(fallback, err)
	}
	if revision.BindingID != strings.TrimSpace(reference.BindingID) || revision.Version != reference.BindingVersion || revision.TenantID != tenantID {
		fallback.State = SourceResolutionInvalid
		fallback.FailureCode = "BINDING_IDENTITY_MISMATCH"
		return fallback
	}
	resolution := sourceResolutionBase(BindingUseValidate, revision)
	if revision.Status != sourceaccess.RevisionActive {
		resolution.FailureCode = "BINDING_INACTIVE"
		return resolution
	}
	if !bindingAllows(revision, sourceaccess.OperationLookup) {
		resolution.FailureCode = "CAPABILITY_UNAVAILABLE"
		return resolution
	}
	if err := validateSelectedValueField(reference.ValueField, revision); err != nil {
		resolution.State = SourceResolutionInvalid
		resolution.FailureCode = "VALUE_FIELD_INVALID"
		return resolution
	}
	answer = strings.TrimSpace(answer)
	result, err := s.bindings.LookupBinding(ctx, tenantID, reference.BindingID, reference.BindingVersion, sourceaccess.LookupRequest{Values: []sourceaccess.Scalar{sourceaccess.StringValue(answer)}})
	if err != nil {
		return sourceFailureResolution(resolution, err)
	}
	resolution.Receipt = cloneOperationReceipt(&result.Receipt)
	resolution.State = resolutionState(revision, result.Receipt, s.now().UTC())
	if resolution.State != SourceResolutionCurrent {
		return resolution
	}
	switch len(result.Records) {
	case 0:
		resolution.State = SourceResolutionNotFound
		return resolution
	case 1:
	default:
		resolution.State = SourceResolutionAmbiguous
		resolution.FailureCode = "MULTIPLE_SOURCE_VALUES"
		return resolution
	}
	value, ok := selectedScalar(result.Records[0], reference.ValueField)
	if !ok {
		resolution.State = SourceResolutionInvalid
		resolution.FailureCode = "VALUE_FIELD_MISSING"
		return resolution
	}
	resolution.Value = &value
	if strings.TrimSpace(value.Text) != answer {
		resolution.State = SourceResolutionNotFound
		resolution.FailureCode = "VALUE_MISMATCH"
	}
	return resolution
}

func matchingResolution(values []SourceResolution, reference FieldBindingReference) (SourceResolution, bool) {
	for _, value := range values {
		if value.Mode == reference.Mode && value.BindingID == reference.BindingID && value.BindingVersion == reference.BindingVersion {
			return value, true
		}
	}
	return SourceResolution{}, false
}

func cloneRequestBindings(input []RequestBindingReference) []RequestBindingReference {
	out := append([]RequestBindingReference(nil), input...)
	for index := range out {
		if input[index].Resolution != nil {
			resolution := cloneSourceResolution(*input[index].Resolution)
			out[index].Resolution = &resolution
		}
	}
	return out
}

func cloneSourceResolutions(input []SourceResolution) []SourceResolution {
	out := make([]SourceResolution, len(input))
	for index := range input {
		out[index] = cloneSourceResolution(input[index])
	}
	return out
}

func cloneSourceResolution(input SourceResolution) SourceResolution {
	if input.Value != nil {
		value := *input.Value
		input.Value = &value
	}
	input.Records = cloneRecords(input.Records)
	input.Receipt = cloneOperationReceipt(input.Receipt)
	return input
}

func cloneRecords(input []sourceaccess.Record) []sourceaccess.Record {
	out := make([]sourceaccess.Record, len(input))
	for index, record := range input {
		copyRecord := make(sourceaccess.Record, len(record))
		for key, value := range record {
			copyRecord[key] = value
		}
		out[index] = copyRecord
	}
	return out
}

func cloneOperationReceipt(input *sourceaccess.OperationReceipt) *sourceaccess.OperationReceipt {
	if input == nil {
		return nil
	}
	copyValue := *input
	if input.Position != nil {
		position := *input.Position
		copyValue.Position = &position
	}
	return &copyValue
}

func cloneAnswerProvenance(input map[string]AnswerProvenance) map[string]AnswerProvenance {
	out := make(map[string]AnswerProvenance, len(input))
	for key, value := range input {
		if value.SourceValue != nil {
			sourceValue := *value.SourceValue
			value.SourceValue = &sourceValue
		}
		value.SourceReceipt = cloneOperationReceipt(value.SourceReceipt)
		value.Validations = cloneSourceResolutions(value.Validations)
		out[key] = value
	}
	return out
}

// RespondentRequest returns the minimum request view needed to answer a form.
// It deliberately hides validation rules, evidence searches, lookup selectors,
// source schema field names, source rows and connector failure details.
func RespondentRequest(request Request) Request {
	request.Fields = cloneFields(request.Fields)
	request.SourceBindings = nil
	for fieldIndex := range request.Fields {
		field := &request.Fields[fieldIndex]
		visibleResolutions := make([]SourceResolution, 0, 2)
		visibleKeys := make(map[string]struct{}, 2)
		for _, resolution := range field.SourceResolutions {
			if resolution.State != SourceResolutionCurrent || (resolution.Mode != BindingUsePrefill && resolution.Mode != BindingUseOptions) {
				continue
			}
			visible := cloneSourceResolution(resolution)
			visible.Records = nil
			visible.FailureCode = ""
			if visible.Mode != BindingUsePrefill {
				visible.Value = nil
			}
			visibleResolutions = append(visibleResolutions, visible)
			visibleKeys[bindingResolutionKey(visible.Mode, visible.BindingID, visible.BindingVersion)] = struct{}{}
		}
		visibleBindings := make([]FieldBindingReference, 0, len(visibleResolutions))
		for _, reference := range field.Bindings {
			if _, ok := visibleKeys[bindingResolutionKey(reference.Mode, reference.BindingID, reference.BindingVersion)]; !ok {
				continue
			}
			reference.ValueField = ""
			reference.LookupValue = nil
			visibleBindings = append(visibleBindings, reference)
		}
		field.Bindings = visibleBindings
		field.SourceResolutions = visibleResolutions
	}
	return request
}

func respondentRequests(values []Request) []Request {
	out := make([]Request, len(values))
	for index := range values {
		out[index] = RespondentRequest(values[index])
	}
	return out
}

func manageableRequestViews(values []Request, principal string) []Request {
	out := make([]Request, len(values))
	for index := range values {
		if strings.TrimSpace(values[index].CreatedBy) != strings.TrimSpace(principal) {
			out[index] = RespondentRequest(values[index])
			continue
		}
		out[index] = values[index]
	}
	return out
}

func bindingResolutionKey(mode BindingUseMode, bindingID string, bindingVersion int64) string {
	return string(mode) + ":" + strings.TrimSpace(bindingID) + ":" + strconv.FormatInt(bindingVersion, 10)
}

func normalizedFieldTypeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "short_text" {
		return "text"
	}
	return value
}
