package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

type exactFormAuthority struct {
	delegateFormID      string
	ownerDelegateFormID string
	collisionFormID     string
	inputs              []authority.ResolveInput
	batchCalls          int
	scalarCalls         int
}

func (s *exactFormAuthority) Resolve(_ context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	s.scalarCalls++
	s.inputs = append(s.inputs, input)
	return s.resolution(input), nil
}

func (s *exactFormAuthority) resolution(input authority.ResolveInput) authority.Resolution {
	principalID := "broad-program-owner"
	candidates := []authority.Principal{{ID: principalID, DisplayName: principalID}}
	origins := []authority.EffectiveOrigin{{PrincipalID: principalID, OriginPrincipalID: principalID}}
	if input.ObjectType == "FORM_TEMPLATE" {
		switch input.Responsibility {
		case authority.ResponsibilityOwner:
			principalID = "owner-" + input.ObjectID
			candidates = []authority.Principal{{ID: principalID, DisplayName: principalID}, {ID: "program-owner", DisplayName: "Program owner"}}
			origins = []authority.EffectiveOrigin{{PrincipalID: principalID, OriginPrincipalID: principalID}, {PrincipalID: "program-owner", OriginPrincipalID: "program-owner"}}
			if input.ObjectID == s.ownerDelegateFormID {
				candidates = append(candidates, authority.Principal{ID: "owner-delegate", DisplayName: "Acting Program owner"})
				origins = append(origins, authority.EffectiveOrigin{PrincipalID: "owner-delegate", OriginPrincipalID: "program-owner"})
			}
		case authority.ResponsibilityPerformer:
			principalID = "respondent-" + input.ObjectID
			candidates = []authority.Principal{{ID: principalID, DisplayName: principalID}}
			origins = []authority.EffectiveOrigin{{PrincipalID: principalID, OriginPrincipalID: principalID}}
		case authority.ResponsibilityReviewer:
			principalID = "reviewer-" + input.ObjectID
			if input.ObjectID == s.collisionFormID {
				principalID = "respondent-" + input.ObjectID
			}
			candidates = []authority.Principal{{ID: principalID, DisplayName: principalID}}
			origins = []authority.EffectiveOrigin{{PrincipalID: principalID, OriginPrincipalID: principalID}}
			if input.ObjectID == s.delegateFormID {
				candidates = append(candidates, authority.Principal{ID: "reviewer-delegate", DisplayName: "Acting form reviewer"})
				origins = append(origins, authority.EffectiveOrigin{PrincipalID: "reviewer-delegate", OriginPrincipalID: principalID})
			}
		}
	} else {
		switch input.Responsibility {
		case authority.ResponsibilityPerformer:
			principalID = "broad-program-respondent"
		case authority.ResponsibilityReviewer:
			principalID = "program-reviewer"
		}
		candidates = []authority.Principal{{ID: principalID, DisplayName: principalID}}
		origins = []authority.EffectiveOrigin{{PrincipalID: principalID, OriginPrincipalID: principalID}}
	}
	return authority.Resolution{
		Principal:           authority.Principal{ID: principalID, DisplayName: principalID},
		CandidatePrincipals: candidates,
		EffectiveOrigins:    origins,
	}
}

func (s *exactFormAuthority) ResolveMany(_ context.Context, inputs []authority.ResolveInput) ([]authority.ResolveOutcome, error) {
	s.batchCalls++
	outcomes := make([]authority.ResolveOutcome, len(inputs))
	for index, input := range inputs {
		s.inputs = append(s.inputs, input)
		outcomes[index] = authority.ResolveOutcome{Resolution: s.resolution(input)}
	}
	return outcomes, nil
}

func (*exactFormAuthority) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}

func (*exactFormAuthority) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}

func (*exactFormAuthority) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

