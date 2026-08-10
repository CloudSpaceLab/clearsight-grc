package federation

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/alexedwards/scs/v2"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	sessionTenantID      = "auth.tenant_id"
	sessionPrincipalID   = "auth.principal_id"
	sessionLegalEntityID = "auth.legal_entity_id"
	sessionSessionID     = "auth.session_id"
	sessionIssuedAt      = "auth.issued_at"
	sessionAssurance     = "auth.assurance"
	transactionState     = "oidc.state"
	transactionNonce     = "oidc.nonce"
	transactionVerifier  = "oidc.verifier"
	transactionTenant    = "oidc.tenant"
	transactionReturnTo  = "oidc.return_to"
)

type Config struct {
	Issuer          string
	ClientID        string
	ClientSecret    string
	RedirectURL     string
	SessionLifetime time.Duration
	IdleTimeout     time.Duration
	SecureCookies   bool
}

type Service struct {
	issuer   string
	oauth    oauth2.Config
	verifier *oidc.IDTokenVerifier
	sessions *scs.SessionManager
	access   access.Resolver
	now      func() time.Time
}

func New(ctx context.Context, cfg Config, store scs.Store, resolver access.Resolver) (*Service, error) {
	cfg.Issuer = strings.TrimSpace(cfg.Issuer)
	cfg.ClientID = strings.TrimSpace(cfg.ClientID)
	cfg.ClientSecret = strings.TrimSpace(cfg.ClientSecret)
	cfg.RedirectURL = strings.TrimSpace(cfg.RedirectURL)
	if cfg.Issuer == "" || cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RedirectURL == "" || store == nil || resolver == nil {
		return nil, fmt.Errorf("OIDC issuer, client, redirect, session store and access resolver are required")
	}
	if err := validateAbsoluteURL(cfg.Issuer, cfg.SecureCookies, false); err != nil {
		return nil, fmt.Errorf("invalid OIDC issuer: %w", err)
	}
	if err := validateAbsoluteURL(cfg.RedirectURL, cfg.SecureCookies, true); err != nil {
		return nil, fmt.Errorf("invalid OIDC redirect URL: %w", err)
	}
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}
	if cfg.SessionLifetime <= 0 {
		cfg.SessionLifetime = 8 * time.Hour
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = 30 * time.Minute
	}
	sessions := scs.New()
	sessions.Store = store
	sessions.Lifetime = cfg.SessionLifetime
	sessions.IdleTimeout = cfg.IdleTimeout
	sessions.HashTokenInStore = true
	sessions.Cookie.Name = "clearsight_session"
	sessions.Cookie.HttpOnly = true
	sessions.Cookie.SameSite = http.SameSiteLaxMode
	sessions.Cookie.Secure = cfg.SecureCookies
	sessions.Cookie.Persist = false
	sessions.Cookie.Path = "/"

	return &Service{
		issuer: cfg.Issuer,
		oauth: oauth2.Config{
			ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret, RedirectURL: cfg.RedirectURL,
			Endpoint: provider.Endpoint(), Scopes: []string{oidc.ScopeOpenID},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		sessions: sessions,
		access: resolver,
		now: time.Now,
	}, nil
}

func (s *Service) Middleware(next http.Handler) http.Handler {
	return s.sessions.LoadAndSave(next)
}

func (s *Service) Authenticate(r *http.Request) (identity.Actor, bool, error) {
	ctx := r.Context()
	principalID := s.sessions.GetString(ctx, sessionPrincipalID)
	if principalID == "" {
		return identity.Actor{}, false, nil
	}
	tenantID := s.sessions.GetString(ctx, sessionTenantID)
	legalEntityID := s.sessions.GetString(ctx, sessionLegalEntityID)
	resolved, err := s.access.ResolvePrincipal(ctx, tenantID, principalID, legalEntityID)
	if err != nil {
		_ = s.sessions.Destroy(ctx)
		if errors.Is(err, access.ErrPrincipalUnavailable) {
			return identity.Actor{}, false, identity.ErrInvalidIdentity
		}
		return identity.Actor{}, false, fmt.Errorf("refresh session access: %w", err)
	}
	actor := identity.Actor{
		TenantID: resolved.TenantID, PrincipalID: resolved.PrincipalID, LegalEntityID: resolved.LegalEntityID,
		Kind: resolved.Kind, RoleCodes: resolved.RoleCodes, PermissionCodes: resolved.PermissionCodes,
		DepartmentGrants: resolved.DepartmentGrants,
		AuthenticationMethod: "OIDC", AssuranceLevel: s.sessions.GetString(ctx, sessionAssurance),
		SessionID: s.sessions.GetString(ctx, sessionSessionID), IssuedAt: s.sessions.GetTime(ctx, sessionIssuedAt),
		ExpiresAt: s.sessions.Deadline(ctx),
	}
	if err := actor.Valid(s.now().UTC()); err != nil {
		_ = s.sessions.Destroy(ctx)
		return identity.Actor{}, false, err
	}
	return actor, true, nil
}

func (s *Service) Begin(w http.ResponseWriter, r *http.Request) {
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}
	state, err := randomToken(32)
	if err != nil {
		http.Error(w, "sign-in could not be started", http.StatusInternalServerError)
		return
	}
	nonce, err := randomToken(32)
	if err != nil {
		http.Error(w, "sign-in could not be started", http.StatusInternalServerError)
		return
	}
	verifier := oauth2.GenerateVerifier()
	if err := s.sessions.Destroy(r.Context()); err != nil {
		http.Error(w, "sign-in could not be started", http.StatusInternalServerError)
		return
	}
	s.sessions.Put(r.Context(), transactionState, state)
	s.sessions.Put(r.Context(), transactionNonce, nonce)
	s.sessions.Put(r.Context(), transactionVerifier, verifier)
	s.sessions.Put(r.Context(), transactionTenant, tenantID)
	s.sessions.Put(r.Context(), transactionReturnTo, safeReturnPath(r.URL.Query().Get("return_to")))

	destination := s.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier))
	http.Redirect(w, r, destination, http.StatusFound)
}

