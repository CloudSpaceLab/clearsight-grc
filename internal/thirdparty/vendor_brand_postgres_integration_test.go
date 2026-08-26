//go:build postgres && postgresintegration

package thirdparty

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const vendorBrandBaselineVendorID = "33333333-3333-7333-8333-333333333335"

func TestVendorBrandMigrationBackfillsIdentityOnceBeforeTheFirstUpdate(t *testing.T) {
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

	down := readVendorBrandMigration(t, "../../migrations/000047_vendor_brand_assets.down.sql")
	up := readVendorBrandMigration(t, "../../migrations/000047_vendor_brand_assets.up.sql")
	if _, err := pool.Exec(ctx, down); err != nil {
		t.Fatalf("roll back vendor brand migration: %v", err)
	}
	migrationIsDown := true
	t.Cleanup(func() {
		if migrationIsDown {
			if _, cleanupErr := pool.Exec(context.Background(), up); cleanupErr != nil {
				t.Errorf("restore vendor brand migration: %v", cleanupErr)
			}
		}
	})

	if _, err := pool.Exec(ctx, `TRUNCATE tenants CASCADE`); err != nil {
		t.Fatal(err)
	}
	baselineAt := time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'third-party-bank','Third Party Bank');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction)
		VALUES($2::uuid,$1::uuid,'ENTITY-A','Entity A','Nigeria');
		INSERT INTO principals(id,tenant_id,kind,display_name,status)
		VALUES($3::uuid,$1::uuid,'PERSON','Vendor Owner','ACTIVE');
		INSERT INTO third_parties(
			id,tenant_id,legal_name,trading_name,registration_ref,jurisdiction,source_id,external_ref,status,created_at,updated_at,version
		) VALUES($4::uuid,$1::uuid,'Legacy Processing Limited','Legacy Processing','RC-10001','Nigeria','procurement','vendor-10001','ACTIVE',$5,$5,4);
		INSERT INTO third_party_relationships(
			tenant_id,legal_entity_id,vendor_id,service_name,business_owner_principal_id,criticality,privacy_role,status,created_at,updated_at,version
		) VALUES($1::uuid,$2::uuid,$4::uuid,'Card processing',$3::uuid,'IMPORTANT','PROCESSOR','ACTIVE',$5,$5,1)`,
		thirdPartyTenantID, thirdPartyEntityA, thirdPartyPrincipal, vendorBrandBaselineVendorID, baselineAt); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, up); err != nil {
		t.Fatalf("apply vendor brand migration: %v", err)
	}
	migrationIsDown = false

	assertVendorIdentitySnapshot(t, pool, 4, "VendorIdentityCreated", map[string]string{
		"legal_name":       "Legacy Processing Limited",
		"trading_name":     "Legacy Processing",
		"registration_ref": "RC-10001",
		"jurisdiction":     "Nigeria",
		"website_domain":   "",
		"status":           "ACTIVE",
	})

	service := NewService(NewPostgresRepository(pool))
	service.now = func() time.Time { return baselineAt.Add(time.Hour) }
	service.ConfigureIdentityAuthority(&vendorIdentityGuardStub{})
	updated, err := service.UpdateVendorIdentity(
		vendorIdentityContext("third-party-bank", thirdPartyEntityA, thirdPartyPrincipal, service.now()),
		Actor{TenantID: "untrusted", LegalEntityID: "untrusted", PrincipalID: "untrusted"},
		vendorBrandBaselineVendorID,
		UpdateVendorIdentityInput{
			ExpectedVersion: 4,
			LegalName:       "Current Processing Limited",
			TradingName:     "Current Processing",
			RegistrationRef: "RC-20002",
			Jurisdiction:    "Ghana",
			WebsiteDomain:   "vendor.example",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 5 {
		t.Fatalf("updated vendor version = %d, want 5", updated.Version)
	}
	assertVendorIdentitySnapshot(t, pool, 5, "VendorIdentityUpdated", map[string]string{
		"legal_name":       "Current Processing Limited",
		"trading_name":     "Current Processing",
		"registration_ref": "RC-20002",
		"jurisdiction":     "Ghana",
		"website_domain":   "vendor.example",
		"status":           "ACTIVE",
	})

	if _, err := pool.Exec(ctx, down); err != nil {
		t.Fatalf("roll back vendor brand migration after update: %v", err)
	}
	migrationIsDown = true
	if _, err := pool.Exec(ctx, up); err != nil {
		t.Fatalf("reapply vendor brand migration: %v", err)
	}
	migrationIsDown = false
	var eventCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM third_party_events
		WHERE tenant_id=$1::uuid AND aggregate_type='VENDOR' AND aggregate_id=$2::uuid`,
		thirdPartyTenantID, vendorBrandBaselineVendorID).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if eventCount != 2 {
		t.Fatalf("vendor identity event count after migration reapply = %d, want 2", eventCount)
	}
}

func TestVendorBrandDatabaseRejectsLegacyIPv4Spellings(t *testing.T) {
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

	for _, domain := range []string{"2130706433", "127.1", "0177.0.0.1", "0x7f000001", "0x7f.0.0.0x1", "0300.0250.0001.0001", "127.0x0.01"} {
		var valid bool
		err := pool.QueryRow(ctx, `SELECT third_party_website_domain_valid($1)`, domain).Scan(&valid)
		if err != nil {
			t.Fatalf("validate %q: %v", domain, err)
		}
		if valid {
			t.Errorf("database accepted legacy IPv4 spelling %q", domain)
		}
	}
	for _, domain := range []string{"2130706433.example", "127.1.vendor.example"} {
		var valid bool
		if err := pool.QueryRow(ctx, `SELECT third_party_website_domain_valid($1)`, domain).Scan(&valid); err != nil {
			t.Fatalf("validate %q: %v", domain, err)
		}
		if !valid {
			t.Errorf("database rejected hostname with numeric labels %q", domain)
		}
	}
}

func readVendorBrandMigration(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func assertVendorIdentitySnapshot(t *testing.T, pool *pgxpool.Pool, version int64, eventType string, want map[string]string) {
	t.Helper()
	var gotEventType string
	var actorID *string
	var payloadBytes []byte
	if err := pool.QueryRow(context.Background(), `
		SELECT event_type,actor_principal_id::text,payload
		FROM third_party_events
		WHERE tenant_id=$1::uuid AND aggregate_type='VENDOR' AND aggregate_id=$2::uuid AND aggregate_version=$3`,
		thirdPartyTenantID, vendorBrandBaselineVendorID, version).Scan(&gotEventType, &actorID, &payloadBytes); err != nil {
		t.Fatal(err)
	}
	if gotEventType != eventType {
		t.Fatalf("event type at version %d = %q, want %q", version, gotEventType, eventType)
	}
	if version == 4 && actorID != nil {
		t.Fatalf("baseline event invented actor %q", *actorID)
	}
	var got map[string]string
	if err := json.Unmarshal(payloadBytes, &got); err != nil {
		t.Fatal(err)
	}
	for field, wantValue := range want {
		if got[field] != wantValue {
			t.Errorf("version %d payload %s = %q, want %q", version, field, got[field], wantValue)
		}
	}
}
