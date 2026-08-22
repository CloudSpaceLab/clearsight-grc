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
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
)

const (
	DocumentProposalWorkflowKind = "DOCUMENT_IMPORT"
	documentProposalSubjectType  = "DOCUMENT_PROPOSAL"
	documentProposalReviewStep   = "document-proposal-review"
	documentProposalAuthStep     = "document-proposal-authorization"
)

type DocumentProposalProjector struct {
	Repo      *PostgresRepository
	Documents *documentimport.Service
	Authority authority.Service
	Now       func() time.Time

	cursorMu      sync.Mutex
	cursorTenant  string
	cursorHandoff string
}

type projectedDocumentProposalTask struct {
	StepKey        string
	Responsibility string
	PrincipalID    string
	Title          string
	Status         Status
	Context        map[string]string
}

func (p *DocumentProposalProjector) Publish(ctx context.Context, event workflowruntime.OutboxEvent) error {
	if p == nil || p.Repo == nil || p.Documents == nil || p.Authority == nil || event.AggregateType != "DOCUMENT_IMPORT" {
		return nil
	}
	if event.EventType != documentimport.EventDocumentProposalAccepted && event.EventType != documentimport.EventDocumentProposalHandoffChanged {
		return nil
	}
	return p.ReconcileDocument(ctx, event.TenantID, event.AggregateID, p.currentTime())
}

// Maintain cycles through active handoffs so current authority/policy changes
// converge without another assignment store or a synthetic document mutation.
// Reconciliation is write-free when the desired projection is unchanged.
func (p *DocumentProposalProjector) Maintain(ctx context.Context, now time.Time, limit int) (int, error) {
	if p == nil || p.Repo == nil || p.Documents == nil || p.Authority == nil {
		return 0, nil
	}
	if limit < 1 || limit > 500 {
		limit = 100
	}
	if now.IsZero() {
		now = p.currentTime()
	} else {
		now = now.UTC()
	}
	p.cursorMu.Lock()
	cursorTenant, cursorHandoff := p.cursorTenant, p.cursorHandoff
	p.cursorMu.Unlock()

	rows, err := p.Repo.pool.Query(ctx, `
		SELECT DISTINCT t.slug,di.id::text,p.value->'handoff'->>'id' AS handoff_id
		FROM document_imports di
		JOIN tenants t ON t.id=di.tenant_id
		CROSS JOIN LATERAL jsonb_array_elements(di.proposals) AS p(value)
		WHERE p.value->>'status'='ACCEPTED'
		  AND p.value->'handoff'->>'status' IN ('AWAITING_REVIEW','AWAITING_AUTHORIZATION')
		  AND ($1='' OR (t.slug,p.value->'handoff'->>'id') > ($1,$2))
		ORDER BY t.slug,p.value->'handoff'->>'id'
		LIMIT $3`, cursorTenant, cursorHandoff, limit)
	if err != nil {
		return 0, fmt.Errorf("list active document proposal handoffs: %w", err)
	}
	defer rows.Close()
	type target struct{ tenant, document, handoff string }
	targets := make([]target, 0, limit)
	for rows.Next() {
		var value target
		if err := rows.Scan(&value.tenant, &value.document, &value.handoff); err != nil {
			return len(targets), err
		}
		targets = append(targets, value)
	}
	if err := rows.Err(); err != nil {
		return len(targets), err
	}
	if len(targets) == 0 {
		p.cursorMu.Lock()
		p.cursorTenant, p.cursorHandoff = "", ""
		p.cursorMu.Unlock()
		return 0, nil
	}

	seenDocuments := map[string]struct{}{}
	processed := 0
	for _, target := range targets {
		key := target.tenant + "\x00" + target.document
		if _, seen := seenDocuments[key]; seen {
			continue
		}
		seenDocuments[key] = struct{}{}
		if err := p.ReconcileDocument(ctx, target.tenant, target.document, now); err != nil {
			return processed, err
		}
		processed++
	}
	last := targets[len(targets)-1]
	p.cursorMu.Lock()
	p.cursorTenant, p.cursorHandoff = last.tenant, last.handoff
	p.cursorMu.Unlock()
	return processed, nil
}

