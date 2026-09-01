//go:build postgres

package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) ResolveLegalEntity(ctx context.Context, tenant, identifier string) (string, error) {
	identifier = strings.TrimSpace(identifier)
	var resolved string
	err := r.pool.QueryRow(ctx, `SELECT le.id::text FROM legal_entities le JOIN tenants t ON t.id=le.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND le.id::text=$2 AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until)`, tenant, identifier).Scan(&resolved)
	if err == nil {
		return resolved, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	rows, err := r.pool.Query(ctx, `SELECT le.id::text FROM legal_entities le JOIN tenants t ON t.id=le.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND le.code=$2 AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.id LIMIT 2`, tenant, identifier)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	matches := []string{}
	for rows.Next() {
		if err := rows.Scan(&resolved); err != nil {
			return "", err
		}
		matches = append(matches, resolved)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", ErrNotFound
	}
	if len(matches) > 1 {
		return "", ErrLegalEntityAmbiguous
	}
	return matches[0], nil
}

func (r *PostgresRepository) CreateProgram(ctx context.Context, program Program, event Event) (Program, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Program{}, err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,owner_principal_id,authority_principal_id,jurisdiction,scope,effective_from,effective_until,created_at,updated_at,version)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,NULLIF($9,'')::uuid,NULLIF($10,'')::uuid,$11,$12,$13,$14,$15,$15,$16)`,
		program.ID, program.TenantID, program.LegalEntityID, program.Code, program.Name, program.Type, program.Status, program.OwningFunction, program.OwnerPrincipalID, program.AuthorityPrincipalID, program.Jurisdiction, rawJSON(program.Scope, `{}`), program.EffectiveFrom, program.EffectiveUntil, program.CreatedAt, program.Version)
	if err != nil {
		if isUniqueViolation(err) {
			return Program{}, ErrDuplicate
		}
		return Program{}, err
	}
	if err = insertContinuityEvent(ctx, tx, event); err != nil {
		return Program{}, err
	}
	if err = insertOutbox(ctx, tx, event); err != nil {
		return Program{}, err
	}
	if _, err = queueProgramStateTx(ctx, tx, program.TenantID, program.ID, program.Version, "PROGRAM_CREATED", program.ID, event.ActorID, event.OccurredAt); err != nil {
		return Program{}, err
	}
	if err = r.commitContinuityEvents(ctx, tx, event); err != nil {
		return Program{}, err
	}
	return program, nil
}

func (r *PostgresRepository) ListPrograms(ctx context.Context, tenant string, limit int) ([]ProgramAggregate, error) {
	// Ordinary portfolio lists read the authoritative relational projection in
	// one bounded query. Event replay remains reserved for exact Get*/history
	// reads where reconstruction semantics are required.
	return r.listCurrentPrograms(ctx, tenant, limit)
}

func (r *PostgresRepository) GetProgram(ctx context.Context, tenant, id string) (ProgramAggregate, error) {
	events, err := r.ProgramEvents(ctx, tenant, id, nil)
	if err != nil {
		return ProgramAggregate{}, err
	}
	aggregate, err := reconstructProgram(events)
	if err != nil {
		return ProgramAggregate{}, err
	}
	state, err := r.ProgramStateAt(ctx, tenant, id, nil)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if state != nil {
		aggregate.CurrentState = state
	}
	return decorateProgram(aggregate), nil
}

func (r *PostgresRepository) ApplyProgramEvent(ctx context.Context, tenant, id string, expected int64, event Event) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var current int64
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	err = tx.QueryRow(ctx, `SELECT p.version FROM programs p JOIN tenants t ON t.id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND p.id=$2::uuid AND (NOT $3 OR ((t.id::text=$4 OR t.slug=$4) AND p.legal_entity_id IS NOT NULL AND ($5='*' OR p.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=p.tenant_id AND (le.id::text=$5 OR le.code=$5) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.valid_from DESC,le.id LIMIT 1)))) FOR UPDATE`, tenant, id, enforce, actorTenant, actorEntity).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	if current != expected || event.AggregateVersion != expected+1 {
		return 0, ErrVersionConflict
	}
	if err = applyProgramProjection(ctx, tx, event); err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicate
		}
		return 0, err
	}
	tag, err := tx.Exec(ctx, `UPDATE programs SET version=$3,updated_at=$4 WHERE id=$2::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND version=$5`, tenant, id, event.AggregateVersion, event.OccurredAt, expected)
	if err != nil {
		return 0, err
	}
	if tag.RowsAffected() != 1 {
		return 0, ErrVersionConflict
	}
	if err = insertContinuityEvent(ctx, tx, event); err != nil {
		if isUniqueViolation(err) {
			return 0, ErrVersionConflict
		}
		return 0, err
	}
	if err = insertOutbox(ctx, tx, event); err != nil {
		return 0, err
	}
	if event.Type != EventProgramStateUpdated {
		if _, err = queueProgramStateTx(ctx, tx, tenant, id, event.AggregateVersion, event.Type, event.ID, event.ActorID, event.OccurredAt); err != nil {
			return 0, err
		}
	}
	if err = r.commitContinuityEvents(ctx, tx, event); err != nil {
		return 0, err
	}
	return event.AggregateVersion, nil
}

func (r *PostgresRepository) RecordProgramTrigger(ctx context.Context, trigger Trigger) (bool, error) {
	tag, err := r.pool.Exec(ctx, `INSERT INTO program_trigger_events(id,tenant_id,program_id,trigger_type,subject_type,subject_id,dedupe_key,payload,observed_at,source)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(tenant_id,program_id,dedupe_key) DO NOTHING`,
		trigger.ID, trigger.TenantID, trigger.ProgramID, strings.ToUpper(trigger.Type), trigger.SubjectType, trigger.SubjectID, trigger.DedupeKey, rawJSON(trigger.Payload, `{}`), trigger.ObservedAt, trigger.Source)
	if err != nil {
		if isForeignKeyViolation(err) {
			return false, ErrNotFound
		}
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PostgresRepository) ProgramEvents(ctx context.Context, tenant, id string, until *time.Time) ([]Event, error) {
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	query := `SELECT ce.id::text,t.slug,ce.aggregate_type,ce.aggregate_id::text,ce.aggregate_version,ce.event_type,ce.payload,ce.actor_type,COALESCE(ce.actor_id::text,''),ce.occurred_at
		FROM continuity_events ce JOIN tenants t ON t.id=ce.tenant_id JOIN programs p ON p.tenant_id=ce.tenant_id AND p.id=ce.aggregate_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND ce.aggregate_type='PROGRAM' AND ce.aggregate_id=$2::uuid
		  AND (NOT $3 OR ((t.id::text=$4 OR t.slug=$4) AND p.legal_entity_id IS NOT NULL AND ($5='*' OR p.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=p.tenant_id AND (le.id::text=$5 OR le.code=$5) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.valid_from DESC,le.id LIMIT 1))))`
	args := []any{tenant, id, enforce, actorTenant, actorEntity}
	if until != nil {
		query += ` AND ce.occurred_at<=$6`
		args = append(args, *until)
	}
	query += ` ORDER BY ce.aggregate_version`
	return r.scanEvents(ctx, query, args...)
}

func (r *PostgresRepository) CreateMatter(ctx context.Context, matter Matter, event Event) (Matter, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Matter{}, err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,source_type,source_id,trigger_type,trigger_id,trigger_key,known_facts,missing_facts,contradictions,owner_principal_id,required_authority,due_at,closed_at,closure_reason,reopen_count,created_at,updated_at,version,legal_entity_id)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,NULLIF($18,'')::uuid,$19,$20,$21,$22,$23,$24,$24,$25,$26::uuid)`,
		matter.ID, matter.TenantID, matter.Reference, matter.Type, matter.Status, matter.Priority, matter.Title, matter.Summary, rawJSON(matter.Scope, `{}`), matter.SourceType, matter.SourceID, matter.TriggerType, matter.TriggerID, matter.TriggerKey, rawJSON(matter.KnownFacts, `{}`), rawJSON(matter.MissingFacts, `[]`), rawJSON(matter.Contradictions, `[]`), matter.OwnerPrincipalID, matter.RequiredAuthority, matter.DueAt, matter.ClosedAt, matter.ClosureReason, matter.ReopenCount, matter.CreatedAt, matter.Version, matter.LegalEntityID)
	if err != nil {
		if isUniqueViolation(err) {
			return Matter{}, ErrDuplicate
		}
		return Matter{}, err
	}
	if err = insertContinuityEvent(ctx, tx, event); err != nil {
		return Matter{}, err
	}
	if err = insertOutbox(ctx, tx, event); err != nil {
		return Matter{}, err
	}
	if err = r.commitContinuityEvents(ctx, tx, event); err != nil {
		return Matter{}, err
	}
	return matter, nil
}

