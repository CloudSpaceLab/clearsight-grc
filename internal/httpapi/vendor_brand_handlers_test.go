package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

type failingHTTPBrandProjectionRepository struct{ *thirdparty.MemoryRepository }

func (r *failingHTTPBrandProjectionRepository) GetVendorBrandProjection(context.Context, thirdparty.Scope, string) (thirdparty.VendorBrandProjection, error) {
	return thirdparty.VendorBrandProjection{}, errors.New("brand projection unavailable")
}

func vendorBrandTestHandler(t *testing.T) http.Handler {
	t.Helper()
	repo := thirdparty.NewMemoryRepository()
	store := evidence.NewMemoryObjectStore()
	guard, err := commandauth.New(nil, commandauth.ModeOff, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	service := thirdparty.NewService(repo)
	service.ConfigureIdentityAuthority(guard)
	return New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "test-memory", Identity: identity.NewDevelopmentAuthenticator("bank", "verified-owner", "entity-a"), CommandGuard: guard, ThirdParty: service, VendorBrands: thirdparty.NewVendorBrandService(repo, store, guard)})
}

func TestVendorIdentityAndBrandRoutesUseExplicitIdentityPath(t *testing.T) {
	handler := vendorBrandTestHandler(t)
	create := httptest.NewRecorder()
	handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/v1/vendors", bytes.NewBufferString(`{"legal_name":"Northstar Systems","website_domain":"northstar.example","service_name":"Application support","criticality":"IMPORTANT","privacy_role":"PROCESSOR"}`)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create = %d %s", create.Code, create.Body.String())
	}
	var aggregate thirdparty.Aggregate
	if err := json.NewDecoder(create.Body).Decode(&aggregate); err != nil {
		t.Fatal(err)
	}

	update := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/v1/vendor-identities/"+aggregate.Vendor.ID, bytes.NewBufferString(`{"expected_version":1,"legal_name":"Northstar Systems Limited","website_domain":"northstar.example","actor_id":"forged"}`))
	handler.ServeHTTP(update, updateRequest)
	if update.Code != http.StatusOK {
		t.Fatalf("identity update = %d %s", update.Code, update.Body.String())
	}
	var view thirdparty.VendorIdentityView
	if err := json.NewDecoder(update.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if view.Vendor.Version != 2 || view.Brand.State != thirdparty.VendorBrandPending {
		t.Fatalf("identity view = %#v", view)
	}

	upload := httptest.NewRecorder()
	uploadRequest := httptest.NewRequest(http.MethodPut, "/api/v1/vendor-identities/"+aggregate.Vendor.ID+"/brand", bytes.NewReader(httpBrandPNG(t)))
	uploadRequest.Header.Set("Content-Type", "image/png")
	uploadRequest.Header.Set("If-Match", `"0"`)
	uploadRequest.Header.Set("Idempotency-Key", "brand-upload-1")
	handler.ServeHTTP(upload, uploadRequest)
	if upload.Code != http.StatusOK {
		t.Fatalf("upload = %d %s", upload.Code, upload.Body.String())
	}
	if err := json.NewDecoder(upload.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if view.Brand.State != thirdparty.VendorBrandApprovedLogo || view.Brand.AssetToken == "" {
		t.Fatalf("brand view = %#v", view.Brand)
	}

	imageResponse := httptest.NewRecorder()
	imageRequest := httptest.NewRequest(http.MethodGet, "/api/v1/vendor-identities/"+aggregate.Vendor.ID+"/brand?version="+view.Brand.AssetToken, nil)
	handler.ServeHTTP(imageResponse, imageRequest)
	if imageResponse.Code != http.StatusOK || imageResponse.Header().Get("Content-Type") != "image/png" || imageResponse.Header().Get("ETag") == "" || imageResponse.Header().Get("Cache-Control") != "private, max-age=86400, immutable" {
		t.Fatalf("brand image = %d headers=%v body=%s", imageResponse.Code, imageResponse.Header(), imageResponse.Body.String())
	}

	notModified := httptest.NewRecorder()
	conditional := httptest.NewRequest(http.MethodGet, imageRequest.URL.String(), nil)
	conditional.Header.Set("If-None-Match", imageResponse.Header().Get("ETag"))
	handler.ServeHTTP(notModified, conditional)
	if notModified.Code != http.StatusNotModified || notModified.Body.Len() != 0 {
		t.Fatalf("conditional = %d %q", notModified.Code, notModified.Body.String())
	}
}

func TestVendorBrandCurrentURLRevalidatesAndNeverDisclosesStorageMetadata(t *testing.T) {
	handler := vendorBrandTestHandler(t)
	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/v1/vendor-identities/unknown/brand", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing = %d %s", missing.Code, missing.Body.String())
	}
	if bytes.Contains(missing.Body.Bytes(), []byte("artifact")) || bytes.Contains(missing.Body.Bytes(), []byte("digest")) {
		t.Fatalf("storage metadata leaked: %s", missing.Body.String())
	}
}

