//go:build postgres && postgresintegration

package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReferenceVendorPersistsAcrossFreshServiceComposition(t *testing.T) {
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

	const (
		tenantID      = "78787878-7878-7787-8787-787878787871"
		entityID      = "78787878-7878-7787-8787-787878787872"
		actorID       = "78787878-7878-7787-8787-787878787873"
		ownerID       = "78787878-7878-7787-8787-787878787874"
		contributorID = "78787878-7878-7787-8787-787878787875"
		reviewerID    = "78787878-7878-7787-8787-787878787876"
		signatoryID   = "78787878-7878-7787-8787-787878787877"
	)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'reference-vendor-bank','Reference Vendor Bank')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction) VALUES($1::uuid,$2::uuid,'BANK-NG','Reference Vendor Bank','Nigeria')`, entityID, tenantID); err != nil {
		t.Fatal(err)
	}
	for _, principal := range []struct{ id, name string }{
		{actorID, "Reference Installer"},
		{ownerID, "Reference Program Owner"},
		{contributorID, "Reference Contributor"},
		{reviewerID, "Reference Reviewer"},
		{signatoryID, "Reference Signatory"},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status) VALUES($1::uuid,$2::uuid,'PERSON',$3,'ACTIVE')`, principal.id, tenantID, principal.name); err != nil {
			t.Fatal(err)
		}
	}

	config := bankverticals.SeedConfig{
		TenantID:               "reference-vendor-bank",
		LegalEntityID:          "BANK-NG",
		BankName:               "Reference Vendor Bank",
		ActorID:                actorID,
		OwnerPrincipalID:       ownerID,
		ContributorPrincipalID: contributorID,
		ReviewerPrincipalID:    reviewerID,
		SignatoryPrincipalID:   signatoryID,
		Now:                    time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC),
	}
	if got := referenceVendorCount(t, pool, tenantID, "third_parties"); got != 0 {
		t.Fatalf("empty reference scope has %d vendors, want 0", got)
	}

	continuityService := continuity.NewServiceWithClock(continuity.NewPostgresRepository(pool), func() time.Time { return config.Now })
	installer := bankverticals.NewService(continuityService, nil)
	vendors := thirdparty.NewService(thirdparty.NewPostgresRepository(pool))
	first, err := installer.EnsureReferenceVendor(ctx, config, vendors)
	if err != nil {
		t.Fatal(err)
	}
	if first.Relationship.LegalEntityID != entityID {
		t.Fatalf("reference vendor entity=%q, want canonical %q", first.Relationship.LegalEntityID, entityID)
	}
	firstEventCount := referenceVendorCount(t, pool, tenantID, "third_party_events")
	firstOutboxCount := referenceVendorOutboxCount(t, pool, tenantID)
	if firstEventCount == 0 || firstOutboxCount == 0 {
		t.Fatalf("reference vendor creation did not emit canonical history: events=%d outbox=%d", firstEventCount, firstOutboxCount)
	}

	freshContinuity := continuity.NewServiceWithClock(continuity.NewPostgresRepository(pool), func() time.Time { return config.Now.Add(time.Hour) })
	freshInstaller := bankverticals.NewService(freshContinuity, nil)
	freshVendors := thirdparty.NewService(thirdparty.NewPostgresRepository(pool))
	second, err := freshInstaller.EnsureReferenceVendor(ctx, config, freshVendors)
	if err != nil {
		t.Fatal(err)
	}
	if second.Vendor.ID != first.Vendor.ID || second.Relationship.ID != first.Relationship.ID {
		t.Fatalf("fresh service composition changed persisted identity: first=%s/%s second=%s/%s", first.Vendor.ID, first.Relationship.ID, second.Vendor.ID, second.Relationship.ID)
	}

	if got := referenceVendorCount(t, pool, tenantID, "third_parties"); got != 1 {
		t.Fatalf("persisted reference vendors=%d, want 1", got)
	}
	if got := referenceVendorCount(t, pool, tenantID, "third_party_relationships"); got != 1 {
		t.Fatalf("persisted reference relationships=%d, want 1", got)
	}
	if got := referenceVendorCount(t, pool, tenantID, "third_party_events"); got != firstEventCount {
		t.Fatalf("fresh-service rerun added relationship history: before=%d after=%d", firstEventCount, got)
	}
	if got := referenceVendorOutboxCount(t, pool, tenantID); got != firstOutboxCount {
		t.Fatalf("fresh-service rerun added relationship outbox events: before=%d after=%d", firstOutboxCount, got)
	}
}

func referenceVendorCount(t *testing.T, pool *pgxpool.Pool, tenantID, table string) int {
	t.Helper()
	allowed := map[string]bool{
		"third_parties":             true,
		"third_party_relationships": true,
		"third_party_events":        true,
	}
	if !allowed[table] {
		t.Fatalf("unsupported reference vendor table %q", table)
	}
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM `+table+` WHERE tenant_id=$1::uuid`, tenantID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func referenceVendorOutboxCount(t *testing.T, pool *pgxpool.Pool, tenantID string) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM outbox_events WHERE tenant_id=$1::uuid AND aggregate_type='VENDOR_RELATIONSHIP'`, tenantID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
