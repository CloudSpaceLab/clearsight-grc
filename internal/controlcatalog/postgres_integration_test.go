//go:build postgres && postgresintegration

package controlcatalog

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	riskdomain "github.com/CloudSpaceLab/clearsight-grc/internal/risk"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresControlCatalogUsesExistingProgramImplementationAndExactEntityScope(t *testing.T) {
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

	tenantID, entityA, entityB := catalogID(t), catalogID(t), catalogID(t)
	programA, programB := catalogID(t), catalogID(t)
	objectiveA, objectiveB := catalogID(t), catalogID(t)
	implementationA, implementationB := catalogID(t), catalogID(t)
	suffix := tenantID[len(tenantID)-8:]
	tenantSlug := "catalog-" + suffix
	entityACode, entityBCode := "CAT-A-" + suffix, "CAT-B-" + suffix
	now := time.Date(2026, 10, 2, 13, 15, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,$3)`, tenantID, tenantSlug, "Catalog "+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from) VALUES
		($1::uuid,$2::uuid,$3,'Entity A','NG',$6),
		($4::uuid,$2::uuid,$5,'Entity B','GH',$6)`,
		entityA, tenantID, entityACode, entityB, entityBCode, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,scope,effective_from,created_at,updated_at,version) VALUES
		($1::uuid,$2::uuid,$3::uuid,$4,'Program A','OPERATIONS','ACTIVE','Risk','{}'::jsonb,$8,$8,$8,1),
		($5::uuid,$2::uuid,$6::uuid,$7,'Program B','OPERATIONS','ACTIVE','Risk','{}'::jsonb,$8,$8,$8,1)`,
		programA, tenantID, entityA, "PA-" + suffix, programB, entityB, "PB-" + suffix, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO control_objectives(id,tenant_id,program_id,code,name,outcome,status,created_at,version) VALUES
		($1::uuid,$2::uuid,$3::uuid,'OBJ-A','Objective A','Outcome A','ACTIVE',$7,1),
		($4::uuid,$2::uuid,$5::uuid,'OBJ-B','Objective B','Outcome B','ACTIVE',$7,1)`,
		objectiveA, tenantID, programA, objectiveB, programB, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO control_implementations(
		id,tenant_id,program_id,objective_id,name,description,implementation_type,scope,status,effective_from,created_at,updated_at,version) VALUES
		($1::uuid,$2::uuid,$3::uuid,$4::uuid,'Implementation A','A','OWNER_REVIEW','{}'::jsonb,'IMPLEMENTED',$8,$8,$8,1),
		($5::uuid,$2::uuid,$6::uuid,$7::uuid,'Implementation B','B','OWNER_REVIEW','{}'::jsonb,'IMPLEMENTED',$8,$8,$8,1)`,
		implementationA, tenantID, programA, objectiveA, implementationB, programB, objectiveB, now); err != nil {
		t.Fatal(err)
	}

	service := NewService(NewPostgresRepository(pool))
	service.Now = func() time.Time { return now }
	definition, linkA, err := service.Promote(ctx, PromoteInput{
		TenantID: tenantSlug, LegalEntityID: entityACode, Code: "ACCESS-" + suffix,
		Name: "Privileged access review", Objective: "Privileged access remains approved.",
		Description: "Quarterly review.", Category: "Access",
		ProgramID: programA, ImplementationID: implementationA,
	})
	if err != nil {
		t.Fatal(err)
	}
	if definition.TenantID != tenantID || linkA.LegalEntityID != entityA {
		t.Fatalf("definition=%#v link=%#v", definition, linkA)
	}
	if _, err := service.GetImplementationLink(ctx, tenantSlug, entityBCode, linkA.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-entity link read error=%v", err)
	}

	linkB, err := service.LinkImplementation(ctx, LinkImplementationInput{
		TenantID: tenantSlug, LegalEntityID: entityBCode, DefinitionID: definition.ID,
		ProgramID: programB, ImplementationID: implementationB,
	})
	if err != nil {
		t.Fatal(err)
	}
	if linkB.DefinitionID != definition.ID || linkB.LegalEntityID != entityB {
		t.Fatalf("second link=%#v", linkB)
	}
	if _, err := service.LinkImplementation(ctx, LinkImplementationInput{
		TenantID: tenantSlug, LegalEntityID: entityA, DefinitionID: definition.ID,
		ProgramID: programA, ImplementationID: implementationA,
	}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate implementation link error=%v", err)
	}

	riskService := riskdomain.NewService(riskdomain.NewPostgresRepository(pool))
	riskService.Now = func() time.Time { return now }
	createdRisk, err := riskService.Create(ctx, riskdomain.CreateInput{
		TenantID: tenantSlug, LegalEntityID: entityACode, Code: "RISK-" + suffix,
		Name: "Access governance risk", Statement: "Privileged access may remain inappropriate.",
		Impact: "Unauthorized access may affect critical systems.",
	})
	if err != nil {
		t.Fatal(err)
	}
	linkedRisk, riskControl, err := riskService.LinkControl(ctx, riskdomain.LinkControlInput{
		TenantID: tenantSlug, LegalEntityID: entityACode, RiskID: createdRisk.ID,
		ExpectedRiskVersion: createdRisk.Version, CatalogLinkID: linkA.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if linkedRisk.Version != 2 || riskControl.CatalogLinkID != linkA.ID {
		t.Fatalf("linked risk=%#v control=%#v", linkedRisk, riskControl)
	}
	_, _, err = riskService.LinkControl(ctx, riskdomain.LinkControlInput{
		TenantID: tenantSlug, LegalEntityID: entityACode, RiskID: createdRisk.ID,
		ExpectedRiskVersion: linkedRisk.Version, CatalogLinkID: linkB.ID,
	})
	if !errors.Is(err, riskdomain.ErrInvalid) {
		t.Fatalf("cross-entity Risk control link error=%v", err)
	}
	afterBadLink, err := riskService.Get(ctx, riskdomain.Scope{TenantID: tenantSlug, LegalEntityID: entityACode}, createdRisk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterBadLink.Risk.Version != 2 || len(afterBadLink.Controls) != 1 {
		t.Fatalf("failed cross-entity link changed Risk: %#v", afterBadLink)
	}

	badID := catalogID(t)
	if _, err := pool.Exec(ctx, `
		INSERT INTO control_catalog_implementation_links(
		id,tenant_id,legal_entity_id,definition_id,program_id,implementation_id,created_at)
		VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7)`,
		badID, tenantID, entityB, definition.ID, programA, implementationA, now); err == nil {
		t.Fatal("direct SQL cross-entity control link unexpectedly succeeded")
	}
}

func catalogID(t *testing.T) string {
	t.Helper()
	value, err := platformid.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	return value
}
