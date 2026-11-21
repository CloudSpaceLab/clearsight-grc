//go:build postgres && postgresintegration

package monitoring

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresFormLineageSerializesConcurrentRevisions(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	unique, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	schema := "form_lineage_" + strings.ReplaceAll(unique, "-", "")
	if _, err = admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`)
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	const tenant = "00000000-0000-4000-8000-000000000001"
	const entity = "00000000-0000-4000-8000-000000000002"
	const form = "00000000-0000-4000-8000-000000000003"
	_, err = pool.Exec(ctx, `CREATE TABLE tenants(id uuid PRIMARY KEY,slug text); CREATE TABLE monitoring_form_templates(tenant_id uuid,legal_entity_id uuid,id uuid,version bigint); INSERT INTO tenants VALUES('`+tenant+`'::uuid,'bank'); INSERT INTO monitoring_form_templates VALUES('`+tenant+`'::uuid,'`+entity+`'::uuid,'`+form+`'::uuid,1)`)
	if err != nil {
		t.Fatal(err)
	}
	value := FormTemplate{ID: form, TenantID: "bank", LegalEntityID: entity, Lifecycle: Lifecycle{Version: 2}}
	first, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Rollback(ctx)
	if err = lockFormRevisionLineage(ctx, first, value); err != nil {
		t.Fatal(err)
	}
	if _, err = first.Exec(ctx, `INSERT INTO monitoring_form_templates VALUES($1::uuid,$2::uuid,$3::uuid,2)`, tenant, entity, form); err != nil {
		t.Fatal(err)
	}
	second, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Rollback(ctx)
	finished := make(chan error, 1)
	go func() { finished <- lockFormRevisionLineage(ctx, second, value) }()
	select {
	case err := <-finished:
		t.Fatalf("second revision did not wait: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	if err = first.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-finished:
		if !errors.Is(err, ErrConflict) {
			t.Fatalf("expected stale conflict, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("second revision remained blocked")
	}
}
