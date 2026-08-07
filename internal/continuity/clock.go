package continuity

import "time"

// NewServiceWithClock constructs the continuity service with an explicit clock.
// It is useful for deterministic simulations and tests where valid-time and
// observation-window semantics must be evaluated at the same scenario instant.
func NewServiceWithClock(repo Repository, clock func() time.Time) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{repo: repo, now: clock}
}