func (r *PostgresRepository) ListMatters(ctx context.Context, tenant, status string, limit int) ([]MatterAggregate, error) {
	// Keep raw and production repositories aligned on current list visibility;
	// direct service callers must receive the same pre-limit restricted filter.
	return r.listCurrentMatters(ctx, tenant, status, limit)
}

func (r *PostgresRepository) GetMatter(ctx context.Context, tenant, id string) (MatterAggregate, error) {
	events, err := r.MatterEvents(ctx, tenant, id, nil)
	if err != nil {
		return MatterAggregate{}, err
	}
	return reconstructMatter(events)
}

func (r *PostgresRepository) ApplyMatterEvent(ctx context.Context, tenant, id string, expected int64, event Event) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var current int64
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	err = tx.QueryRow(ctx, `SELECT m.version FROM matters m JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND m.id=$2::uuid AND (NOT $3 OR ((t.id::text=$4 OR t.slug=$4) AND m.legal_entity_id IS NOT NULL AND ($5='*' OR m.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=m.tenant_id AND (le.id::text=$5 OR le.code=$5) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.valid_from DESC,le.id LIMIT 1)))) FOR UPDATE`, tenant, id, enforce, actorTenant, actorEntity).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	if current != expected || event.AggregateVersion != expected+1 {
		return 0, ErrVersionConflict
	}
	if err = applyMatterProjection(ctx, tx, event); err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicate
		}
		return 0, err
	}
	tag, err := tx.Exec(ctx, `UPDATE matters SET version=$3,updated_at=$4 WHERE id=$2::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND version=$5`, tenant, id, event.AggregateVersion, event.OccurredAt, expected)
	if err != nil {
		return 0, err
	}
	if tag.RowsAffected() != 1 {
		return 0, ErrVersionConflict
	}
	if err = insertContinuityEvent(ctx, tx, event); err != nil {
		if isUniqueViolation(err) {
			return 0, ErrVersionConflict
		}
		return 0, err
	}
	if err = insertOutbox(ctx, tx, event); err != nil {
		return 0, err
	}
	programRows, err := tx.Query(ctx, `SELECT DISTINCT program_id::text FROM matter_links WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND matter_id=$2::uuid AND program_id IS NOT NULL AND retired_at IS NULL`, tenant, id)
	if err != nil {
		return 0, err
	}
	programIDs := []string{}
	for programRows.Next() {
		var programID string
		if err := programRows.Scan(&programID); err != nil {
			programRows.Close()
			return 0, err
		}
		programIDs = append(programIDs, programID)
	}
	if err := programRows.Err(); err != nil {
		programRows.Close()
		return 0, err
	}
	programRows.Close()
	if event.Type == EventMatterLinkRetired {
		var retired MatterLink
		if err := json.Unmarshal(event.Payload, &retired); err != nil {
			return 0, err
		}
		if retired.ProgramID != "" && !slices.Contains(programIDs, retired.ProgramID) {
			programIDs = append(programIDs, retired.ProgramID)
		}
	}
	for _, programID := range programIDs {
		if _, err = queueProgramStateTx(ctx, tx, tenant, programID, 0, event.Type, id, event.ActorID, event.OccurredAt); err != nil {
			return 0, err
		}
	}
	if err = r.commitContinuityEvents(ctx, tx, event); err != nil {
		return 0, err
	}
	return event.AggregateVersion, nil
}

