package evidence

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

func TestRequestSubmissionAndVersioning(t *testing.T) {
	service := NewService(NewMemoryRepository(nil, nil), NewMemoryObjectStore())
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	request, err := service.CreateRequest(context.Background(), CreateRequestInput{
		TenantID: "bank", SubjectType: "CONTROL", SubjectID: "c1", Title: "Confirm control operation",
		Purpose: "Complete the quarterly review.", WhyYou: "You operate the control.", Sensitivity: "INTERNAL", AudienceType: "INTERNAL",
		Recipient: RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: testCaptureRecipient}, EstimatedMinutes: 2,
		Deadline: now.Add(time.Hour), Fields: []Field{{ID: "state", Label: "Current state", Type: "single_select", Required: true, Options: []string{"Operating", "Unavailable"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Submit(context.Background(), testSubmission(request, map[string]string{})); err == nil {
		t.Fatal("expected required-field validation")
	}
	receipt, err := service.Submit(context.Background(), testSubmission(request, map[string]string{"state": "Operating"}))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != RequestSubmitted || receipt.Version != 2 {
		t.Fatalf("unexpected receipt: %#v", receipt)
	}
	if _, err := service.Submit(context.Background(), testSubmission(request, map[string]string{"state": "Operating"})); !errors.Is(err, ErrRequestClosed) {
		t.Fatalf("expected closed request, got %v", err)
	}
}

func TestCreateRequestRejectsPastDeadline(t *testing.T) {
	service := NewService(NewMemoryRepository(nil, nil), NewMemoryObjectStore())
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	_, err := service.CreateRequest(context.Background(), CreateRequestInput{
		TenantID: "bank", SubjectType: "CONTROL", SubjectID: "c1", Title: "Expired request",
		Purpose: "Test deadline enforcement.", WhyYou: "You own the control.", Sensitivity: "INTERNAL", AudienceType: "INTERNAL",
		Recipient: RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: testCaptureRecipient}, EstimatedMinutes: 2,
		Deadline: now.Add(-time.Minute), Fields: []Field{{ID: "note", Label: "Note", Type: "text"}},
	})
	if err == nil {
		t.Fatal("expected past deadline to be rejected")
	}
}

func TestExpiredRequestRejectsSubmissionAndArtifactAndPersistsExpiry(t *testing.T) {
	repo := NewMemoryRepository(nil, nil)
	service := NewService(repo, NewMemoryObjectStore())
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	request, err := service.CreateRequest(context.Background(), CreateRequestInput{
		TenantID: "bank", SubjectType: "CONTROL", SubjectID: "c1", Title: "Short request",
		Purpose: "Test expiry.", WhyYou: "You own the control.", Sensitivity: "INTERNAL", AudienceType: "INTERNAL",
		Recipient: RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: testCaptureRecipient}, EstimatedMinutes: 2,
		Deadline: now.Add(time.Minute), Fields: []Field{{ID: "note", Label: "Note", Type: "text"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	if _, err := service.Submit(context.Background(), testSubmission(request, map[string]string{"note": "late"})); !errors.Is(err, ErrRequestClosed) {
		t.Fatalf("late submission should be closed, got %v", err)
	}
	if _, err := service.StoreArtifact(context.Background(), ArtifactInput{TenantID: "bank", RequestID: request.ID, FileName: "proof.txt", MediaType: "text/plain", CreatedBy: testCaptureRecipient}, bytes.NewBufferString("evidence")); !errors.Is(err, ErrRequestClosed) {
		t.Fatalf("late artifact should be closed, got %v", err)
	}
	count, err := service.Maintain(context.Background(), now, 10)
	if err != nil || count != 1 {
		t.Fatalf("expire maintenance count=%d err=%v", count, err)
	}
	persisted, err := repo.GetRequest(context.Background(), "bank", request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != RequestExpired || persisted.Version != request.Version+1 {
		t.Fatalf("expiry was not persisted: %#v", persisted)
	}
}

func TestInvitationRedeemIsAudienceBoundOneTimeAndNonDestructiveOnMismatch(t *testing.T) {
	service := NewService(NewMemoryRepository(nil, nil), NewMemoryObjectStore())
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	request, err := service.CreateRequest(context.Background(), CreateRequestInput{
		TenantID: "bank", SubjectType: "VENDOR", SubjectID: "v1", Title: "Provide current certificate",
		Purpose: "Complete vendor assurance.", WhyYou: "You are the vendor contact.", Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR",
		Recipient: RecipientInput{Type: RecipientExternalAudience, Audience: "security@example.com"}, EstimatedMinutes: 3,
		Deadline: now.Add(24 * time.Hour), Fields: []Field{{ID: "confirm", Label: "Certificate is current", Type: "single_select", Required: true, Options: []string{"Yes", "No"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := service.IssueInvitation(context.Background(), IssueInvitationInput{TenantID: "bank", RequestID: request.ID, Audience: "security@example.com", Purpose: "Vendor assurance response", TTLMinutes: 60})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RedeemInvitation(context.Background(), issued.Token, "other@example.com"); !errors.Is(err, ErrInvitationInvalid) {
		t.Fatalf("expected audience mismatch rejection, got %v", err)
	}
	session, err := service.RedeemInvitation(context.Background(), issued.Token, " Security@Example.com ")
	if err != nil {
		t.Fatal(err)
	}
	if session.SessionToken == "" || session.AudienceHint != "s***@example.com" {
		t.Fatalf("unexpected session: %#v", session)
	}
	if _, err := service.RedeemInvitation(context.Background(), issued.Token, "security@example.com"); !errors.Is(err, ErrInvitationInvalid) {
		t.Fatalf("expected replay rejection, got %v", err)
	}
	_, loaded, err := service.SessionRequest(context.Background(), session.SessionToken)
	if err != nil || loaded.ID != request.ID {
		t.Fatalf("session request: %#v %v", loaded, err)
	}
}

func TestArtifactManifestAndSizeLimit(t *testing.T) {
	store := NewMemoryObjectStore()
	service := NewService(NewMemoryRepository(nil, nil), store)
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	service.Configure(time.Minute, 8)
	request, err := service.CreateRequest(context.Background(), CreateRequestInput{
		TenantID: "bank", SubjectType: "CONTROL", SubjectID: "c1", Title: "Upload evidence",
		Purpose: "Record current proof.", WhyYou: "You own the control.", Sensitivity: "INTERNAL", AudienceType: "INTERNAL",
		Recipient: RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: testCaptureRecipient}, EstimatedMinutes: 2,
		Deadline: now.Add(time.Hour), Fields: []Field{{ID: "note", Label: "Note", Type: "text"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := service.StoreArtifact(context.Background(), ArtifactInput{TenantID: "bank", RequestID: request.ID, FileName: "proof.txt", MediaType: "text/plain", CreatedBy: testCaptureRecipient}, bytes.NewBufferString("evidence"))
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Status != ArtifactStoredUnscanned || artifact.SHA256 == "" || artifact.SizeBytes != 8 {
		t.Fatalf("unexpected artifact: %#v", artifact)
	}
	if _, err := service.StoreArtifact(context.Background(), ArtifactInput{TenantID: "bank", RequestID: request.ID, FileName: "large.txt", MediaType: "text/plain", CreatedBy: testCaptureRecipient}, bytes.NewBufferString("too-large")); !errors.Is(err, ErrArtifactTooLarge) {
		t.Fatalf("expected size rejection, got %v", err)
	}
}

func TestSourceFreshnessMaintenance(t *testing.T) {
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	last := now.Add(-2 * time.Hour)
	repo := NewMemoryRepository([]Source{{ID: "s1", TenantID: "bank", Code: "IAM", Name: "IAM", Type: SourceSystem, AuthorityClass: "SYSTEM_OF_RECORD", ExpectedFreshnessMinutes: 30, LastSuccessAt: &last, Health: HealthCurrent, Status: SourceActive, Version: 1}}, nil)
	service := NewService(repo, NewMemoryObjectStore())
	count, err := service.Maintain(context.Background(), now, 10)
	if err != nil || count != 1 {
		t.Fatalf("maintain count=%d err=%v", count, err)
	}
	sources, _ := service.ListSources(context.Background(), "bank", 10)
	if sources[0].Health != HealthStale {
		t.Fatalf("expected stale source: %#v", sources[0])
	}
}
