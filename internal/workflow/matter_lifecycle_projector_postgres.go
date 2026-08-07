//go:build postgres

package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5"
)

const matterLifecycleProjectionVersion = "matter-work-v1"

type MatterLifecycleProjector struct {
	Repo       *PostgresRepository
	Continuity *continuity.Service
	Authority  authority.Service
	Now        func() time.Time

	cursorMu     sync.Mutex
	cursorTenant string
	cursorMatter string
}

type projectedMatterWork struct {
	Key            string
	Responsibility string
	PrincipalID    string
	Title          string
	Status         Status
	DueAt          *time.Time
	Context        map[string]string
}

func (p *MatterLifecycleProjector) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if p == nil || p.Repo == nil || p.Continuity == nil || p.Authority == nil || event.AggregateType != "MATTER" {
		return nil
	}
	return p.ReconcileMatter(ctx, event.TenantID, event.AggregateID, p.currentTime())
}

// Maintain provides bounded convergence when authority/delegation changes
// without a Matter event and backfills work after worker restart. It pages the
// canonical Matter population instead of treating Workflow rows as truth.
func (p *MatterLifecycleProjector) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if p == nil || p.Repo == nil || p.Continuity == nil || p.Authority == nil {
		return 0, nil
	}
	if limit < 1 {
		limit = 50
	}
	if now.IsZero() {
		now = p.currentTime()
	} else {
		now = now.UTC()
	}

	p.cursorMu.Lock()
	cursorTenant, cursorMatter := p.cursorTenant, p.cursorMatter
	p.cursorMu.Unlock()

	rows, err := p.Repo.pool.Query(ctx, `
		SELECT t.slug,m.id::text
		FROM matters m
		JOIN tenants t ON t.id=m.tenant_id
		WHERE ($1='' OR (t.slug,m.id::text) > ($1,$2))
		ORDER BY t.slug,m.id::text
		LIMIT $3`, cursorTenant, cursorMatter, limit)
	if err != nil {
		return 0, fmt.Errorf("list matters for work reconciliation: %w", err)
	}
	defer rows.Close()

	type target struct{ tenant, matter string }
	targets := make([]target, 0, limit)
	for rows.Next() {
		var value target
		if err := rows.Scan(&value.tenant, &value.matter); err != nil {
			return len(targets), err
		}
		targets = append(targets, value)
	}
	if err := rows.Err(); err != nil {
		return len(targets), err
	}
	if len(targets) == 0 {
		p.cursorMu.Lock()
		p.cursorTenant, p.cursorMatter = "", ""
		p.cursorMu.Unlock()
		return 0, nil
	}

	var reconcileErrors []error
	for _, target := range targets {
		if ctx.Err() != nil {
			return len(targets), ctx.Err()
		}
		if err := p.ReconcileMatter(ctx, target.tenant, target.matter, now); err != nil {
			reconcileErrors = append(reconcileErrors, fmt.Errorf("reconcile matter %s/%s: %w", target.tenant, target.matter, err))
		}
	}
	last := targets[len(targets)-1]
	p.cursorMu.Lock()
	p.cursorTenant, p.cursorMatter = last.tenant, last.matter
	p.cursorMu.Unlock()
	return len(targets), errors.Join(reconcileErrors...)
}

func (p *MatterLifecycleProjector) ReconcileMatter(ctx context.Context, tenant, matterID string, at time.Time) error {
	tenant, matterID = strings.TrimSpace(tenant), strings.TrimSpace(matterID)
	if tenant == "" || matterID == "" {
		return nil
	}
	if at.IsZero() {
		at = p.currentTime()
	} else {
		at = at.UTC()
	}

	aggregate, err := p.Continuity.GetMatter(ctx, tenant, matterID)
	if err != nil {
		if errors.Is(err, continuity.ErrNotFound) {
			return p.completeMatterWorkflow(ctx, tenant, matterID, at)
		}
		return err
	}
	requirements, ambiguities := continuity.CompileMatterWork(aggregate, at)
	if len(requirements) == 0 && len(ambiguities) == 0 {
		return p.completeMatterWorkflow(ctx, tenant, matterID, at)
	}

	legalEntity, legalEntityState, err := p.matterLegalEntity(ctx, tenant, matterID, at)
	if err != nil {
		return err
	}

	projected := make([]projectedMatterWork, 0, len(requirements)+len(ambiguities))
	for _, requirement := range requirements {
		item, err := p.resolveRequirement(ctx, aggregate, requirement, legalEntity, legalEntityState, at)
		if err != nil {
			return err
		}
		projected = append(projected, item)
	}
	for _, ambiguity := range ambiguities {
		projected = append(projected, projectAmbiguity(aggregate, ambiguity))
	}
	return p.syncMatterWorkflow(ctx, aggregate, projected, at)
}

