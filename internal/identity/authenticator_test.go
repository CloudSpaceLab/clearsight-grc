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

func signedRequest(t *testing.T, secret string, actor Actor, now time.Time) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(actor)
	if err != nil {
		t.Fatal(err)
	}
	envelope := base64.RawURLEncoding.EncodeToString(payload)
	timestamp := now.Format("1136239445")
	_ = timestamp
	return httptest.NewRecorder()
}

func TestSignedAuthenticator(t *testing.T) {
	secret := "0123456789abcdef0123456789abcdef"
	now := time.Date(2026, 8, 5, 18, 0, 0, 0, time.UTC)
	authenticator, err := NewSignedAuthenticator(secret, 2*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	authenticator.now = func() time.Time { return now }
	actor := Actor{TenantID: "bank-demo", PrincipalID: "person-1", LegalEntityID: "bank-ng", Kind: "PERSON", AuthenticationMethod: "SSO", AssuranceLevel: "MFA", SessionID: "session-1", IssuedAt: now, ExpiresAt: now.Add(10 * time.Minute)}
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