func pendingMonitoringForm(t *testing.T, repository *monitoring.MemoryRepository, programID, id, submittedBy string, now time.Time) monitoring.FormTemplate {
	t.Helper()
	form, err := repository.CreateFormRevision(t.Context(), monitoring.FormTemplate{
		ID: id, TenantID: "bank", LegalEntityID: "entity-a", ProgramID: programID,
		Code: strings.ToUpper(id), Name: "Review " + id, Purpose: "Confirm that the control operated.",
		Fields: []monitoring.TemplateField{{ID: "operated", Label: "Did the control operate?", Type: "single_select", Required: true, Options: []string{"Yes", "No"}}},
		Lifecycle: monitoring.Lifecycle{
			Status: monitoring.LifecyclePendingApproval, IsCurrent: true, Version: 2,
			CreatedBy: submittedBy, SubmittedBy: submittedBy, CreatedAt: now, UpdatedAt: now,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return form
}

func monitoringFormAuthorityFixture(t *testing.T) (*continuity.Service, continuity.ProgramAggregate, *monitoring.MemoryRepository, *monitoring.Service, monitoring.FormTemplate, monitoring.FormTemplate, *exactFormAuthority) {
	t.Helper()
	now := time.Now().UTC()
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	program, err := continuityService.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "PRIVACY", Name: "Privacy", Type: "PRIVACY", OwningFunction: "Privacy",
		OwnerPrincipalID: "program-owner", AuthorityPrincipalID: "program-authorizer", EffectiveFrom: now, ActorID: "program-owner", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	repository := monitoring.NewMemoryRepository()
	formA := pendingMonitoringForm(t, repository, program.Program.ID, "form-a", "maker-a", now)
	formB := pendingMonitoringForm(t, repository, program.Program.ID, "form-b", "maker-b", now)
	resolver := &exactFormAuthority{delegateFormID: formA.ID}
	return continuityService, program, repository, monitoring.NewService(repository, nil), formA, formB, resolver
}

func TestMonitoringFormTransitionUsesExactFormAuthorityAndVerifiedActor(t *testing.T) {
	continuityService, program, repository, monitoringService, formA, formB, resolver := monitoringFormAuthorityFixture(t)
	guard, err := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	handler := func(principalID string) http.Handler {
		return New(Dependencies{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", principalID, "entity-a"),
			Continuity: continuityService, Monitoring: monitoringService, Authority: resolver, CommandGuard: guard,
		})
	}
	transition := func(principalID string, form monitoring.FormTemplate, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		path := fmt.Sprintf("/api/v1/programs/%s/form-templates/%s/transition", program.Program.ID, form.ID)
		handler(principalID).ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body)))
		return response
	}

	delegateBody := `{"tenant_id":"bank","actor_id":"program-reviewer","expected_version":2,"to":"ACTIVE"}`
	if response := transition("reviewer-delegate", formA, delegateBody); response.Code != http.StatusOK {
		t.Fatalf("exact form reviewer delegate returned %d: %s", response.Code, response.Body.String())
	}
	if active, err := repository.FormRevision(t.Context(), "bank", "entity-a", program.Program.ID, formA.ID, 3); err != nil || active.ApprovedBy != "reviewer-delegate" || active.Status != monitoring.LifecycleActive {
		t.Fatalf("delegated form approval = %#v, err %v", active, err)
	}

	forgedTenantBody := `{"tenant_id":"another-bank","actor_id":"reviewer-form-b","expected_version":2,"to":"ACTIVE"}`
	if response := transition("program-reviewer", formB, forgedTenantBody); response.Code != http.StatusForbidden {
		t.Fatalf("forged tenant transition returned %d: %s", response.Code, response.Body.String())
	}
	broadReviewerBody := `{"tenant_id":"bank","actor_id":"reviewer-form-b","expected_version":2,"to":"ACTIVE"}`
	if response := transition("program-reviewer", formB, broadReviewerBody); response.Code != http.StatusForbidden {
		t.Fatalf("Program-wide reviewer substituted for exact form reviewer: %d %s", response.Code, response.Body.String())
	}
	if response := transition("reviewer-delegate", formB, broadReviewerBody); response.Code != http.StatusForbidden {
		t.Fatalf("delegate for form A acted on form B: %d %s", response.Code, response.Body.String())
	}
	if response := transition("reviewer-form-b", formB, broadReviewerBody); response.Code != http.StatusOK {
		t.Fatalf("exact form B reviewer returned %d: %s", response.Code, response.Body.String())
	}

	for _, input := range resolver.inputs {
		if input.DecisionType != "program.monitoring.form.transition" {
			continue
		}
		if input.ObjectType != "FORM_TEMPLATE" || input.ObjectID == "" || input.ObjectID == program.Program.ID || input.LegalEntityID != program.Program.LegalEntityID {
			t.Fatalf("form transition authority was not exact: %#v", input)
		}
	}
}

