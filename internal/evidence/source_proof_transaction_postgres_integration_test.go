//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSourceProofLocksEntityAndSourcesInCallerTransaction(t *testing.T) {
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
	ids := make([]string, 4)
	for i := range ids {
		ids[i], err = id.NewUUIDv7()
		if err != nil {
			t.Fatal(err)
		}
	}
	tenant, entity, actor, source := ids[0], ids[1], ids[2], ids[3]
	slug := "source-proof-" + tenant
	defer func() {
		for _, query := range []string{`DELETE FROM evidence_sources WHERE tenant_id=$1::uuid`, `DELETE FROM principals WHERE tenant_id=$1::uuid`, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, `DELETE FROM tenants WHERE id=$1::uuid`} {
			_, _ = pool.Exec(context.Background(), query, tenant)
		}
	}()
	_, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$5,'Source proof'); INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES($2::uuid,$1::uuid,'ENTITY','Entity','NG',$6); INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($3::uuid,$1::uuid,'PERSON','Owner','ACTIVE',$6); INSERT INTO evidence_sources(id,tenant_id,legal_entity_id,code,name,source_type,authority_class,owner_principal_id,expected_freshness_minutes,health,status,version) VALUES($4::uuid,$1::uuid,$2::uuid,'SOURCE','Source','SYSTEM','SYSTEM_OF_RECORD',$3::uuid,15,'UNKNOWN','ACTIVE',1)`, pgx.QueryExecModeSimpleProtocol, tenant, entity, actor, source, slug, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresRepository(pool)
	// The proof must use its existing transaction even when a separate entity
	// resolver is configured; otherwise a saturated pool can deadlock.
	service := NewService(repo, NewMemoryObjectStore())
	service.ConfigureLegalEntityResolver(failingProofResolver{})
	proof, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = service.ValidateActiveSourcesForEntity(sourceaccess.WithCatalogProofTransaction(ctx, proof), slug, entity, []string{source}); err != nil {
		t.Fatalf("transactional source proof: %v", err)
	}
	if err = proof.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if err = service.ValidateActiveSourcesForEntity(sourceaccess.WithCatalogProofTransaction(ctx, nil), slug, entity, []string{source}); !errors.Is(err, ErrSourceScopeRequired) {
		t.Fatalf("nil proof transaction: %v", err)
	}
	if err = repo.ValidateActiveSourcesForEntity(sourceaccess.WithCatalogProofTransaction(ctx, nil), slug, entity, []string{source}); !errors.Is(err, ErrSourceScopeRequired) {
		t.Fatalf("nil proof transaction used nontransactional validation: %v", err)
	}
	for _, tc := range []struct{ name, query, target string }{{"source", `UPDATE evidence_sources SET status='RETIRED' WHERE id=$1::uuid`, source}, {"entity", `UPDATE legal_entities SET valid_until=clock_timestamp() WHERE id=$1::uuid`, entity}} {
		t.Run(tc.name, func(t *testing.T) {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			if err = repo.ValidateActiveSourcesForEntity(sourceaccess.WithCatalogProofTransaction(ctx, tx), slug, entity, []string{source}); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			writer, err := pool.Acquire(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer writer.Release()
			pid := writer.Conn().PgConn().PID()
			go func() { _, err := writer.Exec(ctx, tc.query, tc.target); done <- err }()
			deadline := time.Now().Add(10 * time.Second)
			waiting := false
			for time.Now().Before(deadline) {
				select {
				case err := <-done:
					t.Fatalf("%s update did not wait: %v", tc.name, err)
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
				t.Fatalf("%s update never waited on a lock", tc.name)
			}
			if err = tx.Rollback(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("update stayed blocked")
			}
		})
		if tc.name == "source" {
			if _, err = pool.Exec(ctx, `UPDATE evidence_sources SET status='ACTIVE' WHERE id=$1::uuid`, source); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err = pool.Exec(ctx, `UPDATE legal_entities SET valid_until=NULL WHERE id=$1::uuid`, entity); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, query, target string
		want                error
	}{{"source-first", `UPDATE evidence_sources SET status='RETIRED' WHERE id=$1::uuid`, source, ErrSourceScopeMismatch}, {"entity-first", `UPDATE legal_entities SET valid_until=clock_timestamp() WHERE id=$1::uuid`, entity, ErrSourceScopeRequired}} {
		t.Run(tc.name, func(t *testing.T) {
			writer, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer writer.Rollback(ctx)
			if _, err = writer.Exec(ctx, tc.query, tc.target); err != nil {
				t.Fatal(err)
			}
			proof, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer proof.Rollback(ctx)
			done := make(chan error, 1)
			go func() {
				done <- repo.ValidateActiveSourcesForEntity(sourceaccess.WithCatalogProofTransaction(ctx, proof), slug, entity, []string{source})
			}()
			pid := proof.Conn().PgConn().PID()
			deadline := time.Now().Add(10 * time.Second)
			waiting := false
			for time.Now().Before(deadline) {
				select {
				case err := <-done:
					t.Fatalf("proof did not wait for revocation: %v", err)
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
				t.Fatal("proof never waited on revocation")
			}
			if err = writer.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if !errors.Is(err, tc.want) {
					t.Fatalf("proof after revocation: %v", err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("proof stayed blocked")
			}
		})
		if tc.name == "source-first" {
			if _, err = pool.Exec(ctx, `UPDATE evidence_sources SET status='ACTIVE' WHERE id=$1::uuid`, source); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// A transactional proof must not call the independent resolver.
type failingProofResolver struct{}

func (failingProofResolver) ResolveLegalEntity(context.Context, string, string) (string, error) {
	return "", errors.New("unexpected resolver checkout")
}
