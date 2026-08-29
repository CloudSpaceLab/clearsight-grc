package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

type formProposalHTTPDocuments struct {
	documents map[string]documentimport.Document
}

func (r formProposalHTTPDocuments) Get(_ context.Context, tenantID, documentID string) (documentimport.Document, error) {
	value, ok := r.documents[documentID]
	if !ok || value.TenantID != tenantID {
		return documentimport.Document{}, documentimport.ErrNotFound
	}
	return value, nil
}

func TestFormProposalRoutesAreRegisteredAndClassified(t *testing.T) {
	want := map[string]routeClass{
		"POST /api/v1/document-imports/{id}/form-template-proposals": routeAuthenticatedWrite,
		"GET /api/v1/forms/proposals/{id}":                           routeAuthenticatedRead,
		"POST /api/v1/forms/proposals/{id}/accept":                   routeAuthenticatedWrite,
		"POST /api/v1/forms/proposals/{id}/reject":                   routeAuthenticatedWrite,
	}
	for _, route := range (&API{}).productionRoutes() {
		key := route.Method + " " + route.Path
		class, ok := want[key]
		if !ok {
			continue
		}
		if route.Class != class {
			t.Fatalf("%s class = %s, want %s", key, route.Class, class)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("missing form proposal routes: %#v", want)
	}
}

func TestFormProposalHTTPCreateGetAcceptAndIdempotentRetry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	guard, err := commandauth.New(nil, commandauth.ModeOff, logger)
	if err != nil {
		t.Fatal(err)
	}
	forms := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	forms.ConfigureCommandGuard(guard)
	documents := formProposalHTTPDocuments{documents: map[string]documentimport.Document{
		"doc-1": {
			ID: "doc-1", TenantID: "bank-a", LegalEntityID: "entity-a", Version: 3,
			FileName: "vendor-questionnaire.docx", Purpose: "Collect vendor profile data.",
			SHA256: strings.Repeat("a", 64), ExtractionStatus: documentimport.ExtractionExtracted,
			ParserVersion: "DOCX_XML_STREAM_V3", AdapterVersion: "DOCX_STRUCTURE_ADAPTER_V1",
			Elements: []documentimport.ExtractedElement{
				{Kind: documentimport.ElementHeading, Text: "Vendor profile", Anchor: documentimport.SourceAnchor{Paragraph: "paragraph-1"}},
				{Kind: documentimport.ElementFormControl, Anchor: documentimport.SourceAnchor{Paragraph: "paragraph-2"}, Control: &documentimport.FormControl{Kind: "TEXT", Label: "Legal name"}},
			},
		},
	}}
	proposals := monitoring.NewFormProposalService(monitoring.NewMemoryFormProposalStore(), documents, forms)
	handler := New(Dependencies{
		Logger:        logger,
		Identity:      identity.NewDevelopmentAuthenticator("bank-a", "maker-a", "entity-a"),
		CommandGuard:  guard,
		Monitoring:    forms,
		FormProposals: proposals,
	})

	create := httptest.NewRecorder()
	handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/v1/document-imports/doc-1/form-template-proposals", bytes.NewBufferString(`{"expected_document_version":3}`)))
	if create.Code != http.StatusAccepted {
		t.Fatalf("create returned %d: %s", create.Code, create.Body.String())
	}
	var proposal monitoring.FormTemplateProposal
	if err := json.Unmarshal(create.Body.Bytes(), &proposal); err != nil {
		t.Fatal(err)
	}
	if proposal.Status != monitoring.FormProposalReviewRequired || proposal.Version != 2 || len(proposal.FieldChanges) != 1 {
		t.Fatalf("unexpected proposal: %#v", proposal)
	}

	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/forms/proposals/"+proposal.ID, nil))
	if get.Code != http.StatusOK || !bytes.Contains(get.Body.Bytes(), []byte(proposal.ID)) {
		t.Fatalf("get returned %d: %s", get.Code, get.Body.String())
	}

	acceptPayload, err := json.Marshal(monitoring.AcceptFormProposalInput{ExpectedVersion: proposal.Version, ChangeIDs: []string{proposal.FieldChanges[0].ID}})
	if err != nil {
		t.Fatal(err)
	}
	accept := httptest.NewRecorder()
	handler.ServeHTTP(accept, httptest.NewRequest(http.MethodPost, "/api/v1/forms/proposals/"+proposal.ID+"/accept", bytes.NewReader(acceptPayload)))
	if accept.Code != http.StatusOK {
		t.Fatalf("accept returned %d: %s", accept.Code, accept.Body.String())
	}
	var accepted monitoring.FormTemplateProposal
	if err := json.Unmarshal(accept.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.Status != monitoring.FormProposalAccepted || accepted.ResultTemplateID == "" || accepted.ResultTemplateVersion != 1 || !slices.Equal(accepted.AcceptedChangeIDs, []string{proposal.FieldChanges[0].ID}) {
		t.Fatalf("unexpected accepted proposal: %#v", accepted)
	}

	// Retrying the exact command is idempotent even though the original expected
	// version is now stale; a different selected-change set is not treated as the
	// same command.
	retry := httptest.NewRecorder()
	handler.ServeHTTP(retry, httptest.NewRequest(http.MethodPost, "/api/v1/forms/proposals/"+proposal.ID+"/accept", bytes.NewReader(acceptPayload)))
	if retry.Code != http.StatusOK {
		t.Fatalf("idempotent retry returned %d: %s", retry.Code, retry.Body.String())
	}

	draft := httptest.NewRecorder()
	handler.ServeHTTP(draft, httptest.NewRequest(http.MethodGet, "/api/v1/forms/templates/"+accepted.ResultTemplateID+"/revisions/1", nil))
	if draft.Code != http.StatusOK || !bytes.Contains(draft.Body.Bytes(), []byte(`"status":"DRAFT"`)) {
		t.Fatalf("resulting draft returned %d: %s", draft.Code, draft.Body.String())
	}
}
