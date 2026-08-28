package evidence

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMemoryOTPConcurrentFailuresEachConsumeAttempt(t *testing.T) {
	now := time.Date(2026, 8, 28, 0, 30, 0, 0, time.UTC)
	store := NewMemoryDistributionAccessStore(nil)
	challenge := OTPChallenge{
		ID: "challenge-a", TenantID: "tenant-a", LegalEntityID: "entity-a",
		DistributionID: "distribution-a", RouteID: "route-a", RecipientID: "recipient-a",
		Digest: []byte("01234567890123456789012345678901"), ExpiresAt: now.Add(OTPValidity),
		MaxAttempts: OTPMaxAttempts, MaxResends: OTPMaxResends, CreatedAt: now,
	}
	if err := store.CreateOTPChallenge(context.Background(), challenge); err != nil {
		t.Fatal(err)
	}

	failedAttempt := cloneOTPChallenge(challenge)
	failedAttempt.Attempts = 1
	var wg sync.WaitGroup
	errs := make(chan error, OTPMaxAttempts)
	for range OTPMaxAttempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- store.UpdateOTPChallenge(context.Background(), failedAttempt, 0, 0, challenge.Digest)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent failed attempt was not counted: %v", err)
		}
	}

	store.mu.RLock()
	persisted := store.challenges[challenge.ID]
	store.mu.RUnlock()
	if persisted.Attempts != OTPMaxAttempts {
		t.Fatalf("concurrent guesses consumed %d attempts, want %d", persisted.Attempts, OTPMaxAttempts)
	}
	if err := store.UpdateOTPChallenge(context.Background(), failedAttempt, 0, 0, challenge.Digest); err == nil {
		t.Fatal("attempt counter exceeded the configured cap")
	}

	route := AccessRoute{ID: challenge.RouteID}
	snapshot, err := store.ActiveOTPChallenge(context.Background(), route, challenge.RecipientID, now.Add(time.Minute))
	if err != nil || !snapshot.Found || snapshot.Challenge.Attempts != OTPMaxAttempts {
		t.Fatalf("exhausted challenge did not remain authoritative until expiry: %+v %v", snapshot, err)
	}
	replacement := challenge
	replacement.ID = "challenge-b"
	replacement.CreatedAt = now.Add(time.Minute)
	replacement.ExpiresAt = replacement.CreatedAt.Add(OTPValidity)
	if err := store.CreateOTPChallenge(context.Background(), replacement); err == nil {
		t.Fatal("attempt exhaustion was bypassed with a fresh challenge before expiry")
	}
}
