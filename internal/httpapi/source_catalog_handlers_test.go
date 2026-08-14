package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func TestSourceCatalogCreateConnectionUsesVerifiedActorScope(t *testing.T) {
	tenantID := "11111111-1111-7111-8111-111111111111"
	sourceID := "12222222-2222-7222-8222-222222222222"
	principalID := "13333333-3333-7333-8333-333333333333"
	repository := sourceaccess.NewMemoryCatalogRepository([]sourceaccess.SourceScope{{TenantID: tenantID, SourceID: sourceID}})
	service := sourceaccess.NewCatalogService(repository, nil, sourceaccess.DefaultCatalogAdapters())
	api := &API{deps: Dependencies{SourceCatalog: service}}
	body := `{"code":"CORE_BANKING","name":"Core banking","adapter_kind":"POSTGRES","adapter_version":"postgres-v1","secret_ref":"env://CORE_BANKING_READER_DSN","declared_capabilities":["INSPECT","PAGE"]}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/config/sources/"+sourceID+"/connections", strings.NewReader(body))
	request.SetPathValue("source_id", sourceID)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{TenantID: tenantID, PrincipalID: principalID, ExpiresAt: time.Now().Add(time.Hour)}))
	response := httptest.NewRecorder()

	api.createSourceConnectionDraft(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var created sourceaccess.ConnectionRevision
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.TenantID != tenantID || created.SourceID != sourceID || created.CreatedBy != principalID || created.Status != sourceaccess.RevisionDraft || created.IsCurrent {
		t.Fatalf("handler allowed client-owned scope or lifecycle: %#v", created)
	}
}

func TestSourceCatalogRoutesUseConfigurationPermissions(t *testing.T) {
	api := &API{}
	expected := map[string]string{
		http.MethodGet + " /api/v1/config/sources/{source_id}/connections":               identity.PermissionConfigRead,
		http.MethodPost + " /api/v1/config/sources/{source_id}/connections":              identity.PermissionConfigWrite,
		http.MethodGet + " /api/v1/config/source-connections/{connection_id}":             identity.PermissionConfigRead,
		http.MethodGet + " /api/v1/config/source-connections/{connection_id}/views":       identity.PermissionConfigRead,
		http.MethodPost + " /api/v1/config/source-connections/{connection_id}/views":      identity.PermissionConfigWrite,
		http.MethodGet + " /api/v1/config/source-connections/{connection_id}/where-used":  identity.PermissionConfigRead,
		http.MethodGet + " /api/v1/config/source-views/{view_id}":                          identity.PermissionConfigRead,
		http.MethodPost + " /api/v1/config/source-views/{view_id}/inspect":                 identity.PermissionConfigWrite,
		http.MethodGet + " /api/v1/config/source-views/{view_id}/bindings":                 identity.PermissionConfigRead,
		http.MethodPost + " /api/v1/config/source-views/{view_id}/bindings":                identity.PermissionConfigWrite,
		http.MethodGet + " /api/v1/config/source-views/{view_id}/where-used":               identity.PermissionConfigRead,
		http.MethodGet + " /api/v1/config/source-bindings/{binding_id}":                    identity.PermissionConfigRead,
		http.MethodPost + " /api/v1/config/source-bindings/{binding_id}/preview":           identity.PermissionConfigRead,
		http.MethodGet + " /api/v1/config/source-bindings/{binding_id}/where-used":         identity.PermissionConfigRead,
	}
	for _, route := range api.routes() {
		key := route.Method + " " + route.Path
		permission, exists := expected[key]
		if !exists {
			continue
		}
		if route.Permission != permission {
			t.Fatalf("%s permission=%q want=%q", key, route.Permission, permission)
		}
		delete(expected, key)
	}
	if len(expected) != 0 {
		t.Fatalf("missing source catalog routes: %#v", expected)
	}
}

func TestSourceCatalogErrorsDoNotLeakConnectionDetails(t *testing.T) {
	response := httptest.NewRecorder()
	writeSourceCatalogError(response, errors.Join(sourceaccess.ErrConnection, errors.New("postgres://reader:secret@bank-core/risk query SELECT * FROM customers")))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", response.Code)
	}
	body := response.Body.String()
	for _, secret := range []string{"reader", "secret", "bank-core", "SELECT", "customers"} {
		if strings.Contains(body, secret) {
			t.Fatalf("response leaked %q: %s", secret, body)
		}
	}
}

func TestCatalogVersionRejectsNonPositiveValues(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/?version=0", nil)
	response := httptest.NewRecorder()
	if _, ok := catalogVersion(response, request); ok || response.Code != http.StatusBadRequest {
		t.Fatalf("invalid version accepted: ok=%v status=%d", ok, response.Code)
	}
	request = httptest.NewRequest(http.MethodGet, "/?version=2", nil)
	response = httptest.NewRecorder()
	value, ok := catalogVersion(response, request)
	if !ok || value != 2 {
		t.Fatalf("valid version rejected: value=%d ok=%v", value, ok)
	}
}

func TestSourceCatalogRequiresIdentityEvenWhenCalledDirectly(t *testing.T) {
	api := &API{deps: Dependencies{SourceCatalog: sourceaccess.NewCatalogService(sourceaccess.NewMemoryCatalogRepository(nil), nil, nil)}}
	request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(context.Background())
	request.SetPathValue("source_id", "source")
	response := httptest.NewRecorder()
	api.listSourceConnections(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
