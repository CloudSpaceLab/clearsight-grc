package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestImportReadGuardDeniesUnavailableActorScopedStore(t *testing.T) {
	api := &API{deps: Dependencies{DocumentImports: documentimport.NewService(documentimport.NewMemoryRepository(), nil)}}
	called := false
	guarded := api.requireDocumentImportRead(func(http.ResponseWriter, *http.Request) { called = true })
	request := httptest.NewRequest(http.MethodGet, "/api/v1/document-imports/doc/coverage", nil)
	request.SetPathValue("id", "doc")
	response := httptest.NewRecorder()
	guarded(response, request)
	if response.Code != http.StatusUnauthorized || called {
		t.Fatalf("anonymous=%d called=%v", response.Code, called)
	}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "person", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}
	request = request.WithContext(identity.WithActor(request.Context(), actor))
	response = httptest.NewRecorder()
	guarded(response, request)
	if response.Code != http.StatusServiceUnavailable || called || response.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("unavailable=%d called=%v", response.Code, called)
	}
}

func TestImportCreationFailsClosedUntilTransactionBoundAuthorityExists(t *testing.T) {
	for name, register := range map[string]func(*API, *http.ServeMux){
		"routes":     (*API).registerRoutes,
		"production": (*API).registerProductionRoutes,
	} {
		t.Run(name, func(t *testing.T) {
			api := &API{deps: Dependencies{DocumentImports: documentimport.NewService(documentimport.NewMemoryRepository(), nil)}}
			mux := http.NewServeMux()
			register(api, mux)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/document-imports", nil)
			actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "person", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}
			request = request.WithContext(identity.WithActor(request.Context(), actor))
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusServiceUnavailable || response.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatalf("unguarded import creation: status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}
