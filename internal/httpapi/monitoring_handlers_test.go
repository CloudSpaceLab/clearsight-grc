package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type monitoringRequestRecorder struct {
	input evidence.CreateRequestInput
}

func (r *monitoringRequestRecorder) CreateRequest(_ context.Context, input evidence.CreateRequestInput) (evidence.Request, error) {
	r.input = input
	return evidence.Request{ID: "request-1", TenantID: input.TenantID, SubjectType: input.SubjectType, SubjectID: input.SubjectID, FormTemplateID: input.FormTemplateID, FormTemplateVersion: input.FormTemplateVersion, Version: 1}, nil
}

type monitoringSourceFixture struct {
	tenant        string
	legalEntityID string
	bindings      map[string]sourceaccess.BindingRevision
	bindingErr    error
	validationErr error
}

func newMonitoringSourceFixture(tenant, legalEntityID string, bindingIDs ...string) *monitoringSourceFixture {
	now := time.Now().UTC()
	fixture := &monitoringSourceFixture{tenant: tenant, legalEntityID: legalEntityID, bindings: map[string]sourceaccess.BindingRevision{}}
	for _, bindingID := range bindingIDs {
		fixture.bindings[bindingID] = sourceaccess.BindingRevision{
			BindingID: bindingID, TenantID: tenant, SourceID: "source-" + bindingID,
			Operations:        []sourceaccess.Operation{sourceaccess.OperationPage},
			RevisionLifecycle: sourceaccess.RevisionLifecycle{Status: sourceaccess.RevisionActive, IsCurrent: true, EffectiveFrom: &now, Version: 1},
		}
	}
	return fixture
}

func (f *monitoringSourceFixture) Binding(_ context.Context, tenant, bindingID string, version int64) (sourceaccess.BindingRevision, error) {
	if f.bindingErr != nil {
		return sourceaccess.BindingRevision{}, f.bindingErr
	}
	binding, ok := f.bindings[bindingID]
	if !ok || tenant != f.tenant || version != binding.Version {
		return sourceaccess.BindingRevision{}, sourceaccess.ErrCatalogNotFound
	}
	return binding, nil
}

func (f *monitoringSourceFixture) PreviewBinding(context.Context, string, string, int64, sourceaccess.PageRequest) (sourceaccess.RecordPage, error) {
	return sourceaccess.RecordPage{}, sourceaccess.ErrCatalogStorage
}

func (f *monitoringSourceFixture) ValidateActiveSourcesForEntity(_ context.Context, tenant, legalEntityID string, sourceIDs []string) error {
	if f.validationErr != nil {
		return f.validationErr
	}
	if tenant != f.tenant || legalEntityID != f.legalEntityID || len(sourceIDs) != 1 {
		return evidence.ErrSourceScopeMismatch
	}
	for _, binding := range f.bindings {
		if binding.SourceID == sourceIDs[0] {
			return nil
		}
	}
	return evidence.ErrSourceScopeMismatch
}

func configureMonitoringSources(service *monitoring.Service, fixture *monitoringSourceFixture) {
	service.ConfigureSourceReader(fixture)
	service.ConfigureSourceValidator(fixture)
}