func (p *DocumentProposalProjector) ReconcileDocument(ctx context.Context, tenant, documentID string, at time.Time) error {
	tenant, documentID = strings.TrimSpace(tenant), strings.TrimSpace(documentID)
	if tenant == "" || documentID == "" {
		return nil
	}
	if at.IsZero() {
		at = p.currentTime()
	} else {
		at = at.UTC()
	}
	document, err := p.Documents.Get(ctx, tenant, documentID)
	if err != nil {
		if errors.Is(err, documentimport.ErrNotFound) {
			return nil
		}
		return err
	}
	for _, proposal := range document.Proposals {
		if proposal.Handoff == nil {
			continue
		}
		if err := p.reconcileHandoff(ctx, document, proposal, at); err != nil {
			return fmt.Errorf("reconcile document proposal %s: %w", proposal.ID, err)
		}
	}
	return nil
}

func (p *DocumentProposalProjector) reconcileHandoff(ctx context.Context, document documentimport.Document, proposal documentimport.Proposal, at time.Time) error {
	handoff := proposal.Handoff
	if handoff == nil || strings.TrimSpace(handoff.ID) == "" {
		return nil
	}
	base := map[string]string{
		"type":               "DOCUMENT_PROPOSAL",
		"document_import_id": document.ID,
		"proposal_id":        proposal.ID,
		"handoff_id":         handoff.ID,
		"handoff_status":     string(handoff.Status),
		"handoff_version":    strconv.FormatInt(handoff.Version, 10),
		"document_version":   strconv.FormatInt(document.Version, 10),
		"action_target_type": "DOCUMENT_IMPORT",
		"action_target_id":   document.ID,
		"scope":              strings.TrimSpace(proposal.Title),
		"source_anchor":      proposalSourceAnchor(proposal),
	}

	tasks := make([]projectedDocumentProposalTask, 0, 2)
	switch handoff.Status {
	case documentimport.HandoffAwaitingReview:
		active, err := p.resolveActiveTask(ctx, document, proposal, "Review imported proposal", documentProposalReviewStep, at, base)
		if err != nil {
			return err
		}
		tasks = append(tasks, active)
	case documentimport.HandoffAwaitingAuthorization:
		tasks = append(tasks, completedHandoffTask(documentProposalReviewStep, string(authority.ResponsibilityReviewer), handoff.ReviewerPrincipalID, "Review imported proposal", base))
		active, err := p.resolveActiveTask(ctx, document, proposal, "Authorize proposal conversion", documentProposalAuthStep, at, base)
		if err != nil {
			return err
		}
		tasks = append(tasks, active)
	case documentimport.HandoffReturned, documentimport.HandoffRejected:
		tasks = append(tasks, completedHandoffTask(documentProposalReviewStep, string(authority.ResponsibilityReviewer), handoff.ReviewerPrincipalID, "Review imported proposal", base))
		if handoff.AuthorizerPrincipalID != "" {
			tasks = append(tasks, completedHandoffTask(documentProposalAuthStep, string(authority.ResponsibilityAuthorizer), handoff.AuthorizerPrincipalID, "Authorize proposal conversion", base))
		}
	case documentimport.HandoffApproved, documentimport.HandoffConversionFailed:
		tasks = append(tasks,
			completedHandoffTask(documentProposalReviewStep, string(authority.ResponsibilityReviewer), handoff.ReviewerPrincipalID, "Review imported proposal", base),
			completedHandoffTask(documentProposalAuthStep, string(authority.ResponsibilityAuthorizer), handoff.AuthorizerPrincipalID, "Authorize proposal conversion", base),
		)
	default:
		return nil
	}
	return p.syncHandoffWorkflow(ctx, document, proposal, tasks, at)
}

