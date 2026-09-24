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
	// ErrReportTooLarge is returned when an asynchronous run reaches its row
	// or byte ceiling. The run is terminally failed with a narrower code and no
	// partial artefact is published.
	ErrReportTooLarge = errors.New("reporting: bounded report limit exceeded")
	// ErrArtifactIntegrity is returned when stored bytes do not match the digest
	// calculated over the exact bytes offered to object storage.
	ErrArtifactIntegrity = errors.New("reporting: stored artefact integrity mismatch")
)
