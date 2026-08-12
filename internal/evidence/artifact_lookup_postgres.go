//go:build postgres

package evidence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) GetArtifact(ctx context.Context, tenant, requestID, artifactID string) (Artifact, error) {
	row := r.pool.QueryRow(ctx, `SELECT ca.id::text,t.id::text,ca.request_id::text,COALESCE(ca.submission_id::text,''),ca.file_name,ca.media_type,ca.size_bytes,ca.sha256,ca.storage_key,ca.status,COALESCE(ca.created_by::text,''),ca.created_at FROM capture_artifacts ca JOIN tenants t ON t.id=ca.tenant_id WHERE ca.id=$1::uuid AND ca.request_id=$2::uuid AND (t.id::text=$3 OR t.slug=$3)`, artifactID, requestID, tenant)
	var value Artifact
	if err := row.Scan(&value.ID, &value.TenantID, &value.RequestID, &value.SubmissionID, &value.FileName, &value.MediaType, &value.SizeBytes, &value.SHA256, &value.StorageKey, &value.Status, &value.CreatedBy, &value.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Artifact{}, ErrNotFound
		}
		return Artifact{}, err
	}
	return value, nil
}
