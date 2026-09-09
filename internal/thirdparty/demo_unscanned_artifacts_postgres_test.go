//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/jackc/pgx/v5"
	"strings"
	"testing"
	"time"
)

func TestPostgresDemoUnscannedWorkAcceptanceRechecksExactSource(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	now := time.Now().UTC()
	const programID = "33333333-3333-7333-8333-333333333375"
	const linkID = "33333333-3333-7333-8333-333333333376"
	const workID = "33333333-3333-7333-8333-333333333377"
	const captureID = "33333333-3333-7333-8333-333333333378"
	relationship := seedAssessmentRelationship(t, pool, "Demo unscanned work")
	if _, err := pool.Exec(ctx, `INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,effective_from) VALUES($1::uuid,$2::uuid,$3::uuid,'DEMO-WORK','Demo document review','COMPLIANCE','ACTIVE','Third-party risk','NG',$4);
 INSERT INTO third_party_relationship_program_links(id,tenant_id,legal_entity_id,relationship_id,program_id,purpose_code,purpose_label,state,created_by_principal_id,version,created_at,updated_at) VALUES($5::uuid,$2::uuid,$3::uuid,$6::uuid,$1::uuid,'EVIDENCE_REFRESH','Evidence refresh','ACTIVE',$7::uuid,1,$4,$4);
 UPDATE monitoring_form_templates SET fields='[{"id":"report","section_id":"company","label":"Report","type":"file","required":true}]' WHERE id=$8::uuid`, pgx.QueryExecModeSimpleProtocol, programID, thirdPartyTenantID, thirdPartyEntityA, now, linkID, relationship.Relationship.ID, thirdPartyPrincipal, assessmentTemplateID); err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresRepository(pool)
	work, err := repo.CreateVendorWork(ctx, VendorWorkRequest{ID: workID, TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA, RelationshipID: relationship.Relationship.ID, RelationshipLinkID: linkID, TargetType: LinkTargetProgram, TargetID: programID, RequestKind: VendorWorkGeneral, Purpose: "Review the sample report.", Instructions: "Provide the sample report.", OwnerPrincipalID: thirdPartyPrincipal, FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3, Presentation: formcontract.PresentationWizard, State: VendorWorkPreparing, DeliveryState: VendorWorkDeliveryNotSent, DueAt: now.Add(time.Hour), Version: 1, CreatedAt: now, UpdatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	capture := evidence.NewService(evidence.NewPostgresRepository(pool), evidence.NewMemoryObjectStore())
	ctx = evidence.WithRequestOriginAuthority(ctx, VendorWorkOrigin)
	request, err := capture.CreateRequest(ctx, evidence.CreateRequestInput{TenantID: work.TenantID, LegalEntityID: work.LegalEntityID, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: work.RelationshipID, Title: "Sample report", Purpose: work.Purpose, WhyYou: work.Instructions, Sensitivity: "INTERNAL", AudienceType: "INTERNAL", Recipient: evidence.RecipientInput{Type: evidence.RecipientInternalPrincipal, PrincipalID: thirdPartyPrincipal}, EstimatedMinutes: 2, Deadline: work.DueAt, Origin: evidence.RequestOrigin{Type: VendorWorkOrigin, ID: work.ID, Version: 1}, Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard}, Sections: []formcontract.Section{{ID: "company", Title: "Company"}}, Fields: []evidence.Field{{ID: "report", SectionID: "company", Label: "Report", Type: "file", Required: true}}, FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3, CreatedBy: thirdPartyPrincipal})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := capture.StoreArtifact(ctx, evidence.ArtifactInput{TenantID: work.TenantID, RequestID: request.ID, FileName: "report.txt", MediaType: "text/plain", CreatedBy: thirdPartyPrincipal}, strings.NewReader("Synthetic sample report"))
	if err != nil {
		t.Fatal(err)
	}
	submitted, err := capture.Submit(ctx, evidence.Submission{TenantID: work.TenantID, LegalEntityID: work.LegalEntityID, RequestID: request.ID, SubmittedBy: thirdPartyPrincipal, Channel: "INTERNAL", ExpectedVersion: request.Version, Answers: map[string]formcontract.AnswerValue{"report": {ArtifactIDs: []string{artifact.ID}}}})
	if err != nil {
		t.Fatal(err)
	}
	work, err = repo.AttachVendorWorkCapture(ctx, Scope{TenantID: work.TenantID, LegalEntityID: work.LegalEntityID}, work.ID, work.Version, VendorWorkCaptureLink{ID: captureID, TenantID: work.TenantID, LegalEntityID: work.LegalEntityID, WorkRequestID: work.ID, RequestID: request.ID, Sequence: 1, Purpose: "INITIAL", OriginVersion: 1, CreatedAt: now}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE third_party_work_capture_links SET submission_id=$1::uuid WHERE id=$2::uuid;UPDATE third_party_work_requests SET state='UNDER_REVIEW',submission_id=$1::uuid,response_received_at=$3,review_started_at=$3 WHERE id=$4::uuid`, pgx.QueryExecModeSimpleProtocol, submitted.SubmissionID, captureID, now, work.ID); err != nil {
		t.Fatal(err)
	}
	accept := func(expected int64) error {
		_, err := repo.TransitionVendorWork(ctx, Scope{TenantID: work.TenantID, LegalEntityID: work.LegalEntityID}, work.ID, expected, VendorWorkAccepted, thirdPartyPrincipal, "Reviewed the sample report.", now)
		return err
	}
	if err = accept(work.Version); !errors.Is(err, ErrVendorWorkAcceptanceBlocked) {
		t.Fatalf("default policy accepted file: %v", err)
	}
	repo.ConfigureDemoUnscannedArtifacts(true)
	if err = accept(work.Version + 1); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale work version accepted: %v", err)
	}
	for _, status := range []string{"QUARANTINED", "DELETED"} {
		if _, err = pool.Exec(ctx, `UPDATE capture_artifacts SET status=$2 WHERE id=$1::uuid`, artifact.ID, status); err != nil {
			t.Fatal(err)
		}
		if err = accept(work.Version); !errors.Is(err, ErrVendorWorkAcceptanceBlocked) {
			t.Fatalf("accepted %s file: %v", status, err)
		}
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_artifacts SET status='STORED_UNSCANNED' WHERE id=$1::uuid`, artifact.ID); err != nil {
		t.Fatal(err)
	}
	repo.ConfigureDemoUnscannedArtifacts(false)
	if err = accept(work.Version); !errors.Is(err, ErrVendorWorkAcceptanceBlocked) {
		t.Fatalf("revoked policy accepted file: %v", err)
	}
	repo.ConfigureDemoUnscannedArtifacts(true)
	if err = accept(work.Version); err != nil {
		t.Fatal(err)
	}
	var status string
	var audited bool
	if err = pool.QueryRow(ctx, `SELECT a.status,EXISTS(SELECT 1 FROM third_party_work_events e WHERE e.work_request_id=$2::uuid AND e.event_type='VendorWorkAccepted' AND e.payload->'demo_unscanned_artifact_ids' ? a.id::text) FROM capture_artifacts a WHERE a.id=$1::uuid`, artifact.ID, work.ID).Scan(&status, &audited); err != nil {
		t.Fatal(err)
	}
	if status != "STORED_UNSCANNED" || !audited {
		t.Fatalf("status=%s audited=%v", status, audited)
	}
}
