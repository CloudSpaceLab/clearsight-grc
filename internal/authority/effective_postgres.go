//go:build postgres

package authority

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// effectivePostgresService keeps the bounded execution path implemented by
// postgresService while extending integrity diagnostics to routes that are
// intentionally excluded from execution because their selector resolves to no
// current principal.
type effectivePostgresService struct{ *postgresService }

func NewEffectivePostgresService(pool *pgxpool.Pool) Service {
	return &effectivePostgresService{postgresService: &postgresService{pool: pool}}
}

func (s *effectivePostgresService) Integrity(ctx context.Context, tenantID string) ([]IntegrityFinding, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	findings, err := s.postgresService.Integrity(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT ear.source_rule_id
		FROM effective_authority_routes ear
		WHERE ear.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND ear.valid_from<=clock_timestamp()
		  AND (ear.valid_until IS NULL OR clock_timestamp()<ear.valid_until)
		  AND CASE
			WHEN ear.selector_kind IN ('PRINCIPAL','TEAM','QUEUE','COMMITTEE') THEN NOT EXISTS (
				SELECT 1 FROM principals p
				WHERE p.tenant_id=ear.tenant_id
				  AND (p.id::text=ear.selector_ref OR p.external_ref=ear.selector_ref)
				  AND p.status='ACTIVE'
				  AND p.valid_from<=clock_timestamp()
				  AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)
				  AND (ear.selector_kind='PRINCIPAL' OR p.kind=ear.selector_kind)
			)
			WHEN ear.selector_kind='POSITION' THEN NOT EXISTS (
				SELECT 1 FROM org_positions op
				JOIN principals p ON p.id=op.occupant_principal_id
				WHERE op.tenant_id=ear.tenant_id
				  AND (op.code=ear.selector_ref OR op.id::text=ear.selector_ref)
				  AND op.valid_from<=clock_timestamp()
				  AND (op.valid_until IS NULL OR clock_timestamp()<op.valid_until)
				  AND p.status='ACTIVE'
				  AND p.valid_from<=clock_timestamp()
				  AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)
			)
			WHEN ear.selector_kind='ROLE' THEN NOT EXISTS (
				SELECT 1 FROM role_templates rt
				JOIN position_role_bindings prb ON prb.role_template_id=rt.id
				JOIN org_positions op ON op.id=prb.position_id
				JOIN principals p ON p.id=op.occupant_principal_id
				WHERE rt.tenant_id=ear.tenant_id
				  AND (rt.code=ear.selector_ref OR rt.id::text=ear.selector_ref)
				  AND rt.valid_from<=clock_timestamp()
				  AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
				  AND prb.valid_from<=clock_timestamp()
				  AND (prb.valid_until IS NULL OR clock_timestamp()<prb.valid_until)
				  AND op.valid_from<=clock_timestamp()
				  AND (op.valid_until IS NULL OR clock_timestamp()<op.valid_until)
				  AND p.status='ACTIVE'
				  AND p.valid_from<=clock_timestamp()
				  AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)
			)
			ELSE true
		  END
		ORDER BY ear.source_rule_id`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("inspect unresolved authority selectors: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ruleID string
		if err := rows.Scan(&ruleID); err != nil {
			return nil, err
		}
		findings = append(findings, IntegrityFinding{
			Type:           "UNRESOLVED_SELECTOR",
			Severity:       "CRITICAL",
			Summary:        "A current authority route does not resolve to an active principal.",
			RequiredAction: "Assign an occupant, bind the role, or replace the selector before relying on this route.",
			RuleIDs:        []string{ruleID},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return findings, nil
}
