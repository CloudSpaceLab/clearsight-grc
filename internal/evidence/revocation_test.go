package evidence

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInvitationAndSessionRevocation(t *testing.T) {
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository(nil, nil)
	service := NewService(repo, NewMemoryObjectStore())
	service.now = func() time.Time { return now }
	const audience = "contact@example.com"
	request, err := service.CreateRequest(context.Background(), CreateRequestInput{
		TenantID: "bank", SubjectType: "VENDOR", SubjectID: "v", Title: "Provide evidence",
		Purpose: "Complete assurance.", WhyYou: "You are the vendor contact.", Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR",
		Recipient: RecipientInput{Type: RecipientExternalAudience, Audience: audience}, EstimatedMinutes: 2,
		Deadline: now.Add(time.Hour), Fields: []Field{{ID: "answer", Label: "Answer", Type: "text", Required: true}}, CreatedBy: "creator",
	})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := service.IssueInvitation(context.Background(), IssueInvitationInput{TenantID: "bank", RequestID: request.ID, Audience: audience, Purpose: "Vendor response", TTLMinutes: 30, CreatedBy: "creator"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.RevokeInvitationAsRequester(context.Background(), RevokeInvitationAsRequesterInput{TenantID: "bank", RequestID: request.ID, InvitationID: issued.InvitationID, ActorPrincipalID: "creator"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RedeemInvitation(context.Background(), issued.Token, audience); !errors.Is(err, ErrInvitationInvalid) {
		t.Fatalf("expected revoked invitation rejection, got %v", err)
	}
	issued, err = service.IssueInvitation(context.Background(), IssueInvitationInput{TenantID: "bank", RequestID: request.ID, Audience: audience, Purpose: "Vendor response", TTLMinutes: 30, CreatedBy: "creator"})
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.RedeemInvitation(context.Background(), issued.Token, audience)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.RevokeSessionAsRequester(context.Background(), RevokeSessionAsRequesterInput{TenantID: "bank", RequestID: request.ID, SessionID: session.SessionID, ActorPrincipalID: "creator"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.SessionRequest(context.Background(), session.SessionToken); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("expected revoked session rejection, got %v", err)
	}
}
