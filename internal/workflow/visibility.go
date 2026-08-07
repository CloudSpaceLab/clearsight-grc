package workflow

import (
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

const MatterActionWorkflowKind = "MATTER_ACTION"

// MatterActionVisibleTo verifies that an actor-facing Task is the supported
// Matter Action projection and that the actor may read its owning Matter.
// Projection metadata is populated from canonical rows by the repository, not
// from client-controlled Task context.
func MatterActionVisibleTo(task Task, principalID string) bool {
	if task.WorkflowKind != MatterActionWorkflowKind || strings.TrimSpace(task.MatterID) == "" || strings.TrimSpace(principalID) == "" || len(task.MatterScope) == 0 {
		return false
	}
	return continuity.MatterVisibleTo(continuity.Matter{TenantID: task.TenantID, ID: task.MatterID, Scope: task.MatterScope}, principalID)
}
