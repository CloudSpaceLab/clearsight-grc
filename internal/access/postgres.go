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

func (r *PostgresResolver) ResolveOIDC(ctx context.Context, tenantID, issuer, subject string) (Resolution, error) {
	tenantID = strings.TrimSpace(tenantID)
	issuer = strings.TrimSpace(issuer)
	subject = strings.TrimSpace(subject)
	if tenantID == "" || issuer == "" || subject == "" {
		return Resolution{}, ErrIdentityNotProvisioned
	}

	var value Resolution
	err := r.pool.QueryRow(ctx, `
		SELECT t.slug,p.id::text,le.code,p.display_name,p.kind
		FROM principal_identities pi
		JOIN tenants t ON t.id=pi.tenant_id
		JOIN principals p ON p.id=pi.principal_id AND p.tenant_id=pi.tenant_id
		JOIN legal_entities le ON le.id=pi.legal_entity_id AND le.tenant_id=pi.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND pi.issuer=$2
		  AND pi.subject=$3
		  AND pi.status='ACTIVE'
		  AND p.status='ACTIVE'
		  AND p.valid_from<=clock_timestamp()
		  AND (p.valid_until IS NULL OR clock_timestamp()<p.valid_until)
		  AND le.valid_from<=clock_timestamp()
		  AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)`, tenantID, issuer, subject).
		Scan(&value.TenantID, &value.PrincipalID, &value.LegalEntityID, &value.DisplayName, &value.Kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return Resolution{}, ErrIdentityNotProvisioned
	}
	if err != nil {
		return Resolution{}, fmt.Errorf("resolve OIDC principal: %w", err)
	}
	return r.withRoles(ctx, value)
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
		  AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)`, tenantID, principalID, legalEntityID).
		Scan(&value.TenantID, &value.PrincipalID, &value.LegalEntityID, &value.DisplayName, &value.Kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return Resolution{}, ErrPrincipalUnavailable
	}
	if err != nil {
		return Resolution{}, fmt.Errorf("resolve principal: %w", err)
	}
	return r.withRoles(ctx, value)
}

func (r *PostgresResolver) withRoles(ctx context.Context, value Resolution) (Resolution, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT rt.code,rt.capabilities,op.department_path
		FROM tenants t
		JOIN principals p ON p.tenant_id=t.id
		JOIN org_positions op ON op.tenant_id=t.id AND op.occupant_principal_id=p.id
		JOIN position_role_bindings prb ON prb.tenant_id=t.id AND prb.position_id=op.id
		JOIN role_templates rt ON rt.tenant_id=t.id AND rt.id=prb.role_template_id
		WHERE t.slug=$1
		  AND p.id::text=$2
		  AND (op.legal_entity_id IS NULL OR op.legal_entity_id=(
		      SELECT le.id FROM legal_entities le
		      WHERE le.tenant_id=t.id AND (le.id::text=$3 OR le.code=$3)
		  ))
		  AND op.valid_from<=clock_timestamp()
		  AND (op.valid_until IS NULL OR clock_timestamp()<op.valid_until)
		  AND prb.valid_from<=clock_timestamp()
		  AND (prb.valid_until IS NULL OR clock_timestamp()<prb.valid_until)
		  AND rt.valid_from<=clock_timestamp()
		  AND (rt.valid_until IS NULL OR clock_timestamp()<rt.valid_until)
		ORDER BY cardinality(op.department_path),op.department_path,rt.code`, value.TenantID, value.PrincipalID, value.LegalEntityID)
	if err != nil {
		return Resolution{}, fmt.Errorf("resolve principal roles: %w", err)
	}
	defer rows.Close()

	allRoles := make([]string, 0, 8)
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
		allRoles = append(allRoles, roleCodes...)
		if len(path) == 0 {
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

	value.RoleCodes = identity.NormalizeRoleCodes(allRoles)
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
