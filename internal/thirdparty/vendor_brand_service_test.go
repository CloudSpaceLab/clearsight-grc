package thirdparty

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func TestVendorIdentityAuthorityModeOffStillUsesVerifiedIdentity(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	created, err := service.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	guard, err := commandauth.New(nil, commandauth.ModeOff, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	service.ConfigureIdentityAuthority(guard)
	ctx := vendorIdentityContext("bank", "entity", "verified-owner", time.Now().UTC())
	updated, err := service.UpdateVendorIdentity(ctx, Actor{TenantID: "forged", LegalEntityID: "forged", PrincipalID: "forged"}, created.Vendor.ID, UpdateVendorIdentityInput{ExpectedVersion: 1, LegalName: "Updated vendor"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.LegalName != "Updated vendor" || repo.vendorIdentityEvents[len(repo.vendorIdentityEvents)-1].ActorPrincipalID != "verified-owner" {
		t.Fatalf("mode-off update was not bound to verified identity: %#v", updated)
	}
}

func TestVendorBrandApprovedOverridePrecedenceAndRemoval(t *testing.T) {
	repo := NewMemoryRepository()
	store := evidence.NewMemoryObjectStore()
	identityService := NewService(repo)
	input := validCreateInput()
	input.WebsiteDomain = "acme.example"
	created, err := identityService.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, input)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)
	discoveredPNG := testBrandPNG(t, color.RGBA{R: 15, G: 80, B: 120, A: 255})
	digest := sha256.Sum256(discoveredPNG)
	info, err := store.Put(context.Background(), vendorBrandObjectKey("bank", created.Vendor.ID, 1, hex.EncodeToString(digest[:])), bytes.NewReader(discoveredPNG), VendorBrandMaximumUploadBytes)
	if err != nil {
		t.Fatal(err)
	}
	repo.vendorBrandAssets["discovered"] = VendorBrandAsset{ID: "discovered", TenantID: "bank", VendorID: created.Vendor.ID, SourceKind: VendorBrandAssetDiscovered, State: VendorBrandAssetCurrent, SourceDomain: "acme.example", ArtifactKey: info.Key, SourceDigest: info.SHA256, MediaType: "image/png", PixelWidth: 16, PixelHeight: 16, ByteSize: info.SizeBytes, RetrievedAt: &now, CreatedAt: now, UpdatedAt: now, Version: 1}
	guard := &vendorIdentityGuardStub{}
	service := NewVendorBrandService(repo, store, guard)
	identityService.ConfigureVendorBrands(service)
	service.now = func() time.Time { return now.Add(time.Minute) }
	ctx := vendorIdentityContext("bank", "entity", "verified-owner", now)

	view, err := service.PutApprovedBrand(ctx, created.Vendor.ID, 0, "upload-1", "image/png", bytes.NewReader(testBrandPNG(t, color.RGBA{R: 200, G: 10, B: 30, A: 255})))
	if err != nil {
		t.Fatal(err)
	}
	if view.Brand.State != VendorBrandApprovedLogo || view.Brand.Source != VendorBrandSourceApprovedUpload || view.Brand.Version != 1 || view.Brand.AssetToken == "" {
		t.Fatalf("approved presentation = %#v", view.Brand)
	}
	updatedRelationship, err := identityService.UpdateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, created.Relationship.ID, UpdateRelationshipInput{ExpectedVersion: 1, ServiceName: created.Relationship.ServiceName, Criticality: created.Relationship.Criticality, PrivacyRole: created.Relationship.PrivacyRole})
	if err != nil || updatedRelationship.Brand.State != VendorBrandApprovedLogo {
		t.Fatalf("approved relationship response = %#v, %v", updatedRelationship.Brand, err)
	}
	if _, err := service.PutApprovedBrand(ctx, created.Vendor.ID, 0, "upload-2", "image/png", bytes.NewReader(testBrandPNG(t, color.Black))); !errors.Is(err, ErrBrandVersionConflict) {
		t.Fatalf("stale upload error = %v", err)
	}
	restored, err := service.RemoveApprovedBrand(ctx, created.Vendor.ID, 1, "remove-1")
	if err != nil {
		t.Fatal(err)
	}
	if restored.Brand.State != VendorBrandWebsiteIcon || restored.Brand.Source != VendorBrandSourceVendorWebsite || restored.Brand.Version != 2 {
		t.Fatalf("restored presentation = %#v", restored.Brand)
	}
	removedEvent := repo.vendorBrandEvents[len(repo.vendorBrandEvents)-1]
	if removedEvent.AssetID == "" || removedEvent.AssetVersion != 2 {
		t.Fatalf("removed event asset = %#v", removedEvent)
	}
	updatedRelationship, err = identityService.UpdateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, created.Relationship.ID, UpdateRelationshipInput{ExpectedVersion: 2, ServiceName: created.Relationship.ServiceName, Criticality: created.Relationship.Criticality, PrivacyRole: created.Relationship.PrivacyRole})
	if err != nil || updatedRelationship.Brand.State != VendorBrandWebsiteIcon {
		t.Fatalf("discovered relationship response = %#v, %v", updatedRelationship.Brand, err)
	}
	replayed, err := service.RemoveApprovedBrand(ctx, created.Vendor.ID, 1, "remove-1")
	if err != nil || replayed.Brand.Version != 2 {
		t.Fatalf("remove replay = %#v, %v", replayed.Brand, err)
	}
}

