package federation

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresSessionStore struct{ pool *pgxpool.Pool }

func NewPostgresSessionStore(pool *pgxpool.Pool) *PostgresSessionStore {
	return &PostgresSessionStore{pool: pool}
}

func (s *PostgresSessionStore) Delete(token string) error {
	return s.DeleteCtx(context.Background(), token)
}

func (s *PostgresSessionStore) Find(token string) ([]byte, bool, error) {
	return s.FindCtx(context.Background(), token)
}

func (s *PostgresSessionStore) Commit(token string, data []byte, expiry time.Time) error {
	return s.CommitCtx(context.Background(), token, data, expiry)
}

func (s *PostgresSessionStore) DeleteCtx(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM web_sessions WHERE token=$1`, token)
	return err
}

func (s *PostgresSessionStore) FindCtx(ctx context.Context, token string) ([]byte, bool, error) {
	var data []byte
	err := s.pool.QueryRow(ctx, `
		SELECT data FROM web_sessions
		WHERE token=$1 AND expiry>clock_timestamp()`, token).Scan(&data)
	if err == pgx.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func (s *PostgresSessionStore) CommitCtx(ctx context.Context, token string, data []byte, expiry time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO web_sessions(token,data,expiry)
		VALUES($1,$2,$3)
		ON CONFLICT(token) DO UPDATE SET data=EXCLUDED.data,expiry=EXCLUDED.expiry`, token, data, expiry.UTC())
	return err
}

func (s *PostgresSessionStore) DeleteExpired(ctx context.Context, limit int) (int64, error) {
	if limit <= 0 {
		limit = 500
	}
	command, err := s.pool.Exec(ctx, `
		DELETE FROM web_sessions
		WHERE token IN (
			SELECT token FROM web_sessions
			WHERE expiry<=clock_timestamp()
			ORDER BY expiry
			LIMIT $1
		)`, limit)
	if err != nil {
		return 0, err
	}
	return command.RowsAffected(), nil
}