func (r *PostgresRepository) MatterByTriggerKey(ctx context.Context, tenant, triggerKey string) (Matter, error) {
	var id string
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	err := r.pool.QueryRow(ctx, `SELECT m.id::text FROM matters m JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND m.trigger_key=$2 AND m.status NOT IN ('CLOSED','CANCELLED') AND (NOT $3 OR ((t.id::text=$4 OR t.slug=$4) AND m.legal_entity_id IS NOT NULL AND ($5='*' OR m.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=m.tenant_id AND (le.id::text=$5 OR le.code=$5) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.valid_from DESC,le.id LIMIT 1)))) ORDER BY m.created_at DESC LIMIT 1`, tenant, triggerKey, enforce, actorTenant, actorEntity).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Matter{}, ErrNotFound
	}
	if err != nil {
		return Matter{}, err
	}
	value, err := r.GetMatter(ctx, tenant, id)
	return value.Matter, err
}

func (r *PostgresRepository) MatterEvents(ctx context.Context, tenant, id string, until *time.Time) ([]Event, error) {
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	query := `SELECT ce.id::text,t.slug,ce.aggregate_type,ce.aggregate_id::text,ce.aggregate_version,ce.event_type,ce.payload,ce.actor_type,COALESCE(ce.actor_id::text,''),ce.occurred_at
		FROM continuity_events ce JOIN tenants t ON t.id=ce.tenant_id JOIN matters m ON m.tenant_id=ce.tenant_id AND m.id=ce.aggregate_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND ce.aggregate_type='MATTER' AND ce.aggregate_id=$2::uuid
		  AND (NOT $3 OR ((t.id::text=$4 OR t.slug=$4) AND m.legal_entity_id IS NOT NULL AND ($5='*' OR m.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=m.tenant_id AND (le.id::text=$5 OR le.code=$5) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.valid_from DESC,le.id LIMIT 1))))`
	args := []any{tenant, id, enforce, actorTenant, actorEntity}
	if until != nil {
		query += ` AND ce.occurred_at<=$6`
		args = append(args, *until)
	}
	query += ` ORDER BY ce.aggregate_version`
	return r.scanEvents(ctx, query, args...)
}

func (r *PostgresRepository) ResponsePackageHistory(ctx context.Context, tenant, matterID, responseID string, limit int) ([]ResponseHistoryItem, bool, error) {
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	rows, err := r.pool.Query(ctx, `SELECT ce.payload->>'status',ce.occurred_at,
		CASE WHEN ce.actor_type='SYSTEM' THEN 'Automated process' ELSE COALESCE(NULLIF(trim(p.display_name),''),'Recorded person unavailable') END,
		ce.aggregate_version
		FROM continuity_events ce
		JOIN tenants t ON t.id=ce.tenant_id
		JOIN matters m ON m.tenant_id=ce.tenant_id AND m.id=ce.aggregate_id
		LEFT JOIN principals p ON p.tenant_id=ce.tenant_id AND p.id=ce.actor_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND ce.aggregate_type='MATTER' AND ce.aggregate_id=$2::uuid
		  AND (NOT $3 OR ((t.id::text=$4 OR t.slug=$4) AND m.legal_entity_id IS NOT NULL AND ($5='*' OR m.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=m.tenant_id AND (le.id::text=$5 OR le.code=$5) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.valid_from DESC,le.id LIMIT 1))))
		  AND ce.event_type IN ('RESPONSE_PACKAGE_ADDED','RESPONSE_PACKAGE_STATE_CHANGED')
		  AND ce.payload->>'id'=$6
		ORDER BY ce.aggregate_version DESC LIMIT $7`, tenant, matterID, enforce, actorTenant, actorEntity, responseID, limit+1)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	values := make([]ResponseHistoryItem, 0, limit+1)
	for rows.Next() {
		var value ResponseHistoryItem
		if err = rows.Scan(&value.Status, &value.OccurredAt, &value.ActorLabel, &value.AggregateVersion); err != nil {
			return nil, false, err
		}
		values = append(values, value)
	}
	if err = rows.Err(); err != nil {
		return nil, false, err
	}
	if len(values) == 0 {
		return nil, false, ErrNotFound
	}
	hasMore := len(values) > limit
	if hasMore {
		values = values[:limit]
	}
	return values, hasMore, nil
}

func (r *PostgresRepository) OpenMatterCount(ctx context.Context, tenant, programID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT count(DISTINCT m.id) FROM matters m JOIN matter_links ml ON ml.matter_id=m.id AND ml.tenant_id=m.tenant_id JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND ml.program_id=$2::uuid AND ml.retired_at IS NULL AND m.status NOT IN ('CLOSED','CANCELLED')`, tenant, programID).Scan(&count)
	return count, err
}

func (r *PostgresRepository) LinkedProgramIDs(ctx context.Context, tenant, matterID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT ml.program_id::text FROM matter_links ml JOIN tenants t ON t.id=ml.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND ml.matter_id=$2::uuid AND ml.program_id IS NOT NULL AND ml.retired_at IS NULL ORDER BY ml.program_id::text`, tenant, matterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		values = append(values, id)
	}
	return values, rows.Err()
}

