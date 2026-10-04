//go:build postgres

package oversight

import (
	"context"
	"fmt"

	"github.com/CloudSpaceLab/clearsight-grc/internal/metricid"
	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) buildMetricMembers(
	ctx context.Context,
	scope Scope,
	now time.Time,
	organizationScopeIDs []string,
) ([]MetricMember, error) {
	rows, err := r.pool.Query(ctx, `
		WITH matter_scope AS (
			SELECT m.id,m.title,m.priority,m.due_at,m.tenant_id
			FROM matters m
			WHERE m.tenant_id=$1::uuid
			  AND m.legal_entity_id=$2::uuid
			  AND ($4::uuid[] IS NULL OR m.organization_scope_id=ANY($4::uuid[]))
			  AND m.status NOT IN ('CLOSED','CANCELLED')
			  AND NOT EXISTS (
			    SELECT 1 FROM demo_record_archives archive
			    WHERE archive.tenant_id=m.tenant_id
			      AND archive.legal_entity_id=m.legal_entity_id
			      AND archive.record_type='MATTER'
			      AND archive.record_id=m.id
			      AND archive.restored_at IS NULL
			  )
			  AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL'))
		), matter_members AS (
			SELECT $5::text metric_id,'MATTER'::text target_type,m.id target_id,m.title label,
			       'MATTER'::text subject_type,m.id subject_id
			FROM matter_scope m WHERE m.priority>=4
			UNION ALL
			SELECT $6::text,'MATTER',m.id,m.title,'MATTER',m.id
			FROM matter_scope m WHERE m.due_at<$3::timestamptz
			UNION ALL
			SELECT $7::text,'MATTER',m.id,m.title,'MATTER',m.id
			FROM matter_scope m
			WHERE EXISTS (
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
		), routing_members AS (
			SELECT $8::text metric_id,'WORKFLOW_TASK'::text target_type,wt.id target_id,wt.title label,
			       wi.subject_type,wi.subject_id
			FROM workflow_tasks wt
			JOIN workflow_instances wi ON wi.tenant_id=wt.tenant_id AND wi.id=wt.workflow_id
			LEFT JOIN matters m ON wi.subject_type='MATTER' AND m.tenant_id=wi.tenant_id AND m.id=wi.subject_id
			LEFT JOIN programs p ON wi.subject_type='PROGRAM' AND p.tenant_id=wi.tenant_id AND p.id=wi.subject_id
			WHERE wt.tenant_id=$1::uuid
			  AND wt.status IN ('READY','BLOCKED','ESCALATED')
			  AND wt.principal_id IS NULL
			  AND COALESCE(m.legal_entity_id,p.legal_entity_id)=$2::uuid
			  AND NOT EXISTS (
			    SELECT 1 FROM demo_record_archives archive
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
			      AND (NOT (m.scope ? 'access') OR upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL'))
			    )
			  )
			  AND wi.subject_type IN ('MATTER','PROGRAM')
		)
		SELECT metric_id,target_type,target_id::text,label,subject_type,subject_id::text
		FROM (
			SELECT * FROM matter_members
			UNION ALL
			SELECT * FROM routing_members
		) members
		ORDER BY metric_id,target_type,target_id`,
		scope.TenantID,
		scope.LegalEntityID,
		now,
		organizationScopeIDs,
		metricid.CriticalHighOpen,
		metricid.OverdueOpen,
		metricid.OutcomeFailures,
		metricid.RoutingGaps,
	)
	if err != nil {
		return nil, fmt.Errorf("load exact oversight metric members: %w", err)
	}
	defer rows.Close()

	values := make([]MetricMember, 0, 64)
	for rows.Next() {
		var value MetricMember
		if err := rows.Scan(
			&value.MetricID,
			&value.TargetType,
			&value.TargetID,
			&value.Label,
			&value.SubjectType,
			&value.SubjectID,
		); err != nil {
			return nil, fmt.Errorf("scan exact oversight metric member: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate exact oversight metric members: %w", err)
	}
	return values, nil
}

func applyMetricMemberCounts(counts *Counts, members []MetricMember) {
	if counts == nil {
		return
	}
	counts.CriticalHigh = 0
	counts.Overdue = 0
	counts.RoutingFailures = 0
	counts.OutcomeFailures = 0
	for _, member := range members {
		switch member.MetricID {
		case metricid.CriticalHighOpen:
			counts.CriticalHigh++
		case metricid.OverdueOpen:
			counts.Overdue++
		case metricid.RoutingGaps:
			counts.RoutingFailures++
		case metricid.OutcomeFailures:
			counts.OutcomeFailures++
		}
	}
}

var _ pgx.Rows
