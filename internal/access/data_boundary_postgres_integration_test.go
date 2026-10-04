//go:build postgres && postgresintegration

package access

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresLegalEntityDataBoundaryMakerCheckerAndIsolation(t *testing.T) {
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

	const (
		tenantID  = "8e700000-0000-4000-8000-000000000001"
		entityA   = "8e700000-0000-4000-8000-000000000002"
		entityB   = "8e700000-0000-4000-8000-000000000003"
		makerID   = "8e700000-0000-4000-8000-000000000004"
		checkerID = "8e700000-0000-4000-8000-000000000005"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM legal_entity_data_boundary_revisions WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM legal_entity_data_boundaries WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'data-boundary-test','Data Boundary Test');
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
			($2::uuid,$1::uuid,'ENTITY-A','Entity A','NG',$6),
			($3::uuid,$1::uuid,'ENTITY-B','Entity B','GH',$6);
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
			($4::uuid,$1::uuid,'PERSON','Boundary maker','ACTIVE',$6),
			($5::uuid,$1::uuid,'PERSON','Boundary checker','ACTIVE',$6)
	`, tenantID, entityA, entityB, makerID, checkerID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	admin := NewPostgresAdministrator(pool)
	revision, err := admin.ProposeLegalEntityDataBoundary(ctx, ProposeLegalEntityDataBoundaryInput{
		TenantID: "data-boundary-test", LegalEntityID: entityA,
		ResidencyRegion: "ng-primary", DetailTransferMode: DetailTransferAllowlist,
		AllowedDestinationRegions: []string{"eu-west", "GH", "EU-WEST"},
		ExpectedVersion:           0, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if revision.Status != "PENDING" || revision.BaseVersion != 0 ||
		revision.ProposedResidencyRegion != "NG-PRIMARY" ||
		!reflect.DeepEqual(revision.ProposedDestinationRegions, []string{"EU-WEST", "GH"}) {
		t.Fatalf("proposed boundary revision=%#v", revision)
	}

	err = admin.ApproveLegalEntityDataBoundary(ctx, DecideLegalEntityDataBoundaryInput{
		TenantID: "data-boundary-test", LegalEntityID: entityA,
		RevisionID: revision.ID, ActorID: makerID, Rationale: "self approve",
	})
	if !errors.Is(err, ErrAdminMakerChecker) {
		t.Fatalf("maker self-approval error=%v", err)
	}

	if err := admin.ApproveLegalEntityDataBoundary(ctx, DecideLegalEntityDataBoundaryInput{
		TenantID: "data-boundary-test", LegalEntityID: entityA,
		RevisionID: revision.ID, ActorID: checkerID, Rationale: "Residency and destinations reviewed.",
	}); err != nil {
		t.Fatal(err)
	}

	active, err := admin.legalEntityDataBoundary(ctx, tenantID, entityA)
	if err != nil {
		t.Fatal(err)
	}
	if !active.Configured || active.Version != 1 || active.ResidencyRegion != "NG-PRIMARY" ||
		active.DetailTransferMode != DetailTransferAllowlist ||
		!reflect.DeepEqual(active.AllowedDestinationRegions, []string{"EU-WEST", "GH"}) {
		t.Fatalf("active boundary=%#v", active)
	}
	if !LegalEntityDetailTransferAllowed(active, "gh") || LegalEntityDetailTransferAllowed(active, "ZA") {
		t.Fatalf("active transfer decision is inconsistent: %#v", active)
	}

	_, err = admin.ProposeLegalEntityDataBoundary(ctx, ProposeLegalEntityDataBoundaryInput{
		TenantID: "data-boundary-test", LegalEntityID: entityA,
		ResidencyRegion: "NG-PRIMARY", DetailTransferMode: DetailTransferAggregateOnly,
		ExpectedVersion: 0, ActorID: makerID,
	})
	if !errors.Is(err, ErrAdminConflict) {
		t.Fatalf("stale proposal error=%v", err)
	}

	other, err := admin.legalEntityDataBoundary(ctx, tenantID, entityB)
	if err != nil {
		t.Fatal(err)
	}
	if other.Configured || other.Version != 0 || other.DetailTransferMode != DetailTransferAggregateOnly ||
		other.ResidencyRegion != "" || len(other.AllowedDestinationRegions) != 0 {
		t.Fatalf("unconfigured sibling boundary=%#v", other)
	}

	if err := admin.RejectLegalEntityDataBoundary(ctx, DecideLegalEntityDataBoundaryInput{
		TenantID: "data-boundary-test", LegalEntityID: entityB,
		RevisionID: revision.ID, ActorID: checkerID, Rationale: "wrong entity",
	}); !errors.Is(err, ErrAdminNotFound) {
		t.Fatalf("cross-entity revision decision error=%v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO legal_entity_data_boundaries(
			tenant_id,legal_entity_id,residency_region,detail_transfer_mode,allowed_destination_regions
		) VALUES($1::uuid,$2::uuid,'GH','AGGREGATE_ONLY',ARRAY['NG']::text[])
	`, tenantID, entityB); err == nil {
		t.Fatal("database accepted destinations for an aggregate-only policy")
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO legal_entity_data_boundary_revisions(
			tenant_id,legal_entity_id,base_version,
			proposed_residency_region,proposed_detail_transfer_mode,proposed_destination_regions,
			maker_id,status
		) VALUES($1::uuid,$2::uuid,0,'GH','AGGREGATE_ONLY',ARRAY[]::text[],$3::uuid,'REJECTED')
	`, tenantID, entityB, makerID); err == nil {
		t.Fatal("database accepted a decided boundary revision without checker rationale or decision time")
	}

}
