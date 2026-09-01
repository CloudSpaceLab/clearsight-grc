//go:build postgres

package formpolicy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

type postgresRow interface{ Scan(...any) error }

const postgresPolicyColumns = `p.id::text,p.tenant_id::text,p.legal_entity_id::text,p.code,p.name,p.purpose,p.action_class,p.automation_policy_id::text,p.automation_policy_version,p.eligibility,p.matter_action,p.blast_radius,p.outcome_contract,p.rollout_mode,p.status,p.maker_id::text,COALESCE(p.checker_id::text,''),p.checksum,COALESCE(p.approved_simulation_id::text,''),COALESCE(p.supersedes_policy_id::text,''),COALESCE(p.rollback_of_policy_id::text,''),p.effective_from,p.effective_until,p.submitted_at,p.approved_at,p.activated_at,p.suspended_at,p.retired_at,p.version,p.record_version,p.created_at,p.updated_at`

func scanPostgresPolicy(row postgresRow) (Policy, error) {
	var value Policy
	var eligibility, action, blast, outcome []byte
	var rollout, status string
	err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.Code, &value.Name, &value.Purpose, &value.ActionClass,
		&value.AutomationPolicyID, &value.AutomationPolicyVersion, &eligibility, &action, &blast, &outcome, &rollout, &status,
		&value.MakerID, &value.CheckerID, &value.Checksum, &value.ApprovedSimulationID, &value.SupersedesPolicyID, &value.RollbackOfPolicyID,
		&value.EffectiveFrom, &value.EffectiveUntil, &value.SubmittedAt, &value.ApprovedAt, &value.ActivatedAt, &value.SuspendedAt, &value.RetiredAt,
		&value.Version, &value.RecordVersion, &value.CreatedAt, &value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Policy{}, ErrNotFound
	}
	if err != nil {
		return Policy{}, err
	}
	if json.Unmarshal(eligibility, &value.Eligibility) != nil || json.Unmarshal(action, &value.Action) != nil || json.Unmarshal(blast, &value.BlastRadius) != nil || json.Unmarshal(outcome, &value.Outcome) != nil {
		return Policy{}, fmt.Errorf("%w: stored policy definition cannot be decoded", ErrInvalid)
	}
	value.Rollout, value.Status = RolloutMode(rollout), PolicyStatus(status)
	if err := validateStoredPolicy(value); err != nil {
		return Policy{}, err
	}
	return value, nil
}

func validateStoredPolicy(value Policy) error {
	input := CreateInput{Code: value.Code, Name: value.Name, Purpose: value.Purpose, AutomationPolicyID: value.AutomationPolicyID, AutomationPolicyVersion: value.AutomationPolicyVersion, Eligibility: value.Eligibility, Action: value.Action, BlastRadius: value.BlastRadius, Outcome: value.Outcome, Rollout: value.Rollout, EffectiveFrom: value.EffectiveFrom, EffectiveUntil: value.EffectiveUntil}
	normalized := input
	if err := normalizeCreateInput(&normalized, value.CreatedAt); err != nil || !reflect.DeepEqual(normalized, input) || value.ActionClass != ActionClassCreateMatter || value.Version < 1 || value.RecordVersion < 1 || value.Checksum != policyChecksum(value) {
		return fmt.Errorf("%w: stored policy definition failed validation", ErrInvalid)
	}
	switch value.Status {
	case PolicyDraft, PolicyPendingApproval, PolicyApproved, PolicyActive, PolicySuspended, PolicyRetired:
	default:
		return fmt.Errorf("%w: stored policy status is invalid", ErrInvalid)
	}
	return nil
}

