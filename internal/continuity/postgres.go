//go:build postgres

package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateProgram(ctx context.Context, program Program, event Event) (Program, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Program{}, err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,owner_principal_id,authority_principal_id,jurisdiction,scope,effective_from,effective_until,created_at,updated_at,version)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),NULLIF($3,'')::uuid,$4,$5,$6,$7,$8,NULLIF($9,'')::uuid,NULLIF($10,'')::uuid,$11,$12,$13,$14,$15,$15,$16)`,
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
	if err = tx.Commit(ctx); err != nil {
		return Program{}, err
	}
	return program, nil
}

func (r *PostgresRepository) ListPrograms(ctx context.Context, tenant string, limit int) ([]ProgramAggregate, error) {
	rows, err := r.pool.Query(ctx, `SELECT p.id::text FROM programs p JOIN tenants t ON t.id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) ORDER BY CASE p.status WHEN 'ACTIVE' THEN 0 WHEN 'PAUSED' THEN 1 WHEN 'DRAFT' THEN 2 ELSE 3 END,p.updated_at DESC,p.id LIMIT $2`, tenant, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	values := make([]ProgramAggregate, 0, len(ids))
	for _, id := range ids {
		value, err := r.GetProgram(ctx, tenant, id)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (r *PostgresRepository) GetProgram(ctx context.Context, tenant, id string) (ProgramAggregate, error) {
	events, err := r.ProgramEvents(ctx, tenant, id, nil)
	if err != nil {
		return ProgramAggregate{}, err
	}
	return reconstructProgram(events)
}

func (r *PostgresRepository) ApplyProgramEvent(ctx context.Context, tenant, id string, expected int64, event Event) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var current int64
	err = tx.QueryRow(ctx, `SELECT p.version FROM programs p JOIN tenants t ON t.id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND p.id=$2::uuid FOR UPDATE`, tenant, id).Scan(&current)
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
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return event.AggregateVersion, nil
}

func (r *PostgresRepository) RecordProgramTrigger(ctx context.Context, trigger Trigger) (bool, error) {
	tag, err := r.pool.Exec(ctx, `INSERT INTO program_trigger_events(id,tenant_id,program_id,trigger_type,subject_type,subject_id,dedupe_key,payload,observed_at,source)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(tenant_id,dedupe_key) DO NOTHING`,
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
	query := `SELECT ce.id::text,t.slug,ce.aggregate_type,ce.aggregate_id::text,ce.aggregate_version,ce.event_type,ce.payload,ce.actor_type,COALESCE(ce.actor_id::text,''),ce.occurred_at
		FROM continuity_events ce JOIN tenants t ON t.id=ce.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND ce.aggregate_type='PROGRAM' AND ce.aggregate_id=$2::uuid`
	args := []any{tenant, id}
	if until != nil {
		query += ` AND ce.occurred_at<=$3`
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
	_, err = tx.Exec(ctx, `INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,source_type,source_id,trigger_type,trigger_id,trigger_key,known_facts,missing_facts,contradictions,owner_principal_id,required_authority,due_at,closed_at,closure_reason,reopen_count,created_at,updated_at,version)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,NULLIF($18,'')::uuid,$19,$20,$21,$22,$23,$24,$24,$25)`,
		matter.ID, matter.TenantID, matter.Reference, matter.Type, matter.Status, matter.Priority, matter.Title, matter.Summary, rawJSON(matter.Scope, `{}`), matter.SourceType, matter.SourceID, matter.TriggerType, matter.TriggerID, matter.TriggerKey, rawJSON(matter.KnownFacts, `{}`), rawJSON(matter.MissingFacts, `[]`), rawJSON(matter.Contradictions, `[]`), matter.OwnerPrincipalID, matter.RequiredAuthority, matter.DueAt, matter.ClosedAt, matter.ClosureReason, matter.ReopenCount, matter.CreatedAt, matter.Version)
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
	if err = tx.Commit(ctx); err != nil {
		return Matter{}, err
	}
	return matter, nil
}

func (r *PostgresRepository) ListMatters(ctx context.Context, tenant, status string, limit int) ([]MatterAggregate, error) {
	query := `SELECT m.id::text FROM matters m JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1)`
	args := []any{tenant}
	if status == "OPEN" {
		query += ` AND m.status NOT IN ('CLOSED','CANCELLED')`
	} else if status != "" {
		query += ` AND m.status=$2`
		args = append(args, status)
	}
	query += fmt.Sprintf(` ORDER BY m.priority DESC,m.due_at NULLS LAST,m.updated_at DESC,m.id LIMIT $%d`, len(args)+1)
	args = append(args, limit)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	values := make([]MatterAggregate, 0, len(ids))
	for _, id := range ids {
		value, err := r.GetMatter(ctx, tenant, id)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
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
	err = tx.QueryRow(ctx, `SELECT m.version FROM matters m JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND m.id=$2::uuid FOR UPDATE`, tenant, id).Scan(&current)
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
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return event.AggregateVersion, nil
}

func (r *PostgresRepository) MatterByTriggerKey(ctx context.Context, tenant, triggerKey string) (Matter, error) {
	var id string
	err := r.pool.QueryRow(ctx, `SELECT m.id::text FROM matters m JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND m.trigger_key=$2 AND m.status NOT IN ('CLOSED','CANCELLED') ORDER BY m.created_at DESC LIMIT 1`, tenant, triggerKey).Scan(&id)
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
	query := `SELECT ce.id::text,t.slug,ce.aggregate_type,ce.aggregate_id::text,ce.aggregate_version,ce.event_type,ce.payload,ce.actor_type,COALESCE(ce.actor_id::text,''),ce.occurred_at
		FROM continuity_events ce JOIN tenants t ON t.id=ce.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND ce.aggregate_type='MATTER' AND ce.aggregate_id=$2::uuid`
	args := []any{tenant, id}
	if until != nil {
		query += ` AND ce.occurred_at<=$3`
		args = append(args, *until)
	}
	query += ` ORDER BY ce.aggregate_version`
	return r.scanEvents(ctx, query, args...)
}

func (r *PostgresRepository) OpenMatterCount(ctx context.Context, tenant, programID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT count(DISTINCT m.id) FROM matters m JOIN matter_links ml ON ml.matter_id=m.id AND ml.tenant_id=m.tenant_id JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND ml.program_id=$2::uuid AND m.status NOT IN ('CLOSED','CANCELLED')`, tenant, programID).Scan(&count)
	return count, err
}

