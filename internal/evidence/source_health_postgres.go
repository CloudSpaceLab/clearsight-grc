//go:build postgres

package evidence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) RecordScopedSourceObservation(ctx context.Context, observation SourceObservation, evaluatedAt time.Time) (Source, error) {
	observation, err := normalizeSourceObservationScope(observation)
	if err != nil {
		return Source{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Source{}, err
	}
	defer tx.Rollback(ctx)
	current, err := scanSource(tx.QueryRow(ctx, `
		SELECT es.id::text,t.id::text,COALESCE(es.legal_entity_id::text,''),es.code,es.name,es.source_type,
		       es.authority_class,COALESCE(es.owner_principal_id::text,''),es.expected_freshness_minutes,
		       es.last_observed_at,es.last_success_at,es.health,es.status,es.version,es.created_at,es.updated_at
		  FROM evidence_sources es JOIN tenants t ON t.id=es.tenant_id
		 WHERE es.id=$1::uuid AND (t.id::text=$2 OR t.slug=$2)
		 FOR UPDATE OF es`, observation.SourceID, observation.TenantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Source{}, ErrNotFound
	}
	if err != nil {
		return Source{}, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO source_observations(
			id,tenant_id,source_id,observed_at,success,unavailable,latency_ms,detail,recorded_by,
			scope_kind,connection_id,connection_version,view_id,view_version,binding_id,binding_version
		) VALUES (
			$1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,NULLIF($9,'')::uuid,
			$10,NULLIF($11,'')::uuid,NULLIF($12,0),NULLIF($13,'')::uuid,NULLIF($14,0),NULLIF($15,'')::uuid,NULLIF($16,0)
		)`, observation.ID, observation.TenantID, observation.SourceID, observation.ObservedAt,
		observation.Success, observation.Unavailable, observation.LatencyMS, observation.Detail, observation.RecordedBy,
		observation.Scope, observation.ConnectionID, observation.ConnectionVersion, observation.ViewID, observation.ViewVersion,
		observation.BindingID, observation.BindingVersion)
	if err != nil {
		return Source{}, fmt.Errorf("record scoped source observation: %w", err)
	}
	health, lastObserved, lastSuccess, err := aggregateSourceHealthTx(ctx, tx, current, evaluatedAt)
	if err != nil {
		return Source{}, err
	}
	updated, err := scanSource(tx.QueryRow(ctx, `
		UPDATE evidence_sources
		   SET last_observed_at=$3,last_success_at=$4,health=$5,version=version+1,updated_at=$6
		 WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2)
		 RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),COALESCE(legal_entity_id::text,''),code,name,
		           source_type,authority_class,COALESCE(owner_principal_id::text,''),expected_freshness_minutes,
		           last_observed_at,last_success_at,health,status,version,created_at,updated_at`,
		observation.SourceID, observation.TenantID, nullableTime(lastObserved), nullableTime(lastSuccess), health, evaluatedAt))
	if err != nil {
		return Source{}, err
	}
	if current.Health != updated.Health {
		if err := insertSourceHealthChanged(ctx, tx, observation.TenantID, observation.SourceID, current.Health, updated.Health, updated.Code, evaluatedAt); err != nil {
			return Source{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Source{}, err
	}
	return updated, nil
}

func (r *PostgresRepository) EvaluateScopedSourceHealth(ctx context.Context, now time.Time, limit int) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		SELECT es.id::text,t.id::text,COALESCE(es.legal_entity_id::text,''),es.code,es.name,es.source_type,
		       es.authority_class,COALESCE(es.owner_principal_id::text,''),es.expected_freshness_minutes,
		       es.last_observed_at,es.last_success_at,es.health,es.status,es.version,es.created_at,es.updated_at
		  FROM evidence_sources es JOIN tenants t ON t.id=es.tenant_id
		 WHERE es.status='ACTIVE'
		 ORDER BY es.id
		 LIMIT $1
		 FOR UPDATE OF es SKIP LOCKED`, limit)
	if err != nil {
		return 0, err
	}
	sources := make([]Source, 0, limit)
	for rows.Next() {
		value, scanErr := scanSource(rows)
		if scanErr != nil {
			rows.Close()
			return 0, scanErr
		}
		sources = append(sources, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	changed := 0
	for _, source := range sources {
		health, lastObserved, lastSuccess, aggregateErr := aggregateSourceHealthTx(ctx, tx, source, now)
		if aggregateErr != nil {
			return changed, aggregateErr
		}
		if health == source.Health {
			continue
		}
		if _, err := tx.Exec(ctx, `
			UPDATE evidence_sources
			   SET health=$3,last_observed_at=$4,last_success_at=$5,version=version+1,updated_at=$6
			 WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2)`,
			source.ID, source.TenantID, health, nullableTime(lastObserved), nullableTime(lastSuccess), now); err != nil {
			return changed, err
		}
		if err := insertSourceHealthChanged(ctx, tx, source.TenantID, source.ID, source.Health, health, source.Code, now); err != nil {
			return changed, err
		}
		changed++
	}
	if err := tx.Commit(ctx); err != nil {
		return changed, err
	}
	return changed, nil
}

func (r *PostgresRepository) ListSourceScopeHealth(ctx context.Context, tenantID, sourceID string, now time.Time, limit int) ([]SourceScopeHealth, error) {
	source, err := scanSource(r.pool.QueryRow(ctx, `
		SELECT es.id::text,t.id::text,COALESCE(es.legal_entity_id::text,''),es.code,es.name,es.source_type,
		       es.authority_class,COALESCE(es.owner_principal_id::text,''),es.expected_freshness_minutes,
		       es.last_observed_at,es.last_success_at,es.health,es.status,es.version,es.created_at,es.updated_at
		  FROM evidence_sources es JOIN tenants t ON t.id=es.tenant_id
		 WHERE es.id=$1::uuid AND (t.id::text=$2 OR t.slug=$2)`, sourceID, tenantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		WITH ranked AS (
			SELECT so.scope_kind,COALESCE(so.connection_id::text,''),COALESCE(so.connection_version,0),
			       COALESCE(so.view_id::text,''),COALESCE(so.view_version,0),
			       COALESCE(so.binding_id::text,''),COALESCE(so.binding_version,0),
			       so.observed_at,so.success,so.unavailable,so.latency_ms,
			       max(so.observed_at) FILTER (WHERE so.success) OVER (
			           PARTITION BY so.scope_kind,so.connection_id,so.connection_version,so.view_id,so.view_version,so.binding_id,so.binding_version
			       ) AS last_success_at,
			       row_number() OVER (
			           PARTITION BY so.scope_kind,so.connection_id,so.connection_version,so.view_id,so.view_version,so.binding_id,so.binding_version
			           ORDER BY so.observed_at DESC,so.id DESC
			       ) AS row_number
			  FROM source_observations so
			  JOIN tenants t ON t.id=so.tenant_id
			 WHERE (t.id::text=$1 OR t.slug=$1) AND so.source_id=$2::uuid
		)
		SELECT scope_kind,coalesce,coalesce_1,coalesce_2,coalesce_3,coalesce_4,coalesce_5,
		       observed_at,success,unavailable,latency_ms,last_success_at
		  FROM ranked
		 WHERE row_number=1
		 ORDER BY scope_kind,coalesce,coalesce_2,coalesce_4,observed_at DESC
		 LIMIT $3`, tenantID, sourceID, healthLimit(limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]SourceScopeHealth, 0)
	for rows.Next() {
		var observation SourceObservation
		var lastSuccess sql.NullTime
		if err := rows.Scan(
			&observation.Scope, &observation.ConnectionID, &observation.ConnectionVersion,
			&observation.ViewID, &observation.ViewVersion, &observation.BindingID, &observation.BindingVersion,
			&observation.ObservedAt, &observation.Success, &observation.Unavailable, &observation.LatencyMS, &lastSuccess,
		); err != nil {
			return nil, err
		}
		observation.SourceID = source.ID
		value := sourceScopeHealthFromObservation(source, observation, now)
		if lastSuccess.Valid {
			copy := lastSuccess.Time
			value.LastSuccessAt = &copy
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func aggregateSourceHealthTx(ctx context.Context, tx pgx.Tx, source Source, now time.Time) (SourceHealth, *time.Time, *time.Time, error) {
	var rank int
	var observed, success sql.NullTime
	err := tx.QueryRow(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (scope_kind,connection_id,connection_version,view_id,view_version,binding_id,binding_version)
			       observed_at,success,unavailable
			  FROM source_observations
			 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			   AND source_id=$2::uuid
			 ORDER BY scope_kind,connection_id,connection_version,view_id,view_version,binding_id,binding_version,observed_at DESC,id DESC
		), history AS (
			SELECT max(observed_at) AS last_observed_at,
			       max(observed_at) FILTER (WHERE success) AS last_success_at
			  FROM source_observations
			 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			   AND source_id=$2::uuid
		)
		SELECT COALESCE(max(CASE
		           WHEN unavailable THEN 5
		           WHEN NOT success THEN 3
		           WHEN observed_at + make_interval(mins=>$4) <= $3 THEN 4
		           ELSE 1
		       END),0),history.last_observed_at,history.last_success_at
		  FROM latest CROSS JOIN history
		 GROUP BY history.last_observed_at,history.last_success_at`, source.TenantID, source.ID, now, source.ExpectedFreshnessMinutes).Scan(&rank, &observed, &success)
	if err != nil {
		return HealthUnknown, nil, nil, err
	}
	lastObserved := laterNullable(source.LastObservedAt, observed)
	lastSuccess := laterNullable(source.LastSuccessAt, success)
	if rank == 0 {
		if source.Health == HealthUnavailable {
			return HealthUnavailable, lastObserved, lastSuccess, nil
		}
		if lastSuccess != nil && !now.Before(lastSuccess.Add(time.Duration(source.ExpectedFreshnessMinutes)*time.Minute)) {
			return HealthStale, lastObserved, lastSuccess, nil
		}
		return source.Health, lastObserved, lastSuccess, nil
	}
	return sourceHealthFromRank(rank), lastObserved, lastSuccess, nil
}

func laterNullable(existing *time.Time, candidate sql.NullTime) *time.Time {
	var result *time.Time
	if existing != nil {
		copy := *existing
		result = &copy
	}
	if candidate.Valid && (result == nil || candidate.Time.After(*result)) {
		copy := candidate.Time
		result = &copy
	}
	return result
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func insertSourceHealthChanged(ctx context.Context, tx pgx.Tx, tenantID, sourceID string, from, to SourceHealth, code string, at time.Time) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'EVIDENCE_SOURCE',$2::uuid,'SourceHealthChanged',
		       jsonb_build_object('from',$3::text,'to',$4::text,'source_code',$5::text),$6,$6,$6)`, tenantID, sourceID, from, to, code, at)
	return err
}
