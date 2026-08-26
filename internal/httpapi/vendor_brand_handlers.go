package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func (a *API) vendorBrandService(w http.ResponseWriter) (*thirdparty.VendorBrandService, bool) {
	if a.deps.VendorBrands == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "vendor_brand_unavailable", "Vendor brand assets are temporarily unavailable. Vendor records remain available.")
		return nil, false
	}
	return a.deps.VendorBrands, true
}

func (a *API) getVendorIdentity(w http.ResponseWriter, r *http.Request) {
	service, ok := a.vendorBrandService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to view this vendor identity.")
		return
	}
	view, err := service.GetIdentity(r.Context(), actor, r.PathValue("vendor_id"))
	if err != nil {
		writeVendorIdentityError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, view)
}

func (a *API) updateVendorIdentity(w http.ResponseWriter, r *http.Request) {
	service, ok := a.thirdPartyService(w)
	if !ok {
		return
	}
	brands, ok := a.vendorBrandService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to update this vendor identity.")
		return
	}
	var request struct {
		thirdparty.UpdateVendorIdentityInput
		ActorID       string `json:"actor_id,omitempty"`
		TenantID      string `json:"tenant_id,omitempty"`
		LegalEntityID string `json:"legal_entity_id,omitempty"`
	}
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check the vendor identity fields and submit one JSON object.")
		return
	}
	vendor, err := service.UpdateVendorIdentity(r.Context(), actor, r.PathValue("vendor_id"), request.UpdateVendorIdentityInput)
	if err != nil {
		writeVendorIdentityError(w, err)
		return
	}
	view := brands.IdentityForVendorBestEffort(r.Context(), actor, vendor)
	httpx.WriteJSON(w, http.StatusOK, view)
}

func writeVendorIdentityError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, thirdparty.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "vendor_identity_not_found", "This vendor identity was not found in your current legal-entity scope.")
	case errors.Is(err, thirdparty.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "vendor_identity_version_conflict", "This vendor identity changed. Reload it before saving again.")
	case errors.Is(err, thirdparty.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "vendor_identity_invalid", "Check the vendor name, registration details, jurisdiction and website domain.")
	default:
		writeVendorBrandError(w, err)
	}
}

func (a *API) uploadVendorBrand(w http.ResponseWriter, r *http.Request) {
	service, ok := a.vendorBrandService(w)
	if !ok {
		return
	}
	version, err := parseBrandIfMatch(r)
	if err != nil {
		writeBrandIfMatchError(w, err)
		return
	}
	if strings.TrimSpace(r.Header.Get("Idempotency-Key")) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "idempotency_key_required", "Provide an Idempotency-Key before uploading the approved vendor icon.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, thirdparty.VendorBrandMaximumUploadBytes+1)
	view, err := service.PutApprovedBrand(r.Context(), r.PathValue("vendor_id"), version, strings.TrimSpace(r.Header.Get("Idempotency-Key")), r.Header.Get("Content-Type"), r.Body)
	if err != nil {
		writeVendorBrandError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, view)
}

func (a *API) removeVendorBrand(w http.ResponseWriter, r *http.Request) {
	service, ok := a.vendorBrandService(w)
	if !ok {
		return
	}
	version, err := parseBrandIfMatch(r)
	if err != nil {
		writeBrandIfMatchError(w, err)
		return
	}
	if strings.TrimSpace(r.Header.Get("Idempotency-Key")) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "idempotency_key_required", "Provide an Idempotency-Key before removing the approved vendor icon.")
		return
	}
	view, err := service.RemoveApprovedBrand(r.Context(), r.PathValue("vendor_id"), version, strings.TrimSpace(r.Header.Get("Idempotency-Key")))
	if err != nil {
		writeVendorBrandError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, view)
}

