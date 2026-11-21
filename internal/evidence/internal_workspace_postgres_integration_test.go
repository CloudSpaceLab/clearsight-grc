//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestInternalPostgresWorkspaceIdentityVersionProvenanceAndRollback(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	var tenant, entity, maker, person, form, program, position string
	if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&tenant, &entity, &maker, &person, &form, &program, &position); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	setupDistributionFixture(t, ctx, pool, tenant, entity, maker, person, form, now)
	t.Cleanup(func() {
		cleanupDistributionFailureTrigger(context.Background(), pool)
		cleanupDistributionTenant(context.Background(), pool, tenant)
	})
	if _, err := pool.Exec(ctx, `
		UPDATE monitoring_form_templates SET fields='[{"id":"q1","section_id":"general","label":"Access reviewed?","type":"yes_no","required":true},{"id":"file","section_id":"general","label":"Document","type":"file"}]'::jsonb WHERE id=$1::uuid;
		INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,scope,effective_from) VALUES($2::uuid,$3::uuid,$4::uuid,'INTERNAL-WS','Internal workspace','COMPLIANCE','ACTIVE','Risk','NG','{"access":"INTERNAL"}'::jsonb,$6);
		INSERT INTO org_positions(id,tenant_id,legal_entity_id,code,title,occupant_principal_id,valid_from) VALUES($5::uuid,$3::uuid,$4::uuid,'RESPONDENT','Respondent',$7::uuid,$6)`, pgx.QueryExecModeSimpleProtocol, form, program, tenant, entity, position, now.Add(-time.Hour), person); err != nil {
		t.Fatal(err)
	}
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), postgresTestRecipientProtector{})
	store.now = func() time.Time { return now }
	bundle, err := store.CreateDistribution(ctx, CreateDistributionInput{TenantID: tenant, LegalEntityID: entity, FormTemplateID: form, FormTemplateVersion: 1, SubjectType: "PROGRAM", SubjectID: program, Title: "Access review", Purpose: "Review access", AccessPolicy: AccessDirectEmailOTP, EstimatedMinutes: 5, Deadline: now.Add(time.Hour), RouteExpiresAt: now.Add(time.Hour), CreatedBy: maker, Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientInternalPrincipal, PrincipalID: person}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE capture_form_distributions SET status='OPEN' WHERE id=$1::uuid`, bundle.Distribution.ID); err != nil {
		t.Fatal(err)
	}
	requestID := bundle.Recipients[0].RequestID
	actor := identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: person, Kind: "PERSON", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}
	respondent := identity.WithActor(ctx, actor)
	other := actor
	other.PrincipalID = maker
	if _, _, err := store.getInternalPostgresWorkspace(identity.WithActor(ctx, other), requestID, now); !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("wrong principal: %v", err)
	}
	wrongEntity := actor
	wrongEntity.LegalEntityID = "ad111111-1111-7111-8111-111111111118"
	if _, _, err := store.getInternalPostgresWorkspace(identity.WithActor(ctx, wrongEntity), requestID, now); !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("wrong entity: %v", err)
	}
	_, view, err := store.getInternalPostgresWorkspace(respondent, requestID, now)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	t.Run("document upload rechecks recipient and commits audit", func(t *testing.T) {
		objects := &revokingUploadStore{MemoryObjectStore: NewMemoryObjectStore(), revoke: func() {}}
		documents := NewService(store.repo, objects)
		access := (&DistributionAccessService{store: store, now: func() time.Time { return now }}).WithInternalWorkspaceAuthorizer(func(_ context.Context, who identity.Actor, request Request, command string) (string, error) {
			if who.PrincipalID != person || request.ID != requestID || command != "forms.response.upload" {
				return "", errors.New("denied")
			}
			return "v1:performer", nil
		})
		created, err := documents.StoreArtifactForInternalWorkspace(respondent, access, requestID, ArtifactInput{FieldID: "file", FileName: "report.txt", MediaType: "text/plain"}, strings.NewReader("sample document"))
		if err != nil {
			t.Fatal(err)
		}
		var jobs, audits, outbox int
		if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_artifact_scan_jobs WHERE artifact_id=$1::uuid), (SELECT count(*) FROM capture_distribution_events WHERE distribution_id=$2::uuid AND event_type=$3 AND actor_id=$4::uuid AND payload->>'authority_route'='v1:performer'), (SELECT count(*) FROM outbox_events WHERE aggregate_id=$2::uuid AND event_type='FORM_RESPONSE_DOCUMENT_UPLOADED' AND payload->>'artifact_id'=$1::text)`, created.ID, bundle.Distribution.ID, "FORM_RESPONSE_DOCUMENT_UPLOADED_"+created.ID, person).Scan(&jobs, &audits, &outbox); err != nil {
			t.Fatal(err)
		}
		if jobs != 1 || audits != 1 || outbox != 1 {
			t.Fatalf("upload audit: jobs=%d audit=%d outbox=%d", jobs, audits, outbox)
		}
		cleanupTrigger := func() {
			_, _ = pool.Exec(context.Background(), `DROP TRIGGER IF EXISTS internal_upload_outbox_failure_test ON outbox_events; DROP FUNCTION IF EXISTS internal_upload_outbox_failure_test()`, pgx.QueryExecModeSimpleProtocol)
		}
		cleanupTrigger()
		t.Cleanup(cleanupTrigger)
		_, err = pool.Exec(ctx, fmt.Sprintf(`CREATE FUNCTION internal_upload_outbox_failure_test() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.aggregate_id='%s'::uuid AND NEW.event_type='FORM_RESPONSE_DOCUMENT_UPLOADED' THEN RAISE EXCEPTION 'injected outbox failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER internal_upload_outbox_failure_test BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION internal_upload_outbox_failure_test()`, bundle.Distribution.ID), pgx.QueryExecModeSimpleProtocol)
		if err != nil {
			t.Fatal(err)
		}
		_, err = documents.StoreArtifactForInternalWorkspace(respondent, access, requestID, ArtifactInput{FieldID: "file", FileName: "failed.txt", MediaType: "text/plain"}, strings.NewReader("sample document"))
		if err == nil {
			t.Fatal("upload succeeded despite outbox failure")
		}
		if _, err := objects.Open(ctx, objects.key); err == nil {
			t.Fatal("failed upload left object")
		}
		var leaked int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_artifacts WHERE storage_key=$1`, objects.key).Scan(&leaked); err != nil || leaked != 0 {
			t.Fatalf("failed upload artifact: %d %v", leaked, err)
		}
		cleanupTrigger()
		var revokeErr error
		objects.revoke = func() {
			_, revokeErr = pool.Exec(ctx, `UPDATE capture_distribution_recipients SET state='REVOKED' WHERE request_id=$1::uuid`, requestID)
		}
		_, err = documents.StoreArtifactForInternalWorkspace(respondent, access, requestID, ArtifactInput{FieldID: "file", FileName: "second.txt", MediaType: "text/plain"}, strings.NewReader("sample document"))
		if revokeErr != nil || !errors.Is(err, ErrWorkspaceUnavailable) {
			t.Fatalf("revoked upload: %v / %v", err, revokeErr)
		}
		if _, err := objects.Open(ctx, objects.key); err == nil {
			t.Fatal("revoked upload left object")
		}
		if _, err := pool.Exec(ctx, `UPDATE capture_distribution_recipients SET state='PENDING' WHERE request_id=$1::uuid`, requestID); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("slug identity uses the exact current performer route", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,valid_from,policy_version) VALUES($1::uuid,$2::uuid,$3::uuid,'PERFORMER','EVIDENCE_REQUEST',$4::uuid,$5,'v1')`, tenant, entity, person, requestID, now.Add(-time.Hour)); err != nil {
			t.Fatal(err)
		}
		resolver := authority.NewEffectivePostgresService(pool)
		var authorizationErr error
		service := (&DistributionAccessService{store: store, now: func() time.Time { return now }}).WithInternalWorkspaceAuthorizer(func(ctx context.Context, actor identity.Actor, request Request, command string) (string, error) {
			if actor.TenantID != request.TenantID || actor.LegalEntityID != request.LegalEntityID {
				return "", errors.New("authority scope mismatch")
			}
			route, err := resolver.Resolve(ctx, authority.ResolveInput{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "EVIDENCE_REQUEST", ObjectID: request.ID, Responsibility: authority.ResponsibilityPerformer, DecisionType: command, Materiality: 3})
			authorizationErr = err
			if err != nil {
				return "", err
			}
			if !route.AllowsPrincipal(actor.PrincipalID) {
				return "", errors.New("performer denied")
			}
			return route.PolicyVersion + ":" + route.RuleID, nil
		})
		slugActor := actor
		slugActor.TenantID = "distribution-" + tenant[len(tenant)-12:]
		if _, err := service.SaveInternalWorkspace(identity.WithActor(ctx, slugActor), requestID, SaveWorkspaceInput{ExpectedVersion: view.Workspace.Version, Edits: []FieldEdit{{FieldID: "q1", Value: formcontract.AnswerValue{}}}}); err != nil {
			t.Fatalf("slug performer: %v (authority: %v)", err, authorizationErr)
		}
	})
	if _, err := store.submitInternalPostgresWorkspace(respondent, requestID, SubmitWorkspaceInput{ExpectedVersion: view.Workspace.Version}, now); err == nil {
		t.Fatal("missing answer was accepted")
	}
	saved, err := store.saveInternalPostgresWorkspace(respondent, requestID, SaveWorkspaceInput{ExpectedVersion: view.Workspace.Version, Edits: []FieldEdit{{FieldID: "q1", Value: formcontract.TextAnswer("Yes")}}}, now)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	var recordedEpoch, currentEpoch int64
	if err := pool.QueryRow(ctx, `SELECT (event.payload->>'authority_epoch')::bigint,epoch.version FROM capture_distribution_events event JOIN tenant_authority_epochs epoch ON epoch.tenant_id=event.tenant_id WHERE event.tenant_id=$1::uuid AND event.distribution_id=$2::uuid AND event.event_type=$3`, tenant, bundle.Distribution.ID, fmt.Sprintf("FORM_RESPONSE_WORKSPACE_SAVED_%d", saved.Workspace.Version)).Scan(&recordedEpoch, &currentEpoch); err != nil || recordedEpoch < 1 || recordedEpoch != currentEpoch {
		t.Fatalf("saved response authority epoch: event=%d current=%d err=%v", recordedEpoch, currentEpoch, err)
	}
	checkRevocationFirst := func(t *testing.T, revoke, restore, id, targetRequestID string) {
		t.Helper()
		revocation, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer revocation.Rollback(ctx)
		if _, err := revocation.Exec(ctx, revoke, id); err != nil {
			t.Fatal(err)
		}
		command, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer command.Rollback(ctx)
		var pid int
		if err := command.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
			t.Fatal(err)
		}
		result := make(chan error, 1)
		go func() {
			_, _, err := loadInternalPostgresWorkspaceState(respondent, command, actor, targetRequestID, now, true)
			result <- err
		}()
		deadline := time.Now().Add(5 * time.Second)
		for {
			var waiting bool
			if err := pool.QueryRow(ctx, `SELECT COALESCE(wait_event_type='Lock',false) FROM pg_stat_activity WHERE pid=$1`, pid).Scan(&waiting); err != nil {
				t.Fatal(err)
			}
			if waiting {
				break
			}
			select {
			case err := <-result:
				t.Fatalf("command returned before revocation committed: %v", err)
			default:
			}
			if time.Now().After(deadline) {
				t.Fatal("command did not wait for revocation")
			}
			time.Sleep(10 * time.Millisecond)
		}
		if err := revocation.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := pool.Exec(ctx, restore, id); err != nil {
				t.Error(err)
			}
		})
		if err := <-result; !errors.Is(err, ErrWorkspaceUnavailable) {
			t.Fatalf("command accepted committed revocation: %v", err)
		}
	}
	t.Run("eligibility rows cannot be revoked during a command", func(t *testing.T) {
		commandTx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer commandTx.Rollback(ctx)
		if _, _, err := loadInternalPostgresWorkspaceState(respondent, commandTx, actor, requestID, now, true); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct{ name, query, id string }{
			{"program scope", `UPDATE programs SET scope='{"access":"RESTRICTED","allowed_principal_ids":[]}'::jsonb WHERE id=$1::uuid`, program},
			{"position", `UPDATE org_positions SET valid_until=clock_timestamp() WHERE id=$1::uuid`, position},
		} {
			t.Run(tc.name, func(t *testing.T) {
				revocation, err := pool.Begin(ctx)
				if err != nil {
					t.Fatal(err)
				}
				defer revocation.Rollback(ctx)
				if _, err := revocation.Exec(ctx, `SET LOCAL lock_timeout='200ms'`); err != nil {
					t.Fatal(err)
				}
				_, err = revocation.Exec(ctx, tc.query, tc.id)
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
					t.Fatalf("expected lock timeout, got %v", err)
				}
			})
		}
	})
	t.Run("committed revocation denies the next command", func(t *testing.T) {
		for _, tc := range []struct{ name, revoke, restore, id string }{
			{"program scope", `UPDATE programs SET scope='{"access":"RESTRICTED","allowed_principal_ids":[]}'::jsonb WHERE id=$1::uuid`, `UPDATE programs SET scope='{"access":"INTERNAL"}'::jsonb WHERE id=$1::uuid`, program},
			{"position", `UPDATE org_positions SET valid_until=clock_timestamp() WHERE id=$1::uuid`, `UPDATE org_positions SET valid_until=NULL WHERE id=$1::uuid`, position},
		} {
			t.Run(tc.name, func(t *testing.T) {
				if _, err := pool.Exec(ctx, tc.revoke, tc.id); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if _, err := pool.Exec(ctx, tc.restore, tc.id); err != nil {
						t.Error(err)
					}
				})
				if _, err := store.submitInternalPostgresWorkspace(respondent, requestID, SubmitWorkspaceInput{ExpectedVersion: saved.Workspace.Version}, now); !errors.Is(err, ErrWorkspaceUnavailable) {
					t.Fatalf("revoked submit: %v", err)
				}
			})
		}
	})
	t.Run("revocation first", func(t *testing.T) {
		t.Run("program scope", func(t *testing.T) {
			checkRevocationFirst(t, `UPDATE programs SET scope='{"access":"RESTRICTED","allowed_principal_ids":[]}'::jsonb WHERE id=$1::uuid`, `UPDATE programs SET scope='{"access":"INTERNAL"}'::jsonb WHERE id=$1::uuid`, program, requestID)
		})
		t.Run("position", func(t *testing.T) {
			checkRevocationFirst(t, `UPDATE org_positions SET valid_until=clock_timestamp() WHERE id=$1::uuid`, `UPDATE org_positions SET valid_until=NULL WHERE id=$1::uuid`, position, requestID)
		})
	})
	t.Run("matter scope revocation", func(t *testing.T) {
		var matter string
		if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()::text`).Scan(&matter); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,priority,title,summary,scope) VALUES($1::uuid,$2::uuid,$3::uuid,'INTERNAL-WORKSPACE','CONTROL_GAP',3,'Access review','Review access','{"access":"INTERNAL"}'::jsonb)`, matter, tenant, entity); err != nil {
			t.Fatal(err)
		}
		matterBundle, err := store.CreateDistribution(ctx, CreateDistributionInput{TenantID: tenant, LegalEntityID: entity, FormTemplateID: form, FormTemplateVersion: 1, SubjectType: "MATTER", SubjectID: matter, Title: "Access review", Purpose: "Review access", AccessPolicy: AccessDirectEmailOTP, EstimatedMinutes: 5, Deadline: now.Add(time.Hour), RouteExpiresAt: now.Add(time.Hour), CreatedBy: maker, Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientInternalPrincipal, PrincipalID: person}}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE capture_form_distributions SET status='OPEN' WHERE id=$1::uuid`, matterBundle.Distribution.ID); err != nil {
			t.Fatal(err)
		}
		matterRequest := matterBundle.Recipients[0].RequestID
		revoke := `UPDATE matters SET scope='{"access":"RESTRICTED","allowed_principal_ids":[]}'::jsonb WHERE id=$1::uuid`
		restore := `UPDATE matters SET scope='{"access":"INTERNAL"}'::jsonb WHERE id=$1::uuid`
		command, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := loadInternalPostgresWorkspaceState(respondent, command, actor, matterRequest, now, true); err != nil {
			command.Rollback(ctx)
			t.Fatal(err)
		}
		revocation, err := pool.Begin(ctx)
		if err != nil {
			command.Rollback(ctx)
			t.Fatal(err)
		}
		if _, err := revocation.Exec(ctx, `SET LOCAL lock_timeout='200ms'`); err != nil {
			t.Fatal(err)
		}
		_, err = revocation.Exec(ctx, revoke, matter)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
			t.Errorf("expected lock timeout, got %v", err)
		}
		revocation.Rollback(ctx)
		command.Rollback(ctx)
		checkRevocationFirst(t, revoke, restore, matter, matterRequest)
	})
	t.Run("directory eligibility revocations", func(t *testing.T) {
		var source, user, group, role, binding string
		if err := pool.QueryRow(ctx, `SELECT gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text,gen_random_uuid()::text`).Scan(&source, &user, &group, &role, &binding); err != nil {
			t.Fatal(err)
		}
		_, err := pool.Exec(ctx, `
			UPDATE org_positions SET valid_until=clock_timestamp() WHERE id=$1::uuid;
			INSERT INTO scim_sources(id,tenant_id,code,token_hash) VALUES($2::uuid,$3::uuid,'DIRECTORY',decode(md5($2::text)||md5($2::text),'hex'));
			INSERT INTO scim_users(id,tenant_id,source_id,principal_id,user_name) VALUES($4::uuid,$3::uuid,$2::uuid,$5::uuid,'respondent');
			INSERT INTO directory_groups(id,tenant_id,source_id,display_name) VALUES($6::uuid,$3::uuid,$2::uuid,'Respondents');
			INSERT INTO directory_group_members(tenant_id,group_id,scim_user_id) VALUES($3::uuid,$6::uuid,$4::uuid);
			INSERT INTO role_templates(id,tenant_id,code,name,valid_from) VALUES($7::uuid,$3::uuid,'RESPONDENT','Respondent',$10);
			INSERT INTO directory_group_role_bindings(id,tenant_id,group_id,role_template_id,legal_entity_id,valid_from) VALUES($8::uuid,$3::uuid,$6::uuid,$7::uuid,$9::uuid,$10)`, pgx.QueryExecModeSimpleProtocol, position, source, tenant, user, person, group, role, binding, entity, now.Add(-time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := pool.Exec(ctx, `UPDATE org_positions SET valid_until=NULL WHERE id=$1::uuid`, position); err != nil {
				t.Error(err)
			}
		})
		t.Run("source revocation first", func(t *testing.T) {
			checkRevocationFirst(t, `UPDATE scim_sources SET status='REVOKED' WHERE id=$1::uuid`, `UPDATE scim_sources SET status='ACTIVE' WHERE id=$1::uuid`, source, requestID)
		})
		for _, tc := range []struct{ name, revoke, restore, id string }{
			{"source", `UPDATE scim_sources SET status='REVOKED' WHERE id=$1::uuid`, `UPDATE scim_sources SET status='ACTIVE' WHERE id=$1::uuid`, source},
			{"user", `UPDATE scim_users SET active=false WHERE id=$1::uuid`, `UPDATE scim_users SET active=true WHERE id=$1::uuid`, user},
			{"group", `UPDATE directory_groups SET deleted_at=clock_timestamp() WHERE id=$1::uuid`, `UPDATE directory_groups SET deleted_at=NULL WHERE id=$1::uuid`, group},
			{"membership", `DELETE FROM directory_group_members WHERE scim_user_id=$1::uuid`, `INSERT INTO directory_group_members(tenant_id,group_id,scim_user_id) SELECT tenant_id,id,$1::uuid FROM directory_groups WHERE id=$2::uuid`, user},
			{"binding", `UPDATE directory_group_role_bindings SET valid_until=clock_timestamp() WHERE id=$1::uuid`, `UPDATE directory_group_role_bindings SET valid_until=NULL WHERE id=$1::uuid`, binding},
			{"role", `UPDATE role_templates SET valid_until=clock_timestamp() WHERE id=$1::uuid`, `UPDATE role_templates SET valid_until=NULL WHERE id=$1::uuid`, role},
		} {
			t.Run(tc.name, func(t *testing.T) {
				commandTx, err := pool.Begin(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if _, _, err := loadInternalPostgresWorkspaceState(respondent, commandTx, actor, requestID, now, true); err != nil {
					commandTx.Rollback(ctx)
					t.Fatal(err)
				}
				revocation, err := pool.Begin(ctx)
				if err != nil {
					commandTx.Rollback(ctx)
					t.Fatal(err)
				}
				if _, err := revocation.Exec(ctx, `SET LOCAL lock_timeout='200ms'`); err != nil {
					t.Fatal(err)
				}
				_, err = revocation.Exec(ctx, tc.revoke, tc.id)
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
					t.Errorf("expected lock timeout, got %v", err)
				}
				revocation.Rollback(ctx)
				commandTx.Rollback(ctx)
				if _, err := pool.Exec(ctx, tc.revoke, tc.id); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					var err error
					if tc.name == "membership" {
						_, err = pool.Exec(ctx, tc.restore, tc.id, group)
					} else {
						_, err = pool.Exec(ctx, tc.restore, tc.id)
					}
					if err != nil {
						t.Error(err)
					}
				})
				if _, err := store.submitInternalPostgresWorkspace(respondent, requestID, SubmitWorkspaceInput{ExpectedVersion: saved.Workspace.Version}, now); !errors.Is(err, ErrWorkspaceUnavailable) {
					t.Fatalf("revoked submit: %v", err)
				}
			})
		}
	})
	if _, err := store.submitInternalPostgresWorkspace(respondent, requestID, SubmitWorkspaceInput{ExpectedVersion: view.Workspace.Version}, now); !errors.Is(err, ErrWorkspaceConflict) {
		t.Fatalf("stale submit: %v", err)
	}
	// An outbox failure must roll back the submission, revision, audit and workspace state.
	trigger := "internal_workspace_outbox_failure_test"
	cleanupTrigger := func() {
		_, _ = pool.Exec(context.Background(), `DROP TRIGGER IF EXISTS internal_workspace_outbox_failure_test ON outbox_events; DROP FUNCTION IF EXISTS internal_workspace_outbox_failure_test()`, pgx.QueryExecModeSimpleProtocol)
	}
	cleanupTrigger()
	t.Cleanup(cleanupTrigger)
	_, err = pool.Exec(ctx, fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.aggregate_id='%s'::uuid AND NEW.event_type='FORM_RESPONSE_REVISION_SUBMITTED' THEN RAISE EXCEPTION 'injected outbox failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER %s BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION %s()`, trigger, bundle.Distribution.ID, trigger, trigger), pgx.QueryExecModeSimpleProtocol)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.submitInternalPostgresWorkspace(respondent, requestID, SubmitWorkspaceInput{ExpectedVersion: saved.Workspace.Version}, now); err == nil {
		t.Fatal("submission succeeded despite outbox failure")
	}
	var submissions, revisions, submittedEvents int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_submissions WHERE request_id=$1::uuid), (SELECT count(*) FROM capture_response_revisions r JOIN capture_submissions s ON s.id=r.submission_id WHERE s.request_id=$1::uuid), (SELECT count(*) FROM capture_distribution_events WHERE distribution_id=$2::uuid AND event_type LIKE 'FORM_RESPONSE_REVISION_SUBMITTED%')`, requestID, bundle.Distribution.ID).Scan(&submissions, &revisions, &submittedEvents); err != nil {
		t.Fatal(err)
	}
	if submissions != 0 || revisions != 0 || submittedEvents != 0 {
		t.Fatalf("failed submission leaked state: submissions=%d revisions=%d events=%d", submissions, revisions, submittedEvents)
	}
	_, afterFailure, err := store.getInternalPostgresWorkspace(respondent, requestID, now)
	if err != nil || afterFailure.Workspace.Version != saved.Workspace.Version {
		t.Fatalf("workspace changed after failed submission: version=%d err=%v", afterFailure.Workspace.Version, err)
	}
	cleanupTrigger()
	type outcome struct {
		result WorkspaceSubmissionResult
		err    error
	}
	results := make(chan outcome, 2)
	for range 2 {
		go func() {
			result, err := store.submitInternalPostgresWorkspace(respondent, requestID, SubmitWorkspaceInput{ExpectedVersion: saved.Workspace.Version}, now)
			results <- outcome{result, err}
		}()
	}
	var result WorkspaceSubmissionResult
	var successes, conflicts int
	for range 2 {
		outcome := <-results
		if outcome.err == nil {
			result = outcome.result
			successes++
		} else if errors.Is(outcome.err, ErrWorkspaceConflict) {
			conflicts++
		} else {
			t.Fatalf("concurrent submit: %v", outcome.err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
	var submittedBy, channel, assurance string
	if err := pool.QueryRow(ctx, `SELECT s.submitted_by::text,s.channel,r.achieved_assurance FROM capture_submissions s JOIN capture_response_revisions r ON r.submission_id=s.id WHERE s.id=$1::uuid`, result.Submission.SubmissionID).Scan(&submittedBy, &channel, &assurance); err != nil {
		t.Fatal(err)
	}
	if submittedBy != person || channel != "INTERNAL" || assurance != string(AssuranceInternalAuthenticated) {
		t.Fatalf("provenance: %s %s %s", submittedBy, channel, assurance)
	}
	var missingActor int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_distribution_events WHERE distribution_id=$1::uuid AND event_type LIKE 'FORM_RESPONSE_%' AND actor_id IS DISTINCT FROM $2::uuid`, bundle.Distribution.ID, person).Scan(&missingActor); err != nil || missingActor != 0 {
		t.Fatalf("audit actor: %d %v", missingActor, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE capture_distribution_recipients SET state='REVOKED' WHERE request_id=$1::uuid`, requestID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.getInternalPostgresWorkspace(respondent, requestID, now); !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("revoked read: %v", err)
	}
	if _, err := store.saveInternalPostgresWorkspace(respondent, requestID, SaveWorkspaceInput{ExpectedVersion: result.Workspace.Version}, now); !errors.Is(err, ErrWorkspaceUnavailable) {
		t.Fatalf("revoked save: %v", err)
	}
}
