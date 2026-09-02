package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func TestGovernedFormDistributionAcceptsVerifiedScopeBinding(t *testing.T) {
	api := &API{}
	policy := commandPolicy{
		ObjectType:      "LEGAL_ENTITY",
		Responsibility:  authority.ResponsibilityOwner,
		Materiality:     3,
		BindLegalEntity: true,
		ActorField:      noActorField,
	}

	handler := api.command("forms.distribution.create", policy, func(w http.ResponseWriter, r *http.Request) {
		var input createFormDistributionRequest
		if err := httpx.DecodeJSON(w, r, &input); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The distribution request must be valid JSON.")
			return
		}
		if input.LegalEntityID != "entity-1" {
			t.Fatalf("verified scope was not preserved for the handler: %#v", input)
		}
		if len(input.Recipients) != 1 || input.Recipients[0].AudienceHint != "v***@example.com" || input.Recipients[0].ContactLabel != "Vendor security lead" {
			t.Fatalf("canonical external recipient fields were not decoded: %#v", input.Recipients)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/forms/distributions", strings.NewReader(`{
		"form_template_id":"form-1",
		"form_template_version":1,
		"subject_type":"PROGRAM",
		"subject_id":"program-1",
		"title":"Confirm current evidence",
		"purpose":"Collect the evidence required for the current review.",
		"access_policy":"DIRECT_MAGIC_LINK",
		"estimated_minutes":5,
		"deadline":"2026-09-03T12:00:00Z",
		"route_expires_at":"2026-09-03T10:00:00Z",
		"recipients":[{
			"role":"TO",
			"type":"EXTERNAL_AUDIENCE",
			"address":"vendor@example.com",
			"audience_hint":"v***@example.com",
			"contact_label":"Vendor security lead"
		}]
	}`))
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank-1", LegalEntityID: "entity-1", PrincipalID: "owner-1", Kind: "PERSON",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	handler(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("governed distribution body was rejected after scope binding: %d %s", response.Code, response.Body.String())
	}
}
