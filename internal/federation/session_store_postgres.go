//go:build postgres

package federation

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errSessionOriginUnavailable = errors.New("session origin unavailable")
var errSessionNoLongerActive = errors.New("session is no longer active")

// PostgresSessionStore keeps the existing SCS store and web_sessions table.
// A request may refresh only the existing token it loaded; it cannot recreate
// an old token that has been deleted or has expired.
type PostgresSessionStore struct {
	*pgxstore.PostgresStore
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// SCS uses the context-aware methods. The legacy interface methods cannot
// classify a token safely without the request origin.
func (s *PostgresSessionStore) Commit(string, []byte, time.Time) error {
	return errSessionOriginUnavailable
}
func (s *PostgresSessionStore) Delete(string) error { return errSessionOriginUnavailable }
func NewPostgresSessionStore(pool *pgxpool.Pool, logger *slog.Logger) *PostgresSessionStore {
	return &PostgresSessionStore{PostgresStore: pgxstore.NewWithConfig(pool, pgxstore.Config{CleanUpInterval: 5 * time.Minute, TableName: "web_sessions"}), pool: pool, logger: logger}
}
func (s *PostgresSessionStore) FindCtx(ctx context.Context, token string) ([]byte, bool, error) {
	data, found, err := s.PostgresStore.FindCtx(ctx, token)
	if state, ok := ctx.Value(originalSessionTokenKey{}).(*sessionRequestState); ok && err == nil && found && token == state.hash {
		state.mu.Lock()
		if state.loaded == nil {
			state.loaded = append([]byte(nil), data...)
		}
		state.mu.Unlock()
	}
	return data, found, err
}
func unchangedSession(state *sessionRequestState, data []byte, expiry time.Time) bool {
	state.mu.Lock()
	baseline := append([]byte(nil), state.loaded...)
	state.mu.Unlock()
	if len(baseline) == 0 {
		return false
	}
	codec := scs.GobCodec{}
	before, beforeValues, e1 := codec.Decode(baseline)
	after, afterValues, e2 := codec.Decode(data)
	return e1 == nil && e2 == nil && before.Equal(after) && !expiry.After(after) && reflect.DeepEqual(beforeValues, afterValues)
}

// LockVerifiedOIDCSession binds the existing SCS row to the verified actor for
// a READ COMMITTED material transaction. Callers must recheck the returned row
// and expiry in their decision statement and must never log or expose it.
func LockVerifiedOIDCSession(ctx context.Context, tx pgx.Tx, actor identity.Actor) (OIDCSessionProof, error) {
	if tx == nil {
		return OIDCSessionProof{}, ErrOIDCSessionProofUnavailable
	}
	proof, err := VerifiedOIDCSession(ctx, actor)
	if err != nil {
		return OIDCSessionProof{}, ErrOIDCSessionProofUnavailable
	}
	var current []byte
	err = tx.QueryRow(ctx, `SELECT data FROM web_sessions WHERE token=$1 AND expiry>clock_timestamp() AND current_setting('transaction_isolation')='read committed' FOR SHARE`, proof.TokenHash).Scan(&current)
	if err != nil {
		if ctx.Err() != nil {
			return OIDCSessionProof{}, ctx.Err()
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return OIDCSessionProof{}, ErrOIDCSessionProofUnavailable
		}
		return OIDCSessionProof{}, err
	}
	state, ok := ctx.Value(originalSessionTokenKey{}).(*sessionRequestState)
	if !ok || !unchangedSession(state, current, actor.ExpiresAt) {
		return OIDCSessionProof{}, ErrOIDCSessionProofUnavailable
	}
	proof.Data = append([]byte(nil), current...)
	return proof, nil
}
func (s *PostgresSessionStore) CommitCtx(ctx context.Context, token string, data []byte, expiry time.Time) error {
	state, ok := ctx.Value(originalSessionTokenKey{}).(*sessionRequestState)
	if !ok {
		return errSessionOriginUnavailable
	}
	if token == state.hash {
		// A missing row stays missing after logout, rotation or expiry.
		_, err := s.pool.Exec(ctx, `UPDATE web_sessions SET data=$2,expiry=$3 WHERE token=$1 AND expiry>clock_timestamp() AND $3::timestamptz>clock_timestamp()`, token, data, expiry)
		if err != nil && s.logger != nil && !state.securityChange.Load() && unchangedSession(state, data, expiry) {
			s.logger.Warn("session idle refresh failed")
			return nil
		}
		return err
	}
	tag, err := s.pool.Exec(ctx, `INSERT INTO web_sessions(token,data,expiry) SELECT $1,$2,$3 WHERE $3::timestamptz>clock_timestamp() ON CONFLICT(token) DO NOTHING`, token, data, expiry)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errSessionNoLongerActive
	}
	return nil
}
func (s *PostgresSessionStore) DeleteCtx(ctx context.Context, token string) error {
	state, ok := ctx.Value(originalSessionTokenKey{}).(*sessionRequestState)
	if !ok {
		return errSessionOriginUnavailable
	}
	state.securityChange.Store(true)
	if token == emptySessionTokenHash() {
		return nil
	}
	if token != state.hash {
		return errSessionNoLongerActive
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM web_sessions WHERE token=$1 AND expiry>clock_timestamp()`, token)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errSessionNoLongerActive
	}
	return nil
}