func (a *API) openVendorBrand(w http.ResponseWriter, r *http.Request) {
	service, ok := a.vendorBrandService(w)
	if !ok {
		return
	}
	actor, err := thirdPartyActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to view this vendor icon.")
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("version"))
	asset, reader, err := service.OpenBrand(r.Context(), actor, r.PathValue("vendor_id"), token)
	if err != nil {
		writeVendorBrandError(w, err)
		return
	}
	defer reader.Close()
	etag := `"` + thirdparty.BrandAssetToken(asset) + `"`
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("ETag", etag)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if token != "" {
		w.Header().Set("Cache-Control", "private, max-age=86400, immutable")
	} else {
		w.Header().Set("Cache-Control", "private, no-cache")
	}
	w.Header().Set("Content-Length", strconv.FormatInt(asset.ByteSize, 10))
	w.WriteHeader(http.StatusOK)
	_, _ = io.CopyN(w, reader, asset.ByteSize)
}

func parseBrandIfMatch(r *http.Request) (int64, error) {
	raw := strings.TrimSpace(r.Header.Get("If-Match"))
	if raw == "" {
		return 0, errBrandIfMatchMissing
	}
	if len(raw) < 3 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return 0, errBrandIfMatchInvalid
	}
	value := raw[1 : len(raw)-1]
	if value == "" {
		return 0, errBrandIfMatchInvalid
	}
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return 0, errBrandIfMatchInvalid
		}
	}
	version, err := strconv.ParseInt(value, 10, 64)
	if err != nil || version < 0 {
		return 0, errBrandIfMatchInvalid
	}
	return version, nil
}

var (
	errBrandIfMatchMissing = errors.New("missing If-Match")
	errBrandIfMatchInvalid = errors.New("invalid If-Match")
)

func writeBrandIfMatchError(w http.ResponseWriter, err error) {
	if errors.Is(err, errBrandIfMatchMissing) {
		httpx.WriteError(w, http.StatusPreconditionRequired, "brand_version_required", "Provide the current vendor brand version in If-Match.")
		return
	}
	httpx.WriteError(w, http.StatusBadRequest, "brand_version_invalid", "Use a non-negative vendor brand version in If-Match.")
}

func writeVendorBrandError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, thirdparty.ErrNotFound), errors.Is(err, thirdparty.ErrVendorBrandAssetUnavailable):
		httpx.WriteError(w, http.StatusNotFound, "vendor_brand_not_found", "No vendor icon is available for this vendor in your current legal-entity scope.")
	case errors.Is(err, thirdparty.ErrBrandVersionConflict), errors.Is(err, thirdparty.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "vendor_brand_version_conflict", "The vendor brand changed. Reload the vendor before trying again.")
	case errors.Is(err, thirdparty.ErrVendorBrandOverrideNotFound):
		httpx.WriteError(w, http.StatusNotFound, "approved_vendor_icon_not_found", "No approved vendor icon is available to remove.")
	case errors.Is(err, thirdparty.ErrIdempotencyKeyRequired):
		httpx.WriteError(w, http.StatusBadRequest, "idempotency_key_required", "Provide an Idempotency-Key before changing the approved vendor icon.")
	case errors.Is(err, thirdparty.ErrVendorIdentityAuthorityUnavailable), errors.Is(err, commandauth.ErrGuardUnavailable):
		httpx.WriteError(w, http.StatusServiceUnavailable, "authority_unavailable", "The authority route could not be checked. No change was made.")
	case errors.Is(err, commandauth.ErrNotAuthorized), errors.Is(err, thirdparty.ErrVendorIdentityMismatch):
		httpx.WriteError(w, http.StatusForbidden, "approval_required", "You are not currently authorized to change this vendor identity.")
	case errors.Is(err, evidence.ErrArtifactTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "vendor_brand_too_large", "Upload an approved icon no larger than 512 KiB.")
	case errors.Is(err, thirdparty.ErrUnsupportedVendorBrandMedia), errors.Is(err, thirdparty.ErrInvalidVendorBrandImage), errors.Is(err, thirdparty.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "vendor_brand_invalid", "Upload a valid PNG, JPEG, WebP or ICO image within the supported dimensions.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "vendor_brand_change_failed", "The vendor brand change could not be completed. No change was made.")
	}
}
