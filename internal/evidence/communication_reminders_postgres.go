//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresCommunicationReminderRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresCommunicationReminderRepository(pool *pgxpool.Pool) *PostgresCommunicationReminderRepository {
	return &PostgresCommunicationReminderRepository{pool: pool}
}

func (repository *PostgresCommunicationReminderRepository) ScheduleDueCommunicationReminders(ctx context.Context, now time.Time, limit int) (int, error) {
	if repository == nil || repository.pool == nil {
		return 0, ErrCommunicationUnavailable
	}
	rows, err := repository.pool.Query(ctx, `
		SELECT id::text,tenant_id::text,deadline,reminder_policy
		FROM capture_form_distributions
		WHERE status='OPEN' AND deadline>$1 AND route_expires_at>$1 AND reminder_policy<>'{}'::jsonb
		ORDER BY deadline,id
		LIMIT $2`, now.UTC(), limit)
	if err != nil {
		return 0, err
	}
	type candidate struct {
		distributionID string
		tenantID       string
		deadline       time.Time
		policy         map[string]any
	}
	candidates := make([]candidate, 0, limit)
	for rows.Next() {
		var value candidate
		if err := rows.Scan(&value.distributionID, &value.tenantID, &value.deadline, &value.policy); err != nil {
			rows.Close()
			return 0, err
		}
		candidates = append(candidates, value)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	created := 0
	for _, candidate := range candidates {
		specs, err := communicationReminderSpecs(candidate.policy)
		if err != nil {
			return created, err
		}
		for _, spec := range specs {
			dueAt := candidate.deadline.Add(-time.Duration(spec.HoursBefore) * time.Hour)
			if dueAt.After(now) {
				continue
			}
			inserted, err := repository.insertReminder(ctx, candidate.tenantID, candidate.distributionID, candidate.deadline, spec, now)
			if err != nil {
				return created, err
			}
			if inserted {
				created++
			}
		}
	}
	return created, nil
}

func (repository *PostgresCommunicationReminderRepository) insertReminder(ctx context.Context, tenantID, distributionID string, deadline time.Time, spec communicationReminderSpec, now time.Time) (bool, error) {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "form-reminder:"+distributionID); err != nil {
		return false, err
	}
	var eligible bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM capture_form_distributions
			WHERE tenant_id=$1::uuid AND id=$2::uuid AND status='OPEN' AND deadline=$3 AND deadline>$4 AND route_expires_at>$4
		)`, tenantID, distributionID, deadline.UTC(), now.UTC()).Scan(&eligible); err != nil {
		return false, err
	}
	if !eligible {
		return false, tx.Commit(ctx)
	}
	deadlineText := deadline.UTC().Format(time.RFC3339Nano)
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM outbox_events
			WHERE tenant_id=$1::uuid AND aggregate_type='FORM_DISTRIBUTION' AND aggregate_id=$2::uuid
			  AND event_type='FORM_COMMUNICATION_REMINDER_DUE'
			  AND payload->>'action'=$3 AND payload->>'offset_hours'=$4 AND payload->>'deadline'=$5
		)`, tenantID, distributionID, spec.Action, fmt.Sprintf("%d", spec.HoursBefore), deadlineText).Scan(&exists); err != nil {
		return false, err
	}
	if exists {
		return false, tx.Commit(ctx)
	}
	eventID, err := id.NewUUIDv7()
	if err != nil {
		return false, err
	}
	payload, err := json.Marshal(map[string]any{
		"action":       spec.Action,
		"offset_hours": spec.HoursBefore,
		"deadline":     deadlineText,
	})
	if err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO outbox_events(id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		VALUES($1::uuid,$2::uuid,'FORM_DISTRIBUTION',$3::uuid,'FORM_COMMUNICATION_REMINDER_DUE',$4::jsonb,$5,$5)`,
		eventID, tenantID, distributionID, payload, now.UTC()); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

var _ communicationReminderSchedulerRepository = (*PostgresCommunicationReminderRepository)(nil)
