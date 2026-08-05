package capture

import (
	"errors"
	"testing"
	"time"
)

func TestInvitationIsOneTimeAndRequestScoped(t *testing.T) {
	now := time.Date(2026, 8, 5, 2, 0, 0, 0, time.UTC)
	service := NewInvitationService(func() time.Time { return now })
	token, err := service.Issue("request-1", "external@example.com", time.Hour)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	session, err := service.Redeem(token)
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if session.RequestID != "request-1" || session.Audience != "external@example.com" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if _, err := service.Redeem(token); !errors.Is(err, ErrInvitationUsed) {
		t.Fatalf("expected replay rejection, got %v", err)
	}
}

func TestInvitationExpires(t *testing.T) {
	now := time.Date(2026, 8, 5, 2, 0, 0, 0, time.UTC)
	service := NewInvitationService(func() time.Time { return now })
	token, err := service.Issue("request-1", "external@example.com", time.Minute)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	now = now.Add(2 * time.Minute)
	if _, err := service.Redeem(token); !errors.Is(err, ErrInvitationExpired) {
		t.Fatalf("expected expiry rejection, got %v", err)
	}
}
