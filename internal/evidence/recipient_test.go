package evidence

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInternalRecipientOwnsQueueAndAuthenticatedSubmission(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository(nil, nil)
	service := NewService(repo, nil)
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	request, err := service.CreateRequest(ctx, CreateRequestInput{
		TenantID:         "bank",
		SubjectType:      "CONTROL",
		SubjectID:        "control-1",
		Title:            "Confirm control owner",
		Purpose:          "Close the current ownership fact.",
		WhyYou:           "You own this control.",
		Sensitivity:      "INTERNAL",
		AudienceType:     "INTERNAL",
		Recipient:        RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: "actor-a"},
		EstimatedMinutes: 2,
		Deadline:         now.Add(time.Hour),
		Fields:           []Field{{ID: "owner", Label: "Owner", Type: "text", Required: true}},
		CreatedBy:        "creator",
	})
	if err != nil {
		t.Fatal(err)
	}

	mine, err := service.ListVisibleRequests(ctx, "bank", "actor-a", 10, func(Request) bool { return true })
	if err != nil || len(mine) != 1 || mine[0].ID != request.ID {
		t.Fatalf("assigned request missing from recipient queue: %#v err=%v", mine, err)
	}
	other, err := service.ListVisibleRequests(ctx, "bank", "actor-b", 10, func(Request) bool { return true })
	if err != nil || len(other) != 0 {
		t.Fatalf("readable request leaked into another actor queue: %#v err=%v", other, err)
	}

	_, err = service.Submit(ctx, Submission{TenantID: "bank", RequestID: request.ID, SubmittedBy: "actor-b", Channel: "MAGIC_LINK", Answers: map[string]string{"owner": "Operations"}, ExpectedVersion: 1})
	if !errors.Is(err, ErrRecipientMismatch) {
		t.Fatalf("authenticated non-recipient bypassed assignment with channel spoof: %v", err)
	}

	receipt, err := service.Submit(ctx, Submission{TenantID: "bank", RequestID: request.ID, SubmittedBy: "actor-a", Channel: "INTERNAL", Answers: map[string]string{"owner": "Operations"}, ExpectedVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != RequestSubmitted {
		t.Fatalf("assigned recipient submission did not complete request: %#v", receipt)
	}
}

func TestExternalInvitationMustMatchCanonicalAudience(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository(nil, nil)
	service := NewService(repo, nil)
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	request, err := service.CreateRequest(ctx, CreateRequestInput{
		TenantID:         "bank",
		SubjectType:      "MATTER",
		SubjectID:        "matter-1",
		Title:            "Confirm customer statement",
		Purpose:          "Collect one bounded external fact.",
		WhyYou:           "You are the intended external respondent.",
		Sensitivity:      "CONFIDENTIAL",
		AudienceType:     "CUSTOMER",
		Recipient:        RecipientInput{Type: RecipientExternalAudience, Audience: "Customer@example.com"},
		EstimatedMinutes: 2,
		Deadline:         now.Add(2 * time.Hour),
		Fields:           []Field{{ID: "confirm", Label: "Confirm", Type: "text", Required: true}},
		CreatedBy:        "creator",
	})
	if err != nil {
		t.Fatal(err)
	}
	if request.Recipient.AudienceHint != "c***@example.com" || len(request.Recipient.AudienceHash) != 32 {
		t.Fatalf("external recipient was not normalized and masked: %#v", request.Recipient)
	}

	_, err = service.IssueInvitation(ctx, IssueInvitationInput{TenantID: "bank", RequestID: request.ID, Audience: "other@example.com", Purpose: "Respond", TTLMinutes: 30, CreatedBy: "creator"})
	if !errors.Is(err, ErrRecipientMismatch) {
		t.Fatalf("invitation audience was allowed to drift from canonical request recipient: %v", err)
	}
	issued, err := service.IssueInvitation(ctx, IssueInvitationInput{TenantID: "bank", RequestID: request.ID, Audience: "customer@example.com", Purpose: "Respond", TTLMinutes: 30, CreatedBy: "creator"})
	if err != nil {
		t.Fatal(err)
	}
	if issued.AudienceHint != request.Recipient.AudienceHint || issued.Token == "" {
		t.Fatalf("canonical external invitation was not issued correctly: %#v", issued)
	}
}

func TestLegacyUnassignedRequestDoesNotBecomeActorWork(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository(nil, nil)
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	_, err := repo.CreateRequest(ctx, Request{
		ID:               "legacy-request",
		TenantID:         "bank",
		SubjectType:      "CONTROL",
		SubjectID:        "control-legacy",
		Title:            "Legacy request",
		Purpose:          "Historical request without recipient truth.",
		WhyYou:           "Legacy descriptive copy.",
		Sensitivity:      "INTERNAL",
		AudienceType:     "INTERNAL",
		EstimatedMinutes: 2,
		Deadline:         now.Add(time.Hour),
		Fields:           []Field{{ID: "value", Label: "Value", Type: "text", Required: true}},
		Status:           RequestReady,
		Version:          1,
		CreatedAt:        now,
		UpdatedAt:        now,
		CreatedBy:        "creator",
	})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }
	visible, err := service.ListVisibleRequests(ctx, "bank", "actor-a", 10, func(Request) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	if len(visible) != 0 {
		t.Fatalf("legacy request was inferred into actor work: %#v", visible)
	}
	legacy, err := service.GetRequest(ctx, "bank", "legacy-request")
	if err != nil {
		t.Fatal(err)
	}
	if !RequestManageableBy(legacy, "creator") || RequestAssignedTo(legacy, "creator") {
		t.Fatalf("legacy request management/work semantics are wrong: %#v", legacy)
	}
}
