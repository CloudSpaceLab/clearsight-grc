//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"errors"
	"fmt"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPostgresWorkspaceArtifactSourceRevocationLocks(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant := mustResponseWorkspaceID(t)
	entity := mustResponseWorkspaceID(t)
	maker := mustResponseWorkspaceID(t)
	person := mustResponseWorkspaceID(t)
	form := mustResponseWorkspaceID(t)
	subject := mustResponseWorkspaceID(t)
	artifact := mustResponseWorkspaceID(t)
	now := time.Now().UTC().Truncate(time.Second)
	setupDistributionFixture(t, ctx, pool, tenant, entity, maker, person, form, now)
	t.Cleanup(func() { cleanupDistributionTenant(context.Background(), pool, tenant) })
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), postgresTestRecipientProtector{})
	store.now = func() time.Time { return now }
	bundle, err := store.CreateDistribution(ctx, CreateDistributionInput{TenantID: tenant, LegalEntityID: entity, FormTemplateID: form, FormTemplateVersion: 1, SubjectType: "PROGRAM", SubjectID: subject, Title: "Test", Purpose: "Review documents", AccessPolicy: AccessSharedEmailOTP, EstimatedMinutes: 5, Deadline: now.Add(time.Hour), RouteExpiresAt: now.Add(time.Hour), CreatedBy: maker, Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientInternalPrincipal, PrincipalID: person}, {Role: RecipientTo, Type: RecipientInternalPrincipal, PrincipalID: maker}}})
	if err != nil {
		t.Fatalf("distribution: %v", err)
	}
	if len(bundle.Recipients) != 2 || bundle.Recipients[0].RequestID == bundle.Recipients[1].RequestID {
		t.Fatal("expected distinct source and actor recipients")
	}
	source := bundle.Recipients[0]
	if _, err = pool.Exec(ctx, `INSERT INTO capture_artifacts(id,tenant_id,request_id,file_name,media_type,size_bytes,sha256,storage_key,status,created_at) VALUES($1::uuid,$2::uuid,$3::uuid,'report.txt','text/plain',1,$4,$5,'STORED_UNSCANNED',$6)`, artifact, tenant, source.RequestID, fmt.Sprintf("%064d", 0), "test-"+artifact, now); err != nil {
		t.Fatalf("artifact: %v", err)
	}
	t.Run("command first holds source recipient until commit", func(t *testing.T) {
		command, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer command.Rollback(ctx)
		if _, err = loadScopedPostgresWorkspaceArtifact(ctx, command, bundle.Distribution.ID, entity, tenant, "", artifact, false); err != nil {
			t.Fatal(err)
		}
		revoke, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer revoke.Rollback(ctx)
		if _, err = revoke.Exec(ctx, `SET LOCAL lock_timeout='200ms'`); err != nil {
			t.Fatal(err)
		}
		_, err = revoke.Exec(ctx, `UPDATE capture_distribution_recipients SET state='REVOKED' WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenant, source.ID)
		var dbErr *pgconn.PgError
		if !errors.As(err, &dbErr) || dbErr.Code != "55P03" {
			t.Fatalf("revocation did not wait for the exact source recipient: %v", err)
		}
	})
	t.Run("revocation first is rechecked after wait", func(t *testing.T) {
		revoke, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer revoke.Rollback(ctx)
		if _, err = revoke.Exec(ctx, `UPDATE capture_distribution_recipients SET state='REVOKED' WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenant, source.ID); err != nil {
			t.Fatal(err)
		}
		command, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer command.Rollback(ctx)
		var pid int
		if err = command.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
			t.Fatal(err)
		}
		result := make(chan error, 1)
		go func() {
			_, e := loadScopedPostgresWorkspaceArtifact(ctx, command, bundle.Distribution.ID, entity, tenant, "", artifact, false)
			result <- e
		}()
		deadline := time.Now().Add(5 * time.Second)
		for {
			select {
			case e := <-result:
				t.Fatalf("command returned before pending revocation committed: %v", e)
			default:
			}
			var waiting bool
			if e := pool.QueryRow(ctx, `SELECT COALESCE(wait_event_type='Lock',false) FROM pg_stat_activity WHERE pid=$1`, pid).Scan(&waiting); e != nil {
				t.Fatal(e)
			}
			if waiting {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("command did not wait on recipient lock")
			}
		}
		if err = revoke.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		select {
		case err = <-result:
			if !errors.Is(err, ErrNotFound) {
				t.Fatalf("revoked source accepted: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("command did not finish after revocation")
		}
	})
}

