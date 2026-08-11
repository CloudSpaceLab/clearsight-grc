//go:build postgres

package governance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) CreatePolicyRevision(ctx context.Context, tenantID, policyID string, expected int64, actor string, definition []byte, checksum string, at time.Time) (RoutingPolicyRevision, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return RoutingPolicyRevision{}, err
	}
	defer tx.Rollback(ctx)

	var status PolicyState
	var currentVersion int
	var policyVersion int64
	err = tx.QueryRow(ctx, `
		SELECT status,current_version,version
		FROM routing_policies
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id::text=$2
		FOR UPDATE`, tenantID, policyID).Scan(&status, &currentVersion, &policyVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoutingPolicyRevision{}, ErrNotFound
	}
	if err != nil {
		return RoutingPolicyRevision{}, err
	}
	if status != PolicyActive {
		return RoutingPolicyRevision{}, ErrInvalidTransition
	}
	if policyVersion != expected {
		return RoutingPolicyRevision{}, ErrVersionConflict
	}

	var latestVersion int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(max(version),$2) FROM routing_policy_versions WHERE policy_id=$1::uuid`, policyID, currentVersion).Scan(&latestVersion); err != nil {
		return RoutingPolicyRevision{}, err
	}
	nextVersion := latestVersion + 1
	_, err = tx.Exec(ctx, `
		INSERT INTO routing_policy_versions(policy_id,version,definition,checksum,created_by,created_at)
		VALUES($1::uuid,$2,$3::jsonb,$4,$5::uuid,$6)`, policyID, nextVersion, definition, checksum, actor, at)
	if err != nil {
		return RoutingPolicyRevision{}, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE routing_policies
		SET updated_at=$3,version=version+1
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id::text=$2`, tenantID, policyID, at)
	if err != nil {
		return RoutingPolicyRevision{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RoutingPolicyRevision{}, err
	}
	return RoutingPolicyRevision{
		PolicyID: policyID, TenantID: tenantID, Version: nextVersion, BaseVersion: currentVersion,
		Definition: append([]byte(nil), definition...), Checksum: checksum, MakerID: actor, CreatedAt: at,
	}, nil
}