func (p *DocumentProposalProjector) resolveActiveTask(ctx context.Context, document documentimport.Document, proposal documentimport.Proposal, primaryAction, stepKey string, at time.Time, base map[string]string) (projectedDocumentProposalTask, error) {
	contextValues := cloneStringMap(base)
	contextValues["primary_action"] = primaryAction
	route, err := (documentimport.HandoffAuthorityResolver{Authority: p.Authority}).Resolve(ctx, document, *proposal.Handoff, "", at)
	if err != nil {
		return projectedDocumentProposalTask{}, fmt.Errorf("resolve document proposal authority: %w", err)
	}
	responsibility := ""
	status := StatusBlocked
	principalID := ""
	if route != nil {
		responsibility = route.Responsibility
		contextValues["routing_status"] = route.Status
		contextValues["authority_rule_id"] = route.RuleID
		contextValues["authority_policy_version"] = route.PolicyVersion
		contextValues["authority_explanation"] = route.Explanation
		if route.PrincipalName != "" {
			contextValues["assigned_principal_name"] = route.PrincipalName
		}
		if route.Status == "DIRECT" && route.PrincipalID != "" {
			principalID, status = route.PrincipalID, StatusReady
		}
	}
	if responsibility == "" {
		switch proposal.Handoff.Status {
		case documentimport.HandoffAwaitingReview:
			responsibility = string(authority.ResponsibilityReviewer)
		case documentimport.HandoffAwaitingAuthorization:
			responsibility = string(authority.ResponsibilityAuthorizer)
		}
	}
	contextValues["why_now"] = "An accepted document proposal requires governed " + strings.ToLower(responsibility) + "."
	return projectedDocumentProposalTask{StepKey: stepKey, Responsibility: responsibility, PrincipalID: principalID, Title: primaryAction + ": " + proposal.Title, Status: status, Context: contextValues}, nil
}

func completedHandoffTask(stepKey, responsibility, principalID, title string, base map[string]string) projectedDocumentProposalTask {
	contextValues := cloneStringMap(base)
	contextValues["primary_action"] = title
	contextValues["routing_status"] = "COMPLETED"
	return projectedDocumentProposalTask{StepKey: stepKey, Responsibility: responsibility, PrincipalID: principalID, Title: title, Status: StatusCompleted, Context: contextValues}
}

