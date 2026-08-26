//go:build postgres

package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) SaveProgramState(ctx context.Context, tenant, programID string, expectedProgramVersion int64, state ProgramStateSnapshot) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var currentVersion, projectionVersion int64
	if err := tx.QueryRow(ctx, `SELECT p.version FROM programs p JOIN tenants t ON t.id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND p.id=$2::uuid FOR UPDATE`, tenant, programID).Scan(&currentVersion); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	if currentVersion != expectedProgramVersion {
		return 0, ErrVersionConflict
	}
	if err := tx.QueryRow(ctx, `SELECT COALESCE(max(projection_version),0)+1 FROM program_state_snapshots WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND program_id=$2::uuid`, tenant, programID).Scan(&projectionVersion); err != nil {
		return 0, err
	}
	state.ProgramVersion = expectedProgramVersion
	state.ProjectionVersion = projectionVersion
	dimensions, _ := json.Marshal(state.Dimensions)
	reasons, _ := json.Marshal(state.Reasons)
	_, err = tx.Exec(ctx, `INSERT INTO program_state_snapshots(id,tenant_id,program_id,overall_state,dimensions,reasons,open_matter_count,trigger_type,trigger_id,generated_at,program_version,projection_version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, state.ID, tenant, programID, state.Overall, dimensions, reasons, state.OpenMatterCount, state.TriggerType, state.TriggerID, state.GeneratedAt, state.ProgramVersion, projectionVersion)
	if err != nil {
		return 0, err
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return 0, err
	}
	eventID, err := id.NewUUIDv7()
	if err != nil {
		return 0, err
	}
	event := Event{ID: eventID, TenantID: tenant, AggregateType: "PROGRAM_STATE", AggregateID: programID, AggregateVersion: projectionVersion, Type: EventProgramStateUpdated, Payload: payload, ActorType: ActorSystem, OccurredAt: state.GeneratedAt}
	if err = insertContinuityEvent(ctx, tx, event); err != nil {
		return 0, err
	}
	if err = insertOutbox(ctx, tx, event); err != nil {
		return 0, err
	}
	if err = r.commitContinuityEvents(ctx, tx, event); err != nil {
		return 0, err
	}
	return projectionVersion, nil
}

func (r *PostgresRepository) ProgramStateAt(ctx context.Context, tenant, programID string, at *time.Time) (*ProgramStateSnapshot, error) {
	query := `SELECT pss.id::text,t.slug,pss.program_id::text,pss.overall_state,pss.dimensions,pss.reasons,pss.open_matter_count,pss.trigger_type,pss.trigger_id,pss.generated_at,pss.program_version,pss.projection_version FROM program_state_snapshots pss JOIN tenants t ON t.id=pss.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND pss.program_id=$2::uuid`
	args := []any{tenant, programID}
	if at != nil {
		query += ` AND pss.generated_at<=$3`
		args = append(args, *at)
	}
	query += ` ORDER BY pss.generated_at DESC,pss.projection_version DESC LIMIT 1`
	var state ProgramStateSnapshot
	var dimensions, reasons json.RawMessage
	err := r.pool.QueryRow(ctx, query, args...).Scan(&state.ID, &state.TenantID, &state.ProgramID, &state.Overall, &dimensions, &reasons, &state.OpenMatterCount, &state.TriggerType, &state.TriggerID, &state.GeneratedAt, &state.ProgramVersion, &state.ProjectionVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(dimensions, &state.Dimensions); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(reasons, &state.Reasons); err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *PostgresRepository) QueueProgramState(ctx context.Context, tenant, programID string, sourceVersion int64, reason, triggerID, requestedBy string) (ProjectionJob, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ProjectionJob{}, err
	}
	defer tx.Rollback(ctx)
	job, err := queueProgramStateTx(ctx, tx, tenant, programID, sourceVersion, reason, triggerID, requestedBy, time.Now().UTC())
	if err != nil {
		return ProjectionJob{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ProjectionJob{}, err
	}
	return job, nil
}

func queueProgramStateTx(ctx context.Context, tx pgx.Tx, tenant, programID string, sourceVersion int64, reason, triggerID, requestedBy string, now time.Time) (ProjectionJob, error) {
	if sourceVersion <= 0 {
		if err := tx.QueryRow(ctx, `SELECT p.version FROM programs p JOIN tenants t ON t.id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND p.id=$2::uuid`, tenant, programID).Scan(&sourceVersion); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ProjectionJob{}, ErrNotFound
			}
			return ProjectionJob{}, err
		}
	}
	jobID, err := id.NewUUIDv7()
	if err != nil {
		return ProjectionJob{}, err
	}
	row := tx.QueryRow(ctx, `INSERT INTO continuity_projection_jobs(id,tenant_id,projection_name,aggregate_type,aggregate_id,source_aggregate_version,reason,trigger_id,requested_by,status,available_at,created_at,updated_at) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),'PROGRAM_STATE','PROGRAM',$3::uuid,$4,$5,$6,$7,'READY',$8,$8,$8) ON CONFLICT (tenant_id,projection_name,aggregate_id) WHERE status IN ('READY','CLAIMED') DO UPDATE SET source_aggregate_version=GREATEST(continuity_projection_jobs.source_aggregate_version,EXCLUDED.source_aggregate_version),reason=EXCLUDED.reason,trigger_id=EXCLUDED.trigger_id,requested_by=CASE WHEN EXCLUDED.requested_by='' THEN continuity_projection_jobs.requested_by ELSE EXCLUDED.requested_by END,status='READY',available_at=LEAST(continuity_projection_jobs.available_at,EXCLUDED.available_at),claimed_at=NULL,claimed_by='',updated_at=EXCLUDED.updated_at RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),projection_name,aggregate_type,aggregate_id::text,source_aggregate_version,reason,trigger_id,requested_by,status,attempts,available_at,claimed_at,claimed_by,completed_at,last_error,created_at,updated_at`, jobID, tenant, programID, sourceVersion, reason, triggerID, requestedBy, now)
	return scanProjectionJob(row)
}

func (r *PostgresRepository) ClaimProgramState(ctx context.Context, workerID string, now time.Time, lease time.Duration, limit int) ([]ProjectionJob, error) {
	rows, err := r.pool.Query(ctx, `WITH selected AS (SELECT id FROM continuity_projection_jobs WHERE projection_name='PROGRAM_STATE' AND ((status='READY' AND available_at<=$2) OR (status='CLAIMED' AND claimed_at<$2-$3::interval)) ORDER BY available_at,created_at FOR UPDATE SKIP LOCKED LIMIT $4) UPDATE continuity_projection_jobs j SET status='CLAIMED',claimed_at=$2,claimed_by=$1,attempts=j.attempts+1,updated_at=$2 FROM selected WHERE j.id=selected.id RETURNING j.id::text,(SELECT slug FROM tenants WHERE id=j.tenant_id),j.projection_name,j.aggregate_type,j.aggregate_id::text,j.source_aggregate_version,j.reason,j.trigger_id,j.requested_by,j.status,j.attempts,j.available_at,j.claimed_at,j.claimed_by,j.completed_at,j.last_error,j.created_at,j.updated_at`, workerID, now, lease.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []ProjectionJob{}
	for rows.Next() {
		job, err := scanProjectionJob(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, job)
	}
	return values, rows.Err()
}

func (r *PostgresRepository) CompleteProgramState(ctx context.Context, job ProjectionJob, now time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE continuity_projection_jobs SET status='COMPLETED',completed_at=$3,updated_at=$3,last_error='' WHERE id=$1::uuid AND status='CLAIMED' AND claimed_by=$2`, job.ID, job.ClaimedBy, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrVersionConflict
	}
	return nil
}

