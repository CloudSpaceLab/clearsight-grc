//go:build postgres && postgresintegration

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/demodocuments"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sampleTestTenant = "00000000-0000-4000-8000-000000000001"
const sampleTestEntity = "00000000-0000-4000-8000-000000000002"
const sampleTestOwner = "00000000-0000-4000-8000-000000000107"
const sampleTestChecker = "00000000-0000-4000-8000-000000000106"

func sampleTestSetup(t *testing.T) (*pgxpool.Pool, config.Config, bankverticals.SeedConfig) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	// This suite has the same disposable-database requirement as internal integration suites.
	if _, err = pool.Exec(context.Background(), "TRUNCATE tenants CASCADE"); err != nil {
		t.Fatal(err)
	}
	foundation, err := os.ReadFile("../../deploy/scripts/seed-demo-foundation.sh")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.SplitN(strings.SplitN(string(foundation), "<<'SQL'", 2)[1], "\nSQL", 2)[0]
	sql = strings.ReplaceAll(sql, ":'demo_staff_email'", "''")
	if _, err = pool.Exec(context.Background(), sql); err != nil {
		t.Fatal(err)
	}
	var key, hmac [32]byte
	key[0], hmac[0] = 1, 2
	cfg := config.Config{Environment: "development", DemoMode: true, ArtifactRoot: t.TempDir(), CapturePublicBaseURL: "http://localhost:8080", CaptureSessionTTL: 20 * time.Minute, MaxArtifactBytes: 20 << 20,
		RecipientSecurity: config.RecipientSecurityConfig{ActiveKeyID: "test", Keyring: map[string][32]byte{"test": key}, AccessHMACKey: hmac}}
	seed := bankverticals.SeedConfig{TenantID: sampleTestTenant, LegalEntityID: sampleTestEntity, ActorID: sampleTestOwner, OwnerPrincipalID: sampleTestOwner, ReviewerPrincipalID: sampleTestChecker}
	return pool, cfg, seed
}

func sampleTestWorker(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	p := thirdparty.NewAssessmentProvisioner(thirdparty.NewPostgresRepository(pool), continuity.NewService(continuity.NewPostgresRepository(pool)), "document-sample-test")
	p.ConfigureAuthority(authority.NewEffectivePostgresService(pool))
	go func() {
		defer close(done)
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = p.Maintain(ctx, time.Now().UTC(), 1)
			}
		}
	}()
	t.Cleanup(func() { cancel(); <-done })
}