func (r *PostgresRepository) PendingPolicyRevision(ctx context.Context, tenantID, policyID string) (RoutingPolicyRevision, error) {
	var revision RoutingPolicyRevision
	var approvedAt, effectiveFrom, effectiveUntil *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT rp.id::text,t.slug,rpv.version,rp.current_version,rpv.definition,rpv.checksum,
		       COALESCE(rpv.created_by::text,''),rpv.created_at,COALESCE(rpv.approved_by::text,''),
		       rpv.approved_at,rpv.effective_from,rpv.effective_until
		FROM routing_policies rp
		JOIN tenants t ON t.id=rp.tenant_id
		JOIN routing_policy_versions rpv ON rpv.policy_id=rp.id
		WHERE rp.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND rp.id::text=$2
		  AND rpv.version>rp.current_version
		  AND rpv.approved_at IS NULL
		ORDER BY rpv.version DESC
		LIMIT 1`, tenantID, policyID).Scan(
		&revision.PolicyID, &revision.TenantID, &revision.Version, &revision.BaseVersion,
		&revision.Definition, &revision.Checksum, &revision.MakerID, &revision.CreatedAt,
		&revision.ApprovedBy, &approvedAt, &effectiveFrom, &effectiveUntil,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoutingPolicyRevision{}, ErrNotFound
	}
	if err != nil {
		return RoutingPolicyRevision{}, err
	}
	revision.ApprovedAt = approvedAt
	revision.EffectiveFrom = effectiveFrom
	revision.EffectiveUntil = effectiveUntil
	return revision, nil
}

func (r *PostgresRepository) ActivatePolicyRevision(ctx context.Context, tenantID, policyID string, expected int64, revisionVersion int, actor, rationale string, at time.Time) (RoutingPolicy, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return RoutingPolicy{}, err
	}
	defer tx.Rollback(ctx)

	var status PolicyState
	var currentVersion int
	var policyVersion int64
	err = tx.QueryRow(ctx, `
		SELECT status,current_version,version
		FROM routing_policies
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id::text=$2
		FOR UPDATE`, tenantID, policyID).Scan(&status, &currentVersion, &policyVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoutingPolicy{}, ErrNotFound
	}
	if err != nil {
		return RoutingPolicy{}, err
	}
	if status != PolicyActive {
		return RoutingPolicy{}, ErrInvalidTransition
	}
	if policyVersion != expected {
		return RoutingPolicy{}, ErrVersionConflict
	}

	var latestVersion int
	var maker string
	err = tx.QueryRow(ctx, `
		SELECT rpv.version,COALESCE(rpv.created_by::text,'')
		FROM routing_policy_versions rpv
		WHERE rpv.policy_id=$1::uuid AND rpv.version>$2 AND rpv.approved_at IS NULL
		ORDER BY rpv.version DESC
		LIMIT 1`, policyID, currentVersion).Scan(&latestVersion, &maker)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoutingPolicy{}, ErrNotFound
	}
	if err != nil {
		return RoutingPolicy{}, err
	}
	if latestVersion != revisionVersion {
		return RoutingPolicy{}, ErrRevisionStale
	}
	if maker == "" || maker == actor {
		return RoutingPolicy{}, ErrMakerChecker
	}

	_, err = tx.Exec(ctx, `
		UPDATE routing_policy_versions
		SET effective_until=$3
		WHERE policy_id=$1::uuid AND version=$2 AND (effective_until IS NULL OR $3<effective_until)`, policyID, currentVersion, at)
	if err != nil {
		return RoutingPolicy{}, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE routing_policy_versions
		SET approved_by=$3::uuid,approved_at=$4,effective_from=$4,effective_until=NULL
		WHERE policy_id=$1::uuid AND version=$2 AND approved_at IS NULL`, policyID, revisionVersion, actor, at)
	if err != nil {
		return RoutingPolicy{}, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE routing_policies
		SET current_version=$3,checker_id=$4::uuid,approved_at=$5,updated_at=$5,version=version+1
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id::text=$2`, tenantID, policyID, revisionVersion, actor, at)
	if err != nil {
		return RoutingPolicy{}, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO governance_decisions(tenant_id,object_type,object_id,from_state,to_state,actor_id,rationale,decided_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'ROUTING_POLICY',$2::uuid,'ACTIVE','ACTIVE',$3::uuid,$4,$5)`, tenantID, policyID, actor, rationale, at)
	if err != nil {
		return RoutingPolicy{}, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'ROUTING_POLICY',$2::uuid,'RoutingPolicyRevisionActivated',
		       jsonb_build_object('from_version',$3::int,'to_version',$4::int,'actor_id',$5::text,'rationale',$6::text),$7,$7)`,
		tenantID, policyID, currentVersion, revisionVersion, actor, rationale, at)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RoutingPolicy{}, err
	}
	return r.GetPolicy(ctx, tenantID, policyID)
}

func (r *PostgresRepository) EscalationReferenceConflicts(ctx context.Context, tenantID string, definition []byte) ([]ConflictFinding, error) {
	sequences, err := ParseEscalationSequences(json.RawMessage(definition))
	if err != nil {
		return nil, err
	}
	roleSet := map[string]struct{}{}
	groupSet := map[string]struct{}{}
	for _, sequence := range sequences {
		for _, step := range sequence.Steps {
			for _, role := range step.SourceRoles {
				roleSet[role] = struct{}{}
			}
			for _, role := range step.TargetRoles {
				roleSet[role] = struct{}{}
			}
			for _, groupID := range step.TargetGroupIDs {
				groupSet[groupID] = struct{}{}
			}
		}
	}

	roles := make([]string, 0, len(roleSet))
	for role := range roleSet {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	groups := make([]string, 0, len(groupSet))
	for groupID := range groupSet {
		groups = append(groups, groupID)
	}
	sort.Strings(groups)

	findings := make([]ConflictFinding, 0)
	for _, role := range roles {
		var count int
		if err := r.pool.QueryRow(ctx, `
			SELECT count(*)
			FROM role_templates
			WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND code=$2
			  AND valid_from<=clock_timestamp() AND (valid_until IS NULL OR clock_timestamp()<valid_until)`, tenantID, role).Scan(&count); err != nil {
			return nil, err
		}
		if count != 1 {
			findings = append(findings, ConflictFinding{Code: "ESCALATION_ROLE_REFERENCE", Summary: fmt.Sprintf("Escalation role %s does not resolve to exactly one active role template.", role)})
		}
	}
	for _, groupID := range groups {
		var count int
		if err := r.pool.QueryRow(ctx, `
			SELECT count(*)
			FROM directory_groups dg
			JOIN scim_sources ss ON ss.tenant_id=dg.tenant_id AND ss.id=dg.source_id AND ss.status='ACTIVE'
			WHERE dg.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND dg.id::text=$2 AND dg.deleted_at IS NULL`, tenantID, groupID).Scan(&count); err != nil {
			return nil, err
		}
		if count != 1 {
			findings = append(findings, ConflictFinding{Code: "ESCALATION_GROUP_REFERENCE", Summary: fmt.Sprintf("Escalation directory group %s is not active in the tenant.", groupID)})
		}
	}
	return findings, nil
}

func latestPendingRevisionDefinition(ctx context.Context, repo Repository, tenantID, policyID, actor string) ([]byte, error) {
	revision, err := repo.PendingPolicyRevision(ctx, tenantID, policyID)
	if errors.Is(err, ErrNotFound) {
		policy, policyErr := repo.GetPolicy(ctx, tenantID, policyID)
		if policyErr != nil {
			return nil, policyErr
		}
		return append([]byte(nil), policy.Definition...), nil
	}
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(revision.MakerID) != strings.TrimSpace(actor) {
		return nil, ErrMakerChecker
	}
	return append([]byte(nil), revision.Definition...), nil
}
