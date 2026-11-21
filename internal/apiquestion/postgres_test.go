//go:build postgres

package apiquestion

import (
	"context"
	"errors"
	"testing"
)

func TestDraftWriteRequiresVerifiedIdentity(t *testing.T) {
	_, err := (&PostgresStore{}).WriteDraft(context.Background(), "form", 1, "question", 0, DraftInput{Mode: "MANUAL"})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected unavailable for missing identity, got %v", err)
	}
}
