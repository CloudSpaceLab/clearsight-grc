//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	assessmentTemplateID = "33333333-3333-7333-8333-333333333341"
	assessmentOneID      = "33333333-3333-7333-8333-333333333342"
	assessmentTwoID      = "33333333-3333-7333-8333-333333333343"
	assessmentThreeID    = "33333333-3333-7333-8333-333333333344"
)

func TestPostgresAssessmentRequestResolverRequiresExactCurrentCollectingLink(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	relationship := seedAssessmentRelationship(t, pool, "Managed identity verification")
	now := time.Now().UTC()
	repository := NewPostgresRepository(pool)
	assessment, err := repository.CreateAssessment(ctx, postgresAssessmentRecord(assessmentOneID, relationship, now))
	if err != nil {
		t.Fatal(err)
	}
	origin := evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: assessment.ID, Version: 1}
	ctx = evidence.WithRequestOriginAuthority(ctx, origin.Type)
	evidenceService := evidence.NewService(evidence.NewPostgresRepository(pool), evidence.NewMemoryObjectStore())
	request, err := evidenceService.CreateRequest(ctx, evidence.CreateRequestInput{
		TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: relationship.Relationship.ID,
		Title: "Vendor due diligence", Purpose: "Collect the vendor response.", WhyYou: "Provide the information required for the bank's review.",
		Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR",
		Recipient:        evidence.RecipientInput{Type: evidence.RecipientExternalAudience, Audience: "review@vendor.example"},
		EstimatedMinutes: 10, Deadline: now.Add(24 * time.Hour), Origin: origin,
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard},
		Sections:     []formcontract.Section{{ID: "company", Title: "Company details"}},
		Fields:       []evidence.Field{{ID: "confirmed", SectionID: "company", Label: "Confirm the supplied details", Type: string(formcontract.TypeYesNo), Required: true}},
		CreatedBy:    thirdPartyPrincipal,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE third_party_assessments SET status='COLLECTING',current_request_id=$3::uuid,version=4,updated_at=$4
		WHERE tenant_id=$1::uuid AND id=$2::uuid;
		INSERT INTO third_party_assessment_request_links(
			tenant_id,legal_entity_id,assessment_id,request_id,purpose,sequence,origin_type,origin_id,origin_sequence,is_current,created_at
		) VALUES($1::uuid,$5::uuid,$2::uuid,$3::uuid,'INITIAL',1,$6,$2::uuid,1,true,$4)`,
		pgx.QueryExecModeSimpleProtocol, thirdPartyTenantID, assessment.ID, request.ID, now, thirdPartyEntityA, AssessmentRequestOrigin); err != nil {
		t.Fatal(err)
	}

	target, err := repository.ResolveAssessmentRequest(ctx, "third-party-bank", origin, request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if target.TenantID != "third-party-bank" || target.LegalEntityID != thirdPartyEntityA || target.AssessmentID != assessment.ID || target.AssessmentVersion != 4 || target.RequestID != request.ID {
		t.Fatalf("target = %#v", target)
	}
	for _, mismatch := range []struct {
		tenant    string
		origin    evidence.RequestOrigin
		requestID string
	}{
		{tenant: "missing-bank", origin: origin, requestID: request.ID},
		{tenant: "third-party-bank", origin: evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: assessment.ID, Version: 2}, requestID: request.ID},
		{tenant: "third-party-bank", origin: origin, requestID: assessmentTwoID},
	} {
		if _, err := repository.ResolveAssessmentRequest(ctx, mismatch.tenant, mismatch.origin, mismatch.requestID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("mismatch %#v returned %v", mismatch, err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE third_party_assessment_request_links SET is_current=false WHERE tenant_id=$1::uuid AND assessment_id=$2::uuid`, thirdPartyTenantID, assessment.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.ResolveAssessmentRequest(ctx, "third-party-bank", origin, request.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("non-current link returned %v", err)
	}
}

