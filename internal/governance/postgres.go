//go:build postgres

package governance

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

const policySelect = `SELECT rp.id::text,t.slug,rp.legal_entity_id::text,rp.code,rp.name,rp.status,rp.current_version,rpv.definition,rpv.checksum,
	COALESCE(rp.maker_id::text,''),COALESCE(rp.checker_id::text,''),COALESCE(rpv.effective_from,'epoch'::timestamptz),COALESCE(rpv.effective_until,'epoch'::timestamptz),
	COALESCE(rp.submitted_at,'epoch'::timestamptz),COALESCE(rp.approved_at,'epoch'::timestamptz),COALESCE(rp.retired_at,'epoch'::timestamptz),rp.created_at,rp.updated_at,rp.version,
	COALESCE(latest.from_state,''),COALESCE(latest.to_state,''),COALESCE(latest.actor_id::text,''),COALESCE(latest.rationale,''),COALESCE(latest.decided_at,'epoch'::timestamptz)
	FROM routing_policies rp JOIN tenants t ON t.id=rp.tenant_id JOIN routing_policy_versions rpv ON rpv.policy_id=rp.id AND rpv.version=rp.current_version
	LEFT JOIN LATERAL (
		SELECT gd.from_state,gd.to_state,gd.actor_id,gd.rationale,gd.decided_at
		FROM governance_decisions gd
		WHERE gd.tenant_id=rp.tenant_id AND gd.object_type='ROUTING_POLICY' AND gd.object_id=rp.id
		ORDER BY gd.decided_at DESC,gd.id DESC LIMIT 1
	) latest ON true`

func scanPolicy(row pgx.Row) (RoutingPolicy, error) {
	var p RoutingPolicy
	var effectiveFrom, effectiveUntil, submitted, approved, retired, decisionAt time.Time
	var decision GovernanceDecisionSummary
	err := row.Scan(&p.ID, &p.TenantID, &p.LegalEntityID, &p.Code, &p.Name, &p.Status, &p.CurrentVersion, &p.Definition, &p.Checksum, &p.MakerID, &p.CheckerID, &effectiveFrom, &effectiveUntil, &submitted, &approved, &retired, &p.CreatedAt, &p.UpdatedAt, &p.Version, &decision.FromState, &decision.ToState, &decision.ActorID, &decision.Rationale, &decisionAt)
	if err != nil {
		return RoutingPolicy{}, err
	}
	p.EffectiveFrom = pointerTime(effectiveFrom)
	p.EffectiveUntil = pointerTime(effectiveUntil)
	p.SubmittedAt = pointerTime(submitted)
	p.ApprovedAt = pointerTime(approved)
	p.RetiredAt = pointerTime(retired)
	if !decisionAt.Equal(time.Unix(0, 0).UTC()) {
		decision.DecidedAt = decisionAt.UTC()
		decision.RecordVersion = p.Version
		p.LatestDecision = &decision
	}
	return p, nil
}
func pointerTime(v time.Time) *time.Time {
	if v.Equal(time.Unix(0, 0).UTC()) {
		return nil
	}
	value := v.UTC()
	return &value
}

