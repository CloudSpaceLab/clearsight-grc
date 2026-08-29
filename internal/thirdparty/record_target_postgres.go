//go:build postgres

package thirdparty

import (
	"context"
	"fmt"
)

func (r *PostgresRepository) CurrentRelationshipDocuments(ctx context.Context, scope Scope, relationshipID, documentType string) ([]AssessmentDocument, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT d.id::text,t.slug,d.legal_entity_id::text,d.relationship_id::text,d.assessment_id::text,d.request_id::text,d.artifact_id::text,
		       COALESCE(d.supersedes_document_id::text,''),d.document_type,d.reference,d.issued_by,d.issued_on,d.expires_on,d.evidence_class,d.status,
		       COALESCE(d.validated_by_principal_id::text,''),COALESCE(d.validated_at,d.updated_at),d.version,d.created_at,d.updated_at
		FROM third_party_documents d JOIN tenants t ON t.id=d.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND d.legal_entity_id::text=$2 AND d.relationship_id::text=$3
		  AND d.document_type=$4 AND d.status IN ('VALIDATED','EXPIRED')
		ORDER BY d.id LIMIT 2`, scope.TenantID, scope.LegalEntityID, relationshipID, documentType)
	if err != nil {
		return nil, fmt.Errorf("list current relationship documents: %w", err)
	}
	defer rows.Close()
	items := []AssessmentDocument{}
	for rows.Next() {
		var value AssessmentDocument
		if err := rows.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.RelationshipID, &value.AssessmentID, &value.RequestID, &value.ArtifactID,
			&value.SupersedesDocumentID, &value.DocumentType, &value.Reference, &value.IssuedBy, &value.IssuedOn, &value.ExpiresOn, &value.EvidenceClass, &value.Status,
			&value.ValidatedByPrincipalID, &value.ValidatedAt, &value.Version, &value.CreatedAt, &value.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
