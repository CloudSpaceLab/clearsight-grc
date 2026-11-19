//go:build postgres && postgresintegration

package monitoring_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/frameworkcatalog"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
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
	exec(`UPDATE monitoring_form_templates SET fields=(SELECT jsonb_agg(CASE WHEN ordinal=1 THEN field || '{"answer_validity_days":7,"validation_binding":{"binding_id":"quality-test","binding_version":1,"value_field":"answer"}}'::jsonb WHEN ordinal=2 THEN field || '{"answer_validity_days":30}'::jsonb ELSE field END ORDER BY ordinal) FROM jsonb_array_elements(fields) WITH ORDINALITY AS values(field,ordinal)) WHERE id=$1::uuid AND version=3`, formID)
	exec(`INSERT INTO evidence_sources(id,tenant_id,legal_entity_id,code,name,source_type,authority_class,expected_freshness_minutes,health,status,version) VALUES($1::uuid,$2::uuid,$3::uuid,'QUALITY-TEST','Isolated question quality test','SYSTEM','SYSTEM_OF_RECORD',60,'UNKNOWN','ACTIVE',1)`, unrelated, tenant, entity)
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
	capture.ConfigureSourceBindings(questionQualityReader{tenant: tenant, source: unrelated})
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
	t.Run("recurring approved dispatch", func(t *testing.T) {
		repo := monitoring.NewPostgresRepository(pool)
		maintainer := monitoring.AssessmentScheduleMaintainer{
			Repository: repo, Service: service, Requests: capture,
			Authority: scheduleTestAuthority{owner: owner, checker: checker},
		}
		location, err := time.LoadLocation("Africa/Lagos")
		if err != nil {
			t.Fatal(err)
		}
		today := time.Now().In(location)
		first := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, location)
		input := monitoring.AssessmentScheduleInput{
			AdoptionID: option.AdoptionID, FormTemplateID: option.FormID, FormTemplateVersion: option.FormVersion,
			RespondentPrincipalID: respondent, FirstPeriodStart: first.Format(time.DateOnly), TimeZone: location.String(),
			IntervalMonths: 3, ResponseDays: 15, CycleLimit: 2, Rationale: "Test quarterly evidence collection",
		}
		schedule, err := service.ProposeAssessmentSchedule(ownerContext, actor, program.Program.ID, input)
		if err != nil || schedule.State != "PENDING" {
			t.Fatalf("proposal: %+v %v", schedule, err)
		}
		if _, err = service.ChangeAssessmentSchedule(ownerContext, actor, program.Program.ID, schedule.ID, "ACTIVE", "self approval", schedule.Version); !errors.Is(err, monitoring.ErrConflict) {
			t.Fatalf("self approval: %v", err)
		}
		before := schedule.NextAttemptAt.Add(-time.Second)
		if n, err := maintainer.Maintain(ctx, before, 1); err != nil || n != 0 {
			t.Fatalf("early dispatch: %d %v", n, err)
		}
		reviewer := monitoring.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: checker}
		if _, err = service.ChangeAssessmentSchedule(checkerContext, reviewer, program.Program.ID, schedule.ID, "ACTIVE", "stale decision", schedule.Version+1); !errors.Is(err, monitoring.ErrConflict) {
			t.Fatalf("stale decision: %v", err)
		}
		schedule, err = service.ChangeAssessmentSchedule(checkerContext, reviewer, program.Program.ID, schedule.ID, "ACTIVE", "Independent quarterly approval", schedule.Version)
		if err != nil || schedule.State != "ACTIVE" {
			t.Fatalf("approval: %+v %v", schedule, err)
		}
		opens := schedule.NextAttemptAt
		if n, err := maintainer.Maintain(ctx, opens, 1); err != nil || n != 1 {
			t.Fatalf("dispatch: %d %v", n, err)
		}
		items, err := service.ListAssessmentSchedules(ownerContext, actor, program.Program.ID)
		if err != nil || len(items) != 1 || items[0].CycleIndex != 1 || items[0].LastRequestID == "" {
			t.Fatalf("dispatch result: %+v %v", items, err)
		}
		requestID, nextOpen := items[0].LastRequestID, items[0].NextAttemptAt
		// Simulate a committed request followed by a worker interruption before schedule progress was saved.
		exec("UPDATE program_assessment_schedules SET cycle_index=0,next_attempt_at=$2 WHERE id=$1::uuid", schedule.ID, opens)
		if n, err := maintainer.Maintain(ctx, opens, 1); n != 1 || err != nil {
			t.Fatalf("replay: %d %v", n, err)
		}
		items, err = service.ListAssessmentSchedules(ownerContext, actor, program.Program.ID)
		if err != nil || len(items) != 1 || items[0].LastRequestID != requestID || items[0].CycleIndex != 1 {
			t.Fatalf("replay duplicated the assignment: %+v %v", items, err)
		}
		var count int
		if err = pool.QueryRow(ctx, `SELECT count(*) FROM capture_requests WHERE tenant_id=$1::uuid AND subject_id=$2 AND origin_type='FRAMEWORK_ASSESSMENT'`, tenant, program.Program.ID).Scan(&count); err != nil || count != 1 {
			t.Fatalf("assignment count %d: %v", count, err)
		}
		exec("UPDATE org_positions SET occupant_principal_id=NULL WHERE tenant_id=$1::uuid AND occupant_principal_id=$2::uuid", tenant, respondent)
		if n, err := maintainer.Maintain(ctx, nextOpen, 1); n != 1 || err != nil {
			t.Fatalf("blocked cycle: %d %v", n, err)
		}
		items, err = service.ListAssessmentSchedules(ownerContext, actor, program.Program.ID)
		if err != nil || len(items) != 1 || items[0].State != "BLOCKED" || items[0].CycleIndex != 1 {
			t.Fatalf("lost eligibility did not block: %+v %v", items, err)
		}
		stopped, err := service.ChangeAssessmentSchedule(ownerContext, actor, program.Program.ID, schedule.ID, "STOPPED", "Replace assignment after staff move", items[0].Version)
		if err != nil || stopped.State != "STOPPED" {
			t.Fatalf("stop: %+v %v", stopped, err)
		}
		exec("UPDATE org_positions SET occupant_principal_id=$2::uuid WHERE tenant_id=$1::uuid AND code='DB-OPERATIONS'", tenant, respondent)
	})
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
	savedQuality, err := capture.GetSubmissionForRequest(respondentContext, tenant, request.ID)
	if err != nil {
		t.Fatal(err)
	}
	first := savedQuality.AnswerProvenance[request.Fields[0].ID]
	second := savedQuality.AnswerProvenance[request.Fields[1].ID]
	if first.ExpiresAt == nil || second.ExpiresAt == nil || !first.ExpiresAt.Equal(savedQuality.SubmittedAt.AddDate(0, 0, 7)) || !second.ExpiresAt.Equal(savedQuality.SubmittedAt.AddDate(0, 0, 30)) || len(first.Validations) != 1 || first.Validations[0].State != evidence.SourceResolutionCurrent || first.Validations[0].Receipt == nil {
		t.Fatalf("persisted question quality missing: %+v %+v", first, second)
	}
	if _, err = capture.SaveInternalResponseDraft(respondentContext, request.ID, evidence.SaveDraftInput{Answers: answers, ExpectedVersion: 1}); !errors.Is(err, evidence.ErrRequestClosed) {
		t.Fatalf("draft after submission: %v", err)
	}
	workspace, err := capture.ProgramAssessmentWorkspace(checkerContext, program.Program.ID)
	if err != nil || workspace.AssessmentCount != 2 || workspace.ResponsesToReview != 1 || workspace.DocumentsToReview != 5 {
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
	if err != nil || workspace.ResponsesToReview != 0 || workspace.DocumentsToReview != 1 || len(workspace.Activities) != 4 {
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
	query := evidence.AssessmentHistoryQuery{ProgramID: program.Program.ID, RequestID: request.ID}
	history, err := capture.ProgramAssessmentHistory(respondentContext, query)
	if err != nil || len(history.Revisions) != 1 || history.Correction == nil || history.Correction.ReviewID != review.ID || history.CanReview {
		t.Fatalf("respondent correction window: %#v, %v", history, err)
	}
	proof := history.Correction.AssessmentCorrection
	if _, err = capture.ProgramAssessmentHistory(unrelatedContext, query); !errors.Is(err, evidence.ErrNotFound) {
		t.Fatalf("unrelated response history: %v", err)
	}
	if _, err = capture.ProgramAssessmentHistory(foreignContext, query); !errors.Is(err, evidence.ErrNotFound) {
		t.Fatalf("cross-entity response history: %v", err)
	}
	correction, err := capture.GetInternalAssessmentDraft(respondentContext, request.ID, &proof)
	if err != nil || correction.Version != 0 || len(correction.Answers) != len(answers) {
		t.Fatalf("correction seeded from original: %#v, %v", correction, err)
	}
	if _, err = capture.GetInternalAssessmentDraft(checkerContext, request.ID, &proof); !errors.Is(err, evidence.ErrRecipientMismatch) {
		t.Fatalf("reviewer read correction draft: %v", err)
	}
	staleProof := proof
	staleProof.ReviewID = uuid()
	if _, err = capture.GetInternalAssessmentDraft(respondentContext, request.ID, &staleProof); !errors.Is(err, evidence.ErrRequestClosed) {
		t.Fatalf("forged correction read: %v", err)
	}
	var textID, firstFile, secondFile string
	for _, field := range request.Fields {
		if field.Type == "long_text" {
			textID = field.ID
		}
		if field.Type == "file" {
			secondFile, firstFile = firstFile, field.ID
		}
	}
	if textID == "" || firstFile == "" || secondFile == "" {
		t.Fatal("expected two attachments and a narrative")
	}
	correction.Answers[textID] = formcontract.AnswerValue{Text: text("DEMO: The supplier exception was removed and the accountable owner signed the corrected record.")}
	replacement, err := capture.StoreArtifact(respondentContext, evidence.ArtifactInput{TenantID: tenant, RequestID: request.ID, FieldID: firstFile, FileName: "DEMO-corrected-review-record.pdf", MediaType: "application/pdf", CreatedBy: respondent, Correction: &proof}, strings.NewReader("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n%%EOF"))
	if err != nil {
		t.Fatal("correction attachment", err)
	}
	exec(`UPDATE capture_artifacts SET status='AVAILABLE' WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenant, replacement.ID)
	correction.Answers[firstFile] = formcontract.AnswerValue{ArtifactIDs: []string{replacement.ID}}
	correction, err = capture.SaveInternalResponseDraft(respondentContext, request.ID, evidence.SaveDraftInput{Answers: correction.Answers, Correction: &proof})
	if err != nil || correction.Version <= draft.Version {
		t.Fatalf("correction draft save %#v %v", correction, err)
	}
	if _, err = capture.SaveInternalResponseDraft(respondentContext, request.ID, evidence.SaveDraftInput{Answers: answers, Correction: &proof}); !errors.Is(err, evidence.ErrVersionConflict) {
		t.Fatalf("stale correction draft overwritten: %v", err)
	}
	correction, err = capture.GetInternalAssessmentDraft(respondentContext, request.ID, &proof)
	if err != nil || correction.Answers[firstFile].ArtifactIDs[0] != replacement.ID ||
		len(correction.Attachments) != 1 || correction.Attachments[0].FileName != replacement.FileName {
		t.Fatalf("correction draft reload %#v %v", correction, err)
	}
	command := evidence.Submission{TenantID: tenant, LegalEntityID: entity, RequestID: request.ID, ExpectedVersion: receipt.Version, SubmittedBy: respondent, Channel: "INTERNAL", Answers: correction.Answers, Correction: &proof}
	priorSecond := correction.Answers[secondFile]
	correction.Answers[secondFile] = answers[firstFile]
	if _, err = capture.Submit(respondentContext, command); !errors.Is(err, evidence.ErrNotFound) {
		t.Fatalf("original file reused on a different question: %v", err)
	}
	correction.Answers[secondFile] = priorSecond
	revised, err := capture.Submit(respondentContext, command)
	if err != nil || revised.Revision != 2 || revised.SubmissionID == receipt.SubmissionID {
		t.Fatalf("resubmit %#v %v", revised, err)
	}
	if _, err = capture.Submit(respondentContext, command); !errors.Is(err, evidence.ErrRequestClosed) {
		t.Fatalf("replayed resubmission: %v", err)
	}
	workspace, err = capture.ProgramAssessmentWorkspace(checkerContext, program.Program.ID)
	if err != nil || workspace.ResponsesToReview != 1 || workspace.DocumentsToReview != 5 || workspace.Items[0].LatestAssessmentSubmissionID != revised.SubmissionID {
		t.Fatalf("resubmitted queue %#v %v", workspace, err)
	}
	reviewInput.ExpectedVersion = 1
	reviewInput.SubmissionID = receipt.SubmissionID
	if _, err = capture.ReviewInternalSubmission(checkerContext, program.Program.ID, request.ID, reviewInput); !errors.Is(err, evidence.ErrVersionConflict) {
		t.Fatalf("review of superseded response: %v", err)
	}
	query.SubmissionID = receipt.SubmissionID
	original, err := capture.ProgramAssessmentHistory(respondentContext, query)
	if err != nil || original.Submission.Answers[firstFile].ArtifactIDs[0] != answers[firstFile].ArtifactIDs[0] || len(original.Reviews) != 1 || original.Reviews[0].ID != review.ID {
		t.Fatalf("original response and review changed: %#v %v", original, err)
	}
	ownerHistory, err := capture.ProgramAssessmentHistory(ownerContext, query)
	if err != nil || len(ownerHistory.Submission.Answers) != 0 || len(ownerHistory.Reviews) != 1 {
		t.Fatalf("owner received restricted answer data: %#v %v", ownerHistory, err)
	}
	query.SubmissionID = revised.SubmissionID
	revision, err := capture.ProgramAssessmentHistory(checkerContext, query)
	if err != nil || !revision.CanReview || revision.Submission.CorrectionOf != receipt.SubmissionID || revision.Submission.Answers[firstFile].ArtifactIDs[0] != replacement.ID ||
		len(revision.Reviews) != 0 || revision.Correction != nil || !revision.Submission.AnswerProvenance[request.Fields[0].ID].ExpiresAt.Equal(*first.ExpiresAt) {
		t.Fatalf("revised response: %#v %v", revision, err)
	}
	reviewInput.SubmissionID = revised.SubmissionID
	for i := range decisions {
		decisions[i].Outcome = "REVIEWED"
	}
	for _, field := range request.Fields {
		if field.ID == firstFile {
			for i, decision := range decisions {
				if decision.GroupID == field.SectionID {
					decisions[i].Questions = []evidence.QuestionReviewDecision{{FieldID: firstFile, Outcome: "REVIEWED", Rationale: "Corrected signed record verified.", ArtifactIDs: []string{uuid()}}}
				}
			}
		}
	}
	reviewInput.Groups = decisions
	finalReview, err := capture.ReviewInternalSubmission(checkerContext, program.Program.ID, request.ID, reviewInput)
	if err != nil || finalReview.SubmissionID != revised.SubmissionID || finalReview.Status != "REVIEWED" {
		t.Fatalf("independent review of revision 2: %#v %v", finalReview, err)
	}
	for _, group := range finalReview.Groups {
		for _, question := range group.Questions {
			if len(question.ArtifactIDs) != 1 || question.ArtifactIDs[0] != replacement.ID {
				t.Fatalf("question review did not use exact revised evidence: %#v", question)
			}
		}
	}
	workspace, err = capture.ProgramAssessmentWorkspace(checkerContext, program.Program.ID)
	if err != nil || workspace.ResponsesToReview != 0 || workspace.DocumentsToReview != 0 {
		t.Fatalf("reviewed revision left items in queue: %#v %v", workspace, err)
	}
	exec(`UPDATE programs SET authority_principal_id=$3::uuid WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenant, program.Program.ID, unrelated)
	reviewInput.ExpectedVersion = 2
	if _, err = capture.ReviewInternalSubmission(checkerContext, program.Program.ID, request.ID, reviewInput); !errors.Is(err, evidence.ErrRecipientMismatch) {
		t.Fatalf("revoked reviewer accepted: %v", err)
	}
	var outbox int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_id=$2::uuid AND event_type IN ('ProgramAssessmentCreated','InternalResponseDraftSaved','ProgramAssessmentReviewed')`, tenant, request.ID).Scan(&outbox); err != nil || outbox != 5 {
		t.Fatalf("transactional events %d %v", outbox, err)
	}
	usage, err := programs.FrameworkProgramUsage(ownerContext)
	if err != nil || len(usage) != 1 || usage[0].TemplateCode != "ISO_27001_2022" || usage[0].ActiveProgramCount != 1 {
		t.Fatalf("scoped framework usage: %+v %v", usage, err)
	}
	t.Logf("isolated tenant %s; assessment %s; submission %s; review %s; pinned form v3; five files; one follow-up", tenant, request.ID, receipt.SubmissionID, review.ID)
	_ = approved
}

type scheduleTestAuthority struct {
	authority.Service
	owner   string
	checker string
}

func (r scheduleTestAuthority) Resolve(_ context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	principal := r.owner
	if input.Responsibility == authority.ResponsibilityAuthorizer {
		principal = r.checker
	}
	return authority.Resolution{Principal: authority.Principal{ID: principal}}, nil
}

type questionQualityReader struct{ tenant, source string }

func (r questionQualityReader) Binding(context.Context, string, string, int64) (sourceaccess.BindingRevision, error) {
	return sourceaccess.BindingRevision{TenantID: r.tenant, SourceID: r.source, BindingID: "quality-test", Name: "Isolated source fixture", Operations: []sourceaccess.Operation{sourceaccess.OperationLookup}, SelectedFields: []string{"answer"}, RequiredFreshnessMinutes: 60, RevisionLifecycle: sourceaccess.RevisionLifecycle{Status: sourceaccess.RevisionActive, Version: 1}}, nil
}
func (r questionQualityReader) PreviewBinding(context.Context, string, string, int64, sourceaccess.PageRequest) (sourceaccess.RecordPage, error) {
	return sourceaccess.RecordPage{}, sourceaccess.ErrCapabilityUnavailable
}
func (r questionQualityReader) LookupBinding(_ context.Context, _ string, _ string, _ int64, q sourceaccess.LookupRequest) (sourceaccess.LookupResult, error) {
	return sourceaccess.LookupResult{Records: []sourceaccess.Record{{"answer": q.Values[0]}}, Receipt: sourceaccess.OperationReceipt{SourceID: r.source, BindingID: "quality-test", BindingVersion: "1", Operation: sourceaccess.OperationLookup, ObservedAt: time.Now().UTC(), Completeness: sourceaccess.CompletenessComplete}}, nil
}
