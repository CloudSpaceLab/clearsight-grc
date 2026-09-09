//go:build postgres && postgresintegration

package monitoring

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresFormProposalAcceptanceIsAtomicAndAuditable(t *testing.T) {
	for _, useTenantUUID := range []bool{false, true} {
		name := "tenant_slug"
		if useTenantUUID {
			name = "verified_tenant_uuid"
		}
		t.Run(name, func(t *testing.T) { testPostgresFormProposalAcceptanceIsAtomicAndAuditable(t, useTenantUUID) })
	}
}

func TestPostgresFindingFollowUpAcceptanceIsAtomicAndAuditable(t *testing.T) {
	testPostgresFormProposalAcceptanceIsAtomicAndAuditable(t, true, "assessment-1")
}

func testPostgresFormProposalAcceptanceIsAtomicAndAuditable(t *testing.T, useTenantUUID bool, assessmentIDs ...string) {
	assessmentID := ""
	if len(assessmentIDs) > 0 {
		assessmentID = assessmentIDs[0]
	}
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const (
		tenantID    = "9d111111-1111-7111-8111-111111111111"
		entityID    = "9d111111-1111-7111-8111-111111111112"
		principalID = "9d111111-1111-7111-8111-111111111113"
		documentID  = "9d111111-1111-7111-8111-111111111114"
		proposalID  = "9d111111-1111-7111-8111-111111111115"
		templateID  = "9d111111-1111-7111-8111-111111111116"
		changeID    = "change_legal_name"
	)
	const tenantSlug = "form-proposal-atomic-test"
	actorTenantID := tenantSlug
	if useTenantUUID {
		actorTenantID = tenantID
	}
	sha256 := strings.Repeat("a", 64)
	now := time.Date(2026, 8, 29, 15, 45, 0, 0, time.UTC)

	cleanupFormProposalIntegration(ctx, pool, tenantID)
	defer cleanupFormProposalIntegration(context.Background(), pool, tenantID)

	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Form Proposal Atomic Test');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($3::uuid,$1::uuid,'FP-ATOMIC','Form Proposal Entity','NG');
		INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($4::uuid,$1::uuid,'PERSON','Form Proposal Maker');
		INSERT INTO document_imports(
			id,tenant_id,legal_entity_id,file_name,media_type,purpose,source_type,size_bytes,sha256,storage_key,
			artifact_status,extraction_status,extraction_method,analysis_status,analysis_method,created_by,created_at,updated_at,version
		) VALUES(
			$5::uuid,$1::uuid,$3::uuid,'questionnaire.docx','application/vnd.openxmlformats-officedocument.wordprocessingml.document',
			'Create a governed form','DOCUMENT',1,$6,'imports/form-proposal-atomic-test','AVAILABLE','EXTRACTED','DOCX_XML_STREAM_V3',
			'NO_PROPOSALS','NONE',$4::uuid,$7,$7,1
		)`, pgx.QueryExecModeSimpleProtocol, tenantID, tenantSlug, entityID, principalID, documentID, sha256, now); err != nil {
		t.Fatal(err)
	}

	contract, err := formcontract.Normalize(formcontract.Contract{
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationAutomatic, AllowModeSwitch: true},
		ScoringMode:  formcontract.ScoringNone,
		Sections:     []formcontract.Section{{ID: formcontract.DefaultSectionID, Title: "General"}},
		Fields: []formcontract.Field{{
			ID: "legal_name", SectionID: formcontract.DefaultSectionID, Label: "Legal name", Type: formcontract.TypeShortText,
			CollectionIntent: formcontract.IntentCapture, BrowserCachePolicy: formcontract.BrowserCacheAllowed,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	store := NewPostgresFormProposalStore(pool)
	created, err := store.Create(ctx, FormTemplateProposal{
		FindingAssessmentID: assessmentID,
		ID:                  proposalID, TenantID: tenantSlug, LegalEntityID: entityID,
		SourceKind: FormProposalSourceDocument, SourceDocumentID: documentID, SourceDocumentVersion: 1, SourceSHA256: sha256,
		Status: FormProposalGenerating, CreatedBy: principalID, CreatedAt: now, UpdatedAt: now, Version: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.FindingAssessmentID != assessmentID {
		t.Fatal("assessment identity lost in persistence")
	}
	if assessmentID != "" {
		second := created
		second.ID = "9d111111-1111-7111-8111-111111111117"
		second.FindingAssessmentID = "assessment-2"
		got, err := store.Create(ctx, second)
		if err != nil || got.ID != second.ID || got.FindingAssessmentID != second.FindingAssessmentID {
			t.Fatalf("assessment receipt collided: %#v %v", got, err)
		}
		again, err := store.Create(ctx, second)
		if err != nil || again.ID != got.ID {
			t.Fatalf("duplicate assessment receipt: %#v %v", again, err)
		}
	}
	generatedInput := created
	generatedInput.ProposedContract = contract
	generatedInput.FieldChanges = []documentimport.FormFieldChange{{ID: changeID, Kind: "ADD_FIELD", Field: contract.Fields[0]}}
	generatedInput.Provenance = FormProposalProvenance{
		FormProposalProvenance: documentimport.FormProposalProvenance{
			ProposalVersion: "FORM_TEMPLATE_PROPOSAL_V1", SourceDocumentID: documentID, SourceSHA256: sha256,
			SourceVersion: 1, ParserVersion: "DOCX_XML_STREAM_V3", ExtractionStatus: string(documentimport.ExtractionExtracted),
		},
	}
	generatedInput.UpdatedAt = now.Add(time.Second)
	generated, err := store.CompleteGeneration(ctx, generatedInput, created.Version)
	if err != nil {
		t.Fatal(err)
	}
	if generated.Status != FormProposalReviewRequired || generated.Version != 2 {
		t.Fatalf("proposal did not reach review state: %#v", generated)
	}

	draft := FormTemplate{
		ID: templateID, TenantID: actorTenantID, LegalEntityID: entityID,
		Code: "IMPORTED-VENDOR", Name: "Imported vendor questionnaire", Purpose: "Review imported vendor facts.",
		OwnerPrincipalID: principalID, ScoringMode: contract.ScoringMode, Presentation: contract.Presentation,
		Sections: contract.Sections, Fields: contract.Fields,
		Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1, CreatedBy: principalID, CreatedAt: now.Add(2 * time.Second), UpdatedAt: now.Add(2 * time.Second)},
	}
	mutation := FormProposalReviewMutation{
		AssessmentConfirmed: assessmentID != "",
		TenantID:            actorTenantID, LegalEntityID: entityID, ProposalID: proposalID, ExpectedVersion: generated.Version,
		Status: FormProposalAccepted, ReviewerID: principalID, ChangeIDs: []string{changeID}, At: now.Add(2 * time.Second),
	}
	if assessmentID != "" {
		unconfirmed := mutation
		unconfirmed.AssessmentConfirmed = false
		if _, err := store.AcceptWithDraft(ctx, unconfirmed, draft); !errors.Is(err, ErrFormProposalSelection) {
			t.Fatalf("unconfirmed assessment accepted: %v", err)
		}
	}

	const otherTenantID = "9d111111-1111-7111-8111-111111111119"
	cleanupFormProposalIntegration(ctx, pool, otherTenantID)
	defer cleanupFormProposalIntegration(context.Background(), pool, otherTenantID)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'other-form-proposal-tenant','Other proposal tenant')`, otherTenantID); err != nil {
		t.Fatal(err)
	}

	for _, mismatch := range []struct{ name, tenant, entity string }{
		{"unknown tenant", "unknown-form-proposal-tenant", entityID},
		{"another tenant UUID", "9d111111-1111-7111-8111-111111111119", entityID},
		{"another legal entity", actorTenantID, "9d111111-1111-7111-8111-111111111118"},
	} {
		invalidDraft := draft
		invalidDraft.TenantID, invalidDraft.LegalEntityID = mismatch.tenant, mismatch.entity
		if _, err := store.AcceptWithDraft(ctx, mutation, invalidDraft); !errors.Is(err, ErrFormProposalSourceChanged) {
			t.Fatalf("%s acceptance error = %v, want source changed", mismatch.name, err)
		}
	}
	// A source version change must still reject the whole material transaction.
	if _, err := pool.Exec(ctx, `UPDATE document_imports SET version=2 WHERE id=$1::uuid`, documentID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AcceptWithDraft(ctx, mutation, draft); !errors.Is(err, ErrFormProposalSourceChanged) {
		t.Fatalf("changed source acceptance error = %v, want source changed", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE document_imports SET version=1 WHERE id=$1::uuid`, documentID); err != nil {
		t.Fatal(err)
	}

	if _, err := pool.Exec(ctx, `
		CREATE FUNCTION form_proposal_accept_outbox_failure_test() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.aggregate_type='FORM_TEMPLATE_PROPOSAL' AND NEW.event_type='FORM_TEMPLATE_PROPOSAL_ACCEPTED' THEN
				RAISE EXCEPTION 'forced form proposal acceptance outbox failure';
			END IF;
			RETURN NEW;
		END $$;
		CREATE TRIGGER form_proposal_accept_outbox_failure_test
			BEFORE INSERT ON outbox_events FOR EACH ROW EXECUTE FUNCTION form_proposal_accept_outbox_failure_test()`, pgx.QueryExecModeSimpleProtocol); err != nil {
		t.Fatal(err)
	}

	if _, err := store.AcceptWithDraft(ctx, mutation, draft); err == nil {
		t.Fatal("expected forced acceptance outbox failure")
	}
	var draftRows, formEvents, acceptedOutbox int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM monitoring_form_templates WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, templateID).Scan(&draftRows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM monitoring_events WHERE tenant_id=$1::uuid AND aggregate_type='MONITORING_FORM' AND aggregate_id=$2::uuid`, tenantID, templateID).Scan(&formEvents); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_type='FORM_TEMPLATE_PROPOSAL' AND aggregate_id=$2::uuid AND event_type=$3`, tenantID, proposalID, EventFormProposalAccepted).Scan(&acceptedOutbox); err != nil {
		t.Fatal(err)
	}
	rolledBack, err := store.Get(ctx, tenantSlug, entityID, proposalID)
	if err != nil {
		t.Fatal(err)
	}
	if draftRows != 0 || formEvents != 0 || acceptedOutbox != 0 || rolledBack.Status != FormProposalReviewRequired || rolledBack.Version != generated.Version || len(rolledBack.AcceptedChangeIDs) != 0 {
		t.Fatalf("failed acceptance leaked state: draft=%d events=%d outbox=%d proposal=%#v", draftRows, formEvents, acceptedOutbox, rolledBack)
	}

	if _, err := pool.Exec(ctx, `
		DROP TRIGGER form_proposal_accept_outbox_failure_test ON outbox_events;
		DROP FUNCTION form_proposal_accept_outbox_failure_test()`, pgx.QueryExecModeSimpleProtocol); err != nil {
		t.Fatal(err)
	}

	if assessmentID != "" {
		var workers sync.WaitGroup
		failures := make(chan error, 8)
		for i := 0; i < 8; i++ {
			workers.Add(1)
			go func() { defer workers.Done(); _, err := store.AcceptWithDraft(ctx, mutation, draft); failures <- err }()
		}
		workers.Wait()
		close(failures)
		for err := range failures {
			if err != nil {
				t.Fatalf("concurrent acceptance: %v", err)
			}
		}
	}
	accepted, err := store.AcceptWithDraft(ctx, mutation, draft)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != FormProposalAccepted || accepted.ResultTemplateID != templateID || accepted.ResultTemplateVersion != 1 || !slices.Equal(accepted.AcceptedChangeIDs, []string{changeID}) {
		t.Fatalf("accepted proposal lost audit state: %#v", accepted)
	}
	repeated, err := store.AcceptWithDraft(ctx, mutation, draft)
	if err != nil || repeated.Version != accepted.Version || repeated.ResultTemplateID != accepted.ResultTemplateID {
		t.Fatalf("repeated acceptance changed the result: proposal=%#v error=%v", repeated, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM monitoring_form_templates WHERE tenant_id=$1::uuid AND id=$2::uuid AND version=1 AND status='DRAFT'`, tenantID, templateID).Scan(&draftRows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_type='FORM_TEMPLATE_PROPOSAL' AND aggregate_id=$2::uuid AND event_type=$3`, tenantID, proposalID, EventFormProposalAccepted).Scan(&acceptedOutbox); err != nil {
		t.Fatal(err)
	}
	if draftRows != 1 || acceptedOutbox != 1 {
		t.Fatalf("accepted transaction rows = draft %d / acceptance outbox %d, want 1/1", draftRows, acceptedOutbox)
	}
}

func cleanupFormProposalIntegration(ctx context.Context, pool *pgxpool.Pool, tenantID string) {
	_, _ = pool.Exec(ctx, `DROP TRIGGER IF EXISTS form_proposal_accept_outbox_failure_test ON outbox_events`)
	_, _ = pool.Exec(ctx, `DROP FUNCTION IF EXISTS form_proposal_accept_outbox_failure_test()`)
	_, _ = pool.Exec(ctx, `DELETE FROM outbox_events WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM monitoring_events WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM form_template_proposals WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM monitoring_form_templates WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM document_imports WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
}
