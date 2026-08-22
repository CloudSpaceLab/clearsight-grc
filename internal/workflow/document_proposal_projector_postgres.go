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

// Maintain cycles through currently active handoffs so authority/policy changes
// converge without creating a second assignment store or requiring a new
// document event. The cursor prevents a large active set from pinning the same
// first page on every pass.
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
		"type":              "DOCUMENT_PROPOSAL",
		"document_import_id": document.ID,
		"proposal_id":       proposal.ID,
		"handoff_id":        handoff.ID,
		"handoff_status":    string(handoff.Status),
		"handoff_version":   strconv.FormatInt(handoff.Version, 10),
		"document_version":  strconv.FormatInt(document.Version, 10),
		"action_target_type": "DOCUMENT_IMPORT",
		"action_target_id":   document.ID,
		"scope":              strings.TrimSpace(proposal.Title),
		"source_anchor":      proposalSourceAnchor(proposal),
	}

	tasks := make([]projectedDocumentProposalTask, 0, 2)
	switch handoff.Status {
	case documentimport.HandoffAwaitingReview:
		active, err := p.resolveActiveTask(ctx, document, proposal, authority.ResponsibilityReviewer, "document.proposal.review", "Review imported proposal", documentProposalReviewStep, handoff.IntakePrincipalID, "", at, base)
		if err != nil {
			return err
		}
		tasks = append(tasks, active)
	case documentimport.HandoffAwaitingAuthorization:
		tasks = append(tasks, completedHandoffTask(documentProposalReviewStep, string(authority.ResponsibilityReviewer), handoff.ReviewerPrincipalID, "Review imported proposal", base))
		active, err := p.resolveActiveTask(ctx, document, proposal, authority.ResponsibilityAuthorizer, "document.proposal.authorize", "Authorize proposal conversion", documentProposalAuthStep, handoff.IntakePrincipalID, handoff.ReviewerPrincipalID, at, base)
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

func (p *DocumentProposalProjector) resolveActiveTask(
	ctx context.Context,
	document documentimport.Document,
	proposal documentimport.Proposal,
	responsibility authority.Responsibility,
	decisionType, primaryAction, stepKey, excludeA, excludeB string,
	at time.Time,
	base map[string]string,
) (projectedDocumentProposalTask, error) {
	contextValues := cloneStringMap(base)
	contextValues["primary_action"] = primaryAction
	contextValues["decision_type"] = decisionType
	contextValues["why_now"] = "An accepted document proposal requires governed " + strings.ToLower(string(responsibility)) + "."
	status := StatusBlocked
	principalID := ""
	if strings.TrimSpace(document.LegalEntityID) == "" {
		contextValues["routing_status"] = "NO_LEGAL_ENTITY"
		return projectedDocumentProposalTask{StepKey: stepKey, Responsibility: string(responsibility), Title: primaryAction + ": " + proposal.Title, Status: status, Context: contextValues}, nil
	}
	resolution, err := p.Authority.Resolve(ctx, authority.ResolveInput{
		TenantID: document.TenantID, LegalEntityID: document.LegalEntityID,
		ObjectType: "DOCUMENT_IMPORT", ObjectID: document.ID,
		Responsibility: responsibility, DecisionType: decisionType, Materiality: 3, At: at,
	})
	switch {
	case err == nil:
		contextValues["authority_rule_id"] = resolution.RuleID
		contextValues["authority_policy_version"] = resolution.PolicyVersion
		contextValues["authority_strategy"] = resolution.Strategy
		contextValues["authority_explanation"] = resolution.Explanation
		principals := independentAuthorityPrincipals(resolution, excludeA, excludeB)
		contextValues["authority_candidate_count"] = strconv.Itoa(len(principals))
		switch len(principals) {
		case 0:
			contextValues["routing_status"] = "NO_INDEPENDENT_CANDIDATE"
		case 1:
			principalID, status = principals[0].ID, StatusReady
			contextValues["routing_status"] = "DIRECT"
			contextValues["assigned_principal_name"] = principals[0].DisplayName
		default:
			contextValues["routing_status"] = "CANDIDATE_SET"
		}
	case errors.Is(err, authority.ErrNoRoute):
		contextValues["routing_status"] = "NO_ROUTE"
	case errors.Is(err, authority.ErrAmbiguousRoute):
		contextValues["routing_status"] = "AMBIGUOUS_ROUTE"
	default:
		return projectedDocumentProposalTask{}, fmt.Errorf("resolve document proposal authority: %w", err)
	}
	return projectedDocumentProposalTask{
		StepKey: stepKey, Responsibility: string(responsibility), PrincipalID: principalID,
		Title: primaryAction + ": " + proposal.Title, Status: status, Context: contextValues,
	}, nil
}

func independentAuthorityPrincipals(resolution authority.Resolution, excluded ...string) []authority.Principal {
	excludedIDs := map[string]struct{}{}
	for _, value := range excluded {
		if value = strings.TrimSpace(value); value != "" {
			excludedIDs[value] = struct{}{}
		}
	}
	values := append([]authority.Principal(nil), resolution.CandidatePrincipals...)
	if len(values) == 0 && strings.TrimSpace(resolution.Principal.ID) != "" {
		values = append(values, resolution.Principal)
	}
	seen := map[string]struct{}{}
	result := make([]authority.Principal, 0, len(values))
	for _, value := range values {
		id := strings.TrimSpace(value.ID)
		if id == "" || value.Kind != "PERSON" {
			continue
		}
		if _, blocked := excludedIDs[id]; blocked {
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
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
		INSERT INTO workflow_instances(tenant_id,kind,subject_type,subject_id,state,policy_version,context_version,created_at,updated_at)
		VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2,$3,$4::uuid,$5,'document-proposal-handoff-v1',1,$6,$6)
		ON CONFLICT(tenant_id,kind,subject_type,subject_id)
		DO UPDATE SET state=EXCLUDED.state,updated_at=EXCLUDED.updated_at,version=workflow_instances.version+1
		RETURNING id::text`, document.TenantID, DocumentProposalWorkflowKind, documentProposalSubjectType, handoff.ID, string(workflowState), at).Scan(&workflowID); err != nil {
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
		if err := tx.QueryRow(ctx, `
			INSERT INTO workflow_tasks(tenant_id,workflow_id,step_key,responsibility,principal_id,title,status,context,source_bindings,completed_at,created_at,updated_at)
			VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3,$4,NULLIF($5,'')::uuid,$6,$7,$8::jsonb,'[]'::jsonb,
			       CASE WHEN $7='COMPLETED' THEN $9::timestamptz ELSE NULL END,$9::timestamptz,$9::timestamptz)
			ON CONFLICT(workflow_id,step_key) DO UPDATE SET
				responsibility=EXCLUDED.responsibility,principal_id=EXCLUDED.principal_id,title=EXCLUDED.title,status=EXCLUDED.status,
				context=EXCLUDED.context,completed_at=CASE WHEN EXCLUDED.status='COMPLETED' THEN COALESCE(workflow_tasks.completed_at,EXCLUDED.updated_at) ELSE NULL END,
				version=workflow_tasks.version+1,updated_at=EXCLUDED.updated_at
			RETURNING id::text`, document.TenantID, workflowID, task.StepKey, task.Responsibility, task.PrincipalID, task.Title, string(task.Status), string(contextJSON), at).Scan(&taskID); err != nil {
			return fmt.Errorf("project document proposal task: %w", err)
		}
		metadata, _ := json.Marshal(map[string]string{"handoff_status": string(handoff.Status), "step_key": task.StepKey})
		if _, err := tx.Exec(ctx, `
			INSERT INTO workflow_events(tenant_id,workflow_id,task_id,event_type,safe_metadata,occurred_at)
			VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3::uuid,'TASK_PROJECTED',$4::jsonb,$5)`,
			document.TenantID, workflowID, taskID, string(metadata), at); err != nil {
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
