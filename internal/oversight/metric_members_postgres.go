//go:build postgres

package oversight

import (
	"context"
	"fmt"
	"sort"
	"time"
)

func (r *PostgresRepository) buildMetricMembers(
	ctx context.Context,
	scope Scope,
	now time.Time,
	organizationScopeIDs []string,
) ([]MetricMember, error) {
	values := make([]MetricMember, 0, 64)

	rows, err := r.pool.Query(ctx, `
		WITH scoped AS (
			SELECT m.*,
			       CASE
			         WHEN NOT (scope ? 'access') OR upper(btrim(scope->>'access')) IN ('PUBLIC','INTERNAL') THEN 'INCLUDED'
			         WHEN upper(btrim(scope->>'access'))='RESTRICTED' THEN 'EXCLUDED'
			         ELSE 'UNKNOWN'
			       END scope_state
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
		), classified AS (
			SELECT m.id::text,m.reference,m.title,m.status,m.priority,m.due_at,
			       m.priority>=4 AS critical_high,
			       COALESCE(m.due_at<$3::timestamptz,false) AS overdue,
			       EXISTS (
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
			       ) AS outcome_failure
			FROM scoped m
			WHERE m.scope_state='INCLUDED'
			  AND m.status NOT IN ('CLOSED','CANCELLED')
		)
		SELECT id,reference,title,status,priority,due_at,critical_high,overdue,outcome_failure
		FROM classified
		WHERE critical_high OR overdue OR outcome_failure
		ORDER BY id`,
		scope.TenantID, scope.LegalEntityID, now, organizationScopeIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("load matter metric membership: %w", err)
	}
	for rows.Next() {
		var id, reference, title, state string
		var priority int
		var dueAt *time.Time
		var critical, overdue, outcome bool
		if err := rows.Scan(&id, &reference, &title, &state, &priority, &dueAt, &critical, &overdue, &outcome); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan matter metric membership: %w", err)
		}
		priorityCopy := priority
		appendMatter := func(metricID string) {
			values = append(values, MetricMember{
				MetricID: metricID, MemberType: "MATTER", MemberID: id,
				SubjectType: "MATTER", SubjectID: id, Reference: reference,
				Title: title, State: state, Priority: &priorityCopy, DueAt: cloneMetricMemberTime(dueAt),
			})
		}
		if critical {
			appendMatter(MetricCriticalHighOpen)
		}
		if overdue {
			appendMatter(MetricOverdueOpen)
		}
		if outcome {
			appendMatter(MetricOutcomeFailures)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate matter metric membership: %w", err)
	}
	rows.Close()

	rows, err = r.pool.Query(ctx, `
		SELECT wt.id::text,wt.title,wt.status,wt.due_at,
		       wi.subject_type,wi.subject_id::text,
		       COALESCE(m.reference,p.code,'')
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
		  AND wi.subject_type IN ('MATTER','PROGRAM')
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
		    ($3::uuid[] IS NULL AND (
		      (m.id IS NOT NULL AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL')))
		      OR
		      (p.id IS NOT NULL AND (NOT (p.scope ? 'access') OR upper(btrim(p.scope->>'access')) IN ('PUBLIC','INTERNAL')))
		    ))
		    OR
		    ($3::uuid[] IS NOT NULL
		      AND m.id IS NOT NULL
		      AND m.organization_scope_id=ANY($3::uuid[])
		      AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL')))
		  )
		ORDER BY wt.id`,
		scope.TenantID, scope.LegalEntityID, organizationScopeIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("load routing-gap metric membership: %w", err)
	}
	for rows.Next() {
		var member MetricMember
		if err := rows.Scan(
			&member.MemberID,
			&member.Title,
			&member.State,
			&member.DueAt,
			&member.SubjectType,
			&member.SubjectID,
			&member.Reference,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan routing-gap metric membership: %w", err)
		}
		member.MetricID = MetricRoutingGaps
		member.MemberType = "WORKFLOW_TASK"
		values = append(values, member)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate routing-gap metric membership: %w", err)
	}
	rows.Close()

	sort.Slice(values, func(i, j int) bool {
		if values[i].MetricID != values[j].MetricID {
			return values[i].MetricID < values[j].MetricID
		}
		if values[i].MemberType != values[j].MemberType {
			return values[i].MemberType < values[j].MemberType
		}
		return values[i].MemberID < values[j].MemberID
	})
	return values, nil
}

func applyMetricMemberCounts(value *Snapshot) {
	if value == nil {
		return
	}
	value.Counts.CriticalHigh = 0
	value.Counts.Overdue = 0
	value.Counts.RoutingFailures = 0
	value.Counts.OutcomeFailures = 0
	for _, member := range value.MetricMembers {
		switch member.MetricID {
		case MetricCriticalHighOpen:
			value.Counts.CriticalHigh++
		case MetricOverdueOpen:
			value.Counts.Overdue++
		case MetricRoutingGaps:
			value.Counts.RoutingFailures++
		case MetricOutcomeFailures:
			value.Counts.OutcomeFailures++
		}
	}
}

func cloneMetricMemberTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := value.UTC()
	return &copyValue
}