func (s *Service) Callback(w http.ResponseWriter, r *http.Request) {
	if providerError := strings.TrimSpace(r.URL.Query().Get("error")); providerError != "" {
		_ = s.sessions.Destroy(r.Context())
		http.Error(w, "sign-in was not completed", http.StatusUnauthorized)
		return
	}
	providedState := strings.TrimSpace(r.URL.Query().Get("state"))
	expectedState := s.sessions.GetString(r.Context(), transactionState)
	if providedState == "" || expectedState == "" || !constantTimeEqual(providedState, expectedState) {
		_ = s.sessions.Destroy(r.Context())
		http.Error(w, "sign-in transaction could not be verified", http.StatusUnauthorized)
		return
	}

	_ = s.sessions.PopString(r.Context(), transactionState)
	nonce := s.sessions.PopString(r.Context(), transactionNonce)
	verifier := s.sessions.PopString(r.Context(), transactionVerifier)
	tenantID := s.sessions.PopString(r.Context(), transactionTenant)
	returnTo := safeReturnPath(s.sessions.PopString(r.Context(), transactionReturnTo))
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if nonce == "" || verifier == "" || tenantID == "" || code == "" {
		_ = s.sessions.Destroy(r.Context())
		http.Error(w, "sign-in transaction is incomplete", http.StatusUnauthorized)
		return
	}

	token, err := s.oauth.Exchange(r.Context(), code, oauth2.VerifierOption(verifier))
	if err != nil {
		_ = s.sessions.Destroy(r.Context())
		http.Error(w, "sign-in code could not be exchanged", http.StatusUnauthorized)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || strings.TrimSpace(rawIDToken) == "" {
		_ = s.sessions.Destroy(r.Context())
		http.Error(w, "identity token was not returned", http.StatusUnauthorized)
		return
	}
	idToken, err := s.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		_ = s.sessions.Destroy(r.Context())
		http.Error(w, "identity token could not be verified", http.StatusUnauthorized)
		return
	}
	var claims struct {
		Nonce string   `json:"nonce"`
		ACR   string   `json:"acr"`
		AMR   []string `json:"amr"`
	}
	if err := idToken.Claims(&claims); err != nil || !constantTimeEqual(claims.Nonce, nonce) {
		_ = s.sessions.Destroy(r.Context())
		http.Error(w, "identity token transaction could not be verified", http.StatusUnauthorized)
		return
	}
	resolved, err := s.access.ResolveOIDC(r.Context(), tenantID, s.issuer, idToken.Subject)
	if err != nil {
		_ = s.sessions.Destroy(r.Context())
		if errors.Is(err, access.ErrIdentityNotProvisioned) {
			http.Error(w, "this identity is not provisioned for ClearSight", http.StatusForbidden)
			return
		}
		http.Error(w, "identity access could not be resolved", http.StatusServiceUnavailable)
		return
	}
	if err := s.sessions.RenewToken(r.Context()); err != nil {
		http.Error(w, "sign-in session could not be secured", http.StatusInternalServerError)
		return
	}
	if err := s.sessions.Clear(r.Context()); err != nil {
		_ = s.sessions.Destroy(r.Context())
		http.Error(w, "sign-in session could not be initialized", http.StatusInternalServerError)
		return
	}
	sessionID, err := id.New("ses", 16)
	if err != nil {
		_ = s.sessions.Destroy(r.Context())
		http.Error(w, "sign-in session could not be created", http.StatusInternalServerError)
		return
	}
	now := s.now().UTC()
	assurance := strings.TrimSpace(claims.ACR)
	if assurance == "" && len(claims.AMR) > 0 {
		assurance = "OIDC:" + strings.Join(claims.AMR, ",")
	}
	if assurance == "" {
		assurance = "OIDC"
	}
	s.sessions.Put(r.Context(), sessionTenantID, resolved.TenantID)
	s.sessions.Put(r.Context(), sessionPrincipalID, resolved.PrincipalID)
	s.sessions.Put(r.Context(), sessionLegalEntityID, resolved.LegalEntityID)
	s.sessions.Put(r.Context(), sessionSessionID, sessionID)
	s.sessions.Put(r.Context(), sessionIssuedAt, now)
	s.sessions.Put(r.Context(), sessionAssurance, assurance)
	http.Redirect(w, r, returnTo, http.StatusSeeOther)
}

func (s *Service) Logout(w http.ResponseWriter, r *http.Request) {
	if err := s.sessions.Destroy(r.Context()); err != nil {
		http.Error(w, "sign-out could not be completed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func constantTimeEqual(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func safeReturnPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "/"
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") {
		return "/"
	}
	if parsed.Path == "" {
		return "/"
	}
	return parsed.RequestURI()
}

func validateAbsoluteURL(value string, requireHTTPS, allowQuery bool) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("must be an absolute URL without user info or fragment")
	}
	if requireHTTPS && !strings.EqualFold(parsed.Scheme, "https") {
		return fmt.Errorf("must use https")
	}
	if !strings.EqualFold(parsed.Scheme, "https") && !strings.EqualFold(parsed.Scheme, "http") {
		return fmt.Errorf("must use http or https")
	}
	if !allowQuery && parsed.RawQuery != "" {
		return fmt.Errorf("must not contain a query string")
	}
	return nil
}
