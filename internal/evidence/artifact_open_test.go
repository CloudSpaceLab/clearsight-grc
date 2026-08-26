package evidence

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestOpenArtifactRequiresExactAvailableArtifactAndHidesStorageKey(t *testing.T) {
	repo := NewMemoryRepository(nil, nil)
	store := NewMemoryObjectStore()
	service := NewService(repo, store)
	info, err := store.Put(context.Background(), "tenant/request/document.pdf", bytes.NewBufferString("approved document"), 1024)
	if err != nil {
		t.Fatal(err)
	}
	repo.artifacts["artifact-1"] = Artifact{ID: "artifact-1", TenantID: "tenant-a", RequestID: "request-1", FileName: "document.pdf", MediaType: "application/pdf", SizeBytes: info.SizeBytes, SHA256: info.SHA256, StorageKey: info.Key, Status: ArtifactAvailable, CreatedAt: time.Now()}

	artifact, reader, err := service.OpenArtifact(context.Background(), "tenant-a", "request-1", "artifact-1")
	if err != nil {
		t.Fatalf("open available artifact: %v", err)
	}
	defer reader.Close()
	content, _ := io.ReadAll(reader)
	if string(content) != "approved document" || artifact.StorageKey != "" {
		t.Fatalf("unexpected opened artifact %#v %q", artifact, content)
	}
	if _, _, err := service.OpenArtifact(context.Background(), "tenant-a", "request-other", "artifact-1"); err != ErrNotFound {
		t.Fatalf("expected request-scoped not found, got %v", err)
	}

	repo.artifacts["artifact-1"] = Artifact{ID: "artifact-1", TenantID: "tenant-a", RequestID: "request-1", StorageKey: info.Key, Status: ArtifactQuarantined}
	if _, _, err := service.OpenArtifact(context.Background(), "tenant-a", "request-1", "artifact-1"); err != ErrNotFound {
		t.Fatalf("expected unavailable artifact not found, got %v", err)
	}
}
