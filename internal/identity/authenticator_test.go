package identity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSignedAuthenticator(t *testing.T) {
	secret := "0123456789abcdef0123456789abcdef"
	now := time.Date(2026, 8, 5, 18, 0, 0, 0, time.UTC)
	authenticator, err := NewSignedAuthenticator(secret, 2*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	authenticator.now = func() time.Time { return now }
	actor := Actor{TenantID: "bank-demo", PrincipalID: "person-1", LegalEntityID: "bank-ng", Kind: "PERSON", RoleCodes: []string{"cro", "Risk Owner"}, AuthenticationMethod: "SSO", AssuranceLevel: "MFA", SessionID: "session-1", IssuedAt: now, ExpiresAt: now.Add(10 * time.Minute)}
	payload, _ := json.Marshal(actor)
	envelope := base64.RawURLEncoding.EncodeToString(payload)
	timestamp := "1785952800"
	req := httptest.NewRequest("POST", "https://example.test/api/v1/matters/123/decisions", nil)
	req.Header.Set(HeaderIdentity, envelope)
	req.Header.Set(HeaderTimestamp, timestamp)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(req.Method + "\n" + req.URL.EscapedPath() + "\n" + timestamp + "\n" + envelope))
	req.Header.Set(HeaderSignature, hex.EncodeToString(mac.Sum(nil)))
	verified, present, err := authenticator.Authenticate(req)
	if err != nil || !present {
		t.Fatalf("identity not verified: present=%v err=%v", present, err)
	}
	if verified.PrincipalID != actor.PrincipalID || verified.TenantID != actor.TenantID {
		t.Fatalf("unexpected actor: %#v", verified)
	}
	if len(verified.RoleCodes) != 2 || verified.RoleCodes[0] != "CRO" || verified.RoleCodes[1] != "RISK_OWNER" {
		t.Fatalf("unexpected normalized roles: %#v", verified.RoleCodes)
	}
	req.Header.Set(HeaderSignature, "00")
	if _, _, err := authenticator.Authenticate(req); err == nil {
		t.Fatal("expected altered signature to fail")
	}
}

func TestDevelopmentAuthenticatorRequiresConfiguredIdentity(t *testing.T) {
	authenticator := NewDevelopmentAuthenticator("", "", "")
	req := httptest.NewRequest("GET", "https://example.test/health/live", nil)
	if _, present, err := authenticator.Authenticate(req); err != nil || present {
		t.Fatalf("expected anonymous development request, present=%v err=%v", present, err)
	}
}

func TestDevelopmentAuthenticatorExposesConfiguredAndOverrideRoles(t *testing.T) {
	authenticator := NewDevelopmentAuthenticator("bank", "person", "entity", "Program Owner")
	req := httptest.NewRequest("GET", "https://example.test/api/v1/context", nil)
	actor, present, err := authenticator.Authenticate(req)
	if err != nil || !present || len(actor.RoleCodes) != 1 || actor.RoleCodes[0] != "PROGRAM_OWNER" {
		t.Fatalf("unexpected configured roles: %#v present=%v err=%v", actor, present, err)
	}
	req.Header.Set("X-ClearSight-Demo-Roles", "reviewer, challenger")
	actor, present, err = authenticator.Authenticate(req)
	if err != nil || !present || len(actor.RoleCodes) != 2 || actor.RoleCodes[0] != "REVIEWER" {
		t.Fatalf("unexpected override roles: %#v present=%v err=%v", actor, present, err)
	}
}

func TestDevelopmentSystemAdminIncludesIdentityAdministrationCapabilities(t *testing.T) {
	authenticator := NewDevelopmentAuthenticator("bank", "admin", "entity", "SYSTEM_ADMIN")
	actor, present, err := authenticator.Authenticate(httptest.NewRequest("GET", "https://example.test/api/v1/context", nil))
	if err != nil || !present {
		t.Fatalf("development system admin not authenticated: present=%v err=%v", present, err)
	}
	if !HasPermission(actor, PermissionIdentityRead) || !HasPermission(actor, PermissionIdentityConfigure) {
		t.Fatalf("development system admin missing identity administration permissions: %#v", actor.PermissionCodes)
	}
}

func TestDevelopmentAuthenticatorGrantsVendorReadToOperationalRoles(t *testing.T) {
	for _, role := range []string{"BUSINESS_OWNER", "GRC_ADMIN", "THIRD_PARTY_ADMIN"} {
		t.Run(role, func(t *testing.T) {
			authenticator := NewDevelopmentAuthenticator("bank", "principal", "entity", role)
			actor, present, err := authenticator.Authenticate(httptest.NewRequest("GET", "https://example.test/api/v1/vendors", nil))
			if err != nil || !present {
				t.Fatalf("development identity missing: present=%v err=%v", present, err)
			}
			if !HasPermission(actor, PermissionVendorRead) {
				t.Fatalf("%s lacks vendor read: %#v", role, actor.PermissionCodes)
			}
		})
	}
}
