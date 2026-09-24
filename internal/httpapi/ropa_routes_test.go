package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

func TestRopaRoutesAreRegisteredInProductionCatalog(t *testing.T) {
	registered := make(map[string]routeSpec)
	for _, route := range (&API{}).productionRoutes() {
		registered[route.Method+" "+route.Path] = route
	}
	for _, key := range []string{
		http.MethodGet + " /api/v1/ropa/dashboard",
		http.MethodGet + " /api/v1/ropa/processing-activities",
		http.MethodPost + " /api/v1/ropa/processing-activities",
		http.MethodGet + " /api/v1/ropa/processing-activities/{id}",
		http.MethodGet + " /api/v1/ropa/processing-activities/{id}/history",
		http.MethodPost + " /api/v1/ropa/processing-activities/{id}",
		http.MethodPost + " /api/v1/ropa/processing-activities/{id}/transition",
	} {
		if _, ok := registered[key]; !ok {
			t.Errorf("ROPA route %s must be in productionRoutes()", key)
		}
	}
}

func TestRopaWriteRoutesAreMaterialCommandsWithExpectedPolicies(t *testing.T) {
	routes := (&API{}).productionRoutes()
	want := map[string]struct {
		name           string
		responsibility authority.Responsibility
		materiality    int
		bindEntity     bool
	}{
		"POST /api/v1/ropa/processing-activities": {
			name: "ropa.processing_activity.create", responsibility: authority.ResponsibilityOwner, materiality: 3, bindEntity: true,
		},
		"POST /api/v1/ropa/processing-activities/{id}": {
			name: "ropa.processing_activity.update", responsibility: authority.ResponsibilityOwner, materiality: 3,
		},
		"POST /api/v1/ropa/processing-activities/{id}/transition": {
			name: "ropa.processing_activity.transition", responsibility: authority.ResponsibilityAuthorizer, materiality: 4,
		},
	}
	seen := make(map[string]bool)
	for _, route := range routes {
		key := route.Method + " " + route.Path
		expected, ok := want[key]
		if !ok {
			continue
		}
		seen[key] = true
		if route.Class != routeMaterialCommand || route.Command == nil {
			t.Fatalf("%s is not a material command: %#v", key, route)
		}
		if route.Command.Name != expected.name {
			t.Errorf("%s command name = %q, want %q", key, route.Command.Name, expected.name)
		}
		if route.Command.Policy.ObjectType != "PROCESSING_ACTIVITY" {
			t.Errorf("%s object type = %q, want PROCESSING_ACTIVITY", key, route.Command.Policy.ObjectType)
		}
		if route.Command.Policy.Responsibility != expected.responsibility {
			t.Errorf("%s responsibility = %q, want %q", key, route.Command.Policy.Responsibility, expected.responsibility)
		}
		if route.Command.Policy.Materiality != expected.materiality {
			t.Errorf("%s materiality = %d, want %d", key, route.Command.Policy.Materiality, expected.materiality)
		}
		if route.Command.Policy.BindLegalEntity != expected.bindEntity {
			t.Errorf("%s BindLegalEntity = %v, want %v", key, route.Command.Policy.BindLegalEntity, expected.bindEntity)
		}
	}
	for key := range want {
		if !seen[key] {
			t.Errorf("missing ROPA write route %s", key)
		}
	}
}

func TestRopaDashboardAndListReadsAreNotMaterialCommands(t *testing.T) {
	for _, route := range (&API{}).productionRoutes() {
		if route.Method != http.MethodGet || (route.Path != "/api/v1/ropa/dashboard" && route.Path != "/api/v1/ropa/processing-activities") {
			continue
		}
		if route.Class == routeMaterialCommand || route.Command != nil {
			t.Fatalf("ROPA read %s is classified as a material command: %#v", route.Path, route)
		}
	}
}

