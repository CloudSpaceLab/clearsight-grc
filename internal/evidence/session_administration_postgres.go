//go:build postgres

package evidence

import (
	"context"
	"time"
)

const activeSessionMetadataQuery = `
	SELECT cs.id::text,cs.audience_hint,cs.expires_at,cs.created_at
	FROM capture_sessions cs
	JOIN capture_requests cr ON cr.id=cs.request_id AND cr.tenant_id=cs.tenant_id
	JOIN tenants t ON t.id=cs.tenant_id
	WHERE (t.id::text=$1 OR t.slug=$1)
	  AND cs.request_id=$2::uuid
	  AND cs.revoked_at IS NULL
	  AND cs.expires_at>$3
	  AND cr.status IN ('READY','IN_PROGRESS')
	  AND cr.deadline>$3
	  AND cr.recipient_type='EXTERNAL_AUDIENCE'
	ORDER BY cs.created_at DESC,cs.id DESC
	LIMIT $4`

func (r *PostgresRepository) ListActiveSessionMetadata(ctx context.Context, tenant, requestID string, now time.Time, limit int) ([]ActiveSessionMetadata, error) {
	rows, err := r.pool.Query(ctx, activeSessionMetadataQuery, tenant, requestID, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]ActiveSessionMetadata, 0, limit)
	for rows.Next() {
		var value ActiveSessionMetadata
		if err := rows.Scan(&value.ID, &value.AudienceHint, &value.ExpiresAt, &value.CreatedAt); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

var _ activeSessionAdministrationStore = (*PostgresRepository)(nil)