func (repo *PostgresRepository) CreatePolicy(ctx context.Context, value Policy) (Policy, error) {
	if repo == nil || repo.pool == nil {
		return Policy{}, ErrInvalid
	}
	if err := repo.pool.QueryRow(ctx, `SELECT t.id::text,le.id::text FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id AND (le.id::text=$2 OR le.code=$2) WHERE t.id::text=$1 OR t.slug=$1`, value.TenantID, value.LegalEntityID).Scan(&value.TenantID, &value.LegalEntityID); errors.Is(err, pgx.ErrNoRows) {
		return Policy{}, ErrInvalid
	} else if err != nil {
		return Policy{}, err
	}
	value.Checksum = policyChecksum(value)
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return Policy{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	eligibility, _ := json.Marshal(value.Eligibility)
	action, _ := json.Marshal(value.Action)
	blast, _ := json.Marshal(value.BlastRadius)
	outcome, _ := json.Marshal(value.Outcome)
	tag, err := tx.Exec(ctx, `
		INSERT INTO form_response_policy_definitions(
			id,tenant_id,legal_entity_id,code,name,purpose,action_class,automation_policy_id,automation_policy_version,
			form_template_id,form_template_version,eligibility,matter_action,blast_radius,outcome_contract,rollout_mode,status,
			maker_id,checker_id,checksum,approved_simulation_id,supersedes_policy_id,rollback_of_policy_id,effective_from,effective_until,
			submitted_at,approved_at,activated_at,suspended_at,retired_at,version,record_version,created_at,updated_at)
		SELECT $1::uuid,t.id,le.id,$4,$5,$6,$7,ap.id,$9,$10::uuid,$11,$12::jsonb,$13::jsonb,$14::jsonb,$15::jsonb,$16,$17,
		       $18::uuid,NULLIF($19,'')::uuid,$20,NULLIF($21,'')::uuid,NULLIF($22,'')::uuid,NULLIF($23,'')::uuid,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34
		FROM tenants t
		JOIN legal_entities le ON le.tenant_id=t.id AND (le.id::text=$3 OR le.code=$3)
		JOIN automation_policies ap ON ap.tenant_id=t.id AND ap.id::text=$8 AND ap.version=$9 AND ap.action_class=$7
		WHERE (t.id::text=$2 OR t.slug=$2)`, value.ID, value.TenantID, value.LegalEntityID, value.Code, value.Name, value.Purpose, value.ActionClass,
		value.AutomationPolicyID, value.AutomationPolicyVersion, value.Eligibility.FormTemplateID, value.Eligibility.FormTemplateVersion,
		eligibility, action, blast, outcome, value.Rollout, value.Status, value.MakerID, value.CheckerID, value.Checksum,
		value.ApprovedSimulationID, value.SupersedesPolicyID, value.RollbackOfPolicyID, value.EffectiveFrom, value.EffectiveUntil,
		value.SubmittedAt, value.ApprovedAt, value.ActivatedAt, value.SuspendedAt, value.RetiredAt, value.Version, value.RecordVersion, value.CreatedAt, value.UpdatedAt)
	if err != nil {
		return Policy{}, normalizePostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return Policy{}, ErrInvalid
	}
	if err := insertPostgresPolicyEvent(ctx, tx, value, policyEventType(value)); err != nil {
		return Policy{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Policy{}, err
	}
	return repo.GetPolicy(ctx, value.TenantID, value.LegalEntityID, value.ID)
}

func (repo *PostgresRepository) GetPolicy(ctx context.Context, tenantID, legalEntityID, id string) (Policy, error) {
	return scanPostgresPolicy(repo.pool.QueryRow(ctx, `SELECT `+postgresPolicyColumns+` FROM form_response_policy_definitions p JOIN tenants t ON t.id=p.tenant_id JOIN legal_entities le ON le.id=p.legal_entity_id AND le.tenant_id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND p.id::text=$3`, tenantID, legalEntityID, id))
}

func (repo *PostgresRepository) ListPolicies(ctx context.Context, tenantID, legalEntityID string, limit int) ([]Policy, error) {
	rows, err := repo.pool.Query(ctx, `SELECT `+postgresPolicyColumns+` FROM form_response_policy_definitions p JOIN tenants t ON t.id=p.tenant_id JOIN legal_entities le ON le.id=p.legal_entity_id AND le.tenant_id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) ORDER BY p.code,p.version DESC,p.id DESC LIMIT $3`, tenantID, legalEntityID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []Policy{}
	for rows.Next() {
		value, scanErr := scanPostgresPolicy(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (repo *PostgresRepository) NextPolicyVersion(ctx context.Context, tenantID, legalEntityID, code string) (int64, error) {
	var value int64
	err := repo.pool.QueryRow(ctx, `SELECT COALESCE(MAX(p.version),0)+1 FROM form_response_policy_definitions p JOIN tenants t ON t.id=p.tenant_id JOIN legal_entities le ON le.id=p.legal_entity_id AND le.tenant_id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND p.code=$3`, tenantID, legalEntityID, code).Scan(&value)
	return value, err
}

func (repo *PostgresRepository) UpdatePolicy(ctx context.Context, value Policy, expected int64) (Policy, error) {
	tx, err := repo.pool.Begin(ctx)
	if err != nil {
		return Policy{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var current int64
	if err := tx.QueryRow(ctx, `SELECT record_version FROM form_response_policy_definitions WHERE id::text=$1 AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND legal_entity_id=(SELECT id FROM legal_entities WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND (id::text=$3 OR code=$3)) FOR UPDATE`, value.ID, value.TenantID, value.LegalEntityID).Scan(&current); errors.Is(err, pgx.ErrNoRows) {
		return Policy{}, ErrNotFound
	} else if err != nil {
		return Policy{}, err
	}
	if current != expected {
		return Policy{}, ErrConflict
	}
	tag, err := tx.Exec(ctx, `UPDATE form_response_policy_definitions SET status=$4,checker_id=NULLIF($5,'')::uuid,approved_simulation_id=NULLIF($6,'')::uuid,effective_from=$7,effective_until=$8,submitted_at=$9,approved_at=$10,activated_at=$11,suspended_at=$12,retired_at=$13,record_version=$14,updated_at=$15 WHERE id::text=$1 AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND legal_entity_id=(SELECT id FROM legal_entities WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND (id::text=$3 OR code=$3))`, value.ID, value.TenantID, value.LegalEntityID, value.Status, value.CheckerID, value.ApprovedSimulationID, value.EffectiveFrom, value.EffectiveUntil, value.SubmittedAt, value.ApprovedAt, value.ActivatedAt, value.SuspendedAt, value.RetiredAt, value.RecordVersion, value.UpdatedAt)
	if err != nil {
		return Policy{}, normalizePostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return Policy{}, ErrConflict
	}
	if err := insertPostgresPolicyEvent(ctx, tx, value, policyEventType(value)); err != nil {
		return Policy{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Policy{}, err
	}
	return repo.GetPolicy(ctx, value.TenantID, value.LegalEntityID, value.ID)
}

func (repo *PostgresRepository) HasShadowHistory(ctx context.Context, tenantID, legalEntityID, code string, beforeVersion int64) (bool, error) {
	var value bool
	err := repo.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM form_response_policy_definitions p JOIN tenants t ON t.id=p.tenant_id JOIN legal_entities le ON le.id=p.legal_entity_id AND le.tenant_id=p.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND p.code=$3 AND p.version<$4 AND p.rollout_mode='SHADOW' AND p.status IN ('ACTIVE','SUSPENDED','RETIRED'))`, tenantID, legalEntityID, code, beforeVersion).Scan(&value)
	return value, err
}

func (repo *PostgresRepository) SaveSimulation(ctx context.Context, value SimulationReceipt) (SimulationReceipt, error) {
	_, err := repo.pool.Exec(ctx, `INSERT INTO form_response_policy_simulations(id,tenant_id,legal_entity_id,policy_id,policy_version,policy_checksum,actor_id,population_count,eligible_count,would_create_count,would_reuse_count,blast_suppressed_count,restricted_excluded_count,population_high_water,population_checksum,impact_checksum,observed_at,expires_at) SELECT $1::uuid,t.id,le.id,$4::uuid,$5,$6,$7::uuid,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18 FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id AND (le.id::text=$3 OR le.code=$3) WHERE t.id::text=$2 OR t.slug=$2`, value.ID, value.TenantID, value.LegalEntityID, value.PolicyID, value.PolicyVersion, value.PolicyChecksum, value.ActorID, value.PopulationCount, value.EligibleCount, value.WouldCreateCount, value.WouldReuseCount, value.BlastSuppressedCount, value.RestrictedExcludedCount, value.PopulationHighWater, value.PopulationChecksum, value.ImpactChecksum, value.ObservedAt, value.ExpiresAt)
	if err != nil {
		return SimulationReceipt{}, normalizePostgresError(err)
	}
	return repo.GetSimulation(ctx, value.TenantID, value.LegalEntityID, value.ID)
}

func (repo *PostgresRepository) GetSimulation(ctx context.Context, tenantID, legalEntityID, id string) (SimulationReceipt, error) {
	var value SimulationReceipt
	err := repo.pool.QueryRow(ctx, `SELECT s.id::text,s.tenant_id::text,s.legal_entity_id::text,s.policy_id::text,s.policy_version,s.policy_checksum,s.actor_id::text,s.population_count,s.eligible_count,s.would_create_count,s.would_reuse_count,s.blast_suppressed_count,s.restricted_excluded_count,s.population_high_water,s.population_checksum,s.impact_checksum,s.observed_at,s.expires_at FROM form_response_policy_simulations s JOIN tenants t ON t.id=s.tenant_id JOIN legal_entities le ON le.id=s.legal_entity_id AND le.tenant_id=s.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND s.id::text=$3`, tenantID, legalEntityID, id).Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.PolicyID, &value.PolicyVersion, &value.PolicyChecksum, &value.ActorID, &value.PopulationCount, &value.EligibleCount, &value.WouldCreateCount, &value.WouldReuseCount, &value.BlastSuppressedCount, &value.RestrictedExcludedCount, &value.PopulationHighWater, &value.PopulationChecksum, &value.ImpactChecksum, &value.ObservedAt, &value.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SimulationReceipt{}, ErrNotFound
	}
	return value, err
}

func (repo *PostgresRepository) CreateExecution(ctx context.Context, value ExecutionReceipt) (ExecutionReceipt, bool, error) {
	tag, err := repo.pool.Exec(ctx, `INSERT INTO form_response_policy_executions(id,tenant_id,legal_entity_id,policy_id,policy_version,automation_policy_id,automation_policy_version,response_revision_id,state,matter_id,reason_code,created_matter,created_at) SELECT $1::uuid,t.id,le.id,$4::uuid,$5,$6::uuid,$7,$8::uuid,$9,NULLIF($10,'')::uuid,$11,$12,$13 FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id AND (le.id::text=$3 OR le.code=$3) WHERE t.id::text=$2 OR t.slug=$2 ON CONFLICT (tenant_id,legal_entity_id,policy_id,policy_version,response_revision_id) DO NOTHING`, value.ID, value.TenantID, value.LegalEntityID, value.PolicyID, value.PolicyVersion, value.AutomationPolicyID, value.AutomationPolicyVersion, value.ResponseRevisionID, value.State, value.MatterID, value.ReasonCode, value.CreatedMatter, value.CreatedAt)
	if err != nil {
		return ExecutionReceipt{}, false, normalizePostgresError(err)
	}
	stored, getErr := repo.getExecution(ctx, value.TenantID, value.LegalEntityID, value.PolicyID, value.PolicyVersion, value.ResponseRevisionID)
	if getErr != nil {
		return ExecutionReceipt{}, false, getErr
	}
	inserted := tag.RowsAffected() == 1
	if !inserted && executionFingerprint(stored) != executionFingerprint(value) {
		return ExecutionReceipt{}, false, ErrConflict
	}
	return stored, inserted, nil
}

func (repo *PostgresRepository) getExecution(ctx context.Context, tenantID, legalEntityID, policyID string, policyVersion int64, responseID string) (ExecutionReceipt, error) {
	var value ExecutionReceipt
	err := repo.pool.QueryRow(ctx, `SELECT e.id::text,e.tenant_id::text,e.legal_entity_id::text,e.policy_id::text,e.policy_version,e.automation_policy_id::text,e.automation_policy_version,e.response_revision_id::text,e.state,COALESCE(e.matter_id::text,''),e.reason_code,e.created_matter,e.created_at FROM form_response_policy_executions e JOIN tenants t ON t.id=e.tenant_id JOIN legal_entities le ON le.id=e.legal_entity_id AND le.tenant_id=e.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND e.policy_id::text=$3 AND e.policy_version=$4 AND e.response_revision_id::text=$5`, tenantID, legalEntityID, policyID, policyVersion, responseID).Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.PolicyID, &value.PolicyVersion, &value.AutomationPolicyID, &value.AutomationPolicyVersion, &value.ResponseRevisionID, &value.State, &value.MatterID, &value.ReasonCode, &value.CreatedMatter, &value.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ExecutionReceipt{}, ErrNotFound
	}
	return value, err
}

func (repo *PostgresRepository) OpenEpisode(ctx context.Context, value AdverseEpisode) (AdverseEpisode, bool, error) {
	tag, err := repo.pool.Exec(ctx, `INSERT INTO form_response_policy_adverse_episodes(id,tenant_id,legal_entity_id,policy_code,policy_id,policy_version,subject_type,subject_id,state,matter_id,last_response_revision_id,opened_at,closed_at,updated_at,record_version) SELECT $1::uuid,t.id,le.id,$4,$5::uuid,$6,$7,$8::uuid,$9,NULLIF($10,'')::uuid,$11::uuid,$12,$13,$14,$15 FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id AND (le.id::text=$3 OR le.code=$3) WHERE t.id::text=$2 OR t.slug=$2 ON CONFLICT (tenant_id,legal_entity_id,policy_code,subject_type,subject_id) WHERE state='OPEN' DO NOTHING`, value.ID, value.TenantID, value.LegalEntityID, value.PolicyCode, value.PolicyID, value.PolicyVersion, value.SubjectType, value.SubjectID, value.State, value.MatterID, value.LastResponseRevisionID, value.OpenedAt, value.ClosedAt, value.UpdatedAt, value.RecordVersion)
	if err != nil {
		return AdverseEpisode{}, false, normalizePostgresError(err)
	}
	stored, getErr := repo.getOpenEpisode(ctx, value.TenantID, value.LegalEntityID, value.PolicyCode, value.SubjectType, value.SubjectID)
	return stored, tag.RowsAffected() == 1, getErr
}

func (repo *PostgresRepository) getOpenEpisode(ctx context.Context, tenantID, legalEntityID, policyCode, subjectType, subjectID string) (AdverseEpisode, error) {
	var value AdverseEpisode
	err := repo.pool.QueryRow(ctx, `SELECT e.id::text,e.tenant_id::text,e.legal_entity_id::text,e.policy_code,e.policy_id::text,e.policy_version,e.subject_type,e.subject_id::text,e.state,COALESCE(e.matter_id::text,''),e.last_response_revision_id::text,e.opened_at,e.closed_at,e.updated_at,e.record_version FROM form_response_policy_adverse_episodes e JOIN tenants t ON t.id=e.tenant_id JOIN legal_entities le ON le.id=e.legal_entity_id AND le.tenant_id=e.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND (le.id::text=$2 OR le.code=$2) AND e.policy_code=$3 AND e.subject_type=$4 AND e.subject_id::text=$5 AND e.state='OPEN'`, tenantID, legalEntityID, policyCode, subjectType, subjectID).Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.PolicyCode, &value.PolicyID, &value.PolicyVersion, &value.SubjectType, &value.SubjectID, &value.State, &value.MatterID, &value.LastResponseRevisionID, &value.OpenedAt, &value.ClosedAt, &value.UpdatedAt, &value.RecordVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdverseEpisode{}, ErrNotFound
	}
	return value, err
}

func insertPostgresPolicyEvent(ctx context.Context, tx pgx.Tx, value Policy, eventType string) error {
	payload, _ := json.Marshal(map[string]any{"version": value.RecordVersion, "policy_version": value.Version, "record_version": value.RecordVersion, "status": value.Status, "rollout_mode": value.Rollout, "checksum": value.Checksum})
	_, err := tx.Exec(ctx, `WITH scope AS (SELECT t.id tenant_id,le.id legal_entity_id FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id AND (le.id::text=$3 OR le.code=$3) WHERE t.id::text=$2 OR t.slug=$2), event AS (INSERT INTO form_response_policy_events(tenant_id,legal_entity_id,policy_id,policy_version,record_version,event_type,actor_id,payload,occurred_at) SELECT tenant_id,legal_entity_id,$1::uuid,$4,$5,$6,$7::uuid,$8::jsonb,$9 FROM scope RETURNING tenant_id) INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at) SELECT tenant_id,'FORM_RESPONSE_POLICY',$1::uuid,$6,$8::jsonb,$9,$9 FROM event`, value.ID, value.TenantID, value.LegalEntityID, value.Version, value.RecordVersion, eventType, value.LastActorID, payload, value.UpdatedAt)
	return normalizePostgresError(err)
}

func policyEventType(value Policy) string {
	if value.RollbackOfPolicyID != "" && value.RecordVersion == 1 {
		return "FORM_RESPONSE_POLICY_ROLLBACK_DRAFTED"
	}
	switch value.Status {
	case PolicyDraft:
		return "FORM_RESPONSE_POLICY_CREATED"
	case PolicyPendingApproval:
		return "FORM_RESPONSE_POLICY_SUBMITTED"
	case PolicyApproved:
		return "FORM_RESPONSE_POLICY_APPROVED"
	case PolicyActive:
		return "FORM_RESPONSE_POLICY_ACTIVATED"
	case PolicySuspended:
		return "FORM_RESPONSE_POLICY_SUSPENDED"
	case PolicyRetired:
		return "FORM_RESPONSE_POLICY_RETIRED"
	default:
		return "FORM_RESPONSE_POLICY_CHANGED"
	}
}

func normalizePostgresError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return errors.Join(ErrConflict, err)
		case "23503", "23514", "22P02":
			return errors.Join(ErrInvalid, err)
		}
	}
	if strings.Contains(err.Error(), "no rows") {
		return ErrNotFound
	}
	return err
}