func TestPostgresAssessmentRequestReissueCommitsSafeAuditAndRevokesPriorSession(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	relationship := seedAssessmentRelationship(t, pool, "Managed secure document exchange")
	now := time.Now().UTC()
	repository := NewPostgresRepository(pool)
	assessment, err := repository.CreateAssessment(ctx, postgresAssessmentRecord(assessmentOneID, relationship, now))
	if err != nil {
		t.Fatal(err)
	}
	origin := evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: assessment.ID, Version: 1}
	ctx = evidence.WithRequestOriginAuthority(ctx, origin.Type)
	evidenceService := evidence.NewService(evidence.NewPostgresRepository(pool), evidence.NewMemoryObjectStore())
	audience := "review@vendor.example"
	request, err := evidenceService.CreateRequest(ctx, evidence.CreateRequestInput{
		TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: relationship.Relationship.ID,
		Title: "Vendor due diligence", Purpose: "Collect the vendor response.", WhyYou: "Provide the information required for the bank's review.",
		Sensitivity: "CONFIDENTIAL", AudienceType: "VENDOR",
		Recipient:        evidence.RecipientInput{Type: evidence.RecipientExternalAudience, Audience: audience},
		EstimatedMinutes: 10, Deadline: now.Add(24 * time.Hour), Origin: origin,
		Presentation:   evidenceAssessmentPresentation(),
		Sections:       []formcontract.Section{{ID: "company", Title: "Company details"}},
		Fields:         []evidence.Field{{ID: "confirmed", SectionID: "company", Label: "Confirm the supplied details", Type: string(formcontract.TypeYesNo), Required: true}},
		FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3, CreatedBy: thirdPartyPrincipal,
	})
	if err != nil {
		t.Fatal(err)
	}
	initial, err := evidenceService.IssueInvitation(ctx, evidence.IssueInvitationInput{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA, RequestID: request.ID, Audience: audience, Purpose: "Complete the request.", TTLMinutes: 60, CreatedBy: thirdPartyPrincipal})
	if err != nil {
		t.Fatal(err)
	}
	priorSession, err := evidenceService.RedeemInvitation(ctx, initial.Token, audience)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE third_party_assessments SET status='COLLECTING',current_request_id=$3::uuid,version=4,updated_at=$4
		WHERE tenant_id=$1::uuid AND id=$2::uuid;
		INSERT INTO third_party_assessment_request_links(
			tenant_id,legal_entity_id,assessment_id,request_id,purpose,sequence,origin_type,origin_id,origin_sequence,invitation_id,is_current,created_at
		) VALUES($1::uuid,$5::uuid,$2::uuid,$3::uuid,'INITIAL',1,$6,$2::uuid,1,$7::uuid,true,$4)`,
		pgx.QueryExecModeSimpleProtocol, thirdPartyTenantID, assessment.ID, request.ID, now, thirdPartyEntityA, AssessmentRequestOrigin, initial.InvitationID); err != nil {
		t.Fatal(err)
	}
	preparedLink, preparedAssessment, err := repository.PrepareRequestReissue(ctx, PrepareRequestReissueRecord{
		Scope: Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}, AssessmentID: assessment.ID, ExpectedVersion: 4,
		ActorPrincipalID: thirdPartyPrincipal, RequestID: request.ID, ExpectedInvitationID: initial.InvitationID, PreparedAt: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if preparedAssessment.Status != AssessmentCollecting || preparedAssessment.Version != 5 || preparedLink.InvitationID != "" {
		t.Fatalf("replacement preparation changed lifecycle or retained invitation: assessment=%#v link=%#v", preparedAssessment, preparedLink)
	}
	replacement, err := evidenceService.IssueInvitation(ctx, evidence.IssueInvitationInput{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA, RequestID: request.ID, Audience: audience, Purpose: "Complete the request.", TTLMinutes: 60, CreatedBy: thirdPartyPrincipal})
	if err != nil {
		t.Fatal(err)
	}
	link, updated, err := repository.FinalizeRequestReissue(ctx, FinalizeRequestReissueRecord{
		Scope: Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}, AssessmentID: assessment.ID, ExpectedVersion: preparedAssessment.Version,
		ActorPrincipalID: thirdPartyPrincipal, RequestID: request.ID,
		InvitationID: replacement.InvitationID, ReissuedAt: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != AssessmentCollecting || updated.Version != 6 || link.InvitationID != replacement.InvitationID {
		t.Fatalf("replacement changed lifecycle or failed to update link: assessment=%#v link=%#v", updated, link)
	}
	if _, _, err := evidenceService.SessionRequest(ctx, priorSession.SessionToken); !errors.Is(err, evidence.ErrSessionInvalid) {
		t.Fatalf("prior session remained usable: %v", err)
	}
	if _, err := evidenceService.RedeemInvitation(ctx, replacement.Token, audience); err != nil {
		t.Fatalf("replacement invitation was not redeemable: %v", err)
	}
	var actorID, eventPayload, outboxPayload string
	if err := pool.QueryRow(ctx, `SELECT actor_principal_id::text,payload::text FROM third_party_events WHERE aggregate_type='THIRD_PARTY_ASSESSMENT' AND aggregate_id=$1::uuid AND event_type='AssessmentRequestReissued'`, assessment.ID).Scan(&actorID, &eventPayload); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT payload::text FROM outbox_events WHERE aggregate_type='THIRD_PARTY_ASSESSMENT' AND aggregate_id=$1::uuid AND event_type='AssessmentRequestReissued'`, assessment.ID).Scan(&outboxPayload); err != nil {
		t.Fatal(err)
	}
	if actorID != thirdPartyPrincipal {
		t.Fatalf("audit actor = %q", actorID)
	}
	var preparedActorID, preparedEventPayload, preparedOutboxPayload string
	if err := pool.QueryRow(ctx, `SELECT actor_principal_id::text,payload::text FROM third_party_events WHERE aggregate_type='THIRD_PARTY_ASSESSMENT' AND aggregate_id=$1::uuid AND event_type='AssessmentRequestReissuePrepared'`, assessment.ID).Scan(&preparedActorID, &preparedEventPayload); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT payload::text FROM outbox_events WHERE aggregate_type='THIRD_PARTY_ASSESSMENT' AND aggregate_id=$1::uuid AND event_type='AssessmentRequestReissuePrepared'`, assessment.ID).Scan(&preparedOutboxPayload); err != nil {
		t.Fatal(err)
	}
	if preparedActorID != thirdPartyPrincipal {
		t.Fatalf("prepared audit actor = %q", preparedActorID)
	}
	for _, protected := range []string{audience, initial.Token, replacement.Token} {
		if strings.Contains(eventPayload, protected) || strings.Contains(outboxPayload, protected) || strings.Contains(preparedEventPayload, protected) || strings.Contains(preparedOutboxPayload, protected) {
			t.Fatalf("protected value leaked into audit persistence: prepared_event=%s prepared_outbox=%s event=%s outbox=%s", preparedEventPayload, preparedOutboxPayload, eventPayload, outboxPayload)
		}
	}
}

func TestPostgresAssessmentDocumentReviewCommitsDocumentAssessmentEventAndOutbox(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	relationship := seedAssessmentRelationship(t, pool, "Managed assurance reporting")
	now := time.Now().UTC()
	repository := NewPostgresRepository(pool)
	assessment, err := repository.CreateAssessment(ctx, postgresAssessmentRecord(assessmentOneID, relationship, now))
	if err != nil {
		t.Fatal(err)
	}
	ctx = evidence.WithRequestOriginAuthority(ctx, AssessmentRequestOrigin)
	evidenceService := evidence.NewService(evidence.NewPostgresRepository(pool), evidence.NewMemoryObjectStore())
	request, err := evidenceService.CreateRequest(ctx, evidence.CreateRequestInput{
		TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA, SubjectType: "VENDOR_RELATIONSHIP", SubjectID: relationship.Relationship.ID,
		Title: "Vendor due diligence", Purpose: "Collect the current assurance report.", WhyYou: "Provide the report required for the bank's review.",
		Sensitivity: "CONFIDENTIAL", AudienceType: "INTERNAL",
		Recipient:        evidence.RecipientInput{Type: evidence.RecipientInternalPrincipal, PrincipalID: thirdPartyPrincipal},
		EstimatedMinutes: 5, Deadline: now.Add(24 * time.Hour),
		Origin:       evidence.RequestOrigin{Type: AssessmentRequestOrigin, ID: assessment.ID, Version: 1},
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard},
		Sections:     []formcontract.Section{{ID: "documents", Title: "Assurance documents"}},
		Fields: []evidence.Field{{
			ID: "assurance_report", SectionID: "documents", Label: "Assurance report", Type: string(formcontract.TypeVendorDocument), Required: true,
			AcceptedFormats: []string{"application/pdf"},
		}},
		FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3, CreatedBy: thirdPartyPrincipal,
	})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := evidenceService.StoreArtifact(ctx, evidence.ArtifactInput{
		TenantID: "third-party-bank", RequestID: request.ID, FileName: "assurance-report.pdf", MediaType: "application/pdf", CreatedBy: thirdPartyPrincipal,
	}, strings.NewReader("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n%%EOF"))
	if err != nil {
		t.Fatal(err)
	}
	documentAnswer := formcontract.DocumentAnswer{
		ArtifactID: artifact.ID, DocumentType: "SOC_2_TYPE_II", Reference: "SOC2-2026", IssuedBy: "Independent auditor", IssuedOn: "2026-06-01", ExpiresOn: "2027-05-31",
	}
	receipt, err := evidenceService.Submit(ctx, evidence.Submission{
		TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA, RequestID: request.ID, SubmittedBy: thirdPartyPrincipal, Channel: "INTERNAL", ExpectedVersion: request.Version,
		Answers: map[string]formcontract.AnswerValue{"assurance_report": {Document: &documentAnswer}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE capture_artifacts SET status='AVAILABLE' WHERE tenant_id=$1::uuid AND request_id=$2::uuid AND submission_id=$3::uuid AND id=$4::uuid;
		UPDATE third_party_assessments SET status='UNDER_REVIEW',current_request_id=$2::uuid,submission_id=$3::uuid,submitted_at=$5,review_started_at=$5,
			reviewer_principal_id=$6::uuid,version=4,updated_at=$5 WHERE tenant_id=$1::uuid AND id=$7::uuid;
		INSERT INTO third_party_assessment_request_links(
			tenant_id,legal_entity_id,assessment_id,request_id,purpose,sequence,origin_type,origin_id,origin_sequence,is_current,created_at
		) VALUES($1::uuid,$8::uuid,$7::uuid,$2::uuid,'INITIAL',1,$9,$7::uuid,1,true,$5)`,
		pgx.QueryExecModeSimpleProtocol, thirdPartyTenantID, request.ID, receipt.SubmissionID, artifact.ID, now, thirdPartyPrincipal, assessment.ID, thirdPartyEntityA, AssessmentRequestOrigin); err != nil {
		t.Fatal(err)
	}
	artifact.Status = evidence.ArtifactAvailable
	artifact.SubmissionID = receipt.SubmissionID
	expiresOn := time.Date(2027, 5, 31, 0, 0, 0, 0, time.UTC)
	document, updated, err := repository.ReviewAssessmentDocument(ctx, AssessmentDocumentReviewRecord{
		Scope: Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}, AssessmentID: assessment.ID, ExpectedVersion: 4,
		ActorPrincipalID: thirdPartyPrincipal, Artifact: artifact, Document: documentAnswer, Decision: AssessmentDocumentValidate,
		DocumentType: "SOC_2_TYPE_II", EvidenceClass: AssessmentDocumentBankValidated, ExpiresOn: &expiresOn, At: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if document.Status != AssessmentDocumentValidated || document.Version != 1 || updated.Version != 5 || updated.Status != AssessmentUnderReview {
		t.Fatalf("unexpected committed review document=%#v assessment=%#v", document, updated)
	}
	var eventCount, outboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM third_party_events WHERE aggregate_id=$1::uuid AND event_type='AssessmentDocumentValidated'`, assessment.ID).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_id=$1::uuid AND event_type='AssessmentDocumentValidated'`, assessment.ID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if eventCount != 1 || outboxCount != 1 {
		t.Fatalf("document review audit transaction event=%d outbox=%d", eventCount, outboxCount)
	}
}

