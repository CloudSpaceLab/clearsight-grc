package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func thirdPartyTestHandler() http.Handler {
	return New(Dependencies{
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Mode:       "test-memory",
		Identity:   identity.NewDevelopmentAuthenticator("bank", "verified-owner", "entity-a"),
		ThirdParty: thirdparty.NewService(thirdparty.NewMemoryRepository()),
	})
}

func TestCreateVendorRelationshipUsesVerifiedOwnerAndScope(t *testing.T) {
	handler := thirdPartyTestHandler()
	body := []byte(`{"legal_name":"Acme Processing Limited","website_domain":"https://acme.example/about","registered_address":"1 Marina Road\nLagos, Nigeria","service_name":"Card transaction processing","criticality":"IMPORTANT","privacy_role":"PROCESSOR","tenant_id":"bank","legal_entity_id":"entity-a","actor_id":"verified-owner"}`)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendors", bytes.NewReader(body)))
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var value thirdparty.Aggregate
	if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
		t.Fatal(err)
	}
	if value.Relationship.TenantID != "bank" || value.Relationship.LegalEntityID != "entity-a" || value.Relationship.BusinessOwnerPrincipalID != "verified-owner" {
		t.Fatalf("command was not bound to verified identity: %#v", value.Relationship)
	}
	if value.Vendor.WebsiteDomain != "acme.example" || value.Vendor.RegisteredAddress != "1 Marina Road\nLagos, Nigeria" {
		t.Fatalf("vendor identity fields were not stored: %#v", value.Vendor)
	}
}

func TestCreateVendorRelationshipRejectsForgedScope(t *testing.T) {
	handler := thirdPartyTestHandler()
	body := []byte(`{"legal_name":"Acme Processing Limited","service_name":"Card transaction processing","criticality":"IMPORTANT","privacy_role":"PROCESSOR","tenant_id":"other-bank","legal_entity_id":"entity-b","actor_id":"forged-owner"}`)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendors", bytes.NewReader(body)))
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", response.Code, response.Body.String())
	}
}

func TestListVendorsRejectsConflictingLegalEntity(t *testing.T) {
	handler := thirdPartyTestHandler()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/vendors?legal_entity_id=entity-b", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected scope-safe 404, got %d: %s", response.Code, response.Body.String())
	}
}

func TestUpdateVendorRelationshipUsesRouteIDAndCurrentVersion(t *testing.T) {
	handler := thirdPartyTestHandler()
	createBody := []byte(`{"legal_name":"Acme Processing Limited","service_name":"Card processing","criticality":"IMPORTANT","privacy_role":"PROCESSOR"}`)
	createdResponse := httptest.NewRecorder()
	handler.ServeHTTP(createdResponse, httptest.NewRequest(http.MethodPost, "/api/v1/vendors", bytes.NewReader(createBody)))
	var created thirdparty.Aggregate
	if err := json.NewDecoder(createdResponse.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	updateBody := []byte(`{"expected_version":1,"service_name":"Card processing and settlement","criticality":"CRITICAL","privacy_role":"PROCESSOR"}`)
	updatedResponse := httptest.NewRecorder()
	handler.ServeHTTP(updatedResponse, httptest.NewRequest(http.MethodPost, "/api/v1/vendors/"+created.Relationship.ID, bytes.NewReader(updateBody)))
	if updatedResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", updatedResponse.Code, updatedResponse.Body.String())
	}
	var updated thirdparty.Aggregate
	if err := json.NewDecoder(updatedResponse.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Relationship.ID != created.Relationship.ID || updated.Relationship.Version != 2 {
		t.Fatalf("unexpected updated record %#v", updated)
	}
}
