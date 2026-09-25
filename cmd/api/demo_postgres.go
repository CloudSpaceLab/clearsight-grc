//go:build postgres

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/reporting"
	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The PostgreSQL demo is seeded by deploy/scripts/seed-demo-foundation.sh, so
// these name records that script already creates. They are looked up by slug,
// code or external reference rather than hard-coded as UUIDs, so the installer
// and the seed script cannot drift apart.
const (
	demoTenantSlug  = "clearsight-demo"
	demoEntityCode  = "BANK-NG"
	demoProgramCode = "NDPA-2023"
	demoMatterRef   = "DEMO-GAID-2025"

	// These are the principals the demo routing policy actually authorises, and
	// they must be three different people: the report governance rules refuse a
	// definition reviewed by its proposer or activated by either. The demo
	// routing rules in deploy/scripts/seed-demo-foundation.sh bind PROPOSER to
	// the CCO, REVIEWER to the Internal Auditor and AUTHORIZER to the CRO.
	demoMakerPrincipalRef      = "demo-cco"
	demoReviewerPrincipalRef   = "demo-auditor"
	demoAuthorizerPrincipalRef = "demo-cro"
)

type postgresDemoScope struct {
	TenantID              string
	LegalEntityID         string
	MakerPrincipalID      string
	ReviewerPrincipalID   string
	AuthorizerPrincipalID string
}

// ensureDemoSourceEmployeePrincipals creates the principals the seeded
// register names as accountable owners.
//
// The memory repository has no foreign keys, so it never needed these rows; a
// PostgreSQL register does, because an activity owner references a principal.
// The identifiers are derived from the display name, so the same name always
// resolves to the same principal in every composition and a repeat install
// inserts nothing.
//
// This lives here rather than in internal/ropa because creating principals is a
// cross-aggregate concern: the register names its owners, but the identity
// aggregate owns the principal records.
func ensureDemoSourceEmployeePrincipals(ctx context.Context, pool *pgxpool.Pool, tenantID string) error {
	for _, displayName := range ropa.DemoOwnerDisplayNames() {
		principalID := identity.DemoSourceEmployeePrincipalID(displayName)
		if _, err := pool.Exec(ctx, `
			INSERT INTO principals(id, tenant_id, kind, external_ref, display_name)
			VALUES($1::uuid, $2::uuid, 'PERSON', $3, $4)
			ON CONFLICT (id) DO NOTHING`,
			principalID, tenantID, "sample-employee-"+strings.ToLower(displayName), displayName,
		); err != nil {
			return fmt.Errorf("ensure the demo source employee principal for %q: %w", displayName, err)
		}
	}
	return nil
}

// installPostgresDemo seeds the register and its governed report definitions
// for the deployed demo.
//
// The memory composition installs its own demo data, but the deployed demo runs
// this composition. Without this call the live register and report workspace are
// empty while the review evidence — which runs the memory composition — shows a
// populated register. That divergence is what let an empty register reach
// production, so the two compositions are now seeded from the same seeds.
func installPostgresDemo(ctx context.Context, pool *pgxpool.Pool, ropaService *ropa.Service, reportingService *reporting.Service) error {
	scope, err := resolvePostgresDemoScope(ctx, pool)
	if err != nil {
		return err
	}

	// The register seeds first. A report scoped to a Program or Matter is a
	// report over that estate, and an estate that does not exist yet would make
	// the definition a reference to nothing.
	if err := ensureDemoSourceEmployeePrincipals(ctx, pool, scope.TenantID); err != nil {
		return err
	}
	if err := ropa.InstallDemoInto(ctx, ropaService, ropa.DemoScope{
		TenantID:            scope.TenantID,
		LegalEntityID:       scope.LegalEntityID,
		OwnerPrincipalID:    scope.MakerPrincipalID,
		ReviewerPrincipalID: scope.ReviewerPrincipalID,
		ActorPrincipalID:    scope.MakerPrincipalID,
		RequiredAuthorityID: scope.AuthorizerPrincipalID,
		RequiredCISOID:      scope.ReviewerPrincipalID,
	}); err != nil {
		return fmt.Errorf("install the demo processing activity register: %w", err)
	}

	refs := make(map[string]string, 2)
	for code, spec := range map[string]struct {
		query string
		value string
	}{
		"ROPA-CROSS-BORDER-TRANSFERS": {
			query: `SELECT id::text FROM programs WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND code=$3`,
			value: demoProgramCode,
		},
		"ISSUES-OVERDUE-OBLIGATIONS": {
			query: `SELECT id::text FROM matters WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND reference=$3`,
			value: demoMatterRef,
		},
	} {
		var resolved string
		if err := pool.QueryRow(ctx, spec.query, scope.TenantID, scope.LegalEntityID, spec.value).Scan(&resolved); err != nil {
			// A report scoped to an absent Program or Matter is skipped rather
			// than written with a dangling reference, so one missing sample
			// record does not take the whole demo down.
			continue
		}
		refs[code] = resolved
	}

	if err := reporting.InstallPostgresDemo(ctx, reportingService, reporting.DemoScope{
		TenantID:              scope.TenantID,
		LegalEntityID:         scope.LegalEntityID,
		MakerPrincipalID:      scope.MakerPrincipalID,
		ReviewerPrincipalID:   scope.ReviewerPrincipalID,
		AuthorizerPrincipalID: scope.AuthorizerPrincipalID,
		ScopeRefs:             refs,
	}); err != nil {
		return fmt.Errorf("install the demo report definitions: %w", err)
	}
	return nil
}

// resolvePostgresDemoScope reads the seeded demo foundation and returns
// resolved identifiers. Each principal is resolved by external reference so a
// missing record fails loudly here, rather than producing a register whose
// owners point at principals that do not exist.
func resolvePostgresDemoScope(ctx context.Context, pool *pgxpool.Pool) (postgresDemoScope, error) {
	var scope postgresDemoScope
	if err := pool.QueryRow(ctx,
		`SELECT t.id::text, le.id::text
		   FROM tenants t
		   JOIN legal_entities le ON le.tenant_id=t.id
		  WHERE t.slug=$1 AND le.code=$2
		  ORDER BY le.valid_from
		  LIMIT 1`, demoTenantSlug, demoEntityCode,
	).Scan(&scope.TenantID, &scope.LegalEntityID); err != nil {
		return postgresDemoScope{}, fmt.Errorf(
			"resolve demo tenant %q and legal entity %q: %w (has deploy/scripts/seed-demo-foundation.sh been applied?)",
			demoTenantSlug, demoEntityCode, err)
	}

	for field, ref := range map[string]*string{
		"MakerPrincipalID":      &scope.MakerPrincipalID,
		"ReviewerPrincipalID":   &scope.ReviewerPrincipalID,
		"AuthorizerPrincipalID": &scope.AuthorizerPrincipalID,
	} {
		externalRef := map[*string]string{
			&scope.MakerPrincipalID:      demoMakerPrincipalRef,
			&scope.ReviewerPrincipalID:   demoReviewerPrincipalRef,
			&scope.AuthorizerPrincipalID: demoAuthorizerPrincipalRef,
		}[ref]
		if err := pool.QueryRow(ctx,
			`SELECT id::text FROM principals WHERE tenant_id=$1::uuid AND external_ref=$2`,
			scope.TenantID, externalRef,
		).Scan(ref); err != nil {
			return postgresDemoScope{}, fmt.Errorf(
				"resolve demo principal %q for %s: %w (has deploy/scripts/seed-demo-foundation.sh been applied?)",
				externalRef, field, err)
		}
	}
	return scope, nil
}
