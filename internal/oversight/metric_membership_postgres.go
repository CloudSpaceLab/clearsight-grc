//go:build postgres

package oversight

import (
	"context"
	"fmt"
	"time"
)

const (
	MetricSnapshotDrillDefinitionRevision = "home-oversight-v3"
	MetricCriticalHighOpen                 = "critical_high_open"
	MetricOverdueOpen      = "overdue_open"
	MetricRoutingGaps      = "routing_gaps"
	MetricOutcomeFailures  = "outcome_failures"
)

func (r *PostgresRepository) buildMetricMembers(
	ctx context.Context,
	scope Scope,
	now time.Time,
	organizationScopeIDs []string,
) ([]MetricMember, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT metric_id,member_id,target_type,target_id,target_title,state
		FROM (
			SELECT
				'critical_high_open'::text metric_id,
				m.id::text member_id,
				'MATTER'::text target_type,
				m.id::text target_id,
				m.title target_title,
				m.status state
			FROM matters m
			WHERE m.tenant_id=$1::uuid
			  AND m.legal_entity_id=$2::uuid
			  AND ($4::uuid[] IS NULL OR m.organization_scope_id=ANY($4::uuid[]))
			  AND NOT EXISTS (
				SELECT 1
				FROM demo_record_archives archive
				WHERE archive.tenant_id=m.tenant_id
				  AND archive.legal_entity_id=m.legal_entity_id
				  AND archive.record_type='MATTER'
				  AND archive.record_id=m.id
				  AND archive.restored_at IS NULL
			  )
			  AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL'))
			  AND m.status NOT IN ('CLOSED','CANCELLED')
			  AND m.priority>=4

			UNION ALL

			SELECT
				'overdue_open'::text,
				m.id::text,
				'MATTER'::text,
				m.id::text,
				m.title,
				m.status
			FROM matters m
			WHERE m.tenant_id=$1::uuid
			  AND m.legal_entity_id=$2::uuid
			  AND ($4::uuid[] IS NULL OR m.organization_scope_id=ANY($4::uuid[]))
			  AND NOT EXISTS (
				SELECT 1
				FROM demo_record_archives archive
				WHERE archive.tenant_id=m.tenant_id
				  AND archive.legal_entity_id=m.legal_entity_id
				  AND archive.record_type='MATTER'
				  AND archive.record_id=m.id
				  AND archive.restored_at IS NULL
			  )
			  AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL'))
			  AND m.status NOT IN ('CLOSED','CANCELLED')
			  AND m.due_at<$3::timestamptz

			UNION ALL

			SELECT
				'outcome_failures'::text,
				m.id::text,
				'MATTER'::text,
				m.id::text,
				m.title,
				m.status
			FROM matters m
			WHERE m.tenant_id=$1::uuid
			  AND m.legal_entity_id=$2::uuid
			  AND ($4::uuid[] IS NULL OR m.organization_scope_id=ANY($4::uuid[]))
			  AND NOT EXISTS (
				SELECT 1
				FROM demo_record_archives archive
				WHERE archive.tenant_id=m.tenant_id
				  AND archive.legal_entity_id=m.legal_entity_id
				  AND archive.record_type='MATTER'
				  AND archive.record_id=m.id
				  AND archive.restored_at IS NULL
			  )
			  AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL'))
			  AND m.status NOT IN ('CLOSED','CANCELLED')
			  AND EXISTS (
				SELECT 1
				FROM verification_results vr
				WHERE vr.tenant_id=m.tenant_id
				  AND vr.matter_id=m.id
				  AND vr.result IN ('FAIL','INCONCLUSIVE')
				  AND vr.observed_at=(
					SELECT max(latest.observed_at)
					FROM verification_results latest
					WHERE latest.tenant_id=vr.tenant_id
					  AND latest.matter_id=vr.matter_id
					  AND latest.contract_id=vr.contract_id
				  )
			  )

			UNION ALL

			SELECT
				'routing_gaps'::text,
				wt.id::text,
				wi.subject_type,
				wi.subject_id::text,
				COALESCE(m.title,p.name,''),
				wt.status
			FROM workflow_tasks wt
			JOIN workflow_instances wi
			  ON wi.tenant_id=wt.tenant_id
			 AND wi.id=wt.workflow_id
			LEFT JOIN matters m
			  ON wi.subject_type='MATTER'
			 AND m.tenant_id=wi.tenant_id
			 AND m.id=wi.subject_id
			LEFT JOIN programs p
			  ON wi.subject_type='PROGRAM'
			 AND p.tenant_id=wi.tenant_id
			 AND p.id=wi.subject_id
			WHERE wt.tenant_id=$1::uuid
			  AND wt.status IN ('READY','BLOCKED','ESCALATED')
			  AND wt.principal_id IS NULL
			  AND COALESCE(m.legal_entity_id,p.legal_entity_id)=$2::uuid
			  AND NOT EXISTS (
				SELECT 1
				FROM demo_record_archives archive
				WHERE archive.tenant_id=m.tenant_id
				  AND archive.legal_entity_id=m.legal_entity_id
				  AND archive.record_type='MATTER'
				  AND archive.record_id=m.id
				  AND archive.restored_at IS NULL
			  )
			  AND (
				($4::uuid[] IS NULL AND (
					(m.id IS NOT NULL AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL')))
					OR
					(p.id IS NOT NULL AND (NOT (p.scope ? 'access') OR upper(btrim(p.scope->>'access')) IN ('PUBLIC','INTERNAL')))
				))
				OR
				($4::uuid[] IS NOT NULL
				 AND m.id IS NOT NULL
				 AND m.organization_scope_id=ANY($4::uuid[])
				 AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL')))
			  )
		) members
		ORDER BY metric_id,member_id`,
		scope.TenantID,
		scope.LegalEntityID,
		now,
		organizationScopeIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("build oversight metric membership: %w", err)
	}
	defer rows.Close()

	values := make([]MetricMember, 0, 64)
	seen := make(map[string]struct{})
	for rows.Next() {
		var value MetricMember
		if err := rows.Scan(
			&value.MetricID,
			&value.MemberID,
			&value.TargetType,
			&value.TargetID,
			&value.TargetTitle,
			&value.State,
		); err != nil {
			return nil, fmt.Errorf("scan oversight metric membership: %w", err)
		}
		key := value.MetricID + ":" + value.MemberID
		if value.MemberID == "" || value.TargetID == "" || value.TargetTitle == "" ||
			(value.TargetType != "MATTER" && value.TargetType != "PROGRAM") {
			return nil, fmt.Errorf("invalid oversight metric membership %q", key)
		}
		if _, duplicate := seen[key]; duplicate {
			return nil, fmt.Errorf("duplicate oversight metric membership %q", key)
		}
		seen[key] = struct{}{}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate oversight metric membership: %w", err)
	}
	return values, nil
}

func validateMetricMemberCounts(snapshot Snapshot) error {
	counts := map[string]int{
		MetricCriticalHighOpen: 0,
		MetricOverdueOpen:      0,
		MetricRoutingGaps:      0,
		MetricOutcomeFailures:  0,
	}
	for _, value := range snapshot.MetricMembers {
		if _, ok := counts[value.MetricID]; !ok {
			return fmt.Errorf("unknown metric membership %q", value.MetricID)
		}
		counts[value.MetricID]++
	}
	expected := map[string]int{
		MetricCriticalHighOpen: snapshot.Counts.CriticalHigh,
		MetricOverdueOpen:      snapshot.Counts.Overdue,
		MetricRoutingGaps:      snapshot.Counts.RoutingFailures,
		MetricOutcomeFailures:  snapshot.Counts.OutcomeFailures,
	}
	for metricID, want := range expected {
		if counts[metricID] != want {
			return fmt.Errorf("metric membership count mismatch for %s: got %d want %d", metricID, counts[metricID], want)
		}
	}
	return nil
}
