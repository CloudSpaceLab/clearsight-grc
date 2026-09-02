//go:build postgres

package workflow

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]Task, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT wt.id::text,t.id::text,wt.workflow_id::text,wt.step_key,wt.responsibility,
		       COALESCE(wt.principal_id::text,''),wt.title,wt.status,wt.due_at,wt.claimed_at,wt.completed_at,
		       wt.context,wt.source_bindings,wt.version,wt.created_at,wt.updated_at,
		       wi.kind,COALESCE(m.id::text,''),COALESCE(m.priority,0),COALESCE(m.scope,'{}'::jsonb),
		       COALESCE(cr.id::text,''),COALESCE(cr.recipient_principal_id::text,''),
		       (CASE
		         WHEN cr.id IS NULL THEN false
		         WHEN cr.subject_type<>'MATTER' THEN true
		         WHEN em.id IS NULL THEN false
		         WHEN NOT (em.scope ? 'access') THEN true
		         WHEN upper(btrim(em.scope->>'access')) IN ('PUBLIC','INTERNAL') THEN true
		         WHEN upper(btrim(em.scope->>'access'))='RESTRICTED' THEN
		           CASE
		             WHEN jsonb_typeof(em.scope->'allowed_principal_ids')='array'
		              AND NOT EXISTS (
		                SELECT 1 FROM jsonb_array_elements(em.scope->'allowed_principal_ids') AS entry(value)
		                WHERE jsonb_typeof(entry.value)<>'string'
		              )
		              AND EXISTS (
		                SELECT 1 FROM jsonb_array_elements_text(em.scope->'allowed_principal_ids') AS nonblank(value)
		                WHERE btrim(nonblank.value)<>''
		              )
		             THEN EXISTS (
		               SELECT 1 FROM jsonb_array_elements_text(em.scope->'allowed_principal_ids') AS allowed(value)
		               WHERE btrim(allowed.value)=$2
		             )
		             ELSE false
		           END
		         ELSE false
		       END)
		       OR COALESCE(em.owner_principal_id::text,'')=$2
		       OR EXISTS (SELECT 1 FROM matter_actions assigned_evidence_action WHERE assigned_evidence_action.tenant_id=em.tenant_id AND assigned_evidence_action.matter_id=em.id AND assigned_evidence_action.owner_principal_id::text=$2)
		FROM workflow_tasks wt
		JOIN tenants t ON t.id=wt.tenant_id
		JOIN workflow_instances wi ON wi.id=wt.workflow_id AND wi.tenant_id=wt.tenant_id
		LEFT JOIN matter_actions ma
		  ON wi.kind='MATTER_ACTION'
		 AND wi.subject_type='MATTER_ACTION'
		 AND ma.tenant_id=wi.tenant_id
		 AND ma.id=wi.subject_id
		LEFT JOIN matters m ON m.tenant_id=wi.tenant_id AND (
		  (wi.kind='MATTER_ACTION' AND m.id=ma.matter_id) OR
		  (wi.kind='MATTER_LIFECYCLE' AND wi.subject_type='MATTER' AND m.id=wi.subject_id)
		)
		LEFT JOIN capture_requests cr
		  ON wi.kind='EVIDENCE_REQUEST'
		 AND wi.subject_type='EVIDENCE_REQUEST'
		 AND cr.tenant_id=wi.tenant_id
		 AND cr.id=wi.subject_id
		LEFT JOIN matters em
		  ON cr.subject_type='MATTER'
		 AND em.tenant_id=cr.tenant_id
		 AND em.id::text=cr.subject_id
		WHERE (t.slug=$1 OR t.id::text=$1)
		  AND ($2='' OR wt.principal_id::text=$2)
		  AND ($3='' OR wt.status=$3)
		  AND ($4='' OR wi.kind=$4)
		  AND (NOT $5::boolean OR wt.status NOT IN ('COMPLETED','CANCELLED'))
		  AND ($8='' OR $8='*' OR COALESCE(m.legal_entity_id,cr.legal_entity_id)=(
		    SELECT le.id FROM legal_entities le
		    WHERE le.tenant_id=t.id AND (le.id::text=$8 OR le.code=$8)
		    ORDER BY le.valid_from DESC,le.id LIMIT 1
		  ))
		  AND (
		    NOT $6::boolean OR
		    (CASE
		      WHEN wi.kind NOT IN ('MATTER_ACTION','MATTER_LIFECYCLE') OR m.id IS NULL THEN false
		      WHEN NOT (m.scope ? 'access') THEN true
		      WHEN upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL') THEN true
		      WHEN upper(btrim(m.scope->>'access'))='RESTRICTED' THEN
		        CASE
		          WHEN jsonb_typeof(m.scope->'allowed_principal_ids')='array'
		           AND NOT EXISTS (
		             SELECT 1 FROM jsonb_array_elements(m.scope->'allowed_principal_ids') AS entry(value)
		             WHERE jsonb_typeof(entry.value)<>'string'
		           )
		           AND EXISTS (
		             SELECT 1 FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') AS nonblank(value)
		             WHERE btrim(nonblank.value)<>''
		           )
		          THEN EXISTS (
		            SELECT 1 FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') AS allowed(value)
		            WHERE btrim(allowed.value)=$2
		          )
		          ELSE false
		        END
		      ELSE false
		    END)
		    OR COALESCE(m.owner_principal_id::text,'')=$2
		    OR EXISTS (SELECT 1 FROM matter_actions assigned_matter_action WHERE assigned_matter_action.tenant_id=m.tenant_id AND assigned_matter_action.matter_id=m.id AND assigned_matter_action.owner_principal_id::text=$2)
		  )
		  AND (
		    NOT $7::boolean OR
		    CASE
		      WHEN wi.kind IN ('MATTER_ACTION','MATTER_LIFECYCLE') THEN
		        (CASE
		          WHEN m.id IS NULL THEN false
		          WHEN NOT (m.scope ? 'access') THEN true
		          WHEN upper(btrim(m.scope->>'access')) IN ('PUBLIC','INTERNAL') THEN true
		          WHEN upper(btrim(m.scope->>'access'))='RESTRICTED' THEN
		            CASE
		              WHEN jsonb_typeof(m.scope->'allowed_principal_ids')='array'
		               AND NOT EXISTS (
		                 SELECT 1 FROM jsonb_array_elements(m.scope->'allowed_principal_ids') AS entry(value)
		                 WHERE jsonb_typeof(entry.value)<>'string'
		               )
		               AND EXISTS (
		                 SELECT 1 FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') AS nonblank(value)
		                 WHERE btrim(nonblank.value)<>''
		               )
		              THEN EXISTS (
		                SELECT 1 FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') AS allowed(value)
		                WHERE btrim(allowed.value)=$2
		              )
		              ELSE false
		            END
		          ELSE false
		        END)
		        OR COALESCE(m.owner_principal_id::text,'')=$2
		        OR EXISTS (SELECT 1 FROM matter_actions assigned_work_action WHERE assigned_work_action.tenant_id=m.tenant_id AND assigned_work_action.matter_id=m.id AND assigned_work_action.owner_principal_id::text=$2)
		      WHEN wi.kind='EVIDENCE_REQUEST' THEN
		        cr.id IS NOT NULL
		        AND cr.recipient_type='INTERNAL_PRINCIPAL'
		        AND cr.recipient_state='ASSIGNED'
		        AND cr.recipient_principal_id::text=$2
		        AND cr.status IN ('READY','IN_PROGRESS')
		        AND cr.deadline>now()
		        AND ((CASE
		          WHEN cr.subject_type<>'MATTER' THEN true
		          WHEN em.id IS NULL THEN false
		          WHEN NOT (em.scope ? 'access') THEN true
		          WHEN upper(btrim(em.scope->>'access')) IN ('PUBLIC','INTERNAL') THEN true
		          WHEN upper(btrim(em.scope->>'access'))='RESTRICTED' THEN
		            CASE
		              WHEN jsonb_typeof(em.scope->'allowed_principal_ids')='array'
		               AND NOT EXISTS (
		                 SELECT 1 FROM jsonb_array_elements(em.scope->'allowed_principal_ids') AS entry(value)
		                 WHERE jsonb_typeof(entry.value)<>'string'
		               )
		               AND EXISTS (
		                 SELECT 1 FROM jsonb_array_elements_text(em.scope->'allowed_principal_ids') AS nonblank(value)
		                 WHERE btrim(nonblank.value)<>''
		               )
		              THEN EXISTS (
		                SELECT 1 FROM jsonb_array_elements_text(em.scope->'allowed_principal_ids') AS allowed(value)
		                WHERE btrim(allowed.value)=$2
		              )
		              ELSE false
		            END
		          ELSE false
		        END)
		        OR COALESCE(em.owner_principal_id::text,'')=$2
		        OR EXISTS (SELECT 1 FROM matter_actions assigned_request_action WHERE assigned_request_action.tenant_id=em.tenant_id AND assigned_request_action.matter_id=em.id AND assigned_request_action.owner_principal_id::text=$2))
		      ELSE false
		    END
		  )
		ORDER BY
		  CASE WHEN $5::boolean THEN wt.due_at IS NULL ELSE false END ASC,
		  CASE WHEN $5::boolean THEN wt.due_at END ASC NULLS LAST,
		  wt.updated_at DESC,
		  wt.id ASC
		LIMIT $9`,
		filter.TenantID, filter.PrincipalID, string(filter.Status), filter.WorkflowKind,
		filter.ActiveOnly, filter.VisibleMatterWorkOnly, filter.VisibleActorWorkOnly, filter.LegalEntityID, filter.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list workflow tasks: %w", err)
	}
	defer rows.Close()
	values := []Task{}
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, task)
	}
	return values, rows.Err()
}

type scanner interface{ Scan(...any) error }

func scanTask(row scanner) (Task, error) {
	var task Task
	var contextJSON []byte
	var sourceBindingsJSON []byte
	var matterScope []byte
	if err := row.Scan(
		&task.ID, &task.TenantID, &task.WorkflowID, &task.StepKey, &task.Responsibility,
		&task.PrincipalID, &task.Title, &task.Status, &task.DueAt, &task.ClaimedAt, &task.CompletedAt,
		&contextJSON, &sourceBindingsJSON, &task.Version, &task.CreatedAt, &task.UpdatedAt,
		&task.WorkflowKind, &task.MatterID, &task.MatterPriority, &matterScope,
		&task.EvidenceRequestID, &task.EvidenceRecipientID, &task.EvidenceSubjectVisible,
	); err != nil {
		return Task{}, err
	}
	if err := json.Unmarshal(contextJSON, &task.Context); err != nil {
		return Task{}, fmt.Errorf("decode workflow context: %w", err)
	}
	if err := json.Unmarshal(sourceBindingsJSON, &task.SourceBindings); err != nil {
		return Task{}, fmt.Errorf("decode workflow source bindings: %w", err)
	}
	task.MatterScope = append(task.MatterScope[:0], matterScope...)
	return task, nil
}
