//go:build postgres

package workflow

import (
	"context"
	"fmt"
)

func (r *PostgresRepository) ListDocumentProposalActorWork(ctx context.Context, filter ListFilter) ([]Task, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT wt.id::text,t.id::text,wt.workflow_id::text,wt.step_key,wt.responsibility,
		       COALESCE(wt.principal_id::text,''),wt.title,wt.status,wt.due_at,wt.claimed_at,wt.completed_at,
		       wt.context,wt.source_bindings,wt.version,wt.created_at,wt.updated_at,
		       wi.kind,''::text,0,'{}'::jsonb,''::text,''::text,false
		FROM workflow_tasks wt
		JOIN tenants t ON t.id=wt.tenant_id
		JOIN workflow_instances wi ON wi.id=wt.workflow_id AND wi.tenant_id=wt.tenant_id
		JOIN LATERAL (
			SELECT COALESCE(di.legal_entity_id::text,'') AS legal_entity_id,
			       proposal.value->'handoff'->>'status' AS handoff_status
			FROM document_imports di
			CROSS JOIN LATERAL jsonb_array_elements(di.proposals) AS proposal(value)
			WHERE di.tenant_id=wi.tenant_id
			  AND proposal.value->>'status'='ACCEPTED'
			  AND proposal.value->'handoff'->>'id'=wi.subject_id::text
			LIMIT 1
		) current_handoff ON true
		WHERE (t.slug=$1 OR t.id::text=$1)
		  AND wt.principal_id::text=$2
		  AND ($3='*' OR ($3<>'' AND current_handoff.legal_entity_id=$3))
		  AND ($4='' OR wt.status=$4)
		  AND wi.kind=$5
		  AND wi.subject_type='DOCUMENT_PROPOSAL'
		  AND (NOT $6::boolean OR wt.status NOT IN ('COMPLETED','CANCELLED'))
		  AND (
		    (wt.step_key='document-proposal-review' AND current_handoff.handoff_status='AWAITING_REVIEW')
		    OR
		    (wt.step_key='document-proposal-authorization' AND current_handoff.handoff_status='AWAITING_AUTHORIZATION')
		  )
		ORDER BY wt.due_at IS NULL,wt.due_at ASC NULLS LAST,wt.updated_at DESC,wt.id ASC
		LIMIT $7`,
		filter.TenantID, filter.PrincipalID, filter.LegalEntityID, string(filter.Status),
		DocumentProposalWorkflowKind, filter.ActiveOnly, filter.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list actor document proposal work: %w", err)
	}
	defer rows.Close()
	values := make([]Task, 0, filter.Limit)
	for rows.Next() {
		task, scanErr := scanTask(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		task.DocumentProposalVisible = true
		values = append(values, task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

var _ documentProposalActorWorkRepository = (*PostgresRepository)(nil)
