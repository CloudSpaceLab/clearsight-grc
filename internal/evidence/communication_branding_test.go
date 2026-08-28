package evidence

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestCommunicationBrandingStoresInspectedPNG(t *testing.T) {
	t.Parallel()

	var encoded bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 255})
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	objects := NewMemoryObjectStore()
	service := NewCommunicationBrandService(NewMemoryCommunicationBrandStore(), objects)
	asset, err := service.StoreLogo(context.Background(), CommunicationLogoUploadInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", FileName: "logo.png", MediaType: "image/png",
		AltText: "Bank logo", CreatedBy: "maker-a",
	}, bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatalf("store logo: %v", err)
	}
	if asset.Width != 2 || asset.Height != 1 || asset.MediaType != "image/png" || len(asset.DigestHex) != 64 {
		t.Fatalf("unexpected asset: %#v", asset)
	}
	if asset.ArtifactKey == "" || strings.Contains(asset.ArtifactKey, "://") {
		t.Fatalf("unsafe artifact key: %q", asset.ArtifactKey)
	}
	reader, err := objects.Open(context.Background(), asset.ArtifactKey)
	if err != nil {
		t.Fatalf("open stored logo: %v", err)
	}
	_ = reader.Close()
}

func TestCommunicationBrandingRejectsNonPNG(t *testing.T) {
	t.Parallel()

	service := NewCommunicationBrandService(NewMemoryCommunicationBrandStore(), NewMemoryObjectStore())
	_, err := service.StoreLogo(context.Background(), CommunicationLogoUploadInput{
		TenantID: "tenant-a", LegalEntityID: "entity-a", FileName: "logo.png", MediaType: "image/png",
		AltText: "Bank logo", CreatedBy: "maker-a",
	}, strings.NewReader("not a png"))
	if err == nil {
		t.Fatal("expected invalid branding to be rejected")
	}
}
