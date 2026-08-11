package identity

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const demoSessionCookie = "clearsight_demo_session"

var ErrInvalidDemoCredentials = errors.New("invalid demo credentials")

type DemoAccount struct {
	Label       string   `json:"label"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	PrincipalID string   `json:"-"`
	RoleCodes   []string `json:"role_codes"`
}

type DemoSessionAuthenticator interface {
	Authenticator
	Accounts() []DemoAccount
	Login(http.ResponseWriter, string, string) (DemoAccount, error)
	Logout(http.ResponseWriter)
}

type DemoAuthenticator struct {
	tenantID      string
	legalEntityID string
	accounts      []DemoAccount
	byUsername    map[string]DemoAccount
	key           []byte
	ttl           time.Duration
	now           func() time.Time
	headerAuth    *DevelopmentAuthenticator
}

func NewDemoAuthenticator(tenantID, defaultPrincipalID, legalEntityID string) (*DemoAuthenticator, error) {
	tenantID = strings.TrimSpace(tenantID)
	defaultPrincipalID = strings.TrimSpace(defaultPrincipalID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	if tenantID == "" || defaultPrincipalID == "" || legalEntityID == "" {
		return nil, fmt.Errorf("demo tenant, principal and legal entity are required")
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate demo session key: %w", err)
	}
	accounts := []DemoAccount{
		{Label: "Chief Risk Officer", Username: "cro@demo.clearsight.local", Password: "demo", PrincipalID: defaultPrincipalID, RoleCodes: []string{"CRO", "EXECUTIVE"}},
		{Label: "Chief Compliance Officer", Username: "cco@demo.clearsight.local", Password: "demo", PrincipalID: "role-cco", RoleCodes: []string{"CCO", "EXECUTIVE"}},
		{Label: "Chief Information Security Officer", Username: "ciso@demo.clearsight.local", Password: "demo", PrincipalID: "role-ciso", RoleCodes: []string{"CISO", "EXECUTIVE"}},
		{Label: "GRC Administrator", Username: "grc-admin@demo.clearsight.local", Password: "demo", PrincipalID: "role-grc-admin", RoleCodes: []string{"GRC_ADMIN"}},
		{Label: "Internal Auditor", Username: "auditor@demo.clearsight.local", Password: "demo", PrincipalID: "role-auditor", RoleCodes: []string{"AUDITOR", "REVIEWER"}},
		{Label: "Program Owner", Username: "owner@demo.clearsight.local", Password: "demo", PrincipalID: "role-program-owner", RoleCodes: []string{"PROGRAM_OWNER"}},
		{Label: "Evidence Respondent", Username: "evidence@demo.clearsight.local", Password: "demo", PrincipalID: "role-evidence-respondent", RoleCodes: []string{"EVIDENCE_RESPONDENT"}},
	}
	byUsername := make(map[string]DemoAccount, len(accounts))
	for i := range accounts {
		accounts[i].RoleCodes = NormalizeRoleCodes(accounts[i].RoleCodes)
		byUsername[strings.ToLower(accounts[i].Username)] = accounts[i]
	}
	return &DemoAuthenticator{
		tenantID: tenantID, legalEntityID: legalEntityID, accounts: accounts, byUsername: byUsername,
		key: key, ttl: 8 * time.Hour, now: time.Now,
		headerAuth: NewDevelopmentAuthenticator(tenantID, defaultPrincipalID, legalEntityID),
	}, nil
}

func (a *DemoAuthenticator) Accounts() []DemoAccount {
	out := make([]DemoAccount, len(a.accounts))
	for i, account := range a.accounts {
		out[i] = account
		out[i].RoleCodes = append([]string(nil), account.RoleCodes...)
	}
	return out
}

func (a *DemoAuthenticator) Authenticate(r *http.Request) (Actor, bool, error) {
	if hasDemoIdentityHeaders(r) {
		return a.headerAuth.Authenticate(r)
	}
	if authorization := strings.TrimSpace(r.Header.Get("Authorization")); len(authorization) >= 7 && strings.EqualFold(authorization[:7], "Bearer ") {
		return Actor{}, false, nil
	}
	cookie, err := r.Cookie(demoSessionCookie)
	if errors.Is(err, http.ErrNoCookie) {
		return Actor{}, false, nil
	}
	if err != nil {
		return Actor{}, false, ErrInvalidIdentity
	}
	username, expiry, ok := a.verifySession(cookie.Value)
	if !ok || !a.now().UTC().Before(expiry) {
		return Actor{}, false, nil
	}
	account, ok := a.byUsername[username]
	if !ok {
		return Actor{}, false, nil
	}
	now := a.now().UTC()
	return Actor{
		TenantID: a.tenantID, PrincipalID: account.PrincipalID, LegalEntityID: a.legalEntityID, Kind: "PERSON",
		RoleCodes: account.RoleCodes, PermissionCodes: developmentPermissions(account.RoleCodes),
		AuthenticationMethod: "DEMO", AssuranceLevel: "DEMO", SessionID: "demo:" + account.Username,
		IssuedAt: now, ExpiresAt: expiry,
	}, true, nil
}

func (a *DemoAuthenticator) Login(w http.ResponseWriter, username, password string) (DemoAccount, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	account, ok := a.byUsername[username]
	if !ok || !constantTimeCredentialEqual(password, account.Password) {
		return DemoAccount{}, ErrInvalidDemoCredentials
	}
	expires := a.now().UTC().Add(a.ttl)
	http.SetCookie(w, &http.Cookie{
		Name: demoSessionCookie, Value: a.signSession(username, expires), Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: false, MaxAge: int(a.ttl.Seconds()), Expires: expires,
	})
	return account, nil
}

func (a *DemoAuthenticator) Logout(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: demoSessionCookie, Value: "", Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: false, MaxAge: -1, Expires: time.Unix(1, 0).UTC(),
	})
}

func (a *DemoAuthenticator) signSession(username string, expires time.Time) string {
	payload, _ := json.Marshal(struct {
		Username string `json:"u"`
		Expires  int64  `json:"e"`
	}{Username: username, Expires: expires.Unix()})
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, a.key)
	_, _ = mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *DemoAuthenticator) verifySession(value string) (string, time.Time, bool) {
	encoded, signature, found := strings.Cut(strings.TrimSpace(value), ".")
	if !found || encoded == "" || signature == "" {
		return "", time.Time{}, false
	}
	provided, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return "", time.Time{}, false
	}
	mac := hmac.New(sha256.New, a.key)
	_, _ = mac.Write([]byte(encoded))
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return "", time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", time.Time{}, false
	}
	var session struct {
		Username string `json:"u"`
		Expires  int64  `json:"e"`
	}
	if err := json.Unmarshal(payload, &session); err != nil || session.Expires <= 0 || session.Username == "" {
		return "", time.Time{}, false
	}
	return strings.ToLower(session.Username), time.Unix(session.Expires, 0).UTC(), true
}

func hasDemoIdentityHeaders(r *http.Request) bool {
	return strings.TrimSpace(r.Header.Get("X-ClearSight-Demo-Tenant")) != "" ||
		strings.TrimSpace(r.Header.Get("X-ClearSight-Demo-Principal")) != "" ||
		strings.TrimSpace(r.Header.Get("X-ClearSight-Demo-Legal-Entity")) != "" ||
		strings.TrimSpace(r.Header.Get("X-ClearSight-Demo-Roles")) != ""
}

func constantTimeCredentialEqual(provided, expected string) bool {
	providedHash := sha256.Sum256([]byte(provided))
	expectedHash := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(providedHash[:], expectedHash[:]) == 1
}
