//go:build postgres

package oversight

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	refreshInterval = 5 * time.Minute
	retentionPeriod = 90 * 24 * time.Hour
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Latest(ctx context.Context, scope Scope) (Snapshot, error) {
	var value Snapshot
	var payload, highWater []byte
	err := r.pool.QueryRow(ctx, `
		SELECT os.generated_at,os.period_start,os.period_end,os.projection_version,os.source_high_water,
		       os.coverage_population,os.coverage_excluded,os.coverage_unknown,os.payload,
		       t.slug,le.code
		FROM oversight_snapshots os
		JOIN tenants t ON t.id=os.tenant_id
		JOIN legal_entities le ON le.tenant_id=os.tenant_id AND le.id=os.legal_entity_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2)
		ORDER BY os.generated_at DESC,os.id DESC LIMIT 1`, scope.TenantID, scope.LegalEntityID).
		Scan(&value.GeneratedAt, &value.PeriodStart, &value.PeriodEnd, &value.ProjectionVersion, &highWater,
			&value.Coverage.Population, &value.Coverage.Excluded, &value.Coverage.Unknown, &payload,
			&value.TenantID, &value.LegalEntityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Snapshot{}, ErrNotFound
	}
	if err != nil {
		return Snapshot{}, err
	}
	if err := json.Unmarshal(highWater, &value.SourceHighWater); err != nil {
		return Snapshot{}, fmt.Errorf("decode oversight high-water marks: %w", err)
	}
	metadata := struct {
		Counts         Counts               `json:"counts"`
		Interventions  []Intervention       `json:"interventions"`
		Pressure       []CategoryPressure   `json:"pressure"`
		Aging          []AgingBucket        `json:"aging"`
		Performance    []Performance        `json:"performance"`
		Estimates      []ResolutionEstimate `json:"estimates"`
		HistoryQuality HistoryQuality       `json:"history_quality"`
	}{Counts: value.Counts}
	if err := json.Unmarshal(payload, &metadata); err != nil {
		return Snapshot{}, fmt.Errorf("decode oversight snapshot: %w", err)
	}
	value.Counts, value.Interventions, value.Pressure = metadata.Counts, metadata.Interventions, metadata.Pressure
	value.Aging, value.Performance, value.Estimates = metadata.Aging, metadata.Performance, metadata.Estimates
	value.HistoryQuality = metadata.HistoryQuality
	return value, nil
}

type Maintainer struct {
	Repository *PostgresRepository
}

