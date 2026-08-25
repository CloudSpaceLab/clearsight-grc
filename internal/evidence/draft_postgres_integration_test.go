//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRequestOriginAndResponseDraftPersistence(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const (
		tenantID    = "88888888-7777-7777-8777-777777777771"
		otherTenant = "88888888-7777-7777-8777-777777777772"
		requesterID = "88888888-7777-7777-8777-777777777773"
	)
	now := time.Date(2026, 8, 26, 11, 0, 0, 0, time.UTC)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id IN ($1::uuid,$2::uuid)`, tenantID, otherTenant)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id IN ($1::uuid,$2::uuid)`, tenantID, otherTenant)
	})
	mustExecDraft(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES
		($1::uuid,'draft-persistence-test','Draft Persistence Test'),
		($2::uuid,'draft-persistence-other','Draft Persistence Other')`, tenantID, otherTenant)
	mustExecDraft(t, ctx, pool, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from)
		VALUES($1::uuid,$2::uuid,'PERSON','Request owner','ACTIVE',$3)`, requesterID, tenantID, now.Add(-time.Hour))

	repo := NewPostgresRepository(pool)
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }
	origin := RequestOrigin{Type: "THIRD_PARTY_ASSESSMENT", ID: "assessment-42", Version: 1}
	request, err := service.CreateRequest(ctx, CreateRequestInput{
		TenantID: "draft-persistence-test", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "relationship-42",
		Title: "Complete vendor due diligence", Purpose: "Collect the vendor security response.", WhyYou: "You are the vendor security contact.",
		Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR", Recipient: RecipientInput{Type: RecipientExternalAudience, Audience: "security@vendor.example"},
		EstimatedMinutes: 10, Deadline: now.Add(24 * time.Hour), Origin: origin,
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard, AllowModeSwitch: true},
		Sections:     []formcontract.Section{{ID: "security", Title: "Security"}},
		Fields:       []Field{{ID: "security_contact", SectionID: "security", Label: "Security contact", Type: string(formcontract.TypeEmail), Required: true}},
		CreatedBy:    requesterID,
	})
	if err != nil {
		t.Fatalf("create origin request: %v", err)
	}
	stored, err := repo.GetRequestByOrigin(ctx, "draft-persistence-test", origin)
	if err != nil || stored.ID != request.ID || stored.Origin != origin {
		t.Fatalf("origin lookup = %#v, err = %v", stored, err)
	}
	if stored.Presentation.DefaultMode != formcontract.PresentationWizard || len(stored.Sections) != 1 || stored.Sections[0].ID != "security" {
		t.Fatalf("request form snapshot not retained: %#v", stored)
	}
	if _, err := repo.GetRequestByOrigin(ctx, "draft-persistence-other", origin); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant origin lookup error = %v, want not found", err)
	}
	if _, err := service.CreateRequest(ctx, CreateRequestInput{
		TenantID: "draft-persistence-test", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "relationship-42",
		Title: "Duplicate vendor due diligence", Purpose: "Must reuse the original request.", WhyYou: "You are the vendor security contact.",
		Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR", Recipient: RecipientInput{Type: RecipientExternalAudience, Audience: "security@vendor.example"},
		EstimatedMinutes: 10, Deadline: now.Add(24 * time.Hour), Origin: origin,
		Fields: []Field{{ID: "security_contact", Label: "Security contact", Type: string(formcontract.TypeEmail), Required: true}}, CreatedBy: requesterID,
	}); err == nil {
		t.Fatal("expected duplicate tenant origin to be rejected")
	}

	issued, err := service.IssueInvitation(ctx, IssueInvitationInput{TenantID: "draft-persistence-test", RequestID: request.ID, Audience: "security@vendor.example", Purpose: "Complete due diligence", TTL: 2 * time.Hour, CreatedBy: requesterID})
	if err != nil {
		t.Fatalf("issue invitation: %v", err)
	}
	redeemed, err := service.RedeemInvitation(ctx, issued.Token, "security@vendor.example")
	if err != nil {
		t.Fatalf("redeem invitation: %v", err)
	}
	contact := "security@vendor.example"
	answers := map[string]formcontract.AnswerValue{"security_contact": {Text: &contact}}
	first, err := service.SaveDraft(ctx, redeemed.SessionToken, SaveDraftInput{ExpectedVersion: 0, PresentationMode: formcontract.PresentationWizard, Answers: answers})
	if err != nil || first.Version != 1 {
		t.Fatalf("first draft = %#v, err = %v", first, err)
	}
	if _, err := service.SaveDraft(ctx, redeemed.SessionToken, SaveDraftInput{ExpectedVersion: 0, PresentationMode: formcontract.PresentationClassic, Answers: answers}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale draft write error = %v, want conflict", err)
	}
	loaded, err := service.GetDraft(ctx, redeemed.SessionToken)
	if err != nil || loaded.Version != 1 || loaded.PresentationMode != formcontract.PresentationWizard || loaded.Answers["security_contact"].Text == nil || *loaded.Answers["security_contact"].Text != contact {
		t.Fatalf("loaded draft = %#v, err = %v", loaded, err)
	}
	service.now = func() time.Time { return now.Add(3 * time.Hour) }
	if _, err := service.GetDraft(ctx, redeemed.SessionToken); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("expired session draft read error = %v, want session invalid", err)
	}
	service.now = func() time.Time { return now }
	if _, err := repo.SaveDraft(ctx, SaveDraftRecord{TenantID: otherTenant, RequestID: request.ID, SessionID: redeemed.SessionID, ExpectedVersion: 1, PresentationMode: formcontract.PresentationClassic, Answers: answers, UpdatedAt: now}); err == nil {
		t.Fatal("expected cross-tenant draft write to fail")
	}

	if err := service.RevokeSession(ctx, "draft-persistence-test", redeemed.SessionID); err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	if _, err := service.GetDraft(ctx, redeemed.SessionToken); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("revoked session draft read error = %v, want session invalid", err)
	}
}

