package workflow

import (
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

const (
	MatterActionWorkflowKind    = "MATTER_ACTION"
	MatterLifecycleWorkflowKind = "MATTER_LIFECYCLE"
	EvidenceRequestWorkflowKind = "EVIDENCE_REQUEST"
)

// ActorWorkVisibleTo is the defense-in-depth actor-work boundary after the
// PostgreSQL read has already applied canonical visibility before LIMIT.
func ActorWorkVisibleTo(task Task, principalID string) bool {
	principalID = strings.TrimSpace(principalID)
	if principalID == "" || task.PrincipalID != principalID {
		return false
	}
	if supportedMatterWorkflowKind(task.WorkflowKind) {
		return MatterWorkVisibleTo(task, principalID)
	}
	if task.WorkflowKind == EvidenceRequestWorkflowKind {
		return strings.TrimSpace(task.EvidenceRequestID) != "" &&
			task.EvidenceRecipientID == principalID &&
			task.EvidenceSubjectVisible
	}
	if task.WorkflowKind == "DOCUMENT_IMPORT" {
		return task.DocumentProposalVisible
	}
	return false
}

// MatterWorkVisibleTo verifies that an actor-facing Task is one of the supported
// Matter-backed projections and that the actor may read its owning Matter.
// Projection metadata is populated from canonical rows by the repository, not
// from client-controlled Task context.
func MatterWorkVisibleTo(task Task, principalID string) bool {
	if !supportedMatterWorkflowKind(task.WorkflowKind) || strings.TrimSpace(task.MatterID) == "" || strings.TrimSpace(principalID) == "" || len(task.MatterScope) == 0 {
		return false
	}
	return continuity.MatterVisibleTo(continuity.Matter{TenantID: task.TenantID, ID: task.MatterID, Scope: task.MatterScope}, principalID)
}

// MatterActionVisibleTo is retained as the narrow semantic helper used by
// Matter Action-specific tests and callers.
func MatterActionVisibleTo(task Task, principalID string) bool {
	return task.WorkflowKind == MatterActionWorkflowKind && MatterWorkVisibleTo(task, principalID)
}

func supportedMatterWorkflowKind(kind string) bool {
	switch strings.TrimSpace(kind) {
	case MatterActionWorkflowKind, MatterLifecycleWorkflowKind:
		return true
	default:
		return false
	}
}
