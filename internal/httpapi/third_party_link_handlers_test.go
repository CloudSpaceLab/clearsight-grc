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

func thirdPartyLinkTestHandler() http.Handler {
	repo := thirdparty.NewMemoryRelationshipLinkRepository()
	repo.AllowRelationship("bank", "entity-a", "relationship-1")
	repo.AllowTarget("bank", "entity-a", thirdparty.LinkTargetProgram, "program-1")
	return New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Mode: "test-memory",
		Identity:                    identity.NewDevelopmentAuthenticator("bank", "verified-owner", "entity-a", "BUSINESS_OWNER"),
		ThirdPartyRelationshipLinks: thirdparty.NewRelationshipLinkService(repo),
	})
}

func TestVendorRelationshipLinkUsesVerifiedScope(t *testing.T) {
	handler := thirdPartyLinkTestHandler()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendors/relationship-1/links", bytes.NewBufferString(`{"target_type":"PROGRAM","target_id":"program-1","purpose_code":"RELEASE_SUPPORT","purpose_label":"Release support","tenant_id":"other","actor_id":"forged"}`)))
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected forged tenant rejection, got %d: %s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/vendors/relationship-1/links", bytes.NewBufferString(`{"target_type":"PROGRAM","target_id":"program-1","purpose_code":"RELEASE_SUPPORT","purpose_label":"Release support"}`)))
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var link thirdparty.RelationshipLink
	if err := json.NewDecoder(response.Body).Decode(&link); err != nil {
		t.Fatal(err)
	}
	if link.TenantID != "bank" || link.LegalEntityID != "entity-a" || link.CreatedBy != "verified-owner" {
		t.Fatalf("link did not use verified scope: %#v", link)
	}

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/vendors/relationship-1/links", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d: %s", list.Code, list.Body.String())
	}
}
