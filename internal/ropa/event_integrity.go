package ropa

import (
	"bytes"
	"encoding/json"
	"strings"
)

// validateActivityEventEnvelope checks the identity carried by an event row
// before its payload is allowed to affect an aggregate. The exact aggregate
// scope is deliberately mandatory; no blank or tenant-only fields are filled
// in on the write path.
func validateActivityEventEnvelope(event Event, scope ActivityScope, aggregateID string, aggregateVersion int64) error {
	if event.AggregateType != "PROCESSING_ACTIVITY" ||
		event.TenantID != scope.TenantID ||
		event.LegalEntityID != scope.LegalEntityID ||
		event.AggregateID != aggregateID ||
		event.AggregateVersion != aggregateVersion {
		return ErrInvalid
	}
	if !validActivityEventType(event.Type) || strings.TrimSpace(event.ID) == "" {
		return ErrInvalid
	}
	return validateActivityEventActor(event)
}

func validateActivityEventActor(event Event) error {
	switch event.ActorType {
	case "SERVICE":
		// A whitespace value is not an empty service identity; accepting it
		// would turn malformed caller input into a valid event.
		if event.ActorID != "" {
			return ErrInvalid
		}
	case "USER":
		if strings.TrimSpace(event.ActorID) == "" {
			return ErrInvalid
		}
	default:
		return ErrInvalid
	}
	return nil
}

func normalizeActivityEventActor(event Event) (Event, error) {
	if event.ActorType == "USER" {
		event.ActorID = strings.TrimSpace(event.ActorID)
	}
	if err := validateActivityEventActor(event); err != nil {
		return Event{}, err
	}
	return event, nil
}

func validateCreatedActivityEvent(event Event, activity ProcessingActivity) (Event, error) {
	if event.AggregateVersion != activity.Version {
		return Event{}, ErrVersionConflict
	}
	if event.Type != EventActivityCreated {
		return Event{}, ErrInvalid
	}
	if err := validateActivityEventEnvelope(event, ActivityScope{TenantID: activity.TenantID, LegalEntityID: activity.LegalEntityID}, activity.ID, activity.Version); err != nil {
		return Event{}, err
	}
	payloadActivity, err := decodeActivity(event.Payload)
	if err != nil {
		return Event{}, err
	}
	if payloadActivity.ID != activity.ID ||
		payloadActivity.TenantID != activity.TenantID ||
		payloadActivity.LegalEntityID != activity.LegalEntityID ||
		payloadActivity.Version != activity.Version {
		return Event{}, ErrInvalid
	}
	if err := validateActivityWhitespace(payloadActivity); err != nil {
		return Event{}, err
	}
	payloadActivity = normalizeProcessingActivity(payloadActivity)
	expected := normalizeProcessingActivity(activity)
	payloadJSON, err := json.Marshal(payloadActivity)
	if err != nil {
		return Event{}, err
	}
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		return Event{}, err
	}
	if !bytes.Equal(payloadJSON, expectedJSON) {
		return Event{}, ErrInvalid
	}
	event.Payload = mustMarshalActivity(expected)
	if event.OccurredAt.IsZero() {
		event.OccurredAt = expected.UpdatedAt
	}
	return normalizeActivityEventActor(event)
}

func prepareActivityEvent(current ProcessingActivity, expectedVersion int64, event Event) (ProcessingActivity, Event, error) {
	if current.Version != expectedVersion {
		return ProcessingActivity{}, Event{}, ErrVersionConflict
	}
	scope := ActivityScope{TenantID: current.TenantID, LegalEntityID: current.LegalEntityID}
	if event.AggregateVersion != expectedVersion+1 {
		return ProcessingActivity{}, Event{}, ErrVersionConflict
	}
	if err := validateActivityEventEnvelope(event, scope, current.ID, expectedVersion+1); err != nil {
		return ProcessingActivity{}, Event{}, err
	}
	next, err := decodeActivity(event.Payload)
	if err != nil {
		return ProcessingActivity{}, Event{}, err
	}
	if next.ID != current.ID ||
		next.TenantID != current.TenantID ||
		next.LegalEntityID != current.LegalEntityID ||
		next.Version != expectedVersion+1 {
		return ProcessingActivity{}, Event{}, ErrInvalid
	}
	if err := validateActivityWhitespace(next); err != nil {
		return ProcessingActivity{}, Event{}, err
	}
	next = normalizeProcessingActivity(next)
	if next.ID != current.ID ||
		next.TenantID != current.TenantID ||
		next.LegalEntityID != current.LegalEntityID ||
		next.Version != expectedVersion+1 ||
		!next.CreatedAt.Equal(current.CreatedAt) {
		return ProcessingActivity{}, Event{}, ErrInvalid
	}
	if !event.OccurredAt.IsZero() {
		next.UpdatedAt = event.OccurredAt.UTC()
	} else if next.UpdatedAt.IsZero() {
		next.UpdatedAt = current.UpdatedAt
	}
	if err := validateActivity(next); err != nil {
		return ProcessingActivity{}, Event{}, err
	}
	if err := ValidateTransitionForWrite(current, next, event.Type); err != nil {
		return ProcessingActivity{}, Event{}, err
	}
	event, err = normalizeActivityEventActor(event)
	if err != nil {
		return ProcessingActivity{}, Event{}, err
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = next.UpdatedAt
	}
	event.OccurredAt = event.OccurredAt.UTC()
	event.Payload = mustMarshalActivity(next)
	return next, event, nil
}
