package continuity

// AllowedActionTargets exposes the next legal Matter Action states for actor
// work projection. TransitionAction still revalidates the canonical state and
// optimistic Matter version when the command executes.
func AllowedActionTargets(from ActionStatus) []ActionStatus {
	ordered := []ActionStatus{ActionInProgress, ActionImplemented, ActionBlocked, ActionCancelled}
	result := make([]ActionStatus, 0, len(ordered))
	for _, target := range ordered {
		if allowedActionTransition(from, target) {
			result = append(result, target)
		}
	}
	return result
}
