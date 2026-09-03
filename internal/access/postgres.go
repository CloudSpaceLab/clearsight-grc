package access

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresResolver struct{ pool *pgxpool.Pool }

func NewPostgresResolver(pool *pgxpool.Pool) *PostgresResolver {
	return &PostgresResolver{pool: pool}
}

func (r *PostgresResolver) CanReassign(ctx context.Context, request ReassignmentRequest) (ReassignmentDecision, error) {
	request.TenantID = strings.TrimSpace(request.TenantID)
	request.LegalEntityID = strings.TrimSpace(request.LegalEntityID)
	request.ActorPrincipalID = strings.TrimSpace(request.ActorPrincipalID)
	request.CurrentOwnerPrincipalID = strings.TrimSpace(request.CurrentOwnerPrincipalID)
	if request.TenantID == "" || request.LegalEntityID == "" || request.ActorPrincipalID == "" || request.CurrentOwnerPrincipalID == "" {
		return ReassignmentDecision{}, ErrPrincipalUnavailable
	}
	if request.ActorPrincipalID == request.CurrentOwnerPrincipalID {
		if _, err := r.ResolvePrincipal(ctx, request.TenantID, request.ActorPrincipalID, request.LegalEntityID); err != nil {
			if errors.Is(err, ErrPrincipalUnavailable) {
				return ReassignmentDecision{}, nil
			}
			return ReassignmentDecision{}, err
		}
		return ReassignmentDecision{Allowed: true, Basis: "CURRENT_ASSIGNEE"}, nil
	}

	var allowed bool
	var hierarchyVersion int64
	err := r.pool.QueryRow(ctx, `
		WITH RECURSIVE selected_scope AS (
			SELECT t.id AS tenant_id,le.id AS legal_entity_id
			FROM tenants t
			JOIN legal_entities le ON le.tenant_id=t.id
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND (le.id::text=$2 OR le.code=$2)
			  AND le.valid_from<=clock_timestamp()
			  AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
			LIMIT 1
		), owner_positions AS (
			SELECT op.id,op.parent_position_id,op.version,ARRAY[op.id] AS visited,0 AS depth
			FROM org_positions op
			JOIN selected_scope scope ON scope.tenant_id=op.tenant_id AND scope.legal_entity_id=op.legal_entity_id
			JOIN principals owner ON owner.tenant_id=op.tenant_id AND owner.id=op.occupant_principal_id
			WHERE owner.id::text=$4 AND owner.status='ACTIVE'
			  AND owner.valid_from<=clock_timestamp() AND (owner.valid_until IS NULL OR clock_timestamp()<owner.valid_until)
			  AND op.valid_from<=clock_timestamp() AND (op.valid_until IS NULL OR clock_timestamp()<op.valid_until)
		), ancestors AS (
			SELECT * FROM owner_positions
			UNION ALL
			SELECT parent.id,parent.parent_position_id,GREATEST(parent.version,child.version),child.visited||parent.id,child.depth+1
			FROM ancestors child
			JOIN org_positions parent ON parent.id=child.parent_position_id
			JOIN selected_scope scope ON scope.tenant_id=parent.tenant_id AND scope.legal_entity_id=parent.legal_entity_id
			WHERE child.depth<12
			  AND NOT parent.id=ANY(child.visited)
			  AND parent.valid_from<=clock_timestamp() AND (parent.valid_until IS NULL OR clock_timestamp()<parent.valid_until)
		), active_actor AS (
			SELECT p.id
			FROM principals p
			JOIN selected_scope scope ON scope.tenant_id=p.tenant_id
			WHERE p.id::text=$3 AND p.status='ACTIVE'
			  AND p.valid_from<=clock_timestamp() AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)
		)
		SELECT COALESCE(bool_or(a.depth>0 AND p.occupant_principal_id=actor.id),false),COALESCE(max(a.version),0)
		FROM ancestors a
		JOIN org_positions p ON p.id=a.id
		CROSS JOIN active_actor actor
		-- Every owner-position chain must reach an effective in-scope root.
		-- A cycle, invalid parent or depth cutoff is not a completed chain,
		-- even when a matching manager was visited before the failure.
		WHERE NOT EXISTS (
			SELECT 1 FROM owner_positions owner_position
			WHERE NOT EXISTS (
				SELECT 1 FROM ancestors root
				WHERE root.visited[1]=owner_position.id AND root.parent_position_id IS NULL
			)
		)`, request.TenantID, request.LegalEntityID, request.ActorPrincipalID, request.CurrentOwnerPrincipalID).
		Scan(&allowed, &hierarchyVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return ReassignmentDecision{Allowed: false}, nil
	}
	if err != nil {
		return ReassignmentDecision{}, fmt.Errorf("resolve reporting-line reassignment: %w", err)
	}
	if !allowed {
		return ReassignmentDecision{}, nil
	}
	return ReassignmentDecision{Allowed: true, Basis: "REPORTING_ANCESTOR", HierarchyVersion: hierarchyVersion}, nil
}

