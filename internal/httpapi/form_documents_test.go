package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type documentContentStore struct {
	*evidence.MemoryDistributionStore
	value evidence.DocumentOccurrence
	query evidence.DocumentQuery
}

func (s *documentContentStore) ListDocuments(_ context.Context, q evidence.DocumentQuery) (evidence.DocumentPage, error) {
	s.query = q
	return evidence.DocumentPage{Items: []evidence.DocumentOccurrence{s.value}}, nil
}

func TestDocumentContentDeliversVerifiedBytesAndUsesUploadRequest(t *testing.T) {
	ctx := context.Background()
	store := evidence.NewMemoryObjectStore()
	repo := evidence.NewMemoryRepository(nil, []evidence.Request{{ID: "contributor-request", TenantID: "tenant", LegalEntityID: "entity", Status: evidence.RequestReady, Deadline: time.Now().Add(time.Hour)}})
	content := []byte("%PDF-1.7\nverified report")
	info, err := store.Put(ctx, "private-upload-key", bytes.NewReader(content), 1024)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := repo.CreateArtifact(ctx, evidence.Artifact{ID: "artifact", TenantID: "tenant", RequestID: "contributor-request", FileName: "report.pdf", MediaType: "application/pdf", SizeBytes: info.SizeBytes, SHA256: info.SHA256, StorageKey: info.Key, Status: evidence.ArtifactAvailable})
	if err != nil {
		t.Fatal(err)
	}
	documents := &documentContentStore{value: evidence.DocumentOccurrence{ID: "occurrence", ArtifactID: artifact.ID, ArtifactRequestID: artifact.RequestID, RequestID: "submission-request", SubmissionID: "submission", FieldID: "policy", SHA256: info.SHA256, SizeBytes: info.SizeBytes, ArtifactStatus: evidence.ArtifactAvailable}}
	api := &API{deps: Dependencies{FormDistributions: evidence.NewDistributionService(documents), Evidence: evidence.NewService(repo, store)}}
	actor := identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reader", Kind: "PERSON", AuthenticationMethod: "TEST", AssuranceLevel: "HIGH", SessionID: "session", IssuedAt: time.Now().Add(-time.Minute), ExpiresAt: time.Now().Add(time.Hour)}
	for _, download := range []string{"", "?download=true"} {
		r := httptest.NewRequest(http.MethodGet, "/content"+download, nil).WithContext(identity.WithActor(ctx, actor))
		r.SetPathValue("submission_id", "submission")
		r.SetPathValue("field_id", "policy")
		r.SetPathValue("artifact_id", "artifact")
		w := httptest.NewRecorder()
		api.openFormDocument(w, r)
		if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), content) || w.Header().Get("Content-Type") != "application/pdf" || w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("content status=%d headers=%v body=%q", w.Code, w.Header(), w.Body.String())
		}
		want := "inline"
		if download != "" {
			want = "attachment"
		}
		if !strings.HasPrefix(w.Header().Get("Content-Disposition"), want) {
			t.Fatal("wrong disposition")
		}
		if documents.query.TenantID != "tenant" || documents.query.PrincipalID != "reader" || documents.query.SubmissionID != "submission" || documents.query.FieldID != "policy" || documents.query.ArtifactID != "artifact" || documents.query.CurrentOnly {
			t.Fatalf("incorrect exact authority query %+v", documents.query)
		}
	}
}

func TestDocumentRoutesRequireVerifiedScopeAndClosedFilters(t *testing.T) {
	api := &API{deps: Dependencies{FormDistributions: evidence.NewDistributionService(evidence.NewMemoryDistributionStore(evidence.NewMemoryRepository(nil, nil), nil, nil))}}
	actor := identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reader", Kind: "PERSON", AuthenticationMethod: "TEST", AssuranceLevel: "HIGH", SessionID: "session", IssuedAt: time.Now().Add(-time.Minute), ExpiresAt: time.Now().Add(time.Hour)}
	for _, tt := range []struct {
		query  string
		auth   bool
		status int
	}{{"", false, 401}, {"", true, 200}, {"?file_kind=VIDEO", true, 422}, {"?limit=101", true, 422}, {"?current_only=maybe", true, 422}} {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/forms/documents"+tt.query, nil)
		if tt.auth {
			r = r.WithContext(identity.WithActor(context.Background(), actor))
		}
		w := httptest.NewRecorder()
		api.listFormDocuments(w, r)
		if w.Code != tt.status {
			t.Errorf("%+v: %d %s", tt, w.Code, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("missing protected cache control")
		}
	}
}
func TestDocumentContentCannotBeGuessedWithoutOccurrence(t *testing.T) {
	api := &API{deps: Dependencies{FormDistributions: evidence.NewDistributionService(evidence.NewMemoryDistributionStore(evidence.NewMemoryRepository(nil, nil), nil, nil)), Evidence: evidence.NewService(evidence.NewMemoryRepository(nil, nil), evidence.NewMemoryObjectStore())}}
	actor := identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reader", Kind: "PERSON", AuthenticationMethod: "TEST", AssuranceLevel: "HIGH", SessionID: "session", IssuedAt: time.Now().Add(-time.Minute), ExpiresAt: time.Now().Add(time.Hour)}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/forms/documents/sub/file/artifact/content", nil).WithContext(identity.WithActor(context.Background(), actor))
	r.SetPathValue("submission_id", "sub")
	r.SetPathValue("field_id", "file")
	r.SetPathValue("artifact_id", "artifact")
	w := httptest.NewRecorder()
	api.openFormDocument(w, r)
	if w.Code != 404 || strings.Contains(w.Body.String(), "storage_key") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}
