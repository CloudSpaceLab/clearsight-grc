package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestListCompletedFormResponsesUsesVerifiedScopeAndRejectsInvalidFilters(t *testing.T) {
	store := evidence.NewMemoryDistributionStore(evidence.NewMemoryRepository(nil, nil), nil, nil)
	api := &API{deps: Dependencies{FormDistributions: evidence.NewDistributionService(store)}}
	actor := identity.Actor{
		TenantID: "tenant-a", LegalEntityID: "entity-a", PrincipalID: "principal-a", Kind: "PERSON",
		AuthenticationMethod: "TEST", AssuranceLevel: "HIGH", SessionID: "session-a",
		IssuedAt: time.Now().Add(-time.Minute), ExpiresAt: time.Now().Add(time.Hour),
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/forms/responses?legal_entity_id=entity-b&sort=CONCERN_DESC&limit=25", nil)
	request = request.WithContext(identity.WithActor(context.Background(), actor))
	response := httptest.NewRecorder()
	api.listCompletedFormResponses(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"items":[]`) {
		t.Fatalf("verified-scope list status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/forms/responses?raw_min=101", nil)
	request = request.WithContext(identity.WithActor(context.Background(), actor))
	response = httptest.NewRecorder()
	api.listCompletedFormResponses(response, request)
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "response_filters_invalid") {
		t.Fatalf("invalid filter status=%d body=%s", response.Code, response.Body.String())
	}
}

type scalarResponseStore struct {
	*evidence.MemoryDistributionStore
}

func (s *scalarResponseStore) GetCompletedResponse(context.Context, string, string, string, string) (evidence.CompletedResponseSummary, evidence.ResponseRevision, error) {
	return evidence.CompletedResponseSummary{SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "relationship", FormTemplateID: "form", FormTemplateVersion: 1}, evidence.ResponseRevision{ID: "revision", SubmissionID: "submission"}, nil
}

type scalarResponseAuthority struct{ allowed bool }

func (s *scalarResponseAuthority) ResolveDocumentContext(context.Context, evidence.DocumentQuery, evidence.Request, evidence.Submission) (evidence.DocumentContext, error) {
	if !s.allowed {
		return evidence.DocumentContext{}, evidence.ErrNotFound
	}
	return evidence.DocumentContext{WorkRequestID: "work"}, nil
}
func TestCompletedResponseScalarAnswersDenyWorkflowTargetEvenWithEmptyDocumentInventory(t *testing.T) {
	now := time.Now()
	repo := evidence.NewMemoryRepository(nil, []evidence.Request{{ID: "request", TenantID: "tenant", LegalEntityID: "entity", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: "relationship", FormTemplateID: "form", FormTemplateVersion: 1, Origin: evidence.RequestOrigin{Type: "THIRD_PARTY_WORK", ID: "work", Version: 1}, Fields: []evidence.Field{{ID: "secret", Label: "Control detail", Type: "text"}}, Status: evidence.RequestReady, Deadline: now.Add(time.Hour)}})
	if _, err := repo.Submit(context.Background(), evidence.Submission{ID: "submission", TenantID: "tenant", RequestID: "request", SubmittedAt: now, Answers: map[string]formcontract.AnswerValue{"secret": formcontract.TextAnswer("restricted control detail")}}); err != nil {
		t.Fatal(err)
	}
	store := &scalarResponseStore{MemoryDistributionStore: evidence.NewMemoryDistributionStore(repo, nil, nil)}
	authority := &scalarResponseAuthority{}
	evidence.NewDistributionService(store.MemoryDistributionStore).ConfigureDocumentContexts(authority)
	api := &API{deps: Dependencies{FormDistributions: evidence.NewDistributionService(store)}}
	actor := identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "owner", Kind: "PERSON", AuthenticationMethod: "TEST", AssuranceLevel: "HIGH", SessionID: "session", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}
	for _, allow := range []bool{false, true} {
		authority.allowed = allow
		r := httptest.NewRequest(http.MethodGet, "/api/v1/forms/responses/revision", nil).WithContext(identity.WithActor(context.Background(), actor))
		r.SetPathValue("revision_id", "revision")
		w := httptest.NewRecorder()
		api.getCompletedFormResponse(w, r)
		if !allow && (w.Code != 404 || strings.Contains(w.Body.String(), "restricted control detail")) {
			t.Fatalf("denied scalar response %d %s", w.Code, w.Body.String())
		}
		if allow && (w.Code != 200 || !strings.Contains(w.Body.String(), "restricted control detail")) {
			t.Fatalf("permitted scalar response %d %s", w.Code, w.Body.String())
		}
	}
}

func TestCompletedResponseRoutesAreAuthenticatedReads(t *testing.T) {
	api := &API{}
	want := map[string]bool{
		"/api/v1/forms/responses":               false,
		"/api/v1/forms/responses/{revision_id}": false,
	}
	for _, route := range api.formDistributionRoutes() {
		if _, ok := want[route.Path]; !ok {
			continue
		}
		want[route.Path] = route.Class == routeAuthenticatedRead && route.Method == http.MethodGet
	}
	for path, valid := range want {
		if !valid {
			t.Fatalf("completed response route %s is missing or not an authenticated read", path)
		}
	}
}
