package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestMemoryRepositoryOriginAndSessionDraftLifecycle(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository(nil, nil)
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }
	origin := RequestOrigin{Type: "THIRD_PARTY_ASSESSMENT", ID: "assessment-1", Version: 1}
	request, err := service.CreateRequest(WithRequestOriginAuthority(ctx, origin.Type), CreateRequestInput{
		TenantID: "bank-a", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "relationship-1",
		Title: "Complete due diligence", Purpose: "Collect one vendor fact.", WhyYou: "You are the invited vendor contact.",
		Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR", Recipient: RecipientInput{Type: RecipientExternalAudience, Audience: "security@vendor.example"},
		EstimatedMinutes: 2, Deadline: now.Add(time.Hour), Origin: origin,
		Fields: []Field{{ID: "contact", Label: "Contact", Type: string(formcontract.TypeEmail), Required: true}}, CreatedBy: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	byOrigin, err := service.GetRequestByOrigin(ctx, "bank-a", origin)
	if err != nil || byOrigin.ID != request.ID {
		t.Fatalf("origin request = %#v, err = %v", byOrigin, err)
	}
	if _, err := service.GetRequestByOrigin(ctx, "bank-b", origin); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant origin error = %v", err)
	}

	issued, err := service.IssueInvitation(ctx, IssueInvitationInput{TenantID: "bank-a", RequestID: request.ID, Audience: "security@vendor.example", Purpose: "Respond", TTL: 30 * time.Minute, CreatedBy: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	redeemed, err := service.RedeemInvitation(ctx, issued.Token, "security@vendor.example")
	if err != nil {
		t.Fatal(err)
	}
	contact := "security@vendor.example"
	answers := map[string]formcontract.AnswerValue{"contact": {Text: &contact}}
	first, err := service.SaveDraft(ctx, redeemed.SessionToken, SaveDraftInput{Answers: answers, PresentationMode: formcontract.PresentationWizard, ExpectedVersion: 0})
	if err != nil || first.Version != 1 {
		t.Fatalf("first draft = %#v, err = %v", first, err)
	}
	if _, err := service.SaveDraft(ctx, redeemed.SessionToken, SaveDraftInput{Answers: answers, PresentationMode: formcontract.PresentationClassic, ExpectedVersion: 0}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale draft error = %v", err)
	}
	second, err := service.SaveDraft(ctx, redeemed.SessionToken, SaveDraftInput{Answers: answers, PresentationMode: formcontract.PresentationClassic, ExpectedVersion: 1})
	if err != nil || second.Version != 2 || second.PresentationMode != formcontract.PresentationClassic {
		t.Fatalf("updated draft = %#v, err = %v", second, err)
	}
	loaded, err := service.GetDraft(ctx, redeemed.SessionToken)
	if err != nil || loaded.Version != 2 {
		t.Fatalf("loaded draft = %#v, err = %v", loaded, err)
	}
	if _, err := service.SubmitSession(ctx, redeemed.SessionToken, answers, request.Version); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetDraft(ctx, "bank-a", request.ID, redeemed.SessionID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("submitted draft error = %v", err)
	}
}

func TestMemoryDraftAccessEndsWithSession(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository(nil, nil)
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }
	request, err := service.CreateRequest(ctx, CreateRequestInput{
		TenantID: "bank-a", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "relationship-2",
		Title: "Complete due diligence", Purpose: "Collect one vendor fact.", WhyYou: "You are the invited vendor contact.",
		Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR", Recipient: RecipientInput{Type: RecipientExternalAudience, Audience: "vendor@example.com"},
		EstimatedMinutes: 2, Deadline: now.Add(time.Hour), Fields: []Field{{ID: "answer", Label: "Answer", Type: string(formcontract.TypeShortText)}}, CreatedBy: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := service.IssueInvitation(ctx, IssueInvitationInput{TenantID: "bank-a", RequestID: request.ID, Audience: "vendor@example.com", Purpose: "Respond", TTL: 30 * time.Minute, CreatedBy: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	redeemed, err := service.RedeemInvitation(ctx, issued.Token, "vendor@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveDraft(ctx, redeemed.SessionToken, SaveDraftInput{Answers: map[string]formcontract.AnswerValue{}, PresentationMode: formcontract.PresentationClassic}); err != nil {
		t.Fatal(err)
	}
	if err := service.RevokeSessionAsRequester(ctx, RevokeSessionAsRequesterInput{TenantID: "bank-a", RequestID: request.ID, SessionID: redeemed.SessionID, ActorPrincipalID: "owner"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetDraft(ctx, redeemed.SessionToken); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("revoked draft read error = %v", err)
	}
}

func TestSessionDraftLoadsEmptyAndAcceptsIncompleteValidAnswers(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository(nil, nil)
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }
	token := createDraftSession(t, ctx, service, now, []Field{
		{ID: "contact", Label: "Security contact", Type: string(formcontract.TypeEmail), Required: true},
		{ID: "notes", Label: "Review notes", Type: string(formcontract.TypeLongText)},
	})

	empty, err := service.GetDraft(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if empty.Version != 0 || len(empty.Answers) != 0 || empty.PresentationMode != formcontract.PresentationAutomatic {
		t.Fatalf("empty draft = %#v", empty)
	}

	value := "Work is in progress"
	saved, err := service.SaveDraft(ctx, token, SaveDraftInput{
		Answers: map[string]formcontract.AnswerValue{"notes": {Text: &value}}, PresentationMode: " wizard ", ExpectedVersion: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Version != 1 || saved.PresentationMode != formcontract.PresentationWizard {
		t.Fatalf("saved draft = %#v", saved)
	}
}

func TestSessionDraftRejectsInvalidOrHiddenAnswers(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository(nil, nil)
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }
	token := createDraftSession(t, ctx, service, now, []Field{
		{ID: "handles_data", Label: "Handles personal data", Type: string(formcontract.TypeYesNo), Options: []string{"Yes", "No"}},
		{ID: "contact", Label: "Security contact", Type: string(formcontract.TypeEmail), Condition: &formcontract.VisibilityCondition{FieldID: "handles_data", Operator: formcontract.ConditionEquals, Values: []string{"Yes"}}},
	})

	no := "No"
	invalidEmail := "not-an-email"
	tests := []map[string]formcontract.AnswerValue{
		{"unknown": formcontract.TextAnswer("value")},
		{"handles_data": {Text: &no}, "contact": formcontract.TextAnswer("security@vendor.example")},
		{"handles_data": formcontract.TextAnswer("Yes"), "contact": {Text: &invalidEmail}},
	}
	for _, answers := range tests {
		if _, err := service.SaveDraft(ctx, token, SaveDraftInput{Answers: answers, PresentationMode: formcontract.PresentationClassic}); !errors.Is(err, ErrDraftInvalid) {
			t.Fatalf("answers %#v error = %v", answers, err)
		}
	}
}

func createDraftSession(t *testing.T, ctx context.Context, service *Service, now time.Time, fields []Field) string {
	t.Helper()
	request, err := service.CreateRequest(ctx, CreateRequestInput{
		TenantID: "bank-a", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "relationship-draft",
		Title: "Complete due diligence", Purpose: "Collect the requested vendor information.", WhyYou: "You are the invited vendor contact.",
		Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR", Recipient: RecipientInput{Type: RecipientExternalAudience, Audience: "vendor@example.com"},
		EstimatedMinutes: 5, Deadline: now.Add(time.Hour), Fields: fields, CreatedBy: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := service.IssueInvitation(ctx, IssueInvitationInput{TenantID: "bank-a", RequestID: request.ID, Audience: "vendor@example.com", Purpose: "Complete due diligence", TTL: 30 * time.Minute, CreatedBy: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	redeemed, err := service.RedeemInvitation(ctx, issued.Token, "vendor@example.com")
	if err != nil {
		t.Fatal(err)
	}
	return redeemed.SessionToken
}
