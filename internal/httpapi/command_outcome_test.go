package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func TestMaterialHandlerReturnsReceiptForCommittedVendorBrandBinaryCommand(t *testing.T) {
	repo := thirdparty.NewMemoryRepository()
	relationships := thirdparty.NewService(repo)
	actor := thirdparty.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "owner-1"}
	created, err := relationships.CreateRelationship(t.Context(), actor, thirdparty.CreateRelationshipInput{LegalName: "Northstar Systems", ServiceName: "Application support", Criticality: thirdparty.CriticalityImportant, PrivacyRole: thirdparty.PrivacyProcessor})
	if err != nil {
		t.Fatal(err)
	}
	guard, err := commandauth.New(nil, commandauth.ModeOff, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	brands := thirdparty.NewVendorBrandService(repo, evidence.NewMemoryObjectStore(), guard)
	api := &API{deps: Dependencies{ThirdParty: relationships, VendorBrands: brands}}
	request := httptest.NewRequest(http.MethodPut, "/api/v1/vendor-identities/"+created.Vendor.ID+"/brand", nil)
	request.SetPathValue("vendor_id", created.Vendor.ID)
	request.Header.Set("Idempotency-Key", "upload")
	request = request.WithContext(verifiedCommandContext(request.Context(), actor))
	response := httptest.NewRecorder()
	policy := commandPolicy{ObjectType: thirdparty.VendorIdentityObjectType, OutcomeObjectType: "VENDOR_BRAND", OutcomePathValue: "vendor_id"}
	api.executeMaterialHandler(response, request, policy, map[string]any{"tenant_id": actor.TenantID, "expected_version": int64(0)}, func(w http.ResponseWriter, _ *http.Request) {
		if _, uploadErr := brands.PutApprovedBrand(request.Context(), created.Vendor.ID, 0, "upload", "image/png", bytes.NewReader(httpBrandPNG(t))); uploadErr != nil {
			t.Fatal(uploadErr)
		}
		http.Error(w, "simulated presentation failure", http.StatusInternalServerError)
	})
	assertCommittedReceipt(t, response, "VENDOR_BRAND", created.Vendor.ID, 1)
}

func TestMaterialHandlerDoesNotAttributeAnotherBrandCommandToThisRequest(t *testing.T) {
	repo := thirdparty.NewMemoryRepository()
	relationships := thirdparty.NewService(repo)
	actor := thirdparty.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "owner-1"}
	created, err := relationships.CreateRelationship(t.Context(), actor, thirdparty.CreateRelationshipInput{LegalName: "Northstar Systems", ServiceName: "Application support", Criticality: thirdparty.CriticalityImportant, PrivacyRole: thirdparty.PrivacyProcessor})
	if err != nil {
		t.Fatal(err)
	}
	guard, err := commandauth.New(nil, commandauth.ModeOff, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	brands := thirdparty.NewVendorBrandService(repo, evidence.NewMemoryObjectStore(), guard)
	api := &API{deps: Dependencies{ThirdParty: relationships, VendorBrands: brands}}
	request := httptest.NewRequest(http.MethodPut, "/api/v1/vendor-identities/"+created.Vendor.ID+"/brand", nil)
	request.SetPathValue("vendor_id", created.Vendor.ID)
	request.Header.Set("Idempotency-Key", "this-request")
	request = request.WithContext(verifiedCommandContext(request.Context(), actor))
	response := httptest.NewRecorder()
	policy := commandPolicy{ObjectType: thirdparty.VendorIdentityObjectType, OutcomeObjectType: "VENDOR_BRAND", OutcomePathValue: "vendor_id"}
	api.executeMaterialHandler(response, request, policy, map[string]any{"tenant_id": actor.TenantID, "expected_version": int64(0)}, func(w http.ResponseWriter, _ *http.Request) {
		if _, uploadErr := brands.PutApprovedBrand(request.Context(), created.Vendor.ID, 0, "another-request", "image/png", bytes.NewReader(httpBrandPNG(t))); uploadErr != nil {
			t.Fatal(uploadErr)
		}
		http.Error(w, "genuine failure for this request", http.StatusInternalServerError)
	})

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("unrelated command was attributed to this request: %d %s", response.Code, response.Body.String())
	}
}

func TestMaterialHandlerPreservesFailureWhenNoWriteCommitted(t *testing.T) {
	repo := continuity.NewMemoryRepository()
	service := continuity.NewService(repo)
	matter, err := service.CreateMatter(t.Context(), continuity.CreateMatterInput{
		TenantID: "bank-demo", Type: continuity.MatterControlGap, Priority: 3,
		Title: "No commit", Summary: "Verify genuine failure handling", Scope: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	api := &API{deps: Dependencies{Continuity: service}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/actions", strings.NewReader(`{}`))
	request.SetPathValue("id", matter.Matter.ID)
	response := httptest.NewRecorder()
	payload := map[string]any{"tenant_id": "bank-demo", "expected_version": float64(matter.Matter.Version)}
	policy := commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner}

	api.executeMaterialHandler(response, request, policy, payload, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "genuine failure", http.StatusInternalServerError)
	})

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected genuine failure, got %d: %s", response.Code, response.Body.String())
	}
}

func verifiedCommandContext(ctx context.Context, actor thirdparty.Actor) context.Context {
	now := time.Now().UTC()
	return identity.WithActor(ctx, identity.Actor{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID, Kind: "PERSON", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)})
}

func assertCommittedReceipt(t *testing.T, response *httptest.ResponseRecorder, aggregateType, aggregateID string, version int64) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("expected committed receipt, got %d: %s", response.Code, response.Body.String())
	}
	var receipt committedCommandReceipt
	if err := json.Unmarshal(response.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.AggregateType != aggregateType || receipt.AggregateID != aggregateID || receipt.Version != version || !receipt.ResponseDegraded {
		t.Fatalf("unexpected receipt %#v", receipt)
	}
}
