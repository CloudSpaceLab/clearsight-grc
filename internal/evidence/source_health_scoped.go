package evidence

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type SourceObservationScope string

const (
	ObservationScopeSource         SourceObservationScope = "SOURCE"
	ObservationScopeConnection     SourceObservationScope = "CONNECTION"
	ObservationScopeView           SourceObservationScope = "VIEW"
	ObservationScopeBinding        SourceObservationScope = "BINDING"
	HardMaxSourceHealthScopes                             = 500
	MaxSourceObservationFutureSkew                        = 5 * time.Minute
)

type SourceScopeHealth struct {
	SourceID          string                 `json:"source_id"`
	Scope             SourceObservationScope `json:"scope"`
	ConnectionID      string                 `json:"connection_id,omitempty"`
	ConnectionVersion int64                  `json:"connection_version,omitempty"`
	ViewID            string                 `json:"view_id,omitempty"`
	ViewVersion       int64                  `json:"view_version,omitempty"`
	BindingID         string                 `json:"binding_id,omitempty"`
	BindingVersion    int64                  `json:"binding_version,omitempty"`
	Health            SourceHealth           `json:"health"`
	LastObservedAt    time.Time              `json:"last_observed_at"`
	LastSuccessAt     *time.Time             `json:"last_success_at,omitempty"`
	LatencyMS         int                    `json:"latency_ms,omitempty"`
}

type ScopedSourceHealthRepository interface {
	RecordScopedSourceObservation(context.Context, SourceObservation, time.Time) (Source, error)
	EvaluateScopedSourceHealth(context.Context, time.Time, int) (int, error)
	ListSourceScopeHealth(context.Context, string, string, time.Time, int) ([]SourceScopeHealth, error)
}

func normalizeSourceObservationScope(value SourceObservation) (SourceObservation, error) {
	if value.Scope == "" {
		value.Scope = ObservationScopeSource
	}
	value.ConnectionID = strings.TrimSpace(value.ConnectionID)
	value.ViewID = strings.TrimSpace(value.ViewID)
	value.BindingID = strings.TrimSpace(value.BindingID)
	validPair := func(id string, version int64) bool { return id != "" && version > 0 }
	emptyPair := func(id string, version int64) bool { return id == "" && version == 0 }
	switch value.Scope {
	case ObservationScopeSource:
		if !emptyPair(value.ConnectionID, value.ConnectionVersion) || !emptyPair(value.ViewID, value.ViewVersion) || !emptyPair(value.BindingID, value.BindingVersion) {
			return SourceObservation{}, fmt.Errorf("source-scoped observation cannot include child resource identity")
		}
	case ObservationScopeConnection:
		if !validPair(value.ConnectionID, value.ConnectionVersion) || !emptyPair(value.ViewID, value.ViewVersion) || !emptyPair(value.BindingID, value.BindingVersion) {
			return SourceObservation{}, fmt.Errorf("connection-scoped observation requires exactly one Connection revision")
		}
	case ObservationScopeView:
		if !validPair(value.ConnectionID, value.ConnectionVersion) || !validPair(value.ViewID, value.ViewVersion) || !emptyPair(value.BindingID, value.BindingVersion) {
			return SourceObservation{}, fmt.Errorf("view-scoped observation requires exact Connection and View revisions")
		}
	case ObservationScopeBinding:
		if !validPair(value.ConnectionID, value.ConnectionVersion) || !validPair(value.ViewID, value.ViewVersion) || !validPair(value.BindingID, value.BindingVersion) {
			return SourceObservation{}, fmt.Errorf("binding-scoped observation requires exact Connection, View and Binding revisions")
		}
	default:
		return SourceObservation{}, fmt.Errorf("source observation scope is invalid")
	}
	if value.Success && value.Unavailable {
		return SourceObservation{}, fmt.Errorf("a source observation cannot be both successful and unavailable")
	}
	if value.LatencyMS < 0 {
		return SourceObservation{}, fmt.Errorf("latency_ms cannot be negative")
	}
	return value, nil
}

func validateSourceObservationTime(value SourceObservation, evaluatedAt time.Time) error {
	if evaluatedAt.IsZero() || value.ObservedAt.IsZero() {
		return fmt.Errorf("source observation time is required")
	}
	if value.ObservedAt.After(evaluatedAt.Add(MaxSourceObservationFutureSkew)) {
		return fmt.Errorf("source observation time exceeds the permitted clock skew")
	}
	return nil
}

func scopeObservationKey(value SourceObservation) string {
	return strings.Join([]string{
		string(value.Scope), value.ConnectionID, fmt.Sprint(value.ConnectionVersion),
		value.ViewID, fmt.Sprint(value.ViewVersion), value.BindingID, fmt.Sprint(value.BindingVersion),
	}, "\x1f")
}

func observationHealth(value SourceObservation, source Source, now time.Time) SourceHealth {
	if value.Unavailable {
		return HealthUnavailable
	}
	if !value.Success {
		return HealthDegraded
	}
	freshFor := time.Duration(source.ExpectedFreshnessMinutes) * time.Minute
	if freshFor > 0 && !now.Before(value.ObservedAt.Add(freshFor)) {
		return HealthStale
	}
	return HealthCurrent
}

func worseHealth(left, right SourceHealth) SourceHealth {
	if sourceHealthRank(right) > sourceHealthRank(left) {
		return right
	}
	return left
}

func sourceHealthRank(value SourceHealth) int {
	switch value {
	case HealthUnavailable:
		return 5
	case HealthStale:
		return 4
	case HealthDegraded:
		return 3
	case HealthUnknown:
		return 2
	case HealthCurrent:
		return 1
	default:
		return 2
	}
}

func sourceHealthFromRank(value int) SourceHealth {
	switch value {
	case 5:
		return HealthUnavailable
	case 4:
		return HealthStale
	case 3:
		return HealthDegraded
	case 1:
		return HealthCurrent
	default:
		return HealthUnknown
	}
}

func healthLimit(value int) int {
	if value < 1 {
		return 100
	}
	if value > HardMaxSourceHealthScopes {
		return HardMaxSourceHealthScopes
	}
	return value
}
