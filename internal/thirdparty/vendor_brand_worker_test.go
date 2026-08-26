package thirdparty

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func TestVendorBrandMigrationPermitsSafeBrandEventOnExistingVendorStream(t *testing.T) {
	t.Parallel()
	content, err := os.ReadFile("../../migrations/000047_vendor_brand_assets.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	schema := string(content)
	for _, required := range []string{"'VENDOR_BRAND'", "'VendorBrandDiscovered'", "NEW.aggregate_type='VENDOR_BRAND'"} {
		if !strings.Contains(schema, required) {
			t.Fatalf("vendor brand migration missing event contract %q", required)
		}
	}
}

func setupVendorBrandWorker(t *testing.T, now time.Time, domain string) (*MemoryRepository, *evidence.MemoryObjectStore, *VendorBrandWorker, Vendor) {
	t.Helper()
	repository := NewMemoryRepository()
	service := NewService(repository)
	service.now = func() time.Time { return now }
	ids := []string{"vendor-1", "relationship-1", "brand-job-1"}
	service.newID = func() (string, error) { value := ids[0]; ids = ids[1:]; return value, nil }
	input := validCreateInput()
	input.WebsiteDomain = domain
	created, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, input)
	if err != nil {
		t.Fatal(err)
	}
	store := evidence.NewMemoryObjectStore()
	discoverer := &VendorBrandDiscoverer{timeout: 3 * time.Second}
	discoverer.discover = func(context.Context, WebsiteDomain) (DiscoveredVendorBrand, error) {
		pngBytes := brandPNG(t, 32, 32)
		return DiscoveredVendorBrand{PNG: pngBytes, MediaType: "image/png", PixelWidth: 32, PixelHeight: 32, SourceDigest: stringsDigest(string(pngBytes))}, nil
	}
	worker := NewVendorBrandWorker(repository, store, discoverer, "brand-worker")
	return repository, store, worker, created.Vendor
}

