//go:build postgres

package aigovernance

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

type rowScanner interface{ Scan(...any) error }

func scanPolicy(row rowScanner) (Policy, error) {
	var v Policy
	var eligibility, blast, verification, definition []byte
	var rollout string
	err := row.Scan(&v.ID, &v.TenantID, &v.Code, &v.Name, &v.ActionClass, &eligibility, &blast, &verification, &definition, &v.Status, &rollout, &v.MakerID, &v.CheckerID, &v.Checksum, &v.EffectiveFrom, &v.EffectiveUntil, &v.SubmittedAt, &v.ApprovedAt, &v.ActivatedAt, &v.SuspendedAt, &v.RetiredAt, &v.Version, &v.RecordVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return Policy{}, ErrNotFound
	}
	if err != nil {
		return Policy{}, err
	}
	v.Eligibility = append(json.RawMessage(nil), eligibility...)
	v.BlastRadiusLimit = append(json.RawMessage(nil), blast...)
	v.VerificationContract = append(json.RawMessage(nil), verification...)
	if err := json.Unmarshal(definition, &v.Definition); err != nil {
		return Policy{}, fmt.Errorf("decode ai policy definition: %w", err)
	}
	v.RolloutMode = aigateway.RolloutMode(rollout)
	return v, nil
}

const policyColumns = `ap.id::text,t.slug,ap.code,ap.name,ap.action_class,ap.eligibility,ap.blast_radius_limit,ap.verification_contract,ap.ai_definition,ap.status,ap.rollout_mode,COALESCE(ap.maker_id::text,''),COALESCE(ap.checker_id::text,''),ap.checksum,ap.effective_from,ap.effective_until,ap.submitted_at,ap.approved_at,ap.activated_at,ap.suspended_at,ap.retired_at,ap.version,ap.record_version`

func (r *PostgresRepository) CreatePolicy(ctx context.Context, v Policy) (Policy, error) {
	def, _ := json.Marshal(v.Definition)
	row := r.pool.QueryRow(ctx, `INSERT INTO automation_policies(id,tenant_id,code,name,action_class,eligibility,blast_radius_limit,verification_contract,status,effective_from,effective_until,version,ai_definition,rollout_mode,maker_id,checksum,record_version)
VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6::jsonb,$7::jsonb,$8::jsonb,$9,$10,$11,$12,$13::jsonb,$14,NULLIF($15,'')::uuid,$16,$17)
RETURNING id::text,(SELECT slug FROM tenants WHERE id=automation_policies.tenant_id),code,name,action_class,eligibility,blast_radius_limit,verification_contract,ai_definition,status,rollout_mode,COALESCE(maker_id::text,''),COALESCE(checker_id::text,''),checksum,effective_from,effective_until,submitted_at,approved_at,activated_at,suspended_at,retired_at,version,record_version`, v.ID, v.TenantID, v.Code, v.Name, v.ActionClass, string(v.Eligibility), string(v.BlastRadiusLimit), string(v.VerificationContract), v.Status, v.EffectiveFrom, v.EffectiveUntil, v.Version, string(def), v.RolloutMode, v.MakerID, v.Checksum, v.RecordVersion)
	return scanPolicy(row)
}

