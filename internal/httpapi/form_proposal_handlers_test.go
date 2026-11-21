package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

type formProposalHTTPDocuments struct {
	documents map[string]documentimport.Document
}

func TestFormProposalConflictResponsesIdentifyRecovery(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		code    string
		message string
	}{
		{"source", monitoring.ErrFormProposalSourceChanged, "form_proposal_source_changed", "The source document changed. Reload the import and create a new proposal."},
		{"state", monitoring.ErrFormProposalState, "form_proposal_state_conflict", "This proposal cannot create a draft in its current state. Reload the import."},
		{"version", monitoring.ErrConflict, "form_proposal_conflict", monitoring.ErrConflict.Error()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeFormProposalError(w, tc.err)
			var body struct {
				Error   string `json:"error"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != http.StatusConflict || body.Error != tc.code || body.Message != tc.message {
				t.Fatalf("conflict response = %d %s", w.Code, w.Body.String())
			}
		})
	}
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
		"POST /api/v1/document-imports/{id}/form-template-proposals":         routeAuthenticatedWrite,
		"POST /api/v1/forms/proposals/ai":                                    routeAuthenticatedWrite,
		"POST /api/v1/forms/templates/{id}/revisions/{version}/ai-proposals": routeAuthenticatedWrite,
		"GET /api/v1/forms/proposals/{id}":                                   routeAuthenticatedRead,
		"POST /api/v1/forms/proposals/{id}/accept":                           routeAuthenticatedWrite,
		"POST /api/v1/forms/proposals/{id}/reject":                           routeAuthenticatedWrite,
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

func TestAIFormProposalRouteFailsCleanlyWhenGatewayIsNotConfigured(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	guard, err := commandauth.New(nil, commandauth.ModeOff, logger)
	if err != nil {
		t.Fatal(err)
	}
	forms := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	forms.ConfigureCommandGuard(guard)
	proposals := monitoring.NewFormProposalService(monitoring.NewMemoryFormProposalStore(), formProposalHTTPDocuments{}, forms)
	handler := New(Dependencies{
		Logger: logger, Identity: identity.NewDevelopmentAuthenticator("bank-a", "maker-a", "entity-a"),
		CommandGuard: guard, Monitoring: forms, FormProposals: proposals,
	})

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/forms/proposals/ai", bytes.NewBufferString(`{"objective":"Draft a vendor onboarding questionnaire."}`)))
	if response.Code != http.StatusServiceUnavailable || !bytes.Contains(response.Body.Bytes(), []byte(`"error":"form_ai_unavailable"`)) {
		t.Fatalf("unconfigured AI route returned %d: %s", response.Code, response.Body.String())
	}
}

func TestDocumentFormProposalHTTPReadsAndWritesFailClosed(t *testing.T) {
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
	readRepo := &authorizedImportTestRepo{documentimport.NewMemoryRepository()}
	if _, err := readRepo.Create(context.Background(), documents.documents["doc-1"]); err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger:          logger,
		Identity:        identity.NewDevelopmentAuthenticator("bank-a", "maker-a", "entity-a"),
		CommandGuard:    guard,
		Monitoring:      forms,
		FormProposals:   proposals,
		DocumentImports: documentimport.NewService(readRepo, nil),
	})

	create := httptest.NewRecorder()
	handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/v1/document-imports/doc-1/form-template-proposals", bytes.NewBufferString(`{"expected_document_version":3}`)))
	if create.Code != http.StatusServiceUnavailable {
		t.Fatalf("source-backed create must fail closed: %d %s", create.Code, create.Body.String())
	}
	actor := identity.Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "maker-a", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}
	ctx := identity.WithActor(context.Background(), actor)
	proposal, err := proposals.RequestFromDocument(ctx, "doc-1", monitoring.RequestDocumentFormProposalInput{ExpectedDocumentVersion: 3})
	if err != nil {
		t.Fatal(err)
	}
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/forms/proposals/"+proposal.ID, nil))
	if get.Code != http.StatusServiceUnavailable || bytes.Contains(get.Body.Bytes(), []byte(proposal.ID)) {
		t.Fatalf("source-backed read must fail closed: %d %s", get.Code, get.Body.String())
	}
	accept := httptest.NewRecorder()
	handler.ServeHTTP(accept, httptest.NewRequest(http.MethodPost, "/api/v1/forms/proposals/"+proposal.ID+"/accept", bytes.NewBufferString(`{}`)))
	if accept.Code != http.StatusServiceUnavailable {
		t.Fatalf("source-backed accept must fail closed: %d %s", accept.Code, accept.Body.String())
	}
}

func TestSourceBackedAIProposalIsUnavailableBeforeGateway(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	forms := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	proposals := monitoring.NewFormProposalService(monitoring.NewMemoryFormProposalStore(), formProposalHTTPDocuments{}, forms)
	handler := New(Dependencies{Logger: logger, Identity: identity.NewDevelopmentAuthenticator("bank-a", "maker-a", "entity-a"), FormProposals: proposals})
	for _, url := range []string{"/api/v1/forms/proposals/ai", "/api/v1/forms/templates/form-1/revisions/1/ai-proposals"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString(`{"objective":"Review","source_document_id":"private"}`)))
		if response.Code != http.StatusServiceUnavailable || !bytes.Contains(response.Body.Bytes(), []byte(`"document_action_unavailable"`)) {
			t.Fatalf("%s returned %d %s", url, response.Code, response.Body.String())
		}
	}
}