func TestMonitoringRoutesAreRegisteredOnce(t *testing.T) {
	routes := (&API{}).routes()
	want := map[string]bool{
		"GET /api/v1/programs/{id}/form-templates":                        false,
		"POST /api/v1/programs/{id}/form-templates":                       false,
		"POST /api/v1/programs/{id}/form-templates/{form_id}/transition":  false,
		"POST /api/v1/programs/{id}/form-templates/{form_id}/collections": false,
		"GET /api/v1/programs/{id}/monitoring-checks":                     false,
		"POST /api/v1/programs/{id}/monitoring-checks":                    false,
		"POST /api/v1/monitoring-checks/{id}/transition":                  false,
		"POST /api/v1/monitoring-checks/{id}/evaluate-source":             false,
		"GET /api/v1/monitoring-checks/{id}/results":                      false,
		"POST /api/v1/monitoring-results/{result_id}/linked-issue":        false,
	}
	for _, route := range routes {
		key := route.Method + " " + route.Path
		if _, expected := want[key]; expected {
			if want[key] {
				t.Fatalf("duplicate route %s", key)
			}
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("missing route %s", route)
		}
	}
	commands := map[string]string{
		"POST /api/v1/programs/{id}/form-templates":                       "program.monitoring.form.define",
		"POST /api/v1/programs/{id}/form-templates/{form_id}/transition":  "program.monitoring.form.transition",
		"POST /api/v1/programs/{id}/form-templates/{form_id}/collections": "program.monitoring.collect",
		"POST /api/v1/programs/{id}/monitoring-checks":                    "program.monitoring.define",
		"POST /api/v1/monitoring-checks/{id}/transition":                  "program.monitoring.transition",
		"POST /api/v1/monitoring-checks/{id}/evaluate-source":             "program.monitoring.evaluate",
		"POST /api/v1/monitoring-results/{result_id}/linked-issue":        "program.monitoring.issue.create",
	}
	for _, route := range routes {
		key := route.Method + " " + route.Path
		name, expected := commands[key]
		if !expected {
			continue
		}
		if route.Class != routeMaterialCommand || route.Command == nil || route.Command.Name != name {
			t.Fatalf("%s is not registered as material command %s: %#v", key, name, route)
		}
		delete(commands, key)
	}
	if len(commands) != 0 {
		t.Fatalf("material monitoring routes were not found: %#v", commands)
	}
}

func TestCreateFormTemplateUsesVerifiedIdentity(t *testing.T) {
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	program, err := continuityService.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank-a", LegalEntityID: "entity-a", Code: "PRIVACY", Name: "Privacy", Type: "PRIVACY", OwningFunction: "Privacy",
		OwnerPrincipalID: "maker", AuthorityPrincipalID: "authorizer", EffectiveFrom: time.Now().UTC(), ActorID: "maker", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	service := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	handler := New(Dependencies{
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity:   identity.NewDevelopmentAuthenticator("bank-a", "maker", "entity-a"),
		Monitoring: service,
		Continuity: continuityService,
		Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
			authority.ResponsibilityOwner:    {Principal: authority.Principal{ID: "maker"}},
			authority.ResponsibilityReviewer: {Principal: authority.Principal{ID: "reviewer"}},
		}},
	})
	body := []byte(fmt.Sprintf(`{"tenant_id":"bank-a","legal_entity_id":"entity-a","program_id":%q,"created_by":"other","actor_id":"other","code":"RESET","name":"Password reset review","purpose":"Review password reset safeguards.","fields":[{"id":"secure","label":"Identity checks completed","type":"single_select","required":true,"options":["Yes","No"],"scoring":{"weight":1,"answer_scores":{"Yes":0,"No":100},"critical_answers":["No"]}}]}`, program.Program.ID))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID+"/form-templates", bytes.NewReader(body)))
	if response.Code != http.StatusCreated {
		t.Fatalf("create form returned %d: %s", response.Code, response.Body.String())
	}
	var form monitoring.FormTemplate
	if err := json.Unmarshal(response.Body.Bytes(), &form); err != nil {
		t.Fatal(err)
	}
	if form.TenantID != "bank-a" || form.ProgramID != program.Program.ID || form.LegalEntityID != "entity-a" || form.CreatedBy != "maker" || form.Status != monitoring.LifecycleDraft || form.Fields[0].Scoring.ID != "secure" {
		t.Fatalf("created form = %#v", form)
	}
}

func TestMonitoringListCannotCrossTenant(t *testing.T) {
	service := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	handler := New(Dependencies{
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity:   identity.NewDevelopmentAuthenticator("bank-a", "maker", "entity-a"),
		Monitoring: service,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/programs/program-a/form-templates?tenant_id=bank-b", nil))
	if response.Code == http.StatusOK {
		t.Fatalf("cross-tenant list returned %d: %s", response.Code, response.Body.String())
	}
}

func TestMonitoringProgramReadsBindTheExactRecordEntity(t *testing.T) {
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	program, err := continuityService.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank-a", LegalEntityID: "entity-a", Code: "PRIVACY", Name: "Privacy", Type: "PRIVACY", OwningFunction: "Privacy",
		OwnerPrincipalID: "owner-a", AuthorityPrincipalID: "reviewer-a", EffectiveFrom: time.Now().UTC(), ActorID: "owner-a", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	monitoringService := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	configureMonitoringSources(monitoringService, newMonitoringSourceFixture("bank-a", "entity-a", "binding-1"))
	check, err := monitoringService.CreateCheck(t.Context(), monitoring.Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "owner-a"}, monitoring.CreateCheckInput{
		ProgramID: program.Program.ID, Code: "STATUS", Name: "Status check", Claim: "The service remains available.", InputKind: monitoring.InputSource,
		BindingID: "binding-1", BindingVersion: 1, SourceRules: []monitoring.SourceRule{{ID: "available", Field: "available", Operator: monitoring.OperatorEquals, Expected: "true", RiskPoints: 100}},
		FreshnessMinutes: 60, MinimumCoverage: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank-a", "viewer-b", "entity-b"),
		Continuity: continuityService, Monitoring: monitoringService,
	})
	for _, path := range []string{
		"/api/v1/programs/" + program.Program.ID + "/monitoring-checks?tenant_id=bank-a",
		"/api/v1/monitoring-checks/" + check.ID + "/results?tenant_id=bank-a",
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("cross-entity read %s returned %d: %s", path, response.Code, response.Body.String())
		}
	}
}

