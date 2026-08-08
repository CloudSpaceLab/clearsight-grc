//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestEvidenceRecipientTruthIsTenantBoundPreLimitAndAudienceBound(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const tenantID = "99999999-8888-7888-8888-888888888881"
	const requesterID = "99999999-8888-7888-8888-888888888882"
	const recipientA = "99999999-8888-7888-8888-888888888883"
	const recipientB = "99999999-8888-7888-8888-888888888884"
	const restrictedMatterID = "99999999-8888-7888-8888-888888888885"
	const legacyRequestID = "99999999-8888-7888-8888-888888888886"
	now := time.Date(2026, 8, 8, 14, 0, 0, 0, time.UTC)

	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	})
	mustExecRecipient(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'recipient-truth-test','Recipient Truth Test')`, tenantID)
	mustExecRecipient(t, ctx, pool, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$4::uuid,'PERSON','Requester','ACTIVE',$5),
		($2::uuid,$4::uuid,'PERSON','Recipient A','ACTIVE',$5),
		($3::uuid,$4::uuid,'PERSON','Recipient B','ACTIVE',$5)`, requesterID, recipientA, recipientB, tenantID, now.Add(-24*time.Hour))
	mustExecRecipient(t, ctx, pool, `INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,known_facts,missing_facts,contradictions,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,'MAT-RECIPIENT-1','CONTROL_GAP','TRIAGE',3,'Restricted control gap','Only Recipient A may read this Matter',$3::jsonb,'{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$4,$4)`,
		restrictedMatterID, tenantID, `{"access":"RESTRICTED","allowed_principal_ids":["`+requesterID+`","`+recipientA+`"]}`, now.Add(-time.Hour))

	repo := NewPostgresRepository(pool)
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }

	// Other-recipient requests have earlier deadlines. With limit=1 they must
	// not crowd the current actor's own request out of the queue.
	for i := 0; i < 3; i++ {
		_, err := service.CreateRequest(ctx, recipientRequestInput("recipient-truth-test", "CONTROL", "control-b", recipientB, requesterID, now.Add(time.Duration(i+1)*time.Minute)))
		if err != nil {
			t.Fatal(err)
		}
	}
	assigned, err := service.CreateRequest(ctx, recipientRequestInput("recipient-truth-test", "CONTROL", "control-a", recipientA, requesterID, now.Add(time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	visible, err := service.ListVisibleRequests(ctx, "recipient-truth-test", recipientA, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(visible) != 1 || visible[0].ID != assigned.ID {
		t.Fatalf("recipient filtering happened after LIMIT or leaked another recipient: %#v", visible)
	}

	// A readable request description is not enough: Recipient B cannot be
	// assigned a restricted Matter they cannot read.
	_, err = service.CreateRequest(ctx, recipientRequestInput("recipient-truth-test", "MATTER", restrictedMatterID, recipientB, requesterID, now.Add(2*time.Hour)))
	if !errors.Is(err, ErrRecipientInvalid) {
		t.Fatalf("restricted Matter was assigned to a principal without access: %v", err)
	}

	// Existing historical rows are deliberately left unassigned. Even with an
	// earlier deadline and readable subject they must not become actor work.
	mustExecRecipient(t, ctx, pool, `INSERT INTO capture_requests(id,tenant_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,estimated_minutes,deadline,known_facts,fields,status,created_by,version,created_at,updated_at)
		VALUES($1::uuid,$2::uuid,'CONTROL','legacy-control','Legacy request','Historical','Old descriptive copy','INTERNAL','INTERNAL',2,$3,'{}'::jsonb,'[{"id":"value","label":"Value","type":"text","required":true}]'::jsonb,'READY',$4::uuid,1,$5,$5)`,
		legacyRequestID, tenantID, now.Add(30*time.Second), requesterID, now.Add(-time.Hour))
	visible, err = service.ListVisibleRequests(ctx, "recipient-truth-test", recipientA, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(visible) != 1 || visible[0].ID != assigned.ID {
		t.Fatalf("legacy null-recipient request was inferred into actor work: %#v", visible)
	}

	// External audience identity is request state, not invitation copy. The raw
	// address is not stored on capture_requests and only the canonical requester
	// may issue an invitation for the matching audience.
	external, err := service.CreateRequest(ctx, CreateRequestInput{
		TenantID:         "recipient-truth-test",
		SubjectType:      "MATTER",
		SubjectID:        restrictedMatterID,
		Title:            "Confirm external statement",
		Purpose:          "Collect one bounded external fact.",
		WhyYou:           "You are the intended respondent.",
		Sensitivity:      "CONFIDENTIAL",
		AudienceType:     "INVITED_EXTERNAL",
		Recipient:        RecipientInput{Type: RecipientExternalAudience, Audience: "Customer@example.com"},
		EstimatedMinutes: 2,
		Deadline:         now.Add(3 * time.Hour),
		Fields:           []Field{{ID: "confirm", Label: "Confirm", Type: "text", Required: true}},
		CreatedBy:        requesterID,
	})
	if err != nil {
		t.Fatal(err)
	}
	var hashLength int
	var hint string
	var containsRaw bool
	if err := pool.QueryRow(ctx, `SELECT octet_length(recipient_audience_hash),recipient_hint,to_jsonb(capture_requests)::text ILIKE '%customer@example.com%' FROM capture_requests WHERE id=$1::uuid`, external.ID).Scan(&hashLength, &hint, &containsRaw); err != nil {
		t.Fatal(err)
	}
	if hashLength != 32 || hint != "c***@example.com" || containsRaw {
		t.Fatalf("external audience persistence leaked or weakened recipient identity: hash=%d hint=%q raw=%v", hashLength, hint, containsRaw)
	}
	_, err = service.IssueInvitation(ctx, IssueInvitationInput{TenantID: "recipient-truth-test", RequestID: external.ID, Audience: "customer@example.com", Purpose: "Respond", TTLMinutes: 30, CreatedBy: recipientA})
	if !errors.Is(err, ErrRecipientMismatch) {
		t.Fatalf("non-requester issued external invitation: %v", err)
	}
	_, err = service.IssueInvitation(ctx, IssueInvitationInput{TenantID: "recipient-truth-test", RequestID: external.ID, Audience: "other@example.com", Purpose: "Respond", TTLMinutes: 30, CreatedBy: requesterID})
	if !errors.Is(err, ErrRecipientMismatch) {
		t.Fatalf("requester changed invitation audience without changing request recipient: %v", err)
	}
	issued, err := service.IssueInvitation(ctx, IssueInvitationInput{TenantID: "recipient-truth-test", RequestID: external.ID, Audience: "customer@example.com", Purpose: "Respond", TTLMinutes: 30, CreatedBy: requesterID})
	if err != nil || issued.Token == "" {
		t.Fatalf("canonical external invitation failed: %#v err=%v", issued, err)
	}
}

func recipientRequestInput(tenant, subjectType, subjectID, recipient, requester string, deadline time.Time) CreateRequestInput {
	return CreateRequestInput{
		TenantID:         tenant,
		SubjectType:      subjectType,
		SubjectID:        subjectID,
		Title:            "Confirm evidence fact",
		Purpose:          "Collect one bounded fact.",
		WhyYou:           "You are the intended internal respondent.",
		Sensitivity:      "INTERNAL",
		AudienceType:     "INTERNAL",
		Recipient:        RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: recipient},
		EstimatedMinutes: 2,
		Deadline:         deadline,
		Fields:           []Field{{ID: "value", Label: "Value", Type: "text", Required: true}},
		CreatedBy:        requester,
	}
}

func mustExecRecipient(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}