func (r *PostgresRepository) scanEvents(ctx context.Context, query string, args ...any) ([]Event, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []Event{}
	for rows.Next() {
		var value Event
		if err := rows.Scan(&value.ID, &value.TenantID, &value.AggregateType, &value.AggregateID, &value.AggregateVersion, &value.Type, &value.Payload, &value.ActorType, &value.ActorID, &value.OccurredAt); err != nil {
			return nil, err
		}
		value.OccurredAt = value.OccurredAt.UTC()
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, ErrNotFound
	}
	return values, nil
}

func applyProgramProjection(ctx context.Context, tx pgx.Tx, event Event) error {
	switch event.Type {
	case EventProgramStatusChanged:
		var v Program
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE programs SET status=$3,effective_until=$4,updated_at=$5 WHERE id=$2::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)`, v.TenantID, v.ID, v.Status, v.EffectiveUntil, v.UpdatedAt)
		return err
	case EventProgramDetailsUpdated, EventProgramOwnerChanged, EventProgramApprovalAuthorityChanged:
		v, ok, err := programProjectionProgram(event)
		if err != nil {
			return err
		}
		if !ok {
			return ErrInvalidState
		}
		_, err = tx.Exec(ctx, `UPDATE programs SET name=$3,owning_function=$4,owner_principal_id=NULLIF($5,'')::uuid,authority_principal_id=NULLIF($6,'')::uuid,jurisdiction=$7,scope=$8,effective_from=$9,effective_until=$10,updated_at=$11 WHERE id=$2::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)`, v.TenantID, v.ID, v.Name, v.OwningFunction, v.OwnerPrincipalID, v.AuthorityPrincipalID, v.Jurisdiction, rawJSON(v.Scope, `{}`), v.EffectiveFrom, v.EffectiveUntil, v.UpdatedAt)
		return err
	case EventRequirementAdded:
		var v Requirement
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO program_requirements(id,tenant_id,program_id,source_id,code,title,statement,source_anchor,modality,actor,action,object_name,status,effective_from,effective_until,created_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,NULLIF($4,'')::uuid,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`, v.ID, v.TenantID, v.ProgramID, v.SourceID, v.Code, v.Title, v.Statement, v.SourceAnchor, v.Modality, v.Actor, v.Action, v.Object, v.Status, v.EffectiveFrom, v.EffectiveUntil, v.CreatedAt, v.Version)
		return err
	case EventRequirementSuperseded:
		var v requirementSupersededEvent
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		result, err := tx.Exec(ctx, `UPDATE program_requirements SET status=$4,effective_until=$5,version=$6 WHERE id=$3::uuid AND program_id=$2::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND status=$7 AND effective_until IS NULL`, v.Prior.TenantID, v.Prior.ProgramID, v.Prior.ID, v.Prior.Status, v.Prior.EffectiveUntil, v.Prior.Version, RequirementApproved)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ErrVersionConflict
		}
		_, err = tx.Exec(ctx, `INSERT INTO program_requirements(id,tenant_id,program_id,source_id,code,title,statement,source_anchor,modality,actor,action,object_name,status,effective_from,effective_until,created_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,NULLIF($4,'')::uuid,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`, v.Replacement.ID, v.Replacement.TenantID, v.Replacement.ProgramID, v.Replacement.SourceID, v.Replacement.Code, v.Replacement.Title, v.Replacement.Statement, v.Replacement.SourceAnchor, v.Replacement.Modality, v.Replacement.Actor, v.Replacement.Action, v.Replacement.Object, v.Replacement.Status, v.Replacement.EffectiveFrom, v.Replacement.EffectiveUntil, v.Replacement.CreatedAt, v.Replacement.Version)
		return err
	case EventApplicabilityDetermined:
		var v Applicability
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO program_applicability(id,tenant_id,program_id,requirement_id,status,scope,rationale,approved_by,effective_from,effective_until,created_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,$5,$6,$7,$8::uuid,$9,$10,$11,$12)`, v.ID, v.TenantID, v.ProgramID, v.RequirementID, v.Status, rawJSON(v.Scope, `{}`), v.Rationale, v.ApprovedBy, v.EffectiveFrom, v.EffectiveUntil, v.CreatedAt, v.Version)
		return err
	case EventControlObjectiveAdded:
		var v ControlObjective
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO control_objectives(id,tenant_id,program_id,code,name,outcome,status,created_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,$9)`, v.ID, v.TenantID, v.ProgramID, v.Code, v.Name, v.Outcome, v.Status, v.CreatedAt, v.Version)
		return err
	case EventControlImplementationAdded:
		var v ControlImplementation
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO control_implementations(id,tenant_id,program_id,objective_id,name,description,implementation_type,owner_principal_id,scope,status,effective_from,effective_until,created_at,updated_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,$5,$6,$7,NULLIF($8,'')::uuid,$9,$10,$11,$12,$13,$13,$14)`, v.ID, v.TenantID, v.ProgramID, v.ObjectiveID, v.Name, v.Description, v.ImplementationType, v.OwnerPrincipalID, rawJSON(v.Scope, `{}`), v.Status, v.EffectiveFrom, v.EffectiveUntil, v.CreatedAt, v.Version)
		return err
	case EventControlImplementationRevised, EventControlImplementationOwnerChanged, EventControlImplementationStatusChanged:
		var v controlImplementationLifecycleEvent
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		result, err := tx.Exec(ctx, `UPDATE control_implementations SET name=$5,description=$6,implementation_type=$7,owner_principal_id=NULLIF($8,'')::uuid,scope=$9,status=$10,effective_from=$11,effective_until=$12,updated_at=$13,version=$14 WHERE id=$3::uuid AND program_id=$2::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND version=$4`, v.Current.TenantID, v.Current.ProgramID, v.Current.ID, v.Prior.Version, v.Current.Name, v.Current.Description, v.Current.ImplementationType, v.Current.OwnerPrincipalID, rawJSON(v.Current.Scope, `{}`), v.Current.Status, v.Current.EffectiveFrom, v.Current.EffectiveUntil, v.Current.UpdatedAt, v.Current.Version)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ErrVersionConflict
		}
		return nil
	case EventRequirementControlLinked:
		var v RequirementControlLink
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO requirement_control_links(id,tenant_id,program_id,requirement_id,implementation_id,created_at) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,$5::uuid,$6)`, v.ID, v.TenantID, v.ProgramID, v.RequirementID, v.ImplementationID, v.CreatedAt)
		return err
	case EventRequirementControlLinkRetired:
		var v RequirementControlLink
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		result, err := tx.Exec(ctx, `UPDATE requirement_control_links SET retired_at=$4,retired_by=$5::uuid,retirement_reason=$6 WHERE id=$3::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND program_id=$2::uuid AND retired_at IS NULL`, v.TenantID, v.ProgramID, v.ID, v.RetiredAt, v.RetiredBy, v.RetirementReason)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ErrNotFound
		}
		return nil
	case EventEvidenceContractAdded:
		var v EvidenceContract
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		if err := validateProgramEvidenceSourcesTx(ctx, tx, v.TenantID, v.ProgramID, v.AcceptableSourceIDs); err != nil {
			return err
		}
		sourceIDs := v.AcceptableSourceIDs
		if sourceIDs == nil {
			sourceIDs = []string{}
		}
		sources, _ := json.Marshal(sourceIDs)
		_, err := tx.Exec(ctx, `INSERT INTO evidence_contracts(id,tenant_id,program_id,requirement_id,control_implementation_id,code,name,claim,acceptable_source_ids,population_scope,freshness_minutes,minimum_coverage,independence_required,contradiction_policy,failure_action,configured_by,status,created_at,updated_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,NULLIF($4,'')::uuid,NULLIF($5,'')::uuid,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NULLIF($16,'')::uuid,$17,$18,$18,$19)`, v.ID, v.TenantID, v.ProgramID, v.RequirementID, v.ControlImplementationID, v.Code, v.Name, v.Claim, sources, rawJSON(v.PopulationScope, `{}`), v.FreshnessMinutes, v.MinimumCoverage, v.IndependenceRequired, v.ContradictionPolicy, v.FailureAction, v.ConfiguredBy, v.Status, v.CreatedAt, v.Version)
		if err != nil {
			return err
		}
		for _, sourceID := range v.AcceptableSourceIDs {
			if strings.TrimSpace(sourceID) == "" {
				continue
			}
			if _, err = tx.Exec(ctx, `INSERT INTO evidence_contract_sources(tenant_id,program_id,contract_id,source_id) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3::uuid,$4::uuid)`, v.TenantID, v.ProgramID, v.ID, sourceID); err != nil {
				return err
			}
		}
		return nil
	case EventEvidenceContractRevised, EventEvidenceContractStatusChanged:
		var v evidenceContractLifecycleEvent
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		if err := validateProgramEvidenceSourcesTx(ctx, tx, v.Current.TenantID, v.Current.ProgramID, v.Current.AcceptableSourceIDs); err != nil {
			return err
		}
		sourceIDs := v.Current.AcceptableSourceIDs
		if sourceIDs == nil {
			sourceIDs = []string{}
		}
		sources, _ := json.Marshal(sourceIDs)
		result, err := tx.Exec(ctx, `UPDATE evidence_contracts SET name=$5,claim=$6,acceptable_source_ids=$7,population_scope=$8,freshness_minutes=$9,minimum_coverage=$10,independence_required=$11,contradiction_policy=$12,failure_action=$13,configured_by=NULLIF($14,'')::uuid,status=$15,updated_at=$16,version=$17 WHERE id=$3::uuid AND program_id=$2::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND version=$4`, v.Current.TenantID, v.Current.ProgramID, v.Current.ID, v.Prior.Version, v.Current.Name, v.Current.Claim, sources, rawJSON(v.Current.PopulationScope, `{}`), v.Current.FreshnessMinutes, v.Current.MinimumCoverage, v.Current.IndependenceRequired, v.Current.ContradictionPolicy, v.Current.FailureAction, v.Current.ConfiguredBy, v.Current.Status, v.Current.UpdatedAt, v.Current.Version)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ErrVersionConflict
		}
		if _, err = tx.Exec(ctx, `DELETE FROM evidence_contract_sources WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND program_id=$2::uuid AND contract_id=$3::uuid`, v.Current.TenantID, v.Current.ProgramID, v.Current.ID); err != nil {
			return err
		}
		for _, sourceID := range v.Current.AcceptableSourceIDs {
			if strings.TrimSpace(sourceID) == "" {
				continue
			}
			if _, err = tx.Exec(ctx, `INSERT INTO evidence_contract_sources(tenant_id,program_id,contract_id,source_id) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3::uuid,$4::uuid)`, v.Current.TenantID, v.Current.ProgramID, v.Current.ID, sourceID); err != nil {
				return err
			}
		}
		return nil
	case EventEvidenceAssessmentRecorded:
		var v EvidenceAssessment
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO evidence_assessments(id,tenant_id,program_id,contract_id,conclusion,coverage,basis,valid_until,assessed_by,assessed_at,created_at) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,$5,$6,$7,$8,NULLIF($9,'')::uuid,$10,$11)`, v.ID, v.TenantID, v.ProgramID, v.ContractID, v.Conclusion, v.Coverage, rawJSON(v.Basis, `{}`), v.ValidUntil, v.AssessedBy, v.AssessedAt, v.CreatedAt)
		return err
	case EventProgramStateUpdated:
		var v ProgramStateSnapshot
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		dimensions, _ := json.Marshal(v.Dimensions)
		reasons, _ := json.Marshal(v.Reasons)
		_, err := tx.Exec(ctx, `INSERT INTO program_state_snapshots(id,tenant_id,program_id,overall_state,dimensions,reasons,open_matter_count,trigger_type,trigger_id,generated_at,program_version,projection_version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,$9,$10,$11,(SELECT COALESCE(max(projection_version),0)+1 FROM program_state_snapshots WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND program_id=$3::uuid))`, v.ID, v.TenantID, v.ProgramID, v.Overall, dimensions, reasons, v.OpenMatterCount, v.TriggerType, v.TriggerID, v.GeneratedAt, v.ProgramVersion)
		return err
	case EventProgramTriggerRecorded:
		return nil
	default:
		return ErrInvalidState
	}
}

func programProjectionProgram(event Event) (Program, bool, error) {
	switch event.Type {
	case EventProgramDetailsUpdated:
		var changed programDetailsUpdatedEvent
		if err := json.Unmarshal(event.Payload, &changed); err != nil {
			return Program{}, true, err
		}
		return changed.Program, true, nil
	case EventProgramOwnerChanged:
		var changed programOwnerChangedEvent
		if err := json.Unmarshal(event.Payload, &changed); err != nil {
			return Program{}, true, err
		}
		return changed.Program, true, nil
	case EventProgramApprovalAuthorityChanged:
		var changed programApprovalAuthorityChangedEvent
		if err := json.Unmarshal(event.Payload, &changed); err != nil {
			return Program{}, true, err
		}
		return changed.Program, true, nil
	default:
		return Program{}, false, nil
	}
}

func applyMatterProjection(ctx context.Context, tx pgx.Tx, event Event) error {
	switch event.Type {
	case EventMatterLinked:
		var v MatterLink
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO matter_links(id,tenant_id,matter_id,program_id,requirement_id,control_id,relationship,created_at) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,NULLIF($4,'')::uuid,NULLIF($5,'')::uuid,NULLIF($6,'')::uuid,$7,$8)`, v.ID, v.TenantID, v.MatterID, v.ProgramID, v.RequirementID, v.ControlID, v.Relationship, v.CreatedAt)
		return err
	case EventMatterLinkRetired:
		var v MatterLink
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		result, err := tx.Exec(ctx, `UPDATE matter_links SET retired_at=$4,retired_by=$5::uuid,retirement_reason=$6 WHERE id=$3::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND matter_id=$2::uuid AND retired_at IS NULL`, v.TenantID, v.MatterID, v.ID, v.RetiredAt, v.RetiredBy, v.RetirementReason)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ErrNotFound
		}
		return nil
	case EventMatterStateChanged, EventMatterDetailsUpdated, EventMatterContextChanged, EventMatterOwnerChanged:
		v, ok, err := matterProjectionMatter(event)
		if err != nil {
			return err
		}
		if !ok {
			return ErrInvalidState
		}
		_, err = tx.Exec(ctx, `UPDATE matters SET status=$3,priority=$4,title=$5,summary=$6,scope=$7,known_facts=$8,missing_facts=$9,contradictions=$10,owner_principal_id=NULLIF($11,'')::uuid,required_authority=$12,due_at=$13,closed_at=$14,closure_reason=$15,reopen_count=$16,updated_at=$17 WHERE id=$2::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)`, v.TenantID, v.ID, v.Status, v.Priority, v.Title, v.Summary, rawJSON(v.Scope, `{}`), rawJSON(v.KnownFacts, `{}`), rawJSON(v.MissingFacts, `[]`), rawJSON(v.Contradictions, `[]`), v.OwnerPrincipalID, v.RequiredAuthority, v.DueAt, v.ClosedAt, v.ClosureReason, v.ReopenCount, v.UpdatedAt)
		return err
	case EventDecisionAdded:
		var v Decision
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO matter_decisions(id,tenant_id,matter_id,decision_type,status,options,selected_option,rationale,conditions,authority_principal_id,decided_at,expires_at,created_at,updated_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,$9,NULLIF($10,'')::uuid,$11,$12,$13,$13,$14)`, v.ID, v.TenantID, v.MatterID, v.Type, v.Status, rawJSON(v.Options, `[]`), v.SelectedOption, v.Rationale, rawJSON(v.Conditions, `[]`), v.AuthorityPrincipalID, v.DecidedAt, v.ExpiresAt, v.CreatedAt, v.Version)
		return err
	case EventActionAdded:
		var v Action
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO matter_actions(id,tenant_id,matter_id,origin_key,title,description,owner_principal_id,required_responsibility,status,due_at,implemented_at,created_at,updated_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,NULLIF($4,''),$5,$6,NULLIF($7,'')::uuid,$8,$9,$10,$11,$12,$12,$13)`, v.ID, v.TenantID, v.MatterID, v.OriginKey, v.Title, v.Description, v.OwnerPrincipalID, ActionResponsibility(v), v.Status, v.DueAt, v.ImplementedAt, v.CreatedAt, v.Version)
		return err
	case EventActionStateChanged, EventActionUpdated, EventActionAssigned:
		v, ok, err := matterProjectionAction(event)
		if err != nil {
			return err
		}
		if !ok {
			return ErrInvalidState
		}
		_, err = tx.Exec(ctx, `UPDATE matter_actions SET title=$4,description=$5,owner_principal_id=NULLIF($6,'')::uuid,required_responsibility=$7,status=$8,due_at=$9,implemented_at=$10,updated_at=$11,version=$12 WHERE id=$3::uuid AND matter_id=$2::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)`, v.TenantID, v.MatterID, v.ID, v.Title, v.Description, v.OwnerPrincipalID, ActionResponsibility(v), v.Status, v.DueAt, v.ImplementedAt, v.UpdatedAt, v.Version)
		return err
	case EventVerificationContractAdded:
		var v VerificationContract
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		if err := validateMatterEvidenceSourcesTx(ctx, tx, v.TenantID, v.MatterID, []string{v.MeasurementSourceID}); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO verification_contracts(id,tenant_id,matter_id,supersedes_contract_id,action_id,expected_outcome,baseline,scope,measurement_source_id,threshold,observation_period_minutes,authority_principal_id,failure_response,status,created_at,updated_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,NULLIF($4,'')::uuid,NULLIF($5,'')::uuid,$6,$7,$8,NULLIF($9,'')::uuid,$10,$11,NULLIF($12,'')::uuid,$13,$14,$15,$15,$16)`, v.ID, v.TenantID, v.MatterID, v.SupersedesContractID, v.ActionID, v.ExpectedOutcome, rawJSON(v.Baseline, `{}`), rawJSON(v.Scope, `{}`), v.MeasurementSourceID, rawJSON(v.Threshold, `{}`), v.ObservationPeriodMinutes, v.AuthorityPrincipalID, v.FailureResponse, v.Status, v.CreatedAt, v.Version)
		return err
	case EventVerificationContractSuperseded:
		var v verificationContractSupersededEvent
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		if err := validateMatterEvidenceSourcesTx(ctx, tx, v.Replacement.TenantID, v.Replacement.MatterID, []string{v.Replacement.MeasurementSourceID}); err != nil {
			return err
		}
		result, err := tx.Exec(ctx, `UPDATE verification_contracts SET status=$4,updated_at=$5,version=$6 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND matter_id=$2::uuid AND id=$3::uuid AND status=$7 AND version=$8`, v.Prior.TenantID, v.Prior.MatterID, v.Prior.ID, v.Prior.Status, v.Prior.UpdatedAt, v.Prior.Version, VerificationActive, v.Prior.Version-1)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ErrVersionConflict
		}
		_, err = tx.Exec(ctx, `INSERT INTO verification_contracts(id,tenant_id,matter_id,supersedes_contract_id,action_id,expected_outcome,baseline,scope,measurement_source_id,threshold,observation_period_minutes,authority_principal_id,failure_response,status,created_at,updated_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,NULLIF($5,'')::uuid,$6,$7,$8,NULLIF($9,'')::uuid,$10,$11,NULLIF($12,'')::uuid,$13,$14,$15,$15,$16)`, v.Replacement.ID, v.Replacement.TenantID, v.Replacement.MatterID, v.Replacement.SupersedesContractID, v.Replacement.ActionID, v.Replacement.ExpectedOutcome, rawJSON(v.Replacement.Baseline, `{}`), rawJSON(v.Replacement.Scope, `{}`), v.Replacement.MeasurementSourceID, rawJSON(v.Replacement.Threshold, `{}`), v.Replacement.ObservationPeriodMinutes, v.Replacement.AuthorityPrincipalID, v.Replacement.FailureResponse, v.Replacement.Status, v.Replacement.CreatedAt, v.Replacement.Version)
		return err
	case EventVerificationContractRetired:
		var v verificationContractRetiredEvent
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		result, err := tx.Exec(ctx, `UPDATE verification_contracts SET status=$4,updated_at=$5,version=$6 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND matter_id=$2::uuid AND id=$3::uuid AND status=$7 AND version=$8`, v.Contract.TenantID, v.Contract.MatterID, v.Contract.ID, v.Contract.Status, v.Contract.UpdatedAt, v.Contract.Version, VerificationActive, v.Contract.Version-1)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ErrVersionConflict
		}
		return nil
	case EventVerificationResultRecorded:
		var v VerificationResult
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO verification_results(id,tenant_id,matter_id,contract_id,result,observations,evidence_references,reviewer_principal_id,reviewer_authority_principal_id,rationale,observed_at,created_at) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,$5,$6,$7,NULLIF($8,'')::uuid,NULLIF($9,'')::uuid,$10,$11,$12)`, v.ID, v.TenantID, v.MatterID, v.ContractID, v.Result, rawJSON(v.Observations, `{}`), rawJSON(v.EvidenceReferences, `[]`), v.ReviewerPrincipalID, v.ReviewerAuthorityPrincipalID, v.Rationale, v.ObservedAt, v.CreatedAt)
		return err
	case EventResponsePackageAdded:
		var v ResponsePackage
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO response_packages(id,tenant_id,matter_id,purpose,audience,status,manifest,approved_by,transmitted_at,acknowledged_at,created_at,updated_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,NULLIF($8,'')::uuid,$9,$10,$11,$11,$12)`, v.ID, v.TenantID, v.MatterID, v.Purpose, v.Audience, v.Status, rawJSON(v.Manifest, `[]`), v.ApprovedBy, v.TransmittedAt, v.AcknowledgedAt, v.CreatedAt, v.Version)
		return err
	case EventResponsePackageStateChanged:
		var v ResponsePackage
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE response_packages SET status=$4,manifest=$5,approved_by=NULLIF($6,'')::uuid,transmitted_at=$7,acknowledged_at=$8,updated_at=$9,version=$10 WHERE id=$3::uuid AND matter_id=$2::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)`, v.TenantID, v.MatterID, v.ID, v.Status, rawJSON(v.Manifest, `[]`), v.ApprovedBy, v.TransmittedAt, v.AcknowledgedAt, v.UpdatedAt, v.Version)
		return err
	default:
		return ErrInvalidState
	}
}

