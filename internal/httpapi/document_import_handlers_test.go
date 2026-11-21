package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

// Explicit test policy; the production memory store intentionally has no
// actor-scoped reader and fails closed.
type authorizedImportTestRepo struct {
	*documentimport.MemoryRepository
}

func (r *authorizedImportTestRepo) ListForActor(ctx context.Context, tenant, entity, principal string, limit int, _, _ time.Time) ([]documentimport.DocumentSummary, error) {
	if principal != "reviewer-1" && principal != "maker-a" {
		return nil, documentimport.ErrNotFound
	}
	return r.ListScoped(ctx, tenant, entity, limit)
}
func (r *authorizedImportTestRepo) GetForActor(ctx context.Context, tenant, entity, principal, id string, _, _ time.Time) (documentimport.Document, error) {
	if principal != "reviewer-1" && principal != "maker-a" {
		return documentimport.Document{}, documentimport.ErrNotFound
	}
	return r.GetScoped(ctx, tenant, entity, id)
}

func TestDocumentImportHandlersBindVerifiedActorAndReview(t *testing.T) {
	service := documentimport.NewService(&authorizedImportTestRepo{documentimport.NewMemoryRepository()}, evidence.NewMemoryObjectStore())
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
	if bytes.Contains(response.Body.Bytes(), []byte(`"storage_key"`)) || response.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("import response exposes storage location or can be cached: %v %s", response.Header(), response.Body.String())
	}
	var document documentimport.Document
	if err := json.NewDecoder(response.Body).Decode(&document); err != nil {
		t.Fatal(err)
	}
	if document.TenantID != "bank-demo" || document.CreatedBy != "reviewer-1" || len(document.Proposals) == 0 {
		t.Fatalf("import was not actor-bound or analysed: %#v", document)
	}
	stored, err := service.GetVisible(request.Context(), "bank-demo", "bank-ng", document.ID)
	if err != nil || stored.StorageKey == "" {
		t.Fatalf("import must retain its internal storage key: %v", err)
	}
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/document-imports", nil)
	listRequest = listRequest.WithContext(identity.WithActor(listRequest.Context(), identity.Actor{TenantID: "bank-demo", PrincipalID: "reviewer-1", LegalEntityID: "bank-ng", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}))
	listResponse := httptest.NewRecorder()
	api.listDocumentImports(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || listResponse.Header().Get("Cache-Control") != "private, no-store" || bytes.Contains(listResponse.Body.Bytes(), []byte(`"storage_key"`)) {
		t.Fatalf("list response: %d %v %s", listResponse.Code, listResponse.Header(), listResponse.Body.String())
	}
	readRequest := httptest.NewRequest(http.MethodGet, "/api/v1/document-imports/"+document.ID, nil)
	readRequest.SetPathValue("id", document.ID)
	readRequest = readRequest.WithContext(identity.WithActor(readRequest.Context(), identity.Actor{TenantID: "bank-demo", PrincipalID: "reviewer-1", LegalEntityID: "bank-ng", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}))
	readResponse := httptest.NewRecorder()
	api.getDocumentImport(readResponse, readRequest)
	if readResponse.Code != http.StatusOK || bytes.Contains(readResponse.Body.Bytes(), []byte(`"storage_key"`)) || readResponse.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("detail response: %d %v %s", readResponse.Code, readResponse.Header(), readResponse.Body.String())
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
	if bytes.Contains(reviewResponse.Body.Bytes(), []byte(`"storage_key"`)) || reviewResponse.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("review response exposes storage location or can be cached: %v %s", reviewResponse.Header(), reviewResponse.Body.String())
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