func (p *MatterLifecycleProjector) resolveRequirement(ctx context.Context, aggregate continuity.MatterAggregate, requirement continuity.WorkRequirement, legalEntity, legalEntityState string, at time.Time) (projectedMatterWork, error) {
	contextValues := baseRequirementContext(aggregate, requirement)
	status := StatusBlocked
	principalID := ""
	contextValues["routing_status"] = legalEntityState

	if legalEntityState == "RESOLVED" {
		resolution, err := p.Authority.Resolve(ctx, authority.ResolveInput{
			TenantID:       aggregate.Matter.TenantID,
			LegalEntityID:  legalEntity,
			ObjectType:     "MATTER",
			ObjectID:       aggregate.Matter.ID,
			Responsibility: authority.Responsibility(requirement.Responsibility),
			DecisionType:   requirement.CommandName,
			Materiality:    requirement.Materiality,
			At:             at,
		})
		switch {
		case err == nil:
			contextValues["authority_rule_id"] = resolution.RuleID
			contextValues["authority_policy_version"] = resolution.PolicyVersion
			contextValues["authority_strategy"] = resolution.Strategy
			principals := uniqueAuthorityPrincipals(resolution)
			contextValues["authority_candidate_count"] = strconv.Itoa(len(principals))
			if required := strings.TrimSpace(requirement.RequiredPrincipalID); required != "" {
				if resolution.AllowsPrincipal(required) {
					principalID, status = required, StatusReady
					contextValues["routing_status"] = "DIRECT"
				} else {
					contextValues["routing_status"] = "REQUIRED_PRINCIPAL_NOT_ELIGIBLE"
				}
			} else if len(principals) == 1 {
				principalID, status = principals[0].ID, StatusReady
				contextValues["routing_status"] = "DIRECT"
			} else {
				contextValues["routing_status"] = "CANDIDATE_SET"
			}
		case errors.Is(err, authority.ErrNoRoute):
			contextValues["routing_status"] = "NO_ROUTE"
		case errors.Is(err, authority.ErrAmbiguousRoute):
			contextValues["routing_status"] = "AMBIGUOUS_ROUTE"
		default:
			return projectedMatterWork{}, fmt.Errorf("resolve current authority for %s: %w", requirement.Key, err)
		}
	}

	return projectedMatterWork{
		Key:            requirement.Key,
		Responsibility: requirement.Responsibility,
		PrincipalID:    principalID,
		Title:          requirement.Title,
		Status:         status,
		DueAt:          requirement.DueAt,
		Context:        contextValues,
	}, nil
}

func baseRequirementContext(aggregate continuity.MatterAggregate, requirement continuity.WorkRequirement) map[string]string {
	values := map[string]string{
		"type":                 "MATTER_WORK",
		"matter_id":            aggregate.Matter.ID,
		"action_target_type":   "MATTER",
		"action_target_id":     aggregate.Matter.ID,
		"primary_action":       requirement.PrimaryAction,
		"why_now":              requirement.WhyNow,
		"material_conclusion":  requirement.WhyNow,
		"intervention_class":   requirement.InterventionClass,
		"work_requirement_key": requirement.Key,
		"command_name":         requirement.CommandName,
		"target_status":        requirement.TargetStatus,
		"subresource_type":     requirement.SubresourceType,
		"subresource_id":       requirement.SubresourceID,
		"decision_type":        requirement.CommandName,
		"materiality":          strconv.Itoa(requirement.Materiality),
		"scope":                firstMatterLabel(aggregate.Matter.Reference, aggregate.Matter.Title),
		"evidence":             "Current governed record",
	}
	if requirement.Verification != nil {
		values["verification_contract_id"] = requirement.Verification.ContractID
		values["verification_expected_outcome"] = requirement.Verification.ExpectedOutcome
		values["verification_evidence_state"] = requirement.Verification.EvidenceState
		values["verification_independent"] = strconv.FormatBool(requirement.Verification.IndependentReview)
		values["evidence"] = requirement.Verification.EvidenceState
	}
	return values
}

func projectAmbiguity(aggregate continuity.MatterAggregate, ambiguity continuity.WorkAmbiguity) projectedMatterWork {
	return projectedMatterWork{
		Key:            ambiguity.Key,
		Responsibility: "UNRESOLVED",
		Title:          ambiguity.Title,
		Status:         StatusBlocked,
		DueAt:          aggregate.Matter.DueAt,
		Context: map[string]string{
			"type":                 "MATTER_WORK",
			"matter_id":            aggregate.Matter.ID,
			"work_requirement_key": ambiguity.Key,
			"routing_status":       "AMBIGUOUS_TRANSITION",
			"why_now":              ambiguity.Reason,
			"material_conclusion":  ambiguity.Reason,
			"subresource_type":     ambiguity.SubresourceType,
			"subresource_id":       ambiguity.SubresourceID,
			"allowed_targets":      strings.Join(ambiguity.AllowedTargets, ","),
			"scope":                firstMatterLabel(aggregate.Matter.Reference, aggregate.Matter.Title),
		},
	}
}

