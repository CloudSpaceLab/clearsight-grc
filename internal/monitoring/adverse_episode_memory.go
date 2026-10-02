package monitoring

import (
	"context"
	"fmt"
	"time"
)

func (r *MemoryRepository) OpenOrUpdateAdverseEpisode(ctx context.Context, observation AdverseEpisodeObservation, episodeID string, now time.Time) (AdverseEpisode, AdverseEpisodeChange, error) {
	if err := ctx.Err(); err != nil {
		return AdverseEpisode{}, AdverseEpisodeNoChange, err
	}
	if err := validateEpisodeObservation(observation); err != nil || episodeID == "" {
		return AdverseEpisode{}, AdverseEpisodeNoChange, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	key, current, found := r.openAdverseEpisodeLocked(observation.TenantID, observation.LegalEntityID, observation.Check.ID)
	if !found {
		opened := AdverseEpisode{
			ID: episodeID, TenantID: observation.TenantID, LegalEntityID: observation.LegalEntityID,
			ProgramID: observation.ProgramID, MonitoringCheckID: observation.Check.ID, State: AdverseEpisodeOpen,
			FirstResultID: observation.Result.ID, LastResultID: observation.Result.ID,
			LastCheckVersion: observation.Check.Version, LastBand: observation.Result.Evaluation.Band,
			LastScore: cloneFloatPointer(observation.Result.Evaluation.Score), LastCoverage: observation.Result.Evaluation.Coverage,
			OpenedAt: observation.Result.EvaluatedAt.UTC(), UpdatedAt: now.UTC(), RecordVersion: 1,
		}
		event, err := newMonitoringEvent(opened.TenantID, AggregateMonitoringAdverseEpisode, opened.ID, opened.RecordVersion, EventMonitoringAdverseEpisodeOpened, episodeEventPayload(opened), "", now)
		if err != nil {
			return AdverseEpisode{}, AdverseEpisodeNoChange, err
		}
		r.adverseEpisodes[episodeMemoryKey(opened.TenantID, opened.LegalEntityID, opened.ID)] = cloneValue(opened)
		r.appendEvent(event)
		return cloneValue(opened), AdverseEpisodeOpened, nil
	}
	if current.ProgramID != observation.ProgramID {
		return AdverseEpisode{}, AdverseEpisodeNoChange, ErrInvalid
	}
	if current.LastResultID == observation.Result.ID {
		return cloneValue(current), AdverseEpisodeNoChange, nil
	}
	worsened := episodeWorsened(current, observation)
	current.LastResultID = observation.Result.ID
	current.LastCheckVersion = observation.Check.Version
	current.LastBand = observation.Result.Evaluation.Band
	current.LastScore = cloneFloatPointer(observation.Result.Evaluation.Score)
	current.LastCoverage = observation.Result.Evaluation.Coverage
	current.UpdatedAt = now.UTC()
	current.RecordVersion++
	eventType := EventMonitoringAdverseEpisodeUpdated
	change := AdverseEpisodeUpdated
	if worsened {
		eventType = EventMonitoringAdverseEpisodeWorsened
		change = AdverseEpisodeWorsened
	}
	event, err := newMonitoringEvent(current.TenantID, AggregateMonitoringAdverseEpisode, current.ID, current.RecordVersion, eventType, episodeEventPayload(current), "", now)
	if err != nil {
		return AdverseEpisode{}, AdverseEpisodeNoChange, err
	}
	r.adverseEpisodes[key] = cloneValue(current)
	r.appendEvent(event)
	return cloneValue(current), change, nil
}

func (r *MemoryRepository) CloseAdverseEpisode(ctx context.Context, observation AdverseEpisodeObservation, now time.Time) (*AdverseEpisode, AdverseEpisodeChange, error) {
	if err := ctx.Err(); err != nil {
		return nil, AdverseEpisodeNoChange, err
	}
	if err := validateEpisodeObservation(observation); err != nil {
		return nil, AdverseEpisodeNoChange, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key, current, found := r.openAdverseEpisodeLocked(observation.TenantID, observation.LegalEntityID, observation.Check.ID)
	if !found {
		return nil, AdverseEpisodeNoChange, nil
	}
	closedAt := observation.Result.EvaluatedAt.UTC()
	current.State = AdverseEpisodeClosed
	current.LastResultID = observation.Result.ID
	current.LastCheckVersion = observation.Check.Version
	current.LastBand = observation.Result.Evaluation.Band
	current.LastScore = cloneFloatPointer(observation.Result.Evaluation.Score)
	current.LastCoverage = observation.Result.Evaluation.Coverage
	current.ClosedAt = &closedAt
	current.UpdatedAt = now.UTC()
	current.RecordVersion++
	event, err := newMonitoringEvent(current.TenantID, AggregateMonitoringAdverseEpisode, current.ID, current.RecordVersion, EventMonitoringAdverseEpisodeCleared, episodeEventPayload(current), "", now)
	if err != nil {
		return nil, AdverseEpisodeNoChange, err
	}
	r.adverseEpisodes[key] = cloneValue(current)
	r.appendEvent(event)
	cloned := cloneValue(current)
	return &cloned, AdverseEpisodeCleared, nil
}

func (r *MemoryRepository) OpenAdverseEpisode(ctx context.Context, tenant, legalEntityID, checkID string) (AdverseEpisode, error) {
	if err := ctx.Err(); err != nil {
		return AdverseEpisode{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, value, found := r.openAdverseEpisodeLocked(tenant, legalEntityID, checkID)
	if !found {
		return AdverseEpisode{}, ErrNotFound
	}
	return cloneValue(value), nil
}

func (r *MemoryRepository) AttachAdverseEpisodeMatter(ctx context.Context, tenant, legalEntityID, episodeID, matterID string, now time.Time) (AdverseEpisode, error) {
	if err := ctx.Err(); err != nil {
		return AdverseEpisode{}, err
	}
	if tenant == "" || legalEntityID == "" || episodeID == "" || matterID == "" {
		return AdverseEpisode{}, ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := episodeMemoryKey(tenant, legalEntityID, episodeID)
	value, ok := r.adverseEpisodes[key]
	if !ok {
		return AdverseEpisode{}, ErrNotFound
	}
	if value.MatterID != "" {
		if value.MatterID == matterID {
			return cloneValue(value), nil
		}
		return AdverseEpisode{}, ErrConflict
	}
	value.MatterID = matterID
	value.UpdatedAt = now.UTC()
	value.RecordVersion++
	event, err := newMonitoringEvent(value.TenantID, AggregateMonitoringAdverseEpisode, value.ID, value.RecordVersion, EventMonitoringAdverseEpisodeMatterLinked, episodeEventPayload(value), "", now)
	if err != nil {
		return AdverseEpisode{}, err
	}
	r.adverseEpisodes[key] = cloneValue(value)
	r.appendEvent(event)
	return cloneValue(value), nil
}

func (r *MemoryRepository) openAdverseEpisodeLocked(tenant, legalEntityID, checkID string) (string, AdverseEpisode, bool) {
	for key, value := range r.adverseEpisodes {
		if value.TenantID == tenant && value.LegalEntityID == legalEntityID && value.MonitoringCheckID == checkID && value.State == AdverseEpisodeOpen {
			return key, value, true
		}
	}
	return "", AdverseEpisode{}, false
}

func episodeMemoryKey(tenant, legalEntityID, episodeID string) string {
	return fmt.Sprintf("%s\x00%s\x00%s", tenant, legalEntityID, episodeID)
}

func cloneFloatPointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
