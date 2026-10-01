//go:build postgres

package runtimecontext

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxLegalEntityScopes = 256

type PostgresResolver struct {
	pool *pgxpool.Pool
}

func NewPostgresResolver(pool *pgxpool.Pool) *PostgresResolver {
	return &PostgresResolver{pool: pool}
}

func (r *PostgresResolver) Resolve(ctx context.Context, scope Scope) (DisplayContext, error) {
	scope.TenantID = strings.TrimSpace(scope.TenantID)
	scope.LegalEntityID = strings.TrimSpace(scope.LegalEntityID)
	scope.PrincipalID = strings.TrimSpace(scope.PrincipalID)
	if r == nil || r.pool == nil || scope.TenantID == "" || scope.LegalEntityID == "" || scope.PrincipalID == "" {
		return DisplayContext{}, ErrInvalid
	}

	var value DisplayContext
	err := r.pool.QueryRow(ctx, `
		SELECT t.name,le.name,p.display_name
		FROM tenants t
		JOIN legal_entities le ON le.tenant_id=t.id
		JOIN principals p ON p.tenant_id=t.id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND p.id::text=$3
		  AND p.status='ACTIVE'
		  AND le.valid_from<=clock_timestamp()
		  AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
		  AND p.valid_from<=clock_timestamp()
		  AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)
		LIMIT 1`, scope.TenantID, scope.LegalEntityID, scope.PrincipalID).
		Scan(&value.TenantName, &value.LegalEntityName, &value.PrincipalName)
	if errors.Is(err, pgx.ErrNoRows) {
		return DisplayContext{}, ErrNotFound
	}
	if err != nil {
		return DisplayContext{}, err
	}
	return value, nil
}

func (r *PostgresResolver) ResolveHierarchy(ctx context.Context, scope Scope) (ScopeHierarchy, error) {
	scope.TenantID = strings.TrimSpace(scope.TenantID)
	scope.LegalEntityID = strings.TrimSpace(scope.LegalEntityID)
	scope.PrincipalID = strings.TrimSpace(scope.PrincipalID)
	if r == nil || r.pool == nil || scope.TenantID == "" || scope.LegalEntityID == "" || scope.PrincipalID == "" {
		return ScopeHierarchy{}, ErrInvalid
	}
	if _, err := r.Resolve(ctx, scope); err != nil {
		return ScopeHierarchy{}, err
	}

	rows, err := r.pool.Query(ctx, `
		WITH selected_tenant AS (
			SELECT t.id,t.slug,t.name
			FROM tenants t
			WHERE t.id::text=$1 OR t.slug=$1
			LIMIT 1
		), active_principal AS (
			SELECT p.id,p.tenant_id
			FROM principals p
			JOIN selected_tenant t ON t.id=p.tenant_id
			WHERE p.id::text=$3
			  AND p.status='ACTIVE'
			  AND p.valid_from<=clock_timestamp()
			  AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)
		)
		SELECT
			t.id::text,
			t.slug,
			t.name,
			le.id::text,
			le.code,
			le.name,
			COALESCE(le.jurisdiction,''),
			(le.id::text=$2 OR le.code=$2) AS current_scope
		FROM selected_tenant t
		JOIN legal_entities le ON le.tenant_id=t.id
		CROSS JOIN active_principal p
		WHERE le.valid_from<=clock_timestamp()
		  AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
		  AND (
			(le.id::text=$2 OR le.code=$2)
			OR EXISTS (
				SELECT 1
				FROM org_positions op
				WHERE op.tenant_id=t.id
				  AND op.occupant_principal_id=p.id
				  AND (op.legal_entity_id IS NULL OR op.legal_entity_id=le.id)
				  AND op.valid_from<=clock_timestamp()
				  AND (op.valid_until IS NULL OR clock_timestamp()<op.valid_until)
			)
			OR EXISTS (
				SELECT 1
				FROM scim_users su
				JOIN scim_sources ss
				  ON ss.tenant_id=su.tenant_id
				 AND ss.id=su.source_id
				 AND ss.status='ACTIVE'
				JOIN directory_group_members dgm
				  ON dgm.tenant_id=su.tenant_id
				 AND dgm.scim_user_id=su.id
				JOIN directory_groups dg
				  ON dg.tenant_id=dgm.tenant_id
				 AND dg.id=dgm.group_id
				 AND dg.source_id=su.source_id
				JOIN directory_group_role_bindings dgrb
				  ON dgrb.tenant_id=dg.tenant_id
				 AND dgrb.group_id=dg.id
				JOIN role_templates rt
				  ON rt.tenant_id=dgrb.tenant_id
				 AND rt.id=dgrb.role_template_id
				WHERE su.tenant_id=t.id
				  AND su.principal_id=p.id
				  AND su.active
				  AND su.deleted_at IS NULL
				  AND dg.deleted_at IS NULL
				  AND dgrb.legal_entity_id=le.id
				  AND dgrb.valid_from<=clock_timestamp()
				  AND (dgrb.valid_until IS NULL OR clock_timestamp()<dgrb.valid_until)
				  AND rt.valid_from<=clock_timestamp()
				  AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
			)
		  )
		ORDER BY current_scope DESC,lower(le.name),le.id
		LIMIT $4`, scope.TenantID, scope.LegalEntityID, scope.PrincipalID, maxLegalEntityScopes+1)
	if err != nil {
		return ScopeHierarchy{}, fmt.Errorf("resolve scope hierarchy: %w", err)
	}
	defer rows.Close()

	state := HierarchyComplete
	nodes := make([]ScopeNode, 0, maxLegalEntityScopes)
	var root ScopeNode
	var current ScopeNode
	for rows.Next() {
		var (
			tenantID, tenantCode, tenantName string
			entityID, entityCode, entityName string
			jurisdiction                     string
			isCurrent                        bool
		)
		if err := rows.Scan(&tenantID, &tenantCode, &tenantName, &entityID, &entityCode, &entityName, &jurisdiction, &isCurrent); err != nil {
			return ScopeHierarchy{}, fmt.Errorf("scan scope hierarchy: %w", err)
		}
		if root.ID == "" {
			root = ScopeNode{ID: tenantID, Code: tenantCode, Name: tenantName, Kind: ScopeKindOrganization}
		}
		node := ScopeNode{
			ID:           entityID,
			Code:         entityCode,
			Name:         entityName,
			Kind:         ScopeKindLegalEntity,
			ParentID:     root.ID,
			Jurisdiction: jurisdiction,
			Current:      isCurrent,
		}
		if isCurrent {
			current = node
		}
		if len(nodes) < maxLegalEntityScopes {
			nodes = append(nodes, node)
		} else {
			state = HierarchyTruncated
		}
	}
	if err := rows.Err(); err != nil {
		return ScopeHierarchy{}, fmt.Errorf("iterate scope hierarchy: %w", err)
	}
	if root.ID == "" || current.ID == "" {
		return ScopeHierarchy{}, ErrNotFound
	}
	return ScopeHierarchy{State: state, Root: root, Current: current, LegalEntities: nodes}, nil
}

var _ Resolver = (*PostgresResolver)(nil)
var _ HierarchyResolver = (*PostgresResolver)(nil)