func TestCreateMonitoringCheckUsesCurrentProgramOwnerAndReviewer(t *testing.T) {
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	program, err := continuityService.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank-a", LegalEntityID: "entity-a", Code: "PRIVACY", Name: "Privacy", Type: "PRIVACY", OwningFunction: "Privacy",
		OwnerPrincipalID: "owner-a", AuthorityPrincipalID: "authorizer-a", EffectiveFrom: time.Now().UTC(), ActorID: "owner-a", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	monitoringService := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	sources := newMonitoringSourceFixture("bank-a", "entity-a", "binding-1")
	configureMonitoringSources(monitoringService, sources)
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner:     {Principal: authority.Principal{ID: "owner-a", DisplayName: "Data Protection Officer"}},
		authority.ResponsibilityReviewer:  {Principal: authority.Principal{ID: "reviewer-a", DisplayName: "Controls reviewer"}},
		authority.ResponsibilityPerformer: {Principal: authority.Principal{ID: "performer-a", DisplayName: "Monitoring analyst"}},
	}}
	makeHandler := func(principal string) http.Handler {
		return New(Dependencies{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank-a", principal, "entity-a"),
			Continuity: continuityService, Monitoring: monitoringService, Authority: resolver,
		})
	}
	body := func() *bytes.Reader {
		return bytes.NewReader([]byte(`{"code":"STATUS","name":"Status check","claim":"The service remains available.","input_kind":"SOURCE","binding_id":"binding-1","binding_version":1,"source_rules":[{"id":"available","field":"available","operator":"EQUALS","expected":"true","risk_points":100}],"freshness_minutes":60,"minimum_coverage":1,"owner_principal_id":"forged-owner","reviewer_principal_id":"forged-reviewer"}`))
	}
	denied := httptest.NewRecorder()
	makeHandler("other-owner").ServeHTTP(denied, httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID+"/monitoring-checks", body()))
	if denied.Code != http.StatusForbidden {
		t.Fatalf("unassigned owner create returned %d: %s", denied.Code, denied.Body.String())
	}

	created := httptest.NewRecorder()
	makeHandler("owner-a").ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID+"/monitoring-checks", body()))
	if created.Code != http.StatusCreated {
		t.Fatalf("assigned owner create returned %d: %s", created.Code, created.Body.String())
	}
	var check monitoring.MonitoringCheck
	if err := json.NewDecoder(created.Body).Decode(&check); err != nil {
		t.Fatal(err)
	}
	if check.OwnerPrincipalID != "owner-a" || check.ReviewerPrincipalID != "reviewer-a" || check.CreatedBy != "owner-a" {
		t.Fatalf("monitoring responsibilities trusted browser input: %#v", check)
	}

	sources.validationErr = errors.New("evidence repository unavailable")
	unavailable := httptest.NewRecorder()
	makeHandler("owner-a").ServeHTTP(unavailable, httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID+"/monitoring-checks", body()))
	if unavailable.Code != http.StatusServiceUnavailable || !bytes.Contains(unavailable.Body.Bytes(), []byte(`"error":"monitoring_source_unavailable"`)) {
		t.Fatalf("source validation infrastructure failure returned %d: %s", unavailable.Code, unavailable.Body.String())
	}

	sources.validationErr = nil
	sources.bindingErr = sourceaccess.ErrCatalogNotFound
	invalid := httptest.NewRecorder()
	makeHandler("owner-a").ServeHTTP(invalid, httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID+"/monitoring-checks", body()))
	if invalid.Code != http.StatusUnprocessableEntity || !bytes.Contains(invalid.Body.Bytes(), []byte(`"error":"monitoring_invalid"`)) {
		t.Fatalf("missing source configuration returned %d: %s", invalid.Code, invalid.Body.String())
	}
}

