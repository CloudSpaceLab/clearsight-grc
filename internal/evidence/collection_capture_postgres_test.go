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

func TestPostgresCollectionSubmitRechecksSourceAndRequest(t *testing.T) {
	for _, change := range []string{"source quarantined", "request changed", "held source unchanged", "source replaced"} {
		t.Run(change, func(t *testing.T) {
			pool, ctx := distributionTestPool(t)
			tenant, entity, actor, form := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
			now := time.Now().UTC()
			setupResponseWorkspaceFixture(t, ctx, pool, tenant, "collection-"+tenant, entity, actor, form, now)
			t.Cleanup(func() { cleanupResponseWorkspaceTenant(context.Background(), pool, tenant) })
			keyring, err := NewRecipientKeyring("key", map[string][32]byte{"key": testSecurityKey(0x43)})
			if err != nil {
				t.Fatal(err)
			}
			store := NewPostgresDistributionStore(NewPostgresRepository(pool), keyring)
			store.now = func() time.Time { return now }
			input := CreateDistributionInput{TenantID: tenant, LegalEntityID: entity, FormTemplateID: form, FormTemplateVersion: 1, SubjectType: "VENDOR", SubjectID: mustResponseWorkspaceID(t), Title: "Supplier records", Purpose: "Confirm supplier records.", AccessPolicy: AccessDirectMagicLink, EstimatedMinutes: 5, Deadline: now.Add(time.Hour), RouteExpiresAt: now.Add(time.Hour), CreatedBy: actor, Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientExternalAudience, Address: "vendor@example.test", AudienceHint: "v***@example.test"}}}
			bundle, err := store.CreateDistribution(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			source, err := store.CreateDistribution(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = pool.Exec(ctx, `UPDATE capture_form_distributions SET status='OPEN' WHERE id=$1::uuid`, bundle.Distribution.ID); err != nil {
				t.Fatal(err)
			}
			access, err := NewDistributionAccessService(store, keyring, &postgresAccessOTPDelivery{}, testSecurityKey(0x44), 20*time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			access.now = func() time.Time { return now }
			routes, err := access.IssueDistributionAccessRoutes(ctx, tenant, entity, bundle.Distribution.ID, actor)
			if err != nil || len(routes) != 1 {
				t.Fatalf("issue route: %v", err)
			}
			redeemed, err := access.RedeemDirectRoute(ctx, routes[0].Selector)
			if err != nil {
				t.Fatal(err)
			}
			session, request, err := access.sessionRequest(ctx, redeemed.SessionToken)
			if err != nil {
				t.Fatal(err)
			}
			artifactID := mustResponseWorkspaceID(t)
			sourceRequestID := source.Recipients[0].RequestID
			artifact, err := store.repo.CreateArtifact(ctx, Artifact{ID: artifactID, TenantID: tenant, RequestID: sourceRequestID, FileName: "certificate.pdf", MediaType: "application/pdf", SizeBytes: 10, SHA256: strings.Repeat("a", 64), StorageKey: "collection-" + artifactID, Status: ArtifactAvailable, CreatedBy: actor, CreatedAt: now})
			if err != nil {
				t.Fatal(err)
			}
			held := captureHeldField(t, now)
			held.SectionID = "general"
			held.CollectionResolution.Source.ArtifactID = artifact.ID
			held.CollectionResolution.Source.SHA256 = artifact.SHA256
			held.CollectionResolution.SourceArtifactRequestID = sourceRequestID
			held.CollectionResolution.Source.RequestID = sourceRequestID
			held.CollectionResolution.Source.SubmissionID = mustResponseWorkspaceID(t)
			held.CollectionResolution.Source.ResponseRevisionID = mustResponseWorkspaceID(t)
			held.CollectionResolution.Source.DistributionID = source.Distribution.ID
			held.CollectionResolution.Source.RelationshipID = input.SubjectID
			if _, err = pool.Exec(ctx, `
			 UPDATE capture_requests SET fields='[{"id":"certificate","type":"vendor_document"}]'::jsonb WHERE id=$3::uuid;
			 INSERT INTO capture_submissions(id,tenant_id,request_id,channel,answers,submitted_at,distribution_id)
			 VALUES($4::uuid,$1::uuid,$3::uuid,'MAGIC_LINK',jsonb_build_object('certificate',jsonb_build_object('document',jsonb_build_object('artifact_id',$8::text))),$7,$5::uuid);
			 INSERT INTO capture_response_revisions(id,tenant_id,legal_entity_id,distribution_id,workspace_id,submission_id,revision,achieved_assurance,state,scoring_policy_version)
			 SELECT $6::uuid,$1::uuid,$2::uuid,$5::uuid,id,$4::uuid,1,'LINK_POSSESSION','FINAL','collection-test' FROM capture_response_workspaces WHERE distribution_id=$5::uuid;
			`, pgx.QueryExecModeSimpleProtocol, tenant, entity, sourceRequestID, held.CollectionResolution.Source.SubmissionID, source.Distribution.ID, held.CollectionResolution.Source.ResponseRevisionID, now, artifact.ID); err != nil {
				t.Fatal(err)
			}
			held.CollectionResolution.Document.ArtifactID = artifact.ID
			request.Fields = append(request.Fields, held)
			fieldsJSON, _ := json.Marshal(request.Fields)
			if _, err = pool.Exec(ctx, `UPDATE capture_requests SET fields=$2::jsonb WHERE id=$1::uuid`, request.ID, fieldsJSON); err != nil {
				t.Fatal(err)
			}
			view, err := access.GetResponseWorkspace(ctx, redeemed.SessionToken)
			if err != nil {
				t.Fatal(err)
			}
			view, err = access.SaveResponseWorkspace(ctx, redeemed.SessionToken, SaveWorkspaceInput{ExpectedVersion: view.Workspace.Version, Edits: []FieldEdit{{FieldID: "registered_address", Value: formcontract.TextAnswer("Lagos")}}})
			if err != nil {
				t.Fatal(err)
			}
			if change == "source quarantined" {
				_, err = pool.Exec(ctx, `UPDATE capture_artifacts SET status='QUARANTINED' WHERE id=$1::uuid`, artifact.ID)
			}
			if change == "request changed" {
				_, err = pool.Exec(ctx, `UPDATE capture_requests SET version=version+1 WHERE id=$1::uuid`, request.ID)
			}
			if change == "source replaced" {
				if _, err = pool.Exec(ctx, `
				 UPDATE capture_response_revisions SET is_current=false WHERE id=$1::uuid;
				 INSERT INTO capture_submissions(id,tenant_id,request_id,channel,answers,submitted_at,distribution_id)
				 SELECT $2::uuid,tenant_id,request_id,'MAGIC_LINK','{}'::jsonb,submitted_at+interval '1 minute',distribution_id FROM capture_submissions WHERE id=$3::uuid;
				 INSERT INTO capture_response_revisions(id,tenant_id,legal_entity_id,distribution_id,workspace_id,submission_id,revision,supersedes_revision_id,achieved_assurance,state,scoring_policy_version)
				 SELECT $4::uuid,tenant_id,legal_entity_id,distribution_id,workspace_id,$2::uuid,2,id,achieved_assurance,state,scoring_policy_version FROM capture_response_revisions WHERE id=$1::uuid;
				`, pgx.QueryExecModeSimpleProtocol, held.CollectionResolution.Source.ResponseRevisionID, mustResponseWorkspaceID(t), held.CollectionResolution.Source.SubmissionID, mustResponseWorkspaceID(t)); err != nil {
					t.Fatal(err)
				}
				refreshed, refreshErr := store.repo.RefreshCollectionRequestReviews(ctx, request)
				if refreshErr != nil || CollectionFieldFulfilled(refreshed.Fields[len(refreshed.Fields)-1], now) {
					t.Fatalf("replaced source still fulfills request: %v", refreshErr)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if change == "held source unchanged" || change == "source replaced" {
				heldJSON, _ := json.Marshal([]Field{held})
				if _, err = pool.Exec(ctx, `UPDATE capture_requests SET fields=$2::jsonb WHERE id=$1::uuid`, request.ID, heldJSON); err != nil {
					t.Fatal(err)
				}
				inserted, reminderErr := NewPostgresCommunicationReminderRepository(pool).insertReminder(ctx, tenant, bundle.Distribution.ID, input.Deadline, communicationReminderSpec{Action: CommunicationReminder, HoursBefore: 1}, now)
				if reminderErr != nil || inserted != (change == "source replaced") {
					t.Fatalf("source currency not reflected by vendor reminder: inserted=%v err=%v", inserted, reminderErr)
				}
				if _, err = pool.Exec(ctx, `UPDATE capture_requests SET fields=$2::jsonb WHERE id=$1::uuid`, request.ID, fieldsJSON); err != nil {
					t.Fatal(err)
				}
			}
			if change != "source replaced" {
				inserted, reminderErr := NewPostgresCommunicationReminderRepository(pool).insertReminder(ctx, tenant, bundle.Distribution.ID, input.Deadline, communicationReminderSpec{Action: CommunicationReminder, HoursBefore: 2}, now)
				if reminderErr != nil || !inserted {
					t.Fatalf("outstanding scalar response lost its reminder: inserted=%v err=%v", inserted, reminderErr)
				}
			}
			result, err := store.SubmitResponseWorkspace(ctx, workspaceSubmitCommand{Session: session, Request: request, Input: SubmitWorkspaceInput{ExpectedVersion: view.Workspace.Version}, Now: now, Validate: func(map[string]formcontract.AnswerValue) error { return nil }, BuildRevision: func(answers map[string]formcontract.AnswerValue) (ResponseRevision, error) {
				return buildResponseRevision(request, session.Assurance, nil, answers)
			}})
			var count int
			if queryErr := pool.QueryRow(ctx, `SELECT count(*) FROM capture_submissions WHERE request_id=$1::uuid`, request.ID).Scan(&count); queryErr != nil {
				t.Fatal(queryErr)
			}
			if change != "held source unchanged" {
				if err == nil || count != 0 {
					t.Fatalf("changed source/request committed: result=%+v err=%v count=%d", result, err, count)
				}
			} else {
				if err != nil || count != 1 {
					t.Fatalf("valid held evidence did not submit: %v count=%d", err, count)
				}
				var heldAnswer bool
				if queryErr := pool.QueryRow(ctx, `SELECT answers?'held' OR answer_provenance?'held' FROM capture_submissions WHERE id=$1::uuid`, result.Submission.SubmissionID).Scan(&heldAnswer); queryErr != nil {
					t.Fatal(queryErr)
				}
				if heldAnswer {
					t.Fatal("bank receipt became respondent evidence")
				}
				var consumedVersion int64
				if queryErr := pool.QueryRow(ctx, `SELECT COALESCE((payload->>'request_version')::bigint,0) FROM capture_distribution_events WHERE distribution_id=$1::uuid AND event_type='FORM_RESPONSE_SCORED_1' AND payload->>'request_id'=$2`, bundle.Distribution.ID, request.ID).Scan(&consumedVersion); queryErr != nil || consumedVersion != request.Version {
					t.Fatalf("submission request version not recorded: %d %v", consumedVersion, queryErr)
				}
			}
		})
	}
}
