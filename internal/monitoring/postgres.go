//go:build postgres

package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateFormRevision(ctx context.Context, value FormTemplate) (FormTemplate, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return FormTemplate{}, mapPostgresError(err)
	}
	defer tx.Rollback(ctx)
	created, err := insertFormRevision(ctx, tx, value)
	if err != nil {
		return FormTemplate{}, err
	}
	event, err := newMonitoringEvent(created.TenantID, AggregateMonitoringForm, created.ID, created.Version, EventMonitoringFormCreated, created, created.CreatedBy, created.CreatedAt)
	if err != nil {
		return FormTemplate{}, err
	}
	if err := insertMonitoringEventAndOutbox(ctx, tx, event); err != nil {
		return FormTemplate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return FormTemplate{}, mapPostgresError(err)
	}
	return created, nil
}

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func insertFormRevision(ctx context.Context, db queryRower, value FormTemplate) (FormTemplate, error) {
	presentation, err := json.Marshal(value.Presentation)
	if err != nil {
		return FormTemplate{}, errors.Join(ErrInvalid, err)
	}
	sections, err := json.Marshal(value.Sections)
	if err != nil {
		return FormTemplate{}, errors.Join(ErrInvalid, err)
	}
	fields, err := json.Marshal(value.Fields)
	if err != nil {
		return FormTemplate{}, errors.Join(ErrInvalid, err)
	}
	created, err := scanForm(db.QueryRow(ctx, `
		INSERT INTO monitoring_form_templates(id,tenant_id,legal_entity_id,program_id,code,name,purpose,fields,status,is_current,effective_from,effective_until,version,created_by,submitted_by,approved_by,rejected_by,created_at,updated_at)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),NULLIF($3,'')::uuid,NULLIF($4,'')::uuid,$5,$6,$7,$8::jsonb,$9,$10,$11,$12,$13,NULLIF($14,'')::uuid,NULLIF($15,'')::uuid,NULLIF($16,'')::uuid,NULLIF($17,'')::uuid,$18,$19)
		RETURNING id::text,tenant_id::text,COALESCE(legal_entity_id::text,''),COALESCE(program_id::text,''),code,name,purpose,fields,status,is_current,effective_from,effective_until,version,COALESCE(created_by::text,''),COALESCE(submitted_by::text,''),COALESCE(approved_by::text,''),COALESCE(rejected_by::text,''),created_at,updated_at`,
		value.ID, value.TenantID, value.LegalEntityID, value.ProgramID, value.Code, value.Name, value.Purpose, fields, value.Status, value.IsCurrent, value.EffectiveFrom, value.EffectiveUntil, value.Version, value.CreatedBy, value.SubmittedBy, value.ApprovedBy, value.RejectedBy, value.CreatedAt, value.UpdatedAt))
	return created, mapPostgresError(err)
}

