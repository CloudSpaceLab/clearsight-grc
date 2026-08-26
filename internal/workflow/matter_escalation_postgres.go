//go:build postgres

package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5"
)

const (
	MatterEscalationWorkClass = "matter-escalation"
	matterEscalationTimerType = "MATTER_ESCALATION"
	matterEscalationConsumer  = "matter-escalation-v1"
)

var errNoEscalationSequence = errors.New("escalation sequence not configured")

type MatterEscalationRuntime interface {
	ScheduleTimer(context.Context, workflowruntime.Timer) (workflowruntime.Timer, error)
	CancelPendingTaskTimers(context.Context, string, string, string) (int, error)
	InboxProcessed(context.Context, string, string, string) (bool, error)
	RecordInbox(context.Context, string, string, string, time.Time) (bool, error)
}

type MatterEscalationCoordinator struct {
	Repo       *PostgresRepository
	Runtime    MatterEscalationRuntime
	Authority  authority.Service
	Continuity *continuity.Service
	Now        func() time.Time
}

type escalationTimerPayload struct {
	Kind                string   `json:"kind"`
	TaskID              string   `json:"task_id"`
	WorkflowID          string   `json:"workflow_id"`
	PolicyVersion       string   `json:"policy_version"`
	SequenceID          string   `json:"sequence_id"`
	Trigger             string   `json:"trigger"`
	StepIndex           int      `json:"step_index"`
	BaselineDueAt       string   `json:"baseline_due_at"`
	BaseDepartmentPath  []string `json:"base_department_path,omitempty"`
	BaseDepartmentState string   `json:"base_department_state,omitempty"`
}

type escalationTask struct {
	TenantID   string
	ID         string
	WorkflowID string
	Principal  string
	Status     Status
	DueAt      *time.Time
	Context    map[string]string
}

func (c *MatterEscalationCoordinator) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if c == nil || c.Repo == nil || c.Runtime == nil || c.Authority == nil || c.Continuity == nil {
		return 0, nil
	}
	if limit < 1 {
		limit = 100
	}
	if now.IsZero() {
		now = c.currentTime()
	} else {
		now = now.UTC()
	}

	cancelled, err := c.cancelTerminalTimers(ctx, limit)
	if err != nil {
		return cancelled, err
	}

	rows, err := c.Repo.pool.Query(ctx, `
		SELECT t.slug,wt.id::text,wt.workflow_id::text,COALESCE(wt.principal_id::text,''),wt.status,wt.due_at,wt.context
		FROM workflow_tasks wt
		JOIN workflow_instances wi ON wi.id=wt.workflow_id AND wi.tenant_id=wt.tenant_id
		JOIN tenants t ON t.id=wt.tenant_id
		WHERE wi.kind=$1 AND wi.state='ACTIVE'
		  AND wt.status IN ('READY','IN_PROGRESS','BLOCKED')
		  AND wt.due_at IS NOT NULL
		  AND COALESCE(wt.context->>'authority_policy_version','')<>''
		ORDER BY wt.due_at,wt.id
		LIMIT $2`, MatterLifecycleWorkflowKind, limit)
	if err != nil {
		return cancelled, fmt.Errorf("list Matter work for escalation scheduling: %w", err)
	}
	defer rows.Close()

	processed := cancelled
	for rows.Next() {
		var task escalationTask
		var rawContext []byte
		if err := rows.Scan(&task.TenantID, &task.ID, &task.WorkflowID, &task.Principal, &task.Status, &task.DueAt, &rawContext); err != nil {
			return processed, err
		}
		if err := json.Unmarshal(rawContext, &task.Context); err != nil {
			return processed, fmt.Errorf("decode escalation task context %s: %w", task.ID, err)
		}
		if task.DueAt == nil {
			continue
		}
		policyVersion := strings.TrimSpace(task.Context["authority_policy_version"])
		sequence, err := c.sequenceForTrigger(ctx, task.TenantID, policyVersion, "OVERDUE")
		if errors.Is(err, errNoEscalationSequence) {
			continue
		}
		if err != nil {
			return processed, fmt.Errorf("resolve escalation sequence for task %s: %w", task.ID, err)
		}
		if strings.TrimSpace(task.Context["matter_id"]) == "" {
			return processed, fmt.Errorf("task %s is missing matter_id escalation context", task.ID)
		}
		payload := escalationTimerPayload{
			Kind: "MATTER_ESCALATION", TaskID: task.ID, WorkflowID: task.WorkflowID,
			PolicyVersion: policyVersion, SequenceID: sequence.ID, Trigger: sequence.Trigger,
			StepIndex: 0, BaselineDueAt: task.DueAt.UTC().Format(time.RFC3339Nano),
		}
		if err := c.scheduleStep(ctx, task.TenantID, task.WorkflowID, task.ID, payload, sequence.Steps[0]); err != nil {
			return processed, fmt.Errorf("schedule escalation for task %s: %w", task.ID, err)
		}
		processed++
	}
	if err := rows.Err(); err != nil {
		return processed, err
	}
	return processed, nil
}

