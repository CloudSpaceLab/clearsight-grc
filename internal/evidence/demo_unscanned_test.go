package evidence

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestDemoUnscannedOpenPreservesStatusAndIntegrity(t *testing.T) {
	ctx := context.Background()
	repo, store := NewMemoryRepository(nil, nil), NewMemoryObjectStore()
	service := NewService(repo, store)
	object, err := store.Put(ctx, "original", bytes.NewBufferString("ordinary uploaded document"), 1024)
	if err != nil {
		t.Fatal(err)
	}
	original := Artifact{ID: "a", TenantID: "t", RequestID: "r", FileName: "ordinary.txt", MediaType: "text/plain", StorageKey: object.Key, SizeBytes: object.SizeBytes, SHA256: object.SHA256, Status: ArtifactStoredUnscanned}
	repo.artifacts["a"] = original
	deny := func(tenant, request string) {
		t.Helper()
		_, reader, err := service.OpenArtifact(ctx, tenant, request, "a")
		if reader != nil {
			reader.Close()
			t.Fatal("returned blocked content")
		}
		if err != ErrNotFound {
			t.Fatalf("expected unavailable, got %v", err)
		}
	}
	deny("t", "r")
	service.ConfigureDemoUnscannedArtifacts(true)
	artifact, reader, err := service.OpenArtifact(ctx, "t", "r", "a")
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(reader)
	reader.Close()
	if err != nil || string(content) != "ordinary uploaded document" || artifact.Status != ArtifactStoredUnscanned || !artifact.DemoUnscannedAllowed || artifact.StorageKey != "" {
		t.Fatalf("invalid opened artifact: %+v %v", artifact, err)
	}
	if repo.artifacts["a"].Status != ArtifactStoredUnscanned || len(repo.scanReceipts) != 0 {
		t.Fatal("exception fabricated a scan")
	}
	deny("other", "r")
	deny("t", "other")
	for _, status := range []ArtifactStatus{ArtifactQuarantined, ArtifactDeleted, ArtifactStatus("UNKNOWN")} {
		changed := original
		changed.Status = status
		changed.DemoUnscannedAllowed = true
		repo.artifacts["a"] = changed
		deny("t", "r")
	}
	repo.artifacts["a"] = original
	service.ConfigureDemoUnscannedArtifacts(false)
	deny("t", "r")
	service.ConfigureDemoUnscannedArtifacts(true)
	if _, err := store.Put(ctx, "original", bytes.NewBufferString("different uploaded bytes!"), 1024); err != nil {
		t.Fatal(err)
	}
	deny("t", "r")
}

func TestDemoUnscannedCollectionEligibilityIsRefreshed(t *testing.T) {
	repo := NewMemoryRepository(nil, nil)
	service := NewService(repo, nil)
	repo.artifacts["a"] = Artifact{ID: "a", TenantID: "t", RequestID: "source", SHA256: "digest", SizeBytes: 4, Status: ArtifactStoredUnscanned}
	request := Request{TenantID: "t", Fields: []Field{{Type: "vendor_document", CollectionResolution: &CollectionResolution{ID: "receipt", Version: 1, SourceArtifactRequestID: "source", BankReviewState: "PENDING", Source: DocumentOccurrence{ArtifactID: "a", SHA256: "digest", SizeBytes: 4, Current: true, ArtifactStatus: ArtifactStoredUnscanned, DemoUnscannedAllowed: true}}}}}
	for _, enabled := range []bool{false, true, false} {
		service.ConfigureDemoUnscannedArtifacts(enabled)
		refreshed := RefreshCollectionResolutions(context.Background(), request, service.GetArtifact, time.Now())
		if got := CollectionFieldFulfilled(refreshed.Fields[0], time.Now()); got != enabled {
			t.Fatalf("enabled=%v fulfilled=%v", enabled, got)
		}
		if refreshed.Fields[0].CollectionResolution.Source.ArtifactStatus != ArtifactStoredUnscanned {
			t.Fatal("scan state changed")
		}
	}
}

func TestDemoUnscannedDocumentCapabilityIsRecomputed(t *testing.T) {
	store := &demoDocumentStore{page: DocumentPage{Items: []DocumentOccurrence{
		{ArtifactStatus: ArtifactStoredUnscanned, DemoUnscannedAllowed: true},
		{ArtifactStatus: ArtifactAvailable, DemoUnscannedAllowed: true},
		{ArtifactStatus: ArtifactQuarantined, DemoUnscannedAllowed: true},
	}}}
	service := NewDistributionService(store)
	for _, enabled := range []bool{false, true, false} {
		service.ConfigureDemoUnscannedArtifacts(enabled)
		page, err := service.ListDocuments(context.Background(), DocumentQuery{TenantID: "t", LegalEntityID: "e", PrincipalID: "p", Limit: 25})
		if err != nil {
			t.Fatal(err)
		}
		for i, item := range page.Items {
			if item.DemoUnscannedAllowed != (enabled && i == 0) {
				t.Fatalf("stale capability enabled=%v item=%+v", enabled, item)
			}
		}
	}
}
