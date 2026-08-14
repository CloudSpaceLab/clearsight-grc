//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	sourceCatalogTenantID = "51111111-1111-7111-8111-111111111111"
	sourceCatalogEntityID = "52222222-2222-7222-8222-222222222222"
	sourceCatalogActorID  = "53333333-3333-7333-8333-333333333333"
	sourceCatalogSourceID = "54444444-4444-7444-8444-444444444444"
)

func TestCreateSourceStoresEndpointOnlyAsReferenceConnection(t *testing.T) {
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
	cleanupSourceCatalogFixture(ctx, pool)
	t.Cleanup(func() { cleanupSourceCatalogFixture(context.Background(), pool) })

	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'evidence-source-catalog','Evidence source catalog')`, sourceCatalogTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,'BANK-NG','Evidence Bank','Nigeria')`, sourceCatalogEntityID, sourceCatalogTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($1::uuid,$2::uuid,'PERSON','source-owner','Source owner')`, sourceCatalogActorID, sourceCatalogTenantID); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	repository := NewPostgresRepository(pool)
	created, err := repository.CreateSource(ctx, Source{
		ID: sourceCatalogSourceID,
		TenantID: sourceCatalogTenantID,
		LegalEntityID: sourceCatalogEntityID,
		Code: "CORE-BANKING",
		Name: "Core banking",
		Type: SourceSystem,
		AuthorityClass: "SYSTEM_OF_RECORD",
		OwnerPrincipalID: sourceCatalogActorID,
		Endpoint: " https://core.example.invalid/reference ",
		ExpectedFreshnessMinutes: 15,
		Health: HealthUnknown,
		Status: SourceActive,
		Version: 1,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Endpoint != "" {
		t.Fatalf("source returned a legacy endpoint: %#v", created)
	}
	encoded, err := json.Marshal(created)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "endpoint") {
		t.Fatalf("source JSON exposed endpoint state: %s", encoded)
	}

	catalog := sourceaccess.NewPostgresCatalogRepository(pool)
	connections, err := catalog.ListCurrentConnections(ctx, sourceCatalogTenantID, sourceCatalogSourceID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(connections) != 1 {
		t.Fatalf("reference connections=%d want=1", len(connections))
	}
	connection := connections[0]
	if connection.Code != sourceaccess.ReferenceConnectionCode || connection.AdapterKind != sourceaccess.AdapterReference || connection.AdapterVersion != sourceaccess.ReferenceAdapterVersion || connection.SecretRef != "" {
		t.Fatalf("unexpected reference connection: %#v", connection)
	}
	var definition map[string]string
	if err := json.Unmarshal(connection.Definition, &definition); err != nil {
		t.Fatal(err)
	}
	if definition["endpoint"] != "https://core.example.invalid/reference" {
		t.Fatalf("reference endpoint=%q", definition["endpoint"])
	}

	var endpointColumns int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='evidence_sources' AND column_name='endpoint'`).Scan(&endpointColumns); err != nil {
		t.Fatal(err)
	}
	if endpointColumns != 0 {
		t.Fatal("evidence_sources still has an endpoint column")
	}

	withoutReferenceID := "55555555-5555-7555-8555-555555555555"
	if _, err := repository.CreateSource(ctx, Source{
		ID: withoutReferenceID,
		TenantID: sourceCatalogTenantID,
		LegalEntityID: sourceCatalogEntityID,
		Code: "MANUAL-SOURCE",
		Name: "Manual source",
		Type: SourceHuman,
		AuthorityClass: "ATTESTATION",
		OwnerPrincipalID: sourceCatalogActorID,
		ExpectedFreshnessMinutes: 1440,
		Health: HealthUnknown,
		Status: SourceActive,
		Version: 1,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	connections, err = catalog.ListCurrentConnections(ctx, sourceCatalogTenantID, withoutReferenceID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(connections) != 0 {
		t.Fatalf("blank endpoint created a reference connection: %#v", connections)
	}
}

func cleanupSourceCatalogFixture(ctx context.Context, pool *pgxpool.Pool) {
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, sourceCatalogTenantID)
}
