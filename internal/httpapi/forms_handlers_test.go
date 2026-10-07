package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

func TestFormLibraryReadIncludesExactAuthorityOperations(t *testing.T) {
	resolver := &exactFormAuthority{}
	api := &API{deps: Dependencies{Authority: resolver}}
	page := monitoring.FormTemplatePage{Items: []monitoring.FormLibraryItem{{Template: monitoring.FormTemplate{
		ID: "draft-a", TenantID: "bank-a", LegalEntityID: "entity-a", Name: "Vendor review",
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleDraft, Version: 2},
	}}}}
	actor := identity.Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "owner-draft-a"}

	result := api.formLibraryPageWithOperations(context.Background(), actor, page, time.Date(2026, 8, 31, 8, 0, 0, 0, time.UTC))
	if len(result.Items) != 1 || !result.Items[0].AuthorityAvailable || len(result.Items[0].Operations) != 2 {
		t.Fatalf("operation response = %#v", result.Items)
	}
	for _, operation := range result.Items[0].Operations {
		if !operation.CanAct {
			t.Fatalf("owner operation must be permitted: %#v", operation)
		}
	}
	transition := result.Items[0].Operations[1]
	if transition.Command != "forms.template.transition" || len(transition.AllowedTargets) != 1 || transition.AllowedTargets[0] != "PENDING_APPROVAL" {
		t.Fatalf("transition operation = %#v", transition)
	}
	if resolver.batchCalls != 1 || resolver.scalarCalls != 0 {
		t.Fatalf("authority calls batch=%d scalar=%d", resolver.batchCalls, resolver.scalarCalls)
	}
	for _, input := range resolver.inputs {
		if input.ObjectType != "FORM_TEMPLATE" || input.ObjectID != "draft-a" || input.LegalEntityID != "entity-a" {
			t.Fatalf("authority input was not exact: %#v", input)
		}
	}
}

func TestActiveFormLibraryRevisionCanBeRevisedByCurrentOwner(t *testing.T) {
	resolver := &exactFormAuthority{}
	api := &API{deps: Dependencies{Authority: resolver}}
	page := monitoring.FormTemplatePage{Items: []monitoring.FormLibraryItem{{Template: monitoring.FormTemplate{
		ID: "active-a", TenantID: "bank-a", LegalEntityID: "entity-a", Name: "Encryption attestation",
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 3},
	}}}}
	actor := identity.Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "owner-active-a"}

	result := api.formLibraryPageWithOperations(context.Background(), actor, page, time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC))
	if len(result.Items) != 1 || len(result.Items[0].Operations) != 2 {
		t.Fatalf("operation response = %#v", result.Items)
	}
	revise := result.Items[0].Operations[0]
	if revise.Command != "forms.template.revise" || revise.Label != "Create revision" || !revise.CanAct {
		t.Fatalf("revise operation = %#v", revise)
	}
}

func TestFormsRoutesAreRegisteredAndClassified(t *testing.T) {
	want := map[string]routeClass{
		"GET /api/v1/forms/templates":                             routeAuthenticatedRead,
		"GET /api/v1/forms/templates/{id}":                        routeAuthenticatedRead,
		"POST /api/v1/forms/templates":                            routeMaterialCommand,
		"GET /api/v1/forms/templates/{id}/revisions/{version}":    routeAuthenticatedRead,
		"POST /api/v1/forms/templates/{id}/revisions":             routeMaterialCommand,
		"POST /api/v1/forms/templates/{id}/transition":            routeMaterialCommand,
		"GET /api/v1/forms/starter-templates":                     routeAuthenticatedRead,
		"POST /api/v1/forms/starter-templates/{code}/instantiate": routeMaterialCommand,
		"GET /api/v1/forms/saved-views":                           routeAuthenticatedRead,
		"POST /api/v1/forms/saved-views":                          routeAuthenticatedWrite,
		"DELETE /api/v1/forms/saved-views/{id}":                   routeAuthenticatedWrite,
	}
	for _, route := range (&API{}).routes() {
		key := route.Method + " " + route.Path
		class, exists := want[key]
		if !exists {
			continue
		}
		if route.Class != class {
			t.Fatalf("%s class = %s, want %s", key, route.Class, class)
		}
		if class == routeMaterialCommand && (route.Command == nil || route.Command.Policy.ActorField != noActorField) {
			t.Fatalf("%s lacks an actor-free material command policy: %#v", key, route)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("missing Forms routes: %#v", want)
	}
}

type formOriginHTTPValidator struct {
	tenantID      string
	legalEntityID string
	principalID   string
	matterID      string
}

func (v formOriginHTTPValidator) MatterOriginExists(_ context.Context, tenantID, legalEntityID, principalID, matterID string) (bool, error) {
	return tenantID == v.tenantID && legalEntityID == v.legalEntityID && principalID == v.principalID && matterID == v.matterID, nil
}

func formsTestHandler(t *testing.T) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	guard, err := commandauth.New(nil, commandauth.ModeOff, logger)
	if err != nil {
		t.Fatal(err)
	}
	service := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	service.ConfigureCommandGuard(guard)
	return New(Dependencies{
		Logger: logger, Identity: identity.NewDevelopmentAuthenticator("bank-a", "maker-a", "entity-a"),
		CommandGuard: guard, Monitoring: service,
	})
}

