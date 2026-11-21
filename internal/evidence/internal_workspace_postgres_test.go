//go:build postgres

package evidence

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInternalPostgresWorkspaceWithoutVerifiedIdentityFailsClosed(t *testing.T) {
	store := &PostgresDistributionStore{}
	_, _, err := store.getInternalPostgresWorkspace(context.Background(), "request", time.Now())
	if !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("unverified read: %v", err)
	}
	_, err = store.saveInternalPostgresWorkspace(context.Background(), "request", SaveWorkspaceInput{ExpectedVersion: 1}, time.Now())
	if !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("unverified save: %v", err)
	}
	_, err = store.submitInternalPostgresWorkspace(context.Background(), "request", SubmitWorkspaceInput{ExpectedVersion: 1}, time.Now())
	if !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("unverified submit: %v", err)
	}
}
