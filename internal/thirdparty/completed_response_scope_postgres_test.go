//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func assertPostgresWorkResponseSummaryScope(t *testing.T, pool *pgxpool.Pool, store *evidence.PostgresDistributionStore, revisionID, requestID, targetID, workID, linkID string) {
	t.Helper()
	ctx := context.Background()
	query := evidence.CompletedResponseQuery{TenantID: thirdPartyTenantID, LegalEntityID: thirdPartyEntityA, PrincipalID: thirdPartyPrincipal, Sort: evidence.ResponseSortNewest, Limit: 1}
	check := func(stage string, visible bool) {
		t.Helper()
		page, err := store.ListCompletedResponses(ctx, query)
		want := 0
		if visible {
			want = 1
		}
		if err != nil || len(page.Items) != want || page.NextCursor != "" {
			t.Fatalf("%s summary page: %+v %v", stage, page, err)
		}
		_, _, err = store.GetCompletedResponse(ctx, query.TenantID, query.LegalEntityID, query.PrincipalID, revisionID)
		if visible && err != nil || !visible && err != evidence.ErrNotFound {
			t.Fatalf("%s exact response: %v", stage, err)
		}
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	check("restricted target", false)
	exec(`UPDATE programs SET scope='{}' WHERE id=$1::uuid`, targetID)
	check("relationship and work owner", true)
	for _, test := range []struct {
		name, change, restore string
		args                  []any
	}{
		{"mismatched origin", `UPDATE third_party_work_capture_links SET origin_version=3,sequence=3 WHERE id=$1::uuid`, `UPDATE third_party_work_capture_links SET origin_version=1,sequence=1 WHERE id=$1::uuid`, []any{linkID}},
		{"mismatched subject", `UPDATE capture_requests SET subject_id='unrelated-relationship' WHERE id=$1::uuid`, `UPDATE capture_requests SET subject_id=(SELECT subject_id::text FROM capture_form_distributions WHERE id=capture_requests.distribution_id) WHERE id=$1::uuid`, []any{requestID}},
		{"malformed target scope", `UPDATE programs SET scope='{"access":"UNKNOWN"}' WHERE id=$1::uuid`, `UPDATE programs SET scope='{}' WHERE id=$1::uuid`, []any{targetID}},
	} {
		exec(test.change, test.args...)
		check(test.name, false)
		exec(test.restore, test.args...)
	}
	const reviewer = "33333333-3333-7333-8333-333333333399"
	exec(`INSERT INTO capture_requests SELECT (jsonb_populate_record(NULL::capture_requests,to_jsonb(req)||jsonb_build_object('id',md5('summary-other-entity')::uuid,'legal_entity_id',$2::uuid,'distribution_id',NULL,'form_template_id',NULL,'form_template_version',NULL,'origin_version',99))).* FROM capture_requests req WHERE id=$1::uuid;
 UPDATE capture_submissions SET request_id=md5('summary-other-entity')::uuid WHERE id=(SELECT submission_id FROM capture_response_revisions WHERE id=$3::uuid)`, pgx.QueryExecModeSimpleProtocol, requestID, thirdPartyEntityB, revisionID)
	check("mismatched request entity", false)
	exec(`UPDATE capture_submissions SET request_id=$1::uuid WHERE id=(SELECT submission_id FROM capture_response_revisions WHERE id=$2::uuid);
 DELETE FROM capture_requests WHERE id=md5('summary-other-entity')::uuid`, pgx.QueryExecModeSimpleProtocol, requestID, revisionID)
	exec(`INSERT INTO principals(id,tenant_id,kind,display_name,status) VALUES($1::uuid,$2::uuid,'PERSON','Response reviewer','ACTIVE');
 INSERT INTO responsibility_assignments(tenant_id,legal_entity_id,principal_id,responsibility,object_type,object_id,priority,valid_from,policy_version,decision_type)
 SELECT tenant_id,legal_entity_id,$1::uuid,'REVIEWER','VENDOR_RELATIONSHIP',relationship_id,100,$4,'response-summary:v1','thirdparty.work.review' FROM third_party_work_requests WHERE id=$3::uuid;
 INSERT INTO authority_grants(tenant_id,legal_entity_id,principal_id,decision_type,limits,valid_from,policy_version)
 VALUES($2::uuid,$5::uuid,$1::uuid,'thirdparty.work.review','{"max_materiality":3}',$4,'response-summary:v1')`, pgx.QueryExecModeSimpleProtocol, reviewer, thirdPartyTenantID, workID, time.Now().Add(-time.Hour), thirdPartyEntityA)
	query.PrincipalID = reviewer
	check("current reviewer route", true)
	exec(`UPDATE responsibility_assignments SET valid_from=$2 WHERE principal_id=$1::uuid`, reviewer, time.Now().Add(time.Hour))
	check("revoked reviewer route", false)
	exec(`UPDATE third_party_work_requests SET reviewer_principal_id=$2::uuid WHERE id=$1::uuid`, workID, reviewer)
	check("named work reviewer", true)
	exec(`UPDATE third_party_work_requests SET reviewer_principal_id=NULL WHERE id=$1::uuid`, workID)
	query.PrincipalID = thirdPartyPrincipal
	exec(`UPDATE programs SET scope='{"access":"RESTRICTED","allowed_principal_ids":[]}' WHERE id=$1::uuid`, targetID)
}