func TestFormsCreateListAndExactRevisionUseSignedScope(t *testing.T) {
	handler := formsTestHandler(t)
	body := []byte(`{"code":"VENDOR","name":"Vendor review","purpose":"Collect current vendor evidence.","presentation":{"default_mode":"AUTOMATIC"},"sections":[{"id":"identity","title":"Identity"}],"fields":[{"id":"name","section_id":"identity","label":"Registered name","type":"short_text","required":true}]}`)
	createdResponse := httptest.NewRecorder()
	handler.ServeHTTP(createdResponse, httptest.NewRequest(http.MethodPost, "/api/v1/forms/templates", bytes.NewReader(body)))
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create returned %d: %s", createdResponse.Code, createdResponse.Body.String())
	}
	var created monitoring.FormTemplate
	if err := json.Unmarshal(createdResponse.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.TenantID != "bank-a" || created.LegalEntityID != "entity-a" || created.CreatedBy != "maker-a" {
		t.Fatalf("created scope = %#v", created)
	}

	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/api/v1/forms/templates?search=vendor&limit=25", nil))
	if listResponse.Code != http.StatusOK || !bytes.Contains(listResponse.Body.Bytes(), []byte(created.ID)) {
		t.Fatalf("list returned %d: %s", listResponse.Code, listResponse.Body.String())
	}

	detailResponse := httptest.NewRecorder()
	handler.ServeHTTP(detailResponse, httptest.NewRequest(http.MethodGet, "/api/v1/forms/templates/"+created.ID, nil))
	if detailResponse.Code != http.StatusOK || !bytes.Contains(detailResponse.Body.Bytes(), []byte(`"id":"`+created.ID+`"`)) || !bytes.Contains(detailResponse.Body.Bytes(), []byte(`"operations"`)) {
		t.Fatalf("exact detail returned %d: %s", detailResponse.Code, detailResponse.Body.String())
	}

	exactResponse := httptest.NewRecorder()
	handler.ServeHTTP(exactResponse, httptest.NewRequest(http.MethodGet, "/api/v1/forms/templates/"+created.ID+"/revisions/1", nil))
	if exactResponse.Code != http.StatusOK || !bytes.Contains(exactResponse.Body.Bytes(), []byte(`"version":1`)) {
		t.Fatalf("exact revision returned %d: %s", exactResponse.Code, exactResponse.Body.String())
	}
}