func (r *PostgresRepository) FailProgramState(ctx context.Context, job ProjectionJob, message string, retryAt time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE continuity_projection_jobs SET status=CASE WHEN attempts>=5 THEN 'FAILED' ELSE 'READY' END,available_at=$3,claimed_at=NULL,claimed_by='',last_error=$4,updated_at=clock_timestamp() WHERE id=$1::uuid AND status='CLAIMED' AND claimed_by=$2`, job.ID, job.ClaimedBy, retryAt, message)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrVersionConflict
	}
	return nil
}

func (r *PostgresRepository) ProjectionHealth(ctx context.Context, tenant string) ([]ProjectionHealth, error) {
	var health ProjectionHealth
	health.TenantID = tenant
	health.Projection = ProjectionProgramState
	health.DisplayName = "Program status updates"
	err := r.pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE status IN ('READY','CLAIMED')),count(*) FILTER (WHERE status='FAILED'),min(created_at) FILTER (WHERE status IN ('READY','CLAIMED')),max(completed_at) FILTER (WHERE status='COMPLETED'),COALESCE((array_agg(last_error ORDER BY updated_at DESC) FILTER (WHERE last_error<>''))[1],''),clock_timestamp() FROM continuity_projection_jobs WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND projection_name='PROGRAM_STATE'`, tenant).Scan(&health.Pending, &health.Failed, &health.OldestPending, &health.LastCompleted, &health.LastError, &health.UpdatedAt)
	if err != nil {
		return nil, err
	}
	health.State = "CURRENT"
	if health.Failed > 0 {
		health.State = "NEEDS_ATTENTION"
	} else if health.Pending > 0 {
		health.State = "UPDATE_PENDING"
	}
	if health.OldestPending != nil {
		health.LagSeconds = int64(health.UpdatedAt.Sub(*health.OldestPending).Seconds())
		if health.LagSeconds > 60 && health.State == "UPDATE_PENDING" {
			health.State = "DELAYED"
		}
	}
	return []ProjectionHealth{health}, nil
}

