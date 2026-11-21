//go:build postgres && postgresintegration

package sourceaccess

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCatalogExactReadsHoldCallerTransactionLocks(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ids := make([]string, 8)
	for i := range ids {
		ids[i], err = id.NewUUIDv7()
		if err != nil {
			t.Fatal(err)
		}
	}
	tenant, actor, source, connectionID, viewID, bindingID := ids[0], ids[1], ids[2], ids[3], ids[4], ids[5]
	slug := "catalog-tx-" + tenant
	defer func() {
		for _, query := range []string{`DELETE FROM source_bindings WHERE tenant_id=$1::uuid`, `DELETE FROM source_views WHERE tenant_id=$1::uuid`, `DELETE FROM source_connections WHERE tenant_id=$1::uuid`, `DELETE FROM evidence_sources WHERE tenant_id=$1::uuid`, `DELETE FROM principals WHERE tenant_id=$1::uuid`, `DELETE FROM tenants WHERE id=$1::uuid`} {
			_, _ = pool.Exec(context.Background(), query, tenant)
		}
	}()
	if _, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Catalog transaction test')`, tenant, slug); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$2::uuid,'PERSON','Owner')`, actor, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO evidence_sources(id,tenant_id,code,name,source_type,authority_class,owner_principal_id,expected_freshness_minutes,health,status,version) VALUES($1::uuid,$2::uuid,'SOURCE','Source','SYSTEM','SYSTEM_OF_RECORD',$3::uuid,15,'UNKNOWN','ACTIVE',1)`, source, tenant, actor); err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresCatalogRepository(pool)
	now := time.Now().UTC().Add(-time.Hour)
	connection := catalogConnectionRevision(now)
	connection.RevisionID = ids[6]
	connection.ConnectionID = connectionID
	connection.TenantID = tenant
	connection.SourceID = source
	connection.CreatedBy = actor
	connection.OwnerPrincipalID = actor
	view := catalogViewRevision(now)
	view.RevisionID = ids[7]
	view.ViewID = viewID
	view.TenantID = tenant
	view.SourceID = source
	view.ConnectionID = connectionID
	view.CreatedBy = actor
	binding := catalogBindingRevision(now)
	binding.RevisionID, err = id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	binding.BindingID = bindingID
	binding.TenantID = tenant
	binding.SourceID = source
	binding.ViewID = viewID
	binding.CreatedBy = actor
	if _, err = repo.CreateConnectionRevision(ctx, connection); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.CreateViewRevision(ctx, view); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.CreateBindingRevision(ctx, binding); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.BindingRevision(WithCatalogProofTransaction(ctx, nil), tenant, bindingID, 1); !errors.Is(err, ErrCatalogStorage) {
		t.Fatalf("nil proof transaction used nontransactional read: %v", err)
	}
	for _, tc := range []struct {
		name, update string
		read         func(context.Context) error
	}{
		{"binding", `UPDATE source_bindings SET status='PAUSED' WHERE binding_id=$1::uuid`, func(c context.Context) error { _, e := repo.BindingRevision(c, tenant, bindingID, 1); return e }},
		{"view", `UPDATE source_views SET status='PAUSED' WHERE view_id=$1::uuid`, func(c context.Context) error { _, e := repo.ViewRevision(c, tenant, viewID, 1); return e }},
		{"connection", `UPDATE source_connections SET status='PAUSED' WHERE connection_id=$1::uuid`, func(c context.Context) error { _, e := repo.ConnectionRevision(c, tenant, connectionID, 1); return e }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			if err = tc.read(WithCatalogProofTransaction(ctx, tx)); err != nil {
				t.Fatal(err)
			}
			changed := make(chan error, 1)
			target := map[string]string{"binding": bindingID, "view": viewID, "connection": connectionID}[tc.name]
			writer, err := pool.Acquire(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer writer.Release()
			pid := writer.Conn().PgConn().PID()
			go func() { _, e := writer.Exec(ctx, tc.update, target); changed <- e }()
			deadline := time.Now().Add(10 * time.Second)
			waiting := false
			for time.Now().Before(deadline) {
				select {
				case err := <-changed:
					t.Fatalf("%s mutation did not wait: %v", tc.name, err)
				default:
				}
				var waitType string
				if err = pool.QueryRow(ctx, `SELECT COALESCE(wait_event_type,'') FROM pg_stat_activity WHERE pid=$1`, pid).Scan(&waitType); err != nil {
					t.Fatal(err)
				}
				if waitType == "Lock" {
					waiting = true
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			if !waiting {
				t.Fatalf("%s mutation never waited on a lock", tc.name)
			}
			if err = tx.Rollback(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-changed:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("mutation stayed blocked")
			}
		})
	}
	for _, tc := range []struct {
		name, update, target string
		read                 func(context.Context) (RevisionStatus, error)
	}{
		{"binding-first", `UPDATE source_bindings SET status='PAUSED' WHERE binding_id=$1::uuid`, bindingID, func(c context.Context) (RevisionStatus, error) {
			v, e := repo.BindingRevision(c, tenant, bindingID, 1)
			return v.Status, e
		}},
		{"view-first", `UPDATE source_views SET status='PAUSED' WHERE view_id=$1::uuid`, viewID, func(c context.Context) (RevisionStatus, error) {
			v, e := repo.ViewRevision(c, tenant, viewID, 1)
			return v.Status, e
		}},
		{"connection-first", `UPDATE source_connections SET status='PAUSED' WHERE connection_id=$1::uuid`, connectionID, func(c context.Context) (RevisionStatus, error) {
			v, e := repo.ConnectionRevision(c, tenant, connectionID, 1)
			return v.Status, e
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writer, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer writer.Rollback(ctx)
			if _, err = writer.Exec(ctx, tc.update, tc.target); err != nil {
				t.Fatal(err)
			}
			proof, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer proof.Rollback(ctx)
			type result struct {
				status RevisionStatus
				err    error
			}
			done := make(chan result, 1)
			go func() { status, err := tc.read(WithCatalogProofTransaction(ctx, proof)); done <- result{status, err} }()
			pid := proof.Conn().PgConn().PID()
			deadline := time.Now().Add(10 * time.Second)
			waiting := false
			for time.Now().Before(deadline) {
				select {
				case got := <-done:
					t.Fatalf("proof did not wait: %+v", got)
				default:
				}
				var waitType string
				if err = pool.QueryRow(ctx, `SELECT COALESCE(wait_event_type,'') FROM pg_stat_activity WHERE pid=$1`, pid).Scan(&waitType); err != nil {
					t.Fatal(err)
				}
				if waitType == "Lock" {
					waiting = true
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			if !waiting {
				t.Fatal("proof never waited on change")
			}
			if err = writer.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case got := <-done:
				if got.err != nil || got.status != RevisionPaused {
					t.Fatalf("post-change status=%s error=%v", got.status, got.err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("proof remained blocked")
			}
		})
	}
}