func (c *MatterEscalationCoordinator) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if event.AggregateType != "WORKFLOW" || event.EventType != "WorkflowTimerFired" {
		return nil
	}
	if c == nil || c.Repo == nil || c.Runtime == nil || c.Authority == nil || c.Continuity == nil {
		return fmt.Errorf("Matter escalation coordinator is not configured")
	}
	var payload escalationTimerPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("decode workflow timer payload: %w", err)
	}
	if payload.Kind != "MATTER_ESCALATION" {
		return nil
	}
	if err := validateEscalationTimerPayload(payload); err != nil {
		return err
	}
	processed, err := c.Runtime.InboxProcessed(ctx, event.TenantID, matterEscalationConsumer, event.ID)
	if err != nil {
		return fmt.Errorf("check escalation inbox: %w", err)
	}
	if processed {
		return nil
	}
	if err := c.processEscalation(ctx, event.TenantID, payload, c.currentTime()); err != nil {
		return err
	}
	if _, err := c.Runtime.RecordInbox(ctx, event.TenantID, matterEscalationConsumer, event.ID, c.currentTime()); err != nil {
		return fmt.Errorf("record escalation inbox: %w", err)
	}
	return nil
}

func (c *MatterEscalationCoordinator) processEscalation(ctx context.Context, tenant string, payload escalationTimerPayload, now time.Time) error {
	task, workflowState, err := c.loadTask(ctx, tenant, payload.TaskID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if task.WorkflowID != payload.WorkflowID {
		return nil
	}
	if task.Status == StatusCompleted || task.Status == StatusCancelled || workflowState != "ACTIVE" || task.DueAt == nil {
		_, err := c.Runtime.CancelPendingTaskTimers(ctx, tenant, task.ID, matterEscalationTimerType)
		return err
	}
	baseline, err := time.Parse(time.RFC3339Nano, payload.BaselineDueAt)
	if err != nil || !task.DueAt.UTC().Equal(baseline.UTC()) {
		return nil
	}
	if strings.TrimSpace(task.Context["authority_policy_version"]) != payload.PolicyVersion {
		return nil
	}
	if err := hydrateEscalationDepartment(&payload, task.Context); err != nil {
		return err
	}
	if task.Context["escalation_attempt_key"] == escalationAttemptKey(payload) {
		return c.scheduleNext(ctx, tenant, payload)
	}

	matterID := strings.TrimSpace(task.Context["matter_id"])
	if matterID == "" {
		return fmt.Errorf("task %s is missing matter_id escalation context", task.ID)
	}
	materiality, err := parseEscalationMateriality(task.Context["materiality"])
	if err != nil {
		return fmt.Errorf("task %s: %w", task.ID, err)
	}
	decisionType := strings.TrimSpace(task.Context["decision_type"])
	legalEntity, legalEntityState, err := (&MatterLifecycleProjector{Repo: c.Repo}).matterLegalEntity(ctx, tenant, matterID, now)
	if err != nil {
		return err
	}
	if legalEntityState != "RESOLVED" {
		return fmt.Errorf("task %s legal entity is not resolved for escalation: %s", task.ID, legalEntityState)
	}
	if payload.StepIndex == 0 && payload.BaseDepartmentState == "" {
		baseDepartment, departmentState, err := c.principalDepartmentPath(ctx, tenant, legalEntity, task.Principal, now)
		if err != nil {
			return fmt.Errorf("resolve task %s department: %w", task.ID, err)
		}
		payload.BaseDepartmentPath = baseDepartment
		payload.BaseDepartmentState = departmentState
	}

	sequence, err := c.sequenceByID(ctx, tenant, payload.PolicyVersion, payload.SequenceID, payload.Trigger)
	if err != nil {
		return err
	}
	if payload.StepIndex < 0 || payload.StepIndex >= len(sequence.Steps) {
		return fmt.Errorf("escalation step index %d is out of range", payload.StepIndex)
	}
	matter, err := c.Continuity.GetMatter(continuity.WithTrustedSystemScope(ctx), tenant, matterID)
	if errors.Is(err, continuity.ErrNotFound) {
		_, cancelErr := c.Runtime.CancelPendingTaskTimers(ctx, tenant, task.ID, matterEscalationTimerType)
		return cancelErr
	}
	if err != nil {
		return fmt.Errorf("load Matter for escalation: %w", err)
	}
	if matter.Matter.Status == continuity.MatterClosed || matter.Matter.Status == continuity.MatterCancelled {
		_, cancelErr := c.Runtime.CancelPendingTaskTimers(ctx, tenant, task.ID, matterEscalationTimerType)
		return cancelErr
	}

	step := sequence.Steps[payload.StepIndex]
	if len(step.SourceRoles) > 0 {
		source := []authority.Principal{{ID: task.Principal}}
		source, err = c.filterEscalationTargetPrincipals(ctx, tenant, legalEntity, source, step.SourceRoles, nil, now)
		if err != nil {
			return err
		}
		if len(source) != 1 {
			applied, err := c.recordUnresolved(ctx, tenant, task, payload, step, "SOURCE_ROLE_NOT_ALLOWED", now)
			if err != nil {
				return err
			}
			if !applied {
				return nil
			}
			return c.scheduleNext(ctx, tenant, payload)
		}
	}

	targetDepartment, departmentState := escalationDepartmentScope(payload.BaseDepartmentPath, payload.BaseDepartmentState, step.DepartmentLevelsUp)
	if step.DepartmentLevelsUp != nil && departmentState != "RESOLVED" {
		applied, err := c.recordUnresolved(ctx, tenant, task, payload, step, departmentState, now)
		if err != nil {
			return err
		}
		if !applied {
			return nil
		}
		return c.scheduleNext(ctx, tenant, payload)
	}

	resolution, err := c.Authority.Resolve(ctx, authority.ResolveInput{
		TenantID: tenant, LegalEntityID: legalEntity, ObjectType: "MATTER", ObjectID: matterID,
		Responsibility: authority.Responsibility(step.Responsibility), DecisionType: decisionType,
		Materiality: materiality, At: now,
	})
	if err != nil {
		reason := "NO_ROUTE"
		if errors.Is(err, authority.ErrAmbiguousRoute) {
			reason = "AMBIGUOUS_ROUTE"
		} else if !errors.Is(err, authority.ErrNoRoute) {
			return fmt.Errorf("resolve escalation authority: %w", err)
		}
		applied, err := c.recordUnresolved(ctx, tenant, task, payload, step, reason, now)
		if err != nil {
			return err
		}
		if !applied {
			return nil
		}
		return c.scheduleNext(ctx, tenant, payload)
	}
	principals := visibleAuthorityPrincipals(matter.Matter, resolution)
	if step.DepartmentLevelsUp != nil {
		principals, err = c.filterDepartmentPrincipals(ctx, tenant, legalEntity, targetDepartment, principals, now)
		if err != nil {
			return err
		}
	}
	if len(step.TargetRoles) > 0 || len(step.TargetGroupIDs) > 0 {
		beforeConstraint := len(principals)
		principals, err = c.filterEscalationTargetPrincipals(ctx, tenant, legalEntity, principals, step.TargetRoles, step.TargetGroupIDs, now)
		if err != nil {
			return err
		}
		if beforeConstraint > 0 && len(principals) == 0 {
			applied, err := c.recordUnresolved(ctx, tenant, task, payload, step, "TARGET_CONSTRAINT_NO_MATCH", now)
			if err != nil {
				return err
			}
			if !applied {
				return nil
			}
			return c.scheduleNext(ctx, tenant, payload)
		}
	}
	if len(principals) != 1 {
		reason := "NO_VISIBLE_CANDIDATE"
		if len(principals) > 1 {
			reason = "CANDIDATE_SET"
		}
		applied, err := c.recordUnresolved(ctx, tenant, task, payload, step, reason, now)
		if err != nil {
			return err
		}
		if !applied {
			return nil
		}
		return c.scheduleNext(ctx, tenant, payload)
	}
	applied, err := c.applyEscalation(ctx, tenant, task, payload, step, principals[0], resolution.RuleID, targetDepartment, now)
	if err != nil {
		return err
	}
	if !applied {
		return nil
	}
	return c.scheduleNext(ctx, tenant, payload)
}

func (c *MatterEscalationCoordinator) cancelTerminalTimers(ctx context.Context, limit int) (int, error) {
	rows, err := c.Repo.pool.Query(ctx, `
		SELECT DISTINCT t.slug,wt.id::text
		FROM workflow_timers timer
		JOIN workflow_tasks wt ON wt.id=timer.task_id AND wt.tenant_id=timer.tenant_id
		JOIN workflow_instances wi ON wi.id=wt.workflow_id AND wi.tenant_id=wt.tenant_id
		JOIN tenants t ON t.id=wt.tenant_id
		WHERE timer.timer_type=$1 AND timer.state='READY'
		  AND (wt.status IN ('COMPLETED','CANCELLED') OR wi.state<>'ACTIVE' OR wt.due_at IS NULL)
		ORDER BY t.slug,wt.id::text
		LIMIT $2`, matterEscalationTimerType, limit)
	if err != nil {
		return 0, fmt.Errorf("list stale escalation timers: %w", err)
	}
	defer rows.Close()
	type target struct{ tenant, task string }
	targets := []target{}
	for rows.Next() {
		var value target
		if err := rows.Scan(&value.tenant, &value.task); err != nil {
			return len(targets), err
		}
		targets = append(targets, value)
	}
	if err := rows.Err(); err != nil {
		return len(targets), err
	}
	cancelled := 0
	for _, value := range targets {
		count, err := c.Runtime.CancelPendingTaskTimers(ctx, value.tenant, value.task, matterEscalationTimerType)
		if err != nil {
			return cancelled, err
		}
		cancelled += count
	}
	return cancelled, nil
}

func (c *MatterEscalationCoordinator) sequenceForTrigger(ctx context.Context, tenant, policyVersion, trigger string) (governance.EscalationSequence, error) {
	sequences, err := c.policySequences(ctx, tenant, policyVersion)
	if err != nil {
		return governance.EscalationSequence{}, err
	}
	for _, sequence := range sequences {
		if sequence.Trigger == trigger {
			return sequence, nil
		}
	}
	return governance.EscalationSequence{}, errNoEscalationSequence
}

func (c *MatterEscalationCoordinator) sequenceByID(ctx context.Context, tenant, policyVersion, sequenceID, trigger string) (governance.EscalationSequence, error) {
	sequences, err := c.policySequences(ctx, tenant, policyVersion)
	if err != nil {
		return governance.EscalationSequence{}, err
	}
	for _, sequence := range sequences {
		if sequence.ID == sequenceID && sequence.Trigger == trigger {
			return sequence, nil
		}
	}
	return governance.EscalationSequence{}, fmt.Errorf("escalation sequence %s/%s is not present in policy %s", trigger, sequenceID, policyVersion)
}

func (c *MatterEscalationCoordinator) policySequences(ctx context.Context, tenant, policyVersion string) ([]governance.EscalationSequence, error) {
	var definition []byte
	err := c.Repo.pool.QueryRow(ctx, `
		SELECT rpv.definition
		FROM routing_policies rp
		JOIN routing_policy_versions rpv ON rpv.policy_id=rp.id
		WHERE rp.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND rp.code || ':v' || rpv.version::text=$2
		LIMIT 1`, tenant, policyVersion).Scan(&definition)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("routing policy version %s not found", policyVersion)
		}
		return nil, fmt.Errorf("load routing policy escalation definition: %w", err)
	}
	return governance.ParseEscalationSequences(definition)
}

