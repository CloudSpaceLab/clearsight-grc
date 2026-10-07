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

const (
	maxLegalEntityScopes             = 256
	maxOrganizationScopes            = 512
	maxOrganizationScopeSelectionIDs = 50000
)

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

	organizationScopes, organizationState, err := r.resolveOrganizationScopes(ctx, scope, current.ID)
	if err != nil {
		return ScopeHierarchy{}, err
	}
	return ScopeHierarchy{
		State: state, Root: root, Current: current, LegalEntities: nodes, OrganizationScopes: organizationScopes,
		OrganizationScopesTruncated: organizationState == HierarchyTruncated,
	}, nil
}

func (r *PostgresResolver) resolveOrganizationScopes(ctx context.Context, scope Scope, currentEntityID string) ([]ScopeNode, HierarchyState, error) {
	page, err := r.queryOrganizationScopes(ctx, scope, currentEntityID, "", maxOrganizationScopes, "", []string{})
	if err != nil {
		return nil, HierarchyUnavailable, err
	}
	state := HierarchyComplete
	if page.HasMore {
		state = HierarchyTruncated
	}
	return page.Items, state, nil
}

func (r *PostgresResolver) SearchOrganizationScopes(ctx context.Context, scope Scope, search string, limit int) (OrganizationScopeSearchPage, error) {
	scope.TenantID = strings.TrimSpace(scope.TenantID)
	scope.LegalEntityID = strings.TrimSpace(scope.LegalEntityID)
	scope.PrincipalID = strings.TrimSpace(scope.PrincipalID)
	search = strings.TrimSpace(search)
	if r == nil || r.pool == nil || scope.TenantID == "" || scope.LegalEntityID == "" || scope.PrincipalID == "" || len(search) < 2 || len(search) > 120 {
		return OrganizationScopeSearchPage{}, ErrInvalid
	}
	if limit <= 0 || limit > 50 {
		limit = 30
	}
	if _, err := r.Resolve(ctx, scope); err != nil {
		return OrganizationScopeSearchPage{}, err
	}
	var currentEntityID string
	err := r.pool.QueryRow(ctx, `
		SELECT le.id::text
		FROM tenants t
		JOIN legal_entities le ON le.tenant_id=t.id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND le.valid_from<=clock_timestamp()
		  AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
		LIMIT 1`, scope.TenantID, scope.LegalEntityID).Scan(&currentEntityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return OrganizationScopeSearchPage{}, ErrNotFound
	}
	if err != nil {
		return OrganizationScopeSearchPage{}, err
	}
	return r.queryOrganizationScopes(ctx, scope, currentEntityID, search, limit, "", []string{})
}

