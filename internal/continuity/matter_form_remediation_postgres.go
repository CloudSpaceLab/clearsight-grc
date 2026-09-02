//go:build postgres

package continuity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) CreateMatterFormBinding(ctx context.Context, binding MatterFormRemediationBinding) (MatterFormRemediationBinding, error) {
	if binding.SubjectType != "MATTER" || binding.SubjectID != binding.MatterID || binding.Status != MatterFormBindingActive || binding.EffectiveFrom.IsZero() || strings.TrimSpace(binding.Purpose) == "" || binding.AudienceClass != "EXTERNAL" || strings.TrimSpace(binding.ResponderClass) == "" {
		return MatterFormRemediationBinding{}, ErrMatterFormBindingInvalid
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return MatterFormRemediationBinding{}, err
	}
	defer tx.Rollback(ctx)
	var matterVersion int64
	var legalEntityID string
	err = tx.QueryRow(ctx, `SELECT m.version,m.legal_entity_id::text
		FROM matters m JOIN tenants t ON t.id=m.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND m.id=$2::uuid AND m.status NOT IN ('CLOSED','CANCELLED')
		  AND EXISTS (SELECT 1 FROM matter_links ml WHERE ml.tenant_id=m.tenant_id AND ml.matter_id=m.id AND ml.program_id=$3::uuid AND ml.retired_at IS NULL)
		FOR UPDATE`, binding.TenantID, binding.MatterID, binding.ProgramID).Scan(&matterVersion, &legalEntityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return MatterFormRemediationBinding{}, ErrNotFound
	}
	if err != nil {
		return MatterFormRemediationBinding{}, err
	}
	if matterVersion != binding.MatterVersionAtBinding || legalEntityID != binding.LegalEntityID {
		return MatterFormRemediationBinding{}, ErrVersionConflict
	}
	var formStatus string
	var formCurrent, approvedUse bool
	err = tx.QueryRow(ctx, `SELECT f.status,f.is_current,f.approved_uses @> ARRAY['MATTER_REMEDIATION']::text[]
		FROM monitoring_form_templates f JOIN tenants t ON t.id=f.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND f.id=$2::uuid AND f.version=$3 AND f.legal_entity_id=$4::uuid FOR SHARE`,
		binding.TenantID, binding.FormTemplateID, binding.FormTemplateVersion, binding.LegalEntityID).Scan(&formStatus, &formCurrent, &approvedUse)
	if errors.Is(err, pgx.ErrNoRows) {
		return MatterFormRemediationBinding{}, ErrMatterFormBindingInvalid
	}
	if err != nil {
		return MatterFormRemediationBinding{}, err
	}
	if formStatus != "ACTIVE" || !formCurrent || !approvedUse {
		return MatterFormRemediationBinding{}, ErrMatterFormBindingInvalid
	}
	rows, err := tx.Query(ctx, `SELECT mappings FROM matter_form_remediation_bindings b JOIN tenants t ON t.id=b.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND b.matter_id=$2::uuid AND b.status='ACTIVE' FOR SHARE`, binding.TenantID, binding.MatterID)
	if err != nil {
		return MatterFormRemediationBinding{}, err
	}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			rows.Close()
			return MatterFormRemediationBinding{}, err
		}
		var existing []MatterFormFieldMapping
		if err := json.Unmarshal(raw, &existing); err != nil {
			rows.Close()
			return MatterFormRemediationBinding{}, err
		}
		for _, left := range existing {
			for _, right := range binding.Mappings {
				if strings.EqualFold(strings.TrimSpace(left.MissingItem), strings.TrimSpace(right.MissingItem)) {
					rows.Close()
					return MatterFormRemediationBinding{}, ErrMatterFormBindingInvalid
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return MatterFormRemediationBinding{}, err
	}
	rows.Close()
	mappings, err := json.Marshal(binding.Mappings)
	if err != nil {
		return MatterFormRemediationBinding{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO matter_form_remediation_bindings(
		id,tenant_id,legal_entity_id,program_id,matter_id,subject_type,subject_id,matter_version_at_binding,form_template_id,form_template_version,mappings,action_id,verification_contract_id,minimum_score,maximum_adverse_score,purpose,audience_class,responder_class,status,effective_from,created_by,created_at,version)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,$5::uuid,$6,$7::uuid,$8,$9::uuid,$10,$11,NULLIF($12,'')::uuid,$13::uuid,$14,$15,$16,$17,$18,$19,$20,$21::uuid,$22,$23)`,
		binding.ID, binding.TenantID, binding.LegalEntityID, binding.ProgramID, binding.MatterID, binding.SubjectType, binding.SubjectID, binding.MatterVersionAtBinding,
		binding.FormTemplateID, binding.FormTemplateVersion, mappings, binding.ActionID, binding.VerificationContractID,
		binding.MinimumScore, binding.MaximumAdverseScore, binding.Purpose, binding.AudienceClass, binding.ResponderClass, binding.Status, binding.EffectiveFrom, binding.CreatedBy, binding.CreatedAt, binding.Version)
	if err != nil {
		if isUniqueViolation(err) {
			return MatterFormRemediationBinding{}, ErrDuplicate
		}
		if isForeignKeyViolation(err) {
			return MatterFormRemediationBinding{}, ErrMatterFormBindingInvalid
		}
		return MatterFormRemediationBinding{}, err
	}
	eventPayload, _ := json.Marshal(binding)
	if _, err = tx.Exec(ctx, `INSERT INTO matter_form_remediation_events(tenant_id,binding_id,binding_version,event_type,actor_principal_id,occurred_at,payload)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3,'MATTER_FORM_REMEDIATION_BOUND',$4::uuid,$5,$6)`, binding.TenantID, binding.ID, binding.Version, binding.CreatedBy, binding.CreatedAt, eventPayload); err != nil {
		return MatterFormRemediationBinding{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER_FORM_REMEDIATION',$2::uuid,'MatterFormRemediationBound',$3,$4,$4,$4)`, binding.TenantID, binding.ID, eventPayload, binding.CreatedAt); err != nil {
		return MatterFormRemediationBinding{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return MatterFormRemediationBinding{}, err
	}
	return binding, nil
}

func (r *PostgresRepository) GetMatterFormBinding(ctx context.Context, tenant, matterID, bindingID string) (MatterFormRemediationBinding, error) {
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	row := r.pool.QueryRow(ctx, matterFormBindingSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND b.matter_id=$2::uuid AND b.id=$3::uuid AND (`+matterFormBindingVisibilitySQL+`)`, tenant, matterID, bindingID, enforce, actorTenant, actorEntity)
	return scanMatterFormBinding(row)
}

func (r *PostgresRepository) ListMatterFormBindings(ctx context.Context, tenant, matterID string, limit int) ([]MatterFormRemediationBinding, error) {
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	rows, err := r.pool.Query(ctx, matterFormBindingSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND b.matter_id=$2::uuid
		AND (NOT $3 OR ((t.id::text=$4 OR t.slug=$4) AND ($5='*' OR b.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=b.tenant_id AND (le.id::text=$5 OR le.code=$5) ORDER BY le.valid_from DESC,le.id LIMIT 1))))
		ORDER BY b.created_at DESC,b.id DESC LIMIT $6`, tenant, matterID, enforce, actorTenant, actorEntity, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []MatterFormRemediationBinding{}
	for rows.Next() {
		value, err := scanMatterFormBinding(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *PostgresRepository) GetMatterFormApplication(ctx context.Context, tenant, bindingID, responseRevisionID string) (MatterFormApplication, error) {
	row := r.pool.QueryRow(ctx, matterFormApplicationSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND a.binding_id=$2::uuid AND ($3='' OR a.response_revision_id::text=$3) ORDER BY a.applied_at DESC,a.id DESC LIMIT 1`, tenant, bindingID, responseRevisionID)
	return scanMatterFormApplication(row)
}

func (r *PostgresRepository) ApplyMatterFormApplication(ctx context.Context, command MatterFormApplicationCommand) (MatterAggregate, MatterFormApplication, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	defer tx.Rollback(ctx)
	if existing, readErr := scanMatterFormApplication(tx.QueryRow(ctx, matterFormApplicationSelect+` WHERE (t.id::text=$1 OR t.slug=$1) AND a.binding_id=$2::uuid AND a.response_revision_id=$3::uuid LIMIT 1`, command.Binding.TenantID, command.Binding.ID, command.ResponseRevisionID)); readErr == nil {
		return decorateMatter(command.Aggregate), existing, nil
	} else if !errors.Is(readErr, ErrNotFound) {
		return MatterAggregate{}, MatterFormApplication{}, readErr
	}
	var currentVersion int64
	err = tx.QueryRow(ctx, `SELECT m.version FROM matters m JOIN tenants t ON t.id=m.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND m.id=$2::uuid AND m.legal_entity_id=$3::uuid AND m.status NOT IN ('CLOSED','CANCELLED') FOR UPDATE`, command.Binding.TenantID, command.Binding.MatterID, command.Binding.LegalEntityID).Scan(&currentVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return MatterAggregate{}, MatterFormApplication{}, ErrNotFound
	}
	if err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	if currentVersion != command.ExpectedMatterVersion {
		return MatterAggregate{}, MatterFormApplication{}, ErrVersionConflict
	}
	var distributionID, submissionID string
	var responseRevision int64
	var answersRaw []byte
	err = tx.QueryRow(ctx, `SELECT d.id::text,r.submission_id::text,r.revision,s.answers
		FROM capture_response_revisions r
		JOIN tenants t ON t.id=r.tenant_id
		JOIN capture_form_distributions d ON d.id=r.distribution_id AND d.tenant_id=r.tenant_id AND d.legal_entity_id=r.legal_entity_id
		JOIN capture_submissions s ON s.id=r.submission_id AND s.tenant_id=r.tenant_id AND s.distribution_id=r.distribution_id
		JOIN capture_requests q ON q.id=s.request_id AND q.tenant_id=s.tenant_id AND q.distribution_id=d.id
		WHERE (t.id::text=$1 OR t.slug=$1) AND r.id=$2::uuid AND r.is_current AND r.state='FINAL'
		  AND d.legal_entity_id=$3::uuid AND d.subject_type='MATTER' AND d.subject_id=$4::uuid
		  AND d.form_template_id=$5::uuid AND d.form_template_version=$6
		  AND q.origin_type=$7 AND q.origin_id=$8 AND q.origin_version=$9
		FOR SHARE OF r,d,s,q`, command.Binding.TenantID, command.ResponseRevisionID, command.Binding.LegalEntityID, command.Binding.MatterID,
		command.Binding.FormTemplateID, command.Binding.FormTemplateVersion, MatterFormRemediationOrigin, command.Binding.ID, command.Binding.Version).Scan(&distributionID, &submissionID, &responseRevision, &answersRaw)
	if errors.Is(err, pgx.ErrNoRows) {
		return MatterAggregate{}, MatterFormApplication{}, ErrMatterFormResponseRejected
	}
	if err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	var answers map[string]formcontract.AnswerValue
	if err := json.Unmarshal(answersRaw, &answers); err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	updatedMatter, applied, err := applyMatterFormAnswers(command.Aggregate.Matter, command.Binding, answers, command.ResponseRevisionID)
	if err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	updatedMatter.Version = currentVersion + 1
	updatedMatter.UpdatedAt = command.AppliedAt
	application := MatterFormApplication{ID: command.ApplicationID, TenantID: command.Binding.TenantID, LegalEntityID: command.Binding.LegalEntityID, BindingID: command.Binding.ID, BindingVersion: command.Binding.Version, MatterID: command.Binding.MatterID, MatterVersion: updatedMatter.Version, DistributionID: distributionID, ResponseRevisionID: command.ResponseRevisionID, ResponseRevision: responseRevision, SubmissionID: submissionID, VerificationContractID: command.Binding.VerificationContractID, AppliedFieldIDs: applied, AppliedBy: command.ActorID, AppliedAt: command.AppliedAt}
	payload, err := json.Marshal(matterFormResponseAppliedEvent{Matter: updatedMatter, Application: application, Rationale: command.Rationale})
	if err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	event := Event{ID: command.EventID, TenantID: command.Binding.TenantID, AggregateType: "MATTER", AggregateID: command.Binding.MatterID, AggregateVersion: updatedMatter.Version, Type: EventMatterFormApplied, Payload: payload, ActorType: actorFor(command.ActorID), ActorID: command.ActorID, OccurredAt: command.AppliedAt}
	if err := applyMatterProjection(ctx, tx, event); err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	result, err := tx.Exec(ctx, `UPDATE matters SET version=$3,updated_at=$4 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id=$2::uuid AND version=$5`, command.Binding.TenantID, command.Binding.MatterID, updatedMatter.Version, command.AppliedAt, currentVersion)
	if err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	if result.RowsAffected() != 1 {
		return MatterAggregate{}, MatterFormApplication{}, ErrVersionConflict
	}
	appliedRaw, _ := json.Marshal(applied)
	_, err = tx.Exec(ctx, `INSERT INTO matter_form_remediation_applications(id,tenant_id,legal_entity_id,binding_id,binding_version,matter_id,matter_version,distribution_id,response_revision_id,response_revision,submission_id,verification_contract_id,applied_field_ids,applied_by,applied_at)
		VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4::uuid,$5,$6::uuid,$7,$8::uuid,$9::uuid,$10,$11::uuid,$12::uuid,$13,$14::uuid,$15)`,
		application.ID, application.TenantID, application.LegalEntityID, application.BindingID, application.BindingVersion, application.MatterID, application.MatterVersion, application.DistributionID, application.ResponseRevisionID, application.ResponseRevision, application.SubmissionID, application.VerificationContractID, appliedRaw, application.AppliedBy, application.AppliedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return MatterAggregate{}, MatterFormApplication{}, ErrVersionConflict
		}
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	if err := insertContinuityEvent(ctx, tx, event); err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	if err := insertOutbox(ctx, tx, event); err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	// The linked form supplies evidence; it does not prove the outcome. Queue
	// an exact, due-dated Matter reconciliation in this same transaction so the
	// configured verification contract becomes current-authority work after its
	// observation period. Action implementation events also reconcile the same
	// contract, making delayed worker delivery idempotent.
	if _, err := tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
		SELECT b.tenant_id,'MATTER',b.matter_id,$4,
		       jsonb_build_object('version',$5::bigint,'binding_id',b.id::text,'application_id',$6::text,'verification_contract_id',vc.id::text),
		       $7,GREATEST($7,COALESCE(ma.implemented_at,vc.created_at)+make_interval(mins=>vc.observation_period_minutes)),
		       GREATEST($7,COALESCE(ma.implemented_at,vc.created_at)+make_interval(mins=>vc.observation_period_minutes))
		FROM matter_form_remediation_bindings b
		JOIN verification_contracts vc ON vc.tenant_id=b.tenant_id AND vc.matter_id=b.matter_id AND vc.id=b.verification_contract_id AND vc.status='ACTIVE'
		LEFT JOIN matter_actions ma ON ma.tenant_id=vc.tenant_id AND ma.matter_id=vc.matter_id AND ma.id=vc.action_id
		WHERE (b.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)) AND b.id=$2::uuid AND b.matter_id=$3::uuid
		ON CONFLICT DO NOTHING`, application.TenantID, application.BindingID, application.MatterID, EventMatterFormVerificationDue, application.MatterVersion, application.ID, application.AppliedAt); err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	if _, err := queueProgramStateTx(ctx, tx, application.TenantID, command.Binding.ProgramID, 0, event.Type, application.MatterID, application.AppliedBy, application.AppliedAt); err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	if err := r.commitContinuityEvents(ctx, tx, event); err != nil {
		return MatterAggregate{}, MatterFormApplication{}, err
	}
	command.Aggregate.Matter = updatedMatter
	command.Aggregate.Closure = assessClosure(command.Aggregate)
	return decorateMatter(command.Aggregate), application, nil
}

const matterFormBindingSelect = `SELECT b.id::text,t.slug,b.legal_entity_id::text,b.program_id::text,b.matter_id::text,b.subject_type,b.subject_id::text,b.matter_version_at_binding,b.form_template_id::text,b.form_template_version,b.mappings,COALESCE(b.action_id::text,''),b.verification_contract_id::text,b.minimum_score,b.maximum_adverse_score,b.purpose,b.audience_class,b.responder_class,b.status,b.effective_from,b.created_by::text,b.created_at,b.version
	FROM matter_form_remediation_bindings b JOIN tenants t ON t.id=b.tenant_id JOIN matters m ON m.tenant_id=b.tenant_id AND m.id=b.matter_id`

const matterFormBindingVisibilitySQL = `NOT $4 OR ((t.id::text=$5 OR t.slug=$5) AND ($6='*' OR b.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=b.tenant_id AND (le.id::text=$6 OR le.code=$6) ORDER BY le.valid_from DESC,le.id LIMIT 1)))`

func scanMatterFormBinding(row pgx.Row) (MatterFormRemediationBinding, error) {
	var value MatterFormRemediationBinding
	var mappings []byte
	var minimum, maximum sql.NullFloat64
	err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.ProgramID, &value.MatterID, &value.SubjectType, &value.SubjectID, &value.MatterVersionAtBinding, &value.FormTemplateID, &value.FormTemplateVersion, &mappings, &value.ActionID, &value.VerificationContractID, &minimum, &maximum, &value.Purpose, &value.AudienceClass, &value.ResponderClass, &value.Status, &value.EffectiveFrom, &value.CreatedBy, &value.CreatedAt, &value.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return MatterFormRemediationBinding{}, ErrNotFound
	}
	if err != nil {
		return MatterFormRemediationBinding{}, err
	}
	if err := json.Unmarshal(mappings, &value.Mappings); err != nil {
		return MatterFormRemediationBinding{}, err
	}
	if minimum.Valid {
		value.MinimumScore = &minimum.Float64
	}
	if maximum.Valid {
		value.MaximumAdverseScore = &maximum.Float64
	}
	return value, nil
}

const matterFormApplicationSelect = `SELECT a.id::text,t.slug,a.legal_entity_id::text,a.binding_id::text,a.binding_version,a.matter_id::text,a.matter_version,a.distribution_id::text,a.response_revision_id::text,a.response_revision,a.submission_id::text,a.verification_contract_id::text,a.applied_field_ids,a.applied_by::text,a.applied_at FROM matter_form_remediation_applications a JOIN tenants t ON t.id=a.tenant_id`

func scanMatterFormApplication(row pgx.Row) (MatterFormApplication, error) {
	var value MatterFormApplication
	var fields []byte
	err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.BindingID, &value.BindingVersion, &value.MatterID, &value.MatterVersion, &value.DistributionID, &value.ResponseRevisionID, &value.ResponseRevision, &value.SubmissionID, &value.VerificationContractID, &fields, &value.AppliedBy, &value.AppliedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return MatterFormApplication{}, ErrNotFound
	}
	if err != nil {
		return MatterFormApplication{}, err
	}
	if err := json.Unmarshal(fields, &value.AppliedFieldIDs); err != nil {
		return MatterFormApplication{}, err
	}
	return value, nil
}

var _ MatterFormRemediationRepository = (*PostgresRepository)(nil)
