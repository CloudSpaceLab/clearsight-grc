//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"time"
)

// These helpers operate exclusively on the bank command's existing transaction.
func ReadCollectionRequestPostgres(ctx context.Context, tx pgx.Tx, tenant, entity, requestID string) (Request, error) {
	return scanRequest(tx.QueryRow(ctx, requestSelect+` WHERE er.tenant_id=$1::uuid AND er.legal_entity_id=$2::uuid AND er.id=$3::uuid FOR UPDATE OF er`, tenant, entity, requestID))
}

func ReadCollectionSourcePostgres(ctx context.Context, tx pgx.Tx, q DocumentQuery) (DocumentOccurrence, error) {
	q.Limit = 1
	if _, err := normalizeDocumentQuery(&q); err != nil {
		return DocumentOccurrence{}, err
	}
	if q.SubmissionID == "" || q.FieldID == "" || q.ArtifactID == "" || q.RelationshipID == "" {
		return DocumentOccurrence{}, ErrNotFound
	}
	// Hold the immutable membership and artifact inspection status while the
	// collection receipt is committed; no second connection or broad scan.
	var locked string
	if err := tx.QueryRow(ctx, `SELECT ar.id::text FROM capture_artifacts ar JOIN capture_submissions s ON s.tenant_id=ar.tenant_id AND s.id=$3::uuid JOIN capture_requests req ON req.tenant_id=s.tenant_id AND req.id=s.request_id WHERE ar.tenant_id=$1::uuid AND ar.id=$2::uuid AND req.legal_entity_id=$4::uuid FOR SHARE OF ar,s,req`, q.TenantID, q.ArtifactID, q.SubmissionID, q.LegalEntityID).Scan(&locked); err != nil {
		return DocumentOccurrence{}, ErrNotFound
	}
	rows, err := tx.Query(ctx, documentInventorySQL(), q.TenantID, q.LegalEntityID, q.PrincipalID, time.Now().UTC(), "", q.RelationshipID, q.ResponseRevisionID, q.CurrentOnly, "", "", time.Time{}, "", q.SubmissionID, q.FieldID, q.ArtifactID, 2, false)
	if err != nil {
		return DocumentOccurrence{}, err
	}
	defer rows.Close()
	var result DocumentOccurrence
	count := 0
	for rows.Next() {
		count++
		var raw []byte
		var artifactRequestID string
		if err := rows.Scan(&raw, &artifactRequestID); err != nil {
			return result, err
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return result, err
		}
		result.ArtifactRequestID = artifactRequestID
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	if count != 1 {
		return DocumentOccurrence{}, ErrNotFound
	}
	return result, nil
}