func (r *PostgresRepository) LinkedProgramIDs(ctx context.Context, tenant, matterID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT ml.program_id::text FROM matter_links ml JOIN tenants t ON t.id=ml.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND ml.matter_id=$2::uuid AND ml.program_id IS NOT NULL ORDER BY ml.program_id::text`, tenant, matterID)
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
	case EventRequirementAdded:
		var v Requirement
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO program_requirements(id,tenant_id,program_id,source_id,code,title,statement,source_anchor,modality,actor,action,object_name,status,effective_from,effective_until,created_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,NULLIF($4,'')::uuid,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`, v.ID, v.TenantID, v.ProgramID, v.SourceID, v.Code, v.Title, v.Statement, v.SourceAnchor, v.Modality, v.Actor, v.Action, v.Object, v.Status, v.EffectiveFrom, v.EffectiveUntil, v.CreatedAt, v.Version)
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
	case EventRequirementControlLinked:
		var v RequirementControlLink
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO requirement_control_links(id,tenant_id,program_id,requirement_id,implementation_id,created_at) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,$5::uuid,$6)`, v.ID, v.TenantID, v.ProgramID, v.RequirementID, v.ImplementationID, v.CreatedAt)
		return err
	case EventEvidenceContractAdded:
		var v EvidenceContract
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		sources, _ := json.Marshal(v.AcceptableSourceIDs)
		_, err := tx.Exec(ctx, `INSERT INTO evidence_contracts(id,tenant_id,program_id,requirement_id,control_implementation_id,code,name,claim,acceptable_source_ids,population_scope,freshness_minutes,minimum_coverage,independence_required,contradiction_policy,failure_action,status,created_at,updated_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,NULLIF($4,'')::uuid,NULLIF($5,'')::uuid,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$17,$18)`, v.ID, v.TenantID, v.ProgramID, v.RequirementID, v.ControlImplementationID, v.Code, v.Name, v.Claim, sources, rawJSON(v.PopulationScope, `{}`), v.FreshnessMinutes, v.MinimumCoverage, v.IndependenceRequired, v.ContradictionPolicy, v.FailureAction, v.Status, v.CreatedAt, v.Version)
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
		_, err := tx.Exec(ctx, `INSERT INTO program_state_snapshots(id,tenant_id,program_id,overall_state,dimensions,reasons,open_matter_count,trigger_type,trigger_id,generated_at,program_version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,$9,$10,$11)`, v.ID, v.TenantID, v.ProgramID, v.Overall, dimensions, reasons, v.OpenMatterCount, v.TriggerType, v.TriggerID, v.GeneratedAt, v.ProgramVersion)
		return err
	case EventProgramTriggerRecorded:
		return nil
	default:
		return ErrInvalidState
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
	case EventMatterStateChanged:
		var v Matter
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE matters SET status=$3,priority=$4,title=$5,summary=$6,scope=$7,known_facts=$8,missing_facts=$9,contradictions=$10,owner_principal_id=NULLIF($11,'')::uuid,required_authority=$12,due_at=$13,closed_at=$14,closure_reason=$15,reopen_count=$16,updated_at=$17 WHERE id=$2::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)`, v.TenantID, v.ID, v.Status, v.Priority, v.Title, v.Summary, rawJSON(v.Scope, `{}`), rawJSON(v.KnownFacts, `{}`), rawJSON(v.MissingFacts, `[]`), rawJSON(v.Contradictions, `[]`), v.OwnerPrincipalID, v.RequiredAuthority, v.DueAt, v.ClosedAt, v.ClosureReason, v.ReopenCount, v.UpdatedAt)
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
		_, err := tx.Exec(ctx, `INSERT INTO matter_actions(id,tenant_id,matter_id,title,description,owner_principal_id,status,due_at,implemented_at,created_at,updated_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,NULLIF($6,'')::uuid,$7,$8,$9,$10,$10,$11)`, v.ID, v.TenantID, v.MatterID, v.Title, v.Description, v.OwnerPrincipalID, v.Status, v.DueAt, v.ImplementedAt, v.CreatedAt, v.Version)
		return err
	case EventActionStateChanged:
		var v Action
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE matter_actions SET status=$4,due_at=$5,implemented_at=$6,updated_at=$7,version=$8 WHERE id=$3::uuid AND matter_id=$2::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)`, v.TenantID, v.MatterID, v.ID, v.Status, v.DueAt, v.ImplementedAt, v.UpdatedAt, v.Version)
		return err
	case EventVerificationContractAdded:
		var v VerificationContract
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO verification_contracts(id,tenant_id,matter_id,action_id,expected_outcome,baseline,scope,measurement_source_id,threshold,observation_period_minutes,authority_principal_id,failure_response,status,created_at,updated_at,version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,NULLIF($4,'')::uuid,$5,$6,$7,NULLIF($8,'')::uuid,$9,$10,NULLIF($11,'')::uuid,$12,$13,$14,$14,$15)`, v.ID, v.TenantID, v.MatterID, v.ActionID, v.ExpectedOutcome, rawJSON(v.Baseline, `{}`), rawJSON(v.Scope, `{}`), v.MeasurementSourceID, rawJSON(v.Threshold, `{}`), v.ObservationPeriodMinutes, v.AuthorityPrincipalID, v.FailureResponse, v.Status, v.CreatedAt, v.Version)
		return err
	case EventVerificationResultRecorded:
		var v VerificationResult
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO verification_results(id,tenant_id,matter_id,contract_id,result,observations,evidence_references,reviewer_principal_id,rationale,observed_at,created_at) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,$5,$6,$7,NULLIF($8,'')::uuid,$9,$10,$11)`, v.ID, v.TenantID, v.MatterID, v.ContractID, v.Result, rawJSON(v.Observations, `{}`), rawJSON(v.EvidenceReferences, `[]`), v.ReviewerPrincipalID, v.Rationale, v.ObservedAt, v.CreatedAt)
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