func TestProgramOperationsUseExactFormAuthority(t *testing.T) {
	_, program, _, monitoringService, formA, formB, resolver := monitoringFormAuthorityFixture(t)
	api := &API{deps: Dependencies{Authority: resolver, Monitoring: monitoringService}}
	find := func(operations []RecordOperation, formID string) RecordOperation {
		t.Helper()
		for _, operation := range operations {
			if operation.Command == "program.monitoring.form.transition" && operation.SubresourceID == formID {
				return operation
			}
		}
		t.Fatalf("form transition operation %s missing: %#v", formID, operations)
		return RecordOperation{}
	}
	load := func(principalID string) programOperationsResponse {
		actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: principalID, Kind: "PERSON"}
		return api.buildProgramOperations(identity.WithActor(t.Context(), actor), actor, program, time.Now().UTC())
	}

	delegate := load("reviewer-delegate")
	if operation := find(delegate.Operations, formA.ID); !operation.CanAct {
		t.Fatalf("exact form delegate operation = %#v", operation)
	}
	if operation := find(delegate.Operations, formB.ID); operation.CanAct {
		t.Fatalf("form A delegate received form B operation = %#v", operation)
	}
	broad := load("program-reviewer")
	for _, formID := range []string{formA.ID, formB.ID} {
		if operation := find(broad.Operations, formID); operation.CanAct {
			t.Fatalf("Program-wide reviewer received exact form operation %s: %#v", formID, operation)
		}
	}

	found := map[string]bool{}
	for _, input := range resolver.inputs {
		if input.DecisionType != "program.monitoring.form.transition" {
			continue
		}
		if input.ObjectType != "FORM_TEMPLATE" {
			t.Fatalf("form operation authority used %s/%s: %#v", input.ObjectType, input.ObjectID, input)
		}
		found[input.ObjectID] = true
	}
	for _, formID := range []string{formA.ID, formB.ID} {
		if !found[formID] {
			t.Fatalf("exact form authority input %s was not resolved: %#v", formID, resolver.inputs)
		}
	}
}

type monitoringCollectionRecorder struct {
	inputs []evidence.CreateRequestInput
}

func (r *monitoringCollectionRecorder) CreateRequest(_ context.Context, input evidence.CreateRequestInput) (evidence.Request, error) {
	r.inputs = append(r.inputs, input)
	return evidence.Request{
		ID: "request-" + input.FormTemplateID, TenantID: input.TenantID, LegalEntityID: input.LegalEntityID,
		SubjectType: input.SubjectType, SubjectID: input.SubjectID,
		FormTemplateID: input.FormTemplateID, FormTemplateVersion: input.FormTemplateVersion, CreatedBy: input.CreatedBy, Version: 1,
	}, nil
}

