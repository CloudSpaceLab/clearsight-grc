//go:build postgres && postgresintegration

package presentationprefs

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPresentationPreferencesArePrincipalScopedAndVersioned(t *testing.T) {
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

	const tenantID = "8f640000-0000-4000-8000-000000000001"
	const principalA = "8f640000-0000-4000-8000-000000000002"
	const principalB = "8f640000-0000-4000-8000-000000000003"
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM user_presentation_preferences WHERE tenant_id=$1::uuid`, tenantID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM principals WHERE id IN ($1::uuid,$2::uuid)`, principalA, principalB)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC()
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'presentation-prefs','Presentation Prefs')`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
		($1::uuid,$3::uuid,'PERSON','Viewer A','ACTIVE',$4),
		($2::uuid,$3::uuid,'PERSON','Viewer B','ACTIVE',$4)`,
		principalA, principalB, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	repository := NewPostgresRepository(pool)
	if _, err := repository.Get(ctx, tenantID, principalA); !errors.Is(err, ErrNotFound) {
		t.Fatalf("initial get error=%v", err)
	}
	first, err := repository.Upsert(ctx, Stored{
		TenantID: tenantID, PrincipalID: principalA,
		HomeFocus: HomeFocusPosture, PortfolioLens: PortfolioLensRisks,
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if first.Version != 1 || first.HomeFocus != HomeFocusPosture || first.PortfolioLens != PortfolioLensRisks {
		t.Fatalf("first=%#v", first)
	}
	if _, err := repository.Get(ctx, tenantID, principalB); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other principal read error=%v", err)
	}
	if _, err := repository.Upsert(ctx, Stored{
		TenantID: tenantID, PrincipalID: principalB,
		HomeFocus: HomeFocusPosture, PortfolioLens: PortfolioLensRisks,
	}, 1); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("missing optimistic update error=%v", err)
	}
	if _, err := repository.Get(ctx, tenantID, principalB); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing optimistic update created preferences: %v", err)
	}
	if _, err := repository.Upsert(ctx, Stored{
		TenantID: tenantID, PrincipalID: principalA,
		HomeFocus: HomeFocusMyWork, PortfolioLens: PortfolioLensPrograms,
	}, 0); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale update error=%v", err)
	}
	second, err := repository.Upsert(ctx, Stored{
		TenantID: tenantID, PrincipalID: principalA,
		HomeFocus: HomeFocusMyWork, PortfolioLens: PortfolioLensPrograms,
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if second.Version != 2 || second.HomeFocus != HomeFocusMyWork || second.PortfolioLens != PortfolioLensPrograms {
		t.Fatalf("second=%#v", second)
	}
}
