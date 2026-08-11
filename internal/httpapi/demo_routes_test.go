package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

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
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: authenticator,
		DemoMode: true,
		Mode:     "memory",
	})

	accounts := httptest.NewRecorder()
	handler.ServeHTTP(accounts, httptest.NewRequest(http.MethodGet, "/api/v1/demo/accounts", nil))
	if accounts.Code != http.StatusOK || !strings.Contains(accounts.Body.String(), `"system-admin@demo.clearsight.local"`) || !strings.Contains(accounts.Body.String(), `"password":"demo"`) {
		t.Fatalf("demo catalogue missing role credentials: %d %s", accounts.Code, accounts.Body.String())
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

	logout := httptest.NewRecorder()
	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/demo/logout", nil)
	logoutRequest.AddCookie(cookies[0])
	handler.ServeHTTP(logout, logoutRequest)
	if logout.Code != http.StatusNoContent || len(logout.Result().Cookies()) != 1 || logout.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("demo logout did not expire session: %d %#v", logout.Code, logout.Result().Cookies())
	}
}
