package continuity

import "strings"

// ActionResponsibility returns the stored responsibility that may execute an
// Action. Older and ordinary Actions remain performer work.
func ActionResponsibility(action Action) string {
	value := strings.ToUpper(strings.TrimSpace(action.RequiredResponsibility))
	if value == "" {
		return "PERFORMER"
	}
	return value
}

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