func validateProgramEvidenceSourcesTx(ctx context.Context, tx pgx.Tx, tenant, programID string, sourceIDs []string) error {
	return validateContractSourcesTx(ctx, tx, programEvidenceSourceValidationSQL, tenant, programID, sourceIDs)
}

func validateMatterEvidenceSourcesTx(ctx context.Context, tx pgx.Tx, tenant, matterID string, sourceIDs []string) error {
	return validateContractSourcesTx(ctx, tx, matterEvidenceSourceValidationSQL, tenant, matterID, sourceIDs)
}

const programEvidenceSourceValidationSQL = `SELECT es.id::text FROM evidence_sources es
	JOIN programs p ON p.tenant_id=es.tenant_id AND p.legal_entity_id=es.legal_entity_id
	JOIN tenants t ON t.id=p.tenant_id
	WHERE (t.id::text=$1 OR t.slug=$1) AND p.id=$2::uuid AND es.status='ACTIVE' AND es.id=ANY($3::uuid[])
	FOR SHARE OF es`

const matterEvidenceSourceValidationSQL = `SELECT es.id::text FROM evidence_sources es
	JOIN matters m ON m.tenant_id=es.tenant_id AND m.legal_entity_id=es.legal_entity_id
	JOIN tenants t ON t.id=m.tenant_id
	WHERE (t.id::text=$1 OR t.slug=$1) AND m.id=$2::uuid AND es.status='ACTIVE' AND es.id=ANY($3::uuid[])
	FOR SHARE OF es`

