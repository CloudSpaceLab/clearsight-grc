//go:build postgres && postgresintegration

package documentimport

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	tabularIntegrationTenantID = "7f111111-1111-7111-8111-111111111111"
	tabularIntegrationActorID  = "7f222222-2222-7222-8222-222222222222"
)

func TestPostgresDocumentImportPersistsTabularParserReceipt(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tabularIntegrationTenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tabularIntegrationTenantID) })
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'tabular-import-integration','Tabular import integration')`, tabularIntegrationTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Tabular actor','ACTIVE',clock_timestamp())`, tabularIntegrationActorID, tabularIntegrationTenantID); err != nil {
		t.Fatal(err)
	}
	service := NewService(NewPostgresRepository(pool), evidence.NewMemoryObjectStore())
	document, err := service.Import(ctx, ImportInput{
		TenantID: tabularIntegrationTenantID, FileName: "inventory.ndjson", MediaType: "application/x-ndjson", CreatedBy: tabularIntegrationActorID,
	}, strings.NewReader("{\"id\":\"1\",\"state\":\"ACTIVE\"}\nnot-json\n{\"id\":\"2\",\"state\":\"REVIEW\"}\n"))
	if err != nil {
		t.Fatal(err)
	}
	processed, err := service.Process(ctx, tabularIntegrationTenantID, document.ID)
	if err != nil {
		t.Fatal(err)
	}
	if processed.Tabular == nil || processed.Tabular.ParserVersion != TabularParserVersion || processed.Tabular.RowsTotal != 3 || processed.Tabular.RowsRejected != 1 || len(processed.Tabular.Resources) != 1 || processed.Tabular.Resources[0].SchemaFingerprint == "" {
		t.Fatalf("tabular parser receipt was not persisted: %#v", processed.Tabular)
	}
	loaded, err := service.Get(ctx, tabularIntegrationTenantID, document.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Tabular == nil || loaded.Tabular.RowsRejected != 1 || len(loaded.Tabular.RowErrors) != 1 || loaded.Tabular.Resources[0].SchemaFingerprint != processed.Tabular.Resources[0].SchemaFingerprint {
		t.Fatalf("tabular parser receipt did not survive reload: %#v", loaded.Tabular)
	}
}
