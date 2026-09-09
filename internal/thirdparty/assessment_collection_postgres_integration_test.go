//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
	"strings"
	"testing"
	"time"
)

func TestPostgresCollectionReconciliationKeepsSourceAndCommitsAtomically(t *testing.T) {
	for _, demo := range []bool{false, true} {
		name := "scanned"
		if demo {
			name = "demo_unscanned"
		}
		t.Run(name, func(t *testing.T) { testPostgresCollectionReconciliationKeepsSourceAndCommitsAtomically(t, demo) })
	}
}
func testPostgresCollectionReconciliationKeepsSourceAndCommitsAtomically(t *testing.T, demo bool) {
	pool := assessmentPostgresPool(t)
	ctx := evidence.WithRequestOriginAuthority(context.Background(), AssessmentRequestOrigin)
	relationship := seedAssessmentRelationship(t, pool, "Held assurance report")
	now := time.Now().UTC()
	repo := NewPostgresRepository(pool)
	a, err := repo.CreateAssessment(ctx, postgresAssessmentRecord(assessmentOneID, relationship, now))
	if err != nil {
		t.Fatal(err)
	}
	capture := evidence.NewService(evidence.NewPostgresRepository(pool), evidence.NewMemoryObjectStore())
	input := evidence.CreateRequestInput{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: relationship.Relationship.ID, Title: "Assurance report", Purpose: "Provide the assurance report.", WhyYou: "You are the vendor contact.", Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR", Recipient: evidence.RecipientInput{Type: evidence.RecipientExternalAudience, Audience: "reports@vendor.example"}, EstimatedMinutes: 5, Deadline: now.Add(24 * time.Hour), Origin: evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: a.ID, Version: 1}, Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard}, Sections: []formcontract.Section{{ID: "documents", Title: "Documents"}}, Fields: []evidence.Field{{ID: "report", SectionID: "documents", Label: "Assurance report", Type: "vendor_document", Required: true, AcceptedFormats: []string{"application/pdf"}}}, FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3, CreatedBy: thirdPartyPrincipal}
	sourceRequest, err := capture.CreateRequest(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	issued, err := capture.IssueInvitation(ctx, evidence.IssueInvitationInput{TenantID: input.TenantID, LegalEntityID: input.LegalEntityID, RequestID: sourceRequest.ID, Audience: input.Recipient.Audience, Purpose: "Provide the assurance report.", TTLMinutes: 60, CreatedBy: thirdPartyPrincipal})
	if err != nil {
		t.Fatal(err)
	}
	session, err := capture.RedeemInvitation(ctx, issued.Token, input.Recipient.Audience)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := capture.StoreArtifact(ctx, evidence.ArtifactInput{TenantID: input.TenantID, RequestID: sourceRequest.ID, SessionToken: session.SessionToken, FileName: "assurance.pdf", MediaType: "application/pdf"}, strings.NewReader("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n%%EOF"))
	if err != nil {
		t.Fatal(err)
	}
	submitted, err := capture.SubmitSession(ctx, session.SessionToken, map[string]formcontract.AnswerValue{"report": {Document: &formcontract.DocumentAnswer{ArtifactID: artifact.ID, DocumentType: "SOC2"}}}, sourceRequest.Version)
	if err != nil {
		t.Fatal(err)
	}
	input.Origin.Version = 2
	target, err := capture.CreateRequest(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `UPDATE capture_artifacts SET status='AVAILABLE' WHERE id=$1::uuid; UPDATE third_party_assessments SET status='COLLECTING',current_request_id=$2::uuid,version=4,updated_at=$3 WHERE id=$4::uuid; INSERT INTO third_party_assessment_request_links(tenant_id,legal_entity_id,assessment_id,request_id,purpose,sequence,origin_type,origin_id,origin_sequence,is_current,created_at) VALUES($5::uuid,$6::uuid,$4::uuid,$7::uuid,'INITIAL',1,'THIRD_PARTY_ASSESSMENT',$4::uuid,1,false,$3),($5::uuid,$6::uuid,$4::uuid,$2::uuid,'CLARIFICATION',2,'THIRD_PARTY_ASSESSMENT',$4::uuid,2,true,$3)`, pgx.QueryExecModeSimpleProtocol, artifact.ID, target.ID, now, a.ID, thirdPartyTenantID, thirdPartyEntityA, sourceRequest.ID)
	if err != nil {
		t.Fatal(err)
	}
	documents := evidence.NewPostgresDistributionStore(evidence.NewPostgresRepository(pool), nil)
	sources, err := documents.ListDocuments(ctx, evidence.DocumentQuery{TenantID: thirdPartyTenantID, LegalEntityID: thirdPartyEntityA, PrincipalID: thirdPartyPrincipal, RelationshipID: relationship.Relationship.ID, SubmissionID: submitted.SubmissionID, FieldID: "report", ArtifactID: artifact.ID, CurrentOnly: true, Limit: 1})
	if err != nil || len(sources.Items) != 1 {
		t.Fatalf("source read: %+v %v", sources, err)
	}
	receiptID, _ := id.NewUUIDv7()
	source := sources.Items[0]
	record := assessmentCollectionRecord{Scope: Scope{TenantID: input.TenantID, LegalEntityID: thirdPartyEntityA}, AssessmentID: a.ID, RequestID: target.ID, FieldID: "report", ActorPrincipalID: thirdPartyPrincipal, ExpectedVersion: 4, ExpectedRequestVersion: target.Version, At: now.Add(time.Minute), Resolution: evidence.CollectionResolution{ID: receiptID, Version: 1, Source: source, SourceArtifactRequestID: sourceRequest.ID, ReconciledBy: thirdPartyPrincipal, ReconciledAt: now.Add(time.Minute), Rationale: "The report covers this service.", BankReviewState: "PENDING"}, Authorize: func(context.Context) error { return errors.New("revoked route") }}
	if _, _, _, err := repo.WriteAssessmentCollection(ctx, record, nil); err == nil {
		t.Fatal("revoked authority accepted")
	}
	var receipts int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM capture_field_collection_resolutions`).Scan(&receipts); err != nil || receipts != 0 {
		t.Fatalf("denial left receipts %d %v", receipts, err)
	}
	record.Authorize = func(context.Context) error { return nil }
	if demo {
		if _, err = pool.Exec(ctx, `UPDATE capture_artifacts SET status='STORED_UNSCANNED' WHERE id=$1::uuid`, artifact.ID); err != nil {
			t.Fatal(err)
		}
		if _, _, _, err = repo.WriteAssessmentCollection(ctx, record, nil); !errors.Is(err, ErrAssessmentCompletionBlocked) {
			t.Fatalf("default policy accepted unscanned source: %v", err)
		}
		repo.ConfigureDemoUnscannedArtifacts(true)
	}

	saved, request, receipt, err := repo.WriteAssessmentCollection(ctx, record, nil)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Status != AssessmentUnderReview || saved.SubmissionID != "" || saved.SubmittedAt != nil || saved.CollectionCompletedAt == nil || request.Version != target.Version+1 || receipt.BankReviewState != "PENDING" {
		t.Fatalf("false collection completion: %+v %+v", saved, receipt)
	}
	if demo && (!receipt.Source.DemoUnscannedAllowed || receipt.Source.ArtifactStatus != evidence.ArtifactStoredUnscanned) {
		t.Fatalf("unscanned receipt lacks truthful exception: %+v", receipt)
	}
	reviewService := NewAssessmentReviewService(NewAssessmentService(repo, nil), repo, capture, nil)
	reviewService.ConfigureDemoUnscannedArtifacts(demo)
	if _, err := reviewService.GetReview(ctx, Actor{TenantID: input.TenantID, LegalEntityID: input.LegalEntityID, PrincipalID: thirdPartyPrincipal}, a.ID); err != nil {
		t.Fatalf("prepared review cannot open: %v", err)
	}
	var submissions, events, outbox int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM capture_submissions),(SELECT count(*) FROM third_party_events WHERE event_type='AssessmentDocumentReconciled'),(SELECT count(*) FROM outbox_events WHERE event_type='AssessmentDocumentReconciled')`).Scan(&submissions, &events, &outbox); err != nil {
		t.Fatal(err)
	}
	if submissions != 1 || events != 1 || outbox != 1 {
		t.Fatalf("transaction fragments: submissions=%d events=%d outbox=%d", submissions, events, outbox)
	}
	if _, _, _, err = repo.WriteAssessmentCollection(ctx, record, nil); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale command accepted: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_field_collection_resolutions SET receipt='{}'::jsonb WHERE id=$1::uuid`, receipt.ID); err == nil {
		t.Fatal("receipt was mutable")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err = repo.verifyPostgresAssessmentCompletionReady(ctx, tx, thirdPartyTenantID, saved); !errors.Is(err, ErrAssessmentCompletionBlocked) {
		t.Fatalf("pending review allowed completion: %v", err)
	}
	_ = tx.Rollback(ctx)
	record.ExpectedVersion = saved.Version
	record.ExpectedRequestVersion = request.Version
	record.Review = true
	record.Resolution = receipt
	record.Resolution.ID, _ = id.NewUUIDv7()
	record.Resolution.BankReviewState = "VALIDATED"
	record.Resolution.ReviewedBy = thirdPartyPrincipal
	reviewedAt := now.Add(2 * time.Minute)
	record.Resolution.ReviewedAt = &reviewedAt
	record.At = reviewedAt
	saved, request, _, err = repo.WriteAssessmentCollection(ctx, record, nil)
	if err != nil {
		t.Fatal(err)
	}
	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.verifyPostgresAssessmentCompletionReady(ctx, tx, thirdPartyTenantID, saved); err != nil {
		t.Fatalf("accepted held evidence blocked completion: %v", err)
	}
	_ = tx.Rollback(ctx)
	if demo {
		repo.ConfigureDemoUnscannedArtifacts(false)
		tx, err = pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err = repo.verifyPostgresAssessmentCompletionReady(ctx, tx, thirdPartyTenantID, saved); !errors.Is(err, ErrAssessmentCompletionBlocked) {
			t.Fatalf("revoked exception allowed completion: %v", err)
		}
		_ = tx.Rollback(ctx)
		repo.ConfigureDemoUnscannedArtifacts(true)
		if _, err = pool.Exec(ctx, `UPDATE capture_artifacts SET status='QUARANTINED' WHERE id=$1::uuid`, artifact.ID); err != nil {
			t.Fatal(err)
		}
		tx, err = pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err = repo.verifyPostgresAssessmentCompletionReady(ctx, tx, thirdPartyTenantID, saved); !errors.Is(err, ErrAssessmentCompletionBlocked) {
			t.Fatalf("quarantined source allowed completion: %v", err)
		}
		_ = tx.Rollback(ctx)
	}
	_, err = pool.Exec(ctx, `INSERT INTO third_party_documents(tenant_id,legal_entity_id,relationship_id,assessment_id,request_id,artifact_id,document_type,evidence_class,status,validated_by_principal_id,validated_at,version,created_at,updated_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,'SOC2','VENDOR_SUPPLIED','REJECTED',$7::uuid,$8,1,$8,$8)`, thirdPartyTenantID, thirdPartyEntityA, relationship.Relationship.ID, a.ID, sourceRequest.ID, artifact.ID, thirdPartyPrincipal, reviewedAt)
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := capture.GetRequest(ctx, input.TenantID, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.CollectionFieldFulfilled(refreshed.Fields[0], now) {
		t.Fatal("rejected source remained usable for collection")
	}
}