func (c *MatterEscalationCoordinator) principalDepartmentPath(ctx context.Context, tenant, legalEntity, principalID string, at time.Time) ([]string, string, error) {
	if strings.TrimSpace(principalID) == "" {
		return nil, "NO_PRINCIPAL", nil
	}
	rows, err := c.Repo.pool.Query(ctx, `
		SELECT DISTINCT op.department_path
		FROM org_positions op
		WHERE op.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND op.occupant_principal_id=$2::uuid
		  AND (op.legal_entity_id IS NULL OR op.legal_entity_id IN (
		      SELECT le.id FROM legal_entities le WHERE le.tenant_id=op.tenant_id AND (le.id::text=$3 OR le.code=$3)
		  ))
		  AND cardinality(op.department_path)>0
		  AND op.valid_from<=$4 AND (op.valid_until IS NULL OR $4<op.valid_until)
		ORDER BY op.department_path
		LIMIT 2`, tenant, principalID, legalEntity, at)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	paths := [][]string{}
	for rows.Next() {
		var path []string
		if err := rows.Scan(&path); err != nil {
			return nil, "", err
		}
		normalized, err := identity.NormalizeDepartmentPath(path)
		if err != nil {
			return nil, "", err
		}
		paths = append(paths, normalized)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	switch len(paths) {
	case 0:
		return nil, "NO_DEPARTMENT", nil
	case 1:
		return paths[0], "RESOLVED", nil
	default:
		return nil, "AMBIGUOUS_DEPARTMENT", nil
	}
}

func (c *MatterEscalationCoordinator) filterDepartmentPrincipals(ctx context.Context, tenant, legalEntity string, department []string, candidates []authority.Principal, at time.Time) ([]authority.Principal, error) {
	if len(department) == 0 || len(candidates) == 0 {
		return candidates, nil
	}
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.ID)
	}
	encoded, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	rows, err := c.Repo.pool.Query(ctx, `
		WITH requested(id) AS (SELECT DISTINCT value::uuid FROM jsonb_array_elements_text($3::jsonb))
		SELECT DISTINCT requested.id::text
		FROM requested
		JOIN org_positions op ON op.occupant_principal_id=requested.id
		WHERE op.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND (op.legal_entity_id IS NULL OR op.legal_entity_id IN (
		      SELECT le.id FROM legal_entities le WHERE le.tenant_id=op.tenant_id AND (le.id::text=$2 OR le.code=$2)
		  ))
		  AND op.department_path=$4::text[]
		  AND op.valid_from<=$5 AND (op.valid_until IS NULL OR $5<op.valid_until)`, tenant, legalEntity, string(encoded), department, at)
	if err != nil {
		return nil, fmt.Errorf("apply escalation department boundary: %w", err)
	}
	defer rows.Close()
	allowed := map[string]struct{}{}
	for rows.Next() {
		var principalID string
		if err := rows.Scan(&principalID); err != nil {
			return nil, err
		}
		allowed[principalID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	filtered := make([]authority.Principal, 0, len(candidates))
	for _, candidate := range candidates {
		if _, ok := allowed[candidate.ID]; ok {
			filtered = append(filtered, candidate)
		}
	}
	return filtered, nil
}

func (c *MatterEscalationCoordinator) filterEscalationTargetPrincipals(ctx context.Context, tenant, legalEntity string, candidates []authority.Principal, roleCodes, groupIDs []string, at time.Time) ([]authority.Principal, error) {
	if len(candidates) == 0 || (len(roleCodes) == 0 && len(groupIDs) == 0) {
		return candidates, nil
	}
	candidateIDs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.ID) != "" {
			candidateIDs = append(candidateIDs, candidate.ID)
		}
	}
	if len(candidateIDs) == 0 {
		return nil, nil
	}
	if roleCodes == nil {
		roleCodes = []string{}
	}
	if groupIDs == nil {
		groupIDs = []string{}
	}
	encodedCandidates, err := json.Marshal(candidateIDs)
	if err != nil {
		return nil, err
	}
	encodedRoles, err := json.Marshal(roleCodes)
	if err != nil {
		return nil, err
	}
	encodedGroups, err := json.Marshal(groupIDs)
	if err != nil {
		return nil, err
	}
	rows, err := c.Repo.pool.Query(ctx, `
		WITH requested(id) AS (
			SELECT DISTINCT value::uuid FROM jsonb_array_elements_text($3::jsonb)
		), requested_roles(code) AS (
			SELECT DISTINCT value FROM jsonb_array_elements_text($4::jsonb)
		), requested_groups(id) AS (
			SELECT DISTINCT value::uuid FROM jsonb_array_elements_text($5::jsonb)
		), current_entity(id) AS (
			SELECT le.id
			FROM tenants t
			JOIN legal_entities le ON le.tenant_id=t.id
			WHERE (t.id::text=$1 OR t.slug=$1)
			  AND (le.id::text=$2 OR le.code=$2)
			  AND le.valid_from<=$6 AND (le.valid_until IS NULL OR $6<le.valid_until)
			LIMIT 1
		), role_matches(principal_id) AS (
			SELECT DISTINCT requested.id
			FROM requested
			JOIN org_positions op ON op.occupant_principal_id=requested.id
			JOIN position_role_bindings prb ON prb.tenant_id=op.tenant_id AND prb.position_id=op.id
			JOIN role_templates rt ON rt.tenant_id=op.tenant_id AND rt.id=prb.role_template_id
			JOIN requested_roles rr ON rr.code=rt.code
			WHERE op.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND (op.legal_entity_id IS NULL OR op.legal_entity_id=(SELECT id FROM current_entity))
			  AND op.valid_from<=$6 AND (op.valid_until IS NULL OR $6<op.valid_until)
			  AND prb.valid_from<=$6 AND (prb.valid_until IS NULL OR $6<prb.valid_until)
			  AND rt.valid_from<=$6 AND (rt.valid_until IS NULL OR $6<rt.valid_until)

			UNION

			SELECT DISTINCT requested.id
			FROM requested
			JOIN scim_users su ON su.principal_id=requested.id AND su.active AND su.deleted_at IS NULL
			JOIN scim_sources ss ON ss.tenant_id=su.tenant_id AND ss.id=su.source_id AND ss.status='ACTIVE'
			JOIN directory_group_members dgm ON dgm.tenant_id=su.tenant_id AND dgm.scim_user_id=su.id
			JOIN directory_groups dg ON dg.tenant_id=dgm.tenant_id AND dg.id=dgm.group_id AND dg.source_id=su.source_id AND dg.deleted_at IS NULL
			JOIN directory_group_role_bindings dgrb ON dgrb.tenant_id=dg.tenant_id AND dgrb.group_id=dg.id AND dgrb.legal_entity_id=(SELECT id FROM current_entity)
			JOIN role_templates rt ON rt.tenant_id=dgrb.tenant_id AND rt.id=dgrb.role_template_id
			JOIN requested_roles rr ON rr.code=rt.code
			WHERE su.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND dgrb.valid_from<=$6 AND (dgrb.valid_until IS NULL OR $6<dgrb.valid_until)
			  AND rt.valid_from<=$6 AND (rt.valid_until IS NULL OR $6<rt.valid_until)
		), group_matches(principal_id) AS (
			SELECT DISTINCT requested.id
			FROM requested
			JOIN scim_users su ON su.principal_id=requested.id AND su.active AND su.deleted_at IS NULL
			JOIN scim_sources ss ON ss.tenant_id=su.tenant_id AND ss.id=su.source_id AND ss.status='ACTIVE'
			JOIN directory_group_members dgm ON dgm.tenant_id=su.tenant_id AND dgm.scim_user_id=su.id
			JOIN directory_groups dg ON dg.tenant_id=dgm.tenant_id AND dg.id=dgm.group_id AND dg.source_id=su.source_id AND dg.deleted_at IS NULL
			JOIN requested_groups rg ON rg.id=dg.id
			WHERE su.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		)
		SELECT DISTINCT requested.id::text
		FROM requested
		WHERE EXISTS (SELECT 1 FROM role_matches rm WHERE rm.principal_id=requested.id)
		   OR EXISTS (SELECT 1 FROM group_matches gm WHERE gm.principal_id=requested.id)`,
		tenant, legalEntity, string(encodedCandidates), string(encodedRoles), string(encodedGroups), at)
	if err != nil {
		return nil, fmt.Errorf("apply escalation role/group target boundary: %w", err)
	}
	defer rows.Close()
	allowed := make(map[string]struct{}, len(candidates))
	for rows.Next() {
		var principalID string
		if err := rows.Scan(&principalID); err != nil {
			return nil, err
		}
		allowed[principalID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	filtered := make([]authority.Principal, 0, len(candidates))
	for _, candidate := range candidates {
		if _, ok := allowed[candidate.ID]; ok {
			filtered = append(filtered, candidate)
		}
	}
	return filtered, nil
}

func (c *MatterEscalationCoordinator) loadTask(ctx context.Context, tenant, taskID string) (escalationTask, string, error) {
	var task escalationTask
	var rawContext []byte
	var workflowState string
	err := c.Repo.pool.QueryRow(ctx, `
		SELECT t.slug,wt.id::text,wt.workflow_id::text,COALESCE(wt.principal_id::text,''),wt.status,wt.due_at,wt.context,wi.state
		FROM workflow_tasks wt
		JOIN workflow_instances wi ON wi.id=wt.workflow_id AND wi.tenant_id=wt.tenant_id
		JOIN tenants t ON t.id=wt.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND wt.id=$2::uuid`, tenant, taskID).
		Scan(&task.TenantID, &task.ID, &task.WorkflowID, &task.Principal, &task.Status, &task.DueAt, &rawContext, &workflowState)
	if err != nil {
		return escalationTask{}, "", err
	}
	if err := json.Unmarshal(rawContext, &task.Context); err != nil {
		return escalationTask{}, "", fmt.Errorf("decode current escalation task context: %w", err)
	}
	return task, workflowState, nil
}

func (c *MatterEscalationCoordinator) applyEscalation(ctx context.Context, tenant string, task escalationTask, payload escalationTimerPayload, step governance.EscalationStep, principal authority.Principal, ruleID string, targetDepartment []string, at time.Time) (bool, error) {
	overlay := escalationOverlay(payload, step, "ROUTED", targetDepartment)
	overlay["escalation_active"] = "true"
	overlay["escalation_principal_id"] = principal.ID
	overlay["escalation_rule_id"] = ruleID
	raw, err := json.Marshal(overlay)
	if err != nil {
		return false, err
	}
	tx, err := c.Repo.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE workflow_tasks
		SET responsibility=$4,principal_id=$5::uuid,status='ESCALATED',context=context || $6::jsonb,updated_at=$7,version=version+1
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND id=$2::uuid AND workflow_id=$3::uuid
		  AND status NOT IN ('COMPLETED','CANCELLED')
		  AND due_at=$8
		  AND COALESCE(context->>'authority_policy_version','')=$9
		  AND COALESCE(context->>'escalation_attempt_key','')<>$10`,
		tenant, task.ID, task.WorkflowID, step.Responsibility, principal.ID, string(raw), at,
		payloadBaseline(payload), payload.PolicyVersion, escalationAttemptKey(payload))
	if err != nil {
		return false, fmt.Errorf("apply workflow escalation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, tx.Commit(ctx)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO workflow_events(tenant_id,workflow_id,event_type,safe_metadata,occurred_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,'WORK_ESCALATED',
		       jsonb_build_object('task_id',$3::text,'sequence_id',$4::text,'step_index',$5::int,'responsibility',$6::text,'principal_id',$7::text),$8)`,
		tenant, task.WorkflowID, task.ID, payload.SequenceID, payload.StepIndex, step.Responsibility, principal.ID, at)
	if err != nil {
		return false, fmt.Errorf("record workflow escalation event: %w", err)
	}
	return true, tx.Commit(ctx)
}

