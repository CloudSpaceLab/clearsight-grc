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

func TestEvidenceRecipientLifecycleIsAuditableAtomicAndRevokesSupersededCapabilities(t *testing.T) {
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

	const (
		tenantID   = "96666666-6666-7666-8666-666666666661"
		creatorID  = "96666666-6666-7666-8666-666666666662"
		recipientA = "96666666-6666-7666-8666-666666666663"
		recipientB = "96666666-6666-7666-8666-666666666664"
	)
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	mustExecRecipient(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'recipient-lifecycle-test','Recipient Lifecycle Test')`, tenantID)
	mustExecRecipient(t, ctx, pool, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$4::uuid,'PERSON','Requester','ACTIVE',$5),
		($2::uuid,$4::uuid,'PERSON','Recipient A','ACTIVE',$5),
		($3::uuid,$4::uuid,'PERSON','Recipient B','ACTIVE',$5)`,
		creatorID, recipientA, recipientB, tenantID, now.Add(-24*time.Hour))

	repo := NewPostgresRepository(pool)
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }

	internal, err := service.CreateRequest(ctx, CreateRequestInput{
		TenantID: "recipient-lifecycle-test", SubjectType: "CONTROL", SubjectID: "control-1",
		Title: "Confirm current control owner", Purpose: "Collect one current control fact.", WhyYou: "You are the intended respondent.",
		Sensitivity: "INTERNAL", AudienceType: "INTERNAL",
		Recipient:        RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: recipientA},
		EstimatedMinutes: 2, Deadline: now.Add(2 * time.Hour),
		Fields: []Field{{ID: "confirm", Label: "Confirm", Type: "text", Required: true}}, CreatedBy: creatorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	wrong, err := service.DeclareWrongRecipient(ctx, DeclareWrongRecipientInput{
		TenantID: "recipient-lifecycle-test", RequestID: internal.ID, ActorPrincipalID: recipientA,
		Reason: "This control moved to another owner.", ExpectedVersion: internal.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if wrong.Recipient.State != RecipientStateReassignmentRequired || wrong.Recipient.Revision != 1 || wrong.Version != 2 {
		t.Fatalf("wrong-recipient state was not persisted canonically: %#v", wrong)
	}
	visibleA, err := service.ListVisibleRequests(ctx, "recipient-lifecycle-test", recipientA, 20, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(visibleA) != 0 {
		t.Fatalf("wrong-recipient request remained visible to old recipient: %#v", visibleA)
	}

	reassigned, err := service.ReassignRecipient(ctx, ReassignRecipientInput{
		TenantID: "recipient-lifecycle-test", RequestID: internal.ID, ActorPrincipalID: creatorID,
		Recipient: RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: recipientB},
		Reason:    "Responsibility transferred to Recipient B.", ExpectedVersion: wrong.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !RequestAssignedTo(reassigned, recipientB) || reassigned.Recipient.Revision != 2 || reassigned.Version != 3 {
		t.Fatalf("canonical reassignment did not converge: %#v", reassigned)
	}
	visibleB, err := service.ListVisibleRequests(ctx, "recipient-lifecycle-test", recipientB, 20, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(visibleB) != 1 || visibleB[0].ID != internal.ID {
		t.Fatalf("reassigned request did not move to new recipient: %#v", visibleB)
	}
	var historyCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_recipient_history WHERE request_id=$1::uuid`, internal.ID).Scan(&historyCount); err != nil {
		t.Fatal(err)
	}
	if historyCount != 2 {
		t.Fatalf("expected wrong-recipient + reassignment history, got %d rows", historyCount)
	}
	var eventTypes []string
	rows, err := pool.Query(ctx, `SELECT event_type FROM capture_recipient_history WHERE request_id=$1::uuid ORDER BY occurred_at,id`, internal.ID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var eventType string
		if err := rows.Scan(&eventType); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		eventTypes = append(eventTypes, eventType)
	}
	rows.Close()
	if len(eventTypes) != 2 || eventTypes[0] != "WRONG_RECIPIENT" || eventTypes[1] != "REASSIGNED" {
		t.Fatalf("unexpected recipient history order: %#v", eventTypes)
	}

	external, err := service.CreateRequest(ctx, CreateRequestInput{
		TenantID: "recipient-lifecycle-test", SubjectType: "CONTROL", SubjectID: "control-2",
		Title: "External confirmation", Purpose: "Collect one bounded external fact.", WhyYou: "You are the intended respondent.",
		Sensitivity: "CONFIDENTIAL", AudienceType: "CUSTOMER",
		Recipient:        RecipientInput{Type: RecipientExternalAudience, Audience: "first@example.com"},
		EstimatedMinutes: 2, Deadline: now.Add(2 * time.Hour),
		Fields: []Field{{ID: "confirm", Label: "Confirm", Type: "text", Required: true}}, CreatedBy: creatorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := service.IssueInvitation(ctx, IssueInvitationInput{
		TenantID: "recipient-lifecycle-test", RequestID: external.ID, Audience: "first@example.com",
		Purpose: "Respond", TTLMinutes: 30, CreatedBy: creatorID,
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := service.RedeemInvitation(ctx, issued.Token, "first@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.SessionRequest(ctx, session.SessionToken); err != nil {
		t.Fatalf("fresh external session was not usable: %v", err)
	}

	reassignedExternal, err := service.ReassignRecipient(ctx, ReassignRecipientInput{
		TenantID: "recipient-lifecycle-test", RequestID: external.ID, ActorPrincipalID: creatorID,
		Recipient: RecipientInput{Type: RecipientExternalAudience, Audience: "second@example.com"},
		Reason:    "Correct the intended customer contact.", ExpectedVersion: external.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reassignedExternal.Recipient.Revision != 2 || reassignedExternal.Recipient.AudienceHint != "s***@example.com" {
		t.Fatalf("external recipient replacement was not canonical: %#v", reassignedExternal.Recipient)
	}
	if _, _, err := service.SessionRequest(ctx, session.SessionToken); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("superseded session survived recipient transaction: %v", err)
	}
	if _, err := service.RedeemInvitation(ctx, issued.Token, "first@example.com"); !errors.Is(err, ErrInvitationInvalid) {
		t.Fatalf("superseded invitation survived recipient transaction: %v", err)
	}
	var activeInvitations, activeSessions int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_invitations WHERE request_id=$1::uuid AND revoked_at IS NULL`, external.ID).Scan(&activeInvitations); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_sessions WHERE request_id=$1::uuid AND revoked_at IS NULL`, external.ID).Scan(&activeSessions); err != nil {
		t.Fatal(err)
	}
	if activeInvitations != 0 || activeSessions != 0 {
		t.Fatalf("capability revocation was not atomic invitations=%d sessions=%d", activeInvitations, activeSessions)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_recipient_history WHERE request_id=$1::uuid AND event_type='REASSIGNED'`, external.ID).Scan(&historyCount); err != nil {
		t.Fatal(err)
	}
	if historyCount != 1 {
		t.Fatalf("external recipient replacement did not append one audit row: %d", historyCount)
	}
}
