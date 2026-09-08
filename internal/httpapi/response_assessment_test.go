package httpapi

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestResponseAssessmentHandlersRequireVerifiedIdentity(t *testing.T) {
	api := &API{deps: Dependencies{FormDistributions: evidence.NewDistributionService(evidence.NewMemoryDistributionStore(evidence.NewMemoryRepository(nil, nil), nil, nil))}}
	for _, method := range []string{"GET", "POST"} {
		r := httptest.NewRequest(method, "/api/v1/forms/responses/response/assessment", strings.NewReader(`{"expected_version":0,"decisions":[],"reviewer_id":"spoof"}`))
		r.SetPathValue("revision_id", "response")
		w := httptest.NewRecorder()
		if method == "GET" {
			api.getResponseAssessment(w, r)
		} else {
			api.recordResponseAssessment(w, r)
		}
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s: %d %s", method, w.Code, w.Body.String())
		}
	}
}

func TestResponseAssessmentPOSTRequiresExpectedVersion(t *testing.T) {
	api := &API{deps: Dependencies{FormDistributions: evidence.NewDistributionService(evidence.NewMemoryDistributionStore(evidence.NewMemoryRepository(nil, nil), nil, nil))}}
	actor := identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reviewer", ExpiresAt: time.Now().Add(time.Hour)}
	r := httptest.NewRequest("POST", "/api/v1/forms/responses/response/assessment", strings.NewReader(`{"decisions":[{"field_id":"report","outcome_id":"poor","rationale":"Missing scope."}]}`))
	r = r.WithContext(identity.WithActor(r.Context(), actor))
	r.SetPathValue("revision_id", "response")
	w := httptest.NewRecorder()
	api.recordResponseAssessment(w, r)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("missing version status=%d body=%s", w.Code, w.Body.String())
	}
}
