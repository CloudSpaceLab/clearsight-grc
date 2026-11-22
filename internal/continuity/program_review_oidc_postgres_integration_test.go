//go:build postgres && postgresintegration

package continuity

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/federation"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type checkpointOIDCAccess struct{ tenant, entity, principal string }

func (a checkpointOIDCAccess) ResolveOIDC(context.Context, string, string, string, string) (access.Resolution, error) {
	return a.resolution(), nil
}
func (a checkpointOIDCAccess) ResolvePrincipal(context.Context, string, string, string) (access.Resolution, error) {
	return a.resolution(), nil
}
func (a checkpointOIDCAccess) resolution() access.Resolution {
	return access.Resolution{TenantID: a.tenant, LegalEntityID: a.entity, PrincipalID: a.principal, Kind: "PERSON"}
}

// checkpointOIDCContext loads a real encoded session through the production
// federation and identity middleware. The OIDC discovery and access resolver
// are isolated fixtures; no direct session proof is fabricated.
type checkpointOIDCFixture struct {
	ctx     context.Context
	service *federation.Service
}

func checkpointOIDCContext(t *testing.T, pool *pgxpool.Pool, tenant, entity, principal string, expires time.Time) context.Context {
	return newCheckpointOIDCFixture(t, pool, tenant, entity, principal, expires).ctx
}
func newCheckpointOIDCFixture(t *testing.T, pool *pgxpool.Pool, tenant, entity, principal string, expires time.Time) checkpointOIDCFixture {
	t.Helper()
	var issuer string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/keys"})
	}))
	issuer = provider.URL
	t.Cleanup(provider.Close)
	store := federation.NewPostgresSessionStore(pool, nil)
	t.Cleanup(store.StopCleanup)
	svc, err := federation.New(context.Background(), federation.Config{Issuer: issuer, ClientID: "fixture", ClientSecret: "fixture", RedirectURL: issuer + "/callback", ApplicationURL: issuer, SessionLifetime: time.Hour, IdleTimeout: 30 * time.Minute}, store, checkpointOIDCAccess{tenant: tenant, entity: entity, principal: principal})
	if err != nil {
		t.Fatal(err)
	}
	token, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	hashed := sha256.Sum256([]byte(token))
	hash := base64.RawURLEncoding.EncodeToString(hashed[:])
	issued := time.Now().UTC().Add(-time.Minute)
	data, err := (scs.GobCodec{}).Encode(expires, map[string]interface{}{"auth.tenant_id": tenant, "auth.principal_id": principal, "auth.legal_entity_id": entity, "auth.session_id": "session-" + token, "auth.issued_at": issued, "auth.assurance": ""})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(context.Background(), `INSERT INTO web_sessions(token,data,expiry) VALUES($1,$2,$3)`, hash, data, expires); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM web_sessions WHERE token=$1`, hash) })
	var captured context.Context
	handler := svc.Middleware(identity.Middleware(svc, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Context()
		w.WriteHeader(http.StatusNoContent)
	})))
	req := httptest.NewRequest(http.MethodGet, "/fixture", nil)
	req.AddCookie(&http.Cookie{Name: "clearsight_session", Value: token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusNoContent || captured == nil {
		t.Fatalf("OIDC fixture authentication: %d", response.Code)
	}
	if _, ok := identity.FromContext(captured); !ok {
		t.Fatal("OIDC fixture has no verified actor")
	}
	return checkpointOIDCFixture{ctx: captured, service: svc}
}
