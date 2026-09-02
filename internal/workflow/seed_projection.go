package workflow

import (
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

// TasksFromMatterAggregates derives the memory/reference Today projection from
// the same stored Matter Actions used by the production projector. It contains
// no sample task facts of its own.
func TasksFromMatterAggregates(values []continuity.MatterAggregate) []Task {
	tasks := make([]Task, 0)
	for _, aggregate := range values {
		for _, action := range aggregate.Actions {
			status, active := seedActionStatus(action.Status)
			if !active || strings.TrimSpace(action.OwnerPrincipalID) == "" {
				continue
			}
			responsibility := strings.TrimSpace(action.RequiredResponsibility)
			if responsibility == "" {
				responsibility = "PERFORMER"
			}
			tasks = append(tasks, Task{
				ID: "matter-action-" + action.ID, TenantID: aggregate.Matter.TenantID, LegalEntityID: aggregate.Matter.LegalEntityID,
				WorkflowID: "matter-action-" + action.ID, WorkflowKind: MatterActionWorkflowKind, StepKey: "matter-action",
				Responsibility: responsibility, PrincipalID: action.OwnerPrincipalID, Title: action.Title, Status: status, DueAt: action.DueAt,
				Context: map[string]string{"type": "MATTER_ACTION", "matter_id": aggregate.Matter.ID, "action_id": action.ID, "action_target_type": "MATTER", "action_target_id": aggregate.Matter.ID, "primary_action": "Update action", "why_now": "You are the current performer for this issue action."},
				Version: action.Version, CreatedAt: action.CreatedAt, UpdatedAt: action.UpdatedAt, MatterID: aggregate.Matter.ID, MatterPriority: aggregate.Matter.Priority, MatterScope: append([]byte(nil), aggregate.Matter.Scope...),
			})
		}
	}
	return tasks
}

func seedActionStatus(value continuity.ActionStatus) (Status, bool) {
	switch value {
	case continuity.ActionPlanned:
		return StatusReady, true
	case continuity.ActionInProgress:
		return StatusInProgress, true
	case continuity.ActionBlocked:
		return StatusBlocked, true
	default:
		return "", false
	}
}
