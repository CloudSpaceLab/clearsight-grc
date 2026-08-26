//go:build postgres

package evidence

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) ResolveSubjectScope(ctx context.Context, tenant, subjectType, subjectID string) (SubjectScope, error) {
	tenant = strings.TrimSpace(tenant)
	subjectType = strings.ToUpper(strings.TrimSpace(subjectType))
	subjectID = strings.TrimSpace(subjectID)
	if tenant == "" || subjectID == "" {
		return SubjectScope{}, ErrSubjectUnsupported
	}
	var legalEntityID string
	var err error
	switch subjectType {
	case "PROGRAM":
		err = r.pool.QueryRow(ctx, `SELECT p.legal_entity_id::text FROM programs p JOIN tenants t ON t.id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND p.id::text=$2`, tenant, subjectID).Scan(&legalEntityID)
	case "MATTER":
		err = r.pool.QueryRow(ctx, `SELECT m.legal_entity_id::text FROM matters m JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND m.id::text=$2`, tenant, subjectID).Scan(&legalEntityID)
	default:
		return SubjectScope{}, ErrSubjectUnsupported
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return SubjectScope{}, ErrSubjectUnsupported
	}
	if err != nil {
		return SubjectScope{}, err
	}
	if strings.TrimSpace(legalEntityID) == "" {
		return SubjectScope{}, ErrSubjectScopeMismatch
	}
	return SubjectScope{TenantID: tenant, LegalEntityID: legalEntityID, SubjectType: subjectType, SubjectID: subjectID}, nil
}

func (r *PostgresRepository) CanReadSubject(ctx context.Context, tenant, principalID, subjectType, subjectID string) (bool, error) {
	tenant = strings.TrimSpace(tenant)
	principalID = strings.TrimSpace(principalID)
	subjectType = strings.ToUpper(strings.TrimSpace(subjectType))
	subjectID = strings.TrimSpace(subjectID)
	if tenant == "" || principalID == "" || subjectID == "" {
		return false, nil
	}
	if _, err := r.ResolveSubjectScope(ctx, tenant, subjectType, subjectID); err != nil {
		if errors.Is(err, ErrSubjectUnsupported) || errors.Is(err, ErrSubjectScopeMismatch) {
			return false, nil
		}
		return false, err
	}
	eligible, err := r.InternalRecipientEligible(ctx, tenant, principalID)
	if err != nil || !eligible {
		return false, err
	}

	var visible bool
	var query string
	switch subjectType {
	case "PROGRAM":
		query = subjectVisibilitySQL("programs", "p")
	case "MATTER":
		query = subjectVisibilitySQL("matters", "m")
	default:
		return false, nil
	}
	err = r.pool.QueryRow(ctx, query, tenant, principalID, subjectID).Scan(&visible)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return visible, err
}

func subjectVisibilitySQL(table, alias string) string {
	// table and alias are selected only from the closed switch above.
	return `SELECT CASE
		WHEN NOT (` + alias + `.scope ? 'access') THEN true
		WHEN upper(btrim(COALESCE(` + alias + `.scope->>'access',''))) IN ('PUBLIC','INTERNAL') THEN true
		WHEN upper(btrim(COALESCE(` + alias + `.scope->>'access','')))='RESTRICTED' THEN
			jsonb_typeof(` + alias + `.scope->'allowed_principal_ids')='array'
			AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements(` + alias + `.scope->'allowed_principal_ids') e(value) WHERE jsonb_typeof(e.value)<>'string')
			AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(` + alias + `.scope->'allowed_principal_ids') a(value) WHERE btrim(a.value)=$2)
		ELSE false END
		FROM ` + table + ` ` + alias + ` JOIN tenants t ON t.id=` + alias + `.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND ` + alias + `.id::text=$3`
}

var _ SubjectScopeResolver = (*PostgresRepository)(nil)
var _ SubjectAccessChecker = (*PostgresRepository)(nil)