func (r *PostgresRepository) TransitionForm(ctx context.Context, input LifecycleTransition) (FormTemplate, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return FormTemplate{}, mapPostgresError(err)
	}
	defer tx.Rollback(ctx)
	current, err := scanForm(tx.QueryRow(ctx, `
		SELECT f.id::text,f.tenant_id::text,COALESCE(f.legal_entity_id::text,''),COALESCE(f.program_id::text,''),f.code,f.name,f.purpose,f.fields,f.status,f.is_current,f.effective_from,f.effective_until,f.version,
		       COALESCE(f.created_by::text,''),COALESCE(f.submitted_by::text,''),COALESCE(f.approved_by::text,''),COALESCE(f.rejected_by::text,''),f.created_at,f.updated_at
		FROM monitoring_form_templates f JOIN tenants t ON t.id=f.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND f.id=$2::uuid AND f.version=$3
		  AND f.legal_entity_id=$4::uuid AND f.program_id=$5::uuid
		FOR UPDATE`, input.TenantID, input.ID, input.ExpectedVersion, input.LegalEntityID, input.ProgramID))
	if err != nil {
		return FormTemplate{}, mapPostgresError(err)
	}
	nextLifecycle, err := transitionLifecycle(current.Lifecycle, input)
	if err != nil {
		return FormTemplate{}, err
	}
	if nextLifecycle.IsCurrent || current.IsCurrent {
		_, err = tx.Exec(ctx, `UPDATE monitoring_form_templates SET status='RETIRED',is_current=false,effective_until=$5,updated_at=$5 WHERE tenant_id=$1::uuid AND id=$2::uuid AND legal_entity_id=$3::uuid AND program_id=$4::uuid AND is_current`, current.TenantID, current.ID, current.LegalEntityID, current.ProgramID, input.At.UTC())
		if err != nil {
			return FormTemplate{}, mapPostgresError(err)
		}
	}
	next := current
	next.Lifecycle = nextLifecycle
	created, err := insertFormRevision(ctx, tx, next)
	if err != nil {
		return FormTemplate{}, err
	}
	event, err := newMonitoringEvent(created.TenantID, AggregateMonitoringForm, created.ID, created.Version, EventMonitoringFormStateChanged, created, input.ActorID, input.At)
	if err != nil {
		return FormTemplate{}, err
	}
	if err := insertMonitoringEventAndOutbox(ctx, tx, event); err != nil {
		return FormTemplate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return FormTemplate{}, mapPostgresError(err)
	}
	return created, nil
}

const formRevisionSQL = `
	SELECT f.id::text,f.tenant_id::text,COALESCE(f.legal_entity_id::text,''),COALESCE(f.program_id::text,''),f.code,f.name,f.purpose,f.fields,f.status,f.is_current,f.effective_from,f.effective_until,f.version,
	       COALESCE(f.created_by::text,''),COALESCE(f.submitted_by::text,''),COALESCE(f.approved_by::text,''),COALESCE(f.rejected_by::text,''),f.created_at,f.updated_at
	FROM monitoring_form_templates f JOIN tenants t ON t.id=f.tenant_id
	WHERE (t.id::text=$1 OR t.slug=$1) AND f.legal_entity_id=$2::uuid AND f.program_id=$3::uuid
	  AND f.id=$4::uuid AND f.version=$5`

func (r *PostgresRepository) FormRevision(ctx context.Context, tenant, legalEntityID, programID, id string, version int64) (FormTemplate, error) {
	value, err := scanForm(r.pool.QueryRow(ctx, formRevisionSQL, tenant, legalEntityID, programID, id, version))
	return value, mapPostgresError(err)
}

const listFormRevisionsSQL = `
	SELECT f.id::text,f.tenant_id::text,COALESCE(f.legal_entity_id::text,''),COALESCE(f.program_id::text,''),f.code,f.name,f.purpose,f.fields,f.status,f.is_current,f.effective_from,f.effective_until,f.version,
	       COALESCE(f.created_by::text,''),COALESCE(f.submitted_by::text,''),COALESCE(f.approved_by::text,''),COALESCE(f.rejected_by::text,''),f.created_at,f.updated_at
	FROM monitoring_form_templates f JOIN tenants t ON t.id=f.tenant_id
	WHERE (t.id::text=$1 OR t.slug=$1) AND f.legal_entity_id=$2::uuid AND f.program_id=$3::uuid
	ORDER BY f.code,f.version DESC,f.id LIMIT $4`

