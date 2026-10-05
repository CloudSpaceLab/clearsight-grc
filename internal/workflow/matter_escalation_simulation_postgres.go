//go:build postgres

package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/jackc/pgx/v5"
)

type escalationSimulationTask struct {
	escalationTask
	MatterID       string
	Title          string
	Responsibility string
}

func (c *MatterEscalationCoordinator) SimulateEscalation(ctx context.Context, input EscalationSimulationInput) (EscalationSimulation, error) {
	if c == nil || c.Repo == nil || c.Authority == nil || c.Continuity == nil {
		return EscalationSimulation{}, ErrEscalationSimulationUnavailable
	}
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.LegalEntityID = strings.TrimSpace(input.LegalEntityID)
	input.PolicyID = strings.TrimSpace(input.PolicyID)
	input.SequenceID = strings.TrimSpace(input.SequenceID)
	if input.TenantID == "" || input.LegalEntityID == "" || input.PolicyID == "" || input.SequenceID == "" {
		return EscalationSimulation{}, fmt.Errorf("tenant, legal entity, policy and sequence are required")
	}
	if input.Limit < 1 {
		input.Limit = 25
	}
	if input.Limit > 50 {
		input.Limit = 50
	}
	at := input.At.UTC()
	if at.IsZero() {
		at = c.currentTime()
	}

	var policyCode string
	var activeVersion, sequenceVersion int
	var definition []byte
	err := c.Repo.pool.QueryRow(ctx, `
		SELECT rp.code,rp.current_version,rpv.version,rpv.definition
		FROM routing_policies rp
		JOIN routing_policy_versions rpv
		  ON rpv.policy_id=rp.id
		 AND rpv.version=CASE WHEN $4::int>0 THEN $4::int ELSE rp.current_version END
		WHERE rp.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND rp.legal_entity_id=$2::uuid
		  AND rp.id=$3::uuid
		  AND rp.status='ACTIVE'`,
		input.TenantID, input.LegalEntityID, input.PolicyID, input.RevisionVersion).
		Scan(&policyCode, &activeVersion, &sequenceVersion, &definition)
	if errors.Is(err, pgx.ErrNoRows) {
		return EscalationSimulation{}, governance.ErrNotFound
	}
	if err != nil {
		return EscalationSimulation{}, fmt.Errorf("load escalation policy for simulation: %w", err)
	}
	sequences, err := governance.ParseEscalationSequences(definition)
	if err != nil {
		return EscalationSimulation{}, err
	}
	var sequence governance.EscalationSequence
	found := false
	for _, candidate := range sequences {
		if candidate.ID == input.SequenceID && candidate.Trigger == "OVERDUE" {
			sequence, found = candidate, true
			break
		}
	}
	if !found {
		return EscalationSimulation{}, governance.ErrNotFound
	}

	tasks, truncated, err := c.listEscalationSimulationTasks(ctx, input, policyCode, activeVersion)
	if err != nil {
		return EscalationSimulation{}, err
	}
	result := EscalationSimulation{
		PolicyID: input.PolicyID, PolicyCode: policyCode, ActiveVersion: activeVersion,
		SequenceVersion: sequenceVersion, SequenceID: sequence.ID, Trigger: sequence.Trigger,
		Checked: len(tasks), Truncated: truncated,
		Scenarios: make([]EscalationSimulationScenario, 0, len(tasks)),
	}
	for _, task := range tasks {
		scenario, err := c.simulateEscalationTask(ctx, task, sequence, at)
		if err != nil {
			return EscalationSimulation{}, fmt.Errorf("simulate escalation task %s: %w", task.ID, err)
		}
		result.Scenarios = append(result.Scenarios, scenario)
	}
	return result, nil
}

