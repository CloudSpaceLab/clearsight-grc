//go:build postgres

package aigovernance

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

const gatewayEmergencyColumns = `c.id::text,t.slug,c.environment,c.frozen,c.reason,c.actor_id::text,c.updated_at,c.record_version`

func scanGatewayEmergency(row rowScanner) (GatewayEmergencyControl, error) {
	var value GatewayEmergencyControl
	if err := row.Scan(&value.ID, &value.TenantID, &value.Environment, &value.Frozen, &value.Reason, &value.ActorID, &value.UpdatedAt, &value.RecordVersion); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GatewayEmergencyControl{}, ErrNotFound
		}
		return GatewayEmergencyControl{}, err
	}
	return value, nil
}

func (r *PostgresRepository) GatewayEmergencyControl(ctx context.Context, tenantID, environment string) (GatewayEmergencyControl, error) {
	return scanGatewayEmergency(r.pool.QueryRow(ctx, `SELECT `+gatewayEmergencyColumns+`
FROM ai_gateway_emergency_controls c
JOIN tenants t ON t.id=c.tenant_id
WHERE (t.id::text=$1 OR t.slug=$1) AND c.environment=$2`, tenantID, environment))
}

func (r *PostgresRepository) SetGatewayEmergencyControl(ctx context.Context, value GatewayEmergencyControl, expected int64) (GatewayEmergencyControl, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return GatewayEmergencyControl{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var stored GatewayEmergencyControl
	if expected == 0 {
		stored, err = scanGatewayEmergency(tx.QueryRow(ctx, `INSERT INTO ai_gateway_emergency_controls(id,tenant_id,environment,frozen,reason,actor_id,updated_at,record_version)
VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6::uuid,$7,$8)
RETURNING id::text,(SELECT slug FROM tenants WHERE id=ai_gateway_emergency_controls.tenant_id),environment,frozen,reason,actor_id::text,updated_at,record_version`,
			value.ID, value.TenantID, value.Environment, value.Frozen, value.Reason, value.ActorID, value.UpdatedAt, value.RecordVersion))
	} else {
		stored, err = scanGatewayEmergency(tx.QueryRow(ctx, `UPDATE ai_gateway_emergency_controls c
SET frozen=$4,reason=$5,actor_id=$6::uuid,updated_at=$7,record_version=$8
FROM tenants t
WHERE c.tenant_id=t.id AND (t.id::text=$1 OR t.slug=$1) AND c.environment=$2 AND c.id::text=$3 AND c.record_version=$9
RETURNING c.id::text,t.slug,c.environment,c.frozen,c.reason,c.actor_id::text,c.updated_at,c.record_version`,
			value.TenantID, value.Environment, value.ID, value.Frozen, value.Reason, value.ActorID, value.UpdatedAt, value.RecordVersion, expected))
		if errors.Is(err, ErrNotFound) {
			return GatewayEmergencyControl{}, ErrConflict
		}
	}
	if err != nil {
		return GatewayEmergencyControl{}, err
	}

	eventType := "AI_GATEWAY_OUTBOUND_UNFROZEN"
	if stored.Frozen {
		eventType = "AI_GATEWAY_OUTBOUND_FROZEN"
	}
	if _, err := tx.Exec(ctx, `INSERT INTO outbox_events(id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
VALUES(uuidv7(),(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'AI_GATEWAY_EMERGENCY_CONTROL',$2::uuid,$3::text,
       jsonb_build_object('actor_id',$4::text,'environment',$5::text,'frozen',$6::boolean,'reason',$7::text,'record_version',$8::bigint),$9::timestamptz,$9::timestamptz,$9::timestamptz)`,
		stored.TenantID, stored.ID, eventType, stored.ActorID, stored.Environment, stored.Frozen, stored.Reason, stored.RecordVersion, stored.UpdatedAt); err != nil {
		return GatewayEmergencyControl{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return GatewayEmergencyControl{}, err
	}
	return stored, nil
}
