//go:build postgres

package autonomy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Ingest(ctx context.Context, signal Signal, drift Drift) (bool, error) {
	payload, err := json.Marshal(signal.Payload)
	if err != nil {
		return false, fmt.Errorf("encode compliance signal: %w", err)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin signal ingestion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		INSERT INTO compliance_signals(id,tenant_id,signal_type,subject_type,subject_id,dedupe_key,source,observed_at,effective_at,payload)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9,$10::jsonb)
		ON CONFLICT(tenant_id,dedupe_key) DO NOTHING`,
		signal.ID, signal.TenantID, string(signal.Type), signal.SubjectType, signal.SubjectID, signal.DedupeKey, signal.Source, signal.ObservedAt, signal.EffectiveAt, string(payload),
	)
	if err != nil {
		return false, fmt.Errorf("insert compliance signal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO drift_assessments(id,tenant_id,subject_type,subject_id,dimension,severity,state,summary,required_action,signal_id,detected_at)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9,$10::uuid,$11)
		ON CONFLICT(tenant_id,subject_type,subject_id,dimension) WHERE state='ACTIVE'
		DO UPDATE SET severity=EXCLUDED.severity,summary=EXCLUDED.summary,required_action=EXCLUDED.required_action,
		              signal_id=EXCLUDED.signal_id,detected_at=EXCLUDED.detected_at`,
		drift.ID, drift.TenantID, drift.SubjectType, drift.SubjectID, drift.Dimension, drift.Severity, drift.State, drift.Summary, drift.RequiredAction, drift.SignalID, drift.DetectedAt,
	); err != nil {
		return false, fmt.Errorf("upsert drift: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit signal ingestion: %w", err)
	}
	return true, nil
}

func (r *PostgresRepository) ListDrifts(ctx context.Context, tenant string) ([]Drift, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT da.id::text,t.slug,da.subject_type,da.subject_id,da.dimension,da.severity,da.state,
		       da.summary,da.required_action,COALESCE(da.signal_id::text,''),da.detected_at
		FROM drift_assessments da JOIN tenants t ON t.id=da.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND da.state='ACTIVE'
		ORDER BY da.severity DESC,da.detected_at DESC LIMIT 100`, tenant)
	if err != nil {
		return nil, fmt.Errorf("list drift: %w", err)
	}
	defer rows.Close()
	values := []Drift{}
	for rows.Next() {
		var value Drift
		if err := rows.Scan(&value.ID, &value.TenantID, &value.SubjectType, &value.SubjectID, &value.Dimension, &value.Severity, &value.State, &value.Summary, &value.RequiredAction, &value.SignalID, &value.DetectedAt); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}