func TestSubmissionDeletesResponseDraftInSameTransaction(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const (
		tenantID    = "77777777-6666-7666-8666-666666666661"
		requesterID = "77777777-6666-7666-8666-666666666662"
	)
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	mustExecDraft(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'draft-submit-test','Draft Submit Test')`, tenantID)
	mustExecDraft(t, ctx, pool, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Request owner','ACTIVE',$3)`, requesterID, tenantID, now.Add(-time.Hour))

	repo := NewPostgresRepository(pool)
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }
	request, err := service.CreateRequest(ctx, CreateRequestInput{
		TenantID: "draft-submit-test", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "relationship-submit",
		Title: "Submit vendor details", Purpose: "Collect one vendor fact.", WhyYou: "You are the invited vendor contact.",
		Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR", Recipient: RecipientInput{Type: RecipientExternalAudience, Audience: "submit@vendor.example"},
		EstimatedMinutes: 2, Deadline: now.Add(4 * time.Hour),
		Fields: []Field{{ID: "contact", Label: "Contact", Type: string(formcontract.TypeEmail), Required: true}}, CreatedBy: requesterID,
	})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := service.IssueInvitation(ctx, IssueInvitationInput{TenantID: "draft-submit-test", RequestID: request.ID, Audience: "submit@vendor.example", Purpose: "Respond", TTL: time.Hour, CreatedBy: requesterID})
	if err != nil {
		t.Fatal(err)
	}
	redeemed, err := service.RedeemInvitation(ctx, issued.Token, "submit@vendor.example")
	if err != nil {
		t.Fatal(err)
	}
	contact := "submit@vendor.example"
	answers := map[string]formcontract.AnswerValue{"contact": {Text: &contact}}
	if _, err := service.SaveDraft(ctx, redeemed.SessionToken, SaveDraftInput{ExpectedVersion: 0, PresentationMode: formcontract.PresentationClassic, Answers: answers}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SubmitSession(ctx, redeemed.SessionToken, answers, request.Version); err != nil {
		t.Fatalf("submit session: %v", err)
	}
	if _, err := repo.GetDraft(ctx, tenantID, request.ID, redeemed.SessionID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("draft after submission error = %v, want not found", err)
	}
}

func mustExecDraft(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}
