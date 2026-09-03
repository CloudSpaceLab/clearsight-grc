//go:build postgres

package monitoring

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const collectionCycleColumns = `id::text,tenant_id::text,program_id::text,monitoring_check_id::text,monitoring_check_version,sequence,
	validity_months,renewal_window_days,reminder_count,reminders_sent,
	COALESCE(current_request_id::text,''),COALESCE(predecessor_request_id::text,''),COALESCE(latest_submission_id::text,''),latest_submitted_at,
	expires_at,renewal_opens_at,next_action_at,recipient_route_type,COALESCE(recipient_principal_id::text,''),COALESCE(recipient_contact_ref,''),recipient_safe_hint,
	delivery_state,delivery_reference,state,COALESCE(lease_owner,''),COALESCE(lease_token::text,''),lease_until,attempts,safe_error,created_at,updated_at`

func (r *PostgresRepository) UpsertCollectionCycle(ctx context.Context, value CollectionCycle) (CollectionCycle, error) {
	validated, err := validateCollectionCycle(value)
	if err != nil {
		return CollectionCycle{}, err
	}
	created, err := scanCollectionCycle(r.pool.QueryRow(ctx, `
		INSERT INTO monitoring_collection_cycles(
			id,tenant_id,program_id,monitoring_check_id,monitoring_check_version,sequence,
			validity_months,renewal_window_days,reminder_count,reminders_sent,
			current_request_id,predecessor_request_id,latest_submission_id,latest_submitted_at,
			expires_at,renewal_opens_at,next_action_at,recipient_route_type,recipient_principal_id,recipient_contact_ref,recipient_safe_hint,
			delivery_state,delivery_reference,state,lease_owner,lease_token,lease_until,attempts,safe_error,created_at,updated_at)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,$5,$6,$7,$8,$9,$10,
			NULLIF($11,'')::uuid,NULLIF($12,'')::uuid,NULLIF($13,'')::uuid,$14,$15,$16,$17,$18,NULLIF($19,'')::uuid,NULLIF($20,''),$21,
			$22,$23,$24,NULLIF($25,''),NULLIF($26,'')::uuid,$27,$28,$29,$30,$31)
		ON CONFLICT(tenant_id,monitoring_check_id,sequence) DO NOTHING
		RETURNING `+collectionCycleColumns,
		validated.ID, validated.TenantID, validated.ProgramID, validated.MonitoringCheckID, validated.MonitoringCheckVersion, validated.Sequence,
		validated.Policy.ValidityMonths, validated.Policy.RenewalWindowDays, validated.Policy.ReminderCount, validated.RemindersSent,
		validated.CurrentRequestID, validated.PredecessorRequestID, validated.LatestSubmissionID, validated.LatestSubmittedAt,
		validated.ExpiresAt, validated.RenewalOpensAt, validated.NextActionAt, validated.Recipient.Type, validated.Recipient.PrincipalID, validated.Recipient.ContactRef, validated.Recipient.SafeHint,
		validated.DeliveryState, validated.DeliveryReference, validated.State, validated.LeaseOwner, validated.LeaseToken, validated.LeaseUntil,
		validated.Attempts, validated.SafeError, validated.CreatedAt, validated.UpdatedAt))
	if errors.Is(err, pgx.ErrNoRows) {
		existing, lookupErr := r.CollectionCycleForSequence(ctx, validated.TenantID, validated.MonitoringCheckID, validated.Sequence)
		if lookupErr != nil {
			return CollectionCycle{}, lookupErr
		}
		if !sameCollectionSchedule(existing, validated) {
			return CollectionCycle{}, ErrConflict
		}
		return existing, nil
	}
	return created, mapPostgresError(err)
}

func (r *PostgresRepository) CollectionCycleForSequence(ctx context.Context, tenant, checkID string, sequence int64) (CollectionCycle, error) {
	value, err := scanCollectionCycle(r.pool.QueryRow(ctx, `SELECT `+collectionCycleColumns+`
		FROM monitoring_collection_cycles
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND monitoring_check_id=$2::uuid AND sequence=$3`, tenant, checkID, sequence))
	return value, mapPostgresError(err)
}

func (r *PostgresRepository) CollectionCycle(ctx context.Context, tenant, cycleID string) (CollectionCycle, error) {
	value, err := scanCollectionCycle(r.pool.QueryRow(ctx, `SELECT `+collectionCycleColumns+`
		FROM monitoring_collection_cycles
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id=$2::uuid`, tenant, cycleID))
	return value, mapPostgresError(err)
}

