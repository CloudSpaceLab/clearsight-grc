package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestResponseReviewHandlersRequireIdentityAndVersion(t *testing.T) {
	api := &API{deps: Dependencies{FormDistributions: evidence.NewDistributionService(evidence.NewMemoryDistributionStore(evidence.NewMemoryRepository(nil, nil), nil, nil))}}
	for _, method := range []string{"GET", "POST"} {
		r := httptest.NewRequest(method, "/api/v1/forms/responses/response/review", strings.NewReader(`{"decision":"REVIEWED","reviewer_id":"spoof"}`))
		r.SetPathValue("revision_id", "response")
		w := httptest.NewRecorder()
		if method == "GET" {
			api.getResponseReviews(w, r)
		} else {
			api.recordResponseReview(w, r)
		}
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s status=%d body=%s", method, w.Code, w.Body.String())
		}
	}
	r := httptest.NewRequest("POST", "/api/v1/forms/responses/response/review", strings.NewReader(`{"decision":"REVIEWED","reviewer_id":"spoof"}`))
	r = r.WithContext(identity.WithActor(r.Context(), identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reviewer", ExpiresAt: time.Now().Add(time.Hour)}))
	r.SetPathValue("revision_id", "response")
	w := httptest.NewRecorder()
	api.recordResponseReview(w, r)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("missing version status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestResponseReviewRouteRequiresExactReviewerAuthority(t *testing.T) {
	var readOK, writeOK bool
	for _, route := range (&API{}).formDistributionRoutes() {
		if route.Path != "/api/v1/forms/responses/{revision_id}/review" {
			continue
		}
		if route.Method == http.MethodGet {
			readOK = route.Class == routeAuthenticatedRead
		}
		if route.Method == http.MethodPost && route.Command != nil {
			p := route.Command.Policy
			writeOK = route.Class == routeMaterialCommand && route.Command.Name == "forms.response.review" && p.ObjectType == "FORM_RESPONSE" && p.ObjectIDPath == "revision_id" && p.Responsibility == authority.ResponsibilityReviewer && p.ActorField == noActorField
		}
	}
	if !readOK || !writeOK {
		t.Fatalf("review routes read=%t write=%t", readOK, writeOK)
	}
}

func TestResponseReviewLabelsUseRecordedReviewer(t *testing.T) {
	resolver := &assessmentLabelResolver{resolution: access.Resolution{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "original", DisplayName: "Ada Okafor"}}
	api := &API{deps: Dependencies{Access: resolver}}
	actor := identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "current"}
	page := evidence.ResponseReviewPage{Items: []evidence.ResponseReview{{ReviewerID: "original"}}}
	result := api.responseReviewsWithLabels(t.Context(), actor, "entity", page)
	if len(result.Items) != 1 || result.Items[0].ReviewerDisplayName != "Ada Okafor" || result.Items[0].ReviewerID != "original" {
		t.Fatalf("history attribution=%+v", result.Items)
	}
	if len(resolver.calls) != 1 || resolver.calls[0] != "tenant:entity:original" {
		t.Fatalf("lookup scope=%v", resolver.calls)
	}
}
