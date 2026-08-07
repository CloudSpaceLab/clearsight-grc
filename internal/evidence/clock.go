package evidence

import "time"

// NewServiceWithClock is intended for deterministic simulations and tests that
// must evaluate evidence deadlines against the same scenario clock as the
// surrounding domain. Production should normally use NewService.
func NewServiceWithClock(repo Repository, store ObjectStore, now func() time.Time) *Service {
	service := NewService(repo, store)
	if now != nil {
		service.now = now
	}
	return service
}
