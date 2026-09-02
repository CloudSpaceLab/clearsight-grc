package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
)

type runtimeContextStub struct {
	value runtimecontext.DisplayContext
	err   error
	scope runtimecontext.Scope
}

func (s *runtimeContextStub) Resolve(_ context.Context, scope runtimecontext.Scope) (runtimecontext.DisplayContext, error) {
	s.scope = scope
	return s.value, s.err
}

func TestDemoLoginRoutesAreAbsentOutsideDemoMode(t *testing.T) {
	authenticator, err := identity.NewDemoAuthenticator("bank-demo", "role-cro", "bank-ng")
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: authenticator,
		DemoMode: false,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/demo/accounts", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("demo account catalogue leaked outside demo mode: %d %s", response.Code, response.Body.String())
	}
}

func TestDemoLoginCreatesRoleSessionAndLogoutClearsIt(t *testing.T) {
	authenticator, err := identity.NewDemoAuthenticator("bank-demo", "role-cro", "bank-ng")
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: authenticator, DemoMode: true, Mode: "memory",
		RuntimeContext: &runtimeContextStub{value: runtimecontext.DisplayContext{
			TenantName: "Demo tenant", LegalEntityName: "Demo legal entity", PrincipalName: "System administrator",
		}},
	})

	accounts := httptest.NewRecorder()
	handler.ServeHTTP(accounts, httptest.NewRequest(http.MethodGet, "/api/v1/demo/accounts", nil))
	if accounts.Code != http.StatusOK || !strings.Contains(accounts.Body.String(), `"system-admin@demo.clearsight.local"`) || !strings.Contains(accounts.Body.String(), `"password":"demo"`) {
		t.Fatalf("demo catalogue missing role credentials: %d %s", accounts.Code, accounts.Body.String())
	}

	signedOutStatus := httptest.NewRecorder()
	handler.ServeHTTP(signedOutStatus, httptest.NewRequest(http.MethodGet, "/api/v1/session/status", nil))
	if signedOutStatus.Code != http.StatusOK || signedOutStatus.Body.String() != "{\"authenticated\":false,\"demo_login_available\":true}\n" {
		t.Fatalf("signed-out session status = %d %s", signedOutStatus.Code, signedOutStatus.Body.String())
	}

	bad := httptest.NewRecorder()
	badRequest := httptest.NewRequest(http.MethodPost, "/api/v1/demo/login", strings.NewReader(`{"username":"cro@demo.clearsight.local","password":"wrong"}`))
	badRequest.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(bad, badRequest)
	if bad.Code != http.StatusUnauthorized {
		t.Fatalf("invalid demo credential returned %d: %s", bad.Code, bad.Body.String())
	}

	login := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/demo/login", strings.NewReader(`{"username":"system-admin@demo.clearsight.local","password":"demo"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(login, loginRequest)
	if login.Code != http.StatusOK {
		t.Fatalf("demo login returned %d: %s", login.Code, login.Body.String())
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one demo session cookie, got %#v", cookies)
	}

	contextResponse := httptest.NewRecorder()
	contextRequest := httptest.NewRequest(http.MethodGet, "/api/v1/context", nil)
	contextRequest.AddCookie(cookies[0])
	handler.ServeHTTP(contextResponse, contextRequest)
	if contextResponse.Code != http.StatusOK || !strings.Contains(contextResponse.Body.String(), `"SYSTEM_ADMIN"`) || !strings.Contains(contextResponse.Body.String(), `"identity_configure":true`) {
		t.Fatalf("demo session did not reach role-aware context: %d %s", contextResponse.Code, contextResponse.Body.String())
	}

	signedInStatus := httptest.NewRecorder()
	signedInStatusRequest := httptest.NewRequest(http.MethodGet, "/api/v1/session/status", nil)
	signedInStatusRequest.AddCookie(cookies[0])
	handler.ServeHTTP(signedInStatus, signedInStatusRequest)
	if signedInStatus.Code != http.StatusOK || signedInStatus.Body.String() != "{\"authenticated\":true,\"demo_login_available\":true}\n" {
		t.Fatalf("signed-in session status = %d %s", signedInStatus.Code, signedInStatus.Body.String())
	}

	logout := httptest.NewRecorder()
	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/demo/logout", nil)
	logoutRequest.AddCookie(cookies[0])
	handler.ServeHTTP(logout, logoutRequest)
	if logout.Code != http.StatusNoContent || len(logout.Result().Cookies()) != 1 || logout.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("demo logout did not expire session: %d %#v", logout.Code, logout.Result().Cookies())
	}
}

func TestActorContextUsesStoredWorkspaceNamesForVerifiedScope(t *testing.T) {
	authenticator, err := identity.NewDemoAuthenticator(identity.DurableDemoTenantID, identity.DurableDemoPrincipalCRO, identity.DurableDemoLegalEntityID)
	if err != nil {
		t.Fatal(err)
	}
	resolver := &runtimeContextStub{value: runtimecontext.DisplayContext{
		TenantName: "Stored Bank", LegalEntityName: "Stored Bank Nigeria", PrincipalName: "Stored Risk Officer",
	}}
	handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: authenticator, DemoMode: true, Mode: "postgres", RuntimeContext: resolver})
	login := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/demo/login", strings.NewReader(`{"username":"cro@demo.clearsight.local","password":"demo"}`))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(login, request)
	if login.Code != http.StatusOK {
		t.Fatalf("demo login returned %d: %s", login.Code, login.Body.String())
	}

	response := httptest.NewRecorder()
	contextRequest := httptest.NewRequest(http.MethodGet, "/api/v1/context", nil)
	contextRequest.AddCookie(login.Result().Cookies()[0])
	handler.ServeHTTP(response, contextRequest)
	for _, expected := range []string{"Stored Bank", "Stored Bank Nigeria", "Stored Risk Officer"} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("stored context missing %q: %s", expected, response.Body.String())
		}
	}
	if resolver.scope != (runtimecontext.Scope{TenantID: identity.DurableDemoTenantID, LegalEntityID: identity.DurableDemoLegalEntityID, PrincipalID: identity.DurableDemoPrincipalCRO}) {
		t.Fatalf("resolver scope = %#v", resolver.scope)
	}
}

func TestActorContextDoesNotInventLabelsWhenDirectoryScopeIsMissing(t *testing.T) {
	handler := New(Dependencies{
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity:       identity.NewDevelopmentAuthenticator("tenant-a", "principal-a", "entity-a"),
		RuntimeContext: &runtimeContextStub{err: runtimecontext.ErrNotFound},
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/context", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	for _, invented := range []string{"Clear Bank", "Chief Risk Officer", "Amaka Okafor"} {
		if strings.Contains(response.Body.String(), invented) {
			t.Fatalf("response invented %q: %s", invented, response.Body.String())
		}
	}
}