func TestVendorBrandRejectsSVGAndDomainMismatchedDiscovery(t *testing.T) {
	repo := NewMemoryRepository()
	store := evidence.NewMemoryObjectStore()
	identityService := NewService(repo)
	input := validCreateInput()
	input.WebsiteDomain = "current.example"
	created, err := identityService.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, input)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	repo.vendorBrandAssets["old"] = VendorBrandAsset{ID: "old", TenantID: "bank", VendorID: created.Vendor.ID, SourceKind: VendorBrandAssetDiscovered, State: VendorBrandAssetCurrent, SourceDomain: "old.example", ArtifactKey: "old", SourceDigest: string(make([]byte, 64)), MediaType: "image/png", PixelWidth: 16, PixelHeight: 16, ByteSize: 100, RetrievedAt: &now, CreatedAt: now, UpdatedAt: now, Version: 1}
	service := NewVendorBrandService(repo, store, &vendorIdentityGuardStub{})
	ctx := vendorIdentityContext("bank", "entity", "owner", now)
	if _, err := service.PutApprovedBrand(ctx, created.Vendor.ID, 0, "svg", "image/svg+xml", bytes.NewBufferString(`<svg/>`)); !errors.Is(err, ErrUnsupportedVendorBrandMedia) {
		t.Fatalf("svg upload error = %v", err)
	}
	view, err := service.GetIdentity(ctx, Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, created.Vendor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Brand.State != VendorBrandPending {
		t.Fatalf("domain-mismatched presentation = %#v", view.Brand)
	}
}

func TestVendorBrandPresentationIsUnavailableWhenDiscoveryIsDisabled(t *testing.T) {
	repo := NewMemoryRepository()
	base := NewService(repo)
	input := validCreateInput()
	input.WebsiteDomain = "vendor.example"
	created, err := base.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, input)
	if err != nil {
		t.Fatal(err)
	}
	service := NewVendorBrandService(repo, evidence.NewMemoryObjectStore(), &vendorIdentityGuardStub{})
	service.ConfigureDiscoveryEnabled(false)
	view, err := service.GetIdentity(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, created.Vendor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Brand.State != VendorBrandUnavailable {
		t.Fatalf("state=%s", view.Brand.State)
	}
}

func TestVersionedVendorBrandTokenKeepsOpeningSupersededAsset(t *testing.T) {
	repo := NewMemoryRepository()
	store := evidence.NewMemoryObjectStore()
	base := NewService(repo)
	created, err := base.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	service := NewVendorBrandService(repo, store, &vendorIdentityGuardStub{})
	service.now = func() time.Time { return now }
	ctx := vendorIdentityContext("bank", "entity", "owner", now)
	first, err := service.PutApprovedBrand(ctx, created.Vendor.ID, 0, "first", "image/png", bytes.NewReader(testBrandPNG(t, color.Black)))
	if err != nil {
		t.Fatal(err)
	}
	token := first.Brand.AssetToken
	service.now = func() time.Time { return now.Add(time.Minute) }
	if _, err = service.PutApprovedBrand(ctx, created.Vendor.ID, 1, "second", "image/png", bytes.NewReader(testBrandPNG(t, color.White))); err != nil {
		t.Fatal(err)
	}
	asset, reader, err := service.OpenBrand(ctx, Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, created.Vendor.ID, token)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if asset.State != VendorBrandAssetSuperseded {
		t.Fatalf("historical asset state=%s", asset.State)
	}
}

