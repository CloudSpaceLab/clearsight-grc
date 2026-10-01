package metric

import "time"

type State string

const (
	StateCritical State = "CRITICAL"
	StateWarning  State = "WARNING"
	StateClear    State = "CLEAR"
	StateUnknown  State = "UNKNOWN"
)

type Direction string

const (
	DirectionUp      Direction = "UP"
	DirectionDown    Direction = "DOWN"
	DirectionFlat    Direction = "FLAT"
	DirectionUnknown Direction = "UNKNOWN"
)

// Snapshot is one explainable statistical fact. It is presentation-neutral:
// UI components may render it as a card, row or chart annotation, but the
// value, coverage and freshness semantics remain server-owned.
type Snapshot struct {
	Code              string    `json:"code"`
	Label             string    `json:"label"`
	Value             int64     `json:"value"`
	Unit              string    `json:"unit"`
	State             State     `json:"state"`
	StateLabel        string    `json:"state_label"`
	Reason            string    `json:"reason"`
	Population        int       `json:"population"`
	Excluded          *int      `json:"excluded,omitempty"`
	Unknown           *int      `json:"unknown,omitempty"`
	Complete          bool      `json:"complete"`
	GeneratedAt       time.Time `json:"generated_at"`
	ProjectionVersion string    `json:"projection_version"`
	PreviousValue     *int64    `json:"previous_value,omitempty"`
	Delta             *int64    `json:"delta,omitempty"`
	Direction         Direction `json:"direction,omitempty"`
	DrillKey          string    `json:"drill_key,omitempty"`
}
