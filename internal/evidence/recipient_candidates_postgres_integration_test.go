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

var _ recipientCandidateRepository = (*PostgresRepository)(nil)
var _ internalRecipientLabelDirectory = (*PostgresRepository)(nil)

func TestPostgresRecipientCandidatesFilterEntitySubjectAndStatusBeforeLimit(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	const tenantID = "9e777777-7777-7777-8777-777777777701"
	const entityID = "9e777777-7777-7777-8777-777777777702"
	const otherEntityID = "9e777777-7777-7777-8777-777777777703"
	const programID = "9e777777-7777-7777-8777-777777777704"
	const requesterID = "9e777777-7777-7777-8777-777777777705"
	const eligibleID = "9e777777-7777-7777-8777-777777777706"
	const blockedID = "9e777777-7777-7777-8777-777777777707"
	const inactiveID = "9e777777-7777-7777-8777-777777777708"
	const otherEntityPrincipalID = "9e777777-7777-7777-8777-777777777709"
	const teamID = "97777777-7777-7777-8777-777777777710"
	const replacementID = "97777777-7777-7777-8777-777777777716"
	const invitationID = "97777777-7777-7777-8777-777777777717"
	const sessionID = "97777777-7777-7777-8777-777777777718"
	now := time.Now().UTC().Truncate(time.Second)

	cleanupRecipientCandidatesPostgres(t, ctx, pool, tenantID)
	t.Cleanup(func() { cleanupRecipientCandidatesPostgres(t, context.Background(), pool, tenantID) })
	mustExecRecipientCandidate(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'recipient-candidates-test','Recipient candidates test')`, tenantID)
	mustExecRecipientCandidate(t, ctx, pool, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
		($1::uuid,$3::uuid,'ENTITY-ONE','Entity One','NG',$4),($2::uuid,$3::uuid,'ENTITY-TWO','Entity Two','NG',$4)`, entityID, otherEntityID, tenantID, now.Add(-time.Hour))
	mustExecRecipientCandidate(t, ctx, pool, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$8::uuid,'PERSON','Requester','ACTIVE',$9),
		($2::uuid,$8::uuid,'PERSON','Zara Eligible','ACTIVE',$9),
		($3::uuid,$8::uuid,'PERSON','Ayo Blocked','ACTIVE',$9),
		($4::uuid,$8::uuid,'PERSON','Bisi Inactive','INACTIVE',$9),
		($5::uuid,$8::uuid,'PERSON','Chidi Other Entity','ACTIVE',$9),
		($6::uuid,$8::uuid,'TEAM','Controls Team','ACTIVE',$9),
		($7::uuid,$8::uuid,'PERSON','Bola Replacement','ACTIVE',$9)`, requesterID, eligibleID, blockedID, inactiveID, otherEntityPrincipalID, teamID, replacementID, tenantID, now.Add(-time.Hour))
	mustExecRecipientCandidate(t, ctx, pool, `INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from) VALUES
		('97777777-7777-7777-8777-777777777711'::uuid,$1::uuid,$2::uuid,'ELIGIBLE','Eligible',$3::uuid,$10),
		('97777777-7777-7777-8777-777777777712'::uuid,$1::uuid,$2::uuid,'BLOCKED','Blocked',$4::uuid,$10),
		('97777777-7777-7777-8777-777777777713'::uuid,$1::uuid,$2::uuid,'INACTIVE','Inactive',$5::uuid,$10),
		('97777777-7777-7777-8777-777777777714'::uuid,$1::uuid,$6::uuid,'OTHER','Other entity',$7::uuid,$10),
		('97777777-7777-7777-8777-777777777715'::uuid,$1::uuid,$2::uuid,'TEAM','Team',$8::uuid,$10),
		('97777777-7777-7777-8777-777777777719'::uuid,$1::uuid,$2::uuid,'REQUESTER','Requester',$9::uuid,$10),
		('97777777-7777-7777-8777-777777777720'::uuid,$1::uuid,$2::uuid,'REPLACEMENT','Replacement',$11::uuid,$10)`, tenantID, entityID, eligibleID, blockedID, inactiveID, otherEntityID, otherEntityPrincipalID, teamID, requesterID, now.Add(-time.Hour), replacementID)
	mustExecRecipientCandidate(t, ctx, pool, `INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,scope,effective_from) VALUES
		($1::uuid,$2::uuid,$3::uuid,'CANDIDATES','Candidate visibility','COMPLIANCE','ACTIVE','Compliance','NG',$4::jsonb,$5)`, programID, tenantID, entityID, `{"access":"RESTRICTED","allowed_principal_ids":["`+requesterID+`","`+eligibleID+`","`+replacementID+`","`+otherEntityPrincipalID+`"]}`, now.Add(-time.Hour))

	repo := NewPostgresRepository(pool)
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }
	created, err := service.CreateRequest(ctx, recipientRequestInput("recipient-candidates-test", entityID, "PROGRAM", programID, eligibleID, requesterID, now.Add(time.Hour)))
	if err != nil {
		t.Fatal(err)
	}

	page, err := service.SearchRecipientCandidates(ctx, ActorRequestScope{TenantID: "recipient-candidates-test", LegalEntityID: entityID, ActorPrincipalID: requesterID}, created.ID, RecipientCandidateSearch{Query: "eligible", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].PrincipalID != eligibleID || page.Items[0].DisplayName != "Zara Eligible" || page.Items[0].ContextLabel != "Eligible" || page.HasMore {
		t.Fatalf("candidate filters ran after limit or leaked scope: %#v", page)
	}
	loaded, err := service.GetRequestForEntity(ctx, "recipient-candidates-test", entityID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Recipient.DisplayName != "Zara Eligible" {
		t.Fatalf("current recipient label = %q", loaded.Recipient.DisplayName)
	}
	if err := repo.ReassignRecipient(ctx, ReassignRecipientInput{
		TenantID: "recipient-candidates-test", LegalEntityID: entityID, RequestID: created.ID, ActorPrincipalID: requesterID,
		Recipient: RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: otherEntityPrincipalID},
		Reason:    "Direct stale reassignment attempt.", ExpectedVersion: created.Version,
	}, Recipient{Type: RecipientInternalPrincipal, PrincipalID: otherEntityPrincipalID, State: RecipientStateAssigned}, now); !errors.Is(err, ErrRecipientInvalid) {
		t.Fatalf("transaction accepted a principal outside the request entity: %v", err)
	}
	mustExecRecipientCandidate(t, ctx, pool, `INSERT INTO capture_invitations(id,tenant_id,request_id,token_hash,audience_hash,audience_hint,purpose,expires_at,max_redemptions,created_by,created_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,'c***@example.com','Respond',$6,1,$7::uuid,$8)`, invitationID, tenantID, created.ID, []byte("candidate-invitation-token"), []byte("candidate-audience"), now.Add(time.Hour), requesterID, now)
	mustExecRecipientCandidate(t, ctx, pool, `INSERT INTO capture_sessions(id,tenant_id,request_id,invitation_id,token_hash,audience_hint,expires_at,created_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,'c***@example.com',$6,$7)`, sessionID, tenantID, created.ID, invitationID, []byte("candidate-session-token"), now.Add(time.Hour), now)
	mustExecRecipientCandidate(t, ctx, pool, `UPDATE org_positions SET valid_until=$2 WHERE occupant_principal_id=$1::uuid AND code='REQUESTER'`, requesterID, now.Add(-time.Minute))
	if err := repo.ReassignRecipient(ctx, ReassignRecipientInput{
		TenantID: "recipient-candidates-test", LegalEntityID: entityID, RequestID: created.ID, ActorPrincipalID: requesterID,
		Recipient: RecipientInput{Type: RecipientInternalPrincipal, PrincipalID: replacementID},
		Reason:    "Direct reassignment after requester authority expired.", ExpectedVersion: created.Version,
	}, Recipient{Type: RecipientInternalPrincipal, PrincipalID: replacementID, State: RecipientStateAssigned}, now); !errors.Is(err, ErrRecipientInvalid) {
		t.Fatalf("transaction accepted a requester with expired entity membership: %v", err)
	}
	var storedPrincipal string
	var storedVersion int64
	if err := pool.QueryRow(ctx, `SELECT recipient_principal_id::text,version FROM capture_requests WHERE id=$1::uuid`, created.ID).Scan(&storedPrincipal, &storedVersion); err != nil {
		t.Fatal(err)
	}
	if storedPrincipal != eligibleID || storedVersion != created.Version {
		t.Fatalf("rejected requester authority changed request principal=%s version=%d", storedPrincipal, storedVersion)
	}
	var historyCount, revokedInvitations, revokedSessions int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_recipient_history WHERE request_id=$1::uuid`, created.ID).Scan(&historyCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_invitations WHERE request_id=$1::uuid AND revoked_at IS NOT NULL`, created.ID).Scan(&revokedInvitations); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_sessions WHERE request_id=$1::uuid AND revoked_at IS NOT NULL`, created.ID).Scan(&revokedSessions); err != nil {
		t.Fatal(err)
	}
	if historyCount != 0 || revokedInvitations != 0 || revokedSessions != 0 {
		t.Fatalf("rejected requester authority mutated history/capabilities history=%d invitations=%d sessions=%d", historyCount, revokedInvitations, revokedSessions)
	}
	if _, err := repo.ListRecipientCandidates(ctx, "recipient-candidates-test", entityID, created.ID, requesterID, 50); !errors.Is(err, ErrNotFound) {
		t.Fatalf("requester with expired entity membership retained candidate access: %v", err)
	}
	mustExecRecipientCandidate(t, ctx, pool, `UPDATE org_positions SET valid_until=NULL WHERE occupant_principal_id=$1::uuid AND code='REQUESTER'`, requesterID)

	mustExecRecipientCandidate(t, ctx, pool, `UPDATE principals SET status='INACTIVE' WHERE id=$1::uuid`, requesterID)
	if _, err := repo.ListRecipientCandidates(ctx, "recipient-candidates-test", entityID, created.ID, requesterID, 50); !errors.Is(err, ErrNotFound) {
		t.Fatalf("inactive requester retained candidate access in exact query: %v", err)
	}
	mustExecRecipientCandidate(t, ctx, pool, `UPDATE principals SET status='ACTIVE' WHERE id=$1::uuid`, requesterID)

	mustExecRecipientCandidate(t, ctx, pool, `UPDATE principals SET status='INACTIVE' WHERE id=$1::uuid`, eligibleID)
	values, err := service.ListRecipientCandidates(ctx, ActorRequestScope{TenantID: "recipient-candidates-test", LegalEntityID: entityID, ActorPrincipalID: requesterID}, created.ID, 50)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range values {
		if candidate.PrincipalID == eligibleID {
			t.Fatalf("inactive principal remained eligible: %#v", values)
		}
	}
	loaded, err = service.GetRequestForEntity(ctx, "recipient-candidates-test", entityID, created.ID)
	if err != nil || loaded.Recipient.DisplayName != "Zara Eligible" {
		t.Fatalf("historical current recipient lost its safe label: %#v err=%v", loaded.Recipient, err)
	}

	mustExecRecipientCandidate(t, ctx, pool, `UPDATE programs SET scope=$2::jsonb WHERE id=$1::uuid`, programID, `{"access":"RESTRICTED","allowed_principal_ids":[]}`)
	_, err = repo.ListRecipientCandidates(ctx, "recipient-candidates-test", entityID, created.ID, requesterID, 50)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("requester retained candidate access after losing subject visibility: %v", err)
	}
}

func cleanupRecipientCandidatesPostgres(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID string) {
	t.Helper()
	for _, query := range []string{
		`DELETE FROM capture_sessions WHERE tenant_id=$1::uuid`,
		`DELETE FROM capture_invitations WHERE tenant_id=$1::uuid`,
		`DELETE FROM capture_recipient_history WHERE tenant_id=$1::uuid`,
		`DELETE FROM capture_requests WHERE tenant_id=$1::uuid`,
		`DELETE FROM programs WHERE tenant_id=$1::uuid`,
		`DELETE FROM org_positions WHERE tenant_id=$1::uuid`,
		`DELETE FROM principals WHERE tenant_id=$1::uuid`,
		`DELETE FROM legal_entities WHERE tenant_id=$1::uuid`,
		`DELETE FROM tenants WHERE id=$1::uuid`,
	} {
		if _, err := pool.Exec(ctx, query, tenantID); err != nil {
			t.Fatalf("cleanup recipient candidates: %v", err)
		}
	}
}

func mustExecRecipientCandidate(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(ctx, query, args...); err != nil {
		t.Fatal(err)
	}
}
