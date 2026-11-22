//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"testing"
	"time"
)

func TestExactDocumentQueryUsesCallerTransaction(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	const tenant = "a2211111-1111-7111-8111-111111111111"
	const entity = "a2211111-1111-7111-8111-111111111112"
	const actor = "a2211111-1111-7111-8111-111111111114"
	const form = "a2211111-1111-7111-8111-111111111115"
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	cleanup := func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM programs WHERE tenant_id=$1::uuid`, tenant)
		cleanupResponseWorkspaceTenant(context.Background(), pool, tenant)
	}
	cleanup()
	defer cleanup()
	setupResponseWorkspaceFixture(t, ctx, pool, tenant, "transaction-document-test", entity, actor, form, now)
	seedCompletedResponseRows(t, ctx, pool, tenant, entity, actor, form, "tx-document", 1, now)
	_, err := pool.Exec(ctx, `
 INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,scope,owning_function,jurisdiction,effective_from)
 VALUES(md5('tx-document-program')::uuid,$1::uuid,$2::uuid,'TX-DOCUMENT','Scoped review','COMPLIANCE','ACTIVE',jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($3::text)),'Compliance','NG',$4);
 UPDATE capture_form_distributions SET subject_type='PROGRAM',subject_id=md5('tx-document-program')::uuid WHERE tenant_id=$1::uuid;
 UPDATE capture_requests SET subject_type='PROGRAM',subject_id=md5('tx-document-program')::uuid::text,fields='[{"id":"file","label":"Policy","type":"file"}]' WHERE tenant_id=$1::uuid;
 UPDATE capture_submissions SET answers=jsonb_build_object('file',jsonb_build_object('artifact_ids',jsonb_build_array(md5('tx-artifact:'||id::text)::uuid::text))) WHERE tenant_id=$1::uuid;
 INSERT INTO capture_artifacts(id,tenant_id,request_id,submission_id,file_name,media_type,size_bytes,sha256,storage_key,status,created_by,created_at)
 SELECT md5('tx-artifact:'||id::text)::uuid,tenant_id,request_id,id,'policy.txt','text/plain',4,repeat('a',64),'tx-key/'||id::text,'AVAILABLE',$3::uuid,submitted_at FROM capture_submissions WHERE tenant_id=$1::uuid;`, pgx.QueryExecModeSimpleProtocol, tenant, entity, actor, now)
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), nil)
	q := DocumentQuery{TenantID: tenant, LegalEntityID: entity, PrincipalID: actor, Limit: 1}
	page, err := store.ListDocuments(ctx, q)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("initial read: %+v %v", page, err)
	}
	item := page.Items[0]
	q.SubmissionID, q.FieldID, q.ArtifactID, q.ResponseRevisionID = item.SubmissionID, item.FieldID, item.ArtifactID, item.ResponseRevisionID
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	inside, err := listDocumentsWithQuerier(ctx, tx, q, true)
	if err != nil || len(inside.Items) != 1 || inside.Items[0].ID != item.ID {
		t.Fatalf("transaction exact read disagrees with pool: %+v %v", inside, err)
	}
	missing := q
	missing.FieldID = ""
	if _, err = listDocumentsWithQuerier(ctx, tx, missing, true); !errors.Is(err, ErrDistributionInvalid) {
		t.Fatalf("missing exact selector accepted: %v", err)
	}
	wrong := q
	wrong.FieldID = "other"
	inside, err = listDocumentsWithQuerier(ctx, tx, wrong, true)
	if err != nil || len(inside.Items) != 0 {
		t.Fatalf("transaction widened exact occurrence: %+v %v", inside, err)
	}
	_, err = tx.Exec(ctx, `UPDATE programs SET scope=jsonb_build_object('access','RESTRICTED','allowed_principal_ids','[]'::jsonb) WHERE tenant_id=$1::uuid`, tenant)
	if err != nil {
		t.Fatal(err)
	}
	inside, err = listDocumentsWithQuerier(ctx, tx, q, true)
	if err != nil || len(inside.Items) != 0 {
		t.Fatalf("transaction ignored restriction: %+v %v", inside, err)
	}
	outside, err := store.ListExactDocuments(ctx, q)
	if err != nil || len(outside.Items) != 1 {
		t.Fatalf("pool unexpectedly observed uncommitted restriction: %+v %v", outside, err)
	}
}