func TestStartFormCollectionResolvesDistinctCurrentResponsibilitiesAndIgnoresBodyActors(t *testing.T) {
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	program, err := continuityService.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank-a", LegalEntityID: "entity-a", Code: "PRIVACY", Name: "Privacy", Type: "PRIVACY", OwningFunction: "Privacy",
		OwnerPrincipalID: "owner-a", AuthorityPrincipalID: "authorizer-a", EffectiveFrom: time.Now().UTC(), ActorID: "owner-a", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	repo := monitoring.NewMemoryRepository()
	requests := &monitoringRequestRecorder{}
	monitoringService := monitoring.NewService(repo, requests)
	activeAt := time.Now().UTC().Add(-time.Hour)
	form := monitoring.FormTemplate{
		ID: "form-1", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: program.Program.ID, Code: "REVIEW", Name: "Control review", Purpose: "Confirm that safeguards operated.",
		Fields:    []monitoring.TemplateField{{ID: "operated", Label: "Did the safeguard operate?", Type: "single_select", Required: true, Options: []string{"Yes", "No"}}},
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 3},
	}
	if _, err := repo.CreateFormRevision(t.Context(), form); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateCheckRevision(t.Context(), monitoring.MonitoringCheck{
		ID: "check-1", TenantID: "bank-a", ProgramID: program.Program.ID, Code: "REVIEW-CHECK", Name: "Control review", Claim: "Safeguards operated.",
		InputKind: monitoring.InputForm, FormTemplateID: form.ID, FormTemplateVersion: form.Version, Thresholds: monitoring.DefaultThresholds(), FreshnessMinutes: 60, MinimumCoverage: 1, FailureAction: monitoring.FailureReview,
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 2},
	}); err != nil {
		t.Fatal(err)
	}
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner: {
			Principal: authority.Principal{ID: "owner-a"}, CandidatePrincipals: []authority.Principal{{ID: "owner-delegate"}},
			EffectiveOrigins: []authority.EffectiveOrigin{{PrincipalID: "owner-a", OriginPrincipalID: "owner-a"}, {PrincipalID: "owner-delegate", OriginPrincipalID: "owner-a"}},
		},
		authority.ResponsibilityPerformer: {Principal: authority.Principal{ID: "respondent-a"}},
		authority.ResponsibilityReviewer:  {Principal: authority.Principal{ID: "reviewer-a"}},
	}}
	makeHandler := func(principal string) http.Handler {
		return New(Dependencies{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank-a", principal, "entity-a"),
			Continuity: continuityService, Monitoring: monitoringService, Authority: resolver,
		})
	}
	body := fmt.Sprintf(`{"form_template_version":3,"program_id":%q,"respondent_principal_id":"forged-respondent","reviewer_principal_id":"forged-reviewer","period_start":"2026-08-19T00:00:00Z","period_end":"2026-08-26T00:00:00Z","deadline":"2026-08-28T00:00:00Z"}`, program.Program.ID)
	response := httptest.NewRecorder()
	makeHandler("owner-delegate").ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID+"/form-templates/"+form.ID+"/collections", bytes.NewBufferString(body)))
	if response.Code != http.StatusCreated {
		t.Fatalf("delegated owner collection returned %d: %s", response.Code, response.Body.String())
	}
	if requests.input.SubjectID != program.Program.ID || requests.input.Recipient.PrincipalID != "respondent-a" || requests.input.KnownFacts["reviewer"] != "reviewer-a" || requests.input.CreatedBy != "owner-delegate" {
		t.Fatalf("collection responsibilities were not server-bound: %#v", requests.input)
	}

	denied := httptest.NewRecorder()
	makeHandler("unrelated-owner").ServeHTTP(denied, httptest.NewRequest(http.MethodPost, "/api/v1/programs/"+program.Program.ID+"/form-templates/"+form.ID+"/collections", bytes.NewBufferString(body)))
	if denied.Code != http.StatusForbidden {
		t.Fatalf("unrelated owner collection returned %d: %s", denied.Code, denied.Body.String())
	}
}