func (m *Maintainer) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if m == nil || m.Repository == nil || m.Repository.pool == nil {
		return 0, ErrInvalid
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	now = now.UTC()
	rows, err := m.Repository.pool.Query(ctx, `
		SELECT t.id::text,le.id::text
		FROM legal_entities le JOIN tenants t ON t.id=le.tenant_id
		WHERE le.valid_from<=$1::timestamptz AND (le.valid_until IS NULL OR $1::timestamptz<le.valid_until)
		  AND NOT EXISTS (
		    SELECT 1 FROM oversight_snapshots os
		    WHERE os.tenant_id=le.tenant_id AND os.legal_entity_id=le.id AND os.projection_version=$2 AND os.generated_at>$1::timestamptz-interval '5 minutes'
		  )
		ORDER BY le.id LIMIT $3`, now, ProjectionVersion, limit)
	if err != nil {
		return 0, err
	}
	var scopes []Scope
	for rows.Next() {
		var scope Scope
		if err := rows.Scan(&scope.TenantID, &scope.LegalEntityID); err != nil {
			rows.Close()
			return 0, err
		}
		scopes = append(scopes, scope)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	completed := 0
	for _, scope := range scopes {
		if ctx.Err() != nil {
			return completed, ctx.Err()
		}
		snapshot, err := m.Repository.build(ctx, scope, now)
		if err != nil {
			return completed, err
		}
		inserted, err := m.Repository.store(ctx, snapshot, now.Truncate(refreshInterval))
		if err != nil {
			return completed, err
		}
		if inserted {
			completed++
		}
	}
	_, cleanupErr := m.Repository.pool.Exec(ctx, `DELETE FROM oversight_snapshots WHERE generated_at<$1`, now.Add(-retentionPeriod))
	return completed, cleanupErr
}

func (r *PostgresRepository) build(ctx context.Context, scope Scope, now time.Time) (Snapshot, error) {
	periodStart := now.Add(-90 * 24 * time.Hour)
	value := Snapshot{
		TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, GeneratedAt: now,
		PeriodStart: periodStart, PeriodEnd: now, ProjectionVersion: ProjectionVersion,
		SourceHighWater: map[string]time.Time{},
		Interventions:   []Intervention{}, Pressure: []CategoryPressure{}, Aging: []AgingBucket{}, Performance: []Performance{}, Estimates: []ResolutionEstimate{},
	}
	var excluded, unknown int
	err := r.pool.QueryRow(ctx, `
		WITH scoped AS (
		  SELECT matters.*,
		         CASE
		           WHEN NOT (scope ? 'access') OR upper(btrim(scope->>'access')) IN ('PUBLIC','INTERNAL') THEN 'INCLUDED'
		           WHEN upper(btrim(scope->>'access'))='RESTRICTED' THEN 'EXCLUDED'
		           ELSE 'UNKNOWN'
		         END scope_state
		  FROM matters WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid
		)
		SELECT count(*),
		       count(*) FILTER (WHERE scope_state='EXCLUDED'),
		       count(*) FILTER (WHERE scope_state='UNKNOWN'),
		       count(*) FILTER (WHERE scope_state='INCLUDED' AND status NOT IN ('CLOSED','CANCELLED') AND priority>=4),
		       count(*) FILTER (WHERE scope_state='INCLUDED' AND status NOT IN ('CLOSED','CANCELLED') AND due_at<$3::timestamptz),
		       count(*) FILTER (WHERE scope_state='INCLUDED' AND status NOT IN ('CLOSED','CANCELLED') AND due_at>=$3::timestamptz AND due_at<$3::timestamptz+interval '7 days'),
		       count(*) FILTER (WHERE scope_state='INCLUDED' AND status NOT IN ('CLOSED','CANCELLED') AND owner_principal_id IS NULL),
		       count(*) FILTER (WHERE scope_state='INCLUDED' AND status NOT IN ('CLOSED','CANCELLED') AND EXISTS (
		         SELECT 1 FROM verification_results vr WHERE vr.tenant_id=matters.tenant_id AND vr.matter_id=matters.id AND vr.result IN ('FAIL','INCONCLUSIVE')
		           AND vr.observed_at=(SELECT max(latest.observed_at) FROM verification_results latest WHERE latest.tenant_id=vr.tenant_id AND latest.matter_id=vr.matter_id AND latest.contract_id=vr.contract_id)
		       ))
		FROM scoped matters`, scope.TenantID, scope.LegalEntityID, now).
		Scan(&value.Coverage.Population, &excluded, &unknown, &value.Counts.CriticalHigh, &value.Counts.Overdue, &value.Counts.DueSoon, &value.Counts.Unassigned, &value.Counts.OutcomeFailures)
	if err != nil {
		return Snapshot{}, err
	}
	value.Coverage.Excluded, value.Coverage.Unknown = &excluded, &unknown
	if err := r.pool.QueryRow(ctx, `
		WITH closed AS (
		  SELECT m.id,
		         EXISTS (SELECT 1 FROM continuity_events ce WHERE ce.tenant_id=m.tenant_id AND ce.aggregate_type='MATTER' AND ce.aggregate_id=m.id AND ce.event_type='MATTER_CREATED') created_ok,
		         EXISTS (SELECT 1 FROM continuity_events ce WHERE ce.tenant_id=m.tenant_id AND ce.aggregate_type='MATTER' AND ce.aggregate_id=m.id AND ce.event_type='MATTER_STATE_CHANGED' AND ce.payload->>'status'='CLOSED') terminal_ok,
		         EXISTS (SELECT 1 FROM continuity_events ce WHERE ce.tenant_id=m.tenant_id AND ce.aggregate_type='MATTER' AND ce.aggregate_id=m.id AND ce.event_type='MATTER_OWNER_CHANGED') reassigned,
		         EXISTS (SELECT 1 FROM continuity_events ce WHERE ce.tenant_id=m.tenant_id AND ce.aggregate_type='MATTER' AND ce.aggregate_id=m.id AND ce.event_type='DECISION_ADDED' AND ce.payload->>'status'='RETURNED') returned,
		         EXISTS (SELECT 1 FROM continuity_events ce WHERE ce.tenant_id=m.tenant_id AND ce.aggregate_type='MATTER' AND ce.aggregate_id=m.id AND ce.event_type='ACTION_STATE_CHANGED' AND ce.payload->>'status'='BLOCKED') blocked,
		         m.reopen_count>0 reopened
		  FROM matters m WHERE m.tenant_id=$1::uuid AND m.legal_entity_id=$2::uuid AND m.status='CLOSED' AND m.closed_at>=$3
		    AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL'))
		)
		SELECT count(*),count(*) FILTER (WHERE created_ok AND terminal_ok),count(*) FILTER (WHERE NOT created_ok),count(*) FILTER (WHERE NOT terminal_ok),
		       count(*) FILTER (WHERE NOT (created_ok AND terminal_ok)),count(*) FILTER (WHERE created_ok AND terminal_ok AND reassigned),
		       count(*) FILTER (WHERE created_ok AND terminal_ok AND returned),count(*) FILTER (WHERE created_ok AND terminal_ok AND blocked),
		       count(*) FILTER (WHERE created_ok AND terminal_ok AND reopened)
		FROM closed`, scope.TenantID, scope.LegalEntityID, periodStart).Scan(
		&value.HistoryQuality.CompletedPopulation, &value.HistoryQuality.CompleteLifecycle, &value.HistoryQuality.MissingCreatedEvent,
		&value.HistoryQuality.MissingTerminalEvent, &value.HistoryQuality.ExcludedFromDurations, &value.HistoryQuality.ReassignedOwnerExcluded,
		&value.HistoryQuality.ReturnedOwnerExcluded, &value.HistoryQuality.BlockedOwnerExcluded, &value.HistoryQuality.ReopenedOwnerExcluded,
	); err != nil {
		return Snapshot{}, err
	}
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM workflow_tasks wt JOIN workflow_instances wi ON wi.tenant_id=wt.tenant_id AND wi.id=wt.workflow_id
		LEFT JOIN matters m ON wi.subject_type='MATTER' AND m.tenant_id=wi.tenant_id AND m.id=wi.subject_id
		LEFT JOIN programs p ON wi.subject_type='PROGRAM' AND p.tenant_id=wi.tenant_id AND p.id=wi.subject_id
		WHERE wt.tenant_id=$1::uuid AND wt.status IN ('READY','BLOCKED','ESCALATED') AND wt.principal_id IS NULL
		  AND COALESCE(m.legal_entity_id,p.legal_entity_id)=$2::uuid
		  AND ((m.id IS NOT NULL AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL')))
		    OR (p.id IS NOT NULL AND (NOT (p.scope ? 'access') OR upper(btrim(p.scope->>'access')) IN ('PUBLIC','INTERNAL'))))`, scope.TenantID, scope.LegalEntityID).Scan(&value.Counts.RoutingFailures); err != nil {
		return Snapshot{}, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT m.id::text,m.title,m.matter_type,m.status,m.priority,COALESCE(m.owner_principal_id::text,''),COALESCE(p.display_name,''),m.due_at
		FROM matters m LEFT JOIN principals p ON p.tenant_id=m.tenant_id AND p.id=m.owner_principal_id
		WHERE m.tenant_id=$1::uuid AND m.legal_entity_id=$2::uuid AND m.status NOT IN ('CLOSED','CANCELLED')
		  AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL'))
		  AND (m.priority>=4 OR m.due_at<$3::timestamptz OR m.owner_principal_id IS NULL)
		ORDER BY (m.due_at<$3::timestamptz) DESC,m.priority DESC,m.due_at NULLS LAST,m.updated_at DESC,m.id DESC LIMIT 30`, scope.TenantID, scope.LegalEntityID, now)
	if err != nil {
		return Snapshot{}, err
	}
	for rows.Next() {
		var item Intervention
		if err := rows.Scan(&item.TargetID, &item.Title, &item.Category, &item.State, &item.Priority, &item.OwnerID, &item.OwnerName, &item.DueAt); err != nil {
			rows.Close()
			return Snapshot{}, err
		}
		item.TargetType = "MATTER"
		item.Reason, item.NextAction = interventionCopy(item, now)
		value.Interventions = append(value.Interventions, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Snapshot{}, err
	}
	rows.Close()

	rows, err = r.pool.Query(ctx, `
		SELECT matter_type,count(*) FILTER (WHERE priority=5),count(*) FILTER (WHERE priority=4),count(*) FILTER (WHERE priority<4),count(*) FILTER (WHERE due_at<$3::timestamptz)
		FROM matters WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND status NOT IN ('CLOSED','CANCELLED')
		  AND (NOT (scope ? 'access') OR upper(btrim(scope->>'access')) IN ('PUBLIC','INTERNAL'))
		GROUP BY matter_type ORDER BY count(*) DESC,matter_type LIMIT 30`, scope.TenantID, scope.LegalEntityID, now)
	if err != nil {
		return Snapshot{}, err
	}
	for rows.Next() {
		var item CategoryPressure
		if err := rows.Scan(&item.Category, &item.Critical, &item.High, &item.Other, &item.Overdue); err != nil {
			rows.Close()
			return Snapshot{}, err
		}
		value.Pressure = append(value.Pressure, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Snapshot{}, err
	}
	rows.Close()

	var age0, age8, age31, age91 int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE created_at>=$3::timestamptz-interval '7 days'),
		       count(*) FILTER (WHERE created_at<$3::timestamptz-interval '7 days' AND created_at>=$3::timestamptz-interval '30 days'),
		       count(*) FILTER (WHERE created_at<$3::timestamptz-interval '30 days' AND created_at>=$3::timestamptz-interval '90 days'),
		       count(*) FILTER (WHERE created_at<$3::timestamptz-interval '90 days')
		FROM matters WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND status NOT IN ('CLOSED','CANCELLED')
		  AND (NOT (scope ? 'access') OR upper(btrim(scope->>'access')) IN ('PUBLIC','INTERNAL'))`, scope.TenantID, scope.LegalEntityID, now).
		Scan(&age0, &age8, &age31, &age91); err != nil {
		return Snapshot{}, err
	}
	value.Aging = []AgingBucket{{Label: "0–7 days", Count: age0}, {Label: "8–30 days", Count: age8}, {Label: "31–90 days", Count: age31}, {Label: "Over 90 days", Count: age91}}

	rows, err = r.pool.Query(ctx, `
		WITH lifecycle AS (
		  SELECT m.id,
		         min(ce.occurred_at) FILTER (WHERE ce.event_type='MATTER_CREATED') started_at,
		         max(ce.occurred_at) FILTER (WHERE ce.event_type='MATTER_STATE_CHANGED' AND ce.payload->>'status'='CLOSED') finished_at,
		         count(*) FILTER (WHERE ce.event_type='MATTER_OWNER_CHANGED') owner_changes,
		         count(*) FILTER (WHERE ce.event_type='DECISION_ADDED' AND ce.payload->>'status'='RETURNED') returns,
		         count(*) FILTER (WHERE ce.event_type='ACTION_STATE_CHANGED' AND ce.payload->>'status'='BLOCKED') blocks,
		         max(m.reopen_count) reopen_count
		  FROM matters m LEFT JOIN continuity_events ce ON ce.tenant_id=m.tenant_id AND ce.aggregate_type='MATTER' AND ce.aggregate_id=m.id
		  WHERE m.tenant_id=$1::uuid AND m.legal_entity_id=$2::uuid GROUP BY m.id
		), owner_work AS (
		  SELECT m.owner_principal_id,count(*) FILTER (WHERE m.status NOT IN ('CLOSED','CANCELLED')) current_load,
		         count(*) FILTER (WHERE m.status='CLOSED' AND m.closed_at>=$3) completed,
		         percentile_cont(.5) WITHIN GROUP (ORDER BY extract(epoch FROM (l.finished_at-l.started_at))/3600.0) FILTER (WHERE m.status='CLOSED' AND m.closed_at>=$3 AND l.started_at IS NOT NULL AND l.finished_at IS NOT NULL AND l.owner_changes=0 AND l.returns=0 AND l.blocks=0 AND l.reopen_count=0) median_hours,
		         percentile_cont(.75) WITHIN GROUP (ORDER BY extract(epoch FROM (l.finished_at-l.started_at))/3600.0) FILTER (WHERE m.status='CLOSED' AND m.closed_at>=$3 AND l.started_at IS NOT NULL AND l.finished_at IS NOT NULL AND l.owner_changes=0 AND l.returns=0 AND l.blocks=0 AND l.reopen_count=0) p75_hours,
		         count(*) FILTER (WHERE m.status='CLOSED' AND m.closed_at>=$3 AND m.due_at IS NOT NULL AND l.started_at IS NOT NULL AND l.finished_at IS NOT NULL AND l.owner_changes=0 AND l.returns=0 AND l.blocks=0 AND l.reopen_count=0) sla_samples,
		         count(*) FILTER (WHERE m.status='CLOSED' AND m.closed_at>=$3 AND m.due_at IS NOT NULL AND l.finished_at<=m.due_at AND l.started_at IS NOT NULL AND l.owner_changes=0 AND l.returns=0 AND l.blocks=0 AND l.reopen_count=0) sla_met,
		         count(*) FILTER (WHERE m.status='CLOSED' AND m.closed_at>=$3 AND l.started_at IS NOT NULL AND l.finished_at IS NOT NULL AND l.owner_changes=0 AND l.returns=0 AND l.blocks=0 AND l.reopen_count=0) measurement_samples,
		         sum(m.reopen_count) FILTER (WHERE m.updated_at>=$3) reopened
		  FROM matters m JOIN lifecycle l ON l.id=m.id WHERE m.tenant_id=$1::uuid AND m.legal_entity_id=$2::uuid AND m.owner_principal_id IS NOT NULL
		    AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL')) GROUP BY m.owner_principal_id
		), blocked AS (
		  SELECT ma.owner_principal_id,count(*) blocked FROM matter_actions ma JOIN matters m ON m.tenant_id=ma.tenant_id AND m.id=ma.matter_id
		  WHERE ma.tenant_id=$1::uuid AND m.legal_entity_id=$2::uuid AND ma.status='BLOCKED'
		    AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL')) GROUP BY ma.owner_principal_id
		), reassignments AS (
		  SELECT (e.payload->>'previous_owner_principal_id')::uuid owner_principal_id,count(*) reassigned
		  FROM matters m JOIN continuity_events e ON e.tenant_id=m.tenant_id AND e.aggregate_type='MATTER' AND e.aggregate_id=m.id
		  WHERE m.tenant_id=$1::uuid AND m.legal_entity_id=$2::uuid AND e.event_type='MATTER_OWNER_CHANGED' AND e.occurred_at>=$3
		    AND NULLIF(e.payload->>'previous_owner_principal_id','') IS NOT NULL
		    AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL'))
		  GROUP BY (e.payload->>'previous_owner_principal_id')::uuid
		), returned_decisions AS (
		  SELECT m.owner_principal_id,count(*) returned
		  FROM matters m JOIN continuity_events e ON e.tenant_id=m.tenant_id AND e.aggregate_type='MATTER' AND e.aggregate_id=m.id
		  WHERE m.tenant_id=$1::uuid AND m.legal_entity_id=$2::uuid AND e.event_type='DECISION_ADDED' AND e.occurred_at>=$3
		    AND e.payload->>'status'='RETURNED'
		    AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL'))
		  GROUP BY m.owner_principal_id
		)
		SELECT ow.owner_principal_id::text,p.display_name,ow.current_load,ow.completed,ow.median_hours,ow.p75_hours,ow.sla_samples,ow.sla_met,ow.measurement_samples,
		       COALESCE(b.blocked,0),COALESCE(ow.reopened,0),COALESCE(r.reassigned,0),COALESCE(rd.returned,0)
		FROM owner_work ow
		JOIN principals p ON p.tenant_id=$1::uuid AND p.id=ow.owner_principal_id
		LEFT JOIN blocked b ON b.owner_principal_id=ow.owner_principal_id
		LEFT JOIN reassignments r ON r.owner_principal_id=ow.owner_principal_id
		LEFT JOIN returned_decisions rd ON rd.owner_principal_id=ow.owner_principal_id
		ORDER BY ow.current_load DESC,ow.completed DESC,p.display_name LIMIT 50`, scope.TenantID, scope.LegalEntityID, periodStart)
	if err != nil {
		return Snapshot{}, err
	}
	for rows.Next() {
		var item Performance
		var median, p75 *float64
		var slaSamples, slaMet, reassigned, returned int
		if err := rows.Scan(&item.OwnerID, &item.OwnerName, &item.CurrentLoad, &item.Completed, &median, &p75, &slaSamples, &slaMet, &item.MeasurementSamples, &item.Blocked, &item.Reopened, &reassigned, &returned); err != nil {
			rows.Close()
			return Snapshot{}, err
		}
		item.MedianHours, item.P75Hours = median, p75
		item.Reassigned, item.Returned = &reassigned, &returned
		if slaSamples > 0 {
			rate := float64(slaMet) / float64(slaSamples)
			item.SLAAttainment = &rate
		}
		value.Performance = append(value.Performance, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Snapshot{}, err
	}
	rows.Close()

	// Replace whole-lifecycle/final-owner timing with exact assignment
	// intervals reconstructed from immutable owner-change events. This keeps
	// reassignment outside an employee's measured cycle once responsibility has
	// moved, while current load and completion counts remain separate measures.
	rows, err = r.pool.Query(ctx, `
		WITH lifecycle AS (
		  SELECT m.id,m.owner_principal_id,m.due_at,
		         min(e.occurred_at) FILTER (WHERE e.event_type='MATTER_CREATED') started_at,
		         max(e.occurred_at) FILTER (WHERE e.event_type='MATTER_STATE_CHANGED' AND e.payload->>'status'='CLOSED') finished_at
		  FROM matters m LEFT JOIN continuity_events e ON e.tenant_id=m.tenant_id AND e.aggregate_type='MATTER' AND e.aggregate_id=m.id
		  WHERE m.tenant_id=$1::uuid AND m.legal_entity_id=$2::uuid AND m.status='CLOSED' AND m.closed_at>=$3
		    AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL'))
		  GROUP BY m.id
		), changes AS (
		  SELECT e.aggregate_id matter_id,e.occurred_at,
		         NULLIF(e.payload->>'previous_owner_principal_id','')::uuid previous_owner_id,
		         NULLIF(e.payload->>'owner_principal_id','')::uuid owner_id,
		         lead(e.occurred_at) OVER (PARTITION BY e.aggregate_id ORDER BY e.occurred_at,e.id) next_change,
		         row_number() OVER (PARTITION BY e.aggregate_id ORDER BY e.occurred_at,e.id) sequence
		  FROM continuity_events e JOIN lifecycle l ON l.id=e.aggregate_id
		  WHERE e.tenant_id=$1::uuid AND e.aggregate_type='MATTER' AND e.event_type='MATTER_OWNER_CHANGED'
		), segments AS (
		  SELECT l.id matter_id,l.owner_principal_id owner_id,l.started_at segment_start,l.finished_at segment_end,l.finished_at,l.due_at
		  FROM lifecycle l WHERE NOT EXISTS (SELECT 1 FROM changes c WHERE c.matter_id=l.id)
		  UNION ALL
		  SELECT l.id,c.previous_owner_id,l.started_at,c.occurred_at,l.finished_at,l.due_at
		  FROM lifecycle l JOIN changes c ON c.matter_id=l.id AND c.sequence=1
		  UNION ALL
		  SELECT l.id,c.owner_id,c.occurred_at,COALESCE(c.next_change,l.finished_at),l.finished_at,l.due_at
		  FROM lifecycle l JOIN changes c ON c.matter_id=l.id
		), valid AS (
		  SELECT s.*,
		         (s.segment_end<s.finished_at)::int reassigned,
		         (SELECT count(*) FROM continuity_events e WHERE e.tenant_id=$1::uuid AND e.aggregate_type='MATTER' AND e.aggregate_id=s.matter_id
		            AND e.occurred_at>=s.segment_start AND e.occurred_at<s.segment_end
		            AND e.event_type='DECISION_ADDED' AND e.payload->>'status'='RETURNED') returned,
		         (SELECT count(*) FROM continuity_events e WHERE e.tenant_id=$1::uuid AND e.aggregate_type='MATTER' AND e.aggregate_id=s.matter_id
		            AND e.occurred_at>=s.segment_start AND e.occurred_at<s.segment_end
		            AND e.event_type='ACTION_STATE_CHANGED' AND e.payload->>'status'='BLOCKED') blocked,
		         (SELECT count(*) FROM continuity_events e WHERE e.tenant_id=$1::uuid AND e.aggregate_type='MATTER' AND e.aggregate_id=s.matter_id
		            AND e.occurred_at>=s.segment_start AND e.occurred_at<s.segment_end
		            AND e.event_type='MATTER_STATE_CHANGED' AND e.payload->>'status'='ASSESSMENT'
		            AND COALESCE((e.payload->>'reopen_count')::int,0)>0) reopened
		  FROM segments s WHERE s.owner_id IS NOT NULL AND s.segment_start IS NOT NULL AND s.segment_end IS NOT NULL AND s.segment_end>s.segment_start
		)
		SELECT v.owner_id::text,p.display_name,
		       percentile_cont(.5) WITHIN GROUP (ORDER BY extract(epoch FROM (segment_end-segment_start))/3600.0),
		       percentile_cont(.75) WITHIN GROUP (ORDER BY extract(epoch FROM (segment_end-segment_start))/3600.0),
		       count(*),
		       count(*) FILTER (WHERE segment_end=finished_at AND due_at IS NOT NULL),
		       count(*) FILTER (WHERE segment_end=finished_at AND due_at IS NOT NULL AND finished_at<=due_at),
		       sum(reassigned),sum(returned),sum(blocked),sum(reopened)
		FROM valid v JOIN principals p ON p.tenant_id=$1::uuid AND p.id=v.owner_id
		GROUP BY v.owner_id,p.display_name`, scope.TenantID, scope.LegalEntityID, periodStart)
	if err != nil {
		return Snapshot{}, err
	}
	performanceByOwner := make(map[string]int, len(value.Performance))
	for index := range value.Performance {
		performanceByOwner[value.Performance[index].OwnerID] = index
	}
	for rows.Next() {
		var ownerID, ownerName string
		var median, p75 float64
		var samples, slaSamples, slaMet, reassigned, returned, blocked, reopened int
		if err := rows.Scan(&ownerID, &ownerName, &median, &p75, &samples, &slaSamples, &slaMet, &reassigned, &returned, &blocked, &reopened); err != nil {
			rows.Close()
			return Snapshot{}, err
		}
		index, ok := performanceByOwner[ownerID]
		if !ok {
			value.Performance = append(value.Performance, Performance{OwnerID: ownerID, OwnerName: ownerName})
			index = len(value.Performance) - 1
			performanceByOwner[ownerID] = index
		}
		item := &value.Performance[index]
		item.MedianHours, item.P75Hours, item.MeasurementSamples = &median, &p75, samples
		item.Reassigned, item.Returned = &reassigned, &returned
		item.Blocked, item.Reopened = blocked, reopened
		item.SLAAttainment = nil
		if slaSamples > 0 {
			rate := float64(slaMet) / float64(slaSamples)
			item.SLAAttainment = &rate
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Snapshot{}, err
	}
	rows.Close()
	value.HistoryQuality.ReassignedOwnerExcluded = 0
	value.HistoryQuality.ReturnedOwnerExcluded = 0
	value.HistoryQuality.BlockedOwnerExcluded = 0
	value.HistoryQuality.ReopenedOwnerExcluded = 0

	rows, err = r.pool.Query(ctx, `
		WITH lifecycle AS (
		  SELECT m.id,m.matter_type,m.scope,m.closed_at,
		         min(ce.occurred_at) FILTER (WHERE ce.event_type='MATTER_CREATED') started_at,
		         max(ce.occurred_at) FILTER (WHERE ce.event_type='MATTER_STATE_CHANGED' AND ce.payload->>'status'='CLOSED') finished_at
		  FROM matters m LEFT JOIN continuity_events ce ON ce.tenant_id=m.tenant_id AND ce.aggregate_type='MATTER' AND ce.aggregate_id=m.id
		  WHERE m.tenant_id=$1::uuid AND m.legal_entity_id=$2::uuid AND m.status='CLOSED' AND m.closed_at>=$3 GROUP BY m.id
		)
		SELECT matter_type,count(*),percentile_cont(.5) WITHIN GROUP (ORDER BY extract(epoch FROM (finished_at-started_at))/3600.0),
		       percentile_cont(.25) WITHIN GROUP (ORDER BY extract(epoch FROM (finished_at-started_at))/3600.0),
		       percentile_cont(.75) WITHIN GROUP (ORDER BY extract(epoch FROM (finished_at-started_at))/3600.0)
		FROM lifecycle WHERE started_at IS NOT NULL AND finished_at IS NOT NULL
		  AND (NOT (scope ? 'access') OR upper(btrim(scope->>'access')) IN ('PUBLIC','INTERNAL'))
		GROUP BY matter_type HAVING count(*)>=5 ORDER BY count(*) DESC,matter_type LIMIT 30`, scope.TenantID, scope.LegalEntityID, periodStart)
	if err != nil {
		return Snapshot{}, err
	}
	for rows.Next() {
		var item ResolutionEstimate
		if err := rows.Scan(&item.Category, &item.SampleSize, &item.MedianHours, &item.LowerHours, &item.UpperHours); err != nil {
			rows.Close()
			return Snapshot{}, err
		}
		item.Confidence, item.EstimatedBy = estimateConfidence(item.SampleSize), "Closed issues of the same type in this legal entity during the selected period"
		value.Estimates = append(value.Estimates, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Snapshot{}, err
	}
	rows.Close()

	var matterHW, actionHW, taskHW, verificationHW, eventHW *time.Time
	if err := r.pool.QueryRow(ctx, `
		SELECT max(m.updated_at),
		       (SELECT max(ma.updated_at) FROM matter_actions ma JOIN matters linked ON linked.tenant_id=ma.tenant_id AND linked.id=ma.matter_id WHERE linked.tenant_id=$1::uuid AND linked.legal_entity_id=$2::uuid AND (NOT (linked.scope ? 'access') OR upper(btrim(linked.scope->>'access')) IN ('PUBLIC','INTERNAL'))),
		       (SELECT max(wt.updated_at) FROM workflow_tasks wt JOIN workflow_instances wi ON wi.tenant_id=wt.tenant_id AND wi.id=wt.workflow_id LEFT JOIN matters linked_m ON wi.subject_type='MATTER' AND linked_m.tenant_id=wi.tenant_id AND linked_m.id=wi.subject_id LEFT JOIN programs linked_p ON wi.subject_type='PROGRAM' AND linked_p.tenant_id=wi.tenant_id AND linked_p.id=wi.subject_id WHERE wt.tenant_id=$1::uuid AND COALESCE(linked_m.legal_entity_id,linked_p.legal_entity_id)=$2::uuid AND ((linked_m.id IS NOT NULL AND (NOT (linked_m.scope ? 'access') OR upper(btrim(linked_m.scope->>'access')) IN ('PUBLIC','INTERNAL'))) OR (linked_p.id IS NOT NULL AND (NOT (linked_p.scope ? 'access') OR upper(btrim(linked_p.scope->>'access')) IN ('PUBLIC','INTERNAL'))))),
		       (SELECT max(vr.observed_at) FROM verification_results vr JOIN matters linked ON linked.tenant_id=vr.tenant_id AND linked.id=vr.matter_id WHERE linked.tenant_id=$1::uuid AND linked.legal_entity_id=$2::uuid AND (NOT (linked.scope ? 'access') OR upper(btrim(linked.scope->>'access')) IN ('PUBLIC','INTERNAL')))
		       ,(SELECT max(ce.occurred_at) FROM continuity_events ce JOIN matters linked ON linked.tenant_id=ce.tenant_id AND linked.id=ce.aggregate_id WHERE ce.aggregate_type='MATTER' AND linked.tenant_id=$1::uuid AND linked.legal_entity_id=$2::uuid AND (NOT (linked.scope ? 'access') OR upper(btrim(linked.scope->>'access')) IN ('PUBLIC','INTERNAL')))
		FROM matters m WHERE m.tenant_id=$1::uuid AND m.legal_entity_id=$2::uuid
		  AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL'))`, scope.TenantID, scope.LegalEntityID).
		Scan(&matterHW, &actionHW, &taskHW, &verificationHW, &eventHW); err != nil {
		return Snapshot{}, err
	}
	for key, watermark := range map[string]*time.Time{"matters": matterHW, "actions": actionHW, "workflow_tasks": taskHW, "verification_results": verificationHW, "continuity_events": eventHW} {
		if watermark != nil {
			value.SourceHighWater[key] = watermark.UTC()
		}
	}
	return value, nil
}

func (r *PostgresRepository) store(ctx context.Context, value Snapshot, slot time.Time) (bool, error) {
	payload, err := json.Marshal(struct {
		Counts         Counts               `json:"counts"`
		Interventions  []Intervention       `json:"interventions"`
		Pressure       []CategoryPressure   `json:"pressure"`
		Aging          []AgingBucket        `json:"aging"`
		Performance    []Performance        `json:"performance"`
		Estimates      []ResolutionEstimate `json:"estimates"`
		HistoryQuality HistoryQuality       `json:"history_quality"`
	}{value.Counts, value.Interventions, value.Pressure, value.Aging, value.Performance, value.Estimates, value.HistoryQuality})
	if err != nil {
		return false, err
	}
	highWater, err := json.Marshal(value.SourceHighWater)
	if err != nil {
		return false, err
	}
	command, err := r.pool.Exec(ctx, `
		INSERT INTO oversight_snapshots(tenant_id,legal_entity_id,period_start,period_end,refresh_slot,generated_at,projection_version,source_high_water,coverage_population,coverage_excluded,coverage_unknown,payload)
		VALUES($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8::jsonb,$9,$10,$11,$12::jsonb)
		ON CONFLICT(tenant_id,legal_entity_id,projection_version,refresh_slot) DO NOTHING`, value.TenantID, value.LegalEntityID, value.PeriodStart, value.PeriodEnd, slot, value.GeneratedAt, value.ProjectionVersion, highWater, value.Coverage.Population, value.Coverage.Excluded, value.Coverage.Unknown, payload)
	return command.RowsAffected() == 1, err
}
