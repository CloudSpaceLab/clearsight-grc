//go:build postgres && postgresintegration

package federation

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresSessionStoreRejectsStaleCommitsAndRotation(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	store := NewPostgresSessionStore(pool)
	t.Cleanup(store.StopCleanup)
	s := scs.New()
	s.Store = store
	s.HashTokenInStore = true
	s.Lifetime = time.Hour
	s.IdleTimeout = time.Minute
	s.Cookie.Name = "clearsight_session"
	create := func() string {
		t.Helper()
		c, e := s.Load(withOriginalSessionToken(ctx, ""), "")
		if e != nil {
			t.Fatal(e)
		}
		s.Put(c, sessionPrincipalID, "principal-1")
		s.Put(c, sessionTenantID, "bank")
		s.Put(c, sessionLegalEntityID, "entity-a")
		token, _, e := s.Commit(c)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM web_sessions WHERE token=$1`, hashSessionToken(token))
		})
		return token
	}
	load := func(token string) context.Context {
		t.Helper()
		c, e := s.Load(withOriginalSessionToken(ctx, token), token)
		if e != nil {
			t.Fatal(e)
		}
		return c
	}
	assertGone := func(token string) {
		t.Helper()
		if got := s.GetString(load(token), sessionPrincipalID); got != "" {
			t.Fatalf("inactive session revived for %q", got)
		}
	}
	t.Run("logout and concurrent idle refresh", func(t *testing.T) {
		token := create()
		logout, inflight := load(token), load(token)
		if e := s.Destroy(logout); e != nil {
			t.Fatal(e)
		}
		if _, _, e := s.Commit(inflight); e != nil {
			t.Fatal(e)
		}
		assertGone(token)
	})
	t.Run("logout wins against scope rotation", func(t *testing.T) {
		token := create()
		logout, rotation := load(token), load(token)
		svc := &Service{sessions: s, access: &fakeAccessResolver{resolution: access.Resolution{TenantID: "bank", PrincipalID: "principal-1", LegalEntityID: "entity-b", Kind: "PERSON"}}, now: time.Now}
		w := httptest.NewRecorder()
		svc.Logout(w, httptest.NewRequest(http.MethodPost, "/logout", nil).WithContext(logout))
		if w.Code != http.StatusNoContent {
			t.Fatalf("logout status: %d", w.Code)
		}
		_, e := svc.SwitchScope(rotation, identity.Actor{TenantID: "bank", PrincipalID: "principal-1", LegalEntityID: "entity-a"}, "entity-b")
		if !errors.Is(e, errSessionNoLongerActive) {
			t.Fatalf("stale rotation: %v", e)
		}
		assertGone(token)
	})
	t.Run("rotation invalidates old in-flight request", func(t *testing.T) {
		token := create()
		rotation, inflight := load(token), load(token)
		svc := &Service{sessions: s, access: &fakeAccessResolver{resolution: access.Resolution{TenantID: "bank", PrincipalID: "principal-1", LegalEntityID: "entity-b", Kind: "PERSON"}}, now: time.Now}
		if _, e := svc.SwitchScope(rotation, identity.Actor{TenantID: "bank", PrincipalID: "principal-1", LegalEntityID: "entity-a"}, "entity-b"); e != nil {
			t.Fatal(e)
		}
		newToken, _, e := s.Commit(rotation)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM web_sessions WHERE token=$1`, hashSessionToken(newToken))
		})
		if newToken == token {
			t.Fatal("token did not rotate")
		}
		if _, _, e = s.Commit(inflight); e != nil {
			t.Fatal(e)
		}
		assertGone(token)
		if got := s.GetString(load(newToken), sessionPrincipalID); got != "principal-1" {
			t.Fatalf("rotated session: %q", got)
		}
		if got := s.GetTime(load(newToken), sessionIssuedAt); got.IsZero() {
			t.Fatal("rotated session lost its issue time")
		}
	})
	t.Run("idle cleanup cannot revive a session", func(t *testing.T) {
		token := create()
		inflight := load(token)
		if _, e := pool.Exec(ctx, `DELETE FROM web_sessions WHERE token=$1`, hashSessionToken(token)); e != nil {
			t.Fatal(e)
		}
		if _, _, e := s.Commit(inflight); e != nil {
			t.Fatal(e)
		}
		assertGone(token)
	})
	t.Run("idle expiry cannot be extended by an in-flight request", func(t *testing.T) {
		token := create()
		inflight := load(token)
		if _, e := pool.Exec(ctx, `UPDATE web_sessions SET expiry=clock_timestamp()-interval '1 second' WHERE token=$1`, hashSessionToken(token)); e != nil {
			t.Fatal(e)
		}
		if _, _, e := s.Commit(inflight); e != nil {
			t.Fatal(e)
		}
		assertGone(token)
	})
	t.Run("expired session cannot rotate or log out as active", func(t *testing.T) {
		token := create()
		rotation, logout := load(token), load(token)
		if _, e := pool.Exec(ctx, `UPDATE web_sessions SET expiry=clock_timestamp()-interval '1 second' WHERE token=$1`, hashSessionToken(token)); e != nil {
			t.Fatal(e)
		}
		if e := s.RenewToken(rotation); !errors.Is(e, errSessionNoLongerActive) {
			t.Fatalf("expired rotation: %v", e)
		}
		if e := s.Destroy(logout); !errors.Is(e, errSessionNoLongerActive) {
			t.Fatalf("expired logout: %v", e)
		}
		assertGone(token)
	})
	t.Run("missing origin fails closed", func(t *testing.T) {
		if e := store.CommitCtx(ctx, "token", []byte("value"), time.Now().Add(time.Hour)); !errors.Is(e, errSessionOriginUnavailable) {
			t.Fatalf("contextless commit: %v", e)
		}
		if e := store.Commit("token", []byte("value"), time.Now().Add(time.Hour)); !errors.Is(e, errSessionOriginUnavailable) {
			t.Fatalf("contextless legacy commit: %v", e)
		}
		if e := store.Delete("token"); !errors.Is(e, errSessionOriginUnavailable) {
			t.Fatalf("contextless legacy delete: %v", e)
		}
	})
	t.Run("store failures remain explicit without exposing session data", func(t *testing.T) {
		closed, e := pgxpool.New(ctx, url)
		if e != nil {
			t.Fatal(e)
		}
		closed.Close()
		bad := NewPostgresSessionStore(closed)
		defer bad.StopCleanup()
		origin := withOriginalSessionToken(ctx, "private-session-token")
		for _, call := range []func() error{func() error {
			return bad.CommitCtx(origin, hashSessionToken("private-session-token"), []byte("private-session-data"), time.Now().Add(time.Hour))
		}, func() error {
			return bad.CommitCtx(origin, hashSessionToken("new-session-token"), []byte("private-session-data"), time.Now().Add(time.Hour))
		}, func() error { return bad.DeleteCtx(origin, hashSessionToken("private-session-token")) }} {
			e := call()
			if e == nil || strings.Contains(e.Error(), "private-session") {
				t.Fatalf("store error unavailable or sensitive: %v", e)
			}
		}
	})
	t.Run("new token collision cannot overwrite", func(t *testing.T) {
		token := create()
		if e := store.CommitCtx(withOriginalSessionToken(ctx, "different"), hashSessionToken(token), []byte("invalid"), time.Now().Add(time.Hour)); !errors.Is(e, errSessionNoLongerActive) {
			t.Fatalf("token collision: %v", e)
		}
		if got := s.GetString(load(token), sessionPrincipalID); got != "principal-1" {
			t.Fatalf("collision altered session: %q", got)
		}
	})
	t.Run("duplicate cookies classify the same token SCS loads", func(t *testing.T) {
		token := create()
		svc := &Service{sessions: s}
		handler := svc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(s.GetString(r.Context(), sessionPrincipalID)))
		}))
		for _, tc := range []struct{ cookies, want string }{{"clearsight_session=" + token + "; clearsight_session=invalid", "principal-1"}, {"clearsight_session=invalid; clearsight_session=" + token, ""}, {"clearsight_session=bad value; clearsight_session=" + token, ""}, {"clearsight_session=bad\\value; clearsight_session=" + token, "principal-1"}} {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Cookie", tc.cookies)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != http.StatusOK || w.Body.String() != tc.want {
				t.Fatalf("duplicate cookie: status=%d body=%q", w.Code, w.Body.String())
			}
		}
	})
	t.Run("new session is committed through middleware", func(t *testing.T) {
		svc := &Service{sessions: s}
		handler := svc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/login" {
				s.Put(r.Context(), sessionPrincipalID, "principal-1")
			}
			_, _ = w.Write([]byte(s.GetString(r.Context(), sessionPrincipalID)))
		}))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/login", nil))
		cookies := w.Result().Cookies()
		if w.Code != http.StatusOK || len(cookies) != 1 || cookies[0].Value == "" {
			t.Fatalf("login response: %d", w.Code)
		}
		token := cookies[0].Value
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM web_sessions WHERE token=$1`, hashSessionToken(token))
		})
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.AddCookie(cookies[0])
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusOK || w.Body.String() != "principal-1" {
			t.Fatalf("new session read: %d %q", w.Code, w.Body.String())
		}
	})
}