func TestVendorIdentityUpdateReturnsCommittedRecordWhenBrandProjectionIsUnavailable(t *testing.T) {
	base := thirdparty.NewMemoryRepository()
	repository := &failingHTTPBrandProjectionRepository{MemoryRepository: base}
	guard, err := commandauth.New(nil, commandauth.ModeOff, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	identities := thirdparty.NewService(repository)
	identities.ConfigureIdentityAuthority(guard)
	actor := thirdparty.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "verified-owner"}
	created, err := identities.CreateRelationship(t.Context(), actor, thirdparty.CreateRelationshipInput{LegalName: "Northstar Systems", ServiceName: "Application support", Criticality: thirdparty.CriticalityImportant, PrivacyRole: thirdparty.PrivacyProcessor})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "test-memory",
		Identity:     identity.NewDevelopmentAuthenticator(actor.TenantID, actor.PrincipalID, actor.LegalEntityID),
		CommandGuard: guard, ThirdParty: identities,
		VendorBrands: thirdparty.NewVendorBrandService(repository, evidence.NewMemoryObjectStore(), guard),
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/vendor-identities/"+created.Vendor.ID, bytes.NewBufferString(`{"expected_version":1,"legal_name":"Northstar Systems Limited","website_domain":"northstar.example"}`))
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("identity update = %d %s", response.Code, response.Body.String())
	}
	var view thirdparty.VendorIdentityView
	if err := json.NewDecoder(response.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if view.Vendor.Version != 2 || view.Vendor.LegalName != "Northstar Systems Limited" || view.Brand.State != thirdparty.VendorBrandPending {
		t.Fatalf("committed identity response = %#v", view)
	}
}

func TestVendorBrandCommandHeaderErrorsAreDistinct(t *testing.T) {
	handler := vendorBrandTestHandler(t)
	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodDelete, "/api/v1/vendor-identities/vendor/brand", nil))
	if missing.Code != http.StatusPreconditionRequired || !bytes.Contains(missing.Body.Bytes(), []byte("brand_version_required")) {
		t.Fatalf("missing If-Match = %d %s", missing.Code, missing.Body.String())
	}
	malformed := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/vendor-identities/vendor/brand", nil)
	request.Header.Set("If-Match", "bad")
	handler.ServeHTTP(malformed, request)
	if malformed.Code != http.StatusBadRequest || !bytes.Contains(malformed.Body.Bytes(), []byte("brand_version_invalid")) {
		t.Fatalf("malformed If-Match = %d %s", malformed.Code, malformed.Body.String())
	}
	for _, value := range []string{`"1`, `1"`, `1`, `"+1"`, `"-0"`} {
		response := httptest.NewRecorder()
		request = httptest.NewRequest(http.MethodDelete, "/api/v1/vendor-identities/vendor/brand", nil)
		request.Header.Set("If-Match", value)
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || !bytes.Contains(response.Body.Bytes(), []byte("brand_version_invalid")) {
			t.Fatalf("If-Match %q = %d %s", value, response.Code, response.Body.String())
		}
	}
	missingKey := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodDelete, "/api/v1/vendor-identities/vendor/brand", nil)
	request.Header.Set("If-Match", `"0"`)
	handler.ServeHTTP(missingKey, request)
	if missingKey.Code != http.StatusBadRequest || !bytes.Contains(missingKey.Body.Bytes(), []byte("idempotency_key_required")) {
		t.Fatalf("missing idempotency key = %d %s", missingKey.Code, missingKey.Body.String())
	}
}

func httpBrandPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{R: 20, G: 90, B: 150, A: 255})
		}
	}
	var body bytes.Buffer
	if err := png.Encode(&body, img); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}
