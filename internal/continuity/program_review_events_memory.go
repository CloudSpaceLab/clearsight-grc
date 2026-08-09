package continuity

import "context"

func (r *MemoryRepository) ProgramEventsAfterVersion(_ context.Context, tenant, programID string, afterVersion int64, limit int) ([]Event, bool, error) {
	if limit <= 0 {
		return []Event{}, false, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := r.programEvents[tenant][programID]
	matched := make([]Event, 0, limit)
	for index := len(all) - 1; index >= 0; index-- {
		event := all[index]
		if event.AggregateVersion <= afterVersion {
			break
		}
		if len(matched) == limit {
			return matched, true, nil
		}
		matched = append(matched, event)
	}
	return matched, false, nil
}
