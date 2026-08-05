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

const policySelect = `SELECT rp.id::text,t.slug,rp.code,rp.name,rp.status,rp.current_version,rpv.definition,rpv.checksum,
COALESCE(rp.maker_id::text,''),COALESCE(rp.checker_id::text,''),COALESCE(rpv.effective_from,'epoch'::timestamptz),COALESCE(rpv.effective_until,'epoch'::timestamptz),
COALESCE(rp.submitted_at,'epoch'::timestamptz),COALESCE(rp.approved_at,'epoch'::timestamptz),COALESCE(rp.retired_at,'epoch'::timestamptz),rp.created_at,rp.updated_at,rp.version
FROM routing_policies rp JOIN tenants t ON t.id=rp.tenant_id JOIN routing_policy_versions rpv ON rpv.policy_id=rp.id AND rpv.version=rp.current_version`

func scanPolicy(row pgx.Row) (RoutingPolicy, error) {
	var p RoutingPolicy
	var effectiveFrom, effectiveUntil, submitted, approved, retired time.Time
	err := row.Scan(&p.ID, &p.TenantID, &p.Code, &p.Name, &p.Status, &p.CurrentVersion, &p.Definition, &p.Checksum, &p.MakerID, &p.CheckerID, &effectiveFrom, &effectiveUntil, &submitted, &approved, &retired, &p.CreatedAt, &p.UpdatedAt, &p.Version)
	if err != nil {
		return RoutingPolicy{}, err
	}
	p.EffectiveFrom = pointerTime(effectiveFrom)
	p.EffectiveUntil = pointerTime(effectiveUntil)
	p.SubmittedAt = pointerTime(submitted)
	p.ApprovedAt = pointerTime(approved)
	p.RetiredAt = pointerTime(retired)
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
func (r *PostgresRepository) GetPolicy(ctx context.Context, tenantID, id string) (RoutingPolicy, error) {
	p, err := scanPolicy(r.pool.QueryRow(ctx, policySelect+` WHERE rp.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND rp.id::text=$2`, tenantID, id))
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
	_, err = tx.Exec(ctx, `INSERT INTO routing_policies(id,tenant_id,code,name,status,current_version,maker_id,created_at,updated_at,version) VALUES($1,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$8,$9)`, p.ID, p.TenantID, p.Code, p.Name, p.Status, p.CurrentVersion, p.MakerID, p.CreatedAt, p.Version)
	if err != nil {
		return RoutingPolicy{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO routing_policy_versions(policy_id,version,definition,checksum,effective_from,created_by,created_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, p.ID, p.CurrentVersion, p.Definition, p.Checksum, p.EffectiveFrom, p.MakerID, p.CreatedAt)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return RoutingPolicy{}, err
	}
	return r.GetPolicy(ctx, p.TenantID, p.ID)
}
func (r *PostgresRepository) TransitionPolicy(ctx context.Context, tenantID, id string, expected int64, from, to PolicyState, actor, rationale string, at time.Time) (RoutingPolicy, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return RoutingPolicy{}, err
	}
	defer tx.Rollback(ctx)
	var current PolicyState
	var version int64
	var maker string
	var currentVersion int
	err = tx.QueryRow(ctx, `SELECT status,version,COALESCE(maker_id::text,''),current_version FROM routing_policies WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id::text=$2 FOR UPDATE`, tenantID, id).Scan(&current, &version, &maker, &currentVersion)
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
	_, err = tx.Exec(ctx, `UPDATE routing_policies SET status=$3,checker_id=CASE WHEN $3='ACTIVE' THEN $4::uuid ELSE checker_id END,submitted_at=CASE WHEN $3='PENDING_APPROVAL' THEN $5 ELSE submitted_at END,approved_at=CASE WHEN $3='ACTIVE' THEN $5 ELSE approved_at END,retired_at=CASE WHEN $3='RETIRED' THEN $5 ELSE retired_at END,updated_at=$5,version=version+1 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id::text=$2`, tenantID, id, to, nullableUUID(actor), at)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if to == PolicyActive {
		_, err = tx.Exec(ctx, `UPDATE routing_policy_versions SET approved_by=$2::uuid,approved_at=$3,effective_from=COALESCE(effective_from,$3) WHERE policy_id=$1::uuid AND version=$4`, id, actor, at, currentVersion)
		if err != nil {
			return RoutingPolicy{}, err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO governance_decisions(tenant_id,object_type,object_id,from_state,to_state,actor_id,rationale,decided_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'ROUTING_POLICY',$2::uuid,$3,$4,$5::uuid,$6,$7)`, tenantID, id, from, to, actor, rationale, at)
	if err != nil {
		return RoutingPolicy{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'ROUTING_POLICY',$2::uuid,'RoutingPolicyStateChanged',jsonb_build_object('from',$3::text,'to',$4::text,'actor_id',$5::text,'rationale',$6::text),$7,$7)`, tenantID, id, from, to, actor, rationale, at)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return RoutingPolicy{}, err
	}
	return r.GetPolicy(ctx, tenantID, id)
}

func (r *PostgresRepository) PolicyConflicts(ctx context.Context, policy RoutingPolicy) ([]ConflictFinding, error) {
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
			err := r.pool.QueryRow(ctx, `SELECT count(*) FROM role_templates rt JOIN position_role_bindings prb ON prb.role_template_id=rt.id AND prb.valid_until IS NULL JOIN org_positions op ON op.id=prb.position_id AND op.valid_until IS NULL JOIN principals p ON p.id=op.occupant_principal_id AND p.valid_until IS NULL WHERE rt.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND rt.code=$2 AND ($3='*' OR op.legal_entity_id IS NULL OR op.legal_entity_id=(SELECT id FROM legal_entities WHERE tenant_id=rt.tenant_id AND (id::text=$3 OR code=$3) LIMIT 1))`, policy.TenantID, rule.Selector.Ref, entity).Scan(&count)
			if err != nil {
				return nil, err
			}
		case "POSITION":
			err := r.pool.QueryRow(ctx, `SELECT count(*) FROM org_positions op JOIN principals p ON p.id=op.occupant_principal_id AND p.valid_until IS NULL WHERE op.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND op.code=$2 AND op.valid_until IS NULL AND ($3='*' OR op.legal_entity_id IS NULL OR op.legal_entity_id=(SELECT id FROM legal_entities WHERE tenant_id=op.tenant_id AND (id::text=$3 OR code=$3) LIMIT 1))`, policy.TenantID, rule.Selector.Ref, entity).Scan(&count)
			if err != nil {
				return nil, err
			}
		default:
			err := r.pool.QueryRow(ctx, `SELECT count(*) FROM principals WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND (id::text=$2 OR external_ref=$2) AND valid_until IS NULL`, policy.TenantID, rule.Selector.Ref).Scan(&count)
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

const delegationSelect = `SELECT d.id::text,t.slug,d.from_principal_id::text,d.to_principal_id::text,d.responsibility,d.scope,d.starts_at,d.ends_at,d.status,d.reason,COALESCE(d.created_by::text,''),COALESCE(d.approved_by::text,''),COALESCE(d.submitted_at,'epoch'::timestamptz),COALESCE(d.approved_at,'epoch'::timestamptz),COALESCE(d.revoked_at,'epoch'::timestamptz),d.created_at,d.updated_at,d.version FROM delegations d JOIN tenants t ON t.id=d.tenant_id`

func scanDelegation(row pgx.Row) (Delegation, error) {
	var d Delegation
	var submitted, approved, revoked time.Time
	err := row.Scan(&d.ID, &d.TenantID, &d.FromPrincipalID, &d.ToPrincipalID, &d.Responsibility, &d.Scope, &d.StartsAt, &d.EndsAt, &d.Status, &d.Reason, &d.MakerID, &d.ApproverID, &submitted, &approved, &revoked, &d.CreatedAt, &d.UpdatedAt, &d.Version)
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
func (r *PostgresRepository) GetDelegation(ctx context.Context, tenantID, id string) (Delegation, error) {
	d, err := scanDelegation(r.pool.QueryRow(ctx, delegationSelect+` WHERE d.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND d.id::text=$2`, tenantID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Delegation{}, ErrNotFound
	}
	return d, err
}
func (r *PostgresRepository) CreateDelegation(ctx context.Context, d Delegation) (Delegation, error) {
	_, err := r.pool.Exec(ctx, `INSERT INTO delegations(id,tenant_id,from_principal_id,to_principal_id,responsibility,scope,starts_at,ends_at,status,reason,created_by,created_at,updated_at,version) VALUES($1,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$12,$13)`, d.ID, d.TenantID, d.FromPrincipalID, d.ToPrincipalID, d.Responsibility, d.Scope, d.StartsAt, d.EndsAt, d.Status, d.Reason, d.MakerID, d.CreatedAt, d.Version)
	if err != nil {
		return Delegation{}, err
	}
	return r.GetDelegation(ctx, d.TenantID, d.ID)
}
func (r *PostgresRepository) TransitionDelegation(ctx context.Context, tenantID, id string, expected int64, from, to DelegationState, actor, rationale string, at time.Time) (Delegation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Delegation{}, err
	}
	defer tx.Rollback(ctx)
	var current DelegationState
	var version int64
	err = tx.QueryRow(ctx, `SELECT status,version FROM delegations WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id::text=$2 FOR UPDATE`, tenantID, id).Scan(&current, &version)
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
	_, err = tx.Exec(ctx, `UPDATE delegations SET status=$3,submitted_at=CASE WHEN $3='PENDING_APPROVAL' THEN $5 ELSE submitted_at END,approved_by=CASE WHEN $3 IN ('APPROVED','ACTIVE') THEN $4::uuid ELSE approved_by END,approved_at=CASE WHEN $3 IN ('APPROVED','ACTIVE') THEN $5 ELSE approved_at END,revoked_by=CASE WHEN $3='REVOKED' THEN $4::uuid ELSE revoked_by END,revoked_at=CASE WHEN $3='REVOKED' THEN $5 ELSE revoked_at END,updated_at=$5,version=version+1 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id::text=$2`, tenantID, id, to, actor, at)
	if err != nil {
		return Delegation{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO governance_decisions(tenant_id,object_type,object_id,from_state,to_state,actor_id,rationale,decided_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'DELEGATION',$2::uuid,$3,$4,$5::uuid,$6,$7)`, tenantID, id, from, to, actor, rationale, at)
	if err != nil {
		return Delegation{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'DELEGATION',$2::uuid,'DelegationStateChanged',jsonb_build_object('from',$3::text,'to',$4::text,'actor_id',$5::text,'rationale',$6::text),$7,$7)`, tenantID, id, from, to, actor, rationale, at)
	if err != nil {
		return Delegation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Delegation{}, err
	}
	return r.GetDelegation(ctx, tenantID, id)
}
func (r *PostgresRepository) HasDelegationCycle(ctx context.Context, d Delegation) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		WITH RECURSIVE edges AS (
			SELECT from_principal_id,to_principal_id
			FROM delegations
			WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
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
	`, d.TenantID, d.FromPrincipalID, d.ToPrincipalID, d.Responsibility, d.StartsAt, d.EndsAt, d.ID).Scan(&exists)
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
func (r *PostgresRepository) ActivateDueDelegations(ctx context.Context, now time.Time, limit int) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `WITH due AS (
		SELECT id FROM delegations
		WHERE status='APPROVED' AND starts_at<=$1 AND $1<ends_at
		ORDER BY starts_at,id LIMIT $2 FOR UPDATE SKIP LOCKED
	), updated AS (
		UPDATE delegations d SET status='ACTIVE',updated_at=$1,version=version+1
		FROM due WHERE d.id=due.id RETURNING d.id,d.tenant_id
	) SELECT id::text,tenant_id::text FROM updated`, now, limit)
	if err != nil {
		return 0, err
	}
	type transition struct{ id, tenant string }
	items := []transition{}
	for rows.Next() {
		var item transition
		if err := rows.Scan(&item.id, &item.tenant); err != nil {
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
		if err := recordSystemDelegationTransition(ctx, tx, item.tenant, item.id, DelegationApproved, DelegationActive, "Delegation start time reached", now); err != nil {
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
	) SELECT u.id::text,u.tenant_id::text,d.from_state FROM updated u JOIN due d ON d.id=u.id`, now, limit)
	if err != nil {
		return 0, err
	}
	type transition struct {
		id, tenant string
		from       DelegationState
	}
	items := []transition{}
	for rows.Next() {
		var item transition
		if err := rows.Scan(&item.id, &item.tenant, &item.from); err != nil {
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
		if err := recordSystemDelegationTransition(ctx, tx, item.tenant, item.id, item.from, DelegationExpired, "Delegation end time reached", now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(items), nil
}
func recordSystemDelegationTransition(ctx context.Context, tx pgx.Tx, tenantID, id string, from, to DelegationState, rationale string, at time.Time) error {
	_, err := tx.Exec(ctx, `INSERT INTO governance_decisions(tenant_id,object_type,object_id,from_state,to_state,actor_type,actor_id,rationale,decided_at) VALUES($1::uuid,'DELEGATION',$2::uuid,$3::text,$4::text,'SYSTEM',NULL,$5,$6)`, tenantID, id, from, to, rationale, at)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at) VALUES($1::uuid,'DELEGATION',$2::uuid,'DelegationStateChanged',jsonb_build_object('from',$3::text,'to',$4::text,'actor_type','SYSTEM','rationale',$5::text),$6,$6)`, tenantID, id, from, to, rationale, at)
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