func (p *DocumentProposalProjector) syncHandoffWorkflow(ctx context.Context, document documentimport.Document, proposal documentimport.Proposal, tasks []projectedDocumentProposalTask, at time.Time) error {
	handoff := proposal.Handoff
	if handoff == nil {
		return nil
	}
	tx, err := p.Repo.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	workflowState := StatusCompleted
	for _, task := range tasks {
		if task.Status == StatusReady || task.Status == StatusBlocked || task.Status == StatusInProgress || task.Status == StatusEscalated {
			workflowState = task.Status
			break
		}
	}
	var workflowID string
	if err := tx.QueryRow(ctx, `
		WITH changed AS (
			INSERT INTO workflow_instances(tenant_id,kind,subject_type,subject_id,state,policy_version,context_version,created_at,updated_at)
			VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3,$4::uuid,$5,'document-proposal-handoff-v1',1,$6,$6)
			ON CONFLICT(tenant_id,kind,subject_type,subject_id) DO UPDATE SET
				state=EXCLUDED.state,updated_at=EXCLUDED.updated_at,version=workflow_instances.version+1
			WHERE workflow_instances.state IS DISTINCT FROM EXCLUDED.state
			RETURNING id::text
		)
		SELECT id FROM changed
		UNION ALL
		SELECT wi.id::text FROM workflow_instances wi
		JOIN tenants t ON t.id=wi.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND wi.kind=$2 AND wi.subject_type=$3 AND wi.subject_id=$4::uuid
		  AND NOT EXISTS (SELECT 1 FROM changed)
		LIMIT 1`, document.TenantID, DocumentProposalWorkflowKind, documentProposalSubjectType, handoff.ID, string(workflowState), at).Scan(&workflowID); err != nil {
		return fmt.Errorf("project document proposal workflow: %w", err)
	}

	desired := map[string]struct{}{}
	for _, task := range tasks {
		desired[task.StepKey] = struct{}{}
		contextJSON, err := json.Marshal(task.Context)
		if err != nil {
			return err
		}
		var taskID string
		var changed bool
		if err := tx.QueryRow(ctx, `
			WITH changed AS (
				INSERT INTO workflow_tasks(tenant_id,workflow_id,step_key,responsibility,principal_id,title,status,context,source_bindings,completed_at,created_at,updated_at)
				VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3,$4,NULLIF($5,'')::uuid,$6,$7,$8::jsonb,'[]'::jsonb,
				       CASE WHEN $7='COMPLETED' THEN $9::timestamptz ELSE NULL END,$9::timestamptz,$9::timestamptz)
				ON CONFLICT(workflow_id,step_key) DO UPDATE SET
					responsibility=EXCLUDED.responsibility,principal_id=EXCLUDED.principal_id,title=EXCLUDED.title,status=EXCLUDED.status,
					context=EXCLUDED.context,
					completed_at=CASE WHEN EXCLUDED.status='COMPLETED' THEN COALESCE(workflow_tasks.completed_at,EXCLUDED.updated_at) ELSE NULL END,
					version=workflow_tasks.version+1,updated_at=EXCLUDED.updated_at
				WHERE workflow_tasks.responsibility IS DISTINCT FROM EXCLUDED.responsibility
				   OR workflow_tasks.principal_id IS DISTINCT FROM EXCLUDED.principal_id
				   OR workflow_tasks.title IS DISTINCT FROM EXCLUDED.title
				   OR workflow_tasks.status IS DISTINCT FROM EXCLUDED.status
				   OR workflow_tasks.context IS DISTINCT FROM EXCLUDED.context
				   OR (EXCLUDED.status='COMPLETED' AND workflow_tasks.completed_at IS NULL)
				   OR (EXCLUDED.status<>'COMPLETED' AND workflow_tasks.completed_at IS NOT NULL)
				RETURNING id::text
			)
			SELECT id,true FROM changed
			UNION ALL
			SELECT wt.id::text,false FROM workflow_tasks wt
			WHERE wt.workflow_id=$2::uuid AND wt.step_key=$3 AND NOT EXISTS (SELECT 1 FROM changed)
			LIMIT 1`, document.TenantID, workflowID, task.StepKey, task.Responsibility, task.PrincipalID, task.Title, string(task.Status), string(contextJSON), at).Scan(&taskID, &changed); err != nil {
			return fmt.Errorf("project document proposal task: %w", err)
		}
		if !changed {
			continue
		}
		metadata, _ := json.Marshal(map[string]string{"handoff_status": string(handoff.Status), "step_key": task.StepKey})
		if _, err := tx.Exec(ctx, `
			INSERT INTO workflow_events(tenant_id,workflow_id,task_id,event_type,safe_metadata,occurred_at)
			VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3::uuid,'TASK_PROJECTED',$4::jsonb,$5)`, document.TenantID, workflowID, taskID, string(metadata), at); err != nil {
			return fmt.Errorf("record document proposal projection event: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE workflow_tasks
		SET status='CANCELLED',principal_id=NULL,completed_at=NULL,version=version+1,updated_at=$3
		WHERE workflow_id=$1::uuid
		  AND step_key IN ($4,$5)
		  AND NOT (step_key = ANY($2::text[]))
		  AND status NOT IN ('COMPLETED','CANCELLED')`, workflowID, mapKeys(desired), at, documentProposalReviewStep, documentProposalAuthStep); err != nil {
		return fmt.Errorf("retire obsolete document proposal task: %w", err)
	}
	return tx.Commit(ctx)
}

func proposalSourceAnchor(proposal documentimport.Proposal) string {
	anchor := proposal.Anchor
	parts := make([]string, 0, 3)
	if anchor.SectionID != "" {
		parts = append(parts, "section "+anchor.SectionID)
	}
	if anchor.Page > 0 {
		parts = append(parts, "page "+strconv.Itoa(anchor.Page))
	}
	if anchor.Sheet != "" {
		parts = append(parts, "sheet "+anchor.Sheet)
	}
	return strings.Join(parts, " · ")
}

func cloneStringMap(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func mapKeys(input map[string]struct{}) []string {
	values := make([]string, 0, len(input))
	for key := range input {
		values = append(values, key)
	}
	sort.Strings(values)
	return values
}

func (p *DocumentProposalProjector) currentTime() time.Time {
	if p != nil && p.Now != nil {
		return p.Now().UTC()
	}
	return time.Now().UTC()
}
