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
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5"
)

const matterLifecycleProjectionVersion = "matter-work-v1"

type MatterLifecycleProjector struct {
	Repo       *PostgresRepository
	Continuity *continuity.Service
	Authority  authority.Service
	Sequence   governance.LifecycleSequenceResolver
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
	// Delayed delivery must converge against current authority, not historical
	// event-time routing.
	return p.ReconcileMatter(ctx, event.TenantID, event.AggregateID, p.currentTime())
}

// Maintain reconciles Matters that can currently yield lifecycle work, plus
// Matters with an existing lifecycle projection that may need reassignment or
// cleanup. Decision/Response candidates are included so routing-policy changes
// converge even when the Matter itself did not emit a new event.
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
		  AND m.status NOT IN ('CLOSED','CANCELLED')
		  AND (
		    EXISTS (
		      SELECT 1 FROM workflow_instances wi
		      WHERE wi.tenant_id=m.tenant_id AND wi.kind=$4
		        AND wi.subject_type='MATTER' AND wi.subject_id=m.id
		    )
		    OR EXISTS (
		      SELECT 1 FROM response_packages rp
		      WHERE rp.tenant_id=m.tenant_id AND rp.matter_id=m.id
		        AND rp.status <> 'ACKNOWLEDGED'
		    )
		    OR EXISTS (
		      SELECT 1 FROM matter_decisions md
		      WHERE md.tenant_id=m.tenant_id AND md.matter_id=m.id
		    )
		    OR EXISTS (
		      SELECT 1
		      FROM verification_contracts vc
		      LEFT JOIN matter_actions ma
		        ON ma.tenant_id=vc.tenant_id AND ma.id=vc.action_id
		      WHERE vc.tenant_id=m.tenant_id AND vc.matter_id=m.id AND vc.status='ACTIVE'
		        AND NOT EXISTS (
		          SELECT 1 FROM verification_results vr
		          WHERE vr.tenant_id=vc.tenant_id AND vr.contract_id=vc.id
		        )
		        AND (
		          (vc.action_id IS NULL AND vc.created_at + make_interval(mins=>vc.observation_period_minutes) <= $3)
		          OR
		          (vc.action_id IS NOT NULL AND ma.status='IMPLEMENTED' AND ma.implemented_at IS NOT NULL
		           AND GREATEST(vc.created_at,ma.implemented_at) + make_interval(mins=>vc.observation_period_minutes) <= $3)
		        )
		    )
		  )
		ORDER BY t.slug,m.id::text
		LIMIT $5`, cursorTenant, cursorMatter, now, MatterLifecycleWorkflowKind, limit)
	if err != nil {
		return 0, fmt.Errorf("list Matters for lifecycle work reconciliation: %w", err)
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
			reconcileErrors = append(reconcileErrors, fmt.Errorf("reconcile Matter %s/%s: %w", target.tenant, target.matter, err))
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

	aggregate, err := p.Continuity.GetMatter(continuity.WithTrustedSystemScope(ctx), tenant, matterID)
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
	if len(ambiguities) > 0 && p.Sequence != nil && legalEntityState == "RESOLVED" {
		choices := make([]continuity.WorkSequenceChoice, 0, len(ambiguities))
		for _, ambiguity := range ambiguities {
			resolution, resolveErr := p.Sequence.ResolveLifecycleSequence(ctx, governance.LifecycleSequenceInput{
				TenantID: tenant, LegalEntityID: legalEntity, MatterID: matterID, MatterType: string(aggregate.Matter.Type),
				CommandName: ambiguity.CommandName, LifecycleType: ambiguity.SubresourceType, LifecycleSubtype: ambiguity.LifecycleSubtype,
				LifecycleState: ambiguity.LifecycleState, Materiality: aggregate.Matter.Priority, At: at,
			})
			switch {
			case resolveErr == nil:
				choices = append(choices, continuity.WorkSequenceChoice{AmbiguityKey: ambiguity.Key, Responsibility: resolution.Responsibility, RuleID: resolution.RuleID, PolicyVersion: resolution.PolicyVersion})
			case errors.Is(resolveErr, governance.ErrNoLifecycleSequence):
				continue
			case errors.Is(resolveErr, governance.ErrAmbiguousLifecycleSequence):
				return fmt.Errorf("resolve lifecycle sequence for %s: %w", ambiguity.Key, resolveErr)
			default:
				return fmt.Errorf("resolve lifecycle sequence for %s: %w", ambiguity.Key, resolveErr)
			}
		}
		requirements, ambiguities = continuity.ApplyWorkSequenceChoices(aggregate, requirements, ambiguities, choices)
	}
	if len(requirements) == 0 {
		return p.completeMatterWorkflow(ctx, tenant, matterID, at)
	}

	projected := make([]projectedMatterWork, 0, len(requirements))
	for _, requirement := range requirements {
		item, err := p.resolveRequirement(ctx, aggregate, requirement, legalEntity, legalEntityState, at)
		if err != nil {
			return err
		}
		projected = append(projected, item)
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
			principals := visibleAuthorityPrincipals(aggregate.Matter, resolution)
			contextValues["authority_candidate_count"] = strconv.Itoa(len(principals))
			if required := strings.TrimSpace(requirement.RequiredPrincipalID); required != "" {
				switch {
				case !resolution.AllowsPrincipal(required):
					contextValues["routing_status"] = "REQUIRED_PRINCIPAL_NOT_ELIGIBLE"
				case !continuity.MatterVisibleTo(aggregate.Matter, required):
					contextValues["routing_status"] = "REQUIRED_PRINCIPAL_NOT_VISIBLE"
				default:
					principalID, status = required, StatusReady
					contextValues["routing_status"] = "DIRECT"
				}
			} else {
				switch len(principals) {
				case 0:
					contextValues["routing_status"] = "NO_VISIBLE_CANDIDATE"
				case 1:
					principalID, status = principals[0].ID, StatusReady
					contextValues["routing_status"] = "DIRECT"
				default:
					contextValues["routing_status"] = "CANDIDATE_SET"
				}
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
		"allowed_targets":      strings.Join(requirement.AllowedTargets, ","),
		"subresource_type":     requirement.SubresourceType,
		"subresource_id":       requirement.SubresourceID,
		"decision_type":        requirement.CommandName,
		"materiality":          strconv.Itoa(requirement.Materiality),
		"scope":                firstMatterLabel(aggregate.Matter.Reference, aggregate.Matter.Title),
		"evidence":             "Current governed record",
	}
	if requirement.SequenceRuleID != "" {
		values["sequence_rule_id"] = requirement.SequenceRuleID
		values["sequence_policy_version"] = requirement.SequencePolicyVersion
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

func visibleAuthorityPrincipals(matter continuity.Matter, resolution authority.Resolution) []authority.Principal {
	values := uniqueAuthorityPrincipals(resolution)
	visible := values[:0]
	for _, value := range values {
		if continuity.MatterVisibleTo(matter, value.ID) {
			visible = append(visible, value)
		}
	}
	return visible
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
