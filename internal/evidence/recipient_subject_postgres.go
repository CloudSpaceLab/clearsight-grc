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
	if subjectType != "MATTER" {
		// Capture currently has no canonical cross-domain visibility rule beyond
		// Matters. Preserve that existing contract rather than inventing one here;
		// recipient identity is still tenant-bound and active-person validated.
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