func (r *PostgresResolver) ResolveOIDC(ctx context.Context, tenantID, legalEntityID, issuer, subject string) (Resolution, error) {
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	issuer = strings.TrimSpace(issuer)
	subject = strings.TrimSpace(subject)
	if tenantID == "" || legalEntityID == "" || issuer == "" || subject == "" {
		return Resolution{}, ErrIdentityNotProvisioned
	}

	var principalID string
	err := r.pool.QueryRow(ctx, `
		SELECT p.id::text
		FROM principal_identities pi
		JOIN tenants t ON t.id=pi.tenant_id
		JOIN principals p ON p.id=pi.principal_id AND p.tenant_id=pi.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND pi.issuer=$2
		  AND pi.subject=$3
		  AND pi.status='ACTIVE'
		  AND p.status='ACTIVE'
		  AND p.valid_from<=clock_timestamp()
		  AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)`, tenantID, issuer, subject).Scan(&principalID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Resolution{}, ErrIdentityNotProvisioned
	}
	if err != nil {
		return Resolution{}, fmt.Errorf("resolve OIDC principal: %w", err)
	}
	return r.ResolvePrincipal(ctx, tenantID, principalID, legalEntityID)
}

func (r *PostgresResolver) ResolvePrincipal(ctx context.Context, tenantID, principalID, legalEntityID string) (Resolution, error) {
	tenantID = strings.TrimSpace(tenantID)
	principalID = strings.TrimSpace(principalID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	if tenantID == "" || principalID == "" || legalEntityID == "" {
		return Resolution{}, ErrPrincipalUnavailable
	}

	var value Resolution
	err := r.pool.QueryRow(ctx, `
		SELECT t.slug,p.id::text,le.code,p.display_name,p.kind
		FROM tenants t
		JOIN principals p ON p.tenant_id=t.id
		JOIN legal_entities le ON le.tenant_id=t.id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND p.id::text=$2
		  AND (le.id::text=$3 OR le.code=$3)
		  AND p.status='ACTIVE'
		  AND p.valid_from<=clock_timestamp()
		  AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)
		  AND le.valid_from<=clock_timestamp()
		  AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
		  AND (
		      EXISTS (
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
		          JOIN scim_sources ss ON ss.tenant_id=su.tenant_id AND ss.id=su.source_id AND ss.status='ACTIVE'
		          JOIN directory_group_members dgm ON dgm.tenant_id=su.tenant_id AND dgm.scim_user_id=su.id
		          JOIN directory_groups dg ON dg.tenant_id=dgm.tenant_id AND dg.id=dgm.group_id AND dg.source_id=su.source_id
		          JOIN directory_group_role_bindings dgrb ON dgrb.tenant_id=dg.tenant_id AND dgrb.group_id=dg.id
		          JOIN role_templates rt ON rt.tenant_id=dgrb.tenant_id AND rt.id=dgrb.role_template_id
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
		  )`, tenantID, principalID, legalEntityID).
		Scan(&value.TenantID, &value.PrincipalID, &value.LegalEntityID, &value.DisplayName, &value.Kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return Resolution{}, ErrPrincipalUnavailable
	}
	if err != nil {
		return Resolution{}, fmt.Errorf("resolve principal: %w", err)
	}
	return r.withRoles(ctx, value)
}

func (r *PostgresResolver) ResolvePrincipals(ctx context.Context, tenantID, legalEntityID string, principalIDs []string) ([]PrincipalResolveOutcome, error) {
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	if tenantID == "" || legalEntityID == "" {
		return nil, ErrPrincipalUnavailable
	}
	if len(principalIDs) > MaxPrincipalBatchSize {
		return nil, ErrPrincipalBatchTooLarge
	}
	outcomes := make([]PrincipalResolveOutcome, len(principalIDs))
	if len(principalIDs) == 0 {
		return outcomes, nil
	}
	normalized := make([]string, len(principalIDs))
	for index, principalID := range principalIDs {
		normalized[index] = strings.TrimSpace(principalID)
		if normalized[index] == "" {
			outcomes[index].Err = ErrPrincipalUnavailable
		}
	}

	rows, err := r.pool.Query(ctx, `
		SELECT p.id::text,t.slug,le.code,p.display_name,p.kind
		FROM tenants t
		JOIN legal_entities le ON le.tenant_id=t.id
		JOIN principals p ON p.tenant_id=t.id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND (le.id::text=$2 OR le.code=$2)
		  AND p.id=ANY($3::uuid[])
		  AND p.status='ACTIVE'
		  AND p.valid_from<=clock_timestamp()
		  AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)
		  AND le.valid_from<=clock_timestamp()
		  AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)
		  AND (
		      EXISTS (
		          SELECT 1 FROM org_positions op
		          WHERE op.tenant_id=t.id AND op.occupant_principal_id=p.id
		            AND (op.legal_entity_id IS NULL OR op.legal_entity_id=le.id)
		            AND op.valid_from<=clock_timestamp()
		            AND (op.valid_until IS NULL OR clock_timestamp()<op.valid_until)
		      )
		      OR EXISTS (
		          SELECT 1
		          FROM scim_users su
		          JOIN scim_sources ss ON ss.tenant_id=su.tenant_id AND ss.id=su.source_id AND ss.status='ACTIVE'
		          JOIN directory_group_members dgm ON dgm.tenant_id=su.tenant_id AND dgm.scim_user_id=su.id
		          JOIN directory_groups dg ON dg.tenant_id=dgm.tenant_id AND dg.id=dgm.group_id AND dg.source_id=su.source_id
		          JOIN directory_group_role_bindings dgrb ON dgrb.tenant_id=dg.tenant_id AND dgrb.group_id=dg.id
		          JOIN role_templates rt ON rt.tenant_id=dgrb.tenant_id AND rt.id=dgrb.role_template_id
		          WHERE su.tenant_id=t.id AND su.principal_id=p.id AND su.active AND su.deleted_at IS NULL
		            AND dg.deleted_at IS NULL AND dgrb.legal_entity_id=le.id
		            AND dgrb.valid_from<=clock_timestamp()
		            AND (dgrb.valid_until IS NULL OR clock_timestamp()<dgrb.valid_until)
		            AND rt.valid_from<=clock_timestamp()
		            AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
		      )
		  )`, tenantID, legalEntityID, normalized)
	if err != nil {
		return nil, fmt.Errorf("resolve principals: %w", err)
	}
	defer rows.Close()
	resolved := make(map[string]Resolution, len(principalIDs))
	for rows.Next() {
		var value Resolution
		if err := rows.Scan(&value.PrincipalID, &value.TenantID, &value.LegalEntityID, &value.DisplayName, &value.Kind); err != nil {
			return nil, fmt.Errorf("scan resolved principal: %w", err)
		}
		resolved[value.PrincipalID] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("resolve principals rows: %w", err)
	}
	for index, principalID := range normalized {
		if outcomes[index].Err != nil {
			continue
		}
		value, ok := resolved[principalID]
		if !ok {
			outcomes[index].Err = ErrPrincipalUnavailable
			continue
		}
		outcomes[index].Resolution = value
	}
	return outcomes, nil
}

func (r *PostgresResolver) withRoles(ctx context.Context, value Resolution) (Resolution, error) {
	rows, err := r.pool.Query(ctx, `
		WITH current_entity AS (
			SELECT le.id
			FROM tenants t
			JOIN legal_entities le ON le.tenant_id=t.id
			WHERE t.slug=$1 AND (le.id::text=$3 OR le.code=$3)
			LIMIT 1
		), effective_roles AS (
			SELECT rt.code,rt.capabilities,op.department_path
			FROM tenants t
			JOIN principals p ON p.tenant_id=t.id
			JOIN org_positions op ON op.tenant_id=t.id AND op.occupant_principal_id=p.id
			JOIN position_role_bindings prb ON prb.tenant_id=t.id AND prb.position_id=op.id
			JOIN role_templates rt ON rt.tenant_id=t.id AND rt.id=prb.role_template_id
			WHERE t.slug=$1
			  AND p.id::text=$2
			  AND (op.legal_entity_id IS NULL OR op.legal_entity_id=(SELECT id FROM current_entity))
			  AND op.valid_from<=clock_timestamp()
			  AND (op.valid_until IS NULL OR clock_timestamp()<op.valid_until)
			  AND prb.valid_from<=clock_timestamp()
			  AND (prb.valid_until IS NULL OR clock_timestamp()<prb.valid_until)
			  AND rt.valid_from<=clock_timestamp()
			  AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)

			UNION ALL

			SELECT rt.code,rt.capabilities,dgrb.department_path
			FROM tenants t
			JOIN scim_users su ON su.tenant_id=t.id AND su.principal_id::text=$2 AND su.active AND su.deleted_at IS NULL
			JOIN scim_sources ss ON ss.tenant_id=su.tenant_id AND ss.id=su.source_id AND ss.status='ACTIVE'
			JOIN directory_group_members dgm ON dgm.tenant_id=su.tenant_id AND dgm.scim_user_id=su.id
			JOIN directory_groups dg ON dg.tenant_id=dgm.tenant_id AND dg.id=dgm.group_id AND dg.source_id=su.source_id AND dg.deleted_at IS NULL
			JOIN directory_group_role_bindings dgrb ON dgrb.tenant_id=dg.tenant_id AND dgrb.group_id=dg.id AND dgrb.legal_entity_id=(SELECT id FROM current_entity)
			JOIN role_templates rt ON rt.tenant_id=t.id AND rt.id=dgrb.role_template_id
			WHERE t.slug=$1
			  AND dgrb.valid_from<=clock_timestamp()
			  AND (dgrb.valid_until IS NULL OR clock_timestamp()<dgrb.valid_until)
			  AND rt.valid_from<=clock_timestamp()
			  AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
		)
		SELECT code,capabilities,department_path
		FROM effective_roles
		ORDER BY cardinality(department_path),department_path,code`, value.TenantID, value.PrincipalID, value.LegalEntityID)
	if err != nil {
		return Resolution{}, fmt.Errorf("resolve principal roles: %w", err)
	}
	defer rows.Close()

	globalRoles := make([]string, 0, 8)
	globalPermissions := make([]string, 0, 8)
	type grantAccumulator struct {
		path        []string
		roles       []string
		permissions []string
	}
	grants := map[string]*grantAccumulator{}
	grantOrder := make([]string, 0, 4)

	for rows.Next() {
		var role string
		var capabilities []string
		var departmentPath []string
		if err := rows.Scan(&role, &capabilities, &departmentPath); err != nil {
			return Resolution{}, err
		}
		path, err := identity.NormalizeDepartmentPath(departmentPath)
		if err != nil {
			return Resolution{}, fmt.Errorf("invalid department path for role %s: %w", role, err)
		}
		roleCodes := identity.NormalizeRoleCodes([]string{role})
		permissions := identity.NormalizePermissionCodes(capabilities)
		if len(path) == 0 {
			globalRoles = append(globalRoles, roleCodes...)
			globalPermissions = append(globalPermissions, permissions...)
			continue
		}
		key := strings.Join(path, "\x1f")
		grant := grants[key]
		if grant == nil {
			grant = &grantAccumulator{path: path}
			grants[key] = grant
			grantOrder = append(grantOrder, key)
		}
		grant.roles = append(grant.roles, roleCodes...)
		grant.permissions = append(grant.permissions, permissions...)
	}
	if err := rows.Err(); err != nil {
		return Resolution{}, err
	}

	value.RoleCodes = identity.NormalizeRoleCodes(globalRoles)
	value.PermissionCodes = identity.NormalizePermissionCodes(globalPermissions)
	sort.Strings(value.RoleCodes)
	sort.Strings(value.PermissionCodes)
	sort.Strings(grantOrder)
	value.DepartmentGrants = make([]identity.DepartmentGrant, 0, len(grantOrder))
	for _, key := range grantOrder {
		grant := grants[key]
		roles := identity.NormalizeRoleCodes(grant.roles)
		permissions := identity.NormalizePermissionCodes(grant.permissions)
		sort.Strings(roles)
		sort.Strings(permissions)
		value.DepartmentGrants = append(value.DepartmentGrants, identity.DepartmentGrant{
			Path: grant.path, RoleCodes: roles, PermissionCodes: permissions,
		})
	}
	return value, nil
}