func TestFormsCreateAndRevisePreserveMatterOrigin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	guard, err := commandauth.New(nil, commandauth.ModeOff, logger)
	if err != nil {
		t.Fatal(err)
	}
	service := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	service.ConfigureCommandGuard(guard)
	service.ConfigureFormOriginValidator(formOriginHTTPValidator{
		tenantID: "bank-a", legalEntityID: "entity-a", principalID: "maker-a", matterID: "matter-a",
	})
	handler := New(Dependencies{
		Logger: logger, Identity: identity.NewDevelopmentAuthenticator("bank-a", "maker-a", "entity-a"),
		CommandGuard: guard, Monitoring: service,
	})

	form := `{"code":"ISSUE-CHECK","name":"Issue evidence check","purpose":"Collect issue evidence.","presentation":{"default_mode":"AUTOMATIC"},"sections":[{"id":"evidence","title":"Evidence"}],"fields":[{"id":"state","section_id":"evidence","label":"Current state","type":"short_text","required":true}]}`
	createBody := []byte(`{"origin":{"type":"MATTER","id":"matter-a"},` + strings.TrimPrefix(form, "{"))
	createdResponse := httptest.NewRecorder()
	handler.ServeHTTP(createdResponse, httptest.NewRequest(http.MethodPost, "/api/v1/forms/templates", bytes.NewReader(createBody)))
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create returned %d: %s", createdResponse.Code, createdResponse.Body.String())
	}
	var created monitoring.FormTemplate
	if err := json.Unmarshal(createdResponse.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Origin == nil || created.Origin.Type != monitoring.FormOriginMatter || created.Origin.ID != "matter-a" {
		t.Fatalf("created origin = %#v", created.Origin)
	}

	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/api/v1/forms/templates?origin_type=MATTER&origin_id=matter-a&limit=25", nil))
	if listResponse.Code != http.StatusOK || !bytes.Contains(listResponse.Body.Bytes(), []byte(`"id":"`+created.ID+`"`)) {
		t.Fatalf("origin-filtered list returned %d: %s", listResponse.Code, listResponse.Body.String())
	}
	for _, raw := range []string{
		"/api/v1/forms/templates?origin_type=MATTER",
		"/api/v1/forms/templates?origin_id=matter-a",
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, raw, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("partial origin filter %q returned %d: %s", raw, response.Code, response.Body.String())
		}
	}

	revisionBody := []byte(`{"expected_version":1,"form":` + form + `}`)
	revisedResponse := httptest.NewRecorder()
	handler.ServeHTTP(revisedResponse, httptest.NewRequest(http.MethodPost, "/api/v1/forms/templates/"+created.ID+"/revisions", bytes.NewReader(revisionBody)))
	if revisedResponse.Code != http.StatusCreated {
		t.Fatalf("revision returned %d: %s", revisedResponse.Code, revisedResponse.Body.String())
	}
	var revised monitoring.FormTemplate
	if err := json.Unmarshal(revisedResponse.Body.Bytes(), &revised); err != nil {
		t.Fatal(err)
	}
	if revised.Origin == nil || revised.Origin.ID != "matter-a" || revised.Version != 2 {
		t.Fatalf("revised origin = %#v, version=%d", revised.Origin, revised.Version)
	}
}

func TestFormsCreateAcceptsAdvancedScoreProfile(t *testing.T) {
	handler := formsTestHandler(t)
	body := []byte(`{
		"code":"ENCRYPTION",
		"name":"Monthly cloud encryption Form",
		"purpose":"Confirm current encryption controls.",
		"scoring_mode":"RISK",
		"score_profile":{
			"version":"risk-v1",
			"mode":"RISK",
			"direction":"HIGH_IS_POOR",
			"contributions":[{
				"id":"encryption-disabled",
				"label":"Encryption not enabled",
				"weight":100,
				"predicate":{"field_id":"encrypted","operator":"EQUALS","values":["No"]},
				"match_points":100,
				"non_match_points":0,
				"missing":"INDETERMINATE"
			}],
			"bands":[
				{"band":"LOW","from":0,"through":24},
				{"band":"MODERATE","from":25,"through":49},
				{"band":"HIGH","from":50,"through":74},
				{"band":"CRITICAL","from":75,"through":100}
			]
		},
		"presentation":{"default_mode":"AUTOMATIC"},
		"sections":[{"id":"encryption","title":"Encryption"}],
		"fields":[{"id":"encrypted","section_id":"encryption","label":"Is encryption enabled?","type":"yes_no","required":true}]
	}`)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/forms/templates", bytes.NewReader(body)))
	if response.Code != http.StatusCreated {
		t.Fatalf("create returned %d: %s", response.Code, response.Body.String())
	}
	var created monitoring.FormTemplate
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ScoreProfile == nil || created.ScoreProfile.Version != "risk-v1" || len(created.ScoreProfile.Contributions) != 1 {
		t.Fatalf("score profile = %#v", created.ScoreProfile)
	}
}

func TestFormsRejectUnknownScopeAndQueryOverrides(t *testing.T) {
	handler := formsTestHandler(t)
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/v1/forms/templates?tenant_id=bank-b", nil),
		httptest.NewRequest(http.MethodPost, "/api/v1/forms/templates", bytes.NewBufferString(`{"tenant_id":"bank-b","code":"VENDOR","name":"Vendor review","purpose":"Collect evidence.","fields":[]}`)),
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code < 400 {
			t.Fatalf("scope override returned %d: %s", response.Code, response.Body.String())
		}
	}
}