func (r *PostgresRepository) ReconcileProgramState(ctx context.Context, tenant string, limit int) (ReconcileResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id::text,p.version,COALESCE(latest.program_version,0),COALESCE(active.queued,false)
		FROM programs p
		JOIN tenants t ON t.id=p.tenant_id
		LEFT JOIN LATERAL (
			SELECT program_version FROM program_state_snapshots pss
			WHERE pss.tenant_id=p.tenant_id AND pss.program_id=p.id
			ORDER BY pss.generated_at DESC,pss.projection_version DESC LIMIT 1
		) latest ON TRUE
		LEFT JOIN LATERAL (
			SELECT true AS queued FROM continuity_projection_jobs j
			WHERE j.tenant_id=p.tenant_id AND j.projection_name='PROGRAM_STATE' AND j.aggregate_id=p.id AND j.status IN ('READY','CLAIMED')
			LIMIT 1
		) active ON TRUE
		WHERE (t.id::text=$1 OR t.slug=$1)
		ORDER BY ((COALESCE(latest.program_version,0)<p.version) AND NOT COALESCE(active.queued,false)) DESC,p.updated_at,p.id
		LIMIT $2`, tenant, limit)
	if err != nil {
		return ReconcileResult{}, err
	}
	defer rows.Close()
	result := ReconcileResult{TenantID: tenant}
	for rows.Next() {
		var programID string
		var current, projected int64
		var alreadyQueued bool
		if err := rows.Scan(&programID, &current, &projected, &alreadyQueued); err != nil {
			return result, err
		}
		result.Checked++
		if projected >= current {
			result.Current++
			continue
		}
		if alreadyQueued {
			result.AlreadyQueued++
			continue
		}
		if _, err := r.QueueProgramState(ctx, tenant, programID, current, "RECONCILE", "", "system"); err != nil {
			return result, err
		}
		result.Queued++
	}
	return result, rows.Err()
}

func (r *PostgresRepository) CreateMatterWithLink(ctx context.Context, bundle MatterLinkBundle) (Matter, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Matter{}, err
	}
	defer tx.Rollback(ctx)
	matter := bundle.Matter
	_, err = tx.Exec(ctx, `INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,source_type,source_id,trigger_type,trigger_id,trigger_key,known_facts,missing_facts,contradictions,owner_principal_id,required_authority,due_at,closed_at,closure_reason,reopen_count,created_at,updated_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,NULLIF($18,'')::uuid,$19,$20,$21,$22,$23,$24,$24,$25)`, matter.ID, matter.TenantID, matter.Reference, matter.Type, matter.Status, matter.Priority, matter.Title, matter.Summary, rawJSON(matter.Scope, `{}`), matter.SourceType, matter.SourceID, matter.TriggerType, matter.TriggerID, matter.TriggerKey, rawJSON(matter.KnownFacts, `{}`), rawJSON(matter.MissingFacts, `[]`), rawJSON(matter.Contradictions, `[]`), matter.OwnerPrincipalID, matter.RequiredAuthority, matter.DueAt, matter.ClosedAt, matter.ClosureReason, matter.ReopenCount, matter.CreatedAt, matter.Version)
	if err != nil {
		if isUniqueViolation(err) {
			return Matter{}, ErrDuplicate
		}
		return Matter{}, err
	}
	if err = insertContinuityEvent(ctx, tx, bundle.MatterEvent); err != nil {
		return Matter{}, err
	}
	if err = insertOutbox(ctx, tx, bundle.MatterEvent); err != nil {
		return Matter{}, err
	}
	if err = applyMatterProjection(ctx, tx, bundle.LinkEvent); err != nil {
		return Matter{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE matters SET version=$3,updated_at=$4 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id=$2::uuid`, matter.TenantID, matter.ID, bundle.LinkEvent.AggregateVersion, bundle.LinkEvent.OccurredAt); err != nil {
		return Matter{}, err
	}
	if err = insertContinuityEvent(ctx, tx, bundle.LinkEvent); err != nil {
		return Matter{}, err
	}
	if err = insertOutbox(ctx, tx, bundle.LinkEvent); err != nil {
		return Matter{}, err
	}
	if _, err = queueProgramStateTx(ctx, tx, matter.TenantID, bundle.Link.ProgramID, 0, EventMatterLinked, matter.ID, bundle.LinkEvent.ActorID, bundle.LinkEvent.OccurredAt); err != nil {
		return Matter{}, err
	}
	if err = r.commitContinuityEvents(ctx, tx, bundle.MatterEvent, bundle.LinkEvent); err != nil {
		return Matter{}, err
	}
	matter.Version = bundle.LinkEvent.AggregateVersion
	matter.UpdatedAt = bundle.LinkEvent.OccurredAt
	return matter, nil
}

