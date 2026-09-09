//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"github.com/jackc/pgx/v5"
	"testing"
	"time"
)

func TestPostgresVendorFormsRetirementUsesExactSubmittedFieldLinks(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant := mustResponseWorkspaceID(t)
	entity := mustResponseWorkspaceID(t)
	owner := mustResponseWorkspaceID(t)
	form := mustResponseWorkspaceID(t)
	vendor := mustResponseWorkspaceID(t)
	rel := mustResponseWorkspaceID(t)
	assessment := mustResponseWorkspaceID(t)
	otherForm := mustResponseWorkspaceID(t)
	now := time.Now().UTC()
	prefix := "vendor-retirement-" + tenant
	setupResponseWorkspaceFixture(t, ctx, pool, tenant, prefix, entity, owner, form, now)
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM third_party_assessment_request_links WHERE tenant_id=$1::uuid;DELETE FROM third_party_assessments WHERE tenant_id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, tenant)
		cleanupVendorFormsFixture(context.Background(), pool, tenant)
	}()
	seedCompletedResponseRows(t, ctx, pool, tenant, entity, owner, form, prefix, 2, now)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, append([]any{pgx.QueryExecModeSimpleProtocol}, args...)...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO third_parties(id,tenant_id,legal_name,status,created_at,updated_at,version) VALUES($1::uuid,$2::uuid,'Sample vendor','ACTIVE',$6,$6,1); INSERT INTO third_party_relationships(id,tenant_id,legal_entity_id,vendor_id,service_name,business_owner_principal_id,criticality,privacy_role,status,created_at,updated_at,version) VALUES($3::uuid,$2::uuid,$4::uuid,$1::uuid,'Processing',$5::uuid,'IMPORTANT','PROCESSOR','PROPOSED',$6,$6,1)`, vendor, tenant, rel, entity, owner, now)
	exec(`INSERT INTO third_party_assessments(id,tenant_id,legal_entity_id,relationship_id,review_kind,stable_episode_key,status,form_template_id,form_template_version,review_due_at,started_by_principal_id,started_at,version,created_at,updated_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'ONBOARDING',repeat('a',64),'COLLECTING',$5::uuid,1,$7::timestamptz+interval '1 day',$6::uuid,$7,1,$7,$7);
 UPDATE capture_form_distributions SET subject_type='VENDOR_RELATIONSHIP',subject_id=$4::uuid WHERE tenant_id=$2::uuid;
 UPDATE capture_requests SET subject_type='VENDOR_RELATIONSHIP',subject_id=$4,origin_type='THIRD_PARTY_ASSESSMENT',origin_id=$1,origin_version=CASE WHEN id=md5($8||':request:1')::uuid THEN 1 ELSE 2 END,fields='[{"id":"a","label":"Control A","type":"short_text"},{"id":"b","label":"Control B","type":"short_text"}]' WHERE tenant_id=$2::uuid;
 INSERT INTO third_party_assessment_request_links(tenant_id,legal_entity_id,assessment_id,request_id,purpose,sequence,origin_type,origin_id,origin_sequence,is_current,created_at) SELECT $2::uuid,$3::uuid,$1::uuid,id,'CLARIFICATION',origin_version,'THIRD_PARTY_ASSESSMENT',$1::uuid,origin_version,false,$7 FROM capture_requests WHERE tenant_id=$2::uuid;
 UPDATE capture_submissions SET answers='{}' WHERE tenant_id=$2::uuid;
 UPDATE capture_response_revisions SET score_result='{"state":"FINAL","final":true,"band":"HIGH","coverage":1}' WHERE tenant_id=$2::uuid`, assessment, tenant, entity, rel, form, owner, now, prefix)
	exec(`INSERT INTO monitoring_form_templates SELECT (jsonb_populate_record(NULL::monitoring_form_templates,to_jsonb(f)||jsonb_build_object('revision_id',$2::uuid,'id',$2::uuid,'code',f.code||'-OTHER'))).* FROM monitoring_form_templates f WHERE id=$1::uuid AND version=1`, form, otherForm)
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), nil)
	var oldID, newID string
	if err := pool.QueryRow(ctx, `SELECT md5($1||':request:1')::uuid::text,md5($1||':request:2')::uuid::text`, prefix).Scan(&oldID, &newID); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, fields, change, want string }{
		{"full", `[{"id":"a"},{"id":"b"}]`, "", "HISTORICAL"},
		{"partial", `[{"id":"a"}]`, "", "PARTIALLY_REPLACED"},
		{"supplemental", `[{"id":"c"}]`, "", "CURRENT"},
		{"other form", `[{"id":"a"},{"id":"b"}]`, "form", "CURRENT"},
		{"other subject", `[{"id":"a"},{"id":"b"}]`, "subject", "CURRENT"},
		{"unlinked", `[{"id":"a"},{"id":"b"}]`, "unlinked", "CURRENT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			exec(`UPDATE capture_requests SET fields=$2::jsonb,form_template_id=$3::uuid,form_template_version=1,subject_id=$4,origin_version=2 WHERE id=$1::uuid`, newID, tc.fields, form, rel)
			switch tc.change {
			case "form":
				exec(`UPDATE capture_requests SET form_template_id=$2::uuid WHERE id=$1::uuid`, newID, otherForm)
			case "subject":
				exec(`UPDATE capture_requests SET subject_id='different-service' WHERE id=$1::uuid`, newID)
			case "unlinked":
				exec(`UPDATE capture_requests SET origin_version=3 WHERE id=$1::uuid`, newID)
			}
			q := VendorFormsQuery{TenantID: tenant, LegalEntityID: entity, PrincipalID: owner, RelationshipIDs: []string{rel}, Limit: 25}
			page, err := store.ListVendorForms(ctx, q)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, row := range page.Items {
				if row.RequestID == oldID {
					found = true
					if row.ResponseCurrency != tc.want || row.Current != (tc.want != "HISTORICAL") {
						t.Fatalf("old row=%+v want %s", row, tc.want)
					}
					if tc.want == "PARTIALLY_REPLACED" && (row.Outdated == nil || !*row.Outdated) {
						t.Fatalf("partial response freshness unknown: %+v", row)
					}
				}
			}
			if !found {
				t.Fatal("original response disappeared")
			}
			summaries, err := store.VendorFormSummaries(ctx, q)
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == "PARTIALLY_REPLACED" && (summaries[0].PartiallyReplacedForms != 1 || summaries[0].OutdatedForms != 1 || summaries[0].HighestConcern != "HIGH") {
				t.Fatalf("partial concern lost: %+v", summaries)
			}
		})
	}
}