func TestVendorBrandReservationCleanerDeletesOnlyUnreferencedExpiredObjects(t *testing.T) {
	repo := NewMemoryRepository()
	store := evidence.NewMemoryObjectStore()
	base := NewService(repo)
	created, err := base.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	key := "vendor-brands/orphan.png"
	if _, err = store.Put(context.Background(), key, bytes.NewBufferString("orphan"), 100); err != nil {
		t.Fatal(err)
	}
	reservationKey := "bank\x00" + created.Vendor.ID + "\x00orphan"
	repo.vendorBrandReservations[reservationKey] = VendorBrandUploadReservation{TenantID: "bank", VendorID: created.Vendor.ID, IdempotencyKey: "orphan", ArtifactKey: key, SourceDigest: strings.Repeat("a", 64), State: "RESERVED", ExpectedVersion: 0, CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-48 * time.Hour)}
	cleaner := &VendorBrandReservationCleaner{Repository: repo, Store: store, Retention: 24 * time.Hour}
	count, err := cleaner.Maintain(context.Background(), now, 10)
	if err != nil || count != 1 {
		t.Fatalf("cleanup=%d,%v", count, err)
	}
	if _, err = store.Open(context.Background(), key); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("orphan still exists: %v", err)
	}
	if _, ok := repo.vendorBrandReservations[reservationKey]; ok {
		t.Fatal("reservation not removed")
	}
}

func TestVendorBrandReservationCleanerPreservesObjectReferencedByAnotherReservation(t *testing.T) {
	repo := NewMemoryRepository()
	store := evidence.NewMemoryObjectStore()
	base := NewService(repo)
	created, err := base.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	key := "vendor-brands/shared.png"
	if _, err = store.Put(context.Background(), key, bytes.NewBufferString("shared"), 100); err != nil {
		t.Fatal(err)
	}
	oldKey := "bank\x00" + created.Vendor.ID + "\x00old"
	repo.vendorBrandReservations[oldKey] = VendorBrandUploadReservation{TenantID: "bank", VendorID: created.Vendor.ID, IdempotencyKey: "old", ArtifactKey: key, SourceDigest: strings.Repeat("a", 64), State: "RESERVED", CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-48 * time.Hour)}
	repo.vendorBrandReservations["bank\x00"+created.Vendor.ID+"\x00new"] = VendorBrandUploadReservation{TenantID: "bank", VendorID: created.Vendor.ID, IdempotencyKey: "new", ArtifactKey: key, SourceDigest: strings.Repeat("a", 64), State: "RESERVED", CreatedAt: now, UpdatedAt: now}
	cleaner := &VendorBrandReservationCleaner{Repository: repo, Store: store, Retention: 24 * time.Hour}
	if _, err = cleaner.Maintain(context.Background(), now, 1); err != nil {
		t.Fatal(err)
	}
	reader, err := store.Open(context.Background(), key)
	if err != nil {
		t.Fatalf("shared object deleted: %v", err)
	}
	reader.Close()
	if stale, ok := repo.vendorBrandReservations[oldKey]; ok && stale.State == "COMMITTED" {
		t.Fatal("stale reservation was falsely marked committed")
	}
}

func TestVendorBrandReservationCleanerConvergesTwoStaleLegacySharedReservations(t *testing.T) {
	repo := NewMemoryRepository()
	store := evidence.NewMemoryObjectStore()
	now := time.Now().UTC()
	key := "vendor-brands/legacy-shared.png"
	if _, err := store.Put(context.Background(), key, bytes.NewBufferString("legacy"), 100); err != nil {
		t.Fatal(err)
	}
	for _, idempotencyKey := range []string{"old-a", "old-b"} {
		mapKey := "bank\x00vendor\x00" + idempotencyKey
		repo.vendorBrandReservations[mapKey] = VendorBrandUploadReservation{TenantID: "bank", VendorID: "vendor", IdempotencyKey: idempotencyKey, ArtifactKey: key, SourceDigest: strings.Repeat("a", 64), State: "RESERVED", CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-48 * time.Hour)}
	}
	cleaner := &VendorBrandReservationCleaner{Repository: repo, Store: store, Retention: 24 * time.Hour}
	count, err := cleaner.Maintain(context.Background(), now, 2)
	if err != nil || count != 2 {
		t.Fatalf("cleanup = %d, %v", count, err)
	}
	if len(repo.vendorBrandReservations) != 0 {
		t.Fatalf("stale reservations remain: %#v", repo.vendorBrandReservations)
	}
	if _, err = store.Open(context.Background(), key); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy orphan remains: %v", err)
	}
}

type failingBrandPresentationRepository struct{ *MemoryRepository }

