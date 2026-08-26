package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRequesterInvitationAdministrationIsSanitizedAndTransactional(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 11, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository(nil, nil)
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }
	request, err := service.CreateRequest(ctx, CreateRequestInput{
		TenantID: "bank", SubjectType: "VENDOR", SubjectID: "vendor-1", Title: "Provide evidence",
		Purpose: "Complete the current review.", WhyYou: "You are the selected respondent.", Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR",
		Recipient: RecipientInput{Type: RecipientExternalAudience, Audience: "contact@example.com"}, EstimatedMinutes: 5,
		Deadline: now.Add(2 * time.Hour), Fields: []Field{{ID: "answer", Label: "Answer", Type: "text", Required: true}}, CreatedBy: "creator",
	})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := service.IssueInvitation(ctx, IssueInvitationInput{TenantID: "bank", RequestID: request.ID, Audience: "contact@example.com", Purpose: "Respond", TTLMinutes: 60, CreatedBy: "creator"})
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.RedeemInvitation(ctx, issued.Token, "contact@example.com")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.ListInvitationMetadata(ctx, ManageInvitationsInput{TenantID: "bank", RequestID: request.ID, ActorPrincipalID: "other"}); !errors.Is(err, ErrRecipientManagerRequired) {
		t.Fatalf("non-requester listed invitations: %v", err)
	}
	metadata, err := service.ListInvitationMetadata(ctx, ManageInvitationsInput{TenantID: "bank", RequestID: request.ID, ActorPrincipalID: "creator"})
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata) != 1 || metadata[0].ID != issued.InvitationID || metadata[0].AudienceHint != "c***@example.com" {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(payload)), "token") || strings.Contains(string(payload), "contact@example.com") {
		t.Fatalf("protected invitation data leaked in metadata: %s", payload)
	}

	replacement, err := service.ReplaceInvitation(ctx, ReplaceInvitationInput{
		TenantID: "bank", RequestID: request.ID, InvitationID: issued.InvitationID, ActorPrincipalID: "creator",
		Audience: "contact@example.com", Purpose: "Respond", TTLMinutes: 45,
	})
	if err != nil {
		t.Fatal(err)
	}
	if replacement.Token == "" || replacement.InvitationID == issued.InvitationID {
		t.Fatalf("replacement did not issue one new capability: %#v", replacement)
	}
	if _, err := service.RedeemInvitation(ctx, issued.Token, "contact@example.com"); !errors.Is(err, ErrInvitationInvalid) {
		t.Fatalf("replaced invitation remained valid: %v", err)
	}
	if _, _, err := service.SessionRequest(ctx, session.SessionToken); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("session from replaced invitation remained valid: %v", err)
	}
	if _, err := service.RedeemInvitation(ctx, replacement.Token, "contact@example.com"); err != nil {
		t.Fatalf("new invitation was not usable: %v", err)
	}
}

func TestLegacyUnattributedRevocationFailsClosed(t *testing.T) {
	service := NewService(NewMemoryRepository(nil, nil), nil)
	if err := service.RevokeInvitation(context.Background(), "bank", "invitation"); !errors.Is(err, ErrRecipientManagerRequired) {
		t.Fatalf("unattributed invitation revocation did not fail closed: %v", err)
	}
	if err := service.RevokeSession(context.Background(), "bank", "session"); !errors.Is(err, ErrRecipientManagerRequired) {
		t.Fatalf("unattributed session revocation did not fail closed: %v", err)
	}
}
