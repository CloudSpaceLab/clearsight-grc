package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRequesterListsOnlyBoundedActiveExternalSessions(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	request := Request{
		ID: "request-1", TenantID: "bank", SubjectType: "VENDOR", SubjectID: "vendor-1",
		AudienceType: "VENDOR", Recipient: Recipient{Type: RecipientExternalAudience, AudienceHint: "n***@example.com"},
		Status: RequestReady, CreatedBy: "creator", Deadline: now.Add(24 * time.Hour),
	}
	repo := NewMemoryRepository(nil, []Request{request})
	repo.sessions["active-new"] = Session{ID: "session-new", TenantID: "bank", RequestID: request.ID, AudienceHint: "n***@example.com", TokenHash: []byte("secret-new"), CreatedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}
	repo.sessions["active-old"] = Session{ID: "session-old", TenantID: "bank", RequestID: request.ID, AudienceHint: "o***@example.com", TokenHash: []byte("secret-old"), CreatedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)}
	repo.sessions["expired-newest"] = Session{ID: "session-expired", TenantID: "bank", RequestID: request.ID, AudienceHint: "e***@example.com", TokenHash: []byte("expired-secret"), CreatedAt: now, ExpiresAt: now}
	revokedAt := now.Add(-time.Minute)
	repo.sessions["revoked"] = Session{ID: "session-revoked", TenantID: "bank", RequestID: request.ID, AudienceHint: "r***@example.com", TokenHash: []byte("revoked-secret"), CreatedAt: now.Add(time.Minute), ExpiresAt: now.Add(time.Hour), RevokedAt: &revokedAt}

	service := NewService(repo, nil)
	service.now = func() time.Time { return now }
	page, err := service.ListActiveSessionMetadata(context.Background(), ManageSessionsInput{
		TenantID: "bank", RequestID: request.ID, ActorPrincipalID: "creator", Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "session-new" || !page.HasMore {
		t.Fatalf("unexpected active session page: %#v", page)
	}
	payload, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	serialized := strings.ToLower(string(payload))
	for _, forbidden := range []string{"token", "hash", "secret-new", "secret-old", "tenant_id", "request_id", "invitation_id"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("active session response exposed %q: %s", forbidden, payload)
		}
	}
}

func TestActiveSessionInventoryFailsClosedOutsideCurrentRequesterScope(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	request := Request{
		ID: "request-1", TenantID: "bank", LegalEntityID: "entity-a", SubjectType: "MATTER", SubjectID: "matter-1",
		AudienceType: "VENDOR", Recipient: Recipient{Type: RecipientExternalAudience, AudienceHint: "n***@example.com"},
		Status: RequestReady, CreatedBy: "creator", Deadline: now.Add(time.Hour),
	}
	candidates := []RecipientCandidate{{
		PrincipalID: "creator", TenantID: "bank", Kind: "PERSON", Active: true,
		LegalEntityIDs: []string{"entity-a"}, ReadableSubjects: map[string]bool{"MATTER:matter-1": true},
	}}
	repo := NewMemoryRepositoryWithRecipientCandidates(nil, []Request{request}, candidates)
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }

	for name, input := range map[string]ManageSessionsInput{
		"different requester": {TenantID: "bank", LegalEntityID: "entity-a", RequestID: request.ID, ActorPrincipalID: "other", Limit: 50},
		"different entity":    {TenantID: "bank", LegalEntityID: "entity-b", RequestID: request.ID, ActorPrincipalID: "creator", Limit: 50},
		"different tenant":    {TenantID: "other-bank", LegalEntityID: "entity-a", RequestID: request.ID, ActorPrincipalID: "creator", Limit: 50},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := service.ListActiveSessionMetadata(context.Background(), input)
			if !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrRecipientManagerRequired) && !errors.Is(err, ErrSubjectScopeMismatch) {
				t.Fatalf("scope failure = %v", err)
			}
		})
	}
}
