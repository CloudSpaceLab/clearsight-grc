package federation

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"golang.org/x/oauth2"
)

type fakeAccessResolver struct {
	resolution access.Resolution
	err        error
	calls      int
}

func (r *fakeAccessResolver) ResolveOIDC(context.Context, string, string, string, string) (access.Resolution, error) {
	r.calls++
	return r.resolution, r.err
}

func (r *fakeAccessResolver) ResolvePrincipal(context.Context, string, string, string) (access.Resolution, error) {
	r.calls++
	return r.resolution, r.err
}

func newTestSessions() *scs.SessionManager {
	sessions := scs.New()
	sessions.Store = memstore.New()
	sessions.Lifetime = time.Hour
	sessions.IdleTimeout = 30 * time.Minute
	return sessions
}

func TestAuthenticateRefreshesCurrentLocalAccess(t *testing.T) {
	now := time.Date(2026, 8, 10, 15, 0, 0, 0, time.UTC)
	sessions := newTestSessions()
	ctx, err := sessions.Load(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	sessions.Put(ctx, sessionTenantID, "bank-demo")
	sessions.Put(ctx, sessionPrincipalID, "principal-1")
	sessions.Put(ctx, sessionLegalEntityID, "BANK-NG")
	sessions.Put(ctx, sessionSessionID, "ses_test")
	sessions.Put(ctx, sessionIssuedAt, now)
	sessions.Put(ctx, sessionAssurance, "urn:example:aal2")

	resolver := &fakeAccessResolver{resolution: access.Resolution{
		TenantID: "bank-demo", PrincipalID: "principal-1", LegalEntityID: "BANK-NG", Kind: "PERSON",
		RoleCodes: []string{"BANK_READER"}, PermissionCodes: []string{identity.PermissionConfigRead},
		DepartmentGrants: []identity.DepartmentGrant{{
			Path: []string{"BANK", "OPERATIONS", "PAYMENTS"}, RoleCodes: []string{"PAYMENT_REVIEWER"}, PermissionCodes: []string{identity.PermissionConfigWrite},
		}},
	}}
	service := &Service{sessions: sessions, access: resolver, now: func() time.Time { return now }}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/context", nil).WithContext(ctx)
	actor, present, err := service.Authenticate(req)
	if err != nil {
		t.Fatal(err)
	}
	if !present || resolver.calls != 1 {
		t.Fatalf("expected one current access resolution, present=%v calls=%d", present, resolver.calls)
	}
	if !identity.HasPermission(actor, identity.PermissionConfigRead) || identity.HasPermission(actor, identity.PermissionConfigWrite) {
		t.Fatalf("unexpected global permissions: %#v", actor.PermissionCodes)
	}
	if !identity.HasDepartmentPermission(actor, []string{"BANK", "OPERATIONS", "PAYMENTS"}, identity.PermissionConfigWrite) {
		t.Fatalf("department permission missing: %#v", actor.DepartmentGrants)
	}
}

func TestAuthenticateDropsUnavailablePrincipalSession(t *testing.T) {
	sessions := newTestSessions()
	ctx, err := sessions.Load(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	sessions.Put(ctx, sessionTenantID, "bank-demo")
	sessions.Put(ctx, sessionPrincipalID, "principal-1")
	sessions.Put(ctx, sessionLegalEntityID, "BANK-NG")
	resolver := &fakeAccessResolver{err: access.ErrPrincipalUnavailable}
	service := &Service{sessions: sessions, access: resolver, now: time.Now}

	actor, present, err := service.Authenticate(httptest.NewRequest(http.MethodGet, "/api/v1/context", nil).WithContext(ctx))
	if err != nil || present || actor.PrincipalID != "" {
		t.Fatalf("unavailable principal must become an unauthenticated session: actor=%#v present=%v err=%v", actor, present, err)
	}
}

func TestBeginCreatesStateNonceAndPKCES256(t *testing.T) {
	sessions := newTestSessions()
	service := &Service{
		oauth: oauth2.Config{
			ClientID: "client", RedirectURL: "https://api.example.test/auth/oidc/callback",
			Endpoint: oauth2.Endpoint{AuthURL: "https://idp.example.test/authorize", TokenURL: "https://idp.example.test/token"},
			Scopes:   []string{"openid"},
		},
		sessions: sessions,
	}
	handler := sessions.LoadAndSave(http.HandlerFunc(service.Begin))
	req := httptest.NewRequest(http.MethodGet, "https://api.example.test/auth/oidc/login?tenant=bank-demo&legal_entity=BANK-NG&return_to=%2Ftoday", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusFound {
		t.Fatalf("expected OIDC redirect, got %d: %s", recorder.Code, recorder.Body.String())
	}
	destination, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	query := destination.Query()
	for _, key := range []string{"state", "nonce", "code_challenge"} {
		if query.Get(key) == "" {
			t.Fatalf("missing %s in authorization redirect: %s", key, destination.String())
		}
	}
	if query.Get("code_challenge_method") != "S256" {
		t.Fatalf("expected PKCE S256, got %q", query.Get("code_challenge_method"))
	}
}

func TestBeginRequiresBoundedTenantAndLegalEntity(t *testing.T) {
	sessions := newTestSessions()
	service := &Service{sessions: sessions}
	handler := sessions.LoadAndSave(http.HandlerFunc(service.Begin))

	for _, rawURL := range []string{
		"https://api.example.test/auth/oidc/login?tenant=bank-demo",
		"https://api.example.test/auth/oidc/login?tenant=bank-demo&legal_entity=" + strings.Repeat("x", maxScopeValueBytes+1),
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, rawURL, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected invalid login scope to be rejected, url=%s status=%d", rawURL, recorder.Code)
		}
	}
}

func TestSafeReturnPathRejectsExternalAndOversizedTargets(t *testing.T) {
	cases := map[string]string{
		"":                              "/",
		"/today?view=mine":              "/today?view=mine",
		"https://evil.test/steal":       "/",
		"//evil.test/steal":             "/",
		"javascript:alert(1)":           "/",
		"/" + strings.Repeat("x", 2048): "/",
	}
	for input, expected := range cases {
		if actual := safeReturnPath(input); actual != expected {
			t.Errorf("safeReturnPath(%q)=%q want %q", input, actual, expected)
		}
	}
}

func TestAssuranceFromClaimsIsBounded(t *testing.T) {
	if got := assuranceFromClaims(strings.Repeat("x", maxAssuranceBytes+1), []string{"pwd", "mfa"}); got != "OIDC:pwd,mfa" {
		t.Fatalf("expected bounded AMR fallback, got %q", got)
	}
	if got := assuranceFromClaims("urn:example:aal2", nil); got != "urn:example:aal2" {
		t.Fatalf("expected ACR to be preserved, got %q", got)
	}
}

func TestValidateAbsoluteURLRequiresHTTPSWhenSecure(t *testing.T) {
	if err := validateAbsoluteURL("http://idp.example.test", true, false); err == nil {
		t.Fatal("secure OIDC must reject a cleartext issuer")
	}
	if err := validateAbsoluteURL("https://idp.example.test/tenant", true, false); err != nil {
		t.Fatal(err)
	}
	if err := validateAbsoluteURL("/auth/oidc/callback", false, true); err == nil {
		t.Fatal("OIDC endpoints must be absolute")
	}
}

func TestAccessErrorsRemainErrors(t *testing.T) {
	sessions := newTestSessions()
	ctx, err := sessions.Load(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	sessions.Put(ctx, sessionTenantID, "bank-demo")
	sessions.Put(ctx, sessionPrincipalID, "principal-1")
	sessions.Put(ctx, sessionLegalEntityID, "BANK-NG")
	resolver := &fakeAccessResolver{err: errors.New("database unavailable")}
	service := &Service{sessions: sessions, access: resolver, now: time.Now}
	_, present, err := service.Authenticate(httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx))
	if err == nil || present {
		t.Fatalf("infrastructure access failure must not look like an anonymous session: present=%v err=%v", present, err)
	}
}