func activeMonitoringForm(t *testing.T, repository *monitoring.MemoryRepository, programID, id string, now time.Time) monitoring.FormTemplate {
	t.Helper()
	activeAt := now.Add(-time.Hour)
	form, err := repository.CreateFormRevision(t.Context(), monitoring.FormTemplate{
		ID: id, TenantID: "bank", LegalEntityID: "entity-a", ProgramID: programID,
		Code: strings.ToUpper(id), Name: "Collect " + id, Purpose: "Confirm that the control operated.",
		Fields: []monitoring.TemplateField{{ID: "operated", Label: "Did the control operate?", Type: "single_select", Required: true, Options: []string{"Yes", "No"}}},
		Lifecycle: monitoring.Lifecycle{
			Status: monitoring.LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 3,
			CreatedBy: "form-maker", SubmittedBy: "form-maker", ApprovedBy: "form-reviewer", CreatedAt: now, UpdatedAt: now,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.CreateCheckRevision(t.Context(), monitoring.MonitoringCheck{
		ID: "check-" + id, TenantID: "bank", ProgramID: programID,
		Code: strings.ToUpper("check-" + id), Name: "Check " + id, Claim: "The control operated.", InputKind: monitoring.InputForm,
		FormTemplateID: form.ID, FormTemplateVersion: form.Version, Thresholds: monitoring.DefaultThresholds(),
		FreshnessMinutes: 60, MinimumCoverage: 1, OwnerPrincipalID: "program-owner", ReviewerPrincipalID: "reviewer-" + id, FailureAction: monitoring.FailureReview,
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 2, CreatedBy: "check-maker", ApprovedBy: "check-reviewer", CreatedAt: now, UpdatedAt: now},
	})
	if err != nil {
		t.Fatal(err)
	}
	return form
}

func monitoringCollectionAuthorityFixture(t *testing.T) (*continuity.Service, continuity.ProgramAggregate, *monitoring.MemoryRepository, *monitoring.Service, *monitoringCollectionRecorder, monitoring.FormTemplate, monitoring.FormTemplate, *exactFormAuthority) {
	t.Helper()
	now := time.Now().UTC()
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	program, err := continuityService.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "COLLECTION", Name: "Control collection", Type: "PRIVACY", OwningFunction: "Privacy",
		OwnerPrincipalID: "program-owner", AuthorityPrincipalID: "program-authorizer", EffectiveFrom: now, ActorID: "program-owner", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	repository := monitoring.NewMemoryRepository()
	formA := activeMonitoringForm(t, repository, program.Program.ID, "form-collect-a", now)
	formB := activeMonitoringForm(t, repository, program.Program.ID, "form-collect-b", now)
	requests := &monitoringCollectionRecorder{}
	resolver := &exactFormAuthority{ownerDelegateFormID: formA.ID}
	return continuityService, program, repository, monitoring.NewService(repository, requests), requests, formA, formB, resolver
}

func TestMonitoringCollectionUsesExactFormRoutesAndIgnoresBodyAssignees(t *testing.T) {
	continuityService, program, _, monitoringService, requests, formA, formB, resolver := monitoringCollectionAuthorityFixture(t)
	otherProgram, err := continuityService.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "OTHER", Name: "Other Program", Type: "PRIVACY", OwningFunction: "Privacy",
		OwnerPrincipalID: "program-owner", AuthorityPrincipalID: "program-authorizer", EffectiveFrom: time.Now().UTC(), ActorID: "program-owner", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	guard, err := commandauth.New(resolver, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	handler := func(principalID, legalEntityID string) http.Handler {
		return New(Dependencies{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", principalID, legalEntityID),
			Continuity: continuityService, Monitoring: monitoringService, Authority: resolver, CommandGuard: guard,
		})
	}
	body := func(version int64) string {
		return fmt.Sprintf(`{"tenant_id":"bank","actor_id":"forged-actor","respondent_principal_id":"forged-respondent","reviewer_principal_id":"forged-reviewer","form_template_version":%d,"period_start":"2026-08-19T00:00:00Z","period_end":"2026-08-26T00:00:00Z","deadline":"2026-08-28T00:00:00Z"}`, version)
	}
	collect := func(principalID, legalEntityID, programID string, form monitoring.FormTemplate, version int64) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		path := fmt.Sprintf("/api/v1/programs/%s/form-templates/%s/collections", programID, form.ID)
		handler(principalID, legalEntityID).ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body(version))))
		return response
	}

	if response := collect("owner-delegate", "entity-a", program.Program.ID, formA, formA.Version); response.Code != http.StatusCreated {
		t.Fatalf("exact form owner delegate returned %d: %s", response.Code, response.Body.String())
	}
	if response := collect("owner-delegate", "entity-a", program.Program.ID, formB, formB.Version); response.Code != http.StatusForbidden {
		t.Fatalf("form A owner delegate acted on form B: %d %s", response.Code, response.Body.String())
	}
	if response := collect("broad-program-owner", "entity-a", program.Program.ID, formB, formB.Version); response.Code != http.StatusForbidden {
		t.Fatalf("broad Program owner substituted for the stored exact-form owner: %d %s", response.Code, response.Body.String())
	}
	if response := collect("program-owner", "entity-a", program.Program.ID, formA, formA.Version+1); response.Code < http.StatusBadRequest {
		t.Fatalf("wrong form version was accepted with %d: %s", response.Code, response.Body.String())
	}
	if response := collect("program-owner", "entity-a", otherProgram.Program.ID, formA, formA.Version); response.Code < http.StatusBadRequest {
		t.Fatalf("wrong Program was accepted with %d: %s", response.Code, response.Body.String())
	}
	if response := collect("program-owner", "entity-b", program.Program.ID, formA, formA.Version); response.Code < http.StatusBadRequest {
		t.Fatalf("wrong legal entity was accepted with %d: %s", response.Code, response.Body.String())
	}

	resolver.collisionFormID = formB.ID
	if response := collect("program-owner", "entity-a", program.Program.ID, formB, formB.Version); response.Code < http.StatusBadRequest || len(requests.inputs) != 1 {
		t.Fatalf("same respondent and reviewer was not rejected before creation: %d %s, requests %#v", response.Code, response.Body.String(), requests.inputs)
	}
	resolver.collisionFormID = ""
	if response := collect("program-owner", "entity-a", program.Program.ID, formB, formB.Version); response.Code != http.StatusCreated {
		t.Fatalf("stored Program owner on exact form B returned %d: %s", response.Code, response.Body.String())
	}

	if len(requests.inputs) != 2 {
		t.Fatalf("created collection count = %d, want 2: %#v", len(requests.inputs), requests.inputs)
	}
	for index, want := range []struct {
		formID       string
		respondent   string
		reviewer     string
		commandActor string
	}{
		{formID: formA.ID, respondent: "respondent-" + formA.ID, reviewer: "reviewer-" + formA.ID, commandActor: "owner-delegate"},
		{formID: formB.ID, respondent: "respondent-" + formB.ID, reviewer: "reviewer-" + formB.ID, commandActor: "program-owner"},
	} {
		input := requests.inputs[index]
		if input.FormTemplateID != want.formID || input.Recipient.PrincipalID != want.respondent || input.KnownFacts["reviewer"] != want.reviewer || input.CreatedBy != want.commandActor {
			t.Fatalf("collection %d trusted body assignees or used the wrong form route: %#v", index, input)
		}
		if input.Recipient.PrincipalID == input.KnownFacts["reviewer"] {
			t.Fatalf("collection %d collapsed respondent and reviewer: %#v", index, input)
		}
	}

	found := map[string]map[authority.Responsibility]bool{}
	for _, input := range resolver.inputs {
		if input.DecisionType != "program.monitoring.collect" {
			continue
		}
		if input.ObjectType != "FORM_TEMPLATE" || input.LegalEntityID != "entity-a" {
			t.Fatalf("collection authority was not bound to an exact form and entity: %#v", input)
		}
		if found[input.ObjectID] == nil {
			found[input.ObjectID] = map[authority.Responsibility]bool{}
		}
		found[input.ObjectID][input.Responsibility] = true
	}
	for _, formID := range []string{formA.ID, formB.ID} {
		for _, responsibility := range []authority.Responsibility{authority.ResponsibilityOwner, authority.ResponsibilityPerformer, authority.ResponsibilityReviewer} {
			if !found[formID][responsibility] {
				t.Fatalf("missing exact %s route for %s: %#v", responsibility, formID, resolver.inputs)
			}
		}
	}
}

