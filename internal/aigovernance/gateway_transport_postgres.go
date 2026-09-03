//go:build postgres

package aigovernance

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

const gatewayTransportColumns = `gc.id::text,t.slug,gc.environment,gc.definition,gc.status,gc.maker_id::text,COALESCE(gc.checker_id::text,''),gc.change_reason,gc.checksum,gc.submitted_at,gc.approved_at,gc.activated_at,gc.suspended_at,gc.retired_at,gc.created_at,gc.updated_at,gc.version,gc.record_version`

func scanGatewayTransport(row rowScanner) (GatewayTransportRevision, error) {
	var value GatewayTransportRevision
	var definition []byte
	err := row.Scan(
		&value.ID, &value.TenantID, &value.Environment, &definition, &value.Status,
		&value.MakerID, &value.CheckerID, &value.ChangeReason, &value.Checksum,
		&value.SubmittedAt, &value.ApprovedAt, &value.ActivatedAt, &value.SuspendedAt, &value.RetiredAt,
		&value.CreatedAt, &value.UpdatedAt, &value.Version, &value.RecordVersion,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return GatewayTransportRevision{}, ErrNotFound
	}
	if err != nil {
		return GatewayTransportRevision{}, err
	}
	if err := json.Unmarshal(definition, &value.Definition); err != nil {
		return GatewayTransportRevision{}, err
	}
	return value, nil
}

func (r *PostgresRepository) CreateGatewayTransport(ctx context.Context, value GatewayTransportRevision) (GatewayTransportRevision, error) {
	definition, _ := json.Marshal(value.Definition)
	return scanGatewayTransport(r.pool.QueryRow(ctx, `INSERT INTO ai_gateway_config_revisions(id,tenant_id,environment,definition,status,maker_id,change_reason,checksum,created_at,updated_at,version,record_version)
VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4::jsonb,$5,$6::uuid,$7,$8,$9,$10,$11,$12)
RETURNING id::text,(SELECT slug FROM tenants WHERE id=ai_gateway_config_revisions.tenant_id),environment,definition,status,maker_id::text,COALESCE(checker_id::text,''),change_reason,checksum,submitted_at,approved_at,activated_at,suspended_at,retired_at,created_at,updated_at,version,record_version`,
		value.ID, value.TenantID, value.Environment, string(definition), value.Status, value.MakerID, value.ChangeReason, value.Checksum, value.CreatedAt, value.UpdatedAt, value.Version, value.RecordVersion))
}

func (r *PostgresRepository) NextGatewayTransportVersion(ctx context.Context, tenantID, environment string) (int64, error) {
	var version int64
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(gc.version),0)+1 FROM ai_gateway_config_revisions gc JOIN tenants t ON t.id=gc.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND gc.environment=$2`, tenantID, environment).Scan(&version)
	return version, err
}

func (r *PostgresRepository) GatewayTransport(ctx context.Context, tenantID, id string) (GatewayTransportRevision, error) {
	return scanGatewayTransport(r.pool.QueryRow(ctx, `SELECT `+gatewayTransportColumns+` FROM ai_gateway_config_revisions gc JOIN tenants t ON t.id=gc.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND gc.id::text=$2`, tenantID, id))
}

func (r *PostgresRepository) ListGatewayTransports(ctx context.Context, tenantID, environment string, limit int) ([]GatewayTransportRevision, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+gatewayTransportColumns+` FROM ai_gateway_config_revisions gc JOIN tenants t ON t.id=gc.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND ($2='' OR gc.environment=$2) ORDER BY gc.environment,gc.version DESC LIMIT $3`, tenantID, environment, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]GatewayTransportRevision, 0)
	for rows.Next() {
		value, err := scanGatewayTransport(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, value)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) ActiveGatewayTransport(ctx context.Context, tenantID, environment string) (GatewayTransportRevision, error) {
	return scanGatewayTransport(r.pool.QueryRow(ctx, `SELECT `+gatewayTransportColumns+` FROM ai_gateway_config_revisions gc JOIN tenants t ON t.id=gc.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND gc.environment=$2 AND gc.status='ACTIVE' LIMIT 1`, tenantID, environment))
}

func (r *PostgresRepository) UpdateGatewayTransport(ctx context.Context, value GatewayTransportRevision, expected int64) (GatewayTransportRevision, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE ai_gateway_config_revisions gc SET status=$3,checker_id=NULLIF($4,'')::uuid,checksum=$5,submitted_at=$6,approved_at=$7,activated_at=$8,suspended_at=$9,retired_at=$10,updated_at=$11,record_version=$12 WHERE gc.id::text=$1 AND gc.tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND gc.record_version=$13`,
		value.ID, value.TenantID, value.Status, value.CheckerID, value.Checksum, value.SubmittedAt, value.ApprovedAt, value.ActivatedAt, value.SuspendedAt, value.RetiredAt, value.UpdatedAt, value.RecordVersion, expected)
	if err != nil {
		return GatewayTransportRevision{}, err
	}
	if tag.RowsAffected() != 1 {
		return GatewayTransportRevision{}, ErrConflict
	}
	return r.GatewayTransport(ctx, value.TenantID, value.ID)
}

func (r *PostgresRepository) ActivateGatewayTransport(ctx context.Context, value GatewayTransportRevision, expected int64) (GatewayTransportRevision, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return GatewayTransportRevision{}, err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `UPDATE ai_gateway_config_revisions SET status='SUSPENDED',suspended_at=$4,updated_at=$4,record_version=record_version+1 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND environment=$2 AND status='ACTIVE' AND id<>$3::uuid`, value.TenantID, value.Environment, value.ID, value.UpdatedAt)
	if err != nil {
		return GatewayTransportRevision{}, err
	}
	tag, err := tx.Exec(ctx, `UPDATE ai_gateway_config_revisions gc SET status='ACTIVE',checker_id=NULLIF($3,'')::uuid,activated_at=$4,suspended_at=NULL,updated_at=$5,record_version=$6 WHERE gc.id::text=$1 AND gc.tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND gc.record_version=$7`, value.ID, value.TenantID, value.CheckerID, value.ActivatedAt, value.UpdatedAt, value.RecordVersion, expected)
	if err != nil {
		return GatewayTransportRevision{}, err
	}
	if tag.RowsAffected() != 1 {
		return GatewayTransportRevision{}, ErrConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return GatewayTransportRevision{}, err
	}
	return r.GatewayTransport(ctx, value.TenantID, value.ID)
}
