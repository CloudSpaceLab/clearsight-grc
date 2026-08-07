//go:build postgres

package autonomy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

func (r *PostgresRepository) Resolve(ctx context.Context, signal Signal, dimension string, resolvedAt time.Time) (bool, error) {
	payload, err := json.Marshal(signal.Payload)
	if err != nil {
		return false, fmt.Errorf("encode compliance signal: %w", err)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin signal resolution: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		INSERT INTO compliance_signals(id,tenant_id,signal_type,subject_type,subject_id,dedupe_key,source,observed_at,effective_at,payload)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9,$10::jsonb)
		ON CONFLICT(tenant_id,dedupe_key) DO NOTHING`,
		signal.ID, signal.TenantID, string(signal.Type), signal.SubjectType, signal.SubjectID, signal.DedupeKey, signal.Source, signal.ObservedAt, signal.EffectiveAt, string(payload),
	)
	if err != nil {
		return false, fmt.Errorf("insert recovery signal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE drift_assessments
		SET state='RESOLVED',resolved_at=$5
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND subject_type=$2 AND subject_id=$3 AND dimension=$4 AND state='ACTIVE'`,
		signal.TenantID, signal.SubjectType, signal.SubjectID, dimension, resolvedAt,
	); err != nil {
		return false, fmt.Errorf("resolve drift: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit signal resolution: %w", err)
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

func (r *PostgresRepository) ListAutomationPolicies(ctx context.Context, tenant string) ([]AutomationPolicy, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (ap.code)
		       ap.id::text,t.slug,ap.code,ap.name,ap.action_class,ap.eligibility,ap.blast_radius_limit,
		       ap.verification_contract,ap.status,
		       COALESCE(ap.effective_from,'epoch'::timestamptz),COALESCE(ap.effective_until,'epoch'::timestamptz),ap.version
		FROM automation_policies ap
		JOIN tenants t ON t.id=ap.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		ORDER BY ap.code,ap.version DESC
		LIMIT 200`, tenant)
	if err != nil {
		return nil, fmt.Errorf("list automation policies: %w", err)
	}
	defer rows.Close()
	values := []AutomationPolicy{}
	for rows.Next() {
		var value AutomationPolicy
		var eligibility, blastRadius, verification []byte
		var effectiveFrom, effectiveUntil time.Time
		if err := rows.Scan(&value.ID, &value.TenantID, &value.Code, &value.Name, &value.ActionClass, &eligibility, &blastRadius, &verification, &value.Status, &effectiveFrom, &effectiveUntil, &value.Version); err != nil {
			return nil, err
		}
		value.Eligibility = append(json.RawMessage(nil), eligibility...)
		value.BlastRadiusLimit = append(json.RawMessage(nil), blastRadius...)
		value.VerificationContract = append(json.RawMessage(nil), verification...)
		value.EffectiveFrom = automationTime(effectiveFrom)
		value.EffectiveUntil = automationTime(effectiveUntil)
		values = append(values, value)
	}
	return values, rows.Err()
}

func automationTime(value time.Time) *time.Time {
	if value.Equal(time.Unix(0, 0).UTC()) {
		return nil
	}
	utc := value.UTC()
	return &utc
}
