//go:build postgres

package monitoring

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *PostgresRepository) OpenOrUpdateAdverseEpisode(ctx context.Context, observation AdverseEpisodeObservation, episodeID string, now time.Time) (AdverseEpisode, AdverseEpisodeChange, error) {
	if r == nil || r.pool == nil || episodeID == "" {
		return AdverseEpisode{}, AdverseEpisodeNoChange, ErrInvalid
	}
	if err := validateEpisodeObservation(observation); err != nil {
		return AdverseEpisode{}, AdverseEpisodeNoChange, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return AdverseEpisode{}, AdverseEpisodeNoChange, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, err := scanAdverseEpisode(tx.QueryRow(ctx, adverseEpisodeSelect+`
		WHERE e.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND e.legal_entity_id=(SELECT id FROM legal_entities WHERE tenant_id=e.tenant_id AND (id::text=$2 OR code=$2))
		  AND e.monitoring_check_id=$3::uuid
		  AND e.state='OPEN'
		FOR UPDATE`, observation.TenantID, observation.LegalEntityID, observation.Check.ID))
	if errors.Is(err, pgx.ErrNoRows) {
		created, createErr := scanAdverseEpisode(tx.QueryRow(ctx, `
			INSERT INTO monitoring_adverse_episodes(
				id,tenant_id,legal_entity_id,program_id,monitoring_check_id,state,
				first_result_id,last_result_id,last_check_version,last_band,last_score,last_coverage,
				opened_at,updated_at,record_version)
			SELECT $3::uuid,t.id,le.id,$4::uuid,$5::uuid,'OPEN',
			       $6::uuid,$6::uuid,$7::bigint,$8::text,$9,$10,$11::timestamptz,$12::timestamptz,1
			FROM tenants t
			JOIN legal_entities le ON le.tenant_id=t.id
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND (le.id::text=$2 OR le.code=$2)
			RETURNING id::text,tenant_id::text,legal_entity_id::text,program_id::text,monitoring_check_id::text,
			          state,COALESCE(matter_id::text,''),first_result_id::text,last_result_id::text,last_check_version,
			          last_band,last_score,last_coverage,opened_at,closed_at,updated_at,record_version`,
			observation.TenantID, observation.LegalEntityID, episodeID, observation.ProgramID,
			observation.Check.ID, observation.Result.ID, observation.Check.Version,
			observation.Result.Evaluation.Band, observation.Result.Evaluation.Score,
			observation.Result.Evaluation.Coverage, observation.Result.EvaluatedAt, now.UTC()))
		if createErr != nil {
			return AdverseEpisode{}, AdverseEpisodeNoChange, mapAdverseEpisodePostgresError(createErr)
		}
		event, eventErr := newMonitoringEvent(created.TenantID, AggregateMonitoringAdverseEpisode, created.ID, created.RecordVersion, EventMonitoringAdverseEpisodeOpened, episodeEventPayload(created), "", now)
		if eventErr != nil {
			return AdverseEpisode{}, AdverseEpisodeNoChange, eventErr
		}
		if eventErr := insertMonitoringEventAndOutbox(ctx, tx, event); eventErr != nil {
			return AdverseEpisode{}, AdverseEpisodeNoChange, eventErr
		}
		if err := tx.Commit(ctx); err != nil {
			return AdverseEpisode{}, AdverseEpisodeNoChange, mapPostgresError(err)
		}
		return created, AdverseEpisodeOpened, nil
	}
	if err != nil {
		return AdverseEpisode{}, AdverseEpisodeNoChange, mapAdverseEpisodePostgresError(err)
	}
	if current.ProgramID != observation.ProgramID {
		return AdverseEpisode{}, AdverseEpisodeNoChange, ErrInvalid
	}
	if current.LastResultID == observation.Result.ID {
		if err := tx.Commit(ctx); err != nil {
			return AdverseEpisode{}, AdverseEpisodeNoChange, mapPostgresError(err)
		}
		return current, AdverseEpisodeNoChange, nil
	}

	eventType := EventMonitoringAdverseEpisodeUpdated
	change := AdverseEpisodeUpdated
	if episodeWorsened(current, observation.Result) {
		eventType = EventMonitoringAdverseEpisodeWorsened
		change = AdverseEpisodeWorsened
	}
	updated, err := scanAdverseEpisode(tx.QueryRow(ctx, `
		UPDATE monitoring_adverse_episodes
		SET last_result_id=$4::uuid,last_check_version=$5::bigint,last_band=$6::text,
		    last_score=$7,last_coverage=$8,updated_at=$9::timestamptz,record_version=record_version+1
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND state='OPEN'
		RETURNING id::text,tenant_id::text,legal_entity_id::text,program_id::text,monitoring_check_id::text,
		          state,COALESCE(matter_id::text,''),first_result_id::text,last_result_id::text,last_check_version,
		          last_band,last_score,last_coverage,opened_at,closed_at,updated_at,record_version`,
		current.TenantID, current.LegalEntityID, current.ID, observation.Result.ID, observation.Check.Version,
		observation.Result.Evaluation.Band, observation.Result.Evaluation.Score, observation.Result.Evaluation.Coverage, now.UTC()))
	if err != nil {
		return AdverseEpisode{}, AdverseEpisodeNoChange, mapAdverseEpisodePostgresError(err)
	}
	event, err := newMonitoringEvent(updated.TenantID, AggregateMonitoringAdverseEpisode, updated.ID, updated.RecordVersion, eventType, episodeEventPayload(updated), "", now)
	if err != nil {
		return AdverseEpisode{}, AdverseEpisodeNoChange, err
	}
	if err := insertMonitoringEventAndOutbox(ctx, tx, event); err != nil {
		return AdverseEpisode{}, AdverseEpisodeNoChange, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AdverseEpisode{}, AdverseEpisodeNoChange, mapPostgresError(err)
	}
	return updated, change, nil
}

func (r *PostgresRepository) CloseAdverseEpisode(ctx context.Context, observation AdverseEpisodeObservation, now time.Time) (*AdverseEpisode, AdverseEpisodeChange, error) {
	if r == nil || r.pool == nil {
		return nil, AdverseEpisodeNoChange, ErrInvalid
	}
	if err := validateEpisodeObservation(observation); err != nil {
		return nil, AdverseEpisodeNoChange, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, AdverseEpisodeNoChange, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, err := scanAdverseEpisode(tx.QueryRow(ctx, adverseEpisodeSelect+`
		WHERE e.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND e.legal_entity_id=(SELECT id FROM legal_entities WHERE tenant_id=e.tenant_id AND (id::text=$2 OR code=$2))
		  AND e.monitoring_check_id=$3::uuid
		  AND e.state='OPEN'
		FOR UPDATE`, observation.TenantID, observation.LegalEntityID, observation.Check.ID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, AdverseEpisodeNoChange, nil
	}
	if err != nil {
		return nil, AdverseEpisodeNoChange, mapAdverseEpisodePostgresError(err)
	}
	if current.ProgramID != observation.ProgramID {
		return nil, AdverseEpisodeNoChange, ErrInvalid
	}
	closedAt := observation.Result.EvaluatedAt.UTC()
	updated, err := scanAdverseEpisode(tx.QueryRow(ctx, `
		UPDATE monitoring_adverse_episodes
		SET state='CLOSED',last_result_id=$4::uuid,last_check_version=$5::bigint,last_band=$6::text,
		    last_score=$7,last_coverage=$8,closed_at=$9::timestamptz,updated_at=$10::timestamptz,
		    record_version=record_version+1
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND state='OPEN'
		RETURNING id::text,tenant_id::text,legal_entity_id::text,program_id::text,monitoring_check_id::text,
		          state,COALESCE(matter_id::text,''),first_result_id::text,last_result_id::text,last_check_version,
		          last_band,last_score,last_coverage,opened_at,closed_at,updated_at,record_version`,
		current.TenantID, current.LegalEntityID, current.ID, observation.Result.ID, observation.Check.Version,
		observation.Result.Evaluation.Band, observation.Result.Evaluation.Score, observation.Result.Evaluation.Coverage,
		closedAt, now.UTC()))
	if err != nil {
		return nil, AdverseEpisodeNoChange, mapAdverseEpisodePostgresError(err)
	}
	event, err := newMonitoringEvent(updated.TenantID, AggregateMonitoringAdverseEpisode, updated.ID, updated.RecordVersion, EventMonitoringAdverseEpisodeCleared, episodeEventPayload(updated), "", now)
	if err != nil {
		return nil, AdverseEpisodeNoChange, err
	}
	if err := insertMonitoringEventAndOutbox(ctx, tx, event); err != nil {
		return nil, AdverseEpisodeNoChange, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, AdverseEpisodeNoChange, mapPostgresError(err)
	}
	return &updated, AdverseEpisodeCleared, nil
}

func (r *PostgresRepository) OpenAdverseEpisode(ctx context.Context, tenant, legalEntityID, checkID string) (AdverseEpisode, error) {
	if r == nil || r.pool == nil || tenant == "" || legalEntityID == "" || checkID == "" {
		return AdverseEpisode{}, ErrInvalid
	}
	value, err := scanAdverseEpisode(r.pool.QueryRow(ctx, adverseEpisodeSelect+`
		WHERE e.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND e.legal_entity_id=(SELECT id FROM legal_entities WHERE tenant_id=e.tenant_id AND (id::text=$2 OR code=$2))
		  AND e.monitoring_check_id=$3::uuid
		  AND e.state='OPEN'`, tenant, legalEntityID, checkID))
	return value, mapAdverseEpisodePostgresError(err)
}

func (r *PostgresRepository) AttachAdverseEpisodeMatter(ctx context.Context, tenant, legalEntityID, episodeID, matterID string, now time.Time) (AdverseEpisode, error) {
	if r == nil || r.pool == nil || tenant == "" || legalEntityID == "" || episodeID == "" || matterID == "" {
		return AdverseEpisode{}, ErrInvalid
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return AdverseEpisode{}, mapPostgresError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, err := scanAdverseEpisode(tx.QueryRow(ctx, adverseEpisodeSelect+`
		WHERE e.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND e.legal_entity_id=(SELECT id FROM legal_entities WHERE tenant_id=e.tenant_id AND (id::text=$2 OR code=$2))
		  AND e.id=$3::uuid
		FOR UPDATE`, tenant, legalEntityID, episodeID))
	if err != nil {
		return AdverseEpisode{}, mapAdverseEpisodePostgresError(err)
	}
	if current.MatterID != "" {
		if current.MatterID == matterID {
			if err := tx.Commit(ctx); err != nil {
				return AdverseEpisode{}, mapPostgresError(err)
			}
			return current, nil
		}
		return AdverseEpisode{}, ErrConflict
	}
	updated, err := scanAdverseEpisode(tx.QueryRow(ctx, `
		UPDATE monitoring_adverse_episodes
		SET matter_id=$4::uuid,updated_at=$5::timestamptz,record_version=record_version+1
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid
		RETURNING id::text,tenant_id::text,legal_entity_id::text,program_id::text,monitoring_check_id::text,
		          state,COALESCE(matter_id::text,''),first_result_id::text,last_result_id::text,last_check_version,
		          last_band,last_score,last_coverage,opened_at,closed_at,updated_at,record_version`,
		current.TenantID, current.LegalEntityID, current.ID, matterID, now.UTC()))
	if err != nil {
		return AdverseEpisode{}, mapAdverseEpisodePostgresError(err)
	}
	event, err := newMonitoringEvent(updated.TenantID, AggregateMonitoringAdverseEpisode, updated.ID, updated.RecordVersion, EventMonitoringAdverseEpisodeMatterLinked, episodeEventPayload(updated), "", now)
	if err != nil {
		return AdverseEpisode{}, err
	}
	if err := insertMonitoringEventAndOutbox(ctx, tx, event); err != nil {
		return AdverseEpisode{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AdverseEpisode{}, mapPostgresError(err)
	}
	return updated, nil
}

const adverseEpisodeSelect = `SELECT e.id::text,e.tenant_id::text,e.legal_entity_id::text,e.program_id::text,
	e.monitoring_check_id::text,e.state,COALESCE(e.matter_id::text,''),e.first_result_id::text,e.last_result_id::text,
	e.last_check_version,e.last_band,e.last_score,e.last_coverage,e.opened_at,e.closed_at,e.updated_at,e.record_version
	FROM monitoring_adverse_episodes e`

func scanAdverseEpisode(row scanner) (AdverseEpisode, error) {
	var value AdverseEpisode
	err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.ProgramID,
		&value.MonitoringCheckID, &value.State, &value.MatterID, &value.FirstResultID, &value.LastResultID,
		&value.LastCheckVersion, &value.LastBand, &value.LastScore, &value.LastCoverage,
		&value.OpenedAt, &value.ClosedAt, &value.UpdatedAt, &value.RecordVersion)
	return value, err
}

func mapAdverseEpisodePostgresError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrConflict
		case "23503", "23514", "22P02", "22001":
			return ErrInvalid
		}
	}
	return err
}
