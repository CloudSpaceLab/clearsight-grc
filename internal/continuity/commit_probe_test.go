package continuity

import (
	"errors"
	"testing"
)

func TestExactCommitResultOnlySuppressesCommitErrorAfterExactProof(t *testing.T) {
	commitErr := errors.New("ambiguous commit")
	probeErr := errors.New("probe unavailable")
	if err := exactCommitResult(commitErr, true, nil); err != nil {
		t.Fatalf("confirmed commit = %v", err)
	}
	if err := exactCommitResult(commitErr, false, nil); !errors.Is(err, commitErr) {
		t.Fatalf("unconfirmed commit = %v", err)
	}
	if err := exactCommitResult(commitErr, false, probeErr); !errors.Is(err, commitErr) || !errors.Is(err, probeErr) {
		t.Fatalf("failed probe = %v", err)
	}
}
