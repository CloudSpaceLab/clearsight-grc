//go:build postgres

package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

const matterActionProjectionConsumer = "workflow-matter-action-v1"

// MatterActionProjector derives actor-facing workflow work from authoritative
// Matter Action events. It never creates or mutates the Matter Action itself.
type MatterActionProjector struct{ Repo *PostgresRepository }

type matterActionPayload struct {
	ID               string     `json:"id"`
	TenantID         string     `json:"tenant_id"`
	MatterID         string     `json:"matter_id"`
	Title            string     `json:"title"`
	OwnerPrincipalID string     `json:"owner_principal_id"`
	Status           string     `json:"status"`
	DueAt            *time.Time `json:"due_at"`
}

func (p *MatterActionProjector) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if p == nil || p.Repo == nil || event.AggregateType != "MATTER" {
		return nil
	}
	if event.EventType != "ACTION_ADDED" && event.EventType != "ACTION_STATE_CHANGED" {
		return nil
	}
	var action matterActionPayload
	if err := json.Unmarshal(event.Payload, &action); err != nil {
		return fmt.Errorf("decode Matter Action projection event: %w", err)
	}
	if strings.TrimSpace(action.ID) == "" || strings.TrimSpace(action.MatterID) == "" {
		return fmt.Errorf("Matter Action projection requires action and matter identifiers")
	}
	status, err := projectedActionStatus(action.Status)
	if err != nil {
		return err
	}
	actionTargets := continuity.AllowedActionTargets(continuity.ActionStatus(strings.ToUpper(strings.TrimSpace(action.Status))))
	allowedTargets := make([]string, len(actionTargets))
	for index := range actionTargets {
		allowedTargets[index] = string(actionTargets[index])
	}
	targetStatus := ""
	if len(allowedTargets) == 1 {
		targetStatus = allowedTargets[0]
	}

	tx, err := p.Repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin Matter Action projection: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		INSERT INTO inbox_receipts(tenant_id,consumer,event_id,processed_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3,$4)
		ON CONFLICT(tenant_id,consumer,event_id) DO NOTHING`, event.TenantID, matterActionProjectionConsumer, event.ID, event.OccurredAt)
	if err != nil {
		return fmt.Errorf("record Matter Action projection receipt: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return tx.Commit(ctx)
	}

	var workflowID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO workflow_instances(tenant_id,kind,subject_type,subject_id,state,policy_version,context_version,due_at,created_at,updated_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER_ACTION','MATTER_ACTION',$2::uuid,$3,'continuity-matter-action-v1',1,$4,$5,$5)
		ON CONFLICT(tenant_id,kind,subject_type,subject_id) WHERE kind='MATTER_ACTION'
		DO UPDATE SET state=EXCLUDED.state,due_at=EXCLUDED.due_at,updated_at=EXCLUDED.updated_at,version=workflow_instances.version+1
		RETURNING id::text`, event.TenantID, action.ID, string(status), action.DueAt, event.OccurredAt).Scan(&workflowID); err != nil {
		return fmt.Errorf("project Matter Action workflow: %w", err)
	}

	contextJSON, err := json.Marshal(map[string]string{
		"type":               "MATTER_ACTION",
		"matter_id":          action.MatterID,
		"action_id":          action.ID,
		"command_name":       "matter.action.transition",
		"target_status":      targetStatus,
		"allowed_targets":    strings.Join(allowedTargets, ","),
		"subresource_type":   "ACTION",
		"subresource_id":     action.ID,
		"action_target_type": "MATTER",
		"action_target_id":   action.MatterID,
		"primary_action":     "Update action",
		"why_now":            "This accountable issue action requires your attention.",
	})
	if err != nil {
		return fmt.Errorf("encode Matter Action workflow context: %w", err)
	}

	var taskID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO workflow_tasks(tenant_id,workflow_id,step_key,responsibility,principal_id,title,status,due_at,context,claimed_at,completed_at,created_at,updated_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,'matter-action','ACCOUNTABLE_OWNER',NULLIF($3,'')::uuid,$4,$5,$6,$7::jsonb,
		       CASE WHEN $5='IN_PROGRESS' THEN $8::timestamptz ELSE NULL END,
		       CASE WHEN $5='COMPLETED' THEN $8::timestamptz ELSE NULL END,$8::timestamptz,$8::timestamptz)
		ON CONFLICT(workflow_id,step_key) DO UPDATE SET
			principal_id=EXCLUDED.principal_id,
			title=EXCLUDED.title,
			status=EXCLUDED.status,
			due_at=EXCLUDED.due_at,
			context=EXCLUDED.context,
			claimed_at=CASE WHEN EXCLUDED.status='IN_PROGRESS' AND workflow_tasks.claimed_at IS NULL THEN EXCLUDED.updated_at ELSE workflow_tasks.claimed_at END,
			completed_at=CASE WHEN EXCLUDED.status='COMPLETED' THEN EXCLUDED.updated_at ELSE NULL END,
			version=workflow_tasks.version+1,
			updated_at=EXCLUDED.updated_at
		RETURNING id::text`, event.TenantID, workflowID, action.OwnerPrincipalID, action.Title, string(status), action.DueAt, string(contextJSON), event.OccurredAt).Scan(&taskID); err != nil {
		return fmt.Errorf("project Matter Action task: %w", err)
	}

	metadata, err := json.Marshal(map[string]string{"source_event": event.EventType, "action_status": action.Status})
	if err != nil {
		return fmt.Errorf("encode Matter Action projection audit: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO workflow_events(tenant_id,workflow_id,task_id,event_type,safe_metadata,occurred_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3::uuid,'TASK_PROJECTED',$4::jsonb,$5)`, event.TenantID, workflowID, taskID, string(metadata), event.OccurredAt); err != nil {
		return fmt.Errorf("record Matter Action projection audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit Matter Action projection: %w", err)
	}
	return nil
}

func projectedActionStatus(status string) (Status, error) {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "PLANNED":
		return StatusReady, nil
	case "IN_PROGRESS":
		return StatusInProgress, nil
	case "BLOCKED":
		return StatusBlocked, nil
	case "IMPLEMENTED":
		return StatusCompleted, nil
	case "CANCELLED":
		return StatusCancelled, nil
	default:
		return "", fmt.Errorf("unsupported Matter Action status %q", status)
	}
}

var _ workflowruntime.Publisher = (*MatterActionProjector)(nil)
