package monitoring

import (
	"encoding/json"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

const (
	AggregateMonitoringForm   = "MONITORING_FORM"
	AggregateMonitoringCheck  = "MONITORING_CHECK"
	AggregateMonitoringResult = "MONITORING_RESULT"

	EventMonitoringFormCreated       = "MONITORING_FORM_CREATED"
	EventMonitoringFormStateChanged  = "MONITORING_FORM_STATE_CHANGED"
	EventMonitoringCheckCreated      = "MONITORING_CHECK_CREATED"
	EventMonitoringCheckStateChanged = "MONITORING_CHECK_STATE_CHANGED"
	EventMonitoringResultRecorded    = "MONITORING_RESULT_RECORDED"
)

// MonitoringEvent is the immutable command journal entry paired with every
// authoritative monitoring write and its transactional outbox record.
type MonitoringEvent struct {
	ID               string          `json:"id"`
	TenantID         string          `json:"tenant_id"`
	AggregateType    string          `json:"aggregate_type"`
	AggregateID      string          `json:"aggregate_id"`
	AggregateVersion int64           `json:"aggregate_version"`
	Type             string          `json:"type"`
	Payload          json.RawMessage `json:"payload"`
	ActorID          string          `json:"actor_id,omitempty"`
	OccurredAt       time.Time       `json:"occurred_at"`
}

func newMonitoringEvent(tenant, aggregateType, aggregateID string, version int64, eventType string, payload any, actorID string, occurredAt time.Time) (MonitoringEvent, error) {
	eventID, err := id.NewUUIDv7()
	if err != nil {
		return MonitoringEvent{}, err
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return MonitoringEvent{}, err
	}
	return MonitoringEvent{
		ID: eventID, TenantID: tenant, AggregateType: aggregateType, AggregateID: aggregateID,
		AggregateVersion: version, Type: eventType, Payload: encoded, ActorID: actorID, OccurredAt: occurredAt.UTC(),
	}, nil
}