func validateContractSourcesTx(ctx context.Context, tx pgx.Tx, query, tenant, aggregateID string, sourceIDs []string) error {
	unique := make([]string, 0, len(sourceIDs))
	seen := map[string]struct{}{}
	for _, sourceID := range sourceIDs {
		sourceID = strings.TrimSpace(sourceID)
		if sourceID == "" {
			continue
		}
		if _, ok := seen[sourceID]; ok {
			continue
		}
		seen[sourceID] = struct{}{}
		unique = append(unique, sourceID)
	}
	if len(unique) == 0 {
		return nil
	}
	rows, err := tx.Query(ctx, query, tenant, aggregateID, unique)
	if err != nil {
		return err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count != len(unique) {
		return ErrEvidenceSourceInvalid
	}
	return nil
}

func matterProjectionMatter(event Event) (Matter, bool, error) {
	var value Matter
	switch event.Type {
	case EventMatterStateChanged:
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return Matter{}, true, err
		}
	case EventMatterDetailsUpdated:
		var changed matterDetailsUpdatedEvent
		if err := json.Unmarshal(event.Payload, &changed); err != nil {
			return Matter{}, true, err
		}
		value = changed.Matter
	case EventMatterContextChanged:
		var changed matterContextChangedEvent
		if err := json.Unmarshal(event.Payload, &changed); err != nil {
			return Matter{}, true, err
		}
		value = changed.Matter
	case EventMatterOwnerChanged:
		var changed matterOwnerChangedEvent
		if err := json.Unmarshal(event.Payload, &changed); err != nil {
			return Matter{}, true, err
		}
		value = changed.Matter
	default:
		return Matter{}, false, nil
	}
	return value, true, nil
}

