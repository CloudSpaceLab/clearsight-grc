//go:build postgres

package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type EvidenceRequestProjector struct{ Repo *PostgresRepository }

type evidenceRequestProjection struct {
	ID                   string
	TenantID             string
	Title                string
	Purpose              string
	SubjectType          string
	SubjectID            string
	RecipientPrincipalID string
	RecipientState       string
	RequestStatus        string
	Deadline             time.Time
	EstimatedMinutes     int
	PrincipalActive      bool
	SubjectVisible       bool
}

// Maintain reconciles only requests whose desired projection differs from the
// existing actor work. It is restart-safe and does not walk stable requests on
// every pass. A request with no existing Workflow projection is materialized
// only when it is currently actionable; terminal/ineligible requests do not
// create empty cancelled work merely for historical completeness.
func (p *EvidenceRequestProjector) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if p == nil || p.Repo == nil {
		return 0, nil
	}
	if limit < 1 || limit > 500 {
		limit = 100
	}
	rows, err := p.Repo.pool.Query(ctx, `
		WITH candidate_requests AS (
			SELECT cr.id,cr.id::text AS request_id,t.slug,cr.title,cr.purpose,cr.subject_type,cr.subject_id,
			       COALESCE(cr.recipient_principal_id::text,'') AS recipient_principal_id,
			       cr.recipient_state,cr.status,cr.deadline,cr.estimated_minutes,cr.updated_at,
			       COALESCE(p.status='ACTIVE' AND p.valid_from<=$2 AND (p.valid_until IS NULL OR $2<p.valid_until),false) AS principal_active,
			       CASE
			         WHEN cr.subject_type<>'MATTER' THEN true
			         WHEN m.id IS NULL THEN false
			         WHEN NOT (m.scope ? 'access') THEN true
			         WHEN upper(btrim(COALESCE(m.scope->>'access',''))) IN ('PUBLIC','INTERNAL') THEN true
			         WHEN upper(btrim(COALESCE(m.scope->>'access','')))='RESTRICTED' THEN
			           CASE
			             WHEN jsonb_typeof(m.scope->'allowed_principal_ids')='array'
			              AND NOT EXISTS (
			                SELECT 1
			                FROM jsonb_array_elements(m.scope->'allowed_principal_ids') AS entry(value)
			                WHERE jsonb_typeof(entry.value)<>'string'
			              )
			              AND EXISTS (
			                SELECT 1
			                FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') AS nonblank(value)
			                WHERE btrim(nonblank.value)<>''
			              )
			             THEN EXISTS (
			               SELECT 1
			               FROM jsonb_array_elements_text(m.scope->'allowed_principal_ids') AS allowed(value)
			               WHERE btrim(allowed.value)=COALESCE(cr.recipient_principal_id::text,'')
			             )
			             ELSE false
			           END
			         ELSE false
			       END AS subject_visible
			FROM capture_requests cr
			JOIN tenants t ON t.id=cr.tenant_id
			LEFT JOIN principals p
			  ON p.id=cr.recipient_principal_id
			 AND p.tenant_id=cr.tenant_id
			 AND p.kind='PERSON'
			LEFT JOIN matters m
			  ON cr.subject_type='MATTER'
			 AND m.tenant_id=cr.tenant_id
			 AND m.id::text=cr.subject_id
			WHERE cr.recipient_type='INTERNAL_PRINCIPAL'
		)
		SELECT c.request_id,c.slug,c.title,c.purpose,c.subject_type,c.subject_id,
		       c.recipient_principal_id,c.recipient_state,c.status,c.deadline,c.estimated_minutes,
		       c.principal_active,c.subject_visible
		FROM candidate_requests c
		LEFT JOIN workflow_instances wi
		  ON wi.tenant_id=(SELECT id FROM tenants WHERE slug=c.slug)
		 AND wi.kind='EVIDENCE_REQUEST'
		 AND wi.subject_type='EVIDENCE_REQUEST'
		 AND wi.subject_id=c.id
		LEFT JOIN workflow_tasks wt
		  ON wt.workflow_id=wi.id
		 AND wt.step_key='evidence-response'
		WHERE
		  (
		    wi.id IS NULL
		    AND c.recipient_state='ASSIGNED'
		    AND c.status IN ('READY','IN_PROGRESS')
		    AND c.deadline>$2
		    AND c.principal_active
		    AND c.subject_visible
		  )
		  OR (
		    wi.id IS NOT NULL
		    AND (
		      c.updated_at>wi.updated_at
		      OR (
		        c.recipient_state='ASSIGNED' AND c.status='READY' AND c.deadline>$2
		        AND c.principal_active AND c.subject_visible
		        AND wt.status IS DISTINCT FROM 'READY'
		      )
		      OR (
		        c.recipient_state='ASSIGNED' AND c.status='IN_PROGRESS' AND c.deadline>$2
		        AND c.principal_active AND c.subject_visible
		        AND wt.status IS DISTINCT FROM 'IN_PROGRESS'
		      )
		      OR (c.status='SUBMITTED' AND wt.status IS DISTINCT FROM 'COMPLETED')
		      OR (
		        (
		          c.recipient_state<>'ASSIGNED'
		          OR c.status NOT IN ('READY','IN_PROGRESS','SUBMITTED')
		          OR c.deadline<=$2
		          OR NOT c.principal_active
		          OR NOT c.subject_visible
		        )
		        AND wt.status NOT IN ('COMPLETED','CANCELLED')
		      )
		    )
		  )
		ORDER BY c.updated_at,c.id
		LIMIT $1`, limit, now)
	if err != nil {
		return 0, fmt.Errorf("list evidence request projection drift: %w", err)
	}
	defer rows.Close()
	values := make([]evidenceRequestProjection, 0, limit)
	for rows.Next() {
		var value evidenceRequestProjection
		if err := rows.Scan(
			&value.ID, &value.TenantID, &value.Title, &value.Purpose, &value.SubjectType, &value.SubjectID,
			&value.RecipientPrincipalID, &value.RecipientState, &value.RequestStatus, &value.Deadline, &value.EstimatedMinutes,
			&value.PrincipalActive, &value.SubjectVisible,
		); err != nil {
			return 0, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	processed := 0
	for _, value := range values {
		if err := p.reconcileEvidenceRequest(ctx, value, now); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (p *EvidenceRequestProjector) reconcileEvidenceRequest(ctx context.Context, request evidenceRequestProjection, now time.Time) error {
	status := StatusCancelled
	principalID := ""
	if request.RecipientState == "ASSIGNED" && request.PrincipalActive && request.SubjectVisible && now.Before(request.Deadline) {
		switch request.RequestStatus {
		case "READY":
			status = StatusReady
			principalID = request.RecipientPrincipalID
		case "IN_PROGRESS":
			status = StatusInProgress
			principalID = request.RecipientPrincipalID
		case "SUBMITTED":
			status = StatusCompleted
		}
	} else if request.RequestStatus == "SUBMITTED" {
		status = StatusCompleted
	}

	contextJSON, err := json.Marshal(map[string]string{
		"type":                EvidenceRequestWorkflowKind,
		"evidence_request_id": request.ID,
		"action_target_type":  "EVIDENCE_REQUEST",
		"action_target_id":    request.ID,
		"primary_action":      "Respond",
		"why_now":             request.Purpose,
		"scope":               strings.TrimSpace(request.SubjectType + " " + request.SubjectID),
		"evidence_state":      "Response requested",
		"estimated_minutes":   fmt.Sprintf("%d", request.EstimatedMinutes),
	})
	if err != nil {
		return err
	}

	tx, err := p.Repo.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var workflowID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO workflow_instances(tenant_id,kind,subject_type,subject_id,state,policy_version,context_version,due_at,created_at,updated_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'EVIDENCE_REQUEST','EVIDENCE_REQUEST',$2::uuid,$3,'capture-recipient-v1',1,$4,$5,$5)
		ON CONFLICT(tenant_id,kind,subject_type,subject_id)
		DO UPDATE SET state=EXCLUDED.state,due_at=EXCLUDED.due_at,updated_at=EXCLUDED.updated_at,version=workflow_instances.version+1
		RETURNING id::text`, request.TenantID, request.ID, string(status), request.Deadline, now).Scan(&workflowID); err != nil {
		return fmt.Errorf("project evidence request workflow: %w", err)
	}
	var taskID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO workflow_tasks(tenant_id,workflow_id,step_key,responsibility,principal_id,title,status,due_at,context,claimed_at,completed_at,created_at,updated_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,'evidence-response','RESPONDENT',NULLIF($3,'')::uuid,$4,$5,$6,$7::jsonb,
		       CASE WHEN $5='IN_PROGRESS' THEN $8::timestamptz ELSE NULL END,
		       CASE WHEN $5='COMPLETED' THEN $8::timestamptz ELSE NULL END,$8::timestamptz,$8::timestamptz)
		ON CONFLICT(workflow_id,step_key) DO UPDATE SET
			principal_id=EXCLUDED.principal_id,title=EXCLUDED.title,status=EXCLUDED.status,due_at=EXCLUDED.due_at,context=EXCLUDED.context,
			claimed_at=CASE WHEN EXCLUDED.status='IN_PROGRESS' AND workflow_tasks.claimed_at IS NULL THEN EXCLUDED.updated_at ELSE workflow_tasks.claimed_at END,
			completed_at=CASE WHEN EXCLUDED.status='COMPLETED' THEN EXCLUDED.updated_at ELSE NULL END,
			version=workflow_tasks.version+1,updated_at=EXCLUDED.updated_at
		RETURNING id::text`, request.TenantID, workflowID, principalID, request.Title, string(status), request.Deadline, string(contextJSON), now).Scan(&taskID); err != nil {
		return fmt.Errorf("project evidence request task: %w", err)
	}
	metadata, _ := json.Marshal(map[string]string{"request_status": request.RequestStatus, "recipient_state": request.RecipientState})
	if _, err := tx.Exec(ctx, `
		INSERT INTO workflow_events(tenant_id,workflow_id,task_id,event_type,safe_metadata,occurred_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3::uuid,'TASK_PROJECTED',$4::jsonb,$5)`,
		request.TenantID, workflowID, taskID, string(metadata), now); err != nil {
		return fmt.Errorf("record evidence request projection event: %w", err)
	}
	return tx.Commit(ctx)
}