func uniqueAuthorityPrincipals(resolution authority.Resolution) []authority.Principal {
	values := append([]authority.Principal(nil), resolution.CandidatePrincipals...)
	if len(values) == 0 && strings.TrimSpace(resolution.Principal.ID) != "" {
		values = append(values, resolution.Principal)
	}
	seen := map[string]struct{}{}
	result := make([]authority.Principal, 0, len(values))
	for _, value := range values {
		id := strings.TrimSpace(value.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (p *MatterLifecycleProjector) matterLegalEntity(ctx context.Context, tenant, matterID string, at time.Time) (string, string, error) {
	rows, err := p.Repo.pool.Query(ctx, `
		SELECT DISTINCT le.code
		FROM matter_links ml
		JOIN programs pr ON pr.tenant_id=ml.tenant_id AND pr.id=ml.program_id
		JOIN legal_entities le ON le.tenant_id=pr.tenant_id AND le.id=pr.legal_entity_id
		WHERE ml.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND ml.matter_id=$2::uuid
		  AND le.valid_from<=$3
		  AND (le.valid_until IS NULL OR $3<le.valid_until)
		ORDER BY le.code
		LIMIT 2`, tenant, matterID, at)
	if err != nil {
		return "", "", fmt.Errorf("resolve Matter legal entity: %w", err)
	}
	linked := []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			rows.Close()
			return "", "", err
		}
		linked = append(linked, code)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", "", err
	}
	rows.Close()
	if len(linked) == 1 {
		return linked[0], "RESOLVED", nil
	}
	if len(linked) > 1 {
		return "", "MULTIPLE_LEGAL_ENTITIES", nil
	}

	rows, err = p.Repo.pool.Query(ctx, `
		SELECT code
		FROM legal_entities
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND valid_from<=$2
		  AND (valid_until IS NULL OR $2<valid_until)
		ORDER BY code
		LIMIT 2`, tenant, at)
	if err != nil {
		return "", "", fmt.Errorf("resolve tenant legal entity fallback: %w", err)
	}
	defer rows.Close()
	fallback := []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return "", "", err
		}
		fallback = append(fallback, code)
	}
	if err := rows.Err(); err != nil {
		return "", "", err
	}
	if len(fallback) == 1 {
		return fallback[0], "RESOLVED", nil
	}
	if len(fallback) > 1 {
		return "", "LEGAL_ENTITY_REQUIRED", nil
	}
	return "", "NO_LEGAL_ENTITY", nil
}