func TestMonitoringTransitionsAndEvaluationUseStoredCurrentResponsibilities(t *testing.T) {
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	program, err := continuityService.CreateProgram(continuity.WithTrustedSystemScope(t.Context()), continuity.CreateProgramInput{
		TenantID: "bank-a", LegalEntityID: "entity-a", Code: "PRIVACY", Name: "Privacy", Type: "PRIVACY", OwningFunction: "Privacy",
		OwnerPrincipalID: "owner-a", AuthorityPrincipalID: "authorizer-a", EffectiveFrom: time.Now().UTC(), ActorID: "owner-a", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	monitoringService := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	configureMonitoringSources(monitoringService, newMonitoringSourceFixture("bank-a", "entity-a", "binding-1"))
	check, err := monitoringService.CreateCheck(t.Context(), monitoring.Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "owner-a"}, monitoring.CreateCheckInput{
		ProgramID: program.Program.ID, Code: "STATUS", Name: "Status check", Claim: "The service remains available.", InputKind: monitoring.InputSource,
		BindingID: "binding-1", BindingVersion: 1, SourceRules: []monitoring.SourceRule{{ID: "available", Field: "available", Operator: monitoring.OperatorEquals, Expected: "true", RiskPoints: 100}},
		FreshnessMinutes: 60, MinimumCoverage: 1, OwnerPrincipalID: "owner-a", ReviewerPrincipalID: "reviewer-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver := &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
		authority.ResponsibilityOwner:     {Principal: authority.Principal{ID: "owner-a", DisplayName: "Data Protection Officer"}},
		authority.ResponsibilityReviewer:  {Principal: authority.Principal{ID: "reviewer-a", DisplayName: "Controls reviewer"}},
		authority.ResponsibilityPerformer: {Principal: authority.Principal{ID: "performer-a", DisplayName: "Monitoring analyst"}},
	}}
	makeHandler := func(principal, entity string) http.Handler {
		return New(Dependencies{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank-a", principal, entity),
			Continuity: continuityService, Monitoring: monitoringService, Authority: resolver,
		})
	}
	transition := func(principal, entity string, version int64, to monitoring.LifecycleStatus) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		body := bytes.NewBufferString(fmt.Sprintf(`{"expected_version":%d,"to":%q}`, version, to))
		makeHandler(principal, entity).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/monitoring-checks/"+check.ID+"/transition", body))
		return response
	}
	if response := transition("other-owner", "entity-a", 1, monitoring.LifecyclePendingApproval); response.Code != http.StatusForbidden {
		t.Fatalf("unassigned draft transition returned %d: %s", response.Code, response.Body.String())
	}
	if response := transition("owner-a", "entity-b", 1, monitoring.LifecyclePendingApproval); response.Code != http.StatusNotFound {
		t.Fatalf("cross-entity draft transition returned %d: %s", response.Code, response.Body.String())
	}
	if response := transition("owner-a", "entity-a", 1, monitoring.LifecyclePendingApproval); response.Code != http.StatusOK {
		t.Fatalf("assigned owner draft transition returned %d: %s", response.Code, response.Body.String())
	}
	if response := transition("owner-a", "entity-a", 2, monitoring.LifecycleActive); response.Code != http.StatusForbidden {
		t.Fatalf("owner approval returned %d: %s", response.Code, response.Body.String())
	}
	if response := transition("reviewer-a", "entity-a", 2, monitoring.LifecycleActive); response.Code != http.StatusOK {
		t.Fatalf("assigned reviewer approval returned %d: %s", response.Code, response.Body.String())
	}

	evaluate := func(principal string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		makeHandler(principal, "entity-a").ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/monitoring-checks/"+check.ID+"/evaluate-source", bytes.NewBufferString(`{"check_version":3}`)))
		return response
	}
	if response := evaluate("owner-a"); response.Code != http.StatusForbidden {
		t.Fatalf("unassigned evaluator returned %d: %s", response.Code, response.Body.String())
	}
	if response := evaluate("performer-a"); response.Code != http.StatusInternalServerError {
		t.Fatalf("assigned evaluator did not reach the configured source boundary, got %d: %s", response.Code, response.Body.String())
	}
}