func (r *PostgresRepository) ApplyTriggerBundle(ctx context.Context, bundle TriggerBundle) (TriggerBundleResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return TriggerBundleResult{}, err
	}
	defer tx.Rollback(ctx)
	trigger := bundle.Trigger
	var currentVersion int64
	if err := tx.QueryRow(ctx, `SELECT p.version FROM programs p JOIN tenants t ON t.id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND p.id=$2::uuid FOR UPDATE`, trigger.TenantID, trigger.ProgramID).Scan(&currentVersion); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TriggerBundleResult{}, ErrNotFound
		}
		return TriggerBundleResult{}, err
	}
	tag, err := tx.Exec(ctx, `INSERT INTO program_trigger_events(id,tenant_id,program_id,trigger_type,subject_type,subject_id,dedupe_key,payload,observed_at,source) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(tenant_id,dedupe_key) DO NOTHING`, trigger.ID, trigger.TenantID, trigger.ProgramID, strings.ToUpper(trigger.Type), trigger.SubjectType, trigger.SubjectID, trigger.DedupeKey, rawJSON(trigger.Payload, `{}`), trigger.ObservedAt, trigger.Source)
	if err != nil {
		return TriggerBundleResult{}, err
	}
	if tag.RowsAffected() == 0 {
		var existing Matter
		err := tx.QueryRow(ctx, `SELECT m.id::text,t.slug,m.reference,m.matter_type,m.status,m.priority,m.title,m.summary,m.scope,m.source_type,COALESCE(m.source_id::text,''),m.trigger_type,COALESCE(m.trigger_id::text,''),m.trigger_key,m.known_facts,m.missing_facts,m.contradictions,COALESCE(m.owner_principal_id::text,''),m.required_authority,m.due_at,m.closed_at,m.closure_reason,m.reopen_count,m.created_at,m.updated_at,m.version FROM matters m JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND m.trigger_key=$2 AND m.status NOT IN ('CLOSED','CANCELLED') ORDER BY m.created_at DESC LIMIT 1`, trigger.TenantID, trigger.DedupeKey).Scan(&existing.ID, &existing.TenantID, &existing.Reference, &existing.Type, &existing.Status, &existing.Priority, &existing.Title, &existing.Summary, &existing.Scope, &existing.SourceType, &existing.SourceID, &existing.TriggerType, &existing.TriggerID, &existing.TriggerKey, &existing.KnownFacts, &existing.MissingFacts, &existing.Contradictions, &existing.OwnerPrincipalID, &existing.RequiredAuthority, &existing.DueAt, &existing.ClosedAt, &existing.ClosureReason, &existing.ReopenCount, &existing.CreatedAt, &existing.UpdatedAt, &existing.Version)
		if errors.Is(err, pgx.ErrNoRows) {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return TriggerBundleResult{}, commitErr
			}
			return TriggerBundleResult{Inserted: false}, nil
		}
		if err != nil {
			return TriggerBundleResult{}, err
		}
		if err = tx.Commit(ctx); err != nil {
			return TriggerBundleResult{}, err
		}
		return TriggerBundleResult{Inserted: false, Matter: &existing}, nil
	}
	if bundle.ProgramEvent.AggregateVersion != currentVersion+1 {
		return TriggerBundleResult{}, ErrVersionConflict
	}
	if err = applyProgramProjection(ctx, tx, bundle.ProgramEvent); err != nil {
		return TriggerBundleResult{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE programs SET version=$3,updated_at=$4 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id=$2::uuid AND version=$5`, trigger.TenantID, trigger.ProgramID, bundle.ProgramEvent.AggregateVersion, bundle.ProgramEvent.OccurredAt, currentVersion); err != nil {
		return TriggerBundleResult{}, err
	}
	if err = insertContinuityEvent(ctx, tx, bundle.ProgramEvent); err != nil {
		return TriggerBundleResult{}, err
	}
	if err = insertOutbox(ctx, tx, bundle.ProgramEvent); err != nil {
		return TriggerBundleResult{}, err
	}
	result := TriggerBundleResult{Inserted: true}
	if bundle.Matter != nil && bundle.MatterEvent != nil && bundle.Link != nil && bundle.LinkEvent != nil {
		matter := *bundle.Matter
		_, err = tx.Exec(ctx, `INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,source_type,source_id,trigger_type,trigger_id,trigger_key,known_facts,missing_facts,contradictions,owner_principal_id,required_authority,due_at,closed_at,closure_reason,reopen_count,created_at,updated_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,NULLIF($18,'')::uuid,$19,$20,$21,$22,$23,$24,$24,$25)`, matter.ID, matter.TenantID, matter.Reference, matter.Type, matter.Status, matter.Priority, matter.Title, matter.Summary, rawJSON(matter.Scope, `{}`), matter.SourceType, matter.SourceID, matter.TriggerType, matter.TriggerID, matter.TriggerKey, rawJSON(matter.KnownFacts, `{}`), rawJSON(matter.MissingFacts, `[]`), rawJSON(matter.Contradictions, `[]`), matter.OwnerPrincipalID, matter.RequiredAuthority, matter.DueAt, matter.ClosedAt, matter.ClosureReason, matter.ReopenCount, matter.CreatedAt, matter.Version)
		if err != nil {
			return TriggerBundleResult{}, err
		}
		if err = insertContinuityEvent(ctx, tx, *bundle.MatterEvent); err != nil {
			return TriggerBundleResult{}, err
		}
		if err = insertOutbox(ctx, tx, *bundle.MatterEvent); err != nil {
			return TriggerBundleResult{}, err
		}
		if err = applyMatterProjection(ctx, tx, *bundle.LinkEvent); err != nil {
			return TriggerBundleResult{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE matters SET version=$3,updated_at=$4 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id=$2::uuid`, matter.TenantID, matter.ID, bundle.LinkEvent.AggregateVersion, bundle.LinkEvent.OccurredAt); err != nil {
			return TriggerBundleResult{}, err
		}
		if err = insertContinuityEvent(ctx, tx, *bundle.LinkEvent); err != nil {
			return TriggerBundleResult{}, err
		}
		if err = insertOutbox(ctx, tx, *bundle.LinkEvent); err != nil {
			return TriggerBundleResult{}, err
		}
		matter.Version = bundle.LinkEvent.AggregateVersion
		matter.UpdatedAt = bundle.LinkEvent.OccurredAt
		result.Matter = &matter
	}
	if _, err = queueProgramStateTx(ctx, tx, trigger.TenantID, trigger.ProgramID, bundle.ProgramEvent.AggregateVersion, trigger.Type, trigger.ID, bundle.ProgramEvent.ActorID, time.Now().UTC()); err != nil {
		return TriggerBundleResult{}, err
	}
	events := []Event{bundle.ProgramEvent}
	if bundle.MatterEvent != nil && bundle.LinkEvent != nil {
		events = append(events, *bundle.MatterEvent, *bundle.LinkEvent)
	}
	if err = r.commitContinuityEvents(ctx, tx, events...); err != nil {
		return TriggerBundleResult{}, err
	}
	return result, nil
}

type projectionScanner interface{ Scan(...any) error }

func scanProjectionJob(row projectionScanner) (ProjectionJob, error) {
	var job ProjectionJob
	err := row.Scan(&job.ID, &job.TenantID, &job.Projection, &job.AggregateType, &job.AggregateID, &job.SourceAggregateVersion, &job.Reason, &job.TriggerID, &job.RequestedBy, &job.Status, &job.Attempts, &job.AvailableAt, &job.ClaimedAt, &job.ClaimedBy, &job.CompletedAt, &job.LastError, &job.CreatedAt, &job.UpdatedAt)
	return job, err
}

var _ ProgramStateRepository = (*PostgresRepository)(nil)
var _ ProjectionRepository = (*PostgresRepository)(nil)
var _ CompoundRepository = (*PostgresRepository)(nil)
var _ TriggerBundleRepository = (*PostgresRepository)(nil)