func (c *MatterEscalationCoordinator) recordUnresolved(ctx context.Context, tenant string, task escalationTask, payload escalationTimerPayload, step governance.EscalationStep, reason string, at time.Time) (bool, error) {
	overlay := escalationOverlay(payload, step, reason, nil)
	raw, err := json.Marshal(overlay)
	if err != nil {
		return false, err
	}
	tx, err := c.Repo.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE workflow_tasks
		SET context=context || $4::jsonb,updated_at=$5,version=version+1
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND id=$2::uuid AND workflow_id=$3::uuid
		  AND status NOT IN ('COMPLETED','CANCELLED')
		  AND due_at=$6
		  AND COALESCE(context->>'authority_policy_version','')=$7
		  AND COALESCE(context->>'escalation_attempt_key','')<>$8`,
		tenant, task.ID, task.WorkflowID, string(raw), at, payloadBaseline(payload), payload.PolicyVersion, escalationAttemptKey(payload))
	if err != nil {
		return false, fmt.Errorf("record unresolved workflow escalation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, tx.Commit(ctx)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO workflow_events(tenant_id,workflow_id,event_type,safe_metadata,occurred_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,'WORK_ESCALATION_UNRESOLVED',
		       jsonb_build_object('task_id',$3::text,'sequence_id',$4::text,'step_index',$5::int,'responsibility',$6::text,'reason',$7::text),$8)`,
		tenant, task.WorkflowID, task.ID, payload.SequenceID, payload.StepIndex, step.Responsibility, reason, at)
	if err != nil {
		return false, fmt.Errorf("record unresolved escalation event: %w", err)
	}
	return true, tx.Commit(ctx)
}

