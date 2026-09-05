package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type evidenceGetRequestFailureRepository struct {
	evidence.Repository
	err error
}

func (r evidenceGetRequestFailureRepository) GetRequest(context.Context, string, string) (evidence.Request, error) {
	return evidence.Request{}, r.err
}

func TestEvidenceReviewerCanListAndReadSubmittedResponseOnly(t *testing.T) {
	now := time.Now().UTC()
	requestValue := evidence.Request{
		ID: "request-review", TenantID: "bank", LegalEntityID: "entity-a", SubjectType: "PROGRAM", SubjectID: "program-a",
		Title: "Monthly encryption review", Status: evidence.RequestReady, CreatedBy: "owner", Deadline: now.Add(time.Hour), Version: 1,
		KnownFacts: map[string]string{"reviewer": "auditor"}, Fields: []evidence.Field{{ID: "reference", Label: "Evidence reference", Type: "short_text"}},
		AudienceType: "INTERNAL", Recipient: evidence.Recipient{Type: evidence.RecipientInternalPrincipal, PrincipalID: "respondent", State: evidence.RecipientStateAssigned},
	}
	repo := evidence.NewMemoryRepository(nil, []evidence.Request{requestValue})
	if _, err := repo.Submit(t.Context(), evidence.Submission{ID: "submission-a", TenantID: "bank", RequestID: requestValue.ID, SubmittedBy: "respondent", Channel: "INTERNAL", Answers: formcontract.TextAnswers(map[string]string{"reference": "ENC-2026-09"}), ExpectedVersion: 1, SubmittedAt: now}); err != nil {
		t.Fatal(err)
	}
	service := evidence.NewService(repo, evidence.NewMemoryObjectStore())
	handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "auditor", "entity-a"), Evidence: service})

	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests?limit=50", nil))
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), requestValue.ID) {
		t.Fatalf("reviewer list = %d %s", listResponse.Code, listResponse.Body.String())
	}
	reviewResponse := httptest.NewRecorder()
	handler.ServeHTTP(reviewResponse, httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/"+requestValue.ID+"/review-submission", nil))
	if reviewResponse.Code != http.StatusOK || !strings.Contains(reviewResponse.Body.String(), "ENC-2026-09") {
		t.Fatalf("review response = %d %s", reviewResponse.Code, reviewResponse.Body.String())
	}

	deniedAPI := &API{deps: Dependencies{Evidence: service}}
	deniedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/"+requestValue.ID+"/review-submission", nil).WithContext(identity.WithActor(t.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "auditor-other"}))
	deniedRequest.SetPathValue("id", requestValue.ID)
	deniedResponse := httptest.NewRecorder()
	deniedAPI.getEvidenceReviewSubmission(deniedResponse, deniedRequest)
	if deniedResponse.Code != http.StatusNotFound {
		t.Fatalf("unassigned reviewer status = %d body=%s", deniedResponse.Code, deniedResponse.Body.String())
	}
}

func TestEvidenceReviewerLoadPreservesRepositoryFailure(t *testing.T) {
	loadErr := errors.New("request store unavailable")
	base := evidence.NewMemoryRepository(nil, nil)
	service := evidence.NewService(evidenceGetRequestFailureRepository{Repository: base, err: loadErr}, evidence.NewMemoryObjectStore())
	api := &API{deps: Dependencies{Evidence: service}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/request-review/review-submission", nil).WithContext(identity.WithActor(t.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "auditor"}))
	request.SetPathValue("id", "request-review")
	response := httptest.NewRecorder()

	api.getEvidenceReviewSubmission(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("repository failure status = %d body=%s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "submission_failed") {
		t.Fatalf("repository failure response = %s", response.Body.String())
	}
}
