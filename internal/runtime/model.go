package runtime

import (
	"encoding/json"
	"time"
)

type TimerState string

const (
	TimerReady     TimerState = "READY"
	TimerClaimed   TimerState = "CLAIMED"
	TimerFired     TimerState = "FIRED"
	TimerCancelled TimerState = "CANCELLED"
	TimerFailed    TimerState = "FAILED"
)

type Timer struct {
	ID         string          `json:"id"`
	TenantID   string          `json:"tenant_id"`
	WorkflowID string          `json:"workflow_id"`
	TaskID     string          `json:"task_id,omitempty"`
	Type       string          `json:"type"`
	DueAt      time.Time       `json:"due_at"`
	State      TimerState      `json:"state"`
	DedupeKey  string          `json:"dedupe_key"`
	Payload    json.RawMessage `json:"payload"`
	Attempts   int             `json:"attempts"`
	LeaseUntil *time.Time      `json:"lease_until,omitempty"`
	LockedBy   string          `json:"locked_by,omitempty"`
	LastError  string          `json:"last_error,omitempty"`
	FailedAt   *time.Time      `json:"failed_at,omitempty"`
}

type OutboxEvent struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	AggregateType  string          `json:"aggregate_type"`
	AggregateID    string          `json:"aggregate_id"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload"`
	OccurredAt     time.Time       `json:"occurred_at"`
	Attempts       int             `json:"attempts"`
	LockedBy       string          `json:"locked_by,omitempty"`
	LeaseUntil     *time.Time      `json:"lease_until,omitempty"`
	NextAttemptAt  *time.Time      `json:"next_attempt_at,omitempty"`
	LastError      string          `json:"last_error,omitempty"`
	DeadLetteredAt *time.Time      `json:"dead_lettered_at,omitempty"`
}
