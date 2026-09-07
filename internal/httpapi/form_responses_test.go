package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
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

func TestCompletedResponseSummaryHTTPUsesCurrentAssessmentReadRoute(t *testing.T) {
	fixture := newReviewHTTPFixture(t)
	reviews := thirdparty.NewAssessmentReviewService(fixture.base.service, fixture.base.repository, fixture.base.evidence, nil)
	fixture.base.distributions.ConfigureDocumentContexts(thirdparty.DocumentContextReader{Assessments: reviews})
	api := &API{deps: Dependencies{FormDistributions: fixture.base.distributions}}
	now := time.Now()
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "summary-reviewer", Kind: "PERSON", AuthenticationMethod: "TEST", AssuranceLevel: "HIGH", SessionID: "session", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}
	var revisionID string
	for _, allowed := range []bool{false, true, false} {
		rules := []authority.Rule{}
		if allowed {
			rules = append(rules, authority.Rule{ID: "response-review", TenantID: "bank", LegalEntityID: "entity-a", ObjectType: "THIRD_PARTY_ASSESSMENT", ObjectID: fixture.assessment.ID, Responsibility: authority.ResponsibilityReviewer, DecisionType: thirdparty.AssessmentReviewCommand, MinMateriality: 3, Principal: authority.Principal{ID: actor.PrincipalID, Kind: "PERSON"}, Priority: 1})
		}
		reviews.ConfigureAuthority(authority.NewResolver("summary-test", rules))
		r := httptest.NewRequest(http.MethodGet, "/api/v1/forms/responses?limit=1&principal_id=verified-owner", nil).WithContext(identity.WithActor(context.Background(), actor))
		w := httptest.NewRecorder()
		api.listCompletedFormResponses(w, r)
		var page evidence.CompletedResponsePage
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil {
			t.Fatalf("summary HTTP: %d %s", w.Code, w.Body.String())
		}
		if !allowed && (len(page.Items) != 0 || page.NextCursor != "") {
			t.Fatalf("denied summary metadata: %s", w.Body.String())
		}
		if allowed {
			if len(page.Items) != 1 {
				t.Fatalf("authorized route omitted response: %s", w.Body.String())
			}
			revisionID = page.Items[0].ID
		}
		if revisionID != "" {
			r = httptest.NewRequest(http.MethodGet, "/api/v1/forms/responses/"+revisionID, nil).WithContext(identity.WithActor(context.Background(), actor))
			r.SetPathValue("revision_id", revisionID)
			w = httptest.NewRecorder()
			api.getCompletedFormResponse(w, r)
			if allowed && w.Code != 200 || !allowed && w.Code != 404 {
				t.Fatalf("exact response HTTP: %d %s", w.Code, w.Body.String())
			}
		}
	}
}
