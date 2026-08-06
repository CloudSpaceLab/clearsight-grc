package httpapi

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestDocumentImportHandlersBindVerifiedActorAndReview(t *testing.T) {
	service := documentimport.NewService(documentimport.NewMemoryRepository(), evidence.NewMemoryObjectStore())
	api := &API{deps: Dependencies{DocumentImports: service, MaxArtifactBytes: 1 << 20}}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "notice.md")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("# Records\n\nThe bank must retain records for five years."))
	_ = writer.WriteField("purpose", "Assess a regulatory notice")
	_ = writer.WriteField("source_type", "REGULATORY")
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/document-imports", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank-demo", PrincipalID: "reviewer-1", LegalEntityID: "bank-ng"}))
	response := httptest.NewRecorder()
	api.createDocumentImport(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var document documentimport.Document
	if err := json.NewDecoder(response.Body).Decode(&document); err != nil {
		t.Fatal(err)
	}
	if document.TenantID != "bank-demo" || document.CreatedBy != "reviewer-1" || len(document.Proposals) == 0 {
		t.Fatalf("import was not actor-bound or analysed: %#v", document)
	}

	reviewBody := bytes.NewBufferString(`{"status":"ACCEPTED","expected_version":1}`)
	reviewRequest := httptest.NewRequest(http.MethodPost, "/api/v1/document-imports/"+document.ID+"/proposals/"+document.Proposals[0].ID+"/review", reviewBody)
	reviewRequest.SetPathValue("id", document.ID)
	reviewRequest.SetPathValue("proposal_id", document.Proposals[0].ID)
	reviewRequest = reviewRequest.WithContext(identity.WithActor(reviewRequest.Context(), identity.Actor{TenantID: "bank-demo", PrincipalID: "reviewer-2", LegalEntityID: "bank-ng"}))
	reviewResponse := httptest.NewRecorder()
	api.reviewDocumentProposal(reviewResponse, reviewRequest)
	if reviewResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", reviewResponse.Code, reviewResponse.Body.String())
	}
	var reviewed documentimport.Document
	if err := json.NewDecoder(reviewResponse.Body).Decode(&reviewed); err != nil {
		t.Fatal(err)
	}
	if reviewed.Proposals[0].Status != documentimport.ProposalAccepted || reviewed.Proposals[0].ReviewedBy != "reviewer-2" {
		t.Fatalf("proposal review was not actor-bound: %#v", reviewed.Proposals[0])
	}

	otherRequest := httptest.NewRequest(http.MethodGet, "/api/v1/document-imports/"+document.ID, nil)
	otherRequest.SetPathValue("id", document.ID)
	otherRequest = otherRequest.WithContext(identity.WithActor(otherRequest.Context(), identity.Actor{TenantID: "other-bank", PrincipalID: "reviewer-x", LegalEntityID: "other-entity"}))
	otherResponse := httptest.NewRecorder()
	api.getDocumentImport(otherResponse, otherRequest)
	if otherResponse.Code != http.StatusNotFound {
		t.Fatalf("expected tenant-isolated 404, got %d: %s", otherResponse.Code, otherResponse.Body.String())
	}
}
