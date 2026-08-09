package continuity

import "context"

func (r *MemoryRepository) ProgramEventsAfterVersion(_ context.Context, tenant, programID string, afterVersion int64, limit int) ([]Event, int, error) {
	if limit <= 0 {
		return []Event{}, 0, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := r.programEvents[tenant][programID]
	matched := make([]Event, 0, minInt(len(all), limit))
	total := 0
	for index := len(all) - 1; index >= 0; index-- {
		event := all[index]
		if event.AggregateVersion <= afterVersion {
			break
		}
		total++
		if len(matched) < limit {
			matched = append(matched, event)
		}
	}
	return matched, total, nil
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
