//go:build postgres && postgresintegration

package monitoring_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/frameworkcatalog"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"

	"strings"
	"testing"
	"time"
)

func TestPostgresAdoptedAssessmentDraftSubmissionAndReview(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	uuid := func() string {
		t.Helper()
		id, e := platformid.NewUUIDv7()
		if e != nil {
			t.Fatal(e)
		}
		return id
	}
	tenant, entity, owner, checker, respondent, unrelated := uuid(), uuid(), uuid(), uuid(), uuid(), uuid()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := pool.Exec(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	exec(`INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Isolated ISO assessment acceptance fixture')`, tenant, "iso-assessment-"+tenant)
	exec(`INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,'ISO-ASSESSMENT','Isolated assessment entity','NG')`, entity, tenant)
	exec(`INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$5::uuid,'PERSON','Program Owner'),($2::uuid,$5::uuid,'PERSON','Independent reviewer'),($3::uuid,$5::uuid,'PERSON','Digital banking operations respondent'),($4::uuid,$5::uuid,'PERSON','Unrelated person')`, owner, checker, respondent, unrelated, tenant)
	var unit string
	err = pool.QueryRow(ctx, `INSERT INTO organization_scopes(tenant_id,legal_entity_id,code,name,kind,department_path,origin) VALUES($1::uuid,$2::uuid,'OPERATIONS','Digital banking operations','DEPARTMENT',ARRAY['Digital banking operations'],'MANAGED') RETURNING id::text`, tenant, entity).Scan(&unit)
	if err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO org_positions(tenant_id,legal_entity_id,code,title,occupant_principal_id,organization_scope_id) VALUES($1::uuid,$2::uuid,'DB-OPERATIONS','Digital banking operations respondent',$3::uuid,$4::uuid)`, tenant, entity, respondent, unit)
	catalog := frameworkcatalog.NewPostgresRepository(pool)
	source, err := catalog.StarterRequirements(ctx, tenant, entity, "ISO_27001_2022")
	if err != nil || len(source) != 5 {
		t.Fatalf("starter: %v, groups=%d", err, len(source))
	}
	formID := source[0].FormID
	// This is an explicitly isolated test fixture, never a bank publication or approval.
	exec(`INSERT INTO monitoring_form_templates(id,tenant_id,legal_entity_id,code,name,purpose,responsible_team,approved_uses,tags,sensitivity,scoring_mode,presentation,sections,fields,status,is_current,effective_from,version,created_by,submitted_by,approved_by,starter_catalog_code,starter_catalog_version)
 SELECT id,tenant_id,legal_entity_id,code,name,purpose,responsible_team,approved_uses,tags,sensitivity,scoring_mode,presentation,sections,fields,'ACTIVE',true,clock_timestamp(),3,$2::uuid,$2::uuid,$3::uuid,starter_catalog_code,starter_catalog_version FROM monitoring_form_templates WHERE id=$1::uuid AND version=1`, formID, owner, checker)
	exec(`UPDATE framework_starter_requirement_links SET form_template_version=3 WHERE tenant_id=$1::uuid AND template_code='ISO_27001_2022'`, tenant)
	source, err = catalog.StarterRequirements(ctx, tenant, entity, "ISO_27001_2022")
	if err != nil {
		t.Fatal(err)
	}
	for i := range source {
		source[i].Decision = "KEEP"
	}
	sys := continuity.WithTrustedSystemEntityScope(ctx, tenant, entity)
	programs := continuity.NewService(continuity.NewCurrentPostgresRepository(pool))
	programs.ConfigureFrameworkEditionLookup(catalog)
	program, err := programs.CreateProgram(sys, continuity.CreateProgramInput{TenantID: tenant, LegalEntityID: entity, Code: "ISO-ASSESSMENT", Name: "Digital Banking Information Security fixture", Type: "COMPLIANCE", OwningFunction: "Security", OwnerPrincipalID: owner, AuthorityPrincipalID: checker, EffectiveFrom: time.Now().UTC(), ActorID: owner})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := programs.ProposeFrameworkAdoption(sys, continuity.ProposeFrameworkAdoptionInput{TenantID: tenant, ProgramID: program.Program.ID, ExpectedVersion: program.Program.Version, ActorID: owner, Rationale: "Isolated test of the adopted ISO questionnaire", TemplateCode: "ISO_27001_2022", Configuration: &continuity.StarterConfiguration{Requirements: source, OwnerID: owner, RespondentScopeIDs: []string{unit}}})
	if err != nil {
		t.Fatal(err)
	}
	approved, err := programs.ApproveFrameworkAdoption(sys, continuity.DecideFrameworkAdoptionInput{TenantID: tenant, ProgramID: program.Program.ID, ExpectedVersion: proposal.Program.Version, ActorID: checker, AdoptionID: proposal.FrameworkAdoptions[0].ID, Rationale: "Independent isolated fixture decision"})
	if err != nil {
		t.Fatal(err)
	}
	evidenceRepo := evidence.NewPostgresRepository(pool)
	capture := evidence.NewService(evidenceRepo, evidence.NewMemoryObjectStore())
	service := monitoring.NewService(monitoring.NewPostgresRepository(pool), capture)
	actor := monitoring.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: owner}
	ownerContext := identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: owner})
	respondentContext := identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: respondent})
	checkerContext := identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: checker})
	options, err := service.AdoptedAssessmentOptions(ownerContext, actor, program.Program.ID)
	if err != nil || len(options) != 1 || options[0].Ready {
		t.Fatalf("draft Program must block: %#v %v", options, err)
	}
	// Program transition is independently covered by its command tests; fixture
	// activation lets this test focus on the collection transaction and evidence.
	exec(`UPDATE programs SET status='ACTIVE',version=version+1 WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenant, program.Program.ID)
	options, err = service.AdoptedAssessmentOptions(ownerContext, actor, program.Program.ID)
	if err != nil || len(options) != 1 || !options[0].Ready || len(options[0].Respondents) != 1 || options[0].Respondents[0].ID != respondent {
		t.Fatalf("setup %#v: %v", options, err)
	}
	option := options[0]
	input := monitoring.StartCollectionInput{ProgramID: program.Program.ID, LegalEntityID: entity, AdoptionID: option.AdoptionID, AssessmentKey: uuid(), ExpectedProgramVersion: option.ProgramVersion, FormTemplateID: option.FormID, FormTemplateVersion: option.FormVersion, RespondentPrincipalID: respondent, PeriodStart: time.Now().UTC().Add(-30 * 24 * time.Hour), PeriodEnd: time.Now().UTC(), Deadline: time.Now().UTC().Add(7 * 24 * time.Hour)}
	bad := input
	bad.RespondentPrincipalID = unrelated
	if _, err = service.StartCollection(ownerContext, actor, bad); err == nil {
		t.Fatal("wrong-unit respondent accepted")
	}
	bad = input
	bad.ExpectedProgramVersion--
	if _, err = service.StartCollection(ownerContext, actor, bad); err == nil {
		t.Fatal("stale Program accepted")
	}
	bad = input
	bad.FormTemplateVersion = 1
	if _, err = service.StartCollection(ownerContext, actor, bad); err == nil {
		t.Fatal("unapproved form accepted")
	}
	foreignTenant, foreignEntity, foreignOwner := uuid(), uuid(), uuid()
	exec("INSERT INTO tenants(id,slug,name) VALUES($1,$2,'Other bank')", foreignTenant, "other-bank-"+foreignTenant)
	exec("INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1,$2,'BANK','Other bank','NG')", foreignEntity, foreignTenant)
	exec("INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1,$2,'PERSON','Other bank owner')", foreignOwner, foreignTenant)
	foreign := monitoring.Actor{TenantID: foreignTenant, LegalEntityID: foreignEntity, PrincipalID: foreignOwner}
	foreignContext := identity.WithActor(ctx, identity.Actor{TenantID: foreignTenant, LegalEntityID: foreignEntity, PrincipalID: foreignOwner})
	if _, err = service.StartCollection(foreignContext, foreign, input); !errors.Is(err, monitoring.ErrNotFound) {
		t.Fatalf("cross-tenant creation: %v", err)
	}
	request, err := service.StartCollection(ownerContext, actor, input)
	if err != nil {
		t.Fatal("create adopted assessment", err)
	}
	repeat, err := service.StartCollection(ownerContext, actor, input)
	if err != nil || repeat.ID != request.ID {
		t.Fatalf("retry must reuse request: %v %s != %s", err, repeat.ID, request.ID)
	}
	if request.FormTemplateVersion != 3 || len(request.Fields) != 15 {
		t.Fatalf("pinned snapshot %#v", request)
	}
	draft, err := capture.GetInternalResponseDraft(respondentContext, request.ID)
	if err != nil || draft.Version != 0 {
		t.Fatal("empty draft", err)
	}
	text := func(value string) *string { return &value }
	answers := map[string]formcontract.AnswerValue{}
	for _, field := range request.Fields {
		switch field.Type {
		case "date":
			answers[field.ID] = formcontract.AnswerValue{Text: text("2026-11-12")}
		case "long_text":
			answers[field.ID] = formcontract.AnswerValue{Text: text("DEMO: Service owners reviewed mobile and internet banking administrators. Two unnecessary grants were removed; one supplier exception needs follow-up.")}
		case "file":
			artifact, e := capture.StoreArtifact(respondentContext, evidence.ArtifactInput{TenantID: tenant, RequestID: request.ID, FieldID: field.ID, FileName: "DEMO-signed-review-record.pdf", MediaType: "application/pdf", CreatedBy: respondent}, strings.NewReader("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n%%EOF"))
			if e != nil {
				t.Fatal("upload", e)
			}
			// Isolated scanner fixture. The production scan gate remains enabled.
			exec(`UPDATE capture_artifacts SET status='AVAILABLE' WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenant, artifact.ID)
			answers[field.ID] = formcontract.AnswerValue{ArtifactIDs: []string{artifact.ID}}
		default:
			t.Fatalf("unexpected starter field %s", field.Type)
		}
	}
	draft, err = capture.SaveInternalResponseDraft(respondentContext, request.ID, evidence.SaveDraftInput{Answers: answers, ExpectedVersion: 0})
	if err != nil || draft.Version != 1 {
		t.Fatal("save draft", err)
	}
	if _, err = capture.SaveInternalResponseDraft(respondentContext, request.ID, evidence.SaveDraftInput{Answers: answers, ExpectedVersion: 0}); !errors.Is(err, evidence.ErrVersionConflict) {
		t.Fatalf("stale draft: %v", err)
	}
	restored, err := capture.GetInternalResponseDraft(respondentContext, request.ID)
	if err != nil || len(restored.Answers) != 15 {
		t.Fatal("reopen draft", err)
	}
	if _, err = capture.GetInternalResponseDraft(ownerContext, request.ID); !errors.Is(err, evidence.ErrRecipientMismatch) {
		t.Fatalf("owner read another person's draft: %v", err)
	}
	receipt, err := capture.Submit(respondentContext, evidence.Submission{TenantID: tenant, LegalEntityID: entity, RequestID: request.ID, ExpectedVersion: request.Version, SubmittedBy: respondent, Channel: "INTERNAL", Answers: answers})
	if err != nil {
		t.Fatal("submit", err)
	}
	if _, err = capture.SaveInternalResponseDraft(respondentContext, request.ID, evidence.SaveDraftInput{Answers: answers, ExpectedVersion: 1}); !errors.Is(err, evidence.ErrRequestClosed) {
		t.Fatalf("draft after submission: %v", err)
	}
	workspace, err := capture.ProgramAssessmentWorkspace(checkerContext, program.Program.ID)
	if err != nil || workspace.AssessmentCount != 1 || workspace.ResponsesToReview != 1 || workspace.DocumentsToReview != 5 {
		t.Fatalf("submitted summary %#v: %v", workspace, err)
	}
	decisions := []evidence.GroupReviewDecision{}
	for _, section := range request.Sections {
		outcome := "REVIEWED"
		if section.ID == "privileged_access" {
			outcome = "FOLLOW_UP_REQUIRED"
		}
		decisions = append(decisions, evidence.GroupReviewDecision{GroupID: section.ID, Outcome: outcome, Rationale: "Isolated review: original records inspected; supplier exception needs an owner.", ArtifactIDs: []string{uuid()}})
	}
	reviewInput := evidence.RecordSubmissionReviewInput{ExpectedVersion: 0, Rationale: "DEMO independent review: resolve the supplier administrator exception.", Groups: decisions}
	if _, err = capture.ReviewInternalSubmission(respondentContext, program.Program.ID, request.ID, reviewInput); !errors.Is(err, evidence.ErrRecipientMismatch) {
		t.Fatalf("self review allowed: %v", err)
	}
	review, err := capture.ReviewInternalSubmission(checkerContext, program.Program.ID, request.ID, reviewInput)
	if err != nil || review.Status != "FOLLOW_UP_REQUIRED" || review.SubmissionID != receipt.SubmissionID {
		t.Fatalf("independent review %#v: %v", review, err)
	}
	for _, group := range review.Groups {
		if len(group.ArtifactIDs) != 1 || group.ArtifactIDs[0] == reviewInput.Groups[0].ArtifactIDs[0] {
			t.Fatal("review must derive actual submitted artifact IDs")
		}
	}
	savedReview, err := capture.ProgramSubmissionReview(ownerContext, program.Program.ID, review.ID)
	if err != nil || savedReview.ID != review.ID {
		t.Fatalf("saved review lookup: %v", err)
	}
	if _, err = capture.ProgramSubmissionReview(foreignContext, program.Program.ID, review.ID); !errors.Is(err, evidence.ErrNotFound) {
		t.Fatalf("cross-tenant review lookup: %v", err)
	}
	if _, err = capture.ReviewInternalSubmission(checkerContext, program.Program.ID, request.ID, reviewInput); !errors.Is(err, evidence.ErrVersionConflict) {
		t.Fatalf("duplicate review version: %v", err)
	}
	workspace, err = capture.ProgramAssessmentWorkspace(checkerContext, program.Program.ID)
	if err != nil || workspace.ResponsesToReview != 0 || workspace.DocumentsToReview != 1 || len(workspace.Activities) != 3 {
		encoded, _ := json.Marshal(workspace)
		t.Fatalf("reviewed workspace %s: %v", encoded, err)
	}
	reviewed, followUp := 0, 0
	for _, decision := range workspace.RequirementReviews {
		if decision.Outcome == "REVIEWED" {
			reviewed++
		}
		if decision.Outcome == "FOLLOW_UP_REQUIRED" {
			followUp++
		}
	}
	if reviewed != 4 || followUp != 1 {
		t.Fatalf("coverage: %v", workspace.RequirementReviews)
	}
	if workspace.Items[0].Recipient.PrincipalID != respondent {
		t.Fatal("recipient metadata lost in Program workspace")
	}
	unrelatedContext := identity.WithActor(ctx, identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: unrelated})
	hidden, err := capture.ProgramAssessmentWorkspace(unrelatedContext, program.Program.ID)
	if err != nil || hidden.AssessmentCount != 0 || len(hidden.Items) != 0 {
		t.Fatal("unrelated person can see assessment", err)
	}
	exec(`UPDATE programs SET authority_principal_id=$3::uuid WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenant, program.Program.ID, unrelated)
	reviewInput.ExpectedVersion = 1
	if _, err = capture.ReviewInternalSubmission(checkerContext, program.Program.ID, request.ID, reviewInput); !errors.Is(err, evidence.ErrRecipientMismatch) {
		t.Fatalf("revoked reviewer accepted: %v", err)
	}
	var outbox int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type IN ('ProgramAssessmentCreated','InternalResponseDraftSaved','ProgramAssessmentReviewed')`, tenant, request.ID).Scan(&outbox); err != nil || outbox != 3 {
		t.Fatalf("transactional events %d %v", outbox, err)
	}
	t.Logf("isolated tenant %s; assessment %s; submission %s; review %s; pinned form v3; five files; one follow-up", tenant, request.ID, receipt.SubmissionID, review.ID)
	_ = approved
}
