//go:build postgres

package federation

import (
	"context"
	"errors"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errSessionOriginUnavailable = errors.New("session origin unavailable")
var errSessionNoLongerActive = errors.New("session is no longer active")

// PostgresSessionStore keeps the existing SCS store and web_sessions table.
// A request may refresh only the existing token it loaded; it cannot recreate
// an old token that has been deleted or has expired.
type PostgresSessionStore struct {
	*pgxstore.PostgresStore
	pool *pgxpool.Pool
}

// SCS uses the context-aware methods. The legacy interface methods cannot
// classify a token safely without the request origin.
func (s *PostgresSessionStore) Commit(string, []byte, time.Time) error {
	return errSessionOriginUnavailable
}
func (s *PostgresSessionStore) Delete(string) error { return errSessionOriginUnavailable }

func NewPostgresSessionStore(pool *pgxpool.Pool) *PostgresSessionStore {
	return &PostgresSessionStore{PostgresStore: pgxstore.NewWithConfig(pool, pgxstore.Config{CleanUpInterval: 5 * time.Minute, TableName: "web_sessions"}), pool: pool}
}
func (s *PostgresSessionStore) CommitCtx(ctx context.Context, token string, data []byte, expiry time.Time) error {
	original, ok := ctx.Value(originalSessionTokenKey{}).(string)
	if !ok {
		return errSessionOriginUnavailable
	}
	if token == original {
		// No-op if logout, rotation or expiry removed the row while the request ran.
		// Reporting a store error here could turn a committed material action into
		// an apparent failure; the old cookie is unusable on the next request.
		_, err := s.pool.Exec(ctx, `UPDATE web_sessions SET data=$2,expiry=$3 WHERE token=$1 AND expiry>clock_timestamp() AND $3::timestamptz>clock_timestamp()`, token, data, expiry)
		return err
	}
	// A genuinely new token must not overwrite any existing session.
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
	original, ok := ctx.Value(originalSessionTokenKey{}).(string)
	if !ok {
		return errSessionOriginUnavailable
	}
	// SCS hashes its empty token when destroying a request with no session.
	if token == emptySessionTokenHash() {
		return nil
	}
	if token != original {
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