func (r *PostgresResolver) ResolveOrganizationScopeSelection(
	ctx context.Context,
	scope Scope,
	requested string,
	includeDescendants bool,
) (OrganizationScopeSelection, error) {
	scope.TenantID = strings.TrimSpace(scope.TenantID)
	scope.LegalEntityID = strings.TrimSpace(scope.LegalEntityID)
	scope.PrincipalID = strings.TrimSpace(scope.PrincipalID)
	requested = strings.TrimSpace(requested)
	if r == nil || r.pool == nil || scope.TenantID == "" || scope.LegalEntityID == "" || scope.PrincipalID == "" || requested == "" {
		return OrganizationScopeSelection{}, ErrInvalid
	}
	if _, err := r.Resolve(ctx, scope); err != nil {
		return OrganizationScopeSelection{}, err
	}

	var currentEntityID string
	err := r.pool.QueryRow(ctx, `
		SELECT le.id::text
		FROM tenants t
		JOIN legal_entities le ON le.tenant_id=t.id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND le.valid_from<=clock_timestamp()
		  AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
		LIMIT 1`, scope.TenantID, scope.LegalEntityID).Scan(&currentEntityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return OrganizationScopeSelection{}, ErrNotFound
	}
	if err != nil {
		return OrganizationScopeSelection{}, err
	}

	selectedPage, err := r.queryOrganizationScopes(ctx, scope, currentEntityID, "", 1, requested, []string{})
	if err != nil {
		return OrganizationScopeSelection{}, err
	}
	if len(selectedPage.Items) != 1 || selectedPage.Items[0].ID != requested || !selectedPage.Items[0].Filterable {
		return OrganizationScopeSelection{}, ErrNotFound
	}
	selected := selectedPage.Items[0]
	result := OrganizationScopeSelection{Node: selected, IDs: []string{selected.ID}}
	if !includeDescendants {
		return result, nil
	}

	page, err := r.queryOrganizationScopes(
		ctx, scope, currentEntityID, "", maxOrganizationScopeSelectionIDs, "", selected.DepartmentPath,
	)
	if err != nil {
		return OrganizationScopeSelection{}, err
	}
	if page.HasMore {
		return OrganizationScopeSelection{}, ErrInvalid
	}
	result.IDs = result.IDs[:0]
	for _, item := range page.Items {
		if item.Filterable {
			result.IDs = append(result.IDs, item.ID)
		}
	}
	if len(result.IDs) == 0 {
		result.IDs = []string{selected.ID}
	}
	return result, nil
}
func (r *PostgresResolver) queryOrganizationScopes(
	ctx context.Context,
	scope Scope,
	currentEntityID string,
	search string,
	limit int,
	exactID string,
	ancestorPath []string,
) (OrganizationScopeSearchPage, error) {
	if ancestorPath == nil {
		ancestorPath = []string{}
	}
	rows, err := r.pool.Query(ctx, `
		WITH principal AS (
			SELECT p.id,p.tenant_id
			FROM principals p
			JOIN tenants t ON t.id=p.tenant_id
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND p.id::text=$2
			  AND p.status='ACTIVE'
			  AND p.valid_from<=clock_timestamp()
			  AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)
		), global_scope AS (
			SELECT (EXISTS (
				SELECT 1
				FROM principal p
				JOIN org_positions op ON op.tenant_id=p.tenant_id AND op.occupant_principal_id=p.id
				JOIN position_role_bindings prb ON prb.tenant_id=op.tenant_id AND prb.position_id=op.id
				JOIN role_templates rt ON rt.tenant_id=prb.tenant_id AND rt.id=prb.role_template_id
				WHERE op.legal_entity_id=$3::uuid
				  AND cardinality(op.department_path)=0
				  AND (
				        'OVERSIGHT_READ'=ANY(rt.capabilities)
				        OR 'IDENTITY_READ'=ANY(rt.capabilities)
				        OR 'CONFIG_READ'=ANY(rt.capabilities)
				      )
				  AND op.valid_from<=clock_timestamp() AND (op.valid_until IS NULL OR clock_timestamp()<op.valid_until)
				  AND prb.valid_from<=clock_timestamp() AND (prb.valid_until IS NULL OR clock_timestamp()<prb.valid_until)
				  AND rt.valid_from<=clock_timestamp() AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
			) OR EXISTS (
				SELECT 1
				FROM principal p
				JOIN scim_users su ON su.tenant_id=p.tenant_id AND su.principal_id=p.id AND su.active AND su.deleted_at IS NULL
				JOIN scim_sources ss ON ss.tenant_id=su.tenant_id AND ss.id=su.source_id AND ss.status='ACTIVE'
				JOIN directory_group_members dgm ON dgm.tenant_id=su.tenant_id AND dgm.scim_user_id=su.id
				JOIN directory_groups dg ON dg.tenant_id=dgm.tenant_id AND dg.id=dgm.group_id AND dg.deleted_at IS NULL
				JOIN directory_group_role_bindings b ON b.tenant_id=dg.tenant_id AND b.group_id=dg.id
				JOIN role_templates rt ON rt.tenant_id=b.tenant_id AND rt.id=b.role_template_id
				WHERE b.legal_entity_id=$3::uuid
				  AND cardinality(b.department_path)=0
				  AND (
				        'OVERSIGHT_READ'=ANY(rt.capabilities)
				        OR 'IDENTITY_READ'=ANY(rt.capabilities)
				        OR 'CONFIG_READ'=ANY(rt.capabilities)
				      )
				  AND b.valid_from<=clock_timestamp() AND (b.valid_until IS NULL OR clock_timestamp()<b.valid_until)
				  AND rt.valid_from<=clock_timestamp() AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
			)) AS has_global_scope,
			(EXISTS (
				SELECT 1
				FROM principal p
				JOIN org_positions op ON op.tenant_id=p.tenant_id AND op.occupant_principal_id=p.id
				JOIN position_role_bindings prb ON prb.tenant_id=op.tenant_id AND prb.position_id=op.id
				JOIN role_templates rt ON rt.tenant_id=prb.tenant_id AND rt.id=prb.role_template_id
				WHERE op.legal_entity_id=$3::uuid
				  AND cardinality(op.department_path)=0
				  AND 'OVERSIGHT_READ'=ANY(rt.capabilities)
				  AND op.valid_from<=clock_timestamp() AND (op.valid_until IS NULL OR clock_timestamp()<op.valid_until)
				  AND prb.valid_from<=clock_timestamp() AND (prb.valid_until IS NULL OR clock_timestamp()<prb.valid_until)
				  AND rt.valid_from<=clock_timestamp() AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
			) OR EXISTS (
				SELECT 1
				FROM principal p
				JOIN scim_users su ON su.tenant_id=p.tenant_id AND su.principal_id=p.id AND su.active AND su.deleted_at IS NULL
				JOIN scim_sources ss ON ss.tenant_id=su.tenant_id AND ss.id=su.source_id AND ss.status='ACTIVE'
				JOIN directory_group_members dgm ON dgm.tenant_id=su.tenant_id AND dgm.scim_user_id=su.id
				JOIN directory_groups dg ON dg.tenant_id=dgm.tenant_id AND dg.id=dgm.group_id AND dg.deleted_at IS NULL
				JOIN directory_group_role_bindings b ON b.tenant_id=dg.tenant_id AND b.group_id=dg.id
				JOIN role_templates rt ON rt.tenant_id=b.tenant_id AND rt.id=b.role_template_id
				WHERE b.legal_entity_id=$3::uuid
				  AND cardinality(b.department_path)=0
				  AND 'OVERSIGHT_READ'=ANY(rt.capabilities)
				  AND b.valid_from<=clock_timestamp() AND (b.valid_until IS NULL OR clock_timestamp()<b.valid_until)
				  AND rt.valid_from<=clock_timestamp() AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
			)) AS has_global_oversight
		)
		SELECT s.id::text,s.code,s.name,s.kind,COALESCE(s.parent_scope_id::text,''),s.department_path,
		       (
		         g.has_global_oversight
		         OR EXISTS (
		           SELECT 1
		           FROM principal p
		           JOIN org_positions op ON op.tenant_id=p.tenant_id AND op.occupant_principal_id=p.id
		           JOIN position_role_bindings prb ON prb.tenant_id=op.tenant_id AND prb.position_id=op.id
		           JOIN role_templates rt ON rt.tenant_id=prb.tenant_id AND rt.id=prb.role_template_id
		           WHERE op.legal_entity_id=$3::uuid
		             AND op.organization_scope_id=s.id
		             AND 'OVERSIGHT_READ'=ANY(rt.capabilities)
		             AND op.valid_from<=clock_timestamp() AND (op.valid_until IS NULL OR clock_timestamp()<op.valid_until)
		             AND prb.valid_from<=clock_timestamp() AND (prb.valid_until IS NULL OR clock_timestamp()<prb.valid_until)
		             AND rt.valid_from<=clock_timestamp() AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
		         )
		         OR EXISTS (
		           SELECT 1
		           FROM principal p
		           JOIN scim_users su ON su.tenant_id=p.tenant_id AND su.principal_id=p.id AND su.active AND su.deleted_at IS NULL
		           JOIN scim_sources ss ON ss.tenant_id=su.tenant_id AND ss.id=su.source_id AND ss.status='ACTIVE'
		           JOIN directory_group_members dgm ON dgm.tenant_id=su.tenant_id AND dgm.scim_user_id=su.id
		           JOIN directory_groups dg ON dg.tenant_id=dgm.tenant_id AND dg.id=dgm.group_id AND dg.deleted_at IS NULL
		           JOIN directory_group_role_bindings b ON b.tenant_id=dg.tenant_id AND b.group_id=dg.id
		           JOIN role_templates rt ON rt.tenant_id=b.tenant_id AND rt.id=b.role_template_id
		           WHERE b.legal_entity_id=$3::uuid
		             AND b.organization_scope_id=s.id
		             AND 'OVERSIGHT_READ'=ANY(rt.capabilities)
		             AND b.valid_from<=clock_timestamp() AND (b.valid_until IS NULL OR clock_timestamp()<b.valid_until)
		             AND rt.valid_from<=clock_timestamp() AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
		         )
		       ) AS filterable
		FROM organization_scopes s
		CROSS JOIN global_scope g
		WHERE s.tenant_id=(SELECT tenant_id FROM principal)
		  AND s.legal_entity_id=$3::uuid
		  AND s.status='ACTIVE'
		  AND s.valid_from<=clock_timestamp()
		  AND (s.valid_until IS NULL OR clock_timestamp()<s.valid_until)
		  AND ($4='' OR s.search_document @@ websearch_to_tsquery('simple'::regconfig,$4))
		  AND ($6='' OR s.id::text=$6)
		  AND (
		    cardinality($7::text[])=0
		    OR (
		      cardinality(s.department_path)>=cardinality($7::text[])
		      AND s.department_path[1:cardinality($7::text[])]=$7::text[]
		    )
		  )
		  AND (
			g.has_global_scope
			OR EXISTS (
				SELECT 1
				FROM principal p
				JOIN org_positions op ON op.tenant_id=p.tenant_id AND op.occupant_principal_id=p.id
				JOIN organization_scopes assigned ON assigned.tenant_id=op.tenant_id AND assigned.legal_entity_id=op.legal_entity_id AND assigned.id=op.organization_scope_id
				WHERE op.legal_entity_id=$3::uuid
				  AND op.valid_from<=clock_timestamp() AND (op.valid_until IS NULL OR clock_timestamp()<op.valid_until)
				  AND cardinality(s.department_path)<=cardinality(assigned.department_path)
				  AND assigned.department_path[1:cardinality(s.department_path)]=s.department_path
			)
			OR EXISTS (
				SELECT 1
				FROM principal p
				JOIN scim_users su ON su.tenant_id=p.tenant_id AND su.principal_id=p.id AND su.active AND su.deleted_at IS NULL
				JOIN scim_sources ss ON ss.tenant_id=su.tenant_id AND ss.id=su.source_id AND ss.status='ACTIVE'
				JOIN directory_group_members dgm ON dgm.tenant_id=su.tenant_id AND dgm.scim_user_id=su.id
				JOIN directory_groups dg ON dg.tenant_id=dgm.tenant_id AND dg.id=dgm.group_id AND dg.deleted_at IS NULL
				JOIN directory_group_role_bindings b ON b.tenant_id=dg.tenant_id AND b.group_id=dg.id
				JOIN role_templates rt ON rt.tenant_id=b.tenant_id AND rt.id=b.role_template_id
				JOIN organization_scopes assigned ON assigned.tenant_id=b.tenant_id AND assigned.legal_entity_id=b.legal_entity_id AND assigned.id=b.organization_scope_id
				WHERE b.legal_entity_id=$3::uuid
				  AND b.valid_from<=clock_timestamp() AND (b.valid_until IS NULL OR clock_timestamp()<b.valid_until)
				  AND rt.valid_from<=clock_timestamp() AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
				  AND cardinality(s.department_path)<=cardinality(assigned.department_path)
				  AND assigned.department_path[1:cardinality(s.department_path)]=s.department_path
			)
		  )
		ORDER BY cardinality(s.department_path),s.department_path,s.id
		LIMIT $5`, scope.TenantID, scope.PrincipalID, currentEntityID, search, limit+1, strings.TrimSpace(exactID), ancestorPath)
	if err != nil {
		return OrganizationScopeSearchPage{}, fmt.Errorf("query organization scopes: %w", err)
	}
	defer rows.Close()

	page := OrganizationScopeSearchPage{Items: make([]ScopeNode, 0, limit)}
	for rows.Next() {
		var item ScopeNode
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Kind, &item.ParentID, &item.DepartmentPath, &item.Filterable); err != nil {
			return OrganizationScopeSearchPage{}, fmt.Errorf("scan organization scope: %w", err)
		}
		if len(page.Items) == limit {
			page.HasMore = true
			continue
		}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return OrganizationScopeSearchPage{}, fmt.Errorf("iterate organization scopes: %w", err)
	}
	return page, nil
}

var _ Resolver = (*PostgresResolver)(nil)
var _ HierarchyResolver = (*PostgresResolver)(nil)
var _ OrganizationScopeSearcher = (*PostgresResolver)(nil)
var _ OrganizationScopeSelectionResolver = (*PostgresResolver)(nil)