func evidenceAssessmentPresentation() formcontract.Presentation {
	return formcontract.Presentation{DefaultMode: formcontract.PresentationWizard}
}

func TestPostgresAssessmentStartCommitsOneEpisodeEventOutboxAndJob(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	relationship := seedAssessmentRelationship(t, pool, "Managed card processing")
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	record := postgresAssessmentRecord(assessmentOneID, relationship, now)
	repo := NewPostgresRepository(pool)

	created, err := repo.CreateAssessment(ctx, record)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != assessmentOneID || created.Status != AssessmentSetupPending || created.StartedByPrincipalID != thirdPartyPrincipal {
		t.Fatalf("unexpected assessment: %#v", created)
	}
	assertAssessmentCount(t, pool, "third_party_assessments", 1)
	assertAssessmentCount(t, pool, "third_party_assessment_jobs", 1)
	assertAssessmentTypedCount(t, pool, "third_party_events", "THIRD_PARTY_ASSESSMENT", 1)
	assertAssessmentTypedCount(t, pool, "outbox_events", "THIRD_PARTY_ASSESSMENT", 1)
	events, err := repo.ListAssessmentEvents(ctx, record.Scope, created.ID, created.Version)
	if err != nil || len(events) != 1 || events[0].AssessmentVersion != 1 || events[0].Payload["status"] != string(AssessmentSetupPending) {
		t.Fatalf("assessment history=%#v err=%v", events, err)
	}

	replay := record
	replay.Assessment.ID = assessmentTwoID
	replay.RelationshipVersion++
	replayed, err := repo.CreateAssessment(ctx, replay)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != created.ID {
		t.Fatalf("stable replay created another episode: first=%s replay=%s", created.ID, replayed.ID)
	}
	assertAssessmentCount(t, pool, "third_party_assessments", 1)
	assertAssessmentCount(t, pool, "third_party_assessment_jobs", 1)
	assertAssessmentTypedCount(t, pool, "third_party_events", "THIRD_PARTY_ASSESSMENT", 1)
	assertAssessmentTypedCount(t, pool, "outbox_events", "THIRD_PARTY_ASSESSMENT", 1)
}