func TestRopaCommandGuardIgnoresForgedBodyScopeAndActor(t *testing.T) {
	handler, repository, service := ropaHTTPTestAPI(t)
	createBody := `{"tenant_id":"forged-tenant","legal_entity_id":"forged-entity","actor_id":"forged-actor","code":"PA-001","name":"Customer onboarding","purpose":"Onboard customers","lawful_basis":"Contract","controller":"Fidelity Bank","processor":"Internal operations","data_subject_categories":"Customers","owner_principal_id":"owner-1","required_authority_principal_id":"authority-1"}`
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, httptest.NewRequest(http.MethodPost, "/api/v1/ropa/processing-activities", strings.NewReader(createBody)))
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create returned %d: %s", createResponse.Code, createResponse.Body.String())
	}
	var created struct {
		Activity ropa.ProcessingActivity `json:"activity"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Activity.TenantID != "bank" || created.Activity.LegalEntityID != "entity-a" {
		t.Fatalf("verified scope was not used: tenant=%q entity=%q", created.Activity.TenantID, created.Activity.LegalEntityID)
	}
	if created.Activity.ID == "" {
		t.Fatal("create response did not include the processing activity ID")
	}

	updateBody := `{"tenant_id":"another-forged-tenant","legal_entity_id":"another-forged-entity","actor_id":"another-forged-actor","activity_id":"another-forged-id","expected_version":1,"description":"Updated from the verified workspace."}`
	updateResponse := httptest.NewRecorder()
	handler.ServeHTTP(updateResponse, httptest.NewRequest(http.MethodPost, "/api/v1/ropa/processing-activities/"+created.Activity.ID, strings.NewReader(updateBody)))
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update returned %d: %s", updateResponse.Code, updateResponse.Body.String())
	}

	transitionBody := `{"tenant_id":"forged-tenant-again","legal_entity_id":"forged-entity-again","actor_id":"forged-actor-again","expected_version":2,"to":"OPEN"}`
	transitionResponse := httptest.NewRecorder()
	handler.ServeHTTP(transitionResponse, httptest.NewRequest(http.MethodPost, "/api/v1/ropa/processing-activities/"+created.Activity.ID+"/transition", strings.NewReader(transitionBody)))
	if transitionResponse.Code != http.StatusOK {
		t.Fatalf("transition returned %d: %s", transitionResponse.Code, transitionResponse.Body.String())
	}

	activity, err := service.GetActivity(context.Background(), ropa.ActivityScope{TenantID: "bank", LegalEntityID: "entity-a"}, created.Activity.ID)
	if err != nil {
		t.Fatalf("read the activity after commands: %v", err)
	}
	if activity.TenantID != "bank" || activity.LegalEntityID != "entity-a" || activity.Status != ropa.StatusOpen || activity.Description != "Updated from the verified workspace." {
		t.Fatalf("commands did not use verified scope/path: %#v", activity)
	}
	events, _, err := repository.ActivityEvents(context.Background(), ropa.ActivityScope{TenantID: "bank", LegalEntityID: "entity-a"}, created.Activity.ID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("event count = %d, want create/update/transition", len(events))
	}
	for _, event := range events {
		if event.ActorID != "verified-actor" {
			t.Fatalf("event trusted a forged actor: %#v", event)
		}
	}
}

func ropaHTTPTestAPI(t *testing.T) (http.Handler, *ropa.MemoryRepository, *ropa.Service) {
	t.Helper()
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	service := ropa.NewService(repository, summaries)
	guard, err := commandauth.New(commandAuthorityStub{principal: "verified-actor"}, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	return New(Dependencies{
		Logger:           slog.New(slog.NewTextHandler(io.Discard, nil)),
		Mode:             "test-memory",
		Identity:         identity.NewDevelopmentAuthenticator("bank", "verified-actor", "entity-a"),
		CommandGuard:     guard,
		Ropa:             service,
		RopaEventsReader: repository,
	}), repository, service
}

func TestRopaHandlersReadTheExactActivityScopeAndClosureBlockers(t *testing.T) {
	handler, _, service := ropaHTTPTestAPI(t)
	activity, err := service.CreateActivity(context.Background(), ropa.CreateActivityInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "PA-READ", Name: "Read this activity",
		Controller: "Fidelity Bank", ActorID: "seed-actor",
	})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ropa/processing-activities/"+activity.ID+"?tenant_id=bank", nil)
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("read returned %d: %s", response.Code, response.Body.String())
	}
	var value struct {
		StateLabel      string                  `json:"state_label"`
		Activity        ropa.ProcessingActivity `json:"activity"`
		ClosureBlockers []string                `json:"closure_blockers"`
	}
	if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
		t.Fatal(err)
	}
	if value.StateLabel != "Not started" || value.Activity.ID != activity.ID || len(value.ClosureBlockers) != 4 {
		t.Fatalf("read response did not include state, activity and four closure blockers: %#v", value)
	}
	for _, expected := range []string{"lawful basis", "named owner", "data subject category", "completed review"} {
		found := false
		for _, blocker := range value.ClosureBlockers {
			if blocker == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("closure blockers %v do not name %q", value.ClosureBlockers, expected)
		}
	}
}
