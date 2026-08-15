package evidence

import (
	"context"
	"sort"
	"time"
)

func (r *MemoryRepository) RecordScopedSourceObservation(_ context.Context, observation SourceObservation, evaluatedAt time.Time) (Source, error) {
	observation, err := normalizeSourceObservationScope(observation)
	if err != nil {
		return Source{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	source, ok := r.sources[observation.SourceID]
	if !ok || source.TenantID != observation.TenantID {
		return Source{}, ErrNotFound
	}
	r.observations[observation.ID] = observation
	health, lastObserved, lastSuccess := r.aggregateSourceHealthLocked(source, evaluatedAt)
	source.Health = health
	source.LastObservedAt = lastObserved
	source.LastSuccessAt = lastSuccess
	source.Version++
	if evaluatedAt.After(source.UpdatedAt) {
		source.UpdatedAt = evaluatedAt
	}
	r.sources[source.ID] = source
	return source, nil
}

func (r *MemoryRepository) EvaluateScopedSourceHealth(_ context.Context, now time.Time, limit int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := make([]string, 0, len(r.sources))
	for id := range r.sources {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	changed := 0
	for _, sourceID := range ids {
		if changed >= limit {
			break
		}
		source := r.sources[sourceID]
		if source.Status != SourceActive {
			continue
		}
		health, lastObserved, lastSuccess := r.aggregateSourceHealthLocked(source, now)
		if health == source.Health {
			continue
		}
		source.Health = health
		source.LastObservedAt = lastObserved
		source.LastSuccessAt = lastSuccess
		source.Version++
		source.UpdatedAt = now
		r.sources[source.ID] = source
		changed++
	}
	return changed, nil
}

func (r *MemoryRepository) ListSourceScopeHealth(_ context.Context, tenantID, sourceID string, now time.Time, limit int) ([]SourceScopeHealth, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	source, ok := r.sources[sourceID]
	if !ok || source.TenantID != tenantID {
		return nil, ErrNotFound
	}
	latest := map[string]SourceObservation{}
	lastSuccess := map[string]time.Time{}
	for _, observation := range r.observations {
		if observation.TenantID != tenantID || observation.SourceID != sourceID {
			continue
		}
		key := scopeObservationKey(observation)
		if current, exists := latest[key]; !exists || observation.ObservedAt.After(current.ObservedAt) || (observation.ObservedAt.Equal(current.ObservedAt) && observation.ID > current.ID) {
			latest[key] = observation
		}
		if observation.Success {
			if current, exists := lastSuccess[key]; !exists || observation.ObservedAt.After(current) {
				lastSuccess[key] = observation.ObservedAt
			}
		}
	}
	values := make([]SourceScopeHealth, 0, len(latest))
	for key, observation := range latest {
		value := sourceScopeHealthFromObservation(source, observation, now)
		if successAt, ok := lastSuccess[key]; ok {
			copy := successAt
			value.LastSuccessAt = &copy
		}
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		left, right := values[i], values[j]
		if left.Scope != right.Scope {
			return left.Scope < right.Scope
		}
		if left.ConnectionID != right.ConnectionID {
			return left.ConnectionID < right.ConnectionID
		}
		if left.ViewID != right.ViewID {
			return left.ViewID < right.ViewID
		}
		if left.BindingID != right.BindingID {
			return left.BindingID < right.BindingID
		}
		return left.LastObservedAt.After(right.LastObservedAt)
	})
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}

func (r *MemoryRepository) aggregateSourceHealthLocked(source Source, now time.Time) (SourceHealth, *time.Time, *time.Time) {
	latest := map[string]SourceObservation{}
	var lastObserved, lastSuccess *time.Time
	if source.LastObservedAt != nil {
		copy := *source.LastObservedAt
		lastObserved = &copy
	}
	if source.LastSuccessAt != nil {
		copy := *source.LastSuccessAt
		lastSuccess = &copy
	}
	for _, observation := range r.observations {
		if observation.TenantID != source.TenantID || observation.SourceID != source.ID {
			continue
		}
		if lastObserved == nil || observation.ObservedAt.After(*lastObserved) {
			copy := observation.ObservedAt
			lastObserved = &copy
		}
		if observation.Success && (lastSuccess == nil || observation.ObservedAt.After(*lastSuccess)) {
			copy := observation.ObservedAt
			lastSuccess = &copy
		}
		key := scopeObservationKey(observation)
		if current, exists := latest[key]; !exists || observation.ObservedAt.After(current.ObservedAt) || (observation.ObservedAt.Equal(current.ObservedAt) && observation.ID > current.ID) {
			latest[key] = observation
		}
	}
	if len(latest) == 0 {
		if source.Health == HealthUnavailable {
			return HealthUnavailable, lastObserved, lastSuccess
		}
		if lastSuccess != nil && !now.Before(lastSuccess.Add(time.Duration(source.ExpectedFreshnessMinutes)*time.Minute)) {
			return HealthStale, lastObserved, lastSuccess
		}
		return source.Health, lastObserved, lastSuccess
	}
	health := HealthCurrent
	for _, observation := range latest {
		health = worseHealth(health, observationHealth(observation, source, now))
	}
	return health, lastObserved, lastSuccess
}

func sourceScopeHealthFromObservation(source Source, observation SourceObservation, now time.Time) SourceScopeHealth {
	return SourceScopeHealth{
		SourceID: source.ID, Scope: observation.Scope,
		ConnectionID: observation.ConnectionID, ConnectionVersion: observation.ConnectionVersion,
		ViewID: observation.ViewID, ViewVersion: observation.ViewVersion,
		BindingID: observation.BindingID, BindingVersion: observation.BindingVersion,
		Health: observationHealth(observation, source, now), LastObservedAt: observation.ObservedAt, LatencyMS: observation.LatencyMS,
	}
}
