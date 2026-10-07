package metricview

import (
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
)

var ErrInvalidObservation = errors.New("metric observation is invalid")

const (
	ObservationSourceOversightSnapshot = "OVERSIGHT_SNAPSHOT"
	ObservationSourceDomainSnapshot    = "DOMAIN_SNAPSHOT"
)

type Observation struct {
	TenantID           string
	LegalEntityID      string
	MetricID           string
	DefinitionRevision string
	SourceKind         string
	SourceID           string
	SourceRevision     string
	SourceHighWater    map[string]time.Time
	GeneratedAt        time.Time
	PeriodStart        time.Time
	PeriodEnd          time.Time
	PostureAsOf        time.Time
	Value              int64
	Condition          Condition
	Currency           string
	MemberCount        *int64
	Freshness          oversight.Freshness
	Completeness       Completeness
	Population         int
	Excluded           *int
	Unknown            *int
}

func ObservationsFromBundle(
	tenantID string,
	legalEntityID string,
	sourceID string,
	sourceHighWater map[string]time.Time,
	bundle Bundle,
) ([]Observation, error) {
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	sourceID = strings.TrimSpace(sourceID)
	if tenantID == "" || legalEntityID == "" || sourceID == "" ||
		bundle.ScopeKind != "LEGAL_ENTITY" || strings.TrimSpace(bundle.ScopeID) != legalEntityID ||
		bundle.GeneratedAt.IsZero() || bundle.PeriodStart.IsZero() || bundle.PeriodEnd.IsZero() ||
		bundle.PeriodStart.After(bundle.PeriodEnd) {
		return nil, ErrInvalidObservation
	}

	observations := make([]Observation, 0, len(bundle.Items))
	seen := make(map[string]struct{}, len(bundle.Items))
	for _, item := range bundle.Items {
		definition, ok := HomeDefinition(item.ID)
		if !ok || definition.Revision != item.DefinitionRevision || definition.Label != item.Label ||
			definition.Unit != item.Unit || definition.Basis != item.Basis || definition.Drill != item.Drill {
			return nil, ErrInvalidObservation
		}
		if _, duplicate := seen[item.ID]; duplicate {
			return nil, ErrInvalidObservation
		}
		seen[item.ID] = struct{}{}
		if item.Value < 0 || item.Population < 0 ||
			(item.Excluded != nil && *item.Excluded < 0) || (item.Unknown != nil && *item.Unknown < 0) {
			return nil, ErrInvalidObservation
		}
		postureAsOf := bundle.PostureAsOf.UTC()
		if postureAsOf.IsZero() {
			postureAsOf = bundle.GeneratedAt.UTC()
		}
		observations = append(observations, Observation{
			TenantID:           tenantID,
			LegalEntityID:      legalEntityID,
			MetricID:           item.ID,
			DefinitionRevision: item.DefinitionRevision,
			SourceKind:         ObservationSourceOversightSnapshot,
			SourceID:           sourceID,
			SourceRevision:     item.SourceRevision,
			SourceHighWater:    cloneHighWater(sourceHighWater),
			GeneratedAt:        bundle.GeneratedAt.UTC(),
			PeriodStart:        bundle.PeriodStart.UTC(),
			PeriodEnd:          bundle.PeriodEnd.UTC(),
			PostureAsOf:        postureAsOf,
			Value:              int64(item.Value),
			Condition:          item.Condition,
			Freshness:          item.Freshness,
			Completeness:       item.Completeness,
			Population:         item.Population,
			Excluded:           cloneInt(item.Excluded),
			Unknown:            cloneInt(item.Unknown),
		})
	}
	if len(observations) != len(homeDefinitions) {
		return nil, ErrInvalidObservation
	}
	return observations, nil
}

func cloneHighWater(value map[string]time.Time) map[string]time.Time {
	copyValue := make(map[string]time.Time, len(value))
	for key, item := range value {
		copyValue[key] = item.UTC()
	}
	return copyValue
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
