package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestCommandCanOmitUnsupportedActorFields(t *testing.T) {
	api := &API{}
	policy := commandPolicy{
		ObjectType:       "PROJECTION",
		Responsibility:   authority.ResponsibilityReviewer,
		Materiality:      3,
		OmitActorBinding: true,
	}
	var received struct {
		TenantID string `json:"tenant_id"`
		Limit    int    `json:"limit"`
	}
	handler := api.command("projection.reconcile", policy, func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&received); err != nil {
			t.Fatalf("strict decode failed after identity binding: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/operations/projections/reconcile", strings.NewReader(`{"limit":10}`))
	now := time.Now().UTC()
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID:      "bank-demo",
		PrincipalID:   "reviewer-1",
		LegalEntityID: "bank-ng",
		Kind:          "PERSON",
		ExpiresAt:     now.Add(time.Hour),
	}))
	response := httptest.NewRecorder()
	handler(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("unexpected response %d: %s", response.Code, response.Body.String())
	}
	if received.TenantID != "bank-demo" || received.Limit != 10 {
		t.Fatalf("unexpected bound payload: %#v", received)
	}
}