func TestDocumentSamplesPersistTwoRevisionsAndRerunWithoutWrites(t *testing.T) {
	pool, cfg, seed := sampleTestSetup(t)
	sampleTestWorker(t, pool)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	before := time.Now().UTC()
	receipt, err := installDocumentSamples(ctx, cfg, pool, seed)
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.ResponseRevisionIDs) != 2 || receipt.ArtifactCount != 6 || receipt.AssessmentStatus != "COLLECTING" {
		t.Fatalf("receipt: %+v", receipt)
	}
	repo := evidence.NewPostgresDistributionStore(evidence.NewPostgresRepository(pool), nil)
	for _, q := range []evidence.DocumentQuery{
		{FormTemplateID: receipt.FormTemplateID}, {RelationshipID: receipt.RelationshipID}, {ResponseRevisionID: receipt.ResponseRevisionIDs[1]},
	} {
		q.TenantID, q.LegalEntityID, q.PrincipalID, q.Limit, q.CurrentOnly = sampleTestTenant, sampleTestEntity, sampleTestOwner, 11, true
		page, err := repo.ListDocuments(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) != 5 {
			t.Fatalf("document count %d", len(page.Items))
		}
		for _, doc := range page.Items {
			if doc.UploadedAt.Before(before) || doc.SubmittedAt.Before(before) || doc.ArtifactStatus != evidence.ArtifactStoredUnscanned {
				t.Fatalf("not a normal current upload: %+v", doc)
			}
		}
	}
	snapshotBefore := sampleTestSnapshot(t, pool)
	var countsBefore, countsAfter string
	countSQL := `SELECT json_build_array((SELECT count(*) FROM capture_access_routes),(SELECT count(*) FROM capture_artifacts),(SELECT count(*) FROM capture_submissions),(SELECT count(*) FROM routing_policy_versions))::text`
	if err = pool.QueryRow(ctx, countSQL).Scan(&countsBefore); err != nil {
		t.Fatal(err)
	}
	repeated, err := installDocumentSamples(ctx, cfg, pool, seed)
	if err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, countSQL).Scan(&countsAfter); err != nil {
		t.Fatal(err)
	}
	if countsBefore != countsAfter || !repeated.AlreadyInstalled {
		t.Fatalf("rerun changed population: %s => %s, %+v", countsBefore, countsAfter, repeated)
	}
	if snapshotBefore != sampleTestSnapshot(t, pool) || !sameSampleJSON(repeated.ResponseRevisionIDs, receipt.ResponseRevisionIDs) {
		t.Fatal("completed rerun changed stored records")
	}
	query := evidence.DocumentQuery{TenantID: sampleTestTenant, LegalEntityID: sampleTestEntity, PrincipalID: sampleTestOwner, RelationshipID: receipt.RelationshipID, Limit: 11}
	history, err := repo.ListDocuments(ctx, query)
	if err != nil || len(history.Items) != 10 {
		t.Fatalf("historical occurrences=%d: %v", len(history.Items), err)
	}
	query.PrincipalID = "00000000-0000-4000-8000-000000000101"
	denied, err := repo.ListDocuments(ctx, query)
	if err != nil || len(denied.Items) != 0 {
		t.Fatalf("unrelated principal can read restricted sample response: %v", err)
	}
}

