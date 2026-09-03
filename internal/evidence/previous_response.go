package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const maxPreviousResponseCount = 50
const maxPreviousResponsesBytes = 128 << 10

type PreviousResponseValue struct {
	Value                string    `json:"value"`
	PreviousRequestID    string    `json:"previous_request_id"`
	PreviousSubmissionID string    `json:"previous_submission_id"`
	PreviousSubmittedAt  time.Time `json:"previous_submitted_at"`
}

// BuildPreviousResponsePrefill carries forward only compatible scalar answers.
// Artifact identifiers remain evidence on the predecessor submission and are
// never presented as files supplied for the successor request.
func BuildPreviousResponsePrefill(previous Request, submission Submission, next []Field) map[string]PreviousResponseValue {
	result := make(map[string]PreviousResponseValue)
	if strings.TrimSpace(previous.ID) == "" || submission.RequestID != previous.ID || strings.TrimSpace(submission.ID) == "" || submission.SubmittedAt.IsZero() {
		return result
	}
	previousFields := make(map[string]Field, len(previous.Fields))
	for _, field := range previous.Fields {
		previousFields[field.ID] = field
	}
	for _, field := range next {
		priorField, exists := previousFields[field.ID]
		if !exists || normalizedFieldTypeName(priorField.Type) != normalizedFieldTypeName(field.Type) {
			continue
		}
		answer, answered := submission.Answers[field.ID]
		value, scalar := answer.ScalarText()
		if !answered || !scalar || value == "" || !reusableScalarAnswer(field, value) {
			continue
		}
		result[field.ID] = PreviousResponseValue{
			Value: value, PreviousRequestID: previous.ID, PreviousSubmissionID: submission.ID, PreviousSubmittedAt: submission.SubmittedAt.UTC(),
		}
	}
	return result
}

func normalizePreviousResponses(fields []Field, values map[string]PreviousResponseValue) map[string]PreviousResponseValue {
	if len(values) == 0 {
		return nil
	}
	fieldsByID := make(map[string]Field, len(fields))
	for _, field := range fields {
		fieldsByID[field.ID] = field
	}
	result := make(map[string]PreviousResponseValue, min(len(values), maxPreviousResponseCount))
	for fieldID, previous := range values {
		if len(result) == maxPreviousResponseCount {
			break
		}
		field, exists := fieldsByID[fieldID]
		if !exists || hasCurrentSourcePrefill(field) {
			continue
		}
		previous.Value = strings.TrimSpace(previous.Value)
		previous.PreviousRequestID = strings.TrimSpace(previous.PreviousRequestID)
		previous.PreviousSubmissionID = strings.TrimSpace(previous.PreviousSubmissionID)
		if previous.Value == "" || previous.PreviousRequestID == "" || previous.PreviousSubmissionID == "" || previous.PreviousSubmittedAt.IsZero() || !reusableScalarAnswer(field, previous.Value) {
			continue
		}
		previous.PreviousSubmittedAt = previous.PreviousSubmittedAt.UTC()
		result[fieldID] = previous
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func reusableScalarAnswer(field Field, value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	fieldType := normalizedFieldTypeName(field.Type)
	switch fieldType {
	case "file", "photo", "signature", "vendor_document", "multi_select":
		return false
	case "single_select", "yes_no":
		return containsOption(field.Options, value)
	case "date":
		_, err := time.Parse("2006-01-02", value)
		return err == nil
	case "integer", "decimal", "percentage", "currency", "number":
		number, err := strconv.ParseFloat(value, 64)
		return err == nil && !math.IsNaN(number) && !math.IsInf(number, 0)
	case "long_text":
		return len(value) <= maxLongAnswerBytes
	default:
		return len(value) <= maxShortAnswerBytes
	}
}

func hasCurrentSourcePrefill(field Field) bool {
	for _, reference := range field.Bindings {
		if reference.Mode != BindingUsePrefill {
			continue
		}
		resolution, ok := matchingResolution(field.SourceResolutions, reference)
		if _, usable := resolvedScalar(resolution); ok && usable {
			return true
		}
	}
	return false
}

func clonePreviousResponses(input map[string]PreviousResponseValue) map[string]PreviousResponseValue {
	if input == nil {
		return nil
	}
	out := make(map[string]PreviousResponseValue, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func validatePreviousResponsesBound(values map[string]PreviousResponseValue) error {
	if len(values) > maxPreviousResponseCount {
		return fmt.Errorf("previous responses may contain at most %d fields", maxPreviousResponseCount)
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("encode previous responses: %w", err)
	}
	if len(encoded) > maxPreviousResponsesBytes {
		return fmt.Errorf("previous responses exceed the %d-byte limit", maxPreviousResponsesBytes)
	}
	return nil
}

func validatePreviousResponseLineage(ctx context.Context, repo Repository, request Request) error {
	if len(request.PreviousResponses) == 0 {
		return nil
	}
	predecessorID := strings.TrimSpace(request.PredecessorRequestID)
	if predecessorID == "" {
		return fmt.Errorf("%w: previous responses require a predecessor request", ErrVersionConflict)
	}
	reader, ok := repo.(SubmissionReader)
	if !ok {
		return fmt.Errorf("%w: previous response verification is unavailable", ErrVersionConflict)
	}
	var submissionID string
	for _, previous := range request.PreviousResponses {
		if previous.PreviousRequestID != predecessorID {
			return fmt.Errorf("%w: previous response request does not match the predecessor", ErrVersionConflict)
		}
		if submissionID == "" {
			submissionID = previous.PreviousSubmissionID
		} else if previous.PreviousSubmissionID != submissionID {
			return fmt.Errorf("%w: previous responses must come from one submission", ErrVersionConflict)
		}
	}
	submission, err := reader.GetSubmission(ctx, request.TenantID, submissionID)
	if err != nil {
		return fmt.Errorf("%w: verify previous submission: %v", ErrVersionConflict, err)
	}
	if submission.RequestID != predecessorID {
		return fmt.Errorf("%w: previous submission does not belong to the predecessor", ErrVersionConflict)
	}
	for fieldID, previous := range request.PreviousResponses {
		answer, ok := submission.Answers[fieldID]
		value, scalar := answer.ScalarText()
		if !ok || !scalar || !submission.SubmittedAt.Equal(previous.PreviousSubmittedAt) || value != previous.Value {
			return fmt.Errorf("%w: previous response does not match the immutable submission", ErrVersionConflict)
		}
	}
	return nil
}
