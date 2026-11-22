//go:build postgres && postgresintegration

package federation

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
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
	store := NewPostgresSessionStore(pool, nil)
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
		rotation, inflight, oldLogout := load(token), load(token), load(token)
		svc := &Service{sessions: s, access: &fakeAccessResolver{resolution: access.Resolution{TenantID: "bank", PrincipalID: "principal-1", LegalEntityID: "entity-b", Kind: "PERSON"}}, now: time.Now}
		if _, e := svc.SwitchScope(rotation, identity.Actor{TenantID: "bank", PrincipalID: "principal-1", LegalEntityID: "entity-a"}, "entity-b"); e != nil {
			t.Fatal(e)
		}
		w := httptest.NewRecorder()
		svc.Logout(w, httptest.NewRequest(http.MethodPost, "/logout", nil).WithContext(oldLogout))
		if w.Code == http.StatusNoContent {
			t.Fatal("stale logout reported that the rotated session was signed out")
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
		bad := NewPostgresSessionStore(closed, nil)
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
	t.Run("later lookup cannot replace the loaded baseline", func(t *testing.T) {
		token := create()
		origin := withOriginalSessionToken(ctx, token)
		first, found, e := store.FindCtx(origin, hashSessionToken(token))
		if e != nil || !found {
			t.Fatalf("first lookup: %v %v", found, e)
		}
		changed, e := scs.GobCodec{}.Encode(time.Now().Add(time.Hour), map[string]interface{}{sessionPrincipalID: "different"})
		if e != nil {
			t.Fatal(e)
		}
		if _, e = pool.Exec(ctx, `UPDATE web_sessions SET data=$2 WHERE token=$1`, hashSessionToken(token), changed); e != nil {
			t.Fatal(e)
		}
		if _, found, e = store.FindCtx(origin, hashSessionToken(token)); e != nil || !found {
			t.Fatalf("second lookup: %v %v", found, e)
		}
		state := origin.Value(originalSessionTokenKey{}).(*sessionRequestState)
		state.mu.Lock()
		preserved := bytes.Equal(state.loaded, first)
		state.mu.Unlock()
		if !preserved {
			t.Fatal("loaded baseline was replaced")
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
	t.Run("unchanged idle refresh failure preserves committed response", func(t *testing.T) {
		token := create()
		var logs bytes.Buffer
		storePool, e := pgxpool.New(ctx, url)
		if e != nil {
			t.Fatal(e)
		}
		failing := NewPostgresSessionStore(storePool, slog.New(slog.NewTextHandler(&logs, nil)))
		defer failing.StopCleanup()
		old := s.Store
		s.Store = failing
		defer func() { s.Store = old }()
		domainConn, e := pool.Acquire(ctx)
		if e != nil {
			t.Fatal(e)
		}
		defer domainConn.Release()
		if _, e = domainConn.Exec(ctx, `CREATE TEMP TABLE assessment_commit_probe(id integer)`); e != nil {
			t.Fatal(e)
		}
		svc := &Service{sessions: s}
		handler := svc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, e := domainConn.Exec(ctx, `INSERT INTO assessment_commit_probe(id) VALUES(1)`); e != nil {
				t.Error(e)
				return
			}
			storePool.Close()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("committed"))
		}))
		r := httptest.NewRequest(http.MethodPost, "/assessment", nil)
		r.AddCookie(&http.Cookie{Name: s.Cookie.Name, Value: token})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		var count int
		if e = domainConn.QueryRow(ctx, `SELECT count(*) FROM assessment_commit_probe`).Scan(&count); e != nil || count != 1 {
			t.Fatalf("material commit: %d %v", count, e)
		}
		if w.Code != http.StatusOK || w.Body.String() != "committed" {
			t.Fatalf("committed response: %d %q", w.Code, w.Body.String())
		}
		if !strings.Contains(logs.String(), "session idle refresh failed") || strings.Contains(logs.String(), token) || strings.Contains(logs.String(), "principal-1") {
			t.Fatalf("unsafe or missing warning: %q", logs.String())
		}
	})
	for _, tc := range []struct {
		name                             string
		mutate, deadline, rotate, logout bool
	}{
		{"changed session", true, false, false, false}, {"changed deadline", false, true, false, false}, {"rotation", false, false, true, false}, {"logout", false, false, false, true},
	} {
		t.Run(tc.name+" failure stays explicit", func(t *testing.T) {
			token := create()
			storePool, e := pgxpool.New(ctx, url)
			if e != nil {
				t.Fatal(e)
			}
			var logs bytes.Buffer
			failing := NewPostgresSessionStore(storePool, slog.New(slog.NewTextHandler(&logs, nil)))
			defer failing.StopCleanup()
			old := s.Store
			s.Store = failing
			defer func() { s.Store = old }()
			svc := &Service{sessions: s}
			handler := svc.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.mutate {
					s.Put(r.Context(), sessionPrincipalID, "different-principal")
				}
				if tc.deadline {
					s.SetDeadline(r.Context(), time.Now().Add(30*time.Minute))
				}
				if tc.rotate {
					if e := s.RenewToken(r.Context()); e != nil {
						t.Error(e)
						return
					}
				}
				storePool.Close()
				if tc.logout {
					svc.Logout(w, r)
					return
				}
				_, _ = w.Write([]byte("attempted"))
			}))
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			r.AddCookie(&http.Cookie{Name: s.Cookie.Name, Value: token})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), token) || strings.Contains(w.Body.String(), "principal-1") || strings.Contains(logs.String(), "session idle refresh failed") {
				t.Fatalf("security-critical failure: %d %q logs=%q", w.Code, w.Body.String(), logs.String())
			}
		})
	}
	t.Run("missing baseline and logger fail closed", func(t *testing.T) {
		closed, e := pgxpool.New(ctx, url)
		if e != nil {
			t.Fatal(e)
		}
		closed.Close()
		var logs bytes.Buffer
		withLogger := NewPostgresSessionStore(closed, slog.New(slog.NewTextHandler(&logs, nil)))
		defer withLogger.StopCleanup()
		withoutLogger := NewPostgresSessionStore(closed, nil)
		defer withoutLogger.StopCleanup()
		origin := withOriginalSessionToken(ctx, "private-token")
		encoded, e := scs.GobCodec{}.Encode(time.Now().Add(time.Hour), map[string]interface{}{"key": "value"})
		if e != nil {
			t.Fatal(e)
		}
		for _, store := range []*PostgresSessionStore{withLogger, withoutLogger} {
			if e := store.CommitCtx(origin, hashSessionToken("private-token"), encoded, time.Now().Add(time.Hour)); e == nil {
				t.Fatal("unproven refresh error was suppressed")
			}
		}
	})
	t.Run("loaded refresh without logger stays explicit", func(t *testing.T) {
		token := create()
		storePool, e := pgxpool.New(ctx, url)
		if e != nil {
			t.Fatal(e)
		}
		failing := NewPostgresSessionStore(storePool, nil)
		defer failing.StopCleanup()
		old := s.Store
		s.Store = failing
		defer func() { s.Store = old }()
		loaded := load(token)
		storePool.Close()
		if _, _, e = s.Commit(loaded); e == nil {
			t.Fatal("unlogged refresh error was suppressed")
		}
	})
	t.Run("expiry beyond the absolute deadline stays explicit", func(t *testing.T) {
		token := create()
		storePool, e := pgxpool.New(ctx, url)
		if e != nil {
			t.Fatal(e)
		}
		var logs bytes.Buffer
		failing := NewPostgresSessionStore(storePool, slog.New(slog.NewTextHandler(&logs, nil)))
		defer failing.StopCleanup()
		old := s.Store
		s.Store = failing
		defer func() { s.Store = old }()
		loaded := load(token)
		state := loaded.Value(originalSessionTokenKey{}).(*sessionRequestState)
		state.mu.Lock()
		data := append([]byte(nil), state.loaded...)
		state.mu.Unlock()
		expiry := s.Deadline(loaded).Add(time.Hour)
		storePool.Close()
		if e = failing.CommitCtx(loaded, hashSessionToken(token), data, expiry); e == nil || strings.Contains(logs.String(), "session idle refresh failed") {
			t.Fatalf("invalid expiry was suppressed: %v %q", e, logs.String())
		}
	})
}