func TestPostgresWorkspaceDraftRejectsConcurrentSourceRevocation(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant := mustResponseWorkspaceID(t)
	entity := mustResponseWorkspaceID(t)
	maker := mustResponseWorkspaceID(t)
	person := mustResponseWorkspaceID(t)
	form := mustResponseWorkspaceID(t)
	subject := mustResponseWorkspaceID(t)
	artifact := mustResponseWorkspaceID(t)
	now := time.Now().UTC().Truncate(time.Second)
	setupDistributionFixture(t, ctx, pool, tenant, entity, maker, person, form, now)
	t.Cleanup(func() { cleanupDistributionTenant(context.Background(), pool, tenant) })
	if _, err := pool.Exec(ctx, `UPDATE monitoring_form_templates SET fields='[{"id":"file","section_id":"general","label":"Document","type":"file"}]'::jsonb WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenant, form); err != nil {
		t.Fatal(err)
	}
	var recipientKey, accessKey [32]byte
	for i := range recipientKey {
		recipientKey[i] = 0x71
		accessKey[i] = 0x72
	}
	keyring, err := NewRecipientKeyring("recipient-v1", map[string][32]byte{"recipient-v1": recipientKey})
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), keyring)
	store.now = func() time.Time { return now }
	bundle, err := store.CreateDistribution(ctx, CreateDistributionInput{TenantID: tenant, LegalEntityID: entity, FormTemplateID: form, FormTemplateVersion: 1, SubjectType: "PROGRAM", SubjectID: subject, Title: "Test", Purpose: "Review documents", AccessPolicy: AccessSharedEmailOTP, EstimatedMinutes: 5, Deadline: now.Add(time.Hour), RouteExpiresAt: now.Add(time.Hour), CreatedBy: maker, Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientExternalAudience, Address: "source@example.test", AudienceHint: "s***@example.test", ContactLabel: "Source"}, {Role: RecipientTo, Type: RecipientExternalAudience, Address: "actor@example.test", AudienceHint: "a***@example.test", ContactLabel: "Actor"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_form_distributions SET status='OPEN' WHERE id=$1::uuid`, bundle.Distribution.ID); err != nil {
		t.Fatal(err)
	}
	delivery := &postgresAccessOTPDelivery{}
	access, err := NewDistributionAccessService(store, keyring, delivery, accessKey, 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	access.now = func() time.Time { return now }
	routes, err := access.IssueDistributionAccessRoutes(ctx, tenant, entity, bundle.Distribution.ID, maker)
	if err != nil || len(routes) != 1 {
		t.Fatalf("route: %v", err)
	}
	start, err := access.StartDistributionAccess(ctx, routes[0].Selector)
	if err != nil || len(start.Recipients) != 2 {
		t.Fatalf("recipients: %v", err)
	}
	receipt, err := access.SendOTP(ctx, routes[0].Selector, start.Recipients[1].SelectorID)
	if err != nil {
		t.Fatal(err)
	}
	redeemed, err := access.VerifyOTP(ctx, routes[0].Selector, receipt.ChallengeID, delivery.code)
	if err != nil {
		t.Fatal(err)
	}
	session, request, err := access.sessionRequest(ctx, redeemed.SessionToken)
	if err != nil {
		t.Fatal(err)
	}
	var sourceID, sourceRequest string
	if err = pool.QueryRow(ctx, `SELECT id::text,request_id::text FROM capture_distribution_recipients WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid AND id<>$3::uuid AND role='TO'`, tenant, bundle.Distribution.ID, session.RecipientID).Scan(&sourceID, &sourceRequest); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO capture_artifacts(id,tenant_id,request_id,file_name,media_type,size_bytes,sha256,storage_key,status,created_at) VALUES($1::uuid,$2::uuid,$3::uuid,'report.txt','text/plain',1,$4,$5,'STORED_UNSCANNED',$6)`, artifact, tenant, sourceRequest, fmt.Sprintf("%064d", 0), "test-"+artifact, now); err != nil {
		t.Fatal(err)
	}
	view, err := access.GetResponseWorkspace(ctx, redeemed.SessionToken)
	if err != nil {
		t.Fatal(err)
	}
	revoke, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer revoke.Rollback(ctx)
	if _, err = revoke.Exec(ctx, `UPDATE capture_distribution_recipients SET state='REVOKED' WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenant, sourceID); err != nil {
		t.Fatal(err)
	}
	var revokePID int
	if err = revoke.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&revokePID); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, err := store.SaveResponseWorkspace(ctx, workspaceSaveCommand{Session: session, Request: request, Now: now, Input: SaveWorkspaceInput{ExpectedVersion: view.Workspace.Version, Edits: []FieldEdit{{FieldID: "file", Value: formcontract.AnswerValue{ArtifactIDs: []string{artifact}}}}}, Validate: func(map[string]formcontract.AnswerValue) error { return nil }})
		result <- err
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		select {
		case e := <-result:
			t.Fatalf("draft save completed before source revocation: %v", e)
		default:
		}
		var waiting int
		if e := pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND $1=ANY(pg_blocking_pids(pid)) AND query LIKE '%FOR SHARE OF a,r%'`, revokePID).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("draft save did not wait on source recipient")
		}
	}
	if err = revoke.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-result:
		if err == nil {
			t.Fatal("draft saved a revoked source document")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("draft save remained blocked")
	}
	var edits, events, outbox int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_response_workspace_edits WHERE tenant_id=$1::uuid AND workspace_id=$2::uuid),(SELECT count(*) FROM capture_distribution_events WHERE tenant_id=$1::uuid AND distribution_id=$3::uuid AND event_type LIKE 'FORM_RESPONSE_WORKSPACE_SAVED%'),(SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_id=$3::uuid AND event_type='FORM_RESPONSE_WORKSPACE_SAVED')`, tenant, view.Workspace.ID, bundle.Distribution.ID).Scan(&edits, &events, &outbox); err != nil || edits != 0 || events != 0 || outbox != 0 {
		t.Fatalf("failed draft left edits=%d events=%d outbox=%d err=%v", edits, events, outbox, err)
	}
	// Block the draft's outbox insert after artifact validation, retaining its
	// locks while a second transaction attempts to revoke the source.
	if _, err = pool.Exec(ctx, `UPDATE capture_distribution_recipients SET state='PENDING' WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenant, sourceID); err != nil {
		t.Fatal(err)
	}
	name := "workspace_lock_" + strings.ReplaceAll(artifact, "-", "")
	ddl := fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $body$ BEGIN IF NEW.aggregate_id='%s'::uuid AND NEW.event_type='FORM_RESPONSE_WORKSPACE_SAVED' THEN PERFORM pg_advisory_xact_lock(hashtextextended('%s',0)); END IF; RETURN NEW; END $body$; CREATE TRIGGER %s BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION %s()`, name, bundle.Distribution.ID, artifact, name, name)
	if _, err = pool.Exec(ctx, ddl, pgx.QueryExecModeSimpleProtocol); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON outbox_events; DROP FUNCTION IF EXISTS %s()`, name, name), pgx.QueryExecModeSimpleProtocol)
	})
	blocker, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(ctx)
	if _, err = blocker.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, artifact); err != nil {
		t.Fatal(err)
	}
	var blockerPID int
	if err = blocker.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&blockerPID); err != nil {
		t.Fatal(err)
	}
	go func() {
		_, err := store.SaveResponseWorkspace(ctx, workspaceSaveCommand{Session: session, Request: request, Now: now, Input: SaveWorkspaceInput{ExpectedVersion: view.Workspace.Version, Edits: []FieldEdit{{FieldID: "file", Value: formcontract.AnswerValue{ArtifactIDs: []string{artifact}}}}}, Validate: func(map[string]formcontract.AnswerValue) error { return nil }})
		result <- err
	}()
	deadline = time.Now().Add(5 * time.Second)
	for {
		select {
		case e := <-result:
			t.Fatalf("draft returned before commit barrier: %v", e)
		default:
		}
		var waiting bool
		if err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE $1=ANY(pg_blocking_pids(pid)))`, blockerPID).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("draft did not reach commit barrier")
		}
	}
	concurrentRevoke, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer concurrentRevoke.Rollback(ctx)
	if _, err = concurrentRevoke.Exec(ctx, `SET LOCAL lock_timeout='200ms'`); err != nil {
		t.Fatal(err)
	}
	_, err = concurrentRevoke.Exec(ctx, `UPDATE capture_distribution_recipients SET state='REVOKED' WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenant, sourceID)
	var dbErr *pgconn.PgError
	if !errors.As(err, &dbErr) || dbErr.Code != "55P03" {
		t.Fatalf("source revocation did not wait for draft: %v", err)
	}
	_ = concurrentRevoke.Rollback(ctx)
	if err = blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-result:
		if err != nil {
			t.Fatalf("draft save: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("draft did not finish after barrier")
	}
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_response_workspace_edits WHERE tenant_id=$1::uuid AND workspace_id=$2::uuid),(SELECT count(*) FROM capture_distribution_events WHERE tenant_id=$1::uuid AND distribution_id=$3::uuid AND event_type LIKE 'FORM_RESPONSE_WORKSPACE_SAVED%'),(SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_id=$3::uuid AND event_type='FORM_RESPONSE_WORKSPACE_SAVED')`, tenant, view.Workspace.ID, bundle.Distribution.ID).Scan(&edits, &events, &outbox); err != nil || edits != 1 || events != 1 || outbox != 1 {
		t.Fatalf("saved draft edits=%d events=%d outbox=%d err=%v", edits, events, outbox, err)
	}
}
