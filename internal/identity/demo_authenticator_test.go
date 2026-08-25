package identity

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDemoAuthenticatorRequiresLoginAndKeepsCapabilityBearerSeparate(t *testing.T) {
	authenticator, err := NewDemoAuthenticator("bank-demo", "role-cro", "bank-ng")
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC)
	authenticator.now = func() time.Time { return base }

	request := httptest.NewRequest(http.MethodGet, "/api/v1/context", nil)
	if actor, present, err := authenticator.Authenticate(request); err != nil || present || actor.PrincipalID != "" {
		t.Fatalf("demo request without login unexpectedly authenticated: actor=%#v present=%v err=%v", actor, present, err)
	}

	request.Header.Set("Authorization", "Bearer bounded-capture-token")
	if _, present, err := authenticator.Authenticate(request); err != nil || present {
		t.Fatalf("capture bearer must not be interpreted as staff demo identity: present=%v err=%v", present, err)
	}
}

func TestDemoAuthenticatorRoleCatalogueAndSignedSession(t *testing.T) {
	authenticator, err := NewDemoAuthenticator("bank-demo", "role-cro", "bank-ng")
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC)
	current := base
	authenticator.now = func() time.Time { return current }

	accounts := authenticator.Accounts()
	if len(accounts) < 8 {
		t.Fatalf("expected role-rich demo catalogue, got %d accounts", len(accounts))
	}
	var systemAdmin, cco DemoAccount
	for _, account := range accounts {
		switch account.Username {
		case "system-admin@demo.clearsight.local":
			systemAdmin = account
		case "cco@demo.clearsight.local":
			cco = account
		}
		if account.Password != "demo" {
			t.Fatalf("demo credentials should be explicit and consistent for %s", account.Username)
		}
	}
	if systemAdmin.Username == "" || cco.Username == "" {
		t.Fatalf("required demo roles missing: system=%#v cco=%#v", systemAdmin, cco)
	}
	if !containsRole(cco.RoleCodes, "COMPLIANCE_OFFICER") {
		t.Fatalf("CCO demo identity must exercise compliance-officer escalation guards: %#v", cco.RoleCodes)
	}

	if _, err := authenticator.Login(httptest.NewRecorder(), systemAdmin.Username, "wrong"); !errors.Is(err, ErrInvalidDemoCredentials) {
		t.Fatalf("expected invalid demo credentials, got %v", err)
	}

	response := httptest.NewRecorder()
	if _, err := authenticator.Login(response, systemAdmin.Username, systemAdmin.Password); err != nil {
		t.Fatal(err)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != demoSessionCookie || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected demo session cookie: %#v", cookies)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/context", nil)
	request.AddCookie(cookies[0])
	actor, present, err := authenticator.Authenticate(request)
	if err != nil || !present {
		t.Fatalf("valid demo session did not authenticate: present=%v err=%v", present, err)
	}
	if actor.PrincipalID != systemAdmin.PrincipalID || !HasPermission(actor, PermissionIdentityConfigure) || actor.AuthenticationMethod != "DEMO" {
		t.Fatalf("unexpected demo actor: %#v", actor)
	}

	tampered := *cookies[0]
	tampered.Value += "x"
	request = httptest.NewRequest(http.MethodGet, "/api/v1/context", nil)
	request.AddCookie(&tampered)
	if _, present, err := authenticator.Authenticate(request); err != nil || present {
		t.Fatalf("tampered demo session should fail closed: present=%v err=%v", present, err)
	}

	current = base.Add(9 * time.Hour)
	request = httptest.NewRequest(http.MethodGet, "/api/v1/context", nil)
	request.AddCookie(cookies[0])
	if _, present, err := authenticator.Authenticate(request); err != nil || present {
		t.Fatalf("expired demo session should fail closed: present=%v err=%v", present, err)
	}
}

func TestDemoAuthenticatorUsesDurablePrincipalIDsForPostgresDemo(t *testing.T) {
	authenticator, err := NewDemoAuthenticator(DurableDemoTenantID, DurableDemoPrincipalCRO, DurableDemoLegalEntityID)
	if err != nil {
		t.Fatalf("new durable demo authenticator: %v", err)
	}
	want := map[string]string{
		"cro@demo.clearsight.local":          DurableDemoPrincipalCRO,
		"cco@demo.clearsight.local":          DurableDemoPrincipalCCO,
		"ciso@demo.clearsight.local":         DurableDemoPrincipalCISO,
		"grc-admin@demo.clearsight.local":    DurableDemoPrincipalGRCAdmin,
		"system-admin@demo.clearsight.local": DurableDemoPrincipalSystemAdmin,
		"auditor@demo.clearsight.local":      DurableDemoPrincipalAuditor,
		"owner@demo.clearsight.local":        DurableDemoPrincipalProgramOwner,
		"evidence@demo.clearsight.local":     DurableDemoPrincipalEvidenceRespondent,
	}
	for _, account := range authenticator.Accounts() {
		if account.PrincipalID != want[account.Username] {
			t.Fatalf("principal for %s = %q, want %q", account.Username, account.PrincipalID, want[account.Username])
		}
	}
}

func containsRole(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