func (p *MatterLifecycleProjector) syncMatterWorkflow(ctx context.Context, aggregate continuity.MatterAggregate, projected []projectedMatterWork, at time.Time) error {
	if len(projected) == 0 {
		return p.completeMatterWorkflow(ctx, aggregate.Matter.TenantID, aggregate.Matter.ID, at)
	}

	tx, err := p.Repo.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var earliest *time.Time
	for _, item := range projected {
		if item.DueAt != nil && (earliest == nil || item.DueAt.Before(*earliest)) {
			value := item.DueAt.UTC()
			earliest = &value
		}
	}

	var workflowID string
	err = tx.QueryRow(ctx, `
		INSERT INTO workflow_instances(tenant_id,kind,subject_type,subject_id,state,policy_version,context_version,due_at,created_at,updated_at,version)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,'MATTER',$3::uuid,'ACTIVE',$4,1,$5,$6,$6,1)
		ON CONFLICT(tenant_id,kind,subject_type,subject_id) DO UPDATE SET
			state='ACTIVE',policy_version=EXCLUDED.policy_version,due_at=EXCLUDED.due_at,
			updated_at=CASE WHEN workflow_instances.state IS DISTINCT FROM 'ACTIVE' OR workflow_instances.policy_version IS DISTINCT FROM EXCLUDED.policy_version OR workflow_instances.due_at IS DISTINCT FROM EXCLUDED.due_at THEN EXCLUDED.updated_at ELSE workflow_instances.updated_at END,
			version=CASE WHEN workflow_instances.state IS DISTINCT FROM 'ACTIVE' OR workflow_instances.policy_version IS DISTINCT FROM EXCLUDED.policy_version OR workflow_instances.due_at IS DISTINCT FROM EXCLUDED.due_at THEN workflow_instances.version+1 ELSE workflow_instances.version END
		RETURNING id::text`, aggregate.Matter.TenantID, MatterLifecycleWorkflowKind, aggregate.Matter.ID, matterLifecycleProjectionVersion, earliest, at).Scan(&workflowID)
	if err != nil {
		return fmt.Errorf("upsert Matter lifecycle workflow: %w", err)
	}

	desiredKeys := make([]string, 0, len(projected))
	changed := int64(0)
	for _, item := range projected {
		desiredKeys = append(desiredKeys, item.Key)
		contextJSON, err := json.Marshal(item.Context)
		if err != nil {
			return err
		}
		command, err := tx.Exec(ctx, `
			INSERT INTO workflow_tasks(tenant_id,workflow_id,step_key,responsibility,principal_id,title,status,due_at,context,created_at,updated_at,version)
			VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3,$4,NULLIF($5,'')::uuid,$6,$7,$8,$9::jsonb,$10,$10,1)
			ON CONFLICT(workflow_id,step_key) DO UPDATE SET
				responsibility=EXCLUDED.responsibility,principal_id=EXCLUDED.principal_id,title=EXCLUDED.title,status=EXCLUDED.status,due_at=EXCLUDED.due_at,context=EXCLUDED.context,
				completed_at=NULL,updated_at=EXCLUDED.updated_at,version=workflow_tasks.version+1
			WHERE workflow_tasks.responsibility IS DISTINCT FROM EXCLUDED.responsibility
			   OR workflow_tasks.principal_id IS DISTINCT FROM EXCLUDED.principal_id
			   OR workflow_tasks.title IS DISTINCT FROM EXCLUDED.title
			   OR workflow_tasks.status IS DISTINCT FROM EXCLUDED.status
			   OR workflow_tasks.due_at IS DISTINCT FROM EXCLUDED.due_at
			   OR workflow_tasks.context IS DISTINCT FROM EXCLUDED.context`,
			aggregate.Matter.TenantID, workflowID, item.Key, item.Responsibility, item.PrincipalID, item.Title, item.Status, item.DueAt, string(contextJSON), at)
		if err != nil {
			return fmt.Errorf("upsert Matter lifecycle task %s: %w", item.Key, err)
		}
		changed += command.RowsAffected()
	}

	command, err := tx.Exec(ctx, `
		UPDATE workflow_tasks
		SET status='CANCELLED',principal_id=NULL,completed_at=$3,updated_at=$3,version=version+1
		WHERE workflow_id=$1::uuid
		  AND status NOT IN ('COMPLETED','CANCELLED')
		  AND NOT (step_key = ANY($2::text[]))`, workflowID, desiredKeys, at)
	if err != nil {
		return fmt.Errorf("cancel obsolete Matter lifecycle tasks: %w", err)
	}
	changed += command.RowsAffected()

	if changed > 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO workflow_events(tenant_id,workflow_id,event_type,safe_metadata,occurred_at)
			VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,'WORK_REQUIREMENTS_RECONCILED',jsonb_build_object('changed_tasks',$3::bigint),$4)`,
			aggregate.Matter.TenantID, workflowID, changed, at)
		if err != nil {
			return fmt.Errorf("record Matter lifecycle reconciliation: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (p *MatterLifecycleProjector) completeMatterWorkflow(ctx context.Context, tenant, matterID string, at time.Time) error {
	tx, err := p.Repo.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var workflowID string
	err = tx.QueryRow(ctx, `
		UPDATE workflow_instances
		SET state='COMPLETED',due_at=NULL,updated_at=$4,version=version+1
		WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND kind=$2 AND subject_type='MATTER' AND subject_id=$3::uuid
		  AND (state<>'COMPLETED' OR due_at IS NOT NULL)
		RETURNING id::text`, tenant, MatterLifecycleWorkflowKind, matterID, at).Scan(&workflowID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return tx.Commit(ctx)
		}
		return fmt.Errorf("complete Matter lifecycle workflow: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE workflow_tasks
		SET status='CANCELLED',principal_id=NULL,completed_at=$2,updated_at=$2,version=version+1
		WHERE workflow_id=$1::uuid AND status NOT IN ('COMPLETED','CANCELLED')`, workflowID, at); err != nil {
		return fmt.Errorf("cancel completed Matter lifecycle tasks: %w", err)
	}
	return tx.Commit(ctx)
}

func (p *MatterLifecycleProjector) currentTime() time.Time {
	if p != nil && p.Now != nil {
		return p.Now().UTC()
	}
	return time.Now().UTC()
}

func firstMatterLabel(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return "Issue"
}
