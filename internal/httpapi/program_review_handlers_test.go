package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestProgramReviewAcceptanceUnavailableAndNotAdvertised(t *testing.T) {
	repo := continuity.NewMemoryRepository()
	api := &API{deps: Dependencies{Continuity: continuity.NewService(repo), Authority: &assignmentAuthorityStub{}}}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "reviewer", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/programs/p/reviews", strings.NewReader(`{"expected_program_version":1,"expected_projection_version":1}`))
	request.SetPathValue("id", "p")
	request = request.WithContext(identity.WithActor(request.Context(), actor))
	response := httptest.NewRecorder()
	api.acceptProgramReview(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("acceptance must be unavailable: status=%d body=%s", response.Code, response.Body.String())
	}
	op, _ := api.resolveProgramOperation(request.Context(), actor, continuity.Program{ID: "p"}, programOperationSpec{Command: "program.review.accept", Responsibility: authority.ResponsibilityReviewer}, programOperationResolution{primary: authority.ResolveOutcome{Resolution: authority.Resolution{Principal: authority.Principal{ID: actor.PrincipalID}}}})
	if op.CanAct || strings.TrimSpace(op.Reason) == "" {
		t.Fatalf("unavailable decision advertised: %+v", op)
	}
}

func TestProgramReviewListFailsClosedWhenStoreCannotEnforceScope(t *testing.T) {
	store := evidence.NewMemoryDistributionStore(evidence.NewMemoryRepository(nil, nil), nil, nil)
	api := &API{deps: Dependencies{FormDistributions: evidence.NewDistributionService(store)}}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "reviewer", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/forms/program-reviews", nil)
	request = request.WithContext(identity.WithActor(request.Context(), actor))
	response := httptest.NewRecorder()
	api.listProgramResponseReviews(response, request)
	if response.Code != http.StatusServiceUnavailable || response.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("status=%d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
	}
}

func TestProgramReviewListRemainsUnregisteredUntilAuthorityParityIsProven(t *testing.T) {
	for _, route := range (&API{}).productionRoutes() {
		if route.Path == "/api/v1/forms/program-reviews" {
			t.Fatal("unproven Program review discovery route must not be registered")
		}
	}
}

func TestProgramReviewListRejectsInvalidPageSize(t *testing.T) {
	store := evidence.NewMemoryDistributionStore(evidence.NewMemoryRepository(nil, nil), nil, nil)
	api := &API{deps: Dependencies{FormDistributions: evidence.NewDistributionService(store)}}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "reviewer", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}
	for _, value := range []string{"0", "101", "invalid"} {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/forms/program-reviews?limit="+value, nil)
		request = request.WithContext(identity.WithActor(request.Context(), actor))
		response := httptest.NewRecorder()
		api.listProgramResponseReviews(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("limit=%s status=%d", value, response.Code)
		}
	}
}

func TestProgramReviewListUsesRecordedSubmitterName(t *testing.T) {
	actor := identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reviewer"}
	resolver := &assessmentLabelResolver{resolution: access.Resolution{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "submitter", DisplayName: "Ada Okafor"}}
	api := &API{deps: Dependencies{Access: resolver}}
	page := evidence.ProgramReviewPage{Coverage: "PROGRAM_RESPONSE_REVIEWS", Items: []evidence.CompletedResponseSummary{{ID: "revision", SubmittedBy: "submitter"}}}
	result := api.programReviewPageWithLabels(context.Background(), actor, page)
	if len(result) != 1 || result[0]["submitted_by_display_name"] != "Ada Okafor" || result[0]["submitted_by"] != "submitter" {
		t.Fatalf("incorrect attribution: %+v", result)
	}
}