func (c *MatterEscalationCoordinator) scheduleNext(ctx context.Context, tenant string, payload escalationTimerPayload) error {
	sequence, err := c.sequenceByID(ctx, tenant, payload.PolicyVersion, payload.SequenceID, payload.Trigger)
	if err != nil {
		return err
	}
	next := payload.StepIndex + 1
	if next >= len(sequence.Steps) {
		return nil
	}
	payload.StepIndex = next
	return c.scheduleStep(ctx, tenant, payload.WorkflowID, payload.TaskID, payload, sequence.Steps[next])
}

func (c *MatterEscalationCoordinator) scheduleStep(ctx context.Context, tenant, workflowID, taskID string, payload escalationTimerPayload, step governance.EscalationStep) error {
	baseline := payloadBaseline(payload)
	if baseline.IsZero() {
		return fmt.Errorf("escalation baseline due_at is invalid")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	timerID, err := id.NewUUIDv7()
	if err != nil {
		return err
	}
	_, err = c.Runtime.ScheduleTimer(ctx, workflowruntime.Timer{
		ID: timerID, TenantID: tenant, WorkflowID: workflowID, TaskID: taskID,
		Type: matterEscalationTimerType, DueAt: baseline.Add(step.After),
		DedupeKey: escalationDedupeKey(payload), Payload: raw,
	})
	return err
}

func escalationOverlay(payload escalationTimerPayload, step governance.EscalationStep, status string, targetDepartment []string) map[string]string {
	targetJSON, _ := json.Marshal(targetDepartment)
	baseJSON, _ := json.Marshal(payload.BaseDepartmentPath)
	sourceRolesJSON, _ := json.Marshal(step.SourceRoles)
	targetRolesJSON, _ := json.Marshal(step.TargetRoles)
	targetGroupsJSON, _ := json.Marshal(step.TargetGroupIDs)
	return map[string]string{
		"escalation_trigger":                payload.Trigger,
		"escalation_sequence_id":            payload.SequenceID,
		"escalation_policy_version":         payload.PolicyVersion,
		"escalation_step_index":             strconv.Itoa(payload.StepIndex),
		"escalation_attempt_key":            escalationAttemptKey(payload),
		"escalation_status":                 status,
		"escalation_responsibility":         step.Responsibility,
		"escalation_baseline_due_at":        payload.BaselineDueAt,
		"escalation_base_department_path":   string(baseJSON),
		"escalation_base_department_state":  payload.BaseDepartmentState,
		"escalation_target_department_path": string(targetJSON),
		"escalation_source_roles":           string(sourceRolesJSON),
		"escalation_target_roles":           string(targetRolesJSON),
		"escalation_target_groups":          string(targetGroupsJSON),
	}
}

func hydrateEscalationDepartment(payload *escalationTimerPayload, context map[string]string) error {
	if payload == nil || payload.BaseDepartmentState != "" {
		return nil
	}
	state := strings.TrimSpace(context["escalation_base_department_state"])
	pathRaw := strings.TrimSpace(context["escalation_base_department_path"])
	if state == "" && pathRaw == "" {
		return nil
	}
	payload.BaseDepartmentState = state
	if pathRaw == "" {
		return nil
	}
	var path []string
	if err := json.Unmarshal([]byte(pathRaw), &path); err != nil {
		return fmt.Errorf("decode escalation base department path: %w", err)
	}
	normalized, err := identity.NormalizeDepartmentPath(path)
	if err != nil {
		return fmt.Errorf("normalize escalation base department path: %w", err)
	}
	payload.BaseDepartmentPath = normalized
	return nil
}

func escalationDepartmentScope(base []string, baseState string, levels *int) ([]string, string) {
	if levels == nil {
		return nil, "RESOLVED"
	}
	if baseState != "RESOLVED" || len(base) == 0 {
		return nil, "DEPARTMENT_SCOPE_UNRESOLVED"
	}
	keep := len(base) - *levels
	if keep < 1 {
		return nil, "DEPARTMENT_ANCESTRY_EXHAUSTED"
	}
	return append([]string(nil), base[:keep]...), "RESOLVED"
}

func parseEscalationMateriality(value string) (int, error) {
	materiality, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || materiality < 0 || materiality > 5 {
		return 0, fmt.Errorf("invalid escalation materiality %q", value)
	}
	return materiality, nil
}

func validateEscalationTimerPayload(payload escalationTimerPayload) error {
	if payload.Kind != "MATTER_ESCALATION" || strings.TrimSpace(payload.TaskID) == "" || strings.TrimSpace(payload.WorkflowID) == "" || strings.TrimSpace(payload.PolicyVersion) == "" || strings.TrimSpace(payload.SequenceID) == "" || payload.Trigger != "OVERDUE" || payload.StepIndex < 0 || payloadBaseline(payload).IsZero() {
		return fmt.Errorf("invalid Matter escalation timer payload")
	}
	if len(payload.BaseDepartmentPath) > 0 {
		if _, err := identity.NormalizeDepartmentPath(payload.BaseDepartmentPath); err != nil {
			return fmt.Errorf("invalid Matter escalation department path: %w", err)
		}
	}
	return nil
}

func payloadBaseline(payload escalationTimerPayload) time.Time {
	value, err := time.Parse(time.RFC3339Nano, payload.BaselineDueAt)
	if err != nil {
		return time.Time{}
	}
	return value.UTC()
}

func escalationAttemptKey(payload escalationTimerPayload) string {
	return fmt.Sprintf("%s:%s:%d:%s", payload.PolicyVersion, payload.SequenceID, payload.StepIndex, payload.BaselineDueAt)
}

func escalationDedupeKey(payload escalationTimerPayload) string {
	digest := sha256.Sum256([]byte(payload.TaskID + "\x00" + escalationAttemptKey(payload)))
	return "matter-escalation:" + hex.EncodeToString(digest[:16])
}

func (c *MatterEscalationCoordinator) currentTime() time.Time {
	if c != nil && c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

var _ workflowruntime.Publisher = (*MatterEscalationCoordinator)(nil)
var _ workflowruntime.Maintainer = (*MatterEscalationCoordinator)(nil)