func (r *failingBrandPresentationRepository) GetVendorBrandProjection(context.Context, Scope, string) (VendorBrandProjection, error) {
	return VendorBrandProjection{}, errors.New("projection unavailable")
}

func TestVendorBrandCommittedUploadDoesNotFailWhenPresentationReadFails(t *testing.T) {
	base := NewMemoryRepository()
	created, err := NewService(base).CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	repo := &failingBrandPresentationRepository{MemoryRepository: base}
	service := NewVendorBrandService(repo, evidence.NewMemoryObjectStore(), &vendorIdentityGuardStub{})
	view, err := service.PutApprovedBrand(vendorIdentityContext("bank", "entity", "owner", time.Now().UTC()), created.Vendor.ID, 0, "upload", "image/png", bytes.NewReader(testBrandPNG(t, color.Black)))
	if err != nil {
		t.Fatalf("committed upload reported failure: %v", err)
	}
	if view.Brand.State != VendorBrandApprovedLogo || view.Brand.Version != 1 {
		t.Fatalf("fallback result = %#v", view.Brand)
	}
	removed, err := service.RemoveApprovedBrand(vendorIdentityContext("bank", "entity", "owner", time.Now().UTC()), created.Vendor.ID, 1, "remove")
	if err != nil {
		t.Fatalf("committed removal reported failure: %v", err)
	}
	if removed.Brand.State != VendorBrandUnavailable || removed.Brand.Version != 2 {
		t.Fatalf("remove fallback = %#v", removed.Brand)
	}
}

func TestVendorIdentityCommittedUpdateUsesBestEffortPresentation(t *testing.T) {
	base := NewMemoryRepository()
	created, err := NewService(base).CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	repo := &failingBrandPresentationRepository{MemoryRepository: base}
	identities := NewService(repo)
	guard := &vendorIdentityGuardStub{}
	identities.ConfigureIdentityAuthority(guard)
	brands := NewVendorBrandService(repo, evidence.NewMemoryObjectStore(), guard)
	ctx := vendorIdentityContext("bank", "entity", "owner", time.Now().UTC())
	updated, err := identities.UpdateVendorIdentity(ctx, Actor{}, created.Vendor.ID, UpdateVendorIdentityInput{ExpectedVersion: 1, LegalName: "Updated vendor", WebsiteDomain: "vendor.example"})
	if err != nil {
		t.Fatal(err)
	}
	view := brands.IdentityForVendorBestEffort(ctx, Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, updated)
	if view.Vendor.Version != 2 || view.Brand.State != VendorBrandPending {
		t.Fatalf("best-effort view = %#v", view)
	}
}

func TestVendorBrandReauthorizesBeforeFinalize(t *testing.T) {
	repo := NewMemoryRepository()
	created, err := NewService(repo).CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	guard := &vendorIdentityGuardStub{failAfter: 2}
	service := NewVendorBrandService(repo, evidence.NewMemoryObjectStore(), guard)
	_, err = service.PutApprovedBrand(vendorIdentityContext("bank", "entity", "owner", time.Now().UTC()), created.Vendor.ID, 0, "revoked", "image/png", bytes.NewReader(testBrandPNG(t, color.Black)))
	if !errors.Is(err, commandauth.ErrNotAuthorized) {
		t.Fatalf("revocation error = %v", err)
	}
	if len(repo.vendorBrandAssets) != 0 || len(repo.vendorBrandEvents) != 0 || len(repo.vendorBrandOutbox) != 0 || len(repo.vendorBrandReceipts) != 0 {
		t.Fatal("revoked upload finalized material state")
	}
	if len(repo.vendorBrandReservations) != 1 {
		t.Fatal("recovery reservation was not retained")
	}
}

func TestRemoveApprovedBrandWithoutOverrideDoesNotAdvanceHistory(t *testing.T) {
	repo := NewMemoryRepository()
	created, err := NewService(repo).CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	service := NewVendorBrandService(repo, evidence.NewMemoryObjectStore(), &vendorIdentityGuardStub{})
	_, err = service.RemoveApprovedBrand(vendorIdentityContext("bank", "entity", "owner", time.Now().UTC()), created.Vendor.ID, 0, "remove")
	if !errors.Is(err, ErrVendorBrandOverrideNotFound) {
		t.Fatalf("remove error = %v", err)
	}
	if len(repo.vendorBrandEvents) != 0 || len(repo.vendorBrandOutbox) != 0 || len(repo.vendorBrandReceipts) != 0 {
		t.Fatal("no-op removal advanced material history")
	}
}

