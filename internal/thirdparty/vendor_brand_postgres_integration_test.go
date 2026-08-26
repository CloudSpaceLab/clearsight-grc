//go:build postgres && postgresintegration

package thirdparty

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image/color"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/jackc/pgx/v5/pgxpool"
)

const vendorBrandBaselineVendorID = "33333333-3333-7333-8333-333333333335"

func TestPostgresVendorBrandIdempotencyKeyCannotChangeCommand(t *testing.T) {
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
	if _, err := pool.Exec(ctx, `TRUNCATE tenants CASCADE`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'third-party-bank','Third Party Bank');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($2::uuid,$1::uuid,'ENTITY-A','Entity A','Nigeria');
		INSERT INTO principals(id,tenant_id,kind,display_name,status) VALUES($3::uuid,$1::uuid,'PERSON','Vendor Owner','ACTIVE')`,
		thirdPartyTenantID, thirdPartyEntityA, thirdPartyPrincipal); err != nil {
		t.Fatal(err)
	}
	repository := NewPostgresRepository(pool)
	actor := Actor{TenantID: "third-party-bank", LegalEntityID: thirdPartyEntityA, PrincipalID: thirdPartyPrincipal}
	created, err := NewService(repository).CreateRelationship(ctx, actor, validPostgresCreateInput("Card transaction processing"))
	if err != nil {
		t.Fatal(err)
	}
	brands := NewVendorBrandService(repository, evidence.NewMemoryObjectStore(), &vendorIdentityGuardStub{})
	verified := vendorIdentityContext(actor.TenantID, actor.LegalEntityID, actor.PrincipalID, time.Now().UTC())
	if _, err := brands.PutApprovedBrand(verified, created.Vendor.ID, 0, "same-command-key", "image/png", bytes.NewReader(testBrandPNG(t, color.Black))); err != nil {
		t.Fatal(err)
	}
	if _, err := brands.RemoveApprovedBrand(verified, created.Vendor.ID, 1, "same-command-key"); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("cross-command idempotency replay = %v", err)
	}
	var receipts, brandEvents int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM third_party_vendor_brand_command_receipts WHERE tenant_id=$1::uuid AND vendor_id=$2::uuid`, thirdPartyTenantID, created.Vendor.ID).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM third_party_events WHERE tenant_id=$1::uuid AND aggregate_type='VENDOR_BRAND' AND aggregate_id=$2::uuid`, thirdPartyTenantID, created.Vendor.ID).Scan(&brandEvents); err != nil {
		t.Fatal(err)
	}
	if receipts != 1 || brandEvents != 1 {
		t.Fatalf("cross-command replay mutated state: receipts=%d events=%d", receipts, brandEvents)
	}
}

func TestVendorBrandCompletionAndIdentityUpdateUseConsistentLockOrder(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const tenantID = "33333333-3333-7333-8333-333333333340"
	const vendorID = "33333333-3333-7333-8333-333333333341"
	const jobID = "33333333-3333-7333-8333-333333333342"
	const leaseToken = "33333333-3333-7333-8333-333333333343"
	const assetID = "33333333-3333-7333-8333-333333333344"
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	now := time.Now().UTC()
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'vendor-lock-bank','Vendor Lock Bank');
		INSERT INTO third_parties(id,tenant_id,legal_name,trading_name,registration_ref,jurisdiction,source_id,external_ref,website_domain,status,created_at,updated_at,version)
		VALUES($2::uuid,$1::uuid,'Lock Order Vendor','Lock Order Vendor','LOCK-1','Nigeria','test','lock-order','vendor.example','ACTIVE',$3,$3,1);
		INSERT INTO third_party_vendor_brand_jobs(id,tenant_id,vendor_id,vendor_version,job_type,website_domain,state,attempts,available_at,lease_token,lease_expires_at,last_failure_code,created_at,updated_at,version)
		VALUES($4::uuid,$1::uuid,$2::uuid,1,'DISCOVER_ICON','vendor.example','LEASED',1,$3,$5::uuid,$3 + interval '5 minutes','',$3,$3,2)`,
		tenantID, vendorID, now, jobID, leaseToken); err != nil {
		t.Fatal(err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(ctx, `SELECT 1 FROM third_parties WHERE tenant_id=$1::uuid AND id=$2::uuid FOR UPDATE`, tenantID, vendorID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE third_parties SET website_domain='changed.example',version=2,updated_at=clock_timestamp() WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, vendorID); err != nil {
		t.Fatal(err)
	}

	claim := VendorBrandJob{ID: jobID, TenantID: "vendor-lock-bank", VendorID: vendorID, VendorVersion: 1, JobType: VendorBrandDiscoveryJobType, WebsiteDomain: "vendor.example", State: VendorBrandJobLeased, Attempts: 1, LeaseToken: leaseToken, LeaseExpiresAt: timePtrVendorBrand(now.Add(5 * time.Minute)), Version: 2}
	vendor := Vendor{ID: vendorID, TenantID: "vendor-lock-bank", WebsiteDomain: "vendor.example", Version: 1}
	asset := validVendorBrandAssetForTest(now, vendor)
	asset.ID = assetID
	completion := make(chan error, 1)
	go func() {
		_, completeErr := NewPostgresRepository(pool).CompleteVendorBrandJob(ctx, claim, asset, now)
		completion <- completeErr
	}()

	deadline := time.Now().Add(3 * time.Second)
	for {
		var waiting bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_stat_activity
				WHERE datname=current_database() AND pid<>pg_backend_pid() AND wait_event_type='Lock'
				  AND query LIKE '%SELECT version,COALESCE(website_domain%'
			)`).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("completion did not block on the vendor lock")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE third_party_vendor_brand_jobs
		SET vendor_version=2,website_domain='changed.example',state='READY',attempts=0,lease_token=NULL,lease_expires_at=NULL,updated_at=clock_timestamp(),version=version+1
		WHERE tenant_id=$1::uuid AND vendor_id=$2::uuid`, tenantID, vendorID); err != nil {
		t.Fatalf("identity update could not lock brand job after vendor row: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-completion; !errors.Is(err, ErrVendorBrandJobLeaseLost) {
		t.Fatalf("completion after concurrent identity update = %v, want lease loss", err)
	}
}

func timePtrVendorBrand(value time.Time) *time.Time { return &value }

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
	downOverrides := readVendorBrandMigration(t, "../../migrations/000048_vendor_brand_overrides.down.sql")
	upOverrides := readVendorBrandMigration(t, "../../migrations/000048_vendor_brand_overrides.up.sql")
	if _, err := pool.Exec(ctx, downOverrides); err != nil {
		t.Fatalf("roll back vendor brand overrides: %v", err)
	}
	if _, err := pool.Exec(ctx, down); err != nil {
		t.Fatalf("roll back vendor brand migration: %v", err)
	}
	migrationIsDown := true
	t.Cleanup(func() {
		if migrationIsDown {
			if _, cleanupErr := pool.Exec(context.Background(), up); cleanupErr != nil {
				t.Errorf("restore vendor brand migration: %v", cleanupErr)
			}
			if _, cleanupErr := pool.Exec(context.Background(), upOverrides); cleanupErr != nil {
				t.Errorf("restore vendor brand overrides: %v", cleanupErr)
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
	if _, err := pool.Exec(ctx, upOverrides); err != nil {
		t.Fatalf("apply vendor brand overrides: %v", err)
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

	if _, err := pool.Exec(ctx, downOverrides); err != nil {
		t.Fatalf("roll back vendor brand overrides after update: %v", err)
	}
	if _, err := pool.Exec(ctx, down); err != nil {
		t.Fatalf("roll back vendor brand migration after update: %v", err)
	}
	migrationIsDown = true
	if _, err := pool.Exec(ctx, up); err != nil {
		t.Fatalf("reapply vendor brand migration: %v", err)
	}
	if _, err := pool.Exec(ctx, upOverrides); err != nil {
		t.Fatalf("reapply vendor brand overrides: %v", err)
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
