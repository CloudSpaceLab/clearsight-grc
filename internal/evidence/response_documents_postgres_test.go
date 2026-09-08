//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"encoding/json"
	"strings"
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
	// Two immutable revisions reuse both an original artifact and a file
	// uploaded under another contributor request in the same distribution.
	source := page.Items[0]
	_, err = pool.Exec(ctx, `
 INSERT INTO capture_requests SELECT (jsonb_populate_record(NULL::capture_requests,to_jsonb(req)||jsonb_build_object('id',md5('document-contributor')::uuid))).* FROM capture_requests req WHERE id=$1::uuid;
 INSERT INTO capture_artifacts SELECT (jsonb_populate_record(NULL::capture_artifacts,to_jsonb(a)||jsonb_build_object('id',md5('contributor-artifact')::uuid,'request_id',md5('document-contributor')::uuid,'submission_id',NULL,'storage_key','contributor-file','file_name','contributor.pdf'))).* FROM capture_artifacts a WHERE id=$2::uuid;
 UPDATE capture_submissions SET answers=jsonb_build_object('file',jsonb_build_object('artifact_ids',jsonb_build_array($2::text,md5('contributor-artifact')::uuid::text))) WHERE id=$3::uuid;
 INSERT INTO capture_submissions SELECT (jsonb_populate_record(NULL::capture_submissions,to_jsonb(s)||jsonb_build_object('id',md5('document-amendment')::uuid,'submitted_at',s.submitted_at+interval '1 minute'))).* FROM capture_submissions s WHERE id=$3::uuid;
 UPDATE capture_response_revisions SET is_current=false WHERE id=$4::uuid;
 INSERT INTO capture_response_revisions SELECT (jsonb_populate_record(NULL::capture_response_revisions,to_jsonb(r)||jsonb_build_object('id',md5('document-amendment-revision')::uuid,'submission_id',md5('document-amendment')::uuid,'revision',2,'is_current',true,'supersedes_revision_id',r.id))).* FROM capture_response_revisions r WHERE id=$4::uuid;
 `, pgx.QueryExecModeSimpleProtocol, source.RequestID, source.ArtifactID, source.SubmissionID, source.ResponseRevisionID)
	if err != nil {
		t.Fatal(err)
	}
	q = DocumentQuery{TenantID: tenant, LegalEntityID: entity, PrincipalID: actor, ResponseRevisionID: source.ResponseRevisionID, Limit: 100}
	history, err := store.ListDocuments(ctx, q)
	if err != nil || len(history.Items) != 2 {
		t.Fatalf("original reused occurrences: %+v %v", history, err)
	}
	var amendmentID string
	if err := pool.QueryRow(ctx, `SELECT md5('document-amendment-revision')::uuid::text`).Scan(&amendmentID); err != nil {
		t.Fatal(err)
	}
	q.ResponseRevisionID = amendmentID
	current, err := store.ListDocuments(ctx, q)
	if err != nil || len(current.Items) != 2 {
		t.Fatalf("amended reused occurrences: %+v %v", current, err)
	}
	for _, old := range history.Items {
		if old.Current {
			t.Fatalf("superseded occurrence current: %+v", old)
		}
		found := false
		for _, newer := range current.Items {
			if newer.ArtifactID == old.ArtifactID {
				found = true
				if !newer.Current || newer.SubmissionID == old.SubmissionID || newer.RequestID != source.RequestID {
					t.Fatalf("reused occurrence lost immutable scope: %+v", newer)
				}
			}
		}
		if !found {
			t.Fatalf("reused artifact missing: %+v", old)
		}
		if old.FileName == "contributor.pdf" && old.ArtifactRequestID == old.RequestID {
			t.Fatal("contributor upload request replaced by submission request")
		}
	}
	// Inspect the executable query, including scope predicates and keyset bound,
	// rather than a simplified surrogate. This is a representative small-fixture
	// plan, explicitly not the separate 200,000-occurrence release benchmark.
	var planJSON []byte
	err = pool.QueryRow(ctx, `EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) `+documentInventorySQL(), tenant, entity, actor, now, "", "", amendmentID, false, "", "", time.Time{}, "", "", "", "", 3, false).Scan(&planJSON)
	if err != nil {
		t.Fatal(err)
	}
	var plans []map[string]any
	if err = json.Unmarshal(planJSON, &plans); err != nil {
		t.Fatal(err)
	}
	plan := plans[0]["Plan"].(map[string]any)
	if plan["Node Type"] != "Limit" || plan["Actual Rows"].(float64) != 2 {
		t.Fatalf("unexpected bounded plan: %s", planJSON)
	}
	indexes := []string{}
	var walk func(map[string]any)
	walk = func(node map[string]any) {
		if name, ok := node["Index Name"].(string); ok {
			indexes = append(indexes, name)
		}
		if children, ok := node["Plans"].([]any); ok {
			for _, child := range children {
				walk(child.(map[string]any))
			}
		}
	}
	walk(plan)
	if len(indexes) == 0 {
		t.Fatal("representative exact-response plan has no index access")
	}
	t.Logf("submitted document exact-response EXPLAIN: rows=%v execution_ms=%v shared_hit_blocks=%v indexes=%s", plan["Actual Rows"], plans[0]["Execution Time"], plan["Shared Hit Blocks"], strings.Join(indexes, ","))
}
