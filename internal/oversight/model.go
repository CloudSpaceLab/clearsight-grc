package oversight

import "time"

const ProjectionVersion = "oversight-v4"

type Freshness string

const (
	FreshnessCurrent Freshness = "CURRENT"
	FreshnessStale   Freshness = "STALE"
)

type Scope struct {
	TenantID      string
	LegalEntityID string
}

type Coverage struct {
	Population int  `json:"population"`
	Excluded   *int `json:"excluded,omitempty"`
	Unknown    *int `json:"unknown,omitempty"`
}

type Counts struct {
	CriticalHigh    int `json:"critical_high"`
	Overdue         int `json:"overdue"`
	DueSoon         int `json:"due_soon"`
	RoutingFailures int `json:"routing_failures"`
	Unassigned      int `json:"unassigned"`
	OutcomeFailures int `json:"outcome_failures"`
}

type Intervention struct {
	TargetType string     `json:"target_type"`
	TargetID   string     `json:"target_id"`
	Title      string     `json:"title"`
	Category   string     `json:"category"`
	State      string     `json:"state"`
	Priority   int        `json:"priority"`
	OwnerID    string     `json:"owner_id,omitempty"`
	OwnerName  string     `json:"owner_name,omitempty"`
	DueAt      *time.Time `json:"due_at,omitempty"`
	Reason     string     `json:"reason"`
	NextAction string     `json:"next_action"`
}

type CategoryPressure struct {
	Category string `json:"category"`
	Critical int    `json:"critical"`
	High     int    `json:"high"`
	Other    int    `json:"other"`
	Overdue  int    `json:"overdue"`
}

type AgingBucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type Performance struct {
	OwnerID            string   `json:"owner_id"`
	OwnerName          string   `json:"owner_name"`
	CurrentLoad        int      `json:"current_load"`
	Completed          int      `json:"completed"`
	MedianHours        *float64 `json:"median_hours,omitempty"`
	P75Hours           *float64 `json:"p75_hours,omitempty"`
	SLAAttainment      *float64 `json:"sla_attainment,omitempty"`
	Reassigned         *int     `json:"reassigned,omitempty"`
	Returned           *int     `json:"returned,omitempty"`
	Blocked            int      `json:"blocked"`
	BlockedHours       float64  `json:"blocked_hours"`
	Reopened           int      `json:"reopened"`
	MeasurementSamples int      `json:"measurement_samples"`
}

type ResolutionEstimate struct {
	Category    string  `json:"category"`
	SampleSize  int     `json:"sample_size"`
	MedianHours float64 `json:"median_hours"`
	LowerHours  float64 `json:"lower_hours"`
	UpperHours  float64 `json:"upper_hours"`
	Confidence  string  `json:"confidence"`
	EstimatedBy string  `json:"estimated_by"`
}

type HistoryQuality struct {
	CompletedPopulation     int `json:"completed_population"`
	CompleteLifecycle       int `json:"complete_lifecycle"`
	MissingCreatedEvent     int `json:"missing_created_event"`
	MissingTerminalEvent    int `json:"missing_terminal_event"`
	ExcludedFromDurations   int `json:"excluded_from_durations"`
	ReassignedOwnerExcluded int `json:"reassigned_owner_excluded"`
	ReturnedOwnerExcluded   int `json:"returned_owner_excluded"`
	BlockedOwnerExcluded    int `json:"blocked_owner_excluded"`
	ReopenedOwnerExcluded   int `json:"reopened_owner_excluded"`
}

type Snapshot struct {
	TenantID          string               `json:"-"`
	LegalEntityID     string               `json:"-"`
	GeneratedAt       time.Time            `json:"generated_at"`
	PeriodStart       time.Time            `json:"period_start"`
	PeriodEnd         time.Time            `json:"period_end"`
	ProjectionVersion string               `json:"projection_version"`
	Freshness         Freshness            `json:"freshness"`
	SourceHighWater   map[string]time.Time `json:"source_high_water"`
	Coverage          Coverage             `json:"coverage"`
	Counts            Counts               `json:"counts"`
	Interventions     []Intervention       `json:"interventions"`
	Pressure          []CategoryPressure   `json:"pressure"`
	Aging             []AgingBucket        `json:"aging"`
	Performance       []Performance        `json:"performance"`
	Estimates         []ResolutionEstimate `json:"estimates"`
	HistoryQuality    HistoryQuality       `json:"history_quality"`
}

func estimateConfidence(samples int) string {
	switch {
	case samples >= 30:
		return "HIGH"
	case samples >= 12:
		return "MEDIUM"
	default:
		return "LOW"
	}
}
