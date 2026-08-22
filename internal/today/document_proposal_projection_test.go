package today

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

func TestDocumentProposalWorkPreservesExactFocusedTarget(t *testing.T) {
	task := workflow.Task{
		ID:                      "task-1",
		TenantID:                "bank",
		WorkflowKind:            workflow.DocumentProposalWorkflowKind,
		StepKey:                 "document-proposal-authorization",
		Responsibility:          "AUTHORIZER",
		PrincipalID:             "authorizer-1",
		Status:                  workflow.StatusReady,
		Title:                   "Authorize proposal conversion",
		DocumentProposalVisible: true,
		Context: map[string]string{
			"type":               "DOCUMENT_PROPOSAL",
			"action_target_type": "DOCUMENT_IMPORT",
			"action_target_id":   "document-1",
			"document_import_id": "document-1",
			"proposal_id":        "proposal-1",
			"primary_action":     "Authorize proposal conversion",
			"handoff_status":     "AWAITING_AUTHORIZATION",
			"handoff_version":    "2",
			"document_version":   "4",
		},
	}

	items := FromWorkflowTasksForActor([]workflow.Task{task}, "authorizer-1")
	if len(items) != 1 {
		t.Fatalf("expected one actor-visible item, got %d", len(items))
	}
	item := items[0]
	if item.ActionTargetType != "DOCUMENT_IMPORT" || item.ActionTargetID != "document-1" || item.ActionTargetSubID != "proposal-1" {
		t.Fatalf("unexpected focused target: type=%q id=%q sub=%q", item.ActionTargetType, item.ActionTargetID, item.ActionTargetSubID)
	}
	if item.Authority == nil || item.Authority.Responsibility != "AUTHORIZER" || item.Authority.DecisionType != "document.proposal.authorize" || item.Authority.Materiality != 4 {
		t.Fatalf("unexpected authority context: %#v", item.Authority)
	}
}

func TestDocumentProposalWorkRejectsUnprovenVisibility(t *testing.T) {
	task := workflow.Task{
		ID:             "task-1",
		TenantID:       "bank",
		WorkflowKind:   workflow.DocumentProposalWorkflowKind,
		PrincipalID:    "reviewer-1",
		Status:         workflow.StatusReady,
		Responsibility: "REVIEWER",
		Context: map[string]string{
			"action_target_type": "DOCUMENT_IMPORT",
			"action_target_id":   "document-1",
			"proposal_id":        "proposal-1",
		},
	}

	if items := FromWorkflowTasksForActor([]workflow.Task{task}, "reviewer-1"); len(items) != 0 {
		t.Fatalf("expected source-domain visibility proof to be required, got %#v", items)
	}
}