func (c *MatterEscalationCoordinator) listEscalationSimulationTasks(ctx context.Context, input EscalationSimulationInput, policyCode string, activeVersion int) ([]escalationSimulationTask, bool, error) {
	rows, err := c.Repo.pool.Query(ctx, `
		SELECT t.slug,wt.id::text,wt.workflow_id::text,COALESCE(wt.principal_id::text,''),wt.status,wt.due_at,
		       wt.context,wt.responsibility,m.id::text,wt.title
		FROM workflow_tasks wt
		JOIN workflow_instances wi ON wi.id=wt.workflow_id AND wi.tenant_id=wt.tenant_id
		JOIN matters m ON m.id=wi.subject_id AND m.tenant_id=wi.tenant_id
		JOIN tenants t ON t.id=wt.tenant_id
		WHERE wi.kind=$1 AND wi.state='ACTIVE'
		  AND m.legal_entity_id=$2::uuid
		  AND wt.status IN ('READY','IN_PROGRESS','BLOCKED','ESCALATED')
		  AND wt.due_at IS NOT NULL
		  AND COALESCE(wt.context->>'authority_policy_version','')=$3
		ORDER BY wt.due_at,wt.id
		LIMIT $4`,
		MatterLifecycleWorkflowKind, input.LegalEntityID, policyCode+":v"+strconv.Itoa(activeVersion), input.Limit+1)
	if err != nil {
		return nil, false, fmt.Errorf("list active work for escalation simulation: %w", err)
	}
	defer rows.Close()
	values := make([]escalationSimulationTask, 0, input.Limit+1)
	for rows.Next() {
		var value escalationSimulationTask
		var rawContext []byte
		if err := rows.Scan(
			&value.TenantID, &value.ID, &value.WorkflowID, &value.Principal, &value.Status, &value.DueAt,
			&rawContext, &value.Responsibility, &value.MatterID, &value.Title,
		); err != nil {
			return nil, false, err
		}
		if err := json.Unmarshal(rawContext, &value.Context); err != nil {
			return nil, false, fmt.Errorf("decode workflow context: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	truncated := len(values) > input.Limit
	if truncated {
		values = values[:input.Limit]
	}
	return values, truncated, nil
}

func (c *MatterEscalationCoordinator) simulateEscalationTask(ctx context.Context, task escalationSimulationTask, sequence governance.EscalationSequence, at time.Time) (EscalationSimulationScenario, error) {
	if task.DueAt == nil {
		return EscalationSimulationScenario{}, fmt.Errorf("task has no due date")
	}
	materiality, err := parseEscalationMateriality(task.Context["materiality"])
	if err != nil {
		return EscalationSimulationScenario{}, err
	}
	decisionType := strings.TrimSpace(task.Context["decision_type"])
	legalEntity, state, err := (&MatterLifecycleProjector{Repo: c.Repo}).matterLegalEntity(ctx, task.TenantID, task.MatterID, at)
	if err != nil {
		return EscalationSimulationScenario{}, err
	}
	if state != "RESOLVED" {
		return EscalationSimulationScenario{}, fmt.Errorf("matter legal entity is %s", state)
	}
	baseDepartment, baseDepartmentState, err := c.principalDepartmentPath(ctx, task.TenantID, legalEntity, task.Principal, at)
	if err != nil {
		return EscalationSimulationScenario{}, err
	}
	matter, err := c.Continuity.GetMatter(continuity.WithTrustedSystemScope(ctx), task.TenantID, task.MatterID)
	if err != nil {
		return EscalationSimulationScenario{}, err
	}
	currentName, err := c.principalDisplayName(ctx, task.TenantID, task.Principal)
	if err != nil {
		return EscalationSimulationScenario{}, err
	}

	scenario := EscalationSimulationScenario{
		TaskID: task.ID, MatterID: task.MatterID, Title: task.Title,
		CurrentResponsibility: task.Responsibility, CurrentPrincipalID: task.Principal, CurrentPrincipalName: currentName,
		DueAt: task.DueAt.UTC(), RecoveryAction: sequence.RecoveryAction,
		Steps: make([]EscalationSimulationStep, 0, len(sequence.Steps)),
	}
	if index, parseErr := strconv.Atoi(strings.TrimSpace(task.Context["escalation_step_index"])); parseErr == nil {
		scenario.CurrentLevel = index + 1
	}
	if index, parseErr := strconv.Atoi(strings.TrimSpace(task.Context["escalation_next_step_index"])); parseErr == nil {
		scenario.NextLevel = index + 1
	}
	if raw := strings.TrimSpace(task.Context["escalation_next_due_at"]); raw != "" {
		if next, parseErr := time.Parse(time.RFC3339Nano, raw); parseErr == nil {
			next = next.UTC()
			scenario.NextDueAt = &next
		}
	}

	for index, step := range sequence.Steps {
		result := EscalationSimulationStep{
			Index: index, DueAt: task.DueAt.UTC().Add(step.After), Responsibility: step.Responsibility,
			Status: "RESOLVED", Candidates: []EscalationSimulationCandidate{},
		}
		if len(step.SourceRoles) > 0 {
			source, err := c.filterEscalationTargetPrincipals(ctx, task.TenantID, legalEntity, []authority.Principal{{ID: task.Principal}}, step.SourceRoles, nil, nil, at)
			if err != nil {
				return EscalationSimulationScenario{}, err
			}
			if len(source) != 1 {
				result.Status = "SOURCE_ROLE_NOT_ALLOWED"
				scenario.Steps = append(scenario.Steps, result)
				continue
			}
		}

		targetDepartment, departmentState := escalationDepartmentScope(baseDepartment, baseDepartmentState, step.DepartmentLevelsUp)
		if step.DepartmentLevelsUp != nil && departmentState != "RESOLVED" {
			result.Status = departmentState
			scenario.Steps = append(scenario.Steps, result)
			continue
		}
		resolution, err := c.Authority.Resolve(ctx, authority.ResolveInput{
			TenantID: task.TenantID, LegalEntityID: legalEntity, ObjectType: "MATTER", ObjectID: task.MatterID,
			Responsibility: authority.Responsibility(step.Responsibility), DecisionType: decisionType,
			Materiality: materiality, At: at,
		})
		if err != nil {
			switch {
			case errors.Is(err, authority.ErrNoRoute):
				result.Status = "NO_ROUTE"
			case errors.Is(err, authority.ErrAmbiguousRoute):
				result.Status = "AMBIGUOUS_ROUTE"
			default:
				return EscalationSimulationScenario{}, fmt.Errorf("resolve current authority: %w", err)
			}
			scenario.Steps = append(scenario.Steps, result)
			continue
		}
		principals := visibleAuthorityPrincipals(matter.Matter, resolution)
		if step.DepartmentLevelsUp != nil {
			principals, err = c.filterDepartmentPrincipals(ctx, task.TenantID, legalEntity, targetDepartment, principals, at)
			if err != nil {
				return EscalationSimulationScenario{}, err
			}
		}
		beforeConstraint := len(principals)
		if len(step.TargetRoles) > 0 || len(step.TargetGroupIDs) > 0 || len(step.TargetPositionIDs) > 0 {
			principals, err = c.filterEscalationTargetPrincipals(ctx, task.TenantID, legalEntity, principals, step.TargetRoles, step.TargetGroupIDs, step.TargetPositionIDs, at)
			if err != nil {
				return EscalationSimulationScenario{}, err
			}
			if beforeConstraint > 0 && len(principals) == 0 {
				result.Status = "TARGET_CONSTRAINT_NO_MATCH"
			}
		}
		if result.Status == "RESOLVED" {
			switch len(principals) {
			case 0:
				result.Status = "NO_VISIBLE_CANDIDATE"
			case 1:
				result.Status = "RESOLVED"
			default:
				result.Status = "CANDIDATE_SET"
			}
		}
		result.Candidates, err = c.escalationCandidateDisplay(ctx, task.TenantID, principals)
		if err != nil {
			return EscalationSimulationScenario{}, err
		}
		scenario.Steps = append(scenario.Steps, result)
	}
	return scenario, nil
}

func (c *MatterEscalationCoordinator) principalDisplayName(ctx context.Context, tenant, principalID string) (string, error) {
	if strings.TrimSpace(principalID) == "" {
		return "", nil
	}
	var displayName string
	err := c.Repo.pool.QueryRow(ctx, `
		SELECT display_name
		FROM principals
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND id=$2::uuid AND status='ACTIVE'`, tenant, principalID).Scan(&displayName)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return displayName, err
}

func (c *MatterEscalationCoordinator) escalationCandidateDisplay(ctx context.Context, tenant string, principals []authority.Principal) ([]EscalationSimulationCandidate, error) {
	if len(principals) == 0 {
		return []EscalationSimulationCandidate{}, nil
	}
	ids := make([]string, 0, len(principals))
	for _, principal := range principals {
		if strings.TrimSpace(principal.ID) != "" {
			ids = append(ids, principal.ID)
		}
	}
	raw, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	rows, err := c.Repo.pool.Query(ctx, `
		WITH requested(id) AS (SELECT DISTINCT value::uuid FROM jsonb_array_elements_text($2::jsonb))
		SELECT requested.id::text,COALESCE(p.display_name,'')
		FROM requested
		LEFT JOIN principals p
		  ON p.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		 AND p.id=requested.id
		ORDER BY COALESCE(p.display_name,''),requested.id`, tenant, string(raw))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]EscalationSimulationCandidate, 0, len(ids))
	for rows.Next() {
		var value EscalationSimulationCandidate
		if err := rows.Scan(&value.PrincipalID, &value.DisplayName); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

var _ EscalationSimulator = (*MatterEscalationCoordinator)(nil)
