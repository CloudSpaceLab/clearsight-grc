//go:build postgres

package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/jackc/pgx/v5"
)

const matterLifecycleProjectionConsumer = "workflow-matter-lifecycle-v1"

// MatterLifecycleProjector turns only explicit, unambiguous lifecycle work
// requirements into actor-facing Workflow Tasks. Canonical lifecycle records
// remain authoritative; these rows are rebuildable routing projections.
type MatterLifecycleProjector struct {
	Repo       *PostgresRepository
	Continuity *continuity.Service
	Authority  authority.Service

	cursorMu sync.Mutex
	cursor   string
}

type lifecycleAssignment struct {
	Status        Status
	PrincipalID   string
	RoutingState  string
	RuleID        string
	PolicyVersion string
}

type lifecycleTaskProjection struct {
	WorkflowState  string
	Status         Status
	PrincipalID    string
	Responsibility string
	Title          string
	DueAt          *time.Time
	PolicyVersion  string
	Context        map[string]string
}

type existingLifecycleTask struct {
	WorkflowID     string
	WorkflowState  string
	TaskID         string
	Status         Status
	PrincipalID    string
	Responsibility string
	Title          string
	DueAt          *time.Time
	PolicyVersion  string
	Context        map[string]string
}

func (p *MatterLifecycleProjector) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if p == nil || p.Repo == nil || p.Continuity == nil || p.Authority == nil || event.AggregateType != "MATTER" || !lifecycleProjectionEvent(event.EventType) {
		return nil
	}
	aggregate, err := p.Continuity.GetMatter(ctx, event.TenantID, event.AggregateID)
	if err != nil {
		return fmt.Errorf("load Matter lifecycle projection state: %w", err)
	}
	tx, err := p.Repo.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin Matter lifecycle projection: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		INSERT INTO inbox_receipts(tenant_id,consumer,event_id,processed_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3,$4)
		ON CONFLICT(tenant_id,consumer,event_id) DO NOTHING`, event.TenantID, matterLifecycleProjectionConsumer, event.ID, event.OccurredAt)
	if err != nil {
		return fmt.Errorf("record Matter lifecycle projection receipt: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return tx.Commit(ctx)
	}
	if err := p.reconcileAggregateTx(ctx, tx, aggregate, time.Now().UTC()); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit Matter lifecycle projection: %w", err)
	}
	return nil
}

func lifecycleProjectionEvent(eventType string) bool {
	switch eventType {
	case continuity.EventMatterLinked,
		continuity.EventMatterStateChanged,
		continuity.EventResponsePackageAdded,
		continuity.EventResponsePackageStateChanged:
		return true
	default:
		return false
	}
}

// Maintain revisits current explicit lifecycle work in bounded UUID order so
// routing/delegation changes converge even when no new Matter event occurs.
func (p *MatterLifecycleProjector) Maintain(ctx context.Context, now time.Time, budget int) (int, error) {
	if p == nil || p.Repo == nil || p.Continuity == nil || p.Authority == nil || budget <= 0 {
		return 0, nil
	}
	p.cursorMu.Lock()
	defer p.cursorMu.Unlock()

	rows, err := p.Repo.pool.Query(ctx, `
		SELECT rp.id::text,m.id::text,t.slug
		FROM response_packages rp
		JOIN matters m ON m.id=rp.matter_id AND m.tenant_id=rp.tenant_id
		JOIN tenants t ON t.id=rp.tenant_id
		WHERE rp.status IN ('REJECTED','TRANSMITTED')
		  AND m.status NOT IN ('CLOSED','CANCELLED')
		  AND (NULLIF($1,'')::uuid IS NULL OR rp.id>NULLIF($1,'')::uuid)
		ORDER BY rp.id
		LIMIT $2`, p.cursor, budget)
	if err != nil {
		return 0, fmt.Errorf("list lifecycle work for reconciliation: %w", err)
	}
	defer rows.Close()

	type candidate struct{ responseID, matterID, tenant string }
	candidates := make([]candidate, 0, budget)
	for rows.Next() {
		var value candidate
		if err := rows.Scan(&value.responseID, &value.matterID, &value.tenant); err != nil {
			return 0, err
		}
		candidates = append(candidates, value)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(candidates) == 0 {
		p.cursor = ""
		return 0, nil
	}

	processed := 0
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return processed, err
		}
		aggregate, err := p.Continuity.GetMatter(ctx, candidate.tenant, candidate.matterID)
		if err != nil {
			return processed, err
		}
		tx, err := p.Repo.pool.Begin(ctx)
		if err != nil {
			return processed, err
		}
		if err := p.reconcileAggregateTx(ctx, tx, aggregate, now.UTC()); err != nil {
			_ = tx.Rollback(ctx)
			return processed, err
		}
		if err := tx.Commit(ctx); err != nil {
			return processed, err
		}
		p.cursor = candidate.responseID
		processed++
	}
	return processed, nil
}

func (p *MatterLifecycleProjector) reconcileAggregateTx(ctx context.Context, tx pgx.Tx, aggregate continuity.MatterAggregate, now time.Time) error {
	requirements := make(map[string]WorkRequirement)
	for _, requirement := range CompileMatterLifecycleWork(aggregate) {
		if err := requirement.Validate(); err != nil {
			return err
		}
		requirements[requirement.SubjectID] = requirement
	}
	for _, response := range aggregate.ResponsePackages {
		requirement, required := requirements[response.ID]
		if err := p.reconcileResponseTx(ctx, tx, aggregate, response, requirement, required, now); err != nil {
			return err
		}
	}
	return nil
}

func (p *MatterLifecycleProjector) reconcileResponseTx(ctx context.Context, tx pgx.Tx, aggregate continuity.MatterAggregate, response continuity.ResponsePackage, requirement WorkRequirement, required bool, now time.Time) error {
	existing, err := loadLifecycleTask(ctx, tx, aggregate.Matter.TenantID, response.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if !required {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		desired := lifecycleTaskProjection{
			WorkflowState:  existing.WorkflowState,
			Status:         existing.Status,
			PrincipalID:    existing.PrincipalID,
			Responsibility: existing.Responsibility,
			Title:          existing.Title,
			DueAt:          existing.DueAt,
			PolicyVersion:  existing.PolicyVersion,
			Context:        clone(existing.Context),
		}
		target := strings.TrimSpace(existing.Context["target_status"])
		if target != "" && target == string(response.Status) {
			desired.Status = StatusCompleted
			desired.WorkflowState = "COMPLETED"
		} else {
			desired.Status = StatusCancelled
			desired.WorkflowState = "CANCELLED"
		}
		desired.Context["routing_state"] = desired.WorkflowState
		return persistLifecycleTask(ctx, tx, aggregate.Matter.TenantID, response.ID, existing, desired, now)
	}

	legalEntityID, err := p.resolveMatterLegalEntity(ctx, tx, aggregate.Matter)
	if err != nil {
		return err
	}
	assignment := lifecycleAssignment{Status: StatusBlocked, RoutingState: "LEGAL_ENTITY_UNRESOLVED", PolicyVersion: "unresolved"}
	if legalEntityID != "" {
		assignment, err = p.resolveAssignment(ctx, aggregate.Matter, requirement, legalEntityID, now)
		if err != nil {
			return err
		}
	}
	desired := lifecycleTaskProjection{
		WorkflowState:  "ACTIVE",
		Status:         assignment.Status,
		PrincipalID:    assignment.PrincipalID,
		Responsibility: string(requirement.Responsibility),
		Title:          requirement.Title,
		DueAt:          requirement.DueAt,
		PolicyVersion:  assignment.PolicyVersion,
		Context: map[string]string{
			"type":               MatterResponseWorkflowKind,
			"matter_id":          aggregate.Matter.ID,
			"response_id":        response.ID,
			"target_status":      requirement.TargetStatus,
			"command_name":       requirement.CommandName,
			"decision_type":      requirement.CommandName,
			"primary_action":     requirement.Title,
			"why_now":            requirement.WhyNow,
			"intervention_class": requirement.InterventionClass,
			"materiality":        fmt.Sprintf("%d", requirement.Materiality),
			"routing_state":      assignment.RoutingState,
			"legal_entity_id":    legalEntityID,
			"routing_rule_id":    assignment.RuleID,
			"policy_version":     assignment.PolicyVersion,
		},
	}
	if desired.Status == StatusBlocked {
		desired.WorkflowState = "BLOCKED"
	}
	if errors.Is(err, pgx.ErrNoRows) {
		existing = existingLifecycleTask{}
	}
	return persistLifecycleTask(ctx, tx, aggregate.Matter.TenantID, response.ID, existing, desired, now)
}

func (p *MatterLifecycleProjector) resolveAssignment(ctx context.Context, matter continuity.Matter, requirement WorkRequirement, legalEntityID string, now time.Time) (lifecycleAssignment, error) {
	resolution, err := p.Authority.Resolve(ctx, authority.ResolveInput{
		TenantID:       matter.TenantID,
		LegalEntityID:  legalEntityID,
		ObjectType:     "MATTER",
		ObjectID:       matter.ID,
		Responsibility: requirement.Responsibility,
		DecisionType:   requirement.CommandName,
		Materiality:    requirement.Materiality,
		At:             now,
	})
	if errors.Is(err, authority.ErrNoRoute) {
		return lifecycleAssignment{Status: StatusBlocked, RoutingState: "NO_ELIGIBLE_ROUTE", PolicyVersion: "unresolved"}, nil
	}
	if errors.Is(err, authority.ErrAmbiguousRoute) {
		return lifecycleAssignment{Status: StatusBlocked, RoutingState: "AMBIGUOUS_ROUTE", PolicyVersion: "unresolved"}, nil
	}
	if err != nil {
		return lifecycleAssignment{}, fmt.Errorf("resolve lifecycle work authority: %w", err)
	}
	assignment := lifecycleAssignment{Status: StatusBlocked, RoutingState: "CANDIDATE_SET", RuleID: resolution.RuleID, PolicyVersion: resolution.PolicyVersion}
	if len(resolution.CandidatePrincipals) > 1 || resolution.Strategy == "CANDIDATE_SET" {
		return assignment, nil
	}
	principal := resolution.Principal
	if strings.TrimSpace(principal.ID) == "" || strings.ToUpper(strings.TrimSpace(principal.Kind)) != "PERSON" {
		assignment.RoutingState = "NON_PERSON_ROUTE"
		return assignment, nil
	}
	if !continuity.MatterVisibleTo(matter, principal.ID) {
		assignment.RoutingState = "ROUTE_NOT_VISIBLE"
		return assignment, nil
	}
	assignment.Status = StatusReady
	assignment.PrincipalID = principal.ID
	assignment.RoutingState = "DIRECT"
	return assignment, nil
}

func (p *MatterLifecycleProjector) resolveMatterLegalEntity(ctx context.Context, tx pgx.Tx, matter continuity.Matter) (string, error) {
	explicit := ""
	var scope map[string]any
	if len(matter.Scope) != 0 && json.Unmarshal(matter.Scope, &scope) == nil {
		if value, ok := scope["legal_entity_id"].(string); ok {
			explicit = strings.TrimSpace(value)
		}
	}

	resolvedExplicit := ""
	if explicit != "" {
		if err := tx.QueryRow(ctx, `
			SELECT id::text
			FROM legal_entities
			WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND (id::text=$2 OR code=$2)
			LIMIT 1`, matter.TenantID, explicit).Scan(&resolvedExplicit); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
		if resolvedExplicit == "" {
			return "", nil
		}
	}

	rows, err := tx.Query(ctx, `
		SELECT DISTINCT p.legal_entity_id::text
		FROM matter_links ml
		JOIN programs p ON p.id=ml.program_id AND p.tenant_id=ml.tenant_id
		WHERE ml.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND ml.matter_id=$2::uuid
		  AND p.legal_entity_id IS NOT NULL
		ORDER BY p.legal_entity_id::text`, matter.TenantID, matter.ID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	linked := make([]string, 0, 2)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return "", err
		}
		linked = append(linked, value)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if resolvedExplicit != "" {
		for _, value := range linked {
			if value != resolvedExplicit {
				return "", nil
			}
		}
		return resolvedExplicit, nil
	}
	if len(linked) == 1 {
		return linked[0], nil
	}
	return "", nil
}

func loadLifecycleTask(ctx context.Context, tx pgx.Tx, tenant, responseID string) (existingLifecycleTask, error) {
	var value existingLifecycleTask
	var contextJSON []byte
	err := tx.QueryRow(ctx, `
		SELECT wi.id::text,wi.state,wi.policy_version,wt.id::text,wt.status,
		       COALESCE(wt.principal_id::text,''),wt.responsibility,wt.title,wt.due_at,wt.context
		FROM workflow_instances wi
		JOIN workflow_tasks wt ON wt.workflow_id=wi.id AND wt.tenant_id=wi.tenant_id AND wt.step_key='lifecycle-response'
		WHERE wi.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND wi.kind='MATTER_RESPONSE'
		  AND wi.subject_type='RESPONSE_PACKAGE'
		  AND wi.subject_id=$2::uuid`, tenant, responseID).Scan(
		&value.WorkflowID,
		&value.WorkflowState,
		&value.PolicyVersion,
		&value.TaskID,
		&value.Status,
		&value.PrincipalID,
		&value.Responsibility,
		&value.Title,
		&value.DueAt,
		&contextJSON,
	)
	if err != nil {
		return existingLifecycleTask{}, err
	}
	if err := json.Unmarshal(contextJSON, &value.Context); err != nil {
		return existingLifecycleTask{}, err
	}
	return value, nil
}

func persistLifecycleTask(ctx context.Context, tx pgx.Tx, tenant, responseID string, existing existingLifecycleTask, desired lifecycleTaskProjection, now time.Time) error {
	if lifecycleTaskEqual(existing, desired) {
		return nil
	}
	contextJSON, err := json.Marshal(desired.Context)
	if err != nil {
		return err
	}

	var workflowID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO workflow_instances(tenant_id,kind,subject_type,subject_id,state,policy_version,context_version,due_at,created_at,updated_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'MATTER_RESPONSE','RESPONSE_PACKAGE',$2::uuid,$3,$4,1,$5,$6,$6)
		ON CONFLICT(tenant_id,kind,subject_type,subject_id) WHERE kind='MATTER_RESPONSE'
		DO UPDATE SET state=EXCLUDED.state,policy_version=EXCLUDED.policy_version,due_at=EXCLUDED.due_at,
		              updated_at=EXCLUDED.updated_at,version=workflow_instances.version+1
		RETURNING id::text`, tenant, responseID, desired.WorkflowState, desired.PolicyVersion, desired.DueAt, now).Scan(&workflowID); err != nil {
		return fmt.Errorf("upsert lifecycle workflow: %w", err)
	}

	var taskID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO workflow_tasks(tenant_id,workflow_id,step_key,responsibility,principal_id,title,status,due_at,context,claimed_at,completed_at,created_at,updated_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,'lifecycle-response',$3,NULLIF($4,'')::uuid,$5,$6,$7,$8::jsonb,NULL,
		       CASE WHEN $6='COMPLETED' THEN $9::timestamptz ELSE NULL END,$9::timestamptz,$9::timestamptz)
		ON CONFLICT(workflow_id,step_key) DO UPDATE SET
			responsibility=EXCLUDED.responsibility,
			principal_id=EXCLUDED.principal_id,
			title=EXCLUDED.title,
			status=EXCLUDED.status,
			due_at=EXCLUDED.due_at,
			context=EXCLUDED.context,
			claimed_at=CASE WHEN EXCLUDED.status IN ('READY','BLOCKED') THEN NULL ELSE workflow_tasks.claimed_at END,
			completed_at=CASE WHEN EXCLUDED.status='COMPLETED' THEN EXCLUDED.updated_at ELSE NULL END,
			version=workflow_tasks.version+1,
			updated_at=EXCLUDED.updated_at
		RETURNING id::text`, tenant, workflowID, desired.Responsibility, desired.PrincipalID, desired.Title, string(desired.Status), desired.DueAt, string(contextJSON), now).Scan(&taskID); err != nil {
		return fmt.Errorf("upsert lifecycle task: %w", err)
	}
	metadata, _ := json.Marshal(map[string]string{
		"response_id":   responseID,
		"target_status": desired.Context["target_status"],
		"routing_state": desired.Context["routing_state"],
		"source":        "MATTER_LIFECYCLE",
	})
	if _, err := tx.Exec(ctx, `
		INSERT INTO workflow_events(tenant_id,workflow_id,task_id,event_type,safe_metadata,occurred_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3::uuid,'TASK_RECONCILED',$4::jsonb,$5)`,
		tenant, workflowID, taskID, string(metadata), now); err != nil {
		return fmt.Errorf("record lifecycle task projection event: %w", err)
	}
	return nil
}

func lifecycleTaskEqual(existing existingLifecycleTask, desired lifecycleTaskProjection) bool {
	if existing.WorkflowID == "" || existing.WorkflowState != desired.WorkflowState || existing.PolicyVersion != desired.PolicyVersion || existing.Status != desired.Status || existing.PrincipalID != desired.PrincipalID || existing.Responsibility != desired.Responsibility || existing.Title != desired.Title || !sameTime(existing.DueAt, desired.DueAt) {
		return false
	}
	if len(existing.Context) != len(desired.Context) {
		return false
	}
	for key, value := range desired.Context {
		if existing.Context[key] != value {
			return false
		}
	}
	return true
}

func sameTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

var _ workflowruntime.Publisher = (*MatterLifecycleProjector)(nil)
var _ workflowruntime.Maintainer = (*MatterLifecycleProjector)(nil)
