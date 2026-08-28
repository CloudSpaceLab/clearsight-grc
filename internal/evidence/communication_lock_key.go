package evidence

import "encoding/json"

// communicationPostgresLockKey serializes advisory-lock dimensions without
// embedding NUL bytes, which PostgreSQL text parameters reject. JSON-array
// encoding is deterministic for strings and preserves component boundaries.
func communicationPostgresLockKey(parts ...string) string {
	encoded, err := json.Marshal(parts)
	if err != nil {
		// String slices are always JSON-marshalable; keep the fallback
		// deterministic and NUL-free if that invariant ever changes.
		return "[]"
	}
	return string(encoded)
}