func matterProjectionAction(event Event) (Action, bool, error) {
	var value Action
	switch event.Type {
	case EventActionStateChanged:
		if err := json.Unmarshal(event.Payload, &value); err != nil {
			return Action{}, true, err
		}
	case EventActionUpdated:
		var changed actionUpdatedEvent
		if err := json.Unmarshal(event.Payload, &changed); err != nil {
			return Action{}, true, err
		}
		value = changed.Action
	case EventActionAssigned:
		var changed actionAssignedEvent
		if err := json.Unmarshal(event.Payload, &changed); err != nil {
			return Action{}, true, err
		}
		value = changed.Action
	default:
		return Action{}, false, nil
	}
	return value, true, nil
}

func insertContinuityEvent(ctx context.Context, tx pgx.Tx, event Event) error {
	_, err := tx.Exec(ctx, `INSERT INTO continuity_events(id,tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_type,actor_id,occurred_at) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4::uuid,$5,$6,$7,$8,NULLIF($9,'')::uuid,$10)`, event.ID, event.TenantID, event.AggregateType, event.AggregateID, event.AggregateVersion, event.Type, rawJSON(event.Payload, `{}`), event.ActorType, event.ActorID, event.OccurredAt)
	return err
}

func insertOutbox(ctx context.Context, tx pgx.Tx, event Event) error {
	_, err := tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3::uuid,$4,$5,$6,$6,$6)`, event.TenantID, event.AggregateType, event.AggregateID, event.Type, rawJSON(event.Payload, `{}`), event.OccurredAt)
	return err
}

func rawJSON(value json.RawMessage, fallback string) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(fallback)
	}
	return value
}

func isUniqueViolation(err error) bool     { return strings.Contains(err.Error(), "SQLSTATE 23505") }
func isForeignKeyViolation(err error) bool { return strings.Contains(err.Error(), "SQLSTATE 23503") }

var _ Repository = (*PostgresRepository)(nil)
