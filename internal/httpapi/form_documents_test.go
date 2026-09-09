package httpapi

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/demodocuments"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type documentContentStore struct {
	*evidence.MemoryDistributionStore
	value  evidence.DocumentOccurrence
	query  evidence.DocumentQuery
	denied bool
}

func (s *documentContentStore) ListDocuments(_ context.Context, q evidence.DocumentQuery) (evidence.DocumentPage, error) {
	s.query = q
	if s.denied {
		return evidence.DocumentPage{Items: []evidence.DocumentOccurrence{}}, nil
	}
	return evidence.DocumentPage{Items: []evidence.DocumentOccurrence{s.value}}, nil
}

func TestDemoDocumentContentRequiresBothServerFlagsAndFreshAuthorizedOccurrence(t *testing.T) {
	for _, test := range []struct {
		name                  string
		capture, distribution bool
		change                string
		want                  int
	}{
		{"demo preview", true, true, "", 200}, {"demo download", true, true, "download", 200},
		{"default", false, false, "", 404}, {"capture flag only", true, false, "", 404}, {"list flag only", false, true, "", 404},
		{"forged capability", true, true, "unknown", 404}, {"changed bytes", true, true, "changed", 404},
		{"truncated bytes", true, true, "truncated", 404}, {"appended bytes", true, true, "appended", 404},
		{"missing occurrence", true, true, "denied", 404}, {"revoked occurrence", true, true, "revoked", 404},
		{"quarantined", true, true, "QUARANTINED", 404}, {"deleted", true, true, "DELETED", 404},
		{"missing identity", true, true, "identity", 401}, {"wrong tenant", true, true, "tenant", 404},
		{"demo acceptance ordinary file", true, true, "", 200}, {"demo acceptance download", true, true, "download", 200},
		{"demo acceptance disabled", false, false, "", 404}, {"demo acceptance capture only", true, false, "", 404}, {"demo acceptance list only", false, true, "", 404},
		{"demo acceptance quarantine", true, true, "QUARANTINED", 404}, {"demo acceptance deleted", true, true, "DELETED", 404},
		{"demo acceptance changed bytes", true, true, "changed", 404}, {"demo acceptance unauthorized", true, true, "denied", 404},
		{"demo acceptance missing identity", true, true, "identity", 401}, {"demo acceptance wrong tenant", true, true, "tenant", 404},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			file := demodocuments.Files()[0]
			content, _ := demodocuments.Read(file.Name)
			allowUnscanned := strings.HasPrefix(test.name, "demo acceptance")
			if allowUnscanned {
				file.Name = "ordinary-upload.pdf"
			}
			store := evidence.NewMemoryObjectStore()
			repo := evidence.NewMemoryRepository(nil, []evidence.Request{{ID: "upload-request", TenantID: "tenant", LegalEntityID: "entity", Status: evidence.RequestReady, Deadline: time.Now().Add(time.Hour)}})
			info, err := store.Put(ctx, "private-demo-key", bytes.NewReader(content), 20<<20)
			if err != nil {
				t.Fatal(err)
			}
			artifact := evidence.Artifact{ID: "artifact", TenantID: "tenant", RequestID: "upload-request", FileName: file.Name, MediaType: file.MediaType, SizeBytes: file.SizeBytes, SHA256: file.SHA256, StorageKey: info.Key, Status: evidence.ArtifactStoredUnscanned}
			if test.change == "QUARANTINED" || test.change == "DELETED" {
				artifact.Status = evidence.ArtifactStatus(test.change)
			}
			if _, err := repo.CreateArtifact(ctx, artifact); err != nil {
				t.Fatal(err)
			}
			documents := &documentContentStore{MemoryDistributionStore: evidence.NewMemoryDistributionStore(repo, nil, nil), value: evidence.DocumentOccurrence{ArtifactID: artifact.ID, ArtifactRequestID: artifact.RequestID, FileName: file.Name, MediaType: file.MediaType, SizeBytes: file.SizeBytes, SHA256: file.SHA256, ArtifactStatus: artifact.Status, DemoPreviewAvailable: true}}
			capture := evidence.NewService(repo, store)
			distributions := evidence.NewDistributionService(documents)
			evidence.ConfigureDemoSamplePreview(capture, nil, test.capture)
			evidence.ConfigureDemoSamplePreview(nil, distributions, test.distribution)
			if allowUnscanned {
				evidence.ConfigureDemoSamplePreview(capture, distributions, false)
				capture.ConfigureDemoUnscannedArtifacts(test.capture)
				distributions.ConfigureDemoUnscannedArtifacts(test.distribution)
			}
			if test.change == "unknown" {
				documents.value.FileName = "genuine-upload.pdf"
			}
			if test.change == "denied" {
				documents.denied = true
			}
			if test.change == "revoked" {
				page, err := distributions.ListDocuments(ctx, evidence.DocumentQuery{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reader", Limit: 1})
				if err != nil || !page.Items[0].DemoPreviewAvailable {
					t.Fatal("expected initial capability")
				}
				documents.denied = true
			}
			changed := append([]byte(nil), content...)
			switch test.change {
			case "changed":
				changed[len(changed)/2] ^= 1
			case "truncated":
				changed = changed[:len(changed)-1]
			case "appended":
				changed = append(changed, 'x')
			}
			if _, err := store.Put(ctx, "private-demo-key", bytes.NewReader(changed), 20<<20); err != nil {
				t.Fatal(err)
			}
			actor := identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reader", Kind: "PERSON", AuthenticationMethod: "TEST", AssuranceLevel: "HIGH", SessionID: "session", IssuedAt: time.Now().Add(-time.Minute), ExpiresAt: time.Now().Add(time.Hour)}
			if test.change == "tenant" {
				actor.TenantID = "other"
			}
			r := httptest.NewRequest(http.MethodGet, "/content?response_revision_id=revision&demo_preview_available=true&demo_unscanned_allowed=true&download="+fmt.Sprint(test.change == "download"), nil)
			if test.change != "identity" {
				r = r.WithContext(identity.WithActor(ctx, actor))
			}
			r.SetPathValue("submission_id", "submission")
			r.SetPathValue("field_id", "field")
			r.SetPathValue("artifact_id", "artifact")
			w := httptest.NewRecorder()
			(&API{deps: Dependencies{FormDistributions: distributions, Evidence: capture}}).openFormDocument(w, r)
			if w.Code != test.want {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			if w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Content-Security-Policy") != "sandbox; default-src 'none'" {
				t.Fatal("missing protected headers")
			}
			if test.want == 200 && !bytes.Equal(w.Body.Bytes(), content) {
				t.Fatal("different content delivered")
			}
			if test.want == 200 && (documents.query.ResponseRevisionID != "revision" || documents.query.LegalEntityID != "entity" || documents.query.PrincipalID != "reader" || documents.query.SubmissionID != "submission" || documents.query.FieldID != "field" || documents.query.ArtifactID != "artifact" || documents.query.Limit != 1) {
				t.Fatalf("lost exact scope: %+v", documents.query)
			}
			if test.want != 200 && (bytes.Contains(w.Body.Bytes(), content) || strings.Contains(w.Body.String(), "private-demo-key")) {
				t.Fatal("content leaked")
			}
			stored, err := repo.GetArtifact(ctx, "tenant", "upload-request", "artifact")
			if err != nil || stored.Status != artifact.Status {
				t.Fatal("preview changed artifact state")
			}
		})
	}
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
	documents := &documentContentStore{MemoryDistributionStore: evidence.NewMemoryDistributionStore(repo, nil, nil), value: evidence.DocumentOccurrence{ID: "occurrence", ArtifactID: artifact.ID, ArtifactRequestID: artifact.RequestID, RequestID: "submission-request", SubmissionID: "submission", FieldID: "policy", SHA256: info.SHA256, SizeBytes: info.SizeBytes, ArtifactStatus: evidence.ArtifactAvailable}}
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
