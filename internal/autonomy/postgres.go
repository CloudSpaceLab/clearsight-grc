//go:build postgres

package autonomy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository { return &PostgresRepository{pool: pool} }
func (r *PostgresRepository) InsertSignal(ctx context.Context, value Signal) (bool, error) {
	payload, _ := json.Marshal(value.Payload)
	tag, err := r.pool.Exec(ctx, `INSERT INTO compliance_signals(id,tenant_id,signal_type,subject_type,subject_id,dedupe_key,source,observed_at,effective_at,payload) VALUES(NULLIF($1,'')::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9,$10::jsonb) ON CONFLICT(tenant_id,dedupe_key) DO NOTHING`, value.ID, value.TenantID, string(value.Type), value.SubjectType, value.SubjectID, value.DedupeKey, value.Source, value.ObservedAt, value.EffectiveAt, string(payload))
	if err != nil { return false, fmt.Errorf("insert compliance signal: %w", err) }
	return tag.RowsAffected() == 1, nil
}
func (r *PostgresRepository) UpsertDrift(ctx context.Context, value Drift) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO drift_assessments(id,tenant_id,subject_type,subject_id,dimension,severity,state,summary,required_action,signal_id,detected_at) VALUES(NULLIF($1,'')::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9,NULLIF($10,'')::uuid,$11) ON CONFLICT(tenant_id,subject_type,subject_id,dimension) WHERE state='ACTIVE' DO UPDATE SET severity=EXCLUDED.severity,summary=EXCLUDED.summary,required_action=EXCLUDED.required_action,signal_id=EXCLUDED.signal_id,detected_at=EXCLUDED.detected_at`, value.ID, value.TenantID, value.SubjectType, value.SubjectID, value.Dimension, value.Severity, value.State, value.Summary, value.RequiredAction, value.SignalID, value.DetectedAt)
	if err != nil { return fmt.Errorf("upsert drift: %w", err) }
	return nil
}
func (r *PostgresRepository) ListDrifts(ctx context.Context, tenant string) ([]Drift, error) {
	rows, err := r.pool.Query(ctx, `SELECT da.id::text,t.slug,da.subject_type,da.subject_id,da.dimension,da.severity,da.state,da.summary,da.required_action,COALESCE(da.signal_id::text,''),da.detected_at FROM drift_assessments da JOIN tenants t ON t.id=da.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND da.state='ACTIVE' ORDER BY da.severity DESC,da.detected_at DESC LIMIT 100`, tenant)
	if err != nil { return nil, fmt.Errorf("list drift: %w", err) }
	defer rows.Close()
	values := []Drift{}
	for rows.Next() {
		var value Drift
		if err := rows.Scan(&value.ID, &value.TenantID, &value.SubjectType, &value.SubjectID, &value.Dimension, &value.Severity, &value.State, &value.Summary, &value.RequiredAction, &value.SignalID, &value.DetectedAt); err != nil { return nil, err }
		values = append(values, value)
	}
	return values, rows.Err()
}
