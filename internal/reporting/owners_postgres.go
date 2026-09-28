//go:build postgres || load

package reporting

import (
	"context"
	"fmt"
)

func (r *PostgresRepository) ListReportOwners(ctx context.Context, scope ReportScope, dataset ReportDataset, limit int) ([]ReportOwnerOption, error) {
	if err := r.validateInput(ctx, scope, "owners"); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > maxReportOwnerOptions {
		limit = maxReportOwnerOptions
	}

	var query string
	switch dataset {
	case DatasetProcessingActivities, DatasetProcessingActivityExceptions:
		query = `
			SELECT DISTINCT p.id::text,p.display_name
			FROM ropa_processing_activities a
			JOIN principals p ON p.id=a.owner_principal_id AND p.tenant_id=a.tenant_id
			WHERE a.tenant_id=$1::uuid AND a.legal_entity_id=$2::uuid
			  AND a.owner_principal_id IS NOT NULL
			ORDER BY p.display_name,p.id
			LIMIT $3`
	case DatasetPrograms:
		query = `
			SELECT DISTINCT p.id::text,p.display_name
			FROM programs a
			JOIN principals p ON p.id=a.owner_principal_id AND p.tenant_id=a.tenant_id
			WHERE a.tenant_id=$1::uuid AND a.legal_entity_id=$2::uuid
			  AND a.owner_principal_id IS NOT NULL
			ORDER BY p.display_name,p.id
			LIMIT $3`
	case DatasetMatters, DatasetMatterExceptions:
		query = `
			SELECT DISTINCT p.id::text,p.display_name
			FROM matters a
			JOIN principals p ON p.id=a.owner_principal_id AND p.tenant_id=a.tenant_id
			WHERE a.tenant_id=$1::uuid AND a.legal_entity_id=$2::uuid
			  AND a.owner_principal_id IS NOT NULL
			ORDER BY p.display_name,p.id
			LIMIT $3`
	case DatasetVendors:
		query = `
			SELECT DISTINCT p.id::text,p.display_name
			FROM third_party_relationships a
			JOIN principals p ON p.id=a.business_owner_principal_id AND p.tenant_id=a.tenant_id
			WHERE a.tenant_id=$1::uuid AND a.legal_entity_id=$2::uuid
			  AND a.business_owner_principal_id IS NOT NULL
			  AND NOT EXISTS (
			    SELECT 1 FROM demo_record_archives archive
			    WHERE archive.tenant_id=a.tenant_id
			      AND archive.legal_entity_id=a.legal_entity_id
			      AND archive.record_type='VENDOR_RELATIONSHIP'
			      AND archive.record_id=a.id
			      AND archive.restored_at IS NULL
			  )
			ORDER BY p.display_name,p.id
			LIMIT $3`
	default:
		return nil, ErrInvalid
	}

	rows, err := r.pool.Query(ctx, query, scope.TenantID, scope.LegalEntityID, limit)
	if err != nil {
		return nil, fmt.Errorf("list report owners: %w", err)
	}
	defer rows.Close()

	values := make([]ReportOwnerOption, 0, limit)
	for rows.Next() {
		var value ReportOwnerOption
		if err := rows.Scan(&value.PrincipalID, &value.DisplayName); err != nil {
			return nil, fmt.Errorf("scan report owner: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list report owners: %w", err)
	}
	return values, nil
}