func (r *PostgresRepository) NextPolicyVersion(ctx context.Context, tenantID, code string) (int64, error) {
	var version int64
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(ap.version),0)+1 FROM automation_policies ap JOIN tenants t ON t.id=ap.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND ap.code=$2`, tenantID, code).Scan(&version)
	return version, err
}

func (r *PostgresRepository) HasShadowHistory(ctx context.Context, tenantID, code string, beforeVersion int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM automation_policies ap JOIN tenants t ON t.id=ap.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND ap.code=$2 AND ap.version<$3 AND ap.rollout_mode='SHADOW' AND ap.status IN ('ACTIVE','SUSPENDED','RETIRED'))`, tenantID, code, beforeVersion).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) Policy(ctx context.Context, t, id string) (Policy, error) {
	return scanPolicy(r.pool.QueryRow(ctx, `SELECT `+policyColumns+` FROM automation_policies ap JOIN tenants t ON t.id=ap.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND ap.id::text=$2`, t, id))
}
func (r *PostgresRepository) ListPolicies(ctx context.Context, t string, limit int) ([]Policy, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+policyColumns+` FROM automation_policies ap JOIN tenants t ON t.id=ap.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) ORDER BY ap.code,ap.version DESC LIMIT $2`, t, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Policy{}
	for rows.Next() {
		v, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *PostgresRepository) UpdatePolicy(ctx context.Context, v Policy, expected int64) (Policy, error) {
	def, _ := json.Marshal(v.Definition)
	tag, err := r.pool.Exec(ctx, `UPDATE automation_policies ap SET status=$3,rollout_mode=$4,checker_id=NULLIF($5,'')::uuid,checksum=$6,effective_from=$7,effective_until=$8,submitted_at=$9,approved_at=$10,activated_at=$11,suspended_at=$12,retired_at=$13,ai_definition=$14::jsonb,record_version=$15 WHERE ap.id::text=$1 AND ap.tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND ap.record_version=$16`, v.ID, v.TenantID, v.Status, v.RolloutMode, v.CheckerID, v.Checksum, v.EffectiveFrom, v.EffectiveUntil, v.SubmittedAt, v.ApprovedAt, v.ActivatedAt, v.SuspendedAt, v.RetiredAt, string(def), v.RecordVersion, expected)
	if err != nil {
		return Policy{}, err
	}
	if tag.RowsAffected() != 1 {
		return Policy{}, ErrConflict
	}
	return r.Policy(ctx, v.TenantID, v.ID)
}

type workloadScan struct {
	Workload
	allowed, metadata, resources []byte
	key                          []byte
}

func scanWorkload(row rowScanner) (Workload, error) {
	var v Workload
	var allowed, metadata, resources, key []byte
	err := row.Scan(&v.ID, &v.WorkloadID, &v.TenantID, &v.Code, &v.Name, &v.Purpose, &v.Environment, &v.OwnerPrincipalID, &v.ServicePrincipalID, &allowed, &v.RequestsPerMinute, &v.TokensPerMinute, &v.CostMicroUSDPerMinute, &v.MaxConcurrent, &metadata, &resources, &v.PolicyID, &v.PolicyVersion, &v.State, &v.MakerID, &v.CheckerID, &v.EffectiveFrom, &v.EffectiveUntil, &v.SubmittedAt, &v.ApprovedAt, &v.ActivatedAt, &v.SuspendedAt, &v.RetiredAt, &v.Checksum, &v.CreatedAt, &v.UpdatedAt, &v.Version, &v.RecordVersion, &key)
	if errors.Is(err, pgx.ErrNoRows) {
		return Workload{}, ErrNotFound
	}
	if err != nil {
		return Workload{}, err
	}
	_ = json.Unmarshal(allowed, &v.AllowedModels)
	_ = json.Unmarshal(metadata, &v.VerifiedMetadata)
	_ = json.Unmarshal(resources, &v.ApprovedResources)
	v.KeySHA256 = hex.EncodeToString(key)
	return v, nil
}

const workloadColumns = `aw.id::text,aw.workload_id,t.slug,aw.code,aw.name,aw.purpose,aw.environment,aw.owner_principal_id::text,COALESCE(aw.service_principal_id::text,''),aw.allowed_models,aw.requests_per_minute,aw.tokens_per_minute,aw.cost_microusd_per_minute,aw.max_concurrent,aw.verified_metadata,aw.approved_resources,aw.policy_id::text,aw.policy_version,aw.state,COALESCE(aw.maker_id::text,''),COALESCE(aw.checker_id::text,''),aw.effective_from,aw.effective_until,aw.submitted_at,aw.approved_at,aw.activated_at,aw.suspended_at,aw.retired_at,aw.checksum,aw.created_at,aw.updated_at,aw.version,aw.record_version,aw.key_sha256`

func (r *PostgresRepository) CreateWorkload(ctx context.Context, v Workload) (Workload, error) {
	allowed, _ := json.Marshal(v.AllowedModels)
	meta, _ := json.Marshal(v.VerifiedMetadata)
	res, _ := json.Marshal(v.ApprovedResources)
	key, _ := hex.DecodeString(v.KeySHA256)
	row := r.pool.QueryRow(ctx, `INSERT INTO ai_workloads(id,workload_id,tenant_id,code,name,purpose,environment,owner_principal_id,service_principal_id,allowed_models,requests_per_minute,tokens_per_minute,cost_microusd_per_minute,max_concurrent,verified_metadata,approved_resources,policy_id,policy_version,state,maker_id,checksum,version,record_version,key_sha256)
VALUES($1::uuid,$2,(SELECT id FROM tenants WHERE id::text=$3 OR slug=$3),$4,$5,$6,$7,$8::uuid,NULLIF($9,'')::uuid,$10::jsonb,$11,$12,$13,$14,$15::jsonb,$16::jsonb,$17::uuid,$18,$19,NULLIF($20,'')::uuid,$21,$22,$23,$24)
RETURNING id::text,workload_id,(SELECT slug FROM tenants WHERE id=ai_workloads.tenant_id),code,name,purpose,environment,owner_principal_id::text,COALESCE(service_principal_id::text,''),allowed_models,requests_per_minute,tokens_per_minute,cost_microusd_per_minute,max_concurrent,verified_metadata,approved_resources,policy_id::text,policy_version,state,COALESCE(maker_id::text,''),COALESCE(checker_id::text,''),effective_from,effective_until,submitted_at,approved_at,activated_at,suspended_at,retired_at,checksum,created_at,updated_at,version,record_version,key_sha256`, v.ID, v.WorkloadID, v.TenantID, v.Code, v.Name, v.Purpose, v.Environment, v.OwnerPrincipalID, v.ServicePrincipalID, string(allowed), v.RequestsPerMinute, v.TokensPerMinute, v.CostMicroUSDPerMinute, v.MaxConcurrent, string(meta), string(res), v.PolicyID, v.PolicyVersion, v.State, v.MakerID, v.Checksum, v.Version, v.RecordVersion, key)
	return scanWorkload(row)
}

func (r *PostgresRepository) NextWorkloadVersion(ctx context.Context, tenantID, workloadID string) (int64, error) {
	var version int64
	err := r.pool.QueryRow(ctx, `SELECT COALESCE(MAX(aw.version),0)+1 FROM ai_workloads aw JOIN tenants t ON t.id=aw.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND aw.workload_id=$2`, tenantID, workloadID).Scan(&version)
	return version, err
}

func (r *PostgresRepository) Workload(ctx context.Context, t, id string) (Workload, error) {
	return scanWorkload(r.pool.QueryRow(ctx, `SELECT `+workloadColumns+` FROM ai_workloads aw JOIN tenants t ON t.id=aw.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND aw.id::text=$2`, t, id))
}
func (r *PostgresRepository) WorkloadByCredential(ctx context.Context, d [32]byte) (Workload, Policy, error) {
	w, err := scanWorkload(r.pool.QueryRow(ctx, `SELECT `+workloadColumns+` FROM ai_workloads aw JOIN tenants t ON t.id=aw.tenant_id WHERE aw.key_sha256=$1 AND aw.state='ACTIVE' AND (aw.effective_from IS NULL OR aw.effective_from<=clock_timestamp()) AND (aw.effective_until IS NULL OR aw.effective_until>clock_timestamp()) LIMIT 1`, d[:]))
	if err != nil {
		return Workload{}, Policy{}, err
	}
	p, err := r.Policy(ctx, w.TenantID, w.PolicyID)
	if err != nil || p.Version != w.PolicyVersion || p.Status != "ACTIVE" {
		return Workload{}, Policy{}, ErrNotFound
	}
	return w, p, nil
}
func (r *PostgresRepository) ListWorkloads(ctx context.Context, t string, limit int) ([]Workload, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+workloadColumns+` FROM ai_workloads aw JOIN tenants t ON t.id=aw.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) ORDER BY aw.code,aw.version DESC LIMIT $2`, t, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Workload{}
	for rows.Next() {
		v, err := scanWorkload(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r *PostgresRepository) UpdateWorkload(ctx context.Context, v Workload, expected int64) (Workload, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE ai_workloads SET state=$3,checker_id=NULLIF($4,'')::uuid,effective_from=$5,effective_until=$6,submitted_at=$7,approved_at=$8,activated_at=$9,suspended_at=$10,retired_at=$11,checksum=$12,record_version=$13,updated_at=clock_timestamp() WHERE id::text=$1 AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND record_version=$14`, v.ID, v.TenantID, v.State, v.CheckerID, v.EffectiveFrom, v.EffectiveUntil, v.SubmittedAt, v.ApprovedAt, v.ActivatedAt, v.SuspendedAt, v.RetiredAt, v.Checksum, v.RecordVersion, expected)
	if err != nil {
		return Workload{}, err
	}
	if tag.RowsAffected() != 1 {
		return Workload{}, ErrConflict
	}
	return r.Workload(ctx, v.TenantID, v.ID)
}
func (r *PostgresRepository) IngestReceipt(ctx context.Context, v DecisionReceipt) (bool, error) {
	reasons, _ := json.Marshal(v.ReasonCodes)
	obligations, _ := json.Marshal(v.Obligations)
	baselineReasons, _ := json.Marshal(v.BaselineReasonCodes)
	var baselineVersion any
	if v.BaselinePolicyID != "" {
		baselineVersion = v.BaselinePolicyVersion
	}
	tag, err := r.pool.Exec(ctx, `INSERT INTO ai_gateway_decision_receipts(
    id,tenant_id,receipt_id,request_id,workload_id,policy_id,policy_code,policy_version,
    decision_action,proposed_action,reason_codes,obligations,
    baseline_policy_id,baseline_policy_code,baseline_policy_version,baseline_rollout_mode,
    baseline_decision_action,baseline_proposed_action,baseline_reason_codes,
    model_alias,route_id,outcome,error_code,observed_at,expires_at,fingerprint)
VALUES(
    uuidv7(),(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3,$4,NULLIF($5,'')::uuid,$6,$7,
    $8,$9,$10::jsonb,$11::jsonb,
    NULLIF($12,'')::uuid,$13,$14,$15,$16,$17,$18::jsonb,
    $19,$20,$21,$22,$23,$24,$25)
ON CONFLICT(tenant_id,receipt_id) DO NOTHING`,
		v.TenantID, v.ReceiptID, v.RequestID, v.WorkloadID, v.PolicyID, v.PolicyCode, v.PolicyVersion,
		v.Decision, v.ProposedAction, string(reasons), string(obligations),
		v.BaselinePolicyID, v.BaselinePolicyCode, baselineVersion, v.BaselineRolloutMode,
		v.BaselineDecision, v.BaselineProposedAction, string(baselineReasons),
		v.ModelAlias, v.RouteID, v.Outcome, v.ErrorCode, v.ObservedAt, v.ExpiresAt, v.Fingerprint)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 1 {
		return true, nil
	}
	var fingerprint string
	err = r.pool.QueryRow(ctx, `SELECT fingerprint FROM ai_gateway_decision_receipts ar JOIN tenants t ON t.id=ar.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) AND ar.receipt_id=$2`, v.TenantID, v.ReceiptID).Scan(&fingerprint)
	if err != nil {
		return false, err
	}
	if fingerprint != v.Fingerprint {
		return false, ErrConflict
	}
	return false, nil
}
func (r *PostgresRepository) CreateGrant(ctx context.Context, v ExecutionGrant, d [32]byte) (ExecutionGrant, error) {
	_, err := r.pool.Exec(ctx, `INSERT INTO ai_execution_grants(id,tenant_id,workload_id,matter_id,decision_id,action_hash,approved_by,state,expires_at,token_sha256,record_version) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4::uuid,$5::uuid,$6,$7::uuid,$8,$9,$10,$11)`, v.ID, v.TenantID, v.WorkloadID, v.MatterID, v.DecisionID, v.ActionHash, v.ApprovedBy, v.State, v.ExpiresAt, d[:], v.RecordVersion)
	if err != nil {
		return ExecutionGrant{}, err
	}
	v.Token = ""
	return v, nil
}
func (r *PostgresRepository) ConsumeGrant(ctx context.Context, t, w, action string, d [32]byte, now time.Time) (ExecutionGrant, error) {
	var g ExecutionGrant
	err := r.pool.QueryRow(ctx, `UPDATE ai_execution_grants ag SET state='USED',used_at=$5,record_version=record_version+1 WHERE ag.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND ag.workload_id=$2 AND ag.action_hash=$3 AND ag.token_sha256=$4 AND ag.state='ACTIVE' AND ag.expires_at>$5 RETURNING ag.id::text,(SELECT slug FROM tenants WHERE id=ag.tenant_id),ag.workload_id,ag.matter_id::text,ag.decision_id::text,ag.action_hash,ag.approved_by::text,ag.state,ag.expires_at,ag.used_at,ag.created_at,ag.record_version`, t, w, action, d[:], now).Scan(&g.ID, &g.TenantID, &g.WorkloadID, &g.MatterID, &g.DecisionID, &g.ActionHash, &g.ApprovedBy, &g.State, &g.ExpiresAt, &g.UsedAt, &g.CreatedAt, &g.RecordVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return ExecutionGrant{}, ErrGrantInvalid
	}
	return g, err
}

func (r *PostgresRepository) MaintainRetention(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 100
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var deleted int
	if err := tx.QueryRow(ctx, `WITH doomed AS (SELECT id FROM ai_gateway_decision_receipts WHERE expires_at<=$1 ORDER BY expires_at,id LIMIT $2 FOR UPDATE SKIP LOCKED), removed AS (DELETE FROM ai_gateway_decision_receipts r USING doomed d WHERE r.id=d.id RETURNING r.id) SELECT count(*) FROM removed`, now, limit).Scan(&deleted); err != nil {
		return 0, err
	}
	remaining := limit - deleted
	expired := int64(0)
	if remaining > 0 {
		tag, updateErr := tx.Exec(ctx, `WITH due AS (SELECT id FROM ai_execution_grants WHERE state='ACTIVE' AND expires_at<=$1 ORDER BY expires_at,id LIMIT $2 FOR UPDATE SKIP LOCKED) UPDATE ai_execution_grants g SET state='EXPIRED',record_version=record_version+1 FROM due d WHERE g.id=d.id`, now, remaining)
		if updateErr != nil {
			return 0, updateErr
		}
		expired = tag.RowsAffected()
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return deleted + int(expired), nil
}
