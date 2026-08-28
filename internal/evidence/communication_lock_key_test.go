package evidence

import (
	"strings"
	"testing"
)

func TestCommunicationPostgresLockKeyIsNULFreeAndComponentSafe(t *testing.T) {
	t.Parallel()

	first := communicationPostgresLockKey("form-communication", "template", "tenant", "entity", "REMINDER", "en-NG")
	second := communicationPostgresLockKey("form-communication", "template", "tenant", "entity", "REMINDERen-", "NG")

	if strings.ContainsRune(first, '\x00') {
		t.Fatalf("PostgreSQL advisory lock key contains NUL: %q", first)
	}
	if first == second {
		t.Fatalf("distinct lock dimensions collided: %q", first)
	}
}
