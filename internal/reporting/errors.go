package reporting

import "errors"

var (
	// ErrInvalid covers a rejected request, filter, definition or transition.
	ErrInvalid = errors.New("reporting: invalid request")
	// ErrNotFound is returned for a definition or run outside the caller's scope.
	ErrNotFound = errors.New("reporting: not found")
	// ErrConflict is returned for a stale expected version or a moved scope.
	ErrConflict = errors.New("reporting: conflicting change")
	// ErrClosureBlocked is returned when a definition cannot advance because a
	// governed precondition is unmet. It is distinct from ErrInvalid so the API
	// can explain which precondition failed.
	ErrClosureBlocked = errors.New("reporting: blocked by a governed precondition")
)
