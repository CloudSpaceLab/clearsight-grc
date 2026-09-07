//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestPostgresDocumentInventoryScopesBeforeLimitAndPaginates(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	const tenant = "9f111111-1111-7111-8111-111111111111"
	const entity = "9f111111-1111-7111-8111-111111111112"
	const actor = "9f111111-1111-7111-8111-111111111114"
	const form = "9f111111-1111-7111-8111-111111111115"
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	cleanup := func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM programs WHERE tenant_id=$1::uuid`, tenant)
		cleanupResponseWorkspaceTenant(context.Background(), pool, tenant)
	}
	cleanup()
	defer cleanup()
	setupResponseWorkspaceFixture(t, ctx, pool, tenant, "documents-test", entity, actor, form, now)
	seedCompletedResponseRows(t, ctx, pool, tenant, entity, actor, form, "documents", 5, now)
	_, err := pool.Exec(ctx, `
 INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,scope,owning_function,jurisdiction,effective_from)
 VALUES(md5('documents-program')::uuid,$1::uuid,$2::uuid,'DOCUMENTS','Document review','COMPLIANCE','ACTIVE','{}','Compliance','NG',$4);
 UPDATE capture_form_distributions SET subject_type='PROGRAM',subject_id=md5('documents-program')::uuid WHERE tenant_id=$1::uuid;
 UPDATE capture_requests SET subject_type='PROGRAM',subject_id=md5('documents-program')::uuid::text,fields='[{"id":"file","label":"Policy","type":"file"}]' WHERE tenant_id=$1::uuid;
 UPDATE capture_submissions SET answers=jsonb_build_object('file',jsonb_build_object('artifact_ids',jsonb_build_array(md5('artifact:'||id::text)::uuid::text))) WHERE tenant_id=$1::uuid;
 INSERT INTO capture_artifacts(id,tenant_id,request_id,submission_id,file_name,media_type,size_bytes,sha256,storage_key,status,created_by,created_at)
 SELECT md5('artifact:'||id::text)::uuid,tenant_id,request_id,id,'policy.pdf','application/pdf',4,repeat('a',64),'private-key/'||id::text,'STORED_UNSCANNED',$3::uuid,submitted_at FROM capture_submissions WHERE tenant_id=$1::uuid;
 UPDATE capture_form_distributions SET subject_type='UNKNOWN' WHERE id=md5('documents:distribution:1')::uuid;
 `, pgx.QueryExecModeSimpleProtocol, tenant, entity, actor, now)
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), nil)
	q := DocumentQuery{TenantID: tenant, LegalEntityID: entity, PrincipalID: actor, Limit: 2, CurrentOnly: true}
	page, err := store.ListDocuments(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.NextCursor == "" {
		t.Fatalf("first page %+v", page)
	}
	q.Cursor = page.NextCursor
	second, err := store.ListDocuments(ctx, q)
	if err != nil || len(second.Items) != 2 || second.NextCursor != "" {
		t.Fatalf("second page %+v %v", second, err)
	}
	for _, a := range page.Items {
		for _, b := range second.Items {
			if a.ID == b.ID {
				t.Fatal("repeated occurrence")
			}
		}
	}
	q.Cursor = ""
	q.FileKind = DocumentWord
	empty, err := store.ListDocuments(ctx, q)
	if err != nil || len(empty.Items) != 0 {
		t.Fatalf("file filter %+v %v", empty, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE programs SET scope=jsonb_build_object('access','RESTRICTED','allowed_principal_ids',jsonb_build_array($2::text)) WHERE tenant_id=$1::uuid`, tenant, actor); err != nil {
		t.Fatal(err)
	}
	q.FileKind = ""
	q.PrincipalID = "9f111111-1111-7111-8111-111111111199"
	denied, err := store.ListDocuments(ctx, q)
	if err != nil || len(denied.Items) != 0 {
		t.Fatalf("restricted documents %+v %v", denied, err)
	}
	q.PrincipalID = actor
	q.LegalEntityID = "9f111111-1111-7111-8111-111111111198"
	denied, err = store.ListDocuments(ctx, q)
	if err != nil || len(denied.Items) != 0 {
		t.Fatalf("cross-entity documents %+v %v", denied, err)
	}
}
