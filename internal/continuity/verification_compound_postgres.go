//go:build postgres

package continuity

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) ApplyVerificationResultBundle(ctx context.Context, bundle VerificationResultBundle) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var current int64
	err = tx.QueryRow(ctx, `SELECT m.version FROM matters m JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND m.id=$2::uuid FOR UPDATE`, bundle.TenantID, bundle.MatterID).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if current != bundle.ExpectedVersion || bundle.ResultEvent.AggregateVersion != bundle.ExpectedVersion+1 {
		return ErrVersionConflict
	}

	if err := applyMatterProjection(ctx, tx, bundle.ResultEvent); err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicate
		}
		return err
	}
	if err := insertContinuityEvent(ctx, tx, bundle.ResultEvent); err != nil {
		return err
	}
	if err := insertOutbox(ctx, tx, bundle.ResultEvent); err != nil {
		return err
	}

	finalEvent := bundle.ResultEvent
	if bundle.TransitionEvent != nil {
		if bundle.TransitionEvent.AggregateVersion != bundle.ResultEvent.AggregateVersion+1 {
			return ErrVersionConflict
		}
		if err := applyMatterProjection(ctx, tx, *bundle.TransitionEvent); err != nil {
			return err
		}
		if err := insertContinuityEvent(ctx, tx, *bundle.TransitionEvent); err != nil {
			return err
		}
		if err := insertOutbox(ctx, tx, *bundle.TransitionEvent); err != nil {
			return err
		}
		finalEvent = *bundle.TransitionEvent
	}
	if bundle.EscalationEvent != nil {
		if bundle.EscalationAction == nil || bundle.EscalationEvent.AggregateVersion != finalEvent.AggregateVersion+1 {
			return ErrVersionConflict
		}
		if err := applyMatterProjection(ctx, tx, *bundle.EscalationEvent); err != nil {
			return err
		}
		if err := insertContinuityEvent(ctx, tx, *bundle.EscalationEvent); err != nil {
			return err
		}
		if err := insertOutbox(ctx, tx, *bundle.EscalationEvent); err != nil {
			return err
		}
		finalEvent = *bundle.EscalationEvent
	}

	tag, err := tx.Exec(ctx, `UPDATE matters SET version=$3,updated_at=$4 WHERE id=$2::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND version=$5`, bundle.TenantID, bundle.MatterID, finalEvent.AggregateVersion, finalEvent.OccurredAt, bundle.ExpectedVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrVersionConflict
	}

	rows, err := tx.Query(ctx, `SELECT DISTINCT program_id::text FROM matter_links WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND matter_id=$2::uuid AND program_id IS NOT NULL AND retired_at IS NULL`, bundle.TenantID, bundle.MatterID)
	if err != nil {
		return err
	}
	programIDs := []string{}
	for rows.Next() {
		var programID string
		if err := rows.Scan(&programID); err != nil {
			rows.Close()
			return err
		}
		programIDs = append(programIDs, programID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, programID := range programIDs {
		if _, err := queueProgramStateTx(ctx, tx, bundle.TenantID, programID, 0, finalEvent.Type, bundle.MatterID, finalEvent.ActorID, finalEvent.OccurredAt); err != nil {
			return err
		}
	}

	if bundle.FollowUpMatter != nil {
		if bundle.FollowUpEvent == nil {
			return ErrInvalidState
		}
		matter := *bundle.FollowUpMatter
		_, err = tx.Exec(ctx, `INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,source_type,source_id,trigger_type,trigger_id,trigger_key,known_facts,missing_facts,contradictions,owner_principal_id,required_authority,due_at,closed_at,closure_reason,reopen_count,created_at,updated_at,version,legal_entity_id)
			VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,NULLIF($18,'')::uuid,$19,$20,$21,$22,$23,$24,$24,1,$25::uuid)`,
			matter.ID, matter.TenantID, matter.Reference, matter.Type, matter.Status, matter.Priority, matter.Title, matter.Summary, rawJSON(matter.Scope, `{}`), matter.SourceType, matter.SourceID, matter.TriggerType, matter.TriggerID, matter.TriggerKey, rawJSON(matter.KnownFacts, `{}`), rawJSON(matter.MissingFacts, `[]`), rawJSON(matter.Contradictions, `[]`), matter.OwnerPrincipalID, matter.RequiredAuthority, matter.DueAt, matter.ClosedAt, matter.ClosureReason, matter.ReopenCount, matter.CreatedAt, matter.LegalEntityID)
		if err != nil {
			if isUniqueViolation(err) {
				return ErrDuplicate
			}
			return err
		}
		if err := insertContinuityEvent(ctx, tx, *bundle.FollowUpEvent); err != nil {
			return err
		}
		if err := insertOutbox(ctx, tx, *bundle.FollowUpEvent); err != nil {
			return err
		}
		if bundle.FollowUpLink != nil {
			if bundle.FollowUpLinkEvent == nil || bundle.FollowUpLinkEvent.AggregateVersion != 2 {
				return ErrVersionConflict
			}
			if err := applyMatterProjection(ctx, tx, *bundle.FollowUpLinkEvent); err != nil {
				return err
			}
			if err := insertContinuityEvent(ctx, tx, *bundle.FollowUpLinkEvent); err != nil {
				return err
			}
			if err := insertOutbox(ctx, tx, *bundle.FollowUpLinkEvent); err != nil {
				return err
			}
			versionTag, err := tx.Exec(ctx, `UPDATE matters SET version=2,updated_at=$3 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id=$2::uuid AND version=1`, bundle.TenantID, matter.ID, bundle.FollowUpLinkEvent.OccurredAt)
			if err != nil {
				return err
			}
			if versionTag.RowsAffected() != 1 {
				return ErrVersionConflict
			}
			if _, err := queueProgramStateTx(ctx, tx, bundle.TenantID, bundle.FollowUpLink.ProgramID, 0, bundle.FollowUpLinkEvent.Type, matter.ID, bundle.FollowUpLinkEvent.ActorID, bundle.FollowUpLinkEvent.OccurredAt); err != nil {
				return err
			}
		}
	}

	events := []Event{bundle.ResultEvent}
	if bundle.TransitionEvent != nil {
		events = append(events, *bundle.TransitionEvent)
	}
	if bundle.FollowUpEvent != nil {
		events = append(events, *bundle.FollowUpEvent)
	}
	if bundle.FollowUpLinkEvent != nil {
		events = append(events, *bundle.FollowUpLinkEvent)
	}
	return r.commitContinuityEvents(ctx, tx, events...)
}