func TestProgramCollectionOperationsMatchExactFormOwnerAuthorityInOneBatch(t *testing.T) {
	_, program, _, monitoringService, _, formA, formB, _ := monitoringCollectionAuthorityFixture(t)
	find := func(t *testing.T, operations []RecordOperation, formID string) RecordOperation {
		t.Helper()
		for _, operation := range operations {
			if operation.Command == "program.monitoring.collect" && operation.SubresourceID == formID {
				return operation
			}
		}
		t.Fatalf("collection operation %s missing: %#v", formID, operations)
		return RecordOperation{}
	}
	assertActor := func(t *testing.T, principalID string, wantA, wantB bool) {
		t.Helper()
		resolver := &exactFormAuthority{ownerDelegateFormID: formA.ID}
		api := &API{deps: Dependencies{Authority: resolver, Monitoring: monitoringService}}
		actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: principalID, Kind: "PERSON"}
		payload := api.buildProgramOperations(identity.WithActor(t.Context(), actor), actor, program, time.Now().UTC())
		for _, test := range []struct {
			form monitoring.FormTemplate
			want bool
		}{{formA, wantA}, {formB, wantB}} {
			operation := find(t, payload.Operations, test.form.ID)
			if operation.CanAct != test.want || operation.AssignedTo == nil || operation.AssignedTo.ID != "program-owner" {
				t.Fatalf("%s collection operation for %s = %#v, want can_act %v and stored owner", principalID, test.form.ID, operation, test.want)
			}
		}
		if resolver.batchCalls != 1 || resolver.scalarCalls != 0 {
			t.Fatalf("operation authority calls = batch %d scalar %d, want one batch and zero scalar", resolver.batchCalls, resolver.scalarCalls)
		}
		found := map[string]bool{}
		for _, input := range resolver.inputs {
			if input.DecisionType != "program.monitoring.collect" {
				continue
			}
			if input.ObjectType != "FORM_TEMPLATE" || input.Responsibility != authority.ResponsibilityOwner || input.Materiality != 2 || input.LegalEntityID != program.Program.LegalEntityID {
				t.Fatalf("collection operation input was not exact: %#v", input)
			}
			found[input.ObjectID] = true
		}
		if !found[formA.ID] || !found[formB.ID] {
			t.Fatalf("one batch did not contain both exact forms: %#v", resolver.inputs)
		}
	}
	t.Run("exact delegate", func(t *testing.T) { assertActor(t, "owner-delegate", true, false) })
	t.Run("broad Program owner", func(t *testing.T) { assertActor(t, "broad-program-owner", false, false) })
}
