//go:build postgres && postgresintegration

package monitoring

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresFormMatterOriginIsScopedPersistedAndImmutable(t *testing.T) {
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
		tenantID    = "9d333333-3333-7333-8333-333333333331"
		entityA     = "9d333333-3333-7333-8333-333333333332"
		entityB     = "9d333333-3333-7333-8333-333333333333"
		principalID = "9d333333-3333-7333-8333-333333333334"
		matterA     = "9d333333-3333-7333-8333-333333333335"
		matterB     = "9d333333-3333-7333-8333-333333333336"
		formID      = "9d333333-3333-7333-8333-333333333337"
		changedID   = "9d333333-3333-7333-8333-333333333338"
		missingID     = "9d333333-3333-7333-8333-333333333339"
		entityBFormID = "9d333333-3333-7333-8333-333333333341"
		tenantSlug    = "form-origin-pg-test"
	)
	now := time.Date(2026, 10, 6, 18, 0, 0, 0, time.UTC)

	cleanupPostgresFormOrigin(ctx, pool, tenantID)
	defer cleanupPostgresFormOrigin(context.Background(), pool, tenantID)

	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Form Origin Test')`, tenantID, tenantSlug); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
			($1::uuid,$3::uuid,'ORIGIN-A','Origin Entity A','NG',$4),
			($2::uuid,$3::uuid,'ORIGIN-B','Origin Entity B','GH',$4)
	`, entityA, entityB, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from)
		VALUES($1::uuid,$2::uuid,'PERSON','Form Origin Maker','ACTIVE',$3)
	`, principalID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO matters(
			id,tenant_id,legal_entity_id,reference,matter_type,status,priority,title,summary,scope,
			known_facts,missing_facts,contradictions,created_at,updated_at,version
		) VALUES
			($1::uuid,$3::uuid,$4::uuid,'MAT-ORIGIN-A','INCIDENT','ASSESSMENT',3,'Origin issue A','Origin issue A.','{}'::jsonb,'{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$6,$6,1),
			($2::uuid,$3::uuid,$5::uuid,'MAT-ORIGIN-B','INCIDENT','ASSESSMENT',3,'Origin issue B','Origin issue B.','{}'::jsonb,'{}'::jsonb,'[]'::jsonb,'[]'::jsonb,$6,$6,1)
	`, matterA, matterB, tenantID, entityA, entityB, now); err != nil {
		t.Fatal(err)
	}

	repository := NewPostgresRepository(pool)
	guard, err := commandauth.New(formAuthorityStub{principal: principalID}, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repository, nil)
	service.ConfigureCommandGuard(guard)
	service.ConfigureFormOriginValidator(formOriginValidatorStub{allowed: map[string]bool{
		tenantSlug + "\x00" + entityA + "\x00" + principalID + "\x00" + matterA: true,
	}})
	service.newID = func() (string, error) { return formID, nil }

	input := validLibraryFormInput()
	input.Origin = &FormOrigin{Type: FormOriginMatter, ID: matterA}
	created, err := service.CreateLibraryForm(formActorContext(tenantSlug, entityA, principalID), input)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := repository.ReusableFormRevision(ctx, tenantID, entityA, created.ID, created.Version)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Origin == nil || stored.Origin.Type != FormOriginMatter || stored.Origin.ID != matterA {
		t.Fatalf("stored form origin = %#v", stored.Origin)
	}

	entityBForm := stored
	entityBForm.ID = entityBFormID
	entityBForm.LegalEntityID = entityB
	entityBForm.Version = 1
	entityBForm.Origin = &FormOrigin{Type: FormOriginMatter, ID: matterB}
	entityBForm.CreatedAt = now.Add(time.Minute)
	entityBForm.UpdatedAt = entityBForm.CreatedAt
	if _, err := repository.CreateFormRevision(ctx, entityBForm); err != nil {
		t.Fatalf("create entity-B origin form: %v", err)
	}

	pageA, err := repository.ListFormLibrary(ctx, FormLibraryFilter{
		TenantID: tenantSlug, LegalEntityID: entityA, OriginType: FormOriginMatter, OriginID: matterA, Limit: 25,
	})
	if err != nil || len(pageA.Items) != 1 || pageA.Items[0].Template.ID != formID {
		t.Fatalf("entity-A origin page = %#v, err = %v", pageA, err)
	}
	crossPage, err := repository.ListFormLibrary(ctx, FormLibraryFilter{
		TenantID: tenantSlug, LegalEntityID: entityA, OriginType: FormOriginMatter, OriginID: matterB, Limit: 25,
	})
	if err != nil || len(crossPage.Items) != 0 {
		t.Fatalf("entity-A cross-origin page = %#v, err = %v", crossPage, err)
	}
	pageB, err := repository.ListFormLibrary(ctx, FormLibraryFilter{
		TenantID: tenantSlug, LegalEntityID: entityB, OriginType: FormOriginMatter, OriginID: matterB, Limit: 25,
	})
	if err != nil || len(pageB.Items) != 1 || pageB.Items[0].Template.ID != entityBFormID {
		t.Fatalf("entity-B origin page = %#v, err = %v", pageB, err)
	}

	changed := stored
	changed.Version++
	changed.Origin = &FormOrigin{Type: FormOriginMatter, ID: matterB}
	changed.CreatedAt = now.Add(time.Minute)
	changed.UpdatedAt = changed.CreatedAt
	if _, err := repository.CreateFormRevision(ctx, changed); !errors.Is(err, ErrInvalid) {
		t.Fatalf("changed origin persistence error = %v, want invalid", err)
	}

	crossEntity := stored
	crossEntity.ID = changedID
	crossEntity.Version = 1
	crossEntity.Origin = &FormOrigin{Type: FormOriginMatter, ID: matterB}
	crossEntity.CreatedAt = now.Add(2 * time.Minute)
	crossEntity.UpdatedAt = crossEntity.CreatedAt
	if _, err := repository.CreateFormRevision(ctx, crossEntity); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-entity origin persistence error = %v, want invalid", err)
	}

	missing := stored
	missing.ID = missingID
	missing.Version = 1
	missing.Origin = &FormOrigin{Type: FormOriginMatter, ID: "9d333333-3333-7333-8333-333333333340"}
	missing.CreatedAt = now.Add(3 * time.Minute)
	missing.UpdatedAt = missing.CreatedAt
	if _, err := repository.CreateFormRevision(ctx, missing); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing origin persistence error = %v, want invalid", err)
	}
}

func cleanupPostgresFormOrigin(ctx context.Context, pool *pgxpool.Pool, tenantID string) {
	_, _ = pool.Exec(ctx, `DELETE FROM outbox_events WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM monitoring_events WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM monitoring_form_templates WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM matters WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM principals WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM legal_entities WHERE tenant_id=$1::uuid`, tenantID)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
}
