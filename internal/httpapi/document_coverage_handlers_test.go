package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentcoverage"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestDocumentCoverageHandlersExposeReviewAndRecompare(t *testing.T) {
	api, document, initial := coverageHandlerFixture(t)

	getRequest := coverageRequest(http.MethodGet, "/api/v1/document-imports/"+document.ID+"/coverage", nil, document.ID, "reviewer-1")
	getResponse := httptest.NewRecorder()
	api.getDocumentCoverage(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getResponse.Code, getResponse.Body.String())
	}
	var view documentcoverage.View
	if err := json.NewDecoder(getResponse.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if view.Status != documentcoverage.ViewReady || len(view.Candidates) != 1 || view.Candidates[0].Anchor.Page != 7 {
		t.Fatalf("coverage response lost source-backed detail: %#v", view)
	}

	invalidBody := bytes.NewBufferString(`{"expected_version":1,"decisions":[{"candidate_id":"candidate-1","decision":"NOT_APPLICABLE"}]}`)
	invalidRequest := coverageRequest(http.MethodPost, "/api/v1/document-imports/"+document.ID+"/coverage/review", invalidBody, document.ID, "reviewer-1")
	invalidResponse := httptest.NewRecorder()
	api.reviewDocumentCoverage(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected invalid review 422, got %d: %s", invalidResponse.Code, invalidResponse.Body.String())
	}

	validBody := bytes.NewBufferString(`{"expected_version":1,"decisions":[{"candidate_id":"candidate-1","decision":"NOT_APPLICABLE","reason":"The legal entity does not perform this regulated activity."}]}`)
	validRequest := coverageRequest(http.MethodPost, "/api/v1/document-imports/"+document.ID+"/coverage/review", validBody, document.ID, "reviewer-1")
	validResponse := httptest.NewRecorder()
	api.reviewDocumentCoverage(validResponse, validRequest)
	if validResponse.Code != http.StatusOK {
		t.Fatalf("expected valid review 200, got %d: %s", validResponse.Code, validResponse.Body.String())
	}

	staleBody := bytes.NewBufferString(`{"expected_version":1,"decisions":[{"candidate_id":"candidate-1","decision":"REJECT_MATCH"}]}`)
	staleRequest := coverageRequest(http.MethodPost, "/api/v1/document-imports/"+document.ID+"/coverage/review", staleBody, document.ID, "reviewer-1")
	staleResponse := httptest.NewRecorder()
	api.reviewDocumentCoverage(staleResponse, staleRequest)
	if staleResponse.Code != http.StatusConflict {
		t.Fatalf("expected stale review 409, got %d: %s", staleResponse.Code, staleResponse.Body.String())
	}

	recompareRequest := coverageRequest(http.MethodPost, "/api/v1/document-imports/"+document.ID+"/coverage/recompare", bytes.NewBufferString(`{}`), document.ID, "reviewer-1")
	recompareResponse := httptest.NewRecorder()
	api.recompareDocumentCoverage(recompareResponse, recompareRequest)
	if recompareResponse.Code != http.StatusAccepted {
		t.Fatalf("expected recompare 202, got %d: %s", recompareResponse.Code, recompareResponse.Body.String())
	}
	_ = initial
}

func TestDocumentCoverageApplyCreatesGovernedDraft(t *testing.T) {
	api, document, initial := coverageHandlerFixture(t)
	if len(initial.Suggestions) != 1 || initial.Suggestions[0].Type != documentcoverage.SuggestionCreateProgram {
		t.Fatalf("unexpected fixture suggestion: %#v", initial.Suggestions)
	}
	body := bytes.NewBufferString(`{"expected_version":1}`)
	request := coverageRequest(http.MethodPost, "/api/v1/document-imports/"+document.ID+"/coverage/suggestions/"+initial.Suggestions[0].ID+"/apply", body, document.ID, "reviewer-1")
	request.SetPathValue("suggestion_id", initial.Suggestions[0].ID)
	response := httptest.NewRecorder()
	api.applyDocumentCoverageSuggestion(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var result documentcoverage.ApplySuggestionResult
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.ObjectType != "PROGRAM" || result.ObjectID == "" || result.Assessment.Suggestions[0].Status != documentcoverage.SuggestionApplied {
		t.Fatalf("unexpected apply result: %#v", result)
	}
	program, err := api.deps.Continuity.GetProgram(context.Background(), document.TenantID, result.ObjectID)
	if err != nil {
		t.Fatal(err)
	}
	if program.Program.Status != continuity.ProgramDraft || program.Program.LegalEntityID != document.LegalEntityID {
		t.Fatalf("suggestion must create a scoped draft: %#v", program.Program)
	}
}

func coverageHandlerFixture(t *testing.T) (*API, documentimport.Document, documentcoverage.Assessment) {
	t.Helper()
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	statement := "A data controller in Nigeria must retain processing records annually under section 41."
	obligation := documentimport.ParseObligation(statement, "REQUIREMENT_CANDIDATE")
	document := documentimport.Document{
		ID: "document-coverage-http", TenantID: "bank-demo", LegalEntityID: "bank-ng", FileName: "ndpc-guidance.pdf",
		SHA256:           "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ExtractionStatus: documentimport.ExtractionExtracted, AnalysisStatus: documentimport.AnalysisReviewRequired,
		Proposals: []documentimport.Proposal{{
			ID: "candidate-1", Kind: "REQUIREMENT_CANDIDATE", Statement: statement,
			Anchor: documentimport.Anchor{SectionID: "page-7", Quote: statement, Page: 7}, Obligation: &obligation,
		}}, CreatedAt: now, UpdatedAt: now, Version: 2,
	}
	documents := documentimport.NewMemoryRepository()
	if _, err := documents.Create(context.Background(), document); err != nil {
		t.Fatal(err)
	}
	continuityService := continuity.NewService(continuity.NewMemoryRepository())
	coverageService := documentcoverage.NewService(documentcoverage.NewMemoryRepository(), documents, continuityService)
	initial, err := coverageService.Process(context.Background(), document.TenantID, document.ID)
	if err != nil {
		t.Fatal(err)
	}
	return &API{deps: Dependencies{Coverage: coverageService, Continuity: continuityService}}, document, initial
}

func coverageRequest(method, target string, body *bytes.Buffer, documentID, principalID string) *http.Request {
	var request *http.Request
	if body == nil {
		request = httptest.NewRequest(method, target, nil)
	} else {
		request = httptest.NewRequest(method, target, body)
		request.Header.Set("Content-Type", "application/json")
	}
	request.SetPathValue("id", documentID)
	actor := identity.Actor{TenantID: "bank-demo", PrincipalID: principalID, LegalEntityID: "bank-ng"}
	return request.WithContext(identity.WithActor(request.Context(), actor))
}