func (r *PostgresRepository) ClaimDueCollectionCycles(ctx context.Context, worker string, now time.Time, lease time.Duration, limit int) ([]CollectionCycle, error) {
	if strings.TrimSpace(worker) == "" || now.IsZero() || lease <= 0 {
		return nil, ErrInvalid
	}
	rows, err := r.pool.Query(ctx, `
		WITH due AS (
			SELECT id AS due_id FROM monitoring_collection_cycles
			WHERE next_action_at <= $1 AND (
				state IN ('SCHEDULED','AWAITING_RESPONSE') OR (state='CLAIMED' AND lease_until <= $1)
			)
			ORDER BY next_action_at,id
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		UPDATE monitoring_collection_cycles c
		SET state='CLAIMED',lease_owner=$3,lease_token=uuidv7(),lease_until=$1 + ($4::bigint * interval '1 microsecond'),updated_at=$1
		FROM due WHERE c.id=due.due_id
		RETURNING `+collectionCycleColumns, now.UTC(), boundedCollectionLimit(limit), strings.TrimSpace(worker), lease.Microseconds())
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()
	values := make([]CollectionCycle, 0)
	for rows.Next() {
		value, scanErr := scanCollectionCycle(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *PostgresRepository) CompleteCollectionAction(ctx context.Context, claim CollectionCycle, completion CollectionActionCompletion) (CollectionCycle, error) {
	if completion.At.IsZero() || claim.LeaseToken == "" || claim.LeaseOwner == "" || len(strings.TrimSpace(completion.SafeError)) > 1000 {
		return CollectionCycle{}, ErrInvalid
	}
	switch completion.State {
	case CycleScheduled, CycleAwaitingResponse:
		if completion.NextActionAt == nil {
			return CollectionCycle{}, ErrInvalid
		}
	case CycleComplete, CycleCancelled, CycleBlocked:
	default:
		return CollectionCycle{}, ErrInvalid
	}
	updated, err := scanCollectionCycle(r.pool.QueryRow(ctx, `
		UPDATE monitoring_collection_cycles SET
			state=$5,current_request_id=COALESCE(NULLIF($6,'')::uuid,current_request_id),
			delivery_state=COALESCE(NULLIF($7,''),delivery_state),delivery_reference=CASE WHEN $8='' THEN delivery_reference ELSE $8 END,
			next_action_at=$9,reminders_sent=COALESCE($10,reminders_sent),safe_error=$11,
			lease_owner=NULL,lease_token=NULL,lease_until=NULL,updated_at=$12
		WHERE tenant_id=$1::uuid AND id=$2::uuid AND state='CLAIMED' AND lease_token=$3::uuid AND lease_owner=$4 AND lease_until >= $12
		RETURNING `+collectionCycleColumns,
		claim.TenantID, claim.ID, claim.LeaseToken, claim.LeaseOwner, completion.State, strings.TrimSpace(completion.CurrentRequestID),
		completion.DeliveryState, strings.TrimSpace(completion.DeliveryReference), completion.NextActionAt, nullableInt(completion.RemindersSent), strings.TrimSpace(completion.SafeError), completion.At.UTC()))
	if errors.Is(err, pgx.ErrNoRows) {
		return CollectionCycle{}, ErrConflict
	}
	return updated, mapPostgresError(err)
}

func (r *PostgresRepository) FailCollectionAction(ctx context.Context, claim CollectionCycle, safeError string, retryAt *time.Time, maxAttempts int, at time.Time) (CollectionCycle, error) {
	if claim.LeaseToken == "" || claim.LeaseOwner == "" || strings.TrimSpace(safeError) == "" || len(strings.TrimSpace(safeError)) > 1000 || maxAttempts < 1 || maxAttempts > 20 || at.IsZero() {
		return CollectionCycle{}, ErrInvalid
	}
	updated, err := scanCollectionCycle(r.pool.QueryRow(ctx, `
		UPDATE monitoring_collection_cycles SET
			attempts=attempts+1,
			state=CASE WHEN attempts+1 >= $5 OR $6::timestamptz IS NULL THEN 'FAILED' ELSE 'SCHEDULED' END,
			delivery_state=CASE WHEN attempts+1 >= $5 OR $6::timestamptz IS NULL THEN 'FAILED' ELSE delivery_state END,
			next_action_at=CASE WHEN attempts+1 >= $5 THEN NULL ELSE $6::timestamptz END,
			safe_error=$7,lease_owner=NULL,lease_token=NULL,lease_until=NULL,updated_at=$8
		WHERE tenant_id=$1::uuid AND id=$2::uuid AND state='CLAIMED' AND lease_token=$3::uuid AND lease_owner=$4 AND lease_until >= $8
		RETURNING `+collectionCycleColumns,
		claim.TenantID, claim.ID, claim.LeaseToken, claim.LeaseOwner, maxAttempts, retryAt, strings.TrimSpace(safeError), at.UTC()))
	if errors.Is(err, pgx.ErrNoRows) {
		return CollectionCycle{}, ErrConflict
	}
	return updated, mapPostgresError(err)
}

func (r *PostgresRepository) CancelCollectionCyclesByCheck(ctx context.Context, tenant, checkID string, at time.Time) (int, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(checkID) == "" || at.IsZero() {
		return 0, ErrInvalid
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE monitoring_collection_cycles SET state='CANCELLED',next_action_at=NULL,lease_owner=NULL,lease_token=NULL,lease_until=NULL,updated_at=$3
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND monitoring_check_id=$2::uuid
			AND state NOT IN ('COMPLETE','CANCELLED','FAILED')`, tenant, checkID, at.UTC())
	if err != nil {
		return 0, mapPostgresError(err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *PostgresRepository) CompleteCollectionCyclesBeforeSequence(ctx context.Context, tenant, checkID string, sequence int64, at time.Time) (int, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(checkID) == "" || sequence < 1 || at.IsZero() {
		return 0, ErrInvalid
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE monitoring_collection_cycles SET state='COMPLETE',next_action_at=NULL,lease_owner=NULL,lease_token=NULL,lease_until=NULL,updated_at=$4
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND monitoring_check_id=$2::uuid AND sequence<$3
			AND state NOT IN ('COMPLETE','CANCELLED','FAILED')`, tenant, checkID, sequence, at.UTC())
	if err != nil {
		return 0, mapPostgresError(err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *PostgresRepository) ListCollectionSummaries(ctx context.Context, tenant, programID string, limit int) ([]CollectionSummary, error) {
	if strings.TrimSpace(tenant) == "" || strings.TrimSpace(programID) == "" {
		return nil, ErrInvalid
	}
	rows, err := r.pool.Query(ctx, `SELECT `+collectionCycleColumns+`
		FROM monitoring_collection_cycles c
		WHERE c.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND c.program_id=$2::uuid
			AND NOT EXISTS (
				SELECT 1 FROM monitoring_collection_cycles newer
				WHERE newer.tenant_id=c.tenant_id AND newer.monitoring_check_id=c.monitoring_check_id AND newer.sequence>c.sequence
			)
		ORDER BY c.monitoring_check_id
		LIMIT $3`, tenant, programID, boundedCollectionLimit(limit))
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()
	cycles := make([]CollectionCycle, 0)
	generatedAt := time.Time{}
	for rows.Next() {
		value, scanErr := scanCollectionCycle(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		cycles = append(cycles, value)
		if value.UpdatedAt.After(generatedAt) {
			generatedAt = value.UpdatedAt
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	summaries := make([]CollectionSummary, len(cycles))
	for index, value := range cycles {
		summaries[index] = collectionSummary(value, generatedAt)
	}
	return summaries, nil
}

func scanCollectionCycle(row scanner) (CollectionCycle, error) {
	var value CollectionCycle
	err := row.Scan(
		&value.ID, &value.TenantID, &value.ProgramID, &value.MonitoringCheckID, &value.MonitoringCheckVersion, &value.Sequence,
		&value.Policy.ValidityMonths, &value.Policy.RenewalWindowDays, &value.Policy.ReminderCount, &value.RemindersSent,
		&value.CurrentRequestID, &value.PredecessorRequestID, &value.LatestSubmissionID, &value.LatestSubmittedAt,
		&value.ExpiresAt, &value.RenewalOpensAt, &value.NextActionAt, &value.Recipient.Type, &value.Recipient.PrincipalID, &value.Recipient.ContactRef, &value.Recipient.SafeHint,
		&value.DeliveryState, &value.DeliveryReference, &value.State, &value.LeaseOwner, &value.LeaseToken, &value.LeaseUntil,
		&value.Attempts, &value.SafeError, &value.CreatedAt, &value.UpdatedAt,
	)
	return value, err
}

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

var _ CollectionCycleRepository = (*PostgresRepository)(nil)
