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

func TestPostgresDemoCollectionStatusAndRemindersUseCurrentPolicy(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant, entity, owner, form, vendor, relationship := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	now := time.Now().UTC()
	prefix := "demo-held-" + tenant
	setupResponseWorkspaceFixture(t, ctx, pool, tenant, prefix, entity, owner, form, now)
	t.Cleanup(func() { cleanupVendorFormsFixture(context.Background(), pool, tenant) })
	seedCompletedResponseRows(t, ctx, pool, tenant, entity, owner, form, prefix, 1, now)
	if _, err := pool.Exec(ctx, `INSERT INTO third_parties(id,tenant_id,legal_name,status,created_at,updated_at,version) VALUES($1::uuid,$2::uuid,'Sample vendor','ACTIVE',$6,$6,1);
 INSERT INTO third_party_relationships(id,tenant_id,legal_entity_id,vendor_id,service_name,business_owner_principal_id,criticality,privacy_role,status,created_at,updated_at,version) VALUES($3::uuid,$2::uuid,$4::uuid,$1::uuid,'Sample processing',$5::uuid,'IMPORTANT','PROCESSOR','PROPOSED',$6,$6,1);
 UPDATE capture_form_distributions SET subject_type='VENDOR_RELATIONSHIP',subject_id=$3::uuid WHERE tenant_id=$2::uuid;
 UPDATE capture_requests SET status='READY',subject_type='VENDOR_RELATIONSHIP',subject_id=$3,fields='[{"id":"certificate","section_id":"general","label":"Certificate","type":"vendor_document"}]' WHERE tenant_id=$2::uuid;`, pgx.QueryExecModeSimpleProtocol, vendor, tenant, relationship, entity, owner, now); err != nil {
		t.Fatal(err)
	}
	var sourceRequest, sourceSubmission, sourceRevision, sourceDistribution string
	if err := pool.QueryRow(ctx, `SELECT req.id::text,s.id::text,r.id::text,req.distribution_id::text FROM capture_requests req JOIN capture_submissions s ON s.request_id=req.id JOIN capture_response_revisions r ON r.submission_id=s.id WHERE req.tenant_id=$1::uuid`, tenant).Scan(&sourceRequest, &sourceSubmission, &sourceRevision, &sourceDistribution); err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresRepository(pool)
	store := NewPostgresDistributionStore(repo, postgresTestRecipientProtector{})
	artifact, err := repo.CreateArtifact(ctx, Artifact{ID: mustResponseWorkspaceID(t), TenantID: tenant, RequestID: sourceRequest, SubmissionID: sourceSubmission, FileName: "certificate.pdf", MediaType: "application/pdf", SizeBytes: 10, SHA256: strings.Repeat("a", 64), StorageKey: "demo-held-" + tenant, Status: ArtifactStoredUnscanned, CreatedBy: owner, CreatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_requests SET status='SUBMITTED' WHERE id=$1::uuid`, sourceRequest); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_submissions SET channel='MAGIC_LINK',answers=jsonb_build_object('certificate',jsonb_build_object('document',jsonb_build_object('artifact_id',$2::text))) WHERE id=$1::uuid`, sourceSubmission, artifact.ID); err != nil {
		t.Fatal(err)
	}
	target, err := store.CreateDistribution(ctx, CreateDistributionInput{TenantID: tenant, LegalEntityID: entity, FormTemplateID: form, FormTemplateVersion: 1, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: relationship, Title: "Sample report review", Purpose: "Provide the sample report.", AccessPolicy: AccessDirectMagicLink, EstimatedMinutes: 2, Deadline: now.Add(time.Hour), RouteExpiresAt: now.Add(time.Hour), CreatedBy: owner, Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientExternalAudience, Address: "vendor@example.test", AudienceHint: "Vendor contact"}}})
	if err != nil {
		t.Fatal(err)
	}
	held := captureHeldField(t, now)
	held.SectionID = "general"
	held.CollectionResolution.Source = DocumentOccurrence{ID: "source-occurrence", SubmissionChannel: "MAGIC_LINK", ArtifactID: artifact.ID, RequestID: sourceRequest, ArtifactRequestID: sourceRequest, SubmissionID: sourceSubmission, ResponseRevisionID: sourceRevision, DistributionID: sourceDistribution, FieldID: "certificate", RelationshipID: relationship, ArtifactStatus: ArtifactStoredUnscanned, Current: true, SHA256: artifact.SHA256, SizeBytes: artifact.SizeBytes}
	held.CollectionResolution.SourceArtifactRequestID = sourceRequest
	held.CollectionResolution.Document.ArtifactID = artifact.ID
	reminder := NewPostgresCommunicationReminderRepository(pool)
	clearTestReminders := func() {
		t.Helper()
		if _, err := pool.Exec(ctx, `DELETE FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type='FORM_COMMUNICATION_REMINDER_DUE'`, tenant, target.Distribution.ID); err != nil {
			t.Fatal(err)
		}
	}
	q := VendorFormsQuery{TenantID: tenant, LegalEntityID: entity, PrincipalID: owner, RelationshipIDs: []string{relationship}, Limit: 25}
	for index, test := range []struct {
		name                     string
		enabled, forged, expired bool
		status                   ArtifactStatus
		outstanding              int
	}{
		{name: "default off ignores forged stored allowance", forged: true, status: ArtifactStoredUnscanned, outstanding: 1},
		{name: "demo enabled ignores stale false stored allowance", enabled: true, status: ArtifactStoredUnscanned},
		{name: "revocation restores action", status: ArtifactStoredUnscanned, outstanding: 1},
		{name: "quarantined stays outstanding", enabled: true, forged: true, status: ArtifactQuarantined, outstanding: 1},
		{name: "expired stays outstanding", enabled: true, expired: true, status: ArtifactStoredUnscanned, outstanding: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			clearTestReminders()
			repo.ConfigureDemoUnscannedArtifacts(test.enabled)
			reminder.ConfigureDemoUnscannedArtifacts(test.enabled)
			held.CollectionResolution.Source.DemoUnscannedAllowed = test.forged
			held.CollectionResolution.Document.ExpiresOn = ""
			if test.expired {
				held.CollectionResolution.Document.ExpiresOn = now.Add(-24 * time.Hour).Format("2006-01-02")
			}
			raw, _ := json.Marshal([]Field{held})
			if _, err = pool.Exec(ctx, `UPDATE capture_requests SET fields=$2::jsonb WHERE id=$1::uuid;
 UPDATE capture_artifacts SET status=$4 WHERE id=$3::uuid;
 UPDATE capture_form_distributions SET status='OPEN',reminder_policy=jsonb_build_object('reminder_hours_before',jsonb_build_array($6::int)) WHERE id=$5::uuid`, pgx.QueryExecModeSimpleProtocol, target.Recipients[0].RequestID, string(raw), artifact.ID, test.status, target.Distribution.ID, index+1); err != nil {
				t.Fatal(err)
			}
			summaries, err := store.VendorFormSummaries(ctx, q)
			if err != nil || len(summaries) != 1 || summaries[0].OutstandingForms != test.outstanding {
				t.Fatalf("summary=%+v err=%v", summaries, err)
			}
			filtered := q
			filtered.Filter = "AWAITING_VENDOR"
			page, err := store.ListVendorForms(ctx, filtered)
			if err != nil || len(page.Items) != test.outstanding {
				t.Fatalf("filtered page=%+v err=%v", page, err)
			}
			all, err := store.ListVendorForms(ctx, q)
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range all.Items {
				if row.RequestID == target.Recipients[0].RequestID && (row.ResponseState == "NO_VENDOR_ACTION") != (test.outstanding == 0) {
					t.Fatalf("row state=%+v", row)
				}
			}
			count, err := reminder.ScheduleDueCommunicationReminders(ctx, now, 100)
			if err != nil {
				t.Fatalf("scheduled=%d err=%v", count, err)
			}
			var targetReminders int
			if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type='FORM_COMMUNICATION_REMINDER_DUE'`, tenant, target.Distribution.ID).Scan(&targetReminders); err != nil || targetReminders != test.outstanding {
				t.Fatalf("target reminders=%d error=%v", targetReminders, err)
			}
			clearTestReminders()
			inserted, err := reminder.insertReminder(ctx, tenant, target.Distribution.ID, target.Distribution.Deadline, communicationReminderSpec{Action: CommunicationReminder, HoursBefore: 100 + index}, now)
			if err != nil || inserted != (test.outstanding == 1) {
				t.Fatalf("transaction recheck inserted=%v err=%v", inserted, err)
			}
		})
	}
	// A candidate selected while strict must be rechecked after enabling the
	// exception, rather than enqueueing a now-unnecessary reminder.
	held.CollectionResolution.Document.ExpiresOn = ""
	raw, _ := json.Marshal([]Field{held})
	if _, err = pool.Exec(ctx, `UPDATE capture_requests SET fields=$2::jsonb WHERE id=$1::uuid;UPDATE capture_artifacts SET status='STORED_UNSCANNED' WHERE id=$3::uuid`, pgx.QueryExecModeSimpleProtocol, target.Recipients[0].RequestID, string(raw), artifact.ID); err != nil {
		t.Fatal(err)
	}
	clearTestReminders()
	reminder.ConfigureDemoUnscannedArtifacts(true)
	inserted, err := reminder.insertReminder(ctx, tenant, target.Distribution.ID, target.Distribution.Deadline, communicationReminderSpec{Action: CommunicationReminder, HoursBefore: 200}, now)
	if err != nil || inserted {
		t.Fatalf("candidate ignored newly enabled policy: %v %v", inserted, err)
	}
	reminder.ConfigureDemoUnscannedArtifacts(false)
	inserted, err = reminder.insertReminder(ctx, tenant, target.Distribution.ID, target.Distribution.Deadline, communicationReminderSpec{Action: CommunicationReminder, HoursBefore: 200}, now)
	if err != nil || !inserted {
		t.Fatalf("candidate ignored policy revocation: %v %v", inserted, err)
	}
}
