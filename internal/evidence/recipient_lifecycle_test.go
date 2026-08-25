package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestWrongRecipientRemovesWorkUntilRequesterReassigns(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository(nil, nil)
	service := NewService(repo, nil)
	now := time.Date(2026, 8, 9, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	request, err := service.CreateRequest(ctx, CreateRequestInput{
		TenantID: "bank", SubjectType: "CONTROL", SubjectID: "control-1",
		Title: "Confirm owner", Purpose: "Collect the current owner.", WhyYou: "You were selected as respondent.",
		Sensitivity: "INTERNAL", AudienceType: "INTERNAL",
		Recipient:        RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: "actor-a"},
		EstimatedMinutes: 2, Deadline: now.Add(time.Hour),
		Fields: []Field{{ID: "owner", Label: "Owner", Type: "text", Required: true}}, CreatedBy: "creator",
	})
	if err != nil {
		t.Fatal(err)
	}

	wrong, err := service.DeclareWrongRecipient(ctx, DeclareWrongRecipientInput{
		TenantID: "bank", RequestID: request.ID, ActorPrincipalID: "actor-a",
		Reason: "This control moved to Operations.", ExpectedVersion: request.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if wrong.Recipient.State != RecipientStateReassignmentRequired || wrong.Version != request.Version+1 {
		t.Fatalf("wrong-recipient state was not canonical: %#v", wrong)
	}
	mine, err := service.ListVisibleRequests(ctx, "bank", "actor-a", 10, func(Request) bool { return true })
	if err != nil || len(mine) != 0 {
		t.Fatalf("wrong-recipient request remained actor work: %#v err=%v", mine, err)
	}
	_, err = service.Submit(ctx, Submission{
		TenantID: "bank", RequestID: request.ID, SubmittedBy: "actor-a", Channel: "INTERNAL",
		Answers: formcontract.TextAnswers(map[string]string{"owner": "Operations"}), ExpectedVersion: wrong.Version,
	})
	if !errors.Is(err, ErrRecipientMismatch) {
		t.Fatalf("wrong recipient retained submission authority: %v", err)
	}

	if _, err := service.ReassignRecipient(ctx, ReassignRecipientInput{
		TenantID: "bank", RequestID: request.ID, ActorPrincipalID: "actor-a",
		Recipient: RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: "actor-b"},
		Reason:    "Redirect to Operations.", ExpectedVersion: wrong.Version,
	}); !errors.Is(err, ErrRecipientManagerRequired) {
		t.Fatalf("recipient was allowed to self-redirect without requester authority: %v", err)
	}

	reassigned, err := service.ReassignRecipient(ctx, ReassignRecipientInput{
		TenantID: "bank", RequestID: request.ID, ActorPrincipalID: "creator",
		Recipient: RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: "actor-b"},
		Reason:    "Operations now owns the response.", ExpectedVersion: wrong.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !RequestAssignedTo(reassigned, "actor-b") || reassigned.Recipient.Revision != request.Recipient.Revision+1 || reassigned.Version != wrong.Version+1 {
		t.Fatalf("reassignment did not converge canonical recipient truth: %#v", reassigned)
	}
	newMine, err := service.ListVisibleRequests(ctx, "bank", "actor-b", 10, func(Request) bool { return true })
	if err != nil || len(newMine) != 1 || newMine[0].ID != request.ID {
		t.Fatalf("reassigned request did not move to new recipient: %#v err=%v", newMine, err)
	}
}

func TestExternalRecipientChangeRevokesExistingCapability(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository(nil, nil)
	service := NewService(repo, nil)
	now := time.Date(2026, 8, 9, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	request, err := service.CreateRequest(ctx, CreateRequestInput{
		TenantID: "bank", SubjectType: "CONTROL", SubjectID: "control-1",
		Title: "External confirmation", Purpose: "Collect one external fact.", WhyYou: "You are the intended respondent.",
		Sensitivity: "CONFIDENTIAL", AudienceType: "CUSTOMER",
		Recipient:        RecipientInput{Type: RecipientExternalAudience, Audience: "first@example.com"},
		EstimatedMinutes: 2, Deadline: now.Add(2 * time.Hour),
		Fields: []Field{{ID: "confirm", Label: "Confirm", Type: "text", Required: true}}, CreatedBy: "creator",
	})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := service.IssueInvitation(ctx, IssueInvitationInput{
		TenantID: "bank", RequestID: request.ID, Audience: "first@example.com", Purpose: "Respond", TTLMinutes: 30, CreatedBy: "creator",
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.RedeemInvitation(ctx, issued.Token, "first@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.SessionRequest(ctx, session.SessionToken); err != nil {
		t.Fatalf("fresh capability was invalid before reassignment: %v", err)
	}

	reassigned, err := service.ReassignRecipient(ctx, ReassignRecipientInput{
		TenantID: "bank", RequestID: request.ID, ActorPrincipalID: "creator",
		Recipient: RecipientInput{Type: RecipientExternalAudience, Audience: "second@example.com"},
		Reason:    "Customer contact was corrected.", ExpectedVersion: request.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reassigned.Recipient.AudienceHint == request.Recipient.AudienceHint || reassigned.Recipient.Revision != request.Recipient.Revision+1 {
		t.Fatalf("external recipient did not change canonically: %#v", reassigned.Recipient)
	}
	if _, _, err := service.SessionRequest(ctx, session.SessionToken); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("superseded external session survived recipient change: %v", err)
	}
	if _, err := service.RedeemInvitation(ctx, issued.Token, "first@example.com"); !errors.Is(err, ErrInvitationInvalid) {
		t.Fatalf("superseded invitation survived recipient change: %v", err)
	}
}
