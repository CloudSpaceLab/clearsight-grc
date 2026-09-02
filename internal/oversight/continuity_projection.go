package oversight

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

// FromMatterAggregates builds the in-memory runtime snapshot from the same
// seeded continuity records used by the rest of the application. It never
// substitutes a presentation-specific metric fixture.
func FromMatterAggregates(tenantID, legalEntityID string, aggregates []continuity.MatterAggregate, now time.Time) Snapshot {
	now = now.UTC()
	excluded, unknown := 0, 0
	value := Snapshot{
		TenantID: tenantID, LegalEntityID: legalEntityID, GeneratedAt: now,
		PeriodStart: now.Add(-90 * 24 * time.Hour), PeriodEnd: now, ProjectionVersion: ProjectionVersion,
		SourceHighWater: map[string]time.Time{}, Coverage: Coverage{Excluded: &excluded, Unknown: &unknown},
		Interventions: []Intervention{}, Pressure: []CategoryPressure{}, Performance: []Performance{}, Estimates: []ResolutionEstimate{},
	}
	pressure := map[string]*CategoryPressure{}
	type ownerHistory struct {
		current, completed, blocked, reopened int
	}
	owners := map[string]*ownerHistory{}
	ageCounts := []int{0, 0, 0, 0}
	for _, aggregate := range aggregates {
		matter := aggregate.Matter
		if matter.TenantID != tenantID || matter.LegalEntityID != legalEntityID {
			continue
		}
		value.Coverage.Population++
		switch oversightScopeState(matter.Scope) {
		case "EXCLUDED":
			excluded++
			continue
		case "UNKNOWN":
			unknown++
			continue
		}
		if matter.UpdatedAt.After(value.SourceHighWater["matters"]) {
			value.SourceHighWater["matters"] = matter.UpdatedAt
		}
		for _, action := range aggregate.Actions {
			if action.UpdatedAt.After(value.SourceHighWater["actions"]) {
				value.SourceHighWater["actions"] = action.UpdatedAt
			}
		}
		for _, result := range aggregate.VerificationResults {
			if result.ObservedAt.After(value.SourceHighWater["verification_results"]) {
				value.SourceHighWater["verification_results"] = result.ObservedAt
			}
		}
		var owner *ownerHistory
		if matter.OwnerPrincipalID != "" {
			owner = owners[matter.OwnerPrincipalID]
			if owner == nil {
				owner = &ownerHistory{}
				owners[matter.OwnerPrincipalID] = owner
			}
			for _, action := range aggregate.Actions {
				if action.Status == continuity.ActionBlocked {
					owner.blocked++
				}
			}
			owner.reopened += matter.ReopenCount
		}
		if matter.Status == continuity.MatterClosed || matter.Status == continuity.MatterCancelled {
			if matter.Status == continuity.MatterClosed && matter.ClosedAt != nil && !matter.ClosedAt.Before(value.PeriodStart) {
				value.HistoryQuality.CompletedPopulation++
				value.HistoryQuality.MissingCreatedEvent++
				value.HistoryQuality.MissingTerminalEvent++
				value.HistoryQuality.ExcludedFromDurations++
				if owner != nil {
					owner.completed++
				}
			}
			continue
		}
		if owner != nil {
			owner.current++
		}
		entry := pressure[string(matter.Type)]
		if entry == nil {
			entry = &CategoryPressure{Category: string(matter.Type)}
			pressure[string(matter.Type)] = entry
		}
		switch matter.Priority {
		case 5:
			entry.Critical++
		case 4:
			entry.High++
		default:
			entry.Other++
		}
		if matter.Priority >= 4 {
			value.Counts.CriticalHigh++
		}
		if matter.OwnerPrincipalID == "" {
			value.Counts.Unassigned++
		}
		if matter.DueAt != nil && matter.DueAt.Before(now) {
			value.Counts.Overdue++
			entry.Overdue++
		} else if matter.DueAt != nil && matter.DueAt.Before(now.Add(7*24*time.Hour)) {
			value.Counts.DueSoon++
		}
		age := now.Sub(matter.CreatedAt)
		switch {
		case age <= 7*24*time.Hour:
			ageCounts[0]++
		case age <= 30*24*time.Hour:
			ageCounts[1]++
		case age <= 90*24*time.Hour:
			ageCounts[2]++
		default:
			ageCounts[3]++
		}
		if matter.Priority >= 4 || matter.OwnerPrincipalID == "" || (matter.DueAt != nil && matter.DueAt.Before(now)) {
			item := Intervention{TargetType: "MATTER", TargetID: matter.ID, Title: matter.Title, Category: string(matter.Type), State: string(matter.Status), Priority: matter.Priority, OwnerID: matter.OwnerPrincipalID, DueAt: matter.DueAt}
			item.Reason, item.NextAction = interventionCopy(item, now)
			value.Interventions = append(value.Interventions, item)
		}
	}
	for _, entry := range pressure {
		value.Pressure = append(value.Pressure, *entry)
	}
	sort.Slice(value.Pressure, func(i, j int) bool {
		left := value.Pressure[i].Critical + value.Pressure[i].High + value.Pressure[i].Other
		right := value.Pressure[j].Critical + value.Pressure[j].High + value.Pressure[j].Other
		if left == right {
			return value.Pressure[i].Category < value.Pressure[j].Category
		}
		return left > right
	})
	sort.Slice(value.Interventions, func(i, j int) bool {
		leftOverdue := value.Interventions[i].DueAt != nil && value.Interventions[i].DueAt.Before(now)
		rightOverdue := value.Interventions[j].DueAt != nil && value.Interventions[j].DueAt.Before(now)
		if leftOverdue != rightOverdue {
			return leftOverdue
		}
		return value.Interventions[i].Priority > value.Interventions[j].Priority
	})
	if len(value.Interventions) > 30 {
		value.Interventions = value.Interventions[:30]
	}
	value.Aging = []AgingBucket{{Label: "0–7 days", Count: ageCounts[0]}, {Label: "8–30 days", Count: ageCounts[1]}, {Label: "31–90 days", Count: ageCounts[2]}, {Label: "Over 90 days", Count: ageCounts[3]}}
	for ownerID, history := range owners {
		item := Performance{OwnerID: ownerID, OwnerName: ownerID, CurrentLoad: history.current, Completed: history.completed, Blocked: history.blocked, Reopened: history.reopened}
		value.Performance = append(value.Performance, item)
	}
	sort.Slice(value.Performance, func(i, j int) bool {
		if value.Performance[i].CurrentLoad == value.Performance[j].CurrentLoad {
			return value.Performance[i].OwnerID < value.Performance[j].OwnerID
		}
		return value.Performance[i].CurrentLoad > value.Performance[j].CurrentLoad
	})
	return value
}

func percentile(values []float64, quantile float64) float64 {
	ordered := append([]float64(nil), values...)
	sort.Float64s(ordered)
	if len(ordered) == 0 {
		return 0
	}
	position := quantile * float64(len(ordered)-1)
	lower := int(position)
	upper := lower + 1
	if upper >= len(ordered) {
		return ordered[lower]
	}
	fraction := position - float64(lower)
	return ordered[lower] + (ordered[upper]-ordered[lower])*fraction
}

func oversightScopeState(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "INCLUDED"
	}
	var scope map[string]any
	if err := json.Unmarshal(raw, &scope); err != nil {
		return "UNKNOWN"
	}
	value, exists := scope["access"]
	if !exists {
		return "INCLUDED"
	}
	access, ok := value.(string)
	if !ok {
		return "UNKNOWN"
	}
	switch strings.ToUpper(strings.TrimSpace(access)) {
	case "PUBLIC", "INTERNAL":
		return "INCLUDED"
	case "RESTRICTED":
		return "EXCLUDED"
	default:
		return "UNKNOWN"
	}
}
