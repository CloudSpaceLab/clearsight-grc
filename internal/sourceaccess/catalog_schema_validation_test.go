package sourceaccess

import (
	"errors"
	"testing"
	"time"
)

func TestCatalogViewStableKeysMustExistInNativeSchema(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	view := catalogViewRevision(now)
	view.StableKeys = []string{"missing_key"}
	if _, err := normalizeViewRevision(view); !errors.Is(err, ErrCatalogInvalid) {
		t.Fatalf("stable key outside native schema should fail, got %v", err)
	}
}
