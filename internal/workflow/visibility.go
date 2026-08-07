package workflow

import (
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

const MatterActionWorkflowKind = "MATTER_ACTION"

func IsSupportedMatterWorkKind(kind string) bool {
	switch strings.ToUpper(strings.TrimSpace(kind)) {
	case MatterActionWorkflowKind, MatterResponseWorkflowKind:
		return true
	default:
		return false
	}
}

// MatterWorkVisibleTo verifies that an actor-facing Task is one of the
// supported Matter-derived projections and that the actor may read its owning
// Matter. Projection metadata comes from canonical rows, not Task context.
func MatterWorkVisibleTo(task Task, principalID string) bool {
	if !IsSupportedMatterWorkKind(task.WorkflowKind) || strings.TrimSpace(task.MatterID) == "" || strings.TrimSpace(principalID) == "" || len(task.MatterScope) == 0 {
		return false
	}
	return continuity.MatterVisibleTo(continuity.Matter{TenantID: task.TenantID, ID: task.MatterID, Scope: task.MatterScope}, principalID)
}

// MatterActionVisibleTo remains as a narrow compatibility helper for tests and
// call sites that intentionally require Action-derived work only.
func MatterActionVisibleTo(task Task, principalID string) bool {
	return task.WorkflowKind == MatterActionWorkflowKind && MatterWorkVisibleTo(task, principalID)
}
