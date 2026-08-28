//go:build postgres

package evidence

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func loadPostgresDistributionArtifact(ctx context.Context, store *PostgresDistributionStore, session DistributionAccessSession, tenantID, artifactID string) (Artifact, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return Artifact{}, ErrNotFound
	}
	row := store.repo.pool.QueryRow(ctx, `
		SELECT a.id::text,a.tenant_id::text,a.request_id::text,COALESCE(a.submission_id::text,''),
		       a.file_name,a.media_type,a.size_bytes,a.sha256,a.storage_key,a.status,COALESCE(a.created_by::text,''),a.created_at
		FROM capture_artifacts a
		JOIN capture_distribution_recipients r
		  ON r.request_id=a.request_id AND r.tenant_id=a.tenant_id
		WHERE a.id=$1::uuid AND a.tenant_id=$2::uuid
		  AND r.distribution_id=$3::uuid AND r.legal_entity_id=$4::uuid
		  AND r.role='TO' AND r.state<>'REVOKED'`,
		artifactID, tenantID, session.DistributionID, session.LegalEntityID)
	var value Artifact
	if err := row.Scan(
		&value.ID, &value.TenantID, &value.RequestID, &value.SubmissionID,
		&value.FileName, &value.MediaType, &value.SizeBytes, &value.SHA256,
		&value.StorageKey, &value.Status, &value.CreatedBy, &value.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Artifact{}, ErrNotFound
		}
		return Artifact{}, err
	}
	return value, nil
}