// Hash complete rows without printing recipient or route secrets on failures.
func sampleTestSnapshot(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	hash := sha256.New()
	for _, table := range []string{"third_parties", "third_party_relationships", "third_party_assessments", "monitoring_form_templates", "capture_requests", "capture_form_distributions", "capture_distribution_recipients", "capture_response_workspaces", "capture_response_workspace_edits", "capture_response_revisions", "capture_access_routes", "capture_distribution_sessions", "capture_submissions", "capture_artifacts", "routing_policies", "routing_policy_versions"} {
		var rows []byte
		if err := pool.QueryRow(context.Background(), `SELECT COALESCE(jsonb_agg(r ORDER BY r::text),'[]'::jsonb) FROM (SELECT to_jsonb(t) r FROM `+table+` t) s`).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		_, _ = hash.Write(rows)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func TestDocumentSamplesStopOnEditedOrUnknownRecordsWithoutWrites(t *testing.T) {
	for _, test := range []struct{ name, sql string }{
		{"request_known_fact", `UPDATE capture_requests SET known_facts=known_facts||'{"vendor_legal_name":"Operator correction"}'::jsonb`},
		{"request_instruction", `UPDATE capture_requests SET why_you='Operator instruction'`},
		{"request_estimate", `UPDATE capture_requests SET estimated_minutes=60`},
		{"request_period", `UPDATE capture_requests SET collection_period_start='2026-01-01',collection_period_end='2026-12-31'`},
		{"missing_assessment_link", `DELETE FROM third_party_assessment_request_links`},
		{"revoked_response_access", `UPDATE capture_access_routes SET revoked_at=clock_timestamp()`},
		{"typed_answer", `UPDATE capture_submissions SET answers=answers||'{"contact":{"text":"Operator contact"}}'::jsonb`},
		{"unknown_submission", `INSERT INTO capture_submissions(id,tenant_id,request_id,channel,answers,submitted_at,distribution_id) SELECT md5('extra-submission')::uuid,tenant_id,request_id,channel,answers,submitted_at,distribution_id FROM capture_submissions LIMIT 1`},
		{"workspace_extra_edit", `INSERT INTO capture_response_workspace_edits(id,tenant_id,legal_entity_id,distribution_id,workspace_id,recipient_id,request_id,base_version,result_version,patch,created_at) SELECT md5('extra-edit')::uuid,tenant_id,legal_entity_id,distribution_id,workspace_id,recipient_id,request_id,100,101,patch,created_at FROM capture_response_workspace_edits LIMIT 1`},
		{"form_pause", `UPDATE monitoring_form_templates SET status='PAUSED' WHERE is_current`},
		{"relationship_edit", `UPDATE third_party_relationships SET service_name='Operator service'`},
		{"locked_distribution", `UPDATE capture_form_distributions SET status='LOCKED'`},
		{"duplicate_artifact", `INSERT INTO capture_artifacts(id,tenant_id,request_id,file_name,media_type,size_bytes,sha256,storage_key,status,created_at) SELECT md5('extra-artifact')::uuid,tenant_id,request_id,file_name,media_type,size_bytes,sha256,'extra-object',status,created_at FROM capture_artifacts LIMIT 1`},
	} {
		t.Run(test.name, func(t *testing.T) {
			pool, cfg, seed := sampleTestSetup(t)
			sampleTestWorker(t, pool)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			if _, err := installDocumentSamples(ctx, cfg, pool, seed); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, test.sql); err != nil {
				t.Fatal(err)
			}
			before := sampleTestSnapshot(t, pool)
			if _, err := installDocumentSamples(ctx, cfg, pool, seed); err == nil {
				t.Fatal("edited sample accepted")
			}
			if sampleTestSnapshot(t, pool) != before {
				t.Fatal("refused rerun changed records")
			}
		})
	}
}

func TestDocumentSamplesResumeInterruptedWork(t *testing.T) {
	for _, stage := range []string{"form_draft", "form_pending", "assessment_setup", "request_prepared", "request_issued", "send_route_revoked", "three_uploads", "six_uploads", "first_saved", "first_submitted", "second_saved"} {
		t.Run(stage, func(t *testing.T) {
			pool, cfg, seed := sampleTestSetup(t)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			i := &documentSampleInstaller{pool: pool, cfg: cfg, seed: seed}
			if err := i.configure(); err != nil {
				t.Fatal(err)
			}
			ownerCtx, err := i.actorContext(ctx, seed.ActorID)
			if err != nil {
				t.Fatal(err)
			}
			v, err := i.ensureVendor(ownerCtx)
			if err != nil {
				t.Fatal(err)
			}
			form, err := i.forms.CreateLibraryForm(ownerCtx, documentSampleForm())
			if err != nil {
				t.Fatal(err)
			}
			if stage != "form_draft" {
				form, err = i.forms.TransitionLibraryForm(ownerCtx, form.ID, monitoring.TransitionInput{ExpectedVersion: form.Version, To: monitoring.LifecyclePendingApproval})
				if err != nil {
					t.Fatal(err)
				}
			}
			if stage != "form_draft" && stage != "form_pending" {
				form, err = i.ensureForm(ctx)
				if err != nil {
					t.Fatal(err)
				}
				a, err := i.assessments.StartAssessment(ownerCtx, i.actor(), v.Relationship.ID, thirdparty.StartAssessmentInput{RelationshipVersion: v.Relationship.Version, ReviewKind: thirdparty.AssessmentReviewOnboarding, FormTemplateID: form.ID, FormTemplateVersion: form.Version, ReviewDueAt: time.Now().UTC().Add(14 * 24 * time.Hour)})
				if err != nil {
					t.Fatal(err)
				}
				if stage != "assessment_setup" {
					p := thirdparty.NewAssessmentProvisioner(thirdparty.NewPostgresRepository(pool), continuity.NewService(continuity.NewPostgresRepository(pool)), "partial-sample-test")
					p.ConfigureAuthority(authority.NewEffectivePostgresService(pool))
					if _, err = p.Maintain(ctx, time.Now().UTC(), 1); err != nil {
						t.Fatal(err)
					}
					a, err = i.assessments.GetAssessment(ctx, i.actor(), a.ID)
					if err != nil {
						t.Fatal(err)
					}
					if stage == "request_prepared" {
						_, err = pool.Exec(ctx, `CREATE FUNCTION sample_test_request_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected sample request interruption'; END $$; CREATE TRIGGER sample_test_request_failure BEFORE INSERT ON third_party_assessment_request_links FOR EACH ROW EXECUTE FUNCTION sample_test_request_failure()`)
						if err != nil {
							t.Fatal(err)
						}
						t.Cleanup(func() {
							_, _ = pool.Exec(context.Background(), `DROP TRIGGER IF EXISTS sample_test_request_failure ON third_party_assessment_request_links; DROP FUNCTION IF EXISTS sample_test_request_failure()`)
						})
					}
					out, err := i.requests.SendRequest(ownerCtx, i.actor(), a.ID, thirdparty.SendAssessmentRequestInput{ExpectedVersion: a.Version, Audience: documentSampleAudience, Deadline: a.ReviewDueAt.Add(-24 * time.Hour), InvitationTTLMinutes: 60})
					if stage == "request_prepared" {
						if err == nil {
							t.Fatal("request interruption did not occur")
						}
						if _, err = pool.Exec(ctx, `DROP TRIGGER sample_test_request_failure ON third_party_assessment_request_links; DROP FUNCTION sample_test_request_failure()`); err != nil {
							t.Fatal(err)
						}
					} else {
						if err != nil {
							t.Fatal(err)
						}
						if out.State != thirdparty.SendRequestLinkCreatedEmailNotSent {
							t.Fatal("fictional request was unexpectedly delivered")
						}
						if stage != "request_issued" && stage != "send_route_revoked" {
							sampleTestPartialResponse(t, ctx, i, out.Request, stage)
						}
					}
				}
			}
			if stage == "send_route_revoked" {
				_, err = pool.Exec(ctx, `UPDATE routing_policy_versions v SET definition=jsonb_set(v.definition,'{rules}',v.definition->'rules'||jsonb_build_array(jsonb_build_object('id','sample-send-revoked','legal_entity_id',$1::text,'object_type','THIRD_PARTY_ASSESSMENT','object_id','*','responsibility','ACCOUNTABLE_OWNER','decision_type','thirdparty.assessment.send_request','min_materiality',0,'priority',1000,'selector',jsonb_build_object('kind','PRINCIPAL','ref',$2::text)))) FROM routing_policies p WHERE p.id=v.policy_id AND p.current_version=v.version AND p.status='ACTIVE'`, seed.LegalEntityID, seed.ReviewerPrincipalID)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = i.guard.Authorize(ownerCtx, commandauth.Request{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID, ObjectType: "THIRD_PARTY_RELATIONSHIP", ObjectID: v.Relationship.ID, Responsibility: authority.ResponsibilityOwner, DecisionType: thirdparty.AssessmentStartCommand, Materiality: 3}); err != nil {
					t.Fatalf("start authority unexpectedly changed: %v", err)
				}
				before := sampleTestSnapshot(t, pool)
				if _, err = installDocumentSamples(ctx, cfg, pool, seed); !errors.Is(err, commandauth.ErrNotAuthorized) {
					t.Fatalf("specific send-route revocation: %v", err)
				}
				if before != sampleTestSnapshot(t, pool) {
					t.Fatal("send-route revocation changed respondent access or records")
				}
				return
			}
			sampleTestWorker(t, pool)
			receipt, err := installDocumentSamples(ctx, cfg, pool, seed)
			if err != nil {
				t.Fatal(err)
			}
			if receipt.ArtifactCount != 6 || len(receipt.ResponseRevisionIDs) != 2 {
				t.Fatalf("incomplete recovery: %+v", receipt)
			}
			if _, err = installDocumentSamples(ctx, cfg, pool, seed); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type sampleTestNoMail struct{ calls int }

func (s *sampleTestNoMail) Deliver(context.Context, evidence.InvitationDeliveryRequest) (evidence.InvitationDeliveryReceipt, error) {
	s.calls++
	return evidence.InvitationDeliveryReceipt{}, nil
}

func TestDocumentSamplesCanonicalCommunicationSkipsEmail(t *testing.T) {
	pool, cfg, seed := sampleTestSetup(t)
	sampleTestWorker(t, pool)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	receipt, err := installDocumentSamples(ctx, cfg, pool, seed)
	if err != nil {
		t.Fatal(err)
	}
	i := &documentSampleInstaller{pool: pool, cfg: cfg, seed: seed}
	if err = i.configure(); err != nil {
		t.Fatal(err)
	}
	repo := evidence.NewPostgresRepository(pool)
	mail := &sampleTestNoMail{}
	worker, err := evidence.NewCommunicationDeliveryWorker(evidence.NewPostgresCommunicationDeliveryRepository(repo), evidence.NewCommunicationService(evidence.NewPostgresCommunicationStore(repo)), i.access, evidence.NewInvitationDeliveryService(mail), "https://capture.example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	var event workflowruntime.OutboxEvent
	err = pool.QueryRow(ctx, `SELECT e.id::text,e.tenant_id::text,e.aggregate_type,e.aggregate_id::text,e.event_type,e.payload,e.occurred_at FROM outbox_events e JOIN capture_form_distributions d ON d.id=e.aggregate_id JOIN capture_requests r ON r.distribution_id=d.id WHERE r.id=$1::uuid AND e.aggregate_type='FORM_DISTRIBUTION' AND e.event_type='FORM_DISTRIBUTION_OPEN' ORDER BY e.occurred_at LIMIT 1`, receipt.RequestID).Scan(&event.ID, &event.TenantID, &event.AggregateType, &event.AggregateID, &event.EventType, &event.Payload, &event.OccurredAt)
	if err != nil {
		t.Fatal(err)
	}
	before := sampleTestSnapshot(t, pool)
	if err = worker.Publish(ctx, event); err != nil {
		t.Fatal(err)
	}
	if mail.calls != 0 || before != sampleTestSnapshot(t, pool) {
		t.Fatal("secondary worker sent mail or changed sample access")
	}
	var status, failure string
	if err = pool.QueryRow(ctx, `SELECT status,failure_code FROM form_delivery_attempts WHERE outbox_event_id=$1::uuid`, event.ID).Scan(&status, &failure); err != nil {
		t.Fatal(err)
	}
	if status != "SKIPPED" || failure != "WORKFLOW_OWNED_COMMUNICATION" {
		t.Fatalf("email skip = %s/%s", status, failure)
	}
}

func TestDocumentSamplesPreserveExistingReferenceAndOperatorRecords(t *testing.T) {
	pool, cfg, seed := sampleTestSetup(t)
	sampleTestWorker(t, pool)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	i := &documentSampleInstaller{pool: pool, cfg: cfg, seed: seed}
	if err := i.configure(); err != nil {
		t.Fatal(err)
	}
	ownerCtx, err := i.actorContext(ctx, seed.ActorID)
	if err != nil {
		t.Fatal(err)
	}
	vendor, err := i.vendors.CreateRelationship(ownerCtx, i.actor(), thirdparty.CreateRelationshipInput{LegalName: "Existing operator vendor", ServiceName: "Existing managed infrastructure", SourceID: "reference_data", ExternalRef: "vendor:managed-infrastructure", Criticality: thirdparty.CriticalityStandard, PrivacyRole: thirdparty.PrivacyNone})
	if err != nil {
		t.Fatal(err)
	}
	input := documentSampleForm()
	input.Code = "EXISTING-OPERATOR-FORM"
	input.Tags = []string{"operator-owned"}
	form, err := i.forms.CreateLibraryForm(ownerCtx, input)
	if err != nil {
		t.Fatal(err)
	}
	// Program-bound form codes occupy a different namespace from the unbound sample.
	if _, err = pool.Exec(ctx, `INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,effective_from) VALUES(md5('operator-program')::uuid,$1::uuid,$2::uuid,'OPERATOR-PROGRAM','Existing operator Program','COMPLIANCE','ACTIVE','Risk','NG',clock_timestamp())`, seed.TenantID, seed.LegalEntityID); err != nil {
		t.Fatal(err)
	}
	input.Code = documentSampleFormCode
	input.ProgramID = "" // Resolve the test-created Program without assuming its UUID encoding.
	if err = pool.QueryRow(ctx, `SELECT id::text FROM programs WHERE tenant_id=$1::uuid AND code='OPERATOR-PROGRAM'`, seed.TenantID).Scan(&input.ProgramID); err != nil {
		t.Fatal(err)
	}
	bound, err := i.forms.CreateLibraryForm(ownerCtx, input)
	if err != nil {
		t.Fatal(err)
	}
	var before, after []byte
	snapshot := `SELECT jsonb_build_array((SELECT to_jsonb(v) FROM third_parties v WHERE id=$1::uuid),(SELECT to_jsonb(r) FROM third_party_relationships r WHERE id=$2::uuid),(SELECT to_jsonb(f) FROM monitoring_form_templates f WHERE id=$3::uuid))`
	if err = pool.QueryRow(ctx, snapshot, vendor.Vendor.ID, vendor.Relationship.ID, form.ID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if _, err = installDocumentSamples(ctx, cfg, pool, seed); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, snapshot, vendor.Vendor.ID, vendor.Relationship.ID, form.ID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("sample installation changed existing records")
	}
	retained, err := i.forms.GetLibraryForm(ownerCtx, bound.ID, bound.Version)
	if err != nil || !sameSampleJSON(retained, bound) {
		t.Fatalf("Program-bound form changed: %v", err)
	}
}

func TestDocumentSamplesRejectChangedStoredBytes(t *testing.T) {
	pool, cfg, seed := sampleTestSetup(t)
	sampleTestWorker(t, pool)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	receipt, err := installDocumentSamples(ctx, cfg, pool, seed)
	if err != nil {
		t.Fatal(err)
	}
	i := &documentSampleInstaller{pool: pool, cfg: cfg, seed: seed}
	if err = i.configure(); err != nil {
		t.Fatal(err)
	}
	artifacts, err := i.sampleArtifacts(ctx, receipt.RequestID)
	if err != nil {
		t.Fatal(err)
	}
	a := artifacts["sample-security-declaration.pdf"]
	if _, err = i.objects.Put(ctx, a.StorageKey, strings.NewReader("changed stored bytes"), 1024); err != nil {
		t.Fatal(err)
	}
	before := sampleTestSnapshot(t, pool)
	if _, err = installDocumentSamples(ctx, cfg, pool, seed); err == nil {
		t.Fatal("changed stored sample bytes accepted")
	}
	if before != sampleTestSnapshot(t, pool) {
		t.Fatal("changed-byte refusal mutated records")
	}
}

func sampleTestPartialResponse(t *testing.T, ctx context.Context, i *documentSampleInstaller, r evidence.Request, stage string) {
	t.Helper()
	resumed, err := i.dispatch.Resume(ctx, i.seed.TenantID, i.seed.LegalEntityID, r.ID, i.seed.ActorID, time.Now().UTC().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	session, err := i.access.RedeemDirectRoute(ctx, resumed.Route.Selector)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := map[string]evidence.Artifact{}
	for n, file := range demodocuments.Files() {
		if stage == "three_uploads" && n == 3 {
			return
		}
		field := "security"
		for _, answer := range sampleDocumentAnswers(2) {
			if answer.file == file.Name {
				field = answer.field
			}
		}
		data, _ := demodocuments.Read(file.Name)
		artifact, err := i.evidence.StoreArtifactForDistributionSession(ctx, i.access, session.SessionToken, evidence.ArtifactInput{FieldID: field, FileName: file.Name, MediaType: file.MediaType}, bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		artifacts[file.Name] = artifact
	}
	if stage == "six_uploads" {
		return
	}
	workspace, err := i.access.GetResponseWorkspace(ctx, session.SessionToken)
	if err != nil {
		t.Fatal(err)
	}
	edits := sampleEdits(artifacts)
	for n := range edits[:9] {
		edits[n].BaseSequence = workspace.FieldSequences[edits[n].FieldID]
	}
	workspace, err = i.access.SaveResponseWorkspace(ctx, session.SessionToken, evidence.SaveWorkspaceInput{ExpectedVersion: workspace.Workspace.Version, PresentationMode: formcontract.PresentationClassic, Edits: edits[:9]})
	if err != nil {
		t.Fatal(err)
	}
	if stage == "first_saved" {
		return
	}
	if _, err = i.access.SubmitResponseWorkspace(ctx, session.SessionToken, evidence.SubmitWorkspaceInput{ExpectedVersion: workspace.Workspace.Version}); err != nil {
		t.Fatal(err)
	}
	if stage == "first_submitted" {
		return
	}
	workspace, err = i.access.GetResponseWorkspace(ctx, session.SessionToken)
	if err != nil {
		t.Fatal(err)
	}
	for n := 9; n < len(edits); n++ {
		edits[n].BaseSequence = workspace.FieldSequences[edits[n].FieldID]
	}
	if _, err = i.access.SaveResponseWorkspace(ctx, session.SessionToken, evidence.SaveWorkspaceInput{ExpectedVersion: workspace.Workspace.Version, PresentationMode: formcontract.PresentationClassic, Edits: edits[9:]}); err != nil {
		t.Fatal(err)
	}
}

func TestDocumentSamplesSerializeConcurrentInstallers(t *testing.T) {
	pool, cfg, seed := sampleTestSetup(t)
	sampleTestWorker(t, pool)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	results := make(chan documentSampleReceipt, 2)
	errs := make(chan error, 2)
	for n := 0; n < 2; n++ {
		go func() { r, err := installDocumentSamples(ctx, cfg, pool, seed); results <- r; errs <- err }()
	}
	first, second := <-results, <-results
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
	if first.AlreadyInstalled == second.AlreadyInstalled || !sameSampleJSON(first.ResponseRevisionIDs, second.ResponseRevisionIDs) {
		t.Fatal("concurrent installers did not reuse one result")
	}
}

func TestDocumentSamplesRefuseMissingCurrentAuthority(t *testing.T) {
	pool, cfg, seed := sampleTestSetup(t)
	if _, err := pool.Exec(context.Background(), `UPDATE routing_policies SET status='RETIRED'`); err != nil {
		t.Fatal(err)
	}
	before := sampleTestSnapshot(t, pool)
	if _, err := installDocumentSamples(context.Background(), cfg, pool, seed); err == nil {
		t.Fatal("missing authority accepted")
	}
	if sampleTestSnapshot(t, pool) != before {
		t.Fatal("missing authority changed records")
	}
}

func TestDocumentSampleReceiptNeverIncludesAccessSecrets(t *testing.T) {
	data, err := json.Marshal(documentSampleReceipt{})
	if err != nil {
		t.Fatal(err)
	}
	for _, word := range []string{"token", "selector", "capture_url", "session"} {
		if strings.Contains(string(data), word) {
			t.Fatalf("receipt exposes %s", word)
		}
	}
}

func TestDocumentSamplesRefuseUnsafeConfiguration(t *testing.T) {
	for _, cfg := range []config.Config{{Environment: "production", DemoMode: true}, {Environment: "development"}, {Environment: "development", DemoMode: true}} {
		if _, err := installDocumentSamples(context.Background(), cfg, nil, bankverticals.SeedConfig{}); err == nil {
			t.Fatal("unsafe configuration accepted")
		}
	}
}

func TestDocumentSampleFormRecoveryIndex(t *testing.T) {
	pool, cfg, seed := sampleTestSetup(t)
	ctx := context.Background()
	up, err := os.ReadFile("../../migrations/000085_unbound_form_history_lookup.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000085_unbound_form_history_lookup.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname='monitoring_form_templates_unbound_history_idx')`).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		if _, err = pool.Exec(ctx, string(up)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = pool.Exec(ctx, string(down)); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname='monitoring_form_templates_unbound_history_idx')`).Scan(&exists); err != nil || exists {
		t.Fatalf("index rollback: %v", err)
	}
	if _, err = pool.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	i := &documentSampleInstaller{pool: pool, cfg: cfg, seed: seed}
	if err = i.configure(); err != nil {
		t.Fatal(err)
	}
	actorCtx, err := i.actorContext(ctx, seed.ActorID)
	if err != nil {
		t.Fatal(err)
	}
	form, err := i.forms.CreateLibraryForm(actorCtx, documentSampleForm())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO monitoring_form_templates(id,tenant_id,legal_entity_id,code,name,purpose,fields,status,is_current,version,created_by) SELECT md5('sample-index-'||n)::uuid,tenant_id,legal_entity_id,'OTHER-'||n,name,purpose,fields,'DRAFT',false,1,created_by FROM monitoring_form_templates CROSS JOIN generate_series(1,1500) n WHERE id=$1::uuid`, form.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, "ANALYZE monitoring_form_templates"); err != nil {
		t.Fatal(err)
	}
	rows, err := pool.Query(ctx, "EXPLAIN (ANALYZE, BUFFERS) "+documentSampleFormLookup, seed.TenantID, seed.LegalEntityID, documentSampleFormCode)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for rows.Next() {
		var line string
		if err = rows.Scan(&line); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		lines = append(lines, line)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		t.Fatal(err)
	}
	plan := strings.Join(lines, "\n")
	t.Log(plan)
	if !strings.Contains(plan, "monitoring_form_templates_unbound_history_idx") || strings.Contains(plan, "Seq Scan") {
		t.Fatal("exact recovery query did not use unbound history index")
	}
}

func TestDocumentSampleAnswerMetadataMatchesAuthoredDocuments(t *testing.T) {
	artifacts := map[string]evidence.Artifact{}
	for _, file := range demodocuments.Files() {
		artifacts[file.Name] = evidence.Artifact{ID: file.Name}
	}
	for revision := 1; revision <= 2; revision++ {
		answers := sampleAnswers(revision, artifacts)
		securityRef, issued, expires := "NSI-SEC-2025-01", "2025-09-01", "2026-08-31"
		if revision == 2 {
			securityRef, issued, expires = "NSI-SEC-2026-02", "2026-09-01", "2027-09-01"
		}
		for field, ref := range map[string]string{"security": securityRef, "insurance": "BRK-INS-2026-04", "recovery": "NSI-BCP-2026-03", "office": "NSI-ADDR-2026-01", "subprocessors": ""} {
			if answers[field].Document.Reference != ref {
				t.Errorf("revision %d %s reference=%q want %q", revision, field, answers[field].Document.Reference, ref)
			}
		}
		if answers["security"].Document.IssuedOn != issued || answers["security"].Document.ExpiresOn != expires {
			t.Fatal("security document dates differ from authored scenario")
		}
		if answers["insurance"].Document.IssuedOn != "2026-04-01" || answers["insurance"].Document.ExpiresOn != "2026-09-30" {
			t.Fatal("insurance dates differ from authored scenario")
		}
		if _, ok := answers["archive_evidence"]; ok {
			t.Fatal("missing optional evidence was manufactured")
		}
	}
}
