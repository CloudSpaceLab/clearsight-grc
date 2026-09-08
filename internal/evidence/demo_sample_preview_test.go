package evidence

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/demodocuments"
)

func TestDemoSamplePreviewPreservesUnscannedStatusAndChecksStoredBytes(t *testing.T) {
	for _, file := range demodocuments.Files() {
		t.Run(file.Name, func(t *testing.T) {
			ctx := context.Background()
			content, _ := demodocuments.Read(file.Name)
			inspected, media, err := inspectArtifact(file.Name, file.MediaType, bytes.NewReader(content), 20<<20)
			if err != nil || media != file.MediaType || !bytes.Equal(inspected, content) {
				t.Fatalf("ordinary upload inspection: %s %v", media, err)
			}
			repo, store := NewMemoryRepository(nil, nil), NewMemoryObjectStore()
			service := NewService(repo, store)
			info, err := store.Put(ctx, "private", bytes.NewReader(content), 20<<20)
			if err != nil {
				t.Fatal(err)
			}
			original := Artifact{ID: "a", TenantID: "t", RequestID: "r", FileName: file.Name, MediaType: file.MediaType, SizeBytes: file.SizeBytes, SHA256: file.SHA256, StorageKey: info.Key, Status: ArtifactStoredUnscanned}
			repo.artifacts["a"] = original
			deny := func(tenant, request string) {
				t.Helper()
				_, reader, err := service.OpenDemoSampleArtifact(ctx, tenant, request, "a")
				if reader != nil {
					reader.Close()
					t.Fatal("returned forbidden bytes")
				}
				if err != ErrNotFound {
					t.Fatalf("want not found, got %v", err)
				}
			}
			deny("t", "r") // Disabled by default, even for an exact sample.
			ConfigureDemoSamplePreview(service, nil, true)
			deny("other", "r")
			deny("t", "other")
			opened, reader, err := service.OpenDemoSampleArtifact(ctx, "t", "r", "a")
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(reader)
			reader.Close()
			if err != nil || !bytes.Equal(got, content) || opened.StorageKey != "" || opened.Status != ArtifactStoredUnscanned || repo.artifacts["a"].Status != ArtifactStoredUnscanned {
				t.Fatal("demo open changed status or content")
			}
			if len(repo.scanReceipts) != 0 || len(repo.scanJobs) != 0 {
				t.Fatal("demo preview created inspection state")
			}
			if _, reader, err := service.OpenArtifact(ctx, "t", "r", "a"); err != ErrNotFound || reader != nil {
				t.Fatal("ordinary open allowed unscanned sample")
			}
			for _, mutate := range []func(*Artifact){
				func(a *Artifact) { a.FileName = "unknown.pdf" }, func(a *Artifact) { a.MediaType = "text/plain" },
				func(a *Artifact) { a.SHA256 = "wrong" }, func(a *Artifact) { a.SizeBytes++ },
				func(a *Artifact) { a.Status = ArtifactQuarantined }, func(a *Artifact) { a.Status = ArtifactDeleted },
			} {
				changed := original
				mutate(&changed)
				repo.artifacts["a"] = changed
				deny("t", "r")
			}
			repo.artifacts["a"] = original
			changed := append([]byte(nil), content...)
			changed[len(changed)/2] ^= 1
			for _, value := range [][]byte{changed, content[:len(content)-1], append(append([]byte(nil), content...), 'x')} {
				if _, err := store.Put(ctx, "private", bytes.NewReader(value), 20<<20); err != nil {
					t.Fatal(err)
				}
				deny("t", "r")
			}
		})
	}
}

type demoDocumentStore struct {
	*MemoryDistributionStore
	page DocumentPage
}

func (s *demoDocumentStore) ListDocuments(context.Context, DocumentQuery) (DocumentPage, error) {
	return s.page, nil
}

func TestDemoDocumentCapabilityIsRecomputedOnCopiedAuthorizedPage(t *testing.T) {
	file := demodocuments.Files()[0]
	known := DocumentOccurrence{FileName: file.Name, MediaType: file.MediaType, SHA256: file.SHA256, SizeBytes: file.SizeBytes, ArtifactStatus: ArtifactStoredUnscanned}
	forged := known
	forged.FileName = "unknown.pdf"
	forged.DemoPreviewAvailable = true
	quarantined := known
	quarantined.ArtifactStatus = ArtifactQuarantined
	quarantined.DemoPreviewAvailable = true
	deleted := known
	deleted.ArtifactStatus = ArtifactDeleted
	deleted.DemoPreviewAvailable = true
	store := &demoDocumentStore{page: DocumentPage{Items: []DocumentOccurrence{known, forged, quarantined, deleted}, NextCursor: "cursor"}}
	service := NewDistributionService(store)
	q := DocumentQuery{TenantID: "t", LegalEntityID: "e", PrincipalID: "p", Limit: 25}
	for _, enabled := range []bool{false, true, false} {
		ConfigureDemoSamplePreview(nil, service, enabled)
		page, err := service.ListDocuments(context.Background(), q)
		if err != nil || page.NextCursor != "cursor" {
			t.Fatalf("page: %+v %v", page, err)
		}
		for i, item := range page.Items {
			if item.DemoPreviewAvailable != (enabled && i == 0) {
				t.Fatalf("capability %d: %+v", i, item)
			}
		}
		if store.page.Items[0].DemoPreviewAvailable || !store.page.Items[1].DemoPreviewAvailable {
			t.Fatal("mutated store page")
		}
	}
}

func TestDemoDocumentCapabilityRetainsRepositoryMembershipAndRevocationChecks(t *testing.T) {
	for _, change := range []string{"tenant", "entity", "principal", "revision", "submission", "field", "artifact", "revocation"} {
		t.Run(change, func(t *testing.T) {
			store, q := documentMemoryFixture()
			file := demodocuments.Files()[0]
			artifact := store.repo.artifacts["artifact"]
			artifact.FileName, artifact.MediaType, artifact.SHA256, artifact.SizeBytes, artifact.Status = file.Name, file.MediaType, file.SHA256, file.SizeBytes, ArtifactStoredUnscanned
			store.repo.artifacts["artifact"] = artifact
			service := NewDistributionService(store)
			ConfigureDemoSamplePreview(nil, service, true)
			q.ResponseRevisionID, q.SubmissionID, q.FieldID, q.ArtifactID = "revision-newer", "newer", "file", "artifact"
			page, err := service.ListDocuments(context.Background(), q)
			if err != nil || len(page.Items) != 1 || !page.Items[0].DemoPreviewAvailable {
				t.Fatalf("authorized sample: %+v %v", page, err)
			}
			switch change {
			case "tenant":
				q.TenantID = "other"
			case "entity":
				q.LegalEntityID = "other"
			case "principal":
				q.PrincipalID = "other"
			case "revision":
				q.ResponseRevisionID = "other"
			case "submission":
				q.SubmissionID = "other"
			case "field":
				q.FieldID = "photo"
			case "artifact":
				q.ArtifactID = "draft"
			case "revocation":
				candidate := store.repo.candidates["reader"]
				candidate.Active = false
				store.repo.candidates["reader"] = candidate
			}
			page, err = service.ListDocuments(context.Background(), q)
			if err != nil || len(page.Items) != 0 {
				t.Fatalf("denied sample: %+v %v", page, err)
			}
		})
	}
}
