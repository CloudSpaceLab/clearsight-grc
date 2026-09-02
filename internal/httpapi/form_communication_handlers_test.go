package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestFormCommunicationRoutesAreGovernedAndComplete(t *testing.T) {
	t.Parallel()

	routes := (&API{}).formCommunicationRoutes()
	if len(routes) != 15 {
		t.Fatalf("communication route count = %d, want 15", len(routes))
	}
	seen := map[string]routeSpec{}
	for _, route := range routes {
		seen[route.Method+" "+route.Path] = route
	}
	for _, key := range []string{
		"GET /api/v1/forms/communications/profiles",
		"GET /api/v1/forms/communications/profiles/{version}",
		"GET /api/v1/forms/communications/templates/{action}/{locale}/revisions/{version}",
		"POST /api/v1/forms/communications/templates/{action}/{locale}/revisions/{version}/preview",
		"POST /api/v1/forms/communications/templates/{action}/{locale}/revisions/{version}/test-send",
		"GET /api/v1/forms/communications/brand-assets",
	} {
		if _, ok := seen[key]; !ok {
			t.Fatalf("missing communication route %s", key)
		}
	}

	for _, route := range routes {
		if route.Method == http.MethodGet {
			if route.Class != routeAuthenticatedRead || route.Permission != identity.PermissionConfigRead {
				t.Fatalf("read route %s %s is not config-read guarded: %#v", route.Method, route.Path, route)
			}
			continue
		}
		if route.Path[len(route.Path)-8:] == "/preview" || route.Path[len(route.Path)-7:] == "/impact" {
			if route.Class != routeAuthenticatedWrite || route.Permission != identity.PermissionConfigRead {
				t.Fatalf("preview/impact route is not bounded config read: %#v", route)
			}
			continue
		}
		if route.Class != routeMaterialCommand || route.Permission != identity.PermissionConfigWrite || route.Command == nil || route.Command.Policy.ActorField != noActorField || !route.Command.Policy.BindLegalEntity {
			t.Fatalf("mutation route is not a verified material config command: %#v", route)
		}
	}
}

func TestFormCommunicationHandlersAcceptVerifiedScopeBoundByCommandMiddleware(t *testing.T) {
	t.Parallel()

	service := evidence.NewCommunicationService(evidence.NewMemoryCommunicationStore())
	api := &API{deps: Dependencies{FormCommunications: service}}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker"}

	profileResponse := httptest.NewRecorder()
	profileRequest := httptest.NewRequest(http.MethodPost, "/api/v1/forms/communications/profiles", bytes.NewBufferString(`{
		"tenant_id":"bank",
		"legal_entity_id":"entity",
		"default_locale":"en",
		"bank_name":"Clear Bank",
		"support_contact":"support@example.test",
		"effective_from":"2026-09-02T18:00:00Z"
	}`))
	profileRequest = profileRequest.WithContext(identity.WithActor(profileRequest.Context(), actor))
	api.createCommunicationProfile(profileResponse, profileRequest)
	if profileResponse.Code != http.StatusCreated {
		t.Fatalf("create profile with verified scope = %d: %s", profileResponse.Code, profileResponse.Body.String())
	}

	profileTransitionResponse := httptest.NewRecorder()
	profileTransitionRequest := httptest.NewRequest(http.MethodPost, "/api/v1/forms/communications/profiles/1/transition", bytes.NewBufferString(`{
		"tenant_id":"bank",
		"legal_entity_id":"entity",
		"expected_version":1,
		"to":"PENDING_APPROVAL"
	}`))
	profileTransitionRequest.SetPathValue("version", "1")
	profileTransitionRequest = profileTransitionRequest.WithContext(identity.WithActor(profileTransitionRequest.Context(), actor))
	api.transitionCommunicationProfile(profileTransitionResponse, profileTransitionRequest)
	if profileTransitionResponse.Code != http.StatusOK {
		t.Fatalf("transition profile with verified scope = %d: %s", profileTransitionResponse.Code, profileTransitionResponse.Body.String())
	}

	templateResponse := httptest.NewRecorder()
	templateRequest := httptest.NewRequest(http.MethodPost, "/api/v1/forms/communications/templates", bytes.NewBufferString(`{
		"tenant_id":"bank",
		"legal_entity_id":"entity",
		"action":"AMENDMENT",
		"locale":"en",
		"subject_template":"Updated request: {{form_title}}",
		"document":[{"type":"paragraph","text":"The request details changed."},{"type":"primary-action","text":"Review request","href":"{{secure_form_link}}"}],
		"effective_from":"2026-09-02T18:00:00Z"
	}`))
	templateRequest = templateRequest.WithContext(identity.WithActor(templateRequest.Context(), actor))
	api.createCommunicationTemplate(templateResponse, templateRequest)
	if templateResponse.Code != http.StatusCreated {
		t.Fatalf("create template with verified scope = %d: %s", templateResponse.Code, templateResponse.Body.String())
	}

	templateTransitionResponse := httptest.NewRecorder()
	templateTransitionRequest := httptest.NewRequest(http.MethodPost, "/api/v1/forms/communications/templates/AMENDMENT/en/revisions/1/transition", bytes.NewBufferString(`{
		"tenant_id":"bank",
		"legal_entity_id":"entity",
		"expected_version":1,
		"to":"PENDING_APPROVAL"
	}`))
	templateTransitionRequest.SetPathValue("action", "AMENDMENT")
	templateTransitionRequest.SetPathValue("locale", "en")
	templateTransitionRequest.SetPathValue("version", "1")
	templateTransitionRequest = templateTransitionRequest.WithContext(identity.WithActor(templateTransitionRequest.Context(), actor))
	api.transitionCommunicationTemplate(templateTransitionResponse, templateTransitionRequest)
	if templateTransitionResponse.Code != http.StatusOK {
		t.Fatalf("transition template with verified scope = %d: %s", templateTransitionResponse.Code, templateTransitionResponse.Body.String())
	}
}
