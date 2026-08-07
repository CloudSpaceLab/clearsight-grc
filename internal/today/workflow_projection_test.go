package today

import (
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

func TestFromWorkflowTasksProjectsOnlyActiveAssignedWork(t *testing.T) {
	now := time.Now().UTC()
	due := now.Add(time.Hour)
	tasks := []workflow.Task{
		{
			ID: "review-1", TenantID: "bank", Responsibility: "AUTHORIZER", Title: "Approve remediation",
			Status: workflow.StatusReady, DueAt: &due,
			Context: map[string]string{
				"scope": "Retail Payments", "action_target_type": "MATTER", "action_target_id": "matter-1",
				"material_conclusion": "The remediation changes a material control conclusion.",
			},
		},
		{ID: "done-1", TenantID: "bank", Responsibility: "REVIEWER", Title: "Already completed", Status: workflow.StatusCompleted},
	}

	items := FromWorkflowTasks(tasks)
	if len(items) != 1 {
		t.Fatalf("expected one active intervention, got %d", len(items))
	}
	item := items[0]
	if item.InterventionClass != InterventionAuthorization {
		t.Fatalf("expected authorization intervention, got %s", item.InterventionClass)
	}
	if item.ActionTargetType != "MATTER" || item.ActionTargetID != "matter-1" {
		t.Fatalf("unexpected target: %s/%s", item.ActionTargetType, item.ActionTargetID)
	}
	if item.Recommendation != nil || item.PreparedWork != nil {
		t.Fatal("workflow projection must not fabricate operator recommendation or prepared-work receipt")
	}
}

func TestFromWorkflowTasksUsesCanonicalTargetFallbacks(t *testing.T) {
	items := FromWorkflowTasks([]workflow.Task{{
		ID: "task-1", Responsibility: "REVIEWER", Title: "Review issue", Status: workflow.StatusReady,
		Context: map[string]string{"matter_id": "matter-42"},
	}})
	if len(items) != 1 || items[0].ActionTargetType != "MATTER" || items[0].ActionTargetID != "matter-42" {
		t.Fatalf("canonical matter target was not projected: %#v", items)
	}
}

func TestFromWorkflowTasksRejectsUnknownActionTargets(t *testing.T) {
	items := FromWorkflowTasks([]workflow.Task{{
		ID: "task-1", Responsibility: "REVIEWER", Title: "Review task", Status: workflow.StatusReady,
		Context: map[string]string{"action_target_type": "SECRET_OBJECT", "action_target_id": "hidden"},
	}})
	if len(items) != 1 {
		t.Fatalf("expected one projected task, got %d", len(items))
	}
	if items[0].ActionTargetType != "" || items[0].ActionTargetID != "" {
		t.Fatal("unknown action target must not become a navigable route")
	}
}
