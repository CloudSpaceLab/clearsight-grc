//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"

	"github.com/jackc/pgx/v5"
)

func TestPostgresVendorFormAttentionAndOutdatedPopulation(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant, entity, owner := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	form, vendor, relationship := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	now := time.Now().UTC()
	prefix := "vendor-attention-" + tenant
	setupResponseWorkspaceFixture(t, ctx, pool, tenant, prefix, entity, owner, form, now)
	t.Cleanup(func() { cleanupVendorFormsFixture(context.Background(), pool, tenant) })
	seedCompletedResponseRows(t, ctx, pool, tenant, entity, owner, form, prefix, 2, now)
	_, err := pool.Exec(ctx, `
 INSERT INTO third_parties(id,tenant_id,legal_name,status,created_at,updated_at,version) VALUES($1::uuid,$2::uuid,'Sample vendor','ACTIVE',$6,$6,1);
 INSERT INTO third_party_relationships(id,tenant_id,legal_entity_id,vendor_id,service_name,business_owner_principal_id,criticality,privacy_role,status,created_at,updated_at,version) VALUES($3::uuid,$2::uuid,$4::uuid,$1::uuid,'Processing',$5::uuid,'IMPORTANT','PROCESSOR','PROPOSED',$6,$6,1);
 UPDATE capture_form_distributions SET subject_type='VENDOR_RELATIONSHIP',subject_id=$3::uuid WHERE tenant_id=$2::uuid;
 UPDATE capture_requests SET subject_type='VENDOR_RELATIONSHIP',subject_id=$3,status='IN_PROGRESS',deadline=$6::timestamptz-interval '1 day',scoring_mode='RISK',sections='[{"id":"checks","title":"Checks"}]',fields='[{"id":"status","section_id":"checks","label":"Security assurance","type":"yes_no","scoring":{"id":"status","weight":1,"answer_scores":{"Yes":0,"No":100}}},{"id":"certificate","section_id":"checks","label":"Certificate","type":"vendor_document"},{"id":"hidden","section_id":"checks","label":"Hidden evidence","type":"vendor_document","condition":{"field_id":"status","operator":"EQUALS","values":["Yes"]}}]' WHERE tenant_id=$2::uuid;
 UPDATE capture_response_revisions SET score_result='{"state":"FINAL","mode":"RISK","band":"HIGH","assessment_review_count":1}' WHERE tenant_id=$2::uuid`, pgx.QueryExecModeSimpleProtocol, vendor, tenant, relationship, entity, owner, now)
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), postgresTestRecipientProtector{})
	q := VendorFormsQuery{TenantID: tenant, LegalEntityID: entity, PrincipalID: owner, RelationshipIDs: []string{relationship}, Limit: 1}
	for _, tc := range []struct {
		name, expiry string
		want         *bool
		count        int
	}{
		{"expired", now.AddDate(0, 0, -1).Format("2006-01-02"), vendorTestBool(true), 2},
		{"today", now.Format("2006-01-02"), vendorTestBool(false), 0},
		{"unknown", "", nil, 0},
		{"invalid", "not-a-date", nil, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			answers, _ := json.Marshal(map[string]any{"status": map[string]any{"text": "No"}, "certificate": map[string]any{"document": map[string]any{"artifact_id": mustResponseWorkspaceID(t), "expires_on": tc.expiry}}, "hidden": map[string]any{"document": map[string]any{"artifact_id": mustResponseWorkspaceID(t), "expires_on": "2020-01-01"}}})
			if _, err := pool.Exec(ctx, `UPDATE capture_submissions SET answers=$2::jsonb WHERE tenant_id=$1::uuid`, tenant, string(answers)); err != nil {
				t.Fatal(err)
			}
			page, err := store.ListVendorForms(ctx, q)
			if err != nil || len(page.Items) != 1 {
				t.Fatalf("page=%+v err=%v", page, err)
			}
			row := page.Items[0]
			if (row.Outdated == nil) != (tc.want == nil) || (row.Outdated != nil && *row.Outdated != *tc.want) {
				t.Fatalf("freshness %+v want %v", row, tc.want)
			}
			if row.ResponseState != "SUBMITTED" || row.AssessmentState != "AWAITING_REVIEW" {
				t.Fatalf("completion/review changed: %+v", row)
			}
			found := false
			for _, item := range row.AttentionItems {
				if item.FieldID == "hidden" {
					t.Fatalf("hidden field disclosed: %+v", row)
				}
				if item.FieldID == "status" && item.State == "GAP" && item.Source == "RESPONSE" {
					found = true
				}
			}
			if !found {
				t.Fatalf("pending review suppressed automatic failure: %+v", row)
			}
			summary, err := store.VendorFormSummaries(ctx, q)
			if err != nil || len(summary) != 1 || summary[0].OutdatedForms != tc.count || summary[0].SubmittedForms != 2 || summary[0].OutstandingForms != 0 {
				t.Fatalf("summary %+v err %v", summary, err)
			}
			unknown := 0
			if tc.want == nil {
				unknown = 2
			}
			if summary[0].FreshnessUnknownForms != unknown {
				t.Fatalf("unknown freshness %+v want %d", summary, unknown)
			}
			denied := q
			denied.PrincipalID = mustResponseWorkspaceID(t)
			rows, err := store.ListVendorForms(ctx, denied)
			if err != nil || len(rows.Items) != 0 {
				t.Fatalf("denied rows %+v %v", rows, err)
			}
			counts, err := store.VendorFormSummaries(ctx, denied)
			if err != nil || counts[0].OutdatedForms != 0 {
				t.Fatalf("denied counts %+v %v", counts, err)
			}
			for _, scope := range []string{"tenant", "entity", "relationship"} {
				denied = q
				switch scope {
				case "tenant":
					denied.TenantID = mustResponseWorkspaceID(t)
				case "entity":
					denied.LegalEntityID = mustResponseWorkspaceID(t)
				case "relationship":
					denied.RelationshipIDs = []string{mustResponseWorkspaceID(t)}
				}
				rows, err := store.ListVendorForms(ctx, denied)
				if err != nil || len(rows.Items) != 0 {
					t.Fatalf("%s leak %+v %v", scope, rows, err)
				}
				counts, err := store.VendorFormSummaries(ctx, denied)
				if err != nil || counts[0].SubmittedForms != 0 || counts[0].OutdatedForms != 0 || counts[0].FreshnessUnknownForms != 0 {
					t.Fatalf("%s count leak %+v %v", scope, counts, err)
				}
			}
		})
	}
	t.Run("custom compliance failure before review", func(t *testing.T) {
		req := Request{ScoringMode: formcontract.ScoringCompliance, Sections: []formcontract.Section{{ID: "checks", Title: "Checks"}}, Fields: []Field{{ID: "status", SectionID: "checks", Label: "Contract assurance", Type: "yes_no"}}, ScoreProfile: &formcontract.ScoreProfile{Version: "custom-v4", Mode: formcontract.ScoringCompliance, Direction: formcontract.DirectionLowIsPoor, Bands: formcontract.DefaultConcernBands(), Contributions: []formcontract.ScoreContribution{{ID: "base", Label: "Base check", Weight: 1, Predicate: formcontract.Predicate{FieldID: "status", Operator: formcontract.PredicateEquals, Values: []string{"No"}}, MatchPoints: 100, NonMatchPoints: 100, Missing: formcontract.MissingIndeterminate}}, Rules: []formcontract.ScoreRule{{ID: "assurance-gap", Label: "Contracted assurance absent", Predicate: formcontract.Predicate{FieldID: "status", Operator: formcontract.PredicateEquals, Values: []string{"No"}}, Effect: formcontract.RuleEffect{Kind: formcontract.EffectFloor, Value: 100}}}}}
		revision, err := buildResponseRevision(req, AccessAssurance(""), nil, formcontract.TextAnswers(map[string]string{"status": "No"}))
		if err != nil {
			t.Fatal(err)
		}
		revision.Score.AssessmentReviewCount = 1
		fields, _ := json.Marshal(req.Fields)
		profile, _ := json.Marshal(req.ScoreProfile)
		score, _ := json.Marshal(revision.Score)
		_, err = pool.Exec(ctx, `UPDATE capture_requests SET scoring_mode='COMPLIANCE',fields=$2::jsonb,score_profile=$3::jsonb WHERE tenant_id=$1::uuid; UPDATE capture_response_revisions SET score_result=$4::jsonb WHERE tenant_id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, tenant, string(fields), string(profile), string(score))
		if err != nil {
			t.Fatal(err)
		}
		page, err := store.ListVendorForms(ctx, q)
		if err != nil || len(page.Items) != 1 {
			t.Fatalf("page %+v %v", page, err)
		}
		row := page.Items[0]
		if row.AssessmentState != "AWAITING_REVIEW" || row.RequiredCount == nil || len(row.AttentionItems) != 1 || row.AttentionItems[0].RuleID != "assurance-gap" || row.AttentionItems[0].Source != "RESPONSE" {
			t.Fatalf("custom rule failure suppressed %+v", row)
		}
	})
}

func vendorTestBool(value bool) *bool { return &value }

func TestPostgresVendorFormHeldEvidenceAndReviewFreshness(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant, entity, owner := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	form, vendor, relationship, assessment := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	now := time.Now().UTC()
	prefix := "vendor-held-attention-" + tenant
	setupResponseWorkspaceFixture(t, ctx, pool, tenant, prefix, entity, owner, form, now)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM third_party_assessments WHERE tenant_id=$1::uuid`, tenant)
		cleanupVendorFormsFixture(context.Background(), pool, tenant)
	})
	seedCompletedResponseRows(t, ctx, pool, tenant, entity, owner, form, prefix, 2, now)
	_, err := pool.Exec(ctx, `
 INSERT INTO third_parties(id,tenant_id,legal_name,status,created_at,updated_at,version) VALUES($1::uuid,$2::uuid,'Sample vendor','ACTIVE',$6,$6,1);
 INSERT INTO third_party_relationships(id,tenant_id,legal_entity_id,vendor_id,service_name,business_owner_principal_id,criticality,privacy_role,status,created_at,updated_at,version) VALUES($3::uuid,$2::uuid,$4::uuid,$1::uuid,'Processing',$5::uuid,'IMPORTANT','PROCESSOR','PROPOSED',$6,$6,1);
 UPDATE capture_form_distributions SET subject_type='VENDOR_RELATIONSHIP',subject_id=$3::uuid WHERE tenant_id=$2::uuid;
 UPDATE capture_requests SET subject_type='VENDOR_RELATIONSHIP',subject_id=$3,status='READY',scoring_mode='NONE',fields='[{"id":"certificate","section_id":"general","label":"Current certificate","type":"vendor_document","required":true}]' WHERE tenant_id=$2::uuid;
 UPDATE capture_submissions SET answers='{}' WHERE tenant_id=$2::uuid;
 INSERT INTO third_party_assessments(id,tenant_id,legal_entity_id,relationship_id,review_kind,stable_episode_key,status,form_template_id,form_template_version,review_due_at,started_by_principal_id,started_at,version,created_at,updated_at) VALUES($7::uuid,$2::uuid,$4::uuid,$3::uuid,'ONBOARDING',repeat('a',64),'SUBMITTED',$8::uuid,1,$6::timestamptz+interval '1 day',$5::uuid,$6,1,$6,$6)`, pgx.QueryExecModeSimpleProtocol, vendor, tenant, relationship, entity, owner, now, assessment, form)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := pool.Query(ctx, `SELECT req.id::text,s.id::text,r.id::text,req.distribution_id::text FROM capture_requests req JOIN capture_submissions s ON s.request_id=req.id JOIN capture_response_revisions r ON r.submission_id=s.id WHERE req.tenant_id=$1::uuid ORDER BY req.id`, tenant)
	if err != nil {
		t.Fatal(err)
	}
	type ids struct{ request, submission, revision, distribution string }
	var values []ids
	for rows.Next() {
		var v ids
		if err := rows.Scan(&v.request, &v.submission, &v.revision, &v.distribution); err != nil {
			t.Fatal(err)
		}
		values = append(values, v)
	}
	rows.Close()
	if len(values) != 2 {
		t.Fatal(values)
	}
	source, target := values[0], values[1]
	repo := NewPostgresRepository(pool)
	store := NewPostgresDistributionStore(repo, postgresTestRecipientProtector{})
	artifact, err := repo.CreateArtifact(ctx, Artifact{ID: mustResponseWorkspaceID(t), TenantID: tenant, RequestID: source.request, SubmissionID: source.submission, FileName: "certificate.pdf", MediaType: "application/pdf", SizeBytes: 10, SHA256: strings.Repeat("a", 64), StorageKey: prefix, Status: ArtifactAvailable, CreatedBy: owner, CreatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	held := captureHeldField(t, now)
	held.SectionID = "general"
	held.CollectionResolution.Source = DocumentOccurrence{ID: "source", ArtifactID: artifact.ID, RequestID: source.request, ArtifactRequestID: source.request, SubmissionID: source.submission, ResponseRevisionID: source.revision, DistributionID: source.distribution, FieldID: "certificate", RelationshipID: relationship, AssessmentID: assessment, ArtifactStatus: ArtifactAvailable, Current: true, SHA256: artifact.SHA256, SizeBytes: artifact.SizeBytes}
	held.CollectionResolution.SourceArtifactRequestID = source.request
	held.CollectionResolution.Document.ArtifactID = artifact.ID
	_, err = pool.Exec(ctx, `UPDATE capture_requests SET status='SUBMITTED' WHERE tenant_id=$3::uuid; UPDATE capture_requests SET origin_type='THIRD_PARTY_ASSESSMENT',origin_id=$2,origin_version=1 WHERE id=$1::uuid;
 INSERT INTO third_party_assessment_request_links(tenant_id,legal_entity_id,assessment_id,request_id,purpose,sequence,origin_type,origin_id,origin_sequence,created_at) VALUES($3::uuid,$4::uuid,$2::uuid,$1::uuid,'INITIAL',1,'THIRD_PARTY_ASSESSMENT',$2::uuid,1,$5);
 UPDATE capture_submissions SET answers=jsonb_build_object('certificate',jsonb_build_object('document',jsonb_build_object('artifact_id',$6::text,'expires_on',to_char($5::timestamptz+interval '1 year','YYYY-MM-DD')))) WHERE id=$7::uuid`, pgx.QueryExecModeSimpleProtocol, source.request, assessment, tenant, entity, now, artifact.ID, source.submission)
	if err != nil {
		t.Fatal(err)
	}
	q := VendorFormsQuery{TenantID: tenant, LegalEntityID: entity, PrincipalID: owner, RelationshipIDs: []string{relationship}, Limit: 25}
	for _, tc := range []struct {
		name              string
		expiry            string
		status            string
		want              *bool
		outdated, unknown int
		source            string
	}{
		{"held current", now.AddDate(0, 0, 1).Format("2006-01-02"), "", vendorTestBool(false), 0, 0, "RESPONSE"},
		{"held expired", now.AddDate(0, 0, -1).Format("2006-01-02"), "", vendorTestBool(true), 1, 0, "RESPONSE"},
		{"bank review expired", now.AddDate(0, 0, 1).Format("2006-01-02"), "EXPIRED", vendorTestBool(true), 2, 0, "REVIEW"},
		{"bank review rejected", now.AddDate(0, 0, 1).Format("2006-01-02"), "REJECTED", vendorTestBool(false), 0, 0, "REVIEW"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			held.CollectionResolution.Document.ExpiresOn = tc.expiry
			raw, _ := json.Marshal([]Field{held})
			if _, err := pool.Exec(ctx, `UPDATE capture_requests SET fields=$2::jsonb WHERE id=$1::uuid`, target.request, string(raw)); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `DELETE FROM third_party_documents WHERE tenant_id=$1::uuid`, tenant); err != nil {
				t.Fatal(err)
			}
			if tc.status != "" {
				_, err := pool.Exec(ctx, `INSERT INTO third_party_documents(id,tenant_id,legal_entity_id,relationship_id,assessment_id,request_id,artifact_id,document_type,evidence_class,status,expires_on,validated_by_principal_id,validated_at,created_at,updated_at,version) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7::uuid,'Certification','BANK_VALIDATED',$8,$9::date,CASE WHEN $8='REJECTED' THEN $10::uuid ELSE NULL END,CASE WHEN $8='REJECTED' THEN $11::timestamptz ELSE NULL END,$11,$11,1)`, mustResponseWorkspaceID(t), tenant, entity, relationship, assessment, source.request, artifact.ID, tc.status, now.AddDate(0, 0, 2).Format("2006-01-02"), owner, now)
				if err != nil {
					t.Fatal(err)
				}
			}
			page, err := store.ListVendorForms(ctx, q)
			if err != nil || len(page.Items) != 2 {
				t.Fatalf("page %+v %v", page, err)
			}
			for _, row := range page.Items {
				if row.RequestID != target.request {
					continue
				}
				if row.Outdated == nil || *row.Outdated != *tc.want {
					t.Fatalf("target freshness %+v", row)
				}
				for _, item := range row.AttentionItems {
					if item.State == "EXPIRED" && item.Source != tc.source {
						t.Fatalf("source %+v", row)
					}
				}
				if tc.status == "REJECTED" {
					found := false
					for _, item := range row.AttentionItems {
						found = found || item.State == "GAP" && item.Source == "REVIEW"
					}
					if !found {
						t.Fatalf("rejected review omitted %+v", row)
					}
				}
			}
			summary, err := store.VendorFormSummaries(ctx, q)
			if err != nil || summary[0].OutdatedForms != tc.outdated || summary[0].FreshnessUnknownForms != tc.unknown {
				t.Fatalf("summary %+v %v", summary, err)
			}
		})
	}
}
