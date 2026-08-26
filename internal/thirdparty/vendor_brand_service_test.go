package thirdparty

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	service.now = func() time.Time { return now.Add(time.Minute) }
	ctx := vendorIdentityContext("bank", "entity", "verified-owner", now)

	view, err := service.PutApprovedBrand(ctx, created.Vendor.ID, 0, "upload-1", "image/png", bytes.NewReader(testBrandPNG(t, color.RGBA{R: 200, G: 10, B: 30, A: 255})))
	if err != nil {
		t.Fatal(err)
	}
	if view.Brand.State != VendorBrandApprovedLogo || view.Brand.Source != VendorBrandSourceApprovedUpload || view.Brand.Version != 1 || view.Brand.AssetToken == "" {
		t.Fatalf("approved presentation = %#v", view.Brand)
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

func TestVendorBrandPresentationIsUnavailableWhenDiscoveryIsDisabled(t *testing.T){repo:=NewMemoryRepository();base:=NewService(repo);input:=validCreateInput();input.WebsiteDomain="vendor.example";created,err:=base.CreateRelationship(context.Background(),Actor{TenantID:"bank",LegalEntityID:"entity",PrincipalID:"owner"},input);if err!=nil{t.Fatal(err)};service:=NewVendorBrandService(repo,evidence.NewMemoryObjectStore(),&vendorIdentityGuardStub{});service.ConfigureDiscoveryEnabled(false);view,err:=service.GetIdentity(context.Background(),Actor{TenantID:"bank",LegalEntityID:"entity",PrincipalID:"owner"},created.Vendor.ID);if err!=nil{t.Fatal(err)};if view.Brand.State!=VendorBrandUnavailable{t.Fatalf("state=%s",view.Brand.State)}}

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

func TestVendorBrandReservationCleanerPreservesObjectReferencedByAnotherReservation(t *testing.T){repo:=NewMemoryRepository();store:=evidence.NewMemoryObjectStore();base:=NewService(repo);created,err:=base.CreateRelationship(context.Background(),Actor{TenantID:"bank",LegalEntityID:"entity",PrincipalID:"owner"},validCreateInput());if err!=nil{t.Fatal(err)};now:=time.Now().UTC();key:="vendor-brands/shared.png";if _,err=store.Put(context.Background(),key,bytes.NewBufferString("shared"),100);err!=nil{t.Fatal(err)};oldKey:="bank\x00"+created.Vendor.ID+"\x00old";repo.vendorBrandReservations[oldKey]=VendorBrandUploadReservation{TenantID:"bank",VendorID:created.Vendor.ID,IdempotencyKey:"old",ArtifactKey:key,SourceDigest:strings.Repeat("a",64),State:"RESERVED",CreatedAt:now.Add(-48*time.Hour),UpdatedAt:now.Add(-48*time.Hour)};repo.vendorBrandReservations["bank\x00"+created.Vendor.ID+"\x00new"]=VendorBrandUploadReservation{TenantID:"bank",VendorID:created.Vendor.ID,IdempotencyKey:"new",ArtifactKey:key,SourceDigest:strings.Repeat("a",64),State:"RESERVED",CreatedAt:now,UpdatedAt:now};cleaner:=&VendorBrandReservationCleaner{Repository:repo,Store:store,Retention:24*time.Hour};if _,err=cleaner.Maintain(context.Background(),now,1);err!=nil{t.Fatal(err)};reader,err:=store.Open(context.Background(),key);if err!=nil{t.Fatalf("shared object deleted: %v",err)};reader.Close()}

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
