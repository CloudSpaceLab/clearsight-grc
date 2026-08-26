package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestResponseHistoryHandlerReturnsBoundedSafeHistoryAndHidesOtherEntity(t *testing.T) {
	service := continuity.NewService(continuity.NewMemoryRepository())
	trusted := continuity.WithTrustedSystemScope(context.Background())
	matter, err := service.CreateMatter(trusted, continuity.CreateMatterInput{TenantID: "bank", LegalEntityID: "entity-a", Type: continuity.MatterAuthorityRequest, Priority: 3, Title: "Response", Summary: "Prepare response.", Scope: json.RawMessage(`{}`)})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddResponsePackage(trusted, continuity.AddResponsePackageInput{TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version, Purpose: "Provide records", Audience: "Regulator", Manifest: json.RawMessage(`[]`), ActorID: "internal-person-id"})
	if err != nil {
		t.Fatal(err)
	}
	responseID := matter.ResponsePackages[0].ID
	api := &API{deps: Dependencies{Continuity: service}}

	request := httptest.NewRequest(http.MethodGet, "/?tenant_id=bank&limit=20", nil)
	request.SetPathValue("id", matter.Matter.ID)
	request.SetPathValue("response_id", responseID)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", PrincipalID: "reader", LegalEntityID: "entity-a"}))
	response := httptest.NewRecorder()
	api.getMatterResponseHistory(response, request)
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "internal-person-id") || !strings.Contains(response.Body.String(), `"has_more":false`) {
		t.Fatalf("unexpected response history: %d %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/?tenant_id=bank", nil)
	request.SetPathValue("id", matter.Matter.ID)
	request.SetPathValue("response_id", responseID)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: "bank", PrincipalID: "other", LegalEntityID: "entity-b"}))
	response = httptest.NewRecorder()
	api.getMatterResponseHistory(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected generic not found, got %d: %s", response.Code, response.Body.String())
	}
}