func (r *PostgresRepository) ListPolicies(ctx context.Context, tenantID string) ([]RoutingPolicy, error) {
	rows, err := r.pool.Query(ctx, policySelect+` WHERE rp.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) ORDER BY rp.code`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RoutingPolicy{}
	for rows.Next() {
		p, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (r *PostgresRepository) ListPoliciesForEntity(ctx context.Context, tenantID, legalEntityID string, limit int) ([]RoutingPolicy, error) {
	rows, err := r.pool.Query(ctx, policySelect+` WHERE rp.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND rp.legal_entity_id=$2::uuid ORDER BY rp.code,rp.id LIMIT $3`, tenantID, legalEntityID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RoutingPolicy{}
	for rows.Next() {
		p, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (r *PostgresRepository) GetPolicy(ctx context.Context, tenantID, id string) (RoutingPolicy, error) {
	p, err := scanPolicy(r.pool.QueryRow(ctx, policySelect+` WHERE rp.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND rp.id::text=$2`, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return RoutingPolicy{}, ErrNotFound
	}
	return p, err
}
func (r *PostgresRepository) GetPolicyForEntity(ctx context.Context, tenantID, legalEntityID, id string) (RoutingPolicy, error) {
	p, err := scanPolicy(r.pool.QueryRow(ctx, policySelect+` WHERE rp.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND rp.legal_entity_id=$2::uuid AND rp.id::text=$3`, tenantID, legalEntityID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return RoutingPolicy{}, ErrNotFound
	}
	return p, err
}
func (r *PostgresRepository) CreatePolicy(ctx context.Context, p RoutingPolicy) (RoutingPolicy, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return RoutingPolicy{}, err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO routing_policies(id,tenant_id,legal_entity_id,code,name,status,current_version,maker_id,created_at,updated_at,version) VALUES($1,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,$9,$9,$10)`, p.ID, p.TenantID, p.LegalEntityID, p.Code, p.Name, p.Status, p.CurrentVersion, p.MakerID, p.CreatedAt, p.Version)
	if err != nil {
		return RoutingPolicy{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO routing_policy_versions(policy_id,legal_entity_id,version,definition,checksum,effective_from,created_by,created_at) VALUES($1,$2::uuid,$3,$4,$5,$6,$7,$8)`, p.ID, p.LegalEntityID, p.CurrentVersion, p.Definition, p.Checksum, p.EffectiveFrom, p.MakerID, p.CreatedAt)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return RoutingPolicy{}, err
	}
	return r.GetPolicyForEntity(ctx, p.TenantID, p.LegalEntityID, p.ID)
}
func (r *PostgresRepository) TransitionPolicy(ctx context.Context, tenantID, legalEntityID, id string, expected int64, from, to PolicyState, actor, rationale string, at time.Time) (RoutingPolicy, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return RoutingPolicy{}, err
	}
	defer tx.Rollback(ctx)
	var current PolicyState
	var version int64
	var maker string
	var currentVersion int
	err = tx.QueryRow(ctx, `SELECT status,version,COALESCE(maker_id::text,''),current_version FROM routing_policies WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND legal_entity_id=$3::uuid AND id::text=$2 FOR UPDATE`, tenantID, id, legalEntityID).Scan(&current, &version, &maker, &currentVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoutingPolicy{}, ErrNotFound
	}
	if err != nil {
		return RoutingPolicy{}, err
	}
	if version != expected {
		return RoutingPolicy{}, ErrVersionConflict
	}
	if current != from {
		return RoutingPolicy{}, ErrInvalidTransition
	}
	_, err = tx.Exec(ctx, `UPDATE routing_policies SET status=$3,checker_id=CASE WHEN $3='ACTIVE' THEN $4::uuid ELSE checker_id END,submitted_at=CASE WHEN $3='PENDING_APPROVAL' THEN $5 ELSE submitted_at END,approved_at=CASE WHEN $3='ACTIVE' THEN $5 ELSE approved_at END,retired_at=CASE WHEN $3='RETIRED' THEN $5 ELSE retired_at END,updated_at=$5,version=version+1 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND legal_entity_id=$6::uuid AND id::text=$2`, tenantID, id, to, nullableUUID(actor), at, legalEntityID)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if to == PolicyActive {
		_, err = tx.Exec(ctx, `UPDATE routing_policy_versions SET approved_by=$2::uuid,approved_at=$3,effective_from=COALESCE(effective_from,$3) WHERE policy_id=$1::uuid AND legal_entity_id=$5::uuid AND version=$4`, id, actor, at, currentVersion, legalEntityID)
		if err != nil {
			return RoutingPolicy{}, err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO governance_decisions(tenant_id,object_type,object_id,from_state,to_state,actor_id,rationale,decided_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'ROUTING_POLICY',$2::uuid,$3,$4,$5::uuid,$6,$7)`, tenantID, id, from, to, actor, rationale, at)
	if err != nil {
		return RoutingPolicy{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'ROUTING_POLICY',$2::uuid,'RoutingPolicyStateChanged',jsonb_build_object('from',$3::text,'to',$4::text,'actor_id',$5::text,'rationale',$6::text,'legal_entity_id',$8::text),$7,$7)`, tenantID, id, from, to, actor, rationale, at, legalEntityID)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return RoutingPolicy{}, err
	}
	return r.GetPolicyForEntity(ctx, tenantID, legalEntityID, id)
}

func (r *PostgresRepository) ActivatePolicy(ctx context.Context, tenantID, legalEntityID, id string, expected int64, actor, rationale string, at time.Time) (RoutingPolicy, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return RoutingPolicy{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended((SELECT id::text FROM tenants WHERE id::text=$1 OR slug=$1)||':'||$2,0))`, tenantID, legalEntityID); err != nil {
		return RoutingPolicy{}, err
	}
	var policy RoutingPolicy
	err = tx.QueryRow(ctx, `SELECT rp.status,rp.version,COALESCE(rp.maker_id::text,''),rp.current_version,rpv.definition,rpv.checksum
		FROM routing_policies rp JOIN routing_policy_versions rpv ON rpv.policy_id=rp.id AND rpv.version=rp.current_version
		WHERE rp.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND rp.legal_entity_id=$2::uuid AND rp.id::text=$3 FOR UPDATE`, tenantID, legalEntityID, id).Scan(
		&policy.Status, &policy.Version, &policy.MakerID, &policy.CurrentVersion, &policy.Definition, &policy.Checksum,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoutingPolicy{}, ErrNotFound
	}
	if err != nil {
		return RoutingPolicy{}, err
	}
	policy.ID, policy.TenantID, policy.LegalEntityID = id, tenantID, legalEntityID
	if policy.Version != expected {
		return RoutingPolicy{}, ErrVersionConflict
	}
	if policy.Status != PolicyPendingApproval {
		return RoutingPolicy{}, ErrInvalidTransition
	}
	if actor == "" || actor == policy.MakerID {
		return RoutingPolicy{}, ErrMakerChecker
	}
	if _, err := tx.Exec(ctx, `SELECT id FROM routing_policies
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND legal_entity_id=$2::uuid AND status='ACTIVE' FOR UPDATE`, tenantID, legalEntityID); err != nil {
		return RoutingPolicy{}, err
	}
	authorityDefinition, err := authorityOnlyPolicyDefinition(policy.Definition)
	if err != nil {
		return RoutingPolicy{}, err
	}
	policy.Definition = authorityDefinition
	findings, err := policyConflicts(ctx, tx, policy)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if len(findings) > 0 {
		return RoutingPolicy{}, fmt.Errorf("%w: %s", ErrConflict, findings[0].Summary)
	}
	var activeConflicts int
	if err := tx.QueryRow(ctx, `SELECT count(*)
		FROM jsonb_array_elements(COALESCE($4::jsonb->'rules','[]'::jsonb)) candidate
		JOIN routing_policies active ON active.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND active.legal_entity_id=$2::uuid AND active.status='ACTIVE' AND active.id::text<>$3
		JOIN routing_policy_versions av ON av.policy_id=active.id AND av.version=active.current_version
		CROSS JOIN LATERAL jsonb_array_elements(COALESCE(av.definition->'rules','[]'::jsonb)) existing
		WHERE COALESCE(candidate->>'responsibility','')=COALESCE(existing->>'responsibility','')
		  AND COALESCE(candidate->>'object_type','*')=COALESCE(existing->>'object_type','*')
		  AND COALESCE(candidate->>'object_id','*')=COALESCE(existing->>'object_id','*')
		  AND COALESCE(candidate->>'decision_type','')=COALESCE(existing->>'decision_type','')
		  AND COALESCE((candidate->>'min_materiality')::int,0)=COALESCE((existing->>'min_materiality')::int,0)
		  AND COALESCE((candidate->>'priority')::int,0)=COALESCE((existing->>'priority')::int,0)
		  AND candidate->'selector' IS DISTINCT FROM existing->'selector'`, tenantID, legalEntityID, id, authorityDefinition).Scan(&activeConflicts); err != nil {
		return RoutingPolicy{}, err
	}
	if activeConflicts > 0 {
		return RoutingPolicy{}, ErrConflict
	}
	if _, err := tx.Exec(ctx, `UPDATE routing_policies SET status='ACTIVE',checker_id=$4::uuid,approved_at=$5,updated_at=$5,version=version+1
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND legal_entity_id=$2::uuid AND id::text=$3`, tenantID, legalEntityID, id, actor, at); err != nil {
		return RoutingPolicy{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE routing_policy_versions SET approved_by=$3::uuid,approved_at=$4,effective_from=COALESCE(effective_from,$4)
		WHERE policy_id=$1::uuid AND legal_entity_id=$2::uuid AND version=$5`, id, legalEntityID, actor, at, policy.CurrentVersion); err != nil {
		return RoutingPolicy{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO governance_decisions(tenant_id,object_type,object_id,from_state,to_state,actor_id,rationale,decided_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'ROUTING_POLICY',$2::uuid,$3,$4,$5::uuid,$6,$7)`, tenantID, id, PolicyPendingApproval, PolicyActive, actor, rationale, at); err != nil {
		return RoutingPolicy{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'ROUTING_POLICY',$2::uuid,'RoutingPolicyStateChanged',jsonb_build_object('from',$3::text,'to',$4::text,'actor_id',$5::text,'rationale',$6::text,'legal_entity_id',$7::text),$8,$8)`, tenantID, id, PolicyPendingApproval, PolicyActive, actor, rationale, legalEntityID, at); err != nil {
		return RoutingPolicy{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RoutingPolicy{}, err
	}
	return r.GetPolicy(ctx, tenantID, id)
}

func (r *PostgresRepository) PolicyConflicts(ctx context.Context, policy RoutingPolicy) ([]ConflictFinding, error) {
	return policyConflicts(ctx, r.pool, policy)
}

type policyConflictQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func policyConflicts(ctx context.Context, querier policyConflictQuerier, policy RoutingPolicy) ([]ConflictFinding, error) {
	var definition struct {
		Rules []struct {
			ID            string `json:"id"`
			LegalEntityID string `json:"legal_entity_id"`
			Selector      struct {
				Kind string `json:"kind"`
				Ref  string `json:"ref"`
			} `json:"selector"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(policy.Definition, &definition); err != nil {
		return nil, err
	}
	findings := []ConflictFinding{}
	for _, rule := range definition.Rules {
		var count int
		entity := rule.LegalEntityID
		if entity == "" {
			entity = "*"
		}
		switch strings.ToUpper(rule.Selector.Kind) {
		case "ROLE":
			err := querier.QueryRow(ctx, `SELECT count(*) FROM role_templates rt JOIN position_role_bindings prb ON prb.role_template_id=rt.id AND prb.valid_until IS NULL JOIN org_positions op ON op.id=prb.position_id AND op.valid_until IS NULL JOIN principals p ON p.id=op.occupant_principal_id AND p.valid_until IS NULL WHERE rt.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND rt.code=$2 AND op.legal_entity_id=$3::uuid`, policy.TenantID, rule.Selector.Ref, entity).Scan(&count)
			if err != nil {
				return nil, err
			}
		case "POSITION":
			err := querier.QueryRow(ctx, `SELECT count(*) FROM org_positions op JOIN principals p ON p.id=op.occupant_principal_id AND p.valid_until IS NULL WHERE op.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND op.code=$2 AND op.valid_until IS NULL AND op.legal_entity_id=$3::uuid`, policy.TenantID, rule.Selector.Ref, entity).Scan(&count)
			if err != nil {
				return nil, err
			}
		default:
			err := querier.QueryRow(ctx, `SELECT count(*) FROM principals WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND (id::text=$2 OR external_ref=$2) AND valid_until IS NULL`, policy.TenantID, rule.Selector.Ref).Scan(&count)
			if err != nil {
				return nil, err
			}
		}
		if count != 1 {
			findings = append(findings, ConflictFinding{Code: "SELECTOR_CARDINALITY", Summary: fmt.Sprintf("Rule %s selector resolved to %d active principals; exactly one is required before activation.", rule.ID, count)})
		}
	}
	return findings, nil
}

const delegationSelect = `SELECT d.id::text,t.slug,d.legal_entity_id::text,d.from_principal_id::text,d.to_principal_id::text,d.responsibility,d.scope,d.starts_at,d.ends_at,d.status,d.reason,COALESCE(d.created_by::text,''),COALESCE(d.approved_by::text,''),COALESCE(d.submitted_at,'epoch'::timestamptz),COALESCE(d.approved_at,'epoch'::timestamptz),COALESCE(d.revoked_at,'epoch'::timestamptz),d.created_at,d.updated_at,d.version FROM delegations d JOIN tenants t ON t.id=d.tenant_id`

func scanDelegation(row pgx.Row) (Delegation, error) {
	var d Delegation
	var submitted, approved, revoked time.Time
	err := row.Scan(&d.ID, &d.TenantID, &d.LegalEntityID, &d.FromPrincipalID, &d.ToPrincipalID, &d.Responsibility, &d.Scope, &d.StartsAt, &d.EndsAt, &d.Status, &d.Reason, &d.MakerID, &d.ApproverID, &submitted, &approved, &revoked, &d.CreatedAt, &d.UpdatedAt, &d.Version)
	if err != nil {
		return Delegation{}, err
	}
	d.SubmittedAt = pointerTime(submitted)
	d.ApprovedAt = pointerTime(approved)
	d.RevokedAt = pointerTime(revoked)
	return d, nil
}
func (r *PostgresRepository) ListDelegations(ctx context.Context, tenantID string) ([]Delegation, error) {
	rows, err := r.pool.Query(ctx, delegationSelect+` WHERE d.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) ORDER BY d.created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Delegation{}
	for rows.Next() {
		d, err := scanDelegation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (r *PostgresRepository) ListDelegationsForEntity(ctx context.Context, tenantID, legalEntityID string, limit int) ([]Delegation, error) {
	rows, err := r.pool.Query(ctx, delegationSelect+` WHERE d.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND d.legal_entity_id=$2::uuid ORDER BY d.created_at DESC,d.id LIMIT $3`, tenantID, legalEntityID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Delegation{}
	for rows.Next() {
		d, err := scanDelegation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
func (r *PostgresRepository) GetDelegation(ctx context.Context, tenantID, id string) (Delegation, error) {
	d, err := scanDelegation(r.pool.QueryRow(ctx, delegationSelect+` WHERE d.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND d.id::text=$2`, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Delegation{}, ErrNotFound
	}
	return d, err
}
func (r *PostgresRepository) GetDelegationForEntity(ctx context.Context, tenantID, legalEntityID, id string) (Delegation, error) {
	d, err := scanDelegation(r.pool.QueryRow(ctx, delegationSelect+` WHERE d.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND d.legal_entity_id=$2::uuid AND d.id::text=$3`, tenantID, legalEntityID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Delegation{}, ErrNotFound
	}
	return d, err
}
func (r *PostgresRepository) CreateDelegation(ctx context.Context, d Delegation) (Delegation, error) {
	_, err := r.pool.Exec(ctx, `INSERT INTO delegations(id,tenant_id,legal_entity_id,from_principal_id,to_principal_id,responsibility,scope,starts_at,ends_at,status,reason,created_by,created_at,updated_at,version) VALUES($1,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$13,$14)`, d.ID, d.TenantID, d.LegalEntityID, d.FromPrincipalID, d.ToPrincipalID, d.Responsibility, d.Scope, d.StartsAt, d.EndsAt, d.Status, d.Reason, d.MakerID, d.CreatedAt, d.Version)
	if err != nil {
		return Delegation{}, err
	}
	return r.GetDelegationForEntity(ctx, d.TenantID, d.LegalEntityID, d.ID)
}
func (r *PostgresRepository) TransitionDelegation(ctx context.Context, tenantID, legalEntityID, id string, expected int64, from, to DelegationState, actor, rationale string, at time.Time) (Delegation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Delegation{}, err
	}
	defer tx.Rollback(ctx)
	var current DelegationState
	var version int64
	err = tx.QueryRow(ctx, `SELECT status,version FROM delegations WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND legal_entity_id=$3::uuid AND id::text=$2 FOR UPDATE`, tenantID, id, legalEntityID).Scan(&current, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Delegation{}, ErrNotFound
	}
	if err != nil {
		return Delegation{}, err
	}
	if version != expected {
		return Delegation{}, ErrVersionConflict
	}
	if current != from {
		return Delegation{}, ErrInvalidTransition
	}
	_, err = tx.Exec(ctx, `UPDATE delegations SET status=$3,submitted_at=CASE WHEN $3='PENDING_APPROVAL' THEN $5 ELSE submitted_at END,approved_by=CASE WHEN $3 IN ('APPROVED','ACTIVE') THEN $4::uuid ELSE approved_by END,approved_at=CASE WHEN $3 IN ('APPROVED','ACTIVE') THEN $5 ELSE approved_at END,revoked_by=CASE WHEN $3='REVOKED' THEN $4::uuid ELSE revoked_by END,revoked_at=CASE WHEN $3='REVOKED' THEN $5 ELSE revoked_at END,updated_at=$5,version=version+1 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND legal_entity_id=$6::uuid AND id::text=$2`, tenantID, id, to, actor, at, legalEntityID)
	if err != nil {
		return Delegation{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO governance_decisions(tenant_id,object_type,object_id,from_state,to_state,actor_id,rationale,decided_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'DELEGATION',$2::uuid,$3,$4,$5::uuid,$6,$7)`, tenantID, id, from, to, actor, rationale, at)
	if err != nil {
		return Delegation{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'DELEGATION',$2::uuid,'DelegationStateChanged',jsonb_build_object('from',$3::text,'to',$4::text,'actor_id',$5::text,'rationale',$6::text,'legal_entity_id',$8::text),$7,$7)`, tenantID, id, from, to, actor, rationale, at, legalEntityID)
	if err != nil {
		return Delegation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Delegation{}, err
	}
	return r.GetDelegationForEntity(ctx, tenantID, legalEntityID, id)
}
func (r *PostgresRepository) HasDelegationCycle(ctx context.Context, d Delegation) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		WITH RECURSIVE edges AS (
			SELECT from_principal_id,to_principal_id
			FROM delegations
			WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND legal_entity_id=$8::uuid
			  AND responsibility=$4
			  AND status IN ('APPROVED','ACTIVE')
			  AND starts_at<$6 AND $5<ends_at
			  AND id::text<>$7
		), walk(node,path) AS (
			SELECT $3::uuid,ARRAY[$3::uuid]
			UNION ALL
			SELECT e.to_principal_id,w.path||e.to_principal_id
			FROM walk w JOIN edges e ON e.from_principal_id=w.node
			WHERE NOT e.to_principal_id=ANY(w.path)
		)
		SELECT EXISTS(SELECT 1 FROM walk WHERE node=$2::uuid)
	`, d.TenantID, d.FromPrincipalID, d.ToPrincipalID, d.Responsibility, d.StartsAt, d.EndsAt, d.ID, d.LegalEntityID).Scan(&exists)
	return exists, err
}
func (r *PostgresRepository) DelegationConflicts(ctx context.Context, tenantID, principalID, responsibility string) ([]ConflictFinding, error) {
	rows, err := r.pool.Query(ctx, `SELECT sr.code,'Delegate occupies prohibited role '||rt.code||' for responsibility '||sr.responsibility FROM segregation_rules sr JOIN role_templates rt ON rt.tenant_id=sr.tenant_id AND rt.code=sr.prohibited_role_code AND rt.valid_until IS NULL JOIN position_role_bindings prb ON prb.role_template_id=rt.id AND prb.valid_until IS NULL JOIN org_positions op ON op.id=prb.position_id AND op.valid_until IS NULL WHERE sr.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND sr.status='ACTIVE' AND sr.responsibility=$3 AND op.occupant_principal_id=$2::uuid`, tenantID, principalID, responsibility)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ConflictFinding{}
	for rows.Next() {
		var f ConflictFinding
		if err := rows.Scan(&f.Code, &f.Summary); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
func (r *PostgresRepository) ActivateDelegation(ctx context.Context, tenantID, legalEntityID, id string, expected int64, actor, rationale string, at time.Time) (Delegation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Delegation{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended((SELECT id::text FROM tenants WHERE id::text=$1 OR slug=$1)||':'||$2,0))`, tenantID, legalEntityID); err != nil {
		return Delegation{}, err
	}
	var candidate Delegation
	err = tx.QueryRow(ctx, `
		SELECT d.id::text,d.from_principal_id::text,d.to_principal_id::text,d.responsibility,d.starts_at,d.ends_at,
		       d.status,d.version,COALESCE(d.created_by::text,'')
		FROM delegations d
		WHERE d.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND d.legal_entity_id=$2::uuid AND d.id::text=$3
		FOR UPDATE`, tenantID, legalEntityID, id).Scan(
		&candidate.ID, &candidate.FromPrincipalID, &candidate.ToPrincipalID, &candidate.Responsibility,
		&candidate.StartsAt, &candidate.EndsAt, &candidate.Status, &candidate.Version, &candidate.MakerID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Delegation{}, ErrNotFound
	}
	if err != nil {
		return Delegation{}, err
	}
	candidate.TenantID, candidate.LegalEntityID = tenantID, legalEntityID
	if candidate.Version != expected {
		return Delegation{}, ErrVersionConflict
	}
	if candidate.Status != DelegationPendingApproval {
		return Delegation{}, ErrInvalidTransition
	}
	if actor == "" || actor == candidate.MakerID || actor == candidate.FromPrincipalID || actor == candidate.ToPrincipalID {
		return Delegation{}, ErrMakerChecker
	}
	if _, err := tx.Exec(ctx, `SELECT id FROM delegations
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND legal_entity_id=$2::uuid AND responsibility=$3
		  AND status IN ('APPROVED','ACTIVE') AND starts_at<$5 AND $4<ends_at
		FOR UPDATE`, tenantID, legalEntityID, candidate.Responsibility, candidate.StartsAt, candidate.EndsAt); err != nil {
		return Delegation{}, err
	}
	var cycle bool
	if err := tx.QueryRow(ctx, `WITH RECURSIVE edges AS (
			SELECT from_principal_id,to_principal_id FROM delegations
			WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND legal_entity_id=$2::uuid AND responsibility=$5
			  AND status IN ('APPROVED','ACTIVE') AND starts_at<$7 AND $6<ends_at AND id::text<>$8
		), walk(node,path) AS (
			SELECT $4::uuid,ARRAY[$4::uuid]
			UNION ALL
			SELECT e.to_principal_id,w.path||e.to_principal_id FROM walk w JOIN edges e ON e.from_principal_id=w.node
			WHERE NOT e.to_principal_id=ANY(w.path)
		)
		SELECT EXISTS(SELECT 1 FROM walk WHERE node=$3::uuid)`, tenantID, legalEntityID,
		candidate.FromPrincipalID, candidate.ToPrincipalID, candidate.Responsibility,
		candidate.StartsAt, candidate.EndsAt, candidate.ID).Scan(&cycle); err != nil {
		return Delegation{}, err
	}
	if cycle {
		return Delegation{}, ErrConflict
	}
	participantsEligible, err := delegationParticipantsEligible(ctx, tx, tenantID, legalEntityID, candidate, at)
	if err != nil {
		return Delegation{}, err
	}
	if !participantsEligible {
		return Delegation{}, ErrDelegationEligibility
	}
	var conflicts int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM segregation_rules sr
		JOIN role_templates rt ON rt.tenant_id=sr.tenant_id AND rt.code=sr.prohibited_role_code AND rt.valid_until IS NULL
		JOIN position_role_bindings prb ON prb.role_template_id=rt.id AND prb.valid_until IS NULL
		JOIN org_positions op ON op.id=prb.position_id AND op.valid_until IS NULL
		WHERE sr.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND sr.status='ACTIVE' AND sr.responsibility=$4 AND op.legal_entity_id=$2::uuid
		  AND op.occupant_principal_id=$3::uuid`, tenantID, legalEntityID, candidate.ToPrincipalID, candidate.Responsibility).Scan(&conflicts); err != nil {
		return Delegation{}, err
	}
	if conflicts > 0 {
		return Delegation{}, ErrConflict
	}
	target := DelegationApproved
	if !at.Before(candidate.StartsAt) && at.Before(candidate.EndsAt) {
		target = DelegationActive
	}
	if _, err := tx.Exec(ctx, `UPDATE delegations SET status=$4,approved_by=$5::uuid,approved_at=$6,updated_at=$6,version=version+1
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND legal_entity_id=$2::uuid AND id::text=$3`, tenantID, legalEntityID, id, target, actor, at); err != nil {
		return Delegation{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO governance_decisions(tenant_id,object_type,object_id,from_state,to_state,actor_id,rationale,decided_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'DELEGATION',$2::uuid,$3,$4,$5::uuid,$6,$7)`, tenantID, id, DelegationPendingApproval, target, actor, rationale, at); err != nil {
		return Delegation{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'DELEGATION',$2::uuid,'DelegationStateChanged',jsonb_build_object('from',$3::text,'to',$4::text,'actor_id',$5::text,'rationale',$6::text,'legal_entity_id',$7::text),$8,$8)`, tenantID, id, DelegationPendingApproval, target, actor, rationale, legalEntityID, at); err != nil {
		return Delegation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Delegation{}, err
	}
	return r.GetDelegationForEntity(ctx, tenantID, legalEntityID, id)
}

func delegationParticipantsEligible(ctx context.Context, tx pgx.Tx, tenantID, legalEntityID string, candidate Delegation, at time.Time) (bool, error) {
	var giverMember, giverAuthority, recipientEligible bool
	if err := tx.QueryRow(ctx, `
		SELECT
			EXISTS (
				SELECT 1 FROM principals p JOIN org_positions op ON op.tenant_id=p.tenant_id AND op.occupant_principal_id=p.id
				WHERE p.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND p.id=$3::uuid
				  AND p.kind='PERSON' AND p.status='ACTIVE' AND p.valid_from<=$6 AND (p.valid_until IS NULL OR $6<p.valid_until)
				  AND op.legal_entity_id=$2::uuid AND op.valid_from<=$6 AND (op.valid_until IS NULL OR $6<op.valid_until)
			),
			EXISTS (
				SELECT 1 FROM responsibility_assignments ra
				WHERE ra.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND ra.legal_entity_id=$2::uuid
				  AND ra.responsibility=$5 AND ra.valid_from<=$6 AND (ra.valid_until IS NULL OR $6<ra.valid_until)
				  AND ra.valid_from<=$7 AND (ra.valid_until IS NULL OR $8<=ra.valid_until)
				  AND ((ra.object_type='LEGAL_ENTITY' AND (ra.object_id IS NULL OR ra.object_id=$2::uuid)) OR (ra.object_type='*' AND ra.object_id IS NULL))
				  AND (
					ra.principal_id=$3::uuid
					OR ra.position_id IN (
						SELECT op.id FROM org_positions op WHERE op.tenant_id=ra.tenant_id AND op.legal_entity_id=$2::uuid
						  AND op.occupant_principal_id=$3::uuid AND op.valid_from<=$6 AND (op.valid_until IS NULL OR $6<op.valid_until)
					)
					OR ra.role_template_id IN (
						SELECT prb.role_template_id FROM position_role_bindings prb
						JOIN org_positions op ON op.tenant_id=prb.tenant_id AND op.id=prb.position_id
						WHERE prb.tenant_id=ra.tenant_id AND op.legal_entity_id=$2::uuid AND op.occupant_principal_id=$3::uuid
						  AND prb.valid_from<=$6 AND (prb.valid_until IS NULL OR $6<prb.valid_until)
						  AND op.valid_from<=$6 AND (op.valid_until IS NULL OR $6<op.valid_until)
					)
				  )
			) OR EXISTS (
				SELECT 1 FROM delegations source
				WHERE source.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND source.legal_entity_id=$2::uuid
				  AND source.id<>$9::uuid AND source.to_principal_id=$3::uuid AND source.responsibility=$5 AND source.status='ACTIVE'
				  AND source.starts_at<=$6 AND $6<source.ends_at AND source.starts_at<=$7 AND $8<=source.ends_at
				  AND COALESCE(source.scope->>'object_type','')='' AND COALESCE(source.scope->>'object_id','')=''
				  AND COALESCE(source.scope->>'decision_type','')='' AND NOT (source.scope ? 'min_materiality') AND NOT (source.scope ? 'max_materiality')
			),
			EXISTS (
				SELECT 1 FROM principals p JOIN org_positions op ON op.tenant_id=p.tenant_id AND op.occupant_principal_id=p.id
				WHERE p.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND p.id=$4::uuid
				  AND p.kind='PERSON' AND p.status='ACTIVE' AND p.valid_from<=$6 AND (p.valid_until IS NULL OR $6<p.valid_until)
				  AND op.legal_entity_id=$2::uuid AND op.valid_from<=$6 AND (op.valid_until IS NULL OR $6<op.valid_until)
			)`, tenantID, legalEntityID, candidate.FromPrincipalID, candidate.ToPrincipalID, candidate.Responsibility, at, candidate.StartsAt, candidate.EndsAt, candidate.ID).Scan(&giverMember, &giverAuthority, &recipientEligible); err != nil {
		return false, err
	}
	return giverMember && giverAuthority && recipientEligible, nil
}
func (r *PostgresRepository) ActivateDueDelegations(ctx context.Context, now time.Time, limit int) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id::text,tenant_id::text,legal_entity_id::text,from_principal_id::text,to_principal_id::text,responsibility,starts_at,ends_at
		FROM delegations WHERE status='APPROVED' AND starts_at<=$1 AND $1<ends_at
		ORDER BY legal_entity_id,starts_at,id LIMIT $2 FOR UPDATE SKIP LOCKED`, now, limit)
	if err != nil {
		return 0, err
	}
	items := []Delegation{}
	for rows.Next() {
		var item Delegation
		if err := rows.Scan(&item.ID, &item.TenantID, &item.LegalEntityID, &item.FromPrincipalID, &item.ToPrincipalID, &item.Responsibility, &item.StartsAt, &item.EndsAt); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	for _, item := range items {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1||':'||$2,0))`, item.TenantID, item.LegalEntityID); err != nil {
			return 0, err
		}
		var cycle bool
		if err := tx.QueryRow(ctx, `WITH RECURSIVE edges AS (
				SELECT from_principal_id,to_principal_id FROM delegations
				WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND responsibility=$5
				  AND status IN ('APPROVED','ACTIVE') AND starts_at<$7 AND $6<ends_at AND id::text<>$8
			), walk(node,path) AS (
				SELECT $4::uuid,ARRAY[$4::uuid]
				UNION ALL SELECT e.to_principal_id,w.path||e.to_principal_id FROM walk w JOIN edges e ON e.from_principal_id=w.node
				WHERE NOT e.to_principal_id=ANY(w.path)
			) SELECT EXISTS(SELECT 1 FROM walk WHERE node=$3::uuid)`, item.TenantID, item.LegalEntityID,
			item.FromPrincipalID, item.ToPrincipalID, item.Responsibility, item.StartsAt, item.EndsAt, item.ID).Scan(&cycle); err != nil {
			return 0, err
		}
		if cycle {
			return 0, ErrConflict
		}
		participantsEligible, err := delegationParticipantsEligible(ctx, tx, item.TenantID, item.LegalEntityID, item, now)
		if err != nil {
			return 0, err
		}
		if !participantsEligible {
			return 0, ErrDelegationEligibility
		}
		var conflicts int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM segregation_rules sr
			JOIN role_templates rt ON rt.tenant_id=sr.tenant_id AND rt.code=sr.prohibited_role_code AND rt.valid_until IS NULL
			JOIN position_role_bindings prb ON prb.role_template_id=rt.id AND prb.valid_until IS NULL
			JOIN org_positions op ON op.id=prb.position_id AND op.valid_until IS NULL
			WHERE sr.tenant_id=$1::uuid AND sr.status='ACTIVE' AND sr.responsibility=$4
			  AND op.legal_entity_id=$2::uuid AND op.occupant_principal_id=$3::uuid`, item.TenantID, item.LegalEntityID, item.ToPrincipalID, item.Responsibility).Scan(&conflicts); err != nil {
			return 0, err
		}
		if conflicts > 0 {
			return 0, ErrConflict
		}
		if _, err := tx.Exec(ctx, `UPDATE delegations SET status='ACTIVE',updated_at=$2,version=version+1 WHERE id=$1::uuid AND status='APPROVED'`, item.ID, now); err != nil {
			return 0, err
		}
		if err := recordSystemDelegationTransition(ctx, tx, item.TenantID, item.LegalEntityID, item.ID, DelegationApproved, DelegationActive, "Delegation start time reached", now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(items), nil
}
func (r *PostgresRepository) ExpireDueDelegations(ctx context.Context, now time.Time, limit int) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `WITH due AS (
		SELECT id,status AS from_state FROM delegations
		WHERE status IN ('APPROVED','ACTIVE') AND ends_at<=$1
		ORDER BY ends_at,id LIMIT $2 FOR UPDATE SKIP LOCKED
	), updated AS (
		UPDATE delegations d SET status='EXPIRED',updated_at=$1,version=version+1
		FROM due WHERE d.id=due.id RETURNING d.id,d.tenant_id
	) SELECT u.id::text,u.tenant_id::text,source.legal_entity_id::text,d.from_state FROM updated u JOIN due d ON d.id=u.id JOIN delegations source ON source.id=u.id`, now, limit)
	if err != nil {
		return 0, err
	}
	type transition struct {
		id, tenant, entity string
		from               DelegationState
	}
	items := []transition{}
	for rows.Next() {
		var item transition
		if err := rows.Scan(&item.id, &item.tenant, &item.entity, &item.from); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	for _, item := range items {
		if err := recordSystemDelegationTransition(ctx, tx, item.tenant, item.entity, item.id, item.from, DelegationExpired, "Delegation end time reached", now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(items), nil
}
func recordSystemDelegationTransition(ctx context.Context, tx pgx.Tx, tenantID, legalEntityID, id string, from, to DelegationState, rationale string, at time.Time) error {
	_, err := tx.Exec(ctx, `INSERT INTO governance_decisions(tenant_id,object_type,object_id,from_state,to_state,actor_type,actor_id,rationale,decided_at) VALUES($1::uuid,'DELEGATION',$2::uuid,$3::text,$4::text,'SYSTEM',NULL,$5,$6)`, tenantID, id, from, to, rationale, at)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at) VALUES($1::uuid,'DELEGATION',$2::uuid,'DelegationStateChanged',jsonb_build_object('from',$3::text,'to',$4::text,'actor_type','SYSTEM','rationale',$5::text,'legal_entity_id',$7::text),$6,$6)`, tenantID, id, from, to, rationale, at, legalEntityID)
	return err
}
func nullableUUID(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func rawJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`{}`)
	}
	return value
}
func wrapErr(action string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", action, err)
}