func TestVendorBrandWorkerCompletesLeasedJobAndStoresCurrentAsset(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	repository, store, worker, vendor := setupVendorBrandWorker(t, now, "vendor.example")
	processed, err := worker.Maintain(context.Background(), now, 10)
	if err != nil || processed != 1 {
		t.Fatalf("Maintain() = (%d, %v)", processed, err)
	}
	job, err := repository.GetVendorBrandJob(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity"}, vendor.ID)
	if err != nil || job.State != VendorBrandJobCompleted || job.LeaseToken != "" {
		t.Fatalf("completed job = (%#v, %v)", job, err)
	}
	assets, err := repository.ListVendorBrandAssets(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity"}, vendor.ID)
	if err != nil || len(assets) != 1 || assets[0].State != VendorBrandAssetCurrent || assets[0].SourceKind != VendorBrandAssetDiscovered {
		t.Fatalf("assets = (%#v, %v)", assets, err)
	}
	opened, err := store.Open(context.Background(), assets[0].ArtifactKey)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	if data := make([]byte, 8); func() bool { _, err = opened.Read(data); return err != nil && !errors.Is(err, context.Canceled) }() || !bytes.Equal(data, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("stored object is not PNG: %x (%v)", data, err)
	}
	if len(repository.vendorBrandEvents) != 1 || len(repository.vendorBrandOutbox) != 1 {
		t.Fatalf("brand event/outbox counts = %d/%d", len(repository.vendorBrandEvents), len(repository.vendorBrandOutbox))
	}
}

func TestVendorBrandWorkerCancelsStaleVendorVersionWithoutFetching(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	repository, _, worker, vendor := setupVendorBrandWorker(t, now, "vendor.example")
	repository.mu.Lock()
	changed := repository.vendors[vendor.ID]
	changed.Version++
	changed.WebsiteDomain = "other.example"
	repository.vendors[vendor.ID] = changed
	repository.mu.Unlock()
	called := false
	worker.discoverer.discover = func(context.Context, WebsiteDomain) (DiscoveredVendorBrand, error) {
		called = true
		return DiscoveredVendorBrand{}, nil
	}
	processed, err := worker.Maintain(context.Background(), now, 1)
	if err != nil || processed != 1 || called {
		t.Fatalf("stale Maintain() = (%d, %v), discover called=%v", processed, err, called)
	}
	job, _ := repository.GetVendorBrandJob(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity"}, vendor.ID)
	if job.State != VendorBrandJobCancelled || job.LastFailureCode != VendorBrandFailureStale {
		t.Fatalf("stale job = %#v", job)
	}
}

func TestVendorBrandWorkerRetryIsLeasedIdempotentAndBounded(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	repository, _, worker, vendor := setupVendorBrandWorker(t, now, "vendor.example")
	worker.Configure(time.Minute, 2, time.Minute)
	worker.discoverer.discover = func(context.Context, WebsiteDomain) (DiscoveredVendorBrand, error) {
		return DiscoveredVendorBrand{}, ErrVendorBrandTimeout
	}
	processed, err := worker.Maintain(context.Background(), now, 1)
	if processed != 1 || err == nil {
		t.Fatalf("first Maintain() = (%d, %v)", processed, err)
	}
	job, _ := repository.GetVendorBrandJob(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity"}, vendor.ID)
	if job.State != VendorBrandJobReady || job.Attempts != 1 || job.LastFailureCode != VendorBrandFailureTimeout || !job.AvailableAt.After(now) {
		t.Fatalf("retry job = %#v", job)
	}
	processed, err = worker.Maintain(context.Background(), job.AvailableAt, 1)
	if processed != 1 || err == nil {
		t.Fatalf("terminal Maintain() = (%d, %v)", processed, err)
	}
	job, _ = repository.GetVendorBrandJob(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity"}, vendor.ID)
	if job.State != VendorBrandJobFailed || job.Attempts != 2 || job.LastFailureCode != VendorBrandFailureTimeout {
		t.Fatalf("terminal job = %#v", job)
	}
	processed, err = worker.Maintain(context.Background(), now.Add(10*time.Minute), 1)
	if err != nil || processed != 0 {
		t.Fatalf("terminal job was retried: (%d, %v)", processed, err)
	}
}

func TestVendorBrandWorkerRejectsMalformedDiscoveryResultBeforeStorage(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	repository, store, worker, vendor := setupVendorBrandWorker(t, now, "vendor.example")
	worker.discoverer.discover = func(context.Context, WebsiteDomain) (DiscoveredVendorBrand, error) {
		return DiscoveredVendorBrand{PNG: brandPNG(t, 1, 1), MediaType: "image/png", PixelWidth: 1, PixelHeight: 1, SourceDigest: "../../unsafe"}, nil
	}
	processed, err := worker.Maintain(context.Background(), now, 1)
	if processed != 1 || err == nil {
		t.Fatalf("malformed result Maintain() = (%d, %v)", processed, err)
	}
	job, _ := repository.GetVendorBrandJob(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity"}, vendor.ID)
	if job.State != VendorBrandJobReady || job.LastFailureCode != VendorBrandFailureImage {
		t.Fatalf("malformed result job = %#v", job)
	}
	if _, err := store.Open(context.Background(), vendorBrandObjectKey("bank", vendor.ID, vendor.Version, stringsDigest("replacement"))); err == nil {
		t.Fatal("malformed discovery result wrote an object")
	}
}

func TestVendorBrandObjectWriteReusesIdenticalContentAddress(t *testing.T) {
	t.Parallel()
	store := evidence.NewMemoryObjectStore()
	worker := NewVendorBrandWorker(NewMemoryRepository(), store, NewDefaultVendorBrandDiscoverer(), "worker")
	content := brandPNG(t, 8, 8)
	key := "vendor-brands/tenant/vendor/v1/" + stringsDigest(string(content)) + ".png"
	first, err := worker.putObject(context.Background(), key, content)
	if err != nil {
		t.Fatal(err)
	}
	second, err := worker.putObject(context.Background(), key, content)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("idempotent object info differs: %#v != %#v", first, second)
	}
}

func TestVendorBrandWorkerLeaseLossDoesNotPublishAsset(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	repository, store, worker, vendor := setupVendorBrandWorker(t, now, "vendor.example")
	worker.afterStore = func() {
		repository.mu.Lock()
		job := repository.vendorBrandJobs[vendorBrandJobKey("bank", vendor.ID)]
		job.LeaseToken = "other-lease"
		repository.vendorBrandJobs[vendorBrandJobKey("bank", vendor.ID)] = job
		repository.mu.Unlock()
	}
	processed, err := worker.Maintain(context.Background(), now, 1)
	if processed != 1 || !errors.Is(err, ErrVendorBrandJobLeaseLost) {
		t.Fatalf("lease loss Maintain() = (%d, %v)", processed, err)
	}
	assets, _ := repository.ListVendorBrandAssets(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity"}, vendor.ID)
	if len(assets) != 0 {
		t.Fatalf("lease-lost assets = %#v", assets)
	}
	// Content-addressed bytes remain intact because completion may have committed
	// even when its acknowledgement was lost. Unreferenced bytes are harmless and
	// can be reconciled without deleting a winner's referenced artifact.
	pngBytes := brandPNG(t, 32, 32)
	key := vendorBrandObjectKey("bank", vendor.ID, vendor.Version, stringsDigest(string(pngBytes)))
	if opened, err := store.Open(context.Background(), key); err != nil {
		t.Fatalf("lease-lost content-addressed object was deleted: %v", err)
	} else {
		_ = opened.Close()
	}
}

func TestVendorBrandRepositoryRejectsStaleClaimCompletion(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	repository, _, _, vendor := setupVendorBrandWorker(t, now, "vendor.example")
	claims, err := repository.ClaimVendorBrandJobs(context.Background(), "worker-a", now, time.Minute, 5, 1)
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim = (%#v, %v)", claims, err)
	}
	second, err := repository.ClaimVendorBrandJobs(context.Background(), "worker-b", now.Add(2*time.Minute), time.Minute, 5, 1)
	if err != nil || len(second) != 1 {
		t.Fatalf("reclaim = (%#v, %v)", second, err)
	}
	asset := validVendorBrandAssetForTest(now, vendor)
	if _, err := repository.CompleteVendorBrandJob(context.Background(), claims[0], asset, now.Add(2*time.Minute)); !errors.Is(err, ErrVendorBrandJobLeaseLost) {
		t.Fatalf("stale completion error = %v", err)
	}
}

func TestVendorBrandRepositoryRejectsAssetOutsideClaimScope(t *testing.T) {
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	repository, _, _, vendor := setupVendorBrandWorker(t, now, "vendor.example")
	claims, err := repository.ClaimVendorBrandJobs(context.Background(), "worker-a", now, time.Minute, 5, 1)
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim = (%#v, %v)", claims, err)
	}
	asset := validVendorBrandAssetForTest(now, vendor)
	asset.TenantID = "other-bank"
	if _, err := repository.CompleteVendorBrandJob(context.Background(), claims[0], asset, now); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-scope asset completion error = %v", err)
	}
	assets, _ := repository.ListVendorBrandAssets(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity"}, vendor.ID)
	if len(assets) != 0 {
		t.Fatalf("cross-scope asset was stored: %#v", assets)
	}
}

func validVendorBrandAssetForTest(now time.Time, vendor Vendor) VendorBrandAsset {
	next := now.Add(time.Hour)
	return VendorBrandAsset{
		ID: "asset-1", TenantID: vendor.TenantID, VendorID: vendor.ID,
		SourceKind: VendorBrandAssetDiscovered, State: VendorBrandAssetCurrent, SourceDomain: vendor.WebsiteDomain,
		ArtifactKey: "vendor-brands/key.png", SourceDigest: stringsDigest("source"), MediaType: "image/png",
		PixelWidth: 1, PixelHeight: 1, ByteSize: 1, RetrievedAt: &now, NextRefreshAt: &next,
		CreatedAt: now, UpdatedAt: now, Version: 1,
	}
}

func TestVendorBrandDiscovererProductionTransportHasNoProxyOrReuse(t *testing.T) {
	discoverer := NewVendorBrandDiscoverer(&brandResolverStub{}, func(context.Context, string, string) (net.Conn, error) { return nil, errors.New("unused") })
	transport, ok := discoverer.transportFactory(func(context.Context, string, string) (net.Conn, error) { return nil, nil }, "vendor.example").(*http.Transport)
	if !ok || transport.Proxy != nil || !transport.DisableKeepAlives || transport.MaxResponseHeaderBytes <= 0 || transport.TLSClientConfig == nil || transport.TLSClientConfig.ServerName != "vendor.example" {
		t.Fatalf("unsafe transport = %#v", transport)
	}
	_ = netip.MustParseAddr("93.184.216.34")
}