func TestPostgresAssessmentConcurrentStartDeduplicatesAndScopeFailuresWriteNothing(t *testing.T) {
	pool := assessmentPostgresPool(t)
	ctx := context.Background()
	relationship := seedAssessmentRelationship(t, pool, "Payment tokenization")
	now := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	recordA := postgresAssessmentRecord(assessmentOneID, relationship, now)
	recordB := postgresAssessmentRecord(assessmentTwoID, relationship, now)
	repo := NewPostgresRepository(pool)

	var wg sync.WaitGroup
	results := make(chan Assessment, 2)
	errs := make(chan error, 2)
	for _, record := range []CreateAssessmentRecord{recordA, recordB} {
		wg.Add(1)
		go func(value CreateAssessmentRecord) {
			defer wg.Done()
			created, err := repo.CreateAssessment(ctx, value)
			results <- created
			errs <- err
		}(record)
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent start: %v", err)
		}
	}
	var episodeID string
	for value := range results {
		if episodeID == "" {
			episodeID = value.ID
		}
		if value.ID != episodeID {
			t.Fatalf("concurrent starts returned different episodes: %s and %s", episodeID, value.ID)
		}
	}
	assertAssessmentCount(t, pool, "third_party_assessments", 1)
	assertAssessmentCount(t, pool, "third_party_assessment_jobs", 1)

	beforeEvents := assessmentTypedCount(t, pool, "third_party_events", "THIRD_PARTY_ASSESSMENT")
	beforeOutbox := assessmentTypedCount(t, pool, "outbox_events", "THIRD_PARTY_ASSESSMENT")
	outside := recordA
	outside.LegalEntityID = thirdPartyEntityB
	outside.Assessment.LegalEntityID = thirdPartyEntityB
	outside.Assessment.StableEpisodeKey = assessmentEpisodeKey(outside.Scope, relationship.Relationship.ID, AssessmentReviewOnboarding)
	if _, err := repo.CreateAssessment(ctx, outside); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity start returned %v", err)
	}
	if assessmentTypedCount(t, pool, "third_party_events", "THIRD_PARTY_ASSESSMENT") != beforeEvents || assessmentTypedCount(t, pool, "outbox_events", "THIRD_PARTY_ASSESSMENT") != beforeOutbox {
		t.Fatal("failed scoped start wrote event or outbox state")
	}

	staleRelationship := seedAssessmentRelationship(t, pool, "Customer notification delivery")
	stale := postgresAssessmentRecord(assessmentThreeID, staleRelationship, now)
	stale.RelationshipVersion++
	if _, err := repo.CreateAssessment(ctx, stale); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale relationship start returned %v", err)
	}
	if assessmentTypedCount(t, pool, "third_party_events", "THIRD_PARTY_ASSESSMENT") != beforeEvents || assessmentTypedCount(t, pool, "outbox_events", "THIRD_PARTY_ASSESSMENT") != beforeOutbox {
		t.Fatal("stale start wrote event or outbox state")
	}
}

func assessmentPostgresPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(context.Background(), `TRUNCATE tenants CASCADE`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'third-party-bank','Third Party Bank');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES
			($2::uuid,$1::uuid,'ENTITY-A','Entity A','Nigeria'),($3::uuid,$1::uuid,'ENTITY-B','Entity B','Nigeria');
		INSERT INTO principals(id,tenant_id,kind,display_name,status) VALUES($4::uuid,$1::uuid,'PERSON','Vendor Owner','ACTIVE');
		INSERT INTO monitoring_form_templates(id,tenant_id,code,name,purpose,presentation,sections,fields,status,is_current,effective_from,version,created_by)
		VALUES($5::uuid,$1::uuid,'VENDOR-DD','Vendor due diligence','Collect current vendor control information.',
			'{"default_mode":"WIZARD","allow_mode_switch":true}'::jsonb,
			'[{"id":"company","title":"Company details"}]'::jsonb,
			'[{"id":"confirmed","section_id":"company","label":"Confirm the supplied details","type":"yes_no","required":true}]'::jsonb,
			'ACTIVE',true,'2026-08-26T09:00:00Z',3,$4::uuid)`,
		pgx.QueryExecModeSimpleProtocol, thirdPartyTenantID, thirdPartyEntityA, thirdPartyEntityB, thirdPartyPrincipal, assessmentTemplateID); err != nil {
		t.Fatal(err)
	}
	return pool
}

func seedAssessmentRelationship(t *testing.T, pool *pgxpool.Pool, serviceName string) Aggregate {
	t.Helper()
	service := NewService(NewPostgresRepository(pool))
	input := validPostgresCreateInput(serviceName)
	input.ExternalRef = serviceName
	value, err := service.CreateRelationship(context.Background(), Actor{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA, PrincipalID: thirdPartyPrincipal}, input)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func postgresAssessmentRecord(id string, relationship Aggregate, now time.Time) CreateAssessmentRecord {
	scope := Scope{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA}
	return CreateAssessmentRecord{
		Scope: scope, RelationshipID: relationship.Relationship.ID, RelationshipVersion: relationship.Relationship.Version,
		Assessment: Assessment{
			ID: id, TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, RelationshipID: relationship.Relationship.ID,
			ReviewKind: AssessmentReviewOnboarding, SourceTrigger: "INITIAL", StableEpisodeKey: assessmentEpisodeKey(scope, relationship.Relationship.ID, AssessmentReviewOnboarding),
			Status: AssessmentSetupPending, FormTemplateID: assessmentTemplateID, FormTemplateVersion: 3,
			ReviewDueAt: now.Add(14 * 24 * time.Hour), StartedByPrincipalID: thirdPartyPrincipal, StartedAt: now,
			Version: 1, CreatedAt: now, UpdatedAt: now,
		},
	}
}

func assertAssessmentCount(t *testing.T, pool *pgxpool.Pool, table string, want int) {
	t.Helper()
	allowed := map[string]bool{"third_party_assessments": true, "third_party_assessment_jobs": true}
	if !allowed[table] {
		t.Fatalf("unsupported assessment table %q", table)
	}
	var got int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM `+table).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s count=%d want=%d", table, got, want)
	}
}

func assertAssessmentTypedCount(t *testing.T, pool *pgxpool.Pool, table, aggregateType string, want int) {
	t.Helper()
	if got := assessmentTypedCount(t, pool, table, aggregateType); got != want {
		t.Fatalf("%s %s count=%d want=%d", table, aggregateType, got, want)
	}
}

func assessmentTypedCount(t *testing.T, pool *pgxpool.Pool, table, aggregateType string) int {
	t.Helper()
	if table != "third_party_events" && table != "outbox_events" {
		t.Fatalf("unsupported event table %q", table)
	}
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM `+table+` WHERE aggregate_type=$1`, aggregateType).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