func TestApprovedBrandObjectKeysAreReservationUnique(t *testing.T) {
	repo := NewMemoryRepository()
	created, err := NewService(repo).CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	service := NewVendorBrandService(repo, evidence.NewMemoryObjectStore(), &vendorIdentityGuardStub{})
	ctx := vendorIdentityContext("bank", "entity", "owner", time.Now().UTC())
	body := testBrandPNG(t, color.Black)
	first, err := service.PutApprovedBrand(ctx, created.Vendor.ID, 0, "first", "image/png", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.PutApprovedBrand(ctx, created.Vendor.ID, first.Brand.Version, "second", "image/png", bytes.NewReader(body)); err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for _, a := range repo.vendorBrandAssets {
		keys[a.ArtifactKey] = true
	}
	if len(keys) != 2 {
		t.Fatalf("same-content uploads shared immutable key: %#v", keys)
	}
}

func TestApprovedBrandUploadReplayKeepsStableReservationIdentity(t *testing.T) {
	repo := NewMemoryRepository()
	created, err := NewService(repo).CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	service := NewVendorBrandService(repo, evidence.NewMemoryObjectStore(), &vendorIdentityGuardStub{})
	ctx := vendorIdentityContext("bank", "entity", "owner", time.Now().UTC())
	body := testBrandPNG(t, color.Black)
	first, err := service.PutApprovedBrand(ctx, created.Vendor.ID, 0, "stable", "image/png", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	replay, err := service.PutApprovedBrand(ctx, created.Vendor.ID, 0, "stable", "image/png", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("replay = %v", err)
	}
	if replay.Brand.AssetToken != first.Brand.AssetToken || len(repo.vendorBrandEvents) != 1 {
		t.Fatalf("replay changed result: first=%#v replay=%#v events=%d", first.Brand, replay.Brand, len(repo.vendorBrandEvents))
	}
}

func TestCurrentBrandProjectionIgnoresSupersededHistoryAndKeepsSnapshotVersion(t *testing.T) {
	repo := NewMemoryRepository()
	input := validCreateInput()
	input.WebsiteDomain = "vendor.example"
	created, err := NewService(repo).CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}, input)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	digest := strings.Repeat("a", 64)
	discovered := VendorBrandAsset{ID: "discovered", TenantID: "bank", VendorID: created.Vendor.ID, SourceKind: VendorBrandAssetDiscovered, State: VendorBrandAssetCurrent, SourceDomain: "vendor.example", ArtifactKey: vendorBrandObjectKey("bank", created.Vendor.ID, 1, digest), SourceDigest: digest, MediaType: "image/png", PixelWidth: 16, PixelHeight: 16, ByteSize: 20, UpdatedAt: now, Version: 1, AssetToken: strings.Repeat("b", 64)}
	repo.vendorBrandAssets[discovered.ID] = discovered
	for i := 0; i < 1001; i++ {
		repo.vendorBrandAssets[fmt.Sprintf("old-%04d", i)] = VendorBrandAsset{ID: fmt.Sprintf("old-%04d", i), TenantID: "bank", VendorID: created.Vendor.ID, SourceKind: VendorBrandAssetApprovedOverride, State: VendorBrandAssetSuperseded, UpdatedAt: now.Add(time.Duration(i+1) * time.Second)}
	}
	repo.vendorBrandEvents = append(repo.vendorBrandEvents, VendorBrandEvent{TenantID: "bank", VendorID: created.Vendor.ID, EventVersion: 7})
	projection, err := repo.GetVendorBrandProjection(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity"}, created.Vendor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if projection.CurrentDiscovered == nil || projection.CurrentDiscovered.ID != discovered.ID || projection.EventVersion != 7 {
		t.Fatalf("projection = %#v", projection)
	}
}

func TestBulkCurrentBrandProjectionReturnsOneSnapshotPerVisibleVendor(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo)
	actor := Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "owner"}
	first, err := service.CreateRelationship(context.Background(), actor, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	input := validCreateInput()
	input.SourceID = "other"
	input.ExternalRef = "other"
	second, err := service.CreateRelationship(context.Background(), actor, input)
	if err != nil {
		t.Fatal(err)
	}
	values, err := repo.GetVendorBrandProjections(context.Background(), Scope{TenantID: "bank", LegalEntityID: "entity"}, []string{first.Vendor.ID, second.Vendor.ID})
	if err != nil || len(values) != 2 {
		t.Fatalf("bulk projections = %#v, %v", values, err)
	}
}

func testBrandPNG(t *testing.T, fill color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, fill)
		}
	}
	var body bytes.Buffer
	if err := png.Encode(&body, img); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}
