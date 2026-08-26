//go:build postgres

package evidence

import (
	"context"
	"strings"
)

func (r *PostgresRepository) CanReadSubject(ctx context.Context, tenant, principalID, subjectType, subjectID string) (bool, error) {
	tenant = strings.TrimSpace(tenant)
	principalID = strings.TrimSpace(principalID)
	subjectType = strings.ToUpper(strings.TrimSpace(subjectType))
	subjectID = strings.TrimSpace(subjectID)
	if tenant == "" || principalID == "" || subjectType == "" || subjectID == "" {
		return false, nil
	}

	eligible, err := r.InternalRecipientEligible(ctx, tenant, principalID)
	if err != nil || !eligible {
		return false, err
	}
	if subjectType == "PROGRAM" {
		var visible bool
		err = r.pool.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM programs p JOIN tenants t ON t.id=p.tenant_id
			WHERE (t.id::text=$1 OR t.slug=$1) AND p.id::text=$2
		)`, tenant, subjectID).Scan(&visible)
		return visible, err
	}
	if subjectType == "VENDOR_RELATIONSHIP" {
		var visible bool
		err = r.pool.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM third_party_relationships rel JOIN tenants t ON t.id=rel.tenant_id
			WHERE (t.id::text=$1 OR t.slug=$1) AND rel.id::text=$2
		)`, tenant, subjectID).Scan(&visible)
		return visible, err
	}
	if subjectType != "MATTER" {
		return true, nil
	}

	var visible bool
	err = r.pool.QueryRow(ctx, `
		SELECT COALESCE((
			SELECT CASE
				WHEN NOT (m.scope ? 'access') THEN true
				WHEN upper(btrim(COALESCE(m.scope->>'access',''))) IN ('PUBLIC','INTERNAL') THEN true
				WHEN upper(btrim(COALESCE(m.scope->>'access','')))='RESTRICTED' THEN
					CASE
						WHEN jsonb_typeof(m.scope->'allowed_principal_ids')='array'
						 AND NOT EXISTS (
							SELECT 1
							FROM jsonb_array_elements(m.scope->'allowed_principal_ids') AS entry(value)
							WHERE jsonb_typeof(entry.value)<>'string'
						 )
						 AND EXISTS (
							SELECT 1
							FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') AS nonblank(value)
							WHERE btrim(nonblank.value)<>''
						 )
						THEN EXISTS (
							SELECT 1
							FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') AS allowed(value)
							WHERE btrim(allowed.value)=$2
						)
						ELSE false
					END
				ELSE false
			END
			FROM matters m
			JOIN tenants t ON t.id=m.tenant_id
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND m.id::text=$3
			LIMIT 1
		), false)`, tenant, principalID, subjectID).Scan(&visible)
	return visible, err
}

var _ SubjectAccessChecker = (*PostgresRepository)(nil)