func (r *PostgresRepository) ListFormRevisions(ctx context.Context, tenant, legalEntityID, programID string, limit int) ([]FormTemplate, error) {
	rows, err := r.pool.Query(ctx, listFormRevisionsSQL, tenant, legalEntityID, programID, boundedLimit(limit))
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()
	values := make([]FormTemplate, 0)
	for rows.Next() {
		value, scanErr := scanForm(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *PostgresRepository) CreateCheckRevision(ctx context.Context, value MonitoringCheck) (MonitoringCheck, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MonitoringCheck{}, mapPostgresError(err)
	}
	defer tx.Rollback(ctx)
	created, err := insertCheckRevision(ctx, tx, value)
	if err != nil {
		return MonitoringCheck{}, err
	}
	event, err := newMonitoringEvent(created.TenantID, AggregateMonitoringCheck, created.ID, created.Version, EventMonitoringCheckCreated, created, created.CreatedBy, created.CreatedAt)
	if err != nil {
		return MonitoringCheck{}, err
	}
	if err := insertMonitoringEventAndOutbox(ctx, tx, event); err != nil {
		return MonitoringCheck{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MonitoringCheck{}, mapPostgresError(err)
	}
	return created, nil
}

func insertCheckRevision(ctx context.Context, db queryRower, value MonitoringCheck) (MonitoringCheck, error) {
	rules, err := json.Marshal(value.SourceRules)
	if err != nil {
		return MonitoringCheck{}, errors.Join(ErrInvalid, err)
	}
	thresholds, err := json.Marshal(value.Thresholds)
	if err != nil {
		return MonitoringCheck{}, errors.Join(ErrInvalid, err)
	}
	created, err := scanCheck(db.QueryRow(ctx, `
		INSERT INTO monitoring_checks(id,tenant_id,program_id,requirement_id,control_implementation_id,evidence_contract_id,code,name,claim,input_kind,form_template_id,form_template_version,binding_id,binding_version,source_rules,thresholds,freshness_minutes,minimum_coverage,owner_principal_id,reviewer_principal_id,failure_action,status,is_current,effective_from,effective_until,version,created_by,submitted_by,approved_by,rejected_by,created_at,updated_at)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,NULLIF($4,'')::uuid,NULLIF($5,'')::uuid,NULLIF($6,'')::uuid,$7,$8,$9,$10,NULLIF($11,'')::uuid,NULLIF($12,0),NULLIF($13,'')::uuid,NULLIF($14,0),$15::jsonb,$16::jsonb,$17,$18,NULLIF($19,'')::uuid,NULLIF($20,'')::uuid,$21,$22,$23,$24,$25,$26,NULLIF($27,'')::uuid,NULLIF($28,'')::uuid,NULLIF($29,'')::uuid,NULLIF($30,'')::uuid,$31,$32)
		RETURNING id::text,tenant_id::text,program_id::text,COALESCE(requirement_id::text,''),COALESCE(control_implementation_id::text,''),COALESCE(evidence_contract_id::text,''),code,name,claim,input_kind,COALESCE(form_template_id::text,''),COALESCE(form_template_version,0),COALESCE(binding_id::text,''),COALESCE(binding_version,0),source_rules,thresholds,freshness_minutes,minimum_coverage,COALESCE(owner_principal_id::text,''),COALESCE(reviewer_principal_id::text,''),failure_action,status,is_current,effective_from,effective_until,version,COALESCE(created_by::text,''),COALESCE(submitted_by::text,''),COALESCE(approved_by::text,''),COALESCE(rejected_by::text,''),created_at,updated_at`,
		value.ID, value.TenantID, value.ProgramID, value.RequirementID, value.ControlImplementationID, value.EvidenceContractID, value.Code, value.Name, value.Claim, value.InputKind, value.FormTemplateID, value.FormTemplateVersion, value.BindingID, value.BindingVersion, rules, thresholds, value.FreshnessMinutes, value.MinimumCoverage, value.OwnerPrincipalID, value.ReviewerPrincipalID, value.FailureAction, value.Status, value.IsCurrent, value.EffectiveFrom, value.EffectiveUntil, value.Version, value.CreatedBy, value.SubmittedBy, value.ApprovedBy, value.RejectedBy, value.CreatedAt, value.UpdatedAt))
	return created, mapPostgresError(err)
}

func (r *PostgresRepository) TransitionCheck(ctx context.Context, input LifecycleTransition) (MonitoringCheck, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MonitoringCheck{}, mapPostgresError(err)
	}
	defer tx.Rollback(ctx)
	current, err := scanCheck(tx.QueryRow(ctx, checkSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND c.id=$2::uuid AND c.version=$3 FOR UPDATE`, input.TenantID, input.ID, input.ExpectedVersion))
	if err != nil {
		return MonitoringCheck{}, mapPostgresError(err)
	}
	nextLifecycle, err := transitionLifecycle(current.Lifecycle, input)
	if err != nil {
		return MonitoringCheck{}, err
	}
	if nextLifecycle.IsCurrent || current.IsCurrent {
		_, err = tx.Exec(ctx, `UPDATE monitoring_checks SET status='RETIRED',is_current=false,effective_until=$3,updated_at=$3 WHERE tenant_id=$1::uuid AND id=$2::uuid AND is_current`, current.TenantID, current.ID, input.At.UTC())
		if err != nil {
			return MonitoringCheck{}, mapPostgresError(err)
		}
	}
	next := current
	next.Lifecycle = nextLifecycle
	created, err := insertCheckRevision(ctx, tx, next)
	if err != nil {
		return MonitoringCheck{}, err
	}
	event, err := newMonitoringEvent(created.TenantID, AggregateMonitoringCheck, created.ID, created.Version, EventMonitoringCheckStateChanged, created, input.ActorID, input.At)
	if err != nil {
		return MonitoringCheck{}, err
	}
	if err := insertMonitoringEventAndOutbox(ctx, tx, event); err != nil {
		return MonitoringCheck{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MonitoringCheck{}, mapPostgresError(err)
	}
	return created, nil
}

func (r *PostgresRepository) CheckRevision(ctx context.Context, tenant, id string, version int64) (MonitoringCheck, error) {
	value, err := scanCheck(r.pool.QueryRow(ctx, checkSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND c.id=$2::uuid AND c.version=$3`, tenant, id, version))
	return value, mapPostgresError(err)
}

func (r *PostgresRepository) LatestCheckRevision(ctx context.Context, tenant, id string) (MonitoringCheck, error) {
	value, err := scanCheck(r.pool.QueryRow(ctx, checkSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND c.id=$2::uuid ORDER BY c.version DESC LIMIT 1`, tenant, id))
	return value, mapPostgresError(err)
}

func (r *PostgresRepository) ListCheckRevisions(ctx context.Context, tenant, programID string, limit int) ([]MonitoringCheck, error) {
	rows, err := r.pool.Query(ctx, checkSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND c.program_id=$2::uuid ORDER BY c.code,c.version DESC,c.id LIMIT $3`, tenant, programID, boundedLimit(limit))
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()
	values := make([]MonitoringCheck, 0)
	for rows.Next() {
		value, scanErr := scanCheck(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *PostgresRepository) AppendResult(ctx context.Context, value MonitoringResult) (MonitoringResult, error) {
	evaluation, err := json.Marshal(value.Evaluation)
	if err != nil {
		return MonitoringResult{}, errors.Join(ErrInvalid, err)
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MonitoringResult{}, mapPostgresError(err)
	}
	defer tx.Rollback(ctx)
	created, err := scanResult(tx.QueryRow(ctx, `
		INSERT INTO monitoring_results(id,tenant_id,program_id,monitoring_check_id,monitoring_check_version,input_kind,input_reference_id,input_reference_version,evaluation,source_receipt,submission_provenance,evaluated_at,evaluator_version,created_at)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,$5,$6,$7,$8,$9::jsonb,$10::jsonb,$11::jsonb,$12,$13,$14)
		RETURNING id::text,tenant_id::text,program_id::text,monitoring_check_id::text,monitoring_check_version,input_kind,input_reference_id,input_reference_version,evaluation,source_receipt,submission_provenance,evaluated_at,evaluator_version,created_at`,
		value.ID, value.TenantID, value.ProgramID, value.MonitoringCheckID, value.MonitoringCheckVersion, value.InputKind, value.InputReferenceID, value.InputReferenceVersion, evaluation, nullableJSON(value.SourceReceipt), nullableJSON(value.SubmissionProvenance), value.EvaluatedAt, value.EvaluatorVersion, value.CreatedAt))
	if err != nil {
		return MonitoringResult{}, mapPostgresError(err)
	}
	event, err := newMonitoringEvent(created.TenantID, AggregateMonitoringResult, created.ID, 1, EventMonitoringResultRecorded, created, "", created.CreatedAt)
	if err != nil {
		return MonitoringResult{}, err
	}
	if err := insertMonitoringEventAndOutbox(ctx, tx, event); err != nil {
		return MonitoringResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return MonitoringResult{}, mapPostgresError(err)
	}
	return created, nil
}

func insertMonitoringEventAndOutbox(ctx context.Context, tx pgx.Tx, event MonitoringEvent) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO monitoring_events(id,tenant_id,aggregate_type,aggregate_id,aggregate_version,event_type,payload,actor_id,occurred_at)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4::uuid,$5,$6,$7::jsonb,NULLIF($8,'')::uuid,$9)`,
		event.ID, event.TenantID, event.AggregateType, event.AggregateID, event.AggregateVersion, event.Type, event.Payload, event.ActorID, event.OccurredAt); err != nil {
		return mapPostgresError(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3::uuid,$4,$5::jsonb,$6,$6,$6)`,
		event.TenantID, event.AggregateType, event.AggregateID, event.Type, event.Payload, event.OccurredAt); err != nil {
		return mapPostgresError(err)
	}
	return nil
}

func (r *PostgresRepository) ListResults(ctx context.Context, tenant, checkID string, limit int) ([]MonitoringResult, error) {
	rows, err := r.pool.Query(ctx, resultSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND r.monitoring_check_id=$2::uuid ORDER BY r.evaluated_at DESC,r.id DESC LIMIT $3`, tenant, checkID, boundedLimit(limit))
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()
	values := make([]MonitoringResult, 0)
	for rows.Next() {
		value, scanErr := scanResult(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

const resultByIDSQL = resultSelect + `
	WHERE r.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
	  AND r.id=$2::uuid`

func (r *PostgresRepository) Result(ctx context.Context, tenant, id string) (MonitoringResult, error) {
	value, err := scanResult(r.pool.QueryRow(ctx, resultByIDSQL, tenant, id))
	return value, mapPostgresError(err)
}

const checkSelect = `SELECT c.id::text,c.tenant_id::text,c.program_id::text,COALESCE(c.requirement_id::text,''),COALESCE(c.control_implementation_id::text,''),COALESCE(c.evidence_contract_id::text,''),c.code,c.name,c.claim,c.input_kind,COALESCE(c.form_template_id::text,''),COALESCE(c.form_template_version,0),COALESCE(c.binding_id::text,''),COALESCE(c.binding_version,0),c.source_rules,c.thresholds,c.freshness_minutes,c.minimum_coverage,COALESCE(c.owner_principal_id::text,''),COALESCE(c.reviewer_principal_id::text,''),c.failure_action,c.status,c.is_current,c.effective_from,c.effective_until,c.version,COALESCE(c.created_by::text,''),COALESCE(c.submitted_by::text,''),COALESCE(c.approved_by::text,''),COALESCE(c.rejected_by::text,''),c.created_at,c.updated_at FROM monitoring_checks c JOIN tenants t ON t.id=c.tenant_id`
const resultSelect = `SELECT r.id::text,r.tenant_id::text,r.program_id::text,r.monitoring_check_id::text,r.monitoring_check_version,r.input_kind,r.input_reference_id,r.input_reference_version,r.evaluation,r.source_receipt,r.submission_provenance,r.evaluated_at,r.evaluator_version,r.created_at FROM monitoring_results r JOIN tenants t ON t.id=r.tenant_id`
const formProjection = `f.id::text,f.tenant_id::text,f.code,f.name,f.purpose,f.presentation,f.sections,f.fields,f.status,f.is_current,f.effective_from,f.effective_until,f.version,
	COALESCE(f.created_by::text,''),COALESCE(f.submitted_by::text,''),COALESCE(f.approved_by::text,''),COALESCE(f.rejected_by::text,''),f.created_at,f.updated_at`

type scanner interface{ Scan(...any) error }

func scanForm(row scanner) (FormTemplate, error) {
	var value FormTemplate
	var fields []byte
	err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.ProgramID, &value.Code, &value.Name, &value.Purpose, &fields, &value.Status, &value.IsCurrent, &value.EffectiveFrom, &value.EffectiveUntil, &value.Version, &value.CreatedBy, &value.SubmittedBy, &value.ApprovedBy, &value.RejectedBy, &value.CreatedAt, &value.UpdatedAt)
	if err != nil {
		return FormTemplate{}, err
	}
	if err := json.Unmarshal(presentation, &value.Presentation); err != nil {
		return FormTemplate{}, err
	}
	if err := json.Unmarshal(sections, &value.Sections); err != nil {
		return FormTemplate{}, err
	}
	if err := json.Unmarshal(fields, &value.Fields); err != nil {
		return FormTemplate{}, err
	}
	return value, nil
}

func scanCheck(row scanner) (MonitoringCheck, error) {
	var value MonitoringCheck
	var rules, thresholds []byte
	err := row.Scan(&value.ID, &value.TenantID, &value.ProgramID, &value.RequirementID, &value.ControlImplementationID, &value.EvidenceContractID, &value.Code, &value.Name, &value.Claim, &value.InputKind, &value.FormTemplateID, &value.FormTemplateVersion, &value.BindingID, &value.BindingVersion, &rules, &thresholds, &value.FreshnessMinutes, &value.MinimumCoverage, &value.OwnerPrincipalID, &value.ReviewerPrincipalID, &value.FailureAction, &value.Status, &value.IsCurrent, &value.EffectiveFrom, &value.EffectiveUntil, &value.Version, &value.CreatedBy, &value.SubmittedBy, &value.ApprovedBy, &value.RejectedBy, &value.CreatedAt, &value.UpdatedAt)
	if err != nil {
		return MonitoringCheck{}, err
	}
	if err := json.Unmarshal(rules, &value.SourceRules); err != nil {
		return MonitoringCheck{}, err
	}
	if err := json.Unmarshal(thresholds, &value.Thresholds); err != nil {
		return MonitoringCheck{}, err
	}
	return value, nil
}

func scanResult(row scanner) (MonitoringResult, error) {
	var value MonitoringResult
	var evaluation, sourceReceipt, submissionProvenance []byte
	err := row.Scan(&value.ID, &value.TenantID, &value.ProgramID, &value.MonitoringCheckID, &value.MonitoringCheckVersion, &value.InputKind, &value.InputReferenceID, &value.InputReferenceVersion, &evaluation, &sourceReceipt, &submissionProvenance, &value.EvaluatedAt, &value.EvaluatorVersion, &value.CreatedAt)
	if err != nil {
		return MonitoringResult{}, err
	}
	if err := json.Unmarshal(evaluation, &value.Evaluation); err != nil {
		return MonitoringResult{}, err
	}
	value.SourceReceipt = append([]byte(nil), sourceReceipt...)
	value.SubmissionProvenance = append([]byte(nil), submissionProvenance...)
	return value, nil
}

func mapPostgresError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) {
		switch databaseError.Code {
		case "23505":
			return ErrConflict
		case "23503", "23514", "22P02":
			return errors.Join(ErrInvalid, err)
		}
	}
	return fmt.Errorf("monitoring storage: %w", err)
}

func boundedLimit(limit int) int {
	if limit < 1 || limit > 500 {
		return 100
	}
	return limit
}

func nullableJSON(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

var _ Repository = (*PostgresRepository)(nil)
