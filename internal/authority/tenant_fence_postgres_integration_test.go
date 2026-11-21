//go:build postgres && postgresintegration

package authority

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresTenantAuthorityFence(t *testing.T) {
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
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		t.Fatal(err)
	}
	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80
	tenantID := fmt.Sprintf("%x-%x-%x-%x-%x", id[:4], id[4:6], id[6:8], id[8:10], id[10:])
	slug := fmt.Sprintf("authority-fence-test-%x", id[:])
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1,$2,'Authority fence test')`, tenantID, slug); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1 AND slug=$2`, tenantID, slug); err != nil {
			t.Errorf("clean up test tenant: %v", err)
		}
	})
	var tenant pgtype.UUID
	if err := tenant.Scan(tenantID); err != nil {
		t.Fatal(err)
	}

	// The tenant insert trigger must bootstrap an epoch, and the shared command
	// lock must block a writer. PostgreSQL's lock timeout is the test barrier.
	command, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer command.Rollback(ctx)
	version, err := LockTenantAuthorityForCommand(ctx, command, tenant)
	if err != nil || version != 1 {
		t.Fatalf("bootstrap: version=%d err=%v", version, err)
	}
	writer, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Exec(ctx, `SET LOCAL lock_timeout = '100ms'`); err != nil {
		t.Fatal(err)
	}
	_, err = LockTenantAuthorityForMutation(ctx, writer, tenant)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
		t.Fatalf("writer did not wait for command: %v", err)
	}
	_ = writer.Rollback(ctx)
	if err := command.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	// A mutation is exclusive and its epoch increment rolls back with it.
	writer, err = pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	version, err = LockTenantAuthorityForMutation(ctx, writer, tenant)
	if err != nil || version != 2 {
		t.Fatalf("mutation: version=%d err=%v", version, err)
	}
	command, err = pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := command.Exec(ctx, `SET LOCAL lock_timeout = '100ms'`); err != nil {
		t.Fatal(err)
	}
	_, err = LockTenantAuthorityForCommand(ctx, command, tenant)
	if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
		t.Fatalf("command did not wait for writer: %v", err)
	}
	_ = command.Rollback(ctx)
	if err := writer.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	command, err = pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer command.Rollback(ctx)
	version, err = LockTenantAuthorityForCommand(ctx, command, tenant)
	if err != nil || version != 1 {
		t.Fatalf("rollback: version=%d err=%v", version, err)
	}
}
