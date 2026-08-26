package evidence

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestIssuingReplacementInvitationRevokesPriorRequestAccess(t *testing.T) {
	service := NewService(NewMemoryRepository(nil, nil), NewMemoryObjectStore())
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	request, err := service.CreateRequest(context.Background(), CreateRequestInput{
		TenantID: "bank", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "relationship-1", Title: "Vendor due diligence",
		Purpose: "Provide the information required for this vendor review.", WhyYou: "Provide the requested information.",
		Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR", Recipient: RecipientInput{Type: RecipientExternalAudience, Audience: "security@vendor.example"},
		EstimatedMinutes: 5, Deadline: now.Add(24 * time.Hour), Fields: []Field{{ID: "confirm", Label: "Confirm details", Type: "yes_no", Required: true}}, CreatedBy: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.IssueInvitation(context.Background(), IssueInvitationInput{TenantID: "bank", RequestID: request.ID, Audience: "security@vendor.example", Purpose: "Respond", TTLMinutes: 60, CreatedBy: "owner-1"})
	if err != nil {
		t.Fatal(err)
	}
	priorSession, err := service.RedeemInvitation(context.Background(), first.Token, "security@vendor.example")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.IssueInvitation(context.Background(), IssueInvitationInput{TenantID: "bank", RequestID: request.ID, Audience: "security@vendor.example", Purpose: "Respond", TTLMinutes: 60, CreatedBy: "owner-1"})
	if err != nil {
		t.Fatal(err)
	}
	if first.InvitationID == second.InvitationID {
		t.Fatal("replacement reused the prior invitation record")
	}
	if _, err := service.RedeemInvitation(context.Background(), first.Token, "security@vendor.example"); !errors.Is(err, ErrInvitationInvalid) {
		t.Fatalf("prior invitation remained usable: %v", err)
	}
	if _, _, err := service.SessionRequest(context.Background(), priorSession.SessionToken); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("prior capture session remained usable: %v", err)
	}
	if _, err := service.RedeemInvitation(context.Background(), second.Token, "security@vendor.example"); err != nil {
		t.Fatalf("replacement invitation was unavailable: %v", err)
	}
}
